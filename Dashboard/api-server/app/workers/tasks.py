"""
Celery background tasks — PRD v2.0 compliant.

Tasks:
  process_webhook_event_{high|normal|low}  — Priority-routed AI processing
  handle_dead_letter                        — DLQ admin alert
  refresh_expired_tokens                    — Token lifecycle (every 30min)
  check_token_health                        — Full token audit (daily)
  refresh_single_token                      — Single-page token refresh
  check_meta_rate_limits                    — Rate usage monitoring (6-hourly)
  update_escalation_metrics                 — Prometheus gauge sync
  cleanup_old_data                          — Archive resolved conversations
  send_telegram_alert_task                  — Async Telegram notification
  optimize_prompts_weekly                   — DSPy weekly optimizer
"""
import asyncio
import uuid
from datetime import datetime, timezone
import logging
from app.workers.celery_app import celery
from app.ai.dspy_optimizer import get_optimizer

logger = logging.getLogger(__name__)


def _run_async(coro):
    """Run async coroutine in Celery sync context."""
    try:
        loop = asyncio.get_event_loop()
        if loop.is_closed():
            raise RuntimeError("closed")
    except RuntimeError:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
    return loop.run_until_complete(coro)


# ─────────────────────────────────────────────
# Distributed Lock Helpers
# ─────────────────────────────────────────────

def _acquire_redis_lock(key: str, ttl_seconds: int = 300) -> bool:
    """
    Try to acquire a Redis-based distributed lock using SETNX with TTL.
    Returns True if lock acquired, False if already held.
    """
    try:
        import redis as sync_redis
        from app.config import get_settings
        settings = get_settings()
        r = sync_redis.from_url(settings.redis_url, socket_connect_timeout=2, socket_timeout=2)
        acquired = r.set(key, "1", nx=True, ex=ttl_seconds)
        r.close()
        return bool(acquired)
    except Exception:
        # If Redis unavailable, allow processing (fail open)
        return True


def _release_redis_lock(key: str) -> None:
    """Release a previously acquired Redis lock."""
    try:
        import redis as sync_redis
        from app.config import get_settings
        settings = get_settings()
        r = sync_redis.from_url(settings.redis_url, socket_connect_timeout=2, socket_timeout=2)
        r.delete(key)
        r.close()
    except Exception:
        pass


def _acquire_shadow_lock(conv_id: str, ttl_seconds: int = 60) -> bool:
    """Lock for shadow mode decisions to prevent double-approve/reject."""
    return _acquire_redis_lock(f"shadow_lock:{conv_id}", ttl_seconds)

# ─────────────────────────────────────────────
# Webhook Processing (3 priority levels)
# ─────────────────────────────────────────────

# ─────────────────────────────────────────────
# Helper functions for _process_event decomposition
# ─────────────────────────────────────────────


async def _validate_page(event_data: dict, db) -> tuple:
    """
    Look up the page and validate it is active.
    Returns (page, existing_conversation_id_or_None).
    Returns (None, None) if the page is missing or inactive.
    """
    from sqlalchemy import select
    from app.db import Page, Conversation

    page_id = event_data.get("page_id", "")
    comment_id = event_data.get("comment_id", "")

    page_result = await db.execute(select(Page).where(Page.id == page_id))
    page = page_result.scalar_one_or_none()
    if not page or not page.is_active:
        return None, None

    existing = await db.execute(
        select(Conversation.id).where(
            Conversation.comment_id == comment_id,
            Conversation.page_id == str(page.id),
        )
    )
    existing_id = existing.scalar_one_or_none()
    return page, existing_id


async def _upsert_customer(event_data: dict, page, db):
    """Create or update the customer record for the commenter."""
    from sqlalchemy import select
    from app.db import Customer

    fb_user_id = event_data.get("fb_user_id", "")
    user_name = event_data.get("user_name", "Unknown")

    cust_result = await db.execute(
        select(Customer).where(
            Customer.facebook_id == fb_user_id,
            Customer.page_id == str(page.id),
        )
    )
    customer = cust_result.scalar_one_or_none()
    if not customer:
        customer = Customer(
            id=str(uuid.uuid4()),
            page_id=str(page.id),
            facebook_id=fb_user_id,
            full_name=user_name,
            first_contact_date=datetime.now(timezone.utc),
            last_interaction=datetime.now(timezone.utc),
            interaction_count=1,
        )
        db.add(customer)
    else:
        customer.interaction_count += 1
        customer.last_interaction = datetime.now(timezone.utc)

    await db.flush()
    return customer


async def _run_ai_pipeline(comment_text: str, page_id: str, db, customer, page_settings=None) -> dict:
    """Execute the AI pipeline and record metrics/feedback."""
    from app.ai.pipeline import process_comment
    from app.metrics import record_pipeline_result
    from app.services.crm import update_customer_crm
    from app.ai.dspy_optimizer import get_optimizer

    ai_result = await process_comment(comment_text, page_id, db, settings=page_settings)
    record_pipeline_result(ai_result)
    update_customer_crm(
        customer,
        intent=ai_result.get("intent"),
        sentiment=ai_result.get("sentiment"),
    )

    try:
        optimizer = get_optimizer()
        await optimizer.record_feedback(
            comment=comment_text,
            predicted_intent=ai_result.get("intent", "general"),
            actual_intent=None,
            predicted_sentiment=ai_result.get("sentiment", "neutral"),
            actual_sentiment=None,
            ai_reply=ai_result.get("ai_reply"),
        )
    except Exception:
        pass

    return ai_result

async def _create_conversation(
    event_data: dict, page, customer, ai_result: dict, db
):
    """Create the conversation record from the AI pipeline result."""
    from app.db import Conversation

    comment_id = event_data.get("comment_id", "")
    post_id = event_data.get("post_id", "")
    user_name = event_data.get("user_name", "Unknown")
    comment_text = event_data.get("comment_text", "")
    priority = ai_result.get("priority", "normal")

    conv = Conversation(
        id=str(uuid.uuid4()),
        page_id=str(page.id),
        page_name=page.name,
        platform=page.platform,
        comment_id=comment_id,
        post_id=post_id,
        customer_id=str(customer.id),
        customer_name=user_name,
        original_comment=comment_text,
        ai_reply=ai_result.get("ai_reply"),
        status="pending",
        intent=ai_result.get("intent"),
        sentiment=ai_result.get("sentiment"),
        confidence_score=ai_result.get("confidence", 0.0),
        language=ai_result.get("language", "ar"),
        is_shadow_mode=page.shadow_mode,
        escalation_reason=ai_result.get("escalation_reason"),
        guardrail_triggered=ai_result.get("guardrail_triggered", False),
        guardrail_reason=ai_result.get("guardrail_reason"),
        priority=priority,
        urgency=ai_result.get("urgency", "normal"),
        matched_rule_id=ai_result.get("matched_rule_id"),
        processing_time=ai_result.get("processing_time_ms", 0) / 1000.0,
    )
    return conv


async def _handle_escalation(conv, page, customer, event_data: dict, ai_result: dict, db) -> None:
    """Create an escalation record and send admin notification."""
    from app.db import Escalation
    from app.metrics import record_escalation
    from app.services.notifications import notify_admin

    comment_text = event_data.get("comment_text", "")
    user_name = event_data.get("user_name", "Unknown")

    esc_priority = _calc_priority(
        ai_result.get("sentiment", "neutral"),
        ai_result.get("priority", "normal"),
    )
    esc = Escalation(
        id=str(uuid.uuid4()),
        conversation_id=str(conv.id),
        page_id=str(page.id),
        page_name=page.name,
        customer_id=str(customer.id),
        customer_name=user_name,
        original_comment=comment_text,
        reason=ai_result.get("escalation_reason", "Low confidence"),
        priority=esc_priority,
        status="open",
    )
    db.add(esc)
    record_escalation(esc_priority)

    await notify_admin(
        title="Escalation: " + user_name,
        message=(
            "Comment: " + comment_text[:200] + "\n"
            + "Reason: " + esc.reason + "\n"
            + "Page: " + page.name
        ),
        priority=esc_priority,
        escalation_id=str(esc.id),
    )

async def _send_auto_reply(conv, page, ai_result: dict, page_settings, db) -> None:
    """Attempt to auto-reply if conditions are met."""
    from app.services.meta_api import check_meta_rate_limit
    from app.services.runtime_settings import get_runtime_settings

    if not page.auto_reply_enabled:
        return

    settings_obj = await get_runtime_settings(db)
    warmup = getattr(settings_obj, "warmup_mode", True) if settings_obj else True

    if not _is_within_auto_reply_window(page):
        conv.status = "pending"
        logger.info("[WebhookTask] Auto-reply outside active date window")
        return

    reply_rl = await check_meta_rate_limit(str(page.id), "reply", warmup=warmup)
    if reply_rl["allowed"]:
        from app.services.facebook import post_reply_to_comment, post_private_reply_to_comment
        lang = ai_result.get("language", "ar")
        intent = ai_result.get("intent", "general")
        ai_reply_text = ai_result.get("ai_reply")
        
        # Smart Dynamic Routing based on Intent
        # If it is a compliment or general feedback, reply PUBLICLY with the beautiful AI response and do NOT send a private DM.
        is_compliment_or_general = intent in ("compliment", "general")
        
        if is_compliment_or_general:
            public_msg = ai_reply_text
            do_private = False
            priv_msg = None
        else:
            if page_settings:
                public_msg = (
                    page_settings.public_reply_message_ar
                    if lang == "ar"
                    else page_settings.public_reply_message_en
                )
            else:
                public_msg = "تم التواصل معك على الخاص"
            do_private = settings_obj.enable_private_replies if settings_obj else True
            priv_msg = ai_reply_text
        
        # 1. Post Public Comment Reply
        pub_result = await post_reply_to_comment(db, conv, public_msg)
        
        # 2. Post Private Messenger Reply (if required for lead/sales/support)
        priv_result = {"success": False}
        if do_private and priv_msg:
            priv_result = await post_private_reply_to_comment(db, conv, priv_msg)
        else:
            priv_result = {"success": True, "skipped": True}

        if pub_result["success"] or priv_result["success"]:
            conv.ai_reply = public_msg
            conv.admin_reply = priv_msg
            conv.status = "replied"
            conv.replied_at = datetime.now(timezone.utc)
            if not pub_result["success"]:
                logger.warning(f"[WebhookTask] Public reply failed for {conv.id}: {pub_result.get('error')}")
            if do_private and not priv_result["success"]:
                logger.warning(f"[WebhookTask] Private reply failed for {conv.id}: {priv_result.get('error')}")
        else:
            conv.status = "escalated"
            conv.escalation_reason = f"FB reply failed. Public: {pub_result.get('error')}, Private: {priv_result.get('error')}"
    else:
        conv.status = "pending"
        logger.warning("[WebhookTask] Reply rate limit for page %s", page.name)



# ─────────────────────────────────────────────
# Distributed Lock Helpers
# ─────────────────────────────────────────────

def _process_event(self, event_data: dict):
    """Core event processing - thin orchestrator calling focused helpers."""
    async def _inner():
        from app.db import AsyncSessionLocal
        from app.services.notifications import notify_admin
        from app.services.runtime_settings import get_runtime_settings

        async with AsyncSessionLocal() as db:
            # 1. Validate page and check for duplicates
            page, existing_id = await _validate_page(event_data, db)
            if not page:
                return {"status": "ignored", "reason": "page_inactive_or_missing"}
            if existing_id:
                return {"status": "duplicate", "conversation_id": str(existing_id)}

            comment_text = event_data.get("comment_text", "")
            user_name = event_data.get("user_name", "Unknown")
            comment_id = event_data.get("comment_id", "")

            # Distributed lock: prevent concurrent processing of same comment
            lock_key = f"webhook_lock:{page.id}:{comment_id}"
            if not _acquire_redis_lock(lock_key, ttl_seconds=300):
                logger.info("[WebhookTask] Lock not acquired for comment=%s, skipping", comment_id)
                return {"status": "locked", "reason": "comment_already_being_processed"}

            # 2. Load settings
            settings_obj = await get_runtime_settings(db)
            page_settings = await get_runtime_settings(db, str(page.id)) or settings_obj

            # 3. Notify admin of new comment
            await notify_admin(
                title="New comment: " + user_name,
                message="'" + comment_text[:200] + "'",
                priority="low",
            )

            # 4. Upsert customer record
            customer = await _upsert_customer(event_data, page, db)

            # 5. Run AI pipeline
            ai_result = await _run_ai_pipeline(comment_text, str(page.id), db, customer, page_settings=page_settings)

            # 6. Create conversation record
            conv = await _create_conversation(event_data, page, customer, ai_result, db)

            # 7. Route based on action
            action = ai_result.get("action", "reply")

            if page.shadow_mode:
                conv.status = "shadow_pending"
            elif action == "escalate":
                conv.status = "escalated"
                await _handle_escalation(conv, page, customer, event_data, ai_result, db)
            elif action == "flag_review":
                conv.status = "pending"
                if page.auto_reply_enabled and _is_within_auto_reply_window(page):
                    conv.status = "replied"
                    conv.replied_at = datetime.now(timezone.utc)
            elif action == "reply":
                await _send_auto_reply(conv, page, ai_result, page_settings, db)

            db.add(conv)
            await db.commit()
            return {"status": conv.status, "conversation_id": str(conv.id)}

    try:
        return _run_async(_inner())
    except Exception as exc:
        logger.error("[Task] process_event failed: %s", exc)
        raise self.retry(exc=exc, countdown=_retry_countdown(self.request.retries))


def _is_within_auto_reply_window(page) -> bool:
    if not page or not page.auto_reply_end_date:
        return False
    now = datetime.now(timezone.utc)
    end_date = page.auto_reply_end_date
    if end_date.tzinfo is None:
        end_date = end_date.replace(tzinfo=timezone.utc)
    return now <= end_date


def _calc_priority(sentiment: str, pipeline_priority: str = "normal") -> str:
    if sentiment == "angry" or pipeline_priority == "high":
        return "critical"
    if sentiment == "negative":
        return "high"
    return "medium"


def _retry_countdown(attempt: int) -> int:
    """Exponential backoff: 30s, 120s, 600s."""
    return [30, 120, 600][min(attempt, 2)]


@celery.task(bind=True, max_retries=3, name="app.workers.tasks.process_webhook_event_high")
def process_webhook_event_high(self, event_data: dict):
    """High priority - angry/refund/legal events."""
    return _process_event(self, event_data)


@celery.task(bind=True, max_retries=3, name="app.workers.tasks.process_webhook_event")
def process_webhook_event(self, event_data: dict):
    """Normal priority - purchase/price inquiry."""
    return _process_event(self, event_data)


@celery.task(bind=True, max_retries=3, name="app.workers.tasks.process_webhook_event_low")
def process_webhook_event_low(self, event_data: dict):
    """Low priority - general comments."""
    return _process_event(self, event_data)


@celery.task(name="app.workers.tasks.handle_dead_letter")
def handle_dead_letter(event_data: dict, original_task: str, error: str):
    """Dead Letter Queue handler — notify admin after all retries exhausted."""
    async def _alert():
        from app.services.notifications import notify_admin
        await notify_admin(
            title="💀 Dead Letter Queue — Event Failed",
            message=(
                f"Task `{original_task}` exhausted all retries.\n"
                f"Error: {error}\n"
                f"Comment: {str(event_data.get('comment_text', ''))[:100]}\n"
                f"Page: {event_data.get('page_id', 'unknown')}"
            ),
            priority="critical",
        )
    _run_async(_alert())


# ─────────────────────────────────────────────
# Token Lifecycle Management
# ─────────────────────────────────────────────

@celery.task(name="app.workers.tasks.refresh_expired_tokens")
def refresh_expired_tokens():
    """Check and flag tokens expiring within 15 days (PRD §4.1)."""
    async def _refresh():
        from sqlalchemy import select
        from datetime import datetime, timezone, timedelta
        from app.db import AsyncSessionLocal, Page

        soon = datetime.now(timezone.utc) + timedelta(days=15)
        refreshed = 0
        alerted = 0

        async with AsyncSessionLocal() as db:
            result = await db.execute(
                select(Page).where(
                    Page.token_expires_at.isnot(None),
                    Page.token_expires_at <= soon,
                    Page.is_active == True,
                )
            )
            pages = result.scalars().all()
            for page in pages:
                days_left = (page.token_expires_at - datetime.now(timezone.utc)).days
                if page.token_status != "expired":
                    page.token_status = "expiring_soon" if days_left > 0 else "expired"
                    alerted += 1
            await db.commit()

        # Notify admin for each page needing attention
        for page in pages:
            try:
                from app.services.notifications import notify_admin
                days_left = (page.token_expires_at - datetime.now(timezone.utc)).days
                await notify_admin(
                    title=f"⏰ Token Expiring: {page.name}",
                    message=f"Page '{page.name}' ({page.platform}) token expires in {days_left} day(s). Please refresh it from the Tokens page.",
                    priority="high" if days_left <= 3 else "medium",
                )
            except Exception:
                pass

        return {"checked": len(pages), "alerted": alerted}

    return _run_async(_refresh())


@celery.task(name="app.workers.tasks.check_token_health")
def check_token_health():
    """Daily full token health audit — attempt auto-refresh for expiring tokens."""
    async def _health():
        from sqlalchemy import select
        from datetime import datetime, timezone, timedelta
        from app.db import AsyncSessionLocal, Page
        from app.services.token_service import refresh_page_token

        refresh_window = datetime.now(timezone.utc) + timedelta(days=15)
        results = {"refreshed": 0, "failed": 0, "ok": 0}

        async with AsyncSessionLocal() as db:
            result = await db.execute(
                select(Page).where(Page.is_active == True)
            )
            pages = result.scalars().all()

        for page in pages:
            if page.token_expires_at and page.token_expires_at <= refresh_window:
                # Attempt auto-refresh
                outcome = await refresh_page_token(str(page.id))
                if outcome["success"]:
                    results["refreshed"] += 1
                else:
                    results["failed"] += 1
            else:
                results["ok"] += 1

        logger.info("[TokenHealth] %s", results)
        return results

    return _run_async(_health())


@celery.task(bind=True, max_retries=2, name="app.workers.tasks.refresh_single_token")
def refresh_single_token(self, page_id: str):
    """Manually trigger token refresh for a single page."""
    async def _refresh():
        from app.services.token_service import refresh_page_token
        return await refresh_page_token(page_id)
    try:
        return _run_async(_refresh())
    except Exception as exc:
        raise self.retry(exc=exc, countdown=60)


# ─────────────────────────────────────────────
# Meta API Rate Limit Monitoring
# ─────────────────────────────────────────────

@celery.task(name="app.workers.tasks.check_meta_rate_limits")
def check_meta_rate_limits():
    """Check current Meta API usage across all active pages and alert at thresholds."""
    async def _check():
        from sqlalchemy import select
        from app.db import AsyncSessionLocal, Page

        async with AsyncSessionLocal() as db:
            result = await db.execute(select(Page).where(Page.is_active == True))
            pages = result.scalars().all()

        report = []
        for page in pages:
            # Just check current usage without consuming a call
            from app.services.meta_api import _get_redis, BUCKET_WINDOW_SECS, GRAPH_API_LIMIT_PER_HOUR
            import time
            r = _get_redis()
            if r:
                key = f"meta_rl:api:{page.id}"
                now = time.time()
                count = r.zcount(key, now - BUCKET_WINDOW_SECS, now)
                usage_pct = count / GRAPH_API_LIMIT_PER_HOUR
                report.append({"page": page.name, "usage_pct": usage_pct, "calls": count})

        return report

    return _run_async(_check())


# ─────────────────────────────────────────────
# Prometheus Metrics Sync
# ─────────────────────────────────────────────

@celery.task(name="app.workers.tasks.update_escalation_metrics")
def update_escalation_metrics():
    """Update open_escalations Prometheus gauge."""
    async def _update():
        from sqlalchemy import select, func
        from app.db import AsyncSessionLocal, Escalation
        try:
            async with AsyncSessionLocal() as db:
                result = await db.execute(
                    select(func.count(Escalation.id)).where(Escalation.status == "open")
                )
                count = result.scalar() or 0
            from app.metrics import open_escalations_gauge, PROMETHEUS_AVAILABLE
            if PROMETHEUS_AVAILABLE:
                open_escalations_gauge.set(count)
        except Exception as e:
            logger.warning("[Metrics] Escalation gauge update failed: %s", e)

    _run_async(_update())


# ─────────────────────────────────────────────
# Maintenance
# ─────────────────────────────────────────────

@celery.task(name="app.workers.tasks.cleanup_old_data")
def cleanup_old_data():
    """Archive resolved conversations older than 90 days."""
    async def _cleanup():
        from sqlalchemy import delete
        from datetime import datetime, timezone, timedelta
        from app.db import AsyncSessionLocal, Conversation

        cutoff = datetime.now(timezone.utc) - timedelta(days=90)
        async with AsyncSessionLocal() as db:
            result = await db.execute(
                delete(Conversation).where(
                    Conversation.created_at < cutoff,
                    Conversation.status == "resolved",
                )
            )
            await db.commit()
            return result.rowcount

    return _run_async(_cleanup())


@celery.task(name="app.workers.tasks.send_telegram_alert_task")
def send_telegram_alert_task(customer_name: str, comment: str, priority: str, escalation_id: str):
    async def _send():
        from app.services.notifications import notify_admin
        await notify_admin(
            title=f"🚨 New Escalation: {customer_name}",
            message=f"Comment: {comment[:200]}",
            priority=priority,
            escalation_id=escalation_id,
        )
    _run_async(_send())


@celery.task(name="app.workers.tasks.optimize_prompts_weekly")
def optimize_prompts_weekly():
    """Weekly DSPy prompt optimization using accumulated feedback."""
    async def _optimize():
        optimizer = get_optimizer()
        await optimizer.optimize_weekly()

    _run_async(_optimize())


@celery.task(name="app.workers.tasks.poll_whatsapp_messages")
def poll_whatsapp_messages():
    """Poll all active WhatsApp bridges for new messages."""
    async def _poll():
        from sqlalchemy import select
        from app.db import AsyncSessionLocal, WhatsAppBridge, Page, Conversation
        import aiohttp

        async with AsyncSessionLocal() as db:
            result = await db.execute(
                select(WhatsAppBridge).where(WhatsAppBridge.is_active == True)
            )
            bridges = result.scalars().all()

            for bridge in bridges:
                try:
                    async with aiohttp.ClientSession() as session:
                        async with session.get(
                            f"http://127.0.0.1:{bridge.port}/messages",
                            timeout=aiohttp.ClientTimeout(total=10)
                        ) as resp:
                            if resp.status == 200:
                                messages = await resp.json()
                                for msg in messages:
                                    # Process each WhatsApp message
                                    await _process_whatsapp_message(bridge, msg, db)
                    
                    bridge.last_poll_at = datetime.now(timezone.utc)
                    bridge.status = "connected"
                except Exception as e:
                    logger.error(f"[WhatsAppPoll] Failed to poll bridge {bridge.id} on port {bridge.port}: {e}")
                    bridge.status = "error"
  
            await db.commit()

    return _run_async(_poll())


async def _process_whatsapp_message(bridge, msg, db):
    """Helper to process a single WhatsApp message from the bridge."""
    from app.db import Page, Conversation, Customer
    from sqlalchemy import select
    import uuid

    page_id = bridge.page_id
    chat_id = msg.get("chatId")
    message_text = msg.get("body", "")
    customer_name = msg.get("senderName", "WhatsApp User")
    whatsapp_user_id = msg.get("senderId")

    if not message_text:
        return

    # 1. Get Page
    page_result = await db.execute(select(Page).where(Page.id == page_id))
    page = page_result.scalar_one_or_none()
    if not page or not page.is_active:
        return

    # 2. Check for duplicate message_id
    message_id = msg.get("messageId")
    existing = await db.execute(
        select(Conversation).where(Conversation.comment_id == message_id)
    )
    if existing.scalar_one_or_none():
        return

    # 3. Upsert Customer
    cust_result = await db.execute(
        select(Customer).where(Customer.whatsapp_id == whatsapp_user_id)
    )
    customer = cust_result.scalar_one_or_none()
    if not customer:
        customer = Customer(
            id=str(uuid.uuid4()),
            page_id=str(page.id),
            whatsapp_id=whatsapp_user_id,
            full_name=customer_name,
            interaction_count=1,
        )
        db.add(customer)
    else:
        customer.interaction_count += 1
        customer.last_interaction = datetime.now(timezone.utc)

    await db.flush()

    # 4. Run AI Pipeline
    from app.ai.pipeline import process_comment
    ai_result = await process_comment(message_text, str(page.id), db)

    # 5. Create Conversation
    conv = Conversation(
        id=str(uuid.uuid4()),
        page_id=str(page.id),
        page_name=page.name,
        platform="whatsapp",
        comment_id=message_id,
        post_id="whatsapp_chat",
        customer_id=str(customer.id),
        customer_name=customer_name,
        original_comment=message_text,
        ai_reply=ai_result.get("ai_reply"),
        status="pending",
        intent=ai_result.get("intent"),
        sentiment=ai_result.get("sentiment"),
        confidence_score=ai_result.get("confidence", 0.0),
        language=ai_result.get("language", "ar"),
        is_shadow_mode=page.shadow_mode,
        priority=ai_result.get("priority", "normal"),
        urgency=ai_result.get("urgency", "normal"),
        processing_time=ai_result.get("processing_time_ms", 0) / 1000.0,
    )
    db.add(conv)

    # 6. If action == "reply" and not shadow mode, send back to bridge
    if ai_result.get("action") == "reply" and not page.shadow_mode and ai_result.get("ai_reply"):
        await _send_whatsapp_reply(bridge, chat_id, ai_result["ai_reply"], message_id)
        conv.status = "replied"
        conv.replied_at = datetime.now(timezone.utc)
    elif page.shadow_mode:
        conv.status = "shadow_pending"

    await db.flush()


async def _send_whatsapp_reply(bridge, chat_id, text, reply_to=None):
    """Send reply back to WhatsApp bridge."""
    import aiohttp
    payload = {
        "chatId": chat_id,
        "message": text,
    }
    if reply_to:
        payload["replyTo"] = reply_to

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(
                f"http://127.0.0.1:{bridge.port}/send",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=20)
            ) as resp:
                if resp.status != 200:
                    logger.error(f"[WhatsAppReply] Failed to send reply to bridge: {await resp.text()}")
    except Exception as e:
        logger.error(f"[WhatsAppReply] Error sending reply: {e}")


@celery.task(name="app.workers.tasks.process_scheduled_posts")
def process_scheduled_posts():
    """Check for due posts and publish them."""
    async def _process():
        from sqlalchemy import select
        from app.db import AsyncSessionLocal, ScheduledPost, Page
        
        async with AsyncSessionLocal() as db:
            now = datetime.now(timezone.utc)
            result = await db.execute(
                select(ScheduledPost).where(
                    ScheduledPost.status == "pending",
                    ScheduledPost.scheduled_at <= now
                )
            )
            posts = result.scalars().all()
            
            for post in posts:
                try:
                    # Get page token
                    page_result = await db.execute(select(Page).where(Page.id == post.page_id))
                    page = page_result.scalar_one_or_none()
                    if not page:
                        post.status = "failed"
                        post.error = "Page not found"
                        continue
                    
                    # Post to platform
                    if post.platform == "facebook":
                        outcome = await _post_to_facebook(page, post)
                    elif post.platform == "instagram":
                        outcome = await _post_to_instagram(page, post)
                    elif post.platform == "whatsapp":
                        outcome = await _post_to_whatsapp(page, post)
                    else:
                        outcome = {"success": False, "error": f"Unsupported platform: {post.platform}"}
                    
                    if outcome["success"]:
                        post.status = "posted"
                        post.post_id = outcome["post_id"]
                        post.posted_at = datetime.now(timezone.utc)
                    else:
                        post.status = "failed"
                        post.error = outcome.get("error", "Unknown error")
                        
                except Exception as e:
                    logger.error(f"[ScheduledPost] Failed to process post {post.id}: {e}")
                    post.status = "failed"
                    post.error = str(e)
            
            await db.commit()

    return _run_async(_process())


async def _post_to_facebook(page, post):
    """Post to Facebook Page feed."""
    import httpx
    from app.services.token_service import get_valid_token
    token = await get_valid_token(page)
    
    url = f"https://graph.facebook.com/v19.0/{page.page_id}/feed"
    data = {"message": post.message, "access_token": token}
    if post.media_url:
        # If it's a photo, use /photos endpoint instead
        url = f"https://graph.facebook.com/v19.0/{page.page_id}/photos"
        data["url"] = post.media_url
        data["caption"] = post.message

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(url, data=data)
        res = resp.json()
        if "id" in res:
            return {"success": True, "post_id": res["id"]}
        return {"success": False, "error": str(res.get("error", res))}


async def _post_to_instagram(page, post):
    """Post to Instagram Business account (Feed)."""
    # Note: Instagram posting via API is restricted to Business accounts and requires container flow
    return {"success": False, "error": "Instagram posting requires business container flow (not yet fully implemented)"}


async def _post_to_whatsapp(page, post):
    """Post to WhatsApp (Broadcast/Status via bridge)."""
    # Using the bridge /send endpoint
    from app.db import AsyncSessionLocal, WhatsAppBridge
    from sqlalchemy import select
    
    async with AsyncSessionLocal() as db:
        result = await db.execute(select(WhatsAppBridge).where(WhatsAppBridge.page_id == str(page.id)))
        bridge = result.scalar_one_or_none()
    
    if not bridge:
        return {"success": False, "error": "WhatsApp bridge not found for this page"}
       
    import aiohttp
    payload = {"message": post.message}
    # Note: Broadcast logic depends on the bridge implementation
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(f"http://127.0.0.1:{bridge.port}/broadcast", json=payload, timeout=20) as resp:
                if resp.status == 200:
                    return {"success": True, "post_id": "whatsapp_broadcast_" + str(int(datetime.now().timestamp()))}
                return {"success": False, "error": await resp.text()}
    except Exception as e:
        return {"success": False, "error": str(e)}
