from pydantic import BaseModel, ConfigDict, Field
from typing import Optional, List, Any, Literal
from datetime import datetime


# ─── Status / Enum Types ─────────────────────────────────────────────────────

ConversationStatus = Literal[
    "pending", "processing", "replied", "escalated", "resolved", "flagged",
    "shadow_pending",
]
EscalationPriority = Literal["low", "medium", "high", "critical"]
CustomerChurnRisk = Literal["low", "medium", "high"]


class PageOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    platform: str
    page_id: str
    name: str
    avatar_url: Optional[str] = None
    is_active: bool
    auto_reply_enabled: bool
    shadow_mode: bool
    tracking_start_date: Optional[datetime] = None
    auto_reply_end_date: Optional[datetime] = None
    token_status: Optional[str] = None
    token_expires_at: Optional[datetime] = None
    token_last_refreshed_at: Optional[datetime] = None
    token_last_error: Optional[str] = None
    created_at: datetime
    updated_at: datetime


class PageCreate(BaseModel):
    platform: str = Field(max_length=20)
    page_id: str = Field(max_length=100)
    name: str = Field(max_length=200)
    avatar_url: Optional[str] = Field(default=None, max_length=500)
    is_active: bool = True
    auto_reply_enabled: bool = False
    shadow_mode: bool = True
    access_token_encrypted: Optional[str] = Field(default=None, max_length=2000)
    auto_reply_end_date: Optional[datetime] = None


class PageUpdate(BaseModel):
    platform: Optional[str] = Field(default=None, max_length=20)
    name: Optional[str] = Field(default=None, max_length=200)
    avatar_url: Optional[str] = Field(default=None, max_length=500)
    is_active: Optional[bool] = None
    auto_reply_enabled: Optional[bool] = None
    shadow_mode: Optional[bool] = None
    access_token_encrypted: Optional[str] = Field(default=None, max_length=2000)
    auto_reply_end_date: Optional[datetime] = None


class ConversationOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    page_id: str
    page_name: str
    platform: str
    comment_id: str
    post_id: str
    customer_id: Optional[str] = None
    customer_name: str
    customer_avatar_url: Optional[str] = None
    original_comment: str
    ai_reply: Optional[str] = None
    admin_reply: Optional[str] = None
    status: ConversationStatus
    intent: Optional[str] = None
    sentiment: Optional[str] = None
    confidence_score: Optional[float] = None
    language: Optional[str] = None
    is_shadow_mode: bool
    sentiment_history: List[str] = []
    escalation_reason: Optional[str] = None
    processing_time: Optional[float] = None
    pii_detected: Optional[bool] = False
    matched_rule_id: Optional[str] = None
    replied_at: Optional[datetime] = None
    created_at: datetime
    updated_at: datetime



class PaginationParams(BaseModel):
    """Pagination parameters with bounded limits."""
    page: int = Field(1, ge=1, description="Page number (min 1)")
    limit: int = Field(50, ge=1, le=500, description="Items per page (1-500)")


class ConversationListResponse(BaseModel):
    data: List[ConversationOut]
    total: int
    page: int = Field(ge=1, le=500)
    limit: int = Field(ge=1, le=500)


class ManualReplyRequest(BaseModel):
    reply: str = Field(..., max_length=8000, description="Reply text (max 8000 chars)")


class CustomerOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    page_id: Optional[str] = None
    page_name: Optional[str] = None
    platform: Optional[str] = None
    facebook_id: Optional[str] = None
    instagram_id: Optional[str] = None
    whatsapp_id: Optional[str] = None
    username: Optional[str] = None
    full_name: Optional[str] = None
    profile_url: Optional[str] = None
    avatar_url: Optional[str] = None
    first_contact_date: Optional[datetime] = None
    last_interaction: Optional[datetime] = None
    interaction_count: int
    lead_score: int
    purchase_intent: Optional[str] = "Low"
    conversion_status: Optional[str] = "prospect"
    assigned_admin: Optional[str] = None
    tags: List[str] = []
    notes: List[Any] = []
    escalation_history: List[str] = []
    churn_risk: Optional[str] = "low"
    churn_risk_score: Optional[float] = 0.0
    next_best_action: Optional[str] = None
    re_engage_sent_at: Optional[datetime] = None
    gdpr_deleted: Optional[bool] = False
    created_at: datetime
    updated_at: datetime


class CustomerListResponse(BaseModel):
    data: List[CustomerOut]
    total: int
    page: int = Field(ge=1, le=500)
    limit: int = Field(ge=1, le=500)


class CustomerUpdate(BaseModel):
    full_name: Optional[str] = Field(default=None, max_length=200)
    username: Optional[str] = Field(default=None, max_length=100)
    purchase_intent: Optional[str] = None
    conversion_status: Optional[str] = None
    assigned_admin: Optional[str] = Field(default=None, max_length=200)
    lead_score: Optional[int] = Field(default=None, ge=0, le=100)
    tags: Optional[List[str]] = None
    churn_risk: Optional[str] = None


class BulkUpdateCustomersRequest(BaseModel):
    ids: List[str]
    update: CustomerUpdate


class BulkDeleteCustomersRequest(BaseModel):
    ids: List[str]


class AddNoteRequest(BaseModel):
    content: str = Field(max_length=5000)
    author: str = Field(default="admin", max_length=200)


class EscalationOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    conversation_id: str
    page_id: str
    page_name: str
    customer_id: Optional[str] = None
    customer_name: str
    original_comment: str
    reason: str
    priority: Literal["low", "medium", "high", "critical"]
    status: Literal["open", "assigned", "resolved", "closed"]
    admin_notes: Optional[str] = None
    resolved_by: Optional[str] = None
    resolved_at: Optional[datetime] = None
    created_at: datetime
    updated_at: datetime


class EscalationListResponse(BaseModel):
    data: List[EscalationOut]
    total: int
    page: int = Field(ge=1, le=500)
    limit: int = Field(ge=1, le=500)


class ResolveEscalationRequest(BaseModel):
    admin_notes: Optional[str] = None
    resolved_by: str = "admin"


class KnowledgeBaseOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    page_id: Optional[str] = None
    category: str
    question: str
    answer: str
    intent_tags: List[str] = []
    language: str
    is_active: bool
    usage_count: int
    quality_score: Optional[float] = None
    created_at: datetime
    updated_at: datetime


class KnowledgeBaseCreate(BaseModel):
    page_id: Optional[str] = Field(default=None, max_length=100)
    category: str = Field(max_length=100)
    question: str = Field(max_length=2000)
    answer: str = Field(max_length=10000)
    intent_tags: List[str] = []
    language: str = Field(default="ar", max_length=10)
    is_active: bool = True


class KnowledgeBaseUpdate(BaseModel):
    category: Optional[str] = Field(default=None, max_length=100)
    question: Optional[str] = Field(default=None, max_length=2000)
    answer: Optional[str] = Field(default=None, max_length=10000)
    intent_tags: Optional[List[str]] = None
    language: Optional[str] = Field(default=None, max_length=10)
    is_active: Optional[bool] = None


class SettingsOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    page_id: Optional[str] = None
    confidence_threshold: float
    auto_escalate_angry: bool
    telegram_bot_token: Optional[str] = None
    telegram_chat_id: Optional[str] = None
    primary_llm_model: str
    fallback_llm_model: str
    webhook_verify_token: str
    max_retries: int
    rate_limit_warning_threshold: int
    default_language: str
    warmup_mode: bool
    safe_reply_ar: str
    safe_reply_en: str
    public_reply_message_ar: str
    public_reply_message_en: str
    reply_mode: str
    auto_reply_start_date: Optional[datetime] = None
    auto_reply_end_date: Optional[datetime] = None
    whatsapp_notification_phone: Optional[str] = None
    whatsapp_notification_api_key: Optional[str] = None
    enable_private_replies: bool
    
    # Brand Kit fields
    brand_description: Optional[str] = None
    brand_industry: Optional[str] = None
    brand_target_audience: Optional[str] = None
    brand_tone_of_voice: Optional[str] = None
    brand_preferred_hashtags: Optional[str] = None
    brand_restricted_words: Optional[str] = None
    brand_sample_posts: Optional[str] = None

    created_at: datetime
    updated_at: datetime


class SettingsUpdate(BaseModel):
    confidence_threshold: Optional[float] = Field(default=None, ge=0.0, le=1.0)
    auto_escalate_angry: Optional[bool] = None
    telegram_bot_token: Optional[str] = Field(default=None, max_length=200)
    telegram_chat_id: Optional[str] = Field(default=None, max_length=100)
    primary_llm_model: Optional[str] = Field(default=None, max_length=100)
    fallback_llm_model: Optional[str] = Field(default=None, max_length=100)
    webhook_verify_token: Optional[str] = Field(default=None, max_length=200)
    max_retries: Optional[int] = Field(default=None, ge=1, le=10)
    rate_limit_warning_threshold: Optional[int] = Field(default=None, ge=1, le=100)
    default_language: Optional[str] = Field(default=None, max_length=10)
    warmup_mode: Optional[bool] = None
    safe_reply_ar: Optional[str] = Field(default=None, max_length=5000)
    safe_reply_en: Optional[str] = Field(default=None, max_length=5000)
    public_reply_message_ar: Optional[str] = Field(default=None, max_length=2000)
    public_reply_message_en: Optional[str] = Field(default=None, max_length=2000)
    reply_mode: Optional[Literal["public_only", "dm_only", "both"]] = None
    auto_reply_start_date: Optional[datetime] = None
    auto_reply_end_date: Optional[datetime] = None
    whatsapp_notification_phone: Optional[str] = Field(default=None, max_length=50)
    whatsapp_notification_api_key: Optional[str] = Field(default=None, max_length=100)
    enable_private_replies: Optional[bool] = None
    
    # Brand Kit fields
    brand_description: Optional[str] = Field(default=None, max_length=10000)
    brand_industry: Optional[str] = Field(default=None, max_length=200)
    brand_target_audience: Optional[str] = Field(default=None, max_length=5000)
    brand_tone_of_voice: Optional[str] = Field(default=None, max_length=100)
    brand_preferred_hashtags: Optional[str] = Field(default=None, max_length=2000)
    brand_restricted_words: Optional[str] = Field(default=None, max_length=2000)
    brand_sample_posts: Optional[str] = Field(default=None, max_length=10000)


class DashboardStats(BaseModel):
    total_conversations: int
    pending_conversations: int
    open_escalations: int
    total_customers: int
    high_intent_leads: int
    avg_confidence_score: float
    auto_reply_rate: float
    avg_response_time_seconds: float
    shadow_mode_reviews: int
    token_healthy: int
    token_expiring_soon: int
    token_expired: int


class ConversationAnalyticsPoint(BaseModel):
    date: str
    total: int
    replied: int
    escalated: int


class IntentItem(BaseModel):
    intent: str
    count: int
    percentage: float


class SentimentItem(BaseModel):
    sentiment: str
    count: int
    percentage: float


class TokenStatus(BaseModel):
    id: str
    name: str
    platform: str
    token_status: Optional[str] = None
    token_expires_at: Optional[datetime] = None
    token_last_refreshed_at: Optional[datetime] = None
    token_last_error: Optional[str] = None


# ─── Campaign Schemas ────────────────────────────────────────────────────────

class CampaignOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    name: str
    description: Optional[str] = None
    status: str
    target_filter: dict
    customer_ids: List[str]
    message_ar: str
    message_en: str
    media_urls: List[str]
    media_type: Optional[str] = None
    send_at: Optional[datetime] = None
    interval_hours: Optional[int] = None
    max_sends: Optional[int] = None
    total_recipients: int
    sent_count: int
    failed_count: int
    created_by: Optional[str] = None
    created_at: datetime
    updated_at: datetime


class CampaignCreate(BaseModel):
    name: str = Field(max_length=200)
    description: Optional[str] = Field(default=None, max_length=5000)
    target_filter: dict = {}
    customer_ids: List[str] = []
    message_ar: str = Field(default="", max_length=5000)
    message_en: str = Field(default="", max_length=5000)
    media_urls: List[str] = []
    media_type: Optional[str] = None
    send_at: Optional[datetime] = None
    interval_hours: Optional[int] = Field(default=None, ge=1)
    max_sends: Optional[int] = Field(default=None, ge=1)


class CampaignUpdate(BaseModel):
    name: Optional[str] = Field(default=None, max_length=200)
    description: Optional[str] = Field(default=None, max_length=5000)
    status: Optional[Literal["draft", "scheduled", "active", "paused", "completed"]] = None
    target_filter: Optional[dict] = None
    customer_ids: Optional[List[str]] = None
    message_ar: Optional[str] = Field(default=None, max_length=5000)
    message_en: Optional[str] = Field(default=None, max_length=5000)
    media_urls: Optional[List[str]] = None
    media_type: Optional[str] = None
    send_at: Optional[datetime] = None
    interval_hours: Optional[int] = Field(default=None, ge=1)
    max_sends: Optional[int] = Field(default=None, ge=1)


class CampaignListResponse(BaseModel):
    data: List[CampaignOut]
    total: int
    page: int = Field(ge=1, le=500)
    limit: int = Field(ge=1, le=500)

class CustomAIModelOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: str
    name: str
    provider: str
    model_name: str
    api_key_masked: str
    api_base: Optional[str] = None
    is_active: bool
    created_at: datetime
    updated_at: datetime

class CustomAIModelCreate(BaseModel):
    name: str = Field(max_length=200)
    provider: str = Field(max_length=50)
    model_name: str = Field(max_length=100)
    api_key: str = Field(max_length=2000)
    api_base: Optional[str] = Field(default=None, max_length=1000)
    is_active: bool = False

class CustomAIModelUpdate(BaseModel):
    name: Optional[str] = Field(default=None, max_length=200)
    provider: Optional[str] = Field(default=None, max_length=50)
    model_name: Optional[str] = Field(default=None, max_length=100)
    api_key: Optional[str] = Field(default=None, max_length=2000)
    api_base: Optional[str] = Field(default=None, max_length=1000)
    is_active: Optional[bool] = None
