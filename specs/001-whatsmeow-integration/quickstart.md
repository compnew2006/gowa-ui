# Quickstart: Whatsmeow Integration

**Feature**: `001-whatsmeow-integration` | **Date**: 2026-02-17

## Prerequisites

- Go 1.24+
- PostgreSQL 15+
- A spare phone number with WhatsApp installed (not your primary number)
- Whatomate running locally (`make run` or `go run cmd/server/main.go`)

## Setup

### 1. Add whatsmeow dependency

```bash
cd /Users/noiemany/Downloads/whatomate_GOWA/whatomate
go get go.mau.fi/whatsmeow@latest
go get go.mau.fi/whatsmeow/store/sqlstore@latest
```

### 2. Configure provider in config.toml

```toml
[whatsapp]
provider = "whatsmeow"    # "whatsmeow" or "meta"

[whatsmeow]
rate_limit_min_delay_ms = 1000
rate_limit_max_delay_ms = 3000
queue_timeout_seconds = 300
max_instances_per_org = 5
```

### 3. Run migrations

```bash
# Migrations run automatically on startup via GORM AutoMigrate
make run
```

### 4. Create an instance

```bash
curl -X POST http://localhost:9000/api/instances \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Phone"}'
```

### 5. Connect via QR

1. Open `http://localhost:3000` in browser
2. Navigate to Instances page
3. Click "Connect" on the new instance
4. Scan the QR code with WhatsApp on your spare phone
5. Status should change to "connected" within seconds

### 6. Send a test message

```bash
curl -X POST http://localhost:9000/api/messages \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"contact_id": "<uuid>", "message_type": "text", "content": "Hello from whatsmeow!"}'
```

## Key Files (Post-Implementation)

| File | Purpose |
|:-----|:--------|
| `pkg/provider/interface.go` | MessageProvider interface |
| `pkg/whatsmeow/adapter.go` | whatsmeow adapter implementation |
| `pkg/whatsmeow/manager.go` | Instance connection manager |
| `pkg/whatsmeow/queue.go` | Per-instance message queue |
| `internal/models/instance.go` | WhatsAppInstance GORM model |
| `internal/handlers/instances.go` | Instance CRUD + lifecycle handlers |
| `internal/handlers/config.go` | Config endpoint handler |

## Troubleshooting

| Symptom | Cause | Fix |
|:--------|:------|:----|
| QR code not appearing | WebSocket not connected | Check browser console for WS errors |
| "Session expired" after restart | sqlstore not configured | Verify PostgreSQL connection in config.toml |
| Messages delayed | Rate limiter active | Expected behavior; check `whatsmeow.rate_limit_*` config |
| Instance status "banned" | WhatsApp enforcement | Use a different phone number; review sending patterns |
