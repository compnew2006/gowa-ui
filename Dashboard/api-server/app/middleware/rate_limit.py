"""
Redis-based sliding window rate limiter + comment deduplication middleware.
Falls back to in-memory limiting when Redis is unavailable.
"""
from __future__ import annotations
import time
import hashlib
from collections import defaultdict
from typing import Callable
from fastapi import Request, Response
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from app.config import get_settings

settings = get_settings()

# In-memory fallback: {key: [(timestamp, count)]}
_memory_windows: dict[str, list[float]] = defaultdict(list)


def _get_redis():
    try:
        import redis
        r = redis.from_url(settings.redis_url, socket_connect_timeout=1, socket_timeout=1)
        r.ping()
        return r
    except Exception:
        return None


def _rate_limit_redis(r, key: str, limit: int, window_secs: int = 60) -> tuple[bool, int]:
    """Sliding window rate limit using Redis. Returns (allowed, remaining)."""
    now = time.time()
    pipe = r.pipeline()
    pipe.zremrangebyscore(key, 0, now - window_secs)
    pipe.zadd(key, {str(now): now})
    pipe.zcard(key)
    pipe.expire(key, window_secs + 1)
    results = pipe.execute()
    count = results[2]
    remaining = max(0, limit - count)
    return count <= limit, remaining


def _rate_limit_memory(key: str, limit: int, window_secs: int = 60) -> tuple[bool, int]:
    """In-memory sliding window fallback."""
    now = time.time()
    cutoff = now - window_secs
    timestamps = _memory_windows[key]
    _memory_windows[key] = [t for t in timestamps if t > cutoff]
    _memory_windows[key].append(now)
    count = len(_memory_windows[key])
    remaining = max(0, limit - count)
    return count <= limit, remaining


def check_rate_limit(identifier: str, limit: int, prefix: str = "rl") -> tuple[bool, int]:
    key = f"{prefix}:{identifier}"
    r = _get_redis()
    if r:
        return _rate_limit_redis(r, key, limit)
    return _rate_limit_memory(key, limit)


def is_duplicate_webhook(comment_id: str, page_id: str, ttl_secs: int = 300) -> bool:
    """Check if this webhook event was already processed (deduplication)."""
    key = f"dedup:webhook:{page_id}:{comment_id}"
    r = _get_redis()
    if r:
        result = r.set(key, "1", ex=ttl_secs, nx=True)
        return result is None  # None = key already existed = duplicate
    return False  # No dedup without Redis


class RateLimitMiddleware(BaseHTTPMiddleware):
    """Apply rate limiting to all API routes."""

    EXEMPT_PATHS = {"/api/healthz", "/api/metrics", "/api/webhook/meta"}

    MUTATION_METHODS = {"POST", "PATCH", "DELETE"}

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        path = request.url.path

        if path in self.EXEMPT_PATHS or not path.startswith("/api"):
            return await call_next(request)

        # Identify client: prefer forwarded IP, fall back to client host
        client_ip = (
            request.headers.get("X-Forwarded-For", "").split(",")[0].strip()
            or request.headers.get("X-Real-IP", "")
            or (request.client.host if request.client else "unknown")
        )

        # Stricter rate limit for mutation endpoints
        if request.method in self.MUTATION_METHODS:
            limit = settings.rate_limit_mutation_rpm
        else:
            limit = settings.rate_limit_rpm

        allowed, remaining = check_rate_limit(client_ip, limit)

        if not allowed:
            return JSONResponse(
                status_code=429,
                content={"detail": "Rate limit exceeded. Please slow down."},
                headers={
                    "X-RateLimit-Limit": str(limit),
                    "X-RateLimit-Remaining": "0",
                    "Retry-After": "60",
                },
            )

        response = await call_next(request)
        response.headers["X-RateLimit-Limit"] = str(limit)
        response.headers["X-RateLimit-Remaining"] = str(remaining)
        return response
