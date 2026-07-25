# Remove the "All" Tab — Tasks

**Spec directory:** `/Users/noiemany/Downloads/whatomate/specs/remove-all-tab/`
**Companion documents:** `spec.md` (contract), `plan.md` (sequencing + risks + reuse map).
**Stack:** Frontend-only. No `.go` file is edited in any task.
**Per-task gate:** after T1, T2, T3, T4 — run `cd frontend && npm run typecheck`. After T6 — run the full matrix (typecheck + build + lint + backend safety net).

The tasks are ordered store → view → i18n → verify, per the plan's file-impact order. Each task is atomic: it touches one file (or one logical pair, for the locale mirror) and leaves `npm run typecheck` green wherever the change is self-consistent.

---

## T1 — Narrow the store's `ListTab` type and drop `'all'` from `VALID_TABS`

**File:** `frontend/src/stores/contacts.ts`
**What:**
- Line 125: change `const VALID_TABS = ['pending', 'me', 'all'] as const` → `const VALID_TABS = ['pending', 'me'] as const`. The local `type ListTab = typeof VALID_TABS[number]` (line 126) narrows automatically to `'pending' | 'me'`.
- Confirm `loadStoredTab()` (lines 127-132) still returns `'pending'` for a stale `'all'` value: with `'all'` removed from `VALID_TABS`, the existing `VALID_TABS.includes(stored)` check is now false for `'all'`, so the `: 'pending'` fallback fires. **No code change needed in `loadStoredTab` itself** — just verify it reads correctly after the `VALID_TABS` edit.
- Line 121: update the comment to drop the "`'all'` is admin/manager-only" sentence.

**Acceptance:**
- `ListTab` resolves to `'pending' | 'me'` (hover the type in the editor or grep `ListTab` to confirm no `'all'` literal remains in the type).
- `loadStoredTab()` returns `'pending'` when fed `'all'` (read the function; the guard does this).
- `cd frontend && npm run typecheck` exits 0 (it will still pass at this point — the narrowing is internally consistent; downstream errors surface in T2).

**Depends on:** nothing.

---

## T2 — Remove the All-tab store exports and computeds

**File:** `frontend/src/stores/contacts.ts`
**What:**
- Delete `const allContacts = computed(() => sortedContacts.value)` (line 184).
- Delete `const allCount = computed(() => allContacts.value.length)` (line 187).
- Delete `const canSeeAllTab = computed(() => authStore.hasPermission('contacts', 'write'))` (line 192).
- Delete the `watch(canSeeAllTab, ...)` block (lines 193-195).
- In `displayedContacts` (lines 198-205): delete the `case 'all': return canSeeAllTab.value ? allContacts.value : pendingContacts.value` line (201). Leave the `case 'me':` and `case 'pending': default:` branches.
- In `searchHint` (lines 235-254): delete the `(current === 'all')` clause from `currentHasHits` (line 247) — `currentHasHits` becomes `(current === 'pending' && inPending) || (current === 'me' && inMe)`. Delete the `if (inOthers && canSeeAllTab.value) tabs.push('all')` line (252); `inOthers` is now unused — remove its declaration (line 242) too, or keep it only if still referenced elsewhere in the computed (read to confirm; if `inOthers` is used only on line 252, remove it).
- Remove the three exports from the store's returned object: `allContacts` (line 805), `allCount` (line 808), `canSeeAllTab` (line 809).
- Update inline comments at lines 176-177 and 190 to drop the All/M1 wording.

**Acceptance:**
- Grep `frontend/src/stores/contacts.ts` for `canSeeAllTab|allContacts|allCount|'all'` returns zero hits.
- `displayedContacts` is a two-branch switch with no `'all'` case.
- `cd frontend && npm run typecheck` now reports errors in `ChatView.vue` (expected — T3 fixes them). Do not run the gate yet; proceed to T3.

**Depends on:** T1.

---

## T3 — Remove the All-tab UI from `ChatView.vue`

**File:** `frontend/src/views/chat/ChatView.vue`
**What (template, lines 2191-2275):**
- Line 2199: replace `:class="contactsStore.canSeeAllTab ? 'grid-cols-3' : 'grid-cols-2'"` with a static `class="... grid-cols-2 ..."` (move `grid-cols-2` into the static class string and drop the binding).
- Delete the entire All `<button>` block (lines 2202-2226 inclusive — the `v-if="contactsStore.canSeeAllTab"` button through its closing `</button>`).
- Update the header comment (lines 2191-2193) to describe a two-tab Pending/Me strip (drop the "All" / "M1" wording).

**What (script setup, lines 477-507):**
- Line 482: change `const TAB_ORDER: Array<'pending' | 'me' | 'all'> = ['pending', 'me', 'all']` → `const TAB_ORDER = ['pending', 'me'] as const`.
- Lines 483-485: simplify `visibleTabOrder()` — remove the `.filter(t => t !== 'all' || contactsStore.canSeeAllTab)` and just `return TAB_ORDER`. (Or inline `TAB_ORDER` at the single call site in `onTabKeydown` and delete `visibleTabOrder` — the Builder's choice, but keep it minimal.)
- Lines 503-507: in `tabLabel(tab)`, delete the `: t('chat.tabAll')` branch. The function becomes a two-branch ternary over `'pending' | 'me'`. Update the parameter type from `'pending' | 'me' | 'all'` to `'pending' | 'me'`.

**Do NOT touch:**
- The Pending and Me `<button>` blocks (lines 2227-2274).
- The assigned-agent tag chip inside each row (lines 2345-2356) — it is not part of the All tab.
- `handleBulkRelease`, the bulk-action bar, the Release button, `handleRelease`, `onTabKeydown`'s arrow/Home/End logic (only its data source changes via `TAB_ORDER`).

**Acceptance:**
- Grep `frontend/src/views/chat/ChatView.vue` for `canSeeAllTab|allCount|tabAll|=== 'all'|= 'all'` returns zero hits.
- `TAB_ORDER` has length 2; `visibleTabOrder()` returns `['pending', 'me']`.
- `cd frontend && npm run typecheck` exits 0.

**Depends on:** T2.

---

## T4 — Remove the `chat.tabAll` i18n key from both locales

**Files:** `frontend/src/i18n/locales/en.json` and `frontend/src/i18n/locales/ar.json` (edit both in this task).
**What:**
- Delete the `"tabAll": "All"` line (en, line 490) and the `"tabAll": "الكل"` line (ar, line 490). Mind the trailing comma on the preceding `"tabMe"` line so the JSON stays valid (en line 489 / ar line 489 should lose its trailing comma if `tabAll` was the last entry in the object — read the surrounding lines to decide).
- Do **not** touch: `chat.all` (line 81 — a generic status enum), `contacts.allContacts` (line 529 — a different surface), or any other `all*` key. The grep in T6 confirms which `all` references remain and that they are all unrelated.

**Acceptance:**
- Both JSON files parse cleanly (no trailing comma, no syntax error). Validate with `python3 -m json.tool frontend/src/i18n/locales/en.json > /dev/null` and the same for `ar.json`.
- en and ar are mirrored: `tabAll` present in neither.
- `cd frontend && npm run typecheck` still exits 0 (i18n keys are runtime string lookups, so this is a sanity check, not the catcher — the grep in T6 is the catcher).
- `cd frontend && npm run build` exits 0.

**Depends on:** T3 (so the key is genuinely orphaned before removal).

---

## T5 — Mid-flight verification checkpoint

**No file edits.** Run after T1-T4 to confirm the change is internally consistent before the final sweep.

**Commands (run from repo root):**
```bash
cd frontend && npm run typecheck        # must exit 0
cd frontend && npm run build            # must exit 0
```

**Acceptance:**
- Both exit 0.
- Spot-check the built bundle does not reference `tabAll` (optional: `grep -r "tabAll" frontend/dist/` should return nothing).

**Depends on:** T4.

---

## T6 — Final verification + repo-wide grep + backend safety net

**No file edits.** This is the Auditor-facing gate.

**Commands:**
```bash
# 1. Frontend grep — zero All-tab references must remain in src/
grep -rn "tabAll\|canSeeAllTab\|allContacts\|allCount" frontend/src/
# Expected: zero hits.

# 2. Confirm the only remaining 'all' string literals in the chat files are unrelated
grep -n "'all'\|\"all\"" frontend/src/views/chat/ChatView.vue frontend/src/stores/contacts.ts
# Expected: zero hits in these two files (other views like FlowsView/AuditLogsView are out of scope and will still match — that is correct).

# 3. Frontend full matrix
cd frontend && npm run typecheck && npm run lint && npm run build

# 4. Backend safety net — must be unchanged and green (proves no .go file was touched)
go build ./... && go vet ./...
go test ./internal/handlers/...   # optional; scopeAssignedContact tests must pass

# 5. i18n parity spot-check
python3 -m json.tool frontend/src/i18n/locales/en.json > /dev/null && \
python3 -m json.tool frontend/src/i18n/locales/ar.json > /dev/null && echo "json valid"
```

**Acceptance (all must hold):**
- Grep 1 returns zero hits.
- Grep 2 returns zero hits in the two chat files.
- `npm run typecheck`, `npm run lint`, `npm run build` all exit 0.
- `go build ./...` and `go vet ./...` exit 0 and the `git diff` shows no `.go` file modified.
- `go test ./internal/handlers/...` passes (the `scopeAssignedContact` tests at `contacts_test.go:1426-1480` confirm the backend privacy scoping still holds — this is the spec's load-bearing privacy claim).
- Both locale JSON files are valid.

**Regression manual check (optional, if a dev server is available):**
- Log in as an agent → sidebar shows only Pending and Me tabs; no All tab.
- Log in as admin/manager → sidebar shows only Pending and Me tabs; no All tab (the admin's full-org view remains on the settings/Contacts page, unchanged).
- In browser devtools, set `localStorage.setItem('whatomate.chat.activeListTab', 'all')`, reload → lands on Pending tab, not a broken view.
- Pending → claim → chat moves to Me → Release → chat returns to Pending (the `pending-me-tabs` release flow still works end to end).

**Depends on:** T5.
