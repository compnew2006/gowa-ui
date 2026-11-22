# Follow-ups parked from the 2026-11-21 remaining-fixes branch — file before merge

- **Branch:** `integration/remaining-fixes` @ `f4a3315` (39 commits over `db8c58b`).
- **Status key:** each item is one line — what it is, why it is parked, what unblocks it. None of these blocks the merge; none was fixed in the final fix wave.

## Parked items

1. **F1 — harden single-close `CloseChat` to the participant gate** (`internal/handlers/chat_lifecycle.go:379`, Task 14 finding): today any agent who can see a conversation can close it; bulk close already enforces the tight gate. Parked because it needs a user decision — do NOT fix silently. Unblocks: user decision on the gate (participant-only vs `chat:write`).
2. **F2 — template-read permission decision** (`ListTemplates` / `GetTemplate`, `internal/handlers/templates.go:55` / `:163`, Task 14 finding): the read endpoints have no permission check at all; only write/delete are gated. Parked because it needs a user decision — probably load-bearing for the chat TemplatePicker, which may read templates as an agent. Unblocks: user decision on which role/action may read templates.
3. **Active Contacts vs Decision 1 — inbound-only + exclusions in `queryContacts`** (`internal/handlers/widgets.go:890`, counting-definitions §2): the decided definition counts a contact as active only when it sent at least one inbound message in the window, excluding internal/group/newsletter contacts; the widget today counts any message in either direction with no exclusions. Parked as a separate follow-up task. Unblocks: a fix task adding the inbound-only condition and the exclusions.
4. **Decision 7 — Meta-approved template enforcement**: sending must be limited to Meta-approved templates, but the code has no template-status source to enforce against. Parked because it needs a Meta template-status sync or a status column first. Unblocks: template-status sync (or a `status` column) to enforce against.
5. **Per-route tab titles for ALL routes**: only the 5 quick-wins routes have titles (`docs/superpowers/plans/2026-11-21-frontend-quick-wins.md`); every other route still shows the default title. Parked as a follow-up. Unblocks: a pass over the remaining routes in `frontend/src/router/index.ts`.
6. **Task 15 live re-check with agent credentials against the deployed fixed build**: the live checks so far ran against the pre-fix build; the fixed build was never verified live with an agent login. Parked because it needs a deploy. Unblocks: deploy of this branch + agent credentials.
7. **WS-handler tab-total refresh**: the WebSocket handlers only mutate local rows, so a lifecycle action by *another* user leaves your tab badges stale until the next full fetch (same staleness the final fix wave fixed for your own actions, `frontend/src/stores/contacts.ts:refreshTabTotals`). Parked because it needs `refreshTabTotals` exported and debounced for remote events. Unblocks: export + debounce of the helper, then a call from the lifecycle WS handlers.

## Fixed in the final wave (for the record, not parked)

- **Final-review Important #1 — tab badges went stale after lifecycle actions**: `closeChat`, `releaseChat`, `reopenChat`, `bulkCloseChats`, `bulkReleaseChats` and `leaveChat` (when it closes) now call `refreshTabTotals()` before any fallible re-read (Task 7a pattern from `claimChat`), and `refreshTabTotals()` also re-reads the closed total (fourth `limit=1` call, `chat_status=closed`). Pinned by `frontend/src/stores/contacts.test.ts` ("contacts store lifecycle totals refresh (final review)").
