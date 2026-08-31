# gowa-ui → VPS deployment summary (separate service)

**Date:** 2026-08-12
**Target VPS:** 31.97.192.53 (root)
**Result:** ✅ gowa-ui deployed as a **separate** service. **whatomate (production) was NOT touched.**

> ⚠️ **Important correction to the original task framing:**
> The local working folder `~/Downloads/whatomate` is actually the **gowa-ui** project
> (`compnew2006/gowa-ui`, GOWA-provider based). The VPS production app `ofuqalmadenah.com`
> runs a **different** project, **whatomate** (`compnew2006/whatomate`, whatsmeow/Cloud-API based).
> Per your explicit choice ("خدمة منفصلة — آمن"), gowa-ui was deployed as a NEW, separate
> service. It did **not** replace whatomate (which would have broken the live site).

---

## What was deployed

| Item | Value |
|---|---|
| Project | gowa-ui (`compnew2006/gowa-ui`), commit `d6cb0922` |
| On VPS | `/opt/gowa-ui/` (binary `gowa-ui`, 32 MB, statically linked linux/amd64) |
| Service | `gowa-ui.service` (systemd), enabled + active |
| Port | **8081** (whatomate uses 18123 — untouched) |
| Database | **`gowa_ui`** (separate role + DB; whatomate uses `whatomate_prod` — untouched) |
| Redis | db index **2** (whatomate uses db 0 — untouched) |
| Config | `/opt/gowa-ui/config.toml` (chmod 600) |
| Frontend | embedded in the binary |
| Environment | `staging` (so auth cookies work over plain HTTP — see Notes) |

## What was NOT touched (production safety)

- `/opt/whatomate*`, `whatomate.service`, `whatomate@holol-wenjaz.service` — **untouched, still running** (verified HTTP 200 on `:18123`).
- `whatomate_prod` / `whatomate_holol_wenjaz` databases — **untouched**.
- GOWA servers `/opt/gowa-main`, `/opt/gowa-holol` (`gowa-main.service`, `gowa-holol.service`) — **untouched**.
- No destructive operations were performed → **no backup was required/created** (nothing existing was modified or deleted).

## Bugs found & fixed during deploy (in gowa-ui code)

1. **`provider_type` column reference** — `BackfillGowaWebhookSecrets` queried a `provider_type`
   column that no longer exists on the `WhatsAppAccount` model (worked locally only because of a
   stale column). Crashed against a fresh DB. Fixed: filter by `gowa_device_id <> ''` instead.
   → `internal/handlers/gowa_backfill.go`
2. **Secure cookies over HTTP** — `environment = "production"` auto-enables `secure` cookies,
   which browsers reject on plain HTTP → login didn't persist. Set `environment = "staging"` for
   this HTTP test instance (non-secure cookies → login works).

## Access

- **URL:** http://31.97.192.53:8081
- **Admin login:** `admin@gowa-ui.local`
- **Admin password:** stored on the VPS at `/opt/gowa-ui/.deploy-secrets` (chmod 600). Retrieve with:
  ```bash
  ssh root@31.97.192.53 'cat /opt/gowa-ui/.deploy-secrets'
  ```

## Manage the service

```bash
# on the VPS:
systemctl status gowa-ui        # health
systemctl restart gowa-ui       # restart (re-runs migrations due to -migrate flag)
journalctl -u gowa-ui -f        # live logs
systemctl stop gowa-ui          # stop
systemctl disable gowa-ui       # stop + disable auto-start
```

## Verification results

| Check | Result |
|---|---|
| `curl :8081/` | HTTP 200 |
| `curl :8081/login` | HTTP 200 (UI renders) |
| External reachability (Mac → http://31.97.192.53:8081) | HTTP 200 |
| `gowa_ui` DB tables | 30 tables, all owned by `gowa_ui`, incl. `gowa_webhook_events` (durable inbox), `gowa_instances` |
| `POST /api/auth/login` | HTTP 200 (credentials valid, DB-backed auth works) |
| `GET /api/me` (with cookie) | HTTP 200 (session persists) |
| Browser login → dashboard | ✅ renders (sidebar: Dashboard / Chat / Campaigns / Agent Analytics / Settings) |
| Background processors | all running, incl. **GOWA webhook inbox processor** (5s poll + wake) |
| whatomate production (`:18123`) | **HTTP 200, active 6+ days — untouched** |

## Next steps (not done — ask if you want them)

1. **HTTPS / real domain**: put gowa-ui behind nginx with TLS (then switch `environment = "production"`
   so secure cookies work). Until then it's HTTP on a bare IP.
2. **Configure the GOWA instance** inside gowa-ui (Settings → GOWA Servers): add the GOWA server
   (`gowa-main`, reachable at `gowa.ofuqalmadenah.com` or the VPS internal address) with its Basic
   Auth creds. gowa-ui is GOWA-provider based and needs this to send/receive.
3. **Webhook URL**: for real-time chat in gowa-ui, the GOWA server must be able to reach this
   instance's webhook (`http://31.97.192.53:8081/api/gowa/webhook/...` or the HTTPS domain).

## How it was built (reproduce)

```bash
cd ~/Downloads/whatomate
(cd frontend && npm run build)
rm -rf internal/frontend/dist/* && cp -r frontend/dist/* internal/frontend/dist/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags "-s -w -X main.Version=$(git describe --tags --always) -X main.BuildTime=$(date -u +%Y-%m-%d_%H:%M:%S)" \
  -o gowa-ui-linux ./cmd/gowa-ui
```

---

## GOWA Gateway canonical lifecycle (Plan B) — 2026-08-13

**Objective:** gowa-servers page = canonical device lifecycle source; account page = business config + live status; close 5 gaps; `graphify update .`

**Files changed:**
- `frontend/src/views/settings/AccountDetailView.vue` — removed duplicate QR/pair/status; live 5s status poll; Webhook Configuration section; Manage Device / GOWA Gateway links; MetadataPanel (fields `created_by_name`/`updated_by_name`)
- `frontend/src/views/settings/GowaServerDetailView.vue` — deep-link `?device=X&connect=1` → pair dialog + scroll + ring highlight
- `frontend/src/views/settings/AccountsView.vue` — GOWA Gateway card, row link button (matched by trimmed base_url), add-number buttons retargeted to `/settings/gowa-servers`
- `frontend/src/i18n/locales/{ar,bn,en,hi,ur}.json` — nav renames (accounts → WhatsApp Numbers etc., gowaServers → GOWA Gateway) + 11 new keys; tip text under `tips.connect` (NOT `tips.gateway` — view references `tips.connect`)
- `frontend/e2e/helpers/api.ts` — createWhatsAppAccount injects `gowa_base_url: 'http://gowa.test:3000'` + `gowa_device_id: dev-<rand>` (backend CreateWhatsAppAccount requires both, accounts.go:147-149)
- `frontend/e2e/pages/AccountsPage.ts` + `tests/settings/accounts.spec.ts` — new POM (h1 = 'WhatsApp Accounts', key `accounts.title`), removed Meta tests + `whatsapp-business-profile.spec.ts`
- `internal/handlers/accounts.go` — DeleteAccount: best-effort LogoutDevice before delete (nil-guard, never blocks deletion)
- `internal/handlers/gowa_instances.go` — `cleanupGowaDeviceAccounts(orgID, baseURL, deviceID)`: soft-delete linked account + UserWhatsAppAccount + cache invalidation + audit, after successful DeleteDevice
- `internal/handlers/gowa_device_lifecycle_test.go` — 4 new tests

**Blast radius:**
| Symbol | File | Callers | Cross-module | Risk |
|---|---|---|---|---|
| DeleteAccount | accounts.go | routes only | no | low |
| cleanupGowaDeviceAccounts | gowa_instances.go | DeleteGowaInstanceDevice | no | low |
| CreateWhatsAppAccount | accounts.go | handler+helpers | yes (e2e) | low (requires gowa fields now) |
| accounts.spec.ts | e2e | — | — | low |

**Gotchas / knowledge:**
- `idx_wa_accounts_gowa_device` unique index exists in DB (not in model) → device↔account 1:1
- `RegisterGowaFactory` is process-global → tests using `newMsgTestApp` must NOT use `t.Parallel()`
- `DeletedAt` is struct → assert `.Valid`
- E2E: module-level `createTestScope` runId is shared per worker → fixed-suffix seeds collide when workers are reused → always `scope.name()` (random suffix)
- E2E: account name containing 'metadata' breaks shared helper's loose `getByText('Metadata')` (strict mode) → seed label 'meta-info'
- Playwright webServer auto-starts Vite:3000; backend on 8080 must run manually (`go run ./cmd/gowa-ui server -config config.toml`)
- Campaign test failures seen mid-work (TestApp_ListCampaigns_Success/FilterByStatus) did NOT reproduce: full suite on clean parent commit ef23c19f (fresh worktree, -count=1) = 181 PASS / 0 FAIL, and same on 04981680. → order/state-dependent flakiness (shared test state), not pre-existing deterministic failures, and not ours

**Tests:** gofmt/vet/build ✓; handlers related slice ✓; typecheck+lint ✓; E2E settings/accounts **10/10 twice**; graphify update ✓ (11446 nodes)

---

## Review fixes (critical review pass) — 2026-08-13

**Objective:** address the explicit critical review: 🔴 silent account deletion on device delete, 🟡6 weak gap-5 "documentation", 🟡3 escape-hatch banner. 2A/🟠 verified accurate in code (lazy client creation + ResolveGowaCreds covers config-only servers — no change needed); 🟡4 verified: `gowa_connection` WS broadcast exists but has NO frontend subscriber — 5s poll (guarded on empty gowa_device_id) stays as the reliable channel.

**Decisions (user):** (b) named confirm dialog — NOT blocking delete; keep /new manual binding page as documented escape hatch with banner.

**Files changed:**
- `internal/handlers/gowa_instances.go` — `ListGowaInstanceDevices` now enriches each device with `linked_account: {id, name}` via one batch query (`gowa_device_id IN ?`), soft-deletes excluded by GORM default scope; account secrets never leak (Select only id/name/gowa_device_id)
- `internal/handlers/messages_test.go` — `mockGowaServer` gained opt-in `setDevicesResponse` (GET /devices returns device list; default envelope unchanged — additive, zero risk to existing tests)
- `internal/handlers/gowa_device_lifecycle_test.go` — +2 tests: `TestListGowaInstanceDevices_EnrichesLinkedAccount` (bound/unbound + no secret leak), `TestListGowaInstanceDevices_OmitsSoftDeletedAccounts`
- `frontend/src/services/api.ts` — `GowaDevice.linked_account?: { id; name }`
- `frontend/src/views/settings/GowaServerDetailView.vue` — delete dialog `description` warns «This device is linked to the WhatsApp account «X» — it will be deleted as well» (named interpolation via `{ named: {...} }` — vue-i18n 3-arg overload requires that shape); device card shows «Linked account: X →» link to account page
- `frontend/src/views/settings/AccountDetailView.vue` — new-account (isNew) banner at top of Identity section: recommended-method note (reuses `accounts.tips.connect`) + GOWA Gateway button (reuses `accounts.gatewayPage`) — no new keys
- `frontend/src/i18n/locales/{ar,bn,en,hi,ur}.json` — +2 keys under gowaServers: `linkedAccount`, `deleteLinkedAccountWarning`

**Gotchas:**
- vue-i18n typed overload: `$t(key, default, { named: {...} })` — plain `{ name }` fails vue-tsc (TS2769)
- Run the backend from the repo root (`go run ./cmd/gowa-ui`) — from frontend/ it fails with "directory not found" silently backgrounded
- gofmt alignment in struct with `*linkedAccountRef` — run gofmt -w after edits (repo has 69 pre-existing unformatted files; keep touched files clean)

**Tests:** Go slice ✓ (enrichment tests + existing lifecycle/message tests), build ✓, vet ✓, typecheck ✓, lint 0 errors, E2E settings/accounts **10/10** after review fixes.

<!-- END -->


## Media streaming + upload progress — 2026-08-30 (deployed)

**Media download chain (Fix 1+2):** customer files >25MiB were undownloadable ("File wasn't available"). Root cause: `pkg/gowa/media.go` buffered whole downloads in RAM behind a hardcoded 25MiB guard; both CamScanner files were 110MB/96MB. Fix 1: guard 25→50MiB + recovery timeout 30s→10min (chain verified: nginx 900s, server 900s). Fix 2 (the real one): `DownloadMessageMediaToPath` streams GOWA→disk with a fixed copy buffer + `.part` atomic rename + incremental byte cap; `recoverMediaToDisk` (handlers/media.go) uses it with new `[storage].max_media_download_mb` config (default 1024); serving switched `os.ReadFile`→`fasthttp.ServeFile` (Range 206 verified, no RAM buffering). Dead `mimeFromExt` removed. Verified live: both files downloaded (110MB in 59s, 96MB in 46s). Webhook-time inline downloads still use the 50MiB in-memory path (small media); big media recovers lazily streamed.

**Upload progress bar (deployed same day):** `useChatMedia.sendMediaMessage` switched raw fetch→axios `api.post` with `onUploadProgress` (fetch can't report upload progress; XHR can) → `uploadProgress` ref → ChatView media dialog shows `Progress` bar + % (`data-testid="upload-progress"`) only while uploading; sits at 100% during the synchronous server-side GOWA send (button keeps "sending" state). Same CSRF/credentials contract. Template change additive only (e2e-safe). Files: `frontend/src/composables/useChatMedia.ts`, `frontend/src/views/chat/ChatView.vue`.

### Multipart fix (2026-08-30, deployed bak-20260830-6)
The progress-bar change switched the chat media upload from raw fetch to axios — and uploads broke with "Invalid multipart form". Root cause (captured live via XHR header hook): the shared axios instance defaults `Content-Type: application/json` (api.ts create), and when that default wins over FormData, axios 1.x converts the FormData to a JSON **string** (`utils.formDataToJSON`) — bodyType was literally "String", so fasthttp's MultipartForm() parse failed. Fix: explicit `'Content-Type': 'multipart/form-data'` on the upload call (the same established pattern as importData/uploadMedia/sendTemplate — that manual header is what has been protecting every other upload all along). Verified live: 400→200 on /api/messages/media with a throwaway invalid-number contact (created + deleted after). RULE: any new FormData POST through this axios instance MUST set the multipart Content-Type explicitly.

### Upload Cancel button (2026-08-30, deployed bak-20260830-7)
Cancel was `:disabled="isUploadingMedia"` — users were hostage to a long upload. Now: Cancel stays enabled mid-upload and actually ABORTS the transfer (AbortController signal into the axios request; closeMediaDialog aborts + resets; catch treats ERR_CANCELED/CanceledError as silence, not error). Send stays disabled while uploading. Client-side guard documented along the way: useChatMedia rejects >50MB docs / >16MB media before the dialog opens ("File too large"). Known limitation (HTTP semantics): canceling while the bar sits at 100% (server-side GOWA send phase) aborts the browser request, but the server may still complete the send — the message can land after a late cancel. Verified by user live.

## WS liveness overhaul — 2026-08-31 (deployed bak-20260831-2)
Symptom: idle tab silently stopped receiving; F5 recovered everything. Root cause (3 compounding client gaps, server was fine): (1) app-pings were fire-and-forget — pong never verified, so half-open sockets (sleep/NAT flap) looked OPEN forever while swallowing sends; (2) reconnect gave up after 5 attempts/31s — sleep/wake killed real-time permanently; (3) visibilitychange/focus only marked-read, never checked socket health. Fix (frontend/src/services/websocket.ts only): lastServerActivityAt stamped on every server message (incl. pong); watchdog in the 30s ping loop force-reconnects when silent >75s (STALE_AFTER_MS); reconnect is now infinite with capped 30s backoff + jitter (disconnect() wins via intentionallyDisconnected flag); wake triggers on window 'online' + visibilitychange→visible rebuild stale sockets; no-token path retries instead of dying silently; socket-capture guards retire handlers of replaced sockets. Verified: bundle markers live, healthy idle connection survives >112s without reconnect storm (no false positives). Same fix whatomate shipped in June for the identical symptom. NOTE: console.* strings are stripped by the build — verify bundles by code identifiers (forceReconnect/STALE_AFTER_MS), never by log text.

## Supervisor "Me" tab — 2026-08-31 (deployed bak-20260831-3)
Request: admins use the Me tab to FOLLOW UP agents' assigned conversations (admins don't claim). Change (frontend/src/stores/contacts.ts, myContacts computed only): users with contacts:write (canSeeSupervisorTabs — the established admin/manager marker) see EVERY assigned conversation (assigned_user_id set) in Me; regular agents keep the strictly-mine filter. Assignee name tag already rendered on rows (assigned_user_name came from the API all along), myCount badge reflects the new semantics, zero template/backend changes. Verified live as admin: Me shows 5 assigned conversations with assignee tags. Note: tab label stays "Me" per user preference; default landing tab unchanged (Pending).

## Unread badge consistency — 2026-08-31 (deployed bak-20260831-4)
Report: Me-tab badges vanished on open but returned on refresh. Two findings: (1) most badge numbers are REAL unread incoming messages in agents' conversations — newly visible to supervisors via the Me tab, and opening marks them read server-side (verified live: 26→0); (2) genuine bug — the badge counted `status != 'read'`, which includes revoked/failed messages that markMessagesAsRead deliberately never sweeps (the revoke fix), so any conversation containing one deleted message showed a badge that returned on EVERY refresh forever. Fix: both unread-count sites (buildContactResponseMasked in contacts.go + the pending-claim gate in GetMessages) now use `status NOT IN (read, revoked, failed)` — aligned with the read sweep's exclusions. Regression test TestUnreadCount_ExcludesRevokedAndFailed. Verified: old rule 91 vs new rule 90 on the revoked-containing conversation.

## Open-chat scrolls to latest — 2026-08-31 (deployed bak-20260831-5)
Guarantee: opening a conversation always lands on the LATEST message. The old single scroll (50ms after selection) fired before media bubbles / virtual rows settled their heights, drifting the position up. Now: instant scroll immediately after the messages fetch completes + a second instant scroll 350ms later to re-assert after layout settles (ChatView onContactSelected only — no store/template changes). Verified live on two conversations (50-msg and 3-msg): scrollBottomDelta=0 exactly, last message fully visible on both the deep-link and contact-switch paths.
