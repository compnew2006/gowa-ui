---
name: whatomate-agent-orchestrator
description: >
  Project-specific mandatory workflow for Whatomate. Enforces MCP-first code
  operations with Socraticode for code understanding and relationships,
  codebase-memory-mcp for project memory and patterns, and Serena for precise
  source reading, editing, creation, rename, and deletion.
---

# Whatomate Agent Orchestrator

This file is the authoritative operating guide for Pi and other coding agents in this repository.
It combines project facts, best-practice guardrails, and the mandatory MCP ecosystem workflow.

For source-code work, also load the project skill:

```text
/skill:mcp-code-operations
```

Skill path: `.pi/skills/mcp-code-operations/SKILL.md`.

---

## 1. Project Snapshot

**Whatomate** is an open-source WhatsApp business platform.

- Backend: Go single-binary service in `cmd/whatomate`.
- Frontend: Vue 3 + TypeScript app in `frontend/`, embedded into the Go binary for production.
- Domain: WhatsApp Cloud API + WhatsMeow/WhatsApp Web, multi-tenant organizations, chat, campaigns, templates, automation/chatbot flows, analytics, and RBAC.
- Data: PostgreSQL via GORM, Redis for cache/queues/streams.
- Tenancy: organization-scoped requests and database access are mandatory.

### Main Layout

| Path | Purpose | Agent rule |
|---|---|---|
| `cmd/whatomate/` | Main application entrypoint and plugin blank imports | Treat as core; changing plugin imports is allowed when adding a plugin |
| `internal/handlers/` | Core HTTP handlers and `App` dependency container | Core path; bug fixes only unless user approves |
| `internal/models/` | Core GORM models | Core path; schema changes require extra review |
| `internal/database/` | PostgreSQL/Redis setup and core migrations | Core path; do not add plugin models here |
| `internal/middleware/` | Auth, CSRF, rate limiting, tenant scoping | Shared infrastructure; approval required for behavior changes |
| `internal/tenant/` | Organization-scoped database helpers (`ScopedDB`, `GetScopedDB`) | Critical multi-tenant boundary |
| `internal/websocket/` | Real-time hub/client/message plumbing | Shared infrastructure; check blast radius; see §3 WebSocket patterns |
| `internal/worker/`, `internal/queue/` | Background jobs and Redis Streams | Check idempotency and queue contracts |
| `internal/storage/`, `internal/crypto/`, `internal/license/` | Storage, encryption, licensing | Security-sensitive; approval required for behavior changes |
| `internal/config/` | TOML/koanf config loading, validation (JWT, encryption, license, admin) | Shared infrastructure; changes affect all runmodes |
| `internal/observability/` | Metrics and observability setup | Shared infrastructure; changes affect monitoring |
| `internal/retry/` | Exponential backoff utilities | Shared utility; keep generic, no domain logic |
| `internal/contactutil/` | Contact JID and group JID helpers | Shared utility; used by handlers and whatsmeow |
| `internal/templateutil/` | Template placeholder processing | Shared utility; used by campaigns and messages |
| `internal/campaignstats/` | Campaign receipt statistics | Domain utility for campaign feature |
| `internal/licenseissuer/`, `internal/licensestudio/` | License issuing tooling | Security-sensitive; approval required for behavior changes |
| `internal/frontend/` | Frontend embed/serve via `go:embed` | Do not modify generated `dist/`; rebuild via `make build-prod` |
| `pkg/whatsapp/`, `pkg/whatsmeow/`, `pkg/provider/` | Provider clients/interfaces | Public/shared contracts; inspect callers first |
| `pkg/chat_close_ratings/` | Shared chat close rating types | Public package; used by handlers and whatsmeow |
| `plugin/<name>/` | Extension mechanism for new features | Preferred location for new features |
| `frontend/src/` | Vue 3 frontend source | Use domain folders, Pinia stores, services, composables |
| `frontend/src/services/api.ts` | Base HTTP client with auth refresh/interceptors | Domain services import and wrap this; do not bypass |
| `frontend/src/services/*.ts` | Domain service files (auth, contacts, instances, etc.) | Each service wraps `api.ts` for its domain |
| `frontend/src/composables/` | Reusable composition functions | Shared reactive logic; prefer over duplicating in views |
| `frontend/src/stores/` | Pinia stores by domain | State management; one store per domain |
| `frontend/src/views/` | Page-level Vue components | Organized by domain: analytics, chat, chatbot, settings, etc. |
| `frontend/src/router/` | Vue Router configuration | Route definitions and guards |
| `frontend/src/i18n/` | Internationalization | Locale files and i18n setup |
| `frontend/src/types/` | TypeScript type definitions | Shared type declarations |
| `test/` | Test utilities and fixtures (`testutil/`, `fixtures/`) | Shared test helpers; import via `test/testutil` |
| `docs/` | Deployment/product documentation | Keep in sync with behavior changes |
| `deploy/`, `docker/` | Infrastructure and Docker configs | Treat independently from product runtime |
| `scripts/` | Build and dev helper scripts | Treat as tooling; changes don't affect runtime |
| `specs/` | Design specifications | Useful context for understanding features |
| `mcp-server/` | Auxiliary tooling | Treat independently from product runtime |
| `summary.md` | Running session log (root level) | Append only; never replace historical entries |

> **Note:** Both `plugin/` and `plugins/` exist at root. `plugin/` is the canonical location for
> built-in plugins (registered via blank import in `cmd/whatomate/main.go`). `plugins/` contains
> external/plugin-template projects. New product features go in `plugin/`.

---

## 2. Mandatory MCP Ecosystem Policy

For source-code operations, MCPs are mandatory. Internal tools are fallbacks only after explicit user approval.

| Job | Primary MCP | Why |
|---|---|---|
| Understand code, discover behavior, map function relationships | **Socraticode** | Semantic search, symbols, call flow, dependency graph, impact analysis, cycles |
| Find existing project patterns and long-lived architecture memory | **codebase-memory-mcp** | Indexed architecture graph, route/channel relationships, pattern search, ADRs, stale-memory detection |
| Read exact source, edit, create symbols, rename, delete safely | **Serena** | LSP/symbol-aware overview, references, diagnostics, precise edits, safe deletion, memory notes |

### Forbidden for Source Code Unless User Approves Fallback

Do not use internal `read`, `edit`, or `write` for source files.
Do not use shell `cat`, `head`, `tail`, `less`, `grep`, `rg`, `ag`, `find`, or `ls` for code navigation.

Shell is allowed for:

- tests, builds, lint, typecheck, formatters
- package managers
- git operations
- creating empty directories/files only when paired with Serena for source content

### Operation Routing

| Operation | Required route |
|---|---|
| Search/discover code | Socraticode `codebase_search` / `codebase_symbols` → codebase-memory `search_graph` / `search_code` → Serena `find_symbol` |
| Read/analyze file | Serena `get_symbols_overview` → Serena `find_symbol`; Socraticode `codebase_graph_query` for dependencies |
| Analyze relationships | Socraticode `codebase_symbol`, `codebase_flow`, `codebase_impact`; codebase-memory `trace_path` for indexed graph paths |
| Edit existing code | Socraticode impact first → Serena overview/body/references → Serena edit tool |
| Create code | Prefer plugin/extension location; use Serena insertion/replacement for content |
| Remove code | Socraticode impact + Serena references → Serena `safe_delete_symbol`; ask before file deletion |
| Verify changes | Socraticode update/impact/cycles + Serena diagnostics + project tests |
| Save learning | codebase-memory re-index/detect changes + Serena `write_memory` |

---

## 3. Critical Project Invariants

### Plugin-First Architecture

New product features must be implemented as plugins under `plugin/<name>/` whenever possible.

- Register plugins using `init()` plus blank import in `cmd/whatomate/main.go`.
- Plugin interface lives in `internal/core/plugin.go`:
  ```go
  type Plugin interface {
      Name() string
      Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error
      Routes(g *fastglue.Fastglue)
      Migrate(db *gorm.DB) error
  }
  ```
  Registration uses `core.RegisterPlugin(p)`; initialization uses `core.InitPlugins(...)`.
- Plugin migrations use plugin-local `AutoMigrate` in `Migrate()`.
- Do **not** add plugin models to `internal/database/postgres.go`.
- Do **not** modify `internal/handlers/`, `internal/models/`, or `pkg/` for new features unless the user explicitly approves a core/interface change.

### Backend Conventions

- **Binary subcommands**: `whatomate server` (HTTP server), `whatomate worker` (background job processor),
  `whatomate crypto-migrate`, `whatomate queue-migrate-campaigns`, `whatomate admin-reset-password`,
  `whatomate inbound-media-reconcile`, `whatomate legacy-media-reconcile`, `whatomate version`.
  Changes to startup/worker logic must respect the subcommand architecture.
- HTTP framework is `fasthttp` + `fastglue`; do not introduce `net/http` handlers or Gin.
- Handlers return `error` and send JSON via `rc.SendEnvelope()` / `rc.SendErrorEnvelope()`.
- **Response conventions**:
  - Success: `r.SendEnvelope(data)` — wraps data in a standard envelope.
  - Error: `r.SendErrorEnvelope(statusCode, message, data, details)` — always include an HTTP status code.
  - Never write raw JSON or use `fmt.Fprintf` for responses.
- Use `app.requestDB(rc)` inside core `App` handlers; do not use `app.DB` directly for request-scoped work.
  `requestDB` resolves the tenant-scoped database via `tenant.GetScopedDB()` or `tenant.ScopedDB()`.
- For plugin tenant scoping, use `middleware.GetOrganizationID()` and `tenant.ScopedDB()`.
- **Database transactions**: use `db.Transaction(func(tx *gorm.DB) error { ... })` for multi-step writes.
  Within a transaction, always use the `tx` parameter, not the outer `db`.
  For tenant-scoped transactions, pass the scoped DB: `requestDB.Transaction(func(tx *gorm.DB) error { ... })`.
- **App struct** (`internal/handlers/app.go`) is the central dependency container:
  `Config`, `DB`, `Redis`, `Log`, `WhatsApp`, `WhatsmeowManager`, `ObjectStorage`, `WSHub`, `Queue`,
  `HTTPClient`, `MessageProvider`, `License`. Do not add fields without approval.
- GORM models should use soft deletes where appropriate (`gorm.DeletedAt`).
- Redis keys must be namespaced, e.g. `<prefix>:<orgID>:<resource>`.
- **Logging**: use `slog` or `logf.Logger` (from App). Stable keys: `org_id`, `user_id`, `instance_id`,
  `campaign_id`, `chat_id`. Use `slog.Error` for failures, `slog.Warn` for degraded states,
  `slog.Info` for normal operations. Never log secrets or credentials.
- Config is TOML via koanf; `config.toml` is secret/local and must stay ignored.
- Wrap errors with context using `%w`.
- Import order: standard library → third-party → internal packages.

### Frontend Conventions

- Vue 3 Composition API with `<script setup>` and TypeScript.
- **Directory structure**:
  - `stores/` — Pinia stores by domain (auth, contacts, instances, etc.).
  - `services/` — Domain API services (`api.ts` is the base HTTP client; each domain file wraps it).
  - `composables/` — Reusable reactive logic (`useCrudState`, `useColorMode`, `useFlowHistory`, etc.).
  - `views/` — Page-level components organized by domain (`chat/`, `chatbot/`, `settings/`, etc.).
  - `components/` — Reusable UI components organized by domain (`chat/`, `chatbot/`, `shared/`, `ui/`).
  - `router/` — Vue Router definitions and guards.
  - `types/` — Shared TypeScript type definitions.
  - `i18n/` — Internationalization locale files.
  - `lib/` — Utility libraries.
- API calls go through `frontend/src/services/api.ts` so auth refresh/interceptors remain centralized.
  Domain services import and wrap `api.ts`; do not call `fetch` or `axios` directly.
- WebSocket handling goes through `frontend/src/services/websocket.ts` and dispatches into stores.
- Forms use `vee-validate` + `zod`.
- Prefer domain components/views/composables; avoid large cross-domain components.
- Keep frontend files small and focused.

### WebSocket Patterns

- `internal/websocket/` contains `Hub`, `Client`, and message types.
- All messages are `WSMessage{Type, Payload}` structs (defined in `messages.go`).
- To add a new WebSocket message type:
  1. Add a `const TypeXxx = "xxx"` in `messages.go`.
  2. Broadcast from the relevant handler/worker via `a.WSHub.Broadcast(orgID, WSMessage{Type: TypeXxx, Payload: data})`.
  3. Handle in the frontend via `websocket.ts` → dispatch to the appropriate Pinia store.
- Always scope broadcasts to `orgID` — never broadcast globally across tenants.

### Testing Conventions

- Test files use standard Go `_test.go` suffix; tests live alongside source files.
- Shared test utilities live in `test/testutil/` (DB setup, Redis setup, mock logger).
  Import as `"github.com/compnew2006/whatomate/test/testutil"`.
- Test App construction uses `newTestApp(t, ...appOption)` pattern with functional options
  (`withQueue`, `withWhatsApp`, `withHTTPClient`, `withWSHub`). See `internal/handlers/testhelpers_test.go`.
- Tests requiring a database use `testutil.SetupTestDB(t)`; tests requiring Redis use
  `testutil.SetupTestRedis(t)` (skips if `TEST_REDIS_URL` not set).
- **Do not** introduce new `.go.disabled` or `.go.bak` test files; those are legacy.
- Use table-driven tests for multi-case scenarios.
- For handler tests, construct the App with appropriate stubs from `stubs.go`.

### Security and Tenancy

- Treat auth, middleware, tenant scoping, encryption, license verification, provider credentials, and storage as critical paths.
- Any change that could leak data across organizations requires explicit user approval and targeted tests.
- Never commit secrets, local configs, DB dumps, runtime uploads, logs, or generated credentials.

---

## 4. Session Startup Checklist

Run once at the beginning of each coding session.

1. Confirm MCP availability:
   - Serena
   - Socraticode
   - codebase-memory-mcp
   - chrome-devtools when UI verification is needed
2. Load Serena instructions and relevant memories:
   - `project_overview`
   - `tech_stack`
   - `conventions`
   - `core`
   - `suggested_commands`
3. Check Socraticode index:
   - `socraticode_codebase_status(projectPath = <repo>)`
   - If stale, run `socraticode_codebase_update()`.
4. Check codebase-memory project:
   - `codebase_memory_mcp_list_projects()`
   - `codebase_memory_mcp_detect_changes(project = current project)` when needed.
5. Load `/skill:mcp-code-operations` for any source-code work.
6. Classify the request:
   - Investigation: read-only MCP analysis.
   - Trivial change: focused read/edit/verify.
   - Standard change: full gate workflow.
   - Large/cross-module change: use subagents/swarm planning before edits.

---

## 5. Standard Change Workflow

### Phase 1 — Pattern and Memory Gate

Use codebase-memory-mcp before designing a solution:

```text
search_graph(query = task description)
search_code(pattern = important keyword)
get_architecture(project = current project)
```

If an existing pattern exists, follow it. Do not invent a second architecture.

### Phase 2 — Understanding Gate

Use Socraticode to understand the baseline:

```text
codebase_search(query = task description)
codebase_symbols(query = target)
codebase_symbol(name = target)
codebase_flow(entrypoint = target)
codebase_impact(target = symbol or file)
codebase_graph_query(filePath = target file)
```

Then use Serena for exact source context:

```text
get_symbols_overview(relative_path = target file)
find_symbol(name_path_pattern = target, include_body = true only when needed)
find_referencing_symbols(name_path = target)
```

### Phase 3 — Decision Gate

Stop and ask the user before implementation if any of these are true:

- Change touches a core/critical path listed above.
- Public API, provider interface, database model, auth, tenant scoping, queue contract, storage, encryption, or licensing behavior changes.
- Socraticode impact shows cross-module or high fan-out callers.
- Existing codebase-memory pattern conflicts with the proposed approach.
- More than five files are likely to change.
- File deletion is needed.
- MCP fallback to internal source-code tools is needed.

### Phase 4 — Implementation Gate

- Prefer plugin/extension implementation.
- Read every source file with Serena before editing.
- Edit only with Serena:
  - `replace_symbol_body`
  - `replace_content`
  - `insert_before_symbol`
  - `insert_after_symbol`
  - `rename_symbol`
  - `safe_delete_symbol`
- Keep files under 500 lines; extract helpers when needed.
- Preserve unrelated comments and behavior.

### Phase 5 — Verification Gate

Run the narrowest useful tests first, then broader checks when shared code changed.

Backend:

```sh
# All tests
make test

# Targeted package tests (use during development)
go test -p 1 -v ./internal/handlers/... ./internal/models/...

# Database-specific tests (requires ephemeral Postgres container)
make test-db

# Lint
golangci-lint run
```

Frontend:

```sh
cd frontend && npm run typecheck
cd frontend && npm run lint
cd frontend && npm run test:unit
cd frontend && npm run test:e2e
```

Build:

```sh
make build              # Backend only
make build-prod         # Backend + frontend embed
```

MCP verification after edits:

```text
socraticode_codebase_update(projectPath = <repo>)
socraticode_codebase_impact(target = changed symbol or file)
socraticode_codebase_graph_circular(projectPath = <repo>)
serena_get_diagnostics_for_file(relative_path = edited file)
codebase_memory_mcp_detect_changes(project = current project)
```

---

## 6. Completion and Memory Rules

After successful code changes:

1. Update Socraticode index if watcher has not already done it.
2. Re-index or refresh codebase-memory-mcp when architectural patterns changed.
3. Save concise notes with Serena memory:
   - task
   - files changed
   - pattern followed/created
   - tests run
   - risks/gotchas
4. Update `summary.md` (project root) only by appending; never replace historical entries.
5. Do not commit unless the user asks for a commit.
6. **Git conventions** (when user asks for a commit):
   - Write clear, descriptive commit messages (imperative mood: "Add campaign retry logic", not "Added...").
   - Do not commit `.disabled`, `.bak`, or scratch files.
   - Run `make test` before committing when shared code was changed.
   - Do not push unless the user explicitly asks.

After failed attempts:

- Stop after three serious repair attempts.
- Preserve work with git stash or a backup branch if needed.
- Report exact failures, files touched, and recommended next steps.

---

## 7. How to Use the MCP Ecosystem in Pi

### Start a coding request

Tell Pi what you want and explicitly load the skill when the task touches source code:

```text
/skill:mcp-code-operations
Fix the campaign retry bug without changing tenant boundaries.
```

### Force the desired MCP routing

Use these phrases when you want strict behavior:

```text
Use Socraticode first to map impact and function relationships.
Use codebase-memory-mcp to find existing patterns before implementation.
Use Serena only for source reads and edits.
Stop and ask me before any internal-tool fallback.
```

### Best-practice sequence for a code change

```text
1. Socraticode: search and impact.
2. codebase-memory-mcp: existing architecture/patterns.
3. Serena: inspect exact symbols.
4. Serena: edit.
5. Socraticode + Serena: verify impact, cycles, diagnostics.
6. Project tests/lint/build.
7. Serena/codebase-memory: save memory if successful.
```

### Recommended commands for this repository

```sh
make test
make build
cd frontend && npm run typecheck
cd frontend && npm run lint
```

Use targeted package/test commands during development, then broaden verification when shared code changes.

---

## 8. Tool Availability Fallbacks

- **Serena unavailable**: do not edit source code. Ask the user whether to continue with internal tools.
- **Socraticode unavailable**: do not change relationship-sensitive code. Ask before proceeding with Serena-only references.
- **codebase-memory-mcp unavailable**: note that architecture memory is unavailable; proceed only for low-risk work after Socraticode + Serena analysis.
- **chrome-devtools unavailable**: rely on frontend tests and screenshots from available tooling.

Any fallback for source search/edit/create/remove requires explicit approval in the current conversation.
