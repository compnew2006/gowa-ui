# Whatomate — Architecture

> **Updated:** 2026-06-22  
> **Total Functions Analyzed:** 6,710 functions across 3,098 files  
> **Exclusions:** Dashboard/ directory

---

## 1. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Single Go Binary (whatomate)                    │
│                   (embeds Vue SPA via //go:embed all:dist)            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  ┌──────────┐  │
│  │  fastglue     │  │  GORM       │  │  Redis       │  │  otel    │  │
│  │  + fasthttp   │  │  PostgreSQL │  │  Streams/PubSub│  │  metrics │  │
│  └──────┬───────┘  └──────┬──────┘  └──────┬───────┘  └──────────┘  │
│         │                 │                 │                         │
│  ┌──────┴──────────────────┴──────────────────┴──────────────────┐   │
│  │                  Middleware Chain                              │   │
│  │  CORS → Recovery → RateLimit → Auth → TenantScope → RBAC      │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                             │                                        │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                      handlers.App                             │   │
│  │  (central dependency container — Config, DB, Redis, Log,      │   │
│  │   WhatsApp, WhatsmeowManager, WSHub, Queue, License, etc.)    │   │
│  └──┬─────┬─────┬──────┬──────┬──────┬──────┬──────┬──────┬─────┘   │
│     │     │     │      │      │      │      │      │      │          │
│     ▼     ▼     ▼      ▼      ▼      ▼      ▼      ▼      ▼          │
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───────┐      │
│  │WA │ │WA │ │Chat│ │Camp│ │Bot │ │FB  │ │WS  │ │Worker│ │Plugin │  │
│  │Meta│ │Web│ │    │ │aign│ │    │ │    │ │Hub │ │      │ │ *.go  │  │
│  └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ └───┘ └───────┘      │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Embed: all:./frontend/dist  (Vue 3 SPA)                      │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

## 2. Binary Subcommands

```
whatomate
├── server                → HTTP server (fasthttp + fastglue)
├── worker                → Background job processor
├── admin-reset-password  → CLI password reset
├── crypto-migrate        → Encrypt V2 → V3 migration
├── queue-migrate-campaigns → Queue namespace migration
├── inbound-media-reconcile → Stuck media reconciliation
├── legacy-media-reconcile → Legacy media reconciliation
└── version               → Print version
```

## 3. Core Packages

### 3.1 `internal/handlers/` — HTTP Handlers (234 files)

| Group | Files | Key Functions |
|---|---|---|
| **Auth** | `auth_handlers.go`, `auth_crypto.go`, `auth_utils.go`, `auth_expiry.go`, `auth_types.go` | Login, Register, Refresh, SSO |
| **Users** | `users.go` + helpers | User CRUD, settings, chat background |
| **Accounts** | `accounts.go` | WhatsApp Cloud API account CRUD |
| **Instances** | `instances.go` | WhatsApp Web instance management |
| **Contacts** | `contacts.go`, `contacts_management.go`, `contacts_messaging.go` | Contact CRUD, messaging |
| **Messages** | `messages.go` + helpers | Send messages, templates |
| **Campaigns** | `campaigns.go`, `campaign_policy.go`, `campaign_scheduler.go`, `campaign_start.go` | Campaign CRUD, scheduling |
| **Chatbot** | `chatbot.go`, `chatbot_processor.go` | Chatbot settings, keyword rules, flows |
| **Templates** | `templates.go` | Message template CRUD, submit, sync |
| **Analytics** | `analytics.go`, `widgets.go` | Dashboard + custom widgets |
| **Webhooks** | `webhooks.go`, `webhook.go`, `webhook_dispatch.go` | Webhook CRUD, Meta webhook handler |
| **Organization** | `organization.go` | Org settings, members |
| **Roles** | `roles.go` | RBAC roles & permissions |
| **Teams** | `teams.go` | Agent teams |
| **Chat Lifecycle** | `chat_lifecycle.go`, `chat_cleanup.go`, `chat_close_ratings.go` | Chat status, assignment, close |
| **Flows** | `flows.go` | WhatsApp interactive flows |
| **Catalog** | `catalog.go` | Product catalog |
| **Media** | `media.go`, `media_visibility.go` | Media download/serve |
| **Import/Export** | `import_export.go` | CSV/Excel import/export |
| **SSO** | `sso_handlers.go`, `sso_types.go`, `sso_utils.go`, `sso_security.go` | SSO providers |
| **Canned Responses** | `canned_responses.go`, `canned_response_send.go`, `canned_response_media.go` | Quick replies |
| **Tags** | `tags.go` | Contact tagging |
| **Notifications** | `notifications.go` | In-app notifications |
| **API Keys** | `apikeys.go` | API authentication keys |
| **Config** | `config_handler.go` | Runtime config |
| **Business Profile** | `business_profile.go` | WhatsApp profile management |
| **WebSocket** | `websocket.go` | WS connection upgrade |
| **Upload Cleanup** | `uploads_cleanup_*.go` | Upload retention management |

### 3.2 `internal/models/` — GORM Models

| Model | Purpose | Key Fields |
|---|---|---|
| `Organization` | Multi-tenant org | Name, Slug, Settings |
| `User` | User account | Email, PasswordHash, RoleID, SSO fields |
| `UserOrganization` | Org membership | UserID, OrgID, RoleID, IsDefault |
| `WhatsAppAccount` | Cloud API account | AppID, PhoneID, AccessToken (encrypted) |
| `Contact` | Chat contact | PhoneNumber, ProfileName, Status, AssignedUserID |
| `Message` | Chat message | Content, Direction, Type, Status, Media |
| `Template` | Message template | MetaTemplateID, Name, Language, Category |
| `WhatsAppFlow` | Interactive flow | MetaFlowID, FlowJSON, Screens |
| `Campaign` | Bulk campaign | Name, Status, TemplateID, Recipient counts |
| `CampaignRecipient` | Campaign recipient | Phone, Status, Attempts |
| `Webhook` | Outgoing webhook | URL, Events, Headers, Secret |
| `APIKey` | API auth key | KeyPrefix, KeyHash, ExpiresAt |
| `LicenseRecord` | License record | Full licensing fields |
| `Team` | Agent team | Name, AssignmentStrategy |
| `Role` | RBAC role | Name, Permissions (JSONB) |
| `Widget` | Dashboard widget | DataSource, Metric, DisplayType |
| `MediaAsset` | Media file | FileHash, S3Key, MimeType |
| `SSOProvider` | SSO config | Provider, ClientID, AllowedDomains |

### 3.3 `internal/worker/` — Background Workers (21 files)

| Worker | Purpose |
|---|---|
| Campaign send | Processes campaign recipient jobs with rate limiting |
| Inbound media | Downloads incoming WhatsApp media |
| Facebook auto-reply | Auto-replies to Facebook comments |
| Group extraction | Extracts WhatsApp group members |
| Group join | Joins WhatsApp groups |
| Message extraction | Extracts chat history |
| Member extraction | Extracts member profiles |
| Scheduled sends | Sends scheduled messages |
| Send policy | Enforces sending restrictions |
| Uploads cleanup | Cleans up expired uploads |
| Inbound media self-heal | Periodic stuck media recovery |
| WhatsApp filter | Processes filter commands |

### 3.4 `internal/websocket/` — Real-Time

| Component | Purpose |
|---|---|
| `Hub` | Central WebSocket manager |
| `Client` | Per-connection client |
| Message types | 12+ message types for real-time events |

### 3.5 `internal/queue/` — Async Queue

| Component | Purpose |
|---|---|
| `redis.go` | Redis Streams implementation |
| `pubsub.go` | Redis Pub/Sub for campaign stats |
| `queue.go` | Generic queue interface |

### 3.6 Plugin, Module & License Layers

Whatomate separates **extensibility** (Plugin), **per-org feature gating** (Module), and **host-bound capacity + tier entitlements** (License) into three independent layers. See [`PLUGINS_AND_MODULES.md`](./PLUGINS_AND_MODULES.md) for the full guide.

```
Request → /api/<feature>/...
   │
   ▼
[fastglue route registered ONLY if the plugin is compiled in]
   │
   ▼
core.GateModule("<module-key>", handler)
   │
   ├─ Layer 1: License tier gate  (LicenseAllowsModule(tier, key))
   ├─ Layer 2: Module state gate  (ModuleManager.IsEnabled(orgID, key))
   ▼
handler body
   │
   ├─ Layer 3: RBAC  (app.HasPermission — core catalog + plugin namespace)
   └─ Layer 4: Quota (License.CheckQuotaWithDelta — orgs/users/WA/storage)
```

| Layer | Package | Decides | Granularity |
|---|---|---|---|
| **Plugin** | `internal/core` + `plugin/<name>/` | Which code is compiled in | Per-binary |
| **Module** | `internal/core/module_manager.go` + `plugin/module-management/` | Which compiled plugins are ON, globally or per-org | Global + per-org (DB tables) |
| **License** | `internal/license/` | Host-level caps + tier constraining module enablement | Per-host (HWID JWT) |

**Plugin optional interfaces** (both follow the "embed `Plugin` + one method" pattern):
- `ManagedPlugin` → `Manifest()` — participates in the Module system (DB-controlled enable/disable).
- `PermissionProvidingPlugin` → `Permissions()` — contributes plugin-namespaced RBAC (e.g. `plugin.facebook.accounts:pages_manage`), seeded by `core.SyncPluginPermissions` at startup and enforced through the existing `app.HasPermission`.

**Startup wiring** (`cmd/whatomate/main.go`): `InitPlugins` → `SetLicenseTierGetter` → `SyncManagedModules` → `SyncPluginPermissions` → `RegisterPluginRoutes`.

## 4. Provider Layer

### `pkg/provider/` — Message Provider Interface

```
MessageProvider (interface)
├── SendText, SendImage, SendDocument, SendVideo, SendAudio
├── MarkRead, SendReaction, RevokeMessage
├── DownloadMedia, UploadMedia, GetMediaURL
├── SendTextReply
├── PollProvider (SendPoll, SendPollVote)
└── GroupProvider (GetGroups, VerifyMembership, Add/Remove Participants)
```

### `pkg/whatsapp/` — WhatsApp Cloud API (Meta)

```
Client
├── SendText, SendImage, SendTemplate, etc.
├── Template management (create, submit, sync)
├── Flow management (create, publish, deprecate)
├── Catalog management
├── Webhook parsing
└── Analytics
```

### `pkg/whatsmeow/` — WhatsApp Web (WhatsMeow)

```
ConnectionManager
├── Connect/Disconnect/Logout
├── QR code / phone pairing
├── Client pool (multi-instance)
├── Health monitoring
└── Event dispatcher

Adapter (MessageProvider implementation)
├── Send methods (text, image, document, video, audio)
├── Group operations
├── Media download/upload
├── Incoming message processing
├── Typing indicators
└── Presence management
```

## 5. Middleware Chain

| Middleware | Path | Purpose |
|---|---|---|
| `CORS` | Global | Cross-origin headers |
| `Recovery` | Global | Panic recovery |
| `RateLimit` | Per-route | Redis-based rate limiting |
| `Auth` | /api/* | JWT token validation |
| `AuthWithDB` | /api/* | Full auth + RBAC + tenant |
| `TenantScope` | /api/* | Org-scoped DB queries |

## 6. Frontend Architecture

```
frontend/src/
├── services/
│   ├── api.ts              ← Base HTTP client (auth refresh, interceptors)
│   ├── auth.ts             ← Auth API
│   ├── contacts.ts         ← Contacts + messages API
│   ├── instances.ts        ← WhatsApp instances API
│   ├── campaigns.ts        ← Campaigns API
│   ├── templates.ts        ← Templates API
│   ├── chatbot.ts          ← Chatbot API
│   ├── accounts.ts         ← WhatsApp accounts API
│   ├── webhooks.ts         ← Webhooks API
│   ├── analytics.ts        ← Analytics API
│   ├── organizations.ts    ← Organization API
│   ├── users.ts, roles.ts, teams.ts ← User/RBAC API
│   ├── tags.ts, cannedResponses.ts, flows.ts ← Feature API
│   ├── media.ts, notifications.ts, widgets.ts ← Utility API
│   └── license.ts          ← License API
├── stores/                 ← Pinia stores
│   ├── auth.ts
│   ├── contacts.ts
│   ├── instances.ts
│   ├── campaigns.ts
│   ├── templates.ts
│   ├── chatbot.ts
│   ├── config.ts
│   └── license.ts
├── composables/            ← Reusable logic
│   ├── useCrudState.ts
│   ├── usePagination.ts
│   ├── useColorMode.ts
│   ├── useConditionEvaluator.ts
│   ├── useFlowHistory.ts
│   ├── useFlowSimulation.ts
│   └── useApiMocker.ts
├── views/                  ← Page components
│   ├── chat/
│   ├── campaigns/
│   ├── chatbot/
│   ├── settings/
│   ├── analytics/
│   ├── contacts/
│   ├── templates/
│   ├── accounts/
│   ├── instances/
│   ├── teams/
│   ├── users/
│   └── roles/
├── components/             ← Reusable UI components
├── router/index.ts         ← Vue Router with guards
├── i18n/                   ← Internationalization
└── types/                  ← TypeScript types
```

## 7. Plugin Architecture

```
internal/core/plugin.go
├── Plugin (interface)
│   ├── Name() string
│   ├── Init(app, db, rdb, log) error
│   ├── Routes(g *fastglue.Fastglue)
│   └── Migrate(db *gorm.DB) error
├── RegisterPlugin(p)       ← init()-time registration
├── RegisterPluginRoutes()  ← Route registration
└── RunPluginMigrations()   ← AutoMigrate

plugin/
├── campaign-interactive/         ← Interactive campaign templates
└── per-instance-uploads-cleanup/ ← Per-instance upload retention
```

## 8. Data Flow

### Chat Message Flow (Inbound)

```
Meta Webhook → POST /api/webhook
  → WebhookHandler()
    → processIncomingMessageWithoutDuplicateCheck()
      → fetchExistingIncomingMessageIDs()
        → Create message record in DB
          → WSHub.BroadcastToOrg(orgID, new_message)
            → ChatbotProcessor (if enabled)
              → Evaluate keyword rules → Execute flow → Send reply
```

### Chat Message Flow (Outbound)

```
POST /api/messages/send
  → SendOutgoingMessage()
    → resolveProviderInstanceID()       ← Cloud API or WhatsApp Web?
      → toWhatsAppAccount()             ← Encrypt account secrets
        → sendViaProvider()
          → Provider.SendText/Image/etc.
            → updateContactLastMessage()
              → broadcastNewMessage()
                → dispatchMessageSentWebhook()
```

### Campaign Flow

```
POST /api/campaigns → CreateCampaign()
POST /api/campaigns/{id}/start → StartCampaign()
  → Enqueue recipients to Redis Stream
    → Worker.HandleRecipientJob()
      → executeRecipientSend()
        → computeCampaignDelayDuration()
          → provider.SendTemplate()
            → updateRecipientStatusConditional()
              → publishCampaignStats() ← Pub/Sub
                → checkCampaignCompletion()
```

## 9. Key Metrics

| Metric | Value |
|---|---|
| Total functions analyzed | 6,710 |
| Total project files (excl Dashboard) | 2,238 |
| Go backend files (incl tests) | 497 |
| Frontend source files | 505 (167 TS + 336 Vue + 2 CSS) |
| Total API routes | 699 (382 base + rate-limit variants) |
| Database models | 25+ core models |
| Background workers | 15+ distinct worker types |
| WebSocket message types | 12+ |
| Plugins | 2 (campaign-interactive, per-instance-uploads-cleanup) |
| External providers | 2 (WhatsApp Cloud API, WhatsApp Web) |
| Middleware layers | 6 (CORS, Recovery, RateLimit, Auth, TenantScope, RBAC) |

---

*End of Architecture Document — Full function analysis in `FUNCTION_ANALYSIS.md`*
