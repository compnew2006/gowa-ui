# API Contracts: Whatsmeow Integration — Messaging

**Feature**: `001-whatsmeow-integration` | **Date**: 2026-02-17

## Outbound Messages

### POST /api/messages

Send a message via whatsmeow. Uses the same endpoint as existing Meta path. Handler detects active provider from config and routes to correct adapter.

**Request** (text):
```json
{
  "contact_id": "uuid",
  "instance_id": "uuid (optional, uses org default if omitted)",
  "message_type": "text",
  "content": "Hello, how can I help?"
}
```

**Request** (image):
```json
{
  "contact_id": "uuid",
  "message_type": "image",
  "media_url": "/path/to/uploaded/image.jpg",
  "content": "Optional caption"
}
```

**Request** (reply):
```json
{
  "contact_id": "uuid",
  "message_type": "text",
  "content": "Replying to your question...",
  "reply_to_message_id": "uuid"
}
```

**Response** `201 Created`:
```json
{
  "id": "uuid",
  "whatsapp_message_id": "whatsmeow-generated-id",
  "status": "sent",
  "created_at": "2026-02-17T10:00:00Z"
}
```

**Response** (queued, instance disconnected):
```json
{
  "id": "uuid",
  "status": "queued",
  "queue_timeout_at": "2026-02-17T10:05:00Z"
}
```

**Errors**: `400` (invalid type), `404` (contact/instance not found), `409` (instance banned/logged_out), `503` (queue full)

## Inbound Messages (WebSocket)

Incoming messages are pushed to the frontend via WebSocket. No polling endpoint.

### Event: `new_message`

```json
{
  "type": "new_message",
  "data": {
    "id": "uuid",
    "instance_id": "uuid",
    "contact_id": "uuid",
    "direction": "incoming",
    "message_type": "text",
    "content": "Hi, I need help with my order",
    "whatsapp_message_id": "3EB0...",
    "created_at": "2026-02-17T10:01:00Z",
    "contact": {
      "id": "uuid",
      "phone_number": "+5511999999999",
      "profile_name": "John Doe"
    }
  }
}
```

### Event: `message_status_update`

```json
{
  "type": "message_status_update",
  "data": {
    "message_id": "uuid",
    "whatsapp_message_id": "3EB0...",
    "status": "read",
    "timestamp": "2026-02-17T10:02:00Z"
  }
}
```

Status values: `sent` → `delivered` → `read` | `failed`

### Event: `message_reaction`

```json
{
  "type": "message_reaction",
  "data": {
    "message_id": "uuid",
    "whatsapp_message_id": "3EB0...",
    "reaction": "👍",
    "sender": "+5511999999999",
    "timestamp": "2026-02-17T10:03:00Z"
  }
}
```

## Read Receipts

### POST /api/messages/:id/read

Mark a message as read (sends read receipt to WhatsApp contact).

**Response** `200 OK`:
```json
{
  "message": "Read receipt sent"
}
```

## Reactions

### POST /api/messages/:id/react

Send an emoji reaction to a message.

**Request**:
```json
{
  "emoji": "👍"
}
```

**Response** `200 OK`:
```json
{
  "message": "Reaction sent"
}
```

**Errors**: `400` (invalid emoji), `404`
