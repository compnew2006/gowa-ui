# Whatomate — Project Context

## 1. System Goal

Whatomate is an open-source WhatsApp Business platform for multi-tenant organizations. It enables:

- **Real-time customer chat** via WhatsApp (Cloud API or WhatsMeow/Web protocol)
- **Chatbot automation** (keyword rules, visual flow builder, AI context, agent transfers)
- **Bulk messaging campaigns** (individual, group-join, message/group/member extraction)
- **Contact & group management** (tags, collaboration, directory, filters)
- **Analytics dashboard** (messages, agents, chatbot, Meta insights, custom widgets)
- **Facebook integration** (OAuth account verification, managed page refresh/connect/disconnect/remove, comments, page search, people search, retargeting)
- **Multi-provider** switching between Meta Cloud API and WhatsMeow via `MessageProvider` interface

The system serves a single embedded Vue SPA from a standalone Go binary.

---

## 2. Go Backend Structure

```
cmd/whatomate/main.go          # Entrypoint — server, worker, crypto-migrate, admin-reset, etc.
internal/
  config/                      # TOML config via koanf (Config struct)
  crypto/                      # AES-256 encrypt/decrypt for secrets at rest
  database/                    # PostgreSQL + GORM, AutoMigrate, seeding, pre-migration fixes
  frontend/                    # go:embed for Vue SPA dist/
  handlers/                    # All API handlers (methods on handlers.App struct)
  middleware/                  # Auth (JWT+API-key), CSRF, CORS, RateLimit, TenantScope, Permissions
  models/                      # ~50 GORM models (see ARCHITECTURE.md)
  queue/                       # Redis Streams consumer groups (campaigns, inbound_media)
  tenant/                      # Multi-tenant scoping (ScopedDB per org)
  websocket/                   # Hub/client WebSocket via fasthttp
  worker/                      # Background job processing
  license/                     # License enforcement
  observability/               # Metrics (Prometheus), pprof, tracing
pkg/
  provider/                     # MessageProvider interface
  whatsapp/                     # Meta Cloud API adapter
  whatsmeow/                    # WhatsApp Web protocol adapter
```

**Key entrypoint commands:**
- `server` — HTTP server (port 8080)
- `worker` — Redis queue consumer
- `crypto-migrate` — re-encrypt secrets with new key
- `admin-reset-password` — reset admin password
- `queue-migrate-campaigns`, `inbound-media-reconcile`, `legacy-media-reconcile` — maintenance

---

## 3. Vue Frontend Structure

```
frontend/
  src/
    main.ts                    # createApp with Pinia, VueQuery, i18n, router
    router/index.ts              # All routes with permission-based navigation
    stores/                     # Pinia stores (16 total):
      auth, config, contacts, instances, teams, users,
      roles, notes, tags, transfers, agentSelection,
      organizations, license, fbAccounts, canned-responses,
      saved-contents
    i18n/locales/               # en, es, ar
    components/                 # Vue components (shadcn-vue, Tailwind)
    composables/                 # Shared composables
    api/                        # API client (axios-based)
    types/                      # TypeScript types
```

**Frontend tech stack:** Vue 3 + Vite + Pinia + TypeScript + Tailwind CSS v3 + vue-i18n + @tanstack/vue-query

**Route permissions:** Each route has a `meta.permission` field checked by the router guard. Navigation sidebar uses `navigationOrder` with `childPaths` to build menus.

---

## 4. Domains & Entities

### Core Domain
| Entity | File | Purpose |
|---|---|---|
| `Organization` | `models/models.go` | Multi-tenant org (multi-org users) |
| `User` | `models/models.go` | User with role, availability |
| `UserOrganization` | `models/models.go` | Many-to-many user-org membership |
| `Team` / `TeamMember` | `models/models.go` | Agent teams |
| `CustomRole` / `Permission` / `RolePermission` | `models/roles.go` | RBAC |
| `APIKey` | `models/models.go` | Programmatic API access |

### WhatsApp Domain
| Entity | File | Purpose |
|---|---|---|
| `WhatsAppAccount` | `models/models.go` | Connected WA account |
| `WhatsAppInstance` | `models/instance.go` | WhatsMeow instance |
| `Contact` | `models/models.go` | Customer contact |
| `Message` | `models/models.go` | Chat message |
| `MediaAsset` | `models/models.go` | Media storage reference |
| `WhatsAppStatus` | `models/whatsapp_status.go` | Status messages |
| `Template` | `models/models.go` | Meta message templates |
| `WhatsAppFlow` | `models/models.go` | Meta flows |
| `Tag` | `models/tags.go` | Contact tags (composite PK) |

### Campaign Domain
| Entity | File | Purpose |
|---|---|---|
| `BulkMessageCampaign` | `models/bulk.go` | Campaign definition |
| `BulkMessageRecipient` | `models/bulk.go` | Recipient + send status |
| `GroupJoinCampaign` | `models/group_join.go` | Auto group-join campaigns |
| `GroupJoinRecipient` | `models/group_join.go` | Group join target |
| `MessageExtractionCampaign/Result` | `models/message_extraction.go` | Extract messages from groups |
| `GroupExtractionCampaign/Result` | `models/group_extraction.go` | Extract group metadata |
| `MemberExtractionCampaign/Result` | `models/member_extraction.go` | Extract group members |
| `WhatsAppFilterBatch/Result` | `models/whatsapp_filter.go` | Validate WA numbers |

### Chatbot Domain
| Entity | File | Purpose |
|---|---|---|
| `ChatbotSettings` | `models/chatbot.go` | Bot config (business hours, SLA, AI) |
| `KeywordRule` | `models/chatbot.go` | Auto-reply on keywords |
| `ChatbotFlow` / `ChatbotFlowStep` | `models/chatbot.go` | Visual flow builder |
| `ChatbotSession` / `ChatbotSessionMessage` | `models/chatbot.go` | Bot session tracking |
| `AIContext` | `models/chatbot.go` | AI knowledge base |
| `AgentTransfer` | `models/chatbot.go` | Transfer to human agent |

### Agent Selection Domain
| Entity | File | Purpose |
|---|---|---|
| `AgentSelectionSettings` | `models/agent_selection.go` | Menu config |
| `AgentSelectionParticipant` | `models/agent_selection.go` | Available agents |
| `AgentSelectionOption` | `models/agent_selection.go` | Menu options |
| `AgentSelectionSession` | `models/agent_selection.go` | Active sessions |
| `AgentSelectionAuditEvent` | `models/agent_selection.go` | Audit trail |

### Facebook Domain
| Entity | File | Purpose |
|---|---|---|
| `FacebookAccount` / `FacebookOAuthState` | `models/fb_account.go` | FB connection |
| `FacebookComment` / `FacebookCommentReply` | `models/fb_comment.go` | FB comment management |
| `FacebookCommentSettings` | `models/fb_comment.go` | Comment reply rules |
| `FBPageSearch` / `FBPeopleSearch` | `models/fb_page_search.go`, `models/fb_people_search.go` | FB search |

### Other
- `Catalog` / `CatalogProduct` — Meta product catalog
- `Widget` / `WidgetFilter` — Custom analytics widgets
- `ConversationNote` — Per-contact notes
- `ContactCollaborator` — Shared contact access
- `CannedResponse` / `SavedContent` — Quick replies / content library
- `Webhook` / `CustomAction` — Outgoing webhooks
- `SSOProvider` — Single sign-on
- `LicenseRecord` / `LicenseEvent` — License tracking
- `ChatClosureRating` — Post-chat satisfaction

**Total: ~50 GORM models**

---

## 5. API Routes

### Public (no auth)
| Method | Path | Purpose |
|---|---|---|
| GET | `/health`, `/ready` | Health checks |
| POST | `/api/auth/login`, `/api/auth/register`, `/api/auth/refresh` | Auth |
| GET/POST | `/api/webhook` | Meta webhook verify/receive |
| GET/POST | `/api/auth/sso/{provider}/init`, `/callback` | SSO |
| GET | `/ws` | WebSocket |
| POST | `/api/license/bootstrap`, `/api/license/activate` | License |

### Authenticated (JWT or API-Key, TenantScoped)
| Group | Routes |
|---|---|
| **Current User** | `/api/me`, `/api/me/settings`, `/api/me/password`, `/api/me/availability`, `/api/me/organizations` |
| **Users** | `/api/users` CRUD, send restrictions |
| **Roles/Permissions** | `/api/roles` CRUD, `/api/permissions` |
| **API Keys** | `/api/api-keys` CRUD |
| **Accounts** | `/api/accounts` CRUD, test, subscribe, business profile |
| **Contacts** | `/api/contacts` CRUD, soft-delete, assign, tags, collaborators, session-data |
| **Chats** | `/api/chats` list, claim, close, reopen, set-public, messages |
| **Messages** | Send text/media/template, typing, reactions, revoke, read, statuses |
| **Media** | Serve media, retry download |
| **Templates** (Meta only) | CRUD, sync, publish, upload-media |
| **Flows** (Meta only) | CRUD, save/publish/deprecate/duplicate/sync |
| **Instances** (Whatsmeow) | CRUD, health, QR, connect/pair/disconnect/reconnect |
| **Facebook** | Accounts, OAuth, pages, insights, comments, settings, search |
| **Campaigns** | CRUD, start/pause/cancel/retry, recipients, media |
| **Group Campaigns** | Group join, group/member/message extraction |
| **Group Directory** | CRUD, categories, countries, preview, import |
| **Group Participants** | List, add, remove, promote, demote |
| **WhatsApp Filter** | Batches, results, export |
| **Chatbot** | Settings, keywords, flows, AI contexts, transfers, sessions |
| **Agent Selection** | Settings, participants, options, preview, test-send, audit, sessions |
| **Teams** | CRUD, members |
| **Canned Responses** | CRUD, send, increment usage |
| **Saved Contents** | CRUD, categories, preview, import, media |
| **Analytics** | Dashboard, messages, chatbot, agents, comparison, Meta |
| **Widgets** | CRUD, data-sources, data, layout |
| **Org Settings** | General, uploads-cleanup |
| **Organizations** | CRUD, members, current org |
| **SSO** | Settings CRUD (admin) |
| **Webhooks** | CRUD, test |
| **Custom Actions** | CRUD, execute, redirect |
| **Catalogs** (Meta only) | CRUD, sync, products |
| **Notes** | Per-contact conversation notes |
| **Import/Export** | Generic data import/export |
| **Extraction** | Contacts export, stats, history sync |
| **Notifications** | List, dismiss |

**Total: ~200+ API endpoints**

---

## 6. Database & Migrations

- **DB:** PostgreSQL 17 via GORM
- **Cache/Queue:** Redis 7 (rate limiting, streams, sessions)
- **Migrations:** GORM `AutoMigrate` on all models (no versioned migration files)
- **Migration order:** Defined in `GetMigrationModels()` in `internal/database/postgres.go`
- **Seed data:** Default admin user, permissions, roles, default widgets, system roles for all orgs
- **Pre-migration fixes:** `applyPreMigrationFixes()` before auto-migrate
- **Post-migration:** Backfills for organization configs, user organizations, last_inbound_at, SLA tracking

**No external migration tool (no golang-migrate files).** All schema evolution happens through GORM AutoMigrate.

---

## 7. Authentication & Authorization

### Authentication Flow
1. **JWT Bearer token** — Primary auth method. HS256 signed, access token + refresh token + WS token.
2. **API Key** — `X-API-Key` header for programmatic access (validated against `APIKey` table).
3. **Cookie fallback** — `whm_access` cookie for browser sessions.
4. **SSO** — Google/OIDC via `/api/auth/sso/{provider}/init|callback`.

### Authorization
- **Multi-tenancy:** `TenantScope` middleware scopes all DB queries to the requesting user's organization.
- **RBAC:** `CustomRole` with `Permission` and `RolePermission` join table. Handler-level checks via `HasPermission()`.
- **Route-level:** Auth middleware skips public paths; protected routes get both `AuthWithDB` and `TenantScope`.
- **CSRF:** Double-submit cookie (`whm_csrf` cookie + `X-CSRF-Token` header), skipped for Bearer/API-key auth.
- **Organization switching:** `/api/auth/switch-org` with JWT re-issuance.

### Context Keys
- `ContextKeyUserID`, `ContextKeyOrganizationID`, `ContextKeyEmail`, `ContextKeyRoleID`, `ContextKeyIsSuperAdmin`
- `ContextKeyUser`, `ContextKeyOrganization` — Full model objects

---

## 8. Conventions

### Backend
- **Handler pattern:** All handlers are methods on `handlers.App` struct. Return `(*handlers.Envelope, error)`.
- **Envelope:** Standardized JSON response via `(*handlers.Envelope, error)`.
- **Provider abstraction:** `MessageProvider` interface in `pkg/provider/` — Meta vs WhatsMeow.
- **Meta-only routes:** Wrapped with `app.ProviderGuard("meta", handler)`.
- **Config:** TOML via koanf, loaded from `config.toml` (gitignored).
- **Testing:** Real DB + Redis with TRUNCATE CASCADE cleanup. `fasthttp.RequestCtx` + `fastglue.Request` for handler tests. Hardcoded test secrets in `test/testutil/fixtures.go`.
- **No `net/http`:** Everything uses `fasthttp`/`fastglue`.
- **Frontend embed:** `//go:embed all:dist` in `internal/frontend/embed.go`. `make build-prod` copies frontend/dist/ then compiles.

### Frontend
- **Path alias:** `@/` → `src/` (Vite + tsconfig)
- **UI:** shadcn-vue (new-york style) + Tailwind CSS v3
- **State:** Pinia stores with composables
- **i18n:** vue-i18n with en/es/ar locales
- **API:** Centralized in `src/api/` using axios
- **Route permissions:** `meta.permission` on each route, checked by router guard

---

## 9. Dangerous Areas (Do NOT modify without permission)

1. **`internal/database/postgres.go` — `GetMigrationModels()`, `applyPreMigrationFixes()`, `SeedPermissionsAndRoles()`** — Schema migration and seeding logic. Changing the order can break DB initialization.
2. **`internal/middleware/middleware.go` — `AuthWithDB()`, `TenantScope()`** — Core auth/tenant logic. Bugs here break all authenticated endpoints.
3. **`internal/crypto/`** — Encryption key handling. Changing this can permanently lock encrypted secrets.
4. **`cmd/whatomate/main.go` — `setupRoutes()`** — All route registrations. Adding/removing routes here affects the entire API surface.
5. **`internal/frontend/embed.go` — `//go:embed` directive** — Frontend embedding. Must match `make build-prod` output.
6. **`internal/models/models.go` — `BaseModel` struct** — All models inherit from this. Changes propagate globally.
7. **`internal/tenant/` — Tenant scoping logic. Bugs here cause cross-org data leaks.
8. **`test/testutil/fixtures.go`** — Hardcoded test secrets (`TestJWTSecret`, `TestEncryptionKey`). Used by all tests.
9. **License enforcement code** — `internal/license/` — Changing this bypasses the licensing system.
10. **Redis Stream consumer groups** — `internal/queue/` — Worker message processing. Errors can cause message loss.

---

## 10. Running & Testing

### Prerequisites
- Go 1.25.x, Node.js >=20.19 or >=22.12
- PostgreSQL 17 + Redis 7 (Docker: `docker compose -f docker/docker-compose.yml up -d db redis`)

### Running
```bash
make run-migrate          # Server with DB migrations
make run                  # Server without migrations
make backend-watch        # Hot-reload with air
make dev                  # Backend + frontend concurrently
make dev-watch            # Hot-reload both
```

### Frontend Dev
```bash
cd frontend && npm install
npm run dev               # Port 3000, proxies /api→8080, /ws→ws://8080
```

### Production Build
```bash
make build-prod           # Frontend build → embed dir → single Go binary
```

### Testing
```bash
# Go tests (requires DB + Redis)
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"
go test -v ./...
go test -v -race -p 1 ./...  # CI mode (sequential)

# Frontend
cd frontend
npm run test:unit          # Vitest
npm run test:e2e           # Playwright (Chromium)
```

### Linting
```bash
make lint                        # golangci-lint
cd frontend && npm run lint      # ESLint
cd frontend && npm run typecheck # vue-tsc --noEmit
```
