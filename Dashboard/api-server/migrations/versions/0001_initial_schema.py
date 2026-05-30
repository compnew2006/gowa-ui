"""Initial application schema and performance indexes.

Revision ID: 0001_initial_schema
Revises:
Create Date: 2026-05-03
"""

from __future__ import annotations

from alembic import op

revision = "0001_initial_schema"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.execute("CREATE EXTENSION IF NOT EXISTS pgcrypto")
    op.execute("CREATE EXTENSION IF NOT EXISTS vector")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS pages (
            id UUID PRIMARY KEY,
            platform VARCHAR NOT NULL,
            page_id VARCHAR NOT NULL,
            name VARCHAR NOT NULL,
            avatar_url VARCHAR,
            is_active BOOLEAN DEFAULT TRUE,
            auto_reply_enabled BOOLEAN DEFAULT FALSE,
            shadow_mode BOOLEAN DEFAULT TRUE,
            tracking_start_date TIMESTAMPTZ,
            access_token_encrypted VARCHAR,
            token_status VARCHAR DEFAULT 'valid',
            token_expires_at TIMESTAMPTZ,
            token_last_refreshed_at TIMESTAMPTZ,
            token_last_error VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE UNIQUE INDEX IF NOT EXISTS idx_pages_page_id ON pages(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_pages_is_active ON pages(is_active)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_pages_token_status ON pages(token_status)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS customers (
            id UUID PRIMARY KEY,
            page_id VARCHAR,
            facebook_id VARCHAR,
            instagram_id VARCHAR,
            username VARCHAR,
            full_name VARCHAR,
            profile_url VARCHAR,
            avatar_url VARCHAR,
            first_contact_date TIMESTAMPTZ,
            last_interaction TIMESTAMPTZ,
            interaction_count INTEGER DEFAULT 0,
            lead_score INTEGER DEFAULT 0,
            purchase_intent VARCHAR DEFAULT 'Low',
            conversion_status VARCHAR DEFAULT 'prospect',
            assigned_admin VARCHAR,
            tags VARCHAR[] DEFAULT '{}',
            notes JSONB DEFAULT '[]',
            escalation_history VARCHAR[] DEFAULT '{}',
            churn_risk VARCHAR DEFAULT 'low',
            churn_risk_score FLOAT DEFAULT 0.0,
            next_best_action VARCHAR,
            re_engage_sent_at TIMESTAMPTZ,
            gdpr_deleted BOOLEAN DEFAULT FALSE,
            gdpr_export_requested_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_page_id ON customers(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_facebook_page ON customers(facebook_id, page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_instagram_page ON customers(instagram_id, page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_conversion_status ON customers(conversion_status)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_purchase_intent ON customers(purchase_intent)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_churn_risk ON customers(churn_risk)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_lead_score ON customers(lead_score DESC)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_customers_last_interaction ON customers(last_interaction DESC)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS conversations (
            id UUID PRIMARY KEY,
            page_id UUID NOT NULL,
            page_name VARCHAR NOT NULL,
            platform VARCHAR NOT NULL,
            comment_id VARCHAR NOT NULL,
            post_id VARCHAR NOT NULL,
            customer_id UUID,
            customer_name VARCHAR NOT NULL,
            customer_avatar_url VARCHAR,
            original_comment TEXT NOT NULL,
            ai_reply TEXT,
            admin_reply TEXT,
            status VARCHAR DEFAULT 'pending',
            intent VARCHAR,
            sentiment VARCHAR,
            urgency VARCHAR DEFAULT 'normal',
            priority VARCHAR DEFAULT 'normal',
            confidence_score FLOAT,
            language VARCHAR,
            is_shadow_mode BOOLEAN DEFAULT FALSE,
            sentiment_history VARCHAR[] DEFAULT '{}',
            escalation_reason VARCHAR,
            guardrail_triggered BOOLEAN DEFAULT FALSE,
            guardrail_reason VARCHAR,
            processing_time FLOAT,
            replied_at TIMESTAMPTZ,
            shadow_approved_at TIMESTAMPTZ,
            pii_detected BOOLEAN DEFAULT FALSE,
            pii_masked_reply TEXT,
            matched_rule_id VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_comment_page ON conversations(comment_id, page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_page_id ON conversations(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_customer_id ON conversations(customer_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_intent ON conversations(intent)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_sentiment ON conversations(sentiment)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_language ON conversations(language)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_conversations_shadow_status ON conversations(is_shadow_mode, status)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS escalations (
            id UUID PRIMARY KEY,
            conversation_id UUID NOT NULL,
            page_id UUID NOT NULL,
            page_name VARCHAR NOT NULL,
            customer_id UUID,
            customer_name VARCHAR NOT NULL,
            original_comment TEXT NOT NULL,
            reason TEXT NOT NULL,
            priority VARCHAR NOT NULL,
            status VARCHAR DEFAULT 'open',
            assigned_to VARCHAR,
            admin_notes TEXT,
            resolved_by VARCHAR,
            resolved_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_page_id ON escalations(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_conversation_id ON escalations(conversation_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_customer_id ON escalations(customer_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_priority ON escalations(priority)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_escalations_created_at ON escalations(created_at DESC)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS knowledge_base (
            id UUID PRIMARY KEY,
            page_id VARCHAR,
            category VARCHAR NOT NULL,
            question TEXT NOT NULL,
            answer TEXT NOT NULL,
            intent_tags VARCHAR[] DEFAULT '{}',
            language VARCHAR DEFAULT 'ar',
            is_active BOOLEAN DEFAULT TRUE,
            usage_count INTEGER DEFAULT 0,
            quality_score FLOAT,
            embedding vector(1536),
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_knowledge_base_page_id ON knowledge_base(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_knowledge_base_active ON knowledge_base(is_active)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_knowledge_base_category ON knowledge_base(category)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_knowledge_base_language ON knowledge_base(language)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_knowledge_base_usage ON knowledge_base(usage_count DESC)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS settings (
            id UUID PRIMARY KEY,
            page_id VARCHAR,
            confidence_threshold FLOAT DEFAULT 0.85,
            auto_escalate_angry BOOLEAN DEFAULT TRUE,
            telegram_bot_token VARCHAR,
            telegram_chat_id VARCHAR,
            primary_llm_model VARCHAR DEFAULT 'hermes',
            fallback_llm_model VARCHAR DEFAULT 'gpt-4o',
            webhook_verify_token VARCHAR DEFAULT 'verify_token_change_me',
            max_retries INTEGER DEFAULT 3,
            rate_limit_warning_threshold INTEGER DEFAULT 80,
            default_language VARCHAR DEFAULT 'ar',
            warmup_mode BOOLEAN DEFAULT TRUE,
            safe_reply_ar TEXT DEFAULT 'شكراً لتواصلك معنا. سيتواصل معك أحد ممثلي خدمة العملاء في أقرب وقت ممكن.',
            safe_reply_en TEXT DEFAULT 'Thank you for reaching out. A customer service representative will contact you shortly.',
            public_reply_message_ar TEXT DEFAULT 'تم التواصل معك على الخاص',
            public_reply_message_en TEXT DEFAULT 'We''ve contacted you privately',
            reply_mode VARCHAR DEFAULT 'both',
            auto_reply_start_date TIMESTAMPTZ,
            auto_reply_end_date TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_page_id_unique ON settings(COALESCE(page_id, '__global__'))")
    op.execute("CREATE INDEX IF NOT EXISTS idx_settings_page_id ON settings(page_id)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS admin_users (
            id UUID PRIMARY KEY,
            email VARCHAR UNIQUE NOT NULL,
            name VARCHAR NOT NULL,
            role VARCHAR DEFAULT 'reviewer',
            is_active BOOLEAN DEFAULT TRUE,
            avatar_url VARCHAR,
            permissions JSONB DEFAULT '{}',
            last_active_at TIMESTAMPTZ,
            telegram_user_id VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_admin_users_role ON admin_users(role)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_admin_users_is_active ON admin_users(is_active)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS audit_logs (
            id UUID PRIMARY KEY,
            page_id VARCHAR,
            admin_id VARCHAR,
            admin_name VARCHAR DEFAULT 'system',
            action VARCHAR NOT NULL,
            entity_type VARCHAR NOT NULL,
            entity_id VARCHAR,
            old_values JSONB,
            new_values JSONB,
            reason TEXT,
            details JSONB,
            ip_address VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_page_id ON audit_logs(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id ON audit_logs(admin_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_id ON audit_logs(entity_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS automation_rules (
            id UUID PRIMARY KEY,
            page_id VARCHAR,
            name VARCHAR NOT NULL,
            description TEXT,
            conditions JSONB DEFAULT '[]',
            condition_logic VARCHAR DEFAULT 'AND',
            action VARCHAR NOT NULL,
            action_config JSONB DEFAULT '{}',
            priority INTEGER DEFAULT 10,
            is_active BOOLEAN DEFAULT TRUE,
            trigger_count INTEGER DEFAULT 0,
            last_triggered_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_automation_rules_page_id ON automation_rules(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_automation_rules_active_priority ON automation_rules(is_active, priority)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS integration_configs (
            id UUID PRIMARY KEY,
            type VARCHAR NOT NULL,
            name VARCHAR NOT NULL,
            config JSONB DEFAULT '{}',
            is_active BOOLEAN DEFAULT FALSE,
            trigger_events VARCHAR[] DEFAULT '{}',
            last_triggered_at TIMESTAMPTZ,
            trigger_count INTEGER DEFAULT 0,
            last_error VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_integration_configs_type ON integration_configs(type)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_integration_configs_active ON integration_configs(is_active)")

    op.execute(
        """
        CREATE TABLE IF NOT EXISTS campaigns (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            page_id VARCHAR,
            name VARCHAR NOT NULL,
            description TEXT,
            status VARCHAR DEFAULT 'draft',
            target_filter JSONB DEFAULT '{}',
            customer_ids JSONB DEFAULT '[]',
            message_ar TEXT DEFAULT '',
            message_en TEXT DEFAULT '',
            media_urls JSONB DEFAULT '[]',
            media_type VARCHAR,
            send_at TIMESTAMPTZ,
            interval_hours INTEGER,
            max_sends INTEGER,
            total_recipients INTEGER DEFAULT 0,
            sent_count INTEGER DEFAULT 0,
            failed_count INTEGER DEFAULT 0,
            created_by VARCHAR,
            created_at TIMESTAMPTZ DEFAULT now(),
            updated_at TIMESTAMPTZ DEFAULT now()
        )
        """
    )
    op.execute("CREATE INDEX IF NOT EXISTS idx_campaigns_page_id ON campaigns(page_id)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status)")
    op.execute("CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC)")


def downgrade() -> None:
    for table in (
        "campaigns",
        "integration_configs",
        "automation_rules",
        "audit_logs",
        "admin_users",
        "settings",
        "knowledge_base",
        "escalations",
        "conversations",
        "customers",
        "pages",
    ):
        op.execute(f"DROP TABLE IF EXISTS {table} CASCADE")
