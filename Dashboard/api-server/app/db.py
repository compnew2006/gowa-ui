from sqlalchemy.ext.asyncio import create_async_engine, async_sessionmaker, AsyncSession
from sqlalchemy.orm import DeclarativeBase, mapped_column, Mapped
from sqlalchemy import String, Boolean, Float, Integer, Text, DateTime, func, ARRAY, JSON
from sqlalchemy.dialects.postgresql import UUID
from typing import Optional
from datetime import datetime
from app.config import get_settings

settings = get_settings()

engine_kwargs: dict = {
    "echo": False,
    "pool_pre_ping": True,
    "pool_size": settings.db_pool_size,
    "max_overflow": settings.db_max_overflow,
    "pool_recycle": settings.db_pool_recycle,
    "pool_timeout": settings.db_pool_timeout,
}
if settings.ssl_required:
    engine_kwargs["connect_args"] = {"ssl": True}

engine = create_async_engine(settings.async_database_url, **engine_kwargs)
AsyncSessionLocal = async_sessionmaker(engine, expire_on_commit=False, class_=AsyncSession)


class Base(DeclarativeBase):
    pass


class Page(Base):
    __tablename__ = "pages"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    platform: Mapped[str] = mapped_column(String)
    page_id: Mapped[str] = mapped_column(String)
    name: Mapped[str] = mapped_column(String)
    avatar_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    auto_reply_enabled: Mapped[bool] = mapped_column(Boolean, default=False)
    shadow_mode: Mapped[bool] = mapped_column(Boolean, default=True)
    tracking_start_date: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    auto_reply_end_date: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    access_token_encrypted: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    token_status: Mapped[Optional[str]] = mapped_column(String, default="valid")
    token_expires_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    token_last_refreshed_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    token_last_error: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class Conversation(Base):
    __tablename__ = "conversations"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[str] = mapped_column(UUID(as_uuid=False))
    page_name: Mapped[str] = mapped_column(String)
    platform: Mapped[str] = mapped_column(String)
    comment_id: Mapped[str] = mapped_column(String)
    post_id: Mapped[str] = mapped_column(String)
    customer_id: Mapped[Optional[str]] = mapped_column(UUID(as_uuid=False), nullable=True)
    customer_name: Mapped[str] = mapped_column(String)
    customer_avatar_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    original_comment: Mapped[str] = mapped_column(Text)
    ai_reply: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    admin_reply: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    status: Mapped[str] = mapped_column(String, default="pending")
    intent: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    sentiment: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    urgency: Mapped[str] = mapped_column(String, default="normal")
    priority: Mapped[str] = mapped_column(String, default="normal")
    confidence_score: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    language: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    is_shadow_mode: Mapped[bool] = mapped_column(Boolean, default=False)
    sentiment_history: Mapped[list] = mapped_column(ARRAY(String), default=list)
    escalation_reason: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    guardrail_triggered: Mapped[bool] = mapped_column(Boolean, default=False)
    guardrail_reason: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    processing_time: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    replied_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    shadow_approved_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    # PII compliance
    pii_detected: Mapped[bool] = mapped_column(Boolean, default=False)
    pii_masked_reply: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    # Rule engine match
    matched_rule_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class Customer(Base):
    __tablename__ = "customers"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    facebook_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    instagram_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    whatsapp_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    username: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    full_name: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    profile_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    avatar_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    first_contact_date: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    last_interaction: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    interaction_count: Mapped[int] = mapped_column(Integer, default=0)
    lead_score: Mapped[int] = mapped_column(Integer, default=0)
    purchase_intent: Mapped[str] = mapped_column(String, default="Low")
    conversion_status: Mapped[str] = mapped_column(String, default="prospect")
    assigned_admin: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    tags: Mapped[list] = mapped_column(ARRAY(String), default=list)
    notes: Mapped[list] = mapped_column(JSON, default=list)
    escalation_history: Mapped[list] = mapped_column(ARRAY(String), default=list)
    # Predictive CRM
    churn_risk: Mapped[str] = mapped_column(String, default="low")       # low|medium|high
    churn_risk_score: Mapped[float] = mapped_column(Float, default=0.0)  # 0-1
    next_best_action: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    re_engage_sent_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    # GDPR
    gdpr_deleted: Mapped[bool] = mapped_column(Boolean, default=False)
    gdpr_export_requested_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class WhatsAppBridge(Base):
    __tablename__ = "whatsapp_bridges"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[str] = mapped_column(UUID(as_uuid=False), unique=True)
    port: Mapped[int] = mapped_column(Integer)
    session_name: Mapped[str] = mapped_column(String)
    status: Mapped[str] = mapped_column(String, default="disconnected")
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    last_poll_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class Escalation(Base):
    __tablename__ = "escalations"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    conversation_id: Mapped[str] = mapped_column(UUID(as_uuid=False))
    page_id: Mapped[str] = mapped_column(UUID(as_uuid=False))
    page_name: Mapped[str] = mapped_column(String)
    customer_id: Mapped[Optional[str]] = mapped_column(UUID(as_uuid=False), nullable=True)
    customer_name: Mapped[str] = mapped_column(String)
    original_comment: Mapped[str] = mapped_column(Text)
    reason: Mapped[str] = mapped_column(Text)
    priority: Mapped[str] = mapped_column(String)
    status: Mapped[str] = mapped_column(String, default="open")
    assigned_to: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    admin_notes: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    resolved_by: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    resolved_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class KnowledgeBase(Base):
    __tablename__ = "knowledge_base"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    category: Mapped[str] = mapped_column(String)
    question: Mapped[str] = mapped_column(Text)
    answer: Mapped[str] = mapped_column(Text)
    intent_tags: Mapped[list] = mapped_column(ARRAY(String), default=list)
    language: Mapped[str] = mapped_column(String, default="ar")
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    usage_count: Mapped[int] = mapped_column(Integer, default=0)
    quality_score: Mapped[Optional[float]] = mapped_column(Float, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class Settings(Base):
    __tablename__ = "settings"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    confidence_threshold: Mapped[float] = mapped_column(Float, default=0.85)
    auto_escalate_angry: Mapped[bool] = mapped_column(Boolean, default=True)
    telegram_bot_token: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    telegram_chat_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    primary_llm_model: Mapped[str] = mapped_column(String, default="gpt-4o")
    fallback_llm_model: Mapped[str] = mapped_column(String, default="gpt-4o")
    webhook_verify_token: Mapped[str] = mapped_column(String, default="verify_token_change_me")
    max_retries: Mapped[int] = mapped_column(Integer, default=3)
    rate_limit_warning_threshold: Mapped[int] = mapped_column(Integer, default=80)
    default_language: Mapped[str] = mapped_column(String, default="ar", server_default="ar")
    warmup_mode: Mapped[bool] = mapped_column(Boolean, default=True, server_default="true")
    
    # Expanded Agent Automation Settings
    whatsapp_notification_phone: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    whatsapp_notification_api_key: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    enable_private_replies: Mapped[bool] = mapped_column(Boolean, default=True, server_default="true")
    safe_reply_ar: Mapped[str] = mapped_column(Text, default="شكراً لتواصلك معنا. سيتواصل معك أحد ممثلي خدمة العملاء في أقرب وقت ممكن.")
    safe_reply_en: Mapped[str] = mapped_column(Text, default="Thank you for reaching out. A customer service representative will contact you shortly.")
    # Public Reply Setting (DM uses AI reply)
    public_reply_message_ar: Mapped[str] = mapped_column(Text, default="تم التواصل معك على الخاص")
    public_reply_message_en: Mapped[str] = mapped_column(Text, default="We've contacted you privately")
    reply_mode: Mapped[str] = mapped_column(String, default="both")  # "public_only" | "dm_only" | "both"
    # Auto-reply date range (contract period)
    auto_reply_start_date: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    auto_reply_end_date: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    
    # Brand Kit fields
    brand_description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    brand_industry: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    brand_target_audience: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    brand_tone_of_voice: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    brand_preferred_hashtags: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    brand_restricted_words: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    brand_sample_posts: Mapped[Optional[str]] = mapped_column(Text, nullable=True)

    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


# ─── New Models (Features 1-10) ────────────────────────────────────────────


class AdminUser(Base):
    """Team Collaboration: Admin users with role-based access."""
    __tablename__ = "admin_users"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    email: Mapped[str] = mapped_column(String, unique=True)
    name: Mapped[str] = mapped_column(String)
    role: Mapped[str] = mapped_column(String, default="reviewer")  # admin|reviewer|analyst
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    avatar_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    permissions: Mapped[dict] = mapped_column(JSON, default=dict)
    last_active_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    telegram_user_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class AuditLog(Base):
    """Compliance & Team: Full audit trail of all admin actions."""
    __tablename__ = "audit_logs"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    admin_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    admin_name: Mapped[str] = mapped_column(String, default="system")
    action: Mapped[str] = mapped_column(String, index=True)       # approve_reply|reject_reply|resolve|assign|delete|export_data|approve|reject|correct|undo
    entity_type: Mapped[str] = mapped_column(String, index=True)  # conversation|customer|escalation|rule|page
    entity_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    old_values: Mapped[Optional[dict]] = mapped_column(JSON, nullable=True)
    new_values: Mapped[Optional[dict]] = mapped_column(JSON, nullable=True)
    reason: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    details: Mapped[Optional[dict]] = mapped_column(JSON, nullable=True)
    ip_address: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), index=True)


class AutomationRule(Base):
    """Rules Engine: Custom workflow automation rules."""
    __tablename__ = "automation_rules"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    name: Mapped[str] = mapped_column(String)
    description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    # conditions: [{"field": "intent", "op": "eq", "value": "refund"}, ...]
    conditions: Mapped[list] = mapped_column(JSON, default=list)
    condition_logic: Mapped[str] = mapped_column(String, default="AND")  # AND|OR
    # action: escalate|tag|assign|notify|skip|custom_reply
    action: Mapped[str] = mapped_column(String)
    action_config: Mapped[dict] = mapped_column(JSON, default=dict)
    priority: Mapped[int] = mapped_column(Integer, default=10)  # lower = higher priority
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    trigger_count: Mapped[int] = mapped_column(Integer, default=0)
    last_triggered_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class IntegrationConfig(Base):
    """Integration Hub: Connected external services."""
    __tablename__ = "integration_configs"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    type: Mapped[str] = mapped_column(String)   # slack|zapier|whatsapp|teams
    name: Mapped[str] = mapped_column(String)
    config: Mapped[dict] = mapped_column(JSON, default=dict)  # webhook_url, channel, etc.
    is_active: Mapped[bool] = mapped_column(Boolean, default=False)
    trigger_events: Mapped[list] = mapped_column(ARRAY(String), default=list)  # escalation|reply|error
    last_triggered_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    trigger_count: Mapped[int] = mapped_column(Integer, default=0)
    last_error: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class Campaign(Base):
    """Campaigns: Bulk DM campaigns to targeted customer segments."""
    __tablename__ = "campaigns"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[Optional[str]] = mapped_column(String, nullable=True, index=True)
    name: Mapped[str] = mapped_column(String)
    description: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    status: Mapped[str] = mapped_column(String, default="draft")  # draft|scheduled|active|paused|completed

    # Targeting
    target_filter: Mapped[dict] = mapped_column(JSON, default=dict)  # {"purchase_intent": "High", "churn_risk": "high"}
    customer_ids: Mapped[list] = mapped_column(JSON, default=list)   # explicit list of IDs

    # Content
    message_ar: Mapped[str] = mapped_column(Text, default="")
    message_en: Mapped[str] = mapped_column(Text, default="")
    media_urls: Mapped[list] = mapped_column(JSON, default=list)     # list of uploaded media URLs
    media_type: Mapped[Optional[str]] = mapped_column(String, nullable=True)  # image|video|none

    # Scheduling
    send_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)
    interval_hours: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)  # repeat interval
    max_sends: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Stats
    total_recipients: Mapped[int] = mapped_column(Integer, default=0)
    sent_count: Mapped[int] = mapped_column(Integer, default=0)
    failed_count: Mapped[int] = mapped_column(Integer, default=0)

    created_by: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class AgencyProfile(Base):
    """Agency Profile: Branding and global settings for the agency dashboard."""
    __tablename__ = "agency_profiles"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    agency_name: Mapped[str] = mapped_column(String, default="Social Media AI Agency")
    logo_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    primary_color: Mapped[str] = mapped_column(String, default="#3b82f6") # Blue 500
    support_email: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    dashboard_title: Mapped[str] = mapped_column(String, default="لوحة تحكم الوكالة")
    
    # White-labeling
    is_whitelabeled: Mapped[bool] = mapped_column(Boolean, default=False)
    custom_domain: Mapped[Optional[str]] = mapped_column(String, nullable=True)

    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
class ScheduledPost(Base):
    """ScheduledPost: Feed posts for Facebook/Instagram/WhatsApp."""
    __tablename__ = "scheduled_posts"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    page_id: Mapped[str] = mapped_column(String, index=True)
    platform: Mapped[str] = mapped_column(String)  # facebook|instagram|whatsapp
    message: Mapped[str] = mapped_column(Text)
    media_url: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    
    status: Mapped[str] = mapped_column(String, default="pending")  # pending|posted|failed
    post_id: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    error: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    
    scheduled_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    posted_at: Mapped[Optional[datetime]] = mapped_column(DateTime(timezone=True), nullable=True)

    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

class CustomAIModel(Base):
    """CustomAIModel: Dynamic LLM models configured from the frontend with encrypted keys."""
    __tablename__ = "custom_ai_models"

    id: Mapped[str] = mapped_column(UUID(as_uuid=False), primary_key=True)
    name: Mapped[str] = mapped_column(String)
    provider: Mapped[str] = mapped_column(String)  # openai|zhipuai|anthropic|litellm|custom
    model_name: Mapped[str] = mapped_column(String)  # e.g. gpt-4o-mini
    api_key_encrypted: Mapped[str] = mapped_column(String)
    api_base: Mapped[Optional[str]] = mapped_column(String, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=False)
    
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
