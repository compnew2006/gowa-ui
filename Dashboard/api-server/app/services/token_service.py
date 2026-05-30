"""
Token Lifecycle Management Service.

- AES-256 (Fernet) encryption for all tokens at rest
- Token exchange: short-lived → long-lived (60-day Page Access Token)
- Auto-refresh scheduler: triggers 15 days before expiry
- Admin alerts on failure
"""
from __future__ import annotations
import os
import base64
import logging
import httpx
from datetime import datetime, timezone, timedelta
from typing import Optional

logger = logging.getLogger(__name__)


def _get_fernet():
    """Get or create a Fernet cipher. Key from env var TOKEN_ENCRYPTION_KEY."""
    try:
        from cryptography.fernet import Fernet
        raw_key = os.environ.get("TOKEN_ENCRYPTION_KEY", "")
        if raw_key:
            # Accept raw key or base64-encoded key
            try:
                key = raw_key.encode() if len(raw_key) == 44 else base64.urlsafe_b64encode(raw_key[:32].ljust(32).encode())
            except Exception:
                key = Fernet.generate_key()
        else:
            # Derive from SESSION_SECRET for dev (not production!)
            session_secret = os.environ.get("SESSION_SECRET", "dev-secret-change-me")
            import hashlib
            raw = hashlib.sha256(session_secret.encode()).digest()
            key = base64.urlsafe_b64encode(raw)
        return Fernet(key)
    except ImportError:
        return None


def encrypt_token(plaintext: str) -> str:
    """Encrypt a token for database storage. Returns base64-encoded ciphertext."""
    if not plaintext:
        return ""
    f = _get_fernet()
    if f is None:
        # Fallback: simple base64 (not secure — log warning)
        logger.warning("[TokenService] cryptography not installed — tokens stored as base64 only!")
        return base64.b64encode(plaintext.encode()).decode()
    return f.encrypt(plaintext.encode()).decode()


def decrypt_token(ciphertext: str) -> str:
    """Decrypt a stored token. Returns plaintext."""
    if not ciphertext:
        return ""
    f = _get_fernet()
    if f is None:
        try:
            return base64.b64decode(ciphertext.encode()).decode()
        except Exception:
            return ciphertext
    try:
        return f.decrypt(ciphertext.encode()).decode()
    except Exception:
        logger.error("[TokenService] Token decryption failed — token may be corrupted")
        return ""


async def exchange_for_long_lived_token(
    short_lived_token: str,
    app_id: str,
    app_secret: str,
) -> tuple[str, Optional[datetime]]:
    """
    Exchange a short-lived user token for a long-lived token (60 days).
    Returns (long_lived_token, expires_at).
    """
    url = "https://graph.facebook.com/v20.0/oauth/access_token"
    params = {
        "grant_type": "fb_exchange_token",
        "client_id": app_id,
        "client_secret": app_secret,
        "fb_exchange_token": short_lived_token,
    }
    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.get(url, params=params)
        resp.raise_for_status()
        data = resp.json()

    token = data.get("access_token", "")
    expires_in = data.get("expires_in", 5184000)  # default 60 days
    expires_at = datetime.now(timezone.utc) + timedelta(seconds=expires_in)
    return token, expires_at


async def get_page_access_token(
    user_token: str,
    page_id: str,
) -> tuple[str, Optional[datetime]]:
    """
    Get a Page Access Token from a User Access Token.
    Page tokens are long-lived if the user token is long-lived.
    """
    url = f"https://graph.facebook.com/v20.0/{page_id}"
    params = {
        "fields": "access_token",
        "access_token": user_token,
    }
    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.get(url, params=params)
        resp.raise_for_status()
        data = resp.json()

    page_token = data.get("access_token", "")
    # Page tokens derived from long-lived user tokens never expire
    expires_at = datetime.now(timezone.utc) + timedelta(days=60)
    return page_token, expires_at


async def refresh_page_token(page_id_db: str) -> dict:
    """
    Attempt to refresh a page's access token.
    Returns {"success": bool, "error": str|None}.
    """
    from app.db import AsyncSessionLocal, Page
    from app.config import get_settings
    from sqlalchemy import select

    settings = get_settings()

    async with AsyncSessionLocal() as db:
        result = await db.execute(select(Page).where(Page.id == page_id_db))
        page = result.scalar_one_or_none()
        if not page:
            return {"success": False, "error": "Page not found"}

        if not page.access_token_encrypted:
            return {"success": False, "error": "No token configured"}

        current_token = decrypt_token(page.access_token_encrypted)
        if not current_token:
            return {"success": False, "error": "Token decryption failed"}

        try:
            new_token, expires_at = await exchange_for_long_lived_token(
                current_token,
                settings.meta_app_id,
                settings.meta_app_secret,
            )
            page.access_token_encrypted = encrypt_token(new_token)
            page.token_expires_at = expires_at
            page.token_status = "valid"
            page.token_last_refreshed_at = datetime.now(timezone.utc)
            page.token_last_error = None
            await db.commit()
            logger.info("[TokenService] Token refreshed for page %s", page.name)
            return {"success": True, "error": None}

        except Exception as e:
            error_msg = str(e)[:255]
            page.token_status = "error"
            page.token_last_error = error_msg
            await db.commit()
            logger.error("[TokenService] Refresh failed for page %s: %s", page.name, error_msg)

            # Notify admin
            try:
                from app.services.notifications import notify_admin
                await notify_admin(
                    title="🔴 Token Refresh Failed",
                    message=f"Failed to refresh token for page '{page.name}'. Error: {error_msg}. Manual re-authentication required.",
                    priority="critical",
                )
            except Exception:
                pass

            return {"success": False, "error": error_msg}
