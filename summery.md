# Whatomate Session Summary - 2026-05-19

## Task

Fix and deploy the green Whatomate version for:

- Missing WhatsMeow media on the two 07:14 PM chat bubbles in `https://ofuqalmadenah.com/chat/82edad6a-708a-4ce9-af2b-6c8f72b27cac`.
- Retry download falsely reporting success when the underlying media blob is absent.
- Future inbound WhatsMeow dedup reusing stale `media_asset` rows whose object blobs are missing.
- Fatal Assign Contact dialog resize crash: `t.value.getBoundingClientRect is not a function`.
- Confirm license overview is active and green is the running side of the blue/green deployment.

## Approach And Key Decisions

- Confirmed production green was active before debugging.
- Reproduced the 07:14 PM media issue with browser automation and API checks.
- Verified the two current PDFs cannot be reconstructed by the app because their object files are missing and the message rows do not contain WhatsMeow recovery metadata.
- Fixed the code path so future inbound WhatsMeow media dedup only reuses an existing media asset when the blob exists; otherwise it downloads and stores the current inbound payload again.
- Fixed retry behavior so object-backed media must verify the blob exists before returning success.
- Fixed the Vue resizable composable to accept Reka/Vue component refs by resolving `$el` before calling DOM APIs.
- Clarified active-license quota copy so the License page no longer says licensing is disabled when the deployment is active.

## Files Modified

- `pkg/whatsmeow/media_service.go`
- `pkg/whatsmeow/media_service_test.go`
- `internal/handlers/media.go`
- `internal/handlers/media_stream_test.go`
- `frontend/src/lib/useResizable.ts`
- `frontend/src/lib/useResizable.test.ts`
- `frontend/src/components/ui/dialog/DialogContent.vue`
- `frontend/src/i18n/locales/en.json`
- `docs/whatomate_multi_instances_info.md`
- `summary.md`
- `summery.md`

## Deployment

- Branch: `agent/media-dedup-resize-fix`
- Commits:
  - `d1a9b62` - `Fix missing media dedup recovery and resizable refs`
  - `f3575f3` - `Clarify active license quota copy`
- Active VPS green binary:
  - `/opt/whatomate/bin/whatomate.green.20260519_002559`
  - `Whatomate green-20260519_032337-f3575f3-media-dedup-resize (built 2026-05-19_00:23:37)`
- Backup created before final green replacement:
  - `/root/whatomate_backups/20260519_002559_pre_green_copy_tweak`
- Switch command installed/updated:
  - `whatomate-switch status`
  - `whatomate-switch green`
  - `whatomate-switch blue`
  - `whatomate-switch` toggles between latest green and latest blue.

## Verification

- `go test ./pkg/whatsmeow ./internal/handlers` passed.
- `cd frontend && npx vitest run src/lib/useResizable.test.ts` passed.
- `cd frontend && npx eslint src/lib/useResizable.ts src/lib/useResizable.test.ts src/components/ui/dialog/DialogContent.vue` passed.
- `cd frontend && npm run typecheck` still fails on pre-existing project-wide TypeScript errors outside this change, including readonly test fixture tags, existing `body` access typing in `ChatView.vue`, non-exported store types, and deep toast typing.
- `git diff --check` passed before deployment.
- `npm run build` passed with only the existing Vite chunk-size warning.
- Production systemd services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz.service`
  - `whatomate@alarkan-almthalia.service`
  - `whatomate@matbaat-ruya.service`
- Production login smoke:
  - `18123`, `18124`, `18125`, `18126` all returned HTTP `200`.
- Production Playwright verification:
  - 07:14 PM two-file bubble is visible.
  - `GET /api/media/3af5f0d6-af16-4689-8711-6c40dde6c6f7` returns `404`.
  - `POST /api/media/3af5f0d6-af16-4689-8711-6c40dde6c6f7/retry-download` returns `404` with `No recovery information available for this media`.
  - `GET /api/media/d1acbd65-d448-4d20-a37c-5d8d682d6dad` returns `404`.
  - `POST /api/media/d1acbd65-d448-4d20-a37c-5d8d682d6dad/retry-download` returns `404` with `No recovery information available for this media`.
  - Assign Contact dialog opened on an active chat and resized without fatal UI error or `getBoundingClientRect` page error.
  - License page shows `License overview` as `Active`; no Disabled/licensing-disabled copy remains.

## Artifacts

- `playwright-green-chat-media-after-fix.png`
- `playwright-green-assign-resize-after-fix.png`
- `playwright-green-license-after-fix.png`
- Earlier investigation screenshots:
  - `internal-browser-chat-media-investigation.png`
  - `playwright-chat-media-missing.png`
  - `playwright-chat-media-retry-clicks.png`

## Known Limitations

The two already-broken 07:14 PM PDFs still cannot display because the referenced blobs are absent from `/opt/whatomate/uploads` and backups checked during the investigation, and the affected rows have no stored WhatsMeow recovery payload. The deployed fix prevents this stale-dedup path from recurring for future inbound WhatsMeow media and stops retry from returning fake success for unrecoverable historical media.

# 2026-05-25 - Green Text Send Fix Deployment

## Task

Deploy the current project to the VPS as the new green slot, preserve the blue rollback slot, fix the WhatsMeow text-send `400` failure seen on chat `8b04fdf4-3f6c-4226-a003-c0ade8c7b75d`, verify that the license overview is active, remove temporary/source codebases from the VPS after deployment, update the deployment documentation, and keep a one-command blue/green switch.

## Skills and MCPs Applied

- Skills: `devops-engineer`, `debugging-wizard`.
- MCPs/tools: Chrome DevTools for production UI verification, shell/SSH for build and systemd verification.

## Deployment

- Deployed source revision: `c1e34cd` (`Fix whatsmeow plain text sends`).
- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260525_200333`.
- Version: `Whatomate green-20260525_200333-c1e34cd-text-send (built 2026-05-25_20:07:08)`.
- SHA256: `fd8a6947d335531d4ee8ac85f2e2fb35a134d9351dbda972692bfbfb3797f18d`.
- Blue rollback binary left untouched: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260525_192630_pre_green_text_send_fix` (`759M`).

## Fix

- Plain WhatsMeow text messages now build a simple `Conversation` payload.
- Text messages containing URLs still use `ExtendedTextMessage` to preserve close-rating review-link delivery.
- The affected chat page still shows historical failed rows from before the deploy; no production Retry/send was triggered from the browser because that would send a real customer message.

## Verification

- Local tests passed:
  - `go test ./pkg/whatsmeow`
  - `go test ./internal/handlers -run 'TestSendViaProvider|TestApp_SendOutgoingMessage'`
  - `go test ./pkg/whatsmeow ./internal/handlers`
  - `go test ./...`
  - `git diff --check`
- VPS production build passed with embedded license keyring:
  - `GOTOOLCHAIN=go1.25.9+auto VERSION=green-20260525_200333-c1e34cd-text-send LICENSE_KEY_RING_FILE=/root/whatomate-keyring.json make build-prod`
- Services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Local ports `18123`, `18124`, `18125`, and `18126` returned `/login` HTTP `200`.
- Public HTTPS login checks returned `200` for the main domain and all three tenant domains.
- License bootstrap on all four local ports returned `enabled=true`, `status=active`, and `locked=false`.
- Chrome DevTools confirmed `https://ofuqalmadenah.com/settings/license` displays `License overview` as `Active`.
- Chrome DevTools loaded the affected chat page; old `400` failures are visible as historical rows.
- Screenshot saved locally: `/Users/airm2/Downloads/whatomate/deploy-license-active-20260525.png`.

## Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.

## Cleanup

- Removed temporary/source VPS paths after the verified install:
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, and backups.

# 2026-05-27 - Green Current Project Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260527_174500`.
- Version: `Whatomate green-20260527_174500-09191c2-csp (built 2026-05-27_17:42:53)`.
- SHA256: `a140bc30a10d018f05ff1da97bc9505f7ff1d82d241721b78ae74281bd948ff0`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_172753_pre_green_current_project`.

## Verification

- `go test ./...` passed.
- `cd frontend && npm run typecheck` passed.
- `git diff --check` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./cmd/whatomate ./internal/middleware ./internal/frontend` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public login checks returned `200` for all production domains.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools initially found a CSP nonce regression; it was fixed and redeployed.
- Final Chrome DevTools reload showed no console messages and all listed network requests returned `200`.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

# 2026-06-03 - Sandbox Green Facebook OAuth Deployment

## Result

- Deployed the current project as an isolated green sandbox at `https://sandbox.ofuqalmadenah.com`.
- Did not switch, restart, or replace production `https://ofuqalmadenah.com` / `whatomate.service`.
- Sandbox service: `whatomate-sandbox.service`.
- Sandbox port: `127.0.0.1:18127`.
- Sandbox config: `/opt/whatomate-sandbox/config.toml`.
- Sandbox database: `whatomate_sandbox_green_20260602_235053`.
- Active sandbox binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260602_235053`.
- Version: `Whatomate sandbox-green-20260602_235053-current (built 2026-06-03_00:03:36)`.
- SHA256: `215b733c5fe2b2aadefaab315c612c3a0322a00035c986829bb54cba4654dfd2`.
- Backup before deployment: `/root/whatomate_backups/20260602_235053_pre_sandbox_green_deploy`.

## Verification

- Built on VPS with embedded `/root/whatomate-keyring.json`.
- `whatomate-sandbox.service` is active.
- `whatomate.service` remained active.
- License bootstrap on sandbox returned `enabled=true`, `status=active`, `locked=false`.
- `https://sandbox.ofuqalmadenah.com/login` loads through nginx and returns the Vue app.
- Chrome DevTools verified the login page loads, core assets return `200`, `/api/license/bootstrap` returns `200`, and there are no console errors.
- Unauthenticated `/api/facebook/oauth/init?action=connect` returns `401`, confirming the OAuth route is present and auth-protected.

## Sandbox Switch

```bash
whatomate-sandbox-switch green
```

Other sandbox-only commands:

```bash
whatomate-sandbox-switch status
whatomate-sandbox-switch blue
```

## Cleanup

- Removed temporary/source build paths from the VPS after installing the binary:
  - `/root/whatomate_sandbox_build_20260602_235053`
  - `/opt/whatomate-sandbox/src`
  - `/opt/whatomate-sandbox/.cache`
  - `/opt/whatomate-sandbox/.gopath`
- Preserved binaries, sandbox config, backups, and remote deployment docs.

# 2026-06-03 - Sandbox Facebook OAuth Save Fix

## Result

- Fixed the sandbox callback failure that showed `Failed to save Facebook account`.
- Root cause from `whatomate-sandbox.service` logs: `table name "facebook_oauth_states" specified more than once`.
- Code fix: `CallbackFacebookOAuth` now uses fresh GORM sessions when reading/deleting OAuth state and when saving the Facebook account.
- Deployed to sandbox only.
- Active sandbox binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_001fix`.
- Version: `Whatomate sandbox-green-20260603_001fix-fb-oauth-save-fix (built 2026-06-03_00:29:31)`.
- SHA256: `951aefd80614ca84e3dcf1a58ce5b71b3c5f7429cde04383833639118a25ebec`.

## Verification

- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run TestNonExistentFacebookOAuthCompileOnly` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/config` passed.
- Sandbox service active.
- Production service active and production symlink unchanged.
- Sandbox `/login` returned `200`.
- Sandbox license bootstrap returned `status=active`.
- Recent sandbox logs no longer show the previous OAuth save SQL error.

## Retry Note

- Start Facebook OAuth again from `/facebook/accounts`.
- Do not reuse the old Facebook callback URL because its `state` token was consumed by the failed callback.

# 2026-06-03 - Sandbox Facebook Accounts Display Fix

## Result

- Fixed the issue where Facebook OAuth showed a success toast but no accounts appeared in `/facebook/accounts`.
- Root cause: the API returns `{ accounts: [...] }`, while the frontend store was unwrapped the response as a direct array.
- Updated `frontend/src/stores/fbAccounts.ts` to use `unwrapListResponse(response, "accounts")`.
- Updated `frontend/src/views/facebook/FacebookAccountsView.vue` to display linked page names from `account.data.pages`.
- Verified the sandbox database already contained the OAuth account and 7 linked pages.
- Deployed to sandbox only.
- Active sandbox binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_002fb_list_fix`.
- Version: `Whatomate sandbox-green-20260603_002fb_list_fix (built 2026-06-03_00:35:39)`.
- SHA256: `673e8650b7d8a658dfa988ed328402d46d2f94eaa473fd5831ff5e35ae0bcfb3`.

## Verification

- `cd frontend && npm run typecheck` passed.
- Sandbox service active.
- Production service active and production symlink unchanged.
- Sandbox `/login` returned `200`.
- Chrome DevTools loaded the new sandbox frontend assets with no console errors.
- Browser verification of `/facebook/accounts` redirected to login in the Codex browser because that browser session was unauthenticated.

# 2026-06-03 - Current Project Green Sandbox Final Deployment

## Result

- Deployed the current project (including Facebook comments feature, agent selection updates) as new sandbox green binary.
- Active sandbox binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_011052`.
- Version: `Whatomate sandbox-green-20260603_010732-124187f7 (built 2026-06-03_01:07:48)`.
- SHA256: `9dbef8f8b89de8a4ef7dece2a51e354e105e5fbd900f028f629f9ba06456fe9d`.
- Build: Local macOS cross-compiled for linux/amd64 with embedded keyring.
- Old sandbox binaries preserved for rollback.
- Backup: `/root/whatomate_backups/20260603_010704_pre_green_sandbox_deploy`.
- **Production `ofuqalmadenah.com` was NOT touched.**

## Sandbox Switch

```bash
whatomate-sandbox-switch status    # show active sandbox binary
whatomate-sandbox-switch green     # switch to latest sandbox green
whatomate-sandbox-switch blue      # switch to latest production blue
```

## Verification

- `whatomate-sandbox.service`: active on port 18127.
- `whatomate.service` (production): active on port 18123, binary symlink unchanged.
- Sandbox `/login` returned `200`.
- Production `/login` returned `200`.
- License bootstrap on sandbox (18127): `enabled=true`, `status=active`, `locked=false`.
- License bootstrap on production (18123): `enabled=true`, `status=active`, `locked=false`.

## Cleanup

- All temporary/source build paths were already cleaned from VPS.
- Only runtime files remain: binaries, configs, uploads, backups, keyring.

# 2026-06-03 - Facebook Comments Sandbox Save Fix

## Result

- Investigated `/facebook/comments` on `https://sandbox.ofuqalmadenah.com`.
- Confirmed the route and comment tables exist on sandbox, and the linked OAuth account includes page `Ofuqalmadenahافق المدينة`.
- Found no stored comments initially and no recent incoming webhook events.
- Added `facebook_oauth.webhook_verify_token` to `/opt/whatomate-sandbox/config.toml` without printing the token.
- Fixed Facebook comment persistence hardening in `internal/handlers/fb_comments.go`:
  - trims varchar-sized Facebook comment fields to the database limit before save
  - logs the real database error when webhook or sync comment save fails
  - returns the real save error in manual sync failures
- Deployed new sandbox-only green binary:
  - `/opt/whatomate/bin/whatomate.sandbox.green.20260603_045205_fbcomments_savefix`
  - version `sandbox-green-20260603_045205-fbcomments-savefix`
- Removed VPS build source directory after deployment.
- Production `ofuqalmadenah.com` was NOT touched.

## Verification

- Local compile check passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/database ./internal/config ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly`
- `whatomate-sandbox.service`: active.
- `whatomate.service`: active and unchanged.
- Sandbox binary symlink:
  - `/opt/whatomate/bin/whatomate.sandbox.green -> /opt/whatomate/bin/whatomate.sandbox.green.20260603_045205_fbcomments_savefix`
- Production binary symlink unchanged:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260528_111523`
- `https://sandbox.ofuqalmadenah.com/facebook/comments` returned `200`.
- Unauthenticated comments API returned `401`, confirming the route exists behind auth.
- Facebook comments webhook verification returned `200` with the configured token.
- Signed Facebook comments webhook POST returned `200`.

# 2026-06-03 - Facebook Comments Sandbox Enum Hotfix

## Result

- Investigated why comments still did not appear after the save hardening deploy.
- Confirmed Graph API was returning comments, including comments for `Ofuqalmadenahافق المدينة`.
- Root cause found in sandbox logs: GORM rejected the custom enum fields with:
  - `unsupported data type: ... FacebookCommentStatus: Table not set`
- Fixed `internal/models/fb_comment.go` by adding SQL `Value`/`Scan` converters for:
  - `FacebookCommentStatus`
  - `FacebookCommentDirection`
- Deployed new sandbox-only green binary:
  - `/opt/whatomate/bin/whatomate.sandbox.green.20260603_103201_fbcomments_enumfix`
  - version `sandbox-green-20260603_103201-fbcomments-enumfix`
- Removed VPS build source directory after deployment.
- Production `ofuqalmadenah.com` was NOT touched.

## Verification

- Local compile check passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/models ./internal/handlers ./internal/database ./internal/config ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly`
- `whatomate-sandbox.service`: active.
- `whatomate.service`: active and unchanged.
- Sandbox binary symlink:
  - `/opt/whatomate/bin/whatomate.sandbox.green -> /opt/whatomate/bin/whatomate.sandbox.green.20260603_103201_fbcomments_enumfix`
- Production binary symlink unchanged:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green.20260528_111523`
- Unauthenticated comments API returned `401`, confirming the route exists behind auth.
- No new `Facebook synced comment` enum save errors appeared after the hotfix restart.

# 2026-06-03 - Facebook Comment Inbox Feature

## Result

- Added a new `/facebook/comments` feature in the local codebase.
- Backend models added:
  - `FacebookComment`
  - `FacebookCommentReply`
  - `FacebookCommentSettings`
- Backend APIs added:
  - `GET /api/facebook/comments`
  - `POST /api/facebook/comments/sync`
  - `GET /api/facebook/comments/settings`
  - `PUT /api/facebook/comments/settings`
  - `POST /api/facebook/comments/{id}/reply`
  - `PUT /api/facebook/comments/{id}/status`
  - `GET /api/facebook/comments/webhook`
  - `POST /api/facebook/comments/webhook`
- The feature can sync recent page posts/comments, display comments as an inbox, show source page/post, send public comment replies, send private replies, close/reopen comments, and run configured auto replies.
- Webhook receiving includes Meta `hub.challenge` verification and `X-Hub-Signature-256` validation when `facebook_oauth.app_secret` is configured.
- Frontend added `/facebook/comments` with inbox, comment detail/reply panel, settings dialog, and sync dialog.
- Navigation and i18n updated for English, Arabic, and Spanish.
- Not deployed to VPS in this step.

## Verification

- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/database ./internal/config ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly` passed.
- `cd frontend && npm run typecheck` passed.
- Targeted ESLint for the Facebook comments files passed.
- `git diff --check` passed.
- Full frontend lint still reports the existing unrelated `AppLayout.vue` parse issue.

# 2026-05-28 - Green Agent Selection UI Polish Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Version: `Whatomate green-20260528_111523-09191c2-agent-ui (built 2026-05-28_11:18:57)`.
- SHA256: `4abd7096755d01623a54c4e56290fce386ecf256c45f098b521bd518ef08c921`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260528_111523_pre_agent_ui_polish`.
- All four production services are running from the new green binary:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`

## Verification

- `cd frontend && npm run typecheck` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'` passed.
- `git diff --check` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Each service process executable resolves to `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for all production domains:
  - `https://ofuqalmadenah.com`
  - `https://holol-wenjaz.ofuqalmadenah.com`
  - `https://alarkan-almthalia.ofuqalmadenah.com`
  - `https://matbaat-ruya.ofuqalmadenah.com`
- HSTS and CSP headers are present on production responses.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

## Cleanup

- Removed VPS temporary/source paths after verification:
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, license keyring, and backups.

# 2026-06-03 - Sandbox Facebook Comment Author Scope Check

## Result

- Confirmed directly against Meta Graph API for a blank-author comment: the response contained `id` and `created_time` only, with no `from` object.
- Confirmed sandbox DB still had blank author data for most synced comments after sync: `1013/1020` comments lacked `from_name` and `from_id`.
- Added missing Facebook OAuth scope `pages_read_user_content` to newly generated OAuth links.
- Deployed sandbox-only binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_145935_fbcomments_scope_fix`.
- Production binary remained unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`.

## Required Follow-Up

- Existing page tokens will not gain the new scope automatically. Reconnect the Facebook account from `/facebook/accounts`, then run comments sync again.
- If Meta still omits `from`, the app/token likely also needs Meta App Review / Access Verification for the relevant user-profile access.

# 2026-06-03 - Sandbox Nginx Facebook Comments Sync Timeout

## Result

- Confirmed from `/var/log/nginx/sandbox.ofuqalmadenah.com.error.log` that `POST /api/facebook/comments/sync` hit `upstream timed out while reading response header`.
- Updated sandbox nginx vhost only: `/etc/nginx/sites-available/sandbox.ofuqalmadenah.com.conf`.
- Added `proxy_connect_timeout 30s`, `proxy_send_timeout 300s`, and `proxy_read_timeout 300s` under sandbox `location /`.
- Ran `nginx -t` successfully and reloaded nginx.
- Production binary remained unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`.

## Verification

- `nginx` is active.
- `whatomate-sandbox.service` is active.
- Sandbox unauthenticated `GET /api/facebook/comments` returns `401`.
- Sandbox binary remains `/opt/whatomate/bin/whatomate.sandbox.green.20260603_144506_fbcomments_sync_timeout_hotfix`.
- Production binary remains `/opt/whatomate/bin/whatomate.green.20260528_111523`.

# 2026-06-03 - Sandbox Facebook Comments Sync Timeout Hotfix

## Result

- Deployed sandbox-only binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_144506_fbcomments_sync_timeout_hotfix`.
- Production binary remained unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Root cause confirmed from sandbox nginx access log: `POST /api/facebook/comments/sync` returned `499`, meaning the browser/client closed the request before the server responded.
- Increased the frontend timeout for Facebook comments sync only from the default 30 seconds to 180 seconds.

## Verification

- `cd frontend && npm run typecheck` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/models ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly` passed.
- `git diff --check` passed.
- Sandbox service `whatomate-sandbox.service` is active.
- Sandbox unauthenticated `GET /api/facebook/comments` returns `401`.
- Production service `whatomate.service` is active and still points to the old production green binary.

# 2026-06-03 - Sandbox Facebook Comments Batch Actor Hotfix

## Result

- Deployed sandbox-only binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_141431_fbcomments_batch_actor_hotfix`.
- Production binary remained unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Replaced per-comment Facebook actor fallback calls with Graph batch calls of up to 50 comments per request.
- Actor lookup failures now do not block saving/syncing comments; missing author data remains `مستخدم فيسبوك` only when Meta does not return `from`.

## Verification

- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/models ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly` passed.
- `git diff --check` passed.
- Sandbox service `whatomate-sandbox.service` is active.
- Sandbox unauthenticated `GET /api/facebook/comments` returns `401`.
- Production service `whatomate.service` is active and still points to the old production green binary.

# 2026-06-03 - Sandbox Facebook Comments Author Retry and Pagination i18n

## Result

- Deployed sandbox-only binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260603_134253_fbcomments_author_retry_i18n`.
- Production binary remained unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Fixed missing top-level `common.previous` translations for Arabic, English, and Spanish.
- Added a Graph API fallback that fetches `from{id,name}` from the individual comment endpoint when the nested comments response returns an empty actor.
- Kept the comments list at 100 per page with previous/next pagination.

## Verification

- `jq empty frontend/src/i18n/locales/ar.json frontend/src/i18n/locales/en.json frontend/src/i18n/locales/es.json` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers ./internal/models ./cmd/whatomate -run TestNonExistentFacebookCommentsCompileOnly` passed.
- `cd frontend && npm run typecheck` passed.
- Sandbox service `whatomate-sandbox.service` is active.
- Sandbox unauthenticated `GET /api/facebook/comments` returns `401`.
- Production service `whatomate.service` is active and still points to the old production green binary.

# 2026-05-28 - Green Agent Scope Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- Version: `Whatomate green-20260528_020100-09191c2-agent-scope (built 2026-05-28_02:00:01)`.
- SHA256: `4cbcfa440a67fba3d568b25e43f77e7a0352ebf71a0acd74bfbea0a3a1d2eabf`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_181332_pre_green_instance_scope`.
- All four production services are running from the new green binary, replacing the older running green version.

## Verification

- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'` passed.
- `cd frontend && npm run typecheck` passed.
- `git diff --check` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- Each service process executable resolves to `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for all production domains.
- HSTS and CSP headers are present on production responses.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools verified the new Customer routing `Instance scope` UI at `/settings/agent-selection`; all related network requests returned `200`.
- Chrome DevTools found no JavaScript console errors; it reported only non-blocking accessibility issues for existing form labels.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```
