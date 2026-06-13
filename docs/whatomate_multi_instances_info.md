# Whatomate Multi-Instances Info

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
