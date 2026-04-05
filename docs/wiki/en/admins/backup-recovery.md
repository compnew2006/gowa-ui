---
title: Backup & Recovery
---

# Backup & Recovery

This guide covers backup strategies and recovery procedures for Whatomate deployments.

## Backup Components

A complete backup includes:

| Component | Type | Backup Method |
|-----------|------|---------------|
| PostgreSQL Database | Structured data | `pg_dump` |
| Redis | Cache, sessions, queues | RDB snapshot / AOF |
| File Storage | Media files, uploads | File system copy |
| Configuration | Config files, env vars | Version control / secrets manager |

## Database Backup (PostgreSQL)

### Full Backup

```bash
# Backup to file
pg_dump -h <host> -U whatomate -d whatomate -F c -f whatomate_$(date +%Y%m%d).dump

# Backup with compression
pg_dump -h <host> -U whatomate -d whatomate -F c -Z 9 -f whatomate_$(date +%Y%m%d).dump.gz

# Backup specific tables
pg_dump -h <host> -U whatomate -d whatomate -t contacts -t messages -F c -f partial.dump
```

### Automated Backups

**Cron job:**

```bash
# Daily backup at 2 AM
0 2 * * * pg_dump -h localhost -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +\%Y\%m\%d).dump

# Keep last 30 days
0 2 * * * pg_dump -h localhost -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +\%Y\%m\%d).dump && find /backups -name "whatomate_*.dump" -mtime +30 -delete
```

**Docker:**

```yaml
services:
  backup:
    image: postgres:15-alpine
    volumes:
      - ./backups:/backups
    environment:
      PGPASSWORD: ${DB_PASSWORD}
    command: >
      sh -c "pg_dump -h postgres -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +%Y%m%d).dump"
```

### Point-in-Time Recovery

Enable WAL archiving for point-in-time recovery:

```postgresql
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'
```

## Redis Backup

### RDB Snapshot

Redis automatically creates RDB snapshots based on configuration:

```redis
# redis.conf
save 900 1     # Save after 900s if 1 key changed
save 300 10    # Save after 300s if 10 keys changed
save 60 10000  # Save after 60s if 10000 keys changed
dbfilename dump.rdb
dir /data
```

Manual snapshot:

```bash
redis-cli -h <host> -a <password> BGSAVE
```

### AOF (Append Only File)

For better durability, enable AOF:

```redis
# redis.conf
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
```

### Backup RDB File

```bash
# Copy RDB file
cp /data/dump.rdb /backups/redis_$(date +%Y%m%d).rdb

# Or use redis-cli
redis-cli -h <host> -a <password> --rdb /backups/redis_$(date +%Y%m%d).rdb
```

## File Storage Backup

Media files are stored in the local storage path configured in `storage.local_path`:

```bash
# Backup storage directory
tar -czf /backups/storage_$(date +%Y%m%d).tar.gz -C /app/storage .

# Or use rsync for incremental backups
rsync -avz /app/storage/ /backups/storage/
```

### Docker Volume Backup

```bash
# Backup Docker volume
docker run --rm -v whatomate_storage:/data -v $(pwd)/backups:/backup \
  alpine tar -czf /backup/storage_$(date +%Y%m%d).tar.gz -C /data .
```

## Recovery Procedures

### Database Recovery

```bash
# Stop application
docker compose stop whatomate

# Restore from backup
pg_restore -h <host> -U whatomate -d whatomate -c whatomate_20240101.dump

# Restart application
docker compose start whatomate
```

**Important:** The `-c` flag drops existing objects before restoring. Use with caution.

### Redis Recovery

```bash
# Stop Redis
docker compose stop redis

# Restore RDB file
cp /backups/redis_20240101.rdb /data/dump.rdb

# Start Redis
docker compose start redis
```

### File Storage Recovery

```bash
# Restore from backup
tar -xzf /backups/storage_20240101.tar.gz -C /app/storage/
```

### Full Recovery

1. Restore PostgreSQL database
2. Restore Redis data
3. Restore file storage
4. Run migrations to ensure schema is current:
   ```bash
   ./whatomate -migrate
   ```
5. Run crypto migration if encryption key changed:
   ```bash
   ./whatomate crypto-migrate
   ```
6. Start the application
7. Verify health:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

## Disaster Recovery Plan

### Recovery Time Objectives

| Component | RTO | RPO |
|-----------|-----|-----|
| Database | 15 minutes | 24 hours (daily backups) |
| Redis | 5 minutes | 1 hour (RDB snapshots) |
| File Storage | 30 minutes | 24 hours |
| Application | 5 minutes | N/A (stateless) |

### Recovery Steps

1. **Assess the failure** — Identify which components are affected
2. **Provision infrastructure** — Set up replacement servers if needed
3. **Restore data** — Follow recovery procedures above
4. **Verify** — Run health and readiness checks
5. **Notify** — Inform users of service restoration

### Testing Recovery

Test your backup and recovery procedures regularly:

1. Create a backup of the production database
2. Restore to a staging environment
3. Verify data integrity
4. Test application functionality
5. Document any issues

## Organization Deletion

Whatomate uses cascade soft-delete for organization deletion. When an organization is deleted:

1. The organization record is soft-deleted (`deleted_at` set)
2. All related records are cascade soft-deleted:
   - Users
   - WhatsApp accounts
   - WhatsApp instances
   - Contacts
   - Messages
   - Campaigns
   - Templates
   - Chatbot settings
   - Roles and permissions
   - Tags
   - Teams
   - Webhooks
   - Custom actions
   - Activity logs
   - Widgets
   - Lead requests
   - Notifications
   - SSO providers

### Deleting an Organization

```bash
curl -X DELETE https://whatomate.example.com/api/organizations/{id} \
  -H "Authorization: Bearer <token>"
```

**Note:** Only super admins can delete organizations. The deletion is soft — data is preserved and can be recovered by clearing the `deleted_at` timestamp.

### Recovering a Deleted Organization

```sql
-- Restore organization
UPDATE organizations SET deleted_at = NULL WHERE id = <org_id>;

-- Restore related records (cascade)
UPDATE users SET deleted_at = NULL WHERE organization_id = <org_id>;
UPDATE whatsapp_accounts SET deleted_at = NULL WHERE organization_id = <org_id>;
-- ... repeat for all related tables
```

## Backup Verification

Regularly verify backup integrity:

```bash
# Verify database backup
pg_restore -l whatomate_20240101.dump

# Verify Redis backup
redis-check-rdb /backups/redis_20240101.rdb

# Verify storage backup
tar -tzf /backups/storage_20240101.tar.gz | head
```

## See Also

- [Data Migration](data-migration.md) — Database migration procedures
- [Monitoring](monitoring.md) — Health checks for backup verification
- [Troubleshooting](troubleshooting.md) — Recovery-related issues
- [Configuration](configuration.md) — Storage path configuration
