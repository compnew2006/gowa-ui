# Whatomate Green Replacement Deployment Summary

Date: 2026-05-12 15:30:34 EEST / 2026-05-12 12:30:34 UTC
VPS: `31.97.192.53`

## Result

Deployed the current clean workspace revision `a1f143cc` as a new green sandbox build while keeping public blue live.

- Public blue active path: `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.blue.20260511_002729`
- New green binary: `/opt/whatomate/bin/whatomate.green.20260512_122534`
- Green version: `Whatomate a1f143cc-green-20260512_122534 (built 2026-05-12_12:27:39)`
- Green SHA256: `9ef7a2fed8b40516f8a957c5fa37e2190d669ede31110896cdd592641d3a8361`
- Green sandbox service: `whatomate-sandbox`
- Green URL: `https://sandbox.ofuqalmadenah.com`
- Green port: `127.0.0.1:18127`
- Backup: `/root/whatomate_backups/whatomate-green-replace-predeploy-20260512_122115.tar.gz`

## Switch Commands

Promote green to live and stop sandbox:

```bash
ln -sfn /opt/whatomate/bin/whatomate.green.20260512_122534 /opt/whatomate/bin/whatomate && systemctl stop whatomate-sandbox && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya
```

Switch back to blue and run green as sandbox:

```bash
ln -sfn /opt/whatomate/bin/whatomate.blue.20260511_002729 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz whatomate@alarkan-almthalia whatomate@matbaat-ruya && systemctl restart whatomate-sandbox
```

## Verification

- Local backend tests passed:
  - `go test ./internal/database ./internal/handlers ./internal/config ./internal/crypto ./internal/license ./pkg/whatsapp ./pkg/whatsmeow`
- Local frontend production build passed:
  - `cd frontend && npm run build`
- VPS production build passed with embedded license keyring.
- Services active:
  - `whatomate`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
  - `whatomate-sandbox`
- License active on ports `18123`, `18124`, `18125`, `18126`, and `18127`.
- HTTPS returned `200` for:
  - `https://ofuqalmadenah.com`
  - `https://www.ofuqalmadenah.com`
  - `https://sandbox.ofuqalmadenah.com`
- Blue/green authenticated API parity:
  - pending chats totals and first page IDs match
  - open chats totals and IDs match
  - sample chat messages totals and first 100 IDs match
- Chrome DevTools MCP browser verification passed:
  - sandbox login page loaded
  - browser-side login returned `200`
  - `/api/license/bootstrap` returned `enabled=true`, `status=active`, `locked=false`, `key_id=deploy-20260416`
  - pending chat API returned `200`
  - screenshot: `tmp/green-replace-verify-20260512.png`

## Cleanup

Removed VPS temporary/source paths after build:

- `/tmp/whatomate-green-src`
- `/tmp/whatomate-green-keyring.json`
- `/tmp/whatomate-chunk.aa`
- `/tmp/whatomate-linux-amd64.gz`
- `/opt/whatomate-sandbox/src`
- `/opt/whatomate-src`
- `/opt/whatomate-sandbox/.cache`
- `/opt/whatomate-sandbox/.gopath`

Removed old green binaries after verifying the new green:

- `/opt/whatomate/bin/whatomate.green.20260511_002522`
- `/opt/whatomate/bin/whatomate.green.20260511_002922`
- `/opt/whatomate/bin/whatomate.green.20260512_083647`

## Skills

Selected only `devops-engineer` for this task.

Competencies applied: blue/green deployment, systemd service overrides, native Go/Vite production build, embedded license verification, API/browser smoke testing, and VPS source cleanup.
