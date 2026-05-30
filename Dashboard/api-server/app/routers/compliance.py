"""
Compliance & Safety Router (GDPR + PII).

Endpoints:
  GET  /compliance/pii-scan/{conversation_id} — Scan a comment for PII
  POST /compliance/gdpr-export/{customer_id}  — Export all customer data
  POST /compliance/gdpr-delete/{customer_id}  — Right to erasure (soft delete + anonymize)
  GET  /compliance/data-retention             — Check retention policy stats
  GET  /compliance/audit-summary              — Audit log summary
"""
import uuid
import json
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func, update
from datetime import datetime, timezone, timedelta

from app.deps import get_db
from app.db import Customer, Conversation, Escalation, AuditLog

router = APIRouter(tags=["compliance"], prefix="/compliance")


@router.get("/pii-scan/{conversation_id}")
async def scan_conversation_pii(conversation_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Conversation).where(Conversation.id == conversation_id))
    conv = result.scalar_one_or_none()
    if not conv:
        raise HTTPException(404, "Conversation not found")

    from app.services.pii_detector import scan_text
    comment_scan = scan_text(conv.original_comment)
    reply_scan = scan_text(conv.ai_reply or "")

    return {
        "conversation_id": conversation_id,
        "comment_pii": {"detected": comment_scan.detected, "types": comment_scan.pii_types},
        "reply_pii": {"detected": reply_scan.detected, "types": reply_scan.pii_types},
        "masked_comment": comment_scan.masked_text if comment_scan.detected else None,
        "masked_reply": reply_scan.masked_text if reply_scan.detected else None,
    }


@router.post("/gdpr-export/{customer_id}")
async def gdpr_export(customer_id: str, db: AsyncSession = Depends(get_db)):
    """Export all data for a customer (GDPR right to access)."""
    result = await db.execute(select(Customer).where(Customer.id == customer_id))
    customer = result.scalar_one_or_none()
    if not customer:
        raise HTTPException(404, "Customer not found")

    conv_result = await db.execute(
        select(Conversation).where(Conversation.customer_id == customer_id)
        .order_by(Conversation.created_at.desc())
    )
    conversations = conv_result.scalars().all()

    esc_result = await db.execute(
        select(Escalation).where(Escalation.customer_id == customer_id)
    )
    escalations = esc_result.scalars().all()

    # Mark export requested
    customer.gdpr_export_requested_at = datetime.now(timezone.utc)

    # Log the action
    log = AuditLog(
        id=str(uuid.uuid4()),
        admin_name="gdpr_system",
        action="gdpr_export",
        entity_type="customer",
        entity_id=customer_id,
        details={"conversations_count": len(conversations)},
    )
    db.add(log)
    await db.commit()

    return {
        "customer": {
            "id": customer.id,
            "facebook_id": customer.facebook_id,
            "full_name": customer.full_name,
            "username": customer.username,
            "email": None,
            "first_contact": customer.first_contact_date.isoformat() if customer.first_contact_date else None,
            "last_interaction": customer.last_interaction.isoformat() if customer.last_interaction else None,
            "lead_score": customer.lead_score,
            "tags": customer.tags,
            "notes": customer.notes,
        },
        "conversations": [
            {
                "id": c.id,
                "date": c.created_at.isoformat(),
                "comment": c.original_comment,
                "ai_reply": c.ai_reply,
                "status": c.status,
            }
            for c in conversations
        ],
        "escalations": [
            {
                "id": e.id,
                "date": e.created_at.isoformat(),
                "reason": e.reason,
                "status": e.status,
            }
            for e in escalations
        ],
        "exported_at": datetime.now(timezone.utc).isoformat(),
    }


@router.post("/gdpr-delete/{customer_id}")
async def gdpr_delete(customer_id: str, db: AsyncSession = Depends(get_db)):
    """Right to erasure: anonymize customer data."""
    result = await db.execute(select(Customer).where(Customer.id == customer_id))
    customer = result.scalar_one_or_none()
    if not customer:
        raise HTTPException(404, "Customer not found")
    if customer.gdpr_deleted:
        return {"success": True, "message": "Already anonymized"}

    # Anonymize
    customer.full_name = "[Deleted]"
    customer.username = None
    customer.facebook_id = f"deleted_{customer_id[:8]}"
    customer.instagram_id = None
    customer.profile_url = None
    customer.avatar_url = None
    customer.notes = []
    customer.tags = []
    customer.gdpr_deleted = True

    log = AuditLog(
        id=str(uuid.uuid4()),
        admin_name="gdpr_system",
        action="gdpr_delete",
        entity_type="customer",
        entity_id=customer_id,
        details={"anonymized": True},
    )
    db.add(log)
    await db.commit()
    return {"success": True, "message": "Customer data anonymized (GDPR right to erasure)"}


@router.get("/data-retention")
async def data_retention_stats(db: AsyncSession = Depends(get_db)):
    """Check data retention policy compliance."""
    now = datetime.now(timezone.utc)
    cutoff_90 = now - timedelta(days=90)
    cutoff_365 = now - timedelta(days=365)

    old_resolved = (await db.execute(
        select(func.count(Conversation.id)).where(
            Conversation.created_at < cutoff_90,
            Conversation.status == "resolved",
        )
    )).scalar_one()

    very_old = (await db.execute(
        select(func.count(Conversation.id)).where(Conversation.created_at < cutoff_365)
    )).scalar_one()

    total_customers = (await db.execute(select(func.count(Customer.id)))).scalar_one()
    gdpr_deleted = (await db.execute(
        select(func.count(Customer.id)).where(Customer.gdpr_deleted == True)
    )).scalar_one()

    return {
        "retention_policy_days": 90,
        "resolved_older_than_90d": old_resolved,
        "all_conversations_older_than_365d": very_old,
        "total_customers": total_customers,
        "gdpr_deleted_customers": gdpr_deleted,
        "gdpr_deletion_rate": round(gdpr_deleted / max(total_customers, 1) * 100, 1),
        "recommendation": "Run cleanup task to archive resolved conversations older than 90 days" if old_resolved > 100 else "OK",
    }


@router.get("/audit-summary")
async def audit_summary(db: AsyncSession = Depends(get_db)):
    """High-level audit log summary."""
    result = await db.execute(
        select(AuditLog.action, func.count(AuditLog.id).label("count"))
        .group_by(AuditLog.action)
        .order_by(func.count(AuditLog.id).desc())
        .limit(20)
    )
    actions = result.all()
    total = (await db.execute(select(func.count(AuditLog.id)))).scalar_one()
    return {
        "total_audit_events": total,
        "by_action": [{"action": r.action, "count": r.count} for r in actions],
    }
