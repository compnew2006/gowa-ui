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

## Latest Sandbox Deploy - 2026-06-12 01:20 UTC

- VPS: `31.97.192.53`
- Target: `https://sandbox.ofuqalmadenah.com`
- Backup: `/root/whatomate_backups/whatomate-sandbox-green-predeploy-20260612_011507.tar.gz`
- Backup SHA256: `612b71551489badffe2064d9faad63fc706535bf44657c40db4b2d4637731b7f`
- New sandbox green: `/opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b`
- New sandbox green SHA256: `26fa2f11406e4af956ac563f444b52148909810668e9f5f06e7bfbe3228c3044`
- Current sandbox active: `/opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b`
- Current sandbox blue rollback: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
- Public live observed: `/opt/whatomate/bin/whatomate.sandbox.green.20260611_200325-5702241f`
- License bootstrap verified active on sandbox and public live.

Switch sandbox to new green:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```

Rollback sandbox to blue:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.blue /opt/whatomate/bin/whatomate.sandbox.active && systemctl restart whatomate-sandbox
```

Promote this green to public:

```bash
ln -sfn /opt/whatomate/bin/whatomate.sandbox.green.20260612_011906-f518308b /opt/whatomate/bin/whatomate && systemctl restart whatomate
```


## Latest Deploy - 2026-06-12 16:47 UTC

- VPS: `31.97.192.53`
- Target: `https://sandbox.ofuqalmadenah.com`
- New sandbox green: `/opt/whatomate/bin/whatomate.sandbox.green.20260612_164756-f40d584c`
- Current sandbox active: same
- Current production active: same binary (promoted)
- Blue rollback (sandbox): previous green binary
- Blue/green switch scripts: `/usr/local/sbin/whatomate-sandbox-switch` and `/usr/local/sbin/whatomate-switch`
- License: Active (Paid • Lifetime)

### Changes in this deploy
- Added `GET /api/facebook/comments/pages` endpoint — page filter now shows ALL distinct pages with comments, not just from first 100
- Fixed Assign dialog scrolling — added `overflow-hidden` to DialogContent
- Removed instance access filtering from assign user list — all org members shown
- Refactored assign system: extracted shared helpers (validateAgentExists, freshDB, cancelActiveChatbotSession, buildTransferWebhookPayload, buildAgentTransferResponse)
- Fixed missing webhook events for PickNextTransfer and ReturnAgentTransfersToQueue
- Fixed contact stuck as "assigned" when unassigning from transfer
- Added page filter dropdown to Facebook Comments Inbox

### Quick commands
```bash
# Switch sandbox blue/green:
whatomate-sandbox-switch [blue|green|status]

# Switch production blue/green:
whatomate-switch [blue|green|status]

# Rollback:
whatomate-switch blue
whatomate-sandbox-switch blue
```
