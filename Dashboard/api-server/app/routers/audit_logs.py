"""
Audit Logging Router — Track all admin actions
"""
from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, desc
from uuid import uuid4
from datetime import datetime, timezone
from typing import Optional
import json

from app.deps import get_db
from app.db import AuditLog

router = APIRouter(tags=["audit"], prefix="/audit-logs")


async def log_action(
    db: AsyncSession,
    action: str,
    entity_type: str,
    entity_id: Optional[str] = None,
    admin_id: Optional[str] = None,
    admin_name: Optional[str] = None,
    page_id: Optional[str] = None,
    old_values: Optional[dict] = None,
    new_values: Optional[dict] = None,
    reason: Optional[str] = None,
    details: Optional[str] = None,
    ip_address: Optional[str] = None,
):
    """Log an admin action to audit trail."""
    log_entry = AuditLog(
        id=str(uuid4()),
        action=action,
        entity_type=entity_type,
        entity_id=entity_id,
        admin_id=admin_id,
        admin_name=admin_name,
        page_id=page_id,
        old_values=old_values,
        new_values=new_values,
        reason=reason,
        details=details,
        ip_address=ip_address,
        created_at=datetime.now(timezone.utc),
    )
    try:
        db.add(log_entry)
        await db.flush()  # Don't commit, parent transaction handles it
    except Exception as e:
        import logging
        logging.warning(f"[Audit] Logging failed: {e}")


@router.get("")
async def list_audit_logs(
    page: int = Query(1, ge=1),
    limit: int = Query(50, ge=1, le=500),
    page_id: Optional[str] = None,
    admin_id: Optional[str] = None,
    action: Optional[str] = None,
    entity_type: Optional[str] = None,
    entity_id: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    """List audit logs with filters."""
    query = select(AuditLog)

    if page_id:
        query = query.where(AuditLog.page_id == page_id)
    if admin_id:
        query = query.where(AuditLog.admin_id == admin_id)
    if action:
        query = query.where(AuditLog.action == action)
    if entity_type:
        query = query.where(AuditLog.entity_type == entity_type)
    if entity_id:
        query = query.where(AuditLog.entity_id == entity_id)

    # Count total
    count_query = select(__import__("sqlalchemy").func.count()).select_from(query.froms[0])
    for clause in query.whereclause.clauses if query.whereclause else []:
        count_query = count_query.where(clause)
    total = (await db.execute(count_query)).scalar_one() or 0

    # Fetch paginated
    query = (
        query.order_by(desc(AuditLog.created_at))
        .offset((page - 1) * limit)
        .limit(limit)
    )

    result = await db.execute(query)
    logs = result.scalars().all()

    return {
        "total": total,
        "page": page,
        "limit": limit,
        "data": [
            {
                "id": log.id,
                "action": log.action,
                "entity_type": log.entity_type,
                "entity_id": log.entity_id,
                "admin_id": log.admin_id,
                "admin_name": log.admin_name,
                "page_id": log.page_id,
                "old_values": log.old_values,
                "new_values": log.new_values,
                "reason": log.reason,
                "details": log.details,
                "ip_address": log.ip_address,
                "created_at": log.created_at.isoformat() if log.created_at else None,
            }
            for log in logs
        ],
    }


@router.get("/stats")
async def audit_stats(
    page_id: Optional[str] = None,
    days: int = Query(7, ge=1, le=90),
    db: AsyncSession = Depends(get_db),
):
    """Get audit statistics for a period."""
    from datetime import timedelta
    
    query = select(
        AuditLog.action,
        AuditLog.admin_id,
        AuditLog.admin_name,
        __import__("sqlalchemy").func.count().label("count"),
    )

    start_date = datetime.now(timezone.utc) - timedelta(days=days)
    query = query.where(AuditLog.created_at >= start_date)

    if page_id:
        query = query.where(AuditLog.page_id == page_id)

    query = query.group_by(AuditLog.action, AuditLog.admin_id, AuditLog.admin_name)

    result = await db.execute(query)
    rows = result.all()

    # Group by action
    by_action = {}
    by_admin = {}

    for row in rows:
        action, admin_id, admin_name, count = row
        if action not in by_action:
            by_action[action] = 0
        by_action[action] += count

        if admin_id:
            if admin_id not in by_admin:
                by_admin[admin_id] = {"name": admin_name, "actions": {}}
            if action not in by_admin[admin_id]["actions"]:
                by_admin[admin_id]["actions"][action] = 0
            by_admin[admin_id]["actions"][action] += count

    return {
        "period_days": days,
        "by_action": by_action,
        "by_admin": by_admin,
    }
