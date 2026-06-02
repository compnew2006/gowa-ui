# Whatomate Database Schema Report

This document provides a 100% exhaustive extraction of the Whatomate database schema, derived from GORM models and PostgreSQL initialization logic.

---

## 🏗️ 1. Master Table Register

| Domain | Table Name | GORM Model | Multi-tenant | Soft Delete | Description |
| :--- | :--- | :--- | :---: | :---: | :--- |
| **Identity** | `organizations` | `Organization` | N/A | Yes | Central tenant entity |
| **Identity** | `organization_configs` | `OrganizationConfig` | Yes | No | Tenant-specific settings |
| **Identity** | `users` | `User` | No | Yes | System users |
| **Identity** | `user_organizations` | `UserOrganization` | Yes | No | Link between users and tenants |
| **Identity** | `permissions` | `Permission` | No | No | Granular access rights |
| **Identity** | `custom_roles` | `CustomRole` | Yes | Yes | Tenant-defined roles |
| **Identity** | `api_keys` | `APIKey` | Yes | No | Programmatic access tokens |
| **Identity** | `sso_providers` | `SSOProvider` | No | No | External auth configurations |
| **Messaging** | `contacts` | `Contact` | Yes | Yes | Customer profiles |
| **Messaging** | `messages` | `Message` | Yes | Yes | WhatsApp message history |
| **Messaging** | `tags` | `Tag` | Yes | No | Contact labels |
| **Messaging** | `media_assets` | `MediaAsset` | Yes | No | Uploaded/received files |
| **Channels** | `whatsapp_accounts` | `WhatsAppAccount` | Yes | Yes | Meta WABA accounts |
| **Channels** | `whatsapp_instances` | `WhatsAppInstance` | Yes | Yes | Whatsmeow virtual devices |
| **Automation** | `chatbot_settings` | `ChatbotSettings` | Yes | No | AI/Flow global settings |
| **Campaigns**| `bulk_message_campaigns` | `BulkMessageCampaign` | Yes | Yes | Outbound broadcast jobs |
| **Routing**  | `agent_selection_settings` | `AgentSelectionSettings` | Yes | No | Customer-driven agent routing configuration (global or per-instance) |
| **Routing**  | `agent_selection_participants` | `AgentSelectionParticipant` | Yes | No | Agents/teams/queues eligible for the selection menu |
| **Routing**  | `agent_selection_options` | `AgentSelectionOption` | Yes | No | Numbered menu items presented to the customer |
| **Routing**  | `agent_selection_sessions` | `AgentSelectionSession` | Yes | No | Per-conversation routing state machine (delay, sent, selected, timeout, cancelled) |
| **Routing**  | `agent_selection_audit_events` | `AgentSelectionAuditEvent` | Yes | No | Append-only audit log of every routing action |

*(Full list of 49 tables detailed below)*

---

## 📋 2. Detailed Table Specifications

### 🟦 Domain: Identity & Access

#### `organizations`
*   **Description**: Root entity for multi-tenancy.
*   **Columns**:
    *   `id`: `UUID` (PK, Default: gen_random_uuid())
    *   `created_at`: `TIMESTAMP WITH TIME ZONE`
    *   `updated_at`: `TIMESTAMP WITH TIME ZONE`
    *   `deleted_at`: `TIMESTAMP WITH TIME ZONE` (Index)
    *   `name`: `TEXT` (Unique Index)
    *   `slug`: `TEXT` (Unique Index)
    *   `logo_url`: `TEXT`
    *   `timezone`: `TEXT`
    *   `is_active`: `BOOLEAN` (Default: true)
*   **Indexes**: `idx_organizations_deleted_at`, `idx_organizations_name`, `idx_organizations_slug`.
*   **API Usage**: `/api/organizations`, `/api/auth/register`.

#### `users`
*   **Description**: Registered users across all organizations.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `created_at`, `updated_at`, `deleted_at`: (Standard BaseModel)
    *   `email`: `TEXT` (Unique Index)
    *   `password`: `TEXT` (Encrypted)
    *   `first_name`, `last_name`: `TEXT`
    *   `phone`: `TEXT`
    *   `avatar_url`: `TEXT`
    *   `is_active`: `BOOLEAN`
    *   `is_admin`: `BOOLEAN` (Global Admin)
    *   `last_login`: `TIMESTAMP WITH TIME ZONE`
*   **API Usage**: `/api/users`, `/api/auth/login`.

#### `user_organizations`
*   **Description**: Mapping users to organizations with role assignments.
*   **Columns**:
    *   `user_id`: `UUID` (PK, FK: users.id)
    *   `organization_id`: `UUID` (PK, FK: organizations.id)
    *   `role_id`: `UUID` (FK: custom_roles.id)
    *   `is_owner`: `BOOLEAN`
    *   `membership_type`: `TEXT`
*   **Relationships**: BelongsTo `User`, BelongsTo `Organization`.

---

### 🟩 Domain: WhatsApp Messaging

#### `contacts`
*   **Description**: Customers identified by WhatsApp ID (JID).
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK, Multi-tenant)
    *   `whatsapp_id`: `TEXT` (The phone number/JID)
    *   `name`: `TEXT`
    *   `avatar_url`: `TEXT`
    *   `assigned_to`: `UUID` (FK: users.id)
    *   `last_message_at`: `TIMESTAMP`
    *   `status`: `TEXT` (Open, Closed, etc.)
    *   `metadata`: `JSONB`
*   **Multi-tenancy**: Strictly enforced via `organization_id`.
*   **Indexes**: `idx_contacts_org_whatsapp` (Unique).

#### `messages`
*   **Description**: Log of all sent and received messages.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK)
    *   `contact_id`: `UUID` (FK: contacts.id)
    *   `user_id`: `UUID` (FK: users.id - if sent by agent)
    *   `whatsapp_message_id`: `TEXT` (Provider ID)
    *   `type`: `TEXT` (text, image, audio, etc.)
    *   `body`: `TEXT`
    *   `status`: `TEXT` (sent, delivered, read, failed)
    *   `direction`: `TEXT` (inbound, outbound)
    *   `media_path`: `TEXT`
*   **Indexes**: `idx_messages_contact_id`, `idx_messages_whatsapp_id`.

---

## 🔗 3. Relationship Inventory

### Core Associations
1.  **Organization ↔ WhatsAppAccount**: One-to-Many.
2.  **Organization ↔ User**: Many-to-Many via `user_organizations`.
3.  **Organization ↔ Contact**: One-to-Many.
4.  **Contact ↔ Message**: One-to-Many.
5.  **WhatsAppInstance ↔ Message**: One-to-Many (Tracking which instance processed the message).

### Automation Associations
1.  **ChatbotFlow ↔ ChatbotFlowStep**: One-to-Many.
2.  **KeywordRule ↔ ChatbotFlow**: Many-to-One (Rules trigger specific flows).
3.  **Contact ↔ ChatbotSession**: One-to-Many (Historical sessions).

---

## 🛡️ 4. Multi-tenancy & Soft-Delete Audit

### Multi-tenancy Strategy
*   **Mechanism**: Discriminator column `organization_id` (UUID).
*   **Enforcement**: Middleware `TenantScope` in `main.go` automatically injects filters into GORM queries.
*   **Exempt Tables**: `users`, `permissions`, `sso_providers`, `license_records` (Global/System level).

### Soft-Delete Strategy
*   **Mechanism**: `gorm.DeletedAt` column.
*   **Audit**:
    *   ✅ **Soft-Delete Enabled**: `organizations`, `users`, `custom_roles`, `contacts`, `messages`, `bulk_message_campaigns`, `chatbot_flows`, `whatsapp_accounts`, `agent_selection_participants`.
    *   ❌ **Hard-Delete Only**: `user_organizations`, `tags`, `chatbot_sessions`, `api_keys`, `agent_selection_settings`, `agent_selection_options`, `agent_selection_sessions`, `agent_selection_audit_events`.

---

## 🚀 5. Production Notes
*   **Encryption**: `whatsapp_accounts` and `api_keys` contain encrypted secrets marked with `enc3:` prefix in the database.
*   **Concurrency**: Tables like `bulk_message_recipients` use indexes to support high-throughput worker consumption from Redis-backed streams.

---

## 🧭 6. Customer Routing (Agent Selection) Domain

The `agent_selection_*` family powers `/settings/agent-selection` — a feature that lets incoming WhatsApp customers pick the agent, team, or queue that should handle their chat. All five tables are multi-tenant via `organization_id`; none are soft-deletable (use the audit log for history).

#### `agent_selection_settings`
*   **Description**: Per-organization routing configuration. When `instance_id IS NULL`, the row is the **global default**; when set, it overrides for that specific WhatsApp instance only.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK, Multi-tenant)
    *   `instance_id`: `UUID` (FK: whatsapp_instances.id, nullable)
    *   `allowed_instance_ids`: `TEXT[]` (Postgres `StringArray` — instances the menu may listen on)
    *   `enabled`: `BOOLEAN` (master on/off switch)
    *   `trigger_mode`: `TEXT` enum (`first_pending_message` | `keyword` | `after_office_hours` | `chatbot_step` | `manual_test`)
    *   `trigger_keywords`: `TEXT[]` (comma-separated keywords honored when `trigger_mode = 'keyword'`)
    *   `prompt_delay_minutes`: `INT` (0–1440, validated)
    *   `selection_timeout_minutes`: `INT` (1–1440, validated)
    *   `max_invalid_attempts`: `INT` (1–20, validated)
    *   `menu_header_text`, `menu_footer_text`, `invalid_reply_text`, `timeout_response_text`, `unavailable_agent_text`: `TEXT`
    *   `custom_final_option_enabled`: `BOOLEAN`
    *   `custom_final_option_text`: `TEXT`
    *   `hide_unavailable_agents`: `BOOLEAN` (default true; when false, all enabled+active participants appear regardless of `MaxOpenChats`/`IsAvailable`)
    *   `created_at`, `updated_at`: `TIMESTAMP WITH TIME ZONE`
*   **Indexes**: unique `(organization_id, instance_id)`; `idx_agent_selection_settings_org`.
*   **API**: `GET/PUT /api/agent-selection/settings`, `POST /api/agent-selection/test-send`.

#### `agent_selection_participants`
*   **Description**: Eligible agents, teams, or queues shown in the menu. `user_id` is nullable because a participant may be a team/queue reference instead of a single user.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK)
    *   `settings_id`: `UUID` (FK: agent_selection_settings.id)
    *   `user_id`: `UUID` (FK: users.id, nullable)
    *   `team_id`: `UUID` (FK: teams.id, nullable)
    *   `display_name`: `TEXT`
    *   `description`: `TEXT`
    *   `is_active`: `BOOLEAN` (soft availability flag; gated by `ShowOnlyWhenAvailable` + `MaxOpenChats` at menu-build time)
    *   `sort_order`: `INT`
    *   `created_at`, `updated_at`, `deleted_at`: `TIMESTAMP WITH TIME ZONE` (BaseModel — soft-delete enabled)
*   **Indexes**: **partial unique** `idx_agent_selection_participant_user` on `(organization_id, settings_id, user_id) WHERE deleted_at IS NULL` — installed by `fixAgentSelectionParticipantUniqueIndex` in `internal/database/postgres.go` (mirrors the `saved_contents` pattern). Soft-deleting a participant and re-adding the same agent succeeds because the soft-deleted row is excluded from the index.
*   **API**: `GET/POST/DELETE /api/agent-selection/participants` (DELETE requires `agent_selection:delete` permission).

#### `agent_selection_options`
*   **Description**: Numbered menu items (e.g. "1) Print at branch", "2) Talk to support"). Either user-targeted (assigns a chat) or a custom final action.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK)
    *   `settings_id`: `UUID` (FK: agent_selection_settings.id)
    *   `option_type`: `TEXT` enum (`user` | `team` | `queue` | `custom_final`)
    *   `label`: `TEXT`
    *   `description`: `TEXT`
    *   `target_user_id`: `UUID` (FK: users.id, nullable)
    *   `target_team_id`: `UUID` (FK: teams.id, nullable)
    *   `custom_action`: `TEXT` (JSONB-encoded action descriptor for `custom_final` options)
    *   `sort_order`: `INT`
    *   `is_active`: `BOOLEAN`
    *   `created_at`, `updated_at`: `TIMESTAMP WITH TIME ZONE`
*   **API**: `GET/POST/DELETE /api/agent-selection/options` (DELETE requires `agent_selection:delete` permission).

#### `agent_selection_sessions`
*   **Description**: Per-conversation state machine for the routing flow. One row per (contact, instance) — created on inbound trigger, advanced by customer reply, ticked by the background sweeper.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK)
    *   `instance_id`: `UUID` (FK: whatsapp_instances.id, nullable)
    *   `contact_id`: `UUID` (FK: contacts.id)
    *   `whatsapp_account`: `TEXT` (denormalized for quick audit joins)
    *   `status`: `TEXT` enum (`waiting_delay` | `menu_sent` | `selected` | `timeout` | `cancelled` | `failed`)
    *   `prompt_due_at`: `TIMESTAMP WITH TIME ZONE` (when to send the menu)
    *   `expires_at`: `TIMESTAMP WITH TIME ZONE` (selection timeout)
    *   `menu_snapshot`: `JSONB` (frozen rendering of the menu at send time, used to validate customer reply)
    *   `selected_option_id`: `UUID` (FK: agent_selection_options.id, nullable)
    *   `created_at`, `updated_at`: `TIMESTAMP WITH TIME ZONE`
*   **Indexes**: `idx_agent_selection_sessions_contact`, `idx_agent_selection_sessions_status_due`.

#### `agent_selection_audit_events`
*   **Description**: Append-only audit log of every routing action — settings changes, menu sends, customer replies, timeout firings, deletions, test sends, and permission denials.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `organization_id`: `UUID` (FK)
    *   `instance_id`: `UUID` (FK: whatsapp_instances.id, nullable)
    *   `session_id`: `UUID` (FK: agent_selection_sessions.id, nullable)
    *   `event_type`: `TEXT` enum (`settings_updated` | `participant_added` | `participant_deleted` | `option_added` | `option_deleted` | `menu_sent` | `selection_made` | `timeout` | `cancelled` | `test_menu_sent` | `test_send_failed`)
    *   `actor_type`: `TEXT` enum (`user` | `system` | `customer`)
    *   `actor_id`: `UUID` (nullable; user who triggered the event when `actor_type='user'`)
    *   `metadata`: `JSONB` (event-specific context: `timeout_response_text_sent: bool`, `menu_text`, `whatsapp_account`, `error`, etc.)
    *   `created_at`: `TIMESTAMP WITH TIME ZONE`
*   **API**: `GET /api/agent-selection/audit`.

### Routing Domain Relationships
1.  `Organization` 1—N `agent_selection_settings` (per-instance override pattern)
2.  `agent_selection_settings` 1—N `agent_selection_participants`
3.  `agent_selection_settings` 1—N `agent_selection_options`
4.  `agent_selection_settings` 1—N `agent_selection_sessions` (filtered by instance_id)
5.  `agent_selection_sessions` 1—N `agent_selection_audit_events`
6.  `agent_selection_options` 1—N `agent_selection_sessions` (via `selected_option_id`)

### Routing Permissions
*   `agent_selection:read` — list, get, preview, sessions, audit
*   `agent_selection:write` — update settings, add/edit participants, add/edit options, cancel sessions, **send test menu**
*   `agent_selection:delete` — delete participants, delete options, cancel sessions (intentionally split from `:write` to give least-privilege control to managers who should never erase routing config)
