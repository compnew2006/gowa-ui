---
title: WebSocket Events
---

# WebSocket Events

Real-time communication in Whatomate is handled via WebSocket connections. This page documents the connection flow, hub operations, and all message types.

## Connection Flow

### 1. Obtain WebSocket Token

Before connecting, request a short-lived JWT token:

```
GET /api/auth/ws-token
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 30
}
```

The token is a JWT with:
- `subject`: "ws"
- `user_id`: authenticated user ID
- `org_id`: current organization ID
- `exp`: 30 seconds from issuance

### 2. Establish WebSocket Connection

```
GET /ws?token=<ws-token>
```

**Connection Process:**
1. Server validates the JWT token
2. Extracts `user_id` and `org_id` from claims
3. Upgrades HTTP to WebSocket
4. Registers connection with the Hub
5. Starts read and write loops

### 3. Connection Lifecycle

```
Client                    Server
  │                         │
  │─── GET /ws?token=... ──▶│
  │◀─── 101 Switching ─────│
  │                         │
  │◀─── {type: "message",   │
  │      payload: {...}} ───│
  │                         │
  │─── {ping} ─────────────▶│
  │◀─── {pong} ────────────│
  │                         │
  │─── Close ──────────────▶│
  │                         │ Hub.Unregister()
```

## Hub Operations

The WebSocket Hub (`internal/websocket/hub.go`) manages connections:

```go
type Hub struct {
    // orgConnections maps organization ID to set of connections
    orgConnections map[int64]map[*Connection]bool
    mu             sync.RWMutex
}

// Register adds a connection to the hub
func (h *Hub) Register(conn *Connection)

// Unregister removes a connection
func (h *Hub) Unregister(conn *Connection)

// BroadcastToOrg sends a message to all connections in an organization
func (h *Hub) BroadcastToOrg(orgID int64, msg *WSMessage)
```

### Connection Structure

```go
type Connection struct {
    UserID  int64
    OrgID   int64
    Conn    *websocket.Conn
    Send    chan []byte  // Buffered channel for outgoing messages
}
```

## Message Types

All WebSocket messages follow this format:

```json
{
  "type": "message_type",
  "payload": { ... },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 1. message

A new message was received or sent.

```json
{
  "type": "message",
  "payload": {
    "id": 123,
    "contact_id": 456,
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "status": "delivered",
    "created_at": "2026-01-01T00:00:00Z",
    "sender": {
      "id": 456,
      "name": "John Doe",
      "phone": "+1234567890"
    }
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 2. message_status

A message status was updated.

```json
{
  "type": "message_status",
  "payload": {
    "message_id": 123,
    "status": "read",
    "updated_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**Status Values:** `pending`, `sent`, `delivered`, `read`, `failed`

### 3. contact_created

A new contact was created.

```json
{
  "type": "contact_created",
  "payload": {
    "id": 789,
    "phone_number": "+1234567890",
    "name": "New Contact",
    "status": "open",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 4. contact_assigned

A contact was assigned to a user.

```json
{
  "type": "contact_assigned",
  "payload": {
    "contact_id": 789,
    "assigned_to": {
      "id": 5,
      "name": "Agent Smith"
    },
    "assigned_by": {
      "id": 1,
      "name": "Admin User"
    }
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 5. chat_closed

A chat was closed.

```json
{
  "type": "chat_closed",
  "payload": {
    "contact_id": 789,
    "closed_by": {
      "id": 5,
      "name": "Agent Smith"
    },
    "closed_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 6. chat_reopened

A closed chat was reopened.

```json
{
  "type": "chat_reopened",
  "payload": {
    "contact_id": 789,
    "reopened_by": {
      "id": 5,
      "name": "Agent Smith"
    },
    "reopened_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 7. campaign_stats_update

Campaign progress statistics updated.

```json
{
  "type": "campaign_stats_update",
  "payload": {
    "campaign_id": 10,
    "total_recipients": 500,
    "sent": 150,
    "delivered": 120,
    "read": 80,
    "failed": 5,
    "pending": 345,
    "status": "running"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 8. instance_status

WhatsApp instance connection status changed.

```json
{
  "type": "instance_status",
  "payload": {
    "instance_id": 3,
    "name": "support-instance",
    "status": "connected",
    "phone_number": "+1234567890",
    "queue_depth": 5
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**Status Values:** `disconnected`, `connecting`, `qr_ready`, `connected`, `reconnecting`, `reconnect_failed`

### 9. notification

A new notification for the user.

```json
{
  "type": "notification",
  "payload": {
    "id": 42,
    "type": "sla_breach",
    "message": "Chat #789 has exceeded response SLA",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 10. typing

Typing indicator for a contact.

```json
{
  "type": "typing",
  "payload": {
    "contact_id": 789,
    "user_id": 5,
    "state": "composing"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**State Values:** `composing`, `paused`

### 11. presence

Contact presence update.

```json
{
  "type": "presence",
  "payload": {
    "contact_id": 789,
    "status": "online",
    "last_seen": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 12. instance_reconnect_failed

Instance failed to reconnect after multiple attempts.

```json
{
  "type": "instance_reconnect_failed",
  "payload": {
    "instance_id": 3,
    "name": "support-instance",
    "error": "connection timeout after 5 retries",
    "last_attempt": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

## Client-Side Usage

### JavaScript Example

```javascript
// 1. Get WebSocket token
const tokenResp = await fetch('/api/auth/ws-token');
const { token } = await tokenResp.json();

// 2. Connect
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

// 3. Handle messages
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  switch (msg.type) {
    case 'message':
      appendMessage(msg.payload);
      break;
    case 'message_status':
      updateMessageStatus(msg.payload);
      break;
    case 'contact_assigned':
      showAssignmentNotification(msg.payload);
      break;
    case 'campaign_stats_update':
      updateCampaignProgress(msg.payload);
      break;
    case 'instance_status':
      updateInstanceIndicator(msg.payload);
      break;
    case 'typing':
      showTypingIndicator(msg.payload);
      break;
    case 'notification':
      showNotification(msg.payload);
      break;
  }
};

// 4. Handle reconnection
ws.onclose = () => {
  setTimeout(connect, 3000);
};
```

## Broadcasting

Messages are broadcast through the Hub:

```go
// Broadcast to entire organization
app.Hub.BroadcastToOrg(orgID, &WSMessage{
    Type: "message",
    Payload: messageData,
    Timestamp: time.Now().UTC(),
})

// Broadcast from Redis pub/sub (campaign stats)
// Handled by CampaignStatsSubscriber
```

## See Also

- [Architecture](./architecture)
- [API Reference](./api-reference) — WebSocket token endpoint
- [Background Workers](./background-workers) — Campaign stats subscriber
