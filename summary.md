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
