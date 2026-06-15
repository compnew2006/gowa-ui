# Whatomate Multi-Instances Info

## Latest Production Deploy - 2026-06-15 02:35 UTC
- VPS: `31.97.192.53` | Domain: `ofuqalmadenah.com`
- Deployed commit: `f7513f9d` (fix: log real client IP in observedHandler instead of TCP source)
- Production green (ACTIVE): `/opt/whatomate/bin/whatomate.green-20260614_233550-f7513f9d`
- SHA256: `e1bd749bcc560a14e2628ed97e419c0546ce1133fd04151f0c2a6769a0c4affa`
- Production blue (rollback): `/opt/whatomate/bin/whatomate.green-20260614_143926-f7513f9d`
- License: ✅ enabled=true, status=active, locked=false, kind=paid, tier=production, lifetime, key_id=deploy-20260416
- Pre-deploy backup: `/root/whatomate_backups/pre_green_deploy_20260615_022658/`

### One-command switch (blue/green)
```bash
ssh root@31.97.192.53 "whatomate-switch green"    # run new version
ssh root@31.97.192.53 "whatomate-switch blue"     # rollback
ssh root@31.97.192.53 "whatomate-switch toggle"   # 1-command flip green<->blue
ssh root@31.97.192.53 "whatomate-switch status"   # show current
```

### Changes in this deploy (`f7513f9d`)
- fix: log real client IP in observedHandler instead of TCP source

### Verification (2026-06-15 02:35 UTC)
- `whatomate.service` active, version `green-20260614_233550-f7513f9d`
- Migration + plugin migrations completed
- `GET /api/license/bootstrap` -> 200 active (browser + curl)
- License: enabled=true, status=active

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
