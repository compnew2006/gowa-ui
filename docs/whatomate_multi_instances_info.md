# Whatomate Multi-Instances Info

## Latest Production Deploy - 2026-06-13 19:24 UTC
- VPS: `31.97.192.53` | Domain: `ofuqalmadenah.com`
- Deployed commit: `b11fe78f` (defer inbound media to worker + parallelize consumers)
- Production green (ACTIVE): `/opt/whatomate/bin/whatomate.green.20260613_192120-b11fe78f`
- SHA256: `08a3ddb133c695f675bfc6095ec0a4be23a9e564948158135b5b8f752cf0adb8`
- Production blue (rollback): `/opt/whatomate/bin/whatomate.green.20260613_002015-1544b9cc` (previous known-good)
- License: ✅ enabled=true, status=active, locked=false, kind=paid, tier=production, lifetime, key_id=deploy-20260416
- Pre-deploy backup: `/root/whatomate_backups/20260613_192200_pre_20260613_192120-b11fe78f/`

### One-command switch (blue/green)
```bash
ssh root@31.97.192.53 "whatomate-switch green"    # run new version
ssh root@31.97.192.53 "whatomate-switch blue"     # rollback to 1544b9cc
ssh root@31.97.192.53 "whatomate-switch toggle"   # 1-command flip green<->blue
ssh root@31.97.192.53 "whatomate-switch status"   # show current
```

### Changes in this deploy (`b11fe78f`, 3 commits ahead of `1544b9cc`)
- **P0a**: incoming media now deferred to async recovery worker instead of blocking the per-instance event goroutine -> fixes ~7,500 dropped msgs/day (buffer overflow)
- **P0b**: inbound-media consumer fanned out to 4 parallel Redis Streams consumers (was 1) -> prevents the recovery queue becoming a bottleneck
- FB comments: per-page settings, multi-text random replies, WhatsApp notification
- 1 additive GORM migration (fb_comment model) applied via `-migrate` on startup
- Switch script gained `toggle` (parity with sandbox). Config defaults: `defer_inbound_media=true`, `inbound_media_worker_concurrency=4`

### Verification (2026-06-13 19:25 UTC)
- `whatomate.service` active, PID 3841386, version `green-20260613_192120-b11fe78f`
- Migration + plugin migrations completed; `Inbound media worker started`
- `GET /api/license/bootstrap` -> 200 active (browser + curl)
- Frontend bundle = `index-BMFCoqIE.css` / `index-CmZe5DWy.js` (new)
- `holol-wenjaz` tenant active + license active

### Known pre-existing issues (NOT caused by this deploy)
- `whatomate@alarkan-almthalia` + `whatomate@matbaat-ruya`: **REMOVED 2026-06-13** (unit files + instance dirs deleted; crash-loop stopped, NRestarts=13575 each). Orphaned DBs `whatomate_alarkan_almthalia` + `whatomate_matbaat_ruya` remain — drop manually if not needed.
- `whatomate-sandbox.service` crash-looping (separate from production).

## Latest Sandbox Deploy - 2026-06-12 23:05:32 UTC
- Target: `https://sandbox.ofuqalmadenah.com`
- VPS: `31.97.192.53`
- Deployed commit: `1544b9cc` (RBAC enforcement + permission hardening)
- New sandbox green: `/opt/whatomate/bin/whatomate.sandbox.green.20260613_014655-1544b9cc`
- SHA256: `92b1abdd54eb26494df9de3f096dcd192ac8b6e0611250920706d694651d48b8`
- License: ✅ enabled=true, status=active, locked=false
- Blue rollback: `/opt/whatomate/bin/whatomate.sandbox.blue`

### One-line switch command
```bash
ssh root@31.97.192.53 "whatomate-sandbox-switch green"
```

### Rollback (one command)
```bash
ssh root@31.97.192.53 "whatomate-sandbox-switch blue"
```

### Changes in this deploy
- RBAC enforcement for catalogs, group_directory, group_participants (3 new resources, 23 endpoints)
- chat:write enforcement for SendMessage, SendMedia, SendTypingPresence, SendReaction, SendPollVote
- chat:read for MarkMessageRead, templates:read for SendTemplateMessage
- authorizeRequest() helper, sendForbidden() for consistent 403 errors
- Plugin DRY fix, DeleteRole cache invalidation fix
- Frontend RESOURCE_LABELS + 8 docs updated
