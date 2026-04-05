---
title: Data Migration
---

# Data Migration

Whatomate uses GORM AutoMigrate for schema management and a custom crypto migration for encryption upgrades.

## Database Migrations

### GORM AutoMigrate

Whatomate uses GORM's AutoMigrate to manage database schema. On migration, GORM:

- Creates tables for all defined models
- Adds missing columns
- Creates indexes and constraints
- Does **not** delete columns or change column types (to prevent data loss)

### Running Migrations

**Command-line:**

```bash
# Run migrations and exit
./whatomate -migrate

# With custom config
./whatomate -migrate -config /etc/whatomate/config.toml
```

**API endpoint (super admin only):**

```bash
# Trigger migration
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>"

# Check migration status
curl https://whatomate.example.com/api/admin/migrate/status \
  -H "Authorization: Bearer <token>"
```

### Migration Process

When migrations run:

1. **GORM AutoMigrate** — Creates/updates all model tables
2. **Default admin creation** — Creates admin user from config if none exists
3. **Default roles** — Creates admin, agent, manager roles for organizations
4. **Default chatbot settings** — Creates default chatbot configuration per org

### Default Admin Creation

The default admin is created from configuration during the first migration:

```toml
[default_admin]
email = "admin@whatomate.example.com"
password = "Admin@1234"
full_name = "System Administrator"
```

The migration checks if a user with the configured email exists. If not, it:
1. Creates the user with bcrypt-hashed password
2. Creates the organization
3. Creates default roles (admin, agent, manager)
4. Adds the user to the organization with admin role

## Crypto Migration

### Overview

The crypto migration re-encrypts sensitive data from legacy encryption formats to the current AES-256-GCM format (enc3).

### Encryption Versions

| Version | Prefix | Algorithm | Status |
|---------|--------|-----------|--------|
| enc1 | `enc:` | Legacy | Deprecated — migrate to enc3 |
| enc2 | `enc2:` | Legacy v2 | Deprecated — migrate to enc3 |
| enc3 | `enc3:` | AES-256-GCM | Current |

### Running Crypto Migration

```bash
# Preview changes (dry run)
./whatomate crypto-migrate -dry-run

# Include enc2 format in scan
./whatomate crypto-migrate -include-enc2

# Custom batch size
./whatomate crypto-migrate -batch-size 500

# Execute
./whatomate crypto-migrate
```

### What Gets Migrated

| Table | Encrypted Columns |
|-------|-------------------|
| `whatsapp_accounts` | access_token, phone_number_id, business_account_id, webhook_verify_token |
| `sso_providers` | client_secret |
| `chatbot_settings` | ai_api_key |
| `webhooks` | secret |
| `custom_actions` | headers |

### Migration Process

1. Load configuration and validate encryption key
2. Connect to database
3. Scan for legacy encrypted secrets (`enc:` and `enc2:` prefixes)
4. For each record:
   - Decrypt with the old format
   - Re-encrypt with `enc3:` format
   - Update the record
5. Process in batches (configurable, default 1000)
6. Report summary of updated records

### Migration Output

```
Crypto Migration Report
======================
Total records scanned: 1500
Records updated (enc → enc3): 1200
Records updated (enc2 → enc3): 250
Records already enc3: 50
Failed: 0
Duration: 12.5s
```

### Migration Safety

- **Dry run first**: Always run with `-dry-run` to preview changes
- **Backup first**: Create a database backup before running
- **Batch processing**: Large datasets are processed in configurable batches
- **Idempotent**: Running the migration multiple times is safe — already-migrated records are skipped

### Troubleshooting

**Migration fails with decryption error:**
- The encryption key may have changed
- Verify the current `encryption_key` in config matches the one used for the original encryption
- If the key is lost, legacy data cannot be decrypted

**Migration finds no records:**
- All data may already be in enc3 format
- Try with `-include-enc2` if you suspect enc2 data exists

**Migration is slow:**
- Increase batch size: `-batch-size 2000`
- Check database performance and network latency

## Migration Status API

Check migration status via API:

```bash
curl https://whatomate.example.com/api/admin/migrate/status \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "status": "completed",
  "last_run": "2024-01-01T12:00:00Z",
  "models_migrated": 30,
  "errors": []
}
```

## Schema Changes

When adding new models or modifying existing ones:

1. Update the model definition in `internal/models/`
2. Run `whatomate -migrate` to apply changes
3. GORM will create new tables and add missing columns

**Note:** GORM AutoMigrate does not:
- Delete columns that no longer exist in the model
- Change column types
- Delete indexes that no longer exist

For destructive changes, write raw SQL migrations:

```bash
# Apply raw SQL migration
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query": "ALTER TABLE contacts ADD COLUMN priority INT DEFAULT 0"}'
```

## See Also

- [Database Models](../developers/database-models.md) — All model definitions
- [Configuration](configuration.md) — Encryption key configuration
- [Backup & Recovery](backup-recovery.md) — Backup before migrations
- [Troubleshooting](troubleshooting.md) — Migration troubleshooting
