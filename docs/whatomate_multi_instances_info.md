# Whatomate Multi-Instances Info

## Latest Production Deploy - 2026-06-20 10:40 UTC
- VPS: `31.97.192.53` | Domain: `ofuqalmadenah.com`
- Deployed codebase: `/opt/whatomate-green`
- Production green (ACTIVE): `/opt/whatomate-green/whatomate`
- Production blue (rollback): `/opt/whatomate-blue/whatomate`
- Current Symlink: `/opt/whatomate-current` -> `/opt/whatomate-green`
- License: ✅ enabled=true, status=active, locked=false, kind=paid, tier=production, lifetime, key_id=deploy-20260416
- Pre-deploy backup: `/root/backups/whatomate_20260620_102348.tar.gz`

### One-command switch (blue/green)
```bash
# Switch to GREEN (new)
ln -sfn /opt/whatomate-green /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz

# Switch to BLUE (rollback)
ln -sfn /opt/whatomate-blue /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz
```

### Changes in this deploy
- Deploying the local Whatomate codebase as a green update.
- Single binary production build with embedded frontend compiled on the VPS.
- Integrated public key keyring for licensing verified at build time.

### Verification (2026-06-20 10:40 UTC)
- `whatomate.service` active and healthy.
- `whatomate@holol-wenjaz.service` active and healthy.
- License: enabled=true, status=active.
- `/health` → 200 OK.
