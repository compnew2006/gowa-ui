from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func, cast, Float, case
from datetime import datetime, timezone, timedelta
from typing import Optional
from app.deps import get_db
from app.db import Conversation, Customer, Escalation, Page
from app.schemas import (
    DashboardStats,
    ConversationAnalyticsPoint,
    IntentItem,
    SentimentItem,
    TokenStatus,
)

router = APIRouter(tags=["analytics"])


@router.get("/analytics/dashboard", response_model=DashboardStats)
async def dashboard_stats(
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    def _conv_q(extra=None):
        q = select(func.count(Conversation.id))
        if page_id:
            q = q.where(Conversation.page_id == page_id)
        if extra is not None:
            q = q.where(extra)
        return q

    def _cust_q(extra=None):
        q = select(func.count(Customer.id))
        if page_id:
            q = q.where(Customer.page_id == page_id)
        if extra is not None:
            q = q.where(extra)
        return q

    def _esc_q(extra=None):
        q = select(func.count(Escalation.id))
        if page_id:
            q = q.where(Escalation.page_id == page_id)
        if extra is not None:
            q = q.where(extra)
        return q

    total_conversations = (await db.execute(_conv_q())).scalar_one()
    pending = (await db.execute(_conv_q(Conversation.status == "pending"))).scalar_one()
    open_escalations = (await db.execute(_esc_q(Escalation.status == "open"))).scalar_one()
    total_customers = (await db.execute(_cust_q())).scalar_one()
    high_intent = (await db.execute(_cust_q(Customer.purchase_intent == "High"))).scalar_one()
    shadow_reviews = (await db.execute(_conv_q(Conversation.is_shadow_mode == True))).scalar_one()

    conf_q = select(func.avg(Conversation.confidence_score)).where(Conversation.confidence_score.isnot(None))
    if page_id:
        conf_q = conf_q.where(Conversation.page_id == page_id)
    avg_conf = (await db.execute(conf_q)).scalar_one() or 0.0

    replied_q = _conv_q(Conversation.status.in_(["replied", "resolved"]))
    replied_count = (await db.execute(replied_q)).scalar_one()
    auto_reply_rate = (replied_count / total_conversations * 100) if total_conversations > 0 else 0.0

    time_q = select(func.avg(Conversation.processing_time)).where(Conversation.processing_time.isnot(None))
    if page_id:
        time_q = time_q.where(Conversation.page_id == page_id)
    avg_time = (await db.execute(time_q)).scalar_one() or 0.0

    token_healthy = (await db.execute(select(func.count(Page.id)).where(Page.token_status == "valid"))).scalar_one()
    token_expiring = (await db.execute(select(func.count(Page.id)).where(Page.token_status == "expiring_soon"))).scalar_one()
    token_expired = (await db.execute(select(func.count(Page.id)).where(Page.token_status.in_(["expired", "error"])))).scalar_one()

    return DashboardStats(
        total_conversations=total_conversations,
        pending_conversations=pending,
        open_escalations=open_escalations,
        total_customers=total_customers,
        high_intent_leads=high_intent,
        avg_confidence_score=round(float(avg_conf), 4),
        auto_reply_rate=round(float(auto_reply_rate), 2),
        avg_response_time_seconds=round(float(avg_time), 2),
        shadow_mode_reviews=shadow_reviews,
        token_healthy=token_healthy,
        token_expiring_soon=token_expiring,
        token_expired=token_expired,
    )


@router.get("/analytics/conversations", response_model=list[ConversationAnalyticsPoint])
async def conversation_analytics(
    period: str = Query("7d"),
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    days = 30 if period == "30d" else 7
    results = []
    now = datetime.now(timezone.utc)

    for i in range(days - 1, -1, -1):
        day_start = (now - timedelta(days=i)).replace(hour=0, minute=0, second=0, microsecond=0)
        day_end = day_start + timedelta(days=1)

        def _day_q(extra=None):
            q = select(func.count(Conversation.id)).where(
                Conversation.created_at >= day_start,
                Conversation.created_at < day_end,
            )
            if page_id:
                q = q.where(Conversation.page_id == page_id)
            if extra is not None:
                q = q.where(extra)
            return q

        total = (await db.execute(_day_q())).scalar_one()
        replied = (await db.execute(_day_q(Conversation.status.in_(["replied", "resolved"])))).scalar_one()
        escalated = (await db.execute(_day_q(Conversation.status == "escalated"))).scalar_one()

        results.append(ConversationAnalyticsPoint(
            date=day_start.strftime("%Y-%m-%d"),
            total=total,
            replied=replied,
            escalated=escalated,
        ))

    return results


@router.get("/analytics/intents", response_model=list[IntentItem])
async def intent_breakdown(
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    q = (
        select(Conversation.intent, func.count(Conversation.id).label("count"))
        .where(Conversation.intent.isnot(None))
        .group_by(Conversation.intent)
        .order_by(func.count(Conversation.id).desc())
    )
    if page_id:
        q = q.where(Conversation.page_id == page_id)
    result = await db.execute(q)
    rows = result.all()
    total = sum(r.count for r in rows)
    return [
        IntentItem(
            intent=r.intent,
            count=r.count,
            percentage=round(r.count / total * 100, 1) if total else 0.0,
        )
        for r in rows
    ]


@router.get("/analytics/sentiment", response_model=list[SentimentItem])
async def sentiment_breakdown(
    page_id: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db),
):
    q = (
        select(Conversation.sentiment, func.count(Conversation.id).label("count"))
        .where(Conversation.sentiment.isnot(None))
        .group_by(Conversation.sentiment)
        .order_by(func.count(Conversation.id).desc())
    )
    if page_id:
        q = q.where(Conversation.page_id == page_id)
    result = await db.execute(q)
    rows = result.all()
    total = sum(r.count for r in rows)
    return [
        SentimentItem(
            sentiment=r.sentiment,
            count=r.count,
            percentage=round(r.count / total * 100, 1) if total else 0.0,
        )
        for r in rows
    ]


@router.get("/analytics/tokens", response_model=list[TokenStatus])
async def token_statuses(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Page).order_by(Page.name))
    pages = result.scalars().all()
    return [
        TokenStatus(
            id=p.id,
            name=p.name,
            platform=p.platform,
            token_status=p.token_status,
            token_expires_at=p.token_expires_at,
            token_last_refreshed_at=p.token_last_refreshed_at,
            token_last_error=p.token_last_error,
        )
        for p in pages
    ]
