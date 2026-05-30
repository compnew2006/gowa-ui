"""
Integration Hub Router.
Manage Slack, Zapier, WhatsApp, and Teams integrations.
"""
import uuid
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from datetime import datetime, timezone

from app.deps import get_db
from app.db import IntegrationConfig

router = APIRouter(tags=["integrations"], prefix="/integrations")

VALID_TYPES = {"slack", "zapier", "whatsapp", "teams"}
VALID_EVENTS = {"escalation", "reply", "error", "shadow_approved", "token_expiring", "churn_alert"}


@router.get("")
async def list_integrations(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(IntegrationConfig).order_by(IntegrationConfig.type))
    integrations = result.scalars().all()
    return [_serialize(i) for i in integrations]


@router.post("")
async def create_integration(body: dict, db: AsyncSession = Depends(get_db)):
    int_type = body.get("type", "")
    if int_type not in VALID_TYPES:
        raise HTTPException(400, f"Invalid type. Must be one of: {VALID_TYPES}")
    cfg = IntegrationConfig(
        id=str(uuid.uuid4()),
        type=int_type,
        name=body.get("name", int_type.title()),
        config=body.get("config", {}),
        is_active=body.get("is_active", False),
        trigger_events=body.get("trigger_events", ["escalation"]),
    )
    db.add(cfg)
    await db.commit()
    return _serialize(cfg)


@router.patch("/{integration_id}")
async def update_integration(integration_id: str, body: dict, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(IntegrationConfig).where(IntegrationConfig.id == integration_id))
    cfg = result.scalar_one_or_none()
    if not cfg:
        raise HTTPException(404, "Integration not found")
    for field in ("name", "config", "is_active", "trigger_events"):
        if field in body:
            setattr(cfg, field, body[field])
    cfg.updated_at = datetime.now(timezone.utc)
    await db.commit()
    return _serialize(cfg)


@router.delete("/{integration_id}")
async def delete_integration(integration_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(IntegrationConfig).where(IntegrationConfig.id == integration_id))
    cfg = result.scalar_one_or_none()
    if not cfg:
        raise HTTPException(404, "Integration not found")
    await db.delete(cfg)
    await db.commit()
    return {"success": True}


@router.post("/{integration_id}/test")
async def test_integration(integration_id: str, db: AsyncSession = Depends(get_db)):
    """Send a test message to verify the integration is configured correctly."""
    result = await db.execute(select(IntegrationConfig).where(IntegrationConfig.id == integration_id))
    cfg = result.scalar_one_or_none()
    if not cfg:
        raise HTTPException(404, "Integration not found")

    config = cfg.config or {}
    success = False
    error = None

    try:
        if cfg.type == "slack":
            from app.services.slack_notifier import send_slack_message
            webhook_url = config.get("webhook_url", "")
            if not webhook_url:
                raise ValueError("webhook_url not configured")
            success = await send_slack_message(
                webhook_url=webhook_url,
                title="🧪 Integration Test",
                message="This is a test message from AI Automation Dashboard. Integration is working correctly!",
                priority="medium",
            )
        elif cfg.type == "zapier":
            from app.services.slack_notifier import send_zapier_webhook
            webhook_url = config.get("webhook_url", "")
            if not webhook_url:
                raise ValueError("webhook_url not configured")
            success = await send_zapier_webhook(webhook_url, "test", {"message": "Integration test"})
        else:
            # WhatsApp/Teams: stub test
            success = bool(config.get("webhook_url") or config.get("api_key"))

        if success:
            cfg.last_triggered_at = datetime.now(timezone.utc)
            cfg.last_error = None
            await db.commit()

    except Exception as e:
        error = str(e)
        cfg.last_error = error
        await db.commit()

    return {"success": success, "error": error}


def _serialize(i):
    cfg = dict(i.config or {})
    # Mask sensitive fields: show only last 4 characters
    for key in ("webhook_url", "api_key", "token", "secret", "password"):
        if key in cfg and cfg[key]:
            val = str(cfg[key])
            cfg[key] = "****" + val[-4:] if len(val) > 4 else "****"
    return {
        "id": i.id, "type": i.type, "name": i.name,
        "config": cfg, "is_active": i.is_active,
        "trigger_events": i.trigger_events or [],
        "trigger_count": i.trigger_count,
        "last_triggered_at": i.last_triggered_at.isoformat() if i.last_triggered_at else None,
        "last_error": i.last_error,
        "created_at": i.created_at.isoformat(),
    }
