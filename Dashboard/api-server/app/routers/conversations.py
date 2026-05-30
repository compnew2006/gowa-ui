from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.deps import get_db
from app.schemas import ConversationOut, ConversationListResponse, ManualReplyRequest
from app.services.conversations import (
    list_conversations as svc_list_conversations,
    get_conversation as svc_get_conversation,
    manual_reply as svc_manual_reply,
    resolve_conversation as svc_resolve_conversation,
    approve_reply as svc_approve_reply,
)

router = APIRouter(tags=["conversations"])


@router.get("/conversations", response_model=ConversationListResponse)
async def list_conversations(
    page: int = Query(1, ge=1),
    limit: int = Query(20, ge=1, le=100),
    status: str | None = None,
    page_id: str | None = None,
    search: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    data, total = await svc_list_conversations(
        db, page=page, limit=limit, status=status, page_id=page_id, search=search
    )
    return ConversationListResponse(data=data, total=total, page=page, limit=limit)


@router.get("/conversations/{conv_id}", response_model=ConversationOut)
async def get_conversation(conv_id: str, db: AsyncSession = Depends(get_db)):
    conv = await svc_get_conversation(db, conv_id)
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    return conv


@router.post("/conversations/{conv_id}/reply", response_model=ConversationOut)
async def manual_reply(
    conv_id: str,
    body: ManualReplyRequest,
    db: AsyncSession = Depends(get_db),
):
    conv = await svc_manual_reply(db, conv_id, body.reply)
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status == "escalated":
        raise HTTPException(status_code=400, detail=f"فشل النشر على فيسبوك: {conv.escalation_reason}")
    return conv




@router.post("/conversations/{conv_id}/approve", response_model=ConversationOut)
async def approve_reply(
    conv_id: str,
    body: ManualReplyRequest | None = None,
    db: AsyncSession = Depends(get_db),
):
    """Approve the AI reply and post it to Facebook. Optionally override with custom text."""
    reply_text = body.reply if body else None
    conv = await svc_approve_reply(db, conv_id, reply_text)
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    if conv.status == "escalated":
        raise HTTPException(status_code=400, detail=f"فشل النشر على فيسبوك: {conv.escalation_reason}")
    return conv


@router.post("/conversations/{conv_id}/resolve", response_model=ConversationOut)
async def resolve_conversation(conv_id: str, db: AsyncSession = Depends(get_db)):
    conv = await svc_resolve_conversation(db, conv_id)
    if not conv:
        raise HTTPException(status_code=404, detail="Conversation not found")
    return conv
