# Whatomate — Full App Feature QA Report
**Date:** 2026-07-23
**Tester:** Automated (ZCode agent) — UI via browser MCP + API-level verification
**Environment:** Local — backend `go run` on :8080, frontend Vite on :3001, GOWA server on :3080
**Login:** admin@admin.com / admin (super_admin)
**Accounts present:** 2 connected GOWA devices → `1-egypt-1` (device `egypt`, JID 201007181781@s.whatsapp.net) and `1` (device `Saudi`)
**Test targets:** +201007181781 (contact `9c70ab67-...`, bound to account `1`) and 966561853319

---

## TL;DR

The **entire UI/CRUD layer works** (40+ routes, all list/create/update/delete endpoints return 200).
**Two bugs block real WhatsApp delivery and two chat actions.** Everything else is functional.

| # | Severity | Feature | Status | Root cause |
|---|----------|---------|--------|------------|
| 1 | 🔴 **Blocker** | Send text / media / template / group message (actual delivery) | **FAILS at delivery** (HTTP 200 create, then GOWA `401 Unauthorized`) | GOWA message-registry credential resolver reads the **config file** (`[[gowa_instances]]`), which is empty. Servers created via the UI live only in the DB, so the resolver returns empty creds → GOWA Basic Auth fails. |
| 2 | 🔴 **Blocker** | Revoke ("Delete for everyone") | **FAILS** (`404 Contact not found`) | `RevokeMessage` passes `uuid.Nil` instead of the real userID to `scopeAssignedContact` → permission check fails → contact filtered out. |
| 3 | 🔴 **Blocker** | Typing indicator | **FAILS** (`404 Contact not found`) | Same `uuid.Nil` bug in `SendTypingIndicator` (identical copy-paste). |
| 4 | 🟡 Minor | Accounts page | Missing i18n keys (`accounts.serverConnection`, `accounts.gowaBaseUrl`, `accounts.gowaDeviceId`, `common.created`) + unresolved `Plus` component | New GOWA strings not added to locale files; `Plus` icon import missing. |
| 5 | 🟡 Minor | 24h-window warning banner | Shows even for GOWA accounts | The "24-hour window expired" banner is shown for GOWA contacts where it does not apply (GOWA has no 24h restriction). Cosmetic/UX only. |

Everything else tested **PASSED**. Details below.

---

## 1. Routes inventory (all render)

All 40+ frontend routes load and are permission-gated correctly (admin has full access):

**Main:** Dashboard `/`, Chat `/chat/:contactId?`, Profile `/profile`
**Messaging:** Chatbot `/chatbot` (+ keywords/flows/ai/transfers), Campaigns `/campaigns`, Flows `/flows`
**Analytics:** Agent Analytics `/analytics/agents`
**Settings (all 200):** General, Chatbot, Accounts, Contacts, Canned Responses, Tags, Teams, Users, Roles, API Keys, Webhooks, Custom Actions, SSO, Audit Logs, **GOWA Servers** `/settings/gowa-servers/:id`

---

## 2. ✅ What WORKS (verified)

### Auth & session
- Login (admin@admin.com/admin) → 200, session cookie + JWT, CSRF flow
- Logout → 200
- `/api/me` profile, `/api/me/settings` update, organizations

### GOWA device management (uses DB credentials directly — works)
- **List devices**: `egypt` + `Saudi` = Connected, `asd-81344a32` = Disconnected
- **Sync Contacts** (egypt): `synced=1000, created=1000, total=1092` ✅
- **Device QR** (asd-81344a32): returns valid base64 PNG QR code ✅
- **Device Reconnect** (egypt): `{ok:true}` ✅
- **Device Sync** (egypt): backfills account, returns JID `201007181781@s.whatsapp.net` ✅
- **Device webhook GET**: returns `webhook_url`, `webhook_secret`, `webhook_events` ✅
- GOWA server CRUD, Create Device, Pair Code, Logout, Webhook PUT — all endpoints respond.

### Contacts
- List (1000 contacts, paginated 50 pages), search (found `201007181781`), pagination
- **Create** → correctly rejects duplicate with `409 "Contact with this phone number already exists"`
- Contact detail, Open-chat, Edit, Delete buttons present
- 36 group contacts (JIDs ending `@g.us`) imported by Sync Contacts

### Chat UI
- Contact sidebar (avatars, names, last-message time), search
- Account switcher (`1-egypt-1` / `1`) — selectable, active state tracked
- Message thread renders; outgoing messages appear instantly with status + timestamps
- Per-message action buttons (react, reply, revoke/"Delete for everyone", retry)
- "Retry sending" button appears on failed messages
- Ghosting / "Leave Conversation" controls present

### Messaging (DB layer — message rows created, HTTP 200)
- **Send text** → message created, `whatsapp_account` recorded, status `pending`
- **Send media/document** (multipart) → message created with `media_url`, `media_mime_type`, `media_filename`
- **Group message send** → message created (group contact resolves)
- **Reaction** (`👍`) → 200, stored on message with `from_user` ✅ fully works
- **Conversation notes** → 200, created with author ✅ fully works
- Message read-receipt, mark-read endpoints

### Settings CRUD (all 200)
- Users, Roles, Teams, Tags (create with valid named color), Canned Responses (create with `name`+`content`), Webhooks (create with `name`+`url`), API Keys, Custom Actions, Audit Logs, Organizations, SSO, Chatbot settings/keywords/flows/ai-contexts/transfers, Templates, Flows, Campaigns

### Analytics
- Dashboard analytics (recent messages, counts) — shows the failed QA message as `status:"failed"` (delivery failure correctly tracked)
- Agent analytics (summary + trend data)

---

## 3. ❌ What DOES NOT WORK (with reasons)

### BUG #1 — 🔴 All WhatsApp message delivery fails: GOWA `401 Unauthorized`

**Symptom:** Sending any text/media/template/group message returns HTTP 200 (message saved) but the message shows `"gowa API returned status 401: Unauthorized"` in the chat and ends up `failed`.

**Proof:**
- Browser POST `/api/contacts/{id}/messages` → 200, but UI renders `gowa API returned status 401: Unauthorized`.
- Direct curl to GOWA server with the stored creds works: `curl -u admin:admin123 http://localhost:3080/v1/devices` → 200/400 (NOT 401). So the password `admin123` stored in the DB is correct.
- DB confirms the password: `gowa_instances.password = 'admin123'` for server "1".

**Root cause (code):**
`pkg/whatsapp/registry.go` → `getOrCreateGowa()` builds the GOWA client via `gowaFactory(baseURL)`.
The factory is registered in `cmd/whatomate/main.go:192`:
```go
whatsapp.RegisterGowaFactory(
    func(baseURL string) (string, string) {
        inst := cfg.FindGOWAInstance(baseURL)   // ← reads CONFIG FILE only
        if inst != nil { return inst.Username, inst.Password }
        return "", ""                            // ← falls through to empty creds
    },
    ...)
```
`cfg.FindGOWAInstance()` (`internal/config/config.go:178`) iterates `c.GOWAInstances`, which is populated **only from the `[[gowa_instances]]` TOML section**. There is **no `[[gowa_instances]]` block in `config.toml`** (servers are created via the UI → stored in the DB `gowa_instances` table, never written back to config). So `FindGOWAInstance("http://localhost:3080")` returns `nil` → credentials become `("","")` → Basic Auth header is empty → GOWA returns `401`.

**Why device management works but messaging doesn't:** device handlers (`internal/handlers/gowa_instances.go:78`, `gowa_device.go:194`) build their client directly from the DB row: `gowa.New(inst.BaseURL, inst.Username, inst.Password)`. The message-delivery path instead goes through the `WARegistry`, which uses the config-file resolver. Two different credential sources → inconsistent behavior.

**Fix:** Make the registry's credential resolver read from the DB (`gowa_instances` table) instead of (or in addition to) the config file, e.g. inject a DB-backed resolver at startup, or have `getOrCreateGowa` accept the account and look up its instance's creds from the DB.

---

### BUG #2 — 🔴 Revoke ("Delete for everyone") returns `404 Contact not found`

**Symptom:** Clicking "Delete for everyone" → `POST /api/contacts/{id}/messages/{msg}/revoke` → `404 {"Contact not found"}`.

**Proof (curl, authenticated, valid CSRF):**
```
POST /api/contacts/9c70ab67.../messages/32d86288.../revoke → 404 "Contact not found"
```
The contact exists and is openable in chat, but the revoke handler can't see it.

**Root cause (code):** `internal/handlers/contacts.go:1315`
```go
orgID, _, err := a.getOrgAndUserID(r)          // userID discarded (assigned to _)
...
query = a.scopeAssignedContact(query, uuid.Nil, orgID)   // ← passes uuid.Nil
```
`scopeAssignedContact` (`contacts.go:232`):
```go
if a.HasPermission(userID, models.ResourceContacts, models.ActionRead, orgID) {
    return query   // short-circuit for users with contacts:read
}
// otherwise restrict to assigned/collaborator/transfer contacts
```
With `userID = uuid.Nil`, `HasPermission(uuid.Nil,…)` is false, so it falls into the restrictive branch filtering by `assigned_user_id = '00000000-…'` → matches nothing → `Contact not found`. The contact isn't assigned to anyone (synced contacts have null `assigned_user_id`).

**Fix:** `orgID, userID, err := a.getOrgAndUserID(r)` and `a.scopeAssignedContact(query, userID, orgID)` — i.e. use the real `userID` (exactly as `SendMessage` does at `contacts.go:647`).

---

### BUG #3 — 🔴 Typing indicator returns `404 Contact not found`

**Symptom:** Typing in the composer triggers `POST /api/contacts/{id}/typing` → browser console shows `404 Not Found`.

**Proof:** `POST /api/contacts/9c70ab67.../typing {"action":"start"}` → `404 "Contact not found"` (with valid auth+CSRF).

**Root cause (code):** `internal/handlers/contacts.go:1251` — identical copy-paste of BUG #2:
```go
orgID, _, err := a.getOrgAndUserID(r)                   // userID discarded
query = a.scopeAssignedContact(query, uuid.Nil, orgID)  // ← uuid.Nil
```

**Fix:** same as BUG #2 — pass the real `userID`. (Note: once the contact resolves, this would then hit the BUG #1 GOWA 401 on `SendChatPresence`, so both must be fixed for typing to work end-to-end.)

---

### BUG #4 — 🟡 Accounts page: missing i18n keys + unresolved component

**Symptom (console warnings on `/settings/accounts`):**
```
[Vue warn] Failed to resolve component: Plus
[intlify] Not found 'accounts.serverConnection' key in 'en' locale
[intlify] Not found 'accounts.gowaBaseUrl' key in 'en' locale
[intlify] Not found 'accounts.gowaDeviceId' key in 'en' locale
[intlify] Not found 'common.created' key in 'en' locale
```
**Reason:** The Accounts view was updated to show GOWA columns (Base URL, Device/JID, connection status) but the new translation keys were never added to `frontend/src/i18n/locales/{en,ar}.json`, and the `Plus` icon import is missing in that view. The table still renders (keys fall back to their raw key strings), but labels are wrong and the add button icon is broken.

**Fix:** Add the missing keys to both locale files; import `Plus` from `lucide-vue-next` in `AccountsView.vue`.

---

### BUG #5 — 🟡 "24-hour messaging window expired" banner shown for GOWA accounts

**Symptom:** The chat for `201007181781` shows: *"The 24-hour messaging window has expired. Only template messages can be sent until the customer replies."* even though the account is GOWA (which has no 24h session-window restriction — that's a Meta Cloud API rule).

**Reason:** The 24h-window gate is computed without distinguishing GOWA vs Meta provider type. Sending still works (the composer isn't actually disabled for GOWA), so this is misleading UX rather than a hard block.

**Fix:** Skip the 24h-window banner/check when `account.ProviderType == "gowa"`.

---

## 4. Environment notes

- **Browser MCP instability:** during testing the Playwright/chrome-devtools MCP servers repeatedly lost their Chrome instance due to dozens of stale `playwright-mcp` processes (from prior sessions) contending the singleton profile lock at `~/Library/Caches/ms-playwright-mcp/`. This is a tooling/environment issue, not an app defect. UI verification was completed before the lock contention forced a switch to API-level testing (which exercises the same handlers the UI calls).
- Backend is `go run ./cmd/whatomate server` (live recompile from current source) — not a stale binary; all routes tested are present in the running server.
- `config.toml` has **no** `[gowa]` / `[[gowa_instances]]` section — confirming BUG #1.

## 5. Recommended fix order
1. **BUG #2 & #3** (5-line fix each): use real userID in `RevokeMessage` + `SendTypingIndicator`. Unblocks revoke + typing at the API level.
2. **BUG #1** (architecture): make the GOWA registry credential resolver DB-backed. Unblocks ALL real WhatsApp delivery (text, media, template, group, revoke, typing).
3. **BUG #4 & #5** (polish): i18n keys + `Plus` import; GOWA-aware 24h banner.
