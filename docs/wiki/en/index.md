# Welcome to Whatomate Documentation

**Whatomate** is a comprehensive WhatsApp Business API platform that enables organizations to manage customer communications at scale. It supports dual provider architectures — Meta Cloud API and direct WhatsApp Web (WhatsMeow) — with powerful features for messaging, campaigns, chatbot automation, and team collaboration.

---

## Choose Your Path

### For Users

Learn how to use Whatomate for daily operations:

- [Authentication & Login](users/authentication.md) — Sign in, manage your profile, and switch organizations
- [Managing Contacts](users/contacts.md) — Add, organize, and manage your customer contacts
- [Chat & Messaging](users/chat-messaging.md) — Send messages, manage chats, and use interactive features
- [Campaigns](users/campaigns.md) — Create and manage bulk messaging campaigns
- [Chatbot & Automation](users/chatbot.md) — Set up automated responses and AI-powered chatbots
- [Templates & Flows](users/templates-flows.md) — Create message templates and interactive flows
- [Analytics & Reports](users/analytics.md) — Monitor performance and generate reports
- [Canned Responses](users/canned-responses.md) — Use pre-written responses for quick replies
- [Teams & Roles](users/teams-roles.md) — Collaborate with teams and manage permissions
- [Tags & Organization](users/tags-organization.md) — Organize contacts with tags and labels

### For Developers

Understand the architecture and extend Whatomate:

- [Architecture](developers/architecture.md) — System design, tech stack, and component overview
- [API Reference](developers/api-reference.md) — Complete REST API documentation
- [Provider Abstraction](developers/provider-abstraction.md) — Meta and WhatsMeow provider patterns
- [WebSocket Events](developers/websocket-events.md) — Real-time event types and payloads
- [Webhook Integration](developers/webhook-integration.md) — Outbound webhook configuration
- [Background Workers](developers/background-workers.md) — Queue system and worker processing
- [Database & Models](developers/database-models.md) — Schema, models, and relationships
- [Caching System](developers/caching.md) — Redis caching strategies and TTL configuration
- [Testing](developers/testing.md) — Unit tests, integration tests, and E2E testing
- [Contributing](developers/contributing.md) — How to contribute to the codebase

### For System Administrators

Deploy, configure, and maintain Whatomate:

- [Configuration](admins/configuration.md) — Configuration options and environment variables
- [Deployment](admins/deployment.md) — Deployment guides for various environments
- [Security](admins/security.md) — Security features, hardening, and best practices
- [Monitoring](admins/monitoring.md) — Health checks, logging, and observability
- [Troubleshooting](admins/troubleshooting.md) — Common issues and their solutions
- [Data Migration](admins/data-migration.md) — Database migrations and data management
- [Backup & Recovery](admins/backup-recovery.md) — Backup strategies and disaster recovery

---

## Key Features

| Feature | Description |
|---------|-------------|
| Dual Provider | Meta Cloud API or direct WhatsApp Web (WhatsMeow) |
| Multi-tenant | Organization-based isolation with RBAC |
| Campaigns | Bulk messaging with scheduling and analytics |
| Chatbot | Keyword rules, AI integration, and conversation flows |
| Real-time | WebSocket-based live updates and notifications |
| Security | JWT auth, CSRF protection, encryption at rest |
| Analytics | Dashboard, agent metrics, and custom widgets |
| Extensible | Outbound webhooks and custom actions |

---

## Quick Links

- **Source Code**: [GitHub Repository](https://github.com/whatomate/whatomate)
- **API Endpoints**: [API Reference](developers/api-reference.md)
- **Configuration**: [Admin Configuration](admins/configuration.md)
- **FAQ**: [Frequently Asked Questions](faq.md)
- **Release Notes**: [Changelog](release-notes.md)

---

## Getting Help

- Check the [FAQ](faq.md) for common questions
- Review [Troubleshooting](admins/troubleshooting.md) for known issues
- Open an issue on [GitHub](https://github.com/whatomate/whatomate/issues) for bugs or feature requests
