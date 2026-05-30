import uuid
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from datetime import datetime, timezone
from typing import Optional
from app.deps import get_db
from app.db import Settings
from app.schemas import SettingsOut, SettingsUpdate
from app.services.cache import get_json, set_json
from app.services.runtime_settings import (
    SETTINGS_CACHE_TTL,
    invalidate_settings_cache,
    serialize_settings,
    settings_cache_key,
)

router = APIRouter(tags=["settings"])


async def _get_or_create_settings(db: AsyncSession, page_id: Optional[str] = None) -> Settings:
    if page_id:
        result = await db.execute(select(Settings).where(Settings.page_id == page_id).limit(1))
    else:
        result = await db.execute(select(Settings).where(Settings.page_id.is_(None)).limit(1))
    settings = result.scalar_one_or_none()
    if not settings:
        # If no global settings exist at all, seed from first existing row
        if page_id:
            global_result = await db.execute(select(Settings).where(Settings.page_id.is_(None)).limit(1))
            global_settings = global_result.scalar_one_or_none()
            if global_settings:
                settings = Settings(
                    id=str(uuid.uuid4()),
                    page_id=page_id,
                    confidence_threshold=global_settings.confidence_threshold,
                    auto_escalate_angry=global_settings.auto_escalate_angry,
                    telegram_bot_token=global_settings.telegram_bot_token,
                    telegram_chat_id=global_settings.telegram_chat_id,
                    primary_llm_model=global_settings.primary_llm_model,
                    fallback_llm_model=global_settings.fallback_llm_model,
                    webhook_verify_token=global_settings.webhook_verify_token,
                    max_retries=global_settings.max_retries,
                    rate_limit_warning_threshold=global_settings.rate_limit_warning_threshold,
                    default_language=global_settings.default_language,
                    warmup_mode=global_settings.warmup_mode,
                    safe_reply_ar=global_settings.safe_reply_ar,
                    safe_reply_en=global_settings.safe_reply_en,
                    public_reply_message_ar=global_settings.public_reply_message_ar,
                    public_reply_message_en=global_settings.public_reply_message_en,
                    reply_mode=global_settings.reply_mode,
                )
            else:
                settings = Settings(id=str(uuid.uuid4()), page_id=page_id)
        else:
            settings = Settings(id=str(uuid.uuid4()), page_id=None)
        db.add(settings)
        await db.commit()
        await db.refresh(settings)
    return settings


@router.get("/settings", response_model=SettingsOut)
async def get_settings(
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    cache_key = settings_cache_key(page_id)
    cached = await get_json(cache_key)
    if cached:
        return cached

    settings = await _get_or_create_settings(db, page_id)
    data = serialize_settings(settings)
    await set_json(cache_key, data, ttl_secs=SETTINGS_CACHE_TTL)
    return data


@router.patch("/settings", response_model=SettingsOut)
async def update_settings(
    body: SettingsUpdate,
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    settings = await _get_or_create_settings(db, page_id)
    for field, value in body.model_dump(exclude_none=True).items():
        setattr(settings, field, value)
    settings.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(settings)
    await invalidate_settings_cache(page_id)
    return settings


@router.get("/settings/agency-profile")
async def get_agency_profile(db: AsyncSession = Depends(get_db)):
    from app.db import AgencyProfile
    result = await db.execute(select(AgencyProfile).limit(1))
    profile = result.scalar_one_or_none()
    if not profile:
        profile = AgencyProfile(id=str(uuid.uuid4()))
        db.add(profile)
        await db.commit()
        await db.refresh(profile)
    return profile


@router.patch("/settings/agency-profile")
async def update_agency_profile(body: dict, db: AsyncSession = Depends(get_db)):
    from app.db import AgencyProfile
    result = await db.execute(select(AgencyProfile).limit(1))
    profile = result.scalar_one_or_none()
    if not profile:
        profile = AgencyProfile(id=str(uuid.uuid4()))
        db.add(profile)
    
    for field, value in body.items():
        if hasattr(profile, field):
            setattr(profile, field, value)
    
    profile.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(profile)
    return profile
