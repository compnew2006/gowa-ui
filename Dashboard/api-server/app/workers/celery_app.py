"""
Celery configuration with Priority Queue routing.

Queues:
  high   → Angry/Refund/Legal (Escalations, immediate)
  normal → Purchase/Price inquiry
  low    → General comments, compliments

Dead Letter Queue:
  After max_retries exhausted → dead_letter queue + admin alert
"""
from celery import Celery
from celery.schedules import crontab
from app.config import get_settings

settings = get_settings()

celery = Celery(
    "ai_automation",
    broker=settings.redis_url,
    backend=settings.redis_url,
    include=["app.workers.tasks"],
)

celery.conf.update(
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="UTC",
    enable_utc=True,
    task_track_started=True,
    task_acks_late=True,
    worker_prefetch_multiplier=1,

    # Priority queues
    task_queues={
        "high": {
            "exchange": "high",
            "routing_key": "high",
            "queue_arguments": {"x-max-priority": 10},
        },
        "normal": {
            "exchange": "normal",
            "routing_key": "normal",
            "queue_arguments": {"x-max-priority": 5},
        },
        "low": {
            "exchange": "low",
            "routing_key": "low",
            "queue_arguments": {"x-max-priority": 1},
        },
        "dead_letter": {
            "exchange": "dead_letter",
            "routing_key": "dead_letter",
        },
    },

    # Default routing
    task_default_queue="normal",
    task_default_exchange="normal",
    task_default_routing_key="normal",

    # Task routes
    task_routes={
        "app.workers.tasks.process_webhook_event_high": {"queue": "high"},
        "app.workers.tasks.process_webhook_event": {"queue": "normal"},
        "app.workers.tasks.process_webhook_event_low": {"queue": "low"},
        "app.workers.tasks.handle_dead_letter": {"queue": "dead_letter"},
        "app.workers.tasks.send_telegram_alert_task": {"queue": "high"},
        "app.workers.tasks.refresh_single_token": {"queue": "normal"},
        "app.workers.tasks.refresh_expired_tokens": {"queue": "normal"},
        "app.workers.tasks.cleanup_old_data": {"queue": "low"},
        "app.workers.tasks.optimize_prompts_weekly": {"queue": "low"},
        "app.workers.tasks.check_meta_rate_limits": {"queue": "normal"},
        "app.workers.tasks.poll_whatsapp_messages": {"queue": "high"},
    },


    # Retry config per queue
    task_annotations={
        "app.workers.tasks.process_webhook_event_high": {
            "rate_limit": "100/m",
            "max_retries": 3,
        },
        "app.workers.tasks.process_webhook_event": {
            "rate_limit": "200/m",
            "max_retries": 3,
        },
    },
)

# ─────────────────────────────────────────────
# Beat Schedule
# ─────────────────────────────────────────────
celery.conf.beat_schedule = {
    # Token management: check every 30 min, refresh 15 days before expiry
    "refresh-expiring-tokens": {
        "task": "app.workers.tasks.refresh_expired_tokens",
        "schedule": crontab(minute="*/30"),
    },
    # Token refresh: full cycle every 45 days (staggered via daily check)
    "check-token-health-daily": {
        "task": "app.workers.tasks.check_token_health",
        "schedule": crontab(hour=8, minute=0),  # Daily 8am UTC
    },
    # Meta rate limit usage report (daily)
    "check-meta-rate-limits": {
        "task": "app.workers.tasks.check_meta_rate_limits",
        "schedule": crontab(hour="*/6", minute=0),  # Every 6 hours
    },
    # Data cleanup (daily 2am)
    "cleanup-old-conversations": {
        "task": "app.workers.tasks.cleanup_old_data",
        "schedule": crontab(hour=2, minute=0),
    },
    # DSPy prompt optimization (weekly Sunday 3am UTC)
    "optimize-prompts-weekly": {
        "task": "app.workers.tasks.optimize_prompts_weekly",
        "schedule": crontab(day_of_week=0, hour=3, minute=0),
    },
    # Update open escalation gauge for Prometheus
    "update-escalation-metrics": {
        "task": "app.workers.tasks.update_escalation_metrics",
        "schedule": crontab(minute="*/5"),  # Every 5 minutes
    },
    # Poll WhatsApp bridges every 5 seconds
    'poll-whatsapp-messages': {
        'task': 'app.workers.tasks.poll_whatsapp_messages',
        'schedule': 30.0,  # Every 30 seconds
    },
    'process-scheduled-posts': {
        'task': 'app.workers.tasks.process_scheduled_posts',
        'schedule': 60.0,  # Every minute
    },
}
