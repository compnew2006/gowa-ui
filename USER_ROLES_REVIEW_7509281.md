# User-Roles / RBAC Gap Review — GOWA + Media Feature Drop

**Repository:** `whatomate`
**Commit range:** `7509281a92e2399f52bb6ba8c8c262af7c8acd0d` → `829ecf70b1fe64e49233cba43395ff935f73301f` (HEAD)
**Range span:** 16 commits · 69 files changed · +8,928 / −214 lines
**Method:** 6 parallel sub-agents (deep review), each scoped to an independent domain, cross-checked against the project's documented RBAC standard.
**Date:** 2026-07-12

---

## TL;DR — Executive summary

The 16-commit drop adds a **GOWA provider** (multi-instance device management + webhooks), **media burst/zip/redownload**, and **group-message routing**. The backend's RBAC enforcement is **handler-level** (there is no route-level permission middleware — the `Before` RBAC hook at `cmd/whatomate/main.go:614-625` is a documented **no-op**). This means every protected handler *must* self-gate by calling `a.requireAuth(r, models.Resource<X>, models.Action<Y>)` (`internal/handlers/app.go:265`).

**The five new GOWA device-management handlers do not call `requireAuth` at all.** They use `getOrgID` (auth-only, 401, no permission check) instead. Net effect: **any authenticated user — including a lowest-privilege `agent` — can provision GOWA devices, retrieve pairing QR codes, trigger phone pairing, and poll device status on any account in their org.** The `GowaCreateDevice` endpoint additionally hands out the `webhook_secret` and is not org-scoped, so a user from org A can provision devices against instances configured for org B.

Separately, the **GOWA inbound-webhook handler fails open**: HMAC verification is skipped both when the signature header is missing *and* when the account has no `GowaWebhookSecret`. Combined with a globally unscoped device-id lookup, this allows an **unauthenticated attacker to inject messages/contacts, trigger chatbot auto-replies, and mutate message state in any tenant.** The `main.go:543` comment claiming "HMAC verified in handler" is **false**.

The **media zip / redownload** handlers and the **modified core handlers** (accounts, contacts, messages, webhook, chatbot_processor) are **org-scoped correctly** — no IDOR. They reuse `getOrgAndUserID` + manual `HasPermission` instead of the `requireAuth` primitive, which is a **codebase-wide convention** (the sibling contact/chat readers do the same), so it is consistent but deviates from the documented standard. The frontend mostly gates correctly except the new GOWA "Connect Device" button and the media-export/redownload affordances.

### Severity counts

| Severity | Count |
|----------|-------|
| CRITICAL | 8 |
| HIGH | 6 |
| MEDIUM | 9 |
| LOW | 4 |
| **Total findings** | **27** |

### What is correct (no action needed)

- **IDOR / org-scoping on account-scoped GOWA device handlers** (`GowaLoginQR`, `GowaPairCode`, `GowaDeviceStatus`) — `resolveGowaAccount` → `resolveWhatsAppAccountByID` → `findByIDAndOrg` filters by `organization_id`. Cross-org account access is blocked. (The gap is *permission*, not *scope*.)
- **Media handlers IDOR** — every message lookup is scoped by `organization_id` (`media_zip.go:69`, `media_redownload.go:43`, `media.go:212`). Per-contact ownership (`scopeAssignedContact` + team-transfer) is enforced. Enumerating message IDs yields nothing cross-org. `TestServeMediaZip_OrgIsolation` + `TestApp_ServeMedia_CrossOrgIsolation` verify this.
- **Modified core handlers** — no RBAC regressions. Provider abstraction (`a.WhatsApp.X` → `provider.X`) preserves `requireAuth` preambles and org-scoped queries throughout `accounts.go`, `contacts.go`, `messages.go`, `webhook.go`, `chatbot_processor.go`, `campaigns.go`.
- **Meta webhook signature verification** — unchanged and intact (`webhook.go:165-192`).
- **`GowaWebhookSecret` encrypted at rest** — added to `EncryptFields` (`accounts.go:1049`).

---

## The project RBAC standard (baseline used for this review)

Every protected HTTP handler MUST open with:

```go
orgID, userID, err := a.requireAuth(r, models.Resource<X>, models.Action<Y>)
if err != nil {
    return nil // errEnvelopeSent already sent (401 or 403)
}
```

- `requireAuth` (`internal/handlers/app.go:265-276`) = `getOrgAndUserID` (401 if unauthenticated) **+** `HasPermission(userID, resource, action, orgID)` (403 if permission missing).
- `getOrgID` / `getOrgAndUserID` = **auth-only** (401). They do **not** check any permission. Using them as the sole gate is insufficient for any non-public endpoint.
- Resources & actions are constants in `internal/models/roles.go` (`ResourceAccounts`, `ResourceContacts`, `ResourceChat`, …; `ActionRead`, `ActionWrite`, `ActionDelete`, `ActionExport`, …).
- `DefaultPermissions()` (`roles.go:104`) seeds the permission table; `SystemRolePermissions()` (`roles.go:242`) maps `admin`/`manager`/`agent` → allowed `"resource:action"` strings. `admin` = all; `manager` = curated; `agent` = minimal.
- The route-level RBAC `Before` middleware (`main.go:614-625`) is a **no-op** by design (comment at `:622-623`: "Route-level permission checks are now handled at the handler level"). **There is no fallback gate.** If a handler skips `requireAuth`, any authenticated user passes.
- IDOR rule: handlers fetching a record by `{id}` MUST scope the DB query to `orgID` (e.g. `WHERE id = ? AND organization_id = ?`).

Established baseline citations (handlers that do it right):
- `internal/handlers/accounts.go:83,114,229,354,652,952` — all call `requireAuth(r, ResourceAccounts, ActionRead/Write/Delete)`.
- `internal/handlers/apikeys.go`, `teams.go`, `users.go`, `tags.go`, `ivr_flows.go`, `call_transfers.go`, `outgoing_calls.go`, `audit_logs.go` — same pattern.

---

## CRITICAL findings

### C1. GOWA device handlers ship with zero permission gating
**Files:** `internal/handlers/gowa_device.go:24,61,102,133,159,217`
**Routes:** `GET /api/accounts/{id}/gowa/qr`, `POST /api/accounts/{id}/gowa/pair-code`, `GET /api/accounts/{id}/gowa/status`, `GET /api/gowa/instances`, `POST /api/gowa/create-device` (registered at `cmd/whatomate/main.go:671-677`, inside the authenticated group)

All five handlers resolve identity via `getOrgID` (auth-only) instead of `requireAuth`. `getOrgID` (`app.go:61`) never calls `HasPermission`. The RBAC `Before` middleware is a no-op, so **nothing** checks the caller's role/permission.

| Handler | Route | Should use | Actually does |
|---|---|---|---|
| `GowaLoginQR` (`:61`) | `GET /api/accounts/{id}/gowa/qr` | `accounts:write` | `getOrgID` only |
| `GowaPairCode` (`:102`) | `POST /api/accounts/{id}/gowa/pair-code` | `accounts:write` | `getOrgID` only |
| `GowaDeviceStatus` (`:217`) | `GET /api/accounts/{id}/gowa/status` | `accounts:read` | `getOrgID` only |
| `GowaInstances` (`:132`) | `GET /api/gowa/instances` | `accounts:read` | `getOrgID` only |
| `GowaCreateDevice` (`:158`) | `POST /api/gowa/create-device` | `accounts:write` | `getOrgID` only |

**Impact:** Any authenticated `agent` (lowest privilege) can drive the full GOWA device-pairing lifecycle on any GOWA account in their org, and provision new devices. Compare to `CreateAccount`/`UpdateAccount`/`DeleteAccount` (`accounts.go:114,229,354`) which all enforce `accounts:write`/`delete`.

**Confirmed independently by 3 agents** (device auditor, frontend auditor, catalog auditor).

**Fix:** Replace `getOrgID` with the `requireAuth` form at the top of each handler, e.g.:
```go
orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionWrite) // for pair/qr/create
if err != nil { return nil }
```
`GowaDeviceStatus` / `GowaInstances` → `ActionRead`.

---

### C2. `GowaCreateDevice` is not org-scoped — cross-org device provisioning
**File:** `internal/handlers/gowa_device.go:158-213` (esp. `:178` `FindGOWAInstance`, `:208-212` returns `webhook_secret`)

`GowaCreateDevice` calls `a.Config.FindGOWAInstance(baseURL)` which selects a GOWA instance by the caller-supplied `base_url` and returns its credentials — **ignoring `orgID` entirely**. A user from org A can target an instance configured for org B, provision a device, and receive the `webhook_secret` (a global GOWA credential, not org-scoped).

**Impact:** Cross-tenant infrastructure abuse + credential disclosure. Combined with C1, any authenticated user in any org can do this.

**Fix:** Org-scope instance resolution (map instances to orgs, or restrict the selectable instance set by `orgID`), and gate with `requireAuth(r, ResourceAccounts, ActionWrite)`.

---

### C3. `GowaInstances` leaks global infrastructure config to every user
**File:** `internal/handlers/gowa_device.go:132-153`

Returns the full list of configured GOWA instances (names + base URLs) from `a.Config.GOWAInstances` to **any authenticated user** with no permission check and no org filtering. Every org sees the full instance topology.

**Impact:** Infrastructure enumeration / info leak. Should be `accounts:read` minimum, and arguably admin/manager-only since it leaks service topology.

---

### C4. GOWA webhook HMAC verification is bypassable (fail-open)
**File:** `internal/handlers/gowa_webhook.go:72`

The verification guard is:
```go
if sigHeader != "" && account.GowaWebhookSecret != "" {
    if !gowa.VerifyWebhookSignature(rawBody, sigHeader, account.GowaWebhookSecret) {
        ...return 403
    }
}
```
Verification is **completely skipped** if either:
1. The request omits the `X-Hub-Signature-256` header (`sigHeader == ""`) — **unconditional bypass**: an attacker POSTs a forged payload with no signature header and sails through to `processGowaMessage` (`gowa_webhook.go:83`).
2. The account has no `GowaWebhookSecret` configured — the field (`models.go:314`) has no DB default; `CreateAccount` (`accounts.go:130-133`) does not require it; `UpdateAccount` (`accounts.go:298-304`) only writes it `if req.GowaWebhookSecret != ""`. Only the device-provisioning flow (`gowa_device.go:195`) generates one.

The `main.go:543` comment "HMAC verified in handler" is **false**. The correct pattern is fail-closed: `if secret == "" || header == "" || !verify → reject`.

The underlying `gowa.VerifyWebhookSignature` (`pkg/gowa/verify.go:45-67`) is itself sound (constant-time HMAC-SHA256, correct `sha256=` prefix handling) — and is well unit-tested in `pkg/gowa/webhook_test.go:108-148`. The defect is purely the handler's optional guard.

---

### C5. GOWA webhook `device_id` is attacker-controlled and globally resolved
**File:** `internal/handlers/gowa_webhook.go:34,52-54,62-68,106-154`; lookup query `:113-116` has no `organization_id` clause

The account is resolved from `envelope.DeviceID` (body) or `pathDeviceID` (URL path, which **overrides** the body, `:52-54`) via `getGowaAccountByDeviceID`. The lookup matches `gowa_device_id`, `phone_id`, or phone-digit across **all orgs globally**. The route is public (`main.go:548`, skipped at `:567`). Since verification is bypassable (C4), an attacker controls which account/org is selected.

**Impact:** An unauthenticated attacker can target any registered GOWA device by path/body and route processing into that org.

---

### C6. Cross-tenant message/contact injection via forged GOWA webhook
**Files:** `internal/handlers/gowa_webhook.go:337,426-427,467`; `internal/handlers/chatbot_processor.go:162,209,223+`

Once an account is resolved (spoofable per C5), every downstream write uses `account.OrganizationID` as the org context, and the spoofed account is attacker-chosen:
- **Contact creation** — `contactutil.GetOrCreateContact(a.DB, account.OrganizationID, recipientPhone, "")` (`gowa_webhook.go:337`) creates a contact in the victim org.
- **Message DB row** — `outgoing`/`incoming` records use `OrganizationID: orgID` (`gowa_webhook.go:426-427`) → injects messages into the victim org's chat history.
- **Chatbot auto-reply** — a forged inbound message triggers an **automated outbound reply from the victim's WhatsApp number** (`chatbot_processor.go:223+`). An attacker can make the victim org spam arbitrary contacts.
- **WebSocket broadcast** to the victim org's frontend (`gowa_webhook.go:594`).

**Impact:** Full tenant contamination: injected messages, fabricated contacts, weaponized auto-replies, UI spoofing.

---

### C7. reaction / revoked / edited / ack DB updates are org-unscoped
**Files:** `internal/handlers/gowa_webhook.go:654,702`; `internal/handlers/webhook.go:407`; `internal/handlers/chatbot_processor.go:1297`

Status/reaction/revoked/edited updates match solely on `whats_app_message_id` with no `organization_id` clause:
- `processGowaRevoked`: `Where("whats_app_message_id = ?", revoked.RevokedMessageID)` (`gowa_webhook.go:654`)
- `processGowaEdited`: `Where("whats_app_message_id = ?", edited.OriginalMessageID)` (`gowa_webhook.go:702`)
- `processGowaAck` → `updateMessageStatus`: `Where("whats_app_message_id = ?", whatsappMsgID)` (`webhook.go:407`)
- `processGowaReaction` → `handleIncomingReaction` (`chatbot_processor.go:1297`)

A forged webhook supplying any `reacted_message_id` / `revoked_message_id` / `original_message_id` / ack `id` can rewrite message content to `"[message revoked]"` (`gowa_webhook.go:658`), set statuses, or attach reactions to **any message in any org** whose WhatsApp message ID is known/guessed.

---

### C8. RBAC catalog has no `devices`/`gowa` resource — even fixing the handlers leaves them unseedable
**File:** `internal/models/roles.go:48-88` (constants), `:104-239` (`DefaultPermissions`), `:242-319` (`SystemRolePermissions`)

There is no `ResourceDevices` / `ResourceGOWA` constant, no `devices:*` permission seeded, and no `devices:*` mapping for any role. So even if the handlers were fixed to call `requireAuth(r, ResourceDevices, ActionWrite)`, `HasPermission` would **always return false** for everyone (including the admin DB-role check) because the permission is unseeded. The practical workaround is to reuse the existing `ResourceAccounts` (`accounts:read`/`write`/`delete` already seeded and mapped to admin/manager; `accounts:read` to agent) — but that makes `GowaInstances` (which exposes infra base URLs) reachable by agents, and conflates device-provisioning with account CRUD.

**Recommended:** Add `ResourceDevices = "devices"`; seed `devices:read` + `devices:write` in `DefaultPermissions()`; map `devices:write` to admin+manager only (device pairing/provisioning emits a webhook secret — never agent), `devices:read` to admin+manager (agents should not enumerate infra). `GowaInstances` specifically should be admin/manager-only.

---

## HIGH findings

### H1. `GowaCreateDevice` / `GowaInstances` have **no tests**
**File:** `internal/handlers/gowa_device_test.go` (entire file)

The test file has **zero** tests for `GowaCreateDevice` and `GowaInstances`. The 6 tests that exist are happy-path or input-validation (400) only. No test asserts 401 (unauthenticated), 403 (insufficient permission / wrong role), or cross-org (IDOR) rejection. The strings "permission", "forbidden", "403" do not appear anywhere in the file.

### H2. GOWA webhook handler has **no handler-level security tests**
**File:** `internal/handlers/gowa_webhook_test.go` (entire file)

All tests are payload-parsing/normalization tests that construct a `gowa.WebhookPayload` directly from JSON literals and assert on decoded struct fields. None invoke `GowaWebhookHandler`/`handleGowaWebhook`. No test covers: signature rejection (missing header, tampered body, wrong secret), the `GowaWebhookSecret == ""` fail-open path, cross-device/cross-org spoofing, or replay. Only the pure `VerifyWebhookSignature` fn is unit-tested (`pkg/gowa/webhook_test.go:108-148`).

### H3. No replay protection on GOWA webhooks
**Files:** `internal/handlers/gowa_webhook.go` (no checks); `pkg/gowa/webhook.go:18,29`

No nonce/timestamp validation. `envelope.Timestamp` and `msg.Timestamp` are parsed for storage only, never checked for freshness. Message-level idempotency exists only for `message` events (`webhook.go:357-363` dedupes by `msg.ID`) — ack/connection/revoked/edited events are replayable. Replaying a `connection: logout` event flips an account to `disconnected` (`gowa_webhook.go:573-575`); replaying revoked/edited re-mutates messages.

### H4. `RedownloadMedia` has **no tests at all**
**Missing file:** `internal/handlers/media_redownload_test.go` (does not exist)

A handler that overwrites `message.media_url` (`media_redownload.go:115`) and makes outbound (rate-limited, 60s-timeout) provider calls (`:81-85`) is entirely untested for authz, IDOR, error mapping, and provider interaction.

### H5. Frontend "Connect Device" button visible to agents
**File:** `frontend/src/views/settings/AccountDetailView.vue:490`

```html
<Button v-if="!isNew && account && isGowa" ... @click="openGowaConnect">
```
The `v-if` checks `isGowa` and `account` but **omits `canWrite`**. An agent who can read the account-detail page (route gated by `accounts:read` at `router/index.ts:183`) sees and can click "Connect Device", which fires the ungated backend endpoints (C1). Compare the adjacent Save button at `:486` which correctly uses `v-if="canWrite && (hasChanges || isNew)"` — the inconsistency is specifically the new GOWA buttons. `canWrite` is already computed at `:128`.

### H6. `GowaDeviceStatus` has no permission check
**File:** `internal/handlers/gowa_device.go:217-236`

(Subset of C1, called out separately because it is a *read* op with a different recommended action.) Any authenticated user can poll the connection state of any GOWA account in their org. Should be `accounts:read`.

---

## MEDIUM findings

### M1. `ServeMediaZip` / `RedownloadMedia` use `getOrgAndUserID` instead of `requireAuth`
**Files:** `internal/handlers/media_zip.go:34,81`; `internal/handlers/media_redownload.go:29,48`

Both use auth-only `getOrgAndUserID` then do the permission check manually via `HasPermission(ResourceContacts, ActionRead)` + `canAccessContactMedia` in the body. **Effective authz is present** (and IDOR is solid — see "What is correct"), but this deviates from the documented `requireAuth` standard. Note: this matches the established convention of the sibling contact/chat readers (`contacts.go:229,258` also use `getOrgAndUserID`), so it is a **codebase-wide convention**, not a one-off — but the convention itself contradicts the standard.

### M2. `RedownloadMedia` lacks a per-endpoint rate limit / cooldown
**File:** `internal/handlers/media_redownload.go:81-85`

`RedownloadMedia` triggers an outbound provider call per request (60s timeout) and overwrites `media_url` on every call (`:108-118`). Any `contacts:read` user can repeatedly hammer the GOWA provider for any media message in their org. Only the generic global `UserAwareRateLimit` (`main.go:596`) applies. Add a per-message cooldown or per-user rate limit.

### M3. `ServeMediaZip` has no total-byte-size cap (DoS)
**File:** `internal/handlers/media_zip.go:23,103-104,154`

Count cap is enforced (`maxZipMessageIDs = 50`, `:45`), checked on the raw split *before* UUID parsing (conservative). But there is **no cap on total archive byte size** — the whole ZIP is buffered in memory (`buf := &bytes.Buffer{}` at `:103`). 50 large video/PDF entries per concurrent request could allocate sizable buffers.

### M4. Information leak: distinct responses enable device enumeration
**File:** `internal/handlers/gowa_webhook.go:67 vs :75`

Unknown `device_id` returns `200 {"status":"device_not_configured"}` (`:67`); bad signature returns `403 "Invalid signature"` (`:75`). Combined with the bypassable check (C4), an attacker can enumerate valid device IDs (a known-good device yields `200 ok`).

### M5. Unauthenticated fallback triggers outbound calls to every tenant's GOWA server
**File:** `internal/handlers/gowa_webhook.go:122-142`

`getGowaAccountByDeviceID` fallback iterates **all GOWA accounts across all orgs** and calls `GetAppStatus` on each one's GOWA instance. An unauthenticated POST with a random `device_id` triggers outbound API calls to every tenant's GOWA server — SSRF/amplification/oracle vector.

### M6. `RedownloadMedia` performs a DB write gated only by `contacts:read`
**File:** `internal/handlers/media_redownload.go:48` (gate) and `:115` (write)

A read permission authorizes a state-changing operation (`Updates(updates)` on `media_url`/`media_mime_type`). Functionally acceptable (read-implies-redownload-fetch), but it is a write gated by a read permission — note for consistency. If product wants stricter control, gate on `contacts:write` or a dedicated redownload perm.

### M7. Frontend media-export / burst / redownload affordances are ungated
**Files:** `frontend/src/views/chat/ChatView.vue:1979-1999,2081-2093,2196-2259`; `frontend/src/components/chat/MediaBurstDialog.vue:107,111`; `frontend/src/components/chat/MediaRetryButton.vue:25-34`; `frontend/src/composables/useMediaExport.ts:44,68`

The "Collect files" toolbar button + floating chip, the MediaBurstDialog ZIP/separate buttons, and the MediaRetryButton are rendered for every chat user with no `hasPermission` guard. `canWriteContacts` is already computed at `ChatView.vue:123` — the pattern is known but not applied. The backend gates these on `contacts:read` (so no 403-on-click), which is a defensible policy — but bulk ZIP download and provider re-fetch are abuse surfaces. **Confirm the policy is intended**; if restricted, gate with `hasPermission('contacts','export')` (zip) and `hasPermission('contacts','write')` or a dedicated redownload perm.

### M8. RBAC `Before` middleware is a documented no-op
**File:** `cmd/whatomate/main.go:614-625` (esp. comment `:622-623`)

"Route-level permission checks are now handled at the handler level." This means the C1–C3 gaps are **not mitigated by any route-level fallback** — the handler-level `requireAuth` is the only gate. Worth noting because it raises the stakes of every missing `requireAuth` call.

### M9. `GowaWebhookSecret` not required at account create/update
**Files:** `internal/handlers/accounts.go:130-133` (create), `:298-304` (update)

`CreateAccount` validation does not require `GowaWebhookSecret`; `UpdateAccount` only writes it if non-empty. Any GOWA account created without a secret accepts unsigned inbound webhooks (compounds C4). Enforce generation/requirement at create time.

---

## LOW findings

### L1. No dedicated `ResourceMedia` RBAC resource
**File:** `internal/models/roles.go`

Media access is overloaded onto `ResourceContacts`/`ResourceChat`. If product ever needs "can read chat but not download media" or "bulk-zip = export", the current model cannot express it (zip is gated by `contacts:read`, not `ActionExport`). Architectural note, not an exploit.

### L2. `handleIncomingReaction` GOWA `LIKE` fallback is org-unscoped (pre-existing)
**File:** `internal/handlers/chatbot_processor.go:1327`

New `LIKE` fallback for non-FQIA IDs has no `organization_id` filter, matching the pre-existing org-unscoped FQIA-suffix `LIKE` at `:1317`. Observable effect is confined to the receiving org (contact re-resolved org-scoped at `:1335`; broadcast org-scoped at `:1390`). Not a regression; latent correctness/IDOR smell.

### L3. `getWhatsAppAccountCached` query is org-unscoped (pre-existing)
**File:** `internal/handlers/cache.go:242-245`

Broadened from `phone_id = ?` to `phone_id = ? OR gowa_device_id = ? OR gowa_device_id = ?` with no `organization_id` filter. Lookup key is webhook-derived and signature-gated upstream, so practical risk is low. Original query was already org-unscoped; the diff widens match columns but does not introduce the unscoped nature. Not a regression.

### L4. `media_zip_test.go` missing the 403 / 401 / team-transfer positive test cases
**File:** `internal/handlers/media_zip_test.go`

Has `TestServeMediaZip_OrgIsolation` (cross-org IDOR) and `TestServeMediaZip_NoIDs` (400). Missing: the `canAccessContactMedia == false` 403 branch (`media_zip.go:89-91`), the unauthenticated-401 case, and the team-transfer positive case. Compare `media_test.go` which has the full matrix.

---

## Per-handler / per-file verdict matrix

### Backend — new GOWA device handlers (`internal/handlers/gowa_device.go`)
| Handler | Route | `requireAuth` | resource/action | org-scope (IDOR) | authz tests | Verdict |
|---|---|---|---|---|---|---|
| `GowaLoginQR` (`:61`) | `GET /api/accounts/{id}/gowa/qr` | **MISSING** | none | present (`:35`) | MISSING | **CRITICAL** (C1) |
| `GowaPairCode` (`:102`) | `POST /api/accounts/{id}/gowa/pair-code` | **MISSING** | none | present (`:35`) | MISSING | **CRITICAL** (C1) |
| `GowaDeviceStatus` (`:217`) | `GET /api/accounts/{id}/gowa/status` | **MISSING** | none | present (`:35`) | MISSING | **HIGH** (H6) |
| `GowaInstances` (`:132`) | `GET /api/gowa/instances` | **MISSING** | none | N/A (global config) | MISSING | **CRITICAL** (C1, C3) |
| `GowaCreateDevice` (`:158`) | `POST /api/gowa/create-device` | **MISSING** | none | **MISSING** (`:178`) | MISSING | **CRITICAL** (C1, C2) |

### Backend — GOWA webhook (`internal/handlers/gowa_webhook.go`)
| Handler/aspect | Verdict | Finding |
|---|---|---|
| HMAC verification | **CRITICAL** | fail-open on missing header or empty secret (C4) |
| device_id resolution | **CRITICAL** | attacker-controlled, globally unscoped (C5) |
| cross-tenant writes | **CRITICAL** | contact/message/bot-reply injection (C6) |
| reaction/revoked/edited/ack DB updates | **HIGH** | org-unscoped, mutate any message by ID (C7) |
| replay protection | **HIGH** | missing (H3) |
| info leak | **MEDIUM** | device enumeration (M4) |
| SSRF amplification | **MEDIUM** | fallback calls every tenant's GOWA (M5) |
| handler-level tests | **HIGH** | none (H2) |

### Backend — media handlers
| Handler | Route | `requireAuth` | org-scope (IDOR) | tests | Verdict |
|---|---|---|---|---|---|
| `ServeMediaZip` (`media_zip.go:33`) | `GET /api/media/zip` | deviates (M1) | **correct** (`:69`) | partial (L4) | OK on IDOR; M1 |
| `RedownloadMedia` (`media_redownload.go:28`) | `POST /api/media/{message_id}/redownload` | deviates (M1) | **correct** (`:43`) | **none** (H4) | OK on IDOR; H4, M2, M6 |
| `ServeMedia` (`media.go:197`, modified) | `GET /api/media/{message_id}` | pre-existing pattern | **correct** (`:212`) | strong | **OK — no regression** |

### Backend — modified core handlers
| File | Verdict | Notes |
|---|---|---|
| `internal/handlers/accounts.go` | **OK** | `requireAuth` + org-scope preserved; GOWA fields added correctly; `GowaWebhookSecret` encrypted (`:1049`). Validation gap M9. |
| `internal/handlers/contacts.go` | **OK** | `getOrgAndUserID` + `ScopeToOrg` preserved; GOWA branches operate on org-scoped contact/account. |
| `internal/handlers/messages.go` | **OK** | provider swap only; entry points retain auth + org-scoped contact lookup. |
| `internal/handlers/webhook.go` | **OK** | Meta signature verification intact; new args signal "Meta is never group". |
| `internal/handlers/chatbot_processor.go` | **OK** | `account.OrganizationID` used correctly; pre-existing org-unscoped `LIKE` (L2). |
| `internal/handlers/media.go` | **OK** | `ServeMedia` auth/IDOR unchanged; GOWA branch is in internal `DownloadAndSaveMedia`, not the handler. |
| `internal/handlers/cache.go` | **OK** | pre-existing org-unscoped lookup (L3); not a regression. |
| `internal/handlers/campaigns.go` | **OK** | provider swap only; `requireAuth` preserved. |
| `internal/handlers/helpers.go` | **OK** | pure string/map helpers. |
| `internal/handlers/agent_transfers.go` | **OK** | group-chat bot suppression; per-org settings respected. |

### RBAC catalog (`internal/models/roles.go`)
| Aspect | Verdict | Finding |
|---|---|---|
| `ResourceDevices`/`ResourceGOWA` constant | **MISSING** | C8 |
| `devices:*` seeded in `DefaultPermissions()` | **MISSING** | C8 |
| `devices:*` mapped in `SystemRolePermissions()` | **MISSING** | C8 |
| Media handlers reuse `contacts:read` | **OK** | seeded + mapped to manager/agent |

### Frontend
| File | New actions | Gating | Verdict |
|---|---|---|---|
| `views/settings/AccountDetailView.vue` | GOWA connect/pair/status/create | Connect Device button missing `canWrite` (`:490`) | **HIGH** (H5) |
| `views/chat/ChatView.vue` | collect-files, retry-download, group badge | collect/retry ungated | **MEDIUM** (M7) |
| `components/chat/MediaBurstDialog.vue` | ZIP / separate download | ungated | **MEDIUM** (M7) |
| `components/chat/MediaRetryButton.vue` | retry (provider re-fetch) | ungated | **MEDIUM** (M7) |
| `composables/useMediaExport.ts` | zip/separate/redownload | no perm check (delegates to call site) | **MEDIUM** (M7) |
| `composables/useMediaBurst.ts` | none (detection only) | N/A | **OK** |
| `views/settings/AccountsView.vue` | none (display only) | existing `canWrite`/`canDelete` intact | **OK** |
| `components/chat/ContactInfoPanel.vue` | none (display only) | N/A | **OK** |
| `stores/contacts.ts` | none (interface fields only) | N/A | **OK** |
| `services/websocket.ts` | none (passthrough fields) | N/A | **OK** |

---

## Recommended fix sequence

1. **(C4) Fail-close the GOWA webhook HMAC guard** — `internal/handlers/gowa_webhook.go:72`. Reject when `secret == ""` *or* header missing *or* verify fails. This is the single highest-leverage fix; it collapses C5/C6/C7 exploitability from "unauthenticated" to "requires valid signature."
2. **(M9) Require `GowaWebhookSecret` at account create** — `internal/handlers/accounts.go:130-133`. Ensures no GOWA account is ever webhook-unprotected.
3. **(C1) Add `requireAuth(r, ResourceAccounts, ActionRead/Write)`** to all five GOWA device handlers — `internal/handlers/gowa_device.go:24,61,102,133,159,217`.
4. **(C2) Org-scope `GowaCreateDevice` instance resolution** — `internal/handlers/gowa_device.go:178`.
5. **(C7) Add `organization_id` clause** to reaction/revoked/edited/ack DB updates — `gowa_webhook.go:654,702`; `webhook.go:407`; `chatbot_processor.go:1297`.
6. **(C8) Extend the RBAC catalog** — add `ResourceDevices` + seed + role mappings in `internal/models/roles.go` (or commit to reusing `accounts:*` and accept the agent-can-list-instances tradeoff).
7. **(H5) Add `&& canWrite`** to the GOWA-action `v-if`s in `AccountDetailView.vue:490` and the provisioning block.
8. **(H3) Add replay protection** (nonce/timestamp window) to the GOWA webhook handler.
9. **(H1, H2, H4) Add authz/IDOR/spoofing tests** for `GowaCreateDevice`, `GowaInstances`, the GOWA webhook handler-level security paths, and `RedownloadMedia`.
10. **(M7, M2) Decide the media-export/redownload policy** and gate the frontend affordances + add a cooldown on `RedownloadMedia` accordingly.

---

## Methodology & scope notes

- **Repo location:** The `New one/` top-level dir is not a git repo; the whatomate repo lives at `New one/whatomate/`. Verified `7509281a92…` is an ancestor of HEAD `829ecf70b1…` (16 commits, 69 files, +8928/−214).
- **Swarm:** 6 Explore sub-agents dispatched in parallel, each with the RBAC standard embedded and a focused, self-contained scope (GOWA device handlers / GOWA webhook / media handlers / modified-core diff / RBAC catalog / frontend). Findings were cross-checked for consistency — three agents independently flagged the GOWA device `requireAuth` gap (C1), increasing confidence.
- **Out of scope:** The `MetaAI-Free-Hermes-Agent/` repo (separate, does not contain the target commit). Non-RBAC concerns (e.g. GOWA base URL defaulting to `http://localhost:3000` in `media.go:102`) noted only where they compound an authz issue.
- **Read-only audit:** No files were modified. This report is the sole artifact.
