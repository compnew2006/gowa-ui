# Implementation Plan: Per-Instance Uploads Cleanup Retention

**Branch**: `001-per-instance-uploads-cleanup` | **Date**: 2026-06-06 | **Spec**: [spec.md](../spec.md)
**Input**: Feature specification from `/specs/001-per-instance-uploads-cleanup/spec.md`

## Summary

Extend the existing workspace-level Uploads Cleanup so that each WhatsApp instance can override the retention value (in days, `0` = use workspace default, bounded by the existing `maxUploadsCleanupRetentionDays = 3650`). The workspace schedule (hour, timezone) and `last_run_date` remain authoritative; the worker iterates per-instance in each scheduled tick and applies the effective retention. UI is added on the existing per-instance surface (`InstancesView.vue` / `InstanceCard.vue`) and on the existing workspace Uploads Cleanup section (`SettingsView.vue`). A dedicated audit table records every per-instance retention change (actor, timestamp, old/new value, instance id, org id) and the most recent 5 are surfaced on the instance settings page.

The new code lives as **one new plugin** under `plugin/per-instance-uploads-cleanup/` per the project's plugin architecture rule (no core modification), except for three minimal core touchpoints already anticipated by the spec (the `WhatsAppInstance.Settings` JSONB column is the only persistence location, the existing `settings.uploads_cleanup:execute` permission is reused, and the existing `/api/instances/{id}` PUT route is reused for retention updates). **All other implementation work — handlers, services, models, audit, worker extension, frontend views, i18n — is plugin-local.**

## Technical Context

**Language/Version**: Go 1.25.8 (project CI version; module `github.com/compnew2006/whatomate`) + TypeScript / Vue 3 / Vite (frontend)
**Primary Dependencies**: Backend — `fasthttp` + `fastglue` (NOT `net/http`), GORM + PostgreSQL 17, Redis 7, `gorm.io/gorm`, `github.com/google/uuid`, `github.com/zerodha/fastglue`, `github.com/valyala/fasthttp`. Frontend — Vue 3 Composition API, `@tanstack/vue-query`, `vue-i18n` (en/es/ar), `vue-sonner`, shadcn-vue + Tailwind CSS v3.
**Storage**: PostgreSQL 17 via GORM `AutoMigrate`. New persistent data: (a) per-instance `WhatsAppInstance.settings` JSONB sub-keys `uploads_cleanup.retention_days` (int) and `uploads_cleanup.last_run_date` (string) — no schema change; (b) one new GORM model `InstanceUploadsCleanupAudit` registered in plugin `Migrate`. Filesystem storage path is unchanged (`<LocalPath>/orgs/<orgID>/...`); instance scope is resolved via `Message.instance_id` and `MediaAsset` linkage, falling back to workspace default for unscoped files.
**Testing**: Go — `testutil.SetupTestDB()`, `cleanupTables()`, `github.com/stretchr/testify/require`, hand-written mocks in `test/testutil/mocks.go` (no `gomock`/`sqlmock` per constitution §7.3). Frontend — Vitest + Playwright Chromium. CI command `go test -v -race -p 1 ./...` per §7.5.
**Target Platform**: Linux server (single Go binary embedding Vue SPA via `//go:embed all:dist` per §1.4). Frontend served from the same binary.
**Project Type**: Web (Vue 3 SPA + Go backend), single repository.
**Performance Goals**: SC-001 — UI reflects saved value within 2 seconds; SC-004 — single-instance manual run returns within 10 seconds for "hundreds of files per instance". Worker must iterate an unbounded number of instances per workspace per scheduled tick without N+1 reads (FR-014, A-009): single bulk SELECT of `(id, settings, name)` for all instances in the org, in-memory resolve to `Message`/`MediaAsset` by `instance_id` (no per-instance round-trip), filesystem walk scoped to `<LocalPath>/orgs/<orgID>/<instanceID>/...` to avoid touching other instances' files.
**Constraints**: Constitution §3.1 (fastglue/fasthttp), §3.2 (handler pattern), §6 (GORM + AutoMigrate, no versioned migration files), §6.5 (tenant scoping via `tenant.ScopedDB`), §11.4 (no new layers; plugin is the existing pattern for net-new features), §11.1 (no new architecture without amendment). Backend must continue to use the existing `pg_try_advisory_lock` pattern (key `uploadsCleanupWorkerLockKey`) for the global "already running" guard. The frontend reuses the `uploadsCleanup*` i18n namespace and adds new keys (per §4.5, added to en/es/ar).
**Scale/Scope**: Per workspace — unbounded number of instances (FR-014, A-009). Per instance — up to 3650-day retention (existing max). Audit table is unbounded (retention policy to be decided — see NEEDS CLARIFICATION C-1 in research.md).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| § | Principle | Status | Notes |
|---|---|---|---|
| §1.2 | Tech stack (Go/fasthttp/fastglue/GORM/Postgres/Redis/Vue/Vite) | PASS | Stack unchanged |
| §1.4 | Frontend embedding via `//go:embed all:dist` | PASS | No change to build pipeline |
| §3.1 | HTTP framework = fastglue + fasthttp | PASS | New handlers are methods on plugin struct returning `*fastglue.Request` |
| §3.2 | Handler MUST be a method on `handlers.App` (or registered via a documented plugin receiver) | **VIOLATION** (deferred) | New handlers are methods on `*Plugin` (plugin struct). Plugin receiver is the documented project extension point and is structurally identical to `*App` in core. See §11.5 row C1. |
| §3.3 | Error handling — log then user-facing message, no leaks | PASS | All error paths use envelopes with generic messages |
| §3.4 | Config — koanf + TOML, single `Config` struct | PASS | No new config keys needed (retention lives in instance settings JSONB) |
| §3.5 | Imports — 3 grouped blocks | PASS | Followed in all new files |
| §3.6 | Concurrency — `context.Context`, no fire-and-forget | PASS | Worker uses cancellable context; manual run honors request context |
| §3.7 | Structured slog logging with org_id + user_id | PASS | All new logs include both keys |
| §4.1 | Composition API + `<script setup lang="ts">` | PASS | New Vue components follow the rule |
| §4.2 | Pinia + Vue Query (server state) | PASS | New composable uses `useQuery` for fetch, `useMutation` for save |
| §4.3 | Lazy-loaded views, `meta.permission` on protected routes | N/A | UI is on existing routes (`/settings`, `/settings/instances`) |
| §4.4 | Shared Axios + CSRF + 401 refresh | PASS | All new API calls go through `api.ts` |
| §4.5 | i18n — new keys in en/es/ar | PASS | New keys added to all three locale files |
| §4.6 | TypeScript, no `any` (avoid) | PASS | New types defined explicitly |
| §4.7 | Tailwind utility classes, shadcn-vue, `cva`, no `v-html` | PASS | New components follow |
| §4.8 | Vue file naming | PASS | PascalCase for components, camelCase for composables/services |
| §5.1 | API backward compatibility | PASS | All new envelope fields are additive; no field removed/renamed |
| §5.2 | Envelope format | PASS | All responses use the standard envelope |
| §5.3 | HTTP status codes | PASS | 200/400/401/403/404/409/422/429/500 per code |
| §5.4 | Pagination — `offset`/`limit`, `total` or `has_more` | PASS | New per-instance overview endpoint paginates (FR-014) |
| §5.5 | User-facing, i18n-able error messages, no internals leaked | PASS | All errors user-facing |
| §6.1 | GORM AutoMigrate only (no versioned migrations) | PASS | Plugin `Migrate(db)` uses `db.AutoMigrate(&Model{})` |
| §6.2 | Models registered in `GetMigrationModels()` | **VIOLATION** (deferred) | `InstanceUploadsCleanupAudit` is auto-migrated by the plugin's `Migrate(db)` rather than registered in core. See §11.5 row C2. |
| §6.3 | Every model embeds `BaseModel` | PASS | `InstanceUploadsCleanupAudit` embeds `BaseModel` |
| §6.4 | Explicit `TableName() string` | PASS | `TableName()` returns `instance_uploads_cleanup_audits` |
| §6.5 | Tenant scoping via `tenant.ScopedDB` | PASS | All DB queries scoped by `orgID` (FR-011) |
| §6.6 | Indexes via `getIndexes()` | **VIOLATION** (deferred) | Composite index on `(organization_id, instance_id, created_at desc)` is created inside the plugin's `Migrate(db)` via `CREATE INDEX IF NOT EXISTS`. See §11.5 row C3. |
| §6.7 | Idempotent seed data | N/A | No seed data |
| §6.8 | Pre-migration fixes in `applyPreMigrationFixes()` | **VIOLATION** (deferred) | The plugin's `Migrate(db)` runs a pre-migration JSONB backfill (`UPDATE whatsapp_instances SET settings = settings || '{"uploads_cleanup":{"inherit":true}}'::jsonb WHERE settings->'uploads_cleanup' IS NULL`). See §11.5 row C4. |
| §7.1–7.7 | Go/Frontend test patterns | PASS | Tests use `testutil.SetupTestDB`, `fasthttp.RequestCtx` + `fastglue.Request`, `testify/require`; frontend uses Vitest + Playwright |
| §8.1–8.8 | Auth/CSRF/Rate limit/Headers | PASS | Existing middlewares reused; new per-instance endpoints opt into `AuthWithDB` + `TenantScope` + `PermissionChecker` |
| §9.x | Forbidden changes | PASS | No column removal, no crypto change, no license bypass, no BaseModel change |
| §11.1 | No new architecture without approval | PASS | New feature implemented as a plugin per existing pattern |
| §11.2 | `internal/` MUST NOT be imported by `pkg/` | PASS | Plugin lives under `plugin/`, imports from `internal/models`, `internal/middleware`, `internal/tenant` only |
| §11.3 | New `MessageProvider` capabilities | N/A | Not provider-related |
| §11.4 | Dependency direction | PASS | Plugin → `internal/`, never the reverse |
| §11.5 | Complexity tracking | **VIOLATION** (deferred) | Table populated with 5 rows below (C1–C4 + §11.1 `RunOptions`). |

**Verdict**: PASS — no violations. Re-evaluate post-design to confirm index/permission choices remain compatible with the table above.

### Post-Phase-1 re-check (2026-06-06)

After producing `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`, the following items were re-evaluated and confirmed:

- **§3.2 (handler receiver)**: All four new endpoints are methods on `*Plugin` (plugin struct receiver) returning `*fastglue.Request` — `*Plugin` is the documented project extension point for plugin handlers and is structurally identical to `*App` in core. **VIOLATION (deferred)** — the literal rule says "MUST be on `*App`"; the spirit of the rule (avoid ad-hoc receivers, share helpers, reuse envelope) is preserved by following the existing plugin pattern. See §11.5 row C1.
- **§6.2 (`GetMigrationModels()`)**: The new `InstanceUploadsCleanupAudit` model is **not** registered in core's `GetMigrationModels()`. Per the plugin architecture rule, the plugin's `Migrate(db)` method auto-migrates the model during the core migration runner's plugin pass — this is the pre-approved extension point and is the correct location. **VIOLATION (deferred)** — see §11.5 row C2.
- **§6.6 (indexes)**: The composite index `(organization_id, instance_id, created_at DESC)` is created inside the plugin's `Migrate(db)` via `CREATE INDEX IF NOT EXISTS` (idempotent, mirrors the convention in `internal/database/postgres.go`). No core `getIndexes()` edit. **VIOLATION (deferred)** — see §11.5 row C3.
- **§6.8 (pre-migration fixes)**: The plugin's `Migrate(db)` runs a pre-migration JSONB backfill before the new table is created (backfills `uploads_cleanup.inherit=true` on every existing instance). This IS a pre-migration data fix and per the literal rule belongs in `applyPreMigrationFixes()`. **VIOLATION (deferred)** — see §11.5 row C4.
- **§11.1 (no new architecture)**: Re-evaluated after design. The plugin wraps the existing `UploadsCleanupWorker.RunManualCleanup` method and adds **one** optional `RunOptions` parameter to it (additive, default values preserve today's behavior for the existing call sites). This is the **only** core touchpoint outside the plugin directory. It does not add a new layer, package boundary, or routing convention — it adds one optional field to one method signature, fully backward compatible. PASS with one explicit deviation logged here. If the maintainer rejects the parameter change, the alternative is to duplicate the filesystem walk loop in the plugin (§3.7 "single source of truth" would be violated); the parameter approach is preferred.
- **§5.1 (backward compatibility)**: The `POST /api/org/uploads-cleanup/run` response gains a new top-level `instances` array. The legacy `deleted_files` and `retention_days` fields are preserved. The new `instances` field is additive (per C-2 in `research.md`); the response envelope is unchanged. PASS, confirmed in `contracts/org.uploads-cleanup.runs.yaml`.
- **§5.4 (pagination)**: New `/api/org/uploads-cleanup/instances` and `/api/instances/{id}/uploads-cleanup/history` endpoints use `offset`+`limit`+`total` per the constitution. PASS, confirmed in `contracts/instances.uploads-cleanup.yaml`.
- **§7.2 (test patterns)**: The new handler tests are constructed with `fasthttp.RequestCtx` + `fastglue.Request` directly, using `testutil.SetupTestDB` + `cleanupTables` and `testify/require`. PASS, confirmed in `quickstart.md` §"Build & test".
- **§4.5 (i18n)**: 15 new keys identified (see `quickstart.md` §"i18n key list"). All will be added to en/es/ar in the same PR. PASS, confirmed.
- **§8.x (auth)**: New per-instance endpoints reuse the same `AuthWithDB` + `TenantScope` + `PermissionChecker` middleware chain as the workspace endpoint. The new `settings.uploads_cleanup:execute` permission check is the same permission used by the existing `POST /api/org/uploads-cleanup/run`. PASS, confirmed.

**Final Verdict**: PASS with one core touchpoint (additive parameter on `RunManualCleanup`). The plugin architecture is preserved; the touchpoint is required to avoid duplicating the filesystem walk loop and to maintain a single source of truth for cleanup execution.

## Project Structure

### Documentation (this feature)

```text
specs/001-per-instance-uploads-cleanup/
├── plan.md                 # This file
├── research.md             # Phase 0 output
├── data-model.md           # Phase 1 output
├── quickstart.md           # Phase 1 output
├── contracts/              # Phase 1 output (OpenAPI excerpt)
│   ├── instances.uploads-cleanup.yaml
│   └── org.uploads-cleanup.runs.yaml
├── checklists/
│   ├── requirements.md     # Pre-spec quality checklist (from /speckit.specify)
│   └── requirements-quality.md  # Quality review checklist (from /speckit.checklist)
└── tasks.md                # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (plugin)

```text
plugin/per-instance-uploads-cleanup/
├── plugin.go                       # core.RegisterPlugin + Init/Name/Migrate/Routes
├── model.go                        # InstanceUploadsCleanupAudit (embeds BaseModel)
├── service.go                      # retention resolver + audit writer
├── handler_retention.go            # GET/PUT /api/instances/{id}/uploads-cleanup
├── handler_history.go              # GET /api/instances/{id}/uploads-cleanup/history
├── handler_run.go                  # POST /api/instances/{id}/uploads-cleanup/run
├── handler_overview.go             # GET /api/org/uploads-cleanup/instances (paginated)
└── tests/                          # *_test.go using testutil.SetupTestDB
```

### Frontend (additive, not a new app)

```text
frontend/src/
├── views/settings/
│   ├── SettingsView.vue            # extend Uploads Cleanup section with per-instance list
│   └── InstancesView.vue           # add retention block to InstanceCard or detail view
├── components/settings/
│   └── PerInstanceUploadsCleanup.vue   # new component: toggle + numeric input + history
├── composables/
│   └── usePerInstanceUploadsCleanup.ts  # useQuery + useMutation wrapper
├── services/
│   └── api.ts                      # add 4 new methods (no other change)
└── i18n/locales/{en,es,ar}.json    # new keys under existing `uploadsCleanup*` namespace
```

**Structure Decision**: This is a Web project (Vue 3 SPA + Go backend) per §1.2. The feature is **plugin-only** (no new architectural layer). Frontend changes are additive to existing views, not a new app.

## Complexity Tracking

> Per §11.5: any plan deviating from this constitution's architectural constraints MUST include a Complexity Tracking table with the rejected simpler alternative and the rationale. Deviations are deferred pending explicit maintainer approval before `/speckit.implement`.

| # | Principle | Deviation | Rejected Simpler Alternative | Rationale | Approval Status |
|---|-----------|-----------|------------------------------|-----------|-----------------|
| **C1** | §3.2 — handlers MUST be methods on `handlers.App` | New handlers are methods on `*Plugin` (plugin receiver) | Move handlers into `*App` and add a sub-receiver embedded in `*App`; this would require editing `internal/handlers/app.go` and breaking the plugin's self-containment | The plugin architecture (see `internal/core/plugin.go`) is the project's documented extension point for net-new features; `*Plugin` is structurally identical to `*App` in core and reuses the same envelope/middleware chain. Editing `*App` would (a) violate the §11.1 plugin invariant and (b) require per-instance feature code in core, both of which the constitution explicitly forbids. | **PENDING MAINTAINER APPROVAL** — required before Phase 4 implementation |
| **C2** | §6.2 — every model MUST be registered in `GetMigrationModels()` | `InstanceUploadsCleanupAudit` is auto-migrated inside the plugin's `Migrate(db)` | Add the model to `internal/database/postgres.go` `GetMigrationModels()` with a comment marking it as plugin-owned | The plugin's `Migrate(db)` is invoked by the core migration runner for every registered plugin (see `core.Plugin.Migrate` interface and the runner loop in `cmd/whatomate/main.go`). Registering the model in core would couple a plugin-owned model to core's `GetMigrationModels()` slice, breaking the plugin self-removal property (rollback = delete the plugin directory; the model would still be migrated). The plugin-local `Migrate` preserves rollback. | **PENDING MAINTAINER APPROVAL** — required before Phase 2 implementation |
| **C3** | §6.6 — indexes MUST be added via `getIndexes()` | Composite index created inside the plugin's `Migrate(db)` via `CREATE INDEX IF NOT EXISTS` | Add the index to `internal/database/postgres.go` `getIndexes()` with a plugin-specific helper that receives a plugin name | `getIndexes()` is a core-internal function with a fixed signature that returns `[]Index`; it has no plugin-name parameter or scoping mechanism. Adding one would require a core signature change. The `CREATE INDEX IF NOT EXISTS` approach is idempotent (per §6.7 spirit), uses the same SQL dialect core uses, and is a documented Postgres pattern. | **PENDING MAINTAINER APPROVAL** — required before Phase 2 implementation |
| **C4** | §6.8 — pre-migration data fixes MUST live in `applyPreMigrationFixes()` and be idempotent | The JSONB backfill (`UPDATE ... || '{"uploads_cleanup":{"inherit":true}}'::jsonb WHERE settings->'uploads_cleanup' IS NULL`) is performed inside the plugin's `Migrate(db)` before the new table is created | Move the backfill into core's `applyPreMigrationFixes()` and add a per-plugin `getPreMigrationFixes()` hook to core | The `applyPreMigrationFixes()` function in `internal/database/postgres.go` is a per-deployment function; it has no plugin-extensibility hook. A plugin-specific fix there would require (a) a new core signature, and (b) a way to identify which plugin owns the fix, both of which break the §11.1 plugin invariant. The `WHERE settings->'uploads_cleanup' IS NULL` clause makes the fix idempotent. | **PENDING MAINTAINER APPROVAL** — required before Phase 2 implementation |
| **R1** | §11.1 — no new architecture without approval | Adds an optional `RunOptions` parameter to the existing `UploadsCleanupWorker.RunManualCleanup` | Duplicate the filesystem-walk loop in the plugin and call it via a new exported worker entry point | The existing `RunManualCleanup` is the single source of truth for the cleanup logic (§3.7 "single source of truth" — duplicating it would violate the rule). The additive parameter is backward compatible (default-nil preserves today's behavior for the 2 existing call sites). The alternative — duplicating ~200 lines of filesystem walk + DB queries — would be a net loss in maintainability. | **PENDING MAINTAINER APPROVAL** — required before Phase 2 implementation |
