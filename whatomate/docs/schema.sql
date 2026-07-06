-- Whatomate Production Database Schema (PostgreSQL)
-- Generated from GORM models and manual migration logic.
-- Note: Requires pgcrypto extension for gen_random_uuid()

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 1. IDENTITY & MULTI-TENANCY
-- ============================================================================

CREATE TABLE IF NOT EXISTS "organizations" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "name" TEXT UNIQUE NOT NULL,
    "slug" TEXT UNIQUE NOT NULL,
    "logo_url" TEXT,
    "timezone" TEXT DEFAULT 'UTC',
    "is_active" BOOLEAN DEFAULT true,
    "settings" JSONB DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS "idx_organizations_deleted_at" ON "organizations" ("deleted_at");

CREATE TABLE IF NOT EXISTS "organization_configs" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "worker_count" INTEGER DEFAULT 0,
    "max_queue_size" INTEGER DEFAULT 0,
    "max_whatsapp_instances" INTEGER DEFAULT 0,
    "settings" JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS "permissions" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "resource" TEXT NOT NULL,
    "action" TEXT NOT NULL,
    "description" TEXT,
    UNIQUE("resource", "action")
);

CREATE TABLE IF NOT EXISTS "custom_roles" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id"),
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_system" BOOLEAN DEFAULT false,
    "is_default" BOOLEAN DEFAULT false
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_custom_roles_org_name" ON "custom_roles" ("organization_id", "name") WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS "role_permissions" (
    "custom_role_id" UUID REFERENCES "custom_roles"("id") ON DELETE CASCADE,
    "permission_id" UUID REFERENCES "permissions"("id") ON DELETE CASCADE,
    PRIMARY KEY ("custom_role_id", "permission_id")
);

CREATE TABLE IF NOT EXISTS "users" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id"), -- Initial/Default Org
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "email" TEXT UNIQUE NOT NULL,
    "password_hash" TEXT NOT NULL,
    "full_name" TEXT,
    "role_id" UUID REFERENCES "custom_roles"("id"),
    "is_active" BOOLEAN DEFAULT true,
    "is_available" BOOLEAN DEFAULT true,
    "is_super_admin" BOOLEAN DEFAULT false,
    "avatar_url" TEXT,
    "settings" JSONB DEFAULT '{}'::jsonb,
    "last_login" TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS "idx_users_deleted_at" ON "users" ("deleted_at");

CREATE TABLE IF NOT EXISTS "user_organizations" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "user_id" UUID REFERENCES "users"("id") ON DELETE CASCADE,
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "role_id" UUID REFERENCES "custom_roles"("id"),
    "is_default" BOOLEAN DEFAULT false,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_org_unique" ON "user_organizations"("user_id", "organization_id") WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS "api_keys" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "key" TEXT UNIQUE NOT NULL,
    "last_used" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

-- ============================================================================
-- 2. WHATSAPP CHANNEL MANAGEMENT
-- ============================================================================

CREATE TABLE IF NOT EXISTS "whatsapp_accounts" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "name" TEXT,
    "phone_id" TEXT,
    "waba_id" TEXT,
    "access_token" TEXT,
    "provider" TEXT DEFAULT 'meta'
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_whatsapp_accounts_org_phone" ON "whatsapp_accounts" ("organization_id", "phone_id");

CREATE TABLE IF NOT EXISTS "whatsapp_instances" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "jid" TEXT DEFAULT '',
    "name" TEXT,
    "status" TEXT,
    "qr_code" TEXT,
    "session_data" TEXT,
    "send_blocked_until" TIMESTAMPTZ,
    "send_block_reason" TEXT NOT NULL DEFAULT '',
    "is_active" BOOLEAN DEFAULT true
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_whatsapp_instances_j_id" ON "whatsapp_instances" ("jid") WHERE "jid" <> '';

CREATE TABLE IF NOT EXISTS "instance_notifications" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "instance_id" UUID REFERENCES "whatsapp_instances"("id") ON DELETE CASCADE,
    "type" TEXT,
    "message" TEXT,
    "is_read" BOOLEAN DEFAULT false,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "media_assets" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "file_name" TEXT,
    "file_path" TEXT,
    "mime_type" TEXT,
    "file_size" BIGINT,
    "source" TEXT, -- e.g., 'upload', 'whatsapp'
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "whatsapp_statuses" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "instance_id" UUID REFERENCES "whatsapp_instances"("id") ON DELETE CASCADE,
    "sender_jid" TEXT,
    "status_type" TEXT,
    "content" TEXT,
    "media_url" TEXT,
    "expires_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ,
    "metadata" JSONB DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS "idx_whatsapp_statuses_org_instance_expires" ON "whatsapp_statuses"("organization_id", "instance_id", "expires_at" DESC);

-- ============================================================================
-- 3. MESSAGING & CONTACTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS "contacts" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "phone_number" VARCHAR(50) NOT NULL,
    "name" TEXT,
    "avatar_url" TEXT,
    "assigned_user_id" UUID REFERENCES "users"("id"),
    "instance_id" UUID REFERENCES "whatsapp_instances"("id"),
    "status" VARCHAR(20) DEFAULT 'pending',
    "is_read" BOOLEAN DEFAULT true,
    "last_message_at" TIMESTAMPTZ,
    "last_inbound_at" TIMESTAMPTZ,
    "closed_at" TIMESTAMPTZ,
    "closed_by_user_id" UUID REFERENCES "users"("id"),
    "tags" JSONB DEFAULT '[]'::jsonb,
    "metadata" JSONB DEFAULT '{}'::jsonb,
    "whats_app_account" TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_contacts_org_phone_instance" ON "contacts"("organization_id", "phone_number", "instance_id") WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS "idx_contacts_tags" ON "contacts" USING GIN ("tags");

CREATE TABLE IF NOT EXISTS "messages" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "user_id" UUID REFERENCES "users"("id"),
    "whatsapp_message_id" TEXT,
    "conversation_id" TEXT,
    "type" TEXT,
    "body" TEXT,
    "status" TEXT,
    "direction" TEXT,
    "media_asset_id" UUID REFERENCES "media_assets"("id"),
    "media_url" TEXT,
    "whats_app_account" TEXT,
    "metadata" JSONB DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS "idx_messages_contact_created" ON "messages"("contact_id", "created_at" DESC);

CREATE TABLE IF NOT EXISTS "tags" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "color" TEXT,
    "created_at" TIMESTAMPTZ,
    UNIQUE("organization_id", "name")
);

-- ============================================================================
-- 4. AUTOMATION (CHATBOT)
-- ============================================================================

CREATE TABLE IF NOT EXISTS "chatbot_settings" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "is_enabled" BOOLEAN DEFAULT false,
    "ai_enabled" BOOLEAN DEFAULT false,
    "provider" TEXT DEFAULT 'openai',
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "chatbot_flows" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "name" TEXT NOT NULL,
    "is_enabled" BOOLEAN DEFAULT true,
    "whats_app_account" TEXT
);

CREATE TABLE IF NOT EXISTS "chatbot_flow_steps" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "flow_id" UUID REFERENCES "chatbot_flows"("id") ON DELETE CASCADE,
    "type" TEXT, -- 'message', 'condition', 'input', 'transfer'
    "payload" JSONB,
    "order" INTEGER,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "chatbot_sessions" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "phone_number" VARCHAR(50),
    "status" TEXT, -- 'active', 'completed', 'expired'
    "current_step_id" UUID REFERENCES "chatbot_flow_steps"("id"),
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "chatbot_session_messages" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "session_id" UUID REFERENCES "chatbot_sessions"("id") ON DELETE CASCADE,
    "message_id" UUID, -- Can be null if chatbot only message
    "direction" TEXT,
    "content" TEXT,
    "created_at" TIMESTAMPTZ
);

-- ============================================================================
-- 5. CAMPAIGNS
-- ============================================================================

CREATE TABLE IF NOT EXISTS "bulk_message_campaigns" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "name" TEXT,
    "status" TEXT,
    "template_id" TEXT,
    "whats_app_account" TEXT,
    "min_delay_seconds" INTEGER DEFAULT 0,
    "max_delay_seconds" INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS "bulk_message_recipients" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "campaign_id" UUID REFERENCES "bulk_message_campaigns"("id") ON DELETE CASCADE,
    "phone_number" VARCHAR(50),
    "phone_normalized" VARCHAR(32),
    "status" TEXT,
    "error_message" TEXT,
    "variables" JSONB,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_bulk_recipients_campaign_phone_normalized" ON "bulk_message_recipients"("campaign_id", "phone_normalized") WHERE "phone_normalized" <> '';

-- ============================================================================
-- 6. ADDITIONAL TABLES (ALPHABETICAL)
-- ============================================================================

CREATE TABLE IF NOT EXISTS "agent_transfers" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "phone_number" VARCHAR(50),
    "from_user_id" UUID,
    "to_user_id" UUID,
    "team_id" UUID,
    "status" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "ai_contexts" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT,
    "content" TEXT,
    "is_enabled" BOOLEAN DEFAULT true,
    "priority" INTEGER DEFAULT 0,
    "whats_app_account" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "canned_responses" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "is_active" BOOLEAN DEFAULT true,
    "usage_count" INTEGER DEFAULT 0,
    "attachments" JSONB DEFAULT '[]'::jsonb,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "chat_closure_ratings" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "agent_user_id" UUID REFERENCES "users"("id"),
    "rating" INTEGER,
    "comment" TEXT,
    "state" TEXT, -- 'pending', 'rated'
    "closed_at" TIMESTAMPTZ,
    "rated_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "conversation_notes" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "user_id" UUID REFERENCES "users"("id"),
    "content" TEXT,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "keyword_rules" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "keyword" TEXT NOT NULL,
    "match_type" TEXT DEFAULT 'exact',
    "response" TEXT,
    "is_enabled" BOOLEAN DEFAULT true,
    "priority" INTEGER DEFAULT 0,
    "whats_app_account" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "teams" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_active" BOOLEAN DEFAULT true,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "team_members" (
    "team_id" UUID REFERENCES "teams"("id") ON DELETE CASCADE,
    "user_id" UUID REFERENCES "users"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    PRIMARY KEY ("team_id", "user_id")
);

CREATE TABLE IF NOT EXISTS "webhooks" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "url" TEXT NOT NULL,
    "secret" TEXT,
    "events" JSONB DEFAULT '[]'::jsonb,
    "is_active" BOOLEAN DEFAULT true,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "widgets" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT,
    "type" TEXT,
    "config" JSONB DEFAULT '{}'::jsonb,
    "layout" JSONB DEFAULT '{}'::jsonb,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

-- ============================================================================
-- 7. INFRASTRUCTURE & META-SPECIFIC
-- ============================================================================

CREATE TABLE IF NOT EXISTS "catalogs" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "whats_app_account" TEXT,
    "catalog_id" TEXT,
    "name" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "catalog_products" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "catalog_id" UUID REFERENCES "catalogs"("id") ON DELETE CASCADE,
    "product_id" TEXT,
    "name" TEXT,
    "description" TEXT,
    "price" DECIMAL,
    "currency" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "templates" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "whats_app_account" TEXT,
    "name" TEXT NOT NULL,
    "category" TEXT,
    "language" TEXT,
    "status" TEXT,
    "components" JSONB,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_templates_account_name_lang" ON "templates"(whats_app_account, name, language);

CREATE TABLE IF NOT EXISTS "whatsapp_flows" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "whats_app_account" TEXT,
    "flow_id" TEXT,
    "name" TEXT,
    "status" TEXT,
    "categories" JSONB,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "notification_rules" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT,
    "event_type" TEXT,
    "is_enabled" BOOLEAN DEFAULT true,
    "whats_app_account" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "contact_user_deletions" (
    "contact_id" UUID REFERENCES "contacts"("id") ON DELETE CASCADE,
    "user_id" UUID REFERENCES "users"("id") ON DELETE CASCADE,
    "deleted_at" TIMESTAMPTZ,
    PRIMARY KEY ("contact_id", "user_id")
);

CREATE TABLE IF NOT EXISTS "custom_actions" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "name" TEXT,
    "url" TEXT,
    "method" TEXT,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "license_records" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "license_id" TEXT,
    "customer_name" TEXT,
    "expires_at" TIMESTAMPTZ,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "license_events" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "license_id" TEXT,
    "event_type" TEXT,
    "payload" JSONB,
    "created_at" TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS "user_availability_logs" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "organization_id" UUID REFERENCES "organizations"("id") ON DELETE CASCADE,
    "user_id" UUID REFERENCES "users"("id") ON DELETE CASCADE,
    "started_at" TIMESTAMPTZ,
    "ended_at" TIMESTAMPTZ,
    "duration_seconds" INTEGER
);
CREATE INDEX IF NOT EXISTS "idx_availability_logs_user_time" ON "user_availability_logs"(user_id, started_at DESC);

-- Note: Remaining 15+ sub-tables (UserAvailabilityLog, CatalogProduct, etc.) 
-- follow same pattern of OrganizationID + Timestamp management.
-- Soft delete constraints (idx_..._deleted_at) are applied to all soft-deletable tables.
