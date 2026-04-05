---
title: System Architecture
---

# System Architecture

This page covers the overall system architecture, technology stack, directory structure, and component design of Whatomate.

## Architecture Overview

Whatomate follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────┐
│              Frontend (React/Vite)               │
│         Embedded in Go binary via embed          │
├─────────────────────────────────────────────────┤
│              HTTP Server (fasthttp)              │
│  ┌─────────┬──────────┬──────────┬───────────┐  │
│  │Middleware│ Handlers │  Router  │  WebSocket│  │
│  └─────────┴──────────┴──────────┴───────────┘  │
├─────────────────────────────────────────────────┤
│              Business Logic Layer                │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │ Provider │   Queue   │ Chatbot  │ Campaign │  │
│  │  Guard   │  System   │ Engine   │ Worker   │  │
│  └──────────┴───────────┴──────────┴──────────┘  │
├─────────────────────────────────────────────────┤
│              Data Access Layer                   │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │   GORM   │   Redis   │  Crypto  │  Cache   │  │
│  │  (PostgreSQL) │  Client │  System  │  Layer   │  │
│  └──────────┴───────────┴──────────┴──────────┘  │
├─────────────────────────────────────────────────┤
│           External Integrations                  │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │   Meta   │ WhatsMeow │   AI     │  Webhook │  │
│  │  Cloud   │  Client   │ Providers│  Targets │  │
│  └──────────┴───────────┴──────────┴──────────┘  │
└─────────────────────────────────────────────────┘
```

## Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| HTTP Server | `valyala/fasthttp` | High-performance HTTP handling |
| Database | PostgreSQL + GORM | Primary data store, ORM |
| Cache | Redis | Session tokens, cached lookups, rate limiting |
| Queue | Redis | Campaign jobs, media downloads, pub/sub |
| WebSocket | `gorilla/websocket` | Real-time client communication |
| Auth | JWT (HS256) | Token-based authentication |
| Frontend | React 18 + Vite | SPA with embedded build |
| Styling | TailwindCSS | Utility-first CSS |
| Encryption | AES-256-GCM | Sensitive field encryption |

## Directory Structure

```
whatomate/
├── cmd/whatomate/
│   └── main.go              # Application entry point, server setup
├── internal/
│   ├── config/
│   │   └── config.go        # TOML config loader, env overrides
│   ├── models/
│   │   └── *.go             # GORM model definitions (30+ models)
│   ├── handlers/
│   │   ├── auth_handlers.go # Authentication endpoints
│   │   ├── contacts.go      # Contact/chat management
│   │   ├── messages.go      # Message sending
│   │   ├── campaigns.go     # Campaign CRUD
│   │   ├── chatbot.go       # Chatbot settings/rules/flows
│   │   ├── webhook.go       # Meta inbound webhook
│   │   ├── webhooks.go      # Outbound webhook management
│   │   ├── websocket.go     # WebSocket handler
│   │   ├── provider_guard.go# Provider-specific middleware
│   │   ├── cache.go         # Redis cache operations
│   │   └── ...              # Additional handlers
│   ├── middleware/
│   │   ├── auth.go          # JWT/API key authentication
│   │   ├── csrf.go          # CSRF protection
│   │   ├── security.go      # Security headers
│   │   ├── rate_limit.go    # Rate limiting
│   │   ├── logger.go        # Request logging
│   │   └── recovery.go      # Panic recovery
│   ├── worker/
│   │   ├── worker.go        # Campaign worker
│   │   ├── campaign_delay.go# Delay logic
│   │   ├── send_policy.go   # Send policy enforcement
│   │   └── idempotency.go   # Job idempotency
│   ├── queue/
│   │   ├── queue.go         # Redis queue abstraction
│   │   ├── consumer.go      # Job consumer
│   │   ├── publisher.go     # Job publisher
│   │   └── subscriber.go    # Pub/Sub subscriber
│   ├── crypto/
│   │   ├── crypto.go        # AES-256-GCM encryption
│   │   └── migration.go     # Crypto format migration
│   ├── frontend/
│   │   └── embed.go         # Embedded frontend filesystem
│   ├── websocket/
│   │   ├── hub.go           # WebSocket hub (org→connections)
│   │   └── messages.go      # WS message types
│   ├── contactutil/
│   │   └── contact.go       # Contact utilities
│   ├── templateutil/
│   │   └── template.go      # Template placeholder utilities
│   └── database/
│       └── migrations.go    # GORM AutoMigrate wrapper
├── pkg/
│   ├── provider/
│   │   └── provider.go      # MessageProvider interface
│   ├── whatsapp/
│   │   ├── client.go        # Meta Cloud API client
│   │   └── meta_adapter.go  # Meta provider adapter
│   └── whatsmeow/
│       ├── manager.go       # Connection manager
│       ├── adapter.go       # WhatsMeow provider adapter
│       └── queue.go         # Per-instance message queue
├── frontend/                 # React/Vite SPA
│   ├── src/
│   ├── e2e/                 # Playwright E2E tests
│   └── vite.config.js
├── config.example.toml       # Configuration template
└── go.mod
```

## Component Overview

### HTTP Server (fasthttp)

The server uses `valyala/fasthttp` for high-performance request handling. Routes are registered in `cmd/whatomate/main.go` through the `setupRoutes()` function.

```go
// Route registration pattern
app.POST("/api/auth/login", app.Login)
app.GET("/api/contacts", app.RequireAuth(app.ListContacts))
app.POST("/api/contacts/:id/messages", app.RequireAuth(app.SendMessage))
```

### Middleware Chain

Requests pass through a defined middleware chain before reaching handlers:

1. **CORS** — Cross-origin headers (fasthttp level)
2. **Security Headers** — X-Content-Type-Options, X-Frame-Options, etc.
3. **Request Logger** — Method, path, status, duration
4. **Recovery** — Panic catch, 500 response
5. **CSRF Protection** — Token validation for mutating requests
6. **Activity Log** — Audit significant actions
7. **Auth** — JWT or API key validation
8. **RBAC** — Permission checks (handler level)
9. **Provider Guard** — Provider compatibility (handler level)
10. **Rate Limiting** — Per-endpoint limits

### Message Provider Abstraction

The `MessageProvider` interface enables switching between Meta and WhatsMeow:

```go
type MessageProvider interface {
    SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error)
    SendMediaMessage(ctx context.Context, req *MediaMessageRequest) (*SendResult, error)
    SendTemplateMessage(ctx context.Context, req *TemplateMessageRequest) (*SendResult, error)
    MarkRead(ctx context.Context, messageID string) error
    SendTyping(ctx context.Context, contactID string, composing bool) error
}
```

See [Provider Abstraction](./provider-abstraction) for details.

### WebSocket Hub

The WebSocket hub maintains an organization-to-connections map for targeted broadcasts:

```
Client → /ws (JWT auth) → Hub.Register() → Read/Write loops
Hub → BroadcastToOrg() → All org member connections
```

See [WebSocket Events](./websocket-events) for details.

### Background Workers

Workers run as goroutines started in `main.go`:

| Worker | Trigger | Purpose |
|--------|---------|---------|
| SLA Processor | 1 minute | SLA breach checks, auto-close |
| Activity Retention | 1 hour | Delete old activity logs |
| Chat Assignment Reset | 1 minute | Reset stale assignments |
| Instance Auto-Campaign | 1 minute | Automated message sending |
| Campaign Worker | Continuous | Process campaign Redis queue |
| Inbound Media Worker | Continuous | Download inbound media |
| Campaign Stats Subscriber | Continuous | Broadcast stats via WS |

See [Background Workers](./background-workers) for details.

## Data Flow Diagrams

### Message Sending Flow

```
API Request → Auth Middleware → Permission Check → Load Contact/Account
  → Create Message Record (pending) → Send via Provider (async)
  → Update Status (sent/failed) → WebSocket Broadcast → Webhook Dispatch
```

### Incoming Message Flow

```
Meta Webhook → Signature Verification → Parse Payload
  → Find Account → Get/Create Contact → Save Message
  → Chatbot Processing (keywords, flows, AI, fallback)
  → Send Response → WebSocket Broadcast → Webhook Dispatch
```

### Campaign Flow

```
Create Campaign → Import Recipients → Start Campaign
  → Publish to Redis Queue → Workers Pick Up Jobs
  → Apply Delay → Send Message → Update Stats
  → Publish Stats → WebSocket Broadcast
```

### Authentication Flow

```
Login Request → Find User → Verify Password → Generate JWT Pair
  → Store Refresh Token in Redis → Set Cookies → Return User
```

## Configuration System

Configuration is loaded from TOML with environment variable overrides:

```toml
[app]
name = "whatomate"
encryption_key = "your-32-byte-key-here"

[server]
host = "0.0.0.0"
port = 8080
allowed_origins = ["http://localhost:5173"]

[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "secret"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "localhost"
port = 6379

[whatsapp]
provider = "meta"  # or "whatsmeow"
base_url = "https://graph.facebook.com/v18.0"

[jwt]
secret = "your-jwt-secret"
access_token_ttl = "15m"
refresh_token_ttl = "168h"
```

## See Also

- [API Reference](./api-reference)
- [Provider Abstraction](./provider-abstraction)
- [Database Models](./database-models)
- [Background Workers](./background-workers)
