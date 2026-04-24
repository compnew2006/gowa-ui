# Whatomate — Agent Instructions

## Project Identity

- **Go module**: `github.com/compnew2006/whatomate`
- **Binary**: single `whatomate` binary (entrypoint: `cmd/whatomate/main.go`)
- **HTTP framework**: **fastglue + fasthttp** (not `net/http` — do not use `http.Handler`, `httptest`, or `net/http` middleware patterns)
- **Config format**: TOML via [koanf](https://github.com/knadh/koanf) (`config.toml`, gitignored — copy from `config.example.toml`)
- **Go version (CI)**: 1.25.8

## Architecture

```
cmd/whatomate/          # Entrypoint (server, worker, crypto-migrate, etc.)
internal/
  config/               # Config loading + validation (koanf)
  crypto/               # AES-256 encrypt/decrypt for secrets at rest
  database/             # GORM + PostgreSQL 17 setup, migrations, default admin seeding
  frontend/dist/        # Embedded Vue SPA (populated by `make build-prod`)
  handlers/             # All API handlers
  middleware/            # Auth (JWT + API key), CSRF, CORS, rate limiting, tenant scope
  models/               # GORM models
  queue/                # Redis Streams consumer groups (campaigns, inbound_media)
  websocket/            # Hub/client WS via fasthttp/websocket
  worker/               # Job processing (campaign sends, media recovery)
  license/              # License enforcement
pkg/
  provider/             # MessageProvider interface
  whatsapp/             # Meta Cloud API adapter
  whatsmeow/            # WhatsApp Web protocol adapter
frontend/               # Vue 3 SPA (see below)
mcp-server/             # MCP sidecar (separate Node.js package)
```

- **Dual provider**: `config.toml` → `[whatsapp].provider` = `meta` (Cloud API) or `whatsmeow` (Web protocol). Wired in `main.go` via `MessageProvider` interface in `pkg/provider/interface.go`.
- **Frontend embedding**: `//go:embed all:dist` in `internal/frontend/embed.go`. For production, `make build-prod` copies `frontend/dist/` into the embed directory before compiling.

## Development Commands

### Prerequisites

- Go 1.25.x, Node.js >=20.19 or >=22.12
- PostgreSQL 17 + Redis 7 (via Docker: `docker compose -f docker/docker-compose.yml up -d db redis`)

### Backend (port 8080)

```bash
make run-migrate          # Run server with DB migrations
make run                  # Run server without migrations
make backend-watch        # Hot-reload with air (auto-installs if missing)
```

### Frontend (port 3000)

```bash
cd frontend && npm install
npm run dev               # Proxies /api -> localhost:8080, /ws -> ws://localhost:8080
npm run build             # Production build -> frontend/dist/
```

### Full-stack

```bash
make dev                  # Backend + frontend concurrently
make dev-watch            # Hot-reload backend + frontend concurrently
```

### Production build

```bash
make build-prod           # Builds frontend, copies to embed dir, compiles standalone binary
```

## Testing

### Go Tests

**Require running PostgreSQL and Redis.** Set env vars:

```bash
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"
```

```bash
go test -v ./...                                 # All tests (needs DB + Redis)
go test -v ./internal/handlers/...               # Single package
go test -v -run TestFunctionName ./path/to/pkg   # Single test
```

- **CI runs**: `go test -v -race -p 1` (sequential package execution to avoid DB conflicts)
- **Database tests**: `make test-db` spawns an ephemeral Docker Postgres on port 5433
- **No gomock/sqlmock** — real DB + Redis with `TRUNCATE CASCADE` cleanup, hand-written mocks in `test/testutil/mocks.go`
- **Handler tests**: construct `fasthttp.RequestCtx` + `fastglue.Request` directly (not `httptest.Server`)
- **Test fixtures**: `test/testutil/fixtures.go` — factory functions (`CreateTestUser`, `CreateTestOrganization`, etc.)
- **Hardcoded test secrets**: `TestJWTSecret`, `TestEncryptionKey` in `test/testutil/fixtures.go`

### Frontend Tests

```bash
cd frontend
npm run test:unit          # Vitest (unit)
npm run test:e2e           # Playwright (e2e, Chromium only)
```

- E2E tests in `frontend/e2e/tests/`, Page Object Model in `frontend/e2e/pages/`
- E2E requires backend on `BASE_URL` (default `http://localhost:8080`)
- E2E global setup seeds: `admin@test.com`, `manager@test.com`, `agent@test.com` (password: `Password123!`)

### MCP Server Tests

```bash
cd mcp-server && npm ci
npm run lint && npm run typecheck
npm run test                # Unit + integration
npm run test:e2e            # E2E
```

## Linting & Quality

```bash
make lint                        # golangci-lint (CI uses --timeout=5m)
cd frontend && npm run lint      # ESLint
cd frontend && npm run typecheck # vue-tsc --noEmit (strict mode)
```

## Key Conventions

- **Handler pattern**: methods on `handlers.App` struct; return `(*handlers.Envelope, error)`
- **Auth context in tests**: `testutil.SetAuthContext(req, orgID, userID)`
- **Multi-tenancy**: `TenantScope` middleware scopes DB per org; frontend sends `X-Organization-ID` header
- **CSRF**: double-submit cookie (`whm_csrf` + `X-CSRF-Token` header), skipped for Bearer/API-key auth
- **Frontend path alias**: `@/` -> `src/` (Vite + tsconfig)
- **Frontend components**: shadcn-vue (new-york style), Tailwind CSS v3
- **i18n**: vue-i18n; locale files in `frontend/src/i18n/locales/` (en, es, ar shipped)
- **Release**: push tag `v*` triggers GitHub Actions (binary + Docker image)

## Common Pitfalls

- **Do NOT use `net/http` patterns** in backend — this project uses `fasthttp`/`fastglue` everywhere
- **`config.toml` is gitignored** — never commit it; use `config.example.toml` as reference
- **Production binary needs frontend build first** — `go build` alone won't include the SPA
- **Tests will skip** (not fail) if `TEST_DATABASE_URL` or `TEST_REDIS_URL` is unset
- **`-p 1` is required** when running multiple packages that share the test database
- **Go module path** is `github.com/compnew2006/whatomate`, not a path matching local directory
- **Frontend dev proxy** expects backend on port 8080; Vite serves frontend on port 3000
