# Per-Instance Uploads Cleanup Plugin

Manages per-instance file retention policies for uploaded media, with automatic cleanup of expired uploads and a full audit trail.

## Overview

By default, uploaded file cleanup uses organization-wide settings. This plugin allows each WhatsApp instance to override the retention policy — either inheriting the org default or specifying a custom number of days. Cleanup runs can be triggered manually or on schedule, and all configuration changes are audited.

## Registration

Activated via blank import in `cmd/whatomate/main.go`:

```go
import _ "github.com/compnew2006/whatomate/plugin/per-instance-uploads-cleanup"
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/instances/{id}/uploads-cleanup` | Get retention settings for an instance |
| PUT | `/api/instances/{id}/uploads-cleanup` | Update retention settings for an instance |
| GET | `/api/instances/{id}/uploads-cleanup/history` | Get audit history for an instance |
| POST | `/api/instances/{id}/uploads-cleanup/run` | Manually trigger a cleanup run |
| GET | `/api/org/uploads-cleanup/instances` | Overview of all instance retention settings |

## Data Model

### InstanceUploadsCleanupAudit (`instance_uploads_cleanup_audits`)

Tracks every configuration change for audit purposes.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Primary key |
| OrganizationID | uuid | Tenant scope |
| InstanceID | uuid | The instance this audit belongs to |
| ActorUserID | uuid? | User who made the change |
| ActorEmail | string? | Email of the actor |
| OldInherit | bool? | Previous inherit setting |
| NewInherit | bool | New inherit setting |
| OldRetentionDays | int? | Previous retention days |
| NewRetentionDays | int | New retention days |
| Reason | string? | Reason for the change |

### Instance Settings (JSONB)

Stored in `WhatsAppInstance.settings` under the `uploads_cleanup` key:

```json
{
  "uploads_cleanup": {
    "inherit": true,
    "retention_days": 30,
    "last_run_date": "2026-06-08"
  }
}
```

## Files

| File | Purpose |
|------|---------|
| `plugin.go` | Plugin registration, routes, migration |
| `model.go` | `InstanceUploadsCleanupAudit` GORM model |
| `handler_retention.go` | GET/PUT retention, history, run, overview handlers |
| `service.go` | Cleanup business logic |
| `validation.go` | Input validation for retention settings |
| `plugin_test.go` | Plugin registration tests |
| `handler_retention_test.go` | Handler contract tests |
| `service_test.go` | Service logic tests |
| `validation_test.go` | Validation tests |

## Migration

On startup, the plugin:
1. Backfills `uploads_cleanup.inherit: true` for all existing instances that lack the setting
2. Creates the `instance_uploads_cleanup_audits` table via GORM AutoMigrate
3. Creates a composite index on `(organization_id, instance_id, created_at)`

## Tenant Scoping

All queries are scoped to the requesting user's organization via `middleware.GetOrganizationID(rc)` and `tenant.ScopedDB()`.
