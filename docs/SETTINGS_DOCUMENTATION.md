# Settings Route Documentation — `/settings` & `/settings/*`

> Last updated: 2026-04-29  
> Project: Whatomate  
> Module: Settings & Configuration

---

## Table of Contents

1. [Route Overview](#route-overview)
2. [Architecture](#architecture)
3. [Route Map](#route-map)
4. [Sub-Route Details](#sub-route-details)
5. [Permission Model](#permission-model)
6. [API Reference](#api-reference)
7. [Frontend Components](#frontend-components)
8. [Data Flow](#data-flow)
9. [Configuration](#configuration)

---

## Route Overview

The `/settings` path is a major section of the Whatomate application providing organization-level and user-level configuration. It contains **18 sub-routes** spanning organization management, user/role/team administration, WhatsApp instance configuration, chatbot settings, integrations, and system administration.

### Entry Requirements

- **Authentication**: All `/settings/*` routes require authenticated users
- **Route Guard**: Vue Router `beforeEach` guard checks `authStore.isAuthenticated`
- **License Guard**: Locked-license state blocks all routes except `license-cleanup` and `activate`
- **Organization Header**: `X-Organization-ID` header sent for multi-tenant scoping

---

## Architecture

```
/settings                                  Organization + User preferences (4 tabs)
/settings/chatbot                          Chatbot configuration (5 tabs)
/settings/accounts                         Meta Cloud API accounts (CRUD)
/settings/instances                        WhatsApp instances (CRUD + lifecycle)
/settings/instances/health                 Instance health dashboard
/settings/canned-responses                 Message templates with attachments
/settings/contacts                         Contact management + import/export
/settings/closed-chats                     Closed chat history + reopen
/settings/tags                             Organization tags (CRUD)
/settings/users                            User management + send restrictions
/settings/roles                            Role + permission matrix
/settings/teams                            Team management + member assignment
/settings/api-keys                         API key management (create/delete)
/settings/webhooks                         Webhook configuration + testing
/settings/sso                              SSO provider configuration
/settings/license                          License status + activation
/settings/custom-actions                   Custom action CRUD (webhook/URL/JS)
```

### Frontend Stack
- **Framework**: Vue 3 Composition API + `<script setup>`
- **State**: Pinia stores (`auth`, `users`, `roles`, `teams`, `instances`, `contacts`, `tags`, `license`, `config`, `organizations`)
- **UI Library**: shadcn-vue (new-york style) + Tailwind CSS v3
- **API Client**: Axios instance in `frontend/src/services/api.ts`
- **i18n**: vue-i18n with en, es, ar locales

### Backend Stack
- **HTTP**: fasthttp + fastglue (NOT net/http)
- **ORM**: GORM + PostgreSQL 17
- **Queue**: Redis Streams
- **Auth**: JWT + API key, CSRF double-submit cookie
- **Multi-tenancy**: TenantScope middleware + `X-Organization-ID` header

---

## Route Map

### Frontend Router Definitions

| Path | Name | Component | Permission | Notes |
|------|------|-----------|------------|-------|
| `/settings` | `settings` | `SettingsView.vue` | `settings.general` OR `settings.uploads_cleanup` | 4-tab layout |
| `/settings/chatbot` | `chatbot-settings` | `ChatbotSettingsView.vue` | `settings.chatbot` | 5-tab panel |
| `/settings/accounts` | `accounts` | `AccountsView.vue` | `accounts` (metaOnly) | Meta Cloud API only |
| `/settings/instances` | `instances` | `InstancesView.vue` | `accounts` | WhatsApp instances |
| `/settings/instances/health` | `instances-health` | `InstanceHealthView.vue` | `accounts` | Health dashboard |
| `/settings/canned-responses` | `canned-responses` | `CannedResponsesView.vue` | `canned_responses` | Message templates |
| `/settings/contacts` | `contacts` | `ContactsView.vue` | `contacts` | Contact management |
| `/settings/closed-chats` | `closed-chats` | `ClosedChatsView.vue` | `chat` | Closed chat list |
| `/settings/tags` | `tags` | `TagsView.vue` | `tags` | Tag management |
| `/settings/users` | `users` | `UsersView.vue` | `users` | User management |
| `/settings/roles` | `roles` | `RolesView.vue` | `roles` | Role + permissions |
| `/settings/teams` | `teams` | `TeamsView.vue` | `teams` | Team management |
| `/settings/api-keys` | `api-keys` | `APIKeysView.vue` | `api_keys` | API key management |
| `/settings/webhooks` | `webhooks` | `WebhooksView.vue` | `webhooks` | Webhook management |
| `/settings/sso` | `sso-settings` | `SSOSettingsView.vue` | `settings.sso` | SSO configuration |
| `/settings/license` | `license-settings` | `LicenseSettingsView.vue` | `adminOnly` | License admin |
| `/settings/custom-actions` | `custom-actions` | `CustomActionsView.vue` | `custom_actions` | Custom actions |

### Additional Settings Views (not in `/settings` router path but in settings directory)

| Component | Used In | Purpose |
|-----------|---------|---------|
| `PendingChatsView.vue` | Separate route | Unclaimed/pending chats |
| `AssignedChatsView.vue` | Separate route | Assigned chat queue |
| `BusinessProfileDialog.vue` | AccountsView | Business profile editor |
| `FlowsView.vue` | `/templates` route | WhatsApp template flows |
| `CampaignsView.vue` | `/campaigns` route | Campaign management |
| `TemplatesView.vue` | `/templates` route | WhatsApp templates |
| `LicenseCleanupView.vue` | `/license-cleanup` route | License cleanup (admin) |

---

## Sub-Route Details

### 1. `/settings` — General Settings

**Component**: `SettingsView.vue` (~1000+ lines)  
**Permission**: `settings.general` or `settings.uploads_cleanup`

#### Tabs

| Tab | Scope | Features |
|-----|-------|----------|
| **General** | Organization | Name, slug, timezone, date format, phone masking, strict sending, outbound mode, campaign draft-only, strict rollout |
| **Appearance** | User | Theme mode (light/dark/system), theme preset colors |
| **Chat** | User | Media group window, sidebar view mode, chat background (default/images/patterns/upload), print/download buttons |
| **Notifications** | User | Email notifications, new message alerts, notification sound (10 options + preview), campaign updates |

#### API Endpoints
- `GET /api/org/settings` — Fetch organization settings
- `PUT /api/org/settings` — Update organization settings
- `POST /api/org/uploads-cleanup/run` — Run uploads cleanup immediately
- `GET /api/me` — Fetch current user settings
- `PUT /api/me/settings` — Update current user settings (appearance, notifications, chat background)
- `POST /api/me/chat-background` — Upload custom chat background image

#### Key Features
- Uploads cleanup configuration (retention days, schedule hour, run now)
- Strict sending restrictions with rollout mode (audit/enforce)
- Phone number masking for privacy
- Outbound mode (inbound_only/mixed)
- Campaign draft-only mode
- 10 notification sound options with audio preview

---

### 2. `/settings/chatbot` — Chatbot Settings

**Component**: `ChatbotSettingsView.vue`  
**Permission**: `settings.chatbot`

#### Tabs

| Tab | Features |
|-----|----------|
| **Messages** | Greeting message with buttons, fallback message with buttons, session timeout (5-120 min) |
| **Agents** | Queue pickup delay, same-agent assignment preference, current-conversation-only mode |
| **Business Hours** | Per-day schedule (enable/disable, start/end times), out-of-hours message |
| **SLA** | Response timer, escalation timer, resolution timer, auto-close, client inactivity reminders |
| **AI** | AI provider, model, API key, system prompt for automated responses |

#### API Endpoints
- `GET /api/chatbot/settings` — Fetch chatbot settings
- `PUT /api/chatbot/settings` — Update chatbot settings

---

### 3. `/settings/accounts` — Meta Cloud API Accounts

**Component**: `AccountsView.vue`  
**Permission**: `accounts` (metaOnly = true, requires Meta provider)

#### Features
- Account CRUD (name, phone_id, business_id, access_token, app_secret, api_version, webhook_verify_token)
- Connection testing
- Webhook subscription
- Business profile editing (via `BusinessProfileDialog.vue`)
- Setup guide display
- Delete with optional chat cleanup

#### API Endpoints
- `GET /api/accounts` — List accounts
- `POST /api/accounts` — Create account
- `GET /api/accounts/{id}` — Get account
- `PUT /api/accounts/{id}` — Update account
- `DELETE /api/accounts/{id}` — Delete account
- `POST /api/accounts/{id}/test` — Test connection
- `POST /api/accounts/{id}/subscribe` — Subscribe to webhook events
- `GET /api/accounts/{id}/business_profile` — Get business profile
- `PUT /api/accounts/{id}/business_profile` — Update business profile
- `POST /api/accounts/{id}/business_profile/photo` — Update profile picture

---

### 4. `/settings/instances` — WhatsApp Instances

**Component**: `InstancesView.vue`  
**Permission**: `accounts`

#### Features
- Instance CRUD (create, rename, delete)
- Connection lifecycle (connect, disconnect, reconnect)
- QR code scanning with WebSocket-driven updates
- Phone pairing code
- Instance settings: tag customization, auto-sync, auto-download media, auto-reject calls
- Auto-campaign scheduling per instance
- Chat-close rating configuration
- Assigned-chat reset settings
- Health dashboard link

#### API Endpoints
- `GET /api/instances` — List instances
- `POST /api/instances` — Create instance
- `GET /api/instances/{id}` — Get instance
- `PUT /api/instances/{id}` — Update instance
- `DELETE /api/instances/{id}` — Delete instance
- `GET /api/instances/{id}/health` — Get instance health
- `GET /api/instances/{id}/qr` — Get QR code snapshot
- `POST /api/instances/{id}/connect` — Connect instance
- `POST /api/instances/{id}/pair-phone` — Pair via phone number
- `POST /api/instances/{id}/disconnect` — Disconnect instance
- `POST /api/instances/{id}/reconnect` — Reconnect instance
- `POST /api/instances/{id}/auto-campaign/media` — Upload auto-campaign media

---

### 5. `/settings/instances/health` — Instance Health Dashboard

**Component**: `InstanceHealthView.vue` (thin wrapper around `HealthDashboard` component)  
**Permission**: `accounts`

#### Features
- Aggregated health status for all instances
- Individual instance health metrics

---

### 6. `/settings/canned-responses` — Canned Responses

**Component**: `CannedResponsesView.vue`  
**Permission**: `canned_responses`

#### Features
- CRUD for message templates with rich text editor
- Shortcut support for quick access
- Category organization
- File attachments (image/video, up to 16MB)
- Usage count tracking
- Search and category filter

#### API Endpoints
- `GET /api/canned-responses` — List canned responses
- `POST /api/canned-responses` — Create (multipart)
- `GET /api/canned-responses/{id}` — Get
- `PUT /api/canned-responses/{id}` — Update (multipart)
- `DELETE /api/canned-responses/{id}` — Delete
- `POST /api/canned-responses/{id}/send` — Send via chat
- `POST /api/canned-responses/{id}/use` — Increment usage count

---

### 7. `/settings/contacts` — Contact Management

**Component**: `ContactsView.vue`  
**Permission**: `contacts`

#### Features
- Contact list with search, instance filtering, tag assignment
- WhatsApp account association
- Import/export dialog
- Direct chat navigation
- Contact creation dialog

#### API Endpoints
- `GET /api/contacts` — List contacts
- `PUT /api/contacts/{id}` — Update contact
- `DELETE /api/contacts/{id}` — Delete contact
- `GET /api/tags` — List tags (for tag assignment)
- `GET /api/accounts` — List accounts (for instance filter)
- Import/export via `dataService`

---

### 8. `/settings/closed-chats` — Closed Chat History

**Component**: `ClosedChatsView.vue`  
**Permission**: `chat`

#### Features
- Closed chat list with rich filters (search, agent, instance, date range, page size)
- Reopen closed chats
- Navigate to reopened conversation
- Debounced filter updates (350ms)

#### API Endpoints
- `GET /api/contacts/closed` — List closed chats
- `POST /api/contacts/{id}/reopen` — Reopen chat
- `GET /api/users` — List users (for agent filter)
- `GET /api/instances` — List instances (for instance filter)

---

### 9. `/settings/tags` — Tag Management

**Component**: `TagsView.vue`  
**Permission**: `tags`

#### Features
- CRUD for organization-level tags
- Color selection with visual preview
- Search and pagination

#### API Endpoints
- `GET /api/tags` — List tags
- `POST /api/tags` — Create tag
- `PUT /api/tags/{name}` — Update tag
- `DELETE /api/tags/{name}` — Delete tag

---

### 10. `/settings/users` — User Management

**Component**: `UsersView.vue`  
**Permission**: `users`

#### Features
- User CRUD (create, edit, delete)
- Invite link generation
- Add existing user from another organization
- Send restrictions per user (authorized numbers, allowed instances, unclaimed chat view/send)
- Per-organization strict sending restrictions toggle
- Super-admin toggle
- Member role updates

#### API Endpoints
- `GET /api/users` — List users
- `POST /api/users` — Create user
- `PUT /api/users/{id}` — Update user
- `DELETE /api/users/{id}` — Delete user
- `GET /api/roles` — List roles (for role assignment)
- `GET /api/instances` — List instances (for send restrictions)
- `POST /api/auth/register-invite` — Generate invite link
- `POST /api/organizations/members` — Add member from another org
- `GET /api/users/{id}/send-restrictions` — Get send restrictions
- `PUT /api/users/{id}/send-restrictions` — Update send restrictions
- `GET /api/org/settings` — Get org settings (for strict sending)

---

### 11. `/settings/roles` — Role & Permission Management

**Component**: `RolesView.vue`  
**Permission**: `roles`

#### Features
- Role CRUD with permission matrix
- System role protection (read-only for non-super-admins)
- Default role assignment
- User count per role
- Permission matrix component (`PermissionMatrix`)

#### API Endpoints
- `GET /api/roles` — List roles
- `POST /api/roles` — Create role
- `GET /api/roles/{id}` — Get role
- `PUT /api/roles/{id}` — Update role
- `DELETE /api/roles/{id}` — Delete role
- `GET /api/permissions` — List all permissions

---

### 12. `/settings/teams` — Team Management

**Component**: `TeamsView.vue`  
**Permission**: `teams`

#### Features
- Team CRUD (name, description, assignment strategy)
- Member management (add/remove agents and managers)
- Assignment strategies: round_robin, load_balanced, manual
- Active/inactive toggle

#### API Endpoints
- `GET /api/teams` — List teams
- `POST /api/teams` — Create team
- `GET /api/teams/{id}` — Get team
- `PUT /api/teams/{id}` — Update team
- `DELETE /api/teams/{id}` — Delete team
- `GET /api/teams/{id}/members` — List team members
- `POST /api/teams/{id}/members` — Add team member
- `DELETE /api/teams/{id}/members/{userId}` — Remove team member

---

### 13. `/settings/api-keys` — API Key Management

**Component**: `APIKeysView.vue`  
**Permission**: `api_keys`

#### Features
- Create API keys with name and optional expiry
- One-time key display after creation (copy-to-clipboard)
- Key listing with prefix, last used, expiry, status
- Key deletion

#### API Endpoints
- `GET /api/api-keys` — List API keys
- `POST /api/api-keys` — Create API key
- `DELETE /api/api-keys/{id}` — Delete API key

---

### 14. `/settings/webhooks` — Webhook Management

**Component**: `WebhooksView.vue`  
**Permission**: `webhooks`

#### Features
- Webhook CRUD (name, URL, events, headers, secret)
- Enable/disable toggle
- Test delivery with result dialog
- Event type selection

#### API Endpoints
- `GET /api/webhooks` — List webhooks
- `POST /api/webhooks` — Create webhook
- `GET /api/webhooks/{id}` — Get webhook
- `PUT /api/webhooks/{id}` — Update webhook
- `DELETE /api/webhooks/{id}` — Delete webhook
- `POST /api/webhooks/{id}/test` — Test webhook delivery

---

### 15. `/settings/sso` — SSO Configuration

**Component**: `SSOSettingsView.vue`  
**Permission**: `settings.sso`

#### Features
- SSO provider configuration (Google, Microsoft, GitHub, Facebook, custom OIDC)
- Enable/disable per provider
- Auto-create users on SSO login
- Default role assignment
- Allowed email domains
- Redirect URL generation

#### API Endpoints
- `GET /api/settings/sso` — List SSO providers
- `PUT /api/settings/sso/{provider}` — Update provider config
- `DELETE /api/settings/sso/{provider}` — Remove provider

---

### 16. `/settings/license` — License Management

**Component**: `LicenseSettingsView.vue`  
**Permission**: `adminOnly` (super-admin)

#### Features
- License status display (active, locked, expired, grace period)
- Quota usage (users, endpoints, storage, subscription days)
- Organization-level usage breakdown
- Server identity (HWID) display
- License activation via security key

#### API Endpoints
- `GET /api/license/bootstrap` — Get license status (public)
- `POST /api/license/activate` — Activate license (public)

---

### 17. `/settings/custom-actions` — Custom Actions

**Component**: `CustomActionsView.vue`  
**Permission**: `custom_actions`

#### Features
- Custom action CRUD
- Three action types: webhook (URL, method, headers, body), URL (open in new tab), JavaScript (inline code)
- Icon selection
- Display order
- Active/inactive toggle

#### API Endpoints
- `GET /api/custom-actions` — List custom actions
- `POST /api/custom-actions` — Create custom action
- `GET /api/custom-actions/{id}` — Get custom action
- `PUT /api/custom-actions/{id}` — Update custom action
- `DELETE /api/custom-actions/{id}` — Delete custom action
- `POST /api/custom-actions/{id}/execute` — Execute custom action
- `GET /api/custom-actions/redirect/{token}` — Redirect (public, one-time token)

---

## Permission Model

**Resources (35 total):** users, teams, roles, settings.general, settings.chatbot, settings.sso,
settings.uploads_cleanup, accounts, templates, flows.whatsapp, flows.chatbot, campaigns,
chatbot.keywords, chatbot.ai, chat, chat.assign, chat.collaborators, chat.bypass_claim,
contacts, tags, analytics, analytics.agents, transfers, agent_selection, webhooks, api_keys,
canned_responses, custom_actions, organizations, wa_filter, saved_contents,
catalogs, group_directory, group_participants

**Actions (11):** read, write, delete, soft_delete, sync, execute, import, export, pickup, assign, prefix

**Error format:** `"Insufficient permissions: resource:action"` (standardized June 2026)

### Permission Hierarchy

```
Super Admin → Admin → Manager → Agent
```

### Permission Matrix for Settings Routes

| Route | Permission Required | Type | Notes |
|-------|-------------------|------|-------|
| `/settings` | `settings.general` or `settings.uploads_cleanup` | Any of | Route-level guard |
| `/settings/chatbot` | `settings.chatbot` | Single | Route-level guard |
| `/settings/accounts` | `accounts` (metaOnly) | Single + provider check | Only for Meta provider |
| `/settings/instances` | `accounts` | Single | |
| `/settings/instances/health` | `accounts` | Single | |
| `/settings/canned-responses` | `canned_responses` | Single | |
| `/settings/contacts` | `contacts` | Single | |
| `/settings/closed-chats` | `chat` | Single | |
| `/settings/tags` | `tags` | Single | |
| `/settings/users` | `users` | Single | |
| `/settings/roles` | `roles` | Single | |
| `/settings/teams` | `teams` | Single | |
| `/settings/api-keys` | `api_keys` | Single | |
| `/settings/webhooks` | `webhooks` | Single | |
| `/settings/sso` | `settings.sso` | Single | |
| `/settings/license` | `adminOnly` | Super-admin only | |
| `/settings/custom-actions` | `custom_actions` | Single | |

### Permission Actions

Most permissions support `read`, `write`, `delete`, and sometimes `execute` actions:

| Resource | Read | Write | Delete | Execute |
|----------|------|-------|--------|---------|
| `settings.general` | View org settings | Edit org settings | — | — |
| `settings.uploads_cleanup` | View cleanup config | Edit cleanup config | — | Run cleanup now |
| `settings.chatbot` | View chatbot settings | Edit chatbot settings | — | — |
| `users` | List/view users | Create/edit users | Delete users | — |
| `roles` | List/view roles | Create/edit roles | Delete roles | — |
| `teams` | List/view teams | Create/edit teams | Delete teams | — |
| `contacts` | List/view contacts | Edit contacts | Delete contacts | Import/export |

---

## API Reference

### Backend Route Registration

All settings-related routes are registered in `cmd/whatomate/main.go` under the authenticated route group with `TenantScope` middleware.

### Common Patterns

**Request Envelope**:
```json
{
  "data": { ... },
  "message": "Success"
}
```

**Error Envelope**:
```json
{
  "message": "Error description",
  "data": null
}
```

**Authentication Headers**:
- `Authorization: Bearer <JWT>` — JWT authentication
- `X-API-Key: <key>` — API key authentication
- `X-Organization-ID: <org_id>` — Tenant scoping
- `X-CSRF-Token: <token>` — CSRF protection (unless Bearer/API-key auth)
- `Cookie: whm_csrf=<token>` — CSRF double-submit

### Pagination

Most list endpoints support pagination:
```
GET /api/resource?page=1&limit=25&search=query
```

Response includes pagination metadata in the envelope.

---

## Frontend Components

### Component Architecture

```
views/settings/
├── SettingsView.vue           Main settings (4 tabs, org + user prefs)
├── ChatbotSettingsView.vue    Chatbot config (5 tabs)
├── AccountsView.vue           Meta Cloud API accounts
├── InstancesView.vue          WhatsApp instances
├── InstanceHealthView.vue     Health dashboard
├── CannedResponsesView.vue    Message templates
├── ContactsView.vue           Contact management
├── ClosedChatsView.vue        Closed chat history
├── TagsView.vue               Tag management
├── UsersView.vue              User management
├── RolesView.vue              Role & permissions
├── TeamsView.vue              Team management
├── APIKeysView.vue            API keys
├── WebhooksView.vue           Webhooks
├── SSOSettingsView.vue        SSO providers
├── LicenseSettingsView.vue    License management
├── CustomActionsView.vue      Custom actions
├── BusinessProfileDialog.vue  Business profile editor
├── PendingChatsView.vue       Pending chat queue
├── AssignedChatsView.vue      Assigned chat queue
├── CampaignsView.vue          Campaign management
├── FlowsView.vue              WhatsApp flows
├── TemplatesView.vue          WhatsApp templates
└── LicenseCleanupView.vue     License cleanup
```

### Shared Components Used
- `DataTable` — Paginated tables with search, sorting
- `PermissionMatrix` — Role permission grid
- `HealthDashboard` — Instance health visualization
- `ImportExportDialog` — Contact import/export
- `CreateContactDialog` — Contact creation form
- `BusinessProfileDialog` — Business profile editor

### State Management

| Store | Used By | Purpose |
|-------|---------|---------|
| `auth` | All views | Current user, permissions, auth state |
| `users` | UsersView, ClosedChatsView | User CRUD, current user |
| `roles` | RolesView | Role CRUD, permissions |
| `teams` | TeamsView | Team CRUD, members |
| `instances` | InstancesView, ContactsView, ClosedChatsView | Instance data |
| `contacts` | ContactsView, ClosedChatsView, PendingChats, AssignedChats | Contact CRUD |
| `tags` | TagsView, ContactsView | Tag CRUD |
| `license` | LicenseSettingsView | License state, activation |
| `config` | SettingsView | App configuration |
| `organizations` | UsersView, SettingsView | Organization management |

---

## Data Flow

### Settings Save Flow

```
User clicks Save
    ↓
Frontend validation (field-level)
    ↓
API call (PUT /api/org/settings or PUT /api/me/settings)
    ↓
CSRF token validation (middleware)
    ↓
JWT/API-key authentication (middleware)
    ↓
TenantScope middleware (org scoping)
    ↓
Handler validates input
    ↓
GORM update with org WHERE clause
    ↓
Response envelope
    ↓
Toast notification (success/error)
    ↓
Refresh local state from response
```

### Instance Connection Flow

```
User clicks Connect
    ↓
POST /api/instances/{id}/connect
    ↓
Backend initiates whatsmeow connection
    ↓
QR code generated
    ↓
WebSocket event: qr_code_update
    ↓
Frontend displays QR code
    ↓
15s watchdog timer starts
    ↓
If QR expires: poll GET /api/instances/{id}/qr for snapshot
    ↓
User scans QR with WhatsApp
    ↓
WebSocket event: instance_connected
    ↓
Frontend shows connected state
```

### Chatbot Settings Save Flow

```
User edits tab (Messages/Agents/Business Hours/SLA/AI)
    ↓
Tab-specific save button clicked
    ↓
PUT /api/chatbot/settings (full settings payload)
    ↓
Backend merges with existing settings
    ↓
Chatbot processor reloads settings
    ↓
WebSocket broadcast to connected agents
```

---

## Configuration

### Backend Config (`config.toml`)

Settings-related configuration sections:

```toml
[app]
environment = "production"

[whatsapp]
provider = "meta"           # or "whatsmeow"

[uploads_cleanup]
# Controlled via /settings General tab
# retention_days and schedule_hour stored in DB

[license]
enabled = true
# public_key, public_key_kid for license verification
```

### Frontend Config

- Theme preferences stored per-user via `PUT /api/me/settings`
- Chat background stored per-user via `PUT /api/me/settings` or `POST /api/me/chat-background`
- Media group window stored in `localStorage` (key: `chat.mediaGroupWindowSeconds`)
- Print/download button preferences stored in `localStorage` via `configStore`

### Storage Paths

- Chat backgrounds: uploaded via `POST /api/me/chat-background`
- Canned response attachments: uploaded as multipart form data
- Auto-campaign media: `POST /api/instances/{id}/auto-campaign/media`
- Uploads cleanup: configurable retention (1-3650 days)
