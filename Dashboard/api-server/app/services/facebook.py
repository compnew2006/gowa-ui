"""
Facebook Graph API — post replies to comments.

Uses the Page Access Token (encrypted at rest) to publish
a reply to a specific comment via the Graph API.
"""
from __future__ import annotations

import logging

import httpx
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Page, Conversation
from app.services.token_service import decrypt_token

logger = logging.getLogger(__name__)

GRAPH_API_VERSION = "v19.0"


async def post_reply_to_comment(
    db: AsyncSession,
    conversation: Conversation,
    reply_text: str | None = None,
) -> dict:
    """
    Post a reply to the Facebook comment linked to *conversation*.

    If *reply_text* is provided it is used; otherwise the existing
    ``conversation.ai_reply`` is used.

    Returns {"success": bool, "fb_comment_id": str | None, "error": str | None}.
    """
    text = reply_text or conversation.ai_reply
    if not text:
        return {"success": False, "fb_comment_id": None, "error": "No reply text"}

    if not conversation.comment_id:
        return {"success": False, "fb_comment_id": None, "error": "No comment_id"}

    # Resolve page → access token
    result = await db.execute(select(Page).where(Page.id == conversation.page_id))
    page = result.scalar_one_or_none()
    if not page or not page.access_token_encrypted:
        return {"success": False, "fb_comment_id": None, "error": "Page or token not found"}

    access_token = decrypt_token(page.access_token_encrypted)
    if not access_token:
        return {"success": False, "fb_comment_id": None, "error": "Token decryption failed"}

    # Call Graph API — POST /{comment-id}/comments
    url = f"https://graph.facebook.com/{GRAPH_API_VERSION}/{conversation.comment_id}/comments"
    payload = {"message": text, "access_token": access_token}

    try:
        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.post(url, data=payload)
            data = resp.json()

        if resp.status_code >= 400:
            error_msg = data.get("error", {}).get("message", str(data))
            logger.error("[FB] Reply failed: %s | conv=%s", error_msg, conversation.id)
            return {"success": False, "fb_comment_id": None, "error": error_msg}

        fb_comment_id = data.get("id", "")
        logger.info("[FB] Reply posted: fb_id=%s | conv=%s", fb_comment_id, conversation.id)
        return {"success": True, "fb_comment_id": fb_comment_id, "error": None}

    except Exception as exc:
        logger.exception("[FB] Reply request failed")
        return {"success": False, "fb_comment_id": None, "error": str(exc)}


async def post_private_reply_to_comment(
    db: AsyncSession,
    conversation: Conversation,
    message: str,
) -> dict:
    """
    Send a private message to a commenter (Messenger Private Reply).
    Returns {"success": bool, "id": str | None, "error": str | None}.
    """
    if not conversation.comment_id:
        return {"success": False, "id": None, "error": "No comment_id"}

    # Resolve page → access token
    result = await db.execute(select(Page).where(Page.id == conversation.page_id))
    page = result.scalar_one_or_none()
    if not page or not page.access_token_encrypted:
        return {"success": False, "id": None, "error": "Page or token not found"}

    access_token = decrypt_token(page.access_token_encrypted)
    if not access_token:
        return {"success": False, "id": None, "error": "Token decryption failed"}

    # Call Graph API — POST /{page-id}/messages (Messenger Private Reply)
    url = f"https://graph.facebook.com/{GRAPH_API_VERSION}/{page.page_id}/messages"
    params = {"access_token": access_token}
    json_payload = {
        "recipient": {"comment_id": conversation.comment_id},
        "message": {"text": message}
    }

    try:
        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.post(url, params=params, json=json_payload)
            data = resp.json()

        if resp.status_code >= 400:
            error_msg = data.get("error", {}).get("message", str(data))
            logger.error("[FB] Private reply failed: %s | conv=%s", error_msg, conversation.id)
            return {"success": False, "id": None, "error": error_msg}

        message_id = data.get("id", "")
        logger.info("[FB] Private reply sent: id=%s | conv=%s", message_id, conversation.id)
        return {"success": True, "id": message_id, "error": None}

    except Exception as exc:
        logger.exception("[FB] Private reply request failed")
        return {"success": False, "id": None, "error": str(exc)}
