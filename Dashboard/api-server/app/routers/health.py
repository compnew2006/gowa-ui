from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import text
from app.deps import get_db

router = APIRouter(tags=["health"])


@router.get("/healthz")
async def health_check(db: AsyncSession = Depends(get_db)):
    await db.execute(text("SELECT 1"))
    return {"status": "ok", "version": "3.0.0", "stack": "FastAPI + LangGraph"}


@router.get("/healthz/workers")
async def workers_health():
    """
    Check Celery worker health.
    Returns worker count and queue depth by inspecting the Celery broker.
    """
    try:
        from app.workers.celery_app import celery
        from app.config import get_settings

        settings = get_settings()

        # Inspect active workers via Celery's inspect
        inspect = celery.control.inspect()
        active_queues = inspect.active_queues() or {}
        stats = inspect.stats() or {}

        worker_count = len(stats)
        queue_names = set()
        for worker_queues in active_queues.values():
            for q in worker_queues:
                queue_names.add(q.get("name", "unknown"))

        # Check Redis queue depth
        queue_depth = {}
        try:
            import redis as sync_redis
            r = sync_redis.from_url(settings.redis_url, socket_connect_timeout=2, socket_timeout=2)
            for qname in queue_names:
                depth = r.llen(qname)
                if depth > 0:
                    queue_depth[qname] = depth
            # Also check default Celery queue
            default_depth = r.llen("celery")
            if default_depth > 0:
                queue_depth["celery"] = default_depth
            r.close()
        except Exception:
            pass

        return {
            "status": "ok" if worker_count > 0 else "degraded",
            "worker_count": worker_count,
            "workers": list(stats.keys()),
            "queue_depth": queue_depth,
            "total_queued": sum(queue_depth.values()),
        }
    except Exception as e:
        return {
            "status": "error",
            "worker_count": 0,
            "error": str(e),
        }


@router.get("/healthz/ready")
async def readiness_check(db: AsyncSession = Depends(get_db)):
    """
    Readiness check: DB + Redis + Celery all connected.
    Returns 200 only if all critical dependencies are available.
    """
    checks = {
        "database": False,
        "redis": False,
        "celery_workers": False,
    }
    errors = []

    # 1. Database check
    try:
        await db.execute(text("SELECT 1"))
        checks["database"] = True
    except Exception as e:
        errors.append(f"database: {str(e)[:100]}")

    # 2. Redis check
    try:
        from app.config import get_settings
        settings = get_settings()
        import redis as sync_redis
        r = sync_redis.from_url(settings.redis_url, socket_connect_timeout=2, socket_timeout=2)
        r.ping()
        r.close()
        checks["redis"] = True
    except Exception as e:
        errors.append(f"redis: {str(e)[:100]}")

    # 3. Celery worker check
    try:
        from app.workers.celery_app import celery
        inspect = celery.control.inspect()
        stats = inspect.stats() or {}
        if len(stats) > 0:
            checks["celery_workers"] = True
        else:
            errors.append("celery: no active workers found")
    except Exception as e:
        errors.append(f"celery: {str(e)[:100]}")

    all_ok = all(checks.values())
    return {
        "ready": all_ok,
        "checks": checks,
        "errors": errors if errors else None,
    }
