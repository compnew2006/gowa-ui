"""
Team Collaboration Router.
Manage admin users, roles, assignments, and audit log.
"""
import uuid
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func
from datetime import datetime, timezone
from typing import Optional

from app.deps import get_db
from app.db import AdminUser, AuditLog

router = APIRouter(tags=["teams"], prefix="/teams")

VALID_ROLES = {"admin", "reviewer", "analyst"}


@router.get("")
async def list_team(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(AdminUser).order_by(AdminUser.name))
    members = result.scalars().all()
    return [_serialize_user(m) for m in members]


@router.post("")
async def create_member(body: dict, db: AsyncSession = Depends(get_db)):
    role = body.get("role", "reviewer")
    if role not in VALID_ROLES:
        raise HTTPException(400, f"Invalid role. Must be one of: {VALID_ROLES}")
    existing = await db.execute(select(AdminUser).where(AdminUser.email == body.get("email", "")))
    if existing.scalar_one_or_none():
        raise HTTPException(400, "Email already exists")
    member = AdminUser(
        id=str(uuid.uuid4()),
        email=body.get("email", ""),
        name=body.get("name", ""),
        role=role,
        is_active=body.get("is_active", True),
        avatar_url=body.get("avatar_url"),
        telegram_user_id=body.get("telegram_user_id"),
        permissions=_default_permissions(role),
    )
    db.add(member)
    await _log(db, "system", "create_admin", "admin_user", member.id, {"name": member.name, "role": role})
    await db.commit()
    return _serialize_user(member)


@router.patch("/{member_id}")
async def update_member(member_id: str, body: dict, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(AdminUser).where(AdminUser.id == member_id))
    member = result.scalar_one_or_none()
    if not member:
        raise HTTPException(404, "Team member not found")
    for field in ("name", "role", "is_active", "avatar_url", "telegram_user_id"):
        if field in body:
            setattr(member, field, body[field])
    if "role" in body:
        member.permissions = _default_permissions(body["role"])
    member.updated_at = datetime.now(timezone.utc)
    await _log(db, "system", "update_admin", "admin_user", member_id, body)
    await db.commit()
    return _serialize_user(member)


@router.delete("/{member_id}")
async def delete_member(member_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(AdminUser).where(AdminUser.id == member_id))
    member = result.scalar_one_or_none()
    if not member:
        raise HTTPException(404, "Team member not found")
    await db.delete(member)
    await _log(db, "system", "delete_admin", "admin_user", member_id, {"name": member.name})
    await db.commit()
    return {"success": True}


@router.get("/audit-log")
async def get_audit_log(
    page: int = Query(1, ge=1),
    limit: int = Query(50, le=200),
    entity_type: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    q = select(AuditLog).order_by(AuditLog.created_at.desc())
    if entity_type:
        q = q.where(AuditLog.entity_type == entity_type)
    total = (await db.execute(select(func.count()).select_from(q.subquery()))).scalar_one()
    q = q.offset((page - 1) * limit).limit(limit)
    rows = (await db.execute(q)).scalars().all()
    return {
        "total": total,
        "page": page,
        "data": [
            {
                "id": r.id,
                "admin_name": r.admin_name,
                "action": r.action,
                "entity_type": r.entity_type,
                "entity_id": r.entity_id,
                "details": r.details,
                "created_at": r.created_at.isoformat(),
            }
            for r in rows
        ],
    }


def _serialize_user(m):
    return {
        "id": m.id, "email": m.email, "name": m.name, "role": m.role,
        "is_active": m.is_active, "avatar_url": m.avatar_url,
        "telegram_user_id": m.telegram_user_id,
        "permissions": m.permissions, "last_active_at": m.last_active_at.isoformat() if m.last_active_at else None,
        "created_at": m.created_at.isoformat(),
    }


def _default_permissions(role: str) -> dict:
    if role == "admin":
        return {"can_approve": True, "can_reject": True, "can_delete": True, "can_manage_team": True, "can_export": True}
    elif role == "reviewer":
        return {"can_approve": True, "can_reject": True, "can_delete": False, "can_manage_team": False, "can_export": False}
    return {"can_approve": False, "can_reject": False, "can_delete": False, "can_manage_team": False, "can_export": True}


async def _log(db, admin_name: str, action: str, entity_type: str, entity_id: str, details: dict):
    log = AuditLog(
        id=str(uuid.uuid4()),
        admin_name=admin_name,
        action=action,
        entity_type=entity_type,
        entity_id=entity_id,
        details=details,
    )
    db.add(log)
