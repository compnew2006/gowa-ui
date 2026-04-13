# Session Summary

## Date

- 2026-04-13

## Task

- Change uploads cleanup scheduling from `every 24h after server start` to a fixed daily time.
- Add a manual `Run Cleanup Now` action that uses the configured `Uploads Cleanup Retention (days)`.
- Restrict the feature to admins by default while exposing it as a grantable role permission.
- Add UI translations and verify both admin and permission-limited flows.

## Approach And Key Decisions

- Added a dedicated permission resource: `settings.uploads_cleanup`.
- Kept cleanup settings separate from general settings so users can be granted cleanup access without seeing unrelated organization fields.
- Switched worker scheduling to a per-organization fixed daily hour using the organization timezone and a stored last-run date.
- Kept manual cleanup organization-scoped and based on the current retention value.
- Backfilled the new uploads-cleanup permissions into existing system `admin` roles only.
- Fixed two runtime regressions surfaced by E2E:
  - role creation/update/delete paths were reusing stateful GORM request handles after previous queries
  - user creation had the same pattern and failed in PostgreSQL
- Hardened E2E login helpers to support configurable admin credentials and the actual default superadmin password from `config.toml`.

## Files Modified

- Backend implementation:
  - `cmd/whatomate/main.go`
  - `internal/database/postgres.go`
  - `internal/handlers/organization.go`
  - `internal/handlers/roles.go`
  - `internal/handlers/users.go`
  - `internal/handlers/uploads_cleanup_http.go`
  - `internal/handlers/uploads_cleanup_settings.go`
  - `internal/handlers/uploads_cleanup_worker.go`
  - `internal/models/roles.go`
- Frontend implementation:
  - `frontend/src/components/layout/navigation.ts`
  - `frontend/src/components/roles/PermissionMatrix.vue`
  - `frontend/src/i18n/locales/ar.json`
  - `frontend/src/i18n/locales/en.json`
  - `frontend/src/lib/constants.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/router/index.test.ts`
  - `frontend/src/services/api.ts`
  - `frontend/src/stores/roles.ts`
  - `frontend/src/views/settings/SettingsView.vue`
  - `frontend/src/views/settings/SettingsView.test.ts`
- Tests and E2E support:
  - `internal/database/database_test.go`
  - `internal/handlers/organization_test.go`
  - `internal/handlers/uploads_cleanup_http_test.go`
  - `internal/handlers/uploads_cleanup_worker_test.go`
  - `frontend/e2e/global-setup.ts`
  - `frontend/e2e/helpers/api.ts`
  - `frontend/e2e/helpers/auth.ts`
  - `frontend/e2e/pages/GeneralSettingsPage.ts`
  - `frontend/e2e/tests/settings/general-settings.spec.ts`
  - `frontend/e2e/tests/settings/permissions.spec.ts`

## Dependencies Or Environment Changes

- No new runtime dependencies were added.
- Installed Playwright Chromium locally for verification:
  - `npx playwright install chromium`
- E2E helpers now support:
  - `E2E_ADMIN_EMAIL`
  - `E2E_ADMIN_PASSWORD`
  - `E2E_SUPERADMIN_EMAIL`
  - `E2E_SUPERADMIN_PASSWORD`

## Tests Added Or Updated

- Backend:
  - `internal/handlers/organization_test.go`
    - cleanup-permission-only settings access
    - cleanup schedule hour validation
    - cleanup-only update permissions
  - `internal/handlers/uploads_cleanup_http_test.go`
    - manual cleanup success
    - forbidden without execute permission
    - bad request when retention is disabled
  - `internal/handlers/uploads_cleanup_worker_test.go`
    - fixed daily scheduling logic
    - manual cleanup uses organization retention
  - `internal/database/database_test.go`
    - uploads-cleanup permission backfill for admin roles
    - idempotency of that backfill
- Frontend:
  - `frontend/src/views/settings/SettingsView.test.ts`
    - save cleanup settings
    - run cleanup immediately
  - `frontend/src/router/index.test.ts`
    - `/settings` access via `settings.uploads_cleanup`
  - `frontend/e2e/tests/settings/general-settings.spec.ts`
    - cleanup retention field visibility
    - schedule-hour field visibility
    - cleanup save flow
    - run-now flow
  - `frontend/e2e/tests/settings/permissions.spec.ts`
    - user with uploads-cleanup permission sees cleanup controls only

## Verification Results

- Passed:
  - `go test ./internal/handlers -run 'TestApp_(GetOrganizationSettings|UpdateOrganizationSettings|RunUploadsCleanupNow|CreateRole|CreateUser|DeleteRole)|TestUploadsCleanupWorker'`
  - `go test ./internal/database -run 'TestBackfillAdminUploadsCleanupPermissions|TestBackfillAdminChatDeletePermission|TestBackfillSystemChatPrefixPermission'`
  - `npm --prefix frontend run test:unit -- src/views/settings/SettingsView.test.ts src/router/index.test.ts`
  - `npm exec -- eslint ...` on all changed frontend and E2E files from `frontend/`
  - `BASE_URL=http://127.0.0.1:3000 E2E_SUPERADMIN_PASSWORD=adminpassword12 E2E_ADMIN_EMAIL=admin@admin.com E2E_ADMIN_PASSWORD=adminpassword12 npm --prefix frontend run test:e2e -- --grep 'uploads cleanup' e2e/tests/settings/general-settings.spec.ts e2e/tests/settings/permissions.spec.ts`
- Browser QA via Chrome DevTools:
  - verified admin sees general settings plus the uploads cleanup card
  - verified cleanup card shows retention, fixed daily hour, timezone text, save action, and `Run Cleanup Now`
  - verified a custom-role user with `settings.uploads_cleanup:read` and `execute` sees only the cleanup controls and not the general organization fields
- Repro proof:
  - confirmed the old behavior before implementation used immediate startup cleanup plus 24-hour intervals from process start
- Known pre-existing issue outside this change:
  - full `npm --prefix frontend run typecheck` still fails in unrelated frontend files not touched by this feature

## Notes / Limitations

- Existing databases required a permission backfill because adding a new permission constant alone does not update already-seeded system roles.
- The fixed schedule currently stores a daily hour only, not minutes; the worker checks every minute and runs once per day when due.
