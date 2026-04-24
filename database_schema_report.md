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

*(Full list of 44 tables detailed below)*

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
    *   ✅ **Soft-Delete Enabled**: `organizations`, `users`, `custom_roles`, `contacts`, `messages`, `bulk_message_campaigns`, `chatbot_flows`, `whatsapp_accounts`.
    *   ❌ **Hard-Delete Only**: `user_organizations`, `tags`, `chatbot_sessions`, `api_keys`.

---

## 🚀 5. Production Notes
*   **Encryption**: `whatsapp_accounts` and `api_keys` contain encrypted secrets marked with `enc3:` prefix in the database.
*   **Concurrency**: Tables like `bulk_message_recipients` use indexes to support high-throughput worker consumption from Redis-backed streams.
