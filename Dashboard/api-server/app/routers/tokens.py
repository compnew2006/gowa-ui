"""
Token Lifecycle Management Router.
Supports: list health, manual refresh, exchange (short→long-lived), status.
"""
from fastapi import APIRouter, Depends, HTTPException, Body
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from datetime import datetime, timezone
import logging

from app.deps import get_db
from app.db import Page
from app.schemas import TokenStatus, PageOut
from app.services.token_service import encrypt_token, decrypt_token, exchange_for_long_lived_token
from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()
router = APIRouter(tags=["tokens"])


@router.get("/tokens", response_model=list[TokenStatus])
async def list_token_health(db: AsyncSession = Depends(get_db)):
    """List all pages with token health status."""
    result = await db.execute(select(Page).order_by(Page.name))
    pages = result.scalars().all()
    return [
        TokenStatus(
            id=p.id,
            name=p.name,
            platform=p.platform,
            token_status=p.token_status,
            token_expires_at=p.token_expires_at,
            token_last_refreshed_at=p.token_last_refreshed_at,
            token_last_error=p.token_last_error,
        )
        for p in pages
    ]


@router.post("/tokens/{page_id}/refresh", response_model=dict)
async def refresh_token(page_id: str, db: AsyncSession = Depends(get_db)):
    """Trigger manual token refresh via Meta Graph API exchange."""
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")
    if not page.access_token_encrypted:
        raise HTTPException(status_code=400, detail="No access token configured")

    from app.services.token_service import refresh_page_token
    outcome = await refresh_page_token(page_id)

    if outcome["success"]:
        return {"success": True, "message": "Token refreshed successfully"}
    raise HTTPException(
        status_code=502,
        detail=f"Token refresh failed: {outcome.get('error', 'Unknown error')}"
    )


@router.post("/tokens/{page_id}/exchange", response_model=dict)
async def exchange_token(
    page_id: str,
    short_lived_token: str = Body(..., embed=True),
    db: AsyncSession = Depends(get_db),
):
    """
    Exchange a short-lived user token for a long-lived Page Access Token.
    Encrypts and stores the token securely.
    """
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")

    if not settings.meta_app_id or not settings.meta_app_secret:
        raise HTTPException(status_code=503, detail="META_APP_ID and META_APP_SECRET not configured")

    try:
        long_token, expires_at = await exchange_for_long_lived_token(
            short_lived_token,
            settings.meta_app_id,
            settings.meta_app_secret,
        )
        page.access_token_encrypted = encrypt_token(long_token)
        page.token_expires_at = expires_at
        page.token_status = "valid"
        page.token_last_refreshed_at = datetime.now(timezone.utc)
        page.token_last_error = None
        await db.commit()
        logger.info("[Tokens] Token exchanged and stored for page %s", page.name)
        return {
            "success": True,
            "expires_at": expires_at.isoformat() if expires_at else None,
            "message": "Token exchanged and stored with AES-256 encryption",
        }
    except Exception as e:
        raise HTTPException(status_code=502, detail=f"Token exchange failed: {str(e)}")


@router.post("/tokens/{page_id}/set", response_model=dict)
async def set_token(
    page_id: str,
    access_token: str = Body(..., embed=True),
    db: AsyncSession = Depends(get_db),
):
    """
    Directly store an access token (encrypted). 
    Use when you already have a long-lived token.
    """
    result = await db.execute(select(Page).where(Page.id == page_id))
    page = result.scalar_one_or_none()
    if not page:
        raise HTTPException(status_code=404, detail="Page not found")

    page.access_token_encrypted = encrypt_token(access_token)
    page.token_status = "valid"
    page.token_last_refreshed_at = datetime.now(timezone.utc)
    page.token_last_error = None
    await db.commit()
    return {"success": True, "message": "Token stored (AES-256 encrypted)"}
