---
title: Platform Overview
---

# Platform Overview

Whatomate is a comprehensive WhatsApp Business API platform that enables organizations to manage customer communications at scale. It supports dual provider architectures — Meta Cloud API and direct WhatsApp Web (WhatsMeow) — with powerful features for messaging, campaigns, chatbot automation, and team collaboration.

## What is Whatomate?

Whatomate is a multi-tenant, self-hosted WhatsApp Business platform built with Go and React. It provides a unified interface for managing WhatsApp conversations regardless of which provider backend you choose.

## Key Capabilities

| Capability | Description |
|------------|-------------|
| **Messaging** | Send and receive text, media, template, and interactive messages |
| **Contacts** | Organize, search, filter, and assign contacts to team members |
| **Campaigns** | Bulk messaging with scheduling, delays, retry, and real-time analytics |
| **Chatbot** | Keyword rules, AI-powered responses, conversation flows, and agent transfers |
| **Teams** | Role-based access control, team management, and collaboration features |
| **Analytics** | Dashboard metrics, agent performance, chatbot stats, and custom widgets |
| **Integrations** | Outbound webhooks, custom actions, SSO authentication, and API access |
| **Security** | JWT auth, CSRF protection, AES-256-GCM encryption, send restrictions |

## Dual Provider Architecture

Whatomate abstracts WhatsApp connectivity through a `MessageProvider` interface, supporting two backends:

### Meta Cloud API

- Official Meta WhatsApp Business Cloud API
- Requires Meta Business account and app approval
- Supports templates, Flows, catalogs, and products
- Webhook-based message receiving
- Template messages require Meta approval before sending
- Best for: Enterprises needing official API compliance

### WhatsMeow (Direct WhatsApp Web)

- Uses `go.mau.fi/whatsmeow` library for direct WhatsApp Web protocol
- QR code or phone-code pairing for authentication
- No template approval needed — send any message format
- Per-instance message queuing with rate limiting
- Direct WebSocket connection to WhatsApp servers
- Best for: Teams wanting flexibility without Meta approval delays

Provider selection is configured in `config.toml` under `whatsapp.provider`. The choice determines which features are available — templates, Flows, and catalogs are Meta-only features.

## Multi-Tenant Design

Whatomate supports multiple organizations on a single deployment:

- **Organization isolation**: Every resource (contacts, campaigns, chatbots) is scoped to an organization
- **User membership**: Users belong to organizations through `user_organizations` records
- **Organization switching**: Users with access to multiple organizations can switch context seamlessly
- **Independent settings**: Each organization has its own chatbot settings, roles, teams, and configurations
- **Soft-delete cascade**: Deleting an organization soft-deletes all related records

## Role-Based Access Control (RBAC)

Whatomate uses a granular permission system:

| Role | Default Permissions |
|------|-------------------|
| **Admin** | Full access to all features and settings |
| **Manager** | Team management, analytics, escalation handling |
| **Agent** | Handle assigned chats, send messages, use canned responses |
| **Custom** | Organization-defined with specific resource:action permissions |

Permissions are defined as `resource:action` pairs (e.g., `contacts:read`, `messages:write`, `campaigns:delete`) and cached in Redis for performance.

## Real-Time Architecture

Whatomate provides real-time updates through multiple channels:

- **WebSocket**: Client-facing real-time communication for messages, status updates, notifications, and campaign progress
- **Redis Pub/Sub**: Inter-process communication for campaign statistics and cross-instance coordination
- **Redis Queues**: Background job processing for campaigns and media downloads

## Security Model

| Layer | Mechanism |
|-------|-----------|
| Authentication | JWT (HS256) with access/refresh token rotation |
| Session | HTTP-only, Secure, SameSite=Strict cookies |
| Authorization | RBAC with resource:action permission checks |
| Data encryption | AES-256-GCM for sensitive fields (tokens, API keys, secrets) |
| CSRF protection | Double-submit cookie pattern |
| Rate limiting | Per-user and per-IP limits via Redis |
| SSRF prevention | Blocked IP ranges for outbound HTTP requests |
| Send restrictions | Organization and user-level message sending policies |

## Target Audience

Whatomate is designed for:

- **Customer support teams** managing WhatsApp conversations at scale
- **Marketing teams** running bulk messaging campaigns
- **Developers** building WhatsApp-integrated applications
- **Businesses** needing multi-agent WhatsApp support without third-party SaaS

## When to Use Whatomate

| Scenario | Why Whatomate |
|----------|--------------|
| Multi-agent WhatsApp support | Team collaboration, assignment, and real-time chat |
| Bulk messaging campaigns | Scheduled campaigns with delays and analytics |
| Automated customer service | Chatbot with keywords, AI, and flows |
| Self-hosted WhatsApp API | Full control over data and infrastructure |
| Multi-organization management | Single deployment serving multiple teams/brands |

## When NOT to Use Whatomate

- You need a simple personal WhatsApp automation tool (use a lighter solution)
- You require official Meta BSP partnership features not yet supported
- You need SMS or other messaging channels (Whatomate is WhatsApp-only)

## See Also

- [Quick Start Guide](quickstart.md) — Get up and running in minutes
- [User Guide](users/index.md) — End-user documentation
- [Developer Guide](developers/index.md) — Architecture and API reference
- [Admin Guide](admins/index.md) — Deployment and operations
- [FAQ](faq.md) — Frequently asked questions
