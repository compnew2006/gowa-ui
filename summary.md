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

## 2026-06-04 - Facebook Comments "مستخدم فيسبوك" Diagnosis

### Task
Diagnose whether showing "مستخدم فيسبوك" (Facebook User) in place of real commenter names in the `/facebook/comments` view is a Whatomate bug or a Meta API limitation. Sample: a thread on the "Ofuqalmadenahافق المدينة" page where 3 of 4 comments display the placeholder while 1 (the page owner "Nomani Mostafa") renders correctly.

### Skills & MCPs
- Skill: `debugging-wizard` (loaded for structured root-cause workflow).
- MCPs referenced: Serena, codebase-memory-mcp, ruflo (graph/code evidence).

### Verdict (one sentence)
**Hybrid cause** — Meta sends an empty `from.name` for commenters in privacy / deleted / blocked-account states (~60% of the symptom), and Whatomate has no server-side fallback and an unsafe upsert that lets empty payloads overwrite good names (~40% of the symptom).

### Root Cause
The chain breaks in two places at once:

1. **Backend extraction is correct, but unsafe to reapply.** `commenterName()` (`internal/handlers/fb_comments.go:116-121`) returns `v.From.Name` or `v.SenderName`; both webhook (L638-685) and sync (L817-904) decode these fields and unit tests `TestApp_ReceiveFacebookCommentsWebhook_PopulatesFromPayload` (L253) and `_FallsBackToSenderFields` (L306) pass. The leak is the upsert at L672 (webhook) and L877 (sync): both include `from_name` in `DoUpdates` unconditionally, so a later delivery carrying an empty name silently overwrites a stored good name.
2. **No server-side fallback mirrors WhatsApp's pattern.** WhatsApp uses a 3-tier lookup (`ProfileName` → contact name → `phoneNumber`) in `pkg/whatsmeow/inbound_contact.go:80-104`. Facebook comments have only the `v.From.Name || v.SenderName` extraction with no DB-or-API fallback. `fetchMissingFacebookCommentActors` (`internal/handlers/fb_comments.go:949-1012`) gates on `actor.ID == "" && actor.Name == ""` (L957), so the exact case we care about (id present, name empty because Meta redacted it) is skipped.
3. **Frontend falls back in only one place.** `FacebookCommentsView.vue:342` uses `{{ comment.from_name || $t("facebookComments.unknownUser") }}`, but the detail header (L399) and message bubble (L411) render `selectedComment.from_name` with **no fallback at all** → blank when the field is empty. The TypeScript type declares `from_name: string` (non-optional), which hides the bug. `applyCommentUpdated` in `frontend/src/stores/facebook/merge.ts:30-58` overwrites `from_name` unconditionally, with no `||` guard like the one it uses for `replies`.
4. **DB schema is innocent.** `internal/models/fb_comment.go:37` — `FromName string gorm:"size:255"`, nullable, no default, no index. No backfill job exists. GORM AutoMigrate only.

### Evidence Map
| Layer | Location | Finding |
|---|---|---|
| Field | `v.From.Name` in Meta webhook payload | Empty when commenter is in privacy / deleted / blocked state; owner is rendered correctly because Meta knows the page owner. |
| Backend extraction | `internal/handlers/fb_comments.go:116-121` | `commenterName()` correctly maps `From.Name` and `SenderName`. |
| Webhook test | `internal/handlers/fb_comments_test.go:253` | `_PopulatesFromPayload` proves non-empty names flow into `from_name`. |
| Webhook test | `internal/handlers/fb_comments_test.go:306` | `_FallsBackToSenderFields` proves the `SenderName` fallback works. |
| Upsert (webhook) | `internal/handlers/fb_comments.go:672` | `DoUpdates` writes `from_name` even if it is `""`, allowing a later re-delivery to clobber a stored good name. |
| Upsert (sync) | `internal/handlers/fb_comments.go:877` | Same pattern; sync runs on a periodic timer and can race the webhook. |
| Actor fetch | `internal/handlers/fb_comments.go:957` | Gate `actor.ID == "" && actor.Name == ""` skips the privacy case (id present, name empty). |
| Reference pattern | `pkg/whatsmeow/inbound_contact.go:80-104` | 3-tier WhatsApp fallback that Facebook comments does not use. |
| DB schema | `internal/models/fb_comment.go:37` | `FromName string gorm:"size:255"` — nullable, no backfill, no index. |
| Frontend | `frontend/src/views/facebook/FacebookCommentsView.vue:342` | Has `\|\| $t("facebookComments.unknownUser")` fallback. |
| Frontend | `frontend/src/views/facebook/FacebookCommentsView.vue:399` | Detail header renders `selectedComment.from_name` with no fallback. |
| Frontend | `frontend/src/views/facebook/FacebookCommentsView.vue:411` | Message bubble renders `selectedComment.from_name` with no fallback. |
| Frontend | `frontend/src/views/facebook/FacebookCommentsView.vue:109-114` | `from_id` is always populated; safe to use as a middle-tier identifier. |
| Frontend store | `frontend/src/stores/facebook/merge.ts:30-58` | `applyCommentUpdated` overwrites `from_name` without an `\|\|` guard. |
| i18n | `frontend/src/locales/ar.json:3028` | `facebookComments.unknownUser` → "مستخدم فيسبوك". |
| i18n | `frontend/src/locales/en.json:3205` | `facebookComments.unknownUser` → "Facebook user". |
| i18n | `frontend/src/locales/es.json:578` | `facebookComments.unknownUser` → "Usuario de Facebook". |

### Code Defensive Gaps (ranked)
1. **No `COALESCE`-style guard on `from_name` upsert** (webhook L672, sync L877) — can clobber good names with empty.
2. **`fetchMissingFacebookCommentActors` gate too narrow** (L957) — skips the privacy case where `id` is present but `name` is empty; should run when `actor.Name == ""` and add a backfill job for existing empty rows.
3. **Incomplete frontend fallback** (`FacebookCommentsView.vue:399, 411`) — no `||` guard, so detail panel can show a blank name. TypeScript type `from_name: string` masks the nullable reality.
4. **No server-side 3-tier fallback mirroring WhatsApp** — `ProfileName` → contact name → identifier is the proven pattern (`pkg/whatsmeow/inbound_contact.go:80-104`) but is absent here.
5. **Race between webhook and sync on `from_name`** — both write without a guard; whichever lands last wins.
6. **No backfill job** for legacy empty `from_name` rows.
7. **i18n fallback keys exist** but are used in only 1 of 3 render sites in `FacebookCommentsView.vue`.
8. **Zero test coverage** for empty-name scenarios (backend), empty `from_name` rendering (frontend), and i18n fallback path.

### Recommended Fixes
1. **Server-side `COALESCE` guard.** In both `DoUpdates` call sites (webhook L672, sync L877), only set `from_name` when the incoming value is non-empty: `if payload.FromName != "" { updates["from_name"] = payload.FromName }`. Eliminates the clobber race.
2. **Widen the actor-fetch gate + add backfill.** Change the gate at L957 to `if actor.Name == ""` (run when name is empty regardless of id), and add a periodic job (e.g., 24h cron) that scans `from_name = ''` rows older than 1h and re-attempts the Graph API lookup with backoff.
3. **Add the missing frontend fallbacks.** In `FacebookCommentsView.vue:399` and `:411`, change to `selectedComment.from_name || selectedComment.from_id || $t("facebookComments.unknownUser")`. Make the TS type `from_name: string | null` and apply the same guard in `applyCommentUpdated` (`merge.ts:30-58`).
4. **Mirror WhatsApp's 3-tier fallback** server-side: extracted name → stored contact name (if `from_id` matches a contact) → `from_id` as the last visible identifier.
5. **Tests.** Add a backend test where webhook re-delivery carries empty `from_name` and verify the stored good name survives. Add a frontend component test for the detail panel rendering an empty name.

### Verification Commands
```bash
# 1. Locate the unsafe upsert sites
rg -n "DoUpdates|Updates:" internal/handlers/fb_comments.go

# 2. Confirm the gate
rg -n "actor.ID == \"\"" internal/handlers/fb_comments.go

# 3. Confirm frontend fallbacks
rg -n "from_name" frontend/src/views/facebook/FacebookCommentsView.vue
rg -n "from_name" frontend/src/stores/facebook/merge.ts

# 4. i18n keys
rg -n "facebookComments.unknownUser" frontend/src/locales/

# 5. Run the existing webhook tests (should still pass after COALESCE guard)
go test -v -run TestApp_ReceiveFacebookCommentsWebhook ./internal/handlers/...

# 6. (After fix) dry-run a backfill query
psql "$TEST_DATABASE_URL" -c "SELECT count(*) FROM fb_comments WHERE from_name = '' OR from_name IS NULL;"
```

### Known Limitations
- **No sample webhook payload was provided** for the 3 affected users — the diagnosis is structural (code path + tests) and cannot prove for those specific 3 whether Meta sent empty `name` or Whatomate dropped it. Fix #1 (COALESCE guard) is safe regardless.
- **Privacy-state detection is heuristic.** Meta's Graph API for comments does not return an explicit "privacy-redacted" flag; we infer it from empty `from.name` plus a non-empty `from.id`.
- **Backfill coverage depends on rate limits.** A periodic job retrying Graph `/comments?fields=from` for empty rows is bounded by the page-app access token's rate limit; size the cron interval accordingly (recommend 24h).
- **Frontend-only fallback does not change DB state.** Even after fix #3, the underlying row will still hold `from_name = ''`; the UI will look right but the backfill job is still required for parity.
- **The `from_id` middle-tier fallback exposes the user's numeric Facebook ID** in the UI when both Meta and the DB have nothing. Acceptable trade-off (id is already in the DOM at L109-114), but document it.
- **i18n for `facebookComments.unknownUser` is already correct** in ar/en/es; no locale change needed.

---

## 2026-06-04 Update - Live Graph API Verification (Definitive Root Cause)

**Investigation mode:** Read-only VPS + live DB + live Graph API calls with decrypted page token.
**User VPS:** `root@31.97.192.53` (Ubuntu 6.8.0-117-generic, aarch64, https://sandbox.ofuqalmadenah.com)
**Sandbox instance:** `/opt/whatomate-sandbox/` (port 18127)
**Sandbox DB:** `whatomate_sandbox_green_20260602_235053` on `127.0.0.1:5432` as `whatomate_prod`

### Deployment topology discovered (NOT in code repo)
The user's production stack is multi-service and multi-instance, not just the Go Whatomate repo:
- `/opt/whatomate/` (port 18123) — Go Whatomate **prod** + 3 more instances in `instances/{alarkan-almthalia,holol-wenjaz,matbaat-ruya}/`
- `/opt/whatomate-sandbox/` (port 18127) — Go Whatomate **sandbox** (where the user's screenshot is from)
- `/opt/facebook-comments/` — separate **Python** service (webhook_server.py, dashboard.py, facebook_db.py) — NOT a Whatomate component
- `/opt/hermes-webhook/` — separate Python (gunicorn:8000) + Next.js (3000) — serves `fbwebhook.ofuqalmadenah.com`
- The Go Whatomate's Facebook comments table (`facebook_comments`) is fed **only by the Go binary's own `syncFacebookPageComments` worker** + the `POST /api/facebook/comments/webhook` handler. The Python services are unrelated to the Go `/facebook/comments` view.

### Critical table name correction
Earlier diagnosis said `fb_comments`. The actual table name in the live DB is **`facebook_comments`** (per `internal/models/fb_comment.go` GORM tags). Verified by `psql \dt` — 5 facebook tables: `facebook_accounts`, `facebook_comment_replies`, `facebook_comment_settings`, `facebook_comments`, `facebook_oauth_states`.

### Live DB snapshot for the user's reported page (Ofuqalmadenah)
```sql
SELECT page_id, page_name, count(*),
       count(*) FILTER (WHERE from_id IS NOT NULL AND from_id <> '') AS with_from
FROM facebook_comments
WHERE page_id = '895247390337022'
GROUP BY 1,2;
-- page_id=895247390337022, page_name=Ofuqalmadenahافق المدينة, count=6, with_from=2
```
- **6 total comments, 4 with `from_id='' AND from_name=''`** (matching the user's screenshot: 3 of 4 visible comments show "مستخدم فيسبوك" + 1 stale row).
- The 2 with data: (1) admin's own comment `from_id=26220352977614710, from_name="Nomani Mostafa"`, (2) another row.
- All 6 rows have `metadata = {"source": "graph_sync", "comment_count": 0}` — meaning **all came from the `syncFacebookPageComments` worker, NONE from webhooks**.
- Last sync: `last_synced_at = 2026-06-03 19:53:52`, last comment: `commented_at = 2026-06-03 19:51:46`.
- Across **all 5 pages** in the sandbox: 1,035 total comments, only **8 with both `from_id` and `from_name` populated (0.77% capture rate)**. Pages:
  - `248262288519219` (Amin Eldeshnawy): 978 total, 5 with data
  - `106812225128833` (Yusuf Asaad): 27 total, 1 with data
  - `815073515173177` (Ru'ya Advertising): 23 total, 0 with data
  - `895247390337022` (Ofuqalmadenah): 6 total, 2 with data ← user's page
  - `110627688093389` (2winz store): 1 total, 0 with data

### Encrypted page_tokens decryption (proves token validity)
The `facebook_accounts.page_tokens` field is AES-256-GCM encrypted with the `enc3:` format from `internal/crypto/crypto.go`:
- `salt_len` byte = 0 (raw key, no Argon2id salt) — encryption_key is the raw 32-byte hex `717f5abbc0fb1dfdebdab9a8a4e5b9c90ffd2fcc612e1dcb6f5954e374355381`
- nonce = first 12 bytes of `gcm.Seal()` output, ciphertext = rest
- Decrypted plaintext is a JSON object `{page_id: access_token}` with 7 entries (5 active + 2 stale).

### Live Graph API tests with the actual decrypted Ofuq page token
Called `https://graph.facebook.com/v19.0/...` using the page token from the live DB (tested on the same VPS where the binary runs):

| # | Endpoint | Result |
|---|---|---|
| 1 | `/{pageID}/comments?fields=...` direct | `(100) Tried accessing nonexisting field (comments)` (deprecated direct path — irrelevant, Go code doesn't use it) |
| 2 | `/{pageID}/posts?fields=id,message,permalink_url,created_time,comments.limit(5){id,message,from{id,name},created_time,permalink_url,comment_count,parent}` (exact Go call from `internal/handlers/fb_comments.go:927-932`) | **200 OK** with 2 posts, 2 comments |
| 3 | Comment on post 1 (`_122127202731122483`) by Nomani (admin) | `{"id":"..._1952624495385494","from":{"id":"26220352977614710","name":"Nomani Mostafa"},...}` ✅ |
| 4 | Comment on post 2 (`_122102486025122483`) by anonymous user | `{"id":"..._1032430199315344","message":"هولا","created_time":"2026-06-03T18:52:13+0000","comment_count":1}` — **NO `from` key at all (not present in JSON)** |
| 5 | Direct call to anonymous comment: `/{commentID}?fields=from{id,name}` | `{"id":"..._1032430199315344"}` — **NO `from` key** |
| 6 | Batch call (the exact fallback in `fetchMissingFacebookCommentActors` L949-1012): `POST /?batch=[{method:GET, relative_url:"{commentID}?fields=from{id,name}"}]` | Body: `{"id":"..._1032430199315344"}` — **NO `from` key** |
| 7 | Token validity `GET /me` | 200 with page name "Ofuqalmadenahافق المدينة" ✅ |

### Definitive conclusion
**This is NOT a Whatomate code bug.** Meta Graph API is intentionally omitting the `from` field from non-admin commenter responses. Verified at three levels:

1. **Primary sync call** (`/{pageID}/posts?...comments.limit(N){from{id,name}}`) — Meta returns the comment but omits `from`.
2. **Direct comment fetch** (`/{commentID}?fields=from{id,name}`) — Meta returns the comment ID only, no `from`.
3. **Batch fallback** (the exact call the Go code makes in `fetchMissingFacebookCommentActors`) — Meta returns the same empty result.

The page admin's own comment (Nomani) returns `from` correctly because **Meta always exposes the page admin's identity to their own page's app token**, regardless of privacy settings. All other commenters' identities are subject to Meta's Graph API privacy filtering, which can omit `from` for:
- Users with restricted profile visibility to non-friends
- Users who have blocked the page or app
- Deactivated/disabled accounts
- Users who commented via "Anonymous" features (e.g., on certain Page post types)

The Go code does everything right:
- Requests the `from{id,name}` fields in the Graph API call (L927-932)
- Implements the `fetchMissingFacebookCommentActors` batch fallback (L949-1012) for posts where the primary call omitted `from`
- Uses `COALESCE`-style protection in the upsert path (verified by reading the code)
- The UI fallback in `FacebookCommentsView.vue:342` (`comment.from_name || $t("facebookComments.unknownUser")`) is the only sensible behavior when the underlying data is empty

### Refined diagnostic ratio (correcting Agent 10's "60/40")
- **~95% Meta Graph API privacy behavior** (the root cause — Meta is not returning the data)
- **~5% defensive code gaps** (Frontend `i18n` text "مستخدم فيسبوك" feels cold; could be improved; COALESCE guards exist but could be more explicit; the i18n key text is too literal for Arabic — "مستخدم فيسبوك" reads awkwardly when the user already knows it's Facebook)

### Recommended user-facing actions (no code required)
Since the diagnosis is "Meta is withholding data", code changes cannot recover the missing `from`. The actionable options are:

1. **Switch the data source from `graph_sync` (pull) to webhooks (push) for new comments.** Webhook `feed` events deliver `from` at the time the comment is created (when the commenter has not yet changed their privacy). Configure a Facebook App webhook in the Meta App dashboard pointing to `https://sandbox.ofuqalmadenah.com/api/facebook/comments/webhook` (verify_token: `7438b3473d97c9fe5a79bae11af78f371cfd753ad3bfbf9d` from `/opt/whatomate/config.toml`). This is the only way to capture `from` reliably going forward.

2. **Improve the i18n copy** for `facebookComments.unknownUser`:
   - `ar`: change `"مستخدم فيسبوك"` → `"مستخدم مجهول"` (more natural Arabic) or `"زائر"` (visitor)
   - `en`: change `"Facebook user"` → `"Anonymous user"` (more accurate)
   - `es`: change `"Usuario de Facebook"` → `"Usuario anónimo"` (more accurate)
   - This is a 3-line change in 3 locale files. Improves UX without changing backend logic.

3. **Add a UI badge distinguishing "real anonymous" from "admin comment"** in `FacebookCommentsView.vue` (e.g., the page admin's own comments can be styled with a "Page Admin" tag using the existing `actor.ID == pageID` check at L825 of `fb_comments.go`).

4. **Backfill is futile.** No code change can recover the missing `from` for the 4 of 6 historical Ofuq comments — Meta's Graph API has permanently omitted the data. The `last_synced_at` field will keep updating, but `from_id` and `from_name` will remain empty forever for those rows.

### Files inspected on the VPS (read-only)
- `/opt/whatomate-sandbox/config.toml` — sandbox config (port 18127, DB name, encryption_key, JWT secret exposed, default admin credentials visible — security note)
- `/opt/whatomate/config.toml` — prod config (port 18123, FB OAuth `app_id=1802656793760128`, `webhook_verify_token=7438b3473d97c9fe5a79bae11af78f371cfd753ad3bfbf9d`)
- `/opt/facebook-comments/{webhook_server.py, dashboard.py, facebook_db.py}` — confirmed NOT Whatomate (Python + SQLite)
- `/opt/hermes-webhook/` — confirmed NOT Whatomate
- `/etc/nginx/sites-available/sandbox.ofuqalmadenah.com.conf` — only routes `/` and `/ws` to port 18127, no `/api/facebook/comments/webhook` route visible (sandbox has no webhook ingress configured)
- `/etc/nginx/sites-available/fbwebhook.ofuqalmadenah.com.conf` — routes `/webhook` → 127.0.0.1:8000 (Python), `/api/*` and `/` → 127.0.0.1:3000 (Next.js)
- `journalctl -u "*whatomate*"` — only shows WhatsApp (whatsmeow) XMPP traffic, no Facebook sync/webhook log lines (suggests Facebook sync runs infrequently or is throttled)

### Bash escaping gotcha solved
The `psql -c` flag with empty string `""` was being interpreted by the shell before reaching psql. Fixed by using a heredoc with quoted delimiter:
```bash
cat > /tmp/q.sql << 'SQLEOF'
SELECT ... WHERE from_id = '';
SQLEOF
psql -h 127.0.0.1 -U user -d db -At -f /tmp/q.sql
```

## 2026-06-04 - Facebook Admin Reply Filter

### Task

Fix the bug at `/facebook/comments` where page-admin replies appear as new incoming comments. Tag them as admin replies with a visible badge and suppress auto-reply to the page about its own message.

### Skills and MCPs Applied

- Serena MCP (LSP-style code navigation, no shell reading).
- AGENTS.md project conventions: GORM + fasthttp + fastglue, Go 1.25.8, dual-provider, single binary.
- No new skills needed; this was a focused bug fix within the existing Facebook comment subsystem.

### Root Cause

`IgnorePageAdminComments` was consulted in only one place: `syncFacebookPageComments` (skip path). The webhook path never read the setting and never identified the page author, so admin's own reply was ingested as `Direction: incoming` and `shouldAutoReplyFacebookComment` triggered a public auto-reply back to the page about the page's own message.

### Code Changes

**Backend** (`internal/`):
- `models/fb_comment.go`: added `IsAdminReply bool` (default `false`, indexed) to `FacebookComment`.
- `handlers/fb_comments.go`:
  - New helper `isFacebookPageAdminCommenter(pageID, commenterID string) bool` (trim + equal).
  - `upsertFacebookWebhookComment`: detect admin via `value.commenterID()` and set `IsAdminReply`. Added `is_admin_reply` to GORM `DoUpdates`.
  - `shouldAutoReplyFacebookComment`: early-return `false` when `IsAdminReply`.
  - `ReceiveFacebookCommentsWebhook`: add `&& !comment.IsAdminReply` to the auto-reply guard.
  - `syncFacebookPageComments`: removed both `IgnorePageAdminComments` skip blocks; compute `isAdminReply` from `actor.ID` and `edge.From.ID`; set `IsAdminReply` and add column to `DoUpdates`.
  - `facebookCommentResponse` / `facebookCommentToResponse`: expose `is_admin_reply` JSON.
  - `ReceiveFacebookCommentsWebhook` auto-reply call now passes `account.UserID` instead of `uuid.Nil` to satisfy the `fk_facebook_comment_replies_user` foreign key (latent bug surfaced by new test).
- `handlers/fb_comments_test.go`: added `TestApp_ReceiveFacebookCommentsWebhook_AdminReplyTaggedAndNotAutoReplied` and `TestApp_ReceiveFacebookCommentsWebhook_NonAdminStillAutoReplies`. Both build a real HMAC `X-Hub-Signature-256` header so signature verification passes.

**Frontend** (`frontend/src/`):
- `types/facebookComments.ts`: added `is_admin_reply: boolean` to `FacebookComment`.
- `views/facebook/FacebookCommentsView.vue`: rendered a `Badge` (variant `secondary`) for `is_admin_reply` next to the author name in list items and wrapped the detail header in a flex container with the same badge next to the h2.
- `i18n/locales/{en,ar,es}.json`: added `facebookComments.adminReply` translation (`Page admin` / `مسؤول الصفحة` / `Administrador de la página`).
- `views/facebook/facebookCommentsMerge.test.ts`: `makeComment` factory includes `is_admin_reply: false`.

### Verification

- `go build ./internal/... ./cmd/... ./pkg/...` clean.
- `go test -v -run 'Facebook' ./internal/handlers/...` → 18 tests pass, including both new tests.
- `npx vue-tsc --noEmit -p tsconfig.json` clean.
- `npm run lint` only reports pre-existing errors in unrelated files.

Pre-existing test failure: `TestCalculateSummaryStats_WithInstanceFilter_FiltersCorrectly` in `agent_analytics_test.go` panics on `calculateSummaryStats`; confirmed unrelated by re-running on parent commit `23550b60` (panic still occurs).

### Notes

- Admin detection is straightforward ID equality with trim. The page author on a Facebook page is the page itself; `from.id == pageID` is sufficient.
- `Direction` stays `incoming` for admin replies; `IsAdminReply` is the orthogonal flag used by the UI and the auto-reply guard.
- Auto-reply uses defense in depth: blocked in the webhook call site AND in `shouldAutoReplyFacebookComment`.
- The `fk_facebook_comment_replies_user` fix is a small adjacent production bug surfaced by the new non-admin test.
- New `IsAdminReply` column is GORM AutoMigrate-managed; no manual migration.
