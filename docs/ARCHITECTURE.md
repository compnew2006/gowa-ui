# Whatomate — Architecture

## Overview Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Single Go Binary                      │
│  (embeds Vue SPA via //go:embed all:dist)                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐     │
│  │  fastglue     │  │  GORM       │  │  Redis       │     │
│  │  + fasthttp   │  │  PostgreSQL │  │  Streams     │     │
│  └──────┬───────┘  └──────┬──────┘  └──────┬───────┘     │
│         │                 │                  │                 │
│  ┌──────┴──────────────────┴──────────────────┴──────────┐      │
│  │              Middleware Chain                          │      │
│  │  CORS → SecurityHeaders → Auth → TenantScope → RBAC    │      │
│  └──────────────────────┬──────────────────────────────┘      │
│                         │                                     │
│  ┌──────────────────────┴──────────────────────────────┐      │
│  │              handlers.App                           │      │
│  │  (all API handlers as methods)                     │      │
│  └──┬──────────┬──────────┬──────────┬──────────┬─────┘      │
│     │          │          │          │          │               │
│     ▼          ▼          ▼          ▼          ▼               │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌────────┐    │
│  │ WA   │ │Chat  │ │Camp  │ │Bot   │ │FB   │ │WS Hub │    │
│  │Meta  │ │      │ │aigns│ │      │ │      │ │      │    │
│  └──┬───┘ └──────┘ └──────┘ └──────┘ └──────┘ └───────┘    │
│     │         │          │          │                                │
│     ▼         ▼          ▼          ▼                                │
│  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────────┐                       │
│  │ pkg/ │  │Redis│  │Queue│  │WebSocket│                       │
│  │prov.│  │Rate │  │Worker│  │Broadcast│                       │
│  └──────┘  └──────┘  └──────┘  └──────────┘                       │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │                      Frontend (Vue 3 SPA)                     │    │
│  │  Pinia stores ←→ API client ←→ Fasthttp server              │    │
│  │  Vue Router (permission guards) ←→ shadcn-vue + Tailwind     │    │
│  └──────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Backend Architecture

### Entry Point (`cmd/whatomate/main.go`)

Single binary with subcommands:
- **`server`** — Starts HTTP server on configured port (default 8080)
- **`worker`** — Consumes Redis Stream messages (campaign sends, inbound media processing)
- **`crypto-migrate`** — Re-encrypts all secrets with a new AES key
- **`admin-reset-password`** — Resets default admin password
- **`queue-migrate-campaigns`** — Migrates campaign queue format
- **`inbound-media-reconcile`** / **`legacy-media-reconcile`** — Media repair jobs

### Config (`internal/config/`)

TOML configuration via koanf. Main `Config` struct fields:
```
App, Server, Database, Redis, JWT, WhatsApp (provider selection),
Whatsmeow, Observability, AI, Storage, DefaultAdmin, RateLimit,
Campaigns, Cookie, License, FacebookOAuth, Facebook
```

Key config keys: `[whatsapp].provider` = `"meta"` | `"whatsmeow"`

### Middleware Chain (`internal/middleware/`)

Applied via `g.Before()` in order:
1. **CORS** — Cross-origin with configurable allowed origins
2. **SecurityHeaders** — CSP (with nonce for SPA), HSTS, X-Frame-Options
3. **Auth (`AuthWithDB`)** — Extracts JWT from `Authorization: Bearer`, `X-API-Key` header, or `whm_access` cookie
4. **TenantScope** — Resolves organization ID, sets scoped DB instance via `tenant.ScopedDB()`
5. **PermissionChecker** — Route-level RBAC (granular `HasPermission` checks at handler level)

### Handler Layer (`internal/handlers/`)

All handlers are methods on the `handlers.App` struct:
```go
type App struct {
    Config    *config.Config
    DB        *gorm.DB
    RDB       *redis.Client
    Provider  provider.MessageProvider  // Meta or WhatsMeow
    // ... other services
}
```

Handler return convention: `(*handlers.Envelope, error)` — standardized JSON response.

### Provider Abstraction (`pkg/provider/`)

`MessageProvider` interface abstracts WhatsApp operations:
- Meta Cloud API adapter: `pkg/whatsapp/`
- WhatsMeow Web protocol adapter: `pkg/whatsmeow/`

Optional extension interfaces:
- `PollProvider` — `SendPoll(ctx, instanceID, to, question, options, maxSelections)` for native WhatsApp polls (whatsmeow only)

Routes requiring Meta use `app.ProviderGuard("meta", handler)` wrapper.

### Multi-Tenancy (`internal/tenant/`)

- `TenantScope` middleware extracts org ID from context
- `tenant.ScopedDB(db, orgID)` returns a GORM instance with `WHERE organization_id = ?` added to all queries
- All tenant-aware models include `OrganizationID uuid.UUID` field

### WebSocket (`internal/websocket/`)

- Hub pattern with connected clients
- `/ws` endpoint (auth via message-based flow after upgrade)
- Real-time message broadcast to connected agents
- `/api/auth/ws-token` for obtaining WS auth token

### Queue/Worker (`internal/queue/`, `internal/worker/`)

- Redis Streams consumer groups
- Campaign send processing
- Inbound media download and processing
- Legacy media reconciliation

### License (`internal/license/`)

- Bootstrap → activation → enforcement
- Feature gating at startup
- Event logging

---

## Frontend Architecture

### Tech Stack
- **Framework:** Vue 3 (Composition API)
- **Build:** Vite
- **State:** Pinia (16 stores)
- **Routing:** Vue Router 4 with permission-based guards
- **UI:** shadcn-vue (new-york style) + Tailwind CSS v3
- **i18n:** vue-i18n (en, es, ar)
- **Data fetching:** @tanstack/vue-query
- **Notifications:** vue-sonner (toast)
- **TypeScript:** Strict mode

### Router Structure (`frontend/src/router/index.ts`)

```
/login, /register, /auth/sso/callback, /activate, /pricing  (public)
/  → AppLayout
    /dashboard
    /chat, /chat/:contactId
    /profile
    /templates, /flows
    /campaigns
    /instances, /instances/health
    /chatbot, /chatbot/settings, /chatbot/keywords, /chatbot/flows, /chatbot/flows/new, /chatbot/flows/:id/edit
    /chatbot/ai, /chatbot/transfers
    /analytics/agents, /analytics/meta-insights
    /settings, /settings/chatbot, /settings/accounts, /settings/tags, /settings/teams
    /settings/users, /settings/roles, /settings/agent-selection, /settings/api-keys
    /settings/webhooks, /settings/sso, /settings/license, /settings/custom-actions
    /contacts, /closed-chats
    /canned-responses, /saved-contents
    /whatsapp-filter, /group-search, /group-join-campaigns, /group-extraction
    /member-extraction, /group-participants
    /extract, /tags
    /facebook/comments, /facebook/page-search, /facebook/people-search, /facebook/accounts
    /facebook/group-search, /facebook/extract-likes, /facebook/page-messengers
    /facebook/extract-data, /facebook/auto-share, /facebook/retargeting
```

### Navigation (`navigationOrder`)

Sidebar menu built from `navigationOrder` array. Each entry has:
- `path` — route path
- `permission` — required permission string
- `childPaths` — nested children with their own permissions

### Pinia Stores (16)

| Store | Key |
|---|---|
| `auth` | User, JWT, login/logout, org switching |
| `config` | App config, provider type (meta/whatsmeow), features |
| `contacts` | Contact list, filtering, chat assignment |
| `instances` | WhatsMeow instance management |
| `teams` | Team CRUD and members |
| `users` | User management |
| `roles` | Role and permission management |
| `notes` | Conversation notes |
| `tags` | Contact tag management |
| `transfers` | Agent transfer queue |
| `agentSelection` | Customer agent selection menus |
| `organizations` | Multi-org management |
| `license` | License state and enforcement |
| `fbAccounts` | Facebook account connections |
| `canned-responses` | Quick reply templates |
| `saved-contents` | Content library |

---

## Database Schema

### PostgreSQL 17 + GORM

**Migration strategy:** GORM AutoMigrate only (no versioned migration files). Order defined in `GetMigrationModels()`.

### Model Categories (50 models)

#### Core (12)
Organization, OrganizationConfig, Permission, CustomRole, RolePermission, User, UserOrganization, Team, TeamMember, APIKey, LicenseRecord, LicenseEvent, SSOProvider, Webhook, CustomAction

#### WhatsApp (7)
WhatsAppAccount, WhatsAppInstance, InstanceNotification, Contact, MediaAsset, ContactUserDeletion, Message, WhatsAppStatus, ChatClosureRating, Template, WhatsAppFlow

#### Multi-Tenant Extensions (3)
Tag (composite PK: org_id + name), ConversationNote, ContactCollaborator

#### Campaigns (10)
BulkMessageCampaign, BulkMessageRecipient, NotificationRule, GroupJoinCampaign, GroupJoinRecipient, MessageExtractionCampaign, MessageExtractionResult, GroupExtractionCampaign, GroupExtractionResult, MemberExtractionCampaign, MemberExtractionResult, WhatsAppFilterBatch, WhatsAppFilterResult

#### Chatbot (8)
ChatbotSettings, KeywordRule, ChatbotFlow, ChatbotFlowStep, ChatbotSession, ChatbotSessionMessage, AIContext, AgentTransfer, SLATracking

#### Agent Selection (5)
AgentSelectionSettings, AgentSelectionParticipant, AgentSelectionOption, AgentSelectionSession, AgentSelectionAuditEvent

#### Facebook (7)
FacebookAccount, FacebookOAuthState, FacebookComment, FacebookCommentReply, FacebookCommentSettings, FBPageSearch, FBPeopleSearch

#### Other (4)
CannedResponse, SavedContent, Catalog, CatalogProduct, Widget, UserAvailabilityLog, GroupDirectory

#### Plugins (2 models)
InstanceUploadsCleanupAudit (per-instance-uploads-cleanup plugin)

### Indexing
Custom indexes defined in `getIndexes()` function, applied after AutoMigrate in `RunMigrationWithProgress()`.

---

## Authentication Flow

```
┌──────────┐     ┌───────────────┐     ┌───────────────┐
│  Login    │────▶│ POST /auth/login │────▶│ JWT Tokens   │
│  Form    │     │ (rate-limited)│     │ access+refresh │
└──────────┘     └───────────────┘     └──────┬───────┘
                                                       │
                                                       ▼
┌──────────────────────────────────────────────────────────┐
│  Protected API Request                                        │
│  Authorization: Bearer <access_token>                        │
│  OR X-API-Key: <key>                                     │
│  OR Cookie: whm_access=<access_token>                      │
│                                                               │
│  Middleware Chain:                                           │
│  1. AuthWithDB → validate JWT, set context                  │
│  2. TenantScope → resolve org, set scoped DB                │
│  3. PermissionChecker → handler-level RBAC check           │
└──────────────────────────────────────────────────────────┘
```

### Token Types
- **Access token** — Short-lived, subject = `access`, used for API calls
- **Refresh token** — Longer-lived, subject = `refresh`, used to get new access tokens
- **WS token** — For WebSocket authentication

### SSO Flow
1. `GET /api/auth/sso/providers` — List configured providers
2. `GET /api/auth/sso/{provider}/init` — Redirect to OAuth provider
3. `GET /api/auth/sso/{provider}/callback` — Handle callback, issue JWTs

---

## Security Model

### Layers
1. **CORS** — Origin whitelist, configurable via config
2. **CSP** — Content Security Policy with nonce for inline scripts
3. **HSTS** — HTTP Strict Transport Security
4. **JWT** — HS256 with algorithm enforcement (no algorithm confusion)
5. **API Key** — Hashed storage, validated per-request
6. **CSRF** — Double-submit cookie pattern (skipped for Bearer/API-key)
7. **Rate Limiting** — Redis-backed per-endpoint limits (login, register, refresh, SSO, webhook, outbound messages, campaign mutations)
8. **Tenant Isolation** — All DB queries scoped to organization

### Encryption
- AES-256 for secrets at rest (WhatsApp credentials, API keys, etc.)
- `internal/crypto/` handles encrypt/decrypt
- `crypto-migrate` subcommand for key rotation

---

## Deployment

### Single Binary
`make build-prod` produces a standalone binary containing:
1. Go backend
2. Embedded Vue SPA (`//go:embed all:dist`)
3. All assets (migrations, static files)

### Docker
`docker/docker-compose.yml` provides PostgreSQL 17 and Redis 7 for development.

### CI
- Go 1.25.8
- `go test -v -race -p 1` (sequential to avoid DB conflicts)
- Frontend E2E: Playwright Chromium

---

## Key Design Decisions

1. **fasthttp over net/http** — Performance-critical message handling; no standard `http.Handler` patterns
2. **GORM AutoMigrate over versioned migrations** — Simpler for a single-binary app; schema evolution is additive
3. **Provider interface** — Enables dual WhatsApp provider without conditional code in handlers
4. **Handler methods on App struct** — Single source of truth for all handler dependencies (DB, provider, queue, etc.)
5. **Frontend embedding** — Zero-deployment SPA; single binary serves both API and UI
6. **Redis Streams for queues** — Reliable message delivery with consumer groups for workers
7. **Permission-based routing** — Frontend uses permission strings to show/hide navigation items; backend validates at handler level
