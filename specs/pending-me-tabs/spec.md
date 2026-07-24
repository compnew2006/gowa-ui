# Pending / Me Tabs — Specification

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/pending-me-tabs/`
**Status:** Drafted for Plan Reviewer, Builder, Auditor
**Original requirement (Arabic):** Add two tabs (`pending`, `me`) to the top of the chat sidebar. All chats start in `pending`; a chat moves to `me` only when the agent claims it via a button in the conversation area (which replaces message content with a claim screen). On claim, an audit-log entry and an in-conversation system message must record "user X claimed the conversation". When finished, the agent presses "leave conversation" and the chat returns to `pending` (with a system message recording that it was released). Frontend already ships i18n — all UI strings must be translated.

---

## Locked stack (detected from root markers)

- **Backend:** Go 1.25 (`module github.com/shridarpatil/whatomate`), fastglue router, GORM, PostgreSQL. Schema is managed by `AutoMigrate` — there are **no SQL migration files**, and this change introduces **no new columns** (status lives in `Contact.Metadata["chat_status"]` JSONB). Server entrypoint: `cmd/whatomate/main.go` (route registration in `setupRoutes`).
- **Frontend:** Vue 3 `<script setup>` + TypeScript + Pinia + vue-i18n at `frontend/`. The i18n schema source is `frontend/src/i18n/locales/en.json` (top-level `chat` namespace); `ar.json` must mirror key-for-key.
- **Package manager:** npm (`frontend/package.json`, `npm run build`).
- **Real build / verify commands** (read from `frontend/package.json` scripts):
  - Backend: `go build ./...` and `go vet ./...`.
  - Frontend bundle: `cd frontend && npm run build` (which runs `vite build` only — type-checking is **not** part of `build`; it lives in a separate `npm run typecheck` script that runs `vue-tsc --noEmit`). The Builder and Auditor must run **both** `npm run build` and `npm run typecheck` to guarantee the bundle compiles **and** the types check.

## MCP tiering note

Serena, Socraticode, and codebase-memory-mcp were **not** exposed as tools in this session. Source was therefore read through the harness-native `Read`, `Grep`, and `Glob` tools, with shell `grep` used only as a corroborating scan. Every file:line citation below was read through `Read` in this session — none is from memory. This is the documented fallback per the planner tiering ladder; confidence is high because all anchors were directly inspected, not inferred from a graph.

## Goal

Add a `pending` / `me` tab strip to the chat sidebar, where `pending` shows unassigned conversations and `me` shows conversations assigned to the current agent. Movement between the two is driven by an existing claim affordance (claim moves pending → me) and a new **release** affordance (release moves me → pending). Both transitions emit a system message into the conversation timeline and an audit-log entry recording the actor. All UI text — tab labels, button labels, and every system-message string — is translated into English and Arabic.

## Actors and permission matrix

The codebase already gates chat actions by permission constants `models.ResourceChatAssign` (`chat.assign`) and `models.ResourceChatCollaborate` (`chat.collaborate`), with `models.ResourceContacts` (`contacts`) acting as the admin/manager override (see `contacts.go` and `chat_lifecycle.go`). The new release flow slots into the same matrix.

| Actor                                  | See pending tab | See me tab | Claim (pending → me) | Release (me → pending) | View messages of unassigned chat |
|----------------------------------------|:--------------:|:----------:|:--------------------:|:----------------------:|:--------------------------------:|
| Agent (only `chat.assign:write`)       | yes            | yes (own)  | yes                  | yes (if assigned to me) | no — claim screen replaces content |
| Collaborator (`chat.collaborate:write`)| yes            | yes (own)  | no — joins instead   | no — uses existing leave | yes (as collaborator)            |
| Admin / Manager (`contacts:write`)     | yes            | yes (own)  | yes (ghost-claim)    | yes (ghost-release)     | yes — ghost view, no claim screen |

The admin/manager "ghost" behavior is already established for claim, leave, and close (see `LeaveChat` ghost-exit branch at `chat_lifecycle.go:357-363` and the frontend `isAdminOrManager` computed at `contacts.ts:200-202`). Release reuses the same ghost semantics: an admin releasing a chat they do not own records the release but is not treated as a participant.

## Public contracts

### REST routes

**New route.** `PUT /api/contacts/{id}/release` — handler `App.ReleaseChat`. Auth: `a.requireAuth(r, models.ResourceChatAssign, models.ActionWrite)`. This mirrors `ClaimChat` (`PUT /api/contacts/{id}/claim`, `chat_lifecycle.go:39`, same permission) exactly so the auth policy and error envelope shape are identical to the existing claim route.

- **Inputs:** path param `id` (contact UUID).
- **Outputs:** success envelope `{ contact_id, released: true, chat_status: "pending" }`.
- **Errors:** `404` "Contact not found" (org-scoped miss); `403` via `requireAuth`; `403` "You are not allowed to release this chat" when the caller is neither the assignee nor an admin/manager (status code chosen to match `requireAuth`'s forbidden semantics; the body shape still matches the codebase's `SendErrorEnvelope` envelope).
- **Side effects:** sets `AssignedUserID = nil`, calls `contact.SetStatus(models.ChatStatusPending)`, calls `contact.ClearCollaborators()`, persists via GORM `Updates`, writes a system message with `metadata.system_type = "chat_released"` and `metadata.agent_name = <fullName>`, writes an audit-log entry via `a.logAudit` with a non-empty `extraChanges` safeguard, and broadcasts a WebSocket message of type `chat_released`.

**Existing route, augmented.** `PUT /api/contacts/{id}/claim` — handler `App.ClaimChat`. Behavior is unchanged for callers, but the handler now additionally (a) records the agent's `FullName` into the system-message metadata as `agent_name`, and (b) writes an audit-log entry via `a.logAudit` with a non-empty `extraChanges` safeguard. The current `ClaimChat` (`chat_lifecycle.go:39-130`) does neither today.

### WebSocket events

**New event type.** `chat_released`, constant `TypeChatReleased = "chat_released"` added to `internal/websocket/messages.go` next to the existing `TypeChatClaimed` / `TypeChatClosed` / `TypeChatReopened` block at lines 67-71. Payload shape mirrors `TypeChatClaimed` (`chat_lifecycle.go:120-129`): `{ contact_id, released_by, chat_status: "pending" }`. The frontend constant `WS_TYPE_CHAT_RELEASED = 'chat_released'` is added to `frontend/src/services/websocket.ts` next to the existing block at lines 72-76.

The existing `chat_claimed`, `chat_closed`, `chat_reopened`, `collaborator_joined`, `collaborator_left` event types are unchanged; release simply adds a sixth.

### Audit log

Release and claim both call `a.logAudit(orgID, userID, "contact", contactID, models.AuditActionUpdated, &oldContact, &contact, extraChanges...)` (the wrapper is at `internal/handlers/helpers.go:104`). Because `audit.LogAudit` no-ops when action is `updated` **and** the diff is empty, the Builder **must** pass a non-empty `extraChanges` map (e.g. `{"chat_status": {"old": "open", "new": "pending"}, "assigned_user_id": {"old": <uuid>, "new": nil}}`) so the entry actually persists. The audit entry's `action` stays within the codebase's strict `created|updated|deleted` enum — it is recorded as `updated`, which the existing `AuditLogsView.vue` action filter (lines 148-157) already surfaces.

### Frontend store contract

The Pinia store `useContactsStore` (`frontend/src/stores/contacts.ts`) gains three new pieces of state:

- A reactive tab selector `activeListTab: Ref<'pending' | 'me'>` defaulting to `'pending'`.
- Two computeds, `pendingContacts` (filter: `chat_status === 'pending' && !assigned_user_id`) and `myContacts` (filter: `assigned_user_id === authStore.user?.id`).
- A `displayedContacts` computed that returns `pendingContacts` or `myContacts` based on `activeListTab`, and a count pair (`pendingCount`, `myCount`) for the tab badges.
- An action `releaseChat(contactId: string)` that calls `api.put('/contacts/${contactId}/release')`, updates the contact's local status to `pending` and clears `assigned_user_id` / `assigned_user_name`, then re-fetches messages for the open contact so the new system message renders immediately. This mirrors the existing `claimChat` pattern at `contacts.ts:513-531`.

The existing `claimChat`, `leaveChat`, `closeChat`, `joinChat`, `reopenChat` actions are unchanged.

### Sidebar UI contract

A two-button tab strip is inserted into `frontend/src/views/chat/ChatView.vue` between the visibility-toggles block (which ends at line 2024) and the `<!-- Contacts -->` ScrollArea (which starts at line 2026). The strip is styled like the existing account-tab strip (active `bg-emerald-600`, inactive `bg-white/[0.08]`) and renders each tab with a count badge. The `v-for` at line 2030 currently iterates `contactsStore.sortedContacts`; it is changed to iterate `contactsStore.displayedContacts` so the active tab drives the rendered list. No other list-render logic is touched.

### Release button contract

A "Release" button is added to the chat header near the existing Leave button (line 2169). It is shown when `contactsStore.isAssignedToMe` is true (or the user is admin/manager and not on a pending/closed chat, mirroring the Leave button's guard at 2169). Its label is `$t('chat.releasedConversation')`. Clicking it calls a new `handleRelease()` handler in `ChatView.vue` that invokes `contactsStore.releaseChat(currentContact.id)`.

### System-message i18n contract

A new function `getSystemMessageText(message)` in `ChatView.vue` returns the localized system-message string. When `message.metadata.system_type` is present AND that type is in the i18n override set (see below), it returns `$t('chat.system.' + system_type, { agent: message.metadata.agent_name || <regex-fallback from message.content> })`; otherwise it falls back to the existing `getMessageContent(message)`. The system-message render block at lines 2453-2462 is changed to call `getSystemMessageText(message)` instead of `getMessageContent(message)`.

The override set is the six single-actor system types: `chat_claimed`, `chat_released`, `chat_closed`, `chat_reopened`, `collaborator_joined`, `collaborator_left`. The seventh existing type, `collaborator_removed`, is deliberately **excluded** from the override path because its message carries two actors (the removed user as `agent_id` and the manager as `removed_by`, per `chat_lifecycle.go:512-518`) and its legacy content is "<target> was removed by <manager>" — a single-`{agent}` interpolation would silently drop the manager. `collaborator_removed` therefore continues to render via the `getMessageContent` fallback, preserving both names. (If full localization of that type is later desired, it needs its own `{agent}` + `{by}` interpolation key, which is out of scope here.)

The agent name is stored on the message itself as `metadata.agent_name` at creation (so the value is durable and locale-independent). For historical rows that predate this field, the regex fallback extracts the name from the legacy "🔔 <name> <verb>" content strings.

## Data model

No schema change. Status already lives in `Contact.Metadata["chat_status"]` with values `pending` / `open` / `closed` (see `internal/models/chat_status.go:9-13`). Release sets the value back to `pending` via `contact.SetStatus(models.ChatStatusPending)` (`chat_status.go:39-44`) and clears the collaborators array via `contact.ClearCollaborators()` (`chat_status.go:158-163`). The system message is a normal `models.Message` row with `MessageType = text`, `Direction = outgoing`, `metadata.is_system_message = true` and a `system_type` discriminator — exactly the shape `createSystemMessage` already produces (`chat_lifecycle.go:15-34`). AutoMigrate handles anything needed; there is nothing for the Builder to migrate by hand.

## State machine

```
                claim                      close
   PENDING ─────────────────► OPEN ─────────────────► CLOSED
      ▲                         │                        │
      │ release (NEW)           │ release (NEW)          │ reopen
      │                         ▼                        │
      └─────────────────────────┘                        │
                           ▲                             │
                           └─────────────────────────────┘
```

Release is a **new** transition out of `OPEN` (and out of the assigned-but-not-closed state) that returns the conversation to `PENDING` and unassigns the owner. It is distinct from the existing last-participant `LeaveChat` path, which closes the conversation (`chat_lifecycle.go:374-403`). The two flows must not collide: release never closes, and the existing leave/close/collaborate/reopen flows are untouched.

## Approved product decisions (do not re-litigate)

- **D1 — Leave → release to Pending.** The "leave conversation" gesture in the requirement maps to a new `release` action that returns the chat to `pending`, **not** to the existing close. Existing Close (last-participant leave) and Collaborate (join/leave as collaborator) flows are untouched. New endpoint `PUT /contacts/{id}/release`.
- **D2 — Tab membership.** The `pending` tab shows unassigned chats only (`chat_status === 'pending' && !assigned_user_id`). The `me` tab shows chats assigned to the current user (`assigned_user_id === current user id`). Chats assigned to **other** agents appear in neither tab (they are reached via search or the admin's full view, unchanged).
- **D3 — Full i18n of system messages.** Every system message is localized by `system_type`, with `{agent}` interpolation. The agent's display name is stored in `message.metadata.agent_name` at creation. Old rows without `agent_name` fall back via regex on the legacy "🔔 <name> ..." content.
- **D4 — Client-side filtering.** The tab membership is computed in the Pinia store from the already-loaded contact list. There is **no** backend list endpoint change, no new query parameter, no server-side pagination impact.

## Non-functional requirements

- **Idempotency.** Releasing an already-pending chat must be a safe no-op-style success (the contact is already in the target state), mirroring how `ClaimChat` handles the idempotent "already assigned to self" case at `chat_lifecycle.go:78-88`.
- **Concurrency.** The release path must read-modify-write `contact.Metadata` in a single GORM `Updates` call (the same pattern as `chat_lifecycle.go:379-382`), so a concurrent claim cannot observe a half-written metadata blob.
- **Observability.** Both release and claim must produce a persistent audit-log entry; the `extraChanges` safeguard is mandatory because `audit.LogAudit` silently drops `updated` entries with empty diffs.
- **Performance budget.** Tab switching is a pure client-side computed; it adds zero network round-trips. Release adds one `PUT` plus one messages re-fetch — the same cost as the existing claim flow, which is the established budget for this class of action.

## Edge cases and failure modes

- **Release by a non-owner non-admin.** Must return `403 "You are not allowed to release this chat"`. (Status 403, not 409, because this is a policy denial rather than a state conflict — `ClaimChat`'s 409 is specifically for "already assigned to another agent", which is a different condition. The body still uses the codebase's standard `SendErrorEnvelope` shape.)
- **Release of a closed chat.** Allowed only for admin/manager (the same actors who can reopen); agents see the button only when `chat_status === 'open'`. A closed chat released by an admin transitions `closed → pending` and is unassigned.
- **Release with active collaborators.** `ClearCollaborators()` removes them all and the system message records the release; this matches the close path's behavior at `chat_lifecycle.go:377`.
- **System message with no `agent_name`.** The `getSystemMessageText` regex fallback must never throw; if both `agent_name` and the regex fail, it falls back to the raw `message.content`.
- **WS delivery to the releasing agent's own client.** The releasing agent's local state is already updated optimistically in `releaseChat`; the inbound `chat_released` broadcast must reconcile, not duplicate, the change. The existing `chat_claimed` handler in `websocket.ts` is the precedent.
- **i18n key missing for a new `system_type`.** If a future system type has no key, `$t` returns the key string; this is acceptable degraded behavior, never a crash.

## Acceptance criteria

1. The chat sidebar renders a two-tab strip (`pending`, `me`) with live count badges; selecting a tab filters the rendered list without a network call.
2. A chat in `pending` moves to the `me` tab of the claiming agent after claim; the conversation timeline shows a localized "X claimed this conversation" system message; an audit-log entry for resource `contact`, action `updated`, actor = claimer appears at `/settings/audit-logs`.
3. A chat in the `me` tab moves back to `pending` after the agent presses the Release button; the timeline shows a localized "X released this conversation" system message; an audit-log entry for resource `contact`, action `updated`, actor = releaser appears at `/settings/audit-logs`.
4. The existing Close, Reopen, Join, and Collaborator-Leave flows behave exactly as before (regression check).
5. All new UI strings (tab labels, release button, system messages) are present in `en.json` and mirrored key-for-key in `ar.json`.
6. Every system message renders in the active locale, including historical rows that lack `agent_name`.
7. The `chat_released` WebSocket event reaches other connected clients and reconciles their sidebar state.
8. `go build ./...`, `go vet ./...`, `cd frontend && npm run build`, and `cd frontend && npm run typecheck` all pass cleanly.

## Out of scope

- Any backend change to the contacts **list** endpoint (filtering is client-side per D4).
- Any change to the existing Close, Reopen, Join, or collaborator Leave flows.
- Any DB column addition or SQL migration file (status is JSONB; AutoMigrate suffices).
- Server-side pagination or sorting changes for the sidebar.
- Bulk release or bulk reassignment UI.
- Changes to the `audit_logs` **action** enum (it stays `created|updated|deleted`; release and claim are `updated`).
- Renaming or restructuring the existing top-level `chat` i18n namespace.
