# Whatomate Session Summaries

## 2026-05-26 - Customer Agent Selection for WhatsMeow

### Task

Implement the planned feature that lets WhatsMeow customers choose a specific visible agent, team, queue, or custom final option after a configurable delay, while preserving the current pending assignment behavior and adding audit coverage.

### Skills and MCPs Applied

- Skills: `ccc`, `architecture-guardian`, `feature-forge`, `test-master`.
- MCPs/tools: CocoIndex code search/indexing via `ccc`, shell-based Go verification.

### Code Changes

- Added additive Customer Agent Selection models:
  - `AgentSelectionSettings`
  - `AgentSelectionParticipant`
  - `AgentSelectionOption`
  - `AgentSelectionSession`
  - `AgentSelectionAuditEvent`
- Added backend APIs under `/api/agent-selection` for settings, participants, options, preview, sessions, cancellation, and audit.
- Added a delayed background processor that sends the WhatsMeow text menu only while a chat is still pending and unassigned.
- Added a WhatsMeow inbound hook before normal chatbot routing to schedule delayed menus and process active customer replies.
- Added `customer_selection` as a transfer source.
- Added `agent_selection:read/write` permissions.
- Added focused backend tests for snapshot parsing, idempotency helpers, and keyword normalization.
- Added full feature spec in `specs/customer-agent-selection.spec.md`.
- Added frontend API service and Pinia store for Customer Agent Selection.
- Added `/settings/agent-selection` Customer Routing UI with settings, agents, options, preview, sessions, and audit tabs.
- Added navigation and localized labels for Customer Routing.

### Verification

- `go test ./internal/handlers -run 'TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'` passed.
- `go test ./cmd/whatomate ./internal/models ./internal/database ./internal/handlers -run TestNonExistent` passed.
- `go test ./...` passed.
- `ccc index` completed successfully after adding the new files.
- `cd frontend && ./node_modules/.bin/eslint src/views/settings/AgentSelectionView.vue src/stores/agentSelection.ts src/services/api.ts src/router/index.ts src/components/layout/navigation.ts` passed.
- `cd frontend && npm run build` passed with the existing chunk-size warning.
- `cd frontend && npm run typecheck` is still blocked by pre-existing project-wide TypeScript errors outside this feature.

### Remaining Work

- Add DB-backed integration tests for delayed prompt sending and full assignment/transfer commits.
- Add WhatsMeow staging smoke verification after enabling the feature for one instance.

## 2026-05-19 - Media Dedup / Resize Fix

### Task

Fix and deploy the green Whatomate version for:

- Missing WhatsMeow media on the two 07:14 PM chat bubbles in `https://ofuqalmadenah.com/chat/82edad6a-708a-4ce9-af2b-6c8f72b27cac`.
- Retry download falsely reporting success when the underlying media blob is absent.
- Future inbound WhatsMeow dedup reusing stale `media_asset` rows whose object blobs are missing.
- Fatal Assign Contact dialog resize crash: `t.value.getBoundingClientRect is not a function`.
- Confirm license overview is active and green is the running side of the blue/green deployment.

### Approach And Key Decisions

- Confirmed production green was active before debugging.
- Reproduced the 07:14 PM media issue with browser automation and API checks.
- Verified the two current PDFs cannot be reconstructed by the app because their object files are missing and the message rows do not contain WhatsMeow recovery metadata.
- Fixed future inbound WhatsMeow media dedup so it only reuses an existing media asset when the blob exists; otherwise it downloads and stores the current inbound payload again.
- Fixed retry behavior so object-backed media must verify the blob exists before returning success.
- Fixed the Vue resizable composable to accept Reka/Vue component refs by resolving `$el` before calling DOM APIs.
- Clarified active-license quota copy so the License page no longer says licensing is disabled when the deployment is active.

### Files Modified

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

### Deployment

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

### Verification

- `go test ./pkg/whatsmeow ./internal/handlers` passed.
- `cd frontend && npx vitest run src/lib/useResizable.test.ts` passed.
- `cd frontend && npx eslint src/lib/useResizable.ts src/lib/useResizable.test.ts src/components/ui/dialog/DialogContent.vue` passed.
- `cd frontend && npm run typecheck` still fails on pre-existing project-wide TypeScript errors outside this change.
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
  - Affected media stream and retry endpoints return `404` with `No recovery information available for this media`.
  - Assign Contact dialog opened on an active chat and resized without fatal UI error or `getBoundingClientRect` page error.
  - License page shows `License overview` as `Active`; no Disabled/licensing-disabled copy remains.

### Artifacts

- `playwright-green-chat-media-after-fix.png`
- `playwright-green-assign-resize-after-fix.png`
- `playwright-green-license-after-fix.png`
- Earlier investigation screenshots:
  - `internal-browser-chat-media-investigation.png`
  - `playwright-chat-media-missing.png`
  - `playwright-chat-media-retry-clicks.png`

### Known Limitations

The two already-broken 07:14 PM PDFs still cannot display because the referenced blobs are absent from `/opt/whatomate/uploads` and backups checked during the investigation, and the affected rows have no stored WhatsMeow recovery payload. The deployed fix prevents this stale-dedup path from recurring for future inbound WhatsMeow media and stops retry from returning fake success for unrecoverable historical media.

## 2026-05-22 - Uploads Cleanup WhatsMeow Media

### Task

Fix the Uploads Cleanup system from `/settings` so it cleans `/opt/whatomate/uploads/whatsmeow/media/`, explain why both `/opt/whatomate/uploads/` and `/opt/whatomate/uploads/whatsmeow/media/` exist, deploy the fixed green build to the VPS, and verify production behavior.

### Skills and MCPs Applied

- Skills: `debugging-wizard`, `golang-pro`, `ccc`, `browser`.
- MCPs/tools: CocoIndex code search for code discovery, Ruflo memory search/store workflow, Chrome DevTools/browser smoke verification, shell only for tests, build, git, and VPS deployment.

### Root Cause

The configured storage root is `/opt/whatomate/uploads`. General transient uploads are stored directly below this root in category directories such as `images`, `videos`, `audio`, `documents`, and `stickers`. WhatsMeow inbound received-media files are also stored under the same root, but in a nested provider-specific directory: `/opt/whatomate/uploads/whatsmeow/media`.

The cleanup worker only swept the top-level category directories and organization-scoped equivalents. It did not include `whatsmeow/media`, so the Settings cleanup action could complete successfully while leaving WhatsMeow received-media files untouched.

### Code Changes

- `internal/handlers/uploads_cleanup_settings.go`
  - Added `filepath.Join("whatsmeow", "media")` to `uploadsCleanupTargetDirs`.
- `internal/handlers/uploads_cleanup_worker_test.go`
  - Extended the cleanup worker test to prove expired WhatsMeow media is deleted and fresh WhatsMeow media is retained.
- `docs/whatomate_multi_instances_info.md`
  - Added the production green deployment and verification note.

### VPS Changes

- Built and deployed green binary:
  - `/opt/whatomate/bin/whatomate.green.20260522_234238`
  - Version: `Whatomate green-20260522_234238-0d74527-uploads-cleanup`
- Active symlink now resolves to:
  - `/opt/whatomate/bin/whatomate.green.20260522_234238`
- Backup created before install:
  - `/root/whatomate_backup_before_uploads_cleanup_20260522_234238`
- Updated VPS notes:
  - `/root/whatomate_multi_instances_info.md`
  - `/root/whatomate_production_info.md`
- Operational fix on VPS:
  - `/usr/local/bin/whatomate-housekeeping.sh` disk snapshot now tolerates optional missing paths, then reran successfully with exit status `0/SUCCESS`.

### Verification

- Local tests:
  - `go test ./internal/handlers -run 'TestUploadsCleanupWorker|TestApp_RunUploadsCleanupNow'` passed.
  - `go test ./internal/handlers ./pkg/whatsmeow` passed.
  - `go test ./...` passed.
- Frontend build:
  - `cd frontend && npm run build` passed.
- Production API verification:
  - Logged in on port `18123`.
  - Settings returned `uploads_cleanup_retention_days=5`.
  - Created an old throwaway file in `/opt/whatomate/uploads/whatsmeow/media/`.
  - Called `/api/org/uploads-cleanup/run`.
  - Response returned `deleted_files=2337`, `retention_days=5`.
  - The throwaway file was removed.
  - Follow-up check found `whatsmeow_media_old_gt5d=0`.
- Production service verification:
  - `whatomate.service`: active.
  - `whatomate@alarkan-almthalia.service`: active.
  - `whatomate@holol-wenjaz.service`: active.
  - `whatomate@matbaat-ruya.service`: active.
  - Login checks on ports `18123`, `18124`, `18125`, `18126`: HTTP `200`.
- Browser/DevTools verification:
  - Opened `https://ofuqalmadenah.com/settings`.
  - Confirmed Uploads Cleanup controls are visible with retention set to `5`.
  - Chrome DevTools reported no console messages.

### Known Limitations

- No Playwright suite was run against production because the production verification was performed through the live API plus the in-app browser/Chrome DevTools smoke check.
- Cleanup only deletes files older than the configured retention. For the verified organization, retention is `5` days, so files newer than that remain by design.

## 2026-05-25 - Green Text Send Fix Deployment

### Task

Deploy the current project to the VPS as the new green slot, preserve the blue rollback slot, fix the WhatsMeow text-send `400` failure seen on chat `8b04fdf4-3f6c-4226-a003-c0ade8c7b75d`, verify that the license overview is active, remove temporary/source codebases from the VPS after deployment, update the deployment documentation, and keep a one-command blue/green switch.

### Skills and MCPs Applied

- Skills: `devops-engineer`, `debugging-wizard`.
- MCPs/tools: Chrome DevTools for production UI verification, shell/SSH for build and systemd verification.

### Deployment

- Deployed source revision: `c1e34cd` (`Fix whatsmeow plain text sends`).
- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260525_200333`.
- Version: `Whatomate green-20260525_200333-c1e34cd-text-send (built 2026-05-25_20:07:08)`.
- SHA256: `fd8a6947d335531d4ee8ac85f2e2fb35a134d9351dbda972692bfbfb3797f18d`.
- Blue rollback binary left untouched: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260525_192630_pre_green_text_send_fix` (`759M`).

### Fix

- Plain WhatsMeow text messages now build a simple `Conversation` payload.
- Text messages containing URLs still use `ExtendedTextMessage` to preserve close-rating review-link delivery.
- The affected chat page still shows historical failed rows from before the deploy; no production Retry/send was triggered from the browser because that would send a real customer message.

### Verification

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

### Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.

## 2026-05-28 - Green Agent Selection UI Polish Deployment

### Task

Deploy the current UI polish changes to the main Whatomate service and the three dedicated tenants: `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, and `whatomate@matbaat-ruya`.

### Skills and Tools Applied

- Skill selected: `devops-engineer`.
- Tools used: SSH, rsync, native VPS build, systemd, curl.

### Deployment

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Version: `Whatomate green-20260528_111523-09191c2-agent-ui (built 2026-05-28_11:18:57)`.
- SHA256: `4abd7096755d01623a54c4e56290fce386ecf256c45f098b521bd518ef08c921`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260528_111523_pre_agent_ui_polish`.

### Verification

- Local checks passed:
  - `cd frontend && npm run typecheck`
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'`
  - `git diff --check`
- VPS build passed with embedded license keyring.
- Services active and all running from `/opt/whatomate/bin/whatomate.green.20260528_111523`: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- License bootstrap on ports `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for the main domain and all three tenant domains.
- HSTS and CSP headers are present on production.

### Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.

### Cleanup

- Removed temporary/source VPS paths after the verified install:
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, and backups.

## 2026-05-27 - Security Headers Fix

### Task

Fix missing HSTS and CSP security findings.

### Changes

- Added `Strict-Transport-Security: max-age=31536000; includeSubDomains`.
- Ensured `Content-Security-Policy` is set whenever it is missing instead of skipping SPA-style paths.
- Preserved the existing frontend nonce CSP when the embedded SPA serves `index.html`.
- Applied security headers from the fasthttp CORS wrapper so preflight `OPTIONS` responses receive the same protections.

### Verification

- `go test ./internal/middleware -run 'TestSecurityHeaders|TestSetSecurityHeadersPreservesExistingCSP'`
- `go test ./cmd/whatomate -run TestCorsWrapperAppliesSecurityHeadersToPreflight`
- `go test ./internal/frontend`
- `go test ./cmd/whatomate ./internal/middleware`
- `git diff --check`

## 2026-05-27 - Green Current Project Deployment

### Task

Deploy the current working project as the new green slot on VPS `31.97.192.53`, keep blue rollback side-by-side, make the license overview active, clean source code from the VPS, update docs, and provide a one-command switch.

### Skills and Tools Applied

- Skills selected: `devops-engineer`, `debugging-wizard`.
- MCP/tooling: Chrome DevTools for production browser verification; SSH/systemd/curl for server verification.

### Deployment

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260527_174500`.
- Version: `Whatomate green-20260527_174500-09191c2-csp (built 2026-05-27_17:42:53)`.
- SHA256: `a140bc30a10d018f05ff1da97bc9505f7ff1d82d241721b78ae74281bd948ff0`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_172753_pre_green_current_project`.

### Verification

- Local checks passed:
  - `go test ./...`
  - `cd frontend && npm run typecheck`
  - `git diff --check`
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./cmd/whatomate ./internal/middleware ./internal/frontend`
- VPS build passed with embedded license keyring.
- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- License bootstrap on ports `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public login checks returned `200` for the main domain and all three tenant domains.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools initially caught a CSP nonce regression; fixed and redeployed as `green-20260527_174500-09191c2-csp`.
- Final Chrome DevTools reload reported no console messages and all listed network requests returned `200`.
- Production document response includes nonce-based CSP plus HSTS.

### Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.

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

## 2026-05-28 - Green Agent Scope Deployment

### Task

Continue the interrupted green deployment, make the newly built project replace all currently running VPS instances, keep blue rollback available, verify license state, update docs, and clean temporary source code.

### Skills and Tools Applied

- Skill selected: `devops-engineer`.
- MCP/tooling: Chrome DevTools for production browser verification; SSH/systemd/curl for server verification.

### Deployment

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- Version: `Whatomate green-20260528_020100-09191c2-agent-scope (built 2026-05-28_02:00:01)`.
- SHA256: `4cbcfa440a67fba3d568b25e43f77e7a0352ebf71a0acd74bfbea0a3a1d2eabf`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_181332_pre_green_instance_scope`.

### Verification

- Local checks passed:
  - `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'`
  - `cd frontend && npm run typecheck`
  - `git diff --check`
- VPS build passed with embedded license keyring.
- Services active and all running from `/opt/whatomate/bin/whatomate.green.20260528_020100`: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- License bootstrap on ports `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for the main domain and all three tenant domains.
- HSTS and CSP headers are present on production.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools verified `/settings/agent-selection` loads the new `Instance scope` UI and related API calls return `200`.
- Chrome DevTools found no JavaScript console errors; it only reported non-blocking accessibility issues for existing form labels.

### Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.
