"""
Slack & Zapier Integration Notifier.

Sends structured notifications to Slack webhooks and Zapier outbound webhooks
for escalations, replies, and system events.
"""
from __future__ import annotations
import logging
import httpx
from datetime import datetime, timezone
from typing import Optional

logger = logging.getLogger(__name__)


async def send_slack_message(
    webhook_url: str,
    title: str,
    message: str,
    priority: str = "medium",
    fields: Optional[dict] = None,
    escalation_id: Optional[str] = None,
) -> bool:
    """
    Send a formatted Slack block message.
    Returns True on success.
    """
    color = {"critical": "#ef4444", "high": "#f97316", "medium": "#eab308", "low": "#6b7280"}.get(priority, "#6b7280")
    emoji = {"critical": "🔴", "high": "🟠", "medium": "🟡", "low": "⚪"}.get(priority, "📢")

    blocks = [
        {
            "type": "header",
            "text": {"type": "plain_text", "text": f"{emoji} {title}"},
        },
        {
            "type": "section",
            "text": {"type": "mrkdwn", "text": message},
        },
    ]

    if fields:
        field_elements = [
            {"type": "mrkdwn", "text": f"*{k}*\n{v}"}
            for k, v in fields.items()
        ]
        blocks.append({"type": "section", "fields": field_elements[:10]})

    if escalation_id:
        blocks.append({
            "type": "context",
            "elements": [{"type": "mrkdwn", "text": f"Escalation ID: `{escalation_id}`"}],
        })

    payload = {
        "attachments": [{"color": color, "blocks": blocks}]
    }

    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(webhook_url, json=payload)
            if resp.status_code == 200:
                return True
            logger.warning("[Slack] Non-200 response: %s", resp.status_code)
            return False
    except Exception as e:
        logger.warning("[Slack] Send failed: %s", e)
        return False


async def send_zapier_webhook(
    webhook_url: str,
    event_type: str,
    data: dict,
) -> bool:
    """
    Send an outbound Zapier webhook trigger.
    Returns True on success.
    """
    payload = {
        "event": event_type,
        "timestamp": datetime.now(timezone.utc).isoformat(),
        **data,
    }
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(webhook_url, json=payload)
            return resp.status_code in (200, 201)
    except Exception as e:
        logger.warning("[Zapier] Send failed: %s", e)
        return False


async def fire_integration_event(
    db,
    event_type: str,
    data: dict,
    priority: str = "medium",
) -> None:
    """
    Fire an event to all active integrations that listen for this event type.
    Fire-and-forget — never raises.
    """
    try:
        from sqlalchemy import select
        from app.db import IntegrationConfig

        result = await db.execute(
            select(IntegrationConfig).where(
                IntegrationConfig.is_active == True
            )
        )
        integrations = result.scalars().all()

        for integration in integrations:
            if event_type not in (integration.trigger_events or []):
                continue

            cfg = integration.config or {}
            success = False

            if integration.type == "slack" and cfg.get("webhook_url"):
                success = await send_slack_message(
                    webhook_url=cfg["webhook_url"],
                    title=data.get("title", event_type),
                    message=data.get("message", ""),
                    priority=priority,
                    fields=data.get("fields"),
                    escalation_id=data.get("escalation_id"),
                )
            elif integration.type == "zapier" and cfg.get("webhook_url"):
                success = await send_zapier_webhook(
                    webhook_url=cfg["webhook_url"],
                    event_type=event_type,
                    data=data,
                )

            from datetime import datetime, timezone
            integration.last_triggered_at = datetime.now(timezone.utc)
            integration.trigger_count = (integration.trigger_count or 0) + 1
            if not success:
                integration.last_error = f"Failed to deliver {event_type}"
            else:
                integration.last_error = None

        await db.commit()
    except Exception as e:
        logger.warning("[Integrations] Fire event failed: %s", e)
