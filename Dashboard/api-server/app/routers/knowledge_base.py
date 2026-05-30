import uuid
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from typing import Optional
from app.deps import get_db
from app.db import KnowledgeBase
from app.schemas import KnowledgeBaseOut, KnowledgeBaseCreate, KnowledgeBaseUpdate
from app.services.cache import delete_pattern

router = APIRouter(tags=["knowledge_base"])


@router.get("/knowledge-base", response_model=list[KnowledgeBaseOut])
async def list_kb(
    page_id: Optional[str] = Query(None),
    category: Optional[str] = None,
    language: Optional[str] = None,
    search: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    q = select(KnowledgeBase)
    if page_id:
        q = q.where(KnowledgeBase.page_id == page_id)
    if category:
        q = q.where(KnowledgeBase.category == category)
    if language:
        q = q.where(KnowledgeBase.language == language)
    if search:
        q = q.where(
            KnowledgeBase.question.ilike(f"%{search}%")
            | KnowledgeBase.answer.ilike(f"%{search}%")
        )
    q = q.order_by(KnowledgeBase.category, KnowledgeBase.created_at.desc())
    result = await db.execute(q)
    return result.scalars().all()


@router.get("/knowledge-base/{kb_id}", response_model=KnowledgeBaseOut)
async def get_kb_entry(kb_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(KnowledgeBase).where(KnowledgeBase.id == kb_id))
    entry = result.scalar_one_or_none()
    if not entry:
        raise HTTPException(status_code=404, detail="Knowledge base entry not found")
    return entry


@router.post("/knowledge-base", response_model=KnowledgeBaseOut, status_code=201)
async def create_kb_entry(body: KnowledgeBaseCreate, db: AsyncSession = Depends(get_db)):
    entry = KnowledgeBase(id=str(uuid.uuid4()), **body.model_dump())
    db.add(entry)
    await db.commit()
    await db.refresh(entry)
    await delete_pattern("kb:active:*")
    return entry


@router.patch("/knowledge-base/{kb_id}", response_model=KnowledgeBaseOut)
async def update_kb_entry(
    kb_id: str,
    body: KnowledgeBaseUpdate,
    db: AsyncSession = Depends(get_db),
):
    result = await db.execute(select(KnowledgeBase).where(KnowledgeBase.id == kb_id))
    entry = result.scalar_one_or_none()
    if not entry:
        raise HTTPException(status_code=404, detail="Knowledge base entry not found")
    for field, value in body.model_dump(exclude_none=True).items():
        setattr(entry, field, value)
    entry.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(entry)
    await delete_pattern("kb:active:*")
    return entry


@router.delete("/knowledge-base/{kb_id}", status_code=204)
async def delete_kb_entry(kb_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(KnowledgeBase).where(KnowledgeBase.id == kb_id))
    entry = result.scalar_one_or_none()
    if not entry:
        raise HTTPException(status_code=404, detail="Knowledge base entry not found")
    await db.delete(entry)
    await db.commit()
    await delete_pattern("kb:active:*")
