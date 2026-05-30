from fastapi import APIRouter, Depends, HTTPException, Query, UploadFile, File
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from typing import List, Optional
import uuid
import os
import shutil

from app.deps import get_db
from app.db import Campaign, Customer
from app.schemas import CampaignOut, CampaignCreate, CampaignUpdate, CampaignListResponse

router = APIRouter(tags=["campaigns"])

UPLOAD_DIR = "/tmp/campaign_media"
os.makedirs(UPLOAD_DIR, exist_ok=True)


def _build_customer_query(target_filter: dict):
    q = select(Customer).where(Customer.gdpr_deleted == False)
    if target_filter.get("purchase_intent"):
        q = q.where(Customer.purchase_intent == target_filter["purchase_intent"])
    if target_filter.get("conversion_status"):
        q = q.where(Customer.conversion_status == target_filter["conversion_status"])
    if target_filter.get("churn_risk"):
        q = q.where(Customer.churn_risk == target_filter["churn_risk"])
    return q


@router.get("/campaigns", response_model=CampaignListResponse)
async def list_campaigns(
    page: int = Query(1, ge=1),
    limit: int = Query(20, ge=1, le=100),
    page_id: Optional[str] = None,
    status: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    q = select(Campaign)
    if page_id:
        q = q.where(Campaign.page_id == page_id)
    if status:
        q = q.where(Campaign.status == status)

    count_q = select(func.count()).select_from(q.subquery())
    total = (await db.execute(count_q)).scalar_one()

    q = q.order_by(Campaign.created_at.desc()).offset((page - 1) * limit).limit(limit)
    result = await db.execute(q)
    data = result.scalars().all()

    return CampaignListResponse(data=data, total=total, page=page, limit=limit)


@router.get("/campaigns/{campaign_id}", response_model=CampaignOut)
async def get_campaign(campaign_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    return campaign


@router.post("/campaigns", response_model=CampaignOut, status_code=201)
async def create_campaign(body: CampaignCreate, db: AsyncSession = Depends(get_db)):
    # Count recipients based on target filter
    total_recipients = 0
    if body.customer_ids:
        total_recipients = len(body.customer_ids)
    elif body.target_filter:
        count_q = select(func.count()).select_from(
            _build_customer_query(body.target_filter).subquery()
        )
        total_recipients = (await db.execute(count_q)).scalar_one()

    campaign = Campaign(
        id=str(uuid.uuid4()),
        name=body.name,
        description=body.description,
        target_filter=body.target_filter,
        customer_ids=body.customer_ids,
        message_ar=body.message_ar,
        message_en=body.message_en,
        media_urls=body.media_urls,
        media_type=body.media_type,
        send_at=body.send_at,
        interval_hours=body.interval_hours,
        max_sends=body.max_sends,
        total_recipients=total_recipients,
        status="draft",
    )
    db.add(campaign)
    await db.commit()
    await db.refresh(campaign)
    return campaign


@router.patch("/campaigns/{campaign_id}", response_model=CampaignOut)
async def update_campaign(
    campaign_id: str, body: CampaignUpdate, db: AsyncSession = Depends(get_db)
):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")

    for field, value in body.model_dump(exclude_none=True).items():
        setattr(campaign, field, value)

    # Recount recipients if target changed
    if body.customer_ids is not None:
        campaign.total_recipients = len(body.customer_ids)
    elif body.target_filter is not None:
        count_q = select(func.count()).select_from(
            _build_customer_query(body.target_filter).subquery()
        )
        campaign.total_recipients = (await db.execute(count_q)).scalar_one()

    campaign.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(campaign)
    return campaign


@router.delete("/campaigns/{campaign_id}", status_code=204)
async def delete_campaign(campaign_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    await db.delete(campaign)
    await db.commit()


@router.post("/campaigns/{campaign_id}/activate", response_model=CampaignOut)
async def activate_campaign(campaign_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    campaign.status = "active"
    campaign.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(campaign)
    return campaign


@router.post("/campaigns/{campaign_id}/pause", response_model=CampaignOut)
async def pause_campaign(campaign_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")
    campaign.status = "paused"
    campaign.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(campaign)
    return campaign


@router.get("/campaigns/{campaign_id}/preview-recipients")
async def preview_recipients(campaign_id: str, db: AsyncSession = Depends(get_db)):
    """Preview how many customers match the campaign filter."""
    result = await db.execute(select(Campaign).where(Campaign.id == campaign_id))
    campaign = result.scalar_one_or_none()
    if not campaign:
        raise HTTPException(status_code=404, detail="Campaign not found")

    if campaign.customer_ids:
        count = len(campaign.customer_ids)
        customers = []
        for cid in campaign.customer_ids[:5]:
            cr = await db.execute(select(Customer).where(Customer.id == cid))
            c = cr.scalar_one_or_none()
            if c:
                customers.append({"id": c.id, "name": c.full_name or c.username})
    else:
        q = _build_customer_query(campaign.target_filter)
        count_q = select(func.count()).select_from(q.subquery())
        count = (await db.execute(count_q)).scalar_one()
        sample_q = q.limit(5)
        sample_result = await db.execute(sample_q)
        customers = [
            {"id": c.id, "name": c.full_name or c.username}
            for c in sample_result.scalars().all()
        ]

    return {"count": count, "sample": customers}


@router.post("/campaigns/upload-media")
async def upload_media(file: UploadFile = File(...)):
    """Upload image or video for campaign."""
    allowed_types = {
        "image/jpeg", "image/png", "image/gif", "image/webp",
        "video/mp4", "video/quicktime", "video/webm",
    }
    if file.content_type not in allowed_types:
        raise HTTPException(status_code=400, detail=f"Unsupported file type: {file.content_type}")

    ext = file.filename.rsplit(".", 1)[-1] if "." in file.filename else "bin"
    filename = f"{uuid.uuid4()}.{ext}"
    filepath = os.path.join(UPLOAD_DIR, filename)

    with open(filepath, "wb") as f:
        shutil.copyfileobj(file.file, f)

    media_type = "image" if file.content_type.startswith("image") else "video"
    return {"url": f"/api/campaigns/media/{filename}", "media_type": media_type, "filename": filename}


@router.get("/campaigns/media/{filename}")
async def serve_media(filename: str):
    from fastapi.responses import FileResponse
    filepath = os.path.join(UPLOAD_DIR, filename)
    if not os.path.exists(filepath):
        raise HTTPException(status_code=404, detail="File not found")
    return FileResponse(filepath)
