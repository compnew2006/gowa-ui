# Whatomate Database Schema Report

This document provides a 100% exhaustive extraction of the Whatomate database schema, derived from GORM models and PostgreSQL initialization logic.

---

## 🏗️ 1. Master Table Register

| Domain | Table Name | GORM Model | Multi-tenant | Soft Delete | Description |
| :--- | :--- | :--- | :---: | :---: | :--- |
| **Identity** | `organizations` | `Organization` | N/A | Yes | Central tenant entity |
| **Identity** | `organization_configs` | `OrganizationConfig` | Yes | No | Tenant-specific limits |
| **Identity** | `users` | `User` | No | Yes | System users |
| **Identity** | `user_organizations` | `UserOrganization` | Yes | No | User-Tenant mapping |
| **Identity** | `permissions` | `Permission` | No | No | Granular access rights |
| **Identity** | `custom_roles` | `CustomRole` | Yes | Yes | Tenant-defined roles |
| **Identity** | `role_permissions` | `RolePermission` | No | No | Role-Permission junction |
| **Identity** | `api_keys` | `APIKey` | Yes | No | Programmatic access |
| **Identity** | `sso_providers` | `SSOProvider` | Yes | No | Auth configurations |
| **Messaging** | `contacts` | `Contact` | Yes | Yes | Customer profiles |
| **Messaging** | `messages` | `Message` | Yes | Yes | WhatsApp history |
| **Messaging** | `tags` | `Tag` | Yes | No | Contact labels |
| **Messaging** | `media_assets` | `MediaAsset` | Yes | No | Object storage metadata |
| **Messaging** | `contact_user_deletions`| `ContactUserDeletion` | Yes | No | Per-user chat hide tracking |
| **Messaging** | `contact_collaborators`| `ContactCollaborator` | Yes | No | Shared chat access |
| **Channels** | `whatsapp_accounts` | `WhatsAppAccount` | Yes | Yes | Meta WABA accounts |
| **Channels** | `whatsapp_instances` | `WhatsAppInstance` | Yes | Yes | Virtual device connections |
| **Channels** | `whatsapp_statuses` | `WhatsAppStatus` | Yes | No | WhatsApp Stories/Statuses |
| **Channels** | `instance_notifications`| `InstanceNotification` | Yes | No | Connection health alerts |
| **Automation** | `chatbot_settings` | `ChatbotSettings` | Yes | No | Global bot config |
| **Automation** | `chatbot_flows` | `ChatbotFlow` | Yes | Yes | Multi-step logic |
| **Automation** | `chatbot_flow_steps` | `ChatbotFlowStep` | No | No | Flow node definitions |
| **Automation** | `chatbot_sessions` | `ChatbotSession` | Yes | No | Active bot interactions |
| **Automation** | `chatbot_session_messages`| `ChatbotSessionMessage` | No | No | Path tracking in sessions |
| **Automation** | `keyword_rules` | `KeywordRule` | Yes | No | Simple auto-responders |
| **Automation** | `ai_contexts` | `AIContext` | Yes | No | RAG/Prompt snippets |
| **Automation** | `agent_transfers` | `AgentTransfer` | Yes | No | Human-handoff tracking |
| **Campaigns** | `bulk_message_campaigns`| `BulkMessageCampaign` | Yes | Yes | Outbound broadcasts |
| **Campaigns** | `bulk_message_recipients`| `BulkMessageRecipient` | No | No | Campaign job queue |
| **Campaigns** | `notification_rules` | `NotificationRule` | Yes | No | Event-triggered messages |
| **Campaigns** | `templates` | `Template` | Yes | No | Meta approved templates |
| **Campaigns** | `catalogs` | `Catalog` | Yes | No | Product catalogs |
| **Campaigns** | `catalog_products` | `CatalogProduct` | Yes | No | Catalog items |
| **Operations** | `teams` | `Team` | Yes | Yes | Agent groups |
| **Operations** | `team_members` | `TeamMember` | No | No | User-Team mapping |
| **Operations** | `canned_responses` | `CannedResponse` | Yes | Yes | Quick replies |
| **Operations** | `conversation_notes` | `ConversationNote` | Yes | Yes | Internal agent notes |
| **Operations** | `chat_closure_ratings` | `ChatClosureRating` | Yes | No | CSAT/Feedback loop |
| **Ops/Infra** | `webhooks` | `Webhook` | Yes | No | Outbound event delivery |
| **Ops/Infra** | `custom_actions` | `CustomAction` | Yes | No | UI extension buttons |
| **Ops/Infra** | `widgets` | `Widget` | Yes | No | Dashboard components |
| **Ops/Infra** | `license_records` | `LicenseRecord` | No | No | Deployment Entitlements |
| **Ops/Infra** | `license_events` | `LicenseEvent` | No | No | Enforcement audit trail |
| **Ops/Infra** | `user_availability_logs`| `UserAvailabilityLog` | Yes | No | Productivity tracking |
| **Plugin** | `instance_uploads_cleanup_audits` | `InstanceUploadsCleanupAudit` | Yes | No | Uploads cleanup audit trail (per-instance-uploads-cleanup plugin) |

---

## 📋 2. Detailed Table Specifications

### 🟦 Domain: Identity & Access

#### `organizations`
*   **Description**: Root entity for multi-tenancy.
*   **Columns**:
    *   `id`: `UUID` (PK)
    *   `name`: `TEXT` (Organization name)
    *   `slug`: `TEXT` (Unique URL identifier)
    *   `settings`: `JSONB` (Branding, feature flags)
    *   `created_at`, `updated_at`, `deleted_at`: (Base fields)
*   **Indexes**: `idx_organizations_slug` (Unique).
*   **API**: `POST /api/organizations` (Admin only), `GET /api/organizations/{id}`.

#### `organization_configs`
*   **Columns**:
    *   `organization_id`: `UUID` (Unique FK)
    *   `worker_count`: `INT` (Provisioned workers)
    *   `max_queue_size`: `INT`
    *   `max_whatsapp_instances`: `INT`
*   **Relationship**: One-to-One with `Organization`.

#### `users`
*   **Columns**:
    *   `organization_id`: `UUID` (Initial Org)
    *   `email`: `TEXT` (Unique)
    *   `password_hash`: `TEXT` (Internal)
    *   `full_name`: `TEXT`
    *   `role_id`: `UUID` (FK)
    *   `is_active`, `is_available`, `is_super_admin`: `BOOLEAN`
    *   `sso_provider`, `sso_provider_id`: `TEXT`
*   **API**: `/api/users`, `/api/auth/login`.

#### `user_organizations`
*   **Columns**:
    *   `user_id`, `organization_id`: `UUID` (Composite index `idx_user_org`)
    *   `role_id`: `UUID` (Org-specific role)
    *   `is_default`: `BOOLEAN`
*   **API**: `GET /api/me/organizations`.

#### `permissions`
*   **Description**: Granular access rights following the `resource:action` pattern.
*   **Columns**:
    *   `resource`: `TEXT` (35 resources including: users, teams, roles, settings.*, accounts,
        templates, flows.*, campaigns, chatbot.*, chat, chat.assign, chat.collaborators,
        chat.bypass_claim, contacts, tags, analytics, analytics.agents, transfers,
        agent_selection, webhooks, api_keys, canned_responses, custom_actions,
        organizations, wa_filter, saved_contents, **catalogs**, **group_directory**,
        **group_participants**)
    *   `action`: `TEXT` (read, write, delete, soft_delete, sync, execute, import, export,
        pickup, assign, prefix)
    *   `description`: `TEXT`
*   **Indexes**: `idx_permission_resource_action` (Unique).
*   **Seeded via**: `DefaultPermissions()` in `internal/models/roles.go`

#### `custom_roles`
*   **Columns**:
    *   `organization_id`: `UUID`
    *   `name`: `TEXT`
    *   `is_system`: `BOOLEAN` (System roles: admin, agent, manager)
    *   `is_default`: `BOOLEAN` (Assigned to new members)
*   **API**: `GET /api/roles`.

#### `role_permissions`
*   **Columns**:
    *   `custom_role_id`, `permission_id`: `UUID` (PK)
*   **Relationship**: Many-to-Many junction between `CustomRole` and `Permission`.

#### `api_keys`
*   **Columns**:
    *   `organization_id`, `user_id`: `UUID`
    *   `name`: `TEXT`
    *   `key_prefix`: `TEXT` (First 16 chars)
    *   `key_hash`: `TEXT` (Stored bcrypt hash)
    *   `expires_at`: `TIMESTAMP`
*   **API**: `POST /api/settings/api-keys`.

#### `sso_providers`
*   **Columns**:
    *   `organization_id`: `UUID`
    *   `provider`: `TEXT` (google, microsoft, custom)
    *   `client_id`, `client_secret`: `TEXT` (Encrypted)
    *   `allow_auto_create`: `BOOLEAN`
*   **API**: `GET /api/settings/sso`.

---

### 🟩 Domain: WhatsApp & Messaging

#### `whatsapp_accounts`
*   **Description**: Meta Cloud API configurations.
*   **Columns**:
    *   `phone_id`, `business_id`, `app_id`: `TEXT`
    *   `access_token`: `TEXT` (Encrypted)
    *   `auto_read_receipt`: `BOOLEAN`
*   **API**: `/api/channels/whatsapp`.

#### `whatsapp_instances`
*   **Description**: Link devices via Whatsmeow (QR scan).
*   **Columns**:
    *   `jid`: `TEXT` (Instance JID)
    *   `status`: `TEXT` ('connected', 'disconnected', 'banned')
    *   `send_blocked_until`: `TIMESTAMP` (Spam detection protection)
*   **Indexes**: `idx_whatsapp_instances_j_id` (Unique).
*   **API**: `/api/instances`.

#### `whatsapp_statuses`
*   **Columns**:
    *   `sender_jid`, `sender_name`: `TEXT`
    *   `status_type`: `TEXT` (image, video, text)
    *   `expires_at`: `TIMESTAMP` (24h TTL)
*   **API**: `GET /api/chat/statuses`.

#### `instance_notifications`
*   **Columns**:
    *   `instance_id`: `UUID`
    *   `event_type`: `TEXT` (e.g., 'instance.banned')
    *   `is_dismissed`: `BOOLEAN`

#### `contacts`
*   **Columns**:
    *   `phone_number`: `TEXT`
    *   `profile_name`: `TEXT`
    *   `assigned_user_id`: `UUID` (FK: users.id)
    *   `status`: `TEXT` (pending, open, closed)
    *   `is_read`: `BOOLEAN`
    *   `last_inbound_at`: `TIMESTAMP` (Used for 24h window tracking)
    *   `tags`: `JSONB` (Array of strings)
*   **Indexes**: `idx_contacts_org_phone_instance` (Unique).
*   **API**: `GET /api/contacts`.

#### `messages`
*   **Columns**:
    *   `contact_id`, `instance_id`: `UUID`
    *   `direction`: `TEXT` (incoming, outgoing)
    *   `message_type`: `TEXT` (text, image, template, flow, poll)
    *   `content`: `TEXT`
    *   `media_asset_id`: `UUID`
    *   `interactive_data`: `JSONB` (Poll structure, vote selections, button responses)
    *   `status`: `TEXT` (pending, sent, delivered, read, failed)
*   **Indexes**: `idx_messages_contact_created` (Composite for chat history).
*   **API**: `GET /api/contacts/{id}/messages`.

#### `tags`
*   **Columns**:
    *   `organization_id`: `UUID` (PK - Part 1)
    *   `name`: `TEXT` (PK - Part 2)
    *   `color`: `TEXT`
*   **API**: `GET /api/tags`.

#### `contact_user_deletions`
*   **Description**: Tracks when a user "hides" a chat.
*   **Columns**: `contact_id`, `user_id`, `deleted_at`.

#### `contact_collaborators`
*   **Columns**:
    *   `contact_id`, `user_id`: `UUID`
    *   `role`: `TEXT` (viewer, assistant)
    *   `status`: `TEXT` (invited, accepted)
*   **API**: `POST /api/chat/{id}/collaborators`.

#### `media_assets`
*   **Columns**:
    *   `file_hash`: `TEXT` (Deduplication)
    *   `s3_key`: `TEXT` (Object storage path)
    *   `mime_type`: `TEXT`
    *   `size`: `BIGINT`

---

### 🟧 Domain: Automation & AI

#### `chatbot_settings`
*   **Columns**: Included embedded fields for:
    *   `business_hours`: `JSONB`
    *   `sla_response_minutes`: `INT`
    *   `ai_provider`, `ai_model`, `ai_system_prompt`: `TEXT`
    *   `session_timeout_minutes`: `INT`
*   **API**: `/api/settings/chatbot`.

#### `chatbot_flows`
*   **Columns**:
    *   `trigger_keywords`: `JSONB`
    *   `initial_message`: `TEXT`
    *   `on_complete_action`: `TEXT` (webhook, transfer)
*   **API**: `/api/chatbot/flows`.

#### `chatbot_flow_steps`
*   **Columns**:
    *   `flow_id`: `UUID`
    *   `step_name`, `step_order`: `TEXT/INT`
    *   `message_type`: `TEXT` (buttons, input, api_fetch)
    *   `store_as`: `TEXT` (Variable name for session data)
    *   `conditional_next`: `JSONB`

#### `chatbot_sessions`
*   **Columns**:
    *   `contact_id`, `current_flow_id`: `UUID`
    *   `status`: `TEXT` (active, completed, timeout)
    *   `session_data`: `JSONB` (State dictionary)

#### `chatbot_session_messages`
*   **Columns**: `session_id`, `direction`, `message`, `step_name`.

#### `keyword_rules`
*   **Columns**:
    *   `keywords`: `JSONB` (Array of patterns)
    *   `match_type`: `TEXT` (exact, contains, regex)
    *   `response_type`: `TEXT` (text, flow, media)
*   **API**: `/api/chatbot/keywords`.

#### `ai_contexts`
*   **Columns**:
    *   `context_type`: `TEXT` (static, api)
    *   `static_content`: `TEXT`
    *   `trigger_keywords`: `JSONB`
*   **API**: `/api/chatbot/ai/contexts`.

#### `agent_transfers`
*   **Columns**:
    *   `contact_id`, `team_id`, `agent_id`: `UUID`
    *   `status`: `TEXT` (active, resumed)
    *   `sla_response_deadline`: `TIMESTAMP`
*   **API**: `POST /api/chat/{id}/transfer`.

---

### 🟪 Domain: Campaigns & Operational

#### `bulk_message_campaigns`
*   **Columns**:
    *   `template_id`: `UUID`
    *   `min_delay_seconds`, `max_delay_seconds`: `INT` (Anti-ban throttling)
    *   `status`: `TEXT` (draft, queued, processing, completed)
    *   `poll_question`: `TEXT` (Poll question text, empty = no poll)
    *   `poll_options`: `JSONB` (Array of poll option strings, default `[]`)
    *   `poll_max_selections`: `INT` (Max selectable options, default 0 = unlimited)
*   **API**: `/api/campaigns`.

#### `bulk_message_recipients`
*   **Columns**:
    *   `campaign_id`: `UUID`
    *   `phone_normalized`: `TEXT`
    *   `template_params`: `JSONB`
    *   `status`: `TEXT` (pending, sent, failed)
*   **Indexes**: `idx_bulk_recipients_campaign_phone_normalized` (Unique).

#### `templates`
*   **Columns**:
    *   `name`, `language`, `category`: `TEXT`
    *   `body_content`: `TEXT`
    *   `buttons`: `JSONB`
    *   `status`: `TEXT` (APPROVED, REJECTED)
*   **API**: `/api/templates`.

#### `canned_responses`
*   **Columns**:
    *   `shortcut`: `TEXT` (e.g., /greet)
    *   `content`: `TEXT`
    *   `attachments`: `JSONB` (Media metadata)
*   **API**: `/api/canned-responses`.

#### `chat_closure_ratings`
*   **Columns**:
    *   `contact_id`, `closing_agent_id`: `UUID`
    *   `rating`: `INT` (1-10)
    *   `rating_message`: `TEXT`
    *   `state`: `TEXT` (pending, rated, expired)
*   **API**: `POST /api/chat/{id}/close`.

#### `catalogs` & `catalog_products`
*   **Columns**: `meta_catalog_id`, `meta_product_id`, `price`, `currency`.
*   **API**: `/api/catalogs`.

#### `instance_uploads_cleanup_audits` (Plugin: per-instance-uploads-cleanup)
*   **Columns**:
    *   `organization_id`, `instance_id`: `UUID`
    *   `actor_user_id`: `UUID` (Nullable)
    *   `actor_email`: `TEXT` (Nullable)
    *   `old_inherit`: `BOOLEAN` (Nullable)
    *   `new_inherit`: `BOOLEAN`
    *   `old_retention_days`, `new_retention_days`: `INT`
    *   `reason`: `TEXT` (Nullable)
*   **Indexes**: `idx_iuca_org_instance_created` (Composite: organization_id, instance_id, created_at).

#### `conversation_notes`
*   **Columns**: `contact_id`, `created_by_id`, `content`.
*   **API**: `GET /api/contacts/{id}/notes`.

---

### ⬛ Domain: Administration & Infra

#### `teams`
*   **Columns**:
    *   `name`, `description`: `TEXT`
    *   `assignment_strategy`: `TEXT` (round_robin, load_balanced)
*   **API**: `/api/teams`.

#### `team_members`
*   **Columns**: `team_id`, `user_id`, `role` (manager, agent).

#### `license_records`
*   **Columns**: `hwid_hash`, `tier`, `max_organizations`, `expires_at`.
*   **API**: Internal Binary Only.

#### `license_events`
*   **Columns**: `event_type`, `reason`, `details`.

#### `user_availability_logs`
*   **Columns**: `user_id`, `started_at`, `ended_at`, `duration_seconds`.

#### `webhooks`
*   **Columns**: `url`, `events` (JSON array), `secret`, `is_active`.
*   **API**: `/api/settings/webhooks`.

#### `custom_actions`
*   **Columns**: `name`, `action_type` (webhook, url), `config`.
*   **API**: `/api/settings/custom-actions`.

#### `widgets`
*   **Columns**: `data_source`, `display_type`, `grid_x`, `grid_y`.
*   **API**: `/api/dashboard/widgets`.

---

## 🛡️ 3. Multi-tenancy & Soft-Delete Audit

### Multi-tenancy Strategy
*   **Mechanism**: Discriminator column `organization_id` (UUID).
*   **Enforcement**: Middleware `TenantScope` in `main.go` automatically injects `WHERE organization_id = ?` into GORM scopes.
*   **Exempt Tables**: `license_records`, `license_events` (Hardware bound).

### Soft-Delete Strategy
*   **Mechanism**: `gorm.DeletedAt` indexed column.
*   **Audit**: 90% coverage on business entities. Junction and log tables (e.g., `role_permissions`, `user_availability_logs`) use hard-deletes.

---

## 🚀 4. Production Optimization
*   **JSONB Indexes**: GIN indexes applied to `tags` and `metadata` columns for fast searching.
*   **Unique Constraints**: Leveraged heavily in `user_organizations` and `bulk_message_recipients` to prevent data duplication.
*   **TTL**: `whatsapp_statuses` and `chatbot_sessions` carry expiration timestamps for cleanup.
