<div align="center">

<img src="https://placehold.co/120x120/25D366/white?text=W&font=montserrat" alt="Whatomate Logo" width="90" height="90" />

# Whatomate

### WhatsApp Business Messaging Platform

**Chat · Automate · Campaign · Analyze — All in One Place**

<br/>

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.4-4FC08D?style=flat-square&logo=vue.js&logoColor=white)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io/)
[![TypeScript](https://img.shields.io/badge/TypeScript-strict-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-Proprietary-6B7280?style=flat-square)](#license)
[![Uptime](https://img.shields.io/badge/Uptime_SLA-99.9%25-22C55E?style=flat-square)]()

<br/>

[**Quick Start**](#-quick-start) · [**Features**](#-features) · [**Architecture**](#-architecture) · [**Tech Stack**](#-tech-stack) · [**Development**](#-development) · [**Deployment**](#-deployment)

</div>

---

## Overview

Whatomate is a production-ready, multi-tenant WhatsApp Business platform built for teams that communicate with customers at scale. It combines a real-time live chat inbox, an AI-powered chatbot engine, bulk campaign tools, and a customizable analytics dashboard — all delivered from a single Go binary with an embedded Vue 3 SPA.

<br/>

<div align="center">

| 10M+ Messages Delivered | 99.9% Uptime SLA | 50+ Countries | AES-256 Encrypted |
|:-:|:-:|:-:|:-:|
| Enterprise throughput | Battle-tested reliability | Global reach | Security-first design |

</div>

---

## ✨ Features

<details open>
<summary><strong>🗨️ Live Chat</strong></summary>

<br/>

Real-time WebSocket-powered inbox with full conversation lifecycle management and team collaboration.

| Capability | Details |
|---|---|
| **Real-Time Messaging** | Instant delivery, read receipts, typing indicators, auto-reconnect with heartbeat |
| **Rich Media Support** | Images, videos, audio, documents — drag-and-drop upload with full-screen viewer |
| **Smart Contact Sidebar** | Multi-account toggle, real-time search, status filters, resizable compact/wide views |
| **Chat Lifecycle** | Assign, transfer, claim, close, reopen — with bulk operations and auto-reset schedules |
| **Smart Messaging Tools** | Canned responses with media, emoji picker, quick reactions, WhatsApp Flows, templates |
| **Contact Info Panel** | Color-coded tags, collaborators, conversation notes, session data, internal annotations |
| **WhatsApp Status Stories** | View, reply, and compose WhatsApp Status updates from within the chat interface |
| **CSAT Ratings** | Automated 1–10 satisfaction surveys on chat close with multi-language templates |

</details>

<details>
<summary><strong>📢 Campaigns</strong></summary>

<br/>

Send bulk WhatsApp messages with template support, recipient management, scheduling, and live progress tracking.

| Capability | Details |
|---|---|
| **Campaign Creation** | Approved WhatsApp templates with header media; supports Meta Cloud API and whatsmeow |
| **Recipient Management** | CSV import with auto-detection; up to 10,000 recipients with per-recipient parameters |
| **Full Lifecycle Control** | Start, pause, cancel, retry — resume without re-sending already delivered messages |
| **Real-Time Tracking** | WebSocket live counters: sent, delivered, read, failed — per recipient |
| **Scheduled Sending** | Future-date scheduling with automatic background scheduler activation |
| **Safety Guard Rails** | Inbound-only mode, draft-only policy, instance validation, send-block detection |
| **Rate Limiting** | Redis Lua distributed throttling; randomized 20–45 s delays for compliance |
| **Error Recovery** | Per-recipient retry, automatic campaign completion, DLQ-based resilience |

</details>

<details>
<summary><strong>🤖 Chatbot & Automation</strong></summary>

<br/>

Build intelligent auto-responders with a visual flow builder, LLM integration, and seamless human handoff.

| Capability | Details |
|---|---|
| **Visual Flow Builder** | Drag-and-drop multi-step flows: text, buttons, input validation, API fetch, WhatsApp Flows |
| **AI-Powered Responses** | OpenAI, Anthropic Claude, and Google Gemini — configurable temperature, system prompt, history |
| **Keyword Auto-Responses** | Exact, contains, starts_with, regex matching with time-based and tag-based conditions |
| **Agent Transfer Queue** | Chatbot-to-human handoff with team-based queue pickup and real-time WebSocket notifications |
| **SLA Tracking** | First response, resolution, and escalation timers with automatic breach detection |
| **Business Hours** | Per-day schedules with timezone support and custom out-of-hours messages |
| **Inactivity Management** | Configurable auto-remind and auto-close for inactive conversations |
| **Session Inspector** | Admin tool for debugging active chatbot sessions in real time |

</details>

<details>
<summary><strong>📊 Analytics</strong></summary>

<br/>

Customizable dashboards, agent performance metrics, CSAT analytics, and real-time operational insights.

| Capability | Details |
|---|---|
| **Custom Dashboard** | Drag-and-drop widgets with persistent per-user layout — numbers, charts, pie, bar, line |
| **Agent Performance** | Response time, resolution rate, volume, CSAT ratings — with side-by-side comparison |
| **Chatbot Analytics** | Session counts, flow completion rates, drop-off points, transfer rate tracking |
| **CSAT Surveys** | Automated surveys with multi-language support, follow-up reminders, aggregate scoring |
| **Message Volume** | Inbound/outbound trends, delivery status breakdown, per-instance tracking |
| **Data Export** | CSV export for ratings, contacts, campaigns — structured for external BI tools |
| **Meta Insights** | Message cost, conversation category, and account-level WhatsApp Business metrics |
| **Live Counters** | WebSocket-driven active chat count, pending queue length, agent availability |

</details>

<details>
<summary><strong>🔒 Security & Auth</strong></summary>

<br/>

Enterprise-grade authentication, encryption, rate limiting, and multi-tenant data isolation.

| Capability | Details |
|---|---|
| **Single Sign-On** | Google, Microsoft, GitHub, Facebook + custom OIDC; domain restrictions, nonce validation |
| **API Key Auth** | bcrypt-hashed keys with configurable expiry, permission scoping, and usage tracking |
| **AES-256 Encryption** | AES-256-GCM with Argon2 key derivation for all secrets; CLI migration tool included |
| **CSRF Protection** | Double-submit cookie pattern; auto-skipped for Bearer/API key authentication |
| **Rate Limiting** | Redis atomic distributed rate limiting per endpoint — consistent across instances |
| **Security Headers** | Per-request CSP nonce injection, HSTS, X-Frame-Options, Referrer-Policy, Permissions-Policy |
| **Multi-Org Isolation** | Database-level tenant scoping via middleware; users can belong to multiple orgs |
| **Granular RBAC** | 25+ permission resources × read/write/execute/import/export; custom role editor |

</details>

<details>
<summary><strong>⚙️ Platform & Infrastructure</strong></summary>

<br/>

Auto-scaling workers, message resilience queues, media lifecycle management, and full observability.

| Capability | Details |
|---|---|
| **Multi-Language & RTL** | English, Spanish, Arabic shipped; full RTL layout support with per-user preference |
| **Worker Auto-Scaling** | Per-tenant scaling by Redis queue depth with global budget cap to prevent exhaustion |
| **Message Resilience** | Exponential backoff retry, dead letter queue, idempotency keys, inbound media self-heal |
| **Media Management** | Configurable retention policies, cleanup workers, legacy media restore before deletion |
| **License Management** | Ed25519-signed JWT with HWID binding, quota enforcement, and post-expiry grace period |
| **Observability** | `/health` + `/ready` probes, optional Prometheus metrics, Go pprof profiling |
| **MCP Server Sidecar** | Node.js Model Context Protocol package for LLM tool integration and AI automation |
| **Single Binary Deploy** | Embedded Vue SPA compiled into a single Go binary via `//go:embed` |

</details>

---

## 🏗️ Architecture

```
whatomate/
├── cmd/
│   └── whatomate/              # Entrypoint — server, worker, crypto-migrate subcommands
├── internal/
│   ├── config/                 # Config loading & validation (koanf + TOML/ENV)
│   ├── crypto/                 # AES-256-GCM encryption for secrets at rest
│   ├── database/               # GORM + PostgreSQL 17 setup and auto-migrations
│   ├── frontend/dist/          # Embedded Vue SPA (populated by `make build-prod`)
│   ├── handlers/               # All HTTP API route handlers
│   ├── middleware/              # JWT + API key auth, CSRF, CORS, rate limiting
│   ├── models/                 # GORM data models
│   ├── queue/                  # Redis Streams consumer groups
│   ├── websocket/              # WebSocket hub/client via fasthttp
│   ├── worker/                 # Background jobs — campaign sends, media recovery
│   └── license/                # License enforcement and quota management
├── pkg/
│   ├── provider/               # MessageProvider interface (dual-provider abstraction)
│   ├── whatsapp/               # Meta Cloud API adapter
│   └── whatsmeow/              # WhatsApp Web protocol adapter
├── frontend/                   # Vue 3 + TypeScript SPA
└── mcp-server/                 # MCP sidecar (separate Node.js package)
```

### Provider Architecture

Whatomate supports two WhatsApp backends, switchable via a single config key:

```toml
[whatsapp]
provider = "meta"       # Official Meta Cloud API — highest reliability, template-enforced
# provider = "whatsmeow"  # WhatsApp Web protocol — full feature parity, no template requirement
```

### Data Flow

```
Inbound Message
    │
    ▼
Meta Cloud API / whatsmeow
    │
    ▼
Redis Streams (consumer groups)
    │
    ├──► Chatbot Engine → Flow Executor → Agent Transfer Queue
    │
    ├──► WebSocket Hub → Connected Agents (real-time)
    │
    └──► Persistence Layer (PostgreSQL 17)
```

---

## 🛠️ Tech Stack

### Backend

| Technology | Version | Role |
|---|---|---|
| **Go** | 1.25 | High-performance backend with goroutine concurrency |
| **fasthttp** | latest | HTTP server — 10× throughput vs net/http |
| **GORM** | latest | Type-safe ORM with auto-migrations |
| **PostgreSQL** | 17 | Primary database with JSONB support |
| **Redis** | 7 | Streams, pub/sub, distributed rate limiting, queues |

### Frontend

| Technology | Version | Role |
|---|---|---|
| **Vue 3** | 3.4 | Reactive UI with Composition API |
| **TypeScript** | strict | Full type safety across the SPA |
| **Vite** | latest | Lightning-fast HMR and production builds |
| **Tailwind CSS** | v3 | Utility-first styling |
| **Pinia** | latest | State management with devtools |
| **shadcn-vue** | latest | Accessible, composable component library |

### WhatsApp Integration

| Technology | Role |
|---|---|
| **Meta Cloud API** | Official WhatsApp Business messaging |
| **whatsmeow** | WhatsApp Web protocol — full feature parity |
| **goja** | Server-side JavaScript runtime for custom actions |
| **MCP Server** | AI tool integration via Model Context Protocol |

### Tooling

| Technology | Role |
|---|---|
| **koanf** | Flexible TOML/ENV configuration |
| **golangci-lint** | Comprehensive Go code quality enforcement |
| **Playwright** | End-to-end browser automation testing |
| **Vitest** | Fast unit testing with Vite integration |
| **GitHub Actions** | Automated build, test, and release pipeline |

---

## 🚀 Quick Start

### Prerequisites

| Requirement | Version |
|---|---|
| Go | 1.25.x |
| Node.js | ≥ 20.19 or ≥ 22.12 |
| PostgreSQL | 17 |
| Redis | 7 |

### 1. Start Infrastructure

```bash
docker compose -f docker/docker-compose.yml up -d db redis
```

### 2. Configure

```bash
cp config.example.toml config.toml
```

Edit `config.toml`:

```toml
[app]
port = 8080

[database]
host     = "localhost"
port     = 5432
name     = "whatomate"
user     = "postgres"
password = "your-password"

[redis]
addr = "localhost:6379"

[whatsapp]
provider = "meta"    # "meta" for Cloud API · "whatsmeow" for Web protocol
```

### 3. Run

```bash
# Backend — apply migrations and start server (port 8080)
make run-migrate

# Frontend — in a separate terminal (port 3000)
cd frontend && npm install && npm run dev

# Or run both concurrently
make dev
```

Open [http://localhost:3000](http://localhost:3000) — that's it.

---

## 💻 Development

### Backend Commands

```bash
make run              # Start server
make run-migrate      # Start server with DB migrations
make backend-watch    # Hot-reload with air
make lint             # golangci-lint
make test             # Full test suite
```

### Frontend Commands

```bash
cd frontend

npm run dev           # Dev server (port 3000)
npm run build         # Production build
npm run lint          # ESLint
npm run typecheck     # vue-tsc --noEmit
npm run test:unit     # Vitest unit tests
npm run test:e2e      # Playwright end-to-end tests
```

### Running Tests

Go tests require a live PostgreSQL and Redis instance:

```bash
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5432/test?sslmode=disable"
export TEST_REDIS_URL="redis://127.0.0.1:6379/1"

go test -v -race -p 1 ./...

# Or use the ephemeral test database helper
make test-db
```

---

## 📦 Deployment

### Docker Compose (Recommended)

```bash
docker compose -f docker/docker-compose.yml up -d
```

This starts PostgreSQL 17, Redis 7, and the Whatomate application container.

### Single Binary

```bash
# Build: compiles frontend, embeds SPA, produces single binary
make build-prod

# Run
./whatomate server -config config.toml
```

### Release Pipeline

Push a version tag to trigger the automated GitHub Actions pipeline (binary artifact + Docker image):

```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## Security

Whatomate is built with security at every layer:

- **Encryption at rest** — AES-256-GCM with Argon2 key derivation for all stored secrets
- **Transport security** — HTTPS-enforced with HSTS and comprehensive security headers
- **Authentication** — JWT sessions + API keys (bcrypt-hashed) + SSO/OIDC
- **Multi-tenant isolation** — Database-level query scoping; no cross-org data leakage
- **Rate limiting** — Redis-based distributed limiting on all sensitive endpoints
- **Audit logging** — Full audit trail for API key usage and administrative actions

To report a security vulnerability, please contact the maintainers privately. Do not open a public issue.

---

## License

Whatomate is proprietary software. All rights reserved.
Unauthorized copying, distribution, or modification is strictly prohibited.

---

<div align="center">

**Whatomate** — Built for teams that take customer communication seriously.

</div>
