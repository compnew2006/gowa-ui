"""
Bulk Operations Router.

- Bulk conversation actions (mark resolved, assign, tag, export)
- Knowledge base CSV import
- Bulk customer re-engagement
"""
import uuid
import csv
import io
from fastapi import APIRouter, Depends, HTTPException, UploadFile, File, Body
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update
from datetime import datetime, timezone
from typing import List, Optional

from app.deps import get_db
from app.db import Conversation, Customer, KnowledgeBase

router = APIRouter(tags=["bulk"], prefix="/bulk")


@router.post("/conversations/action")
async def bulk_conversation_action(
    ids: List[str] = Body(...),
    action: str = Body(...),
    value: Optional[str] = Body(None),
    db: AsyncSession = Depends(get_db),
):
    """
    Apply action to multiple conversations.
    Actions: resolve | escalate | tag | assign | delete
    """
    if action not in ("resolve", "escalate", "tag", "assign"):
        raise HTTPException(400, f"Unsupported action: {action}")
    if not ids:
        raise HTTPException(400, "No conversation IDs provided")

    result = await db.execute(select(Conversation).where(Conversation.id.in_(ids)))
    convs = result.scalars().all()

    updated = 0
    for conv in convs:
        if action == "resolve":
            conv.status = "resolved"
        elif action == "escalate":
            conv.status = "escalated"
            conv.escalation_reason = value or "Bulk escalation"
        conv.updated_at = datetime.now(timezone.utc)
        updated += 1

    await db.commit()
    return {"success": True, "updated": updated, "action": action}


@router.post("/knowledge-base/import-csv")
async def import_kb_csv(
    file: UploadFile = File(...),
    language: str = "ar",
    db: AsyncSession = Depends(get_db),
):
    """
    Import KB entries from CSV.
    Expected columns: category, question, answer, intent_tags (comma-separated)
    """
    if not file.filename.endswith(".csv"):
        raise HTTPException(400, "Only CSV files supported")

    content = await file.read()
    text = content.decode("utf-8-sig", errors="replace")
    reader = csv.DictReader(io.StringIO(text))

    created = 0
    errors = []

    for i, row in enumerate(reader):
        try:
            q = row.get("question", "").strip()
            a = row.get("answer", "").strip()
            cat = row.get("category", "general").strip()
            if not q or not a:
                errors.append(f"Row {i + 2}: missing question or answer")
                continue
            tags_raw = row.get("intent_tags", "")
            tags = [t.strip() for t in tags_raw.split(",") if t.strip()]
            entry = KnowledgeBase(
                id=str(uuid.uuid4()),
                category=cat,
                question=q,
                answer=a,
                intent_tags=tags,
                language=row.get("language", language),
                is_active=True,
            )
            db.add(entry)
            created += 1
        except Exception as e:
            errors.append(f"Row {i + 2}: {str(e)}")

    await db.commit()
    return {"created": created, "errors": errors[:20]}


@router.post("/customers/re-engage")
async def bulk_re_engage(
    conversion_status: Optional[str] = Body(None),
    churn_risk: Optional[str] = Body(None),
    purchase_intent: Optional[str] = Body(None),
    limit: int = Body(50),
    db: AsyncSession = Depends(get_db),
):
    """
    Flag customers for re-engagement campaign based on filters.
    Returns list of customers to contact.
    """
    q = select(Customer).where(Customer.gdpr_deleted == False)
    if conversion_status:
        q = q.where(Customer.conversion_status == conversion_status)
    if churn_risk:
        q = q.where(Customer.churn_risk == churn_risk)
    if purchase_intent:
        q = q.where(Customer.purchase_intent == purchase_intent)
    q = q.order_by(Customer.lead_score.desc()).limit(limit)

    result = await db.execute(q)
    customers = result.scalars().all()

    # Mark as re-engage pending
    now = datetime.now(timezone.utc)
    for c in customers:
        c.re_engage_sent_at = now

    await db.commit()

    return {
        "total": len(customers),
        "customers": [
            {
                "id": c.id, "name": c.full_name or c.username or "—",
                "facebook_id": c.facebook_id, "lead_score": c.lead_score,
                "churn_risk": c.churn_risk, "next_best_action": c.next_best_action,
                "last_interaction": c.last_interaction.isoformat() if c.last_interaction else None,
            }
            for c in customers
        ],
    }


@router.post("/customers/tag")
async def bulk_tag_customers(
    ids: List[str] = Body(...),
    tag: str = Body(...),
    db: AsyncSession = Depends(get_db),
):
    result = await db.execute(select(Customer).where(Customer.id.in_(ids)))
    customers = result.scalars().all()
    for c in customers:
        if tag not in (c.tags or []):
            c.tags = list(c.tags or []) + [tag]
        c.updated_at = datetime.now(timezone.utc)
    await db.commit()
    return {"success": True, "updated": len(customers), "tag": tag}
