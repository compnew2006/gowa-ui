## Task

Investigate and fix Uploads Cleanup behavior where Retention (days) set to 5 and Run Cleanup Now did not remove old files from `/opt/whatomate/uploads`, and change Daily Cleanup Hour from an integer field to a time-format field.

## Root Cause

Uploads cleanup only scanned legacy root-level media folders:

- `audio/`
- `documents/`
- `images/`
- `stickers/`
- `videos/`

Current local media uploads are saved under organization-scoped paths like `orgs/<organization-id>/documents/...` via `organizationMediaSubdir`, so Run Cleanup Now could complete successfully while leaving old production files under `/opt/whatomate/uploads/orgs/...` untouched.

## Approach And Key Decisions

- Kept the backend storage/API representation as `uploads_cleanup_schedule_hour: number` for compatibility.
- Changed the UI control to `type="time"` and display/persist values as `HH:MM` whole-hour times.
- Accepted legacy schedule inputs such as `3` or `03:00`, but rejected non-whole-hour values such as `03:30`.
- Extended cleanup to scan `uploads/orgs/<org-id>/<media-type>` for the existing transient media target folders only.
- Preserved safety constraints: missing directories are ignored, symlinks are skipped, and non-target directories such as `chat-backgrounds` are not cleaned.

## Files Modified / Created

- `internal/handlers/uploads_cleanup_worker.go`
- `internal/handlers/uploads_cleanup_worker_test.go`
- `frontend/src/views/settings/SettingsView.vue`
- `frontend/src/views/settings/SettingsView.test.ts`
- `frontend/e2e/tests/settings/general-settings.spec.ts`
- `frontend/e2e/tests/settings/uploads-cleanup.spec.ts` (new)
- `frontend/src/i18n/locales/en.json`
- `frontend/src/i18n/locales/ar.json`
- `summary.md`

## Dependencies Or Env Changes

No dependency or environment changes.

## Tests Added / Updated

- Added backend regression `TestUploadsCleanupWorker_RunManualCleanup_DeletesOrganizationScopedUploads` covering deletion under `orgs/<org-id>/documents`, preservation of fresh files, and preservation of `chat-backgrounds`.
- Updated Vue unit coverage so uploads cleanup schedule renders as a time input, displays `03:00`, and sends backend hour `4` for `04:00`.
- Added Playwright Chromium smoke `frontend/e2e/tests/settings/uploads-cleanup.spec.ts` with mocked API calls to verify the real settings page time input and save payload.
- Updated existing settings E2E expectations from integer hour value `4` to time value `04:00`.

## Verification Results

- Baseline before fix:
  - `go test ./internal/handlers -run 'Test.*UploadsCleanup' -count=1` passed before adding regressions.
  - New backend regression failed before implementation because org-scoped expired files were not deleted.
  - New frontend unit regression failed before implementation because the schedule field was still `type="number"`.
- After fix:
  - `go test ./internal/handlers -run 'Test.*UploadsCleanup|TestApp_(Get|Update)OrganizationSettings' -count=1` passed.
  - `go test ./internal/handlers -count=1` passed.
  - `go test ./...` passed all tested packages except pre-existing `github.com/compnew2006/whatomate/tmp` build failure from multiple `main` functions in tmp repro files.
  - `cd frontend && npx vitest run src/views/settings/SettingsView.test.ts` passed.
  - `cd frontend && npm run lint` passed with one pre-existing warning in `frontend/e2e/tests/chat/soft-delete.spec.ts`.
  - `cd frontend && npm run build` passed with existing large chunk warnings.
  - `cd frontend && npm run typecheck` failed on existing unrelated TypeScript errors outside the changed files.
  - `cd frontend && npx playwright test --project=chromium e2e/tests/settings/uploads-cleanup.spec.ts --reporter=list --timeout=30000 --workers=1` passed in Chromium. Global setup logged `localhost:8080` connection refusals because the backend was not running, but the mocked browser test itself passed.
  - Existing `general-settings.spec.ts` Playwright run was attempted and exceeded the 120s tool limit in this local environment because it depends on live backend login/setup.

## Known Limitations

- Full repo `go test ./...` is still blocked only by the pre-existing `tmp` package duplicate-main build issue.
- Frontend typecheck is still blocked by unrelated pre-existing type errors already present in chat, contacts, chatbot, dashboard, and teams files.
- Playwright global setup expects a backend on `localhost:8080`; the added smoke avoids backend dependency by mocking browser API calls, but global setup still logs refused backend connections when no backend is running.
- Serena tools `initial_instructions`, `project_overview`, `style_and_conventions`, `suggested_commands`, and `done_checklist` were not active in this session. Project memories and available Serena tools were used instead, and this summary records the manual completion checklist.
