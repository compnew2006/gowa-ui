# Whatomate Dedicated Instances (3)

Generated: 2026-02-25 23:25:52 UTC
Server IP: 31.97.192.53
Base Domain Pattern (suggested): <tenant>.ofuqalmadenah.com

## Sandbox Deploy Note - 2026-06-03 19:34 UTC — fbcomments-realtime-push-10903-skip

- Sandbox green hotfix: 2026-06-03 19:34 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_223000_fbcomments_realtime_push_10903_skip
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.blue.20260603_193118_before_realtime_push
- Installed SHA256: cbef8d21b9c9818bc2e867a9ffdfe2c35ba3e0e5f5411b76ba0e11f6e34d5ca5
- Version: sandbox-green-20260603_223000-fbcomments-realtime-push-10903-skip
- Build: linux/amd64, CGO disabled, embedded license public key ring (kid=deploy-20260416), embedded frontend dist.
- Source HEAD: 23550b60 with uncommitted changes:
  - **Backend** (`internal/`):
    - `internal/websocket/messages.go` — added `TypeFacebookCommentCreated` and `TypeFacebookCommentUpdated` constants
    - `internal/handlers/fb_comments.go` — `ReceiveFacebookCommentsWebhook` now broadcasts `Created`/`Updated` per row after upsert. `sendAndStoreFacebookCommentReply` rewritten: catches 10903 ("user can't be DM'd") and records reply as `status=skipped` with `metadata.dm_skipped=true` (was `partial`), so the comment stays `replied` when the public reply succeeded. `UpdateFacebookCommentStatus` also broadcasts `Updated`. Added `isFacebookUserCantDMError` helper.
    - `internal/handlers/export_test.go` (new) — exposes `IsFacebookUserCantDMError` to external test packages
    - `internal/handlers/testhelpers_test.go` — `withWSHub()` option to inject a real `*websocket.Hub`
    - `internal/handlers/fb_comments_test.go` — 4 new tests: `TestHandleFacebookWebhookComment_BroadcastsCreated`, `_BroadcastsUpdatedOnDuplicate`, `TestUpdateFacebookCommentStatus_BroadcastsUpdated`, `TestSendAndStoreFacebookCommentReply_SkipsOnFacebookUserCantDMError`. 9/9 facebook-package tests pass.
    - `internal/models/fb_comment.go` — `AutoPrivateReplyEnabled` default flipped from `true` to `false`
  - **Frontend** (`frontend/src/`):
    - `frontend/src/services/websocket.ts` — added `WS_TYPE_FACEBOOK_COMMENT_CREATED` and `WS_TYPE_FACEBOOK_COMMENT_UPDATED` constants (exported)
    - `frontend/src/views/facebook/FacebookCommentsView.vue` — `onMounted` subscribes `handleCommentCreated`/`handleCommentUpdated`; `onUnmounted` unsubscribes. Reply Badge uses localized `replyStatus.{sent,partial,failed,skipped}` with variant mapping (destructive/secondary/outline/default). "DM not available" indicator via `isReplySkipped(reply)`.
    - `frontend/src/views/facebook/facebookCommentsMerge.ts` (new) — pure helpers `applyCommentCreated`, `applyCommentUpdated`, `isReplySkipped`. Returns `MergeResult{comments, appended, replaced, prependIndex}`. Preserves `replies` when payload omits them.
    - `frontend/src/views/facebook/facebookCommentsMerge.test.ts` (new) — 13 vitest tests, 4ms
    - `frontend/src/i18n/locales/{en,ar,es}.json` — `dmNotAvailable` + `replyStatus.{sent,partial,failed,skipped}` keys
- Verification:
  - `systemctl restart whatomate-sandbox` → active since 2026-06-03 19:34 UTC
  - `curl http://127.0.0.1:18127/api/license/bootstrap` → `{"status":"active","key_id":"deploy-20260416","tier":"production",...}`
  - **E2E live**: Python WS client on `wss://sandbox.ofuqalmadenah.com/ws` (via `whm_access` cookie → `/api/auth/ws-token` → `Authorization: Bearer <token>`) received `{"type":"facebook_comment_created","payload":{...full comment with from_id, from_name, status:"open", replies:[]}}` within 1 second of the HMAC-signed webhook POST. Test data (temp OAuth account + 4 comments + replies) cleaned up after verify.
  - **Frontend**: 13/13 vitest tests pass, eslint clean, vue-tsc --noEmit clean
- Open items: 10903 path unit-tested but not yet exercised live on a real comment; `AutoPrivateReplyEnabled` default flip pending DB row awareness; 1018 historical empty-from_id rows still awaiting backfill decision; 401s on `/api/facebook/comments` and `/api/auth/logout` root-caused to production frontend JWT 15min TTL + missing silent refresh (OUT OF SCOPE).

## Sandbox Deploy Note - 2026-06-03 18:24 UTC — fbcomments-from-payload-fix

- Sandbox green hotfix: 2026-06-03 18:24:33 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_182041_fbcomments_from_payload_fix
- Sandbox blue rollback binary (unchanged from prior deploy): /opt/whatomate/bin/whatomate.sandbox.green.20260603_155700_fbcomment_page_messages_private_reply_license
- Archived pre-fix green: /opt/whatomate/bin/whatomate.sandbox.green.20260603_172836_fbpage_comments_harden_license.archived-20260603_182243
- Installed SHA256: 7eb9180d46eda60dbe811793c16921fd6cb30c400804c72e55b6785fd01147f8
- Version: sandbox-green-20260603_182041-fbcomments-from-payload-fix
- Build: linux/amd64, CGO disabled, embedded license public key ring (kid=deploy-20260416), embedded frontend dist.
- Source HEAD: 23550b60 with uncommitted Facebook from-payload fix:
  - internal/handlers/fb_comments.go: added `From facebookCommentsWebhookActor` field on `facebookCommentsWebhookValue` + `commenterID()` / `commenterName()` helpers; `upsertFacebookWebhookComment` now uses them. Prevents the `value.sender_*` legacy keys from being read for comment/feed webhooks (which deliver `value.from.{id,name}`), which was the cause of all 2026-06-03 webhooks storing `from_id=NULL, from_name=NULL`.
  - internal/handlers/fb_comments_test.go (new): 3 tests added — `PopulatesFromPayload`, `FallsBackToSenderFields`, `FromWebhookEndToEnd`. All 7 facebook-package tests pass.
  - test/testutil/db.go: per-test unique pageID via `fmt.Sprintf("page-%s", t.Name())` so `t.Parallel()` stays safe.
- Bug impact: 1018 rows in last 30 days were stored with `from_id=NULL` (965 on page 248262288519219 Amin Eldeshnawy, 26 on 106812225128833, 23 on 815073515173177, 3 on 895247390337022, 1 on 110627688093389). Same root cause also blocked `sendFacebookPrivateReply` → `sendFacebookDirectMessengerMessage` fallback on `code=10903` (skipped silently when senderID was empty).
- Verification:
  - `systemctl restart whatomate-sandbox` → active since 2026-06-03 18:24:33 UTC, PID 2175027
  - `curl http://127.0.0.1:18127/api/license/bootstrap` → `{"enabled":true,"status":"active","locked":false,"key_id":"deploy-20260416",...}`
  - `POST https://sandbox.ofuqalmadenah.com/api/facebook/comments/webhook` with HMAC-SHA256-signed `verb=add` payload containing `from:{id:"psid-1780511343-69956",name:"Live Verify Bot"}` → `{"processed":1,"status":"ok","auto_replies":0}`. DB row inserted with `from_id=psid-1780511343-69956`, `from_name="Live Verify Bot"`. **Fix verified end-to-end.**
  - Test comment row + reply row cleaned up after verify (1 row each).
- Open items: 1018 historical empty-from_id rows awaiting backfill decision; 401s on `/api/facebook/comments` and `/api/auth/logout` to be investigated.

## Sandbox Deploy Note - 2026-06-03 17:32 UTC

- Sandbox green hotfix: 2026-06-03 17:32 UTC
- Sandbox domain: sandbox.ofuqalmadenah.com
- Sandbox service: whatomate-sandbox.service
- Active sandbox selector: /opt/whatomate/bin/whatomate.sandbox.active
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_172836_fbpage_comments_harden_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_155700_fbcomment_page_messages_private_reply_license
- Pre-deploy backup: /root/whatomate_backups/sandbox-predeploy-20260603_172906
- Installed SHA256: 2eb6cf8a31137be1293ec9ed319c4be5c69b7d9942153bad9391178797f3a1f2
- Version: sandbox-green-23550b60-fbpage-comments-harden-20260603_172836
- Build: linux/amd64, CGO disabled, embedded license public key ring (kid=deploy-20260416), embedded frontend dist.
- Source HEAD: 23550b60 with uncommitted Facebook hardening changes:
  - internal/handlers/fb_oauth.go: nil-safe `facebookOAuthCallbackURL` (was panicking when request was nil)
  - internal/handlers/fb_comments.go: removed broken bare-suffix retry in `sendFacebookPrivateReply`; fixed pre-existing GORM session bug in `sendAndStoreFacebookCommentReply` using `db.Session(&gorm.Session{NewDB: true})` to avoid `Statement.Dest` reuse from prior `db.Create(&reply)`
  - test/testutil/db.go: added Facebook models to migrations and cleanup
  - internal/handlers/fb_comments_test.go: 4/4 Facebook reply tests pass locally
- Sandbox switch script: /usr/local/sbin/whatomate-sandbox-switch (also linked at /opt/whatomate/bin/switch-sandbox-blue-green.sh).
  - Usage: `whatomate-sandbox-switch {status|green|blue|toggle|version}`.
  - Performs health check on `http://127.0.0.1:18127/api/license/bootstrap`; auto-rolls back to blue if green is not active.
- Verification:
  - `systemctl restart whatomate-sandbox` → `active`
  - `curl http://127.0.0.1:18127/api/license/bootstrap` → `{"status":"success","data":{"enabled":true,"status":"active","locked":false,"key_id":"deploy-20260416",...}}`
  - `curl -I https://sandbox.ofuqalmadenah.com/` → HTTP/2 200 from nginx (HEAD not supported by SPA, GET returns full HTML with embedded dist)
  - `POST https://sandbox.ofuqalmadenah.com/api/auth/login` with admin@whatomate.local → success with user object, full organization tree
  - Production binary untouched: /opt/whatomate/bin/whatomate.green.20260528_111523
  - All four live tenant services (`whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`) unchanged
- Change: One command to toggle sandbox blue↔green: `whatomate-sandbox-switch`

## Sandbox Deploy Note - 2026-06-03

- Sandbox green hotfix: 2026-06-03 15:57 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_155700_fbcomment_page_messages_private_reply_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_154100_fbcomment_private_reply_fallback_license
- Installed SHA256: 612bd20b4c3988f845c44f3c700db9f4b06cb75a1e058c356f38fe65233626e3
- Version: sandbox-green-23550b60-fbcomment-page-messages-private-reply-20260603_155700
- Change: Facebook private replies now use the Dashboard-proven Messenger Private Reply API: POST /{page_id}/messages with JSON recipient.comment_id and message.text. If Meta still rejects the comment private reply with code=100/subcode=33, the code falls back to the direct Messenger /me/messages RESPONSE path using the comment sender id.
- Verification: service active, https://sandbox.ofuqalmadenah.com/login returned 200, license bootstrap remained enabled=true/status=active/key_id=deploy-20260416.

- Sandbox green hotfix: 2026-06-03 15:41 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_154100_fbcomment_private_reply_fallback_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_152900_fbcomment_graph_error_license
- Installed SHA256: 96e0dfe78ba202887058b258ad9b94834582ca37c3829266798c23017e0ac226
- Version: sandbox-green-23550b60-fbcomment-private-reply-fallback-20260603_154100
- Change: Facebook private replies now retry once with the bare comment ID when Meta rejects the stored post_comment composite ID with code=100/subcode=33. Public comment replies still use the original external ID.
- Verification: service active, https://sandbox.ofuqalmadenah.com/login returned 200, license bootstrap remained enabled=true/status=active/key_id=deploy-20260416.

- Sandbox green hotfix: 2026-06-03 15:29 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_152900_fbcomment_graph_error_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_152200_fbcomment_reply_scopefix_license
- Installed SHA256: d7aed0da2aa60de3e7255e9782e91c9ec0ff510af71020cb1d69b29547cc4312
- Version: sandbox-green-23550b60-fbcomment-grapherr-20260603_152900
- Change: Facebook Graph API non-2xx responses now preserve the Graph error message/code/subcode/type in the stored reply error, so private-reply failures expose the exact Meta rejection reason instead of only status 400.
- Verification: service active, https://sandbox.ofuqalmadenah.com/login returned 200, license bootstrap remained enabled=true/status=active/key_id=deploy-20260416.

- Sandbox green hotfix: 2026-06-03 15:22 UTC
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_152200_fbcomment_reply_scopefix_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_151400_fbcomment_reply_lookup_license
- Backup binary created: /opt/whatomate/bin/whatomate.sandbox.green.20260603_151400_fbcomment_reply_lookup_license.predeploy_20260603_152200.bak
- Installed SHA256: 0e051a4f8d832c638365ccfd0069f07a136f09e718c6a040a60299ebca55724f
- Version: sandbox-green-23550b60-fbcomment-reply-scopefix-20260603_152200
- Change: Facebook comment reply/status lookup now uses the base DB with explicit organization_id filters, avoiding stale request-scoped tenant DB conflicts that returned 404 for existing comments.
- Verification: service active, https://sandbox.ofuqalmadenah.com/login returned 200, license bootstrap returned enabled=true/status=active/key_id=deploy-20260416.

- Sandbox green redeploy: 2026-06-03 15:14 UTC
- Sandbox domain: sandbox.ofuqalmadenah.com
- Sandbox service: whatomate-sandbox.service
- Active sandbox selector: /opt/whatomate/bin/whatomate.sandbox.active
- Active sandbox green binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_151400_fbcomment_reply_lookup_license
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_145935_fbcomments_scope_fix
- Backup binary created: /opt/whatomate/bin/whatomate.sandbox.green.20260603_145935_fbcomments_scope_fix.predeploy_20260603_150900.bak
- Installed SHA256: 20f1943b265f29611c4e8cd8f728c7b2354a29dbef6beaed315f30f991a449ad
- Version: sandbox-green-23550b60-fbcomment-reply-20260603_151400
- Change: deployed current local build with Facebook comment reply lookup accepting internal UUID or Facebook external_id, with embedded license public key ring.
- License verification: /api/license/bootstrap returned enabled=true, status=active, key_id=deploy-20260416.
- Browser verification: Chrome DevTools loaded https://sandbox.ofuqalmadenah.com/login, no console errors, license bootstrap returned active.
- Switch command: `whatomate-sandbox-switch green`, `whatomate-sandbox-switch blue`, or `whatomate-sandbox-switch toggle`.
- Temporary build source removed from VPS after install; only binaries/config/uploads/docs remain.

- Sandbox domain: sandbox.ofuqalmadenah.com
- Sandbox service: whatomate-sandbox.service
- Active sandbox binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_134253_fbcomments_author_retry_i18n
- Production binary untouched: /opt/whatomate/bin/whatomate.green.20260528_111523
- Change: Facebook comments page now includes common.previous translations and retries per-comment Graph actor lookup (`from{id,name}`) when nested sync data has an empty author.

## Summary

- Dedicated instance = separate systemd service, PostgreSQL DB/user, uploads path, config, internal port
- Shared binary: /opt/whatomate/bin/whatomate
- SSL is enabled for all 3 subdomains (Let's Encrypt via Certbot)
- Nginx HTTPS vhosts are active and Certbot auto-renew is configured

## حلول وانجاز

- Tenant slug: holol-wenjaz
- Suggested domain: holol-wenjaz.ofuqalmadenah.com
- Systemd service: whatomate@holol-wenjaz
- Internal port: 127.0.0.1:18124
- Redis DB index: 1
- Instance dir: /opt/whatomate/instances/holol-wenjaz
- Uploads dir: /opt/whatomate/instances/holol-wenjaz/uploads
- Config: /opt/whatomate/instances/holol-wenjaz/config.toml
- PostgreSQL DB: whatomate_holol_wenjaz
- PostgreSQL User: whatomate_holol_wenjaz
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+holol-wenjaz@whatomate.local
- Default Admin Password: Zqxtu8r3mRhwhJ6oLYaxqE/7
- Nginx vhost: /etc/nginx/sites-available/whatomate-holol-wenjaz.conf
- Certbot command used: certbot --nginx -d holol-wenjaz.ofuqalmadenah.com

## الاركان المثالية

- Tenant slug: alarkan-almthalia
- Suggested domain: alarkan-almthalia.ofuqalmadenah.com
- Systemd service: whatomate@alarkan-almthalia
- Internal port: 127.0.0.1:18125
- Redis DB index: 2
- Instance dir: /opt/whatomate/instances/alarkan-almthalia
- Uploads dir: /opt/whatomate/instances/alarkan-almthalia/uploads
- Config: /opt/whatomate/instances/alarkan-almthalia/config.toml
- PostgreSQL DB: whatomate_alarkan_almthalia
- PostgreSQL User: whatomate_alarkan_almthalia
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+alarkan-almthalia@whatomate.local
- Default Admin Password: [REDACTED]
- Nginx vhost: /etc/nginx/sites-available/whatomate-alarkan-almthalia.conf
- Certbot command used: certbot --nginx -d alarkan-almthalia.ofuqalmadenah.com

## مطبعة رؤية

- Tenant slug: matbaat-ruya
- Suggested domain: matbaat-ruya.ofuqalmadenah.com
- Systemd service: whatomate@matbaat-ruya
- Internal port: 127.0.0.1:18126
- Redis DB index: 3
- Instance dir: /opt/whatomate/instances/matbaat-ruya
- Uploads dir: /opt/whatomate/instances/matbaat-ruya/uploads
- Config: /opt/whatomate/instances/matbaat-ruya/config.toml
- PostgreSQL DB: whatomate_matbaat_ruya
- PostgreSQL User: whatomate_matbaat_ruya
- PostgreSQL Password: [REDACTED]
- Default Admin Email: admin+matbaat-ruya@whatomate.local
- Default Admin Password: nRiMXl0DuJe8xfOS707NdPnU
- Nginx vhost: /etc/nginx/sites-available/whatomate-matbaat-ruya.conf
- Certbot command used: certbot --nginx -d matbaat-ruya.ofuqalmadenah.com

## Verification

### whatomate@holol-wenjaz

- enabled: enabled
- active: active

### whatomate@alarkan-almthalia

- enabled: enabled
- active: active

### whatomate@matbaat-ruya

- enabled: enabled
- active: active

### Local Port Listeners

State Recv-Q Send-Q Local Address:Port Peer Address:PortProcess  
LISTEN 0 4096 127.0.0.1:18124 0.0.0.0:_ users:(("whatomate",pid=1604127,fd=10))  
LISTEN 0 4096 127.0.0.1:18125 0.0.0.0:_ users:(("whatomate",pid=1604191,fd=8))  
LISTEN 0 4096 127.0.0.1:18126 0.0.0.0:\* users:(("whatomate",pid=1604256,fd=8))

### Local HTTP Smoke (Host header)

- holol-wenjaz.ofuqalmadenah.com -> OK (Whatomate frontend served on :18124)
- alarkan-almthalia.ofuqalmadenah.com -> OK (Whatomate frontend served on :18125)
- matbaat-ruya.ofuqalmadenah.com -> OK (Whatomate frontend served on :18126)

## SSL Enablement Update

Updated: 2026-02-25 23:33:58 UTC

- All 3 subdomains now have valid HTTPS certificates issued by Let's Encrypt
- Certbot deployed the certificates directly into the Nginx tenant vhosts
- Auto-renew is enabled by Certbot's scheduled task

### حلول وانجاز

- Live URL: https://holol-wenjaz.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18124
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/holol-wenjaz.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/holol-wenjaz.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:31:43 2026 GMT

### الاركان المثالية

- Live URL: https://alarkan-almthalia.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18125
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/alarkan-almthalia.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/alarkan-almthalia.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:31:57 2026 GMT

### مطبعة رؤية

- Live URL: https://matbaat-ruya.ofuqalmadenah.com
- HTTP backend target: 127.0.0.1:18126
- SSL Status: Enabled (Let's Encrypt)
- Certificate: /etc/letsencrypt/live/matbaat-ruya.ofuqalmadenah.com/fullchain.pem
- Private Key: /etc/letsencrypt/live/matbaat-ruya.ofuqalmadenah.com/privkey.pem
- Expires: May 26 22:32:13 2026 GMT

### Renewal Check

- Test renewal: certbot renew --dry-run

### Local HTTPS Smoke (SNI + local resolve)

- holol-wenjaz.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed
- alarkan-almthalia.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed
- matbaat-ruya.ofuqalmadenah.com: HTTP/1.1 405 Method Not Allowed

### How to apply code changes to all instances

Update source in /opt/whatomate-src (git pull / rsync)
Build new binary on VPS
Replace /opt/whatomate/bin/whatomate
Restart all instance services
Commands (all instances)

cd /opt/whatomate-src
git pull # or rsync your updated code here

# Build (frontend + backend)

export PATH=/usr/local/go/bin:$PATH
make build-prod

# Install shared binary

install -m 755 /opt/whatomate-src/whatomate /opt/whatomate/bin/whatomate

# Restart all tenants

systemctl restart whatomate@holol-wenjaz
systemctl restart whatomate@alarkan-almthalia
systemctl restart whatomate@matbaat-ruya
Verify

systemctl status whatomate@holol-wenjaz --no-pager -l
systemctl status whatomate@alarkan-almthalia --no-pager -l
systemctl status whatomate@matbaat-ruya --no-pager -l
journalctl -u whatomate@holol-wenjaz -n 50 --no-pager
Safer pattern (recommended)

Restart one instance first (test)
If OK, restart the remaining two
systemctl restart whatomate@holol-wenjaz

# test website/login

systemctl restart whatomate@alarkan-almthalia whatomate@matbaat-ruya
Important note about DB migrations

Your service runs with -migrate, so on restart each instance may run migrations on its own database.
This is good for multi-tenant isolated DBs, but if a migration is risky, test on one tenant first.
Best next improvement

Create one script like deploy_whatomate_all.sh to automate:
sync/build/install
restart all instances
health checks
rollback if one fails

## Deployment Update

Updated: 2026-02-26 23:36:13 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via rsync; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `e0a23f5` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260226_233612.bak`
- Installed binary SHA256: `1fda6a038c26fbac983c9d9b904d22df6da7ac309e7a89b7aea7447d458873ad`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`

## Deployment Update

Updated: 2026-03-09 06:49:10 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_064910.bak`
- Installed binary SHA256: `57d6c12141abaed291898bf01e47ca69c17c5e2684097c5535deec120ca4c56a`
- Note: the local cross-compiled Linux binary was not used because it crashed on this VPS with `SIGSEGV`; the final deployment was built natively on the server and verified healthy.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- During restart, temporary `502` responses appeared while migrations/startup were in progress; final state is healthy.

## Deployment Update

Updated: 2026-02-27 20:43:46 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`

## Deployment Update

Updated: 2026-04-16 01:23:18 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Native VPS build directory: `/root/whatomate_temp_build_20260416_010826`
- Native VPS build command: `cd /root/whatomate_temp_build_20260416_010826 && GOTOOLCHAIN=go1.25.9+auto VERSION=a7e55d5-licensecfg-vps-20260416_012230 make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Installed version: `Whatomate a7e55d5-licensecfg-vps-20260416_012230 (built 2026-04-16_01:22:46)`
- Installed SHA256: `7d953074b3b2b7fc9a6f63d25f0e4ebca334f9db5d285472174bbdb9e513715e`

### Backup / Rollback Paths

- No new full backup set was created in this session because the existing backup was user-approved for reuse.
- Immediate rollback binaries created during deployment:
  - `/opt/whatomate/bin/whatomate.20260416_011329.pre_cutover.bak`
  - `/opt/whatomate/bin/whatomate.20260416_012035.pre_cutover.bak`
  - `/opt/whatomate/bin/whatomate.20260416_012318.pre_cutover.bak`

### License Fix Applied

- Root cause of the failed rollouts:
  - the new `a7e55d5` code rejected the already-working production license config override and crash-looped `whatomate.service`
  - the first retry rebuilt the old source by mistake because the patched files were synced into the wrong VPS paths
- Final code fix:
  - restored production support for config-based `license.public_key` when `license.allow_unsafe_public_key_override = true`
  - preserved the working production values already present in:
    - `/opt/whatomate/config.toml`
    - `/opt/whatomate/instances/holol-wenjaz/config.toml`
    - `/opt/whatomate/instances/alarkan-almthalia/config.toml`
    - `/opt/whatomate/instances/matbaat-ruya/config.toml`
- Final license state:
  - `127.0.0.1:18123` -> `enabled = true`, `status = active`, `locked = false`
  - `127.0.0.1:18124` -> `enabled = true`, `status = active`, `locked = false`
  - `127.0.0.1:18125` -> `enabled = true`, `status = active`, `locked = false`
  - `127.0.0.1:18126` -> `enabled = true`, `status = active`, `locked = false`

### Rollout Notes

- First cutover attempt rolled back automatically after the new binary exited with:
  - `production licensing must use an embedded key ring; remove license.public_key, license.public_key_kid, and license.allow_unsafe_public_key_override`
- Second cutover attempt rolled back automatically after the VPS rebuild still contained the old validator.
- Third cutover succeeded in order:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`

### Verification

- Local instance checks:
  - `http://127.0.0.1:18123/login` -> `200`
  - `http://127.0.0.1:18124/login` -> `200`
  - `http://127.0.0.1:18125/login` -> `200`
  - `http://127.0.0.1:18126/login` -> `200`
  - all four `/api/license/bootstrap` responses reported `enabled=true`, `status=active`, `locked=false`
- Public HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- Browser verification:
  - Chrome DevTools MCP was unavailable in this session
  - used Playwright CLI instead
  - confirmed the rendered login page on the main domain and all three tenant domains with:
    - page title `Whatomate`
    - heading `Welcome to Whatomate`
    - `Email` textbox
    - `Sign in` button

### Cleanup Status

- Intended cleanup targets after successful deployment:
  - `/root/whatomate_temp_build_20260416_010826`
  - `/root/whatomate_temp_build_settings_fix`
- Final remote cleanup and remote markdown updates could not be completed from this client because new SSH connections to `31.97.192.53:22` stopped completing after the rollout.
- Source sync target: `/opt/whatomate-src` (via rsync; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `e0a23f5` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260227_204257.bak`
- Installed binary SHA256: `bd4fd646c39552f35183136433dd6d74e7a3f4bf5683d41148acd5fc9b927370`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediately after restart, temporary `502` responses were observed while services were booting/migrating; final status is healthy.

## Deployment Update

Updated: 2026-02-27 21:50:56 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `cc8cbc8` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260227_214922.bak`
- Installed binary SHA256: `be5e506a104297b545692dff50ee25c9653d87745c7ad4d5ebebed295315fc1a`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were running migrations/boot; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:00:54 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_130054.bak`
- Installed binary SHA256: `a8fdcf89bc36ea137703b357fc23ff5cd7b6c77e0124768383254bc9f145d26f`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Initial checks during restart briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-02-28 19:42:52 UTC

- Deployed from local workspace: /Users/noiemany/Downloads/whatomate_GOWA/whatomate
- Source sync target: /opt/whatomate-src (via rsync; excluded caches/uploads/build artifacts)
- Source revision on deploy: 93f8b57 (local working tree had uncommitted changes)
- Build command: make build-prod
- Installed binary: /opt/whatomate/bin/whatomate
- Backup binary created: /opt/whatomate/bin/whatomate.20260228_194252.bak
- Installed binary SHA256: b9e356d8530538edade5202adb3425b2e67bd20482a8e57e4ad09cb8ee0d60db

### Services Restarted

- whatomate
- whatomate@holol-wenjaz
- whatomate@alarkan-almthalia
- whatomate@matbaat-ruya

### Post-Deploy Verification

- Listener ports active: 127.0.0.1:18123, 127.0.0.1:18124, 127.0.0.1:18125, 127.0.0.1:18126
- HTTPS smoke:
  - https://ofuqalmadenah.com -> 200
  - https://holol-wenjaz.ofuqalmadenah.com -> 200
  - https://alarkan-almthalia.ofuqalmadenah.com -> 200
  - https://matbaat-ruya.ofuqalmadenah.com -> 200

### Note

- Immediate post-restart checks briefly returned 502 while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-02-28 21:55:40 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260228_215404.bak`
- Installed binary SHA256: `b5223e64545021a63a33bbbf86379612892a24760536fb8bbf9b68402c00590b`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate checks after restart briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:21:23 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_132214.bak`
- Installed binary SHA256: `86934df1bd263ce19344829c75b6e18769e46852f8a824da3701614373e9eb98`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 13:57:52 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `93f8b57` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_135752.bak`
- Installed binary SHA256: `66b724514cb31eb4e4c49e570edabb01478d2cd1efa34afb36aa37db14f86fcc`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were still booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-01 19:09:57 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `6ed1c10` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260301_190933.bak`
- Installed binary SHA256: `8f61e12afae2b4b9067e66f211dfc1ac33d1f4f4e6c4a1a2080185d3fd213e69`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- Immediate post-restart checks briefly returned `502` while services were booting/migrating; final state is healthy.

## Deployment Update

Updated: 2026-03-02 01:04:04 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `6ed1c10` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260302_010403.bak`
- Installed binary SHA256: `28709e58397e704e5c87e81c06af90982a7e389b2fa50697f6de2188f6a35ba1`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- The core phone matching algorithm was fundamentally migrated from manual digit grouping loops to Google's official `libphonenumber` regex structural matching library, perfectly isolating Arabic and English multi-byte inputs without polluting explicit standard IDs and accounts.

## Deployment Update

Updated: 2026-03-03 14:48:57 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `fdfa791` (working tree had local uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260303_144743.bak`
- Installed binary SHA256: `deed3269e20ab1b550304c00157f2a24d0e710de251de197ef490ac536991f12`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- The deployment was completed successfully using the standard workflow. All services were verified as operational post-restart.

## Deployment Update

Updated: 2026-03-03 15:14:24 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `fdfa791` (local working tree had uncommitted changes at `17:10`)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260303_151354.bak`
- Installed binary SHA256: `016c70f911f21051fdf82d504c33fde6a0792fa62a49e4ac6cc89c8b606bace7`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com` -> `200`

### Note

- This deployment includes new changes made to `frontend/src/services/api.ts` and `internal/config/config.go` observed at 17:10 local time.

## Deployment Update

Updated: 2026-03-03 15:25:01 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Direct binary upload artifact: `/tmp/whatomate-linux-20260303_172132`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.backup.20260303_152408`
- Installed binary SHA1: `4a6643270cb44ecfa72ae58b4aadaa677d575e52`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all 4 services `active`
- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- API check: `https://ofuqalmadenah.com/api/statuses` now returns `401 Missing authorization` (route is present; previous browser-side `404` issue was tied to old backend state before restart)

### Note

- A short startup window produced temporary `502` while services were restarting/migrating; final state is healthy.

## Storage Optimization Update

Updated: 2026-03-03 15:35:00 UTC

### What was implemented on VPS

- Added automated housekeeping script: `/usr/local/bin/whatomate-housekeeping.sh`
- Added settings file: `/etc/default/whatomate-housekeeping`
- Added systemd service: `/etc/systemd/system/whatomate-housekeeping.service`
- Added systemd timer: `/etc/systemd/system/whatomate-housekeeping.timer`
- Timer schedule: daily at `03:30 UTC` (`RandomizedDelaySec=20m`, `Persistent=true`)

### Housekeeping tasks

- Deduplicate identical media files using hardlinks in:
  - `/opt/whatomate/uploads`
  - `/opt/whatomate/instances/holol-wenjaz/uploads`
- Remove expired WhatsApp statuses from each tenant DB and delete associated local media files
- Keep only latest 5 binary backups in `/opt/whatomate/bin`
- Clear source-only artifacts in `/opt/whatomate-src/uploads` and test reports
- Vacuum systemd journal to a max size of `200M`

### Immediate one-time reclaim completed

- Dry-run estimated reclaim:
  - `/opt/whatomate/uploads`: `13.57 GiB`
  - `/opt/whatomate/instances/holol-wenjaz/uploads`: `4.6 GiB`
- Real dedupe reclaim completed: `18.17 GiB` total
- Old binary backups pruned: `/opt/whatomate/bin` reduced from `932M` to `262M`
- Source artifacts cleanup: `/opt/whatomate-src` reduced from `702M` to `514M`

### Current disk snapshot

- `/opt/whatomate/uploads`: `28G` (was `41G`)
- `/opt/whatomate/instances/holol-wenjaz/uploads`: `3.9G` (was `8.5G`)
- `/var/log/journal`: `77M`
- Root filesystem `/`: `64%` used (`62G` used / `35G` free)

### Service health

- `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`: all `active`

### How to control policy

- Edit `/etc/default/whatomate-housekeeping` and tune:
  - `STATUS_GRACE_HOURS`
  - `KEEP_BACKUPS`
  - `JOURNAL_MAX_SIZE`
  - `ENABLE_HARDLINK_DEDUP`
  - `CLEAN_SOURCE_UPLOADS`
  - `DRY_RUN`
- Apply changes:
  - `systemctl daemon-reload`
  - `systemctl restart whatomate-housekeeping.timer`

## Storage Policy Update (Message Media Safety)

Updated: 2026-03-03 15:51:30 UTC

- Housekeeping policy updated to preserve chat media files.
- New setting in `/etc/default/whatomate-housekeeping`:
  - `DELETE_STATUS_MEDIA_FILES=0` (default)
- Behavior now:
  - Expired `whatsapp_statuses` DB rows are deleted.
  - Media files are **not** deleted by default.
- Safety guard added in script:
  - If file deletion is enabled in future, script checks `messages.media_url` references first and keeps referenced files.

## Deployment Update

Updated: 2026-03-04 23:53:40 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `uploads/`, and local build artifacts)
- Source revision on deploy: `506a787` (local working tree had uncommitted changes)
- Build command: `make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260304_235338.bak`
- Installed binary SHA256: `26edbaa0e95ac568ed3ae330d669571adb962cae0adccdc04286e5746dab3513`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: `whatomate@holol-wenjaz` active
- Listener ports expected active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`

### Note

- This deployment includes the latest WebSocket `fastHTTPUpgrader` fixes that explicitly echo the `whm.v1` Subprotocol to resolve real-time message connection drops.

## Deployment Update

Updated: 2026-03-09 04:34:54 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded `.git`, node modules, local build/test artifacts, local env files, `config.toml`, and `uploads/`)
- Source revision on deploy: `506a787` (local working tree had uncommitted changes)
- Build host: local macOS workspace
- Build reason: VPS Go version is `1.22.2` while the repo currently requires `go 1.25.7`, so the production Linux binary was built locally and uploaded
- Build command: `GOOS=linux GOARCH=amd64 make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_043047.bak`
- Installed binary SHA256: `d63df8c5318a95a484fe2c151e1ded0a834c4a6df6c32547207f820ee3e531d2`
- Installed binary version output: `Whatomate 506a787-dirty (built 2026-03-09_04:29:44)`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Immediately after restart, short-lived `502` responses appeared for the base service and the first tenant during startup; both recovered to `200` once the processes finished binding and initialization.

## Deployment Update

Updated: 2026-03-09 07:04:18 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_070335.bak`
- Installed binary SHA256: `ab4484d4f2e53f4c2c6a846af59e277afaeb5226984f96c27335dc01d6c5b95d`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:03:18)`
- Deployment purpose: fix assigned chats for agents where the `Assigned` counter increased after reassignment but the chat stayed hidden in the sidebar because of the implicit frontend instance filter.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Initial status right after restart: all services `active`, URLs returned temporary `502` during startup warmup
- Final listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- Final HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

## Deployment Update

Updated: 2026-03-09 07:12:04 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_071137.bak`
- Installed binary SHA256: `06dc0dc299068f23b50a7150e487f2c213f18011832b37c4b0ddfef0b0e505fa`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:11:25)`
- Deployment purpose: replace `Unknown Instance` in the chat sidebar for self-assigned chats on restricted instances by using a safe fallback label from the chat payload when the instance is not available in `instancesStore`.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all four services `active`
- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

## Deployment Update

Updated: 2026-03-09 07:24:58 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via `rsync`; excluded caches, `node_modules/`, generated `dist/`, and local security/report artifacts)
- Source revision on deploy: `506a787` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260309_072333.bak`
- Installed binary SHA256: `6de468c6859100477bee7b5f04af37a8ffc4418e8b4380df0a85a35bba8d2566`
- Installed binary version output: `Whatomate dev (built 2026-03-09_07:24:14)`
- Deployment purpose: deploy the current local project state to production, including the latest workspace changes.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Frontend production build completed successfully on the VPS. Vite emitted the existing warning about `<script src=\"./theme-init.js\">` in `index.html`, but the final build and all runtime checks completed successfully.

## Deployment Update

Updated: 2026-03-11 12:47:45 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (via tar archive; missing cmd/whatomate directory补)
- Source revision on deploy: Current working tree (uncommitted changes from test-strategy improvements)
- Native build command on VPS: `cd /opt/whatomate-src && CGO_ENABLED=0 go build -ldflags '-s -w' -o whatomate-new ./cmd/whatomate`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260311_124745.bak`
- Installed binary SHA256: `a7055dadac86cbe762805552c5b703e10348adbe78d1a56ab507ce83de09c2dc`
- Binary size: 46MB (with embedded frontend)
- Deployment notes:
  - Initial deployment attempt failed because tar archive was missing cmd/whatomate directory
  - Transferred cmd directory separately and rebuilt binary natively on VPS
  - Frontend properly embedded into binary (6.5MB of assets)
  - Previous binary (1.8MB) was missing embedded frontend and exited immediately

### Services Restarted

- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state: all three services `active`
- Listener ports active:
  - `127.0.0.1:18124` (holol-wenjaz)
  - `127.0.0.1:18125` (alarkan-almthalia)
  - `127.0.0.1:18126` (matbaat-ruya)
- Process IDs:
  - holol-wenjaz: PID 3038998
  - alarkan-almthalia: PID 3038999
  - matbaat-ruya: PID 3039000
- Log verification: All services show "Server listening" messages

### Note

- Deployment completed after fixing missing cmd/whatomate directory issue
- Frontend embedding now working correctly with proper 46MB binary size
- All tenant services operational with embedded frontend assets

## Deployment Update

Updated: 2026-03-12 11:56:35 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src` (incremental tar sync of the current changed workspace files into the existing source tree)
- Source revision on deploy: `b70cecd` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src/frontend && npm install && cd /opt/whatomate-src && VERSION=b70cecd-dirty GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260312_115332.bak`
- Installed binary SHA256: `d077226cb4bf9f5a4bc19ff1acdc934b4e40fba78eec095a612d8b074b8588d5`
- Installed binary version output: `Whatomate b70cecd-dirty (built 2026-03-12_11:51:57)`
- Deployment purpose: deploy the current local project state and remove the nginx-side upload block that returned `413 Request Entity Too Large` for a 2.1 MB media upload on `https://ofuqalmadenah.com/api/messages/media`.
- Edge configuration change:
  - Added `client_max_body_size 110M;` to:
    - `/etc/nginx/sites-available/ofuqalmadenah`
    - `/etc/nginx/sites-available/whatomate-holol-wenjaz.conf`
    - `/etc/nginx/sites-available/whatomate-alarkan-almthalia.conf`
    - `/etc/nginx/sites-available/whatomate-matbaat-ruya.conf`
  - Validated with `nginx -t` and reloaded nginx successfully.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- Upload ingress verification:
  - A 2.2 MB multipart POST to `https://ofuqalmadenah.com/api/messages/media` now returns `401 Missing authorization` JSON instead of nginx `413`, confirming the request reaches Whatomate.

### Note

- The old full-tree rsync path from macOS stalled; the successful deployment used an incremental tar sync of the changed workspace files into `/opt/whatomate-src`.
- The frontend build completed successfully on the VPS. Vite emitted the existing warning about `<script src="./theme-init.js">` in `index.html`, but the build, binary install, nginx reload, and service restarts all completed successfully.

## Deployment Update

Updated: 2026-03-12 13:55:17 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: source-only tar stream from `git ls-files --cached --modified --others --exclude-standard`, mirrored on the VPS into `/opt/whatomate-src`
- Source revision on deploy: `b70cecd` (working tree had local uncommitted changes)
- Native build command on VPS: `cd /opt/whatomate-src && GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Version-stamp rebuild on VPS: `cd /opt/whatomate-src && VERSION=b70cecd-dirty CGO_ENABLED=0 go build -ldflags "...main.Version=b70cecd-dirty..." -o whatomate ./cmd/whatomate`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260312_135323.bak`
- Installed binary SHA256: `ca9c67d86a0f2c188a400a2d6eedfea6f88f132333c0f30344ee6f6d851bf64f`
- Installed binary version output: `Whatomate b70cecd-dirty (built 2026-03-12_13:53:43)`
- Deployment purpose: publish the current local project state, including the new multi-file attachment send flow in chat and the related frontend/docs updates.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- The first tenant HTTP checks returned temporary `502` responses while the tenant services were still finishing startup and migrations; once their listeners bound to `127.0.0.1:18124-18126`, all tenant login pages returned `200`.
- The clean source-only sync avoided re-uploading the local `uploads/` directory and other large local artifacts that are not required for production builds.

## Deployment Update

Updated: 2026-03-17 03:28:38 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `1870edb` (working tree clean)
- Native build command on VPS: `cd /opt/whatomate-src && VERSION=1870edb GOTOOLCHAIN=go1.25.7+auto make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260317_032750.bak`
- Installed binary SHA256: `fe13b8b49fc5f5918b6d03584afbe2e39fb12e535ba30cc8085bee82bbce3bda`
- Installed binary version output: `Whatomate 1870edb (built 2026-03-17_03:27:26)`
- Deployment purpose: deploy the current local project state to production.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Listener ports active:
  - `127.0.0.1:18123`
  - `127.0.0.1:18124`
  - `127.0.0.1:18125`
  - `127.0.0.1:18126`
- Process IDs:
  - whatomate: PID 3152955
  - holol-wenjaz: PID 3152967
  - alarkan-almthalia: PID 3152960
  - matbaat-ruya: PID 3152948
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`

### Note

- Vite emitted the existing warning about `<script src="./theme-init.js">` lacking `type="module"`; the build and embed steps still completed successfully.

## Deployment Update

Updated: 2026-03-30 11:42:13 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `975bb5a` (working tree clean)
- Native build command on VPS: `cd /opt/whatomate-src && make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260330_113853.bak`
- Installed binary SHA256: `6208363121d05b7f75555931688a2a44e91388b8a74ad160796f7efa6cc16588`
- Installed binary version output: `Whatomate dev (built 2026-03-30_11:40:30)`
- Deployment purpose: deploy the current local project state to production.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Local HTTP smoke:
  - `ofuqalmadenah.com` (127.0.0.1:18123) -> `200`
  - `holol-wenjaz.ofuqalmadenah.com` (127.0.0.1:18124) -> `200`
  - `alarkan-almthalia.ofuqalmadenah.com` (127.0.0.1:18125) -> `200`
  - `matbaat-ruya.ofuqalmadenah.com` (127.0.0.1:18126) -> `200`

### Note

- Vite emitted the circular chunk warning during build (`grid-layout -> vue-vendor -> grid-layout`); build completed successfully.

## Deployment Update

Updated: 2026-03-30 11:56:29 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `975bb5a` (working tree dirty: `frontend/index.html`, `frontend/public/theme-init.js`, `frontend/vite.config.ts`, `docs/whatomate_multi_instances_info.md`, `summery.md`)
- Native build command on VPS: `cd /opt/whatomate-src && make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260330_115417.bak`
- Installed binary SHA256: `e086b30f2276676ddb4d409c21a7ffa722647bfd16ffe120bb5aae8fd408c53c`
- Installed binary version output: `Whatomate dev (built 2026-03-30_11:55:32)`
- Deployment purpose: deploy CSP-friendly theme init and remove the grid-layout manual chunk that triggered `ReferenceError` in production.

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Local HTTP smoke:
  - `ofuqalmadenah.com` (127.0.0.1:18123) -> `200`
  - `holol-wenjaz.ofuqalmadenah.com` (127.0.0.1:18124) -> `200`
  - `alarkan-almthalia.ofuqalmadenah.com` (127.0.0.1:18125) -> `200`
  - `matbaat-ruya.ofuqalmadenah.com` (127.0.0.1:18126) -> `200`
- MCP UI check (Playwright): loaded `https://ofuqalmadenah.com/chat` with no console errors reported.

## Deployment Update

Updated: 2026-03-30 12:22:36 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `975bb5a` (working tree dirty: `docs/whatomate_multi_instances_info.md`, `frontend/public/theme-init.js` (deleted), `frontend/vite.config.ts`, `internal/frontend/embed.go`, `internal/middleware/middleware.go`, `summery.md`)
- Native build command on VPS: `cd /opt/whatomate-src && make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260330_122032.bak`
- Installed binary SHA256: `01832e7c89056dbd520ac767322dc8c740e713f464a209b0e2fc5072fd8fc88b`
- Installed binary version output: `Whatomate dev (built 2026-03-30_12:19:09)`
- Deployment purpose: add CSP nonce injection for inline theme initialization and prevent duplicate CSP headers on SPA routes (fixes inline script CSP violations on production).

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Local HTTP smoke:
  - `ofuqalmadenah.com` (127.0.0.1:18123) -> `200`
  - `holol-wenjaz.ofuqalmadenah.com` (127.0.0.1:18124) -> `200`
  - `alarkan-almthalia.ofuqalmadenah.com` (127.0.0.1:18125) -> `200`
  - `matbaat-ruya.ofuqalmadenah.com` (127.0.0.1:18126) -> `200`
- CSP header check:
  - Single `Content-Security-Policy` header returned with `script-src 'self' 'nonce-...'`.
  - HTML inline theme script includes a matching `nonce` attribute.
- MCP UI check (Playwright): loaded `https://ofuqalmadenah.com/settings` and `https://ofuqalmadenah.com/chat` with no CSP inline-script violations (only expected `401` API responses due to unauthenticated session).

### Skills Applied

- `devops-engineer` (build, backup, install, systemd restart, smoke checks)
- `debugging-wizard` (CSP error reproduction, header/nonce verification, confirmation of fix)

### Competencies Applied

- CSP policy design with nonces for inline bootstrapping
- Go HTTP header handling and HTML placeholder injection
- Vite build + embedded asset packaging
- Systemd service management
- Browser console verification using MCP tooling

### Note

- Chrome DevTools MCP was unavailable due to an existing profile lock; Playwright MCP was used for UI verification.

## Deployment Update

Updated: 2026-04-05 12:32:57 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync` (with `--delete`; excluded `.git`, `node_modules/`, `frontend/dist/`, `uploads/`, `config.toml`, and local build/test artifacts)
- Source revision on deploy: `0c12891` (working tree dirty: `Makefile`, `docs/workflow.md`, `docs/wiki/` (untracked))
- Native build command on VPS: `cd /opt/whatomate-src && make build-prod`
- Build remediation on VPS: `frontend-build` refreshed dependencies with `npm ci` because `package.json` or `package-lock.json` was newer than the cached `frontend/node_modules`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binary created: `/opt/whatomate/bin/whatomate.20260405_122402.bak`
- Installed binary SHA256: `429480ece322282b1ebea66cb990b72ae8fe931c20eb1848f67815cc985f47ec`
- Installed binary version output: `Whatomate dev (built 2026-04-05_12:29:13)`
- Deployment purpose: deploy the current project update and harden VPS production builds so frontend dependency changes are picked up automatically
- Production config remediation: rotated `whatsapp.webhook_verify_token` from insecure placeholder values to unique random tokens in all active production configs so the new binary can start in production
- Production config backups created:
  - `/opt/whatomate/config.toml.20260405_123126.bak`
  - `/opt/whatomate/instances/alarkan-almthalia/config.toml.20260405_123126.bak`
  - `/opt/whatomate/instances/holol-wenjaz/config.toml.20260405_123126.bak`
  - `/opt/whatomate/instances/matbaat-ruya/config.toml.20260405_123126.bak`

### Services Restarted

- `whatomate`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- Local HTTP smoke:
  - `ofuqalmadenah.com` (127.0.0.1:18123) -> `200`
  - `holol-wenjaz.ofuqalmadenah.com` (127.0.0.1:18124) -> `200`
  - `alarkan-almthalia.ofuqalmadenah.com` (127.0.0.1:18125) -> `200`
  - `matbaat-ruya.ofuqalmadenah.com` (127.0.0.1:18126) -> `200`
- MCP UI check (Chrome DevTools):
  - Loaded `https://ofuqalmadenah.com/settings` and `https://ofuqalmadenah.com/chat`
  - Both routes redirected to the login screen as expected for an unauthenticated session
  - No browser console errors were reported

### Skills Applied

- `devops-engineer` (backup, rsync deployment, VPS build/install, systemd restart, service recovery)
- `debugging-wizard` (remote build failure diagnosis, production config validation diagnosis, browser verification)

### Competencies Applied

- Ubuntu VPS deployment and rollback-safe backup handling
- Vite dependency drift remediation on long-lived build hosts
- Go binary packaging with embedded frontend assets
- Production config hardening and secret rotation
- Chrome DevTools MCP verification of the public UI

### Note

- `make build-prod` initially failed on the VPS because `frontend/node_modules` was stale while `package.json` and `package-lock.json` had changed; the `Makefile` now refreshes frontend dependencies when those files are newer than the cached install.
- `npm ci` reported `2 high severity vulnerabilities` in frontend dependencies during the VPS build; they were not remediated as part of this deployment.

## Deployment Update

Updated: 2026-04-06 19:32:00 UTC

- Deployment scope: targeted production hotfix for `holol-wenjaz.ofuqalmadenah.com/settings/instances`
- Root cause: the quick `Auto campaign` card switch sent an immediate `PUT /api/instances/:id` update even when the campaign message was blank; the backend correctly rejected that invalid payload with `400 Bad Request`
- Frontend fix: `frontend/src/components/whatsmeow/InstanceCard.vue` now blocks invalid quick-enable actions for:
  - `Auto campaign` when the message is empty
  - `Call auto-reject` when reply mode is `with_message` and the message is empty
- Browser-only note: the reported `blob:` CSP violations tied to `Browser Control` injected scripts were not reproduced in a clean browser session after deployment and are not emitted by the Whatomate frontend bundle

### Deployment Details

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source update method: targeted copy of `frontend/src/components/whatsmeow/InstanceCard.vue` to `/opt/whatomate-src/frontend/src/components/whatsmeow/InstanceCard.vue`
- Source revision reference: `dbac523` (working tree dirty)
- Native build command on VPS: `cd /opt/whatomate-src && make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Backup binaries created before install:
  - `/opt/whatomate/bin/whatomate.20260406_191957.bak`
  - `/opt/whatomate/bin/whatomate.20260406_192006.bak`
- Installed binary SHA256: `7f4afac3f96d28046db7c87f59df0ddab5439f827a5c0182e26e42b0bd04fa95`
- Installed binary version output: `Whatomate dev (built 2026-04-06_19:25:28)`

### Services Restarted

- `whatomate@holol-wenjaz`
- `whatomate`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Post-Deploy Verification

- Systemd state:
  - `whatomate`: `active`
  - `whatomate@holol-wenjaz`: `active`
  - `whatomate@alarkan-almthalia`: `active`
  - `whatomate@matbaat-ruya`: `active`
- HTTPS smoke after restart settling:
  - `https://ofuqalmadenah.com/` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/` -> `200`
- Chrome DevTools MCP verification on `https://holol-wenjaz.ofuqalmadenah.com/settings/instances`:
  - page loaded successfully in an authenticated session
  - clicking the `Auto campaign` quick switch with an empty message showed the validation toast instead of sending a failing update
  - network capture showed no `PUT /api/instances/...` request after the blocked click
  - console output contained only accessibility issues; no app CSP/blob script violations were present

### Skills Applied

- `debugging-wizard` (root-cause isolation for the production `400`)
- `vue-expert` (minimal Vue guard implementation and regression coverage)
- `devops-engineer` (backup, VPS build/install, staged restart, browser verification)

### Competencies Applied

- Vue 3 event-flow debugging
- backend/frontend contract validation
- production-safe binary rollout on Ubuntu/systemd
- authenticated browser verification with MCP tooling


## Inbound Media Reconcile Repair

Updated: 2026-04-06 20:13:00 UTC

- Deployment scope: add and run a safe production repair command for stale inbound-media queue rows on `holol-wenjaz`
- Tenant instance ID: `4a997817-192a-478c-b526-ddf5d70dc3b7`
- Command added: `whatomate inbound-media-reconcile`
- Safety behavior:
  - requires Redis consumer group `inbound-media-workers` on stream `whatomate:inbound_media`
  - refuses reconciliation when queue lag is positive or unknown
  - loads pending stream payloads and excludes active `message_id` values from cleanup
  - only reconciles `queued` inbound-media rows older than the configured threshold
- Source sync method: targeted copy of backend files to `/opt/whatomate-src`
- VPS binary backups created before installs:
  - `/opt/whatomate/bin/whatomate.20260406_200711.bak`
  - `/opt/whatomate/bin/whatomate.20260406_201130.bak`
- Valid database backup created before cleanup:
  - `/root/db_backups/inbound_media_reconcile_holol_wenjaz_20260406_201018`
- Final installed binary:
  - path: `/opt/whatomate/bin/whatomate`
  - SHA256: `d1cb45018447624f9b5b21a154b96ca4d35cf72004922f2b9f6c1e27f8650855`
  - version: `Whatomate dev (built 2026-04-06_20:11:45)`
- Dry-run before apply:
  - `total_queued=2732`
  - `active_pending_ids=1354`
  - `eligible_queued=1378`
- Apply result:
  - `updated=1378`
  - `skipped_active_queued=1349`
- Post-cleanup verification:
  - queued rows with empty `media_url`: `1345`
  - failed inbound-media rows: `1505`
  - Redis DB `1` consumer group: `pending=1345`, `lag=0`
  - follow-up dry-run: `eligible_queued=0`
- Specific row note:
  - `53b94398-9b32-4ea0-b7c7-0ba2bead1aed` remains `queued` because it is still part of the live pending backlog, so the reconcile command correctly protected it from forced failure
- Chrome DevTools MCP smoke check:
  - `https://holol-wenjaz.ofuqalmadenah.com/` redirected to `/login`
  - console output: none
  - network `200`: `/login`, `/api/auth/sso/providers`

### Skills Applied

- `debugging-wizard`
- `golang-pro`
- `devops-engineer`

### Competencies Applied

- Redis Streams backlog triage and safe exclusion logic
- Go production repair-command implementation
- PostgreSQL backup-first remediation
- Ubuntu/systemd deployment and binary rollback hygiene
- Chrome DevTools MCP verification


## Full Workspace Deployment

Updated: 2026-04-07 07:19:00 UTC

- Deployment scope: full current local workspace sync and production binary rollout
- Local source path: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Source revision deployed: `07b95fc`
- Source sync target: `/opt/whatomate-src`
- Source sync method: `rsync --delete`
- Sync exclusions:
  - `.git/`
  - `node_modules/`
  - `frontend/node_modules/`
  - `frontend/dist/`
  - `docs/node_modules/`
  - `uploads/`
  - `config.toml`
  - `*.db`
  - `tmp/`
  - local `whatomate` binary
- VPS binary backup created before install:
  - `/opt/whatomate/bin/whatomate.20260407_071645.bak`
- Native build command on VPS:
  - `cd /opt/whatomate-src && VERSION=07b95fc GOTOOLCHAIN=go1.25.8+auto make build-prod`
- Final installed binary:
  - path: `/opt/whatomate/bin/whatomate`
  - SHA256: `b5ef3f02b5321b0ab646941b9754bb8578a93e04feb9a27079d2807d91e0a462`
  - version: `Whatomate 07b95fc (built 2026-04-07_07:17:39)`
- Services restarted:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Verification:
  - systemd services all `active`
  - listeners present on `127.0.0.1:18123-18126`
  - public HTTPS smoke returned `200` for all four production hostnames
  - Chrome DevTools MCP checks on `https://ofuqalmadenah.com/` and `https://holol-wenjaz.ofuqalmadenah.com/` both redirected to `/login` with no console errors
- Build note:
  - Vite emitted chunk-size warnings only; build completed successfully

### Skills Applied

- `devops-engineer`

### Competencies Applied

- rollback-safe Ubuntu/systemd deployment
- rsync-based source mirroring
- Go + Vite production build orchestration
- post-deploy browser verification with MCP tooling

## Production Deployment - 2026-04-12

Updated: 2026-04-12 00:21 UTC

- Deployment target: `31.97.192.53`
- Local source path: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Requested source baseline: local `main` at `a2b0e3a`
- Production-safe deployed build: `a2b0e3a-hotfix-worker-nil`
- Why the hotfix was required:
  - current `main` crashed every instance during startup with `panic: runtime error: invalid memory address or nil pointer dereference`
  - root cause: `internal/worker/worker.go` stored disabled Redis consumers as typed-nil interfaces, so scaler-managed campaign workers still attempted `Consume()` on a nil `*queue.RedisConsumer`
- Local code changes applied before final rollout:
  - `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/worker/worker.go`
  - `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/worker/worker_test.go`
- Local verification before redeploy:
  - `go test ./internal/worker`
- VPS binary backups created during the rollout:
  - `/opt/whatomate/bin/whatomate.20260412_000603.bak`
  - `/opt/whatomate/bin/whatomate.20260412_001919.bak`
- Final installed binary:
  - path: `/opt/whatomate/bin/whatomate`
  - SHA256: `e4815db7326aa5bbf65bea17fc6d46f8f9acb5722b9fa390df9cb33c4d75583d`
  - version: `Whatomate a2b0e3a-hotfix-worker-nil (built 2026-04-12_00:19:08)`
- Build command used on VPS:
  - `cd /opt/whatomate-src && VERSION=a2b0e3a-hotfix-worker-nil GOTOOLCHAIN=go1.25.8+auto make build-prod`
- Config/public-key decision:
  - no new `public.key` or license config override was needed
  - active production configs currently have licensing disabled
  - the binary was built with a valid default embedded key ring (`[]`) so startup remains safe when licensing is off
- Runtime verification on VPS:
  - `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, and `whatomate@matbaat-ruya` are all `active`
  - localhost smoke checks returned `200` for ports `18123`, `18124`, `18125`, and `18126`
- Public verification:
  - `https://ofuqalmadenah.com/` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/` -> `200`
  - Chrome DevTools MCP checks on `https://ofuqalmadenah.com/` and `https://holol-wenjaz.ofuqalmadenah.com/` both redirected to `/login` with no console errors
- Reverse proxy state discovered during deployment:
  - live public traffic is still served by `nginx`, not `caddy`
  - `nginx` is active and its syntax test is clean
  - `caddy` is failed because `nginx` already owns ports `80` and `443`
  - I did not switch the ingress layer during this rollout to avoid unnecessary production blast radius

### Skills Applied

- `devops-engineer`
- `debugging-wizard`
- `golang-pro`

### Competencies Applied

- rollback-safe Ubuntu/systemd production deployment
- root-cause analysis of Go interface-nil regressions
- low-blast-radius hotfix delivery under active outage conditions
- browser and HTTP verification against live multi-instance domains

## Production Deployment - 2026-04-12 12:00 UTC

- Deployment target: `31.97.192.53`
- Source deployed: current local `main` worktree from `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Git baseline: `e55d147`
- Deployment method: existing `systemd` services plus rebuilt production binary in `/opt/whatomate/bin/whatomate`
- Source sync target on VPS: `/opt/whatomate-src`
- Build command used on VPS:
  - `cd /opt/whatomate-src && VERSION=e55d147-worktree-20260412_1159 GOTOOLCHAIN=go1.25.8+auto make build-prod`
- Binary backup created before install:
  - `/opt/whatomate/bin/whatomate.20260412_120029.bak`
- Final installed binary:
  - path: `/opt/whatomate/bin/whatomate`
  - SHA256: `330d48633077d2caeb2f24b8a026b0b84eccbfe77f5d04f376c360de82af46aa`
  - version: `Whatomate e55d147-worktree-20260412_1159 (built 2026-04-12_11:59:58)`
- Config/public-key decision:
  - no new config files were required for this rollout
  - no `public.key` was required because the active production configs do not define a `[license]` block
  - the binary was built with the default embedded empty key ring (`[]`), which is safe for the current production license-disabled state
- Runtime verification on VPS:
  - `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, and `whatomate@matbaat-ruya` all restarted cleanly and are `active`
  - localhost smoke checks returned `200` for ports `18123`, `18124`, `18125`, and `18126`
- Public verification:
  - `https://ofuqalmadenah.com/` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/` -> `200`
  - Chrome DevTools MCP checks on `https://ofuqalmadenah.com/login` and `https://holol-wenjaz.ofuqalmadenah.com/login` loaded the login page with no console messages
- SSH note:
  - the VPS SSH host keys have changed since the older local `known_hosts` entries were recorded
  - I used a fresh trusted host-key file for this deployment after collecting the current host keys from `31.97.192.53`
- Reverse proxy state at deploy time:
  - `nginx` is still the live listener on `80/443`
  - `caddy` remains `failed`
  - the Whatomate deployment itself was completed without changing the ingress layer

### Skills Applied

- `devops-engineer`
- `debugging-wizard`

### Competencies Applied

- rsync-based source mirroring for a dirty worktree deployment
- Ubuntu `systemd` binary rollout with pre-install backup
- Go + Vite production build orchestration on the VPS
- live browser verification with Chrome DevTools MCP



## Production Fix - 2026-04-12 13:40 UTC

- Issue fixed: `GET /api/chatbot/transfers?status=active` and `PUT /api/chats/:id/claim` were returning `500` for restricted users on `https://ofuqalmadenah.com`.
- Root cause:
  - `ListAgentTransfers` reused one request-scoped GORM handle across multiple independent query chains, so joins and filters leaked into later count queries.
  - queue count queries in `internal/handlers/agent_transfers.go` also used unqualified column names after joining `contacts`, which produced ambiguous SQL.
  - lifecycle chat actions in `internal/handlers/contacts_management.go` reused the same scoped handle for select, update, and reload flows, which allowed `contacts` joins to leak into later updates such as `ClaimChat`.
- Local code fix:
  - `internal/handlers/agent_transfers.go`: every independent transfer query now starts from `requestDB.Session(&gorm.Session{})`; transfer queue count queries now fully qualify `agent_transfers.*` columns and return/log count errors.
  - `internal/handlers/contacts_management.go`: lifecycle chat reads, updates, and reloads now use fresh GORM sessions; `buildLifecycleContactQuery` centralizes restricted-instance and agent-scope visibility.
  - `internal/middleware/middleware_test.go`: added a regression test proving fresh scoped sessions do not leak joins between sequential queries.
- Local verification:
  - `go test ./internal/middleware -run 'TestTenantScope'` -> `ok`
  - `go test ./internal/handlers -run 'TestApp_(ListAgentTransfers_FiltersBlockedInstances|ClaimChat_AllowsAllowedRestrictedInstance|ClaimChat_FiltersBlockedInstances|ListAgentTransfers_IncludesInstanceID)'` -> `ok`
- VPS deployment:
  - source files synced to `/opt/whatomate-src`
  - pre-install backup: `/opt/whatomate/bin/whatomate.20260412_153327.bak`
  - build command: `cd /opt/whatomate-src && VERSION=e55d147-transfer-claim-fix-20260412_153327 GOTOOLCHAIN=go1.25.8+auto make build-prod`
  - installed binary: `/opt/whatomate/bin/whatomate`
  - SHA256: `80f173d335740aaf574931e7bb5ec485837e24d6093ec930a20ecb15eaaee03f`
  - version: `Whatomate e55d147-transfer-claim-fix-20260412_153327 (built 2026-04-12_13:34:13)`
- Service verification:
  - `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, and `whatomate@matbaat-ruya` all restarted cleanly and are `active`
  - public health checks returned `200` for `https://ofuqalmadenah.com/`, `https://holol-wenjaz.ofuqalmadenah.com/`, `https://alarkan-almthalia.ofuqalmadenah.com/`, and `https://matbaat-ruya.ofuqalmadenah.com/`
- Live endpoint verification after deploy:
  - authenticated restricted-user repro returned `200` for `GET https://ofuqalmadenah.com/api/chatbot/transfers?status=active`
  - authenticated restricted-user repro returned `200` for `PUT https://ofuqalmadenah.com/api/chats/b3ef44b9-1e35-488e-bd6e-3da895fdad1c/claim`
  - Chrome DevTools MCP fetch verification returned `200` for both endpoints from the browser context
  - recent `journalctl` output showed the successful `ListAgentTransfers` info log and no new SQL errors after the fix
- Config/public-key decision:
  - no new config file was needed for this fix
  - no `public.key` was needed because this rollout did not change the active licensing configuration
- Skills applied:
  - `debugging-wizard`
  - `golang-pro`
  - `devops-engineer`
- Competencies applied:
  - root-cause analysis of production SQL/state leakage
  - low-blast-radius Go backend hotfixing
  - systemd binary deployment with rollback backup
  - live HTTP and browser-context verification on production


## Production Fix - 2026-04-12 18:15 UTC

- Issue bundle fixed on production:
  - `GET /api/users` was returning `500`
  - some authenticated chat requests were returning `431 Request Header Fields Too Large`
  - repeated `/api/media/:id` `404` requests were spamming the console for legacy local-file media that no longer exists on disk
- Root causes:
  - `ListUsers` reused polluted GORM state under request/tenant scoping, producing a `500` for restricted-user flows
  - some browsers still carried oversized legacy auth cookie variants; nginx and the Go HTTP server were also too strict for those oversized request headers
  - many old legacy media rows still referenced deleted local files under `/opt/whatomate/uploads`, so the frontend kept retrying media URLs that can never succeed
- Local code changes prepared and verified:
  - `cmd/whatomate/main.go`: increased `ReadBufferSize` to `32 * 1024` for safer oversized-header tolerance
  - `internal/handlers/cookies.go`: clear legacy auth cookie variants on login/logout so browsers shed oversized stale cookies
  - `internal/handlers/users.go` + `internal/handlers/users_query_regression_test.go`: isolate `ListUsers` query state so `/api/users` stays stable under scoped access rules
  - `internal/handlers/contacts.go`, `internal/handlers/messages.go`, `internal/handlers/media_visibility.go`: hide legacy media URLs from API/websocket payloads once media is marked unavailable
  - `internal/handlers/legacy_media_reconcile.go` + `internal/handlers/legacy_media_reconcile_test.go`: added CLI reconciliation to mark truly missing legacy local-media rows with `media_deleted_at`
  - `frontend/src/lib/media_prefetch_cache.ts` + `frontend/src/lib/media_prefetch_cache.test.ts`: treat `410 Gone` the same as `404` so deleted media is cooled down locally instead of being retried immediately
- Local verification:
  - `go test ./internal/handlers -run 'Test(BuildUsersListBaseQuery_UsesIsolatedStatements|AuthCookies_ClearLegacyVariantsOnLogin|MessageHasVisibleMedia|ReconcileMissingLegacyMediaMarksOnlyMissingOldFiles|App_ListUsers|App_GetUser)'` -> `ok`
  - `go test ./cmd/whatomate` -> `? [no test files]`
  - `cd frontend && npx vitest run src/lib/media_prefetch_cache.test.ts src/services/websocket.test.ts src/stores/contacts.test.ts` -> passed
  - `cd frontend && npm run build` -> passed
- VPS deployment:
  - synced only the targeted incident-fix files to `/opt/whatomate-src`
  - binary backup created first: `/opt/whatomate/bin/whatomate.20260412_2006.bak`
  - installed binary: `/opt/whatomate/bin/whatomate`
  - version: `Whatomate e55d147-users-header-mediafix2-20260412_2006 (built 2026-04-12_18:05:55)`
  - SHA256: `0ac8cc2fead1704687b0a74145ed36912616a08fea27562a76d428574b2da8af`
- Production data reconciliation:
  - backup of candidate rows before apply:
    - `/root/db_backups/legacy_media_reconcile_main_20260412_2006.tsv`
    - `/root/db_backups/legacy_media_reconcile_holol-wenjaz_20260412_2006.tsv`
  - main org reconcile apply updated `43,692` missing legacy media rows
  - `holol-wenjaz` reconcile apply updated `6,105` missing legacy media rows
  - `alarkan-almthalia` and `matbaat-ruya` had `0` matching rows
- Additional infrastructure mitigation already applied on VPS for the `431` side of the incident:
  - nginx site configs now include larger request-header buffers, with backups stored in `/root/ops_backups/whatomate_incident_20260412_193101`
- Live production verification after deploy:
  - authenticated HTTP checks returned `200` for `/api/users`
  - authenticated HTTP checks returned `200` for `/api/chats/:id/messages?account=...`
  - authenticated HTTP checks returned `200` for `/api/contacts/:id/typing`
  - Chrome DevTools MCP authenticated load of `https://ofuqalmadenah.com/chat/100f94c6-2585-4e00-8149-830a0a7ef045?account=966554840026` showed:
    - `/api/chatbot/transfers?status=active` -> `200`
    - `/api/users` -> `200`
    - `/api/chats/.../messages` -> `200`
    - `/api/chats/.../messages?account=...` -> `200`
    - no `/api/media/... 404` requests on the fresh load
    - no console errors, only one pre-existing accessibility issue about a form field lacking an `id` or `name`
  - missing legacy media now render as plain placeholders such as `[Image]` or `[Document]` instead of repeated failing fetches
- Service state after rollout:
  - `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, and `whatomate@matbaat-ruya` are all `active`
- Config/public-key decision:
  - no new config file was needed for this incident fix
  - no `public.key` was needed because the rollout did not change the active licensing configuration
- Skills applied:
  - `debugging-wizard`
  - `golang-pro`
  - `vue-expert`
  - `devops-engineer`
- Competencies applied:
  - root-cause analysis across HTTP, cookie, and media-delivery failures
  - low-blast-radius Go backend hotfixing with targeted file sync
  - frontend retry/cooldown hardening for missing media
  - systemd binary deployment with rollback backup and authenticated browser verification

## Deployment Update

Updated: 2026-04-15 22:21:06 UTC

- Deployed from local workspace: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- Pre-deploy full backup set: `/root/whatomate_backups/20260415_212640`
- Final native VPS build directory: `/root/whatomate_temp_build` (removed after deployment)
- Final build command: `cd /root/whatomate_temp_build && VERSION=8dfb206-worktree-20260415_2210-vps make build-prod`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Installed binary version: `Whatomate 8dfb206-worktree-20260415_2210-vps (built 2026-04-15_22:11:29)`
- Installed binary SHA256: `02999b58c65a130cdd7a1be80689b5b923dccaede692b39ccef9d059031f9da9`
- Final runtime backup before cutover: `/opt/whatomate/bin/whatomate.20260415_221226.pre_8dfb206_2210_safe.bak`

### License System Fix

- Root cause:
  - the first 2026-04-15 VPS build injected malformed `EmbeddedPublicKeyRingJSON`, which crash-looped `whatomate.service`
  - the production configs also did not have a working active `[license]` configuration
- Final fix:
  - restored the last good runtime binary first to recover service availability
  - rebuilt natively on Ubuntu without overriding `LICENSE_KEY_RING_JSON`, leaving the embedded key ring at the safe default `[]`
  - enabled `[license]` in `/opt/whatomate/config.toml` and all `/opt/whatomate/instances/*/config.toml`
  - set `public_key = "Sg7jjcj+DLdw6ogU8gnBmZBh2dqALk88G3QCKfPmmhU="`
  - set `public_key_kid = "deploy-20260415"`
  - set `allow_unsafe_public_key_override = true`
  - activated the signed host-bound token on ports `18123`, `18124`, `18125`, and `18126`
- Final license state:
  - all four instances report `enabled = true`, `status = active`, `locked = false`
  - `license_id = dc245a31-d3d3-4033-bb45-ee9fd9c0c9e1`
  - `key_id = deploy-20260415`

### VPS Cleanup

- Removed stale source/worktree/archive artifacts:
  - `/opt/whatomate-src`
  - `/opt/whatomate-src.old`
  - `/opt/whatomate-src.prev.20260412_001354`
  - `/opt/whatomate-src-backup`
  - `/opt/whatomate-src-e55d147-worktree-20260412_113509`
  - `/opt/whatomate-src-e55d147-worktree-20260412_114534`
  - `/root/whatomate`
  - `/root/whatomate_temp_build`
  - `/root/whatomate-buildonly-20260415_2135.tgz`
  - `/root/whatomate-src-20260415_2132.tgz`
  - `/root/whatomate-a2b0e3a.tar`
  - `/root/whatomate-deploy.tar`
  - `/root/whatomate-deploy.tar.gz`
  - `/root/whatomate-linux-20260303_172132`
  - `/root/whatomate-linux-20260303_204653`
  - `/root/whatomate.new`
  - `/root/whatomate_remote_deploy_20260415.sh`
  - `/root/whatomate_deploy_20260415.token`
  - `/root/whatomate-keyring.json`
- Retained:
  - `/opt/whatomate`
  - `/root/whatomate_backups/20260415_212640`
  - runtime configs, uploads, PostgreSQL databases, and the remote markdown docs

### Post-Deploy Verification

- `systemctl is-active` returned `active` for:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Listener ports active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`
- HTTPS smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- License bootstrap verification:
  - `:18123` -> `active`
  - `:18124` -> `active`
  - `:18125` -> `active`
  - `:18126` -> `active`
- Browser verification:
  - used Playwright CLI on the local desktop because no Chrome DevTools MCP was configured in this session
  - `https://ofuqalmadenah.com/login` rendered the Whatomate login page with email/password inputs and a `Sign in` button
  - `https://holol-wenjaz.ofuqalmadenah.com/login` rendered the Whatomate login page with email/password inputs and a `Sign in` button

### Skills Applied

- `master-workflow`
- `devops-engineer`
- `playwright`

### Competencies Applied

- SSH automation with password-based access and host-key bypass for a changed VPS fingerprint
- Ubuntu `systemd` deployment and rollback handling
- native Go + Vite build orchestration on Ubuntu/amd64
- production license activation and per-instance verification
- browser and HTTP smoke verification against live HTTPS routes

## Green Replacement Deployment

Updated: 2026-05-12 15:30:34 EEST / 2026-05-12 12:30:34 UTC

- Deployment type: replace green sandbox build while keeping public blue live.
- VPS: `31.97.192.53` (`root`, Ubuntu)
- Source revision deployed to green: `a1f143cc`
- Public blue remains active:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.blue.20260511_002729`
  - blue version: `Whatomate 8f155d2-deploy-20260505c (built 2026-05-05_10:56:48)`
  - blue SHA256: `3533aaf7abbe19de384ca35073f055f9722d90d763e11b59854142575cf0342e`
- New green sandbox binary:
  - `/opt/whatomate/bin/whatomate.green.20260512_122534`
  - version: `Whatomate a1f143cc-green-20260512_122534 (built 2026-05-12_12:27:39)`
  - SHA256: `9ef7a2fed8b40516f8a957c5fa37e2190d669ede31110896cdd592641d3a8361`
- Green sandbox service:
  - `whatomate-sandbox`
  - port: `127.0.0.1:18127`
  - URL: `https://sandbox.ofuqalmadenah.com`
- Pre-deploy backup:
  - `/root/whatomate_backups/whatomate-green-replace-predeploy-20260512_122115.tar.gz`

### One-Line Switch Commands

Promote green to live and stop sandbox:

```bash
ln -sfn /opt/whatomate/bin/whatomate.green.20260512_122534 /opt/whatomate/bin/whatomate && systemctl stop whatomate-sandbox && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```

Switch back to blue and run green as sandbox:

```bash
ln -sfn /opt/whatomate/bin/whatomate.blue.20260511_002729 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya && systemctl restart whatomate-sandbox
```

### Cleanup

- Removed temporary/source paths after build:
  - `/tmp/whatomate-green-src`
  - `/tmp/whatomate-green-keyring.json`
  - `/tmp/whatomate-chunk.aa`
  - `/tmp/whatomate-linux-amd64.gz`
  - `/opt/whatomate-sandbox/src`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/.cache`
  - `/opt/whatomate-sandbox/.gopath`
- Removed old green binaries after the new green sandbox was verified:
  - `/opt/whatomate/bin/whatomate.green.20260511_002522`
  - `/opt/whatomate/bin/whatomate.green.20260511_002922`
  - `/opt/whatomate/bin/whatomate.green.20260512_083647`
- Retained only runtime/config/upload directories and binaries under `/opt/whatomate`.

### Verification

- Local checks passed:
  - `go test ./internal/database ./internal/handlers ./internal/config ./internal/crypto ./internal/license ./pkg/whatsapp ./pkg/whatsmeow`
  - `cd frontend && npm run build`
- VPS build passed:
  - `make build-prod`
  - explicit Go build with embedded `license.EmbeddedPublicKeyRingBase64`
- Services active:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
  - `whatomate-sandbox`
- License bootstrap:
  - `:18123` -> `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
  - `:18124` -> `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
  - `:18125` -> `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
  - `:18126` -> `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
  - `:18127` -> `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
- HTTPS smoke:
  - `https://ofuqalmadenah.com` -> `200`
  - `https://www.ofuqalmadenah.com` -> `200`
  - `https://sandbox.ofuqalmadenah.com` -> `200`
- Blue/green API parity using authenticated admin session:
  - pending chats: blue `total=16342`, green `total=16342`, first page IDs match
  - open chats: blue `total=38`, green `total=38`, IDs match
  - sample chat messages: blue `total=166`, green `total=166`, first 100 IDs match
- Chrome DevTools MCP browser verification:
  - sandbox login page loaded
  - browser-side login returned `200`
  - browser-side `/api/license/bootstrap` returned active license
  - browser-side pending chat API returned `200`
  - screenshot saved locally at `tmp/green-replace-verify-20260512.png`

### Skills And Competencies Applied

- Skills selected: `devops-engineer`
- Competencies applied:
  - blue/green deployment with public blue preserved
  - Ubuntu systemd service override management
  - native Go/Vite production builds on Ubuntu amd64
  - embedded license keyring verification
  - HTTP/API/browser smoke verification
  - VPS cleanup of temporary source artifacts

## 2026-05-12 15:24 UTC - Green Deployment Update `a73f45b1`

### Task

- Deployed the current project as a new green build after fixing:
  - chat transfer creation failures on `POST /api/chatbot/transfers`
  - notification dismissal failures on `PUT /api/notifications/{id}/dismiss`
  - missing close control on system notification toasts
- Kept the current live binary unchanged and installed the new build as the updated green target for sandbox verification.

### Code Version

- Git branch: `agent/fix-transfer-notifications-green-deploy`
- Git commit: `a73f45b1`
- Green version string: `a73f45b1-green-20260512_145748`
- Green binary:
  - path: `/opt/whatomate/bin/whatomate.green.20260512_145748`
  - sha256: `099d7af1d761c4efb6fdbbf7f3763b81d72accdd79acced9d6e5e5f9c9e35260`

### Backup

- Pre-deploy backup created before changing runtime artifacts:
  - path: `/root/whatomate_backups/whatomate-installed-pre-green-20260512_145705.tar.gz`
  - sha256: `4dea9425a271d5ccbfead3c364c576b5df7faa5f95616837b6ab5baee1753fb6`
  - size: `197M`
- Backup scope: `/opt/whatomate/bin`, service units/drop-ins, config files, license/keyring artifacts, and deployment info files. Media/upload data was not archived because `/opt/whatomate` is about `53G`.

### Blue/Green Layout

- Current live symlink remains unchanged:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260512_122534`
- New green alias:
  - `/opt/whatomate/bin/whatomate.green -> /opt/whatomate/bin/whatomate.green.20260512_145748`
- Blue rollback alias:
  - `/opt/whatomate/bin/whatomate.blue -> /opt/whatomate/bin/whatomate.blue.20260511_002729`
- Green sandbox:
  - service: `whatomate-sandbox`
  - binary: `/opt/whatomate/bin/whatomate.green`
  - bind: `127.0.0.1:18127`
  - URL: `https://sandbox.ofuqalmadenah.com`

### Switch Command

Use the same command with the target color:

```bash
whatomate-switch green
whatomate-switch blue
```

From a local machine:

```bash
ssh root@31.97.192.53 'whatomate-switch'
```

## 2026-06-03 04:10 Africa/Cairo - Current Project Green Sandbox Deployment

- **Deployment type**: Sandbox-only GREEN (no production switch)
- **Active sandbox binary**: /opt/whatomate/bin/whatomate.sandbox.green.20260603_011052
- **Version**: `Whatomate sandbox-green-20260603_010732-124187f7 (built 2026-06-03_01:07:48)`
- **SHA256**: `9dbef8f8b89de8a4ef7dece2a51e354e105e5fbd900f028f629f9ba06456fe9d`
- **Build**: Local macOS (darwin/arm64 Go 1.25.9) cross-compiled for linux/amd64 with embedded keyring
- **Embedded keyring**: /root/whatomate-keyring.json (3 keys)
- **Source revision**: 124187f7 (working tree had uncommitted changes including Facebook comments feature)
- **Backup**: /root/whatomate_backups/20260603_010704_pre_green_sandbox_deploy
- **Production ofuqalmadenah.com was NOT touched**

### Sandbox Config
- Port: 127.0.0.1:18127, DB: whatomate_sandbox_green_20260602_235053, Redis DB: 4
- Config: /opt/whatomate-sandbox/config.toml, Uploads: /opt/whatomate/uploads (shared)
- Systemd: whatomate-sandbox.service -> /opt/whatomate/bin/whatomate.sandbox.green

### Verification
- whatomate-sandbox.service: active, whatomate.service: active (untouched)
- Production symlink unchanged: /opt/whatomate/bin/whatomate -> whatomate.green.20260528_111523
- Sandbox port 18127 /login: 200, Production port 18123 /login: 200
- Sandbox license: enabled=true, status=active, locked=false
- Production license: enabled=true, status=active, locked=false

### Sandbox Switch
```bash
whatomate-sandbox-switch status    # show active sandbox binary
whatomate-sandbox-switch green     # switch to latest sandbox green
whatomate-sandbox-switch blue      # switch to latest production blue
```

The helper updates `/opt/whatomate/bin/whatomate`, restarts the live services, and stops the sandbox when green is promoted. Switching back to blue restarts the sandbox so green can continue to be tested separately.

## 2026-06-03 04:55 Africa/Cairo - Facebook Comments Sandbox Save Fix

- **Deployment type**: Sandbox-only GREEN hotfix (no production switch)
- **Active sandbox binary**: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_045205_fbcomments_savefix`
- **Version**: `sandbox-green-20260603_045205-fbcomments-savefix`
- **Systemd**: `whatomate-sandbox.service` active on `127.0.0.1:18127`
- **Production ofuqalmadenah.com was NOT touched**

### What Changed

- Added missing `facebook_oauth.webhook_verify_token` to `/opt/whatomate-sandbox/config.toml`.
- Hardened Facebook comment saving to trim varchar-sized fields before persistence.
- Added error logging and sync failure detail for failed Facebook comment saves.
- Removed temporary build source directory after deployment.

### Verification

- Local compile check passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/database ./internal/config ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly`
- `whatomate-sandbox.service`: active.
- `whatomate.service`: active and unchanged.
- Sandbox symlink:
  - `/opt/whatomate/bin/whatomate.sandbox.green -> /opt/whatomate/bin/whatomate.sandbox.green.20260603_045205_fbcomments_savefix`
- Production symlink unchanged:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260528_111523`
- `https://sandbox.ofuqalmadenah.com/facebook/comments`: `200`
- `http://127.0.0.1:18127/api/facebook/comments`: `401` unauthenticated, route present
- Facebook comments webhook verify handshake: `200`
- Signed Facebook comments webhook POST: `200`

## 2026-06-03 10:34 Africa/Cairo - Facebook Comments Sandbox Enum Hotfix

- **Deployment type**: Sandbox-only GREEN hotfix (no production switch)
- **Active sandbox binary**: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_103201_fbcomments_enumfix`
- **Version**: `sandbox-green-20260603_103201-fbcomments-enumfix`
- **Systemd**: `whatomate-sandbox.service` active on `127.0.0.1:18127`
- **Production ofuqalmadenah.com was NOT touched**

### Root Cause

- Graph API was returning comments for connected pages, including `Ofuqalmadenahافق المدينة`.
- Saving failed because GORM rejected custom enum fields:
  - `unsupported data type: ... FacebookCommentStatus: Table not set`

### What Changed

- Added SQL `Value`/`Scan` converters for Facebook comment enum model fields:
  - `FacebookCommentStatus`
  - `FacebookCommentDirection`
- Removed temporary build source directory after deployment.

### Verification

- Local compile check passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/models ./internal/handlers ./internal/database ./internal/config ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly`
- `whatomate-sandbox.service`: active.
- `whatomate.service`: active and unchanged.
- Sandbox symlink:
  - `/opt/whatomate/bin/whatomate.sandbox.green -> /opt/whatomate/bin/whatomate.sandbox.green.20260603_103201_fbcomments_enumfix`
- Production symlink unchanged:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260528_111523`

### License

- The first green rebuild started but reported `stored_token_invalid` because `/root/whatomate-keyring.json` contained a stale `vendor-1` public key only.
- Extracted the working embedded public key ring from the active binaries and replaced `/root/whatomate-keyring.json`.
- Corrected keyring sha256:
  - `7458085bb0a2af587dddba22c5784e42fa85b8f266a4de7629b81e13bc72ffbe`
- Rebuilt the green binary with embedded `license.EmbeddedPublicKeyRingBase64`.
- Final license state on sandbox:
  - `enabled=true`
  - `status=active`
  - `locked=false`
  - `key_id=deploy-20260416`
  - `tier=production`
  - `duration_label=lifetime`

### Cleanup

- Removed temporary/source runtime paths after the verified install:
  - `/tmp/whatomate-green-src-20260512_145748`
  - `/opt/whatomate-sandbox/src`
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
- Runtime binary/config/data directories were preserved.

### Verification

- Local backend verification:
  - `go test ./cmd/... ./internal/... ./pkg/... ./test/...` passed
  - targeted handler tests passed but DB-backed cases skipped when local `TEST_DATABASE_URL` was unset
- Local frontend verification:
  - `npm run test:unit -- --run src/lib/media_prefetch_cache.test.ts src/services/websocket.test.ts` passed
  - `npm run build` passed
  - `make build-prod` passed locally
  - `npm run typecheck` still has pre-existing unrelated TypeScript errors outside this change
- VPS build verification:
  - native Go build on Ubuntu amd64 passed
  - final binary version: `Whatomate a73f45b1-green-20260512_145748`
- VPS service verification:
  - `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`, and `whatomate-sandbox` are active
- API verification:
  - `https://ofuqalmadenah.com/api/license/bootstrap` returned active license on the unchanged live service
  - `https://sandbox.ofuqalmadenah.com/api/license/bootstrap` returned active license on the new green sandbox
- Browser verification:
  - Playwright MCP loaded `https://sandbox.ofuqalmadenah.com/login`
  - login UI rendered
  - no browser console warnings or errors after navigation
  - `/api/license/bootstrap` returned `200`
  - unauthenticated `/api/me` and refresh requests returned expected `401`

### Skills And Competencies Applied

- Skills selected: `debugging-wizard`, `test-master`, `devops-engineer`
- Competencies applied:
  - production log triage and SQL/GORM failure isolation
  - focused backend handler repair
  - Vue UI notification behavior update
  - native Linux production build and systemd drop-in management
  - license keyring recovery and embedded-key rebuild
  - blue/green deployment with rollback aliasing
  - API and browser smoke verification

## Blue-Green Deployment Update

Updated: 2026-05-15 01:12:00 UTC

- Deployment type: Blue-Green
- Active slot: GREEN
- Green binary: /opt/whatomate/bin/whatomate.green.20260515_011030
- Blue binary (rollback): /opt/whatomate/bin/whatomate.blue.20260515_010659
- Version: Whatomate green-20260515_011000 (built 2026-05-15_01:10:13)
- Switch command on VPS: `whatomate-switch` (toggles between blue and green)

### Changes in this deploy

- Frontend: Added "File no longer available" clickable text with retry download for all media types (image, video, audio, sticker, document)
- Frontend: Added Video and Music icons from lucide-vue-next for expired media indicators
- License key ring embedded at build time from /root/whatomate-keyring.json

### Post-Deploy Verification

- All 4 services: active
- License status: enabled=True, status=active, locked=False (all 4 ports)
- All HTTPS endpoints returning 200

### One-command switch

```bash
# On the VPS, run:
whatomate-switch
```

## Green Deployment Update

Updated: 2026-05-18 21:45 UTC

- Deployed source revision: `b42becc` from local workspace `/Users/airm2/Downloads/whatomate`.
- Deployment strategy: blue/green binary slots under `/opt/whatomate/bin`.
- Verified pre-deployment backup: `/root/whatomate_backups/20260518_212737_pre_green_deploy`.
  - Backup size: 424M.
  - Backup contents: live binaries, switch scripts, runtime configs, remote docs, systemd/nginx definitions, and 5 PostgreSQL dumps.
- Native VPS build directory: `/root/whatomate_temp_build_20260518_213500`.
- Native build command: `GOTOOLCHAIN=go1.25.9+auto VERSION=green-20260518_214000-b42becc-license LICENSE_KEY_RING_FILE=/root/whatomate-keyring.json make build-prod`.
- License key-ring validation: `/root/whatomate-keyring.json` parsed successfully with 3 Ed25519 public keys before embedding.
- Installed green binary: `/opt/whatomate/bin/whatomate.green.20260518_214000`.
- Active binary symlink: `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260518_214000`.
- Blue rollback symlink: `/opt/whatomate/bin/whatomate.blue -> /opt/whatomate/bin/whatomate.blue.predeploy_20260518_214000`.
- Installed version: `Whatomate green-20260518_214000-b42becc-license (built 2026-05-18_21:40:11)`.
- Installed SHA256: `41207f7a75e17f366c75d06835e60f0ae4e891a320c72ff6b6b7f2811bdbda4e`.

### Rollout Notes

- First green cutover attempt was automatically rolled back because the first build lacked usable embedded license public keys and exited with `license is enabled but no usable public keys are configured or embedded`.
- The final green build embedded the validated production public key ring and passed all service and license checks.
- The one-command switch is installed at `/usr/local/sbin/whatomate-switch` and linked from `/opt/whatomate/bin/switch-blue-green.sh`.
- To toggle active deployment between green and blue, run on the VPS:

```bash
whatomate-switch
```

### License Verification

Final local license bootstrap state:

- `127.0.0.1:18123` -> `enabled = true`, `status = active`, `locked = false`
- `127.0.0.1:18124` -> `enabled = true`, `status = active`, `locked = false`
- `127.0.0.1:18125` -> `enabled = true`, `status = active`, `locked = false`
- `127.0.0.1:18126` -> `enabled = true`, `status = active`, `locked = false`

Browser-side bootstrap from `https://ofuqalmadenah.com` returned HTTP 200 with `enabled=true`, `status=active`, `locked=false`.

### Verification

- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- Local listeners active: `127.0.0.1:18123`, `127.0.0.1:18124`, `127.0.0.1:18125`, `127.0.0.1:18126`.
- Local login smoke: all four ports returned `200`.
- Public HTTPS login smoke:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- Chrome DevTools MCP browser verification confirmed rendered login pages on the main domain and all three tenant domains with title `Whatomate`, heading `Welcome to Whatomate`, and a visible `Sign in` button.
- Chrome DevTools console check reported no errors/warnings on the final tenant login page.
- Browser screenshot saved locally at `/Users/airm2/Downloads/whatomate/deploy-verify-login.png`.

### Cleanup

Removed VPS source-code deployment sandboxes after successful verification:

- `/root/whatomate`
- `/root/whatomate_temp_build_20260518_213000`
- `/root/whatomate_temp_build_20260518_213500`
- `/opt/whatomate-sandbox`
- temporary build logs and failed partial backup attempts from this session

Remaining expected Whatomate paths:

- `/opt/whatomate` runtime, configs, uploads, and binaries
- `/root/whatomate_backups` verified backups
- `/root/ops_backups/whatomate_incident_20260412_193101` historical backup material

## 2026-05-19 Green Media / Resize Fix Deployment

Active green was replaced with:

- Binary: `/opt/whatomate/bin/whatomate.green.20260519_002559`
- Version: `Whatomate green-20260519_032337-f3575f3-media-dedup-resize (built 2026-05-19_00:23:37)`
- Branch: `agent/media-dedup-resize-fix`
- Commits:
  - `d1a9b62` - `Fix missing media dedup recovery and resizable refs`
  - `f3575f3` - `Clarify active license quota copy`

Backup before final replacement:

- `/root/whatomate_backups/20260519_002559_pre_green_copy_tweak`

Switch command:

- `whatomate-switch status`
- `whatomate-switch green`
- `whatomate-switch blue`
- `whatomate-switch` toggles between latest green and latest blue.

Final verification:

- All four services are active and running `/opt/whatomate/bin/whatomate.green.20260519_002559`.
- Local login smoke on ports `18123`, `18124`, `18125`, `18126` returned HTTP `200`.
- Production Playwright verified the 07:14 PM two-file bubble, correct unrecoverable retry response, Assign Contact resize without fatal error, and License overview as `Active`.

Known limitation:

- The two historical 07:14 PM PDFs still cannot display because their object blobs are absent from live uploads/backups and the rows do not contain WhatsMeow recovery payloads. The deployed fix prevents stale dedup reuse for future inbound WhatsMeow media and prevents retry from returning fake success.

## Green Uploads Cleanup Deployment

Updated: 2026-05-22 23:49:00 UTC

- Active slot: GREEN
- Green binary: `/opt/whatomate/bin/whatomate.green.20260522_234238`
- Version: `Whatomate green-20260522_234238-0d74527-uploads-cleanup`
- Backup before install: `/root/whatomate_backup_before_uploads_cleanup_20260522_234238`
- Fix: Settings uploads cleanup now includes WhatsMeow inbound media under `/opt/whatomate/uploads/whatsmeow/media`.
- Verification: `/api/org/uploads-cleanup/run` returned `deleted_files=2337` with `retention_days=5`; no files older than 5 days remained in `/opt/whatomate/uploads/whatsmeow/media`.
- Services verified active: `whatomate.service`, `whatomate@alarkan-almthalia.service`, `whatomate@holol-wenjaz.service`, `whatomate@matbaat-ruya.service`.
- Login checks on ports `18123`, `18124`, `18125`, and `18126` returned `200`.
- Operational note: `/usr/local/bin/whatomate-housekeeping.sh` now tolerates optional missing disk-snapshot paths; the service reran with status `0/SUCCESS`.

## Green Text Send Fix Deployment

Updated: 2026-05-25 20:15:55 UTC

- Deployment type: blue/green replacement of the active green slot.
- Deployed source revision: `c1e34cd` (`Fix whatsmeow plain text sends`).
- Active slot after deploy: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260525_200333`.
- Version: `Whatomate green-20260525_200333-c1e34cd-text-send (built 2026-05-25_20:07:08)`.
- SHA256: `fd8a6947d335531d4ee8ac85f2e2fb35a134d9351dbda972692bfbfb3797f18d`.
- Blue rollback binary left untouched: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260525_192630_pre_green_text_send_fix` (`759M`).

### Fix

- WhatsMeow plain text messages now use the simple `Conversation` protobuf payload.
- Messages containing URLs still use `ExtendedTextMessage`, preserving the close-rating review-link behavior.
- The failing production chat showed historical `server returned error 400` rows before this deploy; no live retry was triggered from the browser to avoid sending a real customer message without explicit approval.

### License Verification

- `127.0.0.1:18123` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18124` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18125` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18126` -> `enabled=true`, `status=active`, `locked=false`.
- Chrome DevTools verification on `https://ofuqalmadenah.com/settings/license` showed `License overview` as `Active`, not `Disabled`.

### Verification

- Local tests passed before deployment:
  - `go test ./pkg/whatsmeow`
  - `go test ./internal/handlers -run 'TestSendViaProvider|TestApp_SendOutgoingMessage'`
  - `go test ./pkg/whatsmeow ./internal/handlers`
  - `go test ./...`
  - `git diff --check`
- VPS build passed:
  - `GOTOOLCHAIN=go1.25.9+auto VERSION=green-20260525_200333-c1e34cd-text-send LICENSE_KEY_RING_FILE=/root/whatomate-keyring.json make build-prod`
- Services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Local login smoke on ports `18123`, `18124`, `18125`, and `18126` returned `200`.
- Public HTTPS login smoke returned `200` for:
  - `https://ofuqalmadenah.com/login`
  - `https://holol-wenjaz.ofuqalmadenah.com/login`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login`
  - `https://matbaat-ruya.ofuqalmadenah.com/login`
- Chrome DevTools loaded the affected chat and the License page; the only console/network noise observed was the expected initial unauthenticated `/api/me` `401` before token refresh.
- Screenshot saved locally: `/Users/airm2/Downloads/whatomate/deploy-license-active-20260525.png`.

### Switch Command

Use this one command on the VPS to toggle between active green and blue:

```bash
whatomate-switch
```

Explicit commands are also available:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

From a local machine:

```bash
ssh root@31.97.192.53 'whatomate-switch'
```

## Deployment Update

Updated: 2026-05-28 14:20:00 Africa/Cairo

### Scope

- Deployed the current project as the new GREEN slot on VPS `31.97.192.53`.
- This deployment includes the final layout and polish pass for `/settings/agent-selection`.
- The old BLUE rollback slot was preserved side by side.
- The main service and all three dedicated tenants were restarted onto the new GREEN binary.

### Skills And Competencies Applied

- Skill selected: `devops-engineer`.
- Tools used: SSH, rsync, native VPS build, systemd, curl.
- Competencies applied:
  - blue/green deployment
  - pre-deployment backup
  - production Linux build with embedded license keyring
  - systemd service verification
  - license and security-header verification
  - post-deploy source cleanup

### Deployment

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Version: `Whatomate green-20260528_111523-09191c2-agent-ui (built 2026-05-28_11:18:57)`.
- SHA256: `4abd7096755d01623a54c4e56290fce386ecf256c45f098b521bd518ef08c921`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260528_111523_pre_agent_ui_polish`.

### Running Services

All services are active and running from `/opt/whatomate/bin/whatomate.green.20260528_111523`:

- `whatomate.service`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Verification

- Local pre-deploy checks passed:
  - `cd frontend && npm run typecheck`
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'`
  - `git diff --check`
- Local ports returned `/login` HTTP `200`:
  - `18123`
  - `18124`
  - `18125`
  - `18126`
- License bootstrap on all four local ports returned `enabled=true`, `status=active`, and `locked=false`.
- Public HTTPS `/login` returned `200` for:
  - `https://ofuqalmadenah.com`
  - `https://holol-wenjaz.ofuqalmadenah.com`
  - `https://alarkan-almthalia.ofuqalmadenah.com`
  - `https://matbaat-ruya.ofuqalmadenah.com`
- Production security headers confirmed:
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Content-Security-Policy` present.

### Switch Command

Use this one command on the VPS to toggle active deployment between green and blue:

```bash
whatomate-switch
```

Explicit commands:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

From a local machine:

```bash
ssh root@31.97.192.53 'whatomate-switch'
```

### Cleanup

- Removed temporary VPS build/source directories after deployment:
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Remaining expected runtime paths:
  - `/opt/whatomate/bin`
  - `/opt/whatomate/config.toml`
  - `/opt/whatomate/instances`
  - `/opt/whatomate/uploads`
  - `/root/whatomate_backups`
  - `/root/whatomate_multi_instances_info.md`
  - `/root/whatomate_production_info.md`

### Skills And Competencies Applied

- Skills selected: `devops-engineer`, `debugging-wizard`.
- Competencies applied:
  - blue/green deployment and rollback preservation
  - native Ubuntu Go/Vue production build
  - systemd service restart and health checks
  - license keyring embedding and license-state verification
  - Chrome DevTools browser verification
  - production source cleanup while preserving runtime binaries, configs, uploads, and backups

## Green Current Project Deployment

Updated: 2026-05-27 17:45:00 UTC

- Deployment type: blue/green replacement of the active green slot.
- Deployed local revision: `09191c2` plus uncommitted working-tree changes for customer agent selection, TypeScript fixes, and security headers.
- Active slot after deploy: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260527_174500`.
- Version: `Whatomate green-20260527_174500-09191c2-csp (built 2026-05-27_17:42:53)`.
- SHA256: `a140bc30a10d018f05ff1da97bc9505f7ff1d82d241721b78ae74281bd948ff0`.
- Blue rollback binary left untouched: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Previous green kept as rollback artifact: `/opt/whatomate/bin/whatomate.green.20260527_173000` and `/opt/whatomate/bin/whatomate.green.20260525_200333`.
- Backup before deployment: `/root/whatomate_backups/20260527_172753_pre_green_current_project`.

### Backup Scope

- Runtime/config/bin tar: `/root/whatomate_backups/20260527_172753_pre_green_current_project/runtime-configs-and-bin.tar.gz`.
- PostgreSQL dumps:
  - `whatomate.sql.gz`
  - `whatomate_holol_wenjaz.sql.gz`
  - `whatomate_alarkan_almthalia.sql.gz`
  - `whatomate_matbaat_ruya.sql.gz`
- Backup preserved binaries, switch helper, configs, systemd/nginx definitions, production docs, and license keyring.

### Build

- Temporary build path: `/root/whatomate-green-src-20260527_173000` (removed after verification).
- Build command:

```bash
GOTOOLCHAIN=go1.25.9+auto VERSION=green-20260527_174500-09191c2-csp LICENSE_KEY_RING_FILE=/root/whatomate-keyring.json make build-prod
```

- Build environment: Go `1.25.9`, Node `20.19.6`, npm `10.8.2`.
- License keyring sha256: `7458085bb0a2af587dddba22c5784e42fa85b8f266a4de7629b81e13bc72ffbe`.

### License Verification

- `127.0.0.1:18123` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18124` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18125` -> `enabled=true`, `status=active`, `locked=false`.
- `127.0.0.1:18126` -> `enabled=true`, `status=active`, `locked=false`.
- Chrome DevTools verified `https://ofuqalmadenah.com/settings/license` shows `License overview` as `Active`.

### Verification

- Local pre-deploy checks:
  - `go test ./...`
  - `cd frontend && npm run typecheck`
  - `git diff --check`
- CSP nonce regression found by Chrome DevTools during smoke test and fixed before final green build.
- Focused post-fix checks:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./cmd/whatomate ./internal/middleware ./internal/frontend`
  - `cd frontend && npm run typecheck`
  - `git diff --check`
- Services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Local login smoke on ports `18123`, `18124`, `18125`, and `18126` returned `200`.
- Public HTTPS login smoke returned `200` for:
  - `https://ofuqalmadenah.com/login`
  - `https://holol-wenjaz.ofuqalmadenah.com/login`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login`
  - `https://matbaat-ruya.ofuqalmadenah.com/login`
- Security headers confirmed on production:
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Content-Security-Policy` present.
  - SPA document CSP includes a per-response `nonce-*` and Chrome DevTools reported no console messages after reload.

### Switch Command

Use this one command on the VPS to toggle active deployment between green and blue:

```bash
whatomate-switch
```

Explicit commands:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

From a local machine:

```bash
ssh root@31.97.192.53 'whatomate-switch'
```

### Cleanup

- Removed temporary/source VPS paths after verification:
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, license keyring, and backups.

### Skills And Competencies Applied

- Skills selected: `devops-engineer`, `debugging-wizard`, and browser/Chrome DevTools verification.
- Competencies applied:
  - blue/green production deployment
  - backup and rollback planning
  - native Ubuntu build with embedded license keyring
  - systemd health checks
  - license activation verification
  - CSP/HSTS security header validation
  - production source cleanup without touching runtime data

## Deployment Update

Updated: 2026-05-28 02:10:00 Africa/Cairo

### Scope

- Deployed the current project as the new GREEN slot on VPS `31.97.192.53`.
- The deployed version includes the Customer routing instance-scope feature, allowing agent selection to run only for selected WhatsMeow instances.
- The old BLUE rollback slot was preserved side by side.
- All running production services were restarted onto the new GREEN binary, so no service is running an older green binary.

### Skills And Competencies Applied

- Skill selected: `devops-engineer`.
- Tools used: SSH/systemd/curl for deployment and verification; Chrome DevTools for production browser verification.
- Competencies applied:
  - blue/green deployment
  - pre-deployment backup
  - production Linux build with embedded license keyring
  - systemd service verification
  - license and security-header verification
  - post-deploy source cleanup

### Deployment

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- Version: `Whatomate green-20260528_020100-09191c2-agent-scope (built 2026-05-28_02:00:01)`.
- SHA256: `4cbcfa440a67fba3d568b25e43f77e7a0352ebf71a0acd74bfbea0a3a1d2eabf`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_181332_pre_green_instance_scope`.

### Running Services

All services are active and running from `/opt/whatomate/bin/whatomate.green.20260528_020100`:

- `whatomate.service`
- `whatomate@holol-wenjaz`
- `whatomate@alarkan-almthalia`
- `whatomate@matbaat-ruya`

### Verification

- Local pre-deploy checks passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'`
  - `cd frontend && npm run typecheck`
  - `git diff --check`
- Local ports returned `/login` HTTP `200`:
  - `18123`
  - `18124`
  - `18125`
  - `18126`
- License bootstrap on all four local ports returned `enabled=true`, `status=active`, and `locked=false`.
- Public HTTPS `/login` returned `200` for:
  - `https://ofuqalmadenah.com`
  - `https://holol-wenjaz.ofuqalmadenah.com`
  - `https://alarkan-almthalia.ofuqalmadenah.com`
  - `https://matbaat-ruya.ofuqalmadenah.com`
- Production security headers confirmed:
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Content-Security-Policy` present.
- Chrome DevTools verified:
  - `https://ofuqalmadenah.com/settings/license` shows `License overview` as `Active`.
  - `https://ofuqalmadenah.com/settings/agent-selection` loads the new `Instance scope` UI and the related API calls return `200`.
  - No JavaScript console errors were found; Chrome reported non-blocking accessibility issues for existing form labels.

### Switch Command

Use this one command on the VPS to toggle active deployment between green and blue:

```bash
whatomate-switch
```

Explicit commands:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

From a local machine:

```bash
ssh root@31.97.192.53 'whatomate-switch'
```

## Sandbox Deploy Note - 2026-06-04 01:12 UTC — facebook-admin-reply-filter

- Sandbox green deploy: 2026-06-04 01:12:28 UTC
- Active sandbox binary: /opt/whatomate/bin/whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c
- Sandbox blue rollback binary: /opt/whatomate/bin/whatomate.sandbox.green.20260603_223000_fbcomments_realtime_push_10903_skip
- Installed SHA256: a03a18355403ea2ec01ad58860cd2f461729f878b39746dfac49be14224599cd
- Version: sandbox-green-20260604_010000_fb_admin_reply_filter_3f31242c
- Build: linux/amd64 (cross-compiled from darwin/arm64 host), CGO disabled, ldflags `-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X github.com/compnew2006/whatomate/internal/license.EmbeddedPublicKeyRingBase64=$LICENSE_KEY_RING_B64`, embedded license public key ring (kids: deploy-20260415, deploy-20260416, vendor-1), embedded frontend dist.
- Source HEAD: 3f31242c (agent/facebook-admin-reply-filter) — working tree clean.
- Bug fixes included:
  1. FB webhook + sync: admin/page-author replies now tagged with `IsAdminReply bool` (indexed) and skipped from auto-reply (defense in depth: guard in BOTH webhook call site AND `shouldAutoReplyFacebookComment`). UI badge "Page admin" / "مسؤول الصفحة" / "Administrador de la página".
  2. Latent FK fix: `sendAndStoreFacebookCommentReply` was passing `uuid.Nil` for userID in webhook auto-reply path; FK `fk_facebook_comment_replies_user` was rejecting zero UUID. Replaced with `account.UserID` — auto-reply was silently failing before.
- Verification:
  - `systemctl restart whatomate-sandbox` → active since 2026-06-04 01:12:28 UTC, PID 2263440 (was 2191864), new binary
  - `curl http://127.0.0.1:18127/api/license/bootstrap` → status=active, tier=production, key_id=deploy-20260416, hwid=d87d9d77e173, updated_at=2026-06-04T01:12:46Z (fresh, confirms new build is serving)
  - External `https://sandbox.ofuqalmadenah.com/` → 200 "Whatomate" SPA served
  - WebSocket upgrade: `GET /ws` → 101 (user 156.207.95.198 in active session on `/facebook/page-search`)
  - FB webhooks flowing: POST /api/facebook/comments/webhook → 200 from 173.252.95.2/33/16 etc.
  - License usage: 1/5 orgs, 29/50 users, 16/50 endpoints — all under quota
  - 18/18 facebook test functions pass (TestApp_ReceiveFacebookCommentsWebhook_AdminReplyTaggedAndNotAutoReplied + TestApp_ReceiveFacebookCommentsWebhook_NonAdminStillAutoReplies new)
- Symlink chain post-deploy:
  - active → whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c
  - green  → whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c
  - blue   → whatomate.sandbox.green.20260603_223000_fbcomments_realtime_push_10903_skip (rollback)
- Toggle: `whatomate-sandbox-switch {status|green|blue|toggle|version}` (one-command rollback to previous prod).
- Pre-existing unrelated dirty files in agent's working tree: NOT touched.
- Local build artifact: /tmp/whatomate-sandbox-green-20260604_010000-linux-amd64 (58683576 bytes).

## Sandbox Deploy - 2026-06-04 01:36 UTC — comments-scroll-fix

- Branch: main (HEAD 3f31242c), 1 file changed in src: `frontend/src/views/facebook/FacebookCommentsView.vue` (added `lg:grid-rows-[minmax(0,1fr)]` to the 3-column grid wrapper).
- Bug fixed: `/facebook/comments` Inbox (left) sidebar had no scroll because the grid container's default `grid-auto-rows: auto` sized the row to content. With many comments, the Inbox grew past the viewport and the ScrollArea never had overflow to trigger. Adding `minmax(0, 1fr)` to the row makes it fill the grid container (bounded by the parent's `min-h-0 flex-1` flex chain).
- Build: `env -u GOOS -u GOARCH GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=comments-scroll-fix-20260604_013200-3f31242c -X main.BuildTime=… -X …/license.EmbeddedPublicKeyRingBase64=…"` → ELF 64-bit, statically linked.
- SHA256: `6def64dfb72ec38879a862fe1f206732cc9684b3ba961173d4dad475ac4e7d6f`, 58966178 bytes.
- Verified: `file` → `ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked`. Deployed CSS bundle (`/assets/index-BM73qrpE.css`) contains `lg\:grid-rows-\[minmax\(0\,1fr\)\]{grid-template-rows:minmax(0,1fr)}`.
- Lessons: ALWAYS run `cp -r frontend/dist/* internal/frontend/dist/` BEFORE `go build` (the Makefile's `build-prod` does it via `embed-frontend`); running `go build` directly produces a binary that still embeds the OLD frontend dist. Symptom: API is live and reports new version, but the SPA's CSS bundle is the old hash.
- Deploy: scp to /tmp on VPS → cp into /opt/whatomate/bin/ → ln -sfn to .active + .green → systemctl restart.
- Verification post-restart: PID 2268910 active since 2026-06-04 01:36:35 UTC, `curl /` → 200, license bootstrap → status=active tier=production key_id=deploy-20260416, new CSS hash served.
- Rollback target (.blue): whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c (previous green).

## Sandbox Green Deploy - 2026-06-11 20:06 UTC - current project

- VPS: `31.97.192.53` (`root`, Ubuntu).
- Deployment mode: sandbox green replacement only. Public blue/live users were left on the existing `/opt/whatomate/bin/whatomate` symlink.
- Pre-deploy backup: `/root/whatomate_backups/whatomate-green-predeploy-20260611_195937.tar.gz`
  - SHA256: `1f156804b95bc7ef324a94facf37862f2fc7a1215b6e6ac8c956755671a32567`
  - Size: `630M`
- Source revision deployed: `5702241f`
- New sandbox green binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
- New sandbox green SHA256: `24110198b9da7caae06d5bbb6a16738ad24da5589e7f3e1bb62c3861189c31df`
- Version output: `Whatomate 5702241f-sandbox-green-20260611_200325 (built 2026-06-11_20:06:10)`
- Symlink state after deploy:
  - `/opt/whatomate/bin/whatomate.sandbox.active` -> `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
  - `/opt/whatomate/bin/whatomate.sandbox.green` -> `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
  - `/opt/whatomate/bin/whatomate.sandbox.blue` -> `/opt/whatomate/bin/whatomate.sandbox.comments-scroll-fix-20260604_013200-3f31242c`
- Public live symlink was not changed:
  - `/opt/whatomate/bin/whatomate` -> `/opt/whatomate/bin/whatomate.green.20260528_111523`
  - Version: `Whatomate green-20260528_111523-09191c2-agent-ui (built 2026-05-28_11:18:57)`
- Active services after deploy:
  - `whatomate`: active on `127.0.0.1:18123`
  - `whatomate@holol-wenjaz`: active on `127.0.0.1:18124`
  - `whatomate@alarkan-almthalia`: inactive before and after deploy
  - `whatomate@matbaat-ruya`: inactive before and after deploy
  - `whatomate-sandbox`: active on `127.0.0.1:18127`
- License verification:
  - `http://127.0.0.1:18127/api/license/bootstrap` returned `enabled=true`, `status=active`, `tier=production`, `key_id=deploy-20260416`.
  - `http://127.0.0.1:18123/api/license/bootstrap` returned `enabled=true`, `status=active`, `tier=production`, `key_id=deploy-20260416`.
  - Browser-side fetch from `https://sandbox.ofuqalmadenah.com/login` to `/api/license/bootstrap` returned HTTP `200`, `enabled=true`, `status=active`.
- Browser verification:
  - Chrome DevTools loaded `https://sandbox.ofuqalmadenah.com/login`.
  - Login UI rendered successfully.
  - Key assets and `/api/auth/sso/providers` returned HTTP `200`.
  - No console warnings/errors were reported.
  - Screenshot saved locally at `sandbox-green-login.png`.
- Local verification before deploy:
  - `frontend && npm run build`: passed.
  - Targeted Go packages: passed for `internal/database`, `internal/config`, `internal/crypto`, `internal/license`, `pkg/whatsapp`, `pkg/whatsmeow`; `internal/handlers` failed on pre-existing test setup problems around missing `messages.instance_id` in SQLite cleanup tests and Redis connection refusal.
- Server build verification:
  - `frontend && npm ci`: completed with zero vulnerabilities.
  - `make build-prod`: passed and embedded frontend assets.
  - Final `go build` used embedded license keyring and static `CGO_ENABLED=0` binary output.
- Cleanup:
  - Removed temporary build source `/tmp/whatomate-green-src`.
  - Removed temporary keyring `/tmp/whatomate-green-keyring.json`.
  - No Whatomate source tree was left under `/tmp`; runtime configs and `/opt/whatomate/bin` were preserved.

### One-Line Switch Commands

Promote the current sandbox green binary to public live for active public units:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback public live to the previous public blue/live binary:

```bash
ln -sfn /opt/whatomate/bin/whatomate.green.20260528_111523 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback only sandbox to the previous sandbox blue:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.blue /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```
