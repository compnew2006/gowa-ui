# CLAUDE.md — v3.0 — 2026-06-06

This file provides guidance to Claude Code (claude.ai/code) and OpenCode (GLM-4.5 via Z.ai)
when working with code in this repository. Both tools load this file automatically at session start.

***

## RUNTIME DETECTION

```
If running in OpenCode  → prepend ALL tool calls with server prefix:
                          codegraph_*, serena_*, codebase_memory_*, chrome_devtools_*, zai_*, openspace_*
If running in Claude Code → use bare tool names: semantic_search, fn_impact, replace_symbol_body ...
```

***

## Project Overview

Whatomate is a WhatsApp Business Platform with dual-provider support: **Meta Cloud API** and
**Whatsmeow** (WhatsApp Web protocol). Go backend (FastHTTP + fastglue router), Vue 3 +
TypeScript frontend, PostgreSQL, Redis. Single binary production build with embedded frontend.

- **Go module**: `github.com/compnew2006/whatomate`
- **HTTP framework**: fastglue + fasthttp — NOT net/http
- **Config format**: TOML via koanf (`config.toml`, gitignored — copy from `config.example.toml`)
- **Go version**: 1.25.x

***

## Build & Run Commands

```bash
make build && ./whatomate server -config config.toml -migrate   # backend dev
make frontend-dev                                                # frontend dev
make build-prod && ./whatomate server -config config.toml -migrate  # production
make backend-watch                                               # hot-reload backend
make dev-watch                                                   # backend + frontend
make test                                                        # all Go tests
go test -v -run TestFunctionName ./internal/handlers/...        # single test
make test-db                                                     # DB integration (ephemeral Postgres)
cd frontend && npm run test:unit                                 # frontend unit
cd frontend && npm run test:e2e                                  # frontend E2E
cd frontend && npm run lint && npm run typecheck                 # lint + types
make lint                                                        # Go lint
```

### Environment Variables (required for tests)

```bash
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"
```

Tests **skip** (not fail) if these are unset.
Always use `-p 1` when running multiple packages sharing the test DB.

### First-Time Setup

```bash
cp config.example.toml config.toml      # never commit config.toml
docker compose -f docker/docker-compose.yml up -d db redis
make run-migrate                         # first run — applies all migrations
```

***

## Architecture

### Backend (`internal/`)

- **`handlers/`** — HTTP handlers. `app.go` = `App` dependency container. 230+ files.
- **`models/`** — GORM models. `models.go` is central. Soft deletes everywhere.
- **`database/`** — PostgreSQL (GORM) + Redis. Migrations in `postgres.go`.
- **`websocket/`** — `hub.go` broadcasts, `client.go` per connection, `messages.go` event types.
- **`middleware/`** — Auth JWT, CSRF, rate limiting, tenant scoping.
- **`worker/`** — Background jobs: campaigns, scheduled sends, group ops.
- **`queue/`** — Redis Streams consumer groups (`campaigns`, `inbound_media`).
- **`tenant/`** — Multi-tenant scope filtering per request.
- **`storage/`** — Local FS or S3/MinIO abstraction.
- **`crypto/`** — AES-256 encryption for secrets at rest.
- **`license/`** — Host-bound license enforcement, RSA verification.
- **`core/`** — Plugin registration registry and core interfaces. Contains `plugin.go` (registry for third-party feature extensions).

### Dual Provider (`pkg/`)

- **`pkg/whatsapp/`** — Meta Cloud API client.
- **`pkg/whatsmeow/`** — QR-code protocol. QueueManager, event dispatcher, media retry.
- **`pkg/provider/`** — `MessageProvider` interface. Provider set per-instance via `whatsapp.provider`.

> **Multi-instance providers**: each instance can have its own provider — `meta` or `whatsmeow`
> simultaneously. New providers must implement `pkg/provider/interface.go` fully.

### Frontend (`frontend/src/`)

- **`stores/`** — Pinia: auth, instances, contacts, transfers
- **`services/`** — api.ts (axios), websocket.ts, domain services
- **`views/`** — chat/, settings/, chatbot/, facebook/, analytics/, dashboard/, auth/
- **`components/`** — chat/, chatbot/, layout/, shared/
- **`composables/`** — shared Vue logic
- **`i18n/`** — en, ar, es locale files
- **`router/`** — Vue Router with guards

### Key Flows

1. **Send**: Frontend → handler → `MessageProvider.Send()` → Meta/Whatsmeow → WS status broadcast
2. **Webhook**: Meta/Whatsmeow → handler → DB → WS push
3. **Auth**: JWT httpOnly cookies. Access 15min + refresh 1 day. CSRF double-submit.
4. **Tenancy**: `TenantScope` middleware filters all DB queries by org (see Multi-Tenancy below).

***

## 🔴 ARCHITECTURAL INVARIANT — Plugin Architecture

**This is the most critical rule. Violating it causes irreversible damage to core.**

```
New feature = new plugin under plugin/
Core modification = FORBIDDEN without explicit approval
```

### Plugin Directory Structure

```
plugin/
  <feature-name>/
    plugin.go       ← registers via init() + blank import
    handler.go      ← feature handlers (optional)
    model.go        ← feature models (optional)
    routes.go       ← route registration (optional)
```

### Registration Pattern

Every plugin self-registers using Go's blank import mechanism:

```go
// plugin/<feature>/plugin.go
package <feature>

import "github.com/compnew2006/whatomate/internal/core"

func init() {
    core.RegisterPlugin(<feature>Plugin{})
}

// cmd/whatomate/main.go — activate plugin with blank import:
import _ "github.com/compnew2006/whatomate/plugin/<feature>"
```

### Plugin Interface Contract

Every plugin struct must implement the `core.Plugin` interface defined in `internal/core/plugin.go`:

```go
package core

import (
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

type Plugin interface {
	Name() string
	Init(db *gorm.DB, rdb *redis.Client, log *slog.Logger) error
	Routes(g *fastglue.Glue)
	Migrate(db *gorm.DB) error
}
```

- `Name() string`: Returns the unique name of the plugin.
- `Init(...) error`: Receives dependency injections (GORM, Redis, Logger) and binds them to the plugin struct. The core registry calls this on all plugins during startup.
- `Routes(g *fastglue.Glue)`: Registers the plugin's routes on the main router.
- `Migrate(db *gorm.DB) error`: Runs GORM auto-migrations for the plugin's models (e.g. `db.AutoMigrate(&MyPluginModel{})`).

### Plugin Migration Strategy

To avoid breaking the Plugin Invariant and modifying core files (like `postgres.go`):
- All plugin database migrations MUST be defined inside the plugin's `Migrate(db *gorm.DB)` method using GORM `AutoMigrate`.
- The core migration runner automatically iterates through all registered plugins and invokes their `Migrate` methods during startup. Do NOT add plugin models to `internal/database/postgres.go`.

### Plugin Tenant Scoping

Because plugins do not have access to `handlers.App` or its `requestDB` helper method, plugin handlers must resolve the organization ID and scope GORM queries manually using the exported `tenant` and `middleware` packages:

```go
package <feature>

import (
	"net/http"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/zerodha/fastglue"
)

func (p *MyPlugin) handleSomething(rc *fastglue.Request) error {
	// 1. Resolve tenant organization ID from request context (set by auth middleware)
	orgID, ok := middleware.GetOrganizationID(rc)
	if !ok {
		return rc.SendErrorEnvelope(http.StatusUnauthorized, "missing organization", nil, "UNAUTHORIZED")
	}

	// 2. Obtain a scoped database session
	scopedDB := tenant.ScopedDB(p.db, orgID)

	// 3. Query using the scoped database
	var results []MyPluginModel
	if err := scopedDB.Find(&results).Error; err != nil {
		return rc.SendErrorEnvelope(http.StatusInternalServerError, "query failed", nil, "DB_ERROR")
	}

	return rc.SendEnvelope(results)
}
```

### Plugin Test Pattern

Plugin handlers should be tested by constructing the plugin instance directly and passing mocked or test database dependencies:

```go
package <feature>

import (
	"log/slog"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMyPluginHandler(t *testing.T) {
	// Construct plugin directly with test/mock DB dependencies
	p := &MyPlugin{
		db:    testDB,
		redis: testRedis,
		log:   slog.Default(),
	}

	// Construct fastglue.Request and call handler directly
	// ...
	assert.NoError(t, err)
}
```

### Decision Rule for the Agent

```
Before ANY implementation:
  Does this touch internal/handlers/, internal/models/, or pkg/?
    YES → Is this a bug fix or core interface change?
      NO  → Create plugin/ instead. Do NOT modify core.
      YES → Get explicit user approval first.
    NO  → Proceed normally.
```

***

## Multi-Tenancy — Full Pattern

### How scoping works

Every DB query in a handler **must** go through the tenant scoped database. Within core handlers (defined as methods on `App`), the standard way to retrieve a request-scoped, pre-scoped database clone is via the `app.requestDB(rc)` helper method:

```go
// internal/handlers/app.go
// requestDB returns a tenant-scoped *gorm.DB session for the current request context.
// It checks the request context for a pre-scoped DB (set by auth middleware) or scopes it using tenant.ScopedDB.
func (a *App) requestDB(r *fastglue.Request) *gorm.DB { ... }
```

```go
// ✅ CORRECT — always scope DB queries inside App handlers using requestDB
func (app *App) handleListContacts(rc *fastglue.Request) error {
    var contacts []models.Contact
    app.requestDB(rc).Find(&contacts)
    ...
}

// ❌ WRONG — never query without scope in a handler
app.db.Find(&contacts)
```

### What needs scoping (must filter by org_id)

- All records in `models/` that have an `OrgID` or `OrganizationID` field
- Any query inside a handler that returns user-visible data

### What does NOT need scoping

- Internal background workers operating on a specific known org
- System-wide tables (e.g., `plans`, `feature_flags`) with no org column
- License checks

### Redis key namespacing per tenant

```go
// Pattern: "<prefix>:<orgID>:<resource>"
key := fmt.Sprintf("session:%s:%s", orgID, userID)
key := fmt.Sprintf("ratelimit:%s:%s", orgID, endpoint)
```

Never use global Redis keys for tenant-specific data.

***

## GORM Soft Delete Pattern

All database models utilize GORM soft deletes (`gorm.DeletedAt`).

### Querying Soft-Deleted Records
Use `db.Unscoped()` to include deleted records in queries:
```go
// Include soft-deleted rows
var contact models.Contact
app.db.Unscoped().First(&contact, "id = ?", id)
```

### Soft Delete vs Hard Delete
- `db.Delete(&record)` performs a soft delete (sets `deleted_at`).
- `db.Unscoped().Delete(&record)` performs a hard delete (removes row from DB).
Always use soft delete unless explicitly instructed otherwise.

### Cascading & Tenant Scoping
- GORM soft delete does NOT automatically cascade to related tables.
- When deleting a parent record (e.g., `Instance`), ensure related tenant-scoped child records (e.g., `Campaigns`, `Messages`) are explicitly soft-deleted to avoid orphaned records:
  ```go
  // Explicit cascade soft-delete within same transaction
  tx.Where("instance_id = ?", instance.ID).Delete(&models.Campaign{})
  ```

***

## Error Handling Convention

### Error types

```go
// Sentinel errors — for expected domain errors
var ErrNotFound   = errors.New("not found")
var ErrForbidden  = errors.New("forbidden")
var ErrConflict   = errors.New("conflict")

// Wrapped errors — always wrap with context
return fmt.Errorf("handleCreateCampaign: %w", ErrNotFound)
```

### HTTP error response format (JSON)

All handlers return errors via fastglue's envelope:

```go
// ✅ CORRECT — use envelope helper
return rc.SendErrorEnvelope(http.StatusBadRequest, "validation failed", nil, "VALIDATION_ERROR")

// JSON output:
// { "status": "error", "message": "validation failed", "code": "VALIDATION_ERROR", "data": null }
```

Never return raw error strings or non-envelope JSON from handlers.

### Handler error propagation pattern

```go
func (app *App) handleSomething(rc *fastglue.Request) error {
    result, err := app.service.DoWork(ctx)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return rc.SendErrorEnvelope(http.StatusNotFound, "resource not found", nil, "NOT_FOUND")
        }
        app.log.Error("handleSomething: unexpected error", "err", err)
        return rc.SendErrorEnvelope(http.StatusInternalServerError, "internal error", nil, "INTERNAL_ERROR")
    }
    return rc.SendEnvelope(result)
}
```

***

## WebSocket Message Protocol

### Message type field

Every WS message has a `type` field that drives frontend routing:

```go
// internal/websocket/messages.go — message types
type MessageType string

const (
    MsgNewMessage       MessageType = "new_message"
    MsgDeliveryUpdate   MessageType = "delivery_update"
    MsgTypingIndicator  MessageType = "typing"
    MsgAgentTransfer    MessageType = "agent_transfer"
    MsgPresenceUpdate   MessageType = "presence"
)
```

### JSON schema for WS events

```json
{
  "type": "new_message",
  "org_id": "uuid",
  "payload": { /* event-specific data */ },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### Frontend subscription pattern

```typescript
// frontend/src/services/websocket.ts
ws.on('new_message', (payload) => store.addMessage(payload))
ws.on('delivery_update', (payload) => store.updateStatus(payload))
```

New WS event types: add to `messages.go` + broadcast in `hub.go` + subscribe in `websocket.ts`.

***

## Background Workers — Pattern

### Adding a new worker

```go
// internal/worker/<feature>_worker.go
type FeatureWorker struct { db *gorm.DB; redis *redis.Client }

func (w *FeatureWorker) Run(ctx context.Context) error {
    // process from Redis Stream consumer group
}

// internal/worker/register.go — register worker
func RegisterWorkers(app *App) {
    app.workers = append(app.workers, &FeatureWorker{db: app.db, redis: app.redis})
}
```

### Redis Streams consumer group naming

```go
// Pattern: "stream:<domain>" + group: "worker:<feature>"
streamKey  := "stream:campaigns"
groupName  := "worker:campaign-sender"
```

### Retry/backoff strategy

Workers use exponential backoff with jitter. Max retries configurable per worker.
Failed jobs after max retries → dead-letter stream `stream:<domain>:dead`.

***

## Logging Convention

```go
// Framework: slog (structured logging)
app.log.Info("handler called", "org_id", orgID, "user_id", userID)
app.log.Error("operation failed", "err", err, "context", ctx)
app.log.Debug("debug info", "key", value)  // only in dev
```

Log levels: `Debug` (dev only) · `Info` (normal ops) · `Warn` (recoverable) · `Error` (needs attention).
Always include `org_id` and `user_id` in handler logs for traceability.

***

## API Pagination Pattern

```go
// Standard cursor pagination for all list endpoints
type PaginationParams struct {
    Page    int    `query:"page" default:"1"`
    PerPage int    `query:"per_page" default:"20" max:"100"`
}

// Response envelope
type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    Total      int64       `json:"total"`
    Page       int         `json:"page"`
    PerPage    int         `json:"per_page"`
    TotalPages int         `json:"total_pages"`
}
```

Use offset pagination for simple lists. Use cursor pagination (keyset) only for high-volume streams.

***

## Conventions

- Composition API only. Pinia, Vue Query, VeeValidate + Zod.
- Go: `testify`. Frontend: Vitest (unit) + Playwright (E2E).
- Handler tests: construct `fasthttp.RequestCtx` + `fastglue.Request` directly — no httptest.
- Files under 500 lines. Extract when exceeding.
- **Always read a file before editing it.**
- New code = plugin first (see Plugin Architecture above).

***

## HARD RULE — MCP-First, Shell-Second

**Forbidden for code navigation:**
```
cat  head  tail  less  grep  rg  ag  find  ls  ← on source files
```

**Shell allowed ONLY for:** tests, git, package installs, linters, running the app, `php -l`.

***

## SESSION STARTUP — Execute Every Session (in order)

```
1. serena          → initial_instructions()
2. serena          → onboarding()              ← skip if done in last 24h
3. serena          → list_memories()
4. serena          → read_memory("summary")    ← if exists — NEVER replace, only append
5. Check if skills-map.md exists (e.g., search via `serena → search_for_pattern(substring_pattern = "skills-map\\.md", relative_path = "")` or check file)
   → if found: read the file
   → if not found: generate it now using the auto-generation logic below, then read
6. Check if summary.md exists and contains "<!-- END -->" (e.g., search `summary\\.md`). If not found or missing: create/touch it with `# Summary\n\n<!-- END -->` to ensure subsequent appends succeed.
7. codegraph       → semantic_search("recent changes; last task; last modified")
```

***

## SKILLS MAP SYSTEM

`skills-map.md` is a machine-readable index of installed skills.
Read in Phase 1 to select the right skill per task.

### Auto-generate if missing (AI does this — no external script needed)

```
if skills-map.md not found:
  serena → search_for_pattern(substring_pattern = "\\.md$", relative_path = ".claude/skills")
  serena → search_for_pattern(substring_pattern = "\\.md$", relative_path = ".agents/skills")
  for each skill file found:
    read first 20 lines → extract description + keywords
  bash: touch skills-map.md
  serena → replace_content(
    file    = "skills-map.md",
    content = generated routing table (format below)
  )
```

### skills-map.md format

```markdown
# Skills Map — Auto-generated <DATE>
<!-- Regenerate: delete this file and restart session -->

| Skill ID | File | Description | Best For |
|----------|------|-------------|----------|
| `skill-name` | `.claude/skills/skill-name.md` | ... | ... |

## Routing Rules
| Task signals | Load skill |
|---|---|
| style, design, CSS, component, page, view | ui-ux-pro |
| test, spec, playwright, e2e, unit | test-intelligence |
| refactor, extract, rename, cleanup | code-intelligence |
| build, add, implement, feature | code-intelligence + domain skill |
| fix, bug, error, broken | code-intelligence |
| model, migration, query, gorm | domain DB skill |
| handler, endpoint, route, API | domain API skill |
```

### Activate selected skill

```
openspace → search_skills(query = task domain keyword)
openspace → execute_task(task = matched skill name)
```

***

## LOCAL ORCHESTRATOR PROTOCOL

**Master workflow for EVERY request. All 5 phases mandatory. No skipping.**
*(Tool names below are bare — prefix with server name if running in OpenCode)*

***

### PHASE 1 — MEMORY + SKILL SELECTION

```
codebase-memory-mcp → search_graph(query = task description)
codebase-memory-mcp → search_code(pattern = keyword)
serena              → read_memory("summary")      ← read only, never replace
```

Read `skills-map.md` → select + activate matching skill.
Empty memory = normal. Log "no prior patterns" and continue.

***

### PHASE 2 — ANALYSIS

> [!NOTE]
> In Phase 2 Analysis, use `codegraph` tools on the current `HEAD` (no `--staged` flag) to analyze the baseline codebase.

```
codegraph → semantic_search(query = task)
serena    → find_symbol(name = target)
codegraph → where(name = target)                           ← definition + all usages
codegraph → fn_impact(name = target, flags = -T)
codegraph → impact_analysis(target = module/file)
codegraph → context(name = target, flags = -T)
codegraph → co_changes(file = target)
serena    → get_symbols_overview(path = target file)
```

*(Full tool reference → see Appendix)*

***

### PHASE 3 — DECISION GATE

Present before any edit:

| Symbol | File | Direct Callers | Cross-Module? | Risk |
|--------|------|----------------|---------------|------|

**Hard stop — wait for explicit user approval if:**
- Any symbol > 5 callers AND at least 1 cross-module
- `check` returns cycle warning
- `co_changes` returns > 3 coupled files not in scope
- Edit targets `internal/handlers/`, `internal/models/`, or `pkg/` (Plugin Rule — see above)

**Auto-proceed (no approval):**
- All callers in same module/package
- New files under `plugin/` with zero existing callers
- Pure UI/CSS changes with no Go symbol impact

***

### PHASE 4 — IMPLEMENTATION

```bash
git checkout -b agent/<kebab-case-task-name>
```

| Operation | Serena tool |
|---|---|
| Modify existing function | `replace_symbol_body` |
| Targeted text/regex replace | `replace_content` |
| Add code after symbol | `insert_after_symbol` |
| Add code before symbol | `insert_before_symbol` |
| Rename across project | `rename_symbol` |
| Delete with safety check | `safe_delete_symbol` |
| Find usages before edit | `find_referencing_symbols` |
| New file | `bash: touch <path>` then `replace_content` to write content |

> **New files cannot use `replace_content` alone** — the file must exist first.
> Create via `bash: touch <path>`, then populate via `replace_content`.

After each edit, stage the changes first so that the staged analysis sees them:
```bash
git add <edited-file>
```
Then run:
```
codegraph → diff_impact(flags = --staged -T)
```

Unexpected new dependents → pause, report to user, do not commit.

```bash
git add -p && git commit -m "feat(<scope>): <description>"
```

***

### PHASE 5 — VERIFICATION

**UI / frontend changes:**
*Playwright Usage Rule: Use npx CLI for running test suites. Use plugin:ecc:playwright tools for inspecting/debugging individual test failures.*

```bash
npx playwright test --project=chromium
npx playwright test          # full suite if shared components changed
npx playwright test --debug  # if failures
```

**Backend / Go changes:**
```bash
make test && make lint && cd frontend && npm run typecheck
```
```
codegraph → check(flags = --staged --no-new-cycles)
codegraph → diff_impact(flags = --staged -T)
```

**After success — save pattern + append to summary.md:**
```
codebase-memory-mcp → index_repository()
serena → write_memory(name = "feature/<name>-<date>", content = "approach + gotchas")
```

**Append to summary.md (never replace):**
```
if summary.md exists:
  serena → replace_content(
    file    = "summary.md",
    pattern = "<!-- END -->",
    replace = "\n## <task-name> — <date>\n<summary>\n<!-- END -->"
  )
if not exists:
  bash: touch summary.md
  serena → replace_content(file = "summary.md", content = "# Summary\n\n<!-- END -->")
  then append as above
```

summary.md must include: task name + date, files changed, approach, blast radius table, tests, gotchas.

**Verification failure protocol (after 3 loops):**
```bash
# Save work safely
git add -p
git stash push -m "agent/<task>/failed-attempt-<N>"

# If stash fails due to conflicts (SWARM mode):
git branch agent/<task>/backup-<timestamp>  # preserve on branch instead
git reset --hard HEAD                        # clean to last good commit
```
Append failure reason + stash ref to `summary.md`. Surface to user: exact error + files + what was tried.

***

## SWARM PROTOCOL (> 5 files or simultaneous frontend + backend)

> [!IMPORTANT]
> - LEAD: receives full CLAUDE.md (including Appendix)
> - SCOUT/IMPLEMENT/QA: stop reading at "## Appendix"

| Role | Count | Phases | Write access | Prompt scope |
|---|---|---|---|---|
| LEAD | 1 | All | All files, git, summary.md | Full CLAUDE.md |
| SCOUT | 2 | 1–3 only | Read-only | Overview + Phases 1–3 |
| IMPLEMENT | N | 4 only | Declared files only | Overview + Phase 4 + Appendix |
| QA | 1 | 5 only | Test files only | Phase 5 only |
| DOC | 1 | Post-5 | summary.md + memory | Phase 5 + summary rules |

**Token Budget & Context Window Management:**
- LEAD: receives full CLAUDE.md (if context usage > 80k tokens: drop the Appendix and load tool references on demand).

**Ownership (IMPLEMENT agents declare before Phase 4):**
```
serena → write_memory(name = "agent-<N>-owns", content = ["file1.go", "file2.vue"])
```
LEAD checks all ownership for overlap before Phase 4. Overlap → reassign, never concurrent edits.

***

## Architecture Enforcement (Automatic)

After every session with code changes:
```
codegraph → audit(target = modified files, -T)
codegraph → triage(-T)
codegraph → node_roles(role = dead, -T)
codegraph → complexity(-T)
```
Cycle / dead exports / complexity > 20 → report as warning **before** proceeding with original task.

***

## Common Pitfalls

- **Do NOT use `net/http` patterns** — fasthttp/fastglue everywhere
- **`config.toml` gitignored** — never commit; use `config.example.toml`
- **Production binary needs frontend build first** — `go build` alone excludes SPA
- **Tests skip (not fail)** if `TEST_DATABASE_URL` or `TEST_REDIS_URL` unset
- **`-p 1` required** for multi-package test runs sharing the test DB
- **Frontend dev proxy**: backend on 8080, Vite on 3000
- **New feature = plugin**, not core modification (see Plugin Architecture above)

***

***

# Appendix: Tool Reference

*Used by LEAD and IMPLEMENT agents. Agents receiving trimmed prompt stop above this line.*

***

## Codegraph — 34 Tools

| Tool | Purpose |
|---|---|
| `ast_query` | Search stored AST nodes (calls, literals, await, throw) |
| `audit` | Composite: structure + blast radius + complexity |
| `batch_query` | Query multiple targets in one call |
| `branch_compare` | Structural diff between two git refs |
| `brief` | Token-efficient file summary with risk tier |
| `cfg` | Intraprocedural control flow graph |
| `check` | CI gate: manifesto rules + diff predicates (exit 0/1) |
| `co_changes` | Files that historically change together |
| `code_owners` | CODEOWNERS mapping for files and functions |
| `communities` | Module boundary detection via Louvain clustering |
| `complexity` | Per-function: cognitive, cyclomatic, Halstead, MI |
| `context` | Full: source + deps + callers + tests |
| `dataflow` | Data flow edges + data-dependent blast radius |
| `diff_impact` | Changed functions + transitive callers from git diff |
| `execution_flow` | Trace execution from entry point to leaves |
| `export_graph` | Export as DOT / Mermaid / JSON / GraphML / Neo4j |
| `file_deps` | What a file imports and what imports it |
| `file_exports` | Exported symbols with per-symbol consumers |
| `find_cycles` | Detect circular dependencies |
| `fn_impact` | Function-level blast radius (transitive callers) |
| `impact_analysis` | Files affected by changes to a given file |
| `implementations` | Concrete types implementing an interface |
| `interfaces` | Interfaces a struct implements |
| `list_functions` | List functions/methods/classes with filters |
| `module_map` | Most-connected files overview |
| `node_roles` | Classify: entry/core/utility/adapter/dead/leaf |
| `path` | Shortest call path between two symbols |
| `query` | Call graph: callers + callees with transitive chain |
| `semantic_search` | Embeddings + BM25 hybrid search by intent |
| `sequence` | Mermaid sequence diagram from call graph |
| `structure` | Directory tree with cohesion scores |
| `symbol_children` | Sub-declarations of a symbol |
| `triage` | Ranked audit queue by composite risk score |
| `where` | Where a symbol is defined and used |

**Flags:** `-T` (no tests) · `--json` · `-f <file>` · `-k <kind>`

**Semantic search tips (multi-angle RRF):**
```
codegraph semantic_search "validate auth; check token; verify JWT"
codegraph semantic_search "send email; notify user; deliver message"
```

***

## Codebase-Memory-MCP — 14 Tools

| Tool | Purpose |
|---|---|
| `delete_project` | Delete project from knowledge graph |
| `detect_changes` | Detect code changes and impact |
| `get_architecture` | High-level architecture overview |
| `get_code_snippet` | Read source for a function/class/symbol |
| `get_graph_schema` | Schema of the knowledge graph |
| `index_repository` | Index repo into knowledge graph |
| `index_status` | Indexing status of a project |
| `ingest_traces` | Ingest runtime traces to enhance graph |
| `list_projects` | List all indexed projects |
| `manage_adr` | Create/update Architecture Decision Records |
| `query_graph` | Execute Cypher query against graph |
| `search_code` | Graph-augmented code search |
| `search_graph` | Search functions, classes, routes, variables |
| `trace_path` | Trace paths through the code graph |

***

## Serena — 21 Tools

| Tool | Purpose |
|---|---|
| `delete_memory` | Delete a memory entry |
| `edit_memory` | Regex replace within a memory |
| `find_declaration` | Find declaration of a symbol |
| `find_implementations` | All implementations of a symbol |
| `find_referencing_symbols` | All references/callers |
| `find_symbol` | Locate symbol by name path pattern |
| `get_diagnostics_for_file` | LSP diagnostics by severity |
| `get_symbols_overview` | Symbol outline of a file |
| `initial_instructions` | Load Serena operating rules |
| `insert_after_symbol` | Insert code after a symbol |
| `insert_before_symbol` | Insert code before a symbol |
| `list_memories` | List all memory entries |
| `onboarding` | Run project onboarding |
| `read_memory` | Read memory entry by name |
| `rename_memory` | Rename/move a memory entry |
| `rename_symbol` | Safe project-wide rename |
| `replace_content` | Regex-based file content replacement (existing files only) |
| `replace_symbol_body` | Replace entire function/class body |
| `safe_delete_symbol` | Delete only if no references exist |
| `search_for_pattern` | Searches for a regex pattern across project files, returning whole matched lines |
| `write_memory` | Persist architectural note or pattern |


***

## Openspace — 4 Tools

| Tool | Purpose |
|---|---|
| `execute_task` | Execute a named task from openspace registry |
| `fix_skill` | Fix/repair an installed skill |
| `search_skills` | Search available skills by keyword |
| `upload_skill` | Upload/install a new skill |

***

## ECC Plugins — Built-in Tools

### plugin:ecc:playwright — 23 Tools

Allows browser automation and E2E testing.
Use when: running Playwright E2E tests, verifying UI layouts, checking console logs, or debugging tests.

### plugin:ecc:github — 25 Tools

Allows GitHub repository integration.
Use when: managing GitHub issues, pull requests, commits, and workflow runs.

### plugin:ecc:context7 — 2 Tools

| Tool | Purpose |
|---|---|
| `resolve-library-docs` | Fetch up-to-date docs for any library by name |
| `get-library-docs` | Get specific section of library documentation |

Use when: implementing unfamiliar library APIs, checking latest SDK method signatures.

### plugin:ecc:memory — 7 Tools

| Tool | Purpose |
|---|---|
| `create_entities` | Create named entities in memory graph |
| `create_relations` | Create relations between entities |
| `add_observations` | Add facts to existing entities |
| `delete_entities` | Delete entities from graph |
| `delete_observations` | Delete specific observations |
| `delete_relations` | Delete relations between entities |
| `search_nodes` | Search entities by keyword |

Use for: cross-session entity tracking (users, orgs, recurring patterns).

### plugin:ecc:sequential-thinking — 1 Tool

| Tool | Purpose |
|---|---|
| `sequentialthinking` | Multi-step reasoning chain with backtracking |

Use when: complex architectural decisions, multi-file refactors, ambiguous task decomposition.

## Active Technologies
- Go 1.25.8 (project CI version; module `github.com/compnew2006/whatomate`) + TypeScript / Vue 3 / Vite (frontend) + Backend — `fasthttp` + `fastglue` (NOT `net/http`), GORM + PostgreSQL 17, Redis 7, `gorm.io/gorm`, `github.com/google/uuid`, `github.com/zerodha/fastglue`, `github.com/valyala/fasthttp`. Frontend — Vue 3 Composition API, `@tanstack/vue-query`, `vue-i18n` (en/es/ar), `vue-sonner`, shadcn-vue + Tailwind CSS v3. (001-per-instance-uploads-cleanup)
- PostgreSQL 17 via GORM `AutoMigrate`. New persistent data: (a) per-instance `WhatsAppInstance.settings` JSONB sub-keys `uploads_cleanup.retention_days` (int) and `uploads_cleanup.last_run_date` (string) — no schema change; (b) one new GORM model `InstanceUploadsCleanupAudit` registered in plugin `Migrate`. Filesystem storage path is unchanged (`<LocalPath>/orgs/<orgID>/...`); instance scope is resolved via `Message.instance_id` and `MediaAsset` linkage, falling back to workspace default for unscoped files. (001-per-instance-uploads-cleanup)

## Recent Changes
- 001-per-instance-uploads-cleanup: Added Go 1.25.8 (project CI version; module `github.com/compnew2006/whatomate`) + TypeScript / Vue 3 / Vite (frontend) + Backend — `fasthttp` + `fastglue` (NOT `net/http`), GORM + PostgreSQL 17, Redis 7, `gorm.io/gorm`, `github.com/google/uuid`, `github.com/zerodha/fastglue`, `github.com/valyala/fasthttp`. Frontend — Vue 3 Composition API, `@tanstack/vue-query`, `vue-i18n` (en/es/ar), `vue-sonner`, shadcn-vue + Tailwind CSS v3.

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->
