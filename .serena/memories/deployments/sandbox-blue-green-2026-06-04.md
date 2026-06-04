# Sandbox Blue/Green Deploy - 2026-06-04

## What was deployed
- Source: `agent/facebook-admin-reply-filter@3f31242c` (clean working tree)
- Binary: `/tmp/whatomate-sandbox-green-20260604_010000-linux-amd64` (linux/amd64, 58,683,576 bytes)
- Installed to: `/opt/whatomate/bin/whatomate.sandbox.green.20260604_010000_fb_admin_reply_filter_3f31242c`
- SHA256: `a03a18355403ea2ec01ad58860cd2f461729f878b39746dfac49be14224599cd`
- Version: `sandbox-green-20260604_010000_fb_admin_reply_filter_3f31242c` (built 2026-06-04 01:11:00 UTC)

## Keyring embedded
Kids: `deploy-20260415`, `deploy-20260416`, `vendor-1` (source: `/root/whatomate-keyring.json` on VPS, b64 length 412)

## VPS (31.97.192.53)
- Domain: `https://sandbox.ofuqalmadenah.com`
- Service: `whatomate-sandbox.service` (new PID 2263440, started 2026-06-04 01:12:28 UTC)
- Symlinks:
  - active → `20260604_010000_fb_admin_reply_filter_3f31242c`
  - green  → `20260604_010000_fb_admin_reply_filter_3f31242c`
  - blue   → `20260603_223000_fbcomments_realtime_push_10903_skip` (rollback)

## One-command rollback
```bash
/usr/local/sbin/whatomate-sandbox-switch blue    # back to 20260603_223000
/usr/local/sbin/whatomate-sandbox-switch green   # back to 20260604_010000
/usr/local/sbin/whatomate-sandbox-switch toggle  # flip
/usr/local/sbin/whatomate-sandbox-switch status
```

## Verification
- `curl http://127.0.0.1:18127/api/license/bootstrap` → status=active, key_id=deploy-20260416, hwid=d87d9d77e173
- `https://sandbox.ofuqalmadenah.com/` → 200 (SPA)
- `GET /ws` → 101 upgrade (user 156.207.95.198 in active session)
- `POST /api/facebook/comments/webhook` → 200 (real FB traffic)
- License updated_at = 2026-06-04T01:12:46Z (fresh, confirms new build is serving)

## Build recipe (for future deploys)
```bash
# always cross-compile for VPS (linux/amd64, NOT host arch)
export VERSION="sandbox-green-$(date -u +%Y%m%d_%H%M%S)_<tag>_<gitshort>"
export BUILD_TIME="$(date -u +%Y-%m-%d_%H:%M:%S)"
export LICENSE_KEY_RING_B64="$(cat /tmp/whatomate-keyring.b64)"
env -u GOOS -u GOARCH GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath \
    -ldflags="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME \
              -X github.com/compnew2006/whatomate/internal/license.EmbeddedPublicKeyRingBase64=$LICENSE_KEY_RING_B64" \
    -o /tmp/whatomate-sandbox-green-<timestamp>-linux-amd64 ./cmd/whatomate
file /tmp/whatomate-sandbox-green-*.linux-amd64  # MUST show: ELF 64-bit LSB x86-64
```

## Deploy recipe
1. `scp` binary to VPS at `/opt/whatomate/bin/whatomate.sandbox.green.<tag>_<gitshort>`
2. `ln -sfn <current_green> <blue_symlink>` (rollback target)
3. `ln -sfn <new_green> <active_symlink>` (but service still runs old proc)
4. `systemctl restart whatomate-sandbox.service` ← REQUIRED, switch script does NOT do this
5. Verify new PID + start time + license updated_at

## Lessons
- Build for target arch, not host arch — silent failure if you get it wrong (health check is misleading)
- `whatomate-sandbox-switch` updates symlinks but does NOT restart the service
- Always verify the running PID changed AND the license bootstrap `updated_at` is fresh

## Files updated
- `/root/whatomate_multi_instances_info.md` (VPS)
- `/root/whatomate_production_info.md` (VPS)
- `docs/whatomate_multi_instances_info.md` (local)
- `summary.md` (local)
