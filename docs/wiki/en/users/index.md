---
title: User Guide
---

# User Guide

Welcome to the Whatomate User Guide. This section provides step-by-step instructions for using every feature of the Whatomate WhatsApp Business platform.

## Getting Started

Whatomate is a multi-tenant WhatsApp Business platform that supports two connection providers:

| Provider | Description |
|----------|-------------|
| **Meta Cloud API** | Official Meta WhatsApp Business Cloud API with template approval, Flows, and catalogs |
| **WhatsMeow** | Direct WhatsApp Web connection with QR code or phone-code pairing, no template approval needed |

Your organization's available features depend on which provider is configured.

## Guide Topics

### Account & Security

| Topic | Description |
|-------|-------------|
| [Authentication & User Settings](authentication.md) | Login, registration, SSO, password policy, availability, chat background |

### Contacts & Conversations

| Topic | Description |
|-------|-------------|
| [Contacts](contacts.md) | Managing contacts, search, filters, tags, assignment, collaborators |
| [Chat & Messaging](chat-messaging.md) | Sending messages, chat lifecycle, reactions, read receipts, typing indicators |

### Automation & Campaigns

| Topic | Description |
|-------|-------------|
| [Campaigns](campaigns.md) | Bulk messaging campaigns, recipient import, scheduling, retry failed |
| [Chatbot](chatbot.md) | Automated responses, keyword rules, AI integration, flows, agent transfers, SLA |

### Content & Templates

| Topic | Description |
|-------|-------------|
| [Templates & Flows](templates-flows.md) | WhatsApp message templates, Flows, catalogs, and products |
| [Canned Responses](canned-responses.md) | Pre-written responses for quick replies during conversations |

### Organization & Team

| Topic | Description |
|-------|-------------|
| [Teams & Roles](teams-roles.md) | Team management, roles, permissions (RBAC), custom roles |
| [Tags & Organization](tags-organization.md) | Tags, organization settings, lead requests, conversation notes, custom actions |

### Insights

| Topic | Description |
|-------|-------------|
| [Analytics](analytics.md) | Dashboard, message analytics, chatbot analytics, agent analytics, Meta insights |

## Key Concepts

### Chat States

Chats in Whatomate move through four states:

- **Open** — Active conversation, agents can send and receive messages
- **Closed** — Conversation ended, read-only until reopened
- **Pending** — Awaiting agent assignment
- **Claimed** — An agent has taken ownership of the chat

### Organizations

Whatomate supports multiple organizations per user account. You can switch between organizations using the organization switcher. Each organization has its own contacts, campaigns, chatbots, and team members.

### Roles

| Role | Capabilities |
|------|-------------|
| **Admin** | Full access to all features and settings |
| **Manager** | Manage team members, view analytics, handle escalations |
| **Agent** | Handle assigned chats, send messages, use canned responses |
| **Custom** | Defined by your organization with specific permissions |

## See Also

- [Teams & Roles](teams-roles.md) — Learn about permissions and access control
- [Organization Settings](tags-organization.md) — Configure your organization
