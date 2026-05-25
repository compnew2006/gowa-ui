# Whatomate Session Summaries

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
