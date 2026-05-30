"""
Advanced analytics router.

The heavy time-series endpoints use grouped SQL queries instead of one query
per day. The dashboard can also call /analytics/advanced-summary to avoid
multiple HTTP round-trips for the same page view.
"""
from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Any

from fastapi import APIRouter, Depends, Query
from sqlalchemy import case, func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Conversation, Customer
from app.deps import get_db

router = APIRouter(tags=["advanced-analytics"])


def _period_days(period: str) -> int:
    return 30 if period == "30d" else 7


def _date_range(period: str) -> tuple[datetime, datetime, int]:
    days = _period_days(period)
    end = datetime.now(timezone.utc).replace(hour=0, minute=0, second=0, microsecond=0) + timedelta(days=1)
    start = end - timedelta(days=days)
    return start, end, days


def _empty_days(start: datetime, days: int, factory) -> dict[str, dict[str, Any]]:
    return {
        (start + timedelta(days=i)).strftime("%Y-%m-%d"): factory()
        for i in range(days)
    }


def _conversation_filters(page_id: str | None = None):
    filters = []
    if page_id:
        filters.append(Conversation.page_id == page_id)
    return filters


def _customer_filters(page_id: str | None = None):
    filters = [Customer.gdpr_deleted == False]
    if page_id:
        filters.append(Customer.page_id == page_id)
    return filters


async def _roi_metrics(db: AsyncSession, page_id: str | None = None) -> dict[str, Any]:
    conv_filters = _conversation_filters(page_id)
    conv_result = await db.execute(
        select(
            func.count(Conversation.id).label("total"),
            func.coalesce(
                func.sum(
                    case(
                        (
                            Conversation.status.in_(["replied", "resolved"])
                            & Conversation.admin_reply.is_(None),
                            1,
                        ),
                        else_=0,
                    )
                ),
                0,
            ).label("auto_replied"),
            func.coalesce(func.avg(Conversation.processing_time), 0).label("avg_time"),
        ).where(*conv_filters)
    )
    conv_row = conv_result.one()

    cust_filters = _customer_filters(page_id)
    cust_result = await db.execute(
        select(
            func.coalesce(
                func.sum(case((Customer.purchase_intent == "High", 1), else_=0)),
                0,
            ).label("high_intent"),
            func.coalesce(
                func.sum(case((Customer.conversion_status == "converted", 1), else_=0)),
                0,
            ).label("converted"),
        ).where(*cust_filters)
    )
    cust_row = cust_result.one()

    total = int(conv_row.total or 0)
    auto_replied = int(conv_row.auto_replied or 0)
    avg_time = float(conv_row.avg_time or 0)
    high_intent = int(cust_row.high_intent or 0)
    converted = int(cust_row.converted or 0)

    human_time_per_reply_min = 5
    time_saved_min = auto_replied * (human_time_per_reply_min - avg_time / 60)
    cost_saved_usd = time_saved_min / 60 * 15

    return {
        "total_comments_processed": total,
        "auto_replied": auto_replied,
        "auto_reply_rate_pct": round(auto_replied / max(total, 1) * 100, 1),
        "avg_ai_response_time_sec": round(avg_time, 2),
        "estimated_time_saved_hours": round(max(0, time_saved_min / 60), 1),
        "estimated_cost_saved_usd": round(max(0, cost_saved_usd), 2),
        "high_intent_leads_generated": high_intent,
        "converted_customers": converted,
        "conversion_rate_pct": round(converted / max(high_intent, 1) * 100, 1),
    }


async def _conversion_funnel(db: AsyncSession, page_id: str | None = None) -> dict[str, Any]:
    total_comments = (
        await db.execute(select(func.count(Conversation.id)).where(*_conversation_filters(page_id)))
    ).scalar_one()

    status_rows = (
        await db.execute(
            select(Customer.conversion_status, func.count(Customer.id).label("count"))
            .where(*_customer_filters(page_id))
            .group_by(Customer.conversion_status)
        )
    ).all()
    stages = {row.conversion_status: int(row.count or 0) for row in status_rows}
    total_customers = sum(stages.values())
    prospects = stages.get("prospect", 0) + stages.get("warm", 0)

    return {
        "stages": [
            {"stage": "تعليقات واردة", "label": "Total Comments", "count": total_comments, "pct": 100},
            {
                "stage": "عملاء محددون",
                "label": "Identified Customers",
                "count": total_customers,
                "pct": round(total_customers / max(total_comments, 1) * 100, 1),
            },
            {
                "stage": "عملاء محتملون",
                "label": "Prospects",
                "count": prospects,
                "pct": round(prospects / max(total_customers, 1) * 100, 1),
            },
            {
                "stage": "عملاء ساخنون",
                "label": "Hot Leads",
                "count": stages.get("hot", 0),
                "pct": round(stages.get("hot", 0) / max(total_customers, 1) * 100, 1),
            },
            {
                "stage": "تحويلات",
                "label": "Converted",
                "count": stages.get("converted", 0),
                "pct": round(stages.get("converted", 0) / max(total_customers, 1) * 100, 1),
            },
        ]
    }


async def _ai_performance_trend(db: AsyncSession, period: str, page_id: str | None = None) -> list[dict[str, Any]]:
    start, end, days = _date_range(period)
    day_expr = func.date_trunc("day", Conversation.created_at).label("day")
    rows = (
        await db.execute(
            select(
                day_expr,
                func.count(Conversation.id).label("total"),
                func.coalesce(func.avg(Conversation.confidence_score), 0).label("avg_confidence"),
                func.coalesce(
                    func.sum(
                        case(
                            (
                                Conversation.status.in_(["replied", "resolved"])
                                & Conversation.admin_reply.is_(None),
                                1,
                            ),
                            else_=0,
                        )
                    ),
                    0,
                ).label("auto_replied"),
                func.coalesce(
                    func.sum(case((Conversation.status == "escalated", 1), else_=0)),
                    0,
                ).label("escalated"),
            )
            .where(Conversation.created_at >= start, Conversation.created_at < end, *_conversation_filters(page_id))
            .group_by(day_expr)
            .order_by(day_expr)
        )
    ).all()

    data = _empty_days(
        start,
        days,
        lambda: {"total": 0, "avg_confidence": 0, "auto_reply_rate": 0, "escalation_rate": 0},
    )

    for row in rows:
        key = row.day.strftime("%Y-%m-%d")
        total = int(row.total or 0)
        auto_replied = int(row.auto_replied or 0)
        escalated = int(row.escalated or 0)
        data[key] = {
            "total": total,
            "avg_confidence": round(float(row.avg_confidence or 0) * 100, 1),
            "auto_reply_rate": round(auto_replied / max(total, 1) * 100, 1),
            "escalation_rate": round(escalated / max(total, 1) * 100, 1),
        }

    return [{"date": date, **values} for date, values in data.items()]


async def _language_breakdown(db: AsyncSession, page_id: str | None = None) -> list[dict[str, Any]]:
    rows = (
        await db.execute(
            select(Conversation.language, func.count(Conversation.id).label("count"))
            .where(Conversation.language.isnot(None), *_conversation_filters(page_id))
            .group_by(Conversation.language)
            .order_by(func.count(Conversation.id).desc())
        )
    ).all()
    total = sum(int(row.count or 0) for row in rows)
    return [
        {
            "language": row.language or "unknown",
            "count": int(row.count or 0),
            "percentage": round(int(row.count or 0) / max(total, 1) * 100, 1),
        }
        for row in rows
    ]


async def _churn_risk_distribution(db: AsyncSession, page_id: str | None = None) -> dict[str, Any]:
    rows = (
        await db.execute(
            select(Customer.churn_risk, func.count(Customer.id).label("count"))
            .where(*_customer_filters(page_id))
            .group_by(Customer.churn_risk)
        )
    ).all()
    dist = {row.churn_risk: int(row.count or 0) for row in rows}
    total = sum(dist.values())

    high_risk = (
        await db.execute(
            select(Customer)
            .where(Customer.churn_risk == "high", *_customer_filters(page_id))
            .order_by(Customer.churn_risk_score.desc())
            .limit(5)
        )
    ).scalars().all()

    return {
        "total_customers": total,
        "distribution": {
            "low": dist.get("low", 0),
            "medium": dist.get("medium", 0),
            "high": dist.get("high", 0),
        },
        "high_risk_customers": [
            {
                "id": c.id,
                "name": c.full_name or c.username or "—",
                "churn_risk_score": c.churn_risk_score,
                "next_best_action": c.next_best_action,
                "last_interaction": c.last_interaction.isoformat() if c.last_interaction else None,
            }
            for c in high_risk
        ],
    }


async def _response_time_trend(db: AsyncSession, period: str, page_id: str | None = None) -> list[dict[str, Any]]:
    start, end, days = _date_range(period)
    day_expr = func.date_trunc("day", Conversation.created_at).label("day")
    rows = (
        await db.execute(
            select(
                day_expr,
                func.coalesce(func.avg(Conversation.processing_time), 0).label("avg_time"),
            )
            .where(
                Conversation.created_at >= start,
                Conversation.created_at < end,
                Conversation.processing_time.isnot(None),
                *_conversation_filters(page_id),
            )
            .group_by(day_expr)
            .order_by(day_expr)
        )
    ).all()

    data = _empty_days(start, days, lambda: {"avg_response_time_sec": 0})
    for row in rows:
        data[row.day.strftime("%Y-%m-%d")] = {
            "avg_response_time_sec": round(float(row.avg_time or 0), 2),
        }

    return [{"date": date, **values} for date, values in data.items()]


@router.get("/analytics/advanced-summary")
async def advanced_summary(
    period: str = Query("7d"),
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return {
        "roi": await _roi_metrics(db, page_id),
        "funnel": await _conversion_funnel(db, page_id),
        "performance": await _ai_performance_trend(db, period, page_id),
        "language_breakdown": await _language_breakdown(db, page_id),
        "churn_risk": await _churn_risk_distribution(db, page_id),
        "response_time_trend": await _response_time_trend(db, period, page_id),
    }


@router.get("/analytics/roi")
async def roi_metrics(
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _roi_metrics(db, page_id)


@router.get("/analytics/funnel")
async def conversion_funnel(
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _conversion_funnel(db, page_id)


@router.get("/analytics/performance")
async def ai_performance_trend(
    period: str = Query("7d"),
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _ai_performance_trend(db, period, page_id)


@router.get("/analytics/language-breakdown")
async def language_breakdown(
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _language_breakdown(db, page_id)


@router.get("/analytics/churn-risk")
async def churn_risk_distribution(
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _churn_risk_distribution(db, page_id)


@router.get("/analytics/response-time-trend")
async def response_time_trend(
    period: str = Query("7d"),
    page_id: str | None = Query(None),
    db: AsyncSession = Depends(get_db),
):
    return await _response_time_trend(db, period, page_id)
