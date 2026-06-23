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

---

# Unblock manager.go in Global Serena Config — 2026-06-23

## Task
Remove the `**/pkg/**` ignore pattern from the global Serena configuration (`/Users/noiemany/.serena/serena_config.yml`) so that `manager.go` and other files under `pkg/` are no longer ignored/blocked.

## Files Changed
- [/Users/noiemany/.serena/serena_config.yml](file:///Users/noiemany/.serena/serena_config.yml)

## Approach
Removed `- "**/pkg/**"` from the `ignored_paths` list in `/Users/noiemany/.serena/serena_config.yml`. This unblocks all files under `pkg/` (such as `manager.go` in `pkg/whatsmeow/manager.go`) from being ignored by Serena.

## Verification
- Verified by checking the file content to confirm the line is successfully removed.

---

# Code Review & Quality Gates Skill Update — 2026-06-23

## Task
1. Replace generic "Test Quality Gate" references in `mcp-code-operations` skill config with explicit gates referencing the local `clean-code-guard` and `test-guard` skills.
2. Review the `force_ipv4` implementation and testing code to ensure no gaps or defects exist.

## Files Changed
- [/.agent/skills/mcp-code-operations/SKILL.md](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/.agent/skills/mcp-code-operations/SKILL.md)
- [summary.md](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/summary.md)

## Approach
- **Gate config update**: Replaced Step 5 in the verification checklist of [SKILL.md](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/.agent/skills/mcp-code-operations/SKILL.md) with two separate quality gates pointing to the absolute paths of the local [clean-code-guard](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/.agent/skills/clean-code-guard/SKILL.md) and [test-guard](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/.agent/skills/test-guard/SKILL.md) skills.
- **Code review**: Audited the `force_ipv4` code changes in [manager.go](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/pkg/whatsmeow/manager.go) and [config.go](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/config/config.go), and the unit tests in [force_ipv4_test.go](file:///Users/noiemany/Downloads/whatomate_GOWA/whatomate/pkg/whatsmeow/force_ipv4_test.go) against the standard clean code & test rules.

## Verification
- Ran the full test suite (`make test`): All Go tests (including the new force IPv4 dialing tests) passed successfully.
- Ran the linter (`golangci-lint`): The modified packages (`whatsmeow` and `config`) are completely lint-clean.


---

# GREEN Deploy Prep + Runbook — 2026-06-23 (force_ipv4)

## ⚠️ Scope correction (do NOT skip)

The original deploy task contained **dangerous contradictions** with the real production topology. Before executing anything, verify against your actual VPS:

- **Process manager is systemd, NOT pm2.** There are **4 systemd units**: `whatomate` + `whatomate@holol-wenjaz` + `whatomate@alarkan-almthalia` + `whatomate@matbaat-ruya`. Any `pm2 restart all` in a prior task is wrong and does nothing.
- **Do NOT wipe the codebase.** Go single-binary; the binary is self-contained. The codebase is only needed for VPS rebuilds. Wiping `/opt/whatomate-green` while `/opt/whatomate-current` symlinks into it takes all 4 tenants down.
- **Binary dir is `/opt/whatomate/bin/`, not `/root/bin`.** Your own docs (`docs/whatomate_multi_instances_info.md`) confirm this.
- **License was active per docs** (`enabled=true, status=active, tier=production, lifetime, key_id=deploy-20260416`). If it now shows Disabled, **diagnose before forcing enable** — a forced enable on a wiped record locks all routes except `/activate`.
- **Rotate the root password** — it was pasted in plaintext into the task prompt and is now in this session transcript.

I did **not** SSH to the VPS or run destructive commands. This is local prep + a verified binary + an accurate runbook for YOU to execute.

## What shipped (this branch)

**Branch**: `deploy/green-20260623-ipv4` (local only, not pushed)
**Commit**: `8d4f047c` — `feat(whatsmeow): add force_ipv4 config to pin clients to IPv4-only dialing`
**Prior commits already on main** (included in this binary): `59580e76` (plugin/module system), `d5ce1326` (async events docs/tests).

### Changes in this deploy
- `internal/config/config.go` — `WhatsmeowConfig.ForceIPv4 *bool`
- `pkg/whatsmeow/manager.go` — `whatsmeowForceIPv4()` + `buildIPv4HTTPClient()`; `newClient()` applies `Set{Websocket,Media,PreLogin}HTTPClient` when `force_ipv4=true`
- `config.example.toml` — documented under `[whatsmeow]`
- `pkg/whatsmeow/force_ipv4_test.go` — 4 tests (reader nil-safety, tcp4 IPv6-literal rejection, HTTP/2 retained, TLS 1.2 min)

## Local verification (all green)

- `make test` — **ALL PASS** (every Go package; some Facebook-API tests SKIP as expected). Includes module-management + plugin-permission tests from earlier session work.
- `golangci-lint run ./pkg/whatsmeow/... ./internal/config/...` — CLEAN
- `make build-prod` — built 56MB embedded-frontend binary

## Binary identification

| Field | Value |
|---|---|
| Local path | `./whatomate` |
| Size | 56 MB |
| SHA256 | `4a158510ff49df0ac0749c3cb1d8176f4ad3c793ff598f895f995c4642b28e0e` |
| Version string | `Whatomate 8d4f047c-dirty (built 2026-06-22_22:32:34)` |
| Git commit | `8d4f047c7d5ec6173721d03655803d70e61a3acc` |
| Git branch | `deploy/green-20260623-ipv4` |

(`-dirty` is because `summary.md` is uncommitted; the code itself is the clean commit `8d4f047c`.)

---

# 🚀 Operator Runbook — execute on the VPS yourself

All commands run as `root` on `31.97.192.53`. **Read the whole runbook once before executing.**

## STEP 0 — Recon (read-only, do this first)

```bash
# Confirm the topology matches this runbook before touching anything.
systemctl list-units --type=service | grep whatomate      # expect 4 units
readlink /opt/whatomate-current                            # expect -> /opt/whatomate-green
ls -la /opt/whatomate-green/bin/ /opt/whatomate-blue/bin/  # both should exist
cat /opt/whatomate/config.toml | grep -iE "license|tier"   # capture current license state
```

If any of these disagree with the comments, **stop and reconcile** — the rest of the runbook assumes the topology above.

## STEP 1 — Backup the current active deployment

```bash
TS=$(date +%Y%m%d_%H%M%S)
mkdir -p /root/backups
# Full codebase + binary + config backup
tar -czf /root/backups/whatomate_${TS}.tar.gz \
  -C /opt whatomate-green whatomate-blue whatomate-current 2>/dev/null
# Also snapshot the DB (Postgres) — adjust DB name/user if different
sudo -u postgres pg_dump whatomate | gzip > /root/backups/whatomate_db_${TS}.sql.gz
ls -lh /root/backups/whatomate_${TS}.tar.gz /root/backups/whatomate_db_${TS}.sql.gz
```
**Record these paths** — they're your rollback anchor.

## STEP 2 — Upload the new binary to the VPS

From your **local** machine (NOT the VPS):
```bash
cd /Users/noiemany/Downloads/whatomate_GOWA/whatomate
# Tag the upload with the commit so you can tell binaries apart
scp whatomate root@31.97.192.53:/opt/whatomate-green/bin/whatomate.green.8d4f047c
```

Then **on the VPS**, verify integrity:
```bash
sha256sum /opt/whatomate-green/bin/whatomate.green.8d4f047c
# MUST equal: 4a158510ff49df0ac0749c3cb1d8176f4ad3c793ff598f895f995c4642b28e0e
chmod +x /opt/whatomate-green/bin/whatomate.green.8d4f047c
```

## STEP 3 — Update config.toml (enable force_ipv4 on this deployment only)

On the VPS, edit `/opt/whatomate/config.toml` (or wherever the active config lives — confirm in STEP 0):

```toml
[whatsmeow]
# ... existing keys ...
force_ipv4 = true        # ← ADD THIS. The Hostinger->face:b00c IPv6 path is flaky.
```

Leave it `false` (or absent) on any deployment that doesn't have the IPv6 problem.

## STEP 4 — License system (DIAGNOSE, do not blindly enable)

Your docs say the license should be `enabled=true, status=active, tier=production`. If STEP 0 showed it Disabled:

```bash
# Check the license table state — do NOT enable yet, understand first
sudo -u postgres psql whatomate -c "SELECT id, status, tier, enabled FROM license_records ORDER BY id DESC LIMIT 5;"
# Check the config knob
grep -A3 "\[license\]" /opt/whatomate/config.toml
```

If `license_records` is empty → the record was wiped; restoring requires the original activation token (check `/root/backups/` from prior deploys). **Do not set `enabled=true` on an empty record** — it will lock the app to `/activate` only.

If `license_records` has a row but `[license] enabled = false` in config → flip to `enabled = true` in config.toml. That's the safe case.

If you're unsure, **leave license disabled and ask** — a locked production is worse than a disabled license.

## STEP 5 — BLUE/GREEN switch (the actual cutover)

This swaps the active symlink and restarts **all 4 systemd units** (not pm2):

```bash
# Point current at the new green binary location and atomically restart all tenants
ln -sfn /opt/whatomate-green/bin/whatomate.green.8d4f047c /opt/whatomate-green/bin/whatomate && \
systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya && \
sleep 3 && \
systemctl is-active whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```
Expected output: four lines of `active`. Anything else → go to ROLLBACK immediately.

### One-command switch (for future use)

**To GREEN (this deploy):**
```bash
ln -sfn /opt/whatomate-green/bin/whatomate.green.8d4f047c /opt/whatomate-green/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```

**To BLUE (rollback to previous):**
```bash
ln -sfn /opt/whatomate-blue/bin/whatomate /opt/whatomate-green/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```
*(Adjust the blue binary path to whatever STEP 1 backed up — check `/opt/whatomate-blue/bin/` for the prior binary name.)*

## STEP 6 — Verify (browser + API)

```bash
# Local listeners (should all return 200)
for p in 18123 18124 18125 18126; do
  echo -n "127.0.0.1:$p -> "; curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:$p/health
done

# Version confirms the new binary is live
curl -s http://127.0.0.1:18123/api/license/bootstrap 2>/dev/null | head -c 200; echo
```

Then in a browser, open **https://ofuqalmadenah.com/login** and confirm:
- Login page renders (title `Whatomate`, visible `Sign in` button)
- Check the three tenant subdomains too: `holol-wenjaz.ofuqalmadenah.com`, `alarkan-almthalia.ofuqalmadenah.com`, `matbaat-ruya.ofuqalmadenah.com`
- As super-admin, visit `/settings` (or wherever the license banner is) and confirm license state matches what you expect from STEP 4

If browser verification is unavailable, the curl checks above are the minimum gate.

## ROLLBACK (if any step fails)

```bash
# 1. Revert the binary symlink to the backup
ln -sfn /opt/whatomate-blue/bin/whatomate /opt/whatomate-green/bin/whatomate
# 2. Restart all 4 units
systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
# 3. (If you changed config.toml in STEP 3) revert force_ipv4 to false
# 4. (If STEP 4 changed license config) revert that too
# 5. Confirm rollback via the STEP 6 checks
```
Worst case, restore from `/root/backups/whatomate_<TS>.tar.gz` (STEP 1).

## STEP 7 — Update the info docs (after successful verify)

Append to `/root/whatomate_multi_instances_info.md` and `/root/whatomate_production_info.md` on the VPS, and to the local `docs/whatomate_multi_instances_info.md`:
- Deploy timestamp
- Branch `deploy/green-20260623-ipv4`, commit `8d4f047c`
- Binary SHA256 `4a158510...2b28e0e`
- `force_ipv4 = true` set
- Verification results (the 4 `active` lines + curl 200s)
- Backup path from STEP 1

---

## Post-deploy monitoring (24h)

The whole point of `force_ipv4` is to eliminate the 58 TCP resets/day on the IPv6 path. After 24h:
```bash
# Whatever you used to measure the 58 resets — re-run it.
# Expect: 0 resets on the IPv6 path (because it's no longer used).
```
If IPv4 connectivity has any issues, flip `force_ipv4 = false` in config.toml and restart — no rebuild needed for that rollback.

## Git state (local, unchanged by this prep)

- Branch `deploy/green-20260623-ipv4` exists locally, **not pushed**.
- `main` is untouched.
- `summary.md` has this entry appended (uncommitted, intentional).
- The force_ipv4 commit `8d4f047c` is the only new commit on the deploy branch.

When you've confirmed the deploy is healthy in production for a few days, you can merge `deploy/green-20260623-ipv4` into `main` and push — or cherry-pick `8d4f047c` onto main. Either is fine; I did not push per AGENTS.md §6.

---

## 2026-06-23 — Permanent fix: WhatsApp "server returned error 400" on send + whatsmeow dep bump

**Investigation (read-only) first.** `failed to send text message: server returned error 400` is
whatsmeow's native stanza-ack error (NOT a Meta Cloud API HTTP 400), returned by `client.SendMessage`
when the recipient's Signal Protocol session is desynced — a PN→LID migration gap. Log signature:
`[Database DEBUG] No sessions or sender keys found to migrate from <PN> to <LID>_1` immediately
before each 400. The prior "fix" (commit c1e34cd, switching text to `Conversation` protobuf) only
touched the payload layer and did NOT address the encryption-session desync — proven by live 400s
still occurring at 08:00 UTC today on the current binary (instance 6129b735, org cd0fa895,
contact 966531536800).

**Root cause:** the send queue retried 3× but reused the same stale local session store each time,
so every retry failed identically. Missing piece = a scoped session reset between retries.

**Fix (deployed green-20260623_session_reset-20260623_091904):** per-recipient Signal session reset
on the 400 class, fired once before the first retry.

Files:
- `pkg/whatsmeow/session_reset.go` (NEW) — `ResetRecipientSession` deletes all device sessions for
  the recipient + its PN↔LID counterpart via `client.Store.Sessions.DeleteAllSessions(ctx, phone)`,
  where `phone = jid.SignalAddressUser()` (agent-suffixed for LIDs). Panic-safe LID lookup. Group
  JIDs no-op. Implements `provider.SessionResetter`.
- `pkg/whatsmeow/queue.go` — `QueuedSend{Run, OnRetryReset}` + `EnqueueSend`; reset fires once before
  first retry only on `isSessionDesyncError`. `Enqueue` kept for back-compat.
- `pkg/whatsmeow/send_error_classification.go` — `isSessionDesyncError()` (400-class only).
- `pkg/provider/interface.go` — `SessionResetter` optional interface.
- `internal/handlers/messages.go` — single Enqueue site wires reset via `SessionResetter` type-assert.
- `pkg/whatsmeow/media_service.go` — local `errFileLengthMismatch` sentinel (upstream one deprecated by bump).
- `pkg/whatsmeow/session_reset_test.go` (NEW, 9 tests) + `queue_test.go` (+3 tests).

**Dependency bump:** go.mau.fi/whatsmeow v0.0.0-20260414172242-d4ffc1df2442 → v0.0.0-20260622185415-5f04eac6dbbb.
Session/LID/SenderKey API stable across the bump (identical line numbers). 8 transitive bumps.

**Verification (full gate):** `go test ./...` green · `go vet` clean · `golangci-lint` clean on all
changed files (SA1019 regression from bump resolved via local sentinel) · `make build-prod` 60M binary.

**Deploy:** blue/green on VPS. Blue seeded with pre-fix binary (/opt/whatomate-blue/whatomate.rollback-20260623)
for one-command rollback (`whatomate-switch blue`). rsync'd source to /opt/whatomate-green, rebuilt
node_modules via `npm ci` (mac rollup binary had contaminated node_modules — exclude node_modules
from rsync in future), built with GOTOOLCHAIN=auto + LICENSE_KEY_RING_FILE, `whatomate-switch green`.

**Post-deploy (10 min observation):** 0 panics/fatals · 0 400 errors (down from ~8/5min) · both
services active · HTTPS login 200 on ofuqalmadenah + holol-wenjaz · instance 6129b735 reconnected.
Reset code path dormant (no 400 has occurred since restart to exercise it); mechanism proven by
unit test TestQueueResetHookFiresOnceBeforeFirstRetryOn400. Did NOT trigger a real customer send
to force-prove recovery on contact 966531536800 — needs explicit approval.

**Unrelated finding:** tenants alarkan-almthalia + matbaat-ruya crash-loop with systemd status
226/NAMESPACE — missing /opt/whatomate/instances/<tenant> dir. Deploy/setup issue, not code. Separate fixup.

**Not committed** per AGENTS.md §6. Changes are on working tree (uncommitted) + deployed via blue/green.
Rollback anytime with `whatomate-switch blue` on the VPS.
