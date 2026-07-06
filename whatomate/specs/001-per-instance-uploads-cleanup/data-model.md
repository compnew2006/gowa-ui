# Data Model: Per-Instance Uploads Cleanup Retention

**Date**: 2026-06-06
**Branch**: `001-per-instance-uploads-cleanup`
**Spec**: [spec.md](../spec.md) · **Plan**: [plan.md](../plan.md) · **Research**: [research.md](../research.md)

This document describes the entities, fields, validation rules, and state transitions for the feature. All entities live inside the `plugin/per-instance-uploads-cleanup/` plugin; no core model is modified except by the bulk `AutoMigrate` invocation on plugin startup.

---

## Entity: `WhatsAppInstance` (existing — additive change)

**Table**: `whatsapp_instances` (existing)
**Source**: `internal/models/instance.go:10-31`
**Change type**: additive — no new column, no new struct field. The `Settings` JSONB column gains a new sub-namespace.

### New sub-keys inside `Settings` JSONB

| Key | Type | Required | Default | Description |
|---|---|---|---|---|
| `uploads_cleanup.inherit` | boolean | yes (after first write) | `true` | When `true`, the instance uses the workspace default; when `false`, the instance uses `uploads_cleanup.retention_days`. Persisted explicitly so we can distinguish "user said inherit" from "no value ever set" (D-8). |
| `uploads_cleanup.retention_days` | int | required when `inherit = false` | — | Integer days; `0` means **instance-level disable** (overrides the workspace default and forces cleanup to be off for this instance); values `1..maxUploadsCleanupRetentionDays` mean "delete files older than N days". |
| `uploads_cleanup.last_run_date` | string | optional | — | ISO date (`YYYY-MM-DD`) of the most recent successful run for this instance. Independent of the workspace `last_run_date` (which is the org-level last-run). |

### Validation rules

- `uploads_cleanup.inherit` must be a JSON boolean. Anything else → 400.
- `uploads_cleanup.retention_days` must be a JSON integer in `[0, maxUploadsCleanupRetentionDays]` (currently `3650`). Out of range → 400 with code `uploads_cleanup_retention_days` and the human message `uploads_cleanup retention must be a whole number between 0 and 3650`.
- When `inherit = true`, `retention_days` is ignored — the worker reads the workspace default from `Organization.settings.uploads_cleanup_retention_days`.
- When `inherit = false` AND `retention_days` is missing → 400 with code `uploads_cleanup_retention_days_required` and the human message `uploads_cleanup retention is required when "Inherit workspace default" is off`.

### State machine for `inherit × retention_days`

| `inherit` | `retention_days` | Effective behavior |
|---|---|---|
| `true` | (any) | Use workspace default; if workspace default is `0` → disabled for this instance. |
| `false` | `0` | Instance-level disable (overrides workspace default). |
| `false` | `>0` (≤3650) | Delete files belonging to this instance that are older than N days. |
| (key missing) | (key missing) | Treated as `inherit = true` for backward compatibility with instances created before this feature. |

This four-state matrix satisfies the consistency check in the requirements-quality checklist (`CHK022`).

---

## Entity: `InstanceUploadsCleanupAudit` (new)

**Table**: `instance_uploads_cleanup_audits` (new)
**Owner**: `plugin/per-instance-uploads-cleanup/model.go`
**Change type**: new model + new table. Auto-created on plugin startup via `db.AutoMigrate(&InstanceUploadsCleanupAudit{})` in the plugin's `Migrate(db)` method.

### Fields

| Field | GORM | Type | Description |
|---|---|---|---|
| `ID` | inherited from `BaseModel` | uuid.UUID PK | |
| `CreatedAt` | inherited from `BaseModel` | time.Time | Indexable; default sort key. |
| `UpdatedAt` | inherited from `BaseModel` | time.Time | |
| `DeletedAt` | inherited from `BaseModel` | gorm.DeletedAt | Soft delete. |
| `OrganizationID` | `type:uuid;not null;index` | uuid.UUID | Tenant scope. |
| `InstanceID` | `type:uuid;not null;index:idx_iuca_org_instance_created,priority:2` | uuid.UUID | The instance whose retention changed. |
| `ActorUserID` | `type:uuid;index` (nullable) | *uuid.UUID | The user who made the change. Nullable to allow system-driven writes (none planned, but the column is intentionally nullable to avoid a future migration). |
| `ActorEmail` | `type:varchar(255)` (nullable) | *string | Denormalized snapshot of the actor's email at write time. UI displays this; survives user soft-delete. |
| `OldInherit` | nullable | *bool | Previous inherit value. Nullable because the first write on a brand-new instance has no "previous" value. |
| `NewInherit` | `not null` | bool | New inherit value. |
| `OldRetentionDays` | nullable | *int | Previous `retention_days` value. |
| `NewRetentionDays` | nullable | *int | New `retention_days` value. Nullable only when `NewInherit = true` and the input cleared the field. |
| `Reason` | `type:varchar(500)` (nullable) | *string | Optional free-form reason (e.g., `"compliance: 30 days per legal review"`). Frontend exposes a small text input. |

### Indexes

- `(organization_id, instance_id, created_at DESC)` — composite, supports the "last 5 changes" query per instance.
- `idx_iuca_org_instance_created` — the GORM-tagged name of the composite above.
- `organization_id` standalone — supports tenant-scoped housekeeping queries.
- `actor_user_id` — supports future "changes by user" reports.

### TableName

```go
func (InstanceUploadsCleanupAudit) TableName() string {
    return "instance_uploads_cleanup_audits"
}
```

### Tenant scoping

All queries on this model use `tenant.ScopedDB(db, orgID)` per constitution §6.5. The model does **not** have a `Organization` relation field (audit rows are append-only and don't need to navigate to the org); the tenant scope is enforced at query time.

### State transitions

Audit rows are immutable after write. The only lifecycle is:
- **Created** by the retention handler when a `PUT /api/instances/{id}/uploads-cleanup` succeeds.
- **Read** by the history endpoint (`GET .../history?limit=5&offset=0`).
- **Soft-deleted** if a future GDPR-style "right to erasure" flow needs to scrub a user's history (out of scope for this iteration; the column `actor_user_id` is nullable precisely so that this can be addressed later by clearing the FK rather than hard-deleting the row).

### Retention

Per [research.md](../research.md#c-1-audit-retention-policy-open--pending-product-decision), the default is **forever**. The plan does not introduce a TTL job. A future iteration may add one.

---

## Entity: `Organization` (existing — read-only change)

**Table**: `organizations` (existing)
**Source**: `internal/models/models.go`
**Change type**: none. The workspace `uploads_cleanup_retention_days` and `uploads_cleanup_schedule_hour` continue to be authoritative defaults. The worker reads them via `Organization.Settings` (the JSONB column), exactly as it does today.

The only behavioral change is **how the worker interprets** the workspace default relative to per-instance overrides — see [Resolution rules](#resolution-rules) below.

---

## Entity: `Message` (existing — read-only change)

**Table**: `messages` (existing)
**Source**: `internal/models/models.go:510-548`
**Change type**: none. The `InstanceID *uuid.UUID` column at line 513 is the source of truth for file→instance resolution (D-3). The worker queries it via the `media_url` or `media_asset_id` join in `uploads_cleanup_worker.go:438-465`.

---

## Entity: `MediaAsset` (existing — read-only change)

**Table**: `media_assets` (existing)
**Source**: `internal/models/models.go:464-471`
**Change type**: none. `MediaAsset` is the deduplicated storage record; the worker resolves it via `s3_key` matching. `MediaAsset` does **not** carry an `instance_id` column in this iteration; resolution is via the `Message` join.

---

## Resolution rules

The worker resolves the **effective retention** for a file at cleanup time as follows:

1. **Identify the owning instance** for a given file path by querying `messages` for rows whose `media_url` (or joined `media_assets.s3_key`) matches the file's relative path, and taking the `instance_id` of the most recent matching message. If multiple instances share the file (dedup case), the **first** matching instance wins; the alternative — applying the most restrictive retention across all owners — is documented as a future enhancement.
2. **If `instance_id IS NULL`** (legacy / un-scoped file) → the file is **not** deleted by the per-instance sweep; it remains under the workspace default's `deleteExpiredUploadFiles(...)` path, which is the existing `effectiveRetentionDays` path. This is FR-010 verbatim.
3. **If `instance_id` is present** → look up the instance's `WhatsAppInstance.settings.uploads_cleanup.*` keys. Apply D-8's state machine. The resolved value is the per-file retention.
4. **If the instance has been deleted (soft-delete)** → the file is skipped. Pending operations referencing the deleted instance are skipped. (Per EC-2.)

This resolution is O(1) DB round-trips for the iteration: the bulk SELECT in D-4 already loaded every instance's settings, so the per-file lookup is an in-memory map lookup keyed by `instance_id`.

---

## Authorization

Per D-10, the existing `settings.uploads_cleanup` permission is reused. Mapping:

| Permission | Used by |
|---|---|
| `settings.uploads_cleanup:read` | GET endpoints on the new resource |
| `settings.uploads_cleanup:write` | `PUT /api/instances/{id}/uploads-cleanup` |
| `settings.uploads_cleanup:execute` | `POST /api/instances/{id}/uploads-cleanup/run` and the existing `POST /api/org/uploads-cleanup/run` |

Helpers `canAccessUploadsCleanupSettings`, `canWriteUploadsCleanupSettings`, `canExecuteUploadsCleanup` already exist in `uploads_cleanup_http.go:15-27`. The plugin's handlers call them directly. The audit `ActorUserID` is the user who passed the `write` permission check; the audit `ActorEmail` is `User.Email` looked up at write time (a single extra read inside the same request — acceptable; no N+1 issue because writes are infrequent).

---

## Validation rules summary

| Rule | Source | Error code | HTTP status |
|---|---|---|---|
| `retention_days` integer in `[0, 3650]` | FR-001, A-006, D-7 | `uploads_cleanup_retention_days` | 400 |
| `retention_days` required when `inherit = false` | FR-005, D-8 | `uploads_cleanup_retention_days_required` | 400 |
| `inherit` must be a boolean | D-8 | `uploads_cleanup_inherit_invalid` | 400 |
| Caller must have `settings.uploads_cleanup:write` | FR-011, D-10 | (omitted, 403 message) | 403 |
| Instance must belong to caller's org | Tenant scoping §6.5, FR-011 | `instance_not_found` | 404 |
| Manual run when retention resolves to `0` | FR-008, D-6 | `uploads_cleanup_disabled` | 400 |
| Manual run when another run is in progress | FR-007, D-5 | (omitted, 409 message) | 409 |
| History `limit` outside `[1, 100]` | §5.4 | `invalid_limit` | 400 |

All error codes are i18n-friendly; the human messages are localized in the frontend (D-11).

---

## Migration plan

| Step | Action | Tool | Risk |
|---|---|---|---|
| 1 | Add `InstanceUploadsCleanupAudit` model to the plugin | `db.AutoMigrate(&InstanceUploadsCleanupAudit{})` in plugin's `Migrate(db)` | None — additive table only |
| 2 | Add composite index via `getIndexes()` in the plugin's `Migrate` | `db.Exec("CREATE INDEX IF NOT EXISTS ...")` | None — `IF NOT EXISTS` is idempotent |
| 3 | Backfill `uploads_cleanup.inherit = true` on every existing instance (idempotent) | `UPDATE whatsapp_instances SET settings = settings || '{"uploads_cleanup":{"inherit":true}}' WHERE settings->'uploads_cleanup' IS NULL` | None — additive, idempotent, no data loss |

Step 3 is documented as a pre-migration fix in the plugin's `Migrate` and runs before `AutoMigrate`. It is the equivalent of the constitution §6.8 "pre-migration fix" pattern, kept inside the plugin so the core is untouched.

---

## Open questions handed to Phase 2 (tasks)

| ID | Question | Default until told otherwise |
|---|---|---|
| C-1 | Audit retention policy | Forever (matches `AgentSelectionAuditEvent` precedent) |
| Q-OPT-1 | Should the history list show the **diff** (e.g., "5 → 30 days") or the raw old/new values? | Show both labels: "Old: 5 / New: 30 (days)" |
| Q-OPT-2 | Should the "Inherit workspace default" toggle reset (clear) `retention_days` on the backend, or keep the old value for "undo"? | Keep the old value; UI just hides the input. A future "Restore" button can be added without migration. |
