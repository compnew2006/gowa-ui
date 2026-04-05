---
title: Monitoring
---

# Monitoring

Whatomate provides built-in health checks, logging, and metrics for operational monitoring.

## Health Endpoints

### Health Check

```
GET /health
```

Returns basic service availability status:

```json
{
  "status": "ok",
  "service": "whatomate"
}
```

This endpoint does not check dependencies — it only confirms the HTTP server is running.

### Readiness Check

```
GET /ready
```

Verifies all dependencies are available:

```json
{
  "status": "ready"
}
```

Or on failure (HTTP 500):

```json
{
  "status": "not ready",
  "error": "database connection failed"
}
```

The readiness check performs:
1. **Database ping** — Verifies PostgreSQL connectivity
2. **Redis ping** — Verifies Redis connectivity

## Request Logging

Every HTTP request is logged with:

| Field | Description |
|-------|-------------|
| Method | HTTP method (GET, POST, etc.) |
| Path | Request path |
| Remote Address | Client IP address |
| Status Code | Response status |
| Duration | Request processing time |
| Request ID | Unique trace ID |
| User ID | Authenticated user (if applicable) |

Example log output:

```
2024-01-01T12:00:00Z INFO request method=POST path=/api/contacts/123/messages status=201 duration=45ms request_id=abc123 user_id=5
```

## Activity Logs

Significant actions are recorded in the `activity_logs` table:

| Field | Description |
|-------|-------------|
| user_id | Who performed the action |
| action | Action type (create, update, delete, login, etc.) |
| resource | Resource type (user, contact, campaign, etc.) |
| resource_id | Affected resource ID |
| details | Additional context (JSON) |
| created_at | Timestamp |

View activity logs via API:

```bash
curl https://whatomate.example.com/api/activity-logs \
  -H "Authorization: Bearer <token>"
```

Activity logs are retained for 90 days by default, with automatic cleanup by the retention worker.

## API Logs

API request/response logs include:

- Request body (for debugging)
- Response body (for error investigation)
- Authentication status
- Permission check results

Logs are written to stdout/stderr and can be captured by your logging infrastructure.

## WebSocket Connection Monitoring

The WebSocket Hub tracks active connections:

| Metric | Description |
|--------|-------------|
| Total connections | Active WebSocket connections |
| Connections per org | Connections grouped by organization |
| Connection duration | How long each connection has been active |
| Disconnect reasons | Why connections were closed |

Connection events are logged:

```
2024-01-01T12:00:00Z INFO websocket connected user_id=5 org_id=1
2024-01-01T12:05:00Z INFO websocket disconnected user_id=5 reason="client closed"
```

## Instance Health Metrics

WhatsApp instance health is tracked and available via API:

```bash
curl https://whatomate.example.com/api/instances/{id}/health \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "instance_id": 1,
  "status": "connected",
  "uptime": "24h15m",
  "messages_sent_today": 1250,
  "messages_received_today": 890,
  "messages_failed_today": 3,
  "error_rate": "0.24%",
  "queue_depth": 12
}
```

| Metric | Description |
|--------|-------------|
| `status` | Connection status (disconnected, connecting, connected, qr, paired) |
| `uptime` | Time since last successful connection |
| `messages_sent_today` | Outbound messages sent |
| `messages_received_today` | Inbound messages received |
| `messages_failed_today` | Failed send attempts |
| `error_rate` | Percentage of failed messages |
| `queue_depth` | Current message queue depth |

## Campaign Statistics

Campaign progress is tracked in real-time:

```bash
curl https://whatomate.example.com/api/campaigns/{id} \
  -H "Authorization: Bearer <token>"
```

Response includes:

```json
{
  "id": 1,
  "name": "Welcome Campaign",
  "status": "running",
  "total_recipients": 500,
  "sent_count": 245,
  "failed_count": 3,
  "delivered_count": 240,
  "read_count": 180,
  "progress_percentage": 49,
  "started_at": "2024-01-01T10:00:00Z"
}
```

Campaign stats are broadcast via WebSocket in real-time as workers process recipients.

## Background Process Monitoring

The following background processes run continuously:

| Process | Interval | Purpose | Monitoring |
|---------|----------|---------|------------|
| SLA Processor | 1 minute | Check SLA breaches, auto-close chats | Log entries on breach |
| Activity Retention | 1 hour | Delete old activity logs | Log entries on cleanup |
| Chat Assignment Reset | 1 minute | Reset stale assignments | Log entries on reset |
| Instance Auto-Campaign | 1 minute | Send automated messages | Log entries on send |
| Campaign Worker | Continuous | Process campaign queue | Queue depth, throughput |
| Inbound Media Worker | Continuous | Download inbound media | Queue depth |
| Campaign Stats Subscriber | Continuous | Broadcast campaign stats via WS | Connection status |
| WhatsMeow Reconnect | Startup | Reconnect all instances | Instance status |
| Status Reconciliation | Startup (30s timeout) | Clean stale instance statuses | Log entries |

Monitor background processes via logs:

```bash
# Filter for SLA breaches
docker compose logs whatomate | grep "SLA"

# Filter for campaign processing
docker compose logs whatomate | grep "campaign"

# Filter for instance connections
docker compose logs whatomate | grep "instance"
```

## Monitoring Integrations

### Prometheus Metrics (Future)

Export metrics in Prometheus format for external monitoring:

- Request count by endpoint
- Request duration histograms
- Active WebSocket connections
- Campaign queue depth
- Instance connection status
- Error rates

### Log Aggregation

Forward logs to your log aggregation system:

```bash
# Docker logging driver
docker compose up --log-driver fluentd

# Or pipe to external service
docker compose logs -f whatomate | fluent-cat whatomate.logs
```

## Alerting Recommendations

Set up alerts for:

| Condition | Severity | Action |
|-----------|----------|--------|
| `/ready` returns 500 | Critical | Check database/Redis connectivity |
| Instance disconnected > 5 minutes | High | Investigate WhatsApp connection |
| Campaign error rate > 5% | High | Check template/account status |
| Queue depth > 1000 | Warning | Scale workers or investigate backlog |
| Disk space < 10% | Warning | Clean up media storage |
| Memory usage > 90% | Warning | Investigate memory leaks |

## See Also

- [Health & Readiness Checks](deployment.md#health-and-readiness-endpoints) — Endpoint details
- [Troubleshooting](troubleshooting.md) — Diagnosing common issues
- [Deployment](deployment.md) — Production setup with monitoring
