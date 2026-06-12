# Whatomate Multi-Instances Info

Canonical local deployment notes are kept in `docs/MULTI_INSTANCES_DEPLOY_INFO.md`.

Latest deployment recorded here: sandbox green replacement on 2026-06-11 20:06 UTC.

- VPS: `31.97.192.53`
- Backup: `/root/whatomate_backups/whatomate-green-predeploy-20260611_195937.tar.gz`
- Backup SHA256: `1f156804b95bc7ef324a94facf37862f2fc7a1215b6e6ac8c956755671a32567`
- New sandbox green: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
- New sandbox green SHA256: `24110198b9da7caae06d5bbb6a16738ad24da5589e7f3e1bb62c3861189c31df`
- Current sandbox active: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
- Current sandbox blue rollback: `/opt/whatomate/bin/whatomate.sandbox.comments-scroll-fix-20260604_013200-3f31242c`
- Public live was left unchanged: `/opt/whatomate/bin/whatomate.green.20260528_111523`
- License bootstrap verified active on sandbox and public live.

Promote sandbox green to public live:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback public live:

```bash
ln -sfn /opt/whatomate/bin/whatomate.green.20260528_111523 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback sandbox:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.blue /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```
