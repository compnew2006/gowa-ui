---
title: Admin Guide
---

# Admin Guide

This guide covers operational tasks for managing a Whatomate deployment, including configuration, deployment, security, monitoring, troubleshooting, and data management.

## Table of Contents

| Page | Description |
|------|-------------|
| [Configuration](configuration.md) | TOML configuration, environment variables, encryption keys, JWT secrets |
| [Deployment](deployment.md) | Build process, Docker deployment, production checklist |
| [Security](security.md) | JWT auth, CSRF, encryption, send restrictions, security headers, rate limiting |
| [Monitoring](monitoring.md) | Health checks, readiness, logging, metrics |
| [Troubleshooting](troubleshooting.md) | Common issues and resolutions |
| [Data Migration](data-migration.md) | Database migrations, crypto migration, encryption versions |
| [Backup & Recovery](backup-recovery.md) | Backup strategies, disaster recovery, organization deletion |

## Quick Start

1. **Install** — Build from source or use Docker
2. **Configure** — Create `config.toml` with database and Redis settings
3. **Migrate** — Run `whatomate -migrate` to set up the database
4. **Start** — Run `whatomate serve` to start the server
5. **Login** — Use the default admin credentials from config

## Architecture Overview

Whatomate is a single-binary application that embeds:

- **Go HTTP Server** (fasthttp) — REST API and WebSocket
- **Embedded Frontend** (React/Vite) — SPA served from the binary
- **PostgreSQL Database** — Primary data store
- **Redis** — Caching, queuing, pub/sub, session management
- **WhatsApp Providers** — Meta Cloud API or WhatsMeow (direct WhatsApp Web)

```
┌─────────────────────────────────────────────┐
│              Whatomate Binary               │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │   HTTP   │  │  Static  │  │ WebSocket │  │
│  │   API    │  │  Files   │  │   Hub     │  │
│  └────┬─────┘  └──────────┘  └─────┬─────┘  │
│       │                             │        │
│  ┌────┴─────────────────────────────┴─────┐  │
│  │          Handler Layer                  │  │
│  │  (Auth, RBAC, Middleware, Business Log) │  │
│  └────┬─────────────────────────────┬─────┘  │
│       │                             │        │
│  ┌────┴─────┐                  ┌────┴──────┐ │
│  │ Provider │                  │  Worker   │ │
│  │ (Meta/WM)│                  │  (Queue)  │ │
│  └──────────┘                  └───────────┘ │
└─────────────────────────────────────────────┘
         │                           │
    ┌────┴────┐               ┌──────┴──────┐
    │PostgreSQL│               │    Redis    │
    └─────────┘               └─────────────┘
```

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 2 GB | 4+ GB |
| Disk | 10 GB | 50+ GB (for media storage) |
| PostgreSQL | 13+ | 15+ |
| Redis | 6+ | 7+ |

## Key Operational Concepts

- **Multi-tenant**: Multiple organizations share one instance
- **Provider-based**: Choose Meta Cloud API or WhatsMeow per deployment
- **Soft-delete**: All deletions are soft (preserves data with `deleted_at`)
- **Encrypted secrets**: Sensitive fields use AES-256-GCM encryption
- **Background workers**: SLA processing, campaign delivery, activity retention run as goroutines

## See Also

- [Developer Guide](../developers/index.md) — Architecture, API, and development patterns
- [User Guide](../users/index.md) — End-user documentation
