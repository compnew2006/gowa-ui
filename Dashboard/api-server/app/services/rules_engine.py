"""
Automation Rules Engine.

Evaluates custom automation rules against pipeline results.
Rules are ordered by priority (lower number = higher priority).

Supported condition fields:
  intent, sentiment, confidence, language, keyword, customer_churn_risk,
  platform, customer_lead_score, urgency, priority, guardrail_triggered

Supported operators:
  eq, neq, in, not_in, gt, gte, lt, lte, contains

Supported actions:
  escalate         -> force escalation with custom reason
  tag              -> add tag to customer
  assign           -> assign escalation to specific admin
  notify           -> extra admin notification
  skip             -> suppress AI reply (add to DLQ)
  custom_reply     -> override AI reply with template
"""
from __future__ import annotations
import logging
from typing import Any, Optional

logger = logging.getLogger(__name__)

# Supported condition fields for validation
VALID_CONDITION_FIELDS = {
    "intent", "sentiment", "confidence", "language", "keyword",
    "customer_churn_risk", "platform", "customer_lead_score",
    "urgency", "priority", "guardrail_triggered",
}

VALID_OPERATORS = {
    "eq", "neq", "in", "not_in", "gt", "gte", "lt", "lte", "contains",
}

VALID_ACTIONS = {
    "escalate", "tag", "assign", "notify", "skip", "custom_reply",
}


def _eval_condition(condition: dict, context: dict) -> bool:
    """Evaluate a single condition against the pipeline context."""
    field = condition.get("field", "")
    op = condition.get("op", "eq")
    value = condition.get("value")

    # For the "keyword" condition type, check against the comment text
    if field == "keyword":
        comment_text = context.get("comment_text", "")
        if not comment_text:
            return False
        if op == "contains":
            return str(value).lower() in comment_text.lower()
        elif op == "eq":
            return str(value).lower() == comment_text.lower()
        return False

    actual = context.get(field)

    if actual is None:
        return False

    if op == "eq":
        return str(actual).lower() == str(value).lower()
    elif op == "neq":
        return str(actual).lower() != str(value).lower()
    elif op == "in":
        return str(actual).lower() in [str(v).lower() for v in (value or [])]
    elif op == "not_in":
        return str(actual).lower() not in [str(v).lower() for v in (value or [])]
    elif op == "gt":
        try:
            return float(actual) > float(value)
        except (TypeError, ValueError):
            return False
    elif op == "gte":
        try:
            return float(actual) >= float(value)
        except (TypeError, ValueError):
            return False
    elif op == "lt":
        try:
            return float(actual) < float(value)
        except (TypeError, ValueError):
            return False
    elif op == "lte":
        try:
            return float(actual) <= float(value)
        except (TypeError, ValueError):
            return False
    elif op == "contains":
        return str(value).lower() in str(actual).lower()
    return False


def evaluate_rules(rules: list, context: dict) -> Optional[dict]:
    """
    Evaluate rules against a pipeline context dict.
    Returns the first matching rule's action config, or None.

    context keys: intent, sentiment, confidence, language, platform,
                  customer_lead_score, urgency, priority, guardrail_triggered,
                  customer_churn_risk, comment_text
    """
    sorted_rules = sorted(rules, key=lambda r: r.get("priority", 10))

    for rule in sorted_rules:
        if not rule.get("is_active", True):
            continue

        conditions = rule.get("conditions", [])
        logic = rule.get("condition_logic", "AND")

        if not conditions:
            continue

        results = [_eval_condition(c, context) for c in conditions]

        matched = all(results) if logic == "AND" else any(results)

        if matched:
            logger.info(
                "[RulesEngine] Rule matched: '%s' (id=%s) -> action: %s",
                rule.get("name"), rule.get("id"), rule.get("action"),
            )
            return {
                "rule_id": rule.get("id"),
                "rule_name": rule.get("name"),
                "action": rule.get("action"),
                "action_config": rule.get("action_config", {}),
            }

    return None


async def evaluate_rules_from_db(
    page_id: str,
    context: dict,
    db: Any = None,
) -> Optional[dict]:
    """
    Load active rules from the database for the given page and evaluate them.
    Returns the first matching rule's action config, or None.
    """
    if db is None:
        return None

    try:
        from sqlalchemy import select, or_
        from app.db import AutomationRule

        q = select(AutomationRule).where(AutomationRule.is_active == True)
        if page_id:
            q = q.where(
                or_(
                    AutomationRule.page_id == page_id,
                    AutomationRule.page_id.is_(None),
                )
            )
        q = q.order_by(AutomationRule.priority, AutomationRule.created_at)

        result = await db.execute(q)
        rules = result.scalars().all()

        if not rules:
            return None

        serialized = [
            {
                "id": r.id,
                "name": r.name,
                "conditions": r.conditions or [],
                "condition_logic": r.condition_logic or "AND",
                "action": r.action,
                "action_config": r.action_config or {},
                "priority": r.priority,
                "is_active": r.is_active,
            }
            for r in rules
        ]

        match = evaluate_rules(serialized, context)
        if match:
            # Increment trigger count on the matched rule
            for r in rules:
                if r.id == match["rule_id"]:
                    r.trigger_count = (r.trigger_count or 0) + 1
                    from datetime import datetime, timezone
                    r.last_triggered_at = datetime.now(timezone.utc)
                    await db.flush()
                    break
        return match

    except Exception as exc:
        logger.warning("[RulesEngine] DB evaluation failed: %s", exc)
        return None


async def apply_rule_to_pipeline(rule_match: dict, ai_result: dict, customer=None) -> dict:
    """
    Apply a matched rule to the AI pipeline result.
    Modifies ai_result in-place and returns it.
    """
    if not rule_match:
        return ai_result

    action = rule_match.get("action")
    config = rule_match.get("action_config", {})

    if action == "escalate":
        ai_result["action"] = "escalate"
        ai_result["escalation_reason"] = config.get("reason", f"Rule: {rule_match.get('rule_name')}")
        ai_result["priority"] = config.get("priority", "high")

    elif action == "custom_reply":
        template = config.get("template", "")
        if template:
            ai_result["ai_reply"] = template
            ai_result["action"] = "reply"

    elif action == "tag" and customer:
        new_tag = config.get("tag", "")
        if new_tag and new_tag not in (customer.tags or []):
            customer.tags = list(customer.tags or []) + [new_tag]

    elif action == "assign":
        ai_result["assigned_to"] = config.get("admin_email", "")

    elif action == "notify":
        try:
            from app.services.notifications import notify_admin
            await notify_admin(
                title=f"[Rule: {rule_match.get('rule_name', 'Unknown')}] Notification",
                message=config.get("message", "Automation rule triggered"),
                priority=config.get("priority", "medium"),
            )
        except Exception:
            pass

    elif action == "skip":
        ai_result["action"] = "skip"
        ai_result["ai_reply"] = None

    ai_result["matched_rule_id"] = rule_match.get("rule_id")
    return ai_result
