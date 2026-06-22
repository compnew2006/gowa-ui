# GREEN Deployment Summary - 2026-06-15 02:35 UTC

## Actions Taken

### 1. BACKUP
- Created timestamped backup of current VPS deployment
- **Location**: `/root/whatomate_backups/pre_green_deploy_20260615_022658/`
- Contents: all binaries, config.toml, instances/, ssl/

### 2. BUILD
- Built production binary from local source (commit `f7513f9d`)
- Frontend: Vue 3 + Vite (embedded)
- Backend: Go single binary (linux/amd64, CGO_ENABLED=0)
- License key ring embedded at build time

### 3. DEPLOY GREEN
- **New GREEN binary**: `/opt/whatomate/bin/whatomate.green-20260614_233550-f7513f9d`
- **BLUE rollback**: `/opt/whatomate/bin/whatomate.green-20260614_143926-f7513f9d` (previous)
- Symlink `whatomate.green` → new binary
- Symlink `whatomate.blue` → old active binary
- Switched via `whatomate-switch green` (restarted `whatomate` + `whatomate@holol-wenjaz`)

### 4. LICENSE
- Status: **Enabled** ✅
- Status: **Active**
- Tier: **production**
- Kind: **paid**, **lifetime**
- Key ID: `deploy-20260416`

### 5. UPDATED DOCS
- `/root/whatomate_production_info.md` on VPS
- `/root/whatomate_multi_instances_info.md` on VPS
- `/docs/whatomate_multi_instances_info.md` locally

### 6. VERIFICATION
- `https://ofuqalmadenah.com` → HTTP 200
- `https://sandbox.ofuqalmadenah.com` → HTTP 200
- `GET /api/license/bootstrap` → `{"enabled": true, "status": "active", "tier": "production"}`
- Services `whatomate` and `whatomate@holol-wenjaz` both **active**

## One-Command Blue/Green Switch

### Switch to GREEN (new version):
```bash
ssh root@31.97.192.53 "whatomate-switch green"
```

### Rollback to BLUE (previous version):
```bash
ssh root@31.97.192.53 "whatomate-switch blue"
```

### Toggle (flip between green/blue):
```bash
ssh root@31.97.192.53 "whatomate-switch toggle"
```

### Check current status:
```bash
ssh root@31.97.192.53 "whatomate-switch status"
```

## Blue/Green Binaries
| Role | Binary | SHA256 |
|------|--------|--------|
| **GREEN (active)** | `whatomate.green-20260614_233550-f7513f9d` | `e1bd749bcc560a14e2628ed97e419c0546ce1133fd04151f0c2a6769a0c4affa` |
| **BLUE (rollback)** | `whatomate.green-20260614_143926-f7513f9d` | (previous build) |

## Risks / Notes
- Debug mode still enabled in config (`debug = true`) - set to `false` and restart to disable
- Disk is at 85% (16G free of 96G) - monitor for growth
- Sandbox switch script is separate: `whatomate-sandbox-switch`

## Deployment: f35294f8 — Facebook Comments Per-Page Settings + i18n Fix
**Date:** 2026-06-15 02:56 UTC  
**Commit:** `f35294f8`

### Changes
- **Backend fix**: `getOrCreatePageCommentSettings` now uses clean DB session (`NewDB: true`) and looks up correct `AccountID` — fixes per-page settings not persisting to DB.
- **i18n fix**: Added 4 missing translation keys (`multiTextHint`, `autoCommentPlaceholder`, `autoPrivatePlaceholder`, `globalDefaults`) to both ar.json and en.json.
- **Pre-existing fixes included**: `ReplyFacebookComment` scoping fix, `UpdateFacebookCommentStatus` transition validation.

### Deploy
- **Production**: GREEN active (binary `whatomate.green.f35294f8`, SHA256 `f917d309`)
- **Sandbox**: GREEN active (binary `whatomate.sandbox.f35294f8`)
- **Blue rollback**: previous `f7513f9d`
- **Backup**: `/root/whatomate_backups/20260614_235635_pre_f35294f8/`
- **License**: active, key_id=deploy-20260416

## Deployment: fd09ad62 — Filter admin auto-replies from FB Comments UI
**Date:** 2026-06-15 04:18 UTC  
**Commit:** `fd09ad62`

### Change
- Added `if created && comment.IsAdminReply { continue }` in the webhook handler
- This skips auto-replies from the page admin (the system's own replies) so they don't appear in the Facebook Comments inbox
- **Prod binary**: `whatomate.green.fd09ad62` (SHA256 `ecb28ef`)
- **Sandbox binary**: `whatomate.sandbox.fd09ad62`
- **Rollback**: `e445d2fb`

### Status
- ✅ Translations fixed
- ✅ Per-page settings persist after refresh
- ✅ Admin auto-replies hidden from UI
- ❗ WhatsApp notification requires: production per-page config OR connected sandbox instance

---

# GREEN Deployment Summary - 2026-06-20 08:02 UTC

## Actions
1. **Backup**: `/root/whatomate_backups/pre-deploy-20260620_075734.tar.gz` (95MB)
2. **Build**: Cross-compiled `d5ce1326` for linux/amd64 with license keyring embedded
3. **Deploy**: Uploaded `whatomate.green.d5ce1326-deploy20260620_075900` to VPS
4. **Restart**: `systemctl restart whatomate.service`

## Verification
| Check | Result |
|---|---|
| Version | `d5ce1326-deploy20260620_080042` ✅ |
| Service | active ✅ |
| `/health` | 200 ✅ |
| License | enabled=true, status=active ✅ |
| Drops | 0 ✅ |
| Instances | 16 connected |

## One-Command Switch
```bash
# GREEN (active)
ln -sfn /opt/whatomate/bin/whatomate.green.d5ce1326-deploy20260620_075900 /opt/whatomate/bin/whatomate && systemctl restart whatomate

# BLUE (rollback)
ln -sfn /opt/whatomate/bin/whatomate.green.cfbcc1ec-deploy150615 /opt/whatomate/bin/whatomate && systemctl restart whatomate
```

---

# Commit Regression Testing & Code Review — 2026-06-22

## Target Commit
- **Commit**: `59580e76`
- **Title**: `feat: extract Facebook modules to plugins and add DB-controlled module system`

## Verification Summary
- **Backend Tests**: Passed (`go test ./...` -> 33 packages pass)
- **Frontend Unit Tests**: Passed (`npm run test:unit` -> 195 tests pass)
- **TypeScript Typecheck**: Passed (`npm run typecheck` -> successful compilation)
- **Linter**: Passed (`npm run lint` -> successful, 0 errors)
- **Production Build**: Passed (`make build-prod` -> successful build of 55MB embedded binary)

## Fixes Implemented
- Resolved pre-existing test environment issues by adding `happy-dom` to `whatsapp-filter-api.test.ts`.
- Resolved Vue Query client injection failure in `InstanceCard.test.ts` by stubbing `PerInstanceUploadsCleanup`.
- Resolved TypeScript compilation errors in `CampaignsView.vue` and `SavedContentsView.vue` by properly casting Axios header values.
- Resolved eslint warning for empty catch block in `FacebookCommentsView.vue`.

---

# License ↔ Module Bridge + Plugin RBAC — 2026-06-22

## Task
Integrate the existing License system with the existing Module system to control per-org plugin entitlements, add audit for give/ungive, and introduce plugin-namespaced RBAC (`plugin:{name}:{feature}:{action}`).

## Decision gate (recorded)
The task premise did not match the codebase: the License system is host-bound global caps (Ed25519 JWT, HWID-tied), NOT per-tenant; a DB-controlled Module system already implements most of requirement B. User chose:
1. **Bridge License→Module** (no JWT changes) — recommended.
2. **Plugin self-registers permissions** via optional interface.
3. **Verify-then-fill-gaps** (DRY, no duplication).

No internal-tool fallback was used. Socraticode, codebase-memory-mcp, and Serena were the primary tools throughout.

## What was already implemented (verified, untouched)
- Route gating: `core.GateModule` (tenant-scoped, 404 when off).
- Admin give/ungive API: `plugin/module-management` (super-admin + org-admin).
- Quotas (orgs/users/WA endpoints/storage): `license.Service.CheckQuotaWithDelta`.
- Frontend: `ModulesView.vue`, `frontend/src/modules/registry.ts`, `services/modules.ts`.

## Genuine gaps filled
**GAP 1 — License tier → module entitlement bridge**
- `internal/core/license_tiers.go` (new): `tierModules` map + `LicenseAllowsModule(tier, key)`. Tiers trial/starter/pro/enterprise; unknown tier = deny-by-default; `"*"` wildcard.
- `internal/core/module_manager.go`: `SetLicenseTierGetter` injection point; `GateModule` checks tier first (404 when unlicensed), falls back to ModuleManager state when tier == "".
- `plugin/module-management/plugin.go`: `licenseAllows` helper gates `updateGlobal`/`updateOrganization` with 403 `module_not_licensed`.
- `cmd/whatomate/main.go`: `core.SetLicenseTierGetter(...)` wired before `SyncManagedModules`.

**GAP 2 — Audit trail for give/ungive**
- `plugin/module-management/audit.go` (new): `ModuleEvent` model (table `module_events`), plugin-owned via `Migrate()`.
- `plugin/module-management/plugin.go`: `recordEvent` on every update attempt; `GET /api/admin/modules/events` + `GET /api/organizations/{id}/modules/events` (super-admin / org-admin gated).

**GAP 3 — Plugin-namespaced RBAC**
- `internal/core/plugin.go`: `PermissionProvidingPlugin` optional interface (mirrors `ManagedPlugin`); `ResolvePlugins` collects; `core.PluginPermissions()` accessor; `core.SyncPluginPermissions(ctx, db)` idempotent seeder (lives in core, not database, to respect the `database → core` cycle).
- `cmd/whatomate/main.go`: `core.SyncPluginPermissions` called after `SyncManagedModules`.
- `plugin/facebook-accounts`: implements `PermissionProvidingPlugin`, declares `plugin.facebook.accounts:pages_manage`; `pageManagementContext` enforces it for connect/disconnect/remove (layered on existing `accounts:write`).

## Files
**New (4)**: `internal/core/license_tiers.go`, `plugin/module-management/audit.go`, `plugin/facebook-accounts/permissions_test.go`, `plugin/module-management/audit_test.go`.
**New tests (1)**: `internal/core/license_tiers_test.go`.
**Modified (6)**: `internal/core/module_manager.go`, `internal/core/plugin.go`, `plugin/module-management/plugin.go`, `plugin/facebook-accounts/plugin.go`, `plugin/facebook-accounts/page_management.go`, `cmd/whatomate/main.go`.
**Extended tests (3)**: `internal/core/module_manager_test.go`, `internal/core/plugin_test.go`, `plugin/module-management/plugin_test.go`.

## NOT touched (deliberate — AGENTS.md core/critical-path rule)
`internal/license/{token,service}.go`, `models.LicenseRecord`, `internal/licenseissuer/`, `internal/licensestudio/`, `internal/handlers/app.go`.

## Verification
- `go build ./...` — PASS
- `go test -p 1 ./internal/core/... ./plugin/module-management/... ./plugin/facebook-accounts/... ./internal/models/... ./internal/license/... ./internal/middleware/...` — ALL PASS
- `golangci-lint run` on all touched packages — CLEAN
- Serena diagnostics on all 8 edited files — CLEAN (1 pre-existing unrelated `maps.Copy` hint in untouched code)
- codebase-memory `detect_changes` — 16 changed files tracked
- Serena memory written: `feature/license-module-bridge-2026-06-22`

## Deferred (separate tasks)
1. Scheduled activation/expiry (req B) — needs `valid_from`/`valid_until` columns + sweeper.
2. Per-deployment plan overrides (req A) — `license_module_overrides` table.
3. Full plugin RBAC rollout to remaining facebook + instagram + whatsapp plugins (mechanical; pattern proven).
4. Extend `SystemRolePermissions()` to auto-include `core.PluginPermissions()` for admin role (needed only when multiple plugins add perms).

