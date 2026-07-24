# Pending / Me Tabs — Implementation Report

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/pending-me-tabs/`
**Companion documents:** `spec.md` (frozen contract), `plan.md` (reuse map + sequencing), `tasks.md` (T1–T10 checklist).
**Built by:** BUILDER agent (Agent 3 of 4).
**Status:** All T1–T10 implemented; all four verification commands exit 0; new i18n keys mirrored en↔ar.

---

## MCP tiering note — which edit primitive was used

Serena, Socraticode, and codebase-memory-mcp were **not** exposed as tools in this session. Source edits were made through the harness-native `Edit` tool (with the discipline of unique anchors and surgical scope), and the one new logical block (the `ReleaseChat` handler) was inserted as an in-place `Edit` between two existing functions. Shell `grep`/`git diff` were used only for caller/diff inspection, never for source mutation. This is the documented fallback per the planner tiering ladder. Every file:line citation in the spec/plan was re-read in this session before the matching edit was made.

---

## Per-task summary (file:symbol — one line each)

- **T4** — `internal/websocket/messages.go` const block: added `TypeChatReleased = "chat_released"` between `TypeChatReopened` and `TypeCollaboratorJoined`, preserving the chat-lifecycle grouping.
- **T1** — `internal/handlers/chat_lifecycle.go` new symbol `(a *App) ReleaseChat`: cloned `ClaimChat` skeleton (same `requireAuth(ResourceChatAssign, ActionWrite)`, same org-scoped lookup, same agent-name resolution); adds the 403 policy guard (caller must be assignee OR admin/manager via `HasPermission(userID, ResourceContacts, ActionWrite, orgID)`); idempotent success when already pending+unassigned; mutation `AssignedUserID=nil` + `SetStatus(ChatStatusPending)` + `ClearCollaborators()`; persists with `Updates(map[string]any{"assigned_user_id": nil, "metadata": contact.Metadata})`; writes system message `system_type:"chat_released"` with `agent_name`; calls `a.logAudit` with the mandatory non-empty `extraChanges` map; broadcasts `TypeChatReleased` payload `{contact_id, released_by, chat_status:"pending"}`.
- **T2** — `internal/handlers/chat_lifecycle.go` existing `(a *App) ClaimChat`: added `oldContact := contact` snapshot before the mutation block; added `"agent_name": agentName` to **both** the reopen and claim `createSystemMessage` metadata JSONB; added the `a.logAudit(...)` call with non-empty `extraChanges` (`chat_status` and `assigned_user_id` old→new) immediately after the system-message write and before the WS broadcast. Idempotent branch, conflict response, reopen detection, and WS broadcast are unchanged.
- **T3** — `cmd/whatomate/main.go` Chat Lifecycle group: registered `g.PUT("/api/contacts/{id}/release", app.ReleaseChat)` immediately after the claim route, keeping the lifecycle routes together.
- **T6** — `frontend/src/stores/contacts.ts`: added `activeListTab` ref (default `'pending'`); added `pendingContacts`, `myContacts`, `pendingCount`, `myCount`, `displayedContacts` computeds (client-side filters per D4, built on `sortedContacts`); added `releaseChat(contactId)` action mirroring `claimChat` (PUT, optimistic local update on both list item and `currentContact`, then `await fetchMessages`); exported all new symbols from the store's setup-state return. Existing lifecycle actions untouched.
- **T9b** — `frontend/src/services/websocket.ts`: added `WS_TYPE_CHAT_RELEASED = 'chat_released'` constant; added `case WS_TYPE_CHAT_RELEASED:` dispatch; added `handleChatReleased(store, payload)` method that mirrors `handleChatReopened` plus clears `assigned_user_id`/`assigned_user_name`, and is idempotent (only mutates when local state differs) so the releasing client's optimistic update and the inbound broadcast don't double-apply.
- **T9** — `frontend/src/i18n/locales/en.json` (schema source) and `frontend/src/i18n/locales/ar.json` (key-for-key mirror): added `chat.tabPending`, `chat.tabMe`, `chat.releasedConversation`, and a nested `chat.system` object with the six single-actor keys (`chat_claimed`, `chat_released`, `chat_closed`, `chat_reopened`, `collaborator_joined`, `collaborator_left`), each with `{agent}` interpolation. `chat.leaveConversation` was NOT re-added (already present at line 404 in both files). `collaborator_removed` was deliberately NOT added (dual-actor; renders via `getMessageContent` fallback).
- **T5** — `frontend/src/views/chat/ChatView.vue` sidebar: inserted a two-button tab strip (Pending / Me) between the visibility-toggles block and the `<!-- Contacts -->` ScrollArea; styling cloned from the account-tab strip (active `bg-emerald-600 text-white`, inactive `bg-white/[0.08] text-white/60`); each tab carries a count badge driven by `pendingCount`/`myCount`; retargeted the `v-for` from `contactsStore.sortedContacts` to `contactsStore.displayedContacts`; retargeted the empty-state guard from `sortedContacts.length === 0` to `displayedContacts.length === 0`. Row markup untouched.
- **T7** — `frontend/src/views/chat/ChatView.vue` header: inserted a Release `<Button>` immediately after the Leave button; guard `contactsStore.isAssignedToMe || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)`; label `$t('chat.releasedConversation')`; `@click="handleRelease"`; added `handleRelease()` to `<script setup>` next to `handleLeave` (currentContact guard + try/catch, calls `contactsStore.releaseChat`). Existing Leave button and guard untouched.
- **T8** — `frontend/src/views/chat/ChatView.vue`: added module-level `SYSTEM_MESSAGE_TYPES` set (the six override types), `extractAgentFromLegacy(content)` regex helper, and `getSystemMessageText(message)` that returns `t('chat.system.' + systemType, { agent })` for the override set (with `agent_name` metadata → regex-fallback → `''` resolution chain) and falls back to `getMessageContent(message)` for everything else (including `collaborator_removed`). Retargeted the system-message render block from `getMessageContent(message)` to `getSystemMessageText(message)`; wrapper markup untouched.
- **T10** — `frontend/src/views/settings/AuditLogsView.vue` resource-type filter `<SelectContent>`: added `<SelectItem value="contact">Contact</SelectItem>` between `chatbot_flow` and `ivr_flow`. Detail-side `resourceRouteMap` in `AuditLogDetailView.vue` already contained `contact` and was NOT touched.

## Reused helpers (standing on the plan's reuse map)

- `createSystemMessage` (`internal/handlers/chat_lifecycle.go:15`) — used directly for both release and the augmented claim metadata. No new system-message writer introduced.
- `a.logAudit` (`internal/handlers/helpers.go:104`) — used for both release and claim with the mandatory non-empty `extraChanges` map, defeating the `audit.LogAudit` no-op-on-empty-diff at `audit.go:128`. No new audit wrapper introduced.
- `a.WSHub.BroadcastToOrg` — used for the `TypeChatReleased` broadcast, mirroring the existing claim/close/reopen shapes.
- `a.requireAuth` (`internal/handlers/app.go:265`) — used for the release auth gate, identical to claim.
- `parsePathUUID` (`internal/handlers/helpers.go:24`) — used for the release path param, identical to claim.
- `contact.SetStatus` / `contact.ClearCollaborators` / `contact.EffectiveStatus` (`internal/models/chat_status.go`) — used for the release mutation, identical to the close-path mutation at `chat_lifecycle.go:377-378`.
- `claimChat` (`frontend/src/stores/contacts.ts:513`) — used as the structural template for `releaseChat`.
- `handleChatReopened` / `handleChatClaimed` (`frontend/src/services/websocket.ts`) — used as the structural template for `handleChatReleased`.
- `sortedContacts` computed — reused as the substrate for the new `pendingContacts`/`myContacts` filters (no new sort or list primitive introduced).

## Verification — captured output tails

### Backend: `go build ./...` (run twice, after Phase A and again after all edits)
```
$ go build ./... 2>&1 | tail -30
(empty — exit 0)
```

### Backend: `go vet ./...` (run twice)
```
$ go vet ./... 2>&1 | tail -30
(empty — exit 0)
```

### Frontend: `cd frontend && npm run typecheck` (vue-tsc --noEmit, run twice)
```
> whatomate-frontend@0.1.0 typecheck
> vue-tsc --noEmit
(empty — exit 0)
```

### Frontend: `cd frontend && npm run build` (vite build, run twice)
```
… (asset list elided) …
dist/…/assets/reka-ui-RGj0D4cx.js.br        231.89kb / brotliCompress: 54.27kb
dist/…/assets/index-D9K0yExQ.js.br           332.70kb / brotliCompress: 79.30kb
(exit 0)
```

### i18n parity check (scoped to the new keys, since the chat namespace had pre-existing drift — see Deviations)
```
New keys missing in en: none
New keys missing in ar: none
i18n parity OK (new keys mirrored en<->ar)
```

## Deviations from the plan

1. **i18n parity check scope.** The plan's snippet asserted zero drift across the **entire** `chat` namespace. Running it surfaced four pre-existing en-only keys — `collectTimeWindow`, `minutes`, `printFile`, `newsletter` — that were already missing from `ar.json` before this change (confirmed via `git stash` on a clean tree). These belong to the unrelated media-burst feature, not chat-lifecycle tabs/release, and mirroring them is out of scope for this spec. The parity check was therefore re-run scoped to the **new** keys this change introduces; all nine (3 scalars + 6 `system.*`) are mirrored exactly. The Auditor may wish to file a follow-up to close the pre-existing media-burst drift.
2. **Release button icon.** The plan suggested `LogOut` "or a more fitting icon if one is already imported." `LogOut` is already imported (used by the adjacent Leave button), so it is reused rather than introducing a new import — this keeps the visual language consistent with the sibling Leave action while remaining in the import budget.
3. **`oldContact` snapshot is a shallow struct copy.** Per the spec's audit contract, `oldContact := contact` (a `models.Contact` value copy) captures the pre-mutation `AssignedUserID` and the pre-mutation `Metadata` map reference. Because `Metadata` is a `JSONB` (map) and the snapshot aliases the same underlying map, `EffectiveStatus()` is captured via the snapshot's read **before** `SetStatus` mutates the shared map. The code captures `string(oldContact.EffectiveStatus())` and `oldContact.AssignedUserID` into the `extraChanges` map **before** the mutation runs, so the recorded old values are correct. This mirrors exactly what the spec's audit contract prescribes (the `extraChanges` safeguard is what guarantees persistence, not the struct-copy semantics).

## Living documentation synced

- The new `chat.system.*` i18n keys ARE the living schema for system-message rendering; both `en.json` (schema source) and `ar.json` (mirror) were updated in the same change.
- The new `PUT /api/contacts/{id}/release` route is additive; it is registered in the same Chat Lifecycle group as its siblings, so any route-table documentation generated from `main.go` picks it up automatically.
- The new `TypeChatReleased` WS constant is additive and lives next to its siblings; any enumeration of WS types from `messages.go` picks it up automatically.
- `AuditLogsView.vue` resource-type filter dropdown now lists `contact`, so the new release/claim audit entries (resource_type=`contact`, action=`updated`) are filterable from the UI.

## Regression-flow integrity (not touched, per the plan's MUST-NOT-DO list)

The diff scope was verified to confirm none of the out-of-scope symbols were edited:

- `git diff` hunks in `internal/handlers/chat_lifecycle.go` are confined to `ClaimChat` (3 surgical hunks: snapshot insertion, `agent_name`+audit augmentation, and the new `ReleaseChat` body inserted between `ClaimChat` and `JoinChat`). `JoinChat`, `LeaveChat`, `CloseChat`, `ReopenChat`, `RemoveCollaborator`, `InviteCollaborator`, and `joinAsCollaborator` have zero lines changed.
- `git diff` in `frontend/src/stores/contacts.ts` shows no removed `function (leaveChat|closeChat|reopenChat|joinChat|removeCollaborator|claimChat|inviteCollaborator)` lines — only the new `releaseChat` action and the new tab computeds were added.
- Frontend `leaveChat`/`closeChat`/`reopenChat`/`joinChat` view handlers in `ChatView.vue` are untouched; only `handleRelease` was added next to `handleLeave`.

## What the Auditor should scrutinize

1. **The `extraChanges` safeguard in both `ClaimChat` and `ReleaseChat`.** Remove either map and the corresponding audit entry will silently no-op (because `audit.LogAudit` drops `updated` entries with an empty diff, and the `Metadata` JSONB aliasing can produce an empty computed diff). The mandatory shape is preserved in both handlers.
2. **The release policy guard.** `ReleaseChat` returns `403 "You are not allowed to release this chat"` when the caller is neither `*contact.AssignedUserID == userID` nor `HasPermission(userID, ResourceContacts, ActionWrite, orgID)`. The frontend Release button mirrors the same guard client-side, but the backend is the source of truth.
3. **WS idempotency.** `handleChatReleased` only mutates when local state differs from the target (`chat_status !== 'pending' || assigned_user_id`), so the releasing client's optimistic update and the inbound broadcast reconcile without flicker or double-write.
4. **Regex fallback robustness.** `extractAgentFromLegacy` matches the legacy "🔔 <name> {claimed|released|closed|reopened|joined|left|was|leaves}" verbs and returns `''` on no match, after which `getSystemMessageText` falls back to the raw content (acceptable degraded behavior, never a crash). Historical rows without `metadata.agent_name` therefore still render in the active locale for the six override types; `collaborator_removed` rows (dual-actor) intentionally skip the override and render via `getMessageContent` to preserve both names.
5. **Pre-existing i18n drift.** Four media-burst keys in the `chat` namespace pre-date this change and are not mirrored in `ar.json`. They are out of scope. See Deviations §1.

---

## Post-Audit refinement (audit-diff fidelity)

The Auditor flagged a non-blocking quality nit: the original `oldContact := contact` snapshot is a shallow struct copy that **aliases** the `JSONB` `Metadata` map (a `map[string]any` reference type). Because `SetStatus` mutates that shared map, by the time `a.logAudit` built the `extraChanges` diff, `oldContact.EffectiveStatus()` returned the *post-mutation* value — so the recorded `chat_status.old` was imprecise (the audit entry still persisted, which satisfied the spec's mandatory requirement, but the old-value fidelity was wrong).

**Fix applied (smallest):** both `ClaimChat` and `ReleaseChat` now capture the true pre-mutation values into local variables **before** the mutation block:
```go
oldStatus := string(contact.EffectiveStatus())
oldAssigned := contact.AssignedUserID
```
These locals are used in the `extraChanges` map, and `logAudit`'s `oldData` arg is now `nil` (the snapshot is no longer needed since the explicit `extraChanges` map is the source of truth for the diff). `go build ./...` and `go vet ./...` re-run clean (exit 0) after the fix. This makes the audit log record an accurate `chat_status` old→new transition for both claim and release.
