<!--
  Sync Impact Report

  Version change: v1.0.0 → v1.1.0
  Modified principles: N/A (minor version additions)
  Added sections:
    - §1.5 Licensing System Pathways
    - §3.8 Campaign Execution & Worker Policies
    - §4.9 Chat Workflows & Collaboration
    - §8.9 Security Audit Rules
    - §8.10 Unified Safe Origin Evaluator
  Removed sections: N/A

  Templates requiring updates:
    ✅ .opencode/skills/04-speckit.plan/templates/plan-template.md
    ✅ .opencode/skills/02-speckit.specify/templates/spec-template.md
    ✅ .opencode/skills/05-speckit.tasks/templates/tasks-template.md
    ✅ AGENTS.md

  Follow-up TODOs: None
-->

# Constitution of the Whatomate Project

**Version**: 1.1.0
**Ratification Date**: 2026-01-01
**Last Amended**: 2026-06-06

## Preamble

This constitution establishes the binding principles, standards, and
non-negotiable rules for all development work on the Whatomate project.
Every contributor, AI agent, and automation tool MUST adhere to these
provisions. The constitution serves as the "Source of Law" — all design
decisions, code reviews, and implementation plans are judged against it.

---

## §1 Project Identity

### 1.1 Definition

Whatomate is an open-source WhatsApp Business platform delivered as a
single Go binary embedding a Vue 3 SPA. It serves multi-tenant
organizations with real-time chat, chatbot automation, campaigns,
Facebook integration, and analytics.

### 1.2 Technology Stack

- **Language**: Go (module `github.com/compnew2006/whatomate`)
- **HTTP framework**: `fasthttp` + `fastglue` (NOT `net/http`)
- **Database ORM**: GORM with PostgreSQL 17
- **Cache/Queue**: Redis 7 (streams, rate limiting, sessions)
- **Config**: TOML via koanf
- **Frontend**: Vue 3 (Composition API) + Vite + Pinia + TypeScript
- **Frontend UI**: shadcn-vue (new-york) + Tailwind CSS v3
- **Frontend i18n**: vue-i18n (en, es, ar shipped)
- **CI Go version**: 1.25.8

### 1.3 Provider Abstraction

WhatsApp operations MUST go through the `MessageProvider` interface in
`pkg/provider/`. Code MUST NOT import Meta-specific or WhatsMeow-specific
packages outside their respective adapter packages (`pkg/whatsapp/`,
`pkg/whatsmeow/`). Provider-specific routes MUST use
`app.ProviderGuard("meta", handler)` or equivalent.

### 1.4 Frontend Embedding

The Vue SPA MUST be compiled into the Go binary via
`//go:embed all:dist` in `internal/frontend/embed.go`. The production
build MUST run `make build-prod` which builds the frontend first, then
compiles the binary. Raw `go build` without the frontend build step
will produce an incomplete artifact.

### 1.5 Licensing System Pathways

The licensing system supports two distinct deployment pathways:

1.  **Self-Hosted Pathway**:
    - The `/api/license/bootstrap` endpoint MUST remain public, returning the hardware identifier (`hwid_full`).
    - The `/api/license/activate` endpoint MUST verify the license token locally using Ed25519 cryptographic signatures.
    - All token tappings/activation requests on `/api/license/activate` MUST be rate-limited to a maximum of 5 attempts per hour per IP. Audit logs for failed activations MUST store only a hash of the token (`hash(token)`), never the raw token string.
2.  **Hosted Pathway**:
    - The internal `/internal/license/bootstrap` endpoint MUST be protected by mTLS or signed internal JWTs, returning `instance_id`, `hwid_full`, `hwid_hash`, and a single-use `bootstrap_nonce` (TTL <= 5 minutes, bound to a specific `deployment_id`).
    - The Issuer service MUST enforce idempotency via a client-supplied `request_id` cached for 24 hours.
    - Hosted license tokens MUST carry a `deployment_id` claim, and instances MUST reject any tokens where the claim does not match their own local `deployment_id`.
3.  **Key Isolation and Key Rotation**:
    - The private key (`private.key`) MUST reside exclusively within the isolated Private Issuer Service and MUST NOT leak to other backend binaries.
    - Tokens MUST carry a Key ID (`kid`) claim. The system MUST support current and previous key IDs simultaneously. The previous key MUST enter a verify-only state for a grace window of 90 days, after which it is completely rejected.

---

## §2 Governance

### 2.1 Amendment Procedure

1. Any contributor may propose an amendment by opening a PR against
   `.specify/memory/constitution.md`.
2. Amendments require review by at least one maintainer.
3. Substantive principle changes (MAJOR or MINOR per §2.2) MUST be
   accompanied by a Sync Impact Report (see §2.3).
4. Upon approval, the `LAST_AMENDED` date and `CONSTITUTION_VERSION`
   MUST be updated, and all dependent templates MUST be propagated.

### 2.2 Versioning

Version follows semantic versioning:

- **MAJOR**: Backward-incompatible governance changes — principle
  removals, redefinitions of core non-negotiables.
- **MINOR**: New principles or materially expanded guidance.
- **PATCH**: Clarifications, wording refinements, typo fixes.

### 2.3 Sync Impact Report

Every amendment MUST prepend an HTML comment to the constitution file
listing:
- Version change (old → new)
- Modified principles (old title → new title)
- Added / removed sections
- Template files checked and their update status
- Any deferred TODOs

### 2.4 Compliance Review

- All feature specs MUST include a Constitution Check gate
  (see plan-template.md).
- Code reviews MUST flag violations of constitutional principles.
- New dependencies and architectural changes MUST be assessed against
  §11 (Architecture Integrity).

---

## §3 Backend Rules (Go)

### 3.1 HTTP Framework

ALL HTTP handling MUST use `fasthttp` and `fastglue`. The Go standard
library's `net/http` package, `http.Handler`, `httptest.Server`, and
related types MUST NOT be used except in test utilities that adapt
fasthttp types for test convenience.

### 3.2 Handler Pattern

Every API handler MUST be a method on the `handlers.App` struct:

```go
func (a *App) HandlerName(r *fastglue.Request) error
```

Handler return MUST follow the envelope pattern — either
`r.SendEnvelope(data)` for success or
`r.SendErrorEnvelope(httpStatus, message, nil, "")` for errors.

### 3.3 Error Handling

- Handlers MUST return the sentinel `errEnvelopeSent` when an error
  response has already been written, then return `nil` to the framework.
- Helper functions that write error responses MUST return
  `errEnvelopeSent` so callers don't double-write.
- All errors MUST be logged before returning a user-facing message.
- Internal error details MUST NOT leak to API responses.

### 3.4 Config Pattern

All configuration MUST use koanf with TOML files. The `Config` struct
in `internal/config/config.go` is the single source of truth. New
configuration sections MUST be added as sub-structs of the main `Config`.

### 3.5 Imports

Imports MUST be grouped in three blocks separated by blank lines:
1. Standard library
2. Third-party / external packages (alphabetically sorted)
3. Internal project packages (`github.com/compnew2006/whatomate/...`)

Aliased imports MUST be used when the package basename differs from
the last path element.

### 3.6 Concurrency & Goroutines

- All concurrent operations MUST handle cancellation via `context.Context`.
- Goroutine lifetimes MUST be bounded — no fire-and-forget goroutines
  without a lifecycle management mechanism.
- Shared state MUST be protected by `sync.Mutex`, `sync.RWMutex`, or
  channels, not by unstructured shared memory.

### 3.7 Logging

Use the `logf.Logger` instance from `handlers.App.Log`. Structured
logging with key-value pairs is REQUIRED. Log messages MUST include
sufficient context (user ID, org ID, operation) for debugging.

### 3.8 Campaign Execution & Worker Policies

1.  **Anti-Ban Throttling / Delay Floor**:
    - Every campaign MUST define `MinDelaySeconds` and `MaxDelaySeconds` for inter-message delays.
    - The queue worker MUST apply a random delay within this range for each recipient, enforcing a strict minimum floor of 10 seconds.
2.  **Queue Isolation & Consumer Groups**:
    - All campaigns MUST process jobs via tenant-scoped Redis Streams: stream names MUST follow the pattern `whatomate:campaigns:<orgID>` and consumer groups `campaign-workers:<orgID>`.
    - Idempotency locks (2-minute TTL) MUST be acquired in Redis on recipient IDs during processing.
3.  **Worker Autoscaling**:
    - The `WorkerScaler` MUST run a reconcile loop to dynamically scale workers based on organization queue depth.
    - Organizations MUST NOT exceed the maximum workers quota (`max_workers_per_org`) defined by their active license.
    - Scaling cooldowns MUST be applied to prevent thrashing.
    - The scaler MUST temporarily freeze processing for organizations that experience 3 consecutive start failures or have no healthy WhatsApp instances connected.
4.  **Campaign State and Recipient Modification**:
    - Recipient lists are mutable ONLY in `draft` status.
    - Once a campaign status changes to `processing`, modifications to the recipient list are strictly FORBIDDEN.
    - Auto-pause MUST be triggered if a WhatsApp instance is banned or disconnected during campaign execution.
5.  **Strict Sending Restrictions (Inbound-Only)**:
    - When `strict_sending_restrictions_enabled` is true, and the outbound mode is set to `inbound_only`, campaigns MUST reject sending messages to any phone numbers that do not have prior inbound history (excluding system override parameters).

---

## §4 Frontend Rules (Vue)

### 4.1 Component Style

ALL Vue components MUST use the Composition API with
`<script setup lang="ts">`. Options API (`defineComponent({ })`) is
FORBIDDEN for new code.

### 4.2 State Management

- Application state MUST be managed through Pinia stores using the
  setup function (Composition API) style.
- Server state (data from API) SHOULD use `@tanstack/vue-query`.
- Pinia stores MUST follow the naming convention `useXxxStore` with
  the store definition name matching the file base name.

### 4.3 Routing

- All route definitions MUST be in `frontend/src/router/index.ts`.
- Every protected route MUST have a `meta.permission` field.
- Lazy loading via `() => import("@/views/...")` is REQUIRED for all
  view components.
- Navigation sidebar MUST be built from the `navigationOrder` array.

### 4.4 API Client

- All API calls MUST go through the shared Axios instance in
  `src/services/api.ts`.
- CSRF token (`X-CSRF-Token`) MUST be sent on all mutating requests.
- Organization ID header (`X-Organization-ID`) MUST be sent on
  org-scoped requests.
- 401 responses MUST trigger a single refresh attempt before failing.

### 4.5 i18n

- All user-facing text MUST use `vue-i18n` via the `useI18n()` composable.
- New locale keys MUST be added to all three shipped locales
  (`en.json`, `es.json`, `ar.json`).
- Text MUST NOT be hardcoded in templates or composables.

### 4.6 TypeScript

- ALL `.vue` and `.ts` files MUST use TypeScript.
- The `noUnusedLocals` and `noUnusedParameters` tsconfig flags are
  mandatory — unused code MUST be removed, not suppressed.
- The `any` type is permitted but SHOULD be avoided in favor of
  specific types or generics.

### 4.7 CSS & Styling

- Styling MUST use Tailwind CSS utility classes.
- shadcn-vue components in `src/components/ui/` MUST use the
  directory-per-component pattern with an `index.ts` re-export.
- Class-variance-authority (`cva`) MUST be used for component variants.
- The `vue/no-v-html` ESLint rule is ERROR — `v-html` is FORBIDDEN.

### 4.8 File Naming

| Category | Convention | Example |
|---|---|---|
| View components | PascalCase, `*View.vue` | `LoginView.vue` |
| Generic components | PascalCase | `LanguageSwitcher.vue` |
| Composables | camelCase, `use*` | `usePagination.ts` |
| Pinia stores | camelCase | `auth.ts` |
| Services/API | camelCase | `api.ts` |
| Type definitions | camelCase | `contacts.ts` |

### 4.9 Chat Workflows & Collaboration

1.  **Chat Lifecycle States**:
    - Conversations (contacts) MUST follow three lifecycle states: `ChatStatusPending` (incoming, unassigned), `ChatStatusOpen` (assigned to an agent, active), and `ChatStatusClosed` (marked resolved).
    - If `assigned_user_id` is populated, the chat is considered active (effectively open), regardless of its raw database state.
2.  **Multi-Account Integration**:
    - When a contact communicates across multiple WhatsApp accounts/instances, the frontend MUST support account toggling.
    - Sidebar state for selected accounts MUST be unified and persisted via `ChatSidebarUnifier`.
3.  **Collaborator Permissions**:
    - Chats sharing MUST support three collaborator roles:
      - `Owner`: Full access and administrative operations.
      - `Editor`: Allowed to send messages and update status.
      - `Viewer`: Read-only message history, blocked from sending.
4.  **Phone Number Masking**:
    - If phone number masking is enabled, phone numbers in UI elements MUST be formatted as `+1**********23` (retaining country code and the last 2 digits).
    - Masking is bypassed ONLY for super-admins or users with `mask_phone_numbers: false` explicitly set in settings.
5.  **Service Window Tracking**:
    - The UI MUST show a 24-hour service window indicator based on `last_inbound_at`. If `last_inbound_at` is within the past 24 hours, the service window is considered open.

---

## §5 API Compatibility

### 5.1 Backward Compatibility

- API responses in the `data` field of the envelope MUST maintain
  backward-compatible JSON field names. Adding new fields is permitted;
  removing or renaming existing fields is a BREAKING change.
- New fields MUST be optional (omitted or nullable) to avoid breaking
  existing frontend code.
- Breaking changes MUST be announced with a MAJOR version coordination
  and a migration window.

### 5.2 Response Envelope

ALL API responses MUST follow the envelope format:

```json
{
  "status": "success",
  "data": { ... }
}
```

Error responses:

```json
{
  "status": "error",
  "message": "Human-readable message",
  "meta": null
}
```

The envelope structure MUST NOT be changed.

### 5.3 HTTP Status Codes

- `200` — Success with data
- `201` — Resource created
- `400` — Bad request / validation failure
- `401` — Unauthenticated
- `403` — Forbidden (authenticated but not permitted)
- `404` — Resource not found
- `409` — Conflict (duplicate, state mismatch)
- `422` — Unprocessable entity (semantic validation failure)
- `423` — License locked
- `429` — Rate limited
- `500` — Internal server error

### 5.4 Pagination

List endpoints MUST support `offset` and `limit` query parameters.
Responses for paginated endpoints SHOULD include a `total` or
`has_more` field.

### 5.5 Error Messages

- Error messages MUST be user-facing and internationalizable.
- Internal error details (stack traces, SQL queries, file paths)
  MUST NOT appear in API error messages.

---

## §6 Database & Migration Rules

### 6.1 Migration Strategy

This project uses GORM `AutoMigrate` exclusively. There are NO versioned
migration files. All schema evolution MUST happen through model struct
changes processed by `AutoMigrate`.

### 6.2 Model Registration

- Every GORM model MUST be registered in `GetMigrationModels()` in
  `internal/database/postgres.go`.
- The order of model registration determines migration order.
- New models MUST be inserted in a position that respects foreign key
  dependencies (parents before children).

### 6.3 Model Base

Every model MUST embed `BaseModel`:

```go
type BaseModel struct {
    ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    CreatedAt time.Time      `gorm:"autoCreateTime"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 6.4 Table Names

Every model MUST implement `TableName() string` returning an explicit
snake_case plural table name (e.g., `func (Message) TableName() string { return "messages" }`).

### 6.5 Multi-Tenancy

- All tenant-scoped models MUST include an `OrganizationID uuid.UUID` field.
- All queries on tenant-scoped models MUST go through
  `tenant.ScopedDB(db, orgID)`.
- Direct queries bypassing tenant scoping are FORBIDDEN.

### 6.6 Schema Changes

- Adding columns is safe (nullable or with defaults).
- Removing columns MUST be verified: no running code references them.
- Renaming columns MUST be done as add + migrate old data + remove old.
- Indexes MUST be added via `getIndexes()` in the database package,
  not through GORM's `AutoMigrate` index behavior.

### 6.7 Seed Data

- Seeding (default admin user, permissions, roles, widgets) MUST go
  through `SeedPermissionsAndRoles()` and related seed functions in
  `internal/database/`.
- Seed operations MUST be idempotent.

### 6.8 Pre-Migration Fixes

Any pre-migration data fixes MUST be placed in
`applyPreMigrationFixes()` and MUST be idempotent.

---

## §7 Testing Requirements

### 7.1 Infrastructure

- Go tests REQUIRE a running PostgreSQL and Redis instance.
- Environment variables `TEST_DATABASE_URL` and `TEST_REDIS_URL`
  control test database connections.
- Tests MUST use `testutil.SetupTestDB()` for the database connection.
- Tests MUST call `cleanupTables()` (or equivalent `TRUNCATE CASCADE`)
  between test cases.

### 7.2 Go Test Patterns

- Handler tests MUST construct `fasthttp.RequestCtx` + `fastglue.Request`
  directly (NOT `httptest.Server`).
- External test packages (`package handlers_test`) are the norm.
- White-box access MUST be provided via `export_test.go` files.
- Test fixtures MUST use the functional options pattern
  (e.g., `CreateTestUser(t, db, orgID, WithEmail("x@y.com"))`).
- Assertions MUST use `github.com/stretchr/testify/require`.

### 7.3 Mocking

- Mocking MUST use hand-written mocks in `test/testutil/mocks.go`.
- `gomock`, `sqlmock`, and automated mock generators are FORBIDDEN.
- Mock implementations MUST satisfy the real interface.

### 7.4 Test Secrets

Test secrets are defined in `test/testutil/fixtures.go`:

```go
const TestJWTSecret = "unit-test-signing-value-1234567890"
const TestEncryptionKey = "0123456789abcdef0123456789abcdef"
```

These values MUST NOT be changed without updating ALL tests that
depend on them. They MUST NOT be used outside the test suite.

### 7.5 CI Testing

- CI MUST run `go test -v -race -p 1 ./...` (sequential packages).
- Tests MUST live-skip (not fail) when `TEST_DATABASE_URL` or
  `TEST_REDIS_URL` is unset.
- Parallel execution (`t.Parallel()`) within a single package is
  permitted with proper database isolation.

### 7.6 Frontend Tests

- Unit tests use Vitest (`npm run test:unit`).
- E2E tests use Playwright Chromium (`npm run test:e2e`).
- E2E global setup seeds: `admin@test.com`, `manager@test.com`,
  `agent@test.com` (password: `Password123!`).

### 7.7 Coverage

The CI pipeline SHOULD generate coverage reports but there is no
mandatory coverage threshold. New features SHOULD include meaningful
test coverage for core logic paths.

---

## §8 Security & Authentication

### 8.1 Authentication Methods

The system MUST support three authentication methods in priority order:

1. **JWT Bearer token** in `Authorization` header (HS256 signed)
2. **API Key** in `X-API-Key` header
3. **Cookie fallback** (`whm_access`) for browser sessions

### 8.2 JWT Standards

- Algorithm: HS256 only. Algorithm confusion attacks MUST be prevented
  by enforcing the expected algorithm during verification.
- Three token types: `access` (short-lived), `refresh` (longer-lived),
  and `ws` (WebSocket).
- JWT secrets MUST be configurable via config, never hardcoded in
  production code.

### 8.3 Authorization Layers

1. **Authentication** (`AuthWithDB` middleware) — validates credentials,
   sets user/org in request context.
2. **Tenant isolation** (`TenantScope` middleware) — scopes all DB
   queries to the requesting organization.
3. **Permission check** (`PermissionChecker` / `HasPermission`) —
   granular RBAC at the handler level.

### 8.4 Multi-Tenancy

- `ContextKeyUserID`, `ContextKeyOrganizationID`, `ContextKeyRoleID`
  MUST be set by the auth middleware.
- Tenant scoping MUST be applied to ALL database queries for
  org-scoped resources.
- Cross-organization data access is FORBIDDEN by design.

### 8.5 CSRF

- Double-submit cookie pattern: `whm_csrf` cookie + `X-CSRF-Token` header.
- CSRF checks MUST be skipped when Bearer token or API-Key auth is used.

### 8.6 Rate Limiting

- Login, register, refresh, SSO, webhook, outbound message, and
  campaign mutation endpoints MUST be rate-limited via Redis.
- Rate limit configuration is in the `RateLimit` config section.

### 8.7 Encryption at Rest

- Secrets (WhatsApp credentials, API keys) MUST be encrypted with
  AES-256 via `internal/crypto/`.
- The `crypto-migrate` subcommand MUST be available for key rotation.
- Encryption keys MUST NOT be logged or exposed in error messages.

### 8.8 Security Headers

- CSP with nonce for inline scripts is REQUIRED.
- HSTS is REQUIRED.
- X-Frame-Options is REQUIRED.

### 8.9 Security Audit Rules

1.  **Password Strength Policy**:
    - The password validation function (`ValidatePassword`) MUST enforce a minimum of 12 characters and require at least one uppercase letter, one lowercase letter, one digit, and at least one special character (punctuation or symbol).
2.  **Webhook Constraints**:
    - In production environments, webhook URLs MUST strictly use the `https` scheme.
    - Webhook URLs MUST NOT target internal ports (ports under 1024, except standard port 443, and known database/cache ports like 5432 or 6379).
    - Custom headers on outgoing webhooks MUST be filtered to block sensitive headers such as `Host`, `Authorization`, and `Cookie` (case-insensitive).
3.  **Secrets Encryption at Rest**:
    - All webhook secrets and SSO `client_secret` values MUST be encrypted at rest using the `internal/crypto` package.
    - API endpoints MUST NOT return plaintext secrets in GET requests (use boolean indicators like `has_secret: true` instead).
4.  **WebSocket Handshake and Auth Timeout**:
    - Post-upgrade WebSocket authentication MUST be completed within a strict 3-second timeout. If the client fails to authenticate within this window, the server MUST close the connection.
    - Rate limiting MUST be applied to WebSocket connections per IP to prevent connection exhaustion.
5.  **API Key Allocation Cap**:
    - Organizations are limited to a maximum of 10 active API keys to prevent CPU exhaustion during bcrypt password hashing verification.

### 8.10 Unified Safe Origin Evaluator

1.  **Fail-Closed Default**:
    - If the `allowed_origins` config is empty, the system MUST fallback to allowing only same-origin and localhost loopback connections. All other cross-origin requests MUST be blocked.
2.  **Origin Normalization**:
    - Configured origins MUST be normalized (scheme, host, and port normalization) before being matched.
3.  **Centralized Validation**:
    - Both CORS and WebSocket upgrade validators (`CheckOrigin`) MUST use the centralized `IsOriginAllowedForRequest` helper to enforce the exact same origin policy.

---

## §9 Forbidden Changes

The following changes are FORBIDDEN without explicit maintainer approval.

### 9.1 Database & Migration

- Changing the order in `GetMigrationModels()` without verifying
  foreign key dependencies.
- Modifying `applyPreMigrationFix()` in a non-idempotent way.
- Removing or renaming columns without a migration plan.

### 9.2 Authentication & Authorization

- Modifying `AuthWithDB()` — breaking auth breaks all endpoints.
- Modifying `TenantScope()` — bugs here cause cross-org data leaks.
- Bypassing or disabling the permission checker in production routes.

### 9.3 Crypto & Encryption

- Changing the encryption algorithm without a migration plan.
- Modifying key derivation logic — can permanently lock secrets.

### 9.4 Core Infrastructure

- Modifying `setupRoutes()` route registrations without mapping the
  full blast radius of added/removed routes.
- Changing the `BaseModel` struct — all models inherit from it.
- Modifying the frontend `//go:embed` directive without updating
  `make build-prod`.

### 9.5 License System

- Bypassing license enforcement in `internal/license/`.
- Modifying license validation logic without security review.

### 9.6 Worker & Queue

- Modifying Redis Stream consumer group logic without idempotency
  guarantees — errors here can cause message loss or duplicate delivery.

### 9.7 Test Infrastructure

- Changing `TestJWTSecret` or `TestEncryptionKey` in
  `test/testutil/fixtures.go` without updating all consuming tests.

### 9.8 Architecture

- Introducing a new architectural layer or pattern without
  constitution amendment (see §11).
- Adding a new external database or caching system without
  maintainer approval.

---

## §10 Code Style Conventions

### 10.1 Go Conventions

- File naming: `snake_case.go`
- Struct naming: PascalCase. DTOs use `Request`/`Response` suffix.
- Interface naming: single-method interfaces use `-er` suffix
  (e.g., `MessageProvider`).
- Constants: typed constants by domain, PascalCase, grouped in
  `internal/models/constants.go`.
- Receiver naming: single-letter or short abbreviation
  (e.g., `a` for `*App`, `c` for `*Contact`).
- GORM hooks: pointer receiver, `func(*Model) func(*gorm.DB) error`.
- Custom GORM types: `JSONB`, `JSONBArray`, `StringArray` for
  PostgreSQL JSONB columns.

### 10.2 Vue/TypeScript Conventions

- File naming per table in §4.8.
- Composition API with `<script setup lang="ts">` only.
- Pinia stores: setup function style (`defineStore("name", () => { })`).
- Components: directory per component with `index.ts` re-export.
- Imports: use `@/` alias for all project-relative paths.

### 10.3 General

- No TODO comments without an owner or issue reference.
- No dead code — commented-out code MUST be removed.
- Commit messages MUST be concise and describe the change.
- No hardcoded secrets, URLs, or credentials in source code.

---

## §11 Architecture Integrity

### 11.1 No New Architecture Without Approval

Existing architectural patterns MUST be reused. New architectural
layers, patterns, or abstractions MUST NOT be introduced unless:
1. The existing pattern demonstrably cannot meet the requirement, AND
2. The change is approved by a maintainer, AND
3. This constitution is amended to document the new pattern.

### 11.2 Package Boundaries

- `internal/` packages MUST NOT be imported by `pkg/` packages.
- `pkg/provider/` is the ONLY package that should be aware of both
  Meta and WhatsMeow implementations.
- Frontend code MUST NOT import from backend directories and vice versa.

### 11.3 Provider Abstraction

- New WhatsApp provider capabilities MUST be added to the
  `MessageProvider` interface before specific adapter implementations.
- Provider-specific code MUST NOT leak into handler logic.
- Provider-specific routes MUST use `ProviderGuard` wrappers.

### 11.4 Dependency Direction

Dependencies MUST flow inward:
- `cmd/` → `internal/` and `pkg/`
- `internal/handlers/` → `internal/models/`, `internal/queue/`, `pkg/provider/`
- `internal/handlers/` MUST NOT import other handler subdirectories
  cyclically.
- `internal/models/` MUST NOT import from `internal/handlers/` or
  `internal/database/`.

### 11.5 Complexity Accountability

Any plan deviating from this constitution's architectural constraints
MUST include a Complexity Tracking table (see plan-template.md) that
justifies each deviation with the rejected simpler alternative.

---

## §12 Amendment Procedure

### 12.1 Proposal

Any contributor may propose an amendment by opening a pull request
that modifies this file. The PR description MUST include a summary
of changes and the version impact (MAJOR/MINOR/PATCH).

### 12.2 Review

Amendments require:
- At least one maintainer approval.
- A completed Sync Impact Report (§2.3) showing dependent templates
  have been updated or assessed.
- For MAJOR amendments, at least two maintainer approvals.

### 12.3 Ratification

Upon approval:
1. Update `CONSTITUTION_VERSION` and `LAST_AMENDED` date.
2. Prepend the Sync Impact Report comment at the top of this file.
3. Propagate changes to all dependent templates and AGENTS.md.
4. Merge the PR.

### 12.4 Emergency Amendments

Critical security or legal issues MAY bypass the normal amendment
process with a commit directly to the constitution, provided a
retrospective review is held within 7 days.
