# Session Summery

## VPS sandbox green deploy - 2026-06-11

- Deployed current project revision `5702241f` as the new sandbox green runtime on VPS `31.97.192.53`.
- Preserved the public live/blue runtime for users: `/opt/whatomate/bin/whatomate` still points to `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Created pre-deploy backup: `/root/whatomate_backups/whatomate-green-predeploy-20260611_195937.tar.gz`.
- Backup SHA256: `1f156804b95bc7ef324a94facf37862f2fc7a1215b6e6ac8c956755671a32567`.
- New green binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`.
- New green SHA256: `24110198b9da7caae06d5bbb6a16738ad24da5589e7f3e1bb62c3861189c31df`.
- License bootstrap verified active on sandbox and public live: `enabled=true`, `status=active`, `tier=production`, `key_id=deploy-20260416`.
- Browser QA via Chrome DevTools loaded `https://sandbox.ofuqalmadenah.com/login`, found no console errors, and confirmed `/api/license/bootstrap` returned active license data.
- Temporary VPS source and keyring were removed after build: `/tmp/whatomate-green-src`, `/tmp/whatomate-green-keyring.json`.

Promote sandbox green to public live:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback public live:

```bash
ln -sfn /opt/whatomate/bin/whatomate.green.20260528_111523 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```

Rollback sandbox only:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.blue /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```

## VPS sandbox green deploy - 2026-06-12

- Deployed current project revision `f518308b` as the new sandbox green runtime for `https://sandbox.ofuqalmadenah.com`.
- Created pre-deploy backup: `/root/whatomate_backups/whatomate-sandbox-green-predeploy-20260612_011507.tar.gz`.
- Backup SHA256: `612b71551489badffe2064d9faad63fc706535bf44657c40db4b2d4637731b7f`.
- New green binary: `/opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b`.
- New green SHA256: `26fa2f11406e4af956ac563f444b52148909810668e9f5f06e7bfbe3228c3044`.
- Sandbox blue rollback now points to `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`.
- License bootstrap verified active on sandbox and public live: `enabled=true`, `status=active`, `tier=production`, `key_id=deploy-20260416`.
- Browser QA via Chrome DevTools loaded `https://sandbox.ofuqalmadenah.com/login`, found no console errors, and confirmed `/api/license/bootstrap` returned active license data.
- Temporary VPS source and keyring were removed after build: `/tmp/whatomate-green-src`, `/tmp/whatomate-green-keyring.json`.

Switch sandbox to new green:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```

Rollback sandbox to blue:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.blue /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```
