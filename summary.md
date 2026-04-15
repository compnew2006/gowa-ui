## Task

Fix the reported SSO security findings in the Go backend:

- Cross-tenant / unbound custom-SSO account takeover
- Custom-provider SSRF through token and userinfo endpoints
- SSO login CSRF / browser session swapping

## Approach And Key Decisions

- Added browser-bound SSO state cookies and PKCE so a callback only completes in the browser that initiated the flow.
- Bound callback login to organization ownership and stored provider identity instead of trusting global email alone.
- Treated custom providers as higher-risk than built-in providers: existing custom-provider users must already be linked to a provider identity before login succeeds.
- Validated custom provider URLs before saving and again before use, while still allowing local callback fixtures in `test` / `development`.
- Forced OAuth token exchange through the app HTTP client context so runtime SSRF protections are applied to both token and userinfo requests.

## Files Modified

- `internal/handlers/sso_handlers.go`
- `internal/handlers/sso_handlers_test.go`
- `internal/handlers/sso_types.go`
- `internal/handlers/sso_utils.go`
- `internal/handlers/testhelpers_test.go`
- `internal/handlers/sso_security.go` (new)

## Dependencies Or Env Changes

- No new dependencies.
- Test helper now sets `App.Environment = "test"` so custom-provider URL validation can allow local mock OAuth servers in tests while production remains locked down.

## Tests Added / Updated

- Updated init tests to assert browser-bound state cookie + PKCE challenge generation.
- Updated callback tests to store browser token / PKCE verifier in Redis state and present the required cookie.
- Added regressions for:
  - mismatched browser state cookie rejection
  - configured HTTP client usage during token exchange
  - rejecting unlinked existing custom-provider users
  - rejecting cross-tenant existing users
  - rejecting private custom-provider URLs in production
- Updated existing custom-provider success coverage to require an already linked identity.

## Verification Results

- `gofmt -w internal/handlers/sso_security.go internal/handlers/sso_types.go internal/handlers/sso_utils.go internal/handlers/sso_handlers.go internal/handlers/testhelpers_test.go internal/handlers/sso_handlers_test.go`
- `go test ./internal/handlers -run 'Test.*SSO'` ✅
- `go test ./internal/handlers -run 'Test.*(SSO|Webhook|Security|Media|Auth|Middleware)'` ✅
- `go test ./internal/handlers` ✅
- `go test ./...` ❌ unchanged pre-existing root build failure:
  - `./tmp_encrypt.go:8:6: main redeclared in this block`
  - `./tmp_arabic.go:8:6: other declaration of main`

## Known Limitations

- Repo-wide `go test ./...` is still blocked by the pre-existing duplicate root helper binaries in `tmp_*.go`.
- Existing custom-provider accounts now need a stored provider binding to log in; this is intentional to prevent arbitrary identity takeover from admin-controlled custom IdPs.

## VPS Deployment - 2026-04-15 22:21:06 UTC

### Task

Deploy the updated `whatomate` build to the Ubuntu amd64 VPS, create a safe backup first, remove stale source code from the server, fix the disabled license system, verify production, and update the deployment records.

### Approach And Key Decisions

- Built the final binary natively on the VPS instead of cross-compiling from the Mac/ARM workstation.
- Stopped the original bad rollout after the first VPS build produced a malformed embedded license key ring and crash-looped `whatomate.service`.
- Recovered service availability first by restoring the last good binary and then moved licensing to a config-based public-key override instead of linker-injecting the key ring.
- Activated the same signed host-bound license token in each production database-backed instance after restarting them onto the corrected binary.
- Removed only stale `whatomate` source/worktree/archive artifacts from the VPS and kept the runtime tree, configs, uploads, databases, docs, and backups.

### Files Changed

- Local docs updated:
  - `docs/whatomate_multi_instances_info.md`
  - `summary.md`
- Remote docs updated:
  - `/root/whatomate_multi_instances_info.md`
  - `/root/whatomate_production_info.md`

### Backup And Deployment Artifacts

- Full pre-deploy backup set: `/root/whatomate_backups/20260415_212640`
- Final runtime backup before cutover: `/opt/whatomate/bin/whatomate.20260415_221226.pre_8dfb206_2210_safe.bak`
- Installed binary: `/opt/whatomate/bin/whatomate`
- Installed version: `Whatomate 8dfb206-worktree-20260415_2210-vps (built 2026-04-15_22:11:29)`
- Installed SHA256: `02999b58c65a130cdd7a1be80689b5b923dccaede692b39ccef9d059031f9da9`

### License Fix Applied

- Production configs now contain:
  - `license.enabled = true`
  - `license.public_key = "Sg7jjcj+DLdw6ogU8gnBmZBh2dqALk88G3QCKfPmmhU="`
  - `license.public_key_kid = "deploy-20260415"`
  - `license.allow_unsafe_public_key_override = true`
- The final binary was rebuilt without overriding `LICENSE_KEY_RING_JSON`, so the embedded key ring stays at the safe default `[]`.
- Activated license token result:
  - all four instances now report `enabled = true`, `status = active`, `locked = false`
  - `license_id = dc245a31-d3d3-4033-bb45-ee9fd9c0c9e1`
  - `key_id = deploy-20260415`

### Verification Results

- VPS runtime:
  - `systemctl is-active` -> `active` for `whatomate`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`
  - listener ports active on `127.0.0.1:18123-18126`
  - `curl http://127.0.0.1:<port>/api/license/bootstrap` -> `status = active` for all four instances
- Public HTTPS:
  - `https://ofuqalmadenah.com/login` -> `200`
  - `https://holol-wenjaz.ofuqalmadenah.com/login` -> `200`
  - `https://alarkan-almthalia.ofuqalmadenah.com/login` -> `200`
  - `https://matbaat-ruya.ofuqalmadenah.com/login` -> `200`
- Browser verification:
  - local Playwright CLI check confirmed the main login page and `holol-wenjaz` tenant login page rendered with email/password fields and a `Sign in` button
  - Chrome DevTools MCP was not configured in this session, so Playwright CLI was used as the available browser automation tool

### Skills And Competencies Applied

- Skills:
  - `master-workflow`
  - `devops-engineer`
  - `playwright`
- Competencies:
  - SSH automation and remote incident recovery
  - Ubuntu `systemd` service rollout and rollback discipline
  - native Go/Vite build execution on linux/amd64
  - production licensing activation and multi-instance validation
  - live HTTP and browser-based smoke verification

## 2026-04-16 Local License Hardening

### Objective

Harden offline licensing in local code so production trusts only embedded public keys, remove the previous production config-override path, and document the correct usage flow.

### Changes

- Production config validation now rejects:
  - `license.public_key`
  - `license.public_key_kid`
  - `license.allow_unsafe_public_key_override = true`
- Non-production config overrides now require explicit opt-in with `license.allow_unsafe_public_key_override = true`.
- Embedded public key rings now support a safer base64 build-time payload via `EmbeddedPublicKeyRingBase64`.
- `Makefile` now supports `LICENSE_KEY_RING_FILE=/absolute/path/keyring.json` for production builds.
- Added focused tests for:
  - production override rejection
  - non-production opt-in behavior
  - base64 embedded key-ring parsing
  - development key-ring verification flow

### Verification

- `PATH=/opt/homebrew/bin:$PATH /opt/homebrew/bin/go test ./internal/license ./internal/config`
  - result: `ok`

### Guide

- Full usage guide created at `docs/offline_license_secure_build_guide.md`
