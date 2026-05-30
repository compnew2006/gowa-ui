"""
Webhook HMAC-SHA256 Signature Validation.
Verifies X-Hub-Signature-256 header from Meta webhook requests.
Constant-time comparison prevents timing attacks.
"""
from __future__ import annotations
import hashlib
import hmac
import logging
from fastapi import Request, HTTPException

logger = logging.getLogger(__name__)


async def verify_meta_signature(request: Request, app_secret: str) -> bytes:
    """
    Validates the X-Hub-Signature-256 header.
    Returns raw body bytes for downstream use.
    Raises HTTP 401 on failure, HTTP 400 if header missing.
    """
    body = await request.body()

    sig_header = request.headers.get("X-Hub-Signature-256", "")
    if not sig_header:
        logger.warning(
            "Webhook rejected: missing X-Hub-Signature-256 | ip=%s",
            _get_ip(request),
        )
        raise HTTPException(status_code=400, detail="Missing signature header")

    if not sig_header.startswith("sha256="):
        logger.warning("Webhook rejected: malformed signature header | ip=%s", _get_ip(request))
        raise HTTPException(status_code=400, detail="Malformed signature header")

    expected = "sha256=" + hmac.new(
        app_secret.encode("utf-8"),
        body,
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(expected, sig_header):
        logger.warning(
            "Webhook rejected: HMAC mismatch | ip=%s | received=%s",
            _get_ip(request),
            sig_header[:20] + "...",
        )
        raise HTTPException(status_code=401, detail="Invalid webhook signature")

    return body


def _get_ip(request: Request) -> str:
    return (
        request.headers.get("X-Forwarded-For", "").split(",")[0].strip()
        or (request.client.host if request.client else "unknown")
    )
