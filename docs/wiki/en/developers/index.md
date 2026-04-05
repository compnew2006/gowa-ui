---
title: Developer Guide
---

# Developer Guide

Welcome to the Whatomate developer documentation. This guide covers everything you need to build, extend, and maintain the Whatomate WhatsApp Business API platform.

## What is Whatomate?

Whatomate is a multi-tenant WhatsApp Business API platform that supports two provider backends:

- **Meta Cloud API** — Official WhatsApp Business Cloud API with templates, flows, and catalogs
- **WhatsMeow** — Direct WhatsApp Web protocol connection via `go.mau.fi/whatsmeow`

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.21+, `valyala/fasthttp` |
| Database | PostgreSQL (via GORM) |
| Cache/Queue | Redis |
| Frontend | React 18, Vite, TailwindCSS |
| Real-time | WebSocket (gorilla/websocket) |
| Auth | JWT (HS256), HTTP-only cookies |
| Encryption | AES-256-GCM |

## Quick Start

```bash
# Clone and configure
git clone https://github.com/whatomate/whatomate.git
cd whatomate
cp config.example.toml config.toml

# Run migrations
go run cmd/whatomate/main.go -migrate

# Start server
go run cmd/whatomate/main.go
```

## Documentation Structure

| Page | Description |
|------|-------------|
| [Architecture](./architecture) | System architecture, tech stack, directory structure, data flows |
| [API Reference](./api-reference) | Complete REST API reference organized by resource |
| [Provider Abstraction](./provider-abstraction) | MessageProvider interface, Meta/WhatsMeow adapters |
| [WebSocket Events](./websocket-events) | Real-time event types and message formats |
| [Webhook Integration](./webhook-integration) | Outbound webhook system, event types, HMAC signing |
| [Background Workers](./background-workers) | Redis queue system, job types, consumer operations |
| [Database Models](./database-models) | All 30+ GORM models, relationships, schema |
| [Caching](./caching) | Redis cache system, TTL settings, key patterns |
| [Testing](./testing) | Unit tests, E2E tests, coverage, test helpers |
| [Contributing](./contributing) | Code style, PR process, adding features |

## Core Concepts

### Multi-Tenancy

Every resource is scoped to an organization. Users belong to organizations through `user_organizations` membership records and can switch between them.

### Provider Abstraction

The `MessageProvider` interface abstracts differences between Meta and WhatsMeow. All message sending goes through `SendOutgoingMessage()` which routes to the correct provider.

### Real-Time Architecture

- **WebSocket** for client-facing real-time updates (messages, status, notifications)
- **Redis Pub/Sub** for inter-process communication (campaign stats)
- **Redis Queues** for background job processing (campaigns, media downloads)

### Security Model

- JWT-based authentication with access/refresh token rotation
- RBAC with resource:action permission pairs
- AES-256-GCM encryption for sensitive fields
- CSRF protection via double-submit cookie
- SSRF-safe dialer for outbound HTTP requests

## See Also

- [Architecture Overview](./architecture)
- [API Reference](./api-reference)
- [Provider Abstraction](./provider-abstraction)
