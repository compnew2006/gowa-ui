"""
CRM Lead Scoring Engine.

Auto-calculates lead_score (0-100) and purchase_intent based on:
- Intent classification (highest weight)
- Sentiment history
- Interaction count
- Escalation history
- Conversion signals
"""
from __future__ import annotations


INTENT_SCORES = {
    "purchase": 40,
    "price_inquiry": 25,
    "complaint": -5,
    "refund": -10,
    "compliment": 10,
    "general": 2,
}

SENTIMENT_SCORES = {
    "positive": 8,
    "neutral": 2,
    "negative": -5,
    "angry": -15,
}

CONVERSION_STATUS_THRESHOLDS = {
    (0, 20): "cold",
    (20, 40): "prospect",
    (40, 60): "warm",
    (60, 80): "hot",
    (80, 101): "converted",
}


def calculate_lead_score(
    current_score: int,
    intent: str | None,
    sentiment: str | None,
    interaction_count: int,
    escalation_count: int,
) -> tuple[int, str, str]:
    """
    Calculate updated lead_score, purchase_intent, conversion_status.
    Returns (score: 0-100, purchase_intent: Low|Medium|High, conversion_status).
    """
    score = current_score

    # Intent signal (primary driver)
    score += INTENT_SCORES.get(intent or "general", 0)

    # Sentiment signal
    score += SENTIMENT_SCORES.get(sentiment or "neutral", 0)

    # Interaction depth bonus
    if interaction_count >= 10:
        score += 5
    elif interaction_count >= 5:
        score += 3
    elif interaction_count >= 2:
        score += 1

    # Escalation penalty (sign of frustration)
    score -= escalation_count * 3

    # Clamp to 0-100
    score = max(0, min(100, score))

    # Determine purchase_intent
    if score >= 60 or intent == "purchase":
        purchase_intent = "High"
    elif score >= 35 or intent == "price_inquiry":
        purchase_intent = "Medium"
    else:
        purchase_intent = "Low"

    # Determine conversion_status
    conversion_status = "prospect"
    for (low, high), status in CONVERSION_STATUS_THRESHOLDS.items():
        if low <= score < high:
            conversion_status = status
            break

    return score, purchase_intent, conversion_status


def update_customer_crm(
    customer,
    intent: str | None,
    sentiment: str | None,
) -> None:
    """
    In-place update of a Customer ORM object with new CRM scores.
    Call before db.commit().
    """
    escalation_count = len(customer.escalation_history or [])
    score, purchase_intent, conversion_status = calculate_lead_score(
        current_score=customer.lead_score or 0,
        intent=intent,
        sentiment=sentiment,
        interaction_count=customer.interaction_count or 0,
        escalation_count=escalation_count,
    )
    customer.lead_score = score
    customer.purchase_intent = purchase_intent
    customer.conversion_status = conversion_status
