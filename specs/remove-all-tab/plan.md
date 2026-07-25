# Remove the "All" Tab — Plan

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/remove-all-tab/`
**Companion documents:** `spec.md` (frozen contract — the what), `tasks.md` (T1–T6 ordered checklist — the work).
**Scope verdict:** Frontend-only, two files plus two i18n locale files. No backend change.

---

## MCP tiering note (which tools produced this plan)

Serena, Socraticode, and codebase-memory-mcp were **not** exposed as tools. graphify (`graphify-out/graph.json`) was live and was consulted, but it does not resolve Vue `<script setup>` local symbols (`graphify explain "canSeeAllTab"` and `graphify affected "activeListTab"` both returned "No node matching"/"No unique node match"). The blast radius below was therefore mapped with the documented fallback ladder: harness-native `Read` for the two source files and the two locale files, harness-native `Grep` for the repo-wide reference scan, and shell `grep` as a corroborating pass. graphify was used for the one question it *can* answer authoritatively — whether the All tab reaches a dedicated backend route — and `graphify path "ChatView" "ListContacts"` returned "No path found", confirming the All tab adds no backend edge. Every file:line citation in this plan was read through `Read` in this session.

## Deliberation summary (the design judgment, not the steps)

Three voices competed and were resolved before any line of this plan was written.

The first voice said: the All tab is a privacy leak, rip it out everywhere, including the backend. The evidence refutes this. `internal/handlers/contacts.go:100-227` shows `ListContacts` already calls `a.scopeAssignedContact(query, userID, orgID)` at line 116, and `scopeAssignedContact` (lines 236-256) returns the query unchanged only when the caller has `contacts:read`; otherwise it restricts to `assigned_user_id = userID OR active-transfer OR collaborator-in-metadata`. Agents — the population the requirement protects — never received other employees' conversations, even with the All tab present, because the All tab was further gated client-side by `contactsStore.canSeeAllTab = authStore.hasPermission('contacts', 'write')` (`contacts.ts:192`). So the leak the user is describing is a **principle/UX leak**, not a data leak: the very presence of an "All" tab in the sidebar contradicts the mental model that each employee sees only their own conversations. The fix is therefore UI-only; touching the backend would be cargo-culting a security control where none is missing, and would risk the existing `scopeAssignedContact` test coverage at `contacts_test.go:1426-1480` (the agent-without-`contacts:read` case). This plan touches zero `.go` files.

The second voice said: then this is trivial — delete the button, ship it. The evidence refutes this too. The All tab is not one button; it is a fan of interdependent symbols across two files plus i18n, and several of them carry latent state. `activeListTab` is persisted to `localStorage` under `whatomate.chat.activeListTab` (`contacts.ts:129`), so a user who last selected "All" will, after upgrade, hydrate a tab value that no longer exists unless `loadStoredTab()` coerces it. `searchHint` (`contacts.ts:235-254`) has two arms that mention `'all'` — one short-circuits the hint when the current tab is `'all'`, the other pushes `'all'` onto the suggested-tabs list — and both must be removed or the hint will offer a tab that doesn't exist. `visibleTabOrder()` and `TAB_ORDER` in `ChatView.vue` (lines 482-485) drive arrow-key navigation and would otherwise let focus land on a phantom tab. And the `chat.tabAll` i18n key, if left orphaned, becomes a permanent dead-string in both locale files. So the change is small but not empty; the work is "remove a feature cleanly," not "delete a button."

The third voice — the incentive/motive lens — asked why the All tab exists at all if `pending-me-tabs` decision D2 explicitly said other-agent chats "appear in neither tab." Reading the code comments ("M1", lines 2192 and 176) reveals it was a later admin/manager convenience feature bolted onto the original two-tab design, directly contradicting D2. The user's phrasing — "added by mistake, does not achieve the real principle" — is precisely the observation that this bolt-on broke the original contract. The honest resolution is to revert the bolt-on, not to refactor D2. This means the admin who used the All tab to scan all conversations loses that sidebar affordance; the requirement accepts this trade-off explicitly (the admin's full-org view still exists on the settings/Contacts page, a separate surface outside this spec). The plan calls this out in the spec's edge cases so the Reviewer can confirm the product call before the Builder ships.

The synthesis: a surgical, layered removal — store first (so the type narrows and the computeds disappear), then the view (template + script-helpers), then i18n (drop the dead key), then a verification gate that proves both the type system and the bundle are clean and that no `tabAll` reference survives anywhere.

## File-impact order and the reason for it

The change is ordered **store → view → i18n → verify**, because each layer's safety depends on the layer above having already narrowed the contract.

1. **Store first** (`frontend/src/stores/contacts.ts`). The `ListTab` type is the contract every consumer binds to. Narrowing it from `'pending' | 'me' | 'all'` to `'pending' | 'me'` and deleting `allContacts` / `allCount` / `canSeeAllTab` means that any stray reference to them in the view becomes a TypeScript error that `npm run typecheck` will catch — the type system becomes the safety net. If the view were edited first, the store would still export `canSeeAllTab` and the view's `v-if="contactsStore.canSeeAllTab"` would silently keep rendering nothing (no error, just a dead binding) until the store caught up. Store-first turns silent dead bindings into loud type errors.

2. **View second** (`frontend/src/views/chat/ChatView.vue`). Once the store no longer exports the All-tab symbols, the view's references must be removed for the type-check to pass. This is where the visible UX change happens: the All `<button>` block (lines 2202-2226), the `grid-cols-3` conditional (line 2199), the `TAB_ORDER`/`visibleTabOrder`/`tabLabel` All-arms (lines 482-485, 503-507), and the comment blocks (2191-2193, 2345-2348 — the latter only its first sentence mentions 'all'; the agent-tag itself stays).

3. **i18n third** (`frontend/src/i18n/locales/en.json` and `ar.json`). Removing `chat.tabAll` (line 490 in both) after the view no longer references it guarantees the key is genuinely orphaned. If the locale files were edited first, a mid-flight `npm run build` would still succeed (vue-i18n resolves missing keys to the key string at runtime, never throws), hiding any forgotten view reference; doing i18n last, after a grep confirms zero `tabAll` consumers, makes the removal honest.

4. **Verify last.** `npm run typecheck` (catches type regressions from the narrowed `ListTab` and the removed store exports), `npm run build` (catches template/compile regressions), and a repo-wide grep for `tabAll` / `canSeeAllTab` / `allContacts` / `allCount` / `'all'`-in-store-context (catches runtime string references the type system cannot see). The backend `go build ./...` and `go vet ./...` are run as a **safety net only** — they must be unchanged and must pass, proving no `.go` file was accidentally edited.

## Reuse map (existing symbols the Builder inherits — do not reinvent)

Every symbol below was read through `Read` in this session. The Builder builds on these, does not duplicate them.

- **`loadStoredTab()`** (`frontend/src/stores/contacts.ts:127-132`) — the existing persisted-tab loader. The Builder edits `VALID_TABS` (line 125) to drop `'all'`; the existing `VALID_TABS.includes(stored) ? stored : 'pending'` guard then naturally coerces a stale `'all'` to `'pending'`. No new migration helper is needed — the guard is the safeguard.
- **`activeListTab`** (`contacts.ts:133`) and its `watch` persist (lines 136-138) — unchanged; keep using them.
- **`pendingContacts` / `myContacts` / `pendingCount` / `myCount`** (`contacts.ts:178-186`) — unchanged; these are the two-tab substrate.
- **`displayedContacts`** (`contacts.ts:198-205`) — reused; only its `case 'all':` branch (line 201) is deleted, leaving the `me` and `pending/default` branches.
- **`searchHint`** (`contacts.ts:235-254`) — reused; two `'all'` arms deleted (lines 247, 252), the `pending`/`me` suggestion logic stays.
- **`sortedContacts`** (`contacts.ts:151-166`) — the underlying list computed; the All tab's `allContacts` was just `() => sortedContacts.value` (line 184), so removing `allContacts` loses nothing — the Pending and Me filters already operate on `sortedContacts` directly.
- **Tab strip template** (`ChatView.vue:2191-2275`) — the Pending and Me `<button>` blocks (2227-2274) are the reuse template the Builder leaves intact; only the All `<button>` block (2202-2226) and the `grid-cols-3` conditional (2199) are removed.
- **`onTabKeydown` / `visibleTabOrder` / `tabLabel`** (`ChatView.vue:477-507`) — reused; the All-specific branches are deleted, the arrow-key/Home/End logic for two tabs stays.
- **Backend `scopeAssignedContact`** (`internal/handlers/contacts.go:236-256`) — the privacy enforcement the spec leans on. **Not edited**, only cited as proof that the privacy principle already holds below the UI. The Builder must not touch it.

## Risks and mitigations

**R1 — Stale `localStorage` breaks the sidebar for existing users.** A user whose `whatomate.chat.activeListTab` is `'all'` would, after the store narrows, hydrate an invalid tab. *Mitigation:* `loadStoredTab()` already rejects any value not in `VALID_TABS`; removing `'all'` from `VALID_TABS` makes the existing `? stored : 'pending'` fallback fire. Acceptance criterion 3 verifies this. No separate migration code is written — the guard is the mitigation.

**R2 — `searchHint` suggests a phantom "All" tab.** Two arms of `searchHint` (`contacts.ts:247`, `252`) reference `'all'`. If only the store exports are removed but these arms are left, the hint would either never fire (the `current === 'all'` short-circuit is dead but harmless) or push `'all'` onto a tabs array that the view's `tabLabel` can no longer render (the `t('chat.tabAll')` lookup returns the raw key string after i18n removal). *Mitigation:* T3 deletes both arms explicitly; the grep verification in T6 confirms zero `'all'` string references remain in store context.

**R3 — Type-narrowing surfaces hidden consumers.** Narrowing `ListTab` could break a consumer the grep didn't surface (e.g. a dynamically-typed reference). *Mitigation:* the repo-wide grep (run in Phase 2) already confirmed the only `ListTab`-typed consumers are `contacts.ts` and `ChatView.vue`; `npm run typecheck` in T5 is the authoritative catch-all. If typecheck fails, the error names the exact missed consumer.

**R4 — Accidental backend edit.** Because `pending-me-tabs` touched `internal/handlers/chat_lifecycle.go` and `cmd/whatomate/main.go`, a Builder operating from muscle memory might re-touch them. *Mitigation:* T6 runs `go build ./...` and `go vet ./...` and asserts they are unchanged-and-green; the spec's Out-of-scope and the tasks' "MUST NOT" notes reinforce this. The release endpoint, system-message i18n, and audit logging are explicitly fenced off.

**R5 — Orphaned `chat.tabAll` key drifts the i18n mirror.** If `en.json` drops the key but `ar.json` keeps it (or vice versa), the en/ar parity the `pending-me-tabs` spec established is broken. *Mitigation:* T4 edits both files in the same task; T6's grep confirms `tabAll` is gone from both.

**R6 — Admin/manager product regression.** Admins who used the All tab lose that sidebar affordance. This is intentional per the requirement, but if the product owner disagrees the change must be revertible. *Mitigation:* the change is a pure deletion of additive code; `git revert` restores the All tab cleanly. The spec's edge-case section flags this trade-off for Reviewer sign-off before the Builder proceeds.

## Backward-incompatibility and migration path

- **Public API:** none. No backend route changes; `GET /api/contacts` is unchanged.
- **Persistence:** the only persisted state is `localStorage['whatomate.chat.activeListTab']`. The existing `loadStoredTab()` guard coerces stale `'all'` to `'pending'` — no explicit data migration, no server-side migration, no DB change.
- **i18n:** `chat.tabAll` is removed. Any third-party fork that overrides locale strings would lose this key silently (vue-i18n falls back to the key string, never throws). This is acceptable and is the same behavior as any removed i18n key.
- **Rollback:** `git revert` the change commit. The All tab returns with no data loss, because no data was migrated or destroyed.

## Build/test commands the Builder and Auditor run

| Layer   | Command                                  | Purpose                                                        |
|---------|------------------------------------------|----------------------------------------------------------------|
| FE type | `cd frontend && npm run typecheck`       | Catches type regressions from narrowed `ListTab` + removed exports (T5/T6). |
| FE lint | `cd frontend && npm run lint`            | eslint with `--fix`; ensures no unused imports/var after removal (T6). |
| FE bundle | `cd frontend && npm run build`         | vite build; catches template/compile regressions (T5/T6).     |
| BE safety | `go build ./...` and `go vet ./...`    | Proves no `.go` file was touched (T6).                         |
| BE tests | `go test ./internal/handlers/...`       | Optional regression confirmation; `scopeAssignedContact` tests at `contacts_test.go:1426-1480` must still pass (T6). |

The per-task gate (T1-T5) is `npm run typecheck` + `npm run build` after each layer. The final gate (T6) is the full matrix above plus the repo-wide grep for `tabAll` / `canSeeAllTab` / `allContacts` / `allCount`.
