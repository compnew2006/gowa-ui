# Audit Log System — Design Spec

- **Date:** 2026-06-23
- **Status:** Approved (design)
- **Owner:** whatomate-agent-orchestrator
- **Scope:** v1 (synchronous, scope-B coverage, admin-only read)

## 1. Goal

Provide a single canonical audit log that records security-relevant and
operational events across the Whatomate platform — authentication, chat
assignments, admin/management actions, and system lifecycle — with an
admin-only UI to browse and filter them.

This is a **cross-cutting core subsystem**, not a plugin. It extends the
proven audit pattern already present in
`plugin/module-management/audit.go` (the `ModuleEvent` model +
best-effort `recordEvent` helper) to cover all scope-B event categories.

## 2. Scope

### In scope (v1)

- **Coverage (scope-B):** auth (login/logout/refresh/password-reset),
  chat (claim/release/transfer/close/assign), admin (user/role/api-key
  CRUD), system (server/worker start/stop, license denial, config
  change). Per-message send/receive is **out of scope** for v1.
- **Write path:** synchronous, best-effort, via a central `*audit.Service`
  on `*handlers.App`.
- **Read path:** `GET /api/audit-events` (admin-only, tenant-scoped) +
  a Vue 3 admin view.

### Out of scope (deferred)

- CSV/JSON export.
- WebSocket live push of new events.
- Per-message send/receive events.
- Auto-retention / purge worker (schema ready; job deferred).
- Migrating `plugin/module-management` to also write into `audit_events`
  (untouched in v1 to keep blast radius small).

## 3. Architecture

A new cross-cutting package `internal/audit/`, parallel to
`internal/auth`, `internal/middleware`, `internal/tenant`,
`internal/observability`. Used by handlers, middleware, workers, and
plugins alike via the shared `*handlers.App` dependency container.

### 3.1 New/changed locations

| Path | Purpose |
|---|---|
| `internal/audit/service.go` (new) | Central `Service.Record(ctx, evt)` recorder |
| `internal/audit/events.go` (new) | Typed `Source*`, `Category*`, `Action*` constants |
| `internal/audit/builder.go` (new) | Fluent `EventBuilder` for ergonomic one-liner recording |
| `internal/models/audit_event.go` (new) | `AuditEvent` GORM model + `audit_events` table |
| `internal/database/postgres.go` (edit) | Register `&models.AuditEvent{}` in core `AutoMigrate` |
| `internal/handlers/app.go` (edit) | Add `Audit *audit.Service` field + wire it at construction |
| `internal/handlers/audit_handlers.go` (new) | `GET /api/audit-events` admin handler |
| `internal/handlers/routes_*.go` (edit) | Register the route with `RequirePermission(audit, read)` |
| `internal/models/roles.go` (edit) | Grant `audit:read` to admin system role |
| Permission seed (edit) | New row `audit`/`read` via existing seeding migration |
| `frontend/src/views/settings/AuditLogView.vue` (new) | Admin-only audit log browser |
| `frontend/src/stores/audit.ts` (new) | Pinia store |
| `frontend/src/services/audit.ts` (new) | Service wrapping `api.ts` |
| `frontend/src/router/index.ts` (edit) | Route `/settings/audit-log` + admin guard |
| Sidebar nav + i18n locales (edit) | Admin-only nav entry + labels |

### 3.2 Recording strategy

- Each action call site constructs an `AuditEvent` and calls
  `app.Audit.Record(ctx, evt)`.
- `Record` resolves the tenant-scoped DB when an `orgID` is present
  (`tenant.GetScopedDB`); otherwise writes to the global DB (system/global
  events).
- **Best-effort:** a write failure is logged at `slog.Error` and does not
  fail the user's action. This mirrors the contract of the existing
  `module-management.recordEvent` helper exactly.
- **Recording happens *after* the primary action succeeds**, outside any
  open DB transaction, so an audit row is never rolled back when the
  action succeeded, and a failed action never produces a `success=true`
  row. For events worth auditing on failure (login_failed,
  license_denied), the call site explicitly sets `Success(false)`.

### 3.3 Relationship to `plugin/module-management`

Unchanged. `module_events` stays as the module-feature's own table for
backwards compatibility. The new `audit_events` is the canonical
cross-cutting log. Migrating the plugin to additionally emit into
`audit_events` is a follow-up, **not v1**.

## 4. Data Model

`internal/models/audit_event.go`:

```go
// AuditEvent is the canonical cross-cutting audit record.
type AuditEvent struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`

    // Tenancy: nullable for global/system events (e.g. server restart).
    OrganizationID *uuid.UUID `gorm:"type:uuid;index" json:"organization_id,omitempty"`

    // What happened — typed enum-backed strings for indexability + flexibility.
    Category string `gorm:"size:32;not null;index" json:"category"`
    Action   string `gorm:"size:64;not null;index" json:"action"`

    // Who/what initiated it. Source disambiguates system vs user attribution.
    Source      string     `gorm:"size:16;not null;index" json:"source"`
    ActorUserID *uuid.UUID `gorm:"type:uuid" json:"actor_user_id,omitempty"`
    ActorEmail  string     `gorm:"size:255" json:"actor_email,omitempty"`
    ActorRole   string     `gorm:"size:32" json:"actor_role,omitempty"`

    // Target of the action (nullable for events without a single target).
    TargetType string  `gorm:"size:32" json:"target_type,omitempty"`
    TargetID   *string `gorm:"size:64;index" json:"target_id,omitempty"`

    // Outcome of the action.
    Success bool   `gorm:"not null;default:true" json:"success"`
    Reason  string `gorm:"size:255" json:"reason,omitempty"`

    // Free-form structured detail (IP, user-agent, before/after, params).
    Details models.JSONB `gorm:"type:jsonb;default:'{}'" json:"details"`

    // Optional request origin for security forensics.
    IPAddress string `gorm:"size:45" json:"ip_address,omitempty"`
    UserAgent string `gorm:"size:255" json:"user_agent,omitempty"`
}

func (AuditEvent) TableName() string { return "audit_events" }
```

### 4.1 Indexes (composite for the read-side filter combinations)

- `(organization_id, created_at DESC)` — primary list query per org
- `(category, action)` — filter by event type
- `(actor_user_id)` — "what did user X do"
- `(source)` — "show me all system events"
- `(target_id)` — "everything that happened to chat X"
- `(created_at DESC)` — global/system events without an org

### 4.2 Design notes

- **`TargetID` as `*string`:** deliberately stringly-typed to handle
  UUID targets (users), numeric IDs, and JIDs (group chats like
  `120363…@g.us`) uniformly. The companion `TargetType` disambiguates.
- **Typed actor pattern** reuses the established `ModuleEvent` shape
  (`ActorUserID`, `ActorEmail`, nullable `OrganizationID`).
- **`Source` field** cleanly distinguishes system-originated events
  (`system`, `worker`, `scheduled`) from human ones (`user`), so
  "server restarted" has `Source="system"`, `ActorUserID=nil`,
  `OrganizationID=nil`.

### 4.3 Constants (`internal/audit/events.go`)

```go
// Source
const (
    SourceUser      = "user"
    SourceSystem    = "system"
    SourceWorker    = "worker"
    SourceScheduled = "scheduled"
)

// Category
const (
    CategoryAuth     = "auth"
    CategoryChat     = "chat"
    CategoryAdmin    = "admin"
    CategorySystem   = "system"
    CategoryCampaign = "campaign"
    CategoryTemplate = "template"
)

// Action — namespaced by category for readability
const (
    // auth
    ActionLoginSuccess   = "login_success"
    ActionLoginFailed    = "login_failed"
    ActionLogout         = "logout"
    ActionTokenRefreshed = "token_refreshed"
    ActionPasswordReset  = "password_reset"

    // chat (scope-B: claim/release/transfer/close/assign — NOT per-message)
    ActionChatClaimed     = "chat_claimed"
    ActionChatReleased    = "chat_released"
    ActionChatTransferred = "chat_transferred"
    ActionChatClosed      = "chat_closed"
    ActionChatAssigned    = "chat_assigned"

    // admin
    ActionUserCreated   = "user_created"
    ActionUserUpdated   = "user_updated"
    ActionUserDeleted   = "user_deleted"
    ActionUserActivated = "user_activated"
    ActionUserSuspended = "user_suspended"
    ActionRoleCreated   = "role_created"
    ActionRoleUpdated   = "role_updated"
    ActionRoleDeleted   = "role_deleted"
    ActionAPIKeyCreated = "api_key_created"
    ActionAPIKeyRevoked = "api_key_revoked"

    // system
    ActionServerStarted  = "server_started"
    ActionServerStopped  = "server_stopped"
    ActionWorkerStarted  = "worker_started"
    ActionWorkerStopped  = "worker_stopped"
    ActionConfigChanged  = "config_changed"
    ActionLicenseDenied  = "license_denied"
    ActionModuleEnabled  = "module_enabled"
    ActionModuleDisabled = "module_disabled"
)
```

### 4.4 Retention

v1 keeps all rows (no auto-purge). The index on `created_at` makes a
future `internal/worker/audit_retention.go` (delete rows older than N
days per `internal/config`) cheap to add later.

## 5. Recording API

### 5.1 `*audit.Service`

```go
// internal/audit/service.go

// Service is the central audit recorder. Safe for concurrent use.
// One instance lives on *handlers.App.
type Service struct {
    db  *gorm.DB     // global (unscoped) DB — used for system/global events
    log *slog.Logger
}

func New(db *gorm.DB, log *slog.Logger) *Service {
    return &Service{db: db, log: log}
}

// Record persists one event. Best-effort: logs on failure, never panics.
// If evt.OrganizationID is non-nil, the write goes to the tenant-scoped DB;
// otherwise to the global DB.
func (s *Service) Record(ctx context.Context, evt AuditEvent) {
    if s == nil || s.db == nil {
        return
    }
    evt.ID = uuid.New()
    if evt.CreatedAt.IsZero() {
        evt.CreatedAt = time.Now().UTC()
    }

    db := s.db
    if evt.OrganizationID != nil {
        if scoped, err := tenant.GetScopedDB(*evt.OrganizationID); err == nil && scoped != nil {
            db = scoped
        }
    }
    if err := db.WithContext(ctx).Create(&evt).Error; err != nil {
        s.log.Error("audit write failed",
            "error", err, "category", evt.Category,
            "action", evt.Action, "source", evt.Source)
    }
}
```

### 5.2 `EventBuilder` (ergonomics)

```go
// internal/audit/builder.go

// NewEvent starts a builder for the given action. Category is inferred
// from the Action* constant namespace via a lookup table, so call sites
// specify only the action.
func NewEvent(action string) *EventBuilder

func (b *EventBuilder) Category(c string) *EventBuilder
func (b *EventBuilder) Org(id *uuid.UUID) *EventBuilder
func (b *EventBuilder) OrgValue(id uuid.UUID) *EventBuilder

// ActorFromRequest pulls actor from a fastglue request context
// (userID, email, role, ip, ua).
func (b *EventBuilder) ActorFromRequest(r *fastglue.Request) *EventBuilder

// ActorSystem marks a system-originated event (no human actor).
func (b *EventBuilder) ActorSystem(component string) *EventBuilder

func (b *EventBuilder) Target(typ string, id any) *EventBuilder // stringifies id
func (b *EventBuilder) Success(v bool) *EventBuilder
func (b *EventBuilder) Reason(s string) *EventBuilder
func (b *EventBuilder) Detail(k string, v any) *EventBuilder // merges into JSONB Details

// Build returns the immutable event for callers that hold their own *Service.
func (b *EventBuilder) Build() AuditEvent

// Record builds and records in one call. svc may be nil (no-op).
func (b *EventBuilder) Record(ctx context.Context, svc *Service)
```

### 5.3 Integration points (v1 call sites)

Each is a single, well-localized insertion; no broad refactors.

| Event source | File | Insertion point |
|---|---|---|
| Auth — login success/fail | `internal/handlers/auth_handlers.go` | After credential check in `Login` |
| Auth — logout | `internal/handlers/auth_handlers.go` | After cookie clear in `Logout` |
| Auth — token refresh | `internal/handlers/auth_handlers.go` | In refresh path |
| Auth — password reset | `internal/handlers/auth_handlers.go` | On successful reset |
| Chat — claim/release/transfer/close/assign | `internal/handlers/chat_handlers.go` (or agent-assignment file) | At each action's success path |
| Admin — user CRUD | `internal/handlers/users.go` | After successful create/update/delete |
| Admin — role CRUD | `internal/handlers/roles.go` | After successful create/update/delete |
| Admin — API keys | `internal/handlers/api_keys.go` | After create/revoke |
| System — server start/stop | `cmd/whatomate/main.go` | At graceful start and shutdown |
| System — worker start/stop | Worker entry | At graceful start and shutdown |
| System — license denied | License check rejection point | At denial |
| System — config changed | Admin settings save handler | After successful save |

### 5.4 Wiring

- `internal/handlers/app.go`: add `Audit *audit.Service` field.
- App construction (server + worker bootstrap):
  `app.Audit = audit.New(app.DB, app.Log)` immediately after `app.DB` is
  set, before routes register.
- Core `AutoMigrate` list in `internal/database/postgres.go`: add
  `&models.AuditEvent{}`.
- `internal/models/roles.go` `SystemRolePermissions()`: admin gets
  `"audit:read"`; manager/agent do **not**.
- New permission seed row: `Resource="audit"`, `Action="read"` via the
  existing permission-seeding migration path.

## 6. Read Side

### 6.1 Backend handler

```go
// internal/handlers/audit_handlers.go

// ListAuditEvents GET /api/audit-events
// Admin-only (RequirePermission(audit, read)).
// Filters: category, action, source, actor_user_id, target_id,
//          target_type, success, q (text), date_from, date_to
// Pagination: page, per_page (default 50, max 200)
// Sort: created_at DESC (newest first)
func (a *App) ListAuditEvents(r *fastglue.Request) error
```

**Tenancy rules** (no new boundary logic; reuses existing primitives):
- **Super admin** (`middleware.IsSuperAdmin(r)`): sees events across all
  orgs (optional `organization_id` filter). Required for the
  "system restarted" / global oversight use case.
- **Org admin** (has `audit:read`): sees only events where
  `organization_id == their org`. System/global events
  (`organization_id IS NULL`) are also visible (read-only, no actor
  actions to leak).
- **No `audit:read`:** blocked at the route middleware
  (`RequirePermission`), never reaches the handler.

**Response envelope** (`r.SendEnvelope`):

```json
{
  "status": "success",
  "data": {
    "events": [ /* AuditEvent rows */ ],
    "total": 1423,
    "page": 1,
    "per_page": 50
  }
}
```

**Query building:** standard GORM scoped chain — start from
`app.requestDB(r)` (tenant-scoped), apply `.Where(...)` per non-empty
filter, `.Count()` then `.Order("created_at DESC").Offset(...).Limit(...).Find(...)`.
Super-admin branch swaps in `app.DB` (global) and adds the optional org
filter.

**Route registration** (mirrors existing `users`/`roles` route style):

```go
g.GET("/api/audit-events",
    middleware.RequirePermission(a.HasPermission, "audit", "read"),
    a.ListAuditEvents)
```

### 6.2 Frontend (Vue 3)

All new files; no existing frontend code modified.

**Service** (`frontend/src/services/audit.ts`) — wraps the shared
`api.ts` client:

```ts
export interface AuditEventFilters {
  category?; action?; source?; actor_user_id?;
  target_id?; success?; q?; date_from?; date_to?;
  page?; per_page?;
}

export const auditService = {
  list(filters: AuditEventFilters): Promise<{ events, total, page, per_page }>
}
```

**Pinia store** (`frontend/src/stores/audit.ts`) — matches existing
store pattern:
- state: `events`, `total`, `loading`, `filters`, `pagination`
- actions: `fetch()`, `setFilter(k, v)`, `resetFilters()`, `goToPage(n)`

**View** (`frontend/src/views/settings/AuditLogView.vue`,
`<script setup>` + TS):
- Filter bar: category, action, source, actor search, free-text `q`,
  date range, success/fail toggle, "Reset" button.
- Data table: timestamp, category badge (color-coded), action, actor
  (email + role), target, success/fail icon, expandable details row
  (renders JSONB).
- Pagination controls (reuse the repo's existing pagination component
  if any; otherwise a minimal one).
- i18n keys added under `frontend/src/i18n/locales/*.json`
  (`settings.auditLog.*`).
- Admin-only — guarded by the route + the existing role-aware sidebar.

**Router** (`frontend/src/router/index.ts`):

```ts
{
  path: '/settings/audit-log',
  name: 'audit-log',
  component: () => import('@/views/settings/AuditLogView.vue'),
  meta: {
    requiresAuth: true,
    requiredPermission: 'audit:read',
    title: 'Audit Log',
  },
}
```

Sidebar nav entry added under the Settings section (admin-only
visibility).

**No direct `fetch`/`axios`** — everything through `services/api.ts`
so auth refresh/interceptors stay centralized, per AGENTS.md frontend
conventions.

## 7. Testing Strategy

Follows repo conventions: table-driven tests, `newTestApp(t)` +
functional options, `testutil.SetupTestDB(t)`, `testify` assertions.

### 7.1 Model & migration (unit)

- `TableName()` returns `"audit_events"`; JSON serialization round-trips
  nullable fields.
- Migration: after `AutoMigrate`, `db.Migrator().HasTable(&AuditEvent{})`
  is true; idempotent on re-run. Mirrors
  `TestMigrateModuleEventsIsIdempotent`.

### 7.2 Service (unit) — `internal/audit/service_test.go`

- `Record` persists a row; fields preserved on fetch.
- `Record` with non-nil `OrganizationID` routes to scoped DB
  (stub `tenant.GetScopedDB`).
- `Record` with nil `OrganizationID` writes to global DB.
- `Record` when `s.db == nil` is a safe no-op.
- `Record` swallows a write error (logs, never panics) via a forced
  failure.
- Best-effort: a failed audit write does not surface an error to the
  caller.

### 7.3 Builder (unit) — `internal/audit/builder_test.go`

- `NewEvent(ActionUserCreated)` infers `CategoryAdmin`.
- `ActorFromRequest` extracts userID/email/role/ip/ua from a request
  built via `testutil.NewGETRequest` + `testutil.SetAuthContext`.
- `ActorSystem("worker")` sets `Source=system`, nil actor.
- `Target("chat", id)` stringifies uuid and JID correctly.
- `Detail(k,v)` merges into `Details` JSONB without clobbering prior
  keys.
- `Record(ctx, svc)` builds + persists in one call; nil `svc` no-op.

### 7.4 Handler (integration — `TEST_DATABASE_URL`)

`internal/handlers/audit_handlers_test.go`:
- Org admin with `audit:read` sees only their org's events.
- Super admin sees cross-org + global events; can filter by
  `organization_id`.
- Non-admin (no `audit:read`) → 403 at middleware.
- Filters: `category`, `action`, `source`, `actor_user_id`,
  `target_id`, `success`, `q`, date range each narrow correctly.
- Pagination: `page`/`per_page` + `total` correctness; `per_page` capped
  at 200.
- Ordering: newest first.
- Standard envelope shape.

### 7.5 Recording integration (the core guarantee)

A focused test per category proving recording fires at the call site
after success:
- `TestLogin_RecordsAuditEvent_Success` — login, assert one
  `auth/login_success` row with the right actor.
- `TestLogin_RecordsAuditEvent_Failure` — wrong password, assert one
  `auth/login_failed` row with `Success=false`.
- `TestLogout_RecordsAuditEvent`.
- `TestCreateUser_RecordsAuditEvent` — admin creates user, assert
  `admin/user_created` with correct `TargetID`.
- `TestDeleteUser_RecordsAuditEvent`.
- `TestChatClaim_RecordsAuditEvent`.
- `TestServerStart_RecordsAuditEvent` — unit test of the startup
  recorder call.

### 7.6 Tenant boundary (critical invariant)

Per AGENTS.md: "any change that could leak data across organizations
requires targeted tests":
- Org A admin queries audit events; Org B events never appear.
- Global event (`organization_id IS NULL`) is visible to both org
  admins but **not** to a non-admin.
- Super admin sees all three buckets (Org A, Org B, global).

### 7.7 Frontend (deferred to plan)

Minimum: a `vitest` unit test for the Pinia store (`fetch` populates
state, filters drive query params) and one Playwright e2e (admin can
open audit log page; non-admin is redirected away).

### 7.8 Verification gates (from AGENTS.md §5)

After implementation: `make test` (full suite) + targeted
`go test ./internal/audit/... ./internal/handlers/... ./internal/models/...`
+ `cd frontend && npm run typecheck && npm run lint && npm run test:unit`
+ `make build`. MCP re-verification: Socraticode `codebase_update` +
`codebase_graph_circular`, Serena `get_diagnostics_for_file` on every
edited file, `codebase-memory-mcp detect_changes`.

## 8. Scope Summary

- **New core Go files:** `internal/audit/{service,builder,events}.go`,
  `internal/models/audit_event.go`,
  `internal/handlers/audit_handlers.go`, + their `_test.go` files.
- **Small edits to existing core files:** `app.go` (add `Audit` field +
  wiring), `database/postgres.go` (register migration),
  `models/roles.go` + permission seed (admin gets `audit:read`).
- **Targeted insertions (~10 one-liners)** at scope-B call sites: auth,
  chat claim/release/transfer/close/assign, user/role/api-key CRUD,
  server/worker lifecycle, license denial, config save.
- **New frontend files:** service, store, view + route + nav entry +
  i18n keys.
- **No plugin changes.** No core handler refactors beyond the localized
  audit insertions. `module_events` table left untouched.

## 9. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Audit write blocks a user action on slow DB | Best-effort `Record`; logged not propagated. Same contract as existing `recordEvent`. |
| Cross-org data leak in the read API | Tenant-scoped query chain; targeted tenant-boundary tests (§7.6); route gated by `RequirePermission`. |
| `audit_events` grows unbounded | Index on `created_at` ready for future retention worker; v1 ships without purge, documented as deferred. |
| Permission drift (admin loses `audit:read`) | Seeded in `SystemRolePermissions()` (admin only) + permission seed row; covered by existing roles tests. |
| Recording silently no-op at a call site | Per-call-site integration tests (§7.5) lock in emission. |
| Schema churn touching core model location | `AuditEvent` lives in `internal/models/` per AGENTS.md core-model convention; migration registered in `internal/database/postgres.go`. |
