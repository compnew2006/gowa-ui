---
title: Webhook Integration
---

# Webhook Integration

Whatomate provides an outbound webhook system that notifies external services about events occurring within the platform.

## Overview

When events occur (messages sent/received, contacts created, campaigns updated, etc.), Whatomate dispatches HTTP POST requests to configured webhook URLs. Each webhook subscription specifies which events to receive.

## Event Types

| Event | Trigger |
|-------|---------|
| `message.received` | Inbound message received |
| `message.sent` | Outbound message sent |
| `message.delivered` | Message delivery confirmed |
| `message.read` | Message read by recipient |
| `message.failed` | Message send failed |
| `message.status_updated` | Any message status change |
| `contact.created` | New contact created |
| `contact.updated` | Contact details changed |
| `contact.assigned` | Contact assigned to user |
| `contact.reassigned` | Contact reassigned to different user |
| `chat.closed` | Chat session closed |
| `chat.reopened` | Chat session reopened |
| `campaign.started` | Campaign execution started |
| `campaign.paused` | Campaign paused |
| `campaign.completed` | Campaign finished |
| `campaign.cancelled` | Campaign cancelled |
| `user.created` | New user added to organization |
| `user.updated` | User details changed |
| `lead_request.created` | New lead request submitted |
| `sla.breached` | SLA threshold exceeded |

## Payload Format

All webhook payloads follow a consistent structure:

```json
{
  "event": "message.received",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 123,
    "contact_id": 456,
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

### Payload by Event Type

#### message.received

```json
{
  "event": "message.received",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 123,
    "contact_id": 456,
    "contact_name": "John Doe",
    "contact_phone": "+1234567890",
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "account_id": 1,
    "instance_id": null
  }
}
```

#### message.sent

```json
{
  "event": "message.sent",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 124,
    "contact_id": 456,
    "content": "Hi there!",
    "direction": "outbound",
    "status": "sent",
    "provider_message_id": "wamid.HBg...",
    "sent_by": {
      "id": 5,
      "name": "Agent Smith"
    }
  }
}
```

#### contact.created

```json
{
  "event": "contact.created",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 789,
    "phone_number": "+1234567890",
    "name": "New Contact",
    "status": "open",
    "tags": []
  }
}
```

#### campaign.started

```json
{
  "event": "campaign.started",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 10,
    "name": "Holiday Promotion",
    "total_recipients": 500,
    "template_id": 5,
    "started_at": "2026-01-01T00:00:00Z"
  }
}
```

## HMAC Signature Verification

Webhooks can be configured with a secret for payload verification. Whatomate signs each request with HMAC-SHA256:

```
X-Webhook-Signature: sha256=<hex-hmac>
```

### Verification Example (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(`sha256=${expected}`)
  );
}

// Usage in Express
app.post('/webhook', (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const rawBody = JSON.stringify(req.body);
  
  if (!verifyWebhook(rawBody, signature, process.env.WEBHOOK_SECRET)) {
    return res.status(401).send('Invalid signature');
  }
  
  // Process webhook
  handleEvent(req.body.event, req.body.data);
  res.status(200).send('OK');
});
```

### Verification Example (Go)

```go
func verifyWebhook(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := fmt.Sprintf("sha256=%x", mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

## Delivery Attempts

Whatomate attempts webhook delivery with the following behavior:

| Attempt | Delay | Notes |
|---------|-------|-------|
| 1 | Immediate | Synchronous dispatch |
| 2 | 30 seconds | If first attempt fails |
| 3 | 5 minutes | If second attempt fails |
| 4 | 30 minutes | If third attempt fails |
| 5 | 2 hours | Final attempt |

After 5 failed attempts, the webhook is marked as degraded and no further attempts are made until manually re-enabled.

### Response Requirements

- Webhook endpoint must respond with `2xx` status code
- Response timeout: 10 seconds
- Non-2xx responses count as failures

## Test Webhook

You can test a webhook configuration through the API:

```
POST /api/webhooks/{id}/test
```

**Response (200):**
```json
{
  "success": true,
  "status_code": 200,
  "response_time_ms": 145,
  "message": "Webhook delivered successfully"
}
```

**Response (400):**
```json
{
  "success": false,
  "status_code": 500,
  "response_time_ms": 5023,
  "message": "Webhook delivery failed: Internal Server Error"
}
```

## Webhook Management API

### List Webhooks

```
GET /api/webhooks
```

**Response (200):**
```json
{
  "webhooks": [
    {
      "id": 1,
      "url": "https://example.com/webhook",
      "events": ["message.received", "message.sent"],
      "enabled": true,
      "last_triggered": "2026-01-01T00:00:00Z",
      "last_status": "success",
      "created_at": "2025-06-01T00:00:00Z"
    }
  ]
}
```

### Create Webhook

```
POST /api/webhooks
```

**Request Body:**
```json
{
  "url": "https://example.com/webhook",
  "events": ["message.received", "message.sent", "contact.created"],
  "secret": "your-hmac-secret",
  "enabled": true
}
```

### Update Webhook

```
PUT /api/webhooks/{id}
```

### Delete Webhook

```
DELETE /api/webhooks/{id}
```

## Dispatch Implementation

Webhook dispatch is handled asynchronously:

```go
func (app *App) DispatchWebhook(event string, data interface{}) {
    // Find enabled webhooks for this event
    var webhooks []models.Webhook
    app.DB.Where("enabled = ? AND events @> ?", true, pq.Array([]string{event})).
        Find(&webhooks)
    
    for _, wh := range webhooks {
        go app.sendWebhook(wh, event, data)
    }
}

func (app *App) sendWebhook(wh models.Webhook, event string, data interface{}) {
    payload := WebhookPayload{
        Event:          event,
        Timestamp:      time.Now().UTC(),
        OrganizationID: wh.OrganizationID,
        Data:           data,
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Sign with HMAC if secret configured
    if wh.Secret != "" {
        secret := decrypt(wh.Secret)
        mac := hmac.New(sha256.New, []byte(secret))
        mac.Write(body)
        req.Header.Set("X-Webhook-Signature", fmt.Sprintf("sha256=%x", mac.Sum(nil)))
    }
    
    // Send with retry logic
    app.sendWithRetry(req, body)
}
```

## Security Considerations

1. **Always verify HMAC signatures** — Never process webhooks without verification in production
2. **Use HTTPS endpoints** — Webhook URLs should use TLS
3. **Validate payload structure** — Check event types and required fields
4. **Implement idempotency** — Use the payload timestamp and event data to detect duplicates
5. **Respond quickly** — Process asynchronously and return 200 immediately

## See Also

- [API Reference](./api-reference) — Webhook endpoints
- [Architecture](./architecture) — Data flow diagrams
