# Whatomate Production Info

## Latest Deploy - 2026-06-23 20:30 UTC
- Codebase: `/opt/whatomate-green`
- Binary: `/opt/whatomate-green/whatomate`
- Backup: `/root/backups/whatomate_20260623_193829.tar.gz`

## Services
- `whatomate.service` — main production (active since 2026-06-23 20:30 UTC)
- `whatomate@holol-wenjaz.service` — holol tenant (active since 2026-06-23 20:30 UTC)
- `whatomate-sandbox.service` — sandbox (active)

## Blue/Green
- GREEN (active): `/opt/whatomate-green`
- BLUE (rollback):  `/opt/whatomate-blue`
- CURRENT: `/opt/whatomate-current` -> `/opt/whatomate-green`

## Blue-Green Switch Commands
```bash
# Switch to GREEN (new)
ln -sfn /opt/whatomate-green /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz

# Switch to BLUE (rollback)
ln -sfn /opt/whatomate-blue /opt/whatomate-current && systemctl restart whatomate whatomate@holol-wenjaz

# One-command toggle
if [ "$(readlink -f /opt/whatomate-current)" = "/opt/whatomate-green" ]; then
  ln -sfn /opt/whatomate-blue /opt/whatomate-current
else
  ln -sfn /opt/whatomate-green /opt/whatomate-current
fi
systemctl restart whatomate whatomate@holol-wenjaz
```

## License
- Enabled: true, Status: active, Tier: production, Kind: paid, Duration: lifetime
- Key ID: deploy-20260416
- Key ring: /root/whatomate-keyring.json (embedded in binary at build time)
