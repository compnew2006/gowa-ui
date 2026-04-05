---
title: API Reference
---

# API Reference

Complete REST API reference for Whatomate, organized by resource. All endpoints are prefixed with the base URL (e.g., `http://localhost:8080/api`).

## Authentication

All authenticated endpoints require either:
- HTTP-only cookies (`whm_access` for access token)
- Bearer token in `Authorization` header
- API key in `X-API-Key` header

### Login

```
POST /api/auth/login
```

**Request Body:**
```json
{
  "email": "admin@example.com",
  "password": "securepassword"
}
```

**Response (200):**
```json
{
  "expires_in": 900,
  "user": {
    "id": 1,
    "email": "admin@example.com",
    "full_name": "Admin User",
    "role": "admin",
    "is_active": true
  }
}
```

**Error Codes:**
| Code | Status | Description |
|------|--------|-------------|
| `invalid_credentials` | 401 | Email or password incorrect |
| `account_disabled` | 403 | User account is inactive |

### Register

```
POST /api/auth/register
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "Str0ng!Pass",
  "full_name": "New User",
  "invitation_token": "jwt-token-here"
}
```

**Response (200):**
```json
{
  "message": "Registration submitted. Please check your email."
}
```

### Refresh Token

```
POST /api/auth/refresh
```

**Response (200):** New token pair with cookies set.

### Logout

```
POST /api/auth/logout
```

### Switch Organization

```
POST /api/auth/switch-org
```

**Request Body:**
```json
{
  "organization_id": 2
}
```

### Get WebSocket Token

```
GET /api/auth/ws-token
```

**Response (200):**
```json
{
  "token": "short-lived-jwt-for-ws"
}
```

## Users

### List Users

```
GET /api/users
```

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `page` | int | Page number (default: 1) |
| `per_page` | int | Items per page (default: 20) |
| `search` | string | Search email/name |
| `status` | string | Filter: active, inactive |

**Response (200):**
```json
{
  "users": [...],
  "total": 50,
  "page": 1,
  "per_page": 20
}
```

### Create User

```
POST /api/users
```

**Request Body:**
```json
{
  "email": "newuser@example.com",
  "full_name": "New User",
  "role_id": 3,
  "is_active": true
}
```

### Update User

```
PUT /api/users/{id}
```

### Delete User

```
DELETE /api/users/{id}
```

### User Send Restrictions

```
GET  /api/users/{id}/send-restrictions
PUT  /api/users/{id}/send-restrictions
```

## Organizations

### List Organizations

```
GET /api/organizations
```

### Create Organization

```
POST /api/organizations
```

### Get Current Organization

```
GET /api/organizations/current
```

### Delete Organization

```
DELETE /api/organizations/{id}
```

### Organization Members

```
GET    /api/organizations/members
POST   /api/organizations/members
PUT    /api/organizations/members/{id}
DELETE /api/organizations/members/{id}
```

### Organization Settings

```
GET /api/org/settings
PUT /api/org/settings
```

## Roles & Permissions

### List Roles

```
GET /api/roles
```

### Create Role

```
POST /api/roles
```

**Request Body:**
```json
{
  "name": "Senior Agent",
  "is_default": false,
  "permissions": [
    {"resource": "contacts", "action": "read"},
    {"resource": "contacts", "action": "write"},
    {"resource": "messages", "action": "read"},
    {"resource": "messages", "action": "write"}
  ]
}
```

### Update Role

```
PUT /api/roles/{id}
```

### Delete Role

```
DELETE /api/roles/{id}
```

### List Permissions

```
GET /api/permissions
```

## API Keys

### List API Keys

```
GET /api/api-keys
```

### Create API Key

```
POST /api/api-keys
```

**Request Body:**
```json
{
  "name": "Integration Key",
  "permissions": ["contacts:read", "messages:write"],
  "expiry": "2026-12-31T23:59:59Z"
}
```

**Response (200):**
```json
{
  "id": 1,
  "name": "Integration Key",
  "key": "sk_live_abc123...",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### Delete API Key

```
DELETE /api/api-keys/{id}
```

## Accounts (WhatsApp Business)

### List Accounts

```
GET /api/accounts
```

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by status |
| `provider` | string | Filter by provider |
| `search` | string | Search by name |

### Create Account

```
POST /api/accounts
```

**Request Body:**
```json
{
  "name": "Production Account",
  "phone_number_id": "123456789",
  "access_token": "EAAB...",
  "business_account_id": "987654321",
  "webhook_verify_token": "my-secret-token"
}
```

### Update Account

```
PUT /api/accounts/{id}
```

### Delete Account

```
DELETE /api/accounts/{id}
```

### Test Account Connection

```
POST /api/accounts/{id}/test
```

### Subscribe App

```
POST /api/accounts/{id}/subscribe
```

### Business Profile

```
GET  /api/accounts/{id}/business_profile
PUT  /api/accounts/{id}/business_profile
POST /api/accounts/{id}/business_profile/photo
```

## Instances (WhatsMeow)

### List Instances

```
GET /api/instances
```

### Create Instance

```
POST /api/instances
```

**Request Body:**
```json
{
  "name": "support-instance",
  "is_default": true,
  "auto_read_receipt": true,
  "settings": {}
}
```

### Update Instance

```
PUT /api/instances/{id}
```

### Delete Instance

```
DELETE /api/instances/{id}
```

### Get Instance Health

```
GET /api/instances/{id}/health
```

**Response (200):**
```json
{
  "uptime": "24h",
  "messages_sent_today": 150,
  "messages_received_today": 200,
  "messages_failed_today": 2,
  "error_rate": 0.01,
  "queue_depth": 5
}
```

### Get QR Code

```
GET /api/instances/{id}/qr
```

### Connect Instance

```
POST /api/instances/{id}/connect
```

### Pair Phone

```
POST /api/instances/{id}/pair-phone
```

**Request Body:**
```json
{
  "phone_number": "+1234567890",
  "show_push_notification": true,
  "client_type": "android",
  "client_display_name": "Whatomate"
}
```

### Disconnect Instance

```
POST /api/instances/{id}/disconnect
```

### Reconnect Instance

```
POST /api/instances/{id}/reconnect
```

## Contacts

### List Contacts

```
GET /api/contacts
```

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `page` | int | Page number |
| `per_page` | int | Items per page |
| `search` | string | Search phone/name |
| `tags` | string | Filter by tags (has/all/any) |
| `assigned_to` | int | Filter by assigned user |
| `status` | string | Filter: open, closed, pending |

### Create Contact

```
POST /api/contacts
```

**Request Body:**
```json
{
  "phone_number": "+1234567890",
  "name": "John Doe",
  "tags": ["vip", "enterprise"],
  "metadata": {}
}
```

### Get Contact

```
GET /api/contacts/{id}
```

### Update Contact

```
PUT /api/contacts/{id}
```

### Delete Contact

```
DELETE /api/contacts/{id}
```

### Soft Delete Contact

```
POST /api/contacts/{id}/soft-delete
```

### Assign Contact

```
PUT /api/contacts/{id}/assign
```

**Request Body:**
```json
{
  "user_id": 5
}
```

### Contact Session Data

```
GET /api/contacts/{id}/session-data
```

## Messages

### List Messages

```
GET /api/chats/{id}/messages
```

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `before` | string | Cursor for pagination |
| `after` | string | Cursor for pagination |
| `type` | string | Filter by message type |
| `direction` | string | Filter: inbound, outbound |

### Send Message

```
POST /api/contacts/{id}/messages
```

**Request Body:**
```json
{
  "content": "Hello, how can I help you?",
  "account_id": 1,
  "reply_to_message_id": 123
}
```

**Response (200):**
```json
{
  "id": 456,
  "content": "Hello, how can I help you?",
  "direction": "outbound",
  "status": "pending",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### Send Media Message

```
POST /api/messages/media
```

### Send Template Message

```
POST /api/messages/template
```

**Request Body:**
```json
{
  "template_id": 1,
  "contact_id": 10,
  "parameters": {
    "name": "John",
    "order_id": "ORD-123"
  }
}
```

### Send Reaction

```
POST /api/contacts/{id}/messages/{message_id}/reaction
```

### Revoke Message

```
POST /api/contacts/{id}/messages/{message_id}/revoke
```

### Mark Message Read

```
PUT /api/messages/{id}/read
```

### Send Typing

```
POST /api/contacts/{id}/typing
```

## Campaigns

### List Campaigns

```
GET /api/campaigns
```

### Create Campaign

```
POST /api/campaigns
```

**Request Body:**
```json
{
  "name": "Holiday Promotion",
  "whatsapp_account": 1,
  "template_id": 5,
  "body_content": "Hi {{1}}, check out our deals!",
  "min_delay_seconds": 20,
  "max_delay_seconds": 45,
  "scheduled_at": "2026-12-01T09:00:00Z"
}
```

### Update Campaign

```
PUT /api/campaigns/{id}
```

### Delete Campaign

```
DELETE /api/campaigns/{id}
```

### Start Campaign

```
POST /api/campaigns/{id}/start
```

### Pause Campaign

```
POST /api/campaigns/{id}/pause
```

### Cancel Campaign

```
POST /api/campaigns/{id}/cancel
```

### Retry Failed

```
POST /api/campaigns/{id}/retry-failed
```

### Import Recipients

```
POST /api/campaigns/{id}/recipients/import
```

### Get Recipients

```
GET /api/campaigns/{id}/recipients
```

### Upload Campaign Media

```
POST /api/campaigns/{id}/media
```

## Chatbot

### Get Settings

```
GET /api/chatbot/settings
```

### Update Settings

```
PUT /api/chatbot/settings
```

**Request Body:**
```json
{
  "enabled": true,
  "greeting_message": "Welcome! How can I help?",
  "fallback_message": "I didn't understand. Let me connect you to an agent.",
  "session_timeout_minutes": 30,
  "business_hours": {
    "monday": {"open": "09:00", "close": "17:00"}
  },
  "ai_enabled": true,
  "ai_provider": "openai",
  "ai_model": "gpt-4",
  "ai_api_key": "sk-...",
  "ai_system_prompt": "You are a helpful assistant.",
  "sla_response_minutes": 15,
  "sla_resolution_minutes": 60,
  "sla_auto_close_hours": 24
}
```

### Keyword Rules

```
GET    /api/chatbot/keywords
POST   /api/chatbot/keywords
PUT    /api/chatbot/keywords/{id}
DELETE /api/chatbot/keywords/{id}
```

### Chatbot Flows

```
GET    /api/chatbot/flows
POST   /api/chatbot/flows
PUT    /api/chatbot/flows/{id}
DELETE /api/chatbot/flows/{id}
```

### AI Contexts

```
GET    /api/chatbot/ai-contexts
POST   /api/chatbot/ai-contexts
PUT    /api/chatbot/ai-contexts/{id}
DELETE /api/chatbot/ai-contexts/{id}
```

### Agent Transfers

```
GET    /api/chatbot/transfers
POST   /api/chatbot/transfers
POST   /api/chatbot/transfers/pick
PUT    /api/chatbot/transfers/{id}/assign
PUT    /api/chatbot/transfers/{id}/resume
```

## Templates (Meta)

### List Templates

```
GET /api/templates
```

### Create Template

```
POST /api/templates
```

### Update Template

```
PUT /api/templates/{id}
```

### Delete Template

```
DELETE /api/templates/{id}
```

### Sync Templates

```
POST /api/templates/sync
```

### Submit Template

```
POST /api/templates/{id}/publish
```

### Upload Template Media

```
POST /api/templates/upload-media
```

## Flows (Meta)

### List Flows

```
GET /api/flows
```

### Create Flow

```
POST /api/flows
```

### Update Flow

```
PUT /api/flows/{id}
```

### Delete Flow

```
DELETE /api/flows/{id}
```

### Save Flow to Meta

```
POST /api/flows/{id}/save-to-meta
```

### Publish Flow

```
POST /api/flows/{id}/publish
```

### Deprecate Flow

```
POST /api/flows/{id}/deprecate
```

### Duplicate Flow

```
POST /api/flows/{id}/duplicate
```

### Sync Flows

```
POST /api/flows/sync
```

## Catalogs (Meta)

### List Catalogs

```
GET /api/catalogs
```

### Create Catalog

```
POST /api/catalogs
```

### Delete Catalog

```
DELETE /api/catalogs/{id}
```

### Sync Catalogs

```
POST /api/catalogs/sync
```

### List Products

```
GET /api/catalogs/{id}/products
```

### Create/Update/Delete Product

```
POST   /api/catalogs/{id}/products
PUT    /api/products/{id}
DELETE /api/products/{id}
```

## Canned Responses

### List Canned Responses

```
GET /api/canned-responses
```

### Create Canned Response

```
POST /api/canned-responses
```

**Request Body:**
```json
{
  "shortcut": "/greeting",
  "content": "Hello! How can I help you today?",
  "category": "greetings"
}
```

### Update Canned Response

```
PUT /api/canned-responses/{id}
```

### Delete Canned Response

```
DELETE /api/canned-responses/{id}
```

### Send Canned Response

```
POST /api/canned-responses/{id}/send
```

### Increment Usage

```
POST /api/canned-responses/{id}/use
```

## Tags

### List Tags

```
GET /api/tags
```

### Create Tag

```
POST /api/tags
```

**Request Body:**
```json
{
  "name": "vip",
  "color": "#FFD700"
}
```

### Update Tag

```
PUT /api/tags/{name}
```

### Delete Tag

```
DELETE /api/tags/{name}
```

## Teams

### List Teams

```
GET /api/teams
```

### Create Team

```
POST /api/teams
```

**Request Body:**
```json
{
  "name": "Support Team",
  "description": "Primary support team",
  "member_ids": [1, 2, 3]
}
```

### Update Team

```
PUT /api/teams/{id}
```

### Delete Team

```
DELETE /api/teams/{id}
```

### Team Members

```
GET    /api/teams/{id}/members
POST   /api/teams/{id}/members
DELETE /api/teams/{id}/members/{user_id}
```

## Analytics

### Dashboard Stats

```
GET /api/analytics/dashboard
```

### Message Analytics

```
GET /api/analytics/messages
```

### Chatbot Analytics

```
GET /api/analytics/chatbot
```

### Agent Analytics

```
GET /api/analytics/agents
```

### Agent Comparison

```
GET /api/analytics/agents/comparison
```

### Agent Details

```
GET /api/analytics/agents/{id}
```

### Export Agent Ratings

```
GET /api/analytics/agents/ratings/export
```

### Meta Analytics

```
GET    /api/analytics/meta
POST   /api/analytics/meta/refresh
GET    /api/analytics/meta/accounts
```

## Webhooks (Outbound)

### List Webhooks

```
GET /api/webhooks
```

### Create Webhook

```
POST /api/webhooks
```

**Request Body:**
```json
{
  "url": "https://example.com/webhook",
  "events": ["message.received", "message.sent", "contact.created"],
  "secret": "hmac-secret",
  "enabled": true
}
```

### Update Webhook

```
PUT /api/webhooks/{id}
```

### Delete Webhook

```
DELETE /api/webhooks/{id}
```

### Test Webhook

```
POST /api/webhooks/{id}/test
```

## Custom Actions

### List Custom Actions

```
GET /api/custom-actions
```

### Create Custom Action

```
POST /api/custom-actions
```

### Update Custom Action

```
PUT /api/custom-actions/{id}
```

### Delete Custom Action

```
DELETE /api/custom-actions/{id}
```

### Execute Custom Action

```
POST /api/custom-actions/{id}/execute
```

## Conversation Notes

### List Notes

```
GET /api/contacts/{id}/notes
```

### Create Note

```
POST /api/contacts/{id}/notes
```

**Request Body:**
```json
{
  "content": "Customer requested a refund."
}
```

### Update Note

```
PUT /api/contacts/{id}/notes/{note_id}
```

### Delete Note

```
DELETE /api/contacts/{id}/notes/{note_id}
```

## SSO

### Get SSO Providers

```
GET /api/auth/sso/providers
```

### Initiate SSO

```
GET /api/auth/sso/{provider}/init
```

### SSO Callback

```
GET /api/auth/sso/{provider}/callback
```

### SSO Settings

```
GET    /api/settings/sso
PUT    /api/settings/sso
DELETE /api/settings/sso
```

## WebSockets

### Connect

```
GET /ws?token=<ws-token>
```

See [WebSocket Events](./websocket-events) for message types.

## Import/Export

### Export Data

```
POST /api/export
```

**Request Body:**
```json
{
  "table": "contacts",
  "filters": {"status": "open"},
  "format": "csv"
}
```

### Import Data

```
POST /api/import
```

### Export/Import Config

```
GET /api/export/{table}/config
GET /api/import/{table}/config
```

## Lead Requests

### Create Public Lead Request

```
POST /api/public/lead-requests
```

**Request Body:**
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "phone": "+1234567890",
  "message": "Interested in your product.",
  "widget_id": 1
}
```

### List Lead Requests

```
GET /api/lead-requests
```

### Update Lead Request Status

```
PUT /api/lead-requests/{id}/status
```

**Request Body:**
```json
{
  "status": "contacted"
}
```

## Activity Logs

### List Activity Logs

```
GET /api/activity-logs
```

**Query Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `user_id` | int | Filter by user |
| `action` | string | Filter by action |
| `resource` | string | Filter by resource |
| `from` | string | Start date |
| `to` | string | End date |

### Create Activity Log

```
POST /api/activity-logs
```

## Health & Readiness

### Health Check

```
GET /health
```

**Response (200):**
```json
{
  "status": "ok",
  "service": "whatomate"
}
```

### Readiness Check

```
GET /ready
```

**Response (200):**
```json
{
  "status": "ready"
}
```

**Response (500):**
```json
{
  "status": "not ready",
  "error": "database connection failed"
}
```

## Error Response Format

All errors follow a consistent format:

```json
{
  "error": {
    "message": "Human-readable error message",
    "code": "machine_readable_code",
    "field": "field_name"
  }
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 400 | Bad Request — Validation error |
| 401 | Unauthorized — Invalid or missing auth |
| 403 | Forbidden — Permission denied |
| 404 | Not Found — Resource doesn't exist |
| 409 | Conflict — Duplicate or business rule violation |
| 413 | Payload Too Large |
| 429 | Too Many Requests — Rate limited |
| 500 | Internal Server Error |

### Reason Codes

| Code | Description |
|------|-------------|
| `instance_not_found` | Instance doesn't exist |
| `instance_not_connected` | Instance is disconnected |
| `instance_not_allowed` | User can't use this instance |
| `chat_unclaimed` | Chat must be claimed before sending |
| `chat_closed` | Chat is closed and read-only |
| `restriction_violation` | Send restriction policy violated |

## See Also

- [Architecture](./architecture)
- [Provider Abstraction](./provider-abstraction)
- [WebSocket Events](./websocket-events)
- [Webhook Integration](./webhook-integration)
