"""
JWT Authentication Middleware for FastAPI.
Validates Bearer tokens on all /api/* routes, exempting health/webhook/metrics endpoints.
"""
from __future__ import annotations

import os
from datetime import datetime, timedelta, timezone
from typing import Optional

from fastapi import Depends, HTTPException, Request, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from jose import JWTError, jwt
from pydantic import BaseModel
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse

from app.config import get_settings

settings = get_settings()

# ── Bearer token extractor ──────────────────────────────────────────────────
_bearer_scheme = HTTPBearer(auto_error=False)


class TokenData(BaseModel):
    """Decoded JWT payload exposed to route handlers."""
    sub: str
    role: str = "admin"
    exp: datetime


class JWTAuthMiddleware(BaseHTTPMiddleware):
    """
    Validate JWT Bearer tokens on every ``/api/*`` request.

    Exempt routes (no auth required):
        - ``/api/healthz``
        - ``/api/webhook/meta``
        - ``/api/metrics``
    """

    EXEMPT_PATHS = {"/api/healthz", "/api/webhook/meta", "/api/metrics"}

    async def dispatch(self, request: Request, call_next):
        path = request.url.path

        # Skip auth for non-API or exempt paths
        if not path.startswith("/api") or path in self.EXEMPT_PATHS:
            return await call_next(request)

        # Bypass auth when JWT_SECRET is not configured (dev mode)
        if not settings.jwt_secret:
            return await call_next(request)

        auth_header: Optional[str] = request.headers.get("Authorization")
        if not auth_header or not auth_header.startswith("Bearer "):
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Missing or invalid Authorization header"},
                headers={"WWW-Authenticate": "Bearer"},
            )

        token = auth_header[len("Bearer "):]
        try:
            payload = jwt.decode(
                token,
                settings.jwt_secret,
                algorithms=[settings.jwt_algorithm],
            )
            # Attach decoded info to request state for downstream use
            request.state.token_sub = payload.get("sub", "")
            request.state.token_role = payload.get("role", "admin")
        except JWTError:
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Invalid or expired token"},
                headers={"WWW-Authenticate": "Bearer"},
            )

        return await call_next(request)


# ── FastAPI dependency: get_current_admin ───────────────────────────────────

async def get_current_admin(
    credentials: HTTPAuthorizationCredentials = Depends(_bearer_scheme),
) -> TokenData:
    """
    FastAPI dependency that validates the JWT and returns TokenData.

    Usage in a router::

        @router.get("/protected")
        async def protected(admin: TokenData = Depends(get_current_admin)):
            ...
    """
    if not settings.jwt_secret:
        # Dev-mode fallback: no auth configured
        return TokenData(sub="dev", role="admin", exp=datetime.now(timezone.utc))

    if credentials is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Not authenticated",
            headers={"WWW-Authenticate": "Bearer"},
        )

    try:
        payload = jwt.decode(
            credentials.credentials,
            settings.jwt_secret,
            algorithms=[settings.jwt_algorithm],
        )
        sub: str = payload.get("sub", "")
        role: str = payload.get("role", "admin")
        exp: datetime = datetime.fromtimestamp(payload.get("exp", 0), tz=timezone.utc)

        if not sub:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid token payload",
            )

        return TokenData(sub=sub, role=role, exp=exp)
    except JWTError as exc:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid or expired token: {exc}",
            headers={"WWW-Authenticate": "Bearer"},
        )


# ── Helper: create a JWT (for login endpoints) ─────────────────────────────

def create_access_token(
    subject: str,
    role: str = "admin",
    expires_delta: Optional[timedelta] = None,
) -> str:
    """Create a signed JWT. Used by login/token-issuing endpoints."""
    if not settings.jwt_secret:
        raise RuntimeError("JWT_SECRET is not configured; cannot create tokens")

    expire = datetime.now(timezone.utc) + (
        expires_delta or timedelta(minutes=settings.jwt_expire_minutes)
    )
    payload = {
        "sub": subject,
        "role": role,
        "exp": expire,
        "iat": datetime.now(timezone.utc),
    }
    return jwt.encode(payload, settings.jwt_secret, algorithm=settings.jwt_algorithm)
