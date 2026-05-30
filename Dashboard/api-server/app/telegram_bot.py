"""
Telegram Bot integration for escalation alerts.
Requires TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID in settings table or env vars.
"""
from __future__ import annotations


async def send_escalation_alert(
    customer_name: str,
    comment: str,
    priority: str,
    escalation_id: str,
) -> None:
    from app.db import AsyncSessionLocal, Settings
    from sqlalchemy import select

    async with AsyncSessionLocal() as db:
        result = await db.execute(select(Settings).limit(1))
        settings = result.scalar_one_or_none()

    if not settings:
        return

    bot_token = settings.telegram_bot_token
    chat_id = settings.telegram_chat_id

    if not bot_token or not chat_id:
        return

    emoji = "🚨" if priority in ("critical", "high") else "⚠️"
    preview = comment[:120] + ("..." if len(comment) > 120 else "")
    message = (
        f"{emoji} *Escalation Alert* — {priority.upper()}\n\n"
        f"👤 Customer: {customer_name}\n"
        f"💬 Comment: {preview}\n\n"
        f"🔗 ID: `{escalation_id}`"
    )

    try:
        from telegram import Bot
        bot = Bot(token=bot_token)
        await bot.send_message(
            chat_id=chat_id,
            text=message,
            parse_mode="Markdown",
        )
    except Exception as exc:
        print(f"[Telegram] Failed to send alert: {exc}")
