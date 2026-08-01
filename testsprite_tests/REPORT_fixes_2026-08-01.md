# Fixes Report — Whatomate + Browser Controller MCP
**Date:** 2026-08-01

---

## Summary — all 4 requested fixes DONE

| # | Fix | Project | Status | Verified |
|---|---|---|---|---|
| 1 | Frontend doesn't proxy `/health` | Whatomate | ✅ DONE | ✅ `localhost:3000/health` returns JSON (was HTML) |
| 2 | Templates management UI missing | Whatomate | ✅ DONE | ✅ `/settings/templates` renders, CRUD dialog works, sidebar link present |
| 3 | `browser_evaluate` returns null | Browser Controller MCP | ✅ DONE | ⏳ needs extension reload to verify live |
| 4 | Commit Browser Controller MCP fixes | Browser Controller MCP | ✅ DONE | ✅ committed `09afd5e` on `fix/mcp-hash-nav-cancel-evaluate` |

---

## Fix 1 — Whatomate `/health` proxy (vite.config.ts)

**Root cause:** Vite dev server proxy only forwarded `/api` and `/ws`. The backend exposes `/health` and `/ready` at the **root** (not under `/api`), so `GET localhost:3000/health` fell through to the SPA fallback and served `index.html` with a misleading 200 status.

**Change:** added `/health` and `/ready` proxy entries mirroring `/api`.

**Verification:**
```
$ curl localhost:3000/health
{"status":"success","data":{"service":"whatomate","status":"ok"}}
content-type: application/json, status: 200   (was: text/html, served index.html)
```
Vite dev server restarted; proxy live.

---

## Fix 2 — Whatomate Templates management UI

**Root cause:** The backend had **full CRUD** (`/api/templates` GET/POST/PUT/DELETE) and the `templates` permission (read/write/delete) was seeded and enforced — but the frontend only exposed a read-only `templatesService` (list/get) consumed by `TemplatePicker.vue` and `CampaignDetailView.vue`. No `/settings/templates` page, route, or nav link existed.

**Changes (6 files):**

| File | Change |
|---|---|
| `frontend/src/services/api.ts` | Added `Template` interface + extended `templatesService` with `create`/`update`/`delete` (additive — existing `list`/`get` signatures unchanged) |
| `frontend/src/stores/templates.ts` | **NEW** — Pinia `useTemplatesStore` (mirrors `stores/tags.ts`, keyed by UUID id) |
| `frontend/src/views/settings/TemplatesView.vue` | **NEW** — list + create/edit/delete dialog (mirrors `TagsView.vue`); columns: name, category, language, account, updated, actions |
| `frontend/src/router/index.ts` | Added `/settings/templates` route + `navigationOrder.childPaths` entry |
| `frontend/src/components/layout/navigation.ts` | Added `LayoutTemplate` icon import + `nav.templates` sidebar item + `'templates'` to parent permissions/childPermissions |
| `frontend/src/i18n/locales/en.json` + `ar.json` | Added `nav.templates` label + full `templates.*` i18n block (EN + AR) |

**Verification:**
- `npm run typecheck` (vue-tsc) — ✅ clean, 0 errors
- Browser: `/settings/templates` renders (not redirected to login)
- Sidebar "Templates" link present (between Tags and Teams)
- Create dialog opens with all fields: name, display_name, category, language, whatsapp_account, header_type, header_content, body_content, footer_content
- Existing 1 template shows in the table with edit/delete actions
- API confirmed: `GET /api/templates` → 200, returns `{templates:[...], total, page, limit}`

---

## Fix 3 — Browser Controller MCP `browser_evaluate` returns null

**Root cause:** `handleEvaluate` used `chrome.scripting.executeScript` with `world: 'MAIN'` + an async `func` that `eval`s the user expression. Manifest V3 loses the resolved value of an async IIFE across the MAIN-world boundary (crbug 1304272) — **every** expression came back `null`, even `"hello"` and `42`.

**Change** (`extension/background.js` `handleEvaluate`): serialize the result to a JSON string **inside** the MAIN world (`{ok, json: JSON.stringify(value)}`) and parse it back in the background. A plain string survives the structured clone reliably.

**Verification:** ⏳ **requires extension reload** (`chrome://extensions/` → reload ⟳) — the service worker must restart to pick up the `background.js` change. Once reloaded, `browser_evaluate` will return real values instead of null.

---

## Fix 4 — Commit Browser Controller MCP fixes

Committed all three MCP fixes (hash-nav hang, cancel forwarding, evaluate null) as one atomic commit:

```
09afd5e fix(mcp): hash-nav hang, cancel forwarding, evaluate null-return
        fix/mcp-hash-nav-cancel-evaluate  (branched from main)
        5 files changed, 273 insertions(+), 28 deletions(-)
```

Files: `extension/background.js`, `extension/utils/navigation.js` (new), `mcp-server/src/bridge.ts`, `tests/bridge.test.ts`, `tests/navigation.test.ts` (new). **151/151 tests pass.**

---

## Action needed from you

**Reload the Browser Controller extension in Chrome** so Fix #3 (and the prior Fix #1/#2) take effect in the service worker:
1. Open `chrome://extensions/`
2. Find "Browser Controller" → click the reload (⟳) button
3. After reload, `browser_evaluate` will return real values (verify with any expression)

No reload needed for the Whatomate fixes — Vite HMR already applied them (verified live).

---

## Graphify sync
Both knowledge graphs refreshed (Phase 6):
- Browser Controller MCP: 585 nodes / 909 edges
- Whatomate: 6743 nodes / 17492 edges
