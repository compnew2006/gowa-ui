from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from app.deps import get_db
from app.db import Escalation, Conversation
from app.schemas import EscalationOut, EscalationListResponse, ResolveEscalationRequest

router = APIRouter(tags=["escalations"])


@router.get("/escalations", response_model=EscalationListResponse)
async def list_escalations(
    page: int = Query(1, ge=1),
    limit: int = Query(20, ge=1, le=100),
    page_id: str | None = None,
    status: str | None = None,
    priority: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    q = select(Escalation)
    if page_id:
        q = q.where(Escalation.page_id == page_id)
    if status:
        q = q.where(Escalation.status == status)
    if priority:
        q = q.where(Escalation.priority == priority)

    count_q = select(func.count()).select_from(q.subquery())
    total = (await db.execute(count_q)).scalar_one()

    q = q.order_by(Escalation.created_at.desc()).offset((page - 1) * limit).limit(limit)
    result = await db.execute(q)
    data = result.scalars().all()

    return EscalationListResponse(data=data, total=total, page=page, limit=limit)


@router.get("/escalations/{esc_id}", response_model=EscalationOut)
async def get_escalation(esc_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Escalation).where(Escalation.id == esc_id))
    esc = result.scalar_one_or_none()
    if not esc:
        raise HTTPException(status_code=404, detail="Escalation not found")
    return esc


@router.post("/escalations/{esc_id}/resolve", response_model=EscalationOut)
async def resolve_escalation(
    esc_id: str,
    body: ResolveEscalationRequest,
    db: AsyncSession = Depends(get_db),
):
    result = await db.execute(select(Escalation).where(Escalation.id == esc_id))
    esc = result.scalar_one_or_none()
    if not esc:
        raise HTTPException(status_code=404, detail="Escalation not found")

    esc.status = "resolved"
    esc.resolved_at = datetime.now(timezone.utc)
    esc.resolved_by = body.resolved_by
    if body.admin_notes:
        esc.admin_notes = body.admin_notes
    esc.updated_at = datetime.now(timezone.utc)

    conv_result = await db.execute(
        select(Conversation).where(Conversation.id == esc.conversation_id)
    )
    conv = conv_result.scalar_one_or_none()
    if conv:
        conv.status = "resolved"
        conv.updated_at = datetime.now(timezone.utc)

    await db.commit()
    await db.refresh(esc)
    return esc
