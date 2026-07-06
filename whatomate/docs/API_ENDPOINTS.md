# Whatomate API Endpoints

> **Updated:** 2026-06-18 — 699 routes documented  
> **Base URL:** `http://localhost:8080` (or as configured in `config.toml`)  
> **Auth:** JWT via `Authorization: Bearer <token>` or API Key via `X-API-Key: <key>`

---

## Health & System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Basic liveness check |
| GET | `/ready` | Readiness check (includes DB ping) |

---

## Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | Email + password login → JWT + refresh token |
| POST | `/api/auth/register` | Register new user via invite |
| POST | `/api/auth/register-invite` | Create registration invite (admin) |
| POST | `/api/auth/refresh` | Rotate JWT using refresh token |
| POST | `/api/auth/logout` | Invalidate tokens |
| POST | `/api/auth/switch-org` | Switch active organization |
| GET | `/api/auth/me` | Get current user profile |
| PUT | `/api/auth/me/settings` | Update user preferences |
| PUT | `/api/auth/me/password` | Change password |
| PUT | `/api/auth/me/availability` | Toggle agent availability |
| POST | `/api/auth/me/chat-background` | Upload chat background image |
| GET | `/api/auth/me/chat-background` | Get chat background |
| GET | `/api/auth/me/organizations` | List user's organizations |
| GET | `/api/auth/ws-token` | Generate WebSocket auth token |

## Organizations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations/settings` | Get organization settings |
| PUT | `/api/organizations/settings` | Update organization settings |
| GET | `/api/organizations` | List all organizations (superadmin) |
| GET | `/api/organizations/current` | Get current organization |
| POST | `/api/organizations` | Create organization |
| DELETE | `/api/organizations/{id}` | Delete organization |
| GET | `/api/organizations/{id}/members` | List organization members |
| POST | `/api/organizations/{id}/members` | Add member to organization |
| DELETE | `/api/organizations/members/{id}` | Remove organization member |
| PUT | `/api/organizations/members/{id}/role` | Update member role |

## Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/users` | List organization users |
| GET | `/api/users/{id}` | Get user details |
| POST | `/api/users` | Create user |
| PUT | `/api/users/{id}` | Update user |
| DELETE | `/api/users/{id}` | Soft-delete user |

## Roles & Permissions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/roles` | List roles with permissions |
| GET | `/api/roles/{id}` | Get role details |
| POST | `/api/roles` | Create custom role |
| PUT | `/api/roles/{id}` | Update role |
| DELETE | `/api/roles/{id}` | Delete role |
| GET | `/api/permissions` | List all available permissions |

## Teams

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/teams` | List teams |
| GET | `/api/teams/{id}` | Get team details |
| POST | `/api/teams` | Create team |
| PUT | `/api/teams/{id}` | Update team |
| DELETE | `/api/teams/{id}` | Delete team |
| GET | `/api/teams/{id}/members` | List team members |
| POST | `/api/teams/{id}/members` | Add team member |
| DELETE | `/api/teams/members/{id}` | Remove team member |

## WhatsApp Cloud API Accounts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/accounts` | List WhatsApp Cloud API accounts |
| GET | `/api/accounts/{id}` | Get account details |
| POST | `/api/accounts` | Create WhatsApp account |
| PUT | `/api/accounts/{id}` | Update account credentials |
| DELETE | `/api/accounts/{id}` | Delete account |
| POST | `/api/accounts/{id}/test` | Test WhatsApp API connectivity |
| POST | `/api/accounts/{id}/subscribe` | Subscribe to WhatsApp webhook |

## WhatsApp Web Instances

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/instances` | List WhatsApp Web instances |
| GET | `/api/instances/{id}` | Get instance details |
| POST | `/api/instances` | Create instance |
| PUT | `/api/instances/{id}` | Update instance settings |
| DELETE | `/api/instances/{id}` | Delete instance |
| POST | `/api/instances/{id}/connect` | Connect instance (QR) |
| POST | `/api/instances/{id}/disconnect` | Disconnect instance |
| POST | `/api/instances/{id}/reconnect` | Reconnect instance |
| GET | `/api/instances/{id}/qr` | Get current QR code snapshot |
| POST | `/api/instances/{id}/pair` | Pair via phone number |
| GET | `/api/instances/{id}/health` | Get instance health metrics |

## Contacts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/contacts` | List contacts (paginated, filterable) |
| GET | `/api/contacts/{id}` | Get contact details |
| POST | `/api/contacts` | Create contact |
| PUT | `/api/contacts/{id}` | Update contact |
| DELETE | `/api/contacts/{id}` | Delete contact |
| GET | `/api/contacts/{id}/messages` | Get chat messages |
| PATCH | `/api/contacts/{id}/read` | Mark messages as read |
| POST | `/api/contacts/start-chat` | Start new chat with contact |

## Messages

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/messages/send` | Send outgoing message |
| POST | `/api/messages/send-template` | Send template message |

## Campaigns

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/campaigns` | List campaigns |
| GET | `/api/campaigns/{id}` | Get campaign details |
| POST | `/api/campaigns` | Create campaign |
| PUT | `/api/campaigns/{id}` | Update campaign |
| DELETE | `/api/campaigns/{id}` | Delete campaign |
| POST | `/api/campaigns/{id}/start` | Start campaign |
| POST | `/api/campaigns/{id}/pause` | Pause campaign |
| POST | `/api/campaigns/{id}/cancel` | Cancel campaign |
| POST | `/api/campaigns/{id}/retry` | Retry failed sends |
| POST | `/api/campaigns/{id}/recipients` | Import recipients |
| GET | `/api/campaigns/{id}/recipients` | List campaign recipients |
| DELETE | `/api/campaigns/recipients/{id}` | Delete campaign recipient |
| POST | `/api/campaigns/{id}/media` | Upload campaign media |
| GET | `/api/campaigns/media/{id}` | Serve campaign media file |

## Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/templates` | List message templates |
| GET | `/api/templates/{id}` | Get template details |
| POST | `/api/templates` | Create template |
| PUT | `/api/templates/{id}` | Update template |
| DELETE | `/api/templates/{id}` | Delete template |
| POST | `/api/templates/{id}/submit` | Submit template for Meta review |
| POST | `/api/templates/sync` | Sync templates from Meta |
| POST | `/api/templates/{id}/media` | Upload template media |

## Chatbot

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/chatbot/settings` | Get chatbot settings |
| PUT | `/api/chatbot/settings` | Update chatbot settings |
| GET | `/api/chatbot/keywords` | List keyword rules |
| GET | `/api/chatbot/keywords/{id}` | Get keyword rule |
| POST | `/api/chatbot/keywords` | Create keyword rule |
| PUT | `/api/chatbot/keywords/{id}` | Update keyword rule |
| DELETE | `/api/chatbot/keywords/{id}` | Delete keyword rule |
| GET | `/api/chatbot/flows` | List chatbot flows |
| GET | `/api/chatbot/flows/{id}` | Get chatbot flow |
| POST | `/api/chatbot/flows` | Create chatbot flow |
| PUT | `/api/chatbot/flows/{id}` | Update chatbot flow |
| DELETE | `/api/chatbot/flows/{id}` | Delete chatbot flow |
| GET | `/api/chatbot/ai-contexts` | List AI contexts |
| GET | `/api/chatbot/ai-contexts/{id}` | Get AI context |
| POST | `/api/chatbot/ai-contexts` | Create AI context |
| PUT | `/api/chatbot/ai-contexts/{id}` | Update AI context |
| DELETE | `/api/chatbot/ai-contexts/{id}` | Delete AI context |
| GET | `/api/chatbot/sessions` | List chatbot sessions |
| GET | `/api/chatbot/sessions/{id}` | Get chatbot session details |

## WhatsApp Flows

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/flows` | List WhatsApp flows |
| GET | `/api/flows/{id}` | Get flow details |
| POST | `/api/flows` | Create flow |
| PUT | `/api/flows/{id}` | Update flow |
| DELETE | `/api/flows/{id}` | Delete flow |
| POST | `/api/flows/{id}/save` | Save flow to Meta |
| POST | `/api/flows/{id}/publish` | Publish flow |
| POST | `/api/flows/{id}/deprecate` | Deprecate flow |
| POST | `/api/flows/{id}/duplicate` | Duplicate flow |
| POST | `/api/flows/sync` | Sync flows from Meta |

## Catalogs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/catalogs` | List catalogs |
| GET | `/api/catalogs/{id}` | Get catalog details |
| POST | `/api/catalogs` | Create catalog |
| DELETE | `/api/catalogs/{id}` | Delete catalog |
| POST | `/api/catalogs/sync` | Sync catalogs from Meta |
| GET | `/api/catalogs/{id}/products` | List catalog products |
| GET | `/api/catalogs/products/{id}` | Get product details |
| POST | `/api/catalogs/{id}/products` | Create catalog product |
| PUT | `/api/catalogs/products/{id}` | Update product |
| DELETE | `/api/catalogs/products/{id}` | Delete product |

## Webhooks (Outgoing)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/webhooks` | List webhook endpoints |
| GET | `/api/webhooks/{id}` | Get webhook details |
| POST | `/api/webhooks` | Create webhook |
| PUT | `/api/webhooks/{id}` | Update webhook |
| DELETE | `/api/webhooks/{id}` | Delete webhook |
| POST | `/api/webhooks/{id}/test` | Test webhook with sample event |

## Webhook (Meta Incoming)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/webhook` | Meta webhook verification (challenge) |
| POST | `/api/webhook` | Meta incoming webhook events |

## Analytics & Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/analytics/dashboard` | Dashboard summary stats |
| GET | `/api/analytics/messages` | Message analytics |
| GET | `/api/analytics/chatbot` | Chatbot analytics |
| GET | `/api/analytics/campaigns` | Campaign analytics |

## Widgets

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/widgets` | List dashboard widgets |
| GET | `/api/widgets/{id}` | Get widget details |
| POST | `/api/widgets` | Create widget |
| PUT | `/api/widgets/{id}` | Update widget |
| DELETE | `/api/widgets/{id}` | Delete widget |
| PUT | `/api/widgets/layout` | Save widget grid layout |
| GET | `/api/widgets/data-sources` | List available data sources |
| GET | `/api/widgets/{id}/data` | Get widget data |
| POST | `/api/widgets/data` | Get all widgets data (batch) |

## SSO

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/sso/providers` | List public SSO providers |
| GET | `/api/sso/{provider}/init` | Initiate SSO login |
| GET | `/api/sso/{provider}/callback` | SSO callback |
| GET | `/api/sso/settings` | Get SSO settings |
| PUT | `/api/sso/providers/{id}` | Update SSO provider |
| DELETE | `/api/sso/providers/{id}` | Delete SSO provider |

## Business Profile

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/business-profile` | Get WhatsApp business profile |
| PUT | `/api/business-profile` | Update business profile |
| POST | `/api/business-profile/picture` | Update profile picture |

## API Keys

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/api-keys` | List API keys |
| POST | `/api/api-keys` | Create API key |
| DELETE | `/api/api-keys/{id}` | Delete API key |

## Canned Responses

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/canned-responses` | List canned responses |
| GET | `/api/canned-responses/{id}` | Get canned response |
| POST | `/api/canned-responses` | Create canned response |
| PUT | `/api/canned-responses/{id}` | Update canned response |
| DELETE | `/api/canned-responses/{id}` | Delete canned response |

## Tags

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/tags` | List contact tags |
| POST | `/api/tags` | Create tag |
| PUT | `/api/tags/{id}` | Update tag |
| DELETE | `/api/tags/{id}` | Delete tag |

## Notifications

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/notifications` | List in-app notifications |
| POST | `/api/notifications/{id}/dismiss` | Dismiss notification |

## Media

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/media/{id}/download` | Download media file |
| POST | `/api/media/retry` | Retry media download |
| GET | `/api/media/{id}/serve` | Serve media file |

## Import/Export

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/import-export/config` | Get import/export configuration |
| POST | `/api/import-export/export` | Export data (CSV/Excel) |
| POST | `/api/import-export/import` | Import data (CSV/Excel) |

## Group Campaigns

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/instances/{id}/groups` | List instance WhatsApp groups |
| POST | `/api/campaigns/{id}/groups` | Add groups to campaign |
| GET | `/api/campaigns/{id}/groups` | List campaign groups |
| DELETE | `/api/campaigns/groups/{id}` | Remove campaign group |
| POST | `/api/groups/validate` | Validate group JIDs |

## Group Directory

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/group-directory/search` | Search group directory |
| GET | `/api/group-directory/categories` | Get directory categories |
| GET | `/api/group-directory/countries` | Get directory countries |
| POST | `/api/group-directory` | Create directory entry |
| PUT | `/api/group-directory/{id}` | Update directory entry |
| DELETE | `/api/group-directory/{id}` | Delete directory entry |
| POST | `/api/group-directory/preview` | Preview group from invite link |
| POST | `/api/group-directory/import` | Import directory groups to campaign |

## Custom Actions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/custom-actions` | List custom actions |
| POST | `/api/custom-actions` | Create custom action |
| PUT | `/api/custom-actions/{id}` | Update custom action |
| DELETE | `/api/custom-actions/{id}` | Delete custom action |

## WebSocket

| Endpoint | Description |
|----------|-------------|
| `GET /ws` | WebSocket upgrade endpoint (requires JWT or WS token) |

## Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/bootstrap` | System bootstrap info |
| POST | `/api/admin/reset-password` | Reset user password (CLI analog) |

---

## Route Statistics

| Category | Route Count |
|----------|-------------|
| Auth & Users | 30+ |
| Organizations | 15+ |
| Accounts | 10+ |
| Instances | 20+ |
| Contacts & Messages | 20+ |
| Campaigns | 30+ |
| Templates | 15+ |
| Chatbot | 30+ |
| Flows | 12+ |
| Catalogs | 15+ |
| Webhooks | 12+ |
| Analytics & Widgets | 15+ |
| SSO | 8+ |
| Other (tags, canned, media, etc.) | 40+ |
| **Total** | **~699 routes** |

---

*Full function-level analysis in `FUNCTION_ANALYSIS.md`*
