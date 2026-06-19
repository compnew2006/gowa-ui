# Whatomate Multi-Instances Info

## Latest Production Deploy - 2026-06-16 21:50 UTC
- VPS: `31.97.192.53` | Domain: `ofuqalmadenah.com`
- Deployed commit: `5bad3266` (feat(whatsmeow): implement priority event ingestion with sharded FIFO queues)
- Production green (ACTIVE): `/opt/whatomate/bin/whatomate.green.5bad3266-deploy160616`
- SHA256: `992ee8590688032b7ce1a21a01e52b139529a9ed991caf4a740317a41c34ed5f`
- Production blue (rollback): `/opt/whatomate/bin/whatomate.green.cfbcc1ec-deploy150615`
- License: ✅ enabled=true, status=active, locked=false, kind=paid, tier=production, lifetime, key_id=deploy-20260416
- Pre-deploy backup: `/root/whatomate_backups/pre_green_deploy_20260616_214354/`

### One-command switch (blue/green)
```bash
ssh root@31.97.192.53 "whatomate-switch green"    # run new version
ssh root@31.97.192.53 "whatomate-switch blue"     # rollback
ssh root@31.97.192.53 "whatomate-switch toggle"   # 1-command flip green<->blue
ssh root@31.97.192.53 "whatomate-switch status"   # show current
```

### Changes in this deploy (`5bad3266`)
- feat(whatsmeow): implement priority event ingestion with sharded FIFO queues

### Verification (2026-06-16 21:53 UTC)
- `whatomate.service` active, version `5bad3266`
- `whatomate@holol-wenjaz.service` active
- Migration + plugin migrations completed
- `GET /api/license/bootstrap` -> 200 active
- `https://ofuqalmadenah.com/login` -> 200
- License: enabled=true, status=active

### Known pre-existing issues (NOT caused by this deploy)
- `whatomate@alarkan-almthalia` + `whatomate@matbaat-ruya`: **REMOVED 2026-06-13** (unit files + instance dirs deleted; crash-loop stopped, NRestarts=13575 each). Orphaned DBs `whatomate_alarkan_almthalia` + `whatomate_matbaat_ruya` remain — drop manually if not needed.
- `whatomate-sandbox.service` crash-looping (separate from production).
- Debug mode is enabled in config (`debug = true`) — set to `false` and restart to disable
- Disk is at 89% (12G free of 96G)

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

## Production Status Broadcast Queue Hotfix - 2026-06-19
- Target: `https://ofuqalmadenah.com`
- VPS: `31.97.192.53`
- Base commit: `5bad3266`
- Hotfix binary: `/opt/whatomate/bin/whatomate.green.5bad3266-statusfix-20260619_0250`
- Previous active binary: `/opt/whatomate/bin/whatomate.green.5bad3266-deploy160616`
- Backup: `/root/whatomate_backups/pre-statusfix-20260618_235208.tar.gz`
- SHA256: `1b05cf7bde21d44f1c6c5a1eea225492bcdd2600f11bee503ab2198558f7097c`
- Root cause: WhatsApp `status@broadcast` events arrived as `*events.Message` and were handled as high-priority chat events. A status flood could fill one high-priority shard and drop real chat messages.
- Fix: `pkg/whatsmeow/async_events.go` routes messages matched by `isStatusMessageInfo` to low priority; normal chat messages remain high priority.
- Verified: `go test -p 1 -count=1 ./pkg/whatsmeow`; both active VPS services are running; `/health` returned `200` on ports `18123` and `18124`; no overflow/drop signatures appeared after restart.

### Rollback
```bash
ln -sfn /opt/whatomate/bin/whatomate.green.5bad3266-deploy160616 /opt/whatomate/bin/whatomate && systemctl restart whatomate whatomate@holol-wenjaz
```
