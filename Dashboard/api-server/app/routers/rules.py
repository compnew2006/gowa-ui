"""
Automation Rules Engine Router.
Build custom workflow rules: if X then Y.
"""
import uuid
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from datetime import datetime, timezone

from app.deps import get_db
from app.db import AutomationRule
from app.services.rules_engine import evaluate_rules

router = APIRouter(tags=["rules"], prefix="/rules")


@router.get("")
async def list_rules(page_id: str | None = None, db: AsyncSession = Depends(get_db)):
    q = select(AutomationRule)
    if page_id:
        q = q.where(AutomationRule.page_id == page_id)
    q = q.order_by(AutomationRule.priority, AutomationRule.created_at)
    result = await db.execute(q)
    rules = result.scalars().all()
    return [_serialize(r) for r in rules]


@router.post("")
async def create_rule(body: dict, db: AsyncSession = Depends(get_db)):
    rule = AutomationRule(
        id=str(uuid.uuid4()),
        page_id=body.get("page_id"),
        name=body.get("name", "Unnamed Rule"),
        description=body.get("description"),
        conditions=body.get("conditions", []),
        condition_logic=body.get("condition_logic", "AND"),
        action=body.get("action", "escalate"),
        action_config=body.get("action_config", {}),
        priority=body.get("priority", 10),
        is_active=body.get("is_active", True),
    )
    db.add(rule)
    await db.commit()
    return _serialize(rule)


@router.patch("/{rule_id}")
async def update_rule(rule_id: str, body: dict, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(AutomationRule).where(AutomationRule.id == rule_id))
    rule = result.scalar_one_or_none()
    if not rule:
        raise HTTPException(404, "Rule not found")
    for field in ("name", "description", "conditions", "condition_logic", "action", "action_config", "priority", "is_active"):
        if field in body:
            setattr(rule, field, body[field])
    rule.updated_at = datetime.now(timezone.utc)
    await db.commit()
    return _serialize(rule)


@router.delete("/{rule_id}")
async def delete_rule(rule_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(AutomationRule).where(AutomationRule.id == rule_id))
    rule = result.scalar_one_or_none()
    if not rule:
        raise HTTPException(404, "Rule not found")
    await db.delete(rule)
    await db.commit()
    return {"success": True}


@router.post("/{rule_id}/test")
async def test_rule(rule_id: str, context: dict, db: AsyncSession = Depends(get_db)):
    """Test a rule against a sample pipeline context without saving anything."""
    result = await db.execute(select(AutomationRule).where(AutomationRule.id == rule_id))
    rule = result.scalar_one_or_none()
    if not rule:
        raise HTTPException(404, "Rule not found")
    match = evaluate_rules([_serialize(rule)], context)
    return {
        "matched": match is not None,
        "action": match.get("action") if match else None,
        "context_used": context,
    }


@router.post("/evaluate")
async def evaluate_all_rules(context: dict, db: AsyncSession = Depends(get_db)):
    """Evaluate all active rules against a context. Returns first match."""
    page_id = context.get("page_id")
    q = select(AutomationRule).where(AutomationRule.is_active == True)
    if page_id:
        q = q.where(AutomationRule.page_id == page_id)
    q = q.order_by(AutomationRule.priority)
    result = await db.execute(q)
    rules = result.scalars().all()
    match = evaluate_rules([_serialize(r) for r in rules], context)
    return {"matched": match is not None, "match": match}


def _serialize(r):
    return {
        "id": r.id, "name": r.name, "description": r.description,
        "conditions": r.conditions, "condition_logic": r.condition_logic,
        "action": r.action, "action_config": r.action_config,
        "priority": r.priority, "is_active": r.is_active,
        "trigger_count": r.trigger_count,
        "last_triggered_at": r.last_triggered_at.isoformat() if r.last_triggered_at else None,
        "created_at": r.created_at.isoformat(),
    }
