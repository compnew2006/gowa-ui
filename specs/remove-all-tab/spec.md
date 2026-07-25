# Remove the "All" Tab — Specification

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/remove-all-tab/`
**Status:** Drafted for Plan Reviewer, Builder, Auditor
**Original requirement (English + Arabic):** "the tab All in the chat sidebar is not correct tab you have add it by mistek it's not تؤدي الفكرة المطلوبة والمبدأ الحقيقى حيث اريد ان لا يري كل موظف الا يري الا محادثاته" — i.e. the "All" tab in the chat sidebar was added by mistake; it violates the real principle that **each employee should only see their own conversations**. The "All" tab must be removed from the chat sidebar.

---

## Locked stack (detected from root markers)

- **Root markers present:** `go.mod` (Go backend) + `frontend/package.json` (Vue 3 frontend). This is a two-subtree repo; the change in this spec touches **only the frontend subtree**, but the backend is described because the spec must prove the privacy principle already holds there.
- **Backend:** Go 1.25 (`module github.com/shridarpatil/whatomate`), fastglue router, GORM, PostgreSQL. Server entrypoint `cmd/whatomate/main.go`. Contacts list route `g.GET("/api/contacts", app.ListContacts)` at `cmd/whatomate/main.go:744`.
- **Frontend:** Vue 3 `<script setup lang="ts">` + Composition API + Pinia + vue-i18n + shadcn-vue, at `frontend/`. The chat sidebar lives in `frontend/src/views/chat/ChatView.vue`; its store is `frontend/src/stores/contacts.ts`; i18n schema source is `frontend/src/i18n/locales/en.json` mirrored key-for-key in `ar.json`.
- **Package manager:** npm (`frontend/package.json`).
- **Real verify commands** (read from `frontend/package.json` scripts):
  - Frontend type-check: `cd frontend && npm run typecheck` (runs `vue-tsc --noEmit`).
  - Frontend bundle: `cd frontend && npm run build` (runs `vite build` only — type-checking is **not** part of `build`; both must be run).
  - Frontend lint: `cd frontend && npm run lint` (eslint with `--fix`).
  - Frontend unit/e2e: `cd frontend && npm run test` (Playwright, requires `BASE_URL=http://localhost:8080` and a running server — not part of the per-task gate).
  - Backend (unchanged by this spec, listed for completeness): `go build ./...`, `go vet ./...`, `go test ./internal/handlers/...`.

## MCP tiering note

Serena, Socraticode, and codebase-memory-mcp were **not** exposed as tools in this session. The graphify knowledge graph at `graphify-out/graph.json` was live and was consulted per the project/global conventions, but graphify does not index Vue `<script setup>` local symbols (e.g. `canSeeAllTab`, `activeListTab`, `displayedContacts` resolve to "No unique node match" / "No node matching"), so for the precise frontend blast radius the documented fallback was used: harness-native `Read`, `Grep`, and `Glob`, with shell `grep` as a corroborating repo-wide scan. graphify was used to confirm there is **no backend path** specific to the All tab (`graphify path "ChatView" "ListContacts"` → "No path found", confirming the All tab adds no backend route). Every file:line citation below was read through `Read` in this session — none is from memory. Confidence is high because all anchors were directly inspected.

## Relationship to the prior `specs/pending-me-tabs/` spec

`specs/pending-me-tabs/` **is the predecessor of this change and is directly related**. That spec introduced the Pending/Me tab strip and the release flow; its approved decision **D2** explicitly stated that chats assigned to *other* agents "appear in neither tab (they are reached via search or the admin's full view, unchanged)". The "All" tab was a **later addition** (labelled "M1" in the code comments at `ChatView.vue:2192` and `contacts.ts:176`) that contradicts D2: it surfaced every org contact to admin/manager users in a third tab. This spec **reverts that later addition** and restores the original two-tab principle. The release flow, the `PUT /api/contacts/{id}/release` endpoint, the system-message i18n, and the audit-log entries from `pending-me-tabs/` are all **out of scope and must not be touched** — only the "All" tab surface is removed.

## Goal

Remove the "All" tab from the chat sidebar so the sidebar exposes exactly two tabs — **Pending** (unassigned queue) and **Me** (assigned to the current user) — and no UI path remains by which a user can list conversations across the whole organization from the sidebar. This restores the per-employee privacy principle the requirement states, at the UI layer.

## Actors and permission matrix (post-change)

The backend permission model is unchanged. The matrix below describes the post-change sidebar experience.

| Actor                                     | See Pending tab | See Me tab | See All-org tab | Reach a chat assigned to another agent from the sidebar |
|-------------------------------------------|:---------------:|:----------:|:---------------:|:-------------------------------------------------------:|
| Agent (no `contacts:read`)                | yes             | yes (own)  | **no (removed)** | no — backend `scopeAssignedContact` already blocks this |
| Collaborator (`chat.collaborate:write`)   | yes             | yes (own)  | **no (removed)** | no                                                                       |
| Admin / Manager (`contacts:write`)        | yes             | yes (own)  | **no (removed)** | no — not via the sidebar; admin's full org view (e.g. settings/contacts) is a separate, already-existing surface outside this spec |

**Key clarification for the Reviewer/Builder:** The backend `ListContacts` handler (`internal/handlers/contacts.go:100`) already enforces this matrix at the data layer via `scopeAssignedContact` (`contacts.go:236-256`): any caller without `contacts:read` only ever receives contacts assigned to them, transferred to them, or where they are a collaborator. The "All" tab was therefore **never a data-leak** for agents — it was gated by `contacts:write` and only admin/manager ever saw it. The leak this spec closes is the **principle/UX leak**: the requirement is that the sidebar must not present an "All" view at all, even to admins, because it contradicts the "each employee sees only their own conversations" mental model. There is **no backend change** in this spec; the backend scoping is already correct and is referenced here only to prove the privacy principle already holds below the UI.

## Public contracts

There is **no new** public contract. This spec only **removes** UI surface. The contracts below describe what is removed and what is preserved.

### REST routes

**No route is added, changed, or removed.** The All tab did not call a dedicated endpoint — it rendered `contactsStore.allContacts`, which is a passthrough computed over the already-fetched `contacts` list returned by the existing `GET /api/contacts` (`ListContacts`). Confirmed by `graphify path "ChatView" "ListContacts"` returning "No path found" (no All-specific edge) and by reading the store (`contacts.ts:184` `allContacts = computed(() => sortedContacts.value)` — no fetch, no extra param). The `GET /api/contacts` scoping via `scopeAssignedContact` is unchanged and remains the source of truth for who can see which conversations.

### Frontend store contract (`useContactsStore`, `frontend/src/stores/contacts.ts`)

The `ListTab` type narrows from `'pending' | 'me' | 'all'` to `'pending' | 'me'`. Concretely, the following symbols are **removed**:

- `allContacts` computed (line 184) — removed.
- `allCount` computed (line 187) — removed.
- `canSeeAllTab` computed (line 192) — removed.
- The `watch(canSeeAllTab, ...)` fallback (lines 193-195) — removed.
- The `case 'all':` branch of `displayedContacts` (line 201) — removed; `displayedContacts` becomes a two-branch switch.
- The `VALID_TABS` array entry `'all'` (line 125) — removed; `VALID_TABS` becomes `['pending', 'me']`.
- The `'all'` handling inside `searchHint` (lines 247, 252) — the `current === 'all'` short-circuit and the `if (inOthers && canSeeAllTab.value) tabs.push('all')` push are removed; the hint can still suggest `pending`/`me`.
- The store's returned object entries `allContacts`, `allCount`, `canSeeAllTab` (lines 805, 808, 809) — removed.
- The `'all'` literal in the `loadStoredTab` `localStorage` comment (line 121) and the inline comments referencing `'all'` (lines 176-177, 190) — updated to reflect the two-tab world.

The `loadStoredTab()` function gains a **migration safeguard**: if a user's persisted `localStorage` value is `'all'` (set before this change shipped), it must fall back to `'pending'` rather than rendering a tab that no longer exists. This is enforced by `VALID_TABS` no longer containing `'all'`, so the existing `VALID_TABS.includes(stored)` check naturally rejects the stale value; the Builder must confirm this path returns `'pending'` and not `undefined`.

The existing `pendingContacts`, `myContacts`, `pendingCount`, `myCount`, `displayedContacts` (minus its `'all'` branch), `searchHint` (minus its `'all'` arms), `activeListTab`, and all lifecycle actions (`claimChat`, `releaseChat`, `leaveChat`, `closeChat`, `reopenChat`, `joinChat`, `inviteCollaborator`, `removeCollaborator`, `bulkReleaseChats`) are **unchanged**.

### Sidebar UI contract (`frontend/src/views/chat/ChatView.vue`)

The tab strip (currently lines 2191-2275) is reduced from a conditional 2-or-3 column grid to a fixed 2-column grid:

- The `:class="contactsStore.canSeeAllTab ? 'grid-cols-3' : 'grid-cols-2'"` binding (line 2199) becomes a static `grid-cols-2`.
- The entire "All" `<button>` block (lines 2202-2226, the `v-if="contactsStore.canSeeAllTab"` button with `id="tab-all"`, its `@click="contactsStore.activeListTab = 'all'"`, and its `contactsStore.allCount` badge) is removed.
- The Pending and Me buttons (lines 2227-2274) are unchanged.
- The header comment (lines 2191-2193) is updated to drop the "All" / "M1" wording.

The tab keyboard-navigation helpers in `<script setup>` (lines 477-507) are narrowed:

- `TAB_ORDER` (line 482) becomes `['pending', 'me']`.
- `visibleTabOrder()` (lines 483-485) loses the `canSeeAllTab` filter and becomes a plain `return TAB_ORDER` (or is inlined away).
- `tabLabel(tab)` (lines 503-507) loses its `'all'` branch and the `t('chat.tabAll')` call.

The assigned-agent tag inside each contact row (lines 2345-2356, the `<span v-if="contact.assigned_user_name && contact.assigned_user_id !== authStore.user?.id">` chip) is **kept as-is**. It is not part of the All tab — it renders on every tab whenever a row happens to be assigned to another agent (e.g. an admin viewing a Pending chat that someone else just claimed). Removing it is out of scope; it carries independent diagnostic value and its `assignedTo` i18n key (line 353/496) is shared with other surfaces.

### i18n contract

The key `chat.tabAll` (at `en.json:490` / `ar.json:490`, value `"All"` / `"الكل"`) becomes **orphaned** by this change. It is **removed** from both locale files to keep the schema mirror in parity and to avoid a dead key. The sibling keys `chat.tabPending` (488) and `chat.tabMe` (489) are unchanged.

The following look-alike keys are **not** touched (verified, they belong to other features):
- `chat.all` (`en.json:81`) — a generic status enum ("All") used by unrelated filters.
- `contacts.allContacts` (`en.json:529`) — the settings Contacts page title, a different surface.
- `flows.allAccounts`, `auditLogs.allUsers`, `cannedResponses.allCategories`, `campaigns.allStatuses`, `agentTransfers.allActive`, `agentAnalytics.allAgents` — all unrelated filter labels.

## Data model

**No schema change.** No migration. No GORM model change. The All tab was pure client-side state; its removal touches no database column, no `AutoMigrate`, and no JSONB field. The `Contact.Metadata["chat_status"]` values (`pending`/`open`/`closed`) and `AssignedUserID` are unchanged.

## State machine

The chat lifecycle state machine is unchanged (Pending ↔ Open → Closed, with release/claim/reopen transitions as defined in `specs/pending-me-tabs/spec.md`). The only state removed is the **UI tab selector** value `'all'`, which is not part of the chat state machine — it was a local view selector persisted to `localStorage`. The migration safeguard in `loadStoredTab()` handles stale `'all'` values.

## Non-functional requirements

- **Privacy (the core NFR).** After this change, no user — regardless of role — can select a sidebar tab that lists org-wide conversations. Agents were already blocked by the backend; admins now have no sidebar path either. The backend `scopeAssignedContact` remains the enforcement source of truth and is untouched.
- **Backward compatibility of persisted state.** A user who last had the All tab active must not land on a broken/empty view after upgrade. `loadStoredTab()` must coerce a stale `'all'` to `'pending'`. This is verified in acceptance.
- **No network impact.** Removing the tab adds no round-trips and removes none — the All tab never fetched anything the Pending/Me tabs don't already fetch (the contact list is loaded once by `fetchContacts`).
- **No performance regression.** The `displayedContacts` computed loses a branch; `allContacts`/`allCount` computeds are removed, slightly reducing reactivity work.
- **i18n parity.** `en.json` and `ar.json` must stay key-for-key mirrored after the removal of `chat.tabAll`.

## Edge cases and failure modes

- **Stale `localStorage` value `'all'`.** `loadStoredTab()` must return `'pending'` (never `undefined`, never crash). The existing `VALID_TABS.includes(stored)` guard already does this once `'all'` is removed from `VALID_TABS`; the Builder must read the function and confirm the fallback branch is reached.
- **`searchHint` referencing `'all'`.** Two arms of `searchHint` (lines 247 and 252) mention `'all'`. After removal, if a search matches only chats assigned to *other* agents, the hint must not suggest a non-existent tab. The Builder removes both arms; the hint then only ever suggests `pending` and/or `me`, which is correct (a search matching only others' chats shows no hint and the empty-state "no results" message, which is the honest UX).
- **`visibleTabOrder()` after removal.** Must return exactly `['pending', 'me']` with no reference to `canSeeAllTab`; the arrow-key/Home/End logic must still work for two tabs.
- **Orphaned `chat.tabAll` i18n key.** If left in place, `$t('chat.tabAll')` would still resolve but be dead code. Removing it keeps the schema honest; if any hidden reference remains, `npm run typecheck` will not catch it (vue-i18n keys are runtime string lookups, not type-checked), so the Builder must grep-verify zero remaining `tabAll` references after the edit.
- **Admin/manager expectation.** An admin who relied on the All tab to see all conversations must now use the existing settings/Contacts page (which already lists org contacts under `contacts:write`) or search. This is an intentional product trade-off per the requirement; it is called out here so the Reviewer can confirm it is acceptable.
- **No regression to Pending/Me/release.** The release flow, system messages, audit logging, and WS events from `pending-me-tabs/` must continue to work identically. The Builder must not touch `releaseChat`, `handleRelease`, `handleChatReleased`, the `chat.system.*` keys, or the Release button.

## Acceptance criteria

1. The chat sidebar renders exactly two tabs — **Pending** and **Me** — with their existing count badges; no "All" tab is present for any role, including admin/manager.
2. `contactsStore.canSeeAllTab`, `allContacts`, and `allCount` no longer exist on the store; `displayedContacts` has no `'all'` branch; `VALID_TABS` is `['pending', 'me']`.
3. A user whose `localStorage` previously held `whatomate.chat.activeListTab = 'all'` lands on the Pending tab (not a blank/broken view) after the change ships.
4. `searchHint` never suggests an "All" tab; it suggests only Pending and/or Me when a cross-tab match exists.
5. Keyboard navigation (Arrow Left/Right, Home, End) over the tab strip works for the two remaining tabs.
6. The `chat.tabAll` key is removed from both `en.json` and `ar.json`, and a repo-wide grep for `tabAll` returns zero hits in `frontend/src/`.
7. The Release flow, system-message i18n, audit-log entries, and `chat_released` WebSocket event behave exactly as before (regression check against `specs/pending-me-tabs/` acceptance criteria 1-8).
8. `cd frontend && npm run typecheck` exits 0.
9. `cd frontend && npm run build` exits 0.
10. (Backend regression, unchanged by this spec but run as a safety net) `go build ./...` and `go vet ./...` exit 0 — confirming no backend file was accidentally touched.

## Out of scope

- Any backend change (routes, handlers, services, models, migrations). The backend already enforces the privacy principle via `scopeAssignedContact`.
- Any change to the settings/Contacts page (`ContactsView.vue`), analytics, audit logs, flows, or any other surface that legitimately lists org-wide data under admin/manager permissions.
- Removal of the assigned-agent tag chip inside contact rows (it is not part of the All tab; it renders on every tab and is independently useful).
- Any change to the Pending/Me/release system-message i18n (`chat.system.*`) or the Release button.
- Renaming the `ListTab` type or restructuring the store beyond the minimal removal.
- Any DB or Redis change.
