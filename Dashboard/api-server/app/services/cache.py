from __future__ import annotations

import json
import logging
from typing import Any

from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


def _json_default(value: Any):
    if hasattr(value, "isoformat"):
        return value.isoformat()
    raise TypeError(f"Object of type {type(value).__name__} is not JSON serializable")


async def _redis():
    try:
        import redis.asyncio as redis

        client = redis.from_url(settings.redis_url, socket_connect_timeout=1, socket_timeout=1)
        await client.ping()
        return client
    except Exception:
        return None


async def get_json(key: str) -> Any | None:
    client = await _redis()
    if not client:
        return None
    try:
        raw = await client.get(key)
        if raw is None:
            return None
        if isinstance(raw, bytes):
            raw = raw.decode("utf-8")
        return json.loads(raw)
    except Exception as exc:
        logger.debug("[Cache] get failed key=%s error=%s", key, exc)
        return None
    finally:
        await client.aclose()


async def set_json(key: str, value: Any, ttl_secs: int = 300) -> None:
    client = await _redis()
    if not client:
        return
    try:
        await client.set(key, json.dumps(value, default=_json_default), ex=ttl_secs)
    except Exception as exc:
        logger.debug("[Cache] set failed key=%s error=%s", key, exc)
    finally:
        await client.aclose()


async def delete(key: str) -> None:
    client = await _redis()
    if not client:
        return
    try:
        await client.delete(key)
    except Exception as exc:
        logger.debug("[Cache] delete failed key=%s error=%s", key, exc)
    finally:
        await client.aclose()


async def delete_pattern(pattern: str) -> None:
    client = await _redis()
    if not client:
        return
    try:
        async for key in client.scan_iter(match=pattern, count=100):
            await client.delete(key)
    except Exception as exc:
        logger.debug("[Cache] delete pattern failed pattern=%s error=%s", pattern, exc)
    finally:
        await client.aclose()



async def warm_cache(db: Any = None) -> None:
    """Pre-load frequently accessed data into Redis during application startup.

    Warms the cache with:
    - Settings for all active pages
    - Active knowledge base entries per page
    """
    logger.info("[Cache] Starting cache warming...")

    # Warm settings for all active pages
    if db is not None:
        try:
            from sqlalchemy import select
            from app.db import Page, Settings as SettingsModel, KnowledgeBase

            # Load settings for all active pages
            pages_result = await db.execute(
                select(Page).where(Page.is_active == True)
            )
            active_pages = pages_result.scalars().all()

            for page in active_pages:
                # Warm settings cache
                settings_result = await db.execute(
                    select(SettingsModel).where(SettingsModel.page_id == page.page_id)
                )
                page_settings = settings_result.scalars().first()
                if page_settings:
                    cache_key = f"settings:{page.page_id}"
                    await set_json(cache_key, {
                        "confidence_threshold": page_settings.confidence_threshold,
                        "auto_escalate_angry": page_settings.auto_escalate_angry,
                        "default_language": page_settings.default_language,
                        "warmup_mode": page_settings.warmup_mode,
                        "safe_reply_ar": page_settings.safe_reply_ar,
                        "safe_reply_en": page_settings.safe_reply_en,
                        "reply_mode": page_settings.reply_mode,
                    }, ttl_secs=600)
                    logger.debug("[Cache] Warmed settings for page=%s", page.page_id)

                # Warm knowledge base cache
                kb_cache_key = f"kb:active:{page.page_id}"
                kb_result = await db.execute(
                    select(KnowledgeBase)
                    .where(
                        KnowledgeBase.is_active == True,
                        KnowledgeBase.page_id == page.page_id,
                    )
                    .order_by(KnowledgeBase.usage_count.desc())
                    .limit(4)
                )
                kb_entries = kb_result.scalars().all()
                if kb_entries:
                    kb_data = [
                        {"question": e.question, "answer": e.answer, "category": e.category}
                        for e in kb_entries
                    ]
                    await set_json(kb_cache_key, kb_data, ttl_secs=300)
                    logger.debug(
                        "[Cache] Warmed KB entries (%d) for page=%s",
                        len(kb_entries), page.page_id,
                    )

            logger.info(
                "[Cache] Cache warming complete (%d active pages)",
                len(active_pages),
            )
        except Exception as exc:
            logger.warning("[Cache] Cache warming failed: %s", exc)
    else:
        logger.info("[Cache] No DB session provided; skipping cache warming")
