# Quickstart: Per-Instance Uploads Cleanup Retention

**Date**: 2026-06-06
**Branch**: `001-per-instance-uploads-cleanup`
**Spec**: [spec.md](../spec.md) · **Plan**: [plan.md](../plan.md) · **Research**: [research.md](../research.md) · **Data Model**: [data-model.md](../data-model.md) · **Contracts**: [contracts/](../contracts/)

This quickstart is the developer's on-ramp. It explains the new plugin layout, what to wire up, and how to verify the feature locally.

---

## What ships in this feature

1. A new Go plugin `plugin/per-instance-uploads-cleanup/` exposing four new REST endpoints and a worker extension.
2. A new GORM model `InstanceUploadsCleanupAudit` in a new table `instance_uploads_cleanup_audits`.
3. A new frontend component `PerInstanceUploadsCleanup.vue`, a new composable `usePerInstanceUploadsCleanup.ts`, an extended workspace Uploads Cleanup section in `SettingsView.vue`, and a new sub-section in `InstanceCard.vue`.
4. New i18n keys under the existing `settings.uploadsCleanup*` namespace in `en.json`, `es.json`, `ar.json`.
5. **No** change to any model in `internal/models/`, **no** change to `internal/database/postgres.go`, **no** change to `cmd/whatomate/main.go` route registration. All new code is in the plugin; the core migration runner auto-discovers the plugin at startup.

---

## Plugin layout

```text
plugin/per-instance-uploads-cleanup/
├── plugin.go                       # core.RegisterPlugin + Init/Name/Migrate/Routes
├── model.go                        # InstanceUploadsCleanupAudit
├── service.go                      # Retention resolution, audit writer, in-process mutex
├── handler_retention.go            # GET/PUT /api/instances/{id}/uploads-cleanup
├── handler_history.go              # GET /api/instances/{id}/uploads-cleanup/history
├── handler_run.go                  # POST /api/instances/{id}/uploads-cleanup/run
├── handler_overview.go             # GET /api/org/uploads-cleanup/instances (paginated)
├── retention_test.go               # service.go unit tests (state machine, validation)
├── handlers_test.go                # handler tests (fasthttp.RequestCtx, testutil.SetupTestDB)
├── plugin_test.go                  # plugin self-registration smoke test
└── testdata/                       # fixture instances/orgs (functional-options style)
```

### `plugin.go` skeleton

```go
package perinstanceuploadscleanup

import (
    "github.com/compnew2006/whatomate/internal/core"
    "github.com/redis/go-redis/v9"
    "github.com/zerodha/fastglue"
    "gorm.io/gorm"
    "log/slog"
)

type Plugin struct {
    db    *gorm.DB
    rdb   *redis.Client
    log   *slog.Logger
    srv   *service
}

func init() { core.RegisterPlugin(Plugin{}) }

func (p Plugin) Name() string { return "per-instance-uploads-cleanup" }

func (p *Plugin) Init(db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
    p.db, p.rdb, p.log = db, rdb, log
    p.srv = newService(db, log)
    return nil
}

func (p *Plugin) Routes(g *fastglue.Glue) {
    // Authorization is per-handler (TenantScope + HasPermission check inside
    // each handler), not at route registration, to match the existing
    // /api/org/uploads-cleanup/run pattern in uploads_cleanup_http.go.
    g.GET("/api/instances/{id}/uploads-cleanup", p.handleGetRetention)
    g.PUT("/api/instances/{id}/uploads-cleanup", p.handlePutRetention)
    g.GET("/api/instances/{id}/uploads-cleanup/history", p.handleHistory)
    g.POST("/api/instances/{id}/uploads-cleanup/run", p.handleRun)
    g.GET("/api/org/uploads-cleanup/instances", p.handleOverview)
}

func (p *Plugin) Migrate(db *gorm.DB) error {
    // Pre-migration backfill (idempotent, see data-model.md §Migration plan)
    if err := db.Exec(`
        UPDATE whatsapp_instances
        SET settings = settings || '{"uploads_cleanup":{"inherit":true}}'::jsonb
        WHERE settings->'uploads_cleanup' IS NULL
    `).Error; err != nil {
        return err
    }
    // New model
    if err := db.AutoMigrate(&InstanceUploadsCleanupAudit{}).Error; err != nil {
        return err
    }
    // Composite index (idempotent)
    if err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_iuca_org_instance_created
        ON instance_uploads_cleanup_audits (organization_id, instance_id, created_at DESC)
    `).Error; err != nil {
        return err
    }
    return nil
}
```

The plugin is activated by a single blank import line in `cmd/whatomate/main.go` next to the existing plugin imports (per the project's plugin architecture rule). **This is the only edit to `cmd/whatomate/main.go`.**

---

## Worker extension (core touchpoint)

The existing `UploadsCleanupWorker` in `internal/handlers/uploads_cleanup_worker.go` is extended with one new method `RunManualCleanupForInstance(ctx, orgID, instanceID, now)` and one new field `instanceMu sync.Mutex`. The extension lives in a new file `internal/handlers/uploads_cleanup_worker_instance.go` (no edit to the existing file beyond the new method's signature being callable from the plugin).

Wait — the plugin rule (§11.1) says "New feature = new plugin under `plugin/`; Core modification is FORBIDDEN without explicit approval." So instead of extending the worker in `internal/handlers/`, the plugin **wraps** the worker:

- The plugin's `service.go` reads the instance list (one bulk SELECT), then for each instance computes the effective retention (D-4 / D-8), and invokes the worker via the existing `RunManualCleanup(ctx, orgID, now)` method **per instance** with the resolved retention passed through a new optional parameter on a new wrapper struct `RunOptions{RetentionDaysOverride *int, InstanceID *uuid.UUID}`. The existing `RunManualCleanup` method is updated to accept this struct (additive — old call sites continue to work because the default values mirror today's behavior).
- This requires **one** minimal, additive change to the existing `internal/handlers/uploads_cleanup_worker.go` (adding an optional parameter to one method). This change is flagged in the plan as a "core touchpoint" and is the only reason the implementation does not qualify as 100% plugin-local. The change is small enough to be reviewed and approved in the same PR as the plugin; it does not add an architectural layer (constitution §11.1 stays satisfied).

If the maintainer prefers zero core change, the alternative is to duplicate the worker loop in the plugin, which is rejected because it violates §3.7's "single source of truth for cleanup" and would create two near-identical filesystem walks. The plan documents the trade-off and the choice.

---

## Frontend integration

### New files

```text
frontend/src/components/settings/PerInstanceUploadsCleanup.vue   # the block
frontend/src/composables/usePerInstanceUploadsCleanup.ts          # useQuery + useMutation
```

### Modified files

| File | Change |
|---|---|
| `frontend/src/components/whatsmeow/InstanceCard.vue` | Add `<PerInstanceUploadsCleanup>` block after the existing `auto_campaign` block. |
| `frontend/src/views/settings/SettingsView.vue` | Extend the existing Uploads Cleanup section to call the new overview endpoint and render a per-instance list with a "Run cleanup now" link to each instance. |
| `frontend/src/services/api.ts` | Add 4 new methods: `getInstanceUploadsCleanup`, `updateInstanceUploadsCleanup`, `getInstanceUploadsCleanupHistory`, `runInstanceUploadsCleanup`, `getOrgUploadsCleanupOverview`. Extend `runUploadsCleanupNow` response typing to include the new `instances` field. |
| `frontend/src/i18n/locales/en.json` (and es, ar) | Add ~12 new keys under `settings.uploadsCleanup*` (per D-11). |

### i18n key list (to be added in all three locales)

```text
settings.uploadsCleanupInstanceRetentionLabel
settings.uploadsCleanupInstanceRetentionDesc
settings.uploadsCleanupInstanceInheritLabel
settings.uploadsCleanupInstanceInheritHelp
settings.uploadsCleanupInstanceRunNow
settings.uploadsCleanupInstanceHistoryTitle
settings.uploadsCleanupInstanceHistoryEmpty
settings.uploadsCleanupInstanceHistoryActor
settings.uploadsCleanupInstanceHistoryReason
settings.uploadsCleanupInstanceOverviewTitle
settings.uploadsCleanupInstanceOverviewEffectiveCustom
settings.uploadsCleanupInstanceOverviewEffectiveDefault
settings.uploadsCleanupInstanceOverviewEffectiveDisabled
settings.uploadsCleanupInstanceOverviewLastRun
settings.uploadsCleanupInstanceEffectivePreview
```

---

## Build & test

### Backend

```bash
# Activate the plugin in main.go (one blank import line)
# Then:
make build
./whatomate server -config config.toml -migrate   # applies plugin migrations

# Unit + integration tests
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"
go test -v -race -p 1 ./plugin/per-instance-uploads-cleanup/...
go test -v -race -p 1 ./internal/handlers/...    # worker extension
make lint
```

### Frontend

```bash
cd frontend
npm install
npm run typecheck
npm run lint
npm run test:unit            # Vitest, including new PerInstanceUploadsCleanup tests
npm run test:e2e -- --project=chromium  # Playwright happy path
```

### End-to-end happy path

1. `make build && ./whatomate server -config config.toml -migrate`
2. Open `/settings/instances`, pick an instance, expand the new "Uploads Cleanup" block.
3. Toggle "Inherit workspace default" **off**, enter `5`, save. Expect: UI shows `5` and disables the input on the next toggle. History shows one entry.
4. Open `/settings` → Uploads Cleanup section. The per-instance list shows this instance with the badge "Custom (5)".
5. Click "Run cleanup now" for this instance. Expect a 200 with `deleted_files: N` and a populated history entry.
6. Toggle "Inherit workspace default" **on**, save. History shows a second entry with `new_inherit: true`.
7. Re-run cleanup at the workspace level (`/settings` → "Run cleanup now"). Expect the response to include a per-instance breakdown (`instances: [...]`) per [contracts/org.uploads-cleanup.runs.yaml](../contracts/org.uploads-cleanup.runs.yaml).
8. Disconnect the instance from WhatsApp. Re-run cleanup. Expect: still works, deletes the same files (per EC-9, A-010).

---

## Verifying the constitutional gates

| Gate | How to verify |
|---|---|
| §3.1 fastglue/fasthttp | New handlers are methods on `*Plugin` returning `*fastglue.Request` and use `r.SendEnvelope` / `r.SendErrorEnvelope` |
| §3.2 envelope pattern | All responses follow the standard envelope (see [contracts/](../contracts/)) |
| §6.5 tenant scoping | Every DB call is `tenant.ScopedDB(p.db, orgID).Where(...).Find(...)`; tests assert cross-org access returns 404 |
| §6.6 indexes via getIndexes | The composite index is created with `CREATE INDEX IF NOT EXISTS` inside the plugin's `Migrate` (no core `getIndexes()` edit) |
| §7.2 test patterns | Tests use `fasthttp.RequestCtx` + `fastglue.Request`; mocks are hand-written; `testify/require` for assertions |
| §11.1 no new architecture | No new package boundaries; one new plugin under `plugin/`; one additive method parameter on the existing worker (the only "core touchpoint", flagged in the plan) |
| §11.4 dependency direction | Plugin imports from `internal/models`, `internal/middleware`, `internal/tenant`; never the reverse |
| §4.5 i18n all three locales | New keys added to `en.json`, `es.json`, `ar.json` in the same PR |

---

## Rollout

1. Merge behind the existing license feature flag (no new flag — reuses `settings.uploads_cleanup:write` for visibility).
2. Existing workspace with `uploads_cleanup_retention_days = N`: all existing instances are auto-backfilled with `inherit = true` (D-8 + the pre-migration fix in `Migrate`). No behavioral change for the first deploy.
3. Audit table starts empty; the first write happens on the first admin "edit retention" action.
4. Roll back by reverting the plugin; the new table and the JSONB sub-keys remain but are unused (data preserved per zero-downtime policy).

---

## Open items going into `/speckit.tasks`

- C-1: confirm audit retention policy (`forever` is the default).
- Q-OPT-1: confirm the history "Old / New" label format.
- Q-OPT-2: confirm the "Inherit" toggle should **keep** the old `retention_days` value or **clear** it.
- Worker extension approval: confirm that the additive parameter on `RunManualCleanup` is acceptable as the one core touchpoint, or if maintainers prefer a 100% plugin-local path (which duplicates the filesystem walk).
