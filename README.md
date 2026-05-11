<div align="center">

# Whatomate

**Comprehensive WhatsApp Business Messaging Platform**

Chat with customers, automate responses with AI chatbots, run campaigns, and gain insights — all in one place.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.4-4FC08D?style=flat-square&logo=vue.js)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis)](https://redis.io/)
[![License](https://img.shields.io/badge/License-Proprietary-blue?style=flat-square)]()

[Getting Started](#getting-started) · [Features](#features) · [Deep Dives](#deep-dive) · [Tech Stack](#tech-stack) · [Development](#development)

</div>

---

## Trust & Compliance

| | | | |
|:---:|:---:|:---:|:---:|
| **AES-256 Encrypted** | **Open Architecture** | **GDPR Ready** | **Enterprise Grade** |

---

## Stats

| Metric | Value |
|--------|-------|
| Messages delivered | 10M+ |
| Uptime SLA | 99.9% |
| Countries supported | 50+ |
| Customer support | 24/7 |

---

## Features

> A complete suite of tools designed for teams that communicate with customers at scale through WhatsApp.

### 01 · Live Chat
Real-time messaging with customers. Assign conversations, add internal notes, and collaborate across teams.

### 02 · AI Chatbot
Build intelligent chatbots with keyword matching, flow builders, and AI-powered contexts for automated responses.

### 03 · Custom Dashboard
Create personalized dashboards with draggable widgets, charts, and real-time analytics for your KPIs.

### 04 · Campaigns
Send bulk WhatsApp campaigns with templates. Track delivery, reads, and responses in real time.

### 05 · WhatsApp Flows
Design interactive WhatsApp flows with rich UI components for surveys, appointments, and more.

### 06 · Agent Analytics
Monitor agent performance, response times, conversation volumes, and customer satisfaction metrics.

### 07 · Team Management
Organize agents into teams, assign conversations automatically, and manage permissions with fine-grained RBAC.

### 08 · Enterprise Security
AES-256 encryption, JWT + API key authentication, SSO support, and comprehensive audit logging.

---

## Deep Dive

### Chat

A powerful real-time messaging interface with multi-provider support, team collaboration, and enterprise-grade chat management.

| Feature | Description |
|---------|-------------|
| **Real-Time Messaging** | WebSocket-powered live messaging with instant delivery, read receipts, and typing indicators. Live WebSocket connection with auto-reconnect and heartbeat. Message status tracking: sent, delivered, read, failed. Typing presence indicators for both agents and customers. |
| **Rich Media Support** | Send and receive images, videos, audio, and documents with in-chat preview and viewer. Image, video, audio, and document upload with drag-and-drop. Full-screen media viewer with zoom and download. Configurable media retention policies. |
| **Smart Contact Sidebar** | Unified sidebar with multi-account toggle, search, status filters, and tag-based filtering. Multi-account / multi-instance toggle. Real-time search across contacts and messages. Status filters with unread count badges. Resizable sidebar with compact and wide view modes. |
| **Chat Assignment & Lifecycle** | Full chat lifecycle management with assignment, transfer, claim, close, and reopen workflows. Assign chats to agents or teams. Transfer chats between agents. Bulk close, assign, and reopen operations. Automatic chat assignment reset on configurable schedules. |
| **Smart Messaging Tools** | Canned responses, emoji picker, quick reactions, and AI-powered chatbot integration. Canned responses with media attachments. Emoji picker with custom skin tone. Quick reactions on messages. Message templates and WhatsApp Flows. |
| **Contact Info Panel** | Rich contact details with tags, collaborators, metadata, session data, and conversation notes. Tag management with color-coded labels. Collaborator panel for multi-agent visibility. Conversation notes for internal team communication. Session data panel showing chatbot flow context. |
| **WhatsApp Status Stories** | View, reply to, and manage WhatsApp status stories from within the chat interface. Compose and send WhatsApp Status updates. View incoming contact statuses with media viewer. Reply to status messages directly. |
| **Chat Close Ratings** | Automated customer satisfaction surveys with 1–10 rating scales on chat close. Rating request sent automatically when agent closes a chat. Multi-language rating templates. Automated follow-up reminders. Per-agent CSAT score analytics. |

**Chat capabilities:** Reply · Reactions · Revoke · Emoji · Canned Responses · Print · Tags · Notes · Search · RTL Support · Custom Actions · Access Control · Auto-Reset · Templates · Flows · Bulk Operations

---

### Campaigns

Send bulk WhatsApp messages with template support, recipient management, scheduling, and real-time progress tracking.

| Feature | Description |
|---------|-------------|
| **Campaign Creation** | Create campaigns using approved WhatsApp templates with header media and configurable settings. Select from approved WhatsApp templates with rich text body editor. Attach header media for visual campaign messages. Dual provider: Meta Cloud API templates or whatsmeow body messages. |
| **Recipient Management** | Import recipients via CSV or select from existing contacts with personalized template parameters. CSV import with column auto-detection and parameter mapping. Import from existing contacts filtered by creation date. Per-recipient template parameters for personalized messages. Configurable import limit (default 10,000 recipients). |
| **Campaign Execution** | Full lifecycle control with start, pause, cancel, and retry actions. Start campaigns instantly or resume paused campaigns. Pause running campaigns to review progress. Cancel campaigns with automatic cleanup. Retry failed recipients without re-sending already delivered ones. |
| **Real-Time Progress Tracking** | Live progress updates via WebSocket with per-recipient delivery and read status. Live counters: sent, delivered, read, failed. Per-recipient status tracking with error details. Progress bar visualization in campaign list. |
| **Scheduled Sending** | Schedule campaigns for future execution with automatic start via background scheduler. Set date/time for automatic campaign start. Scheduler runs periodically, marks due campaigns as scheduled then starts them. Batch processing with configurable concurrency. |
| **Safety Policies & Guard Rails** | Organization-level policy enforcement, inbound-only mode, and WhatsApp connection validation. Organization-level draft-only policy. Strict inbound-only mode blocks campaigns without prior inbound messages. WhatsApp instance connection validation. Instance send-block detection with descriptive error messages. |
| **Rate Limiting & Send Delay** | Redis-based distributed rate limiting with randomized send delays between messages. Redis Lua script ensures consistent delays across multiple worker instances. Configurable min/max delay range (default 20–45 seconds). Strict mode enforces minimum delay floor for compliance. |
| **Error Recovery & Resilience** | Automatic error handling with per-recipient retry and campaign status management. Retry only failed recipients. Automatic campaign completion when all recipients are processed. |

**Campaign capabilities:** CSV Import · Contact Import · Templates · Header Media · Per-Recipient Params · Live Progress · Recipient Viewer · Pause/Resume · Retry Failed · Scheduling · Inbound Policy · Auto-Template

---

### Settings

Full control over users, roles, teams, WhatsApp instances, integrations, templates, and organization configuration.

| Feature | Description |
|---------|-------------|
| **User Management** | Complete user lifecycle management with role assignment, send restrictions, and activity controls. Create, edit, and deactivate users with full profile management. Assign roles and teams with granular permission inheritance. Configure per-user send restrictions. Activity status tracking across the platform. |
| **Roles & Permissions** | Role-based access control with a full permission matrix editor covering every platform action. Create custom roles with a checkbox matrix. Pre-built roles: Admin, Manager, Agent. Permission scoping per module. |
| **WhatsApp Instance Management** | Connect and manage WhatsApp instances with health monitoring, QR pairing, and settings control. QR code pairing for whatsmeow instances. Meta Cloud API account linking with token management. Real-time health dashboard with connection status. Per-instance send delay, auto-reply, and webhook configuration. |
| **Team Management** | Organize agents into teams for efficient chat assignment and workload distribution. Create teams and assign agents. Configure team-specific chat routing and auto-assignment rules. Team performance visibility in analytics. |
| **WhatsApp Message Templates** | Manage WhatsApp-approved message templates for campaigns and automated messaging. Sync templates directly from Meta Business Manager. Template preview with header media, body text, and buttons. Language and category filtering. |
| **Canned Responses** | Pre-written response templates with media attachments for fast, consistent replies. Create rich canned responses with text, images, videos, and documents. Quick-search and keyboard shortcut insertion. Shared across teams. |
| **Integrations & API** | Webhooks, API keys, SSO, and custom actions to extend and automate your workflow. Webhook endpoints for chat events and campaign status. API key management with scoped permissions. SAML/SSO integration for enterprise identity providers. Custom action buttons for the chat header. |
| **Organization & Preferences** | Organization profile, appearance, notifications, chat background, and license management. Organization name, logo, and workspace branding. Light/dark theme with custom chat wallpaper. Desktop and in-app notification preferences. License key activation with seat management. |

**Settings capabilities:** Users · Roles & RBAC · Teams · Instances · Health Monitor · Templates · Flows · Canned Replies · Contact Tags · API Keys · Webhooks · SSO/SAML · Chatbot Config · License · Notifications · Appearance

---

### Chatbot & Automation

Build intelligent auto-responders, visual conversation flows, AI-powered replies, and seamless agent transfer queues.

| Feature | Description |
|---------|-------------|
| **Visual Flow Builder** | Design multi-step chatbot conversations with a drag-and-drop visual editor and rich step types. Step types: text, template, buttons, input validation, API fetch, WhatsApp Flows. Per-step input validation with regex, retry limits, and custom error messages. Define contact info panel sections from collected flow data. Conditional branching based on user input, time, and contact attributes. |
| **AI-Powered Responses** | Integrate LLMs for intelligent, context-aware auto-replies with conversation history. Support for OpenAI, Anthropic (Claude), and Google (Gemini). Configurable temperature, system prompt, and conversation history window. Static or API-fetched AI contexts for domain-specific knowledge injection. |
| **Keyword Auto-Responses** | Set up rule-based automatic replies triggered by exact, partial, regex, or prefix matching. Four match modes: exact, contains, starts_with, regex. Conditions: time-based activation, contact tag filters, instance scope. Respond with text, templates, or trigger a full chatbot flow. |
| **Agent Transfer Queue** | Seamless handoff from chatbot to human agents with a team-based pickup queue system. Transfer steps in flows route customers to available agent teams. Queue-based pickup — agents claim next available transfer. Real-time transfer notifications via WebSocket. |
| **SLA Tracking & Escalation** | Enforce response and resolution deadlines with automatic breach detection and escalation. First response, resolution, and escalation deadline timers per transfer. Automatic breach warnings and configurable escalation actions. Auto-close transfers that exceed resolution SLA thresholds. |
| **Business Hours** | Define operating hours per day of the week with custom out-of-hours messages. Per-day schedule with open/close times and timezone support. Custom greeting and fallback message for outside business hours. Automatic chatbot behavior toggle based on business hour status. |
| **Client Inactivity Management** | Auto-remind and auto-close inactive customer conversations to free agent capacity. Configurable inactivity timeout with automatic reminder messages. Auto-close conversations after extended inactivity period. Per-instance inactivity settings for different service levels. |
| **Chatbot Configuration** | Fine-grained control over chatbot behavior, sessions, and fallback strategies. Per-account enable/disable toggle with default response and greeting buttons. Configurable session timeout and phone number exclusion list. Admin session inspector for debugging active chatbot conversations. |

**Chatbot capabilities:** Flow Builder · AI Responses · Keywords · Agent Transfers · SLA Tracking · Business Hours · Inactivity Mgmt · Session Timeout · Greeting Buttons · Template Steps · Queue Pickup · Fallback Msgs · AI Contexts · Session Inspector · Per-Account Toggle · Input Validation

---

### Analytics

Customizable dashboards, agent performance metrics, CSAT ratings, and real-time operational insights.

| Feature | Description |
|---------|-------------|
| **Customizable Dashboard** | Build your own analytics dashboard with drag-and-drop widgets and live data. Drag-and-drop widget layout with persistent per-user positioning. Widget types: number, percentage, line chart, bar chart, pie chart. Data sources: messages, contacts, campaigns, transfers, sessions. Real-time data refresh with configurable update intervals. |
| **Agent Performance Analytics** | Deep-dive into individual and team agent metrics with comparison views. Per-agent metrics: response time, resolution rate, chat volume, ratings. Side-by-side agent comparison for performance benchmarking. CSAT ratings export to CSV for external reporting. |
| **Chatbot Analytics** | Monitor chatbot session volume, flow completion rates, and transfer rates. Session counts, flow completion rates, and drop-off points. Keyword trigger frequency and response effectiveness. Transfer rate tracking from chatbot to human agents. |
| **CSAT Ratings & Surveys** | Automated customer satisfaction surveys with multi-language rating requests. 1–10 scale rating requests sent automatically on chat close. Multi-language rating templates. Automated follow-up reminders for unanswered ratings. Aggregate and per-agent CSAT score analytics. |
| **Message Volume Analytics** | Track inbound/outbound message trends, delivery rates, and status distribution. Message volume by direction: inbound vs outbound over time. Delivery status breakdown: sent, delivered, read, failed. Per-instance and per-account message volume tracking. |
| **Data Export & Reporting** | Export analytics data and contact records for external reporting and compliance. CSV export for agent ratings, contact lists, and campaign results. Import/export contacts with per-table configuration. Structured data formats for integration with external BI tools. |
| **Meta Insights (Cloud API)** | WhatsApp Business API analytics for message costs, status, and conversation metrics. Message cost tracking and billing analytics from Meta. Conversation category analytics. Account-level WhatsApp Business metrics and trends. |
| **Real-Time Operational Metrics** | Live counters and WebSocket-driven updates for instant operational awareness. Live active chat count, pending queue length, and agent availability. Real-time campaign progress via WebSocket push updates. Instance connection status and notification event streaming. |

**Analytics capabilities:** Custom Widgets · Charts & Graphs · Agent Compare · CSAT Surveys · Message Volume · Chatbot Stats · CSV Export · Meta Insights · Contact Metrics · Campaign Stats · Transfer Stats · Custom Layout · Data Sources · Live Counters · Performance · Trend Analysis

---

### Security & Auth

Enterprise-grade authentication, encryption, rate limiting, and multi-tenant isolation.

| Feature | Description |
|---------|-------------|
| **Single Sign-On (SSO)** | OAuth2/OIDC integration with major identity providers and custom configurations. Built-in providers: Google, Microsoft, GitHub, Facebook. Custom OIDC provider support for enterprise identity systems. Domain restrictions and default role assignment for SSO users. State token validation, nonce checking, and secure callback handling. |
| **API Key Authentication** | Programmatic access with secure, scoped API keys and usage tracking. bcrypt-hashed keys with configurable expiry and permission scoping. Per-key rate limiting and organization-level access control. Usage tracking and audit logging for all API key requests. |
| **AES-256 Encryption at Rest** | Military-grade encryption for all sensitive secrets stored in the database. AES-256-GCM encryption with Argon2 key derivation. Encrypts access tokens, app secrets, SSO credentials, and AI keys. CLI migration tool for upgrading legacy encryption schemes. |
| **CSRF Protection** | Double-submit cookie pattern to prevent cross-site request forgery attacks. Double-submit cookie (whm_csrf) with X-CSRF-Token header validation. Automatically skipped for Bearer token and API key authentication. Per-request token generation with secure SameSite cookie attributes. |
| **Per-Endpoint Rate Limiting** | Redis-based distributed rate limiting to protect against abuse and brute force. Login, register, token refresh, and SSO endpoint throttling. Webhook outbound and campaign mutation rate limits. Per-endpoint configurable limits with Redis atomic counters. Distributed — consistent across multiple server instances. |
| **Security Headers & CSP** | Comprehensive HTTP security headers with per-request Content Security Policy. CSP with per-request nonce injection for inline script protection. HSTS, X-Content-Type-Options, X-Frame-Options, Referrer-Policy. Permissions-Policy for browser feature control. |
| **Multi-Organization Isolation** | Complete data isolation between organizations with scoped database queries. Per-request tenant scoping via X-Organization-ID header middleware. Users can belong to multiple organizations with instant switching. Database-level query scoping prevents cross-org data leakage. |
| **Granular RBAC Permission System** | 25+ permission resources with fine-grained read/write/execute/import/export actions. 25+ permission resources covering every platform module. Route-level permission middleware for API endpoint protection. Pre-built system roles: Admin, Manager, Agent. Custom role creation with full permission matrix editor. |

**Security capabilities:** SSO/OIDC · API Keys · AES-256 · CSRF Guard · Rate Limiting · Security Headers · Multi-Org · RBAC · JWT Tokens · CORS Policy · Webhook HMAC · Password Policy · Panic Recovery · Permission Guard · Tenant Scope · Signed Invites

---

### Platform & Infrastructure

Multi-language support, license management, auto-scaling workers, resilience queues, and observability.

| Feature | Description |
|---------|-------------|
| **Multi-Language & RTL Support** | Full internationalization with three shipped locales and right-to-left layout support. Three shipped locales: English, Spanish, Arabic with complete UI translations. Full RTL (right-to-left) layout support for Arabic with automatic direction switching. Language switcher component with persistent per-user preference. |
| **Themes & Customization** | Light and dark mode with per-user chat background and accent color preferences. System-aware light/dark theme toggle with smooth transitions. Per-user custom chat wallpaper upload. Configurable organization branding, appearance, and notification preferences. |
| **Dynamic Worker Auto-Scaling** | Per-tenant worker scaling based on queue depth with global budget allocation. Automatic worker scaling per organization based on Redis queue depth. Global worker budget prevents resource exhaustion across all tenants. Per-org worker count, max queue size, and max instance configuration. Campaign send delay with Redis Lua script for consistent throttling. |
| **Message Resilience & Queues** | Multi-layer retry system with dead letter queues, idempotency, and self-healing. Outgoing message retry with exponential backoff on transient failures. Dead letter queue (DLQ) for permanent failure recovery and replay. Idempotency keys prevent duplicate message processing across retries. Inbound media self-heal loop reconciles stale queued downloads. |
| **Media Management & Retention** | Automated media lifecycle with retention policies, cleanup workers, and storage control. Configurable media retention policies with automatic expiry deletion. Uploads cleanup worker removes transient local files past retention window. Legacy media restore re-downloads expired Meta media before deletion. |
| **License Management** | Ed25519-signed JWT licenses with HWID binding, quota enforcement, and grace periods. Ed25519-signed JWT license tokens with machine HWID fingerprint binding. Quota enforcement: max organizations, users, endpoints, and storage per tier. Post-expiry grace period before platform lockout. Quota overage cleanup — admin must reduce usage before continuing. |
| **Observability & Infrastructure** | Health checks, Prometheus metrics, pprof profiling, and Docker deployment support. Liveness (/health) and readiness (/ready) probe endpoints. Optional Prometheus metrics and Go pprof profiling endpoints. Docker Compose setup for PostgreSQL 17, Redis 7, and the application. |
| **MCP Server Sidecar** | Model Context Protocol sidecar for AI tool integration and extended automation. Node.js MCP server package for connecting AI systems to Whatomate. Tool and resource provider interface for LLM-powered workflows. Extensible architecture for adding new AI integration capabilities. |

**Platform capabilities:** Multi-Language · Dark Mode · RTL Support · Auto-Scaling · Redis Streams · Auto-Retry · Dead Letter Queue · Media Retention · License Mgmt · HWID Binding · Health Checks · Prometheus · WebSocket Hub · Single Binary · MCP Sidecar · Marketing URLs

---

## Tech Stack

A carefully chosen stack optimized for performance, security, and developer experience.

### Backend

| Technology | Version | Purpose |
|------------|---------|---------|
| **Go** | 1.25 | High-performance backend with goroutine concurrency |
| **fasthttp** | — | Fast HTTP server — 10x faster than net/http |
| **GORM** | — | Type-safe ORM with auto-migrations |
| **PostgreSQL** | 17 | Advanced relational database with JSONB |

### Frontend

| Technology | Version | Purpose |
|------------|---------|---------|
| **Vue 3** | Composition API | Reactive state with Composition API |
| **TypeScript** | — | Strict mode with full type safety |
| **Vite** | — | Lightning-fast HMR and builds |
| **Tailwind CSS** | v3 | Utility-first CSS framework |

### Infrastructure

| Technology | Version | Purpose |
|------------|---------|---------|
| **Redis** | 7 | Streams, pub/sub, rate limiting, queues |
| **WebSocket** | — | Real-time org-scoped pub/sub messaging |
| **Docker** | — | Containerized deployment with Compose |
| **GitHub Actions** | — | Automated build, test, and release pipeline |

### WhatsApp Integration

| Technology | Version | Purpose |
|------------|---------|---------|
| **Meta Cloud API** | — | Official Cloud API for business messaging |
| **whatsmeow** | — | WhatsApp Web protocol — full feature parity |
| **goja** | — | Server-side JavaScript runtime for custom actions |
| **MCP Server** | Node.js | AI tool integration via Model Context Protocol |

### UI Libraries

| Technology | Purpose |
|------------|---------|
| **shadcn-vue** | Accessible, composable component library |
| **Pinia** | State management with devtools support |
| **vue-i18n** | Multi-language with RTL layout support |
| **Chart.js** | Charts and graphs for analytics dashboards |

### Tooling & Quality

| Technology | Purpose |
|------------|---------|
| **koanf** | Flexible TOML/ENV configuration |
| **golangci-lint** | Comprehensive Go code quality enforcement |
| **Playwright** | End-to-end browser testing automation |
| **Vitest** | Fast unit testing with Vite integration |

---

## Platform Capabilities

From API integrations to multi-language support, Whatomate adapts to your business needs.

| | | | | |
|:---|:---|:---|:---|:---|
| Message Templates | API Keys | Webhooks | Multi-Provider | Multi-Language |
| Role-Based Access | Contact Tags | Canned Responses | Custom Actions | SSO / SAML |

---

## Architecture

```
cmd/whatomate/              # Entrypoint (server, worker, crypto-migrate, etc.)
internal/
  config/                   # Config loading + validation (koanf)
  crypto/                   # AES-256 encrypt/decrypt for secrets at rest
  database/                 # GORM + PostgreSQL 17 setup, migrations
  frontend/dist/            # Embedded Vue SPA (populated by make build-prod)
  handlers/                 # All API handlers
  middleware/                # Auth (JWT + API key), CSRF, CORS, rate limiting
  models/                   # GORM models
  queue/                    # Redis Streams consumer groups
  websocket/                # Hub/client WS via fasthttp/websocket
  worker/                   # Job processing (campaign sends, media recovery)
  license/                  # License enforcement
pkg/
  provider/                 # MessageProvider interface
  whatsapp/                 # Meta Cloud API adapter
  whatsmeow/                # WhatsApp Web protocol adapter
frontend/                   # Vue 3 SPA
mcp-server/                 # MCP sidecar (separate Node.js package)
```

**Dual provider:** `config.toml` → `[whatsapp].provider` = `meta` (Cloud API) or `whatsmeow` (Web protocol).

**Frontend embedding:** `//go:embed all:dist` in `internal/frontend/embed.go`. For production, `make build-prod` copies `frontend/dist/` into the embed directory before compiling.

---

## Getting Started

### Prerequisites

- **Go** 1.25.x
- **Node.js** >=20.19 or >=22.12
- **PostgreSQL** 17 + **Redis** 7

### Quick Start

```bash
# Start infrastructure
docker compose -f docker/docker-compose.yml up -d db redis

# Backend (port 8080)
make run-migrate

# Frontend (port 3000) — in a separate terminal
cd frontend && npm install && npm run dev

# Or run both concurrently
make dev
```

### Production Build

```bash
make build-prod    # Builds frontend, copies to embed dir, compiles standalone binary
./whatomate server -config config.toml
```

### Configuration

Copy `config.example.toml` to `config.toml` and configure:

```toml
[app]
port = 8080

[database]
host = "localhost"
port = 5432

[redis]
addr = "localhost:6379"

[whatsapp]
provider = "meta"    # "meta" for Cloud API, "whatsmeow" for Web protocol
```

---

## Development

### Backend

```bash
make run                # Run server
make run-migrate        # Run with DB migrations
make backend-watch      # Hot-reload with air
make lint               # golangci-lint
make test               # All tests
```

### Frontend

```bash
cd frontend
npm run dev             # Dev server (port 3000)
npm run build           # Production build
npm run lint            # ESLint
npm run typecheck       # vue-tsc --noEmit
npm run test:unit       # Vitest
npm run test:e2e        # Playwright
```

### Testing

```bash
# Go tests (require PostgreSQL + Redis)
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"
go test -v -race -p 1 ./...

# Ephemeral test database
make test-db

# Frontend tests
cd frontend && npm run test:unit
cd frontend && npm run test:e2e
```

---

## Deployment

### Docker

```bash
docker compose -f docker/docker-compose.yml up -d
```

### Release

Push a tag `v*` to trigger the GitHub Actions pipeline (binary + Docker image).

```bash
git tag v1.0.0
git push origin v1.0.0
```

---

<div align="center">

**Ready to transform your WhatsApp communications?**

Join thousands of businesses using Whatomate to deliver exceptional customer experiences at scale.

Whatomate — WhatsApp Business Messaging Platform

</div>
