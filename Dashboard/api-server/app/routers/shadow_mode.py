"""
Shadow Mode Router — PRD §4.17

Shadow Mode: The system generates AI replies but never sends them.
Admin reviews generated replies, approves or rejects, before going Live.

Endpoints:
  GET  /shadow-mode/queue          — List pending shadow reviews
  POST /shadow-mode/{conv_id}/approve   — Approve reply (mark as replied)
  POST /shadow-mode/{conv_id}/reject    — Reject reply (escalate or flag)
  GET  /shadow-mode/stats          — Shadow mode accuracy metrics
  POST /shadow-mode/go-live/{page_id}  — Disable shadow mode for a page (after >80% approval)
"""
from fastapi import APIRouter, Depends, HTTPException, Body
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from typing import Optional
import logging

from app.deps import get_db
from app.db import Conversation, Page
from app.workers.tasks import _acquire_shadow_lock

logger = logging.getLogger(__name__)
router = APIRouter(tags=["shadow-mode"], prefix="/shadow-mode")


@router.get("/queue")
async def get_shadow_queue(
    page_id: Optional[str] = None,
    limit: int = 50,
    db: AsyncSession = Depends(get_db),
):
    """Get pending shadow mode conversations awaiting admin review."""
    query = (
        select(Conversation)
        .where(Conversation.status == "shadow_pending")
        .order_by(Conversation.created_at.desc())
        .limit(limit)
    )
    if page_id:
        query = query.where(Conversation.page_id == page_id)

    result = await db.execute(query)
    convs = result.scalars().all()
    return {
        "total": len(convs),
        "items": [
            {
                "id": c.id,
                "customer_name": c.customer_name,
                "original_comment": c.original_comment,
                "ai_reply": c.ai_reply,
                "intent": c.intent,
                "sentiment": c.sentiment,
                "confidence_score": c.confidence_score,
                "page_name": c.page_name,
                "platform": c.platform,
                "created_at": c.created_at.isoformat() if c.created_at else None,
            }
            for c in convs
        ],
    }


@router.post("/{conv_id}/approve")
async def approve_shadow_reply(
    conv_id: str,
    admin_note: Optional[str] = Body(None, embed=True),
    db: AsyncSession = Depends(get_db),
):
    """
    Approve a shadow reply. Marks it as 'replied' and records positive feedback
    for DSPy learning.
    """
    result = await db.execute(
        select(Conversation).where(Conversation.id == conv_id)
    )
    conv = result.scalar_one_or_none()
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status != "shadow_pending":
        raise HTTPException(status_code=400, detail="Not in shadow_pending status")

    # Concurrency protection: acquire lock to prevent double-approve
    if not _acquire_shadow_lock(conv_id):
        raise HTTPException(status_code=409, detail="This conversation is currently being reviewed by another admin. Please try again.")

    conv.status = "replied"
    conv.replied_at = datetime.now(timezone.utc)
    if admin_note:
        conv.admin_reply = admin_note
    feedback_id = str(conv_id)
    await db.commit()

    # Log audit trail
    try:
        from app.routers.audit_logs import log_action
        await log_action(
            db=db,
            action="approve",
            entity_type="conversation",
            entity_id=conv_id,
            page_id=str(conv.page_id),
            new_values={"status": "replied", "admin_reply": admin_note},
            details=f"Approved AI reply for customer: {conv.customer_name}",
        )
        await db.commit()
    except Exception:
        pass

    # Record positive DSPy feedback
    try:
        from app.ai.dspy_optimizer import get_optimizer
        optimizer = get_optimizer()
        await optimizer.record_feedback(
            comment=conv.original_comment,
            predicted_intent=conv.intent or "general",
            actual_intent=conv.intent,
            predicted_sentiment=conv.sentiment or "neutral",
            actual_sentiment=conv.sentiment,
            ai_reply=conv.ai_reply,
            was_helpful=True,
            feedback_id=feedback_id,
        )
    except Exception:
        pass

    return {"success": True, "status": "replied", "feedback_id": feedback_id}


@router.post("/{conv_id}/reject")
async def reject_shadow_reply(
    conv_id: str,
    reason: Optional[str] = Body(None, embed=True),
    correct_intent: Optional[str] = Body(None, embed=True),
    correct_sentiment: Optional[str] = Body(None, embed=True),
    db: AsyncSession = Depends(get_db),
):
    """
    Reject a shadow reply. Escalates for human handling and records negative
    feedback for DSPy improvement.
    """
    result = await db.execute(
        select(Conversation).where(Conversation.id == conv_id)
    )
    conv = result.scalar_one_or_none()
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status != "shadow_pending":
        raise HTTPException(status_code=400, detail="Not in shadow_pending status")

    # Concurrency protection: acquire lock to prevent double-reject
    if not _acquire_shadow_lock(conv_id):
        raise HTTPException(status_code=409, detail="This conversation is currently being reviewed by another admin. Please try again.")

    conv.status = "escalated"
    conv.escalation_reason = reason or "Admin rejected shadow reply"
    feedback_id = str(conv_id)
    await db.commit()

    # Log audit trail
    try:
        from app.routers.audit_logs import log_action
        await log_action(
            db=db,
            action="reject",
            entity_type="conversation",
            entity_id=conv_id,
            page_id=str(conv.page_id),
            old_values={"status": "shadow_pending", "intent": conv.intent, "sentiment": conv.sentiment},
            new_values={"status": "escalated", "intent": correct_intent, "sentiment": correct_sentiment},
            reason=reason,
            details=f"Rejected AI reply for customer: {conv.customer_name}",
        )
        await db.commit()
    except Exception:
        pass

    # Record corrective DSPy feedback (very valuable for learning)
    try:
        from app.ai.dspy_optimizer import get_optimizer
        optimizer = get_optimizer()
        await optimizer.record_feedback(
            comment=conv.original_comment,
            predicted_intent=conv.intent or "general",
            actual_intent=correct_intent or conv.intent,
            predicted_sentiment=conv.sentiment or "neutral",
            actual_sentiment=correct_sentiment or conv.sentiment,
            ai_reply=conv.ai_reply,
            was_helpful=False,
            feedback_id=feedback_id,
        )
    except Exception:
        pass

    return {"success": True, "status": "escalated", "reason": conv.escalation_reason, "feedback_id": feedback_id}


@router.post("/{conv_id}/undo")
async def undo_decision(
    conv_id: str,
    db: AsyncSession = Depends(get_db),
):
    """
    Undo an approval or rejection. Reverts to shadow_pending status
    and removes feedback from DSPy.
    """
    result = await db.execute(
        select(Conversation).where(Conversation.id == conv_id)
    )
    conv = result.scalar_one_or_none()
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status not in ["replied", "escalated"]:
        raise HTTPException(status_code=400, detail="Can only undo approved or rejected conversations")

    old_status = conv.status
    conv.status = "shadow_pending"
    conv.replied_at = None
    conv.escalation_reason = None
    await db.commit()

    # Log audit trail
    try:
        from app.routers.audit_logs import log_action
        await log_action(
            db=db,
            action="undo",
            entity_type="conversation",
            entity_id=conv_id,
            page_id=str(conv.page_id),
            old_values={"status": old_status},
            new_values={"status": "shadow_pending"},
            details=f"Undid decision for customer: {conv.customer_name}",
        )
        await db.commit()
    except Exception:
        pass

    # Remove feedback from DSPy
    try:
        from app.ai.dspy_optimizer import get_optimizer
        optimizer = get_optimizer()
        await optimizer.remove_feedback(str(conv_id))
    except Exception:
        pass

    return {"success": True, "status": "shadow_pending", "message": "تم الرجوع عن القرار"}


@router.patch("/{conv_id}/correct")
async def correct_feedback(
    conv_id: str,
    correct_intent: Optional[str] = Body(None, embed=True),
    correct_sentiment: Optional[str] = Body(None, embed=True),
    db: AsyncSession = Depends(get_db),
):
    """
    Correct a previous approval/rejection. Updates the feedback
    for DSPy learning with the corrected values.
    """
    result = await db.execute(
        select(Conversation).where(Conversation.id == conv_id)
    )
    conv = result.scalar_one_or_none()
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status not in ["replied", "escalated"]:
        raise HTTPException(status_code=400, detail="Can only correct approved or rejected conversations")

    was_helpful = conv.status == "replied"

    # Log audit trail
    try:
        from app.routers.audit_logs import log_action
        await log_action(
            db=db,
            action="correct",
            entity_type="conversation",
            entity_id=conv_id,
            page_id=str(conv.page_id),
            old_values={"intent": conv.intent, "sentiment": conv.sentiment},
            new_values={"intent": correct_intent, "sentiment": correct_sentiment},
            details=f"Corrected feedback for customer: {conv.customer_name}",
        )
        await db.commit()
    except Exception:
        pass

    # Update feedback in DSPy
    try:
        from app.ai.dspy_optimizer import get_optimizer
        optimizer = get_optimizer()
        await optimizer.record_feedback(
            comment=conv.original_comment,
            predicted_intent=conv.intent or "general",
            actual_intent=correct_intent or conv.intent,
            predicted_sentiment=conv.sentiment or "neutral",
            actual_sentiment=correct_sentiment or conv.sentiment,
            ai_reply=conv.ai_reply,
            was_helpful=was_helpful,
            feedback_id=str(conv_id),  # Update existing feedback
        )
    except Exception:
        pass

    return {
        "success": True,
        "message": "تم تصحيح التقييم",
        "correct_intent": correct_intent or conv.intent,
        "correct_sentiment": correct_sentiment or conv.sentiment,
    }


@router.get("/stats")
async def get_shadow_stats(
    page_id: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    """
    Calculate shadow mode accuracy metrics.
    Approval rate >80% is the threshold to go Live (PRD §4.17).
    """
    base = select(Conversation).where(Conversation.is_shadow_mode == True)
    if page_id:
        base = base.where(Conversation.page_id == page_id)

    result = await db.execute(base)
    all_convs = result.scalars().all()

    total = len(all_convs)
    approved = sum(1 for c in all_convs if c.status == "replied")
    rejected = sum(1 for c in all_convs if c.status == "escalated")
    pending = sum(1 for c in all_convs if c.status == "shadow_pending")

    approval_rate = approved / max(total - pending, 1)
    ready_for_live = approval_rate >= 0.80 and total >= 20

    return {
        "total_shadow": total,
        "approved": approved,
        "rejected": rejected,
        "pending_review": pending,
        "approval_rate": round(approval_rate * 100, 1),
        "ready_for_live": ready_for_live,
        "threshold": "80% approval required to go live",
    }


@router.post("/go-live/{page_id}")
async def go_live(
    page_id: str,
    force: bool = False,
    db: AsyncSession = Depends(get_db),
):
    """
    Disable shadow mode for a page. Requires >80% approval rate unless forced.
    """
    # Check approval rate first
    if not force:
        stats = await get_shadow_stats(page_id=page_id, db=db)
        if not stats["ready_for_live"]:
            raise HTTPException(
                status_code=400,
                detail=(
                    f"Not ready for live. Approval rate: {stats['approval_rate']}% "
                    f"(need ≥80% with ≥20 samples). Use force=true to override."
                ),
            )

    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")

    page.shadow_mode = False
    page.auto_reply_enabled = True
    await db.commit()

    logger.info("[ShadowMode] Page '%s' went LIVE", page.name)
    return {
        "success": True,
        "page": page.name,
        "shadow_mode": False,
        "auto_reply_enabled": True,
        "message": f"Page '{page.name}' is now LIVE — auto-replies enabled!",
    }
