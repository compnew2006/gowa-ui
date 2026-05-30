"""
Meta Graph API Rate Limit Guard.

Implements Token Bucket algorithm per page_id to prevent hitting Meta's limits:
- Graph API: ~200 calls/hour/user-token (conservative: 180)
- Replies: track separately with Warm-up Strategy support

On threshold breach: enters Exponential Backoff Queue + notifies admin.
Falls back to in-memory bucket when Redis unavailable.
"""
from __future__ import annotations
import time
import math
import logging
from collections import defaultdict
from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

# Config
GRAPH_API_LIMIT_PER_HOUR = 180          # Conservative (actual is ~200)
REPLY_LIMIT_PER_HOUR = 80               # Conservative for replies
WARMUP_MULTIPLIER = 0.3                 # During warmup: only 30% of limit
WARNING_THRESHOLD_PCT = 0.80            # Alert at 80% usage
BUCKET_WINDOW_SECS = 3600              # 1 hour rolling window


# In-memory fallback: {key: [timestamp, ...]}
_memory_buckets: dict[str, list[float]] = defaultdict(list)


def _get_redis():
    try:
        import redis as redis_lib
        r = redis_lib.from_url(settings.redis_url, socket_connect_timeout=1, socket_timeout=1)
        r.ping()
        return r
    except Exception:
        return None


def _get_limit(call_type: str, warmup: bool) -> int:
    base = REPLY_LIMIT_PER_HOUR if call_type == "reply" else GRAPH_API_LIMIT_PER_HOUR
    return max(5, int(base * WARMUP_MULTIPLIER)) if warmup else base


def _check_redis_bucket(r, key: str, limit: int) -> tuple[bool, int, float]:
    """Token bucket via Redis sorted set. Returns (allowed, remaining, usage_pct)."""
    now = time.time()
    cutoff = now - BUCKET_WINDOW_SECS
    pipe = r.pipeline()
    pipe.zremrangebyscore(key, 0, cutoff)
    pipe.zadd(key, {str(now): now})
    pipe.zcard(key)
    pipe.expire(key, BUCKET_WINDOW_SECS + 60)
    results = pipe.execute()
    count = results[2]
    remaining = max(0, limit - count)
    usage_pct = count / limit
    return count <= limit, remaining, usage_pct


def _check_memory_bucket(key: str, limit: int) -> tuple[bool, int, float]:
    """In-memory fallback."""
    now = time.time()
    cutoff = now - BUCKET_WINDOW_SECS
    _memory_buckets[key] = [t for t in _memory_buckets[key] if t > cutoff]
    _memory_buckets[key].append(now)
    count = len(_memory_buckets[key])
    remaining = max(0, limit - count)
    usage_pct = count / limit
    return count <= limit, remaining, usage_pct


async def check_meta_rate_limit(
    page_id: str,
    call_type: str = "api",
    warmup: bool = False,
) -> dict:
    """
    Check and record a Meta API call against the rate limit.

    Returns:
        {
            "allowed": bool,
            "remaining": int,
            "usage_pct": float,
            "backoff_seconds": int  (0 if allowed)
        }
    """
    limit = _get_limit(call_type, warmup)
    key = f"meta_rl:{call_type}:{page_id}"

    r = _get_redis()
    if r:
        allowed, remaining, usage_pct = _check_redis_bucket(r, key, limit)
    else:
        allowed, remaining, usage_pct = _check_memory_bucket(key, limit)

    backoff = 0
    if not allowed:
        # Exponential backoff: doubles with each consecutive blocked call
        backoff_key = f"meta_bo:{call_type}:{page_id}"
        backoff_count = 1
        if r:
            bc = r.incr(backoff_key)
            r.expire(backoff_key, BUCKET_WINDOW_SECS)
            backoff_count = bc
        backoff = min(300, int(30 * math.pow(2, backoff_count - 1)))
        logger.warning(
            "[MetaRateLimit] BLOCKED | page=%s type=%s usage=%.0f%% backoff=%ds",
            page_id, call_type, usage_pct * 100, backoff,
        )
        # Async admin alert (fire-and-forget)
        try:
            from app.services.notifications import notify_admin
            await notify_admin(
                title="⚠️ Meta API Rate Limit Hit",
                message=f"Page {page_id} hit {call_type} rate limit ({usage_pct:.0%} usage). Backoff: {backoff}s",
                priority="high",
            )
        except Exception:
            pass
    elif usage_pct >= WARNING_THRESHOLD_PCT:
        logger.warning(
            "[MetaRateLimit] WARNING | page=%s type=%s usage=%.0f%%",
            page_id, call_type, usage_pct * 100,
        )
        try:
            from app.services.notifications import notify_admin
            await notify_admin(
                title="⚠️ Meta API Rate Limit Warning",
                message=f"Page {page_id} at {usage_pct:.0%} of {call_type} limit ({remaining} calls remaining this hour)",
                priority="medium",
            )
        except Exception:
            pass
    else:
        # Reset backoff counter on successful call
        if r:
            r.delete(f"meta_bo:{call_type}:{page_id}")

    return {
        "allowed": allowed,
        "remaining": remaining,
        "usage_pct": usage_pct,
        "backoff_seconds": backoff,
    }
