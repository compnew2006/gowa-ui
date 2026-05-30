import uuid
from fastapi import APIRouter, Depends, Query, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, delete
from datetime import datetime, timezone
from typing import List, Optional
from app.deps import get_db
from app.db import ScheduledPost
from pydantic import BaseModel

router = APIRouter(tags=["posts"])

class PostCreate(BaseModel):
    page_id: str
    platform: str
    message: str
    media_url: Optional[str] = None
    scheduled_at: datetime

class PostOut(PostCreate):
    id: str
    status: str
    post_id: Optional[str] = None
    error: Optional[str] = None
    posted_at: Optional[datetime] = None
    created_at: datetime

@router.get("/posts", response_model=List[PostOut])
async def get_posts(
    page_id: Optional[str] = Query(None),
    status: Optional[str] = Query(None),
    db: AsyncSession = Depends(get_db)
):
    query = select(ScheduledPost)
    if page_id:
        query = query.where(ScheduledPost.page_id == page_id)
    if status:
        query = query.where(ScheduledPost.status == status)
    
    query = query.order_by(ScheduledPost.scheduled_at.desc())
    result = await db.execute(query)
    return result.scalars().all()

@router.post("/posts", response_model=PostOut)
async def create_post(body: PostCreate, db: AsyncSession = Depends(get_db)):
    post = ScheduledPost(
        id=str(uuid.uuid4()),
        page_id=body.page_id,
        platform=body.platform,
        message=body.message,
        media_url=body.media_url,
        scheduled_at=body.scheduled_at,
        status="pending"
    )
    db.add(post)
    await db.commit()
    await db.refresh(post)
    return post

@router.delete("/posts/{id}")
async def delete_post(id: str, db: AsyncSession = Depends(get_db)):
    await db.execute(delete(ScheduledPost).where(ScheduledPost.id == id))
    await db.commit()
    return {"success": True}

class PostContentRequest(BaseModel):
    page_id: Optional[str] = None
    platform: str
    prompt: Optional[str] = None
    tone: Optional[str] = "Professional"
    language: Optional[str] = "ar"

class PostContentResponse(BaseModel):
    message: str

@router.post("/posts/generate-content", response_model=PostContentResponse)
async def generate_post_content(body: PostContentRequest, db: AsyncSession = Depends(get_db)):
    from app.services.runtime_settings import get_runtime_settings
    from app.ai.llm import get_llm
    from app.db import Page

    # Get settings (which contains Brand Kit)
    settings = await get_runtime_settings(db, body.page_id)
    
    # Get connected page name
    page_name = "العلامة التجارية"
    if body.page_id:
        page_res = await db.execute(select(Page).where(Page.id == body.page_id).limit(1))
        page = page_res.scalar_one_or_none()
        if page:
            page_name = page.name

    brand_desc = getattr(settings, "brand_description", None) or "نشاط تجاري متميز يقدم خدمات عالية الجودة."
    brand_ind = getattr(settings, "brand_industry", None) or "عام"
    brand_audience = getattr(settings, "brand_target_audience", None) or "الجمهور العام المهتم بمنتجاتنا."
    brand_tone = getattr(settings, "brand_tone_of_voice", None) or body.tone or "Professional"
    preferred_tags = getattr(settings, "brand_preferred_hashtags", None) or ""
    restricted = getattr(settings, "brand_restricted_words", None) or ""
    samples = getattr(settings, "brand_sample_posts", None) or ""

    user_prompt = body.prompt or "اكتب منشوراً ترحيبياً وتفاعلياً بمناسبة نهاية الأسبوع لعرض آخر الميزات والمنتجات."

    # Build prompt
    prompt_context = f"""
You are an expert social media copywriter.
Generate high-quality, engaging social media copy for the platform: {body.platform.upper()}.

Brand Information:
- Brand Name: {page_name}
- Industry: {brand_ind}
- Description: {brand_desc}
- Target Audience: {brand_audience}
- Preferred Tone: {brand_tone}
- Preferred Hashtags: {preferred_tags}
- Restricted Words (DO NOT USE THESE): {restricted}
- Sample Previous Posts (for style reference): {samples}

User Request:
- Goal/Prompt: {user_prompt}
- Tone requested: {body.tone or brand_tone}
- Language: {body.language or "ar"}

Instructions:
1. Write in the requested language: {body.language or "ar"} (typically Arabic "ar" or English "en").
2. Match the brand's tone of voice and target audience perfectly.
3. ABSOLUTELY DO NOT use any of the restricted words listed above.
4. Integrate the preferred hashtags naturally at the end if appropriate.
5. Create a clear, engaging call-to-action (CTA).
6. Return ONLY the final generated caption/post content. Do not include quotes, greetings, or meta-commentary like "Here is your post:".
"""

    try:
        llm = await get_llm()
        resp = await llm.ainvoke(prompt_context)
        content = resp.content.strip()
        
        # Strip code blocks or quotes if any returned by model
        if content.startswith("```"):
            lines = content.splitlines()
            if len(lines) > 2:
                content = "\n".join(lines[1:-1]).strip()
        
        # Strip starting/ending quotes
        if (content.startswith('"') and content.endswith('"')) or (content.startswith("'") and content.endswith("'")):
            content = content[1:-1].strip()

        return PostContentResponse(message=content)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"AI generation failed: {str(e)}")
