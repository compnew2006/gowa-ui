import uuid
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from app.deps import get_db
from app.db import Page
from app.schemas import PageOut, PageCreate, PageUpdate

router = APIRouter(tags=["pages"])


@router.get("/pages", response_model=list[PageOut])
async def list_pages(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Page).order_by(Page.created_at.desc()))
    pages = result.scalars().all()
    return [_mask_page(p) for p in pages]


@router.get("/pages/{page_id}", response_model=PageOut)
async def get_page(page_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")
    return _mask_page(page)


@router.post("/pages", response_model=PageOut, status_code=201)
async def create_page(body: PageCreate, db: AsyncSession = Depends(get_db)):
    if body.auto_reply_enabled and not body.auto_reply_end_date:
        raise HTTPException(status_code=400, detail="التاريخ غير محدد")
    page = Page(id=str(uuid.uuid4()), **body.model_dump())
    db.add(page)
    await db.commit()
    await db.refresh(page)
    return _mask_page(page)


@router.patch("/pages/{page_id}", response_model=PageOut)
async def update_page(page_id: str, body: PageUpdate, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")
    
    target_enabled = body.auto_reply_enabled if body.auto_reply_enabled is not None else page.auto_reply_enabled
    target_end_date = body.auto_reply_end_date if body.auto_reply_end_date is not None else page.auto_reply_end_date
    if target_enabled and not target_end_date:
        raise HTTPException(status_code=400, detail="التاريخ غير محدد")

    for field, value in body.model_dump(exclude_unset=True).items():
        setattr(page, field, value)
    page.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(page)
    return _mask_page(page)


def _mask_page(page: Page) -> dict:
    """Exclude access_token_encrypted from API responses."""
    data = {
        "id": page.id,
        "platform": page.platform,
        "page_id": page.page_id,
        "name": page.name,
        "avatar_url": page.avatar_url,
        "is_active": page.is_active,
        "auto_reply_enabled": page.auto_reply_enabled,
        "shadow_mode": page.shadow_mode,
        "tracking_start_date": page.tracking_start_date,
        "auto_reply_end_date": page.auto_reply_end_date,
        "token_status": page.token_status,
        "token_expires_at": page.token_expires_at,
        "token_last_refreshed_at": page.token_last_refreshed_at,
        "token_last_error": page.token_last_error,
        "created_at": page.created_at,
        "updated_at": page.updated_at,
    }
    return data


@router.delete("/pages/{page_id}", status_code=204)
async def delete_page(page_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")
    await db.delete(page)
    await db.commit()
