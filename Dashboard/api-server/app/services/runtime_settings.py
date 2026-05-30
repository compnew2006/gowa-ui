from __future__ import annotations

from datetime import datetime
from types import SimpleNamespace
from typing import Any

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Settings
from app.services.cache import delete, delete_pattern, get_json, set_json

SETTINGS_CACHE_TTL = 300
DATETIME_FIELDS = {"auto_reply_start_date", "auto_reply_end_date", "created_at", "updated_at"}
SETTINGS_FIELDS = (
    "id",
    "page_id",
    "confidence_threshold",
    "auto_escalate_angry",
    "telegram_bot_token",
    "telegram_chat_id",
    "primary_llm_model",
    "fallback_llm_model",
    "webhook_verify_token",
    "max_retries",
    "rate_limit_warning_threshold",
    "default_language",
    "warmup_mode",
    "safe_reply_ar",
    "safe_reply_en",
    "public_reply_message_ar",
    "public_reply_message_en",
    "reply_mode",
    "auto_reply_start_date",
    "auto_reply_end_date",
    "whatsapp_notification_phone",
    "whatsapp_notification_api_key",
    "enable_private_replies",
    "brand_description",
    "brand_industry",
    "brand_target_audience",
    "brand_tone_of_voice",
    "brand_preferred_hashtags",
    "brand_restricted_words",
    "brand_sample_posts",
    "created_at",
    "updated_at",
)


def settings_cache_key(page_id: str | None = None) -> str:
    return f"settings:{page_id or 'global'}"


SENSITIVE_FIELDS = {"telegram_bot_token", "webhook_verify_token", "whatsapp_notification_api_key"}


def _mask_sensitive(value: Any, field: str) -> Any:
    """Mask sensitive fields, showing only last 4 characters."""
    if field in SENSITIVE_FIELDS and value:
        val = str(value)
        return "****" + val[-4:] if len(val) > 4 else "****"
    return value


def serialize_settings(settings: Settings) -> dict[str, Any]:
    data = {}
    for field in SETTINGS_FIELDS:
        value = getattr(settings, field, None)
        if field in DATETIME_FIELDS and value is not None:
            value = value.isoformat()
        value = _mask_sensitive(value, field)
        data[field] = value
    return data


def _deserialize_settings(data: dict[str, Any]) -> SimpleNamespace:
    parsed = dict(data)
    for field in DATETIME_FIELDS:
        value = parsed.get(field)
        if isinstance(value, str):
            parsed[field] = datetime.fromisoformat(value)
    return SimpleNamespace(**parsed)


async def get_runtime_settings(db: AsyncSession, page_id: str | None = None) -> Settings | SimpleNamespace | None:
    cached = await get_json(settings_cache_key(page_id))
    if cached:
        return _deserialize_settings(cached)

    if page_id:
        result = await db.execute(select(Settings).where(Settings.page_id == page_id).limit(1))
        settings = result.scalar_one_or_none()
        if not settings:
            result = await db.execute(select(Settings).where(Settings.page_id.is_(None)).limit(1))
            settings = result.scalar_one_or_none()
    else:
        result = await db.execute(select(Settings).where(Settings.page_id.is_(None)).limit(1))
        settings = result.scalar_one_or_none()

    if settings:
        await set_json(settings_cache_key(page_id), serialize_settings(settings), ttl_secs=SETTINGS_CACHE_TTL)
    return settings


async def invalidate_settings_cache(page_id: str | None = None) -> None:
    if page_id:
        await delete(settings_cache_key(page_id))
        return
    await delete_pattern("settings:*")
