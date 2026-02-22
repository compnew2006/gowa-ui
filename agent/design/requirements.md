# Project Requirements: Whatomate

**Project Name**: Whatomate
**Created**: 2026-02-20
**Status**: Active

---

## Overview

Whatomate is a modern, open-source WhatsApp Business Platform designed to support both the official **WhatsApp Cloud API (Meta)** and the **WhatsApp Web Protocol (whatsmeow)**. It aims to provide a single-binary application that is easy to deploy and feature-rich for businesses of all sizes.

---

## Problem Statement

Businesses need a reliable, cost-effective, and scalable way to manage WhatsApp communications. Existing solutions are either expensive, hard to self-host, or limited to a single protocol. Whatomate addresses these pain points by offering a multi-tenant, dual-provider architecture that is easy to manage and extend.

---

## Goals and Objectives

### Primary Goals
1. Provide seamless support for Meta Cloud API and Whatsmeow.
2. Implement a robust multi-tenant architecture with data isolation.
3. Offer a user-friendly interface for chat, template management, and campaigns.
4. Ensure easy deployment as a standalone binary.

### Secondary Goals
1. Implement advanced chatbot automation with AI integrations.
2. Provide granular roles and permissions for team management.
3. Enable real-time analytics and reporting.

---

## Functional Requirements

### Core Features
1. **Multi-tenancy**: Isolated data and configurations for multiple organizations.
2. **Dual Providers**: Support for Meta Cloud API and Whatsmeow QR-code based login.
3. **Real-time Chat**: Live messaging via WebSockets with instance identification.
4. **Template Management**: Create and sync templates with Meta.
5. **Bulk Campaigns**: Broadcast messages to contact lists.
6. **Chatbot Builder**: Keyword-based and flow-based automation.
7. **Chat Lifecycle & Activity Audit**: Manage conversation states and maintain a complete audit trail of user and system activities.

### Additional Features
1. **Canned Responses**: Quick replies with placeholders and slash commands.
2. **Analytics**: Dashboard for tracking engagement and performance.
3. **AI Integration**: Support for OpenAI, Anthropic, and Google AI for auto-replies.

---

## Technical Requirements

### Technology Stack
- **Backend Language**: Go (Fastglue)
- **Frontend Framework**: Vue.js 3, Tailwind CSS, shadcn-vue
- **Database**: PostgreSQL
- **Key-Value Store**: Redis (for queue and cache)
- **Deployment**: Single binary (Go + embedded frontend) or Docker

### Key Dependencies
- `whatsmeow`: WhatsApp Web protocol implementation.
- `fastglue`: High-performance Go web framework.
- `pgx`: PostgreSQL driver.
- `redis`: Redis client.

---

## Success Criteria

- [ ] Successful integration of Whatsmeow with multi-instance support.
- [ ] Multi-tenant isolation verified with security audits.
- [ ] Real-time messaging performance stable under load.
- [ ] Standalone binary builds and runs without external frontend dependencies.

---

**Status**: Active implementation following ACP.
**Last Updated**: 2026-02-20
