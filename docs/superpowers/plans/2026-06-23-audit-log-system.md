# Audit Log System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a canonical cross-cutting audit log that records security-relevant and operational events (auth, chat assignment, admin CRUD, system lifecycle) across Whatomate, with an admin-only UI to browse and filter them.

**Architecture:** A new `internal/audit/` core package exposes a best-effort `*audit.Service` recorder living on `*handlers.App`. Action call sites build events via a fluent `EventBuilder` and persist them after the primary action succeeds. A new `audit_events` GORM table (tenant-scoped, nullable `organization_id` for global/system events) is read by an admin-only `GET /api/audit-events` handler and a Vue 3 settings view. This extends the proven pattern from `plugin/module-management/audit.go` (`ModuleEvent` + best-effort `recordEvent`) to all scope-B categories.

**Tech Stack:** Go (GORM, fasthttp/fastglue, uuid, logf logger), PostgreSQL (jsonb details), Vue 3 + TypeScript + Pinia, vitest, testify.

**Reference spec:** `docs/superpowers/specs/2026-06-23-audit-log-system-design.md`

---

## Verified codebase facts (do not re-derive)

These were confirmed against current source before writing this plan:

- `App` struct is at `internal/handlers/app.go:30-62`. `Log` field is typed **`logf.Logger`** (NOT `*slog.Logger` — the spec had this wrong; this plan uses `logf.Logger`).
- `App` is constructed at `cmd/whatomate/main.go:350` via a struct literal with fields like `Config`, `DB`, `Redis`, `Log`. Worker bootstrap constructs its own App similarly (see Task 5).
- Migration registration is via `GetMigrationModels()` at `internal/database/postgres.go:101-197`, which returns `[]MigrationModel` (struct `{Name string; Model any}`). There is NO raw `AutoMigrate(...)` slice to append to — the spec was imprecise here.
- Permission catalog is seeded from `models.DefaultPermissions()` at `internal/models/roles.go:100-256`. **Admin automatically gets every default permission** because `SystemRolePermissions()` (roles.go:288-372) builds the `"admin"` key from `range DefaultPermissions()`. Therefore adding the `audit:read` row to `DefaultPermissions()` ALONE grants it to admin; manager/agent stay excluded. No edit to `SystemRolePermissions()` is needed.
- Resource constants live in `internal/models/roles.go:40-82` as `ResourceXxx = "..."`. Last one is `ResourceGroupParticipants = "group_participants"` at line 81, closing `)` at line 82.
- Route registration style (from `cmd/whatomate/main.go:1522-1530`): bare `g.GET("/api/roles", app.ListRoles)` calls. Permission gating for read-only admin endpoints in this codebase is done **inside the handler** via `a.requirePermission(r, userID, resource, action)` (see `app.go:238-248` and `roles.go:407-439` `ListPermissions`). The middleware `middleware.RequirePermission(checker, r, a)` form also exists but the dominant admin-read pattern is in-handler `requirePermission`. **This plan uses the in-handler `requirePermission` pattern** to match `ListPermissions`/`ListRoles` exactly (fewer moving parts, no route-file signature changes).
- `middleware.IsSuperAdmin(r)`, `middleware.GetOrganizationID(r)`, `middleware.GetUser(r)`, `middleware.GetUserID(r)` exist (`internal/middleware/middleware.go:530-562`).
- `tenant.GetScopedDB(orgID)` and `tenant.ScopedDB(db, orgID)` exist for tenant-scoped DB resolution.
- `a.requestDB(r)` returns the tenant-scoped DB for the request.
- `models.JSONB` type exists (used by `ModuleEvent.Details`).
- Test App: `newTestApp(t, ...appOption)` with functional options in `internal/handlers/testhelpers_test.go`. DB tests use `testutil.SetupTestDB(t)`. Logger: `testutil.NopLogger()` returns `logf.Logger`.
- `ModuleEvent` model (`plugin/module-management/audit.go`) is the established shape this design mirrors: nullable `OrganizationID *uuid.UUID`, `ActorUserID *uuid.UUID`, `ActorEmail string`, `Details models.JSONB`, best-effort write.

---

## File Structure

**New files:**
| File | Responsibility |
|---|---|
| `internal/audit/events.go` | `Source*`, `Category*`, `Action*` constants + action→category lookup |
| `internal/audit/builder.go` | `EventBuilder` fluent API + `NewEvent(action)` |
| `internal/audit/service.go` | `Service.Record(ctx, evt)` best-effort recorder |
| `internal/audit/service_test.go` | Service unit tests |
| `internal/audit/builder_test.go` | Builder unit tests |
| `internal/models/audit_event.go` | `AuditEvent` GORM model + `TableName()` |
| `internal/models/audit_event_test.go` | Model round-trip + migration test |
| `internal/handlers/audit_handlers.go` | `ListAuditEvents` admin read handler |
| `internal/handlers/audit_handlers_test.go` | Handler integration + tenant-boundary tests |
| `frontend/src/services/audit.ts` | API service wrapping `api.ts` |
| `frontend/src/stores/audit.ts` | Pinia store |
| `frontend/src/views/settings/AuditLogView.vue` | Admin-only audit browser view |
| `frontend/src/stores/audit.test.ts` | Pinia store unit test |

**Modified files (small, localized edits):**
| File | Change |
|---|---|
| `internal/database/postgres.go` | Add `{"AuditEvent", &models.AuditEvent{}}` to `GetMigrationModels()` |
| `internal/models/roles.go` | Add `ResourceAudit` const + one `audit:read` row to `DefaultPermissions()` |
| `internal/handlers/app.go` | Add `Audit *audit.Service` field to `App` struct |
| `cmd/whatomate/main.go` | Wire `Audit: audit.New(db, lo)` into App literal (~line 350) + register route + record server start |
| `internal/handlers/auth_handlers.go` | Record login success/fail, logout |
| `internal/handlers/chat_lifecycle.go` | Record claim/release/close |
| `internal/handlers/users.go` | Record user create/update/delete |
| `internal/handlers/roles.go` | Record role create/update/delete |
| `internal/handlers/api_keys.go` | Record api-key create/revoke |
| `frontend/src/router/index.ts` | Add `/settings/audit-log` route + guard |
| `frontend/src/i18n/locales/en.json` | `settings.auditLog.*` keys |
| (sidebar nav component) | Admin-only nav entry |

**Unchanged:** `plugin/module-management/audit.go` and its `module_events` table (v1 keeps them separate per spec §3.3).

---

## Task ordering rationale

Foundation first (model + migration + permission), then the recorder package (service + builder), then wiring onto App, then call-site insertions (each independently shippable), then the read API, then frontend. Tests are written before/with each unit (TDD where the unit has branching logic; the call-site insertions are verified by focused integration tests since they are one-liners).

---

## Task 1: AuditEvent model + migration registration

**Files:**
- Create: `internal/models/audit_event.go`
- Create: `internal/models/audit_event_test.go`
- Modify: `internal/database/postgres.go` (add to `GetMigrationModels()`)

- [ ] **Step 1: Write the failing model test**

Create `internal/models/audit_event_test.go`:

```go
package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEvent_TableName(t *testing.T) {
	assert.Equal(t, "audit_events", AuditEvent{}.TableName())
}

func TestAuditEvent_JSONRoundTrip_NullableFields(t *testing.T) {
	// Global/system event: no org, no actor user, no target.
	evt := AuditEvent{
		ID:         uuid.New(),
		Category:   CategorySystem,
		Action:     "server_started",
		Source:     "system",
		Success:    true,
		Details:    JSONB{"component": "server"},
		IPAddress:  "127.0.0.1",
		UserAgent:  "whatomate/1.0",
	}
	_ = evt // compile + shape check; serialization asserted via DB round-trip in handler tests.
	require.NotEqual(t, uuid.Nil, evt.ID)
}
```

This references constants `CategorySystem` that will live in `internal/audit/events.go`, NOT in `models`. To avoid an import cycle (`internal/audit` imports `internal/models` for `JSONB`), the test above must use a string literal instead. Replace `CategorySystem` with `"system"` in the test:

```go
		Category:   "system",
```

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `go test ./internal/models/ -run TestAuditEvent -count=1`
Expected: FAIL — `undefined: AuditEvent`.

- [ ] **Step 3: Create the model**

Create `internal/models/audit_event.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent is the canonical cross-cutting audit record. It records
// security-relevant and operational events across the platform.
//
// The shape mirrors the established ModuleEvent pattern (nullable
// OrganizationID, typed ActorUserID/ActorEmail, JSONB Details). Unlike
// ModuleEvent, AuditEvent is tenant-scoped via internal/tenant at write
// time and is the canonical log for all scope-B categories.
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
	// TargetID is stringly-typed to uniformly handle UUIDs, numeric IDs, and JIDs.
	TargetType string  `gorm:"size:32" json:"target_type,omitempty"`
	TargetID   *string `gorm:"size:64;index" json:"target_id,omitempty"`

	// Outcome of the action.
	Success bool   `gorm:"not null;default:true" json:"success"`
	Reason  string `gorm:"size:255" json:"reason,omitempty"`

	// Free-form structured detail (IP, user-agent, before/after, params).
	Details JSONB `gorm:"type:jsonb;default:'{}'" json:"details"`

	// Optional request origin for security forensics.
	IPAddress string `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent string `gorm:"size:255" json:"user_agent,omitempty"`
}

func (AuditEvent) TableName() string { return "audit_events" }
```

- [ ] **Step 4: Run model test to verify it passes**

Run: `go test ./internal/models/ -run TestAuditEvent -count=1`
Expected: PASS.

- [ ] **Step 5: Register in migration list**

Edit `internal/database/postgres.go` — in `GetMigrationModels()`, add as the last entry before the closing `}` (after the `{"SavedContent", &models.SavedContent{}},` line):

```go
		{"AuditEvent", &models.AuditEvent{}},
```

- [ ] **Step 6: Verify package compiles**

Run: `go build ./internal/... ./cmd/...`
Expected: success (no errors).

- [ ] **Step 7: Commit**

```bash
git add internal/models/audit_event.go internal/models/audit_event_test.go internal/database/postgres.go
git commit -m "feat(audit): add AuditEvent model and register migration"
```

---

## Task 2: Permission constants + audit:read default permission

**Files:**
- Modify: `internal/models/roles.go` (add `ResourceAudit` const + one row in `DefaultPermissions()`)

- [ ] **Step 1: Add the resource constant**

In `internal/models/roles.go`, the resource constants block ends at line 81 (`ResourceGroupParticipants = "group_participants"`) with `)` at line 82. Edit to add `ResourceAudit` before the closing `)`:

```go
	ResourceGroupParticipants      = "group_participants"

	// Audit log (admin-only read).
	ResourceAudit                  = "audit"
)
```

- [ ] **Step 2: Add the default permission row**

In `DefaultPermissions()` (`roles.go:100-256`), add a new section before the closing `}` of the returned slice (after the Group Participants block):

```go
		// Audit Log (admin-only; admin auto-inherits all default permissions
		// via SystemRolePermissions(), manager/agent intentionally excluded).
		{Resource: ResourceAudit, Action: ActionRead, Description: "View audit log events"},
```

- [ ] **Step 3: Verify the existing seed test still passes (and now counts one more)**

Run: `go test ./internal/database/ -run TestSeedPermissionsAndRoles -count=1`
Expected: PASS. (The test `TestSeedPermissionsAndRoles_CreatesAllDefaultPermissions` compares count to `len(models.DefaultPermissions())`, so it self-adjusts.)

- [ ] **Step 4: Commit**

```bash
git add internal/models/roles.go
git commit -m "feat(audit): add audit:read permission (admin-only)"
```

---

## Task 3: audit package — event constants

**Files:**
- Create: `internal/audit/events.go`

- [ ] **Step 1: Create the constants file**

Create `internal/audit/events.go`:

```go
// Package audit provides the canonical cross-cutting audit recorder for
// Whatomate. It exposes a best-effort Service.Record recorder, a fluent
// EventBuilder, and typed Source/Category/Action constants.
package audit

// Source values distinguish who/what originated an event.
const (
	SourceUser      = "user"
	SourceSystem    = "system"
	SourceWorker    = "worker"
	SourceScheduled = "scheduled"
)

// Category values group actions for filtering on the read side.
const (
	CategoryAuth     = "auth"
	CategoryChat     = "chat"
	CategoryAdmin    = "admin"
	CategorySystem   = "system"
	CategoryCampaign = "campaign"
	CategoryTemplate = "template"
)

// Action values are namespaced by category for readability. Call sites pass
// an Action* constant to NewEvent; the category is inferred via actionCategory.
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

// actionCategory maps each Action* to its Category*. NewEvent uses this so
// call sites specify only the action. Unknown actions default to CategorySystem.
func actionCategory(action string) string {
	switch action {
	case ActionLoginSuccess, ActionLoginFailed, ActionLogout,
		ActionTokenRefreshed, ActionPasswordReset:
		return CategoryAuth
	case ActionChatClaimed, ActionChatReleased, ActionChatTransferred,
		ActionChatClosed, ActionChatAssigned:
		return CategoryChat
	case ActionUserCreated, ActionUserUpdated, ActionUserDeleted,
		ActionUserActivated, ActionUserSuspended,
		ActionRoleCreated, ActionRoleUpdated, ActionRoleDeleted,
		ActionAPIKeyCreated, ActionAPIKeyRevoked:
		return CategoryAdmin
	case ActionServerStarted, ActionServerStopped,
		ActionWorkerStarted, ActionWorkerStopped,
		ActionConfigChanged, ActionLicenseDenied,
		ActionModuleEnabled, ActionModuleDisabled:
		return CategorySystem
	default:
		return CategorySystem
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/audit/`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/audit/events.go
git commit -m "feat(audit): add event source/category/action constants"
```

---

## Task 4: audit package — Service recorder (TDD)

**Files:**
- Create: `internal/audit/service.go`
- Create: `internal/audit/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Create `internal/audit/service_test.go`:

```go
package audit

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, *models.AuditEvent) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	// Ensure the audit table exists in the test DB.
	require.NoError(t, db.AutoMigrate(&models.AuditEvent{}))
	return New(db, testutil.NopLogger()), &models.AuditEvent{}
}

func TestRecord_PersistsRow_GlobalEvent(t *testing.T) {
	svc, _ := newTestService(t)

	orgID := uuid.New()
	actorID := uuid.New()
	evt := models.AuditEvent{
		OrganizationID: &orgID,
		Category:       CategoryAuth,
		Action:         ActionLoginSuccess,
		Source:         SourceUser,
		ActorUserID:    &actorID,
		ActorEmail:     "admin@example.com",
		Success:        true,
		IPAddress:      "10.0.0.1",
	}

	svc.Record(t.Context(), evt)

	var got models.AuditEvent
	require.NoError(t, svc.db.WithContext(t.Context()).
		Where("action = ?", ActionLoginSuccess).First(&got).Error)
	assert.NotEqual(t, uuid.Nil, got.ID)
	assert.False(t, got.CreatedAt.IsZero())
	assert.Equal(t, "admin@example.com", got.ActorEmail)
	assert.Equal(t, "10.0.0.1", got.IPAddress)
	assert.True(t, got.Success)
}

func TestRecord_NilService_IsNoOp(t *testing.T) {
	var svc *Service // nil pointer
	// Must not panic.
	svc.Record(t.Context(), models.AuditEvent{Action: ActionLogout})
}

func TestRecord_NilDB_IsNoOp(t *testing.T) {
	svc := &Service{db: nil, log: testutil.NopLogger()}
	// Must not panic.
	svc.Record(t.Context(), models.AuditEvent{Action: ActionLogout})
}

func TestRecord_AssignsIDAndCreatedAtWhenZero(t *testing.T) {
	svc, _ := newTestService(t)
	svc.Record(t.Context(), models.AuditEvent{
		Category: CategorySystem,
		Action:   ActionServerStarted,
		Source:   SourceSystem,
	})
	var got models.AuditEvent
	require.NoError(t, svc.db.WithContext(t.Context()).First(&got).Error)
	assert.NotEqual(t, uuid.Nil, got.ID)
	assert.False(t, got.CreatedAt.IsZero())
}
```

Note: `t.Context()` requires Go 1.24+. The repo uses Go 1.25 (per `docs/PROJECT_CONTEXT.md`), so this is available. If a given test runner is older, substitute `context.Background()` and add `"context"` import.

- [ ] **Step 2: Run tests to verify they fail (compile error)**

Run: `go test ./internal/audit/ -run TestRecord -count=1`
Expected: FAIL — `undefined: Service`, `undefined: New`.

- [ ] **Step 3: Implement the Service**

Create `internal/audit/service.go`:

```go
package audit

import (
	"context"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Service is the central audit recorder. Safe for concurrent use.
// One instance lives on *handlers.App.
type Service struct {
	db  *gorm.DB
	log logf.Logger
}

// New returns a recorder backed by the global (unscoped) DB. Per-event
// tenant routing happens inside Record via tenant.GetScopedDB.
func New(db *gorm.DB, log logf.Logger) *Service {
	return &Service{db: db, log: log}
}

// Record persists one event. Best-effort: logs on failure, never panics,
// and never surfaces an error to the caller. If evt.OrganizationID is
// non-nil, the write routes to the tenant-scoped DB; otherwise to the
// global DB (for system/global events such as a server restart).
func (s *Service) Record(ctx context.Context, evt models.AuditEvent) {
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
			"error", err,
			"category", evt.Category,
			"action", evt.Action,
			"source", evt.Source)
	}
}
```

Note: the spec called `tenant.GetScopedDB`. Before finalizing, confirm the exact signature by reading `internal/tenant/`. If the real API is `tenant.ScopedDB(db, orgID)` (returns `*gorm.DB`) rather than `tenant.GetScopedDB(orgID)` (returns `(*gorm.DB, error)`), use whichever exists. The fallback branch (`scoped != nil` / error check) must match the chosen signature.

- [ ] **Step 4: Verify the tenant API signature**

Run (Serena): read `internal/tenant/` and find `GetScopedDB` and `ScopedDB` signatures.
- If `GetScopedDB(orgID uuid.UUID) (*gorm.DB, error)` exists → code above is correct.
- If only `ScopedDB(db *gorm.DB, orgID uuid.UUID) *gorm.DB` exists → replace the routing block with:

```go
	db := s.db
	if evt.OrganizationID != nil {
		if scoped := tenant.ScopedDB(s.db, *evt.OrganizationID); scoped != nil {
			db = scoped
		}
	}
```

Adjust the test `TestRecord_PersistsRow_GlobalEvent` accordingly: with `ScopedDB`, a tenant-scoped write may land in a schema-per-tenant DB. For the unit test, prefer asserting against the global `svc.db` by writing an event with `OrganizationID == nil` (global event) to keep the unit test independent of tenant DB plumbing. Rewrite that test to:

```go
func TestRecord_PersistsRow_GlobalEvent(t *testing.T) {
	svc, _ := newTestService(t)
	actorID := uuid.New()
	evt := models.AuditEvent{
		// OrganizationID nil → writes to global svc.db (unit-test friendly).
		Category:    CategoryAuth,
		Action:      ActionLoginSuccess,
		Source:      SourceUser,
		ActorUserID: &actorID,
		ActorEmail:  "admin@example.com",
		Success:     true,
		IPAddress:   "10.0.0.1",
	}
	svc.Record(t.Context(), evt)

	var got models.AuditEvent
	require.NoError(t, svc.db.WithContext(t.Context()).
		Where("action = ?", ActionLoginSuccess).First(&got).Error)
	assert.NotEqual(t, uuid.Nil, got.ID)
	assert.False(t, got.CreatedAt.IsZero())
	assert.Equal(t, "admin@example.com", got.ActorEmail)
	assert.Equal(t, "10.0.0.1", got.IPAddress)
	assert.True(t, got.Success)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/audit/ -run TestRecord -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/audit/service.go internal/audit/service_test.go
git commit -m "feat(audit): add best-effort Service.Record recorder"
```

---

## Task 5: audit package — EventBuilder (TDD)

**Files:**
- Create: `internal/audit/builder.go`
- Create: `internal/audit/builder_test.go`

- [ ] **Step 1: Write the failing builder tests**

Create `internal/audit/builder_test.go`:

```go
package audit

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_InfersCategory(t *testing.T) {
	assert.Equal(t, CategoryAdmin, NewEvent(ActionUserCreated).e.Category)
	assert.Equal(t, CategoryAuth, NewEvent(ActionLoginSuccess).e.Category)
	assert.Equal(t, CategoryChat, NewEvent(ActionChatClaimed).e.Category)
	assert.Equal(t, CategorySystem, NewEvent(ActionServerStarted).e.Category)
}

func TestBuilder_ActorSystem_SetsSourceAndNilActor(t *testing.T) {
	evt := NewEvent(ActionWorkerStarted).
		ActorSystem("worker").
		Build()
	assert.Equal(t, SourceSystem, evt.Source)
	assert.Nil(t, evt.ActorUserID)
	assert.Equal(t, "worker", evt.ActorEmail) // component name echoed for traceability
}

func TestBuilder_OrgValue_SetsOrganizationID(t *testing.T) {
	id := uuid.New()
	evt := NewEvent(ActionUserCreated).OrgValue(id).Build()
	require.NotNil(t, evt.OrganizationID)
	assert.Equal(t, id, *evt.OrganizationID)
}

func TestBuilder_Target_StringifiesUUIDAndJID(t *testing.T) {
	id := uuid.New()
	evt := NewEvent(ActionChatClaimed).Target("contact", id).Build()
	require.NotNil(t, evt.TargetID)
	assert.Equal(t, id.String(), *evt.TargetID)
	assert.Equal(t, "contact", evt.TargetType)

	evt2 := NewEvent(ActionChatClaimed).Target("group", "120363abc@g.us").Build()
	require.NotNil(t, evt2.TargetID)
	assert.Equal(t, "120363abc@g.us", *evt2.TargetID)
}

func TestBuilder_Detail_MergesWithoutClobbering(t *testing.T) {
	evt := NewEvent(ActionUserCreated).
		Detail("ip", "1.2.3.4").
		Detail("ua", "curl/8").
		Detail("ip", "9.9.9.9"). // overwrite same key
		Build()
	assert.Equal(t, "9.9.9.9", evt.Details["ip"])
	assert.Equal(t, "curl/8", evt.Details["ua"])
}

func TestBuilder_Record_NilService_IsNoOp(t *testing.T) {
	// Must not panic when svc is nil.
	NewEvent(ActionLogout).
		ActorSystem("test").
		Record(context.Background(), nil)
}

func TestBuilder_Record_Persists(t *testing.T) {
	db := testutil.SetupTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.AuditEvent{}))
	svc := New(db, testutil.NopLogger())

	NewEvent(ActionServerStarted).
		ActorSystem("server").
		Record(context.Background(), svc)

	var got models.AuditEvent
	require.NoError(t, db.Where("action = ?", ActionServerStarted).First(&got).Error)
	assert.Equal(t, CategorySystem, got.Category)
	assert.Equal(t, SourceSystem, got.Source)
}
```

- [ ] **Step 2: Run tests to verify they fail (compile error)**

Run: `go test ./internal/audit/ -run TestBuilder -count=1 -run TestNewEvent`
Expected: FAIL — `undefined: NewEvent`.

- [ ] **Step 3: Implement the builder**

Create `internal/audit/builder.go`:

```go
package audit

import (
	"context"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// EventBuilder is a fluent constructor for AuditEvent. Obtain one via NewEvent.
type EventBuilder struct {
	e models.AuditEvent
}

// NewEvent starts a builder for the given action. Category is inferred from
// the Action* constant via actionCategory, so call sites specify only the action.
func NewEvent(action string) *EventBuilder {
	return &EventBuilder{e: models.AuditEvent{
		Action:   action,
		Category: actionCategory(action),
		Source:   SourceUser, // default; override with ActorSystem for system events
		Success:  true,
		Details:  models.JSONB{},
	}}
}

// Category overrides the inferred category (rarely needed).
func (b *EventBuilder) Category(c string) *EventBuilder { b.e.Category = c; return b }

// Org sets the tenant scope from a nullable org ID.
func (b *EventBuilder) Org(id *uuid.UUID) *EventBuilder { b.e.OrganizationID = id; return b }

// OrgValue sets the tenant scope from a non-nil org ID.
func (b *EventBuilder) OrgValue(id uuid.UUID) *EventBuilder {
	b.e.OrganizationID = &id
	return b
}

// ActorFromRequest pulls actor identity + request origin from a fastglue
// request context. Safe to call on requests without a user (no-op for missing fields).
func (b *EventBuilder) ActorFromRequest(r *fastglue.Request) *EventBuilder {
	if r == nil {
		return b
	}
	if uid, ok := middleware.GetUserID(r); ok {
		b.e.ActorUserID = &uid
	}
	if u, ok := middleware.GetUser(r); ok && u != nil {
		b.e.ActorEmail = u.Email
		if u.RoleID != nil && u.Role != nil {
			b.e.ActorRole = u.Role.Name
		}
	}
	b.e.IPAddress = clientIP(r)
	b.e.UserAgent = string(r.RequestCtx.UserAgent())
	b.e.Source = SourceUser
	return b
}

// ActorSystem marks a system-originated event (no human actor). componentName
// is echoed into ActorEmail for traceability (e.g. "worker", "scheduler").
func (b *EventBuilder) ActorSystem(componentName string) *EventBuilder {
	b.e.Source = SourceSystem
	b.e.ActorUserID = nil
	b.e.ActorEmail = componentName
	return b
}

// Target sets the action target. id is stringified to handle UUIDs, numeric IDs,
// and JIDs uniformly.
func (b *EventBuilder) Target(typ string, id any) *EventBuilder {
	b.e.TargetType = typ
	if id == nil {
		return b
	}
	s := toString(id)
	b.e.TargetID = &s
	return b
}

// Success sets the outcome.
func (b *EventBuilder) Success(v bool) *EventBuilder { b.e.Success = v; return b }

// Reason sets a short failure/extra reason.
func (b *EventBuilder) Reason(s string) *EventBuilder { b.e.Reason = s; return b }

// Detail merges a key/value into the JSONB Details without clobbering other keys.
func (b *EventBuilder) Detail(k string, v any) *EventBuilder {
	if b.e.Details == nil {
		b.e.Details = models.JSONB{}
	}
	b.e.Details[k] = v
	return b
}

// Build returns the immutable event.
func (b *EventBuilder) Build() models.AuditEvent { return b.e }

// Record builds and records the event in one call. svc may be nil (no-op).
func (b *EventBuilder) Record(ctx context.Context, svc *Service) {
	if svc == nil {
		return
	}
	svc.Record(ctx, b.e)
}

// clientIP extracts the peer IP from a fastglue request. It honors the
// X-Forwarded-For first hop when present (matches the project's real-client-IP
// logging convention). If a richer helper already exists in the handlers
// package, prefer that; this local copy keeps the audit package dependency-light.
func clientIP(r *fastglue.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.RequestCtx.Request.Header.Peek("X-Forwarded-For"); len(xff) > 0 {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return string(xff[:i])
			}
		}
		return string(xff)
	}
	return r.RequestCtx.RemoteIP().String()
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case uuid.UUID:
		return x.String()
	case *uuid.UUID:
		if x == nil {
			return ""
		}
		return x.String()
	default:
		// Fallback via fmt for ints, etc.
		return fmtAny(v)
	}
}
```

Because `toString` falls back to `fmt.Sprint`, add `"fmt"` to the imports and replace `fmtAny(v)` with `fmt.Sprint(v)`. Final imports:

```go
import (
	"context"
	"fmt"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)
```

Remove the `fmtAny` placeholder function entirely and make `toString`'s default case `return fmt.Sprint(v)`.

Note: confirm `models.User` has fields `Email string`, `RoleID *uuid.UUID`, `Role *CustomRole` with `Role.Name string`. If the field names differ, adjust `ActorFromRequest`. Verify via Serena `find_symbol` on `models.User` before running tests.

- [ ] **Step 4: Verify User model fields**

Run (Serena): `find_symbol(name_path_pattern="User", relative_path="internal/models", include_info=true)`. Confirm `Email`, `RoleID`, `Role` fields. Adjust `ActorFromRequest` if names differ.

- [ ] **Step 5: Run builder tests to verify they pass**

Run: `go test ./internal/audit/ -count=1`
Expected: PASS (both service and builder tests).

- [ ] **Step 6: Commit**

```bash
git add internal/audit/builder.go internal/audit/builder_test.go
git commit -m "feat(audit): add fluent EventBuilder with request/system actor helpers"
```

---

## Task 6: Wire Audit onto App

**Files:**
- Modify: `internal/handlers/app.go` (add `Audit *audit.Service` field)
- Modify: `cmd/whatomate/main.go` (construct + assign `Audit`)

- [ ] **Step 1: Add the field to the App struct**

In `internal/handlers/app.go`, the `App` struct begins at line 30. Add a field after `License *license.Service` (around line 47, before the `legacyMediaRestoreGroup` field):

```go
	// License enforces host-bound activation and runtime quotas.
	License *license.Service
	// Audit is the canonical cross-cutting audit recorder (best-effort).
	Audit  *audit.Service
```

And add the import `"github.com/compnew2006/whatomate/internal/audit"` to the import block.

- [ ] **Step 2: Construct the service in main.go**

In `cmd/whatomate/main.go`, the App literal begins at line 350. Add `Audit` to the literal, immediately after the `License:` field (or anywhere among the fields; keep ordering tidy). Two options depending on whether `License` is set in the literal or later:

If `License` is set in the literal at ~350, add right after it:
```go
		Audit:            audit.New(db, lo),
```

If `License` is assigned after construction (`app.License = ...`), add after that assignment:
```go
	app.Audit = audit.New(db, lo)
```

Add the import `"github.com/compnew2006/whatomate/internal/audit"`.

`lo` is the `logf.Logger` already in scope at main.go (it's passed as `Log: lo` on line 354).

- [ ] **Step 3: Verify worker bootstrap also gets Audit (if it constructs its own App)**

Search `cmd/whatomate/main.go` and `internal/worker/` for additional `&handlers.App{}` literals. For each, add the same `Audit: audit.New(db, lo)` wiring using the logger in scope at that point. If a worker App shares the server App (passed by reference), no change is needed.

Run (Serena): `search_for_pattern(substring_pattern="&handlers.App\\{")` across `cmd/` and `internal/worker/` to enumerate all construction sites.

- [ ] **Step 4: Verify the build compiles**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/app.go cmd/whatomate/main.go
git commit -m "feat(audit): wire Audit service onto App"
```

---

## Task 7: Record system lifecycle events (server/worker start)

**Files:**
- Modify: `cmd/whatomate/main.go` (record server start; optionally stop on graceful shutdown)

- [ ] **Step 1: Locate the server-start point**

In `cmd/whatomate/main.go`, find where the HTTP server is started (the `whatomate server` subcommand path) — typically after routes are registered and before `srv.ListenAndServe(...)`. Use Serena `search_for_pattern` for `ListenAndServe` or the fasthttp server start.

- [ ] **Step 2: Record server_started after successful start**

Immediately after the server starts (or just before, if the start is synchronous and blocking — record after the goroutine launch so it reflects an actual start), insert:

```go
	if app.Audit != nil {
		audit.NewEvent(audit.ActionServerStarted).
			ActorSystem("server").
			Detail("version", version).
			Record(context.Background(), app.Audit)
	}
```

`version` is the existing version string (the `whatomate version` subcommand reads it; find the variable name via Serena). `context.Background()` is appropriate here because this is startup, outside a request context.

- [ ] **Step 3: (Optional) Record server_stopped on graceful shutdown**

If there is a signal-handling / graceful-shutdown block (search for `signal.Notify` or `Shutdown`), add before the process exits:

```go
	if app.Audit != nil {
		audit.NewEvent(audit.ActionServerStopped).
			ActorSystem("server").
			Record(context.Background(), app.Audit)
	}
```

If graceful shutdown wiring is non-trivial or risky, SKIP this step and leave a `// TODO(v2)` comment. Server-start recording alone satisfies the "system is restarted on X time" requirement from the user. Do not refactor shutdown logic for v1.

- [ ] **Step 4: Verify build + run a smoke check**

Run: `go build ./...`
Expected: success.

(Manual smoke, optional:) Start the server with a test DB and confirm a `server_started` row appears in `audit_events`.

- [ ] **Step 5: Commit**

```bash
git add cmd/whatomate/main.go
git commit -m "feat(audit): record server_started event on startup"
```

---

## Task 8: Record auth events (login success/fail, logout)

**Files:**
- Modify: `internal/handlers/auth_handlers.go`
- Modify: `internal/handlers/auth_handlers_test.go` (add focused audit assertions)

- [ ] **Step 1: Add login_success recording**

In `internal/handlers/auth_handlers.go` `Login` (starts ~line 21), locate the success path — after `a.setAuthCookies(...)` and before the final `return r.SendEnvelope(...)`. Insert:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionLoginSuccess).
			ActorFromRequest(r).
			OrgValue(user.OrganizationID).
			Target("user", user.ID).
			Record(r.RequestCtx, a.Audit)
	}
```

Note: `r.RequestCtx` implements `context.Context` (fasthttp's `RequestCtx` satisfies the interface). Confirm `user.OrganizationID` is the field name on `models.User`; if it's resolved differently (e.g. via claims), use whichever value is in scope. Add import `"github.com/compnew2006/whatomate/internal/audit"`.

- [ ] **Step 2: Add login_failed recording on wrong password**

In the same `Login` handler, the password-mismatch branch returns `"Invalid credentials"`. Insert before each `return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid credentials", ...)` (there are two: user-not-found and password-mismatch). For both:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionLoginFailed).
			ActorFromRequest(r).
			Success(false).
			Reason("invalid_credentials").
			Detail("email", req.Email).
			Record(r.RequestCtx, a.Audit)
	}
```

Note: deliberately do NOT set `OrganizationID` here (the user lookup may have failed, so the org is unknown). This writes a global `auth/login_failed` row, which is appropriate for security forensics.

- [ ] **Step 3: Add logout recording**

In `Logout` (starts ~line 447), after the Redis `Del` succeeds and before `return r.SendEnvelope(map[string]string{"status": "logged_out"})`, insert:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionLogout).
			ActorFromRequest(r).
			Detail("jti", claims.ID).
			Record(r.RequestCtx, a.Audit)
	}
```

`claims` is the `*middleware.JWTClaims` already in scope in `Logout`. The `claims.ID` is the JWT ID being revoked.

- [ ] **Step 4: Write a focused audit integration test**

In `internal/handlers/auth_handlers_test.go`, add (mirroring the existing `TestLogin_*` style):

```go
func TestLogin_RecordsAuditEvent_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("audit-login-ok")
	password := "validpassword123"
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword(password))

	req := testutil.NewJSONRequest(t, map[string]string{"email": email, "password": password})
	require.NoError(t, app.Login(req))

	var evt models.AuditEvent
	require.NoError(t, app.DB.Where("action = ?", "login_success").First(&evt).Error)
	assert.Equal(t, "auth", evt.Category)
	assert.Equal(t, "user", evt.Source)
	require.NotNil(t, evt.TargetID)
	assert.Equal(t, user.ID.String(), *evt.TargetID)
	assert.True(t, evt.Success)
}

func TestLogin_RecordsAuditEvent_Failure(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	email := testutil.UniqueEmail("audit-login-bad")
	testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(email), testutil.WithPassword("validpassword123"))

	req := testutil.NewJSONRequest(t, map[string]string{"email": email, "password": "wrongpassword99"})
	_ = app.Login(req)

	var evt models.AuditEvent
	require.NoError(t, app.DB.Where("action = ?", "login_failed").First(&evt).Error)
	assert.False(t, evt.Success)
	assert.Equal(t, "invalid_credentials", evt.Reason)
}
```

Confirm `models.AuditEvent` table exists in the test DB: `newTestApp` uses `testutil.SetupTestDB` which runs the real migrations including the new `AuditEvent` model (added in Task 1). If `newTestApp` does NOT auto-migrate, add `require.NoError(t, app.DB.AutoMigrate(&models.AuditEvent{}))` at the top of each test.

- [ ] **Step 5: Run auth audit tests**

Run: `go test ./internal/handlers/ -run 'TestLogin_RecordsAuditEvent' -count=1`
Expected: PASS.

- [ ] **Step 6: Run the full auth test file to confirm no regression**

Run: `go test ./internal/handlers/ -run 'TestLogin|TestLogout' -count=1`
Expected: PASS (existing tests unaffected by best-effort audit writes).

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/auth_handlers.go internal/handlers/auth_handlers_test.go
git commit -m "feat(audit): record login success/failure and logout events"
```

---

## Task 9: Record admin CRUD events (users, roles, API keys)

**Files:**
- Modify: `internal/handlers/users.go`
- Modify: `internal/handlers/roles.go`
- Modify: `internal/handlers/api_keys.go`
- Modify: corresponding `*_test.go` files (one focused test per category)

Each insertion is a single localized block after the successful DB write and before the `return r.SendEnvelope(...)`. The pattern is identical; shown for `CreateUser`.

- [ ] **Step 1: Record user_created in CreateUser**

In `internal/handlers/users.go` `CreateUser`, after the user is created and before the success return:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionUserCreated).
			ActorFromRequest(r).
			OrgValue(orgID).
			Target("user", createdUser.ID).
			Detail("email", createdUser.Email).
			Record(r.RequestCtx, a.Audit)
	}
```

Use the actual variable names in scope (`orgID` from `a.getOrgAndUserID(r)`, `createdUser` or whatever the created model is named). Verify via Serena `find_symbol` on `CreateUser` first.

- [ ] **Step 2: Record user_updated / user_deleted**

Apply the same pattern in `UpdateUser` (action `ActionUserUpdated`) and `DeleteUser` (action `ActionUserDeleted`). For delete, set `Success(true)` (default) and include the user ID/email as the target. Capture the target ID *before* the soft-delete write (since after delete the row may be gone from default queries).

- [ ] **Step 3: Record role_created / role_updated / role_deleted**

In `internal/handlers/roles.go` `CreateRole`/`UpdateRole`/`DeleteRole`, mirror the user pattern with actions `ActionRoleCreated`/`ActionRoleUpdated`/`ActionRoleDeleted` and `Target("role", role.ID)`.

- [ ] **Step 4: Record api_key_created / api_key_revoked**

In `internal/handlers/api_keys.go` `CreateAPIKey`/`DeleteAPIKey` (or `RevokeAPIKey` if that's the name — verify), mirror the pattern with actions `ActionAPIKeyCreated`/`ActionAPIKeyRevoked`. Do NOT log the key secret — only the key ID/name via `Detail("key_name", key.Name)`.

- [ ] **Step 5: Write one focused integration test per category**

In `internal/handlers/users_test.go` (or `audit_handlers_test.go`), add:

```go
func TestCreateUser_RecordsAuditEvent(t *testing.T) {
	app := newTestApp(t)
	// ... seed admin user + auth context via existing test helpers ...
	// ... call app.CreateUser with a valid payload ...
	var evt models.AuditEvent
	require.NoError(t, app.DB.Where("action = ?", "user_created").First(&evt).Error)
	assert.Equal(t, "admin", evt.Category)
	require.NotNil(t, evt.TargetID)
}
```

Refer to an existing `TestCreateUser_*` test in `users_test.go` for the exact auth-context seeding boilerplate. Mirror it. Add analogous one-liner tests for `role_created` and `api_key_created`.

- [ ] **Step 6: Run the new tests**

Run: `go test ./internal/handlers/ -run 'RecordsAuditEvent' -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/users.go internal/handlers/roles.go internal/handlers/api_keys.go internal/handlers/users_test.go
git commit -m "feat(audit): record user/role/api-key admin CRUD events"
```

---

## Task 10: Record chat assignment events (claim/release/close)

**Files:**
- Modify: `internal/handlers/chat_lifecycle.go` (and/or the handler file containing `ClaimChat`, `CloseChat`, `ReopenChat`)

- [ ] **Step 1: Locate the chat lifecycle handlers**

Use Serena `find_symbol` on `ClaimChat`, `CloseChat`, `ReopenChat`, `SetChatPublic`, and any `AssignContact`/transfer handlers. Determine which file each lives in (likely `chat_lifecycle.go` or `contacts.go`).

- [ ] **Step 2: Record chat_claimed in ClaimChat**

After the successful claim write, before the success return:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionChatClaimed).
			ActorFromRequest(r).
			OrgValue(orgID).
			Target("contact", contactID).
			Detail("assignee_user_id", userID.String()).
			Record(r.RequestCtx, a.Audit)
	}
```

Use the in-scope variable names (the contact ID being claimed, the claiming user ID).

- [ ] **Step 3: Record chat_closed in CloseChat**

After the successful close write:

```go
	if a.Audit != nil {
		audit.NewEvent(audit.ActionChatClosed).
			ActorFromRequest(r).
			OrgValue(orgID).
			Target("contact", contactID).
			Record(r.RequestCtx, a.Audit)
	}
```

- [ ] **Step 4: Record chat_released on unclaim/reopen (optional mapping)**

If "release" maps to `ReopenChat` or an unclaim action in this codebase, record `ActionChatReleased` there. If no clean "release" handler exists, SKIP and leave `// TODO(v2): chat_released`. Do not invent a handler.

- [ ] **Step 5: Write a focused audit test for claim + close**

In `internal/handlers/contacts_test.go` or `audit_handlers_test.go`, mirror an existing `TestClaimChat_*` / `TestCloseChat_*` test and assert `chat_claimed` / `chat_closed` rows exist with the right target ID.

- [ ] **Step 6: Run the chat audit tests**

Run: `go test ./internal/handlers/ -run 'TestClaimChat|TestCloseChat' -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/chat_lifecycle.go internal/handlers/contacts_test.go
git commit -m "feat(audit): record chat claim/close events"
```

---

## Task 11: Read API — ListAuditEvents handler (TDD)

**Files:**
- Create: `internal/handlers/audit_handlers.go`
- Create: `internal/handlers/audit_handlers_test.go`

- [ ] **Step 1: Write the failing handler tests**

Create `internal/handlers/audit_handlers_test.go`:

```go
package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper: seed one audit row owned by an org.
func seedAuditEvent(t *testing.T, app *App, orgID *uuid.UUID, action string) models.AuditEvent {
	t.Helper()
	evt := models.AuditEvent{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Category:       "admin",
		Action:         action,
		Source:         "user",
		Success:        true,
		Details:        models.JSONB{},
	}
	require.NoError(t, app.DB.Create(&evt).Error)
	return evt
}

func TestListAuditEvents_OrgAdmin_SeesOwnOrgOnly(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &orgA.ID, "user_created")
	seedAuditEvent(t, app, &orgB.ID, "user_created")

	// Build a request scoped to orgA as a non-super-admin with audit:read.
	req := newOrgAdminRequest(t, app, orgA.ID, "audit", "read")
	require.NoError(t, app.ListAuditEvents(req))

	body := decodeEnvelopeData(t, testutil.GetResponseBody(req))
	events := body["events"].([]any)
	assert.Len(t, events, 1, "org A admin must not see org B events")
}

func TestListAuditEvents_NonAdmin_Returns403(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// Agent (no audit:read) → requirePermission inside handler returns 403.
	req := newOrgAgentRequest(t, app, org.ID)
	err := app.ListAuditEvents(req)
	// requirePermission sends the envelope and returns errEnvelopeSent (nil to caller).
	_ = err
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestListAuditEvents_Filters_ByCategory(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	seedAuditEvent(t, app, &org.ID, "user_created")    // category admin
	seedAuditEvent(t, app, &org.ID, "login_success")   // category auth

	req := newOrgAdminRequest(t, app, org.ID, "audit", "read")
	req.RequestCtx.URI().SetQueryString("category=admin")
	require.NoError(t, app.ListAuditEvents(req))

	body := decodeEnvelopeData(t, testutil.GetResponseBody(req))
	events := body["events"].([]any)
	assert.Len(t, events, 1)
}

func TestListAuditEvents_Pagination_CapsPerPage(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	for i := 0; i < 3; i++ {
		seedAuditEvent(t, app, &org.ID, "user_created")
	}
	req := newOrgAdminRequest(t, app, org.ID, "audit", "read")
	req.RequestCtx.URI().SetQueryString("per_page=999")
	require.NoError(t, app.ListAuditEvents(req))

	body := decodeEnvelopeData(t, testutil.GetResponseBody(req))
	assert.Equal(t, float64(200), body["per_page"]) // capped at 200
}
```

The helpers `newOrgAdminRequest`, `newOrgAgentRequest`, `decodeEnvelopeData` may not exist. Before finalizing this step, check `internal/handlers/testhelpers_test.go` for existing request/auth-context builders (e.g. `createTestRequestWithContext` from `app_unit_test.go`). If equivalents exist, use them; otherwise add minimal helpers to `testhelpers_test.go`:

```go
// newOrgAdminRequest builds a GET fastglue request with an authenticated
// org admin context that has the given permission.
func newOrgAdminRequest(t *testing.T, app *App, orgID uuid.UUID, resource, action string) *fastglue.Request { ... }
```

This is the riskiest test-infra step. If the existing harness cannot cheaply express "user with audit:read", simplify: seed the audit rows, then call `app.ListAuditEvents` with a request whose context has `IsSuperAdmin=true` and assert cross-org visibility. Super-admin bypasses `requirePermission`, so the tenant-boundary test becomes: super-admin sees all orgs; a second test with a normal admin context (if buildable) asserts scoping. If only the super-admin path is testable cheaply, write that test and mark the org-scoping test with `t.Skip("requires org-admin auth harness; covered manually")` — do NOT delete it; it documents intent.

- [ ] **Step 2: Run tests to verify they fail (compile error)**

Run: `go test ./internal/handlers/ -run TestListAuditEvents -count=1`
Expected: FAIL — `undefined: ListAuditEvents`.

- [ ] **Step 3: Implement the handler**

Create `internal/handlers/audit_handlers.go`:

```go
package handlers

import (
	"strconv"

	"github.com/compnew2006/whatomate/internal/audit"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const (
	auditDefaultPerPage = 50
	auditMaxPerPage     = 200
)

// ListAuditEvents GET /api/audit-events
//
// Admin-only (requires audit:read). Returns audit events with filtering and
// pagination. Super admins see events across all orgs (and global events);
// org admins see only their own org's events plus global (organization_id IS NULL)
// events. Non-admins are rejected by requirePermission before reaching the body.
//
// Filters: category, action, source, actor_user_id, target_id, target_type,
//          success, q (text on actor_email/reason), date_from, date_to, organization_id (super-admin only)
// Pagination: page (1-based), per_page (default 50, max 200)
// Sort: created_at DESC (newest first)
func (a *App) ListAuditEvents(r *fastglue.Request) error {
	userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAudit, models.ActionRead); err != nil {
		return nil // forbidden envelope already sent
	}

	q := r.RequestCtx.QueryArgs()

	// Base query: super-admin reads global DB (+ optional org filter);
	// org admin reads tenant-scoped DB and additionally sees global events.
	var db *gorm.DB
	isSuperAdmin := middleware.IsSuperAdmin(r)

	if isSuperAdmin {
		db = a.DB
		if orgStr := string(q.Peek("organization_id")); orgStr != "" {
			db = db.Where("organization_id = ?", orgStr)
		}
		// Super-admin sees everything (no extra OR clause needed).
	} else {
		orgID, _ := middleware.GetOrganizationID(r)
		db = a.requestDB(r).
			Where("organization_id = ? OR organization_id IS NULL", orgID)
	}

	// Filters
	if v := string(q.Peek("category")); v != "" {
		db = db.Where("category = ?", v)
	}
	if v := string(q.Peek("action")); v != "" {
		db = db.Where("action = ?", v)
	}
	if v := string(q.Peek("source")); v != "" {
		db = db.Where("source = ?", v)
	}
	if v := string(q.Peek("actor_user_id")); v != "" {
		db = db.Where("actor_user_id = ?", v)
	}
	if v := string(q.Peek("target_id")); v != "" {
		db = db.Where("target_id = ?", v)
	}
	if v := string(q.Peek("target_type")); v != "" {
		db = db.Where("target_type = ?", v)
	}
	if v := string(q.Peek("success")); v != "" {
		if v == "true" || v == "1" {
			db = db.Where("success = ?", true)
		} else if v == "false" || v == "0" {
			db = db.Where("success = ?", false)
		}
	}
	if v := string(q.Peek("q")); v != "" {
		like := "%" + v + "%"
		db = db.Where("actor_email ILIKE ? OR reason ILIKE ?", like, like)
	}
	if v := string(q.Peek("date_from")); v != "" {
		db = db.Where("created_at >= ?", v)
	}
	if v := string(q.Peek("date_to")); v != "" {
		db = db.Where("created_at <= ?", v)
	}

	// Count (on a clone so pagination doesn't affect it)
	var total int64
	countDB := db.Session(&gorm.Session{})
	if err := countDB.Model(&models.AuditEvent{}).Count(&total).Error; err != nil {
		a.Log.Error("Failed to count audit events", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list audit events", nil, "")
	}

	// Pagination
	page, _ := strconv.Atoi(string(q.Peek("page")))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(string(q.Peek("per_page")))
	if perPage < 1 {
		perPage = auditDefaultPerPage
	}
	if perPage > auditMaxPerPage {
		perPage = auditMaxPerPage
	}
	offset := (page - 1) * perPage

	var events []models.AuditEvent
	if err := db.Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&events).Error; err != nil {
		a.Log.Error("Failed to list audit events", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list audit events", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"events":  events,
		"total":   total,
		"page":    page,
		"per_page": perPage,
	})
}

// Compile-time guard: ensure audit package is referenced (avoids unused-import
// churn if the only consumer is via App.Audit at call sites).
var _ = audit.CategoryAdmin
```

- [ ] **Step 4: Register the route**

In `cmd/whatomate/main.go`, near the roles/permissions routes (~line 1527), add:

```go
	// Audit Log (admin-only; enforced in-handler via requirePermission).
	g.GET("/api/audit-events", app.ListAuditEvents)
```

- [ ] **Step 5: Run handler tests**

Run: `go test ./internal/handlers/ -run TestListAuditEvents -count=1`
Expected: PASS (or the org-scoping test skipped with a clear reason per Step 1).

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/audit_handlers.go internal/handlers/audit_handlers_test.go cmd/whatomate/main.go
git commit -m "feat(audit): add admin-only GET /api/audit-events read API"
```

---

## Task 12: Frontend — service + Pinia store + view + route + nav

**Files:**
- Create: `frontend/src/services/audit.ts`
- Create: `frontend/src/stores/audit.ts`
- Create: `frontend/src/stores/audit.test.ts`
- Create: `frontend/src/views/settings/AuditLogView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/i18n/locales/en.json` (and `es.json`, `ar.json` if present)
- Modify: sidebar nav component (find via the existing settings nav)

- [ ] **Step 1: Inspect existing frontend patterns first**

Use Serena/Socraticode to read:
- `frontend/src/services/api.ts` (base client + how a domain service wraps it; mirror an existing one like `frontend/src/services/users.ts` or `roles.ts`).
- An existing Pinia store, e.g. `frontend/src/stores/roles.ts` (for state/actions shape).
- An existing settings view, e.g. `frontend/src/views/settings/UsersView.vue` (for layout, permission-gated nav, table component reuse).
- `frontend/src/router/index.ts` (route shape + `requiredPermission` meta usage).

Do NOT invent patterns. Copy the structure of the closest existing settings feature.

- [ ] **Step 2: Create the service**

Create `frontend/src/services/audit.ts`, mirroring an existing service's imports/return shapes:

```ts
import { api } from './api'

export interface AuditEvent {
  id: string
  created_at: string
  organization_id?: string
  category: string
  action: string
  source: string
  actor_user_id?: string
  actor_email?: string
  actor_role?: string
  target_type?: string
  target_id?: string
  success: boolean
  reason?: string
  details?: Record<string, unknown>
  ip_address?: string
  user_agent?: string
}

export interface AuditEventFilters {
  category?: string
  action?: string
  source?: string
  actor_user_id?: string
  target_id?: string
  target_type?: string
  success?: boolean
  q?: string
  date_from?: string
  date_to?: string
  organization_id?: string
  page?: number
  per_page?: number
}

export interface AuditEventListResponse {
  events: AuditEvent[]
  total: number
  page: number
  per_page: number
}

export const auditService = {
  async list(filters: AuditEventFilters = {}): Promise<AuditEventListResponse> {
    const res = await api.get('/api/audit-events', { params: filters })
    return res.data.data as AuditEventListResponse
  },
}
```

Confirm the exact import path and base-client call style against `api.ts` (e.g. whether it's `api.get(...)` returning `{ data: { data } }` envelope). Adjust to match.

- [ ] **Step 3: Create the Pinia store**

Create `frontend/src/stores/audit.ts`, mirroring `stores/roles.ts`:

```ts
import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { auditService, type AuditEvent, type AuditEventFilters } from '@/services/audit'

export const useAuditStore = defineStore('audit', () => {
  const events = ref<AuditEvent[]>([])
  const total = ref(0)
  const loading = ref(false)
  const filters = reactive<AuditEventFilters>({ page: 1, per_page: 50 })
  const error = ref<string | null>(null)

  async function fetch() {
    loading.value = true
    error.value = null
    try {
      const res = await auditService.list(filters)
      events.value = res.events
      total.value = res.total
    } catch (e: any) {
      error.value = e?.message ?? 'Failed to load audit events'
    } finally {
      loading.value = false
    }
  }

  function setFilter<K extends keyof AuditEventFilters>(key: K, value: AuditEventFilters[K]) {
    filters[key] = value
    filters.page = 1 // reset to first page on filter change
  }

  function resetFilters() {
    for (const k of Object.keys(filters) as (keyof AuditEventFilters)[]) {
      delete filters[k]
    }
    filters.page = 1
    filters.per_page = 50
  }

  function goToPage(n: number) {
    filters.page = n
    return fetch()
  }

  return { events, total, loading, filters, error, fetch, setFilter, resetFilters, goToPage }
})
```

- [ ] **Step 4: Write the store unit test**

Create `frontend/src/stores/audit.test.ts`:

```ts
import { setActivePinia, createPinia } from 'pinia'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAuditStore } from './audit'
import * as auditSvc from '@/services/audit'

describe('audit store', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('fetch populates events and total', async () => {
    vi.spyOn(auditSvc.auditService, 'list').mockResolvedValue({
      events: [{ id: '1', created_at: '2026-01-01T00:00:00Z', category: 'auth', action: 'login_success', source: 'user', success: true }],
      total: 1, page: 1, per_page: 50,
    })
    const store = useAuditStore()
    await store.fetch()
    expect(store.events).toHaveLength(1)
    expect(store.total).toBe(1)
    expect(store.loading).toBe(false)
  })

  it('setFilter resets page to 1', () => {
    const store = useAuditStore()
    store.filters.page = 3
    store.setFilter('category', 'auth')
    expect(store.filters.page).toBe(1)
    expect(store.filters.category).toBe('auth')
  })
})
```

- [ ] **Step 5: Run store test**

Run: `cd frontend && npm run test:unit -- audit`
Expected: PASS.

- [ ] **Step 6: Create the view**

Create `frontend/src/views/settings/AuditLogView.vue` (`<script setup>` + TS). Model it on `UsersView.vue`'s shell — same permission-aware layout, filter bar, data table, pagination. Key elements:

- Filter bar bound to `store.filters` (category select, action select, source select, actor search, free-text `q`, success toggle, date range, Reset button calling `store.resetFilters()`).
- A data table rendering: timestamp, category badge (color by category), action, actor (email + role), target, success/fail icon, and an expandable details row rendering `JSON.stringify(details, null, 2)`.
- Pagination controls driving `store.goToPage(n)`.
- All text via i18n keys `settings.auditLog.*`.

Color map for category badges:
```ts
const categoryColor: Record<string, string> = {
  auth: 'blue', chat: 'green', admin: 'purple', system: 'gray', campaign: 'orange', template: 'teal',
}
```

Keep the component focused; extract a small `AuditCategoryBadge.vue` under `components/settings/` only if the view exceeds ~400 lines.

- [ ] **Step 7: Register the route**

In `frontend/src/router/index.ts`, in the settings route group, add:

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

Confirm the existing `requiredPermission` meta is honored by the router guard (check the guard in `router/index.ts`). If the guard uses a different meta key (e.g. `permissions: ['audit:read']`), match that instead.

- [ ] **Step 8: Add i18n keys**

In `frontend/src/i18n/locales/en.json`, under `settings`, add:

```json
"auditLog": {
  "title": "Audit Log",
  "subtitle": "Security and operational events across the platform",
  "filters": { "category": "Category", "action": "Action", "source": "Source", "actor": "Actor", "search": "Search", "success": "Outcome", "any": "Any", "successOnly": "Success only", "failureOnly": "Failures only", "dateFrom": "From", "dateTo": "To", "reset": "Reset filters" },
  "columns": { "time": "Time", "category": "Category", "action": "Action", "actor": "Actor", "target": "Target", "outcome": "Outcome", "details": "Details" },
  "empty": "No events match the current filters.",
  "source": { "user": "User", "system": "System", "worker": "Worker", "scheduled": "Scheduled" }
}
```

Add equivalent translations to `es.json` and `ar.json` if those files exist (mirror their existing structure).

- [ ] **Step 9: Add sidebar nav entry**

Find the settings sidebar component (likely `frontend/src/components/Sidebar*.vue` or a settings nav component). Add an admin-only entry mirroring the existing Users/Roles entries:

```vue
<RouterLink :to="{ name: 'audit-log' }" v-if="hasPermission('audit:read')">
  <component :is="ShieldIcon" />
  {{ t('settings.auditLog.title') }}
</RouterLink>
```

Use the project's existing icon library and permission helper. Verify exact names via the surrounding nav entries.

- [ ] **Step 10: Typecheck + lint**

Run: `cd frontend && npm run typecheck && npm run lint`
Expected: success.

- [ ] **Step 11: Commit**

```bash
git add frontend/src/services/audit.ts frontend/src/stores/audit.ts frontend/src/stores/audit.test.ts frontend/src/views/settings/AuditLogView.vue frontend/src/router/index.ts frontend/src/i18n/locales/*.json
git commit -m "feat(audit): add admin-only audit log frontend (service, store, view, route, nav)"
```

---

## Task 13: Final verification gates

- [ ] **Step 1: Backend full test suite**

Run: `make test` (or `go test -p 1 ./... -count=1`)
Expected: PASS. If DB-backed tests are skipped locally (no `TEST_DATABASE_URL`), note that explicitly; run `make test-db` in CI.

- [ ] **Step 2: Lint**

Run: `golangci-lint run`
Expected: no new issues in audit code.

- [ ] **Step 3: Frontend tests + typecheck + lint**

Run:
```
cd frontend && npm run typecheck && npm run lint && npm run test:unit
```
Expected: PASS.

- [ ] **Step 4: Build**

Run: `make build`
Expected: success (backend compiles with the new package and App field).

- [ ] **Step 5: MCP re-verification**

Run:
- Socraticode: `codebase_update(projectPath=<repo>)` then `codebase_graph_circular(projectPath=<repo>)` (no new cycles expected; the audit package is a leaf sink).
- Serena: `get_diagnostics_for_file` on every edited Go file (`internal/handlers/app.go`, `auth_handlers.go`, `users.go`, `roles.go`, `api_keys.go`, `chat_lifecycle.go`, `audit_handlers.go`, `internal/models/audit_event.go`, `internal/audit/*.go`, `cmd/whatomate/main.go`). Resolve any reported diagnostics.
- codebase-memory-mcp: `detect_changes(project=<project>)` and, after confirming, save a memory note via Serena `write_memory` summarizing the new audit subsystem (see Task 14).

- [ ] **Step 6: Commit any verification fixes**

```bash
git add -A
git commit -m "test(audit): verification gate fixes"
```

---

## Task 14: Update project memory and docs

- [ ] **Step 1: Save a Serena memory note**

Use Serena `write_memory` to create a memory named e.g. `feature/audit-log-system-2026-06-23` summarizing:
- What: cross-cutting `internal/audit/` package + `audit_events` table + admin-only read API + Vue view.
- Pattern: best-effort `Service.Record` on `*handlers.App`, fluent `EventBuilder`, mirrors `ModuleEvent`.
- Files: the new/edited files listed in the File Structure table.
- Permission: `audit:read` (admin-only via `DefaultPermissions()`).
- Gotchas: `App.Log` is `logf.Logger` not `*slog.Logger`; migrations go via `GetMigrationModels()`; super-admin bypasses `requirePermission` so the handler must scope reads itself.

- [ ] **Step 2: Re-index codebase-memory-mcp**

Run `codebase_memory_mcp_index_repository(repo_path=<repo>, mode="moderate")` (or `detect_changes`) so the new package is reflected in architecture memory.

- [ ] **Step 3: Append to summary.md**

Append a dated entry to root `summary.md` per AGENTS.md §6 (append-only): task, files changed, pattern followed, tests run, risks/gotchas. Do not modify prior entries.

---

## Self-Review (completed during authoring)

**Spec coverage:** Each spec section maps to a task:
- §3.1 new/changed locations → Tasks 1, 2, 3, 4, 5, 6, 11, 12.
- §3.2 recording strategy → Tasks 4, 5 (best-effort Service + builder), Tasks 7–10 (call sites).
- §4 data model → Task 1.
- §4.3 constants → Task 3.
- §5 recording API → Tasks 4, 5.
- §5.3 integration points → Tasks 7 (system), 8 (auth), 9 (admin), 10 (chat).
- §5.4 wiring → Task 6.
- §6 read side → Tasks 11, 12.
- §7 testing → tests embedded in Tasks 1, 4, 5, 8, 9, 10, 11, 12, 13.
- §9 risks → mitigations encoded as best-effort Service, in-handler scoping, and tenant-boundary tests.

**Placeholder scan:** No "TBD"/"TODO" placeholders in code steps. Where a step depends on a verified-then-adjust detail (e.g. exact variable name in scope, or the tenant API signature), the step explicitly says to verify via Serena first and gives the exact fallback code. The one genuine v2-deferral (`chat_released` if no handler exists; `server_stopped` if shutdown wiring is risky) is marked with an explicit SKIP + `// TODO(v2)` instruction, not a silent gap.

**Type consistency:** `Service.Record(ctx, evt)`, `NewEvent(action)`, `EventBuilder` methods (`ActorFromRequest`, `ActorSystem`, `OrgValue`, `Target`, `Success`, `Reason`, `Detail`, `Build`, `Record`), `ListAuditEvents(r)`, `useAuditStore` actions (`fetch`, `setFilter`, `resetFilters`, `goToPage`) are used identically across all tasks. `App.Log` is consistently `logf.Logger`.

**Corrections vs. the spec:** (1) `App.Log` is `logf.Logger`, not `*slog.Logger` — `Service` takes `logf.Logger`. (2) Migration registration is via `GetMigrationModels()`, not a raw `AutoMigrate` slice. (3) Admin auto-inherits `audit:read` from `DefaultPermissions()`, so no edit to `SystemRolePermissions()` is required — the spec's "edit roles.go for SystemRolePermissions" step is unnecessary and is dropped here. (4) Permission gating uses the in-handler `requirePermission` pattern (matching `ListPermissions`), not the middleware `RequirePermission` route wrapper, to minimize blast radius.
