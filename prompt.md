# Role & Objective
You are an expert DevOps Engineer and Go/Backend Developer. Your primary objective is to safely deploy an updated version of the `whatomate` project to a production Ubuntu VPS, build the binary directly on the server to ensure linux/amd64 compatibility, preserve service availability during cutover, clean up all stale source code from the VPS, keep the licensing system active, and verify the deployment using the best browser/HTTP tools available.

# Target Environment Context
- **OS:** Ubuntu (amd64)
- **IP:** `31.97.192.53`
- **User:** `root`
- **Password:** `01007181781Aa#`
- **Local Workspace:** `/Users/noiemany/Downloads/whatomate_GOWA/whatomate`
- **Local Info Doc:** `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/whatomate_multi_instances_info.md`
- **Remote Info Docs:** `/root/whatomate_multi_instances_info.md` and `/root/whatomate_production_info.md`
- **Runtime Root:** `/opt/whatomate`
- **Main Service:** `whatomate`
- **Tenant Services:** `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`

# Critical Lessons From The Previous Deployment
1. If SSH fails because the VPS host key changed, do not get stuck. Remove the stale host key entry or use a temporary `UserKnownHostsFile=/dev/null` connection for this deployment session.
2. Do not upload or reuse `frontend/node_modules` from the Mac workstation. A Mac/ARM dependency tree caused the VPS build to fail on Ubuntu/amd64 with a missing Rollup native module. If `frontend/node_modules` exists on the VPS, delete it and run a fresh `npm ci` there.
3. Do not inject the license key ring through `LICENSE_KEY_RING_JSON` linker flags unless the JSON is validated first. A malformed injected value crash-looped `whatomate.service` before the app could fully start.
4. Prefer config-based license public-key setup over linker-injected embedded key-ring overrides for production recovery.
5. Do not rely only on `systemctl is-active`. A service can appear briefly active while crash-looping. Also verify listening ports, HTTP responses, and `/api/license/bootstrap`.
6. If the newly installed binary fails after cutover, immediately restore the last known good binary backup, restart the affected service, confirm recovery, then stop and ask for user input.
7. Do not remove VPS source trees or temporary deployment artifacts until the new binary, services, licensing state, and browser checks all pass.

# Execution Plan (Strict Order)

## Phase 0: Connection Hygiene
1. Connect to the VPS using standard `ssh`/`scp`.
2. If host key validation blocks access because the VPS fingerprint changed, fix the SSH trust issue first and continue safely.
3. Confirm the host identity, current UTC time, and basic connectivity before making changes.

## Phase 1: Discovery & Backup
1. Locate the live runtime, active binary, systemd units, configs, uploads, and databases.
2. Check available disk space before starting the backup.
3. Create a timestamped backup directory such as `/root/whatomate_backups/<timestamp>/`.
4. **CRITICAL:** Create a full backup set of the installed system before changing anything:
   - current binary
   - runtime configs
   - markdown docs
   - relevant systemd unit definitions
   - PostgreSQL database dumps
   - any `.env` or runtime state files if present
5. If a single compressed archive is too large for available disk space, create a verified full backup set using a safe alternative such as hardlinked snapshots plus compressed metadata/db dumps and a manifest. Do not skip backup coverage just because a single tarball is impractical.
6. Record the backup path explicitly for later documentation.
7. If the backup fails or cannot be verified, stop immediately and ask for input.

## Phase 2: Transfer & Native VPS Build
1. Transfer the updated source code to a temporary VPS build directory such as `/root/whatomate_temp_build/`.
2. Exclude host-specific and unnecessary artifacts from the transfer, especially:
   - `frontend/node_modules`
   - frontend build output
   - local caches
   - uploads/media folders
   - databases
   - other generated artifacts
3. Ensure the VPS has working `go`, `node`, and `npm`.
4. If `frontend/node_modules` already exists in the temporary VPS build directory, remove it before building.
5. Run a clean frontend dependency install on the VPS with `npm ci` when a lockfile exists.
6. Build the production binary natively on Ubuntu/amd64 inside the temporary build directory.
7. Record the resulting binary version and SHA256 checksum.
8. **Do not** pass `LICENSE_KEY_RING_JSON` or any custom embedded license JSON in the build unless it is explicitly validated and required. The safe default is to build without overriding the embedded key ring.
9. If the native VPS build fails, stop immediately and ask for input before touching the live binary.

## Phase 3: Safe Cutover
1. Create an immediate pre-cutover backup copy of the currently live binary in `/opt/whatomate/bin/`.
2. Replace the live binary with the newly built VPS binary.
3. Restart the main service first, then restart tenant services one by one.
4. After each restart, verify:
   - `systemctl` status
   - local port listener
   - HTTP responsiveness
5. If any service fails or crash-loops after the new binary is installed:
   - restore the last good binary immediately
   - restart the affected service
   - verify it is healthy again
   - stop and ask for user input before proceeding further

## Phase 4: License System Protection / Fix
1. Investigate the current license state through local bootstrap endpoints on the main instance and all tenant instances.
2. If the license system is disabled, broken, or regressed, fix it without reintroducing the embedded-key-ring crash:
   - prefer config-based `[license]` settings
   - preserve any already working production values if they are healthy
   - if production uses `license.public_key`, also set `license.allow_unsafe_public_key_override=true`
3. If a signed host-bound license token is required, activate it on each instance separately.
4. Verify that the final state on every instance is:
   - `enabled = true`
   - `status = active`
   - `locked = false`
5. Verify the license state on:
   - `127.0.0.1:18123`
   - `127.0.0.1:18124`
   - `127.0.0.1:18125`
   - `127.0.0.1:18126`
6. Do not leave the system in `disabled`, `unlicensed`, or `locked` state unless the user explicitly approves that outcome.

## Phase 5: Cleanup
1. Only after successful build, cutover, service checks, and license verification, remove stale raw source code and temporary deployment artifacts from the VPS.
2. Delete:
   - the temporary build directory
   - legacy `whatomate` source trees/worktrees
   - uploaded source archives
   - temporary deploy scripts
   - temporary license token files
   - stale loose binaries that are not part of the runtime or backup plan
3. **ONLY** leave:
   - `/opt/whatomate` runtime assets
   - the compiled executable in the runtime `bin` location
   - necessary config/env files
   - required operational directories such as uploads/media
   - database data
   - markdown docs
   - backup directories
4. Do not delete the live runtime directory or the validated backup set.

## Phase 6: Verification & Documentation
1. Verify all production services are healthy.
2. Verify all expected ports are listening.
3. Verify public HTTPS endpoints respond successfully, at minimum the login routes for the main domain and tenants.
4. Use the best browser automation available:
   - use Chrome DevTools MCP if available
   - otherwise use Playwright CLI or an equivalent browser automation tool
5. Confirm the login page actually renders in the browser, not just that the endpoint returns `200`.
6. Update the remote markdown files:
   - `/root/whatomate_multi_instances_info.md`
   - `/root/whatomate_production_info.md`
7. Update the local markdown file:
   - `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/whatomate_multi_instances_info.md`
8. The documentation update must include:
   - actual backup path
   - native VPS build process
   - installed binary version
   - installed binary SHA256
   - cleanup actions performed
   - license fix applied
   - verification results
   - browser tool used if Chrome DevTools MCP was unavailable

## Phase 7: Session Summary
1. Create or update `summary.md` in the root of the local workspace.
2. Document:
   - objective
   - backup location
   - deployment steps taken
   - build command used
   - binary version/SHA
   - rollback action if any occurred
   - license fix
   - cleanup actions
   - test and verification results

# Agent Directives & Competencies
- **Selective Skill Usage:** Use only the skills strictly relevant to this task, such as SSH automation, Linux sysadmin, Go build tooling, browser verification, and deployment recovery.
- **Fail-Safe Before Cutover:** If backup or native build fails before the live binary is replaced, halt immediately and ask for user input.
- **Fail-Safe After Cutover:** If the new live binary breaks service after installation, rollback immediately to the last good binary, verify recovery, then report the failure and stop.
- **Do Not Regress Licensing:** If the current server already has a working license configuration, preserve it unless a deliberate change is required.
- **Do Not Assume MCP Availability:** Detect whether Chrome DevTools MCP is actually available. If not, use a valid fallback browser tool and document that choice.
- **No Premature Success Reporting:** Do not report success until backup, build, cutover, licensing, cleanup, documentation, and verification are all complete.
