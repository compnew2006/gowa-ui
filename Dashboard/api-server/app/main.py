from fastapi import FastAPI, Response, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException
from contextlib import asynccontextmanager
from sqlalchemy import text
import json

from app.db import engine
from app.config import get_settings
from app.middleware.logging import (
    configure_logging, CorrelationIDMiddleware, RequestLoggingMiddleware,
)
from app.routers import (
    health, pages, conversations, customers, escalations,
    knowledge_base, analytics, settings_router, tokens, webhook,
)
from app.routers import shadow_mode, prompt_versions, audit_logs, custom_ai_models
from app.routers import teams, rules, bulk, integrations_router, compliance, advanced_analytics
from app.routers import campaigns as campaigns_router
from app.routers import posts as posts_router
from app.routers import console as console_router

settings = get_settings()
configure_logging(settings.environment)


def _init_sentry():
    if not settings.sentry_dsn:
        return
    try:
        import sentry_sdk
        from sentry_sdk.integrations.fastapi import FastApiIntegration
        from sentry_sdk.integrations.sqlalchemy import SqlalchemyIntegration
        sentry_sdk.init(
            dsn=settings.sentry_dsn,
            environment=settings.environment,
            integrations=[
                FastApiIntegration(transaction_style="endpoint"),
                SqlalchemyIntegration(),
            ],
            traces_sample_rate=0.2,
            profiles_sample_rate=0.1,
            send_default_pii=False,
        )
        print(f"[Sentry] Initialized (env={settings.environment})")
    except Exception as e:
        print(f"[Sentry] Init failed: {e}")


_init_sentry()

_LIFESPAN_SQL = [
    "CREATE EXTENSION IF NOT EXISTS vector",
    "CREATE EXTENSION IF NOT EXISTS pgcrypto",
    # knowledge_base embedding column
    """ALTER TABLE knowledge_base ADD COLUMN IF NOT EXISTS embedding vector(1536)""",
    # conversations v1 → v2
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS urgency VARCHAR DEFAULT 'normal'",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS priority VARCHAR DEFAULT 'normal'",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS guardrail_triggered BOOLEAN DEFAULT FALSE",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS guardrail_reason VARCHAR",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS shadow_approved_at TIMESTAMPTZ",
    # conversations v3 (PII + rules)
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS pii_detected BOOLEAN DEFAULT FALSE",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS pii_masked_reply TEXT",
    "ALTER TABLE conversations ADD COLUMN IF NOT EXISTS matched_rule_id VARCHAR",
    # customers v2
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS lead_score INTEGER DEFAULT 0",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS purchase_intent VARCHAR DEFAULT 'Low'",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS conversion_status VARCHAR DEFAULT 'prospect'",
    # customers v3 (predictive CRM + GDPR)
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS churn_risk VARCHAR DEFAULT 'low'",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS churn_risk_score FLOAT DEFAULT 0.0",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS next_best_action VARCHAR",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS re_engage_sent_at TIMESTAMPTZ",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS gdpr_deleted BOOLEAN DEFAULT FALSE",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS gdpr_export_requested_at TIMESTAMPTZ",
    # escalations v2 (team assignment)
    "ALTER TABLE escalations ADD COLUMN IF NOT EXISTS assigned_to VARCHAR",
    # settings v2
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS warmup_mode BOOLEAN DEFAULT TRUE",
    # New tables: admin_users
    """CREATE TABLE IF NOT EXISTS admin_users (
        id UUID PRIMARY KEY,
        email VARCHAR UNIQUE NOT NULL,
        name VARCHAR NOT NULL,
        role VARCHAR DEFAULT 'reviewer',
        is_active BOOLEAN DEFAULT TRUE,
        avatar_url VARCHAR,
        permissions JSONB DEFAULT '{}',
        last_active_at TIMESTAMPTZ,
        telegram_user_id VARCHAR,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    )""",
    # audit_logs
    """CREATE TABLE IF NOT EXISTS audit_logs (
        id UUID PRIMARY KEY,
        admin_id VARCHAR,
        admin_name VARCHAR DEFAULT 'system',
        action VARCHAR NOT NULL,
        entity_type VARCHAR NOT NULL,
        entity_id VARCHAR,
        details JSONB DEFAULT '{}',
        ip_address VARCHAR,
        created_at TIMESTAMPTZ DEFAULT NOW()
    )""",
    # automation_rules
    """CREATE TABLE IF NOT EXISTS automation_rules (
        id UUID PRIMARY KEY,
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
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    )""",
    # integration_configs
    """CREATE TABLE IF NOT EXISTS integration_configs (
        id UUID PRIMARY KEY,
        type VARCHAR NOT NULL,
        name VARCHAR NOT NULL,
        config JSONB DEFAULT '{}',
        is_active BOOLEAN DEFAULT FALSE,
        trigger_events VARCHAR[] DEFAULT '{}',
        last_triggered_at TIMESTAMPTZ,
        trigger_count INTEGER DEFAULT 0,
        last_error VARCHAR,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    )""",
    # Performance indexes
    "CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status)",
    "CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC)",
    "CREATE INDEX IF NOT EXISTS idx_conversations_customer_id ON conversations(customer_id)",
    "CREATE INDEX IF NOT EXISTS idx_conversations_page_id ON conversations(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_conversations_language ON conversations(language)",
    "CREATE INDEX IF NOT EXISTS idx_customers_churn_risk ON customers(churn_risk)",
    "CREATE INDEX IF NOT EXISTS idx_customers_conversion_status ON customers(conversion_status)",
    "CREATE INDEX IF NOT EXISTS idx_customers_lead_score ON customers(lead_score DESC)",
    "CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status)",
    "CREATE INDEX IF NOT EXISTS idx_escalations_priority ON escalations(priority)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type)",
    # Default safe replies columns
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS safe_reply_ar TEXT DEFAULT 'شكراً لتواصلك معنا. سيتواصل معك أحد ممثلي خدمة العملاء في أقرب وقت ممكن.'""",
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS safe_reply_en TEXT DEFAULT 'Thank you for reaching out. A customer service representative will contact you shortly.'""",
    # Public reply & reply mode settings
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS public_reply_message_ar TEXT DEFAULT 'تم التواصل معك على الخاص'""",
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS public_reply_message_en TEXT DEFAULT 'We've contacted you privately'""",
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS reply_mode VARCHAR(20) DEFAULT 'both'""",
    # Settings: auto-reply date range
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS auto_reply_start_date TIMESTAMPTZ""",
    """ALTER TABLE settings ADD COLUMN IF NOT EXISTS auto_reply_end_date TIMESTAMPTZ""",
    # Campaigns table
    """
    CREATE TABLE IF NOT EXISTS campaigns (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    """,
    "CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status)",
    "CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC)",
    # Multi-page isolation: page_id columns
    "ALTER TABLE knowledge_base ADD COLUMN IF NOT EXISTS page_id TEXT",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS page_id TEXT",
    "ALTER TABLE customers ADD COLUMN IF NOT EXISTS page_id TEXT",
    "ALTER TABLE automation_rules ADD COLUMN IF NOT EXISTS page_id TEXT",
    "ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS page_id TEXT",
    "CREATE INDEX IF NOT EXISTS idx_knowledge_base_page_id ON knowledge_base(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_settings_page_id ON settings(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_customers_page_id ON customers(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_automation_rules_page_id ON automation_rules(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_campaigns_page_id ON campaigns(page_id)",
    # Audit logs extensions
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS page_id TEXT",
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS old_values JSONB",
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS new_values JSONB",
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS reason TEXT",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_page_id ON audit_logs(page_id)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_admin_id ON audit_logs(admin_id)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
    "CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC)",
    # Dynamic Custom AI Models
    """CREATE TABLE IF NOT EXISTS custom_ai_models (
        id UUID PRIMARY KEY,
        name VARCHAR NOT NULL,
        provider VARCHAR NOT NULL,
        model_name VARCHAR NOT NULL,
        api_key_encrypted VARCHAR NOT NULL,
        api_base VARCHAR,
        is_active BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    )""",
    # Brand Kit settings columns
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_description TEXT",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_industry VARCHAR",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_target_audience TEXT",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_tone_of_voice VARCHAR",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_preferred_hashtags TEXT",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_restricted_words TEXT",
    "ALTER TABLE settings ADD COLUMN IF NOT EXISTS brand_sample_posts TEXT",
]


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Create all tables from SQLAlchemy metadata
    from app.db import Base
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
        for sql in _LIFESPAN_SQL:
            try:
                await conn.execute(text(sql))
            except Exception:
                pass  # Column already exists or extension unavailable
    yield
    await engine.dispose()


app = FastAPI(
    title="AI Automation API",
    description="AI-powered Facebook & Instagram Comment Automation — FastAPI + LangGraph + DSPy",
    version="3.0.0",
    lifespan=lifespan,
)

# Sentry ASGI middleware (must be outermost)
if settings.sentry_dsn:
    try:
        from sentry_sdk.integrations.asgi import SentryAsgiMiddleware
        app.add_middleware(SentryAsgiMiddleware)
    except Exception:
        pass

# Rate limiting middleware
from app.middleware.rate_limit import RateLimitMiddleware
app.add_middleware(RequestLoggingMiddleware)
app.add_middleware(CorrelationIDMiddleware)
app.add_middleware(RateLimitMiddleware)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Exception handlers for proper error responses
from app.middleware.error_handler import (
    http_exception_handler,
    validation_exception_handler,
    json_decode_exception_handler,
    generic_exception_handler,
)

app.add_exception_handler(StarletteHTTPException, http_exception_handler)
app.add_exception_handler(RequestValidationError, validation_exception_handler)
app.add_exception_handler(json.JSONDecodeError, json_decode_exception_handler)
app.add_exception_handler(Exception, generic_exception_handler)


# Prometheus metrics endpoint
@app.get("/api/metrics", include_in_schema=False)
async def metrics():
    from app.metrics import get_metrics_response
    content, content_type = get_metrics_response()
    return Response(content=content, media_type=content_type)


# ── API v1 routers ─────────────────────────────────────────
# All routers are mounted under /api/v1 for versioned access.
# Backward-compatible /api aliases are included below.

# Core routers
app.include_router(health.router, prefix="/api/v1")
app.include_router(pages.router, prefix="/api/v1")
app.include_router(conversations.router, prefix="/api/v1")
app.include_router(customers.router, prefix="/api/v1")
app.include_router(escalations.router, prefix="/api/v1")
app.include_router(knowledge_base.router, prefix="/api/v1")
app.include_router(analytics.router, prefix="/api/v1")
app.include_router(settings_router.router, prefix="/api/v1")
app.include_router(tokens.router, prefix="/api/v1")
app.include_router(webhook.router, prefix="/api/v1")

# PRD v2.0 routers
app.include_router(shadow_mode.router, prefix="/api/v1")
app.include_router(prompt_versions.router, prefix="/api/v1")

# v3.0 — 10 Feature Expansions
app.include_router(teams.router, prefix="/api/v1")
app.include_router(rules.router, prefix="/api/v1")
app.include_router(bulk.router, prefix="/api/v1")
app.include_router(integrations_router.router, prefix="/api/v1")
app.include_router(compliance.router, prefix="/api/v1")
app.include_router(advanced_analytics.router, prefix="/api/v1")

# Campaigns & Audit
app.include_router(campaigns_router.router, prefix="/api/v1")
app.include_router(audit_logs.router, prefix="/api/v1")
app.include_router(custom_ai_models.router, prefix="/api/v1")


# ── Backward-compatible /api aliases ───────────────────────
# These ensure existing clients (dashboard) continue to work
# without changes. Can be deprecated in a future major version.

app.include_router(health.router, prefix="/api")
app.include_router(pages.router, prefix="/api")
app.include_router(conversations.router, prefix="/api")
app.include_router(customers.router, prefix="/api")
app.include_router(escalations.router, prefix="/api")
app.include_router(knowledge_base.router, prefix="/api")
app.include_router(analytics.router, prefix="/api")
app.include_router(settings_router.router, prefix="/api")
app.include_router(tokens.router, prefix="/api")
app.include_router(webhook.router, prefix="/api")
app.include_router(shadow_mode.router, prefix="/api")
app.include_router(prompt_versions.router, prefix="/api")
app.include_router(teams.router, prefix="/api")
app.include_router(rules.router, prefix="/api")
app.include_router(bulk.router, prefix="/api")
app.include_router(integrations_router.router, prefix="/api")
app.include_router(compliance.router, prefix="/api")
app.include_router(advanced_analytics.router, prefix="/api")
app.include_router(campaigns_router.router, prefix="/api")
app.include_router(posts_router.router, prefix="/api")
app.include_router(console_router.router, prefix="/api")
app.include_router(audit_logs.router, prefix="/api")
app.include_router(custom_ai_models.router, prefix="/api")
