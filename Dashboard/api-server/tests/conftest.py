"""
Shared test fixtures for the AI Automation API server.
"""
from __future__ import annotations

import asyncio
import os
import uuid
from datetime import datetime, timezone
from typing import Any, AsyncGenerator, Generator
from unittest.mock import AsyncMock, MagicMock, patch

# Set env vars BEFORE importing any app modules
os.environ.setdefault("DATABASE_URL", "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true")
os.environ.setdefault("REDIS_URL", "redis://localhost:6379/0")
os.environ.setdefault("META_APP_SECRET", "")
os.environ.setdefault("ENVIRONMENT", "test")
os.environ.setdefault("AUTO_APPLY_SCHEMA_PATCHES", "false")
os.environ.setdefault("DB_POOL_SIZE", "5")
os.environ.setdefault("DB_MAX_OVERFLOW", "10")
os.environ.setdefault("DB_POOL_RECYCLE", "1800")
os.environ.setdefault("DB_POOL_TIMEOUT", "30")

import pytest
import pytest_asyncio

# Patch get_settings before any app module loads (DB_POOL_SIZE not a pydantic field)
_mock_cfg = MagicMock()
_mock_cfg.database_url = "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true"
_mock_cfg.async_database_url = "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true"
_mock_cfg.redis_url = "redis://localhost:6379/0"
_mock_cfg.ssl_required = False
_mock_cfg.confidence_auto_reply = 0.85
_mock_cfg.confidence_flag_review = 0.60
_mock_cfg.DB_POOL_SIZE = 5
_mock_cfg.DB_MAX_OVERFLOW = 10
_mock_cfg.DB_POOL_RECYCLE = 1800
_mock_cfg.DB_POOL_TIMEOUT = 30

import app.config as _config_module
_config_module.get_settings = lambda: _mock_cfg

from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import (
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)

SQLALCHEMY_DATABASE_URL = "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true"
test_engine = create_async_engine(
    SQLALCHEMY_DATABASE_URL, echo=False,
    connect_args={"check_same_thread": False},
)
TestSessionLocal = async_sessionmaker(
    test_engine, class_=AsyncSession, expire_on_commit=False,
)

# SQLite-compatible DDL (ARRAY -> TEXT, UUID -> TEXT)
_CREATE_TABLES_SQL = """
CREATE TABLE IF NOT EXISTS pages (
    id TEXT PRIMARY KEY, platform TEXT, page_id TEXT, name TEXT,
    avatar_url TEXT, is_active BOOLEAN DEFAULT 1, auto_reply_enabled BOOLEAN DEFAULT 0,
    shadow_mode BOOLEAN DEFAULT 1, tracking_start_date TEXT,
    access_token_encrypted TEXT, token_status TEXT DEFAULT 'valid',
    token_expires_at TEXT, token_last_refreshed_at TEXT, token_last_error TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY, page_id TEXT, page_name TEXT, platform TEXT,
    comment_id TEXT, post_id TEXT, customer_id TEXT, customer_name TEXT,
    customer_avatar_url TEXT, original_comment TEXT, ai_reply TEXT,
    admin_reply TEXT, status TEXT DEFAULT 'pending', intent TEXT,
    sentiment TEXT, urgency TEXT DEFAULT 'normal', priority TEXT DEFAULT 'normal',
    confidence_score REAL, language TEXT, is_shadow_mode BOOLEAN DEFAULT 0,
    sentiment_history TEXT DEFAULT '[]', escalation_reason TEXT,
    guardrail_triggered BOOLEAN DEFAULT 0, guardrail_reason TEXT,
    processing_time REAL, replied_at TEXT, shadow_approved_at TEXT,
    pii_detected BOOLEAN DEFAULT 0, pii_masked_reply TEXT,
    matched_rule_id TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY, page_id TEXT, facebook_id TEXT, instagram_id TEXT,
    username TEXT, full_name TEXT, profile_url TEXT, avatar_url TEXT,
    first_contact_date TEXT, last_interaction TEXT, interaction_count INTEGER DEFAULT 0,
    lead_score INTEGER DEFAULT 0, purchase_intent TEXT DEFAULT 'Low',
    conversion_status TEXT DEFAULT 'prospect', assigned_admin TEXT,
    tags TEXT DEFAULT '[]', notes TEXT DEFAULT '[]',
    escalation_history TEXT DEFAULT '[]',
    churn_risk TEXT DEFAULT 'low', churn_risk_score REAL DEFAULT 0.0,
    next_best_action TEXT, re_engage_sent_at TEXT,
    gdpr_deleted BOOLEAN DEFAULT 0, gdpr_export_requested_at TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS escalations (
    id TEXT PRIMARY KEY, conversation_id TEXT, page_id TEXT, page_name TEXT,
    customer_id TEXT, customer_name TEXT, original_comment TEXT,
    reason TEXT, priority TEXT, status TEXT DEFAULT 'open',
    assigned_to TEXT, admin_notes TEXT, resolved_by TEXT, resolved_at TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS knowledge_base (
    id TEXT PRIMARY KEY, page_id TEXT, category TEXT, question TEXT,
    answer TEXT, intent_tags TEXT DEFAULT '[]', language TEXT DEFAULT 'ar',
    is_active BOOLEAN DEFAULT 1, usage_count INTEGER DEFAULT 0,
    quality_score REAL,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS settings (
    id TEXT PRIMARY KEY, page_id TEXT, confidence_threshold REAL DEFAULT 0.85,
    auto_escalate_angry BOOLEAN DEFAULT 1, telegram_bot_token TEXT,
    telegram_chat_id TEXT, primary_llm_model TEXT DEFAULT 'hermes',
    fallback_llm_model TEXT DEFAULT 'gpt-4o',
    webhook_verify_token TEXT DEFAULT 'verify_token_change_me',
    max_retries INTEGER DEFAULT 3, rate_limit_warning_threshold INTEGER DEFAULT 80,
    default_language TEXT DEFAULT 'ar', warmup_mode BOOLEAN DEFAULT 1,
    safe_reply_ar TEXT, safe_reply_en TEXT,
    public_reply_message_ar TEXT, public_reply_message_en TEXT,
    reply_mode TEXT DEFAULT 'both',
    auto_reply_start_date TEXT, auto_reply_end_date TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS admin_users (
    id TEXT PRIMARY KEY, email TEXT UNIQUE, name TEXT,
    role TEXT DEFAULT 'reviewer', is_active BOOLEAN DEFAULT 1,
    avatar_url TEXT, permissions TEXT DEFAULT '{}',
    last_active_at TEXT, telegram_user_id TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY, page_id TEXT, admin_id TEXT,
    admin_name TEXT DEFAULT 'system', action TEXT, entity_type TEXT,
    entity_id TEXT, old_values TEXT, new_values TEXT,
    reason TEXT, details TEXT, ip_address TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS automation_rules (
    id TEXT PRIMARY KEY, page_id TEXT, name TEXT, description TEXT,
    conditions TEXT DEFAULT '[]', condition_logic TEXT DEFAULT 'AND',
    action TEXT, action_config TEXT DEFAULT '{}', priority INTEGER DEFAULT 10,
    is_active BOOLEAN DEFAULT 1, trigger_count INTEGER DEFAULT 0,
    last_triggered_at TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
"""

_DROP_TABLES_SQL = """
DROP TABLE IF EXISTS automation_rules;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS admin_users;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS knowledge_base;
DROP TABLE IF EXISTS escalations;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS pages;
"""


@pytest.fixture(scope="session")
def event_loop() -> Generator[asyncio.AbstractEventLoop, None, None]:
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest_asyncio.fixture(scope="session", autouse=True)
async def setup_database():
    """Create SQLite-compatible tables before tests, drop after."""
    async with test_engine.begin() as conn:
        for stmt in _CREATE_TABLES_SQL.strip().split(";"):
            stmt = stmt.strip()
            if stmt:
                await conn.execute(text(stmt))
    yield
    async with test_engine.begin() as conn:
        for stmt in _DROP_TABLES_SQL.strip().split(";"):
            stmt = stmt.strip()
            if stmt:
                await conn.execute(text(stmt))


@pytest_asyncio.fixture
async def db_session() -> AsyncGenerator[AsyncSession, None]:
    """Yield a clean database session with per-test isolation."""
    async with TestSessionLocal() as session:
        async with session.begin():
            yield session
        await session.rollback()


@pytest.fixture
def mock_settings() -> MagicMock:
    s = MagicMock()
    s.database_url = "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true"
    s.async_database_url = "sqlite+aiosqlite:///file::memory:?cache=shared&uri=true"
    s.redis_url = "redis://localhost:6379/0"
    s.meta_app_secret = "test_app_secret"
    s.meta_app_id = "test_app_id"
    s.webhook_rate_limit_rpm = 200
    s.rate_limit_rpm = 60
    s.openai_api_key = ""
    s.glm_api_key = ""
    s.sentry_dsn = ""
    s.environment = "test"
    s.auto_apply_schema_patches = False
    s.ssl_required = False
    s.DB_POOL_SIZE = 5
    s.DB_MAX_OVERFLOW = 10
    s.DB_POOL_RECYCLE = 1800
    s.DB_POOL_TIMEOUT = 30
    return s


@pytest.fixture
def sample_page_id() -> str:
    return str(uuid.uuid4())


@pytest.fixture
def sample_page(sample_page_id: str) -> dict:
    return {
        "id": sample_page_id,
        "platform": "facebook",
        "page_id": "fb_123456789",
        "name": "Test Business Page",
        "avatar_url": "https://example.com/avatar.png",
        "is_active": True,
        "auto_reply_enabled": False,
        "shadow_mode": True,
    }


@pytest.fixture
def sample_conversation(sample_page_id: str) -> dict:
    return {
        "id": str(uuid.uuid4()),
        "page_id": sample_page_id,
        "page_name": "Test Business Page",
        "platform": "facebook",
        "comment_id": "comment_abc123",
        "post_id": "post_xyz789",
        "customer_id": str(uuid.uuid4()),
        "customer_name": "Ahmed Test",
        "original_comment": "Test comment",
        "ai_reply": "AI reply",
        "status": "shadow_pending",
        "intent": "price_inquiry",
        "sentiment": "neutral",
        "urgency": "normal",
        "priority": "normal",
        "confidence_score": 0.88,
        "language": "ar",
        "is_shadow_mode": True,
        "guardrail_triggered": False,
        "guardrail_reason": None,
    }


@pytest.fixture
def sample_customer(sample_page_id: str) -> dict:
    return {
        "id": str(uuid.uuid4()),
        "page_id": sample_page_id,
        "facebook_id": "fb_user_999",
        "full_name": "Fatima Customer",
        "interaction_count": 3,
        "lead_score": 40,
        "purchase_intent": "Medium",
        "conversion_status": "prospect",
        "churn_risk": "low",
        "churn_risk_score": 0.15,
    }


@pytest.fixture
def sample_knowledge_entry() -> dict:
    return {
        "id": str(uuid.uuid4()),
        "page_id": None,
        "category": "pricing",
        "question": "What is the price?",
        "answer": "200 EGP.",
        "intent_tags": ["price_inquiry"],
        "language": "en",
        "is_active": True,
        "usage_count": 5,
        "quality_score": 0.9,
    }


@pytest.fixture
def sample_settings_orm(sample_page_id: str) -> dict:
    return {
        "id": str(uuid.uuid4()),
        "page_id": None,
        "confidence_threshold": 0.85,
        "auto_escalate_angry": True,
        "primary_llm_model": "hermes",
        "fallback_llm_model": "gpt-4o",
        "webhook_verify_token": "test_verify_token",
        "max_retries": 3,
        "rate_limit_warning_threshold": 80,
        "default_language": "ar",
        "warmup_mode": True,
        "safe_reply_ar": "شكراً لتواصلك معنا.",
        "safe_reply_en": "Thank you for reaching out.",
        "public_reply_message_ar": "تم التواصل معك على الخاص",
        "public_reply_message_en": "We've contacted you privately",
        "reply_mode": "both",
    }


@pytest.fixture
def mock_llm():
    llm = AsyncMock()
    llm.ainvoke = AsyncMock(return_value=MagicMock(content="general"))
    return llm


@pytest.fixture
def mock_cache():
    _store: dict[str, Any] = {}
    async def fake_get_json(key: str):
        return _store.get(key)
    async def fake_set_json(key: str, value: Any, ttl_secs: int = 300):
        _store[key] = value
    async def fake_delete(key: str):
        _store.pop(key, None)
    async def fake_delete_pattern(pattern: str):
        import fnmatch
        to_remove = [k for k in _store if fnmatch.fnmatch(k, pattern)]
        for k in to_remove:
            del _store[k]
    with (
        patch("app.services.cache.get_json", side_effect=fake_get_json),
        patch("app.services.cache.set_json", side_effect=fake_set_json),
        patch("app.services.cache.delete", side_effect=fake_delete),
        patch("app.services.cache.delete_pattern", side_effect=fake_delete_pattern),
    ):
        yield _store


@pytest_asyncio.fixture
async def client(db_session: AsyncSession) -> AsyncGenerator[AsyncClient, None]:
    from app.deps import get_db as _orig
    async def _override():
        yield db_session
    from app.main import app
    app.dependency_overrides[_orig] = _override
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://testserver") as ac:
        yield ac
    app.dependency_overrides.clear()
