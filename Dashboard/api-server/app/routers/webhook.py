"""
Meta webhook router.

The request path now does only security checks, page lookup, deduplication, and
queue enqueue. AI processing, CRM updates, conversation creation, escalation
creation, and notifications run in Celery workers.
"""
from __future__ import annotations

import json
import logging
import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, Request
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import get_settings
from app.db import Page
from app.deps import get_db
from app.metrics import record_webhook_event
from app.middleware.hmac_validator import verify_meta_signature
from app.middleware.rate_limit import check_rate_limit, is_duplicate_webhook
from app.services.meta_api import check_meta_rate_limit
from app.services.runtime_settings import get_runtime_settings

logger = logging.getLogger(__name__)
_cfg = get_settings()
router = APIRouter(tags=["webhook"])


@router.get("/webhook/meta")
async def verify_webhook(
    hub_mode: str = Query(None, alias="hub.mode"),
    hub_verify_token: str = Query(None, alias="hub.verify_token"),
    hub_challenge: str = Query(None, alias="hub.challenge"),
    db: AsyncSession = Depends(get_db),
):
    settings = await get_runtime_settings(db)
    expected_token = settings.webhook_verify_token if settings else "verify_token_change_me"

    if hub_mode == "subscribe" and hub_verify_token == expected_token:
        return int(hub_challenge)
    raise HTTPException(status_code=403, detail="Verification failed")


@router.post("/webhook/meta", status_code=200)
async def receive_webhook(request: Request, db: AsyncSession = Depends(get_db)):
    client_ip = (
        request.headers.get("X-Forwarded-For", "").split(",")[0].strip()
        or (request.client.host if request.client else "unknown")
    )
    allowed, _ = check_rate_limit(client_ip, _cfg.webhook_rate_limit_rpm, prefix="wh")
    if not allowed:
        raise HTTPException(status_code=429, detail="Webhook rate limit exceeded")

    if not _cfg.meta_app_secret and _cfg.environment == "production":
        logger.error("[Webhook] META_APP_SECRET is not configured on production! Rejecting POST request.")
        raise HTTPException(
            status_code=500,
            detail="META_APP_SECRET is not configured on the server. Webhook signature verification is required in production."
        )

    if _cfg.meta_app_secret:
        body_bytes = await verify_meta_signature(request, _cfg.meta_app_secret)
        try:
            body = json.loads(body_bytes)
        except Exception:
            raise HTTPException(status_code=400, detail="Invalid JSON body")
    else:
        try:
            body = await request.json()
        except Exception:
            raise HTTPException(status_code=400, detail="Invalid JSON format or empty request body")
        logger.warning("[Webhook] META_APP_SECRET not set — HMAC validation skipped (development mode)!")

    if body.get("object") != "page":
        return {"status": "ignored", "reason": "unsupported_object"}

    settings_obj = await get_runtime_settings(db)
    warmup = getattr(settings_obj, "warmup_mode", True) if settings_obj else True
    enqueued = 0
    ignored = 0

    for entry in body.get("entry", []):
        fb_page_id = entry.get("id")
        page_result = await db.execute(select(Page).where(Page.page_id == fb_page_id))
        page = page_result.scalar_one_or_none()
        if not page or not page.is_active:
            ignored += 1
            continue

        rl_result = await check_meta_rate_limit(str(page.id), "api", warmup=warmup)
        if not rl_result["allowed"]:
            logger.warning("[Webhook] Meta API rate limit hit for page %s", page.name)
            ignored += 1
            continue

        for change in entry.get("changes", []):
            if change.get("field") != "feed":
                ignored += 1
                continue

            value = change.get("value", {})
            comment_text = value.get("message", "")
            if not comment_text:
                ignored += 1
                continue

            from_id = (value.get("from") or {}).get("id")
            if from_id and from_id == fb_page_id:
                # Comment was written by the page itself (admin reply) - ignore to prevent loop
                logger.info("[Webhook] Comment from page itself ignored: %s", from_id)
                ignored += 1
                continue

            comment_id = value.get("comment_id", str(uuid.uuid4()))
            if is_duplicate_webhook(comment_id, str(page.id), ttl_secs=86400):
                record_webhook_event("comment", page.platform, is_duplicate=True)
                logger.debug("[Webhook] Duplicate event discarded: %s", comment_id)
                ignored += 1
                continue

            event_data = {
                "page_id": str(page.id),
                "comment_text": comment_text,
                "comment_id": comment_id,
                "post_id": value.get("post_id", ""),
                "fb_user_id": (value.get("from") or {}).get("id", ""),
                "user_name": (value.get("from") or {}).get("name", "Unknown"),
            }

            task, queue = _task_for_comment(comment_text)
            try:
                task.apply_async(args=[event_data], queue=queue)
            except Exception as exc:
                logger.exception("[Webhook] Failed to enqueue comment event: %s", exc)
                raise HTTPException(status_code=503, detail="Webhook queue unavailable")

            record_webhook_event("comment", page.platform)
            enqueued += 1

    return {"status": "accepted", "enqueued": enqueued, "ignored": ignored}


def _task_for_comment(comment_text: str):
    from app.workers.tasks import (
        process_webhook_event,
        process_webhook_event_high,
        process_webhook_event_low,
    )

    text = comment_text.lower()
    high_markers = (
        "refund",
        "chargeback",
        "lawyer",
        "legal",
        "court",
        "scam",
        "fraud",
        "استرجاع",
        "استرداد",
        "محامي",
        "قانون",
        "نصاب",
        "احتيال",
    )
    low_markers = ("thanks", "thank you", "great", "جميل", "شكرا", "شكرًا", "ممتاز")

    if any(marker in text for marker in high_markers):
        return process_webhook_event_high, "high"
    if any(marker in text for marker in low_markers):
        return process_webhook_event_low, "low"
    return process_webhook_event, "normal"
