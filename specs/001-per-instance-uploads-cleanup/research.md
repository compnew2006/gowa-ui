# Research: Per-Instance Uploads Cleanup Retention

**Date**: 2026-06-06
**Branch**: `001-per-instance-uploads-cleanup`
**Spec**: [spec.md](../spec.md)
**Plan**: [plan.md](../plan.md)

This document consolidates findings for every "NEEDS CLARIFICATION" or open technical decision identified in [plan.md](../plan.md#technical-context). Each decision records: what was chosen, why, and what alternatives were rejected.

---

## D-1. Per-instance retention storage location

**Decision**: Store retention in the existing `whatsapp_instances.settings` JSONB column, under a new sub-namespace `uploads_cleanup.*` (keys: `retention_days` int, `last_run_date` string, plus the `inherit` boolean required by the UI). No new dedicated column, no new model field.

**Rationale**:
- The instance already carries a `Settings JSONB` column with per-feature sub-namespaces (see `auto_reject_calls.*`, `auto_campaign.*`, `tag_color`, `chat_close_rating.*` in `frontend/src/lib/instance-*.ts` and `internal/handlers/instances.go:321`). Adding to the same pattern is idiomatic.
- A-007 ("stored as structured data on the instance record — in its `Settings` JSON column or a dedicated column") explicitly permits either; the JSONB route avoids an `AutoMigrate` column add and keeps the schema stable.
- The existing `UpdateInstance` handler already validates and persists `req.Settings` through `waManager.ValidateInstanceSettings` and `waManager.EnsureInstanceSettingsDefaults`, so the persistence path is already built.

**Alternatives considered**:
- **Dedicated columns** (`retention_days int`, `last_run_date string`, `inherit bool` on `whatsapp_instances`): more indexable, but requires GORM model changes and breaks A-007's flexibility — and we don't need to index on retention values for any query.
- **EAV table**: rejected — overkill for three values; the JSONB pattern is already in use.

---

## D-2. Per-instance audit log storage

**Decision**: Introduce one new GORM model `InstanceUploadsCleanupAudit` in the plugin with a dedicated table `instance_uploads_cleanup_audits`. Columns: `id`, `created_at`, `updated_at`, `deleted_at`, `organization_id` (uuid, indexed), `instance_id` (uuid, indexed), `actor_user_id` (uuid, nullable, indexed), `actor_email` (string, denormalized snapshot), `old_retention_days` (int, nullable for the first-write case), `new_retention_days` (int, nullable when toggling inherit on), `old_inherit` (bool, nullable), `new_inherit` (bool, nullable), `reason` (string, optional). Composite index on `(organization_id, instance_id, created_at DESC)` to support the "last 5 changes" fetch.

**Rationale**:
- FR-016 mandates actor, timestamp, old value, new value. A dedicated model gives us those fields with first-class indexing.
- Mirrors the existing `AgentSelectionAuditEvent` pattern in `internal/models/agent_selection.go:178-201` (actor + event fields + metadata). Following the same shape keeps the codebase uniform.
- The `agent_user_id` is nullable to support the future case of system-driven changes (e.g., a future bulk-edit admin tool) without an event without an actor.
- Denormalizing `actor_email` is a deliberate trade-off: it lets the UI show the actor even if the user record is later soft-deleted, and avoids a join on the "last 5" path.

**Alternatives considered**:
- **Reuse a generic `ActivityLog` table**: there is no such table in the project. Creating one is out of scope for this feature.
- **Embed audit records as JSON inside the instance's `Settings` JSONB**: would lose query/index ability and complicate the "last 5" fetch with no benefit.
- **Hard-delete old audit rows**: rejected — operationally desirable to keep a history; retention policy is C-1 (see C-1 below).

---

## D-3. Resolution of "file belongs to which instance"

**Decision**: Use `messages.instance_id` (the existing FK on `messages.instance_id` per `internal/models/models.go:513`) as the source of truth for which instance owns a media file. If `instance_id IS NULL` (legacy / un-scoped), fall back to the **workspace** default retention — i.e., un-scoped files are **not** deleted on the per-instance path; they remain on the workspace default's `deleteExpiredUploadFiles(...)` path.

**Rationale**:
- FR-009 requires the worker to determine instance ownership. `messages.instance_id` is already populated for messages created since the instance-id feature shipped.
- FR-010 says "skip cleanup for files that cannot be associated with any instance and fall back to the workspace default retention" — i.e., the worker must detect un-scoped files and route them to the workspace path. This matches the existing `deleteExpiredUploadFiles` semantics in `uploads_cleanup_worker.go:344`.
- The storage path `<LocalPath>/orgs/<orgID>/...` is already used; the worker iterates per-directory, queries messages whose `media_url` matches the relative path, and gets the instance id from the row.

**Alternatives considered**:
- **Re-key the storage path to include instance_id** (`<LocalPath>/orgs/<orgID>/<instanceID>/...`): would simplify per-instance sweeps, but it would orphan existing files and require a one-time data migration — out of scope for this iteration.
- **Query `media_assets` directly**: `media_assets` does not currently have an `instance_id` column; adding it would be a schema change. Not justified for this iteration.

---

## D-4. Worker iteration strategy (FR-014, A-009)

**Decision**: The worker performs a single bulk read of all instances in the org at the start of the tick: `SELECT id, name, settings FROM whatsapp_instances WHERE organization_id = ? AND deleted_at IS NULL`. For each instance, it computes the effective retention (instance `retention_days` if set and not in inherit mode; otherwise workspace default). It then walks the directory `<LocalPath>/orgs/<orgID>/...` exactly once and dispatches each expired file to the resolved instance bucket (or to the workspace bucket if un-scoped). This is one DB read for instance configs and one filesystem walk per tick — strictly O(1) round-trips for iteration.

**Rationale**:
- FR-014 forbids N+1 reads during iteration. The bulk SELECT + single filesystem walk meets that.
- A-009 says "iterate all instances per scheduled tick using an efficient bulk read of instance configs and resolves files to instances without N+1 DB lookups per instance."
- The existing worker already calls `loadOrganizationSchedules` (one bulk read of all orgs' settings) and `deleteExpiredUploadFiles` (one filesystem walk). Extending it means swapping the per-file retention lookup from "workspace default" to "resolved per-instance retention" without changing the iteration shape.

**Alternatives considered**:
- **Per-instance subdirectory walk** (`<LocalPath>/orgs/<orgID>/<instanceID>/...`): would isolate sweeps but requires a path migration; deferred.
- **Map-reduce via Redis Streams**: out of proportion for the workload.

---

## D-5. Concurrent-run guard

**Decision**: Reuse the existing `pg_try_advisory_lock(uploadsCleanupWorkerLockKey)` for the global guard (already present in `uploads_cleanup_worker.go:166-178`). A new **in-process mutex** is added to the `UploadsCleanupWorker` struct (`sync.Mutex`) to serialize the per-instance manual run against any scheduled tick that might fire while the manual run is in progress. The HTTP handler `POST /api/instances/{id}/uploads-cleanup/run` returns `409` with message `"uploads cleanup is already running"` when the lock is held.

**Rationale**:
- Existing behavior is preserved (org-wide "already running" message — see `uploads_cleanup_http.go:42-50`).
- FR-007 requires preventing concurrent runs. The existing advisory lock already does this across processes; the in-process mutex covers the case where the same process tries to run two ticks.
- 409 (Conflict) is the appropriate status per constitution §5.3 (state mismatch).

**Alternatives considered**:
- **Per-instance advisory locks**: more granular, but the spec only requires preventing overlap, not permitting parallel per-instance runs in the same process. Keep it simple.
- **Redis-based locks**: adds a dependency and a failure mode for a problem that's already solved with the existing PG advisory lock.

---

## D-6. Manual per-instance run API

**Decision**: `POST /api/instances/{id}/uploads-cleanup/run` (handler `handler_run.go` in the plugin). Body: empty. Response: `200` with envelope `data: { message, deleted_files, retention_days, instance_id, instance_name }` on success; `400` if the resolved retention is `0` (per existing `errUploadsCleanupDisabled` branch in `uploads_cleanup_http.go:43-49`); `401/403` for auth/permission; `409` if a run is already in progress; `404` if the instance does not belong to the org.

**Rationale**:
- Mirrors the existing `POST /api/org/uploads-cleanup/run` pattern at `cmd/whatomate/main.go:1792` so the frontend can reuse the same idiom.
- FR-008 requires per-instance results in the response (instance id/name, retention used, deleted count).
- The handler reuses the worker method `RunManualCleanup`, extended to accept an optional `instanceID *uuid.UUID` to scope the sweep.

**Alternatives considered**:
- **A new worker method per-instance**: would duplicate `RunManualCleanup`; instead, parametrize the existing method.
- **Async via Redis queue**: overkill; the run completes within SC-004's 10-second budget.

---

## D-7. Retention input validation (FR-012)

**Decision**: The per-instance retention value is an integer in `[0, maxUploadsCleanupRetentionDays]` (the existing constant, currently `3650`). `0` = instance-level disable (overrides workspace default). Negative or non-integer input → `400` with `code: "uploads_cleanup_retention_days"` and the same human message used by the existing workspace validator (`uploads_cleanup retention must be a whole number between 0 and 3650`).

**Rationale**:
- FR-001 explicitly bounds by the existing max. A-006 says "the bound itself is not expanded".
- The existing workspace validator at `internal/handlers/organization.go:186-188` returns the same `code` shape (`uploads_cleanup_retention_days`), so the frontend can render the error against the same field.

**Alternatives considered**:
- **Per-instance max override**: explicitly disallowed by A-006.

---

## D-8. "Inherit workspace default" toggle

**Decision**: The toggle stores a boolean `uploads_cleanup.inherit` in `whatsapp_instances.settings`. When `inherit = true`:
- The numeric input is disabled in the UI and the effective value is shown as a read-only preview computed from the workspace default.
- The worker treats the instance as having no override — i.e., the workspace default applies.
When `inherit = false`:
- The user enters a custom value (`retention_days` int, can be `0` for instance-level disable, or `>0` for N days).
- The worker uses the instance value verbatim.

**Rationale**:
- The Clarifications Q1 answer (in `spec.md` line 102) explicitly chose the "Inherit default" toggle over an "unset" semantic.
- Storing `inherit` separately from `retention_days` lets us distinguish "user said inherit" from "no value ever set" — important for FR-005's "show the effective value as the workspace default" behavior.

**Alternatives considered**:
- **Use `retention_days = -1` or `nil` as the "inherit" sentinel**: rejected — fragile, and collides with the validator's "0 = disabled" semantic.
- **Always store a value, treat "no value" as inherit**: rejected — can't distinguish "user set 0 to disable" from "user has never set anything".

---

## D-9. History endpoint pagination

**Decision**: `GET /api/instances/{id}/uploads-cleanup/history?limit=5&offset=0` (default `limit=5`, max `100`). Returns the most recent `limit` audit rows ordered by `created_at DESC`. The frontend calls this once on the instance settings page render. Response envelope includes `data: { entries: [...], total }`.

**Rationale**:
- FR-016 says "surface a 'Last 5 changes' history list" — the spec names the count (5).
- Constitution §5.4 mandates `offset` + `limit` for list endpoints. The 5-row cap is enforced as a default `limit`; the endpoint still supports pagination in case the UI later wants to show more.

**Alternatives considered**:
- **Cursor pagination**: not warranted at 5 rows.
- **Embed history into the retention GET response**: the responses have different cache lifetimes; separate is cleaner.

---

## D-10. Permission set reuse

**Decision**: Reuse the existing permission `settings.uploads_cleanup` with the three existing actions `read`, `write`, `execute` (per `internal/models/roles.go:55,84-89,122-124`). No new permission is registered.

| Action | Used by |
|---|---|
| `settings.uploads_cleanup:read` | `GET /api/instances/{id}/uploads-cleanup`, `GET /api/instances/{id}/uploads-cleanup/history`, `GET /api/org/uploads-cleanup/instances` |
| `settings.uploads_cleanup:write` | `PUT /api/instances/{id}/uploads-cleanup` |
| `settings.uploads_cleanup:execute` | `POST /api/instances/{id}/uploads-cleanup/run`, `POST /api/org/uploads-cleanup/run` (existing) |

**Rationale**:
- A-002 explicitly says the existing permission set is reused. No new permission is required.
- Mirrors the helper functions `canAccessUploadsCleanupSettings`, `canWriteUploadsCleanupSettings`, `canExecuteUploadsCleanup` in `uploads_cleanup_http.go:15-27` — the new plugin handlers will check the same permissions.

**Alternatives considered**:
- **New per-instance permission (`settings.uploads_cleanup.instance:execute`)**: rejected by A-002. Also creates a 1-permission-on-every-resource surface that adds management overhead with no behavioral benefit.
- **Delegating to `p.app.HasPermission`**: the core `HasPermission` uses `getUserPermissionsCached` which requires a full `App` with Redis, `Log`, and the `permissions`/`role_permissions` tables. Plugins cannot easily reconstruct this in tests. Instead, the plugin implements its own `hasPermission` that queries `users.is_super_admin` (matching the core's behavior at `cache.go:534`) and falls back to the `custom_role_permissions` JOIN chain with explicit `deleted_at IS NULL` soft-delete filtering (GORM's `Raw()` does not add soft-delete scopes automatically).

---

## D-11. i18n namespace placement

**Decision**: New keys go under the existing `settings.uploadsCleanup*` namespace in `frontend/src/i18n/locales/{en,es,ar}.json`. Specifically:
- `settings.uploadsCleanupInstanceRetentionLabel`
- `settings.uploadsCleanupInstanceRetentionDesc`
- `settings.uploadsCleanupInstanceInheritLabel`
- `settings.uploadsCleanupInstanceInheritHelp`
- `settings.uploadsCleanupInstanceRunNow`
- `settings.uploadsCleanupInstanceHistoryTitle`
- `settings.uploadsCleanupInstanceHistoryEmpty`
- `settings.uploadsCleanupInstanceHistoryActor`
- `settings.uploadsCleanupInstanceOverviewTitle`
- `settings.uploadsCleanupInstanceOverviewEffectiveCustom`
- `settings.uploadsCleanupInstanceOverviewEffectiveDefault`
- `settings.uploadsCleanupInstanceOverviewEffectiveDisabled`
- `settings.uploadsCleanupInstanceOverviewLastRun`

**Rationale**:
- Constitution §4.5 requires all user-facing text to use `vue-i18n` and all three shipped locales to be updated. Reusing the namespace keeps related strings together and simplifies translator handoff.

**Alternatives considered**:
- **New `instances.uploadsCleanup*` namespace**: technically correct, but splits cleanup-related translations across two trees.

---

## C-1. Audit retention policy (open — pending product decision)

**Status**: NEEDS CLARIFICATION from product owner.

**Context**: Audit rows accumulate indefinitely. The spec mandates "actor, timestamp, old value, new value for every per-instance retention change". It does not specify how long audit rows must be retained.

**Options**:
- (a) **Forever** — simplest; storage cost scales with the rate of retention changes, which is expected to be low (per admin action).
- (b) **N years (e.g., 7)** — typical compliance default; requires a TTL job.
- (c) **Workspace-configurable** — most flexible; requires a setting and a background sweep.

**Default if not decided before Phase 2**: Forever (option a) — matches the existing `AgentSelectionAuditEvent` precedent (no retention job in the codebase).

**Action item**: surface in the `/speckit.tasks` step as an explicit "decide and document" task; implementation assumes `forever` unless told otherwise.

---

## C-2. Worker scope for the existing `/api/org/uploads-cleanup/run` endpoint

**Status**: Resolved by D-4.

**Decision**: The existing org-wide manual run continues to sweep all instances in the workspace using each instance's effective retention (per FR-006, "cleanup is run per-instance within each scheduled execution" and US2.AS3, "the admin uses 'Run cleanup now' at the workspace level ... runs against all instances using each instance's effective retention"). The response payload gains one new field: `instances: [{ instance_id, instance_name, retention_used, deleted_files }]`. The legacy `deleted_files` and `retention_days` top-level fields are kept for backward compatibility (§5.1) and represent the totals across all instances.

**Rationale**:
- US2.AS3 is explicit about the behavior.
- §5.1 forbids removing/renaming existing fields, but permits additive changes.

---

## C-3. Resolving the "max retention bound" reference (A-006)

**Status**: Resolved by D-7.

**Decision**: The constant `maxUploadsCleanupRetentionDays = 3650` defined in `internal/handlers/uploads_cleanup_settings.go:17` is the canonical bound. The plan references it by name (not by literal value) so that future updates to the constant do not require plan changes.

---

## C-4. Frontend placement of the per-instance retention UI

**Status**: Resolved.

**Decision**: The retention block lives inside the existing `InstanceCard.vue` (file `frontend/src/components/whatsmeow/InstanceCard.vue`, 687 lines). The block is a new sub-section that follows the existing sub-section pattern (see `auto_reject_calls` block at line 429 and `auto_campaign` block at line 474). A new sub-component `frontend/src/components/settings/PerInstanceUploadsCleanup.vue` owns the toggle, the numeric input, the run-now button, and the history list. The "Last 5 changes" history is rendered beneath the block in the same component.

**Rationale**:
- Reuses the existing per-instance surface (A-008) without introducing new navigation.
- Mirrors the structure of the existing per-instance settings (auto-reject, auto-campaign) so the new feature is visually familiar.

**Alternatives considered**:
- **A new dedicated `/settings/instances/{id}/cleanup` route**: out of scope per A-008 ("No new top-level navigation is introduced").
- **Inside the workspace Uploads Cleanup section only**: contradicts US1 ("Configure per-instance retention from the Instance settings").

---

## Summary of decisions

| # | Decision | Reference |
|---|---|---|
| D-1 | Retention in `whatsapp_instances.settings` JSONB | data-model.md §Entity: WhatsAppInstance |
| D-2 | New `instance_uploads_cleanup_audits` table | data-model.md §Entity: InstanceUploadsCleanupAudit |
| D-3 | File→instance via `messages.instance_id` | data-model.md §Resolution rules |
| D-4 | Bulk SELECT + single filesystem walk | design §Worker iteration |
| D-5 | Existing PG advisory lock + in-process mutex | design §Concurrency |
| D-6 | `POST /api/instances/{id}/uploads-cleanup/run` | contracts/instances.uploads-cleanup.yaml |
| D-7 | `[0, 3650]`, same code as workspace validator | contracts/instances.uploads-cleanup.yaml |
| D-8 | Explicit `inherit` boolean | data-model.md §Instance sub-settings |
| D-9 | `GET .../history?limit=5&offset=0` | contracts/instances.uploads-cleanup.yaml |
| D-10 | Reuse `settings.uploads_cleanup:read|write|execute` | data-model.md §Authorization |
| D-11 | Add keys under `settings.uploadsCleanup*` i18n namespace | quickstart.md §i18n |
| C-1 | Audit retention = `forever` (default) | open — confirm in tasks |
| C-2 | Org run reports per-instance breakdown | contracts/org.uploads-cleanup.runs.yaml |
| C-3 | Bound = `maxUploadsCleanupRetentionDays` (3650) | data-model.md §Validation |
| C-4 | Block lives in `InstanceCard.vue` | quickstart.md §Frontend |
| D-12 | Composables must use `unwrapResponse()` not `res.data` | bug fix T058 |

## D-12. Frontend composable response unwrapping

**Decision**: All composable queryFn/mutationFn callbacks must use `unwrapResponse()` from `@/lib/api-utils` instead of accessing `res.data` directly.

**Rationale**:
- The API uses fastglue's `SendEnvelope` which wraps responses in `{ status, data, message }`.
- Axios returns the full body as `res.data`, so `res.data` is the envelope, not the inner payload.
- Stores already use `unwrapResponse()` consistently; the per-instance uploads cleanup composables were the only ones that didn't.
- Accessing `res.data.entries` on the envelope object returns `undefined` because `entries` is actually at `res.data.data.entries`.
- This caused `Cannot read properties of undefined (reading 'length')` fatal crash on `/whatsapp/instances`.

**Files affected**: `frontend/src/composables/usePerInstanceUploadsCleanup.ts` (5 callbacks fixed), `frontend/src/components/settings/PerInstanceUploadsCleanup.vue` (defensive optional chaining added).
