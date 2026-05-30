"""
Conversation service layer.

Business logic extracted from the conversations router.
Handles CRUD operations and state transitions for conversations.
"""
from datetime import datetime, timezone

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Conversation


async def list_conversations(
    db: AsyncSession,
    *,
    page: int = 1,
    limit: int = 20,
    status: str | None = None,
    page_id: str | None = None,
    search: str | None = None,
) -> tuple[list, int]:
    """Return (conversations, total_count) matching the given filters."""
    q = select(Conversation)
    if status:
        q = q.where(Conversation.status == status)
    if page_id:
        q = q.where(Conversation.page_id == page_id)
    if search:
        q = q.where(
            Conversation.original_comment.ilike(f"%{search}%")
            | Conversation.customer_name.ilike(f"%{search}%")
        )

    count_q = select(func.count()).select_from(q.subquery())
    total_result = await db.execute(count_q)
    total = total_result.scalar_one()

    q = q.order_by(Conversation.created_at.desc()).offset((page - 1) * limit).limit(limit)
    result = await db.execute(q)
    data = result.scalars().all()

    return data, total


async def get_conversation(db: AsyncSession, conv_id: str) -> Conversation | None:
    """Fetch a single conversation by ID, or None if not found."""
    result = await db.execute(select(Conversation).where(Conversation.id == conv_id))
    return result.scalar_one_or_none()


async def manual_reply(
    db: AsyncSession, conv_id: str, reply_text: str
) -> Conversation | None:
    """Apply a manual admin reply to a conversation and post it to Facebook."""
    conv = await get_conversation(db, conv_id)
    if not conv:
        return None

    # Post to Facebook
    from app.services.facebook import post_reply_to_comment, post_private_reply_to_comment
    from app.services.runtime_settings import get_runtime_settings
    
    settings_obj = await get_runtime_settings(db)
    intent = conv.intent or "general"
    
    # Smart Dynamic Routing based on Intent
    is_compliment_or_general = intent in ("compliment", "general")
    
    if is_compliment_or_general:
        # For compliments/general comments, the admin manual reply is the PUBLIC comment reply!
        public_msg = reply_text
        do_private = False
        priv_msg = None
    else:
        # For inquiries/complaints/refunds/purchase (leads), send private reply and say "we contacted you" publicly
        public_msg = "\u062a\u0645 \u0627\u0644\u062a\u0648\u0627\u0635\u0644 \u0645\u0639\u0643 \u0639\u0644\u0649 \u0627\u0644\u062e\u0627\u0635"
        do_private = settings_obj.enable_private_replies if settings_obj else True
        priv_msg = reply_text
    
    # 1. Post Public Comment Reply
    pub_result = await post_reply_to_comment(db, conv, public_msg)
    
    # 2. Post Private Messenger Reply (if required)
    priv_result = {"success": False}
    if do_private and priv_msg:
        priv_result = await post_private_reply_to_comment(db, conv, priv_msg)
    else:
        priv_result = {"success": True, "skipped": True}

    if pub_result["success"] or priv_result["success"]:
        conv.ai_reply = public_msg
        conv.admin_reply = priv_msg
        conv.status = "replied"
        conv.replied_at = datetime.now(timezone.utc)
    else:
        conv.status = "escalated"
        conv.escalation_reason = f"FB reply failed. Public: {pub_result.get('error')}, Private: {priv_result.get('error')}"

    conv.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(conv)
    return conv




async def approve_reply(
    db: AsyncSession, conv_id: str, reply_text: str | None = None
) -> Conversation | None:
    """Approve and post the AI reply (or custom text) to Facebook."""
    from app.services.facebook import post_reply_to_comment

    conv = await get_conversation(db, conv_id)
    if not conv:
        return None

    text = reply_text or conv.ai_reply
    if not text:
        return None

    # Post to Facebook
    from app.services.facebook import post_reply_to_comment, post_private_reply_to_comment
    from app.services.runtime_settings import get_runtime_settings
    
    settings_obj = await get_runtime_settings(db)
    intent = conv.intent or "general"
    
    # Smart Dynamic Routing based on Intent
    is_compliment_or_general = intent in ("compliment", "general")
    
    if is_compliment_or_general:
        # For compliments/general comments, the AI generated reply should be the PUBLIC comment reply!
        public_msg = text
        do_private = False
        priv_msg = None
    else:
        # For inquiries/complaints/refunds/purchase (leads), send private reply and say "we contacted you" publicly
        public_msg = "\u062a\u0645 \u0627\u0644\u062a\u0648\u0627\u0635\u0644 \u0645\u0639\u0643 \u0639\u0644\u0649 \u0627\u0644\u062e\u0627\u0635"
        do_private = settings_obj.enable_private_replies if settings_obj else True
        priv_msg = text
    
    # 1. Post Public Comment Reply
    pub_result = await post_reply_to_comment(db, conv, public_msg)
    
    # 2. Post Private Messenger Reply (if required)
    priv_result = {"success": False}
    if do_private and priv_msg:
        priv_result = await post_private_reply_to_comment(db, conv, priv_msg)
    else:
        priv_result = {"success": True, "skipped": True}

    if pub_result["success"] or priv_result["success"]:
        conv.ai_reply = public_msg
        conv.admin_reply = priv_msg
        conv.status = "replied"
        conv.replied_at = datetime.now(timezone.utc)
    else:
        conv.status = "escalated"
        conv.escalation_reason = f"FB reply failed. Public: {pub_result.get('error')}, Private: {priv_result.get('error')}"

    conv.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(conv)
    return conv

async def resolve_conversation(
    db: AsyncSession, conv_id: str
) -> Conversation | None:
    """Mark a conversation as resolved."""
    conv = await get_conversation(db, conv_id)
    if not conv:
        return None
    conv.status = "resolved"
    conv.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(conv)
    return conv
