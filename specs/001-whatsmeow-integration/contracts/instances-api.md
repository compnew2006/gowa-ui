# API Contracts: Whatsmeow Integration — Instance Management

**Feature**: `001-whatsmeow-integration` | **Date**: 2026-02-17

## Instance CRUD

### POST /api/instances

Create a new WhatsApp instance.

**Request**:
```json
{
  "name": "Sales Phone"
}
```

**Response** `201 Created`:
```json
{
  "id": "uuid",
  "name": "Sales Phone",
  "status": "disconnected",
  "phone_number": null,
  "jid": null,
  "is_default": false,
  "created_at": "2026-02-17T10:00:00Z"
}
```

**Errors**: `400` (name empty/duplicate), `401`, `403`

---

### GET /api/instances

List all instances for the current organization.

**Response** `200 OK`:
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Sales Phone",
      "status": "connected",
      "phone_number": "+5511999999999",
      "jid": "5511999999999@s.whatsapp.net",
      "is_default": true,
      "last_connected_at": "2026-02-17T10:05:00Z",
      "created_at": "2026-02-17T10:00:00Z"
    }
  ]
}
```

---

### GET /api/instances/:id

Get a single instance by ID.

**Response** `200 OK`: Same shape as single item in list.

**Errors**: `404`

---

### PATCH /api/instances/:id

Update instance settings (name, is_default, auto_read_receipt).

**Request**:
```json
{
  "name": "Support Phone",
  "is_default": true,
  "auto_read_receipt": true
}
```

**Response** `200 OK`: Updated instance object.

**Errors**: `400`, `404`

---

### DELETE /api/instances/:id

Soft-delete an instance. Disconnects if connected.

**Response** `204 No Content`

**Errors**: `404`

## Instance Lifecycle

### POST /api/instances/:id/connect

Initiate QR code pairing. Returns immediately. QR code is streamed via WebSocket.

**Response** `200 OK`:
```json
{
  "message": "QR code pairing initiated. Watch WebSocket for qr_code events.",
  "instance_id": "uuid"
}
```

**WebSocket events emitted**:
- `{ "type": "qr_code", "data": { "instance_id": "uuid", "qr": "base64-encoded-qr-string" } }`
- `{ "type": "instance_connected", "data": { "instance_id": "uuid", "phone_number": "+5511999999999", "jid": "...@s.whatsapp.net" } }`
- `{ "type": "instance_qr_timeout", "data": { "instance_id": "uuid" } }`

**Errors**: `404`, `409` (already connected)

---

### POST /api/instances/:id/disconnect

Gracefully disconnect an instance.

**Response** `200 OK`:
```json
{
  "message": "Instance disconnected",
  "instance_id": "uuid",
  "status": "disconnected"
}
```

**WebSocket events emitted**:
- `{ "type": "instance_disconnected", "data": { "instance_id": "uuid" } }`

**Errors**: `404`, `409` (already disconnected)

---

### POST /api/instances/:id/reconnect

Reconnect a previously paired instance without requiring a new QR scan.

**Response** `200 OK`:
```json
{
  "message": "Reconnection initiated",
  "instance_id": "uuid"
}
```

**WebSocket events emitted**:
- `{ "type": "instance_connected", "data": { "instance_id": "uuid", "phone_number": "...", "jid": "..." } }`
- `{ "type": "instance_reconnect_failed", "data": { "instance_id": "uuid", "reason": "session_expired" } }`

**Errors**: `404`, `409` (already connected), `422` (no stored session)

---

### GET /api/instances/:id/health

Get detailed health metrics for a single instance.

**Response** `200 OK`:
```json
{
  "instance_id": "uuid",
  "status": "connected",
  "uptime_seconds": 86400,
  "messages_sent_today": 1523,
  "messages_received_today": 987,
  "messages_failed_today": 3,
  "error_rate_percent": 0.15,
  "last_message_at": "2026-02-17T09:55:00Z",
  "queue_depth": 0
}
```

## Config

### GET /api/config

Returns platform configuration including active WhatsApp provider.

**Response** `200 OK`:
```json
{
  "whatsapp_provider": "whatsmeow",
  "features": {
    "templates": false,
    "flows": false,
    "catalog": false,
    "business_profile": false
  }
}
```

## Notifications

### GET /api/notifications

List undismissed notifications for the current organization.

**Response** `200 OK`:
```json
{
  "data": [
    {
      "id": "uuid",
      "instance_id": "uuid",
      "event_type": "banned",
      "message": "Instance 'Sales Phone' was banned by WhatsApp",
      "is_dismissed": false,
      "created_at": "2026-02-17T08:00:00Z"
    }
  ]
}
```

### PATCH /api/notifications/:id/dismiss

Dismiss a notification.

**Response** `200 OK`

## WebSocket Events (Instance-Related)

All events include `instance_id` for routing. Broadcast scope: organization-wide.

| Event Type | Trigger | Payload |
|:-----------|:--------|:--------|
| `qr_code` | QR generated/refreshed | `{ instance_id, qr }` |
| `instance_connected` | QR scanned or reconnect | `{ instance_id, phone_number, jid }` |
| `instance_disconnected` | User action or network | `{ instance_id, reason }` |
| `instance_banned` | WhatsApp enforcement | `{ instance_id }` |
| `instance_logged_out` | Session expired | `{ instance_id }` |
| `instance_qr_timeout` | QR not scanned in time | `{ instance_id }` |
| `instance_reconnect_failed` | Reconnect attempt failed | `{ instance_id, reason }` |
