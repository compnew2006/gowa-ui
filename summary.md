# Whatomate Session Summary - Uploads Cleanup WhatsMeow Media

Date: 2026-05-22
Branch: `agent/fix-uploads-cleanup-whatsmeow-media`

## Task

Fix the Uploads Cleanup system from `/settings` so it cleans `/opt/whatomate/uploads/whatsmeow/media/`, explain why both `/opt/whatomate/uploads/` and `/opt/whatomate/uploads/whatsmeow/media/` exist, deploy the fixed green build to the VPS, and verify production behavior.

## Skills and MCPs Applied

- Skills: `debugging-wizard`, `golang-pro`, `ccc`, `browser`.
- MCPs/tools: CocoIndex code search for code discovery, Ruflo memory search/store workflow, Chrome DevTools/browser smoke verification, shell only for tests, build, git, and VPS deployment.

## Root Cause

The configured storage root is `/opt/whatomate/uploads`. General transient uploads are stored directly below this root in category directories such as `images`, `videos`, `audio`, `documents`, and `stickers`. WhatsMeow inbound received-media files are also stored under the same root, but in a nested provider-specific directory: `/opt/whatomate/uploads/whatsmeow/media`.

The cleanup worker only swept the top-level category directories and organization-scoped equivalents. It did not include `whatsmeow/media`, so the Settings cleanup action could complete successfully while leaving WhatsMeow received-media files untouched.

## Code Changes

- `internal/handlers/uploads_cleanup_settings.go`
  - Added `filepath.Join("whatsmeow", "media")` to `uploadsCleanupTargetDirs`.
- `internal/handlers/uploads_cleanup_worker_test.go`
  - Extended the cleanup worker test to prove expired WhatsMeow media is deleted and fresh WhatsMeow media is retained.
- `docs/whatomate_multi_instances_info.md`
  - Added the production green deployment and verification note.

## VPS Changes

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

## Verification

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

## Known Limitations

- No Playwright suite was run against production because the production verification was performed through the live API plus the in-app browser/Chrome DevTools smoke check.
- Cleanup only deletes files older than the configured retention. For the verified organization, retention is `5` days, so files newer than that remain by design.
