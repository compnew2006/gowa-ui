# Frontend Quick Wins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 5 small, frontend-only UI bugs found in the live CRM audit, on branch `fix/frontend-quick-wins` from `db8c58b`.

**Architecture:** Minimal, in-place Vue fixes. No backend changes, no new dependencies, no redesigns. Each fix keeps `npm run typecheck` green.

**Tech Stack:** Vue 3 (`<script setup lang="ts">`, Composition API), Chart.js via `@/lib/charts`, Vitest.

**Spec:** Live CRM audit (2026-11-21, Super Admin, read-only) + task brief from the parent agent.

## Global Constraints

- Work only in worktree `~/workspace/gowa-ui-frontend-fix`, branch `fix/frontend-quick-wins`. Local commits only — no push, deploy, merge.
- Do not touch `ChatView.vue` `<template>` structure beyond the minimal attribute fixes named below (e2e tests depend on its DOM).
- Do not touch the live CRM, the main clone, or the `fix/agent-least-privilege` worktree.
- A fix that turns out to need backend work or a redesign is parked, code left unchanged, reason recorded.

## Review Focus

- Hiding the Status composer would silently remove the only text-Status posting path — verify what a composer actually sends before hiding one.
- A second y-axis on a Bar chart must not rescale the messages axis or break the legend/tooltip.
- Auto-opening the date picker must not re-open it after Apply, and must not double-fetch on Dashboard/Analytics/AuditLogs watches.
- Router titles must fall back to "Gowa-UI" for routes without a title (login, 404, profile).

---

### Task 1: Translation key rendered literally

**Files:**
- Modify: `frontend/src/views/chat/ChatView.vue:1198`

**Interfaces:**
- Consumes: existing `$t('chat.conversationTabs')` key (exists in `frontend/src/i18n/locales/*.json`).

- [ ] **Step 1: Fix the attribute** — change bare `aria-label="$t('chat.conversationTabs')"` to `:aria-label="$t('chat.conversationTabs')"` (matches every other aria-label in the file).
- [ ] **Step 2: Verify no other bare `aria-label="$t(...)"` remains** — Run: `grep -rn 'aria-label="\$t' frontend/src` — Expected: no matches.
- [ ] **Step 3: Run typecheck** — Run: `cd frontend && npm run typecheck` — Expected: exit 0.
- [ ] **Step 4: Commit** — `fix(chat): bind conversationTabs aria-label so it translates`

### Task 2: Agent Comparison chart plots the wrong metric on the wrong scale

**Files:**
- Modify: `frontend/src/views/analytics/AgentAnalyticsView.vue` (`comparisonChartData`, `comparisonChartOptions`)

**Interfaces:**
- Consumes: `agent_stats[].messages_sent`, `agent_stats[].avg_rating` (already in `AgentPerformanceStats`).

- [ ] **Step 1: Change the second dataset** — `data` maps `a.avg_rating` (not `a.ratings_count`); add `yAxisID: 'y1'`.
- [ ] **Step 2: Add the second axis** — `scales.y1: { type: 'linear', position: 'right', min: 0, max: 5, ticks: { stepSize: 1 }, grid: { drawOnChartArea: false } }`; keep `scales.y` (messages) as-is.
- [ ] **Step 3: Verify** — typecheck green; confirm by reading the computed that messages stay on `y` and ratings on `y1` (no chart-data unit-test pattern exists for SFC computeds in this repo, so no new test is forced here).
- [ ] **Step 4: Commit** — `fix(analytics): plot avg rating on a 0-5 axis in agent comparison`

### Task 3: Custom Range picker never opens

**Files:**
- Modify: `frontend/src/composables/useDateRange.ts`
- Test: `frontend/src/composables/useDateRange.test.ts` (new; follows the `useChat*.test.ts` pattern)

**Interfaces:**
- Consumes: `selectedRange`, `isDatePickerOpen`, `applyCustomRange` (closes the picker).

- [ ] **Step 1: Write the failing test** — selecting `'custom'` sets `isDatePickerOpen` to `true`; `applyCustomRange()` with a complete range sets it back to `false`. Run it — Expected: FAIL (picker never opens).
- [ ] **Step 2: Implement** — in the existing `watch(selectedRange, ...)` (or a new one), set `isDatePickerOpen.value = true` when the new value is `'custom'`. Do not change `applyCustomRange` or the Dashboard/Analytics/AuditLogs watches (they fetch only on non-custom changes).
- [ ] **Step 3: Run the test + typecheck** — Run: `cd frontend && npx vitest run src/composables/useDateRange.test.ts && npm run typecheck` — Expected: PASS, exit 0.
- [ ] **Step 4: Commit** — `fix(date-range): auto-open picker when Custom Range is selected`

### Task 4: Browser tab title is always "Gowa-UI"

**Files:**
- Modify: `frontend/src/router/index.ts` (add `title` to `RouteMeta`, `meta.title` on main routes, `router.afterEach`)
- (No change to `frontend/index.html` — it stays the fallback title.)

**Interfaces:**
- Consumes: route `name` / `meta.title`.

- [ ] **Step 1: Add titles** — `meta: { title: '...' }` on dashboard ("Dashboard"), chat-conversation ("Chat"), campaigns ("Campaigns"), agent-analytics ("Agent Analytics"), settings ("Settings"). Extend `RouteMeta` with `title?: string`.
- [ ] **Step 2: Add afterEach** — `router.afterEach((to) => { document.title = to.meta.title ? \`${to.meta.title} | Gowa-UI\` : 'Gowa-UI' })`.
- [ ] **Step 3: Verify** — typecheck green.
- [ ] **Step 4: Commit** — `feat(router): set document title per route`

### Task 5: Status conversation composer — PARKED as specified, placeholder clarified instead

**Finding (do not implement the brief as written):** The message input in the Status virtual conversation is the *only* text-Status posting UI: `useChatMessaging.sendMessage()` routes a Status contact to `sendStatusText()`, which posts to `status@broadcast` via `statusService.sendText` (`frontend/src/composables/useChatMessaging.ts:96-97,134-161`). The attach button likewise routes media to `sendStatusMedia`. Hiding the input would delete the text-Status feature, not fix a bug. The audit read an empty log + composer as "a chat box in an empty conversation"; the composer is the feature.

**Files:**
- Modify: `frontend/src/views/chat/ChatView.vue` textarea `:placeholder` only (one ternary branch, no template-structure change).

- [ ] **Step 1: Clarify the placeholder** — when the current contact is the Status contact (`isStatusContact(contactsStore.currentContact?.id)`, already imported and used in this file), placeholder is `$t('chat.statusHint')` ("Posts to your WhatsApp Status (story)…", already translated in en/ar). Otherwise unchanged.
- [ ] **Step 2: Verify** — typecheck green; confirm `sendMessage`/`sendStatusText` untouched.
- [ ] **Step 3: Commit** — `fix(chat): label the Status composer as a Status post, not a chat message`
- [ ] **Step 4: Record** — the hide-the-input change is parked in the final report with this evidence, so the parent can confirm or redirect.

## Out of scope (parked, not attempted)

Backend counting/unread numbers, Recent Messages duplicates, Audit Logs flooding, 0 online, Break Time, org/timezone mismatches, narrow-table Actions overflow.
