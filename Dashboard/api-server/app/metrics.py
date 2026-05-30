"""
Prometheus metrics for the AI Automation API.
Exposes: request counts, AI pipeline durations, escalations, confidence scores, token health.
"""
from __future__ import annotations

try:
    from prometheus_client import (
        Counter, Histogram, Gauge, Summary,
        generate_latest, CONTENT_TYPE_LATEST, CollectorRegistry,
        REGISTRY,
    )
    PROMETHEUS_AVAILABLE = True
except ImportError:
    PROMETHEUS_AVAILABLE = False

if PROMETHEUS_AVAILABLE:
    # HTTP request metrics
    http_requests_total = Counter(
        "http_requests_total",
        "Total HTTP requests",
        ["method", "path", "status_code"],
    )

    http_request_duration_seconds = Histogram(
        "http_request_duration_seconds",
        "HTTP request duration",
        ["method", "path"],
        buckets=[0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0],
    )

    # AI Pipeline metrics
    ai_pipeline_duration_seconds = Histogram(
        "ai_pipeline_duration_seconds",
        "AI pipeline processing duration per comment",
        buckets=[0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0],
    )

    ai_confidence_score = Histogram(
        "ai_confidence_score",
        "Distribution of AI confidence scores",
        buckets=[0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0],
    )

    ai_intent_total = Counter(
        "ai_intent_total",
        "Total intent classifications by intent",
        ["intent"],
    )

    ai_sentiment_total = Counter(
        "ai_sentiment_total",
        "Total sentiment classifications by sentiment",
        ["sentiment"],
    )

    ai_action_total = Counter(
        "ai_action_total",
        "Total pipeline actions taken",
        ["action"],  # reply | escalate
    )

    # Conversation metrics
    conversations_total = Counter(
        "conversations_total",
        "Total conversations processed",
        ["page_id", "platform"],
    )

    escalations_total = Counter(
        "escalations_total",
        "Total escalations created",
        ["priority"],
    )

    # Token health
    token_health_gauge = Gauge(
        "token_health_status",
        "Token health status per page (1=valid, 0.5=expiring, 0=expired)",
        ["page_id", "page_name"],
    )

    # Webhook metrics
    webhook_events_total = Counter(
        "webhook_events_total",
        "Total Meta webhook events received",
        ["event_type", "platform"],
    )

    webhook_duplicates_total = Counter(
        "webhook_duplicates_total",
        "Deduplicated (discarded) webhook events",
    )

    # Active open escalations (gauge)
    open_escalations_gauge = Gauge(
        "open_escalations",
        "Currently open escalations",
    )


def get_metrics_response():
    """Return Prometheus metrics as bytes."""
    if not PROMETHEUS_AVAILABLE:
        return b"# Prometheus not available\n", "text/plain"
    return generate_latest(REGISTRY), CONTENT_TYPE_LATEST


def record_pipeline_result(result: dict):
    """Record AI pipeline metrics after processing."""
    if not PROMETHEUS_AVAILABLE:
        return
    try:
        if result.get("processing_time_ms"):
            ai_pipeline_duration_seconds.observe(result["processing_time_ms"] / 1000)
        if result.get("confidence"):
            ai_confidence_score.observe(result["confidence"])
        if result.get("intent"):
            ai_intent_total.labels(intent=result["intent"]).inc()
        if result.get("sentiment"):
            ai_sentiment_total.labels(sentiment=result["sentiment"]).inc()
        if result.get("action"):
            ai_action_total.labels(action=result["action"]).inc()
    except Exception:
        pass


def record_escalation(priority: str):
    if not PROMETHEUS_AVAILABLE:
        return
    try:
        escalations_total.labels(priority=priority).inc()
    except Exception:
        pass


def record_webhook_event(event_type: str, platform: str, is_duplicate: bool = False):
    if not PROMETHEUS_AVAILABLE:
        return
    try:
        webhook_events_total.labels(event_type=event_type, platform=platform).inc()
        if is_duplicate:
            webhook_duplicates_total.inc()
    except Exception:
        pass
