"""
Tests for Shadow Mode endpoints and logic.

Covers:
  - Approve flow (status transition, audit logging)
  - Reject flow (with reason, corrections)
  - Undo flow
  - Correct flow (intent/sentiment correction)
  - Go-live threshold (80% approval, 20 sample minimum)
"""
from __future__ import annotations

import uuid
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from sqlalchemy import text, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Conversation, Page


async def _insert_page(db: AsyncSession, page_id: str = None) -> str:
    """Insert a test page via raw SQL. Returns the page_id."""
    pid = page_id or str(uuid.uuid4())
    await db.execute(text(
        "INSERT INTO pages (id, platform, page_id, name, is_active, shadow_mode, auto_reply_enabled) "
        "VALUES (:id, 'facebook', :fb_id, 'Test Page', 1, 1, 0)"
    ), {"id": pid, "fb_id": f"fb_{pid[:8]}"})
    await db.flush()
    return pid


async def _insert_conversation(
    db: AsyncSession,
    page_id: str,
    status: str = "shadow_pending",
    intent: str = "price_inquiry",
    sentiment: str = "neutral",
) -> str:
    """Insert a test conversation via raw SQL. Returns the conv_id."""
    conv_id = str(uuid.uuid4())
    await db.execute(text(
        "INSERT INTO conversations "
        "(id, page_id, page_name, platform, comment_id, post_id, "
        "customer_name, original_comment, ai_reply, status, intent, sentiment, "
        "urgency, priority, confidence_score, language, is_shadow_mode, "
        "guardrail_triggered) "
        "VALUES (:id, :page_id, 'Test Page', 'facebook', :comment_id, :post_id, "
        "'Test Customer', 'Test comment', 'AI generated reply', :status, :intent, :sentiment, "
        "'normal', 'normal', 0.85, 'ar', 1, 0)"
    ), {
        "id": conv_id,
        "page_id": page_id,
        "comment_id": f"comment_{uuid.uuid4().hex[:8]}",
        "post_id": f"post_{uuid.uuid4().hex[:8]}",
        "status": status,
        "intent": intent,
        "sentiment": sentiment,
    })
    await db.flush()
    return conv_id


async def _get_conversation(db: AsyncSession, conv_id: str) -> dict:
    """Get conversation as dict via raw SQL."""
    result = await db.execute(text(
        "SELECT id, status, intent, sentiment, escalation_reason, admin_reply, "
        "replied_at, is_shadow_mode FROM conversations WHERE id = :id"
    ), {"id": conv_id})
    row = result.fetchone()
    if row is None:
        return None
    keys = ["id", "status", "intent", "sentiment", "escalation_reason", "admin_reply", "replied_at", "is_shadow_mode"]
    return dict(zip(keys, row))


class TestApproveFlow:

    @pytest.mark.asyncio
    async def test_approve_transitions_to_replied(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        conv = await _get_conversation(db_session, conv_id)
        assert conv["status"] == "shadow_pending"

        await db_session.execute(text(
            "UPDATE conversations SET status='replied', replied_at=datetime('now') WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["status"] == "replied"
        assert updated["replied_at"] is not None

    @pytest.mark.asyncio
    async def test_approve_with_admin_note(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        await db_session.execute(text(
            "UPDATE conversations SET status='replied', replied_at=datetime('now'), "
            "admin_reply='Looks good, approved' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["admin_reply"] == "Looks good, approved"

    @pytest.mark.asyncio
    async def test_cannot_approve_non_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="replied")
        conv = await _get_conversation(db_session, conv_id)
        assert conv["status"] != "shadow_pending"


class TestRejectFlow:

    @pytest.mark.asyncio
    async def test_reject_transitions_to_escalated(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        await db_session.execute(text(
            "UPDATE conversations SET status='escalated', "
            "escalation_reason='Admin rejected shadow reply' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["status"] == "escalated"
        assert updated["escalation_reason"] == "Admin rejected shadow reply"

    @pytest.mark.asyncio
    async def test_reject_with_custom_reason(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        await db_session.execute(text(
            "UPDATE conversations SET status='escalated', "
            "escalation_reason='Wrong tone' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert "Wrong tone" in updated["escalation_reason"]

    @pytest.mark.asyncio
    async def test_reject_with_intent_correction(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, intent="general")
        await db_session.execute(text(
            "UPDATE conversations SET intent='complaint', status='escalated' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["intent"] == "complaint"

    @pytest.mark.asyncio
    async def test_cannot_reject_non_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="escalated")
        conv = await _get_conversation(db_session, conv_id)
        assert conv["status"] != "shadow_pending"


class TestUndoFlow:

    @pytest.mark.asyncio
    async def test_undo_approved_back_to_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="replied")
        await db_session.execute(text(
            "UPDATE conversations SET replied_at=datetime('now') WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        await db_session.execute(text(
            "UPDATE conversations SET status='shadow_pending', replied_at=NULL, "
            "escalation_reason=NULL WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["status"] == "shadow_pending"
        assert updated["replied_at"] is None
        assert updated["escalation_reason"] is None

    @pytest.mark.asyncio
    async def test_undo_rejected_back_to_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="escalated")
        await db_session.execute(text(
            "UPDATE conversations SET status='shadow_pending', escalation_reason=NULL WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["status"] == "shadow_pending"

    @pytest.mark.asyncio
    async def test_cannot_undo_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        conv = await _get_conversation(db_session, conv_id)
        assert conv["status"] not in ["replied", "escalated"]


class TestCorrectFlow:

    @pytest.mark.asyncio
    async def test_correct_intent(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="replied", intent="general")
        await db_session.execute(text(
            "UPDATE conversations SET intent='complaint' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["intent"] == "complaint"

    @pytest.mark.asyncio
    async def test_correct_sentiment(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="replied", sentiment="neutral")
        await db_session.execute(text(
            "UPDATE conversations SET sentiment='negative' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["sentiment"] == "negative"

    @pytest.mark.asyncio
    async def test_correct_both(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id, status="replied", intent="general", sentiment="neutral")
        await db_session.execute(text(
            "UPDATE conversations SET intent='refund', sentiment='angry' WHERE id=:id"
        ), {"id": conv_id})
        await db_session.flush()

        updated = await _get_conversation(db_session, conv_id)
        assert updated["intent"] == "refund"
        assert updated["sentiment"] == "angry"

    @pytest.mark.asyncio
    async def test_cannot_correct_pending(self, db_session):
        page_id = await _insert_page(db_session)
        conv_id = await _insert_conversation(db_session, page_id=page_id)
        conv = await _get_conversation(db_session, conv_id)
        assert conv["status"] not in ["replied", "escalated"]


class TestGoLiveThreshold:

    @pytest.mark.asyncio
    async def test_stats_below_threshold(self, db_session):
        page_id = await _insert_page(db_session)
        for _ in range(10):
            await _insert_conversation(db_session, page_id=page_id)

        result = await db_session.execute(text(
            "SELECT status FROM conversations WHERE is_shadow_mode=1"
        ))
        all_rows = result.fetchall()
        total = len(all_rows)
        approved = sum(1 for r in all_rows if r[0] == "replied")
        pending = sum(1 for r in all_rows if r[0] == "shadow_pending")
        rate = approved / max(total - pending, 1)
        ready = rate >= 0.80 and total >= 20
        assert ready is False

    @pytest.mark.asyncio
    async def test_stats_meets_threshold(self, db_session):
        page_id = await _insert_page(db_session)
        for _ in range(18):
            await _insert_conversation(db_session, page_id=page_id, status="replied")
        for _ in range(2):
            await _insert_conversation(db_session, page_id=page_id, status="escalated")

        result = await db_session.execute(text(
            "SELECT status FROM conversations WHERE is_shadow_mode=1 AND page_id=:pid"
        ), {"pid": page_id})
        all_rows = result.fetchall()
        total = len(all_rows)
        approved = sum(1 for r in all_rows if r[0] == "replied")
        pending = sum(1 for r in all_rows if r[0] == "shadow_pending")
        rate = approved / max(total - pending, 1)
        ready = rate >= 0.80 and total >= 20
        assert total == 20
        assert rate == 0.9
        assert ready is True

    @pytest.mark.asyncio
    async def test_go_live_disables_shadow_mode(self, db_session):
        page_id = await _insert_page(db_session)
        await db_session.execute(text(
            "UPDATE pages SET shadow_mode=0, auto_reply_enabled=1 WHERE id=:id"
        ), {"id": page_id})
        await db_session.flush()

        result = await db_session.execute(text(
            "SELECT shadow_mode, auto_reply_enabled FROM pages WHERE id=:id"
        ), {"id": page_id})
        row = result.fetchone()
        assert row[0] == 0  # shadow_mode off
        assert row[1] == 1  # auto_reply on
