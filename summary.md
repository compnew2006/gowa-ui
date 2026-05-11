# Whatomate Green Deployment Summary

Date: 2026-05-11 03:31:46 EEST / 2026-05-11 00:31:46 UTC
VPS: `31.97.192.53`

## Result

Deployed the current workspace as a green binary while preserving the previous production binary as blue rollback.

- Active path: `/opt/whatomate/bin/whatomate`
- Active target: `/opt/whatomate/bin/whatomate.green.20260511_002922`
- Active version: `Whatomate 6b8628f2-green-20260511_002922 (built 2026-05-11_00:29:59)`
- Green SHA256: `bc57ac0764b7712089f11eeb09e2ac949a8f3f8263479761c9e1abcca876fc9a`
- Blue rollback binary: `/opt/whatomate/bin/whatomate.blue.20260511_002729`
- Blue SHA256: `3533aaf7abbe19de384ca35073f055f9722d90d763e11b59854142575cf0342e`
- Pre-deploy backup: `/root/whatomate_backups/whatomate-green-predeploy-20260510_235254.tar.gz`

## Rollback

One-command rollback:

```bash
ln -sfn /opt/whatomate/bin/whatomate.blue.20260511_002729 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```

## Fix Applied During Deploy

The first green canary failed because production data contains duplicate `whats_app_message_id` values and the new migration attempted to create global unique index `idx_messages_wamid_unique`.

Updated `internal/database/postgres.go` to drop that failed unique index if present and create a non-unique lookup index instead:

- `DROP INDEX IF EXISTS idx_messages_wamid_unique`
- `CREATE INDEX IF NOT EXISTS idx_messages_wamid_lookup ON messages(organization_id, whats_app_message_id) WHERE whats_app_message_id <> ''`

The canary was rolled back to blue immediately, rebuilt with the fix, then redeployed successfully.

## Verification

- Focused backend tests passed:
  - `go test ./internal/database ./internal/handlers ./internal/config ./internal/crypto ./internal/license ./pkg/whatsapp ./pkg/whatsmeow`
- Frontend production build passed locally and on the VPS.
- Services active:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- License bootstrap on ports `18123`, `18124`, `18125`, `18126` returned:
  - `enabled=true`
  - `status=active`
  - `locked=false`
  - `key_id=deploy-20260416`
- HTTPS smoke returned `200`:
  - `https://ofuqalmadenah.com`
  - `https://www.ofuqalmadenah.com`
  - `https://holol-wenjaz.ofuqalmadenah.com`
  - `https://alarkan-almthalia.ofuqalmadenah.com`
  - `https://matbaat-ruya.ofuqalmadenah.com`
- Chrome DevTools browser check:
  - `https://holol-wenjaz.ofuqalmadenah.com/settings/license` redirected to `/login`
  - browser-side `fetch('/api/license/bootstrap')` returned active license data
  - screenshot: `tmp/green-deploy-verify-20260511.png`

Temporary VPS build source and keyring files were removed after deployment.
