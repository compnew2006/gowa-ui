"""
Predictive CRM Engine.

Features:
  - Churn Risk scoring (0-1) with 3 levels: low|medium|high
  - Next-Best-Action recommendation based on lead profile
  - Re-engagement detection
"""
from __future__ import annotations
from datetime import datetime, timezone, timedelta
from typing import Optional

CHURN_SIGNALS = {
    "days_since_last_interaction": {30: 0.2, 60: 0.4, 90: 0.6, 120: 0.8},
    "sentiment_angry_weight": 0.25,
    "escalation_weight": 0.1,
    "no_purchase_weight": 0.15,
}

NEXT_BEST_ACTIONS = {
    "converted": "اطلب تقييماً أو شهادة من هذا العميل",
    "hot": "تواصل معه بعرض حصري لإتمام الشراء",
    "warm": "أرسل له محتوى مفيد عن المنتج المناسب لاحتياجاته",
    "prospect": "ابدأ محادثة ودية واعرض عليه المساعدة",
    "cold": "أرسل له رسالة إعادة تفاعل مع عرض خاص",
}


def calculate_churn_risk(
    last_interaction: Optional[datetime],
    interaction_count: int,
    purchase_intent: str,
    conversion_status: str,
    escalation_count: int,
    sentiment_history: list[str],
) -> tuple[float, str]:
    """
    Returns (churn_risk_score: 0.0-1.0, churn_risk_level: low|medium|high).
    """
    score = 0.0
    now = datetime.now(timezone.utc)

    # Days since last interaction
    if last_interaction:
        days = (now - last_interaction).days
        for threshold, weight in sorted(CHURN_SIGNALS["days_since_last_interaction"].items()):
            if days >= threshold:
                score = max(score, weight)

    # Escalation history
    score += min(0.3, escalation_count * CHURN_SIGNALS["escalation_weight"])

    # Recent angry sentiments
    if sentiment_history:
        recent = sentiment_history[-5:]
        angry_ratio = sum(1 for s in recent if s == "angry") / len(recent)
        score += angry_ratio * CHURN_SIGNALS["sentiment_angry_weight"]

    # No purchase despite interactions
    if purchase_intent == "Low" and interaction_count >= 5:
        score += CHURN_SIGNALS["no_purchase_weight"]

    # Already converted = very low churn
    if conversion_status == "converted":
        score *= 0.3

    score = min(1.0, score)

    level = "high" if score >= 0.6 else "medium" if score >= 0.3 else "low"
    return round(score, 3), level


def get_next_best_action(
    conversion_status: str,
    churn_risk: str,
    purchase_intent: str,
    days_since_last_interaction: int,
) -> str:
    """
    Recommend the next best action for a customer.
    """
    # High churn risk override
    if churn_risk == "high":
        return "هذا العميل في خطر مغادرة — تواصل معه فوراً بعرض استثنائي"

    if days_since_last_interaction > 60:
        return "لم يتفاعل منذ فترة طويلة — أرسل رسالة إعادة تفاعل مع حافز"

    return NEXT_BEST_ACTIONS.get(conversion_status, "تابع هذا العميل بمحادثة شخصية")


def update_customer_predictions(customer) -> None:
    """
    In-place update of customer predictive fields.
    Call before db.commit().
    """
    last_interaction = customer.last_interaction
    days = 0
    if last_interaction:
        days = (datetime.now(timezone.utc) - last_interaction).days

    churn_score, churn_level = calculate_churn_risk(
        last_interaction=last_interaction,
        interaction_count=customer.interaction_count or 0,
        purchase_intent=customer.purchase_intent or "Low",
        conversion_status=customer.conversion_status or "prospect",
        escalation_count=len(customer.escalation_history or []),
        sentiment_history=customer.sentiment_history if hasattr(customer, "sentiment_history") else [],
    )

    nba = get_next_best_action(
        conversion_status=customer.conversion_status or "prospect",
        churn_risk=churn_level,
        purchase_intent=customer.purchase_intent or "Low",
        days_since_last_interaction=days,
    )

    customer.churn_risk = churn_level
    customer.churn_risk_score = churn_score
    customer.next_best_action = nba
