# Whatomate Session Summary - 2026-05-19

## Task

Fix and deploy the green Whatomate version for:

- Missing WhatsMeow media on the two 07:14 PM chat bubbles in `https://ofuqalmadenah.com/chat/82edad6a-708a-4ce9-af2b-6c8f72b27cac`.
- Retry download falsely reporting success when the underlying media blob is absent.
- Future inbound WhatsMeow dedup reusing stale `media_asset` rows whose object blobs are missing.
- Fatal Assign Contact dialog resize crash: `t.value.getBoundingClientRect is not a function`.
- Confirm license overview is active and green is the running side of the blue/green deployment.

## Approach And Key Decisions

- Confirmed production green was active before debugging.
- Reproduced the 07:14 PM media issue with browser automation and API checks.
- Verified the two current PDFs cannot be reconstructed by the app because their object files are missing and the message rows do not contain WhatsMeow recovery metadata.
- Fixed the code path so future inbound WhatsMeow media dedup only reuses an existing media asset when the blob exists; otherwise it downloads and stores the current inbound payload again.
- Fixed retry behavior so object-backed media must verify the blob exists before returning success.
- Fixed the Vue resizable composable to accept Reka/Vue component refs by resolving `$el` before calling DOM APIs.
- Clarified active-license quota copy so the License page no longer says licensing is disabled when the deployment is active.

## Files Modified

- `pkg/whatsmeow/media_service.go`
- `pkg/whatsmeow/media_service_test.go`
- `internal/handlers/media.go`
- `internal/handlers/media_stream_test.go`
- `frontend/src/lib/useResizable.ts`
- `frontend/src/lib/useResizable.test.ts`
- `frontend/src/components/ui/dialog/DialogContent.vue`
- `frontend/src/i18n/locales/en.json`
- `docs/whatomate_multi_instances_info.md`
- `summary.md`
- `summery.md`

## Deployment

- Branch: `agent/media-dedup-resize-fix`
- Commits:
  - `d1a9b62` - `Fix missing media dedup recovery and resizable refs`
  - `f3575f3` - `Clarify active license quota copy`
- Active VPS green binary:
  - `/opt/whatomate/bin/whatomate.green.20260519_002559`
  - `Whatomate green-20260519_032337-f3575f3-media-dedup-resize (built 2026-05-19_00:23:37)`
- Backup created before final green replacement:
  - `/root/whatomate_backups/20260519_002559_pre_green_copy_tweak`
- Switch command installed/updated:
  - `whatomate-switch status`
  - `whatomate-switch green`
  - `whatomate-switch blue`
  - `whatomate-switch` toggles between latest green and latest blue.

## Verification

- `go test ./pkg/whatsmeow ./internal/handlers` passed.
- `cd frontend && npx vitest run src/lib/useResizable.test.ts` passed.
- `cd frontend && npx eslint src/lib/useResizable.ts src/lib/useResizable.test.ts src/components/ui/dialog/DialogContent.vue` passed.
- `cd frontend && npm run typecheck` still fails on pre-existing project-wide TypeScript errors outside this change, including readonly test fixture tags, existing `body` access typing in `ChatView.vue`, non-exported store types, and deep toast typing.
- `git diff --check` passed before deployment.
- `npm run build` passed with only the existing Vite chunk-size warning.
- Production systemd services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz.service`
  - `whatomate@alarkan-almthalia.service`
  - `whatomate@matbaat-ruya.service`
- Production login smoke:
  - `18123`, `18124`, `18125`, `18126` all returned HTTP `200`.
- Production Playwright verification:
  - 07:14 PM two-file bubble is visible.
  - `GET /api/media/3af5f0d6-af16-4689-8711-6c40dde6c6f7` returns `404`.
  - `POST /api/media/3af5f0d6-af16-4689-8711-6c40dde6c6f7/retry-download` returns `404` with `No recovery information available for this media`.
  - `GET /api/media/d1acbd65-d448-4d20-a37c-5d8d682d6dad` returns `404`.
  - `POST /api/media/d1acbd65-d448-4d20-a37c-5d8d682d6dad/retry-download` returns `404` with `No recovery information available for this media`.
  - Assign Contact dialog opened on an active chat and resized without fatal UI error or `getBoundingClientRect` page error.
  - License page shows `License overview` as `Active`; no Disabled/licensing-disabled copy remains.

## Artifacts

- `playwright-green-chat-media-after-fix.png`
- `playwright-green-assign-resize-after-fix.png`
- `playwright-green-license-after-fix.png`
- Earlier investigation screenshots:
  - `internal-browser-chat-media-investigation.png`
  - `playwright-chat-media-missing.png`
  - `playwright-chat-media-retry-clicks.png`

## Known Limitations

The two already-broken 07:14 PM PDFs still cannot display because the referenced blobs are absent from `/opt/whatomate/uploads` and backups checked during the investigation, and the affected rows have no stored WhatsMeow recovery payload. The deployed fix prevents this stale-dedup path from recurring for future inbound WhatsMeow media and stops retry from returning fake success for unrecoverable historical media.

# 2026-05-25 - Green Text Send Fix Deployment

## Task

Deploy the current project to the VPS as the new green slot, preserve the blue rollback slot, fix the WhatsMeow text-send `400` failure seen on chat `8b04fdf4-3f6c-4226-a003-c0ade8c7b75d`, verify that the license overview is active, remove temporary/source codebases from the VPS after deployment, update the deployment documentation, and keep a one-command blue/green switch.

## Skills and MCPs Applied

- Skills: `devops-engineer`, `debugging-wizard`.
- MCPs/tools: Chrome DevTools for production UI verification, shell/SSH for build and systemd verification.

## Deployment

- Deployed source revision: `c1e34cd` (`Fix whatsmeow plain text sends`).
- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260525_200333`.
- Version: `Whatomate green-20260525_200333-c1e34cd-text-send (built 2026-05-25_20:07:08)`.
- SHA256: `fd8a6947d335531d4ee8ac85f2e2fb35a134d9351dbda972692bfbfb3797f18d`.
- Blue rollback binary left untouched: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260525_192630_pre_green_text_send_fix` (`759M`).

## Fix

- Plain WhatsMeow text messages now build a simple `Conversation` payload.
- Text messages containing URLs still use `ExtendedTextMessage` to preserve close-rating review-link delivery.
- The affected chat page still shows historical failed rows from before the deploy; no production Retry/send was triggered from the browser because that would send a real customer message.

## Verification

- Local tests passed:
  - `go test ./pkg/whatsmeow`
  - `go test ./internal/handlers -run 'TestSendViaProvider|TestApp_SendOutgoingMessage'`
  - `go test ./pkg/whatsmeow ./internal/handlers`
  - `go test ./...`
  - `git diff --check`
- VPS production build passed with embedded license keyring:
  - `GOTOOLCHAIN=go1.25.9+auto VERSION=green-20260525_200333-c1e34cd-text-send LICENSE_KEY_RING_FILE=/root/whatomate-keyring.json make build-prod`
- Services active:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`
- Local ports `18123`, `18124`, `18125`, and `18126` returned `/login` HTTP `200`.
- Public HTTPS login checks returned `200` for the main domain and all three tenant domains.
- License bootstrap on all four local ports returned `enabled=true`, `status=active`, and `locked=false`.
- Chrome DevTools confirmed `https://ofuqalmadenah.com/settings/license` displays `License overview` as `Active`.
- Chrome DevTools loaded the affected chat page; old `400` failures are visible as historical rows.
- Screenshot saved locally: `/Users/airm2/Downloads/whatomate/deploy-license-active-20260525.png`.

## Switch Command

- `whatomate-switch` toggles between green and blue.
- `whatomate-switch status` shows active, green, and blue binaries.
- `whatomate-switch green` promotes green explicitly.
- `whatomate-switch blue` rolls back to blue explicitly.

## Cleanup

- Removed temporary/source VPS paths after the verified install:
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, and backups.

# 2026-05-27 - Green Current Project Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260527_174500`.
- Version: `Whatomate green-20260527_174500-09191c2-csp (built 2026-05-27_17:42:53)`.
- SHA256: `a140bc30a10d018f05ff1da97bc9505f7ff1d82d241721b78ae74281bd948ff0`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_172753_pre_green_current_project`.

## Verification

- `go test ./...` passed.
- `cd frontend && npm run typecheck` passed.
- `git diff --check` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./cmd/whatomate ./internal/middleware ./internal/frontend` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public login checks returned `200` for all production domains.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools initially found a CSP nonce regression; it was fixed and redeployed.
- Final Chrome DevTools reload showed no console messages and all listed network requests returned `200`.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

# 2026-05-28 - Green Agent Selection UI Polish Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- Version: `Whatomate green-20260528_111523-09191c2-agent-ui (built 2026-05-28_11:18:57)`.
- SHA256: `4abd7096755d01623a54c4e56290fce386ecf256c45f098b521bd518ef08c921`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260528_111523_pre_agent_ui_polish`.
- All four production services are running from the new green binary:
  - `whatomate.service`
  - `whatomate@holol-wenjaz`
  - `whatomate@alarkan-almthalia`
  - `whatomate@matbaat-ruya`

## Verification

- `cd frontend && npm run typecheck` passed.
- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'` passed.
- `git diff --check` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Each service process executable resolves to `/opt/whatomate/bin/whatomate.green.20260528_111523`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for all production domains:
  - `https://ofuqalmadenah.com`
  - `https://holol-wenjaz.ofuqalmadenah.com`
  - `https://alarkan-almthalia.ofuqalmadenah.com`
  - `https://matbaat-ruya.ofuqalmadenah.com`
- HSTS and CSP headers are present on production responses.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```

## Cleanup

- Removed VPS temporary/source paths after verification:
  - `/root/whatomate-green-src-*`
  - `/root/whatomate_temp_build_*`
  - `/root/whatomate_src_*`
  - `/root/whatomate-source-*`
  - `/root/whatomate`
  - `/opt/whatomate-src`
  - `/opt/whatomate-sandbox/src`
- Preserved runtime binaries, configs, uploads, docs, license keyring, and backups.

# 2026-05-28 - Green Agent Scope Deployment

## Result

- Active slot: GREEN.
- Active binary: `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- Version: `Whatomate green-20260528_020100-09191c2-agent-scope (built 2026-05-28_02:00:01)`.
- SHA256: `4cbcfa440a67fba3d568b25e43f77e7a0352ebf71a0acd74bfbea0a3a1d2eabf`.
- Blue rollback preserved: `/opt/whatomate/bin/whatomate.blue.20260521_161500`.
- Backup before deployment: `/root/whatomate_backups/20260527_181332_pre_green_instance_scope`.
- All four production services are running from the new green binary, replacing the older running green version.

## Verification

- `GOCACHE=/private/tmp/whatomate-gocache go test ./internal/handlers -run 'TestAgentSelectionSettingsAppliesToInstance|TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'` passed.
- `cd frontend && npm run typecheck` passed.
- `git diff --check` passed.
- VPS build passed with embedded `/root/whatomate-keyring.json`.
- Services active: `whatomate.service`, `whatomate@holol-wenjaz`, `whatomate@alarkan-almthalia`, `whatomate@matbaat-ruya`.
- Each service process executable resolves to `/opt/whatomate/bin/whatomate.green.20260528_020100`.
- License bootstrap on `18123`, `18124`, `18125`, and `18126`: `enabled=true`, `status=active`, `locked=false`.
- Public `/login` checks returned `200` for all production domains.
- HSTS and CSP headers are present on production responses.
- Chrome DevTools verified `License overview` is `Active`.
- Chrome DevTools verified the new Customer routing `Instance scope` UI at `/settings/agent-selection`; all related network requests returned `200`.
- Chrome DevTools found no JavaScript console errors; it reported only non-blocking accessibility issues for existing form labels.

## Switch

```bash
whatomate-switch
```

Explicit:

```bash
whatomate-switch status
whatomate-switch green
whatomate-switch blue
```
