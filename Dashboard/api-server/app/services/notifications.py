"""
Multi-channel Admin Notification Service.

Channels:
  1. Telegram Bot (primary — instant)
  2. Email via SMTP (secondary — for high/critical)

Priority routing:
  critical → Telegram + Email immediately
  high     → Telegram immediately (Email if configured)
  medium   → Dashboard notification (Telegram if configured)
  low      → Log only
"""
from __future__ import annotations
import logging
import smtplib
import os
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from typing import Optional

logger = logging.getLogger(__name__)


async def notify_admin(
    title: str,
    message: str,
    priority: str = "medium",
    escalation_id: Optional[str] = None,
) -> None:
    """
    Send admin notification across all configured channels based on priority.
    Fire-and-forget — never raises.
    """
    try:
        await _dispatch(title, message, priority, escalation_id)
    except Exception as e:
        logger.error("[Notifications] Dispatch failed: %s", e)


async def _dispatch(title: str, message: str, priority: str, escalation_id: Optional[str]):
    tasks = []

    if priority in ("critical", "high", "medium"):
        tasks.append(_send_telegram(title, message, priority, escalation_id))
        tasks.append(_send_whatsapp(title, message, priority))

    if priority in ("critical", "high"):
        tasks.append(_send_email(title, message, priority, escalation_id))

    if tasks:
        import asyncio
        await asyncio.gather(*tasks, return_exceptions=True)


async def _send_whatsapp(title: str, message: str, priority: str) -> None:
    """Send via WhatsApp (CallMeBot)."""
    try:
        from app.db import AsyncSessionLocal, Settings
        from sqlalchemy import select

        async with AsyncSessionLocal() as db:
            result = await db.execute(select(Settings).limit(1))
            settings = result.scalar_one_or_none()

        if not settings or not settings.whatsapp_notification_phone or not settings.whatsapp_notification_api_key:
            return

        priority_emoji = {"critical": "🚨", "high": "⚠️", "medium": "ℹ️"}.get(priority, "📢")
        text = f"*{priority_emoji} {title}*\n\n{message}"

        import httpx
        url = "https://api.callmebot.com/whatsapp.php"
        async with httpx.AsyncClient(timeout=10) as client:
            await client.post(url, data={
                "phone": settings.whatsapp_notification_phone,
                "text": text,
                "apikey": settings.whatsapp_notification_api_key
            })
    except Exception as e:
        logger.warning("[Notifications] WhatsApp failed: %s", e)


async def _send_telegram(
    title: str,
    message: str,
    priority: str,
    escalation_id: Optional[str],
) -> None:
    """Send via Telegram Bot."""
    try:
        from app.db import AsyncSessionLocal, Settings
        from sqlalchemy import select

        async with AsyncSessionLocal() as db:
            result = await db.execute(select(Settings).limit(1))
            settings = result.scalar_one_or_none()

        if not settings or not settings.telegram_bot_token or not settings.telegram_chat_id:
            return

        priority_emoji = {"critical": "🔴", "high": "🟠", "medium": "🟡", "low": "⚪"}.get(priority, "📢")
        text = f"{priority_emoji} *{title}*\n\n{message}"
        if escalation_id:
            text += f"\n\nEscalation ID: `{escalation_id}`"

        import httpx
        url = f"https://api.telegram.org/bot{settings.telegram_bot_token}/sendMessage"
        async with httpx.AsyncClient(timeout=10) as client:
            await client.post(url, json={
                "chat_id": settings.telegram_chat_id,
                "text": text,
                "parse_mode": "Markdown",
            })
    except Exception as e:
        logger.warning("[Notifications] Telegram failed: %s", e)


async def _send_email(
    title: str,
    message: str,
    priority: str,
    escalation_id: Optional[str],
) -> None:
    """Send via SMTP email."""
    smtp_host = os.environ.get("SMTP_HOST", "")
    smtp_port = int(os.environ.get("SMTP_PORT", "587"))
    smtp_user = os.environ.get("SMTP_USER", "")
    smtp_password = os.environ.get("SMTP_PASSWORD", "")
    admin_email = os.environ.get("ADMIN_EMAIL", "")

    if not all([smtp_host, smtp_user, smtp_password, admin_email]):
        return  # Email not configured

    try:
        priority_emoji = {"critical": "🔴", "high": "🟠", "medium": "🟡"}.get(priority, "📢")
        subject = f"{priority_emoji} [{priority.upper()}] {title}"

        html_body = f"""
        <html><body style="font-family: Arial, sans-serif; direction: rtl;">
        <div style="background: #1a1a2e; color: #eee; padding: 20px; border-radius: 8px;">
            <h2 style="color: {'#ff4444' if priority == 'critical' else '#ff8800' if priority == 'high' else '#ffcc00'}">
                {priority_emoji} {title}
            </h2>
            <p style="font-size: 16px; line-height: 1.6;">{message}</p>
            {"<p style='color: #888; font-size: 12px;'>Escalation ID: " + escalation_id + "</p>" if escalation_id else ""}
        </div>
        </body></html>
        """

        msg = MIMEMultipart("alternative")
        msg["Subject"] = subject
        msg["From"] = smtp_user
        msg["To"] = admin_email
        msg.attach(MIMEText(html_body, "html"))

        with smtplib.SMTP(smtp_host, smtp_port) as server:
            server.ehlo()
            server.starttls()
            server.login(smtp_user, smtp_password)
            server.sendmail(smtp_user, admin_email, msg.as_string())

        logger.info("[Notifications] Email sent to %s", admin_email)
    except Exception as e:
        logger.warning("[Notifications] Email failed: %s", e)
