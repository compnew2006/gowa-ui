# Whatomate Multi-Instances Info

## Latest Production Deploy - 2026-06-23 20:30 UTC
- VPS: `31.97.192.53` | Domain: `ofuqalmadenah.com`
- Deployed codebase: `/opt/whatomate-green`
- Production green (ACTIVE): `/opt/whatomate-green/whatomate`
- Production blue (rollback): `/opt/whatomate-blue/whatomate.license_tier-20260623`
- Current Symlink: `/opt/whatomate-current` -> `/opt/whatomate-green`
- License: ✅ enabled=true, status=active, locked=false, kind=paid, tier=production, lifetime, key_id=deploy-20260416
- Pre-deploy backup: `/root/backups/whatomate_20260623_193829.tar.gz`

### One-command switch (blue/green)
```bash
# Switch to GREEN (new)
ln -sfn /opt/whatomate-green /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz

# Switch to BLUE (rollback)
ln -sfn /opt/whatomate-blue /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz
```

### Changes in this deploy
- Fixed frontend websocket connection logic to retry indefinitely with exponential backoff, preventing UI desyncs for active clients (such as the 5:29 PM/17:29 freeze).
- Resolved license service boot crash on VPS by recompiling with embedded license keyring JSON file `/root/whatomate-keyring.json` locally.
- Deployed update to GREEN and restarted both `whatomate` and `whatomate@holol-wenjaz` services.

### Verification (2026-06-23 20:31 UTC)
- `whatomate.service` active and healthy.
- `whatomate@holol-wenjaz.service` active and healthy.
- License bootstrap status: enabled=true, status=active, key_id=deploy-20260416.
- `/health` → 200 OK for both main (port 18123) and holol-wenjaz (port 18124).
