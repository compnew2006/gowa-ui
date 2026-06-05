# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Whatomate is a WhatsApp Business Platform with dual-provider support: **Meta Cloud API** and **Whatsmeow** (WhatsApp Web protocol). Go backend (FastHTTP + fastglue router), Vue 3 + TypeScript frontend, PostgreSQL, Redis. Single binary production build with embedded frontend.

## Build & Run Commands

```bash
# Backend only (dev)
make build && ./whatomate server -config config.toml -migrate

# Frontend only (dev)
make frontend-dev

# Production build (single binary with embedded frontend)
make build-prod && ./whatomate server -config config.toml -migrate

# Backend with hot-reload (air)
make backend-watch

# Both backend (hot-reload) + frontend dev server
make dev-watch

# Run all Go tests
make test

# Run a specific Go test
go test -v -run TestFunctionName ./internal/handlers/...

# Run database integration tests (starts ephemeral Postgres container)
make test-db

# Frontend unit tests
cd frontend && npm run test:unit

# Frontend E2E tests (requires running backend)
cd frontend && npm run test:e2e

# Frontend lint
cd frontend && npm run lint

# Frontend type check
cd frontend && npm run typecheck

# Go lint
make lint
```

## Architecture

### Backend (Go — `internal/`)

Entry point: `cmd/whatomate/main.go`

- **`internal/handlers/`** — All HTTP handlers. `app.go` defines `App` struct (dependency container injected into all handlers). Routes are registered in `app.go` via `fastglue`. 230+ files, one domain concept per file (e.g., `auth_handlers.go`, `campaign_policy.go`, `agent_transfers.go`).
- **`internal/models/`** — GORM database models and domain types. `models.go` is the central file.
- **`internal/database/`** — PostgreSQL (GORM) and Redis connection setup. Migrations in `postgres.go`.
- **`internal/websocket/`** — WebSocket hub for real-time chat. `hub.go` broadcasts, `client.go` per connection, `messages.go` defines message types.
- **`internal/middleware/`** — Auth JWT, CSRF protection, rate limiting, tenant scoping.
- **`internal/worker/`** — Background job processors: campaigns, scheduled sends, group operations, WhatsApp filter, send policy enforcement.
- **`internal/queue/`** — Job queue abstraction over Redis (pubsub + ordered queues).
- **`internal/tenant/`** — Multi-tenant scope filtering applied per request.
- **`internal/storage/`** — Object storage abstraction (local filesystem or S3/MinIO).
- **`internal/crypto/`** — AES-256 encryption for secrets at rest (API keys, tokens).
- **`internal/license/`** — Host-bound license enforcement with RSA key verification.

### Dual Provider System (`pkg/`)

- **`pkg/whatsapp/`** — Meta Cloud API client (official WhatsApp Business API).
- **`pkg/whatsmeow/`** — Whatsmeow protocol client (WhatsApp Web via QR code). Includes per-instance message queue (`QueueManager`), connection manager, event dispatcher, and media retry.
- **`pkg/provider/`** — `MessageProvider` interface abstracting both providers. Handlers call this interface; provider selection is per-instance via config (`whatsapp.provider = "meta" | "whatsmeow"`).

### Frontend (Vue 3 + TypeScript — `frontend/`)

- **`frontend/src/stores/`** — Pinia stores for state management (`auth.ts`, `instances.ts`, `contacts.ts`, `transfers.ts`, etc.).
- **`frontend/src/services/`** — API layer (`api.ts` = axios instance, `websocket.ts` = WS client, domain services per feature).
- **`frontend/src/views/`** — Page components organized by feature: `chat/`, `settings/`, `chatbot/`, `facebook/`, `analytics/`, `dashboard/`, `auth/`.
- **`frontend/src/components/`** — Reusable components organized by domain (`chat/`, `chatbot/`, `layout/`, `shared/`).
- **`frontend/src/composables/`** — Vue composables for shared logic.
- **`frontend/src/i18n/`** — Multi-language support (en, ar, es JSON locales).
- **`frontend/src/router/`** — Vue Router with route guards.
- **`frontend/vite.config.ts`** — Vite with `@` alias to `src/`.

### Key Flows

1. **Message send**: Frontend → API handler → `MessageProvider.Send()` → Meta API or Whatsmeow queue → WebSocket broadcasts delivery status back to chat UI.
2. **Webhook receive**: Meta/Whatsmeow webhook → handler processes event → stores in DB → WebSocket pushes to connected clients.
3. **Multi-tenancy**: Each request is scoped via `tenant.Scope` middleware filtering DB queries by organization.
4. **Auth**: JWT tokens in httpOnly cookies. Access token (15min) + refresh token (1 day). CSRF token in separate cookie/header pair.

### Configuration

Config via `config.toml` (TOML, parsed with koanf). Key sections: `app`, `server`, `database`, `redis`, `jwt`, `whatsapp`, `whatsmeow`, `storage`, `license`, `rate_limit`. See `config.example.toml` for all options.

## Conventions

- Go module: `github.com/compnew2006/whatomate`
- HTTP framework: `fasthttp` + `fastglue` (NOT net/http). Handlers receive `*fastglue.Request`.
- ORM: GORM with PostgreSQL. Models use soft deletes (`gorm.DeletedAt`).
- Frontend: Composition API only. Pinia for stores, Vue Query for server state, VeeValidate + Zod for forms.
- Tests: Go uses `testify`. Frontend uses Vitest (unit) + Playwright (E2E).
- Files under 500 lines. Extract when exceeding.
- Always read a file before editing it.

## Codegraph

This project uses codegraph for dependency analysis. The graph is at `.codegraph/graph.db`.

### Before modifying code:
1. `codegraph where <name>` — find where the symbol lives
2. `codegraph audit --quick <target>` — understand the structure
3. `codegraph context <name> -T` — get full context (source, deps, callers)
4. `codegraph fn-impact <name> -T` — check blast radius before editing

### After modifying code:
5. `codegraph diff-impact --staged -T` — verify impact before committing

### Navigation
- `codegraph where --file <path>` — file inventory (symbols, imports, exports)
- `codegraph query <name> -T` — function call chain (callers + callees)
- `codegraph path <from> <to> -T` — shortest call path between two symbols
- `codegraph deps <file>` — file-level dependencies
- `codegraph exports <file> -T` — per-symbol export consumers
- `codegraph children <name> -T` — sub-declarations (parameters, properties, constants)
- `codegraph search "<query>"` — semantic search (requires `codegraph embed`)
- `codegraph search "<query>" --mode keyword` — BM25 keyword search

### Impact & analysis
- `codegraph diff-impact main -T` — impact of branch vs main
- `codegraph audit <target> -T` — structural summary + impact + health in one report
- `codegraph triage -T` — ranked audit priority queue
- `codegraph check --staged --no-new-cycles` — CI validation predicates (exit 0/1)
- `codegraph batch t1 t2 t3 -T --json` — batch query multiple targets

### Overview
- `codegraph build .` — rebuild the graph (incremental by default)
- `codegraph map` — module overview (most-connected files)
- `codegraph stats` — graph health and quality score
- `codegraph structure --depth 2` — directory tree with cohesion scores
- `codegraph cycles` — circular dependency detection
- `codegraph triage --level file --sort coupling` — file-level hotspot analysis
- `codegraph roles --role dead -T` — find dead code (unreferenced symbols)
- `codegraph roles --role core -T` — find core symbols (high fan-in)
- `codegraph complexity -T` — per-function complexity metrics
- `codegraph communities --drift -T` — module boundary drift analysis
- `codegraph co-change <file>` — files that historically change together
- `codegraph branch-compare main HEAD -T` — structural diff between refs

### Deep analysis
- `codegraph dataflow <name> -T` — data flow edges (requires `build --dataflow`)
- `codegraph cfg <name> -T` — control flow graph (requires `build --cfg`)
- `codegraph ast --kind call <name> -T` — search stored AST nodes
- `codegraph owners [target]` — CODEOWNERS mapping for symbols
- `codegraph snapshot save <name>` — checkpoint graph DB before refactoring
- `codegraph plot` — interactive HTML dependency graph viewer

### Flags
- `-T` / `--no-tests` — exclude test files (use by default)
- `-j` / `--json` — JSON output for programmatic use
- `-f, --file <path>` — scope to a specific file
- `-k, --kind <kind>` — filter by symbol kind

### Semantic search

Use `codegraph search` to find functions by intent rather than exact name.
Combine multiple angles with `;` for better recall:

    codegraph search "validate auth; check token; verify JWT"

Multi-query uses Reciprocal Rank Fusion — functions ranking highly across
queries surface first. Use 2-4 sub-queries (2-4 words each):
- **Naming variants**: "send email; notify user; deliver message"
- **Abstraction levels**: "handle payment; charge credit card"
- **Input/output**: "parse config; apply settings"
- **Domain + technical**: "onboard tenant; create organization"

### Hooks (optional)

Hooks in `.claude/hooks/` can automatically inject dependency context on reads,
block commits with cycles or dead exports, and show diff-impact before commits.
See `docs/examples/claude-code-hooks/` for setup.