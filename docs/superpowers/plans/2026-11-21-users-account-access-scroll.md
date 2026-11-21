# Users Account Access Scroll Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The WhatsApp Account Access lists on /settings/users scroll inside their dialogs with a scrollbar that is always visible, so a long account list is navigable.

**Architecture:** Extend the shared shadcn-vue ScrollArea wrapper with an opt-in Reka `type="always"` (default stays `hover`, so every other ScrollArea is untouched), then use it for both account lists in UsersView with the existing `max-h-72` bound. No global scrollbar change, no backend change.

**Tech Stack:** Vue 3 + shadcn-vue (Reka UI ScrollArea) + Tailwind, Vitest + vue-tsc for verification.

**Spec:** Parent task brief (user request, 2026-11-21): /settings/users WhatsApp Account Access list shows no scrollbar when long; fix by adding overflow-y auto to the container and ensuring the scrollbar is always visible, consistently in both dialogs, responsive across devices, following Impeccable for UI/UX.

## Global Constraints

- Work only in worktree `~/workspace/gowa-ui-users-scroll`, branch `fix/users-account-access-scroll`, base `db8c58b`. No push, deploy, or merge.
- Preserve the established design system (DESIGN.md): dark-first, emerald accent only, scrollbars 6px with muted-foreground thumb, radius 0.75rem, no side-stripe borders, no gradient text.
- Keep `npm run typecheck` green and the full Vitest suite green.
- Keyboard navigation must keep working (checkboxes focusable, list scrolls on focus); light/dark themes must stay readable.
- Impeccable: its CLI bundle download failed (TLS, `impeccable.style/api/download/bundle/universal` unreachable), so the launcher/context step did not run; the skill text and `reference/craft-floor.md` were read directly from the cloned repo and the project DESIGN.md instead. Do not block on it.

## Review Focus

- Reka `type="always"` must actually render the scrollbar without hover/scroll (not just add a class that does nothing).
- The create/edit dialog list must not keep growing the dialog — the bound must apply there too, not only in the Access dialog.
- The thumb must be visible enough in both themes at the always-visible setting (current `bg-border` thumb is near-invisible).
- No other ScrollArea in the app may change behaviour (default stays `hover`).

---

### Task 1: ScrollArea gains opt-in always-visible type

**Files:**
- Modify: `frontend/src/components/ui/scroll-area/ScrollArea.vue`
- Test: `frontend/src/components/ui/scroll-area/ScrollArea.test.ts` (new)

**Interfaces:**
- Consumes: Reka `ScrollAreaRoot` prop `type?: ScrollType` (`'auto' | 'always' | 'scroll' | 'hover'`, default `'hover'`).
- Produces: `ScrollArea` prop `type?: 'auto' | 'always' | 'scroll' | 'hover'`, default `'hover'`; when `type === 'always'`, thumb class is `bg-muted-foreground/50` instead of `bg-border`.

- [ ] **Step 1: Write the failing test** — mount `ScrollArea` with `type="always"` via @vue/test-utils; assert the rendered scrollbar element is present with always-visible state (and that the default mount keeps the `bg-border` thumb). Assert the `type` prop passes through to the Reka root.
- [ ] **Step 2: Run test to verify it fails** — `npx vitest run src/components/ui/scroll-area/ScrollArea.test.ts` — Expected: FAIL (no `type` prop / thumb unchanged).
- [ ] **Step 3: Implement** — add the `type` prop, pass `:type="props.type ?? 'hover'"` to `ScrollAreaRoot`, and switch the thumb class when `type === 'always'`.
- [ ] **Step 4: Run test to verify it passes** — same command — Expected: PASS.
- [ ] **Step 5: Commit** — `git add frontend/src/components/ui/scroll-area/ && git commit -m "feat(ui): ScrollArea supports always-visible scrollbar type"`

### Task 2: Both account lists in UsersView scroll with an always-visible scrollbar

**Files:**
- Modify: `frontend/src/views/settings/UsersView.vue`

**Interfaces:**
- Consumes: Task 1 `ScrollArea` with `type="always"`.
- Produces: Account Access dialog list and Create/Edit dialog list both bounded at `max-h-72` inside a `ScrollArea type="always"`.

- [ ] **Step 1: Write the failing test** — extend the Task 1 test or add a source-level check: grep that UsersView contains no account-assignment list outside a `ScrollArea type="always"` (the create/edit `<div class="space-y-2 pt-1">` must be gone). (A full UsersView mount test is out of scope: it needs stores/router/API mocks.)
- [ ] **Step 2: Run the check to verify it fails** — Expected: the create/edit plain div is still present.
- [ ] **Step 3: Implement** — Access dialog: `<ScrollArea v-else class="max-h-72">` → `<ScrollArea v-else type="always" class="max-h-72">`. Create/Edit dialog: wrap the `accountAssignments` `v-for` list in `<ScrollArea type="always" class="max-h-72"><div class="space-y-2 pr-2">…</div></ScrollArea>` (keep checkbox ids, labels, and `toggleAccount` wiring unchanged; keep `pr-2` so text clears the scrollbar).
- [ ] **Step 4: Run typecheck + full unit suite** — `npm run typecheck` and `npx vitest run` in `frontend/` — Expected: both green.
- [ ] **Step 5: Commit** — `git commit -m "fix(users): always-visible scrollbar for WhatsApp account access lists"`

## Self-Review

- Spec coverage: always-visible scrollbar (Task 1), bounded scrolling in both dialogs (Task 2), no global change (default `hover` preserved), keyboard/themes/responsive reasoned in the final report (no Playwright MCP in this session, so no live render claim).
- Step scan: each step is one checkable action.
- Type consistency: `type` prop values match Reka `ScrollType`.
- Proportion: plan is shorter than the code it describes.
