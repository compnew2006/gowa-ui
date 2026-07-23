# Research: Chat Status, Claim & Collaboration

**Feature**: 001-chat-claim-collaboration  
**Date**: 2026-07-12

---

## R1: Where to set `chat_status = pending` on incoming messages

**Decision**: Set it in `processIncomingMessageFull` in `internal/handlers/chatbot_processor.go`, immediately after `GetOrCreateContact` returns — NOT in `processGowaMessage` (which only builds an `IncomingTextMessage` struct) and NOT in `GetOrCreateContact` itself (which is a generic utility used by outgoing messages too).

**Rationale**: `processIncomingMessageFull` is the single convergence point for ALL incoming messages regardless of provider. `GetOrCreateContact` is also called by `processGowaOutgoingMessage` (gowa_webhook.go:338) — setting pending there would wrongly mark outgoing-created contacts as pending.

**Implementation**: After the existing `contactutil.GetOrCreateContact(...)` call, add:
```go
if contact.AssignedUserID == nil {
    if contact.Metadata == nil || contact.Metadata["chat_status"] == nil {
        if contact.Metadata == nil { contact.Metadata = models.JSONB{} }
        contact.Metadata["chat_status"] = "pending"
        a.DB.Model(contact).Update("metadata", contact.Metadata)
    }
}
```

**Alternatives considered**:
- In `processGowaMessage` (gowa_webhook.go): Rejected — it only builds `IncomingTextMessage`, doesn't have the contact yet.
- In `GetOrCreateContact` (contactutil pkg): Rejected — it's also used for outgoing messages, would set pending incorrectly.
- Database trigger: Rejected — violates Constitution Principle 7 (AutoMigrate-only, no raw SQL).

---

## R2: New permission resource — `chat.collaborate` vs extending `chat.assign`

**Decision**: Create a NEW resource constant `ResourceChatCollaborate = "chat.collaborate"` distinct from the existing `ResourceChatAssign = "chat.assign"`.

**Rationale**: The two actions have fundamentally different semantics:
- `chat.assign:write` = "take ownership of an unassigned conversation" (claim)
- `chat.collaborate:write` = "join a conversation owned by someone else as a helper" (collaborate)

Conflating them would mean either (a) every agent can join any conversation (security risk) or (b) managers can't claim (workflow breakage). Separate resources let admins configure these independently via `/settings/roles`.

**Default role distribution**:
- `agent`: gets `chat.assign:write` (can claim pending) but NOT `chat.collaborate:write`
- `manager`: gets both
- `admin`: gets all (automatic via `allPermissions`)

**Alternatives considered**:
- Single `chat.assign:write` with scope parameter: Rejected — overloads one permission with two meanings, hard to audit.
- `chat.claim_override` (from earlier plan iteration): Rejected — "override" implies stealing, but we want collaboration (both stay in the conversation).

---

## R3: Storing collaborators in Metadata vs dedicated table

**Decision**: Store collaborators as a JSONB array in `Contact.Metadata["collaborators"]`.

**Rationale**:
- Constitution Principle 8 mandates JSONB for flexible/extended data.
- No migration needed (Metadata already exists, defaults to `{}`).
- Matches the existing `is_group_chat` precedent in the same field.
- Collaborator lists are small (typically 1-3 people) and read-with-contact — no need for JOIN queries.

**Alternatives considered**:
- Dedicated `chat_collaborators` table (contact_id, user_id, joined_at): Rejected for Phase 1 — adds migration overhead, requires JOIN on every contact list query, and the data volume doesn't justify it. Can be revisited if collaboration scales to 10+ per chat.

---

## R4: Frontend WebSocket field mapping

**Decision**: The `handleNewMessage` handler in `websocket.ts` picks 19 specific fields manually. New system-message events (`chat_claimed`, `collaborator_joined`, `collaborator_left`) need their OWN switch cases — they are NOT message types that flow through `handleNewMessage`.

**Rationale**: Constitution Principle 9 requires explicit field mapping (no generic copy). The new events are separate WebSocket message types with distinct payloads, not extensions of `new_message`.

**Implementation**: Add 3 new cases to the `handleMessage` switch in `websocket.ts`:
- `case 'chat_claimed'` — update contact's `assigned_user_id` + `chat_status`
- `case 'collaborator_joined'` — push to contact's `collaborators[]`
- `case 'collaborator_left'` — filter from `collaborators[]`

---

## R5: System message rendering

**Decision**: System messages (claim/join/leave) are stored as regular `Message` records with `metadata.is_system_message = true` and rendered naturally in the message timeline with a distinct CSS class.

**Rationale**: Reuses the entire existing message pipeline (storage, retrieval, WebSocket broadcast, rendering). No new message type or table needed. The frontend can apply a `.chat-bubble-system` class when `message.metadata?.is_system_message` is true.

**Alternatives considered**:
- Separate `SystemEvent` model: Rejected — duplicates message pipeline for no benefit.
- WebSocket-only (don't persist): Rejected — late-joining collaborators wouldn't see the history.

---

## R6: Concurrency safety for claims

**Decision**: Rely on GORM's built-in row-level locking via `Save()` which issues `UPDATE ... WHERE id = ?`. The guard sequence (check assigned_user_id → set assigned_user_id) is wrapped in a transaction with `tx.Clauses(clause.Locking{Strength: "UPDATE"})`.

**Rationale**: Two agents pressing "Claim" simultaneously will both read `assigned_user_id = nil`, but the `SELECT FOR UPDATE` lock serializes them. The second transaction sees the first's commit and returns 409.

**Alternatives considered**:
- Optimistic concurrency (version column): Rejected — adds a column, more complex.
- PostgreSQL advisory lock: Rejected — overkill for this use case.
- Unique partial index on `(id) WHERE assigned_user_id IS NULL`: Rejected — conflicts with the unassign flow.

---

## R7: Backward compatibility for existing conversations

**Decision**: `EffectiveStatus()` returns `ChatStatusOpen` when `chat_status` key is absent from Metadata. No data migration.

**Rationale**: Constitution Principle 7 (AutoMigrate only) and Principle 8 (JSONB flexibility) support this. Existing contacts simply don't have the key — they default to `open` and behave exactly as before. Only NEW incoming messages will trigger `pending`.

**Verification**: `SELECT count(*) FROM contacts WHERE metadata->>'chat_status' IS NULL` will return all pre-existing contacts. The `GetMessages` guard checks `contact.EffectiveStatus() == ChatStatusPending` — which returns `open` for these, so they pass through unaffected.

---

## R8: Auto-revert worker — how to implement the inactivity timeout

**Decision**: Implement a background goroutine worker (like the existing SLA processor) that runs every 5 minutes, queries all `open` conversations whose `last_message_at` exceeds the configured timeout, and reverts them to `pending`.

**Rationale**:
- The project already has background workers: `sla_processor.go` (SLA timers), `worker.go` (campaign/message queue workers). This follows the same pattern.
- The worker reads `Organization.Settings["chat_inactivity_timeout_hours"]` (JSONB, default 24h) — same pattern as `mask_phone_numbers` in settings.
- A 5-minute poll interval is sufficient (sub-minute precision is not needed for a 24h timeout).
- The worker is started in `main.go` alongside other periodic workers.

**Implementation sketch**:
```go
// internal/handlers/chat_inactivity_worker.go
func (a *App) StartChatInactivityWorker(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    for {
        select {
        case <-ticker.C:
            a.revertInactiveChats()
        case <-ctx.Done():
            return
        }
    }
}

func (a *App) revertInactiveChats() {
    // For each org, read timeout from settings
    // Query: SELECT * FROM contacts WHERE metadata->>'chat_status' = 'open'
    //   AND assigned_user_id IS NOT NULL
    //   AND last_message_at < NOW() - INTERVAL 'X hours'
    // For each: clear assigned_user_id, clear collaborators, set pending,
    //   post system message, broadcast WS
}
```

**Alternatives considered**:
- Redis TTL keys: Rejected — adds complexity, doesn't integrate with GORM models naturally.
- Per-conversation goroutine timers: Rejected — doesn't scale to 10k+ conversations.
- PostgreSQL scheduled events (pg_cron): Rejected — violates Constitution Principle 7 (no raw SQL extensions).

---

## R9: Manager kick endpoint — permission and route design

**Decision**: Add `DELETE /api/contacts/{id}/collaborators/{user_id}` requiring `chat.collaborate:write` permission (same as join). This is distinct from the self-leave endpoint `DELETE /api/contacts/{id}/join`.

**Rationale**:
- Two separate DELETE endpoints with clear semantics: `/join` = self-leave (any authenticated collaborator), `/collaborators/{user_id}` = manager removes another.
- Permission: `chat.collaborate:write` — managers already have this; agents don't. This automatically enforces "managers can kick, agents cannot" without a new permission.
- The endpoint removes the specified user from `collaborators`, posts a system message ("🔔 {Manager} removed {User} from the conversation"), and broadcasts WS.

**Alternatives considered**:
- Overload `/join` with a target user_id body: Rejected — ambiguous REST semantics, harder to audit.
- New permission `chat.kick`: Rejected — unnecessary granularity; `chat.collaborate:write` already separates managers from agents.

---

## R10: Close-on-last-leave — when the owner leaves and no collaborators remain

**Decision**: When the primary owner triggers "Leave" (or is removed by a manager) AND no other collaborators remain, the conversation is set to `chat_status = "closed"` rather than being left in an orphaned `open` state.

**Rationale**:
- An `open` conversation with no `assigned_user_id` and no collaborators is logically dead — no one can interact with it.
- Setting it to `closed` makes it visually distinct in the UI (read-only) and prevents it from showing in the active queue.
- If the owner simply wants to release it back to the queue (not close), they should use "Unassign" (which sets it to `pending`), not "Leave".
- The distinction: "Leave" = "I'm done, close it" vs "Unassign" = "Put it back in the queue".

**Implementation**: In `LeaveChat` handler, after removing the user:
```go
// If the owner is leaving AND no collaborators remain → close
if isOwner && len(contact.GetCollaborators()) == 0 {
    contact.SetStatus(models.ChatStatusClosed)
    contact.AssignedUserID = nil
    // System message: "🔔 Conversation closed"
} else if isOwner {
    // Owner leaves but collaborators remain → promote first collaborator? Or leave open?
    // Decision: leave open with remaining collaborators. They can still reply.
    contact.AssignedUserID = nil
    // System message: "🔔 {Owner} left. {N} collaborators remain."
}
```

