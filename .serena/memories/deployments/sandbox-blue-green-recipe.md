# Sandbox Blue/Green Deploy — Build & Verification Recipe

## Build (host = darwin/arm64, target = linux/amd64)

```bash
# CRITICAL: never let host GOOS/GOARCH leak into the cross-compile
env -u GOOS -u GOARCH \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath \
    -ldflags="-s -w \
      -X main.Version=<version-string> \
      -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
      -X github.com/compnew2006/whatomate/internal/license.EmbeddedPublicKeyRingBase64=$(cat /tmp/whatomate-keyring.b64)" \
    -o /tmp/whatomate-sandbox-green-<version>-linux-amd64 \
    ./cmd/whatomate

# Verify (must be ELF, not Mach-O)
file /tmp/whatomate-sandbox-green-<version>-linux-amd64
sha256sum /tmp/whatomate-sandbox-green-<version>-linux-amd64
```

Version string convention: `<purpose>-<UTC-YYYYMMDD_HHMMSS>-<git-short-sha>`.
e.g. `fb-admin-reply-filter-20260604_010000-3f31242c` or `comments-scroll-fix-20260604_013200-3f31242c`.

## Frontend embedding (CRITICAL — easy to forget)

```bash
cd frontend && npm run build && cd ..
rm -rf internal/frontend/dist/*
cp -r frontend/dist/* internal/frontend/dist/
# Then build the Go binary (above)
```

The Makefile's `make build-prod` chains `frontend-build` → `embed-frontend` → `go build`.
If you call `go build` directly after only `npm run build`, the resulting binary still
embeds the OLD frontend — symptoms: API/version is fresh, SPA bundle hashes are old.

## Deploy to VPS (31.97.192.53, sshpass)

```bash
VERSION="<version>"
BIN="/tmp/whatomate-sandbox-green-${VERSION}-linux-amd64"

# 1. Upload
sshpass -p '01007181781Aa#' scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  "$BIN" "root@31.97.192.53:/tmp/${VERSION}-linux-amd64"

# 2. Install + repoint
sshpass -p '01007181781Aa#' ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@31.97.192.53 "
  set -e
  cd /opt/whatomate/bin
  cp -f /tmp/${VERSION}-linux-amd64 whatomate.sandbox.${VERSION}
  ln -sfn whatomate.sandbox.${VERSION} whatomate.sandbox.green
  ln -sfn whatomate.sandbox.${VERSION} whatomate.sandbox.active
  rm -f /tmp/${VERSION}-linux-amd64
  systemctl restart whatomate-sandbox.service
  sleep 4
  systemctl status whatomate-sandbox.service --no-pager | head -7
"
```

## Verification (post-restart)

```bash
# 1. New PID + start time
ssh root@31.97.192.53 'systemctl status whatomate-sandbox.service --no-pager | head -7'
# Expect: Main PID: <new> ... Active: active (running) since <new-time>

# 2. License bootstrap is fresh (updated_at near "now")
curl -sS https://sandbox.ofuqalmadenah.com/api/license/bootstrap | python3 -c "
import json,sys
d=json.load(sys.stdin)['data']
print(f'tier={d[\"tier\"]} key_id={d[\"key_id\"]} status={d[\"status\"]} updated_at={d[\"updated_at\"]}')"

# 3. Frontend bundle is the NEW hash (not cached CDN)
curl -sS https://sandbox.ofuqalmadenah.com/ | grep -oE 'assets/index-[A-Za-z0-9_-]+\.css'
# Then verify the new CSS contains the expected class:
curl -sS https://sandbox.ofuqalmadenah.com/assets/<hash>.css | grep -oE '<expected-class-rule>'
```

## Lessons (from 2026-06-04 deploys)

1. **`GOOS`/`GOARCH` leak.** Without `env -u GOOS -u GOARCH`, the host (darwin/arm64) leaks and the binary is Mach-O, not ELF. The first build on 2026-06-04 produced this; the switch script's health check still passed because the old service was still running. Caught by `file` post-mortem.
2. **Switch script does NOT restart.** `whatomate-sandbox-switch {green|blue|toggle}` only flips symlinks. You must `systemctl restart whatomate-sandbox.service` to load the new binary. The script's `curl /api/license/bootstrap` does NOT catch this — it polls the still-running old process.
3. **Health check is misleading.** `status=active` from the switch script proves the OLD process is healthy. Verify the NEW build is serving by checking `systemctl status` for new PID + start time, and license `updated_at` freshness.
4. **Embedded frontend is part of the Go build.** Changing only the frontend still requires `cp frontend/dist/* internal/frontend/dist/` and a fresh `go build`. `//go:embed` won't pick up the new dist otherwise.
5. **Blue symlink shape.** When using `ln -sfn <relative> whatomate.sandbox.blue`, the target must be a sibling filename (e.g. `whatomate.sandbox.green.X`), NOT an absolute path. If you pass an absolute path, the symlink becomes a malformed `whatomate.sandbox.<absolute-path>`. Always use the same naming convention as the existing chains.
6. **Bundle hash verification.** The `file` check catches arch errors. The CSS/JS bundle hash check (`/assets/index-*.css`) catches "wrong dist embedded" errors. Both are needed for confidence in a frontend-only deploy.

## Symlink chain (post 2026-06-04 01:36 UTC)

- `whatomate.sandbox.active` → `whatomate.sandbox.comments-scroll-fix-20260604_013200-3f31242c` (current)
- `whatomate.sandbox.green` → `whatomate.sandbox.comments-scroll-fix-20260604_013200-3f31242c` (current)
- `whatomate.sandbox.blue` → `whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c` (rollback)

## Rollback command

```bash
ssh root@31.97.192.53 'whatomate-sandbox-switch blue && systemctl restart whatomate-sandbox.service'
```

The `.blue` symlink always points to the previous known-good build. One command + restart = rollback.

## Future improvements

- [ ] Update `whatomate-sandbox-switch` script to verify post-restart: new PID + license `updated_at` freshness, exit non-zero if mismatch.
- [ ] Update `whatomate-sandbox-switch` to fail loudly if the target symlink chain is broken (detect the `whatomate.sandbox./opt/...` malformed-link case from 2026-06-04).
- [ ] Add a `make deploy-sandbox VERSION=…` target that wraps the full build+embed+scp+restart+verify cycle.
- [ ] Track deploy history in a structured file (e.g. `deploys.json`) instead of appending markdown to two long-running doc files.
