-- ============================================================
-- Migration 001: Row-Level Security for Multi-tenancy
-- 
-- Adds tenant_id to all tables and installs RLS policies.
-- Each API request must set app.current_tenant_id in the session.
-- ============================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================================
-- Step 1: Add tenant_id column to all tables
-- ============================================================

ALTER TABLE pages         ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE customers     ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE escalations   ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE knowledge_base ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE settings      ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

-- ============================================================
-- Step 2: Create tenants table
-- ============================================================

CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR NOT NULL,
    slug        VARCHAR NOT NULL UNIQUE,
    plan        VARCHAR NOT NULL DEFAULT 'starter',  -- starter | professional | enterprise
    is_active   BOOLEAN NOT NULL DEFAULT true,
    meta_app_id VARCHAR,
    meta_app_secret VARCHAR,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default tenant (existing data)
INSERT INTO tenants (id, name, slug, plan)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default', 'professional')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Step 3: Create helper function for current tenant
-- ============================================================

CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true)::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN '00000000-0000-0000-0000-000000000001'::UUID;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- ============================================================
-- Step 4: Enable RLS on all tables
-- ============================================================

ALTER TABLE pages          ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversations  ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers      ENABLE ROW LEVEL SECURITY;
ALTER TABLE escalations    ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_base ENABLE ROW LEVEL SECURITY;
ALTER TABLE settings       ENABLE ROW LEVEL SECURITY;

-- ============================================================
-- Step 5: Create RLS policies
-- ============================================================

-- Pages
DROP POLICY IF EXISTS pages_tenant_isolation ON pages;
CREATE POLICY pages_tenant_isolation ON pages
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Conversations
DROP POLICY IF EXISTS conversations_tenant_isolation ON conversations;
CREATE POLICY conversations_tenant_isolation ON conversations
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Customers
DROP POLICY IF EXISTS customers_tenant_isolation ON customers;
CREATE POLICY customers_tenant_isolation ON customers
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Escalations
DROP POLICY IF EXISTS escalations_tenant_isolation ON escalations;
CREATE POLICY escalations_tenant_isolation ON escalations
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Knowledge Base
DROP POLICY IF EXISTS knowledge_base_tenant_isolation ON knowledge_base;
CREATE POLICY knowledge_base_tenant_isolation ON knowledge_base
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Settings
DROP POLICY IF EXISTS settings_tenant_isolation ON settings;
CREATE POLICY settings_tenant_isolation ON settings
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- ============================================================
-- Step 6: Indexes for tenant queries
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_pages_tenant_id          ON pages(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_id  ON conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_customers_tenant_id      ON customers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_escalations_tenant_id    ON escalations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_base_tenant_id ON knowledge_base(tenant_id);
CREATE INDEX IF NOT EXISTS idx_settings_tenant_id       ON settings(tenant_id);

-- ============================================================
-- Step 7: Admin bypass policy (for service role)
-- ============================================================

-- Superuser/service role bypasses RLS automatically in PostgreSQL.
-- For app-level admin access, grant BYPASSRLS to service account:
-- ALTER ROLE your_service_user BYPASSRLS;
