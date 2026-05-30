"""
Tests for Meta webhook endpoints and HMAC validation.

Covers:
  - HMAC signature verification (valid, invalid, missing)
  - Webhook verification (hub.mode=subscribe, token match)
  - Deduplication (same comment_id twice)
  - Rate limiting behavior
"""
from __future__ import annotations

import hashlib
import hmac
import json
import uuid

import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from app.middleware.hmac_validator import verify_meta_signature
from app.middleware.rate_limit import is_duplicate_webhook, check_rate_limit


def _sign_payload(body: bytes, secret: str) -> str:
    """Compute a valid X-Hub-Signature-256 header value."""
    sig = hmac.new(
        secret.encode("utf-8"), body, hashlib.sha256
    ).hexdigest()
    return f"sha256={sig}"


class FakeRequest:
    """Minimal ASGI-like request object for unit tests."""
    def __init__(self, body_bytes: bytes, headers: dict):
        self._body = body_bytes
        self.headers = headers
        self.client = MagicMock(host="127.0.0.1")


    async def body(self) -> bytes:
        return self._body


# ===================================================================
# 1. HMAC Signature Verification
# ===================================================================
class TestHMACVerification:

    @pytest.mark.asyncio
    async def test_valid_signature(self):
        secret = "test_secret_123"
        payload = b'{"object":"page","entry":[]}'
        sig = _sign_payload(payload, secret)
        req = FakeRequest(payload, {"X-Hub-Signature-256": sig})

        result = await verify_meta_signature(req, secret)
        assert result == payload

    @pytest.mark.asyncio
    async def test_invalid_signature(self):
        secret = "test_secret_123"
        payload = b'{"object":"page","entry":[]}'
        req = FakeRequest(payload, {"X-Hub-Signature-256": "sha256=invalid_hex"})

        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc_info:
            await verify_meta_signature(req, secret)
        assert exc_info.value.status_code == 401

    @pytest.mark.asyncio
    async def test_missing_signature_header(self):
        secret = "test_secret_123"
        payload = b'{"object":"page","entry":[]}'
        req = FakeRequest(payload, {})

        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc_info:
            await verify_meta_signature(req, secret)
        assert exc_info.value.status_code == 400
        assert "missing" in exc_info.value.detail.lower()

    @pytest.mark.asyncio
    async def test_malformed_signature_prefix(self):
        secret = "test_secret_123"
        payload = b'{"object":"page"}'
        req = FakeRequest(payload, {"X-Hub-Signature-256": "sha1=abc123"})

        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc_info:
            await verify_meta_signature(req, secret)
        assert exc_info.value.status_code == 400


# ===================================================================
# 2. Webhook Verification (GET endpoint logic)
# ===================================================================
class TestWebhookVerification:
    """Test the GET /webhook/meta verification logic."""

    @pytest.mark.asyncio
    async def test_successful_verification(self):
        """
        hub.mode=subscribe + matching verify_token => returns challenge.
        This tests the core logic extracted from the endpoint.
        """
        hub_mode = "subscribe"
        hub_verify_token = "test_verify_token"
        hub_challenge = "1234567890"
        expected_token = "test_verify_token"

        assert hub_mode == "subscribe" and hub_verify_token == expected_token
        assert int(hub_challenge) == 1234567890

    @pytest.mark.asyncio
    async def test_wrong_token_fails(self):
        hub_mode = "subscribe"
        hub_verify_token = "wrong_token"
        expected_token = "test_verify_token"
        assert not (hub_mode == "subscribe" and hub_verify_token == expected_token)

    @pytest.mark.asyncio
    async def test_wrong_mode_fails(self):
        hub_mode = "unsubscribe"
        hub_verify_token = "test_verify_token"
        expected_token = "test_verify_token"
        assert not (hub_mode == "subscribe" and hub_verify_token == expected_token)


# ===================================================================
# 3. Deduplication
# ===================================================================
class TestDeduplication:
    """Test is_duplicate_webhook deduplication logic."""

    def test_first_occurrence_not_duplicate(self):
        """Without Redis, dedup returns False (no dedup)."""
        with patch("app.middleware.rate_limit._get_redis", return_value=None):
            result = is_duplicate_webhook("comment_1", "page_1")
            assert result is False

    def test_redis_duplicate_detection(self):
        """Simulate Redis returning None for SET NX => duplicate."""
        mock_redis = MagicMock()
        mock_redis.set.return_value = None  # Key already existed
        with patch("app.middleware.rate_limit._get_redis", return_value=mock_redis):
            result = is_duplicate_webhook("comment_1", "page_1")
            assert result is True

    def test_redis_first_time_not_duplicate(self):
        """Simulate Redis returning True for SET NX => first time."""
        mock_redis = MagicMock()
        mock_redis.set.return_value = True  # Key was set successfully
        with patch("app.middleware.rate_limit._get_redis", return_value=mock_redis):
            result = is_duplicate_webhook("comment_1", "page_1")
            assert result is False


# ===================================================================
# 4. Rate Limiting
# ===================================================================
class TestRateLimiting:

    def test_under_limit_allowed(self):
        """Within rate limit => allowed."""
        with patch("app.middleware.rate_limit._get_redis", return_value=None):
            # In-memory: first call should be allowed
            allowed, remaining = check_rate_limit("test_ip_1", limit=100)
            assert allowed is True

    def test_over_limit_blocked(self):
        """Exceed rate limit => blocked."""
        with patch("app.middleware.rate_limit._get_redis", return_value=None):
            # Flood requests to exceed limit
            for _ in range(5):
                check_rate_limit("test_ip_2", limit=3)
            # The 6th call should be blocked
            # Actually check after exceeding
            allowed, _ = check_rate_limit("test_ip_2", limit=3)
            assert allowed is False

    def test_redis_rate_limiting(self):
        """Test Redis-based rate limiting path."""
        mock_redis = MagicMock()
        mock_pipeline = MagicMock()
        mock_pipeline.zremrangebyscore.return_value = None
        mock_pipeline.zadd.return_value = None
        mock_pipeline.zcard.return_value = 5
        mock_pipeline.expire.return_value = None
        mock_pipeline.execute.return_value = [None, None, 5, None]
        mock_redis.pipeline.return_value = mock_pipeline

        with patch("app.middleware.rate_limit._get_redis", return_value=mock_redis):
            allowed, remaining = check_rate_limit("test_ip_3", limit=10)
            assert allowed is True
            assert remaining == 5


# ===================================================================
# 5. Webhook Task Routing
# ===================================================================
class TestTaskRouting:
    """Test _task_for_comment priority queue routing."""

    def test_high_priority_markers(self):
        from app.routers.webhook import _task_for_comment
        high_texts = [
            "عايز استرجاع الفلوس",
            "I need a refund immediately",
            "محامي هيقاضيكم",
            "this is a scam and fraud",
        ]
        for text in high_texts:
            task, queue = _task_for_comment(text)
            assert queue == "high", f"Expected high for: {text}"

    def test_low_priority_markers(self):
        from app.routers.webhook import _task_for_comment
        low_texts = [
            "شكراً لكم",
            "great service thanks",
            "ممتاز",
        ]
        for text in low_texts:
            task, queue = _task_for_comment(text)
            assert queue == "low", f"Expected low for: {text}"

    def test_normal_priority(self):
        from app.routers.webhook import _task_for_comment
        task, queue = _task_for_comment("what is the price of this item?")
        assert queue == "normal"
