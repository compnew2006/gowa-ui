# Pending / Me Tabs — Implementation Plan

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/pending-me-tabs/`
**Companion documents:** `spec.md` (the what), `tasks.md` (the sequenced work).

This plan is the *how*. It assumes the spec is already agreed and does not re-litigate any product decision (D1-D4 in `spec.md`). Every symbol referenced below was read in this session through the native `Read` tool, with file:line citations the Builder can open directly. No code is written here — only the architectural sequencing, the reuse map, the blast radius, and the risks.

## Guiding architecture

The codebase already contains the entire claim/collaborate/close lifecycle, the system-message channel, the WebSocket broadcast layer, the audit-log wrapper, and the frontend store/view wiring for all of it. The single missing piece is the **release** transition (`open → pending` with unassignment), plus the UI affordances (tab strip, release button) and the i18n completeness (system messages). Because every primitive the Builder needs already exists, the right move is to **clone-and-adapt the existing patterns** rather than introduce new abstractions. The plan therefore has very high reuse density and very low new-surface area.

The layered shape of the change is: backend new-transition + audit augmentation → route registration → WS constant → frontend store action → frontend view UI + i18n → optional audit-filter polish. Each layer compiles and tests stay green at its boundary, so the project is shippable at every task boundary.

## Reuse-first map (read these before writing anything)

Every helper the Builder needs is already present. Reuse is a hard gate: do **not** introduce a new system-message writer, a new audit wrapper, a new WS broadcast helper, or a new contact-filter primitive.

### Backend

- **System message writer.** `(a *App) createSystemMessage(orgID, contactID uuid.UUID, content string, metadata models.JSONB)` at `internal/handlers/chat_lifecycle.go:15-34`. It already sets `is_system_message = true`, `MessageType = text`, `Direction = outgoing`. The Builder calls it directly; do not duplicate.
- **Status model.** `models.ChatStatus` constants (`Pending`/`Open`/`Closed`) and the `EffectiveStatus()` / `SetStatus()` / `ClearCollaborators()` / `GetCollaborators()` / `IsCollaborator()` / `HasParticipants()` methods at `internal/models/chat_status.go:7-168`. Release uses `SetStatus(Pending)` + `ClearCollaborators()` exactly as the close path does at `chat_lifecycle.go:377-378`.
- **Audit wrapper.** `(a *App) logAudit(orgID, userID uuid.UUID, resourceType string, resourceID uuid.UUID, action models.AuditAction, oldData, newData any, extraChanges ...map[string]any)` at `internal/handlers/helpers.go:104-106`. Variadic `extraChanges` — pass one non-empty map to defeat the `audit.LogAudit` no-op-on-empty-diff behavior. This is the only correct way to write an audit entry from a handler.
- **Existing handler to clone for `ReleaseChat`.** `App.ClaimChat` at `internal/handlers/chat_lifecycle.go:39-130` is the structural template: same auth (`requireAuth(ResourceChatAssign, ActionWrite)`), same org-scoped contact lookup, same `parsePathUUID(r, "id", "contact")`, same agent-name resolution (`a.DB.First(&agent, "id = ?", userID)` then `agent.FullName`), same `a.WSHub.BroadcastToOrg(...)` shape. The Builder should mirror this skeleton and swap the mutation direction (unassign + set pending instead of assign + set open).
- **Existing last-participant close to NOT collide with.** `App.LeaveChat` at `chat_lifecycle.go:328-450`, specifically the `len(collaborators) == 0` branch at 374-403 that closes the conversation. Release is a **different** action with a different route; do not modify `LeaveChat`.
- **WebSocket types.** `TypeChatClaimed`/`TypeChatClosed`/`TypeChatReopened`/`TypeCollaboratorJoined`/`TypeCollaboratorLeft` constants at `internal/websocket/messages.go:67-71`. Add `TypeChatReleased = "chat_released"` immediately after line 71.
- **Route registration.** Chat-lifecycle routes are grouped at `cmd/whatomate/main.go:755-759`. Register the new release route in the same group: `g.PUT("/api/contacts/{id}/release", app.ReleaseChat)`.

### Frontend

- **Store actions to clone for `releaseChat`.** `claimChat` at `frontend/src/stores/contacts.ts:513-531` is the template: `api.put('/contacts/${id}/...')`, optimistic local update of `chat_status`/`assigned_user_id`/`assigned_user_name`, then `await fetchMessages(contactId)` to surface the system message. `closeChat` (557-569) and `reopenChat` (574-584) reinforce the same pattern.
- **Existing computeds the tab logic builds on.** `isAssignedToMe` (`contacts.ts:170-174`), `isPendingClaim` (151-157), `canManageAllChats` (166-168), `isAdminOrManager` (200-202), `isLastParticipant` (207-211), and the `Contact` interface's `chat_status` / `assigned_user_id` / `assigned_user_name` / `collaborators` fields. The new `pendingContacts` / `myContacts` / `displayedContacts` computeds read the same fields.
- **Existing list iteration to retarget.** `v-for="contact in contactsStore.sortedContacts"` at `frontend/src/views/chat/ChatView.vue:2030`. Change the source to `contactsStore.displayedContacts`; leave the row markup untouched.
- **Existing tab-style UI to clone for the strip.** Account-tab strip styling (active `bg-emerald-600`, inactive `bg-white/[0.08]`) referenced at `ChatView.vue:2319-2339`. The Builder lifts the class pattern, not the component.
- **Existing button to clone for the Release button.** The Leave button at `ChatView.vue:2169-2173` already establishes the exact guard expression (`(contactsStore.isCollaborator && !contactsStore.isAssignedToMe) || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)`), the `<Button variant="ghost" size="sm">` styling, and the icon+label composition. The Release button sits next to it with a guard of `contactsStore.isAssignedToMe || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)`.
- **System-message render block to retarget.** `ChatView.vue:2453-2462`. Change `getMessageContent(message)` to `getSystemMessageText(message)`; leave the wrapper markup untouched.
- **WS constants block to extend.** `frontend/src/services/websocket.ts:72-76` — add `WS_TYPE_CHAT_RELEASED = 'chat_released'` next to `WS_TYPE_CHAT_CLAIMED`.
- **i18n namespace.** Top-level `chat` namespace at `frontend/src/i18n/locales/en.json:296`. The keys `chat.leaveConversation`, `chat.chatNotClaimed`, `chat.claimedSuccessfully` already live there (verified by direct read); add the new keys as siblings.

## Verified corrections to the original task brief

The Orchestrator's brief was accurate on substance but had three details that direct reads corrected. The Builder must follow the corrected versions, not the brief's:

1. **System-message `system_type` values.** The brief's T9 loosely listed `claimed / released / closed / reopened / joined / left / added / ownership_transferred`. The actual values in the codebase (read at `internal/handlers/chat_lifecycle.go:112, 116, 252, 304, 386, 416, 441, 515, 594, 662` and `chatbot_processor.go:209, 231`) are: `chat_claimed`, `chat_reopened`, `chat_closed`, `collaborator_joined`, `collaborator_left`, `collaborator_removed`. There is no `added` or `ownership_transferred` or bare `joined`/`left`. The new release adds `chat_released`. The i18n keys must therefore be `chat.system.chat_claimed`, `chat.system.chat_released`, `chat.system.chat_closed`, `chat.system.chat_reopened`, `chat.system.collaborator_joined`, `chat.system.collaborator_left` — **six** keys total. The seventh existing type, `collaborator_removed`, is deliberately excluded from i18n override because its message carries two actors (`agent_id` = removed user, `removed_by` = manager, per `chat_lifecycle.go:512-518`) and a single-`{agent}` interpolation would drop the manager's name. `collaborator_removed` keeps rendering via the `getMessageContent` fallback.
2. **`chat.leaveConversation` already exists.** Both `en.json:404` and `ar.json:404` (mirrored) already define it. T9 must NOT re-add it; only `tabPending`, `tabMe`, `releasedConversation`, and the seven `chat.system.*` keys are new.
3. **Audit detail-side resource map already done.** `resourceRouteMap` in `frontend/src/views/settings/AuditLogDetailView.vue:36` already contains `contact: (id) => /chat?contact=${id}`. The optional T10 therefore only needs to add `contact` to the **resource-type filter dropdown** at `AuditLogsView.vue:164-173` (which currently lists `account`, `ai_context`, `campaign`, `chatbot_settings`, `chatbot_flow`, `ivr_flow`, `keyword_rule`, `team`, `template` and is missing `contact`). The detail-side work is already complete.

## Sequenced task list with rationale

The full checklist form lives in `tasks.md`; the rationale for the ordering is here.

**Phase A — Backend primitives (T1, T2, T4).** T4 (the WS constant) is a one-liner with no dependencies and is consumed by T1, so it goes first or in parallel. T1 (the `ReleaseChat` handler) is the substantive new code and depends on T4's constant and on the existing `createSystemMessage` / `logAudit` / `SetStatus` / `ClearCollaborators` primitives. T2 (audit + `agent_name` augmentation of the existing `ClaimChat`) is a small, surgical edit to a function the Builder is already studying for T1; doing it in the same phase keeps the claim/release pair symmetric. After Phase A, `go build ./...` and `go vet ./...` must pass.

**Phase B — Backend wiring (T3).** Register the route. Trivial, but isolated so the diff is reviewable. After T3 the endpoint is callable.

**Phase C — Frontend store (T6).** `releaseChat` action + `pendingContacts` / `myContacts` / `displayedContacts` / `activeListTab`. This is the data substrate for every view change, so it precedes the view. After T6, `npm run typecheck` must still pass (the new computeds and action are exercised by nothing yet, but they must type-check).

**Phase D — Frontend view (T5, T7, T8).** T5 (tab strip) consumes `displayedContacts`. T7 (release button) consumes `releaseChat`. T8 (system-message i18n) consumes nothing new from the store but depends on the i18n keys. These three are independent of each other and can be done in any order, but all three depend on T6.

**Phase E — i18n (T9).** Strictly, T9 must land **before** T8 is verified, because T8 calls `$t('chat.system.' + system_type)`. In practice T9 is done in lockstep with T8: write the keys first, then point T8 at them. `en.json` is the schema source; `ar.json` is the key-for-key mirror.

**Phase F — Optional polish (T10).** Add `contact` to the audit resource-type filter dropdown. Cosmetic; can be deferred.

## Blast radius

The change touches a small, well-contained set of files. The risk of regression is correspondingly small, but the **existing** flows that share these files must be regression-checked.

**Backend files touched:**
- `internal/handlers/chat_lifecycle.go` — add `ReleaseChat` (new function, no edit to existing functions except the small T2 augmentation of `ClaimChat` at lines ~101-117 to add `agent_name` and the `logAudit` call).
- `internal/websocket/messages.go` — add one constant (after line 71).
- `cmd/whatomate/main.go` — add one route (after line 759).

**Frontend files touched:**
- `frontend/src/stores/contacts.ts` — add `activeListTab`, three computeds, one action. No edits to existing actions.
- `frontend/src/views/chat/ChatView.vue` — insert tab strip (between 2024 and 2026), insert release button (near 2169), add `getSystemMessageText` and retarget line 2460. No structural rewrite.
- `frontend/src/services/websocket.ts` — add one constant (after line 76) and one handler case in `handleMessage` for `WS_TYPE_CHAT_RELEASED` that updates local contact status, mirroring the existing `WS_TYPE_CHAT_CLAIMED` handler.
- `frontend/src/i18n/locales/en.json` and `ar.json` — add keys under `chat`.

**Existing flows that must keep working (regression set):**
- Claim (pending → open, assigned to me) — `ClaimChat` + `claimChat`.
- Close (last participant leaves) — `LeaveChat` last-participant branch + `closeChat`.
- Reopen (closed → open) — `ReopenChat` + `reopenChat`.
- Collaborator join/leave — `joinChat` / `leaveChat` and the `collaborator_joined`/`collaborator_left` system messages.
- Admin/manager ghost view and ghost-exit — `isAdminOrManager` + the ghost-exit branch of `LeaveChat`.
- Audit log list and detail views — must still render; the new entries appear as `resource_type = contact`, `action = updated`.

## Risks and mitigations

- **Risk: silent audit no-op.** `audit.LogAudit` drops `updated` entries whose diff is empty (per the spec's audit contract). If the Builder forgets the `extraChanges` safeguard, release and claim will appear to succeed but leave no audit trail — a subtle, late-discovered bug. **Mitigation:** the spec names the exact shape of the `extraChanges` map, and T1/T2 acceptance criteria require the Builder to verify an entry appears at `/settings/audit-logs` after each action. The Auditor will check this directly.
- **Risk: release collides with close semantics.** The existing `LeaveChat` last-participant path closes the conversation. If the Builder confuses the two flows, release could accidentally close, or close could accidentally unassign-without-close. **Mitigation:** release is a brand-new handler and route; `LeaveChat` is explicitly out of scope and must not be edited. The state-machine diagram in `spec.md` makes the distinction unambiguous.
- **Risk: optimistic-update race with WS broadcast.** `releaseChat` updates local state, then the `chat_released` broadcast arrives and may re-apply the same change. **Mitigation:** the WS handler must be idempotent (set status to pending + clear assignment only if currently different); the existing `chat_claimed` handler is the precedent and already behaves this way.
- **Risk: historical system messages lack `agent_name`.** Rows written before this change have only the legacy "🔔 <name> ..." content. **Mitigation:** `getSystemMessageText` carries a regex fallback; if both `agent_name` and the regex fail, it returns the raw content. Acceptable degraded behavior, never a crash.
- **Risk: i18n key drift between `en.json` and `ar.json`.** vue-i18n silently falls back to the key string when a translation is missing, which hides the gap in dev. **Mitigation:** T9 acceptance criterion requires a key-for-key parity check (a tiny script that diffs the `chat` namespace across both files) before the task is marked done.
- **Risk: type-check passes locally but the bundle breaks (or vice versa).** `npm run build` runs only `vite build` (no `vue-tsc`); `npm run typecheck` runs only `vue-tsc --noEmit`. Running just one can miss the other's failure. **Mitigation:** the verification block in `tasks.md` requires **both** commands, and the Builder's `implementation_report.md` must capture the tail of each.
- **Risk: closed-chat release by an agent.** The spec allows it for admin/manager only. If the guard on the Release button is wrong, an agent could release a closed chat and lose the closed state. **Mitigation:** the button guard in T7 mirrors the existing Leave button's guard (`!isPendingClaim && !isChatClosed`), and the backend `ReleaseChat` enforces the same policy server-side, so a UI bug cannot bypass the policy.

## Backward compatibility

No public API is removed or renamed. The new `PUT /contacts/{id}/release` is additive. The `chat_released` WS type is additive; clients that do not handle it simply do not react, which is safe because the releasing client's optimistic update plus the next list refresh will reconcile state anyway. The `agent_name` metadata field is additive on system messages; old clients ignore it. The new i18n keys are additive. There is no database migration, no column rename, no breaking change to any existing endpoint or event.

## What the Builder must NOT do

- Do not edit `LeaveChat`, `CloseChat`, `ReopenChat`, `joinChat`, `leaveChat`, `closeChat`, `reopenChat`, or `removeCollaborator`. They are out of scope.
- Do not add a new system-message helper, audit wrapper, or WS broadcast helper. Use `createSystemMessage`, `a.logAudit`, and `a.WSHub.BroadcastToOrg` directly.
- Do not change the audit `action` enum. Release and claim are recorded as `updated`.
- Do not add a backend list endpoint or a new query parameter. Filtering is client-side (D4).
- Do not restructure the `chat` i18n namespace. Add keys as siblings of the existing ones.
- Do not skip `npm run typecheck`. It is a separate script from `npm run build` and both must pass.
