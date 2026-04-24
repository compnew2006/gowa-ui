# WebSocket Protocol Specification: Whatomate Platform

This document details the WebSocket protocol used by Whatomate for real-time communication, including message handling, authentication, and event schemas.

---

## 1. Connection & Authentication

### Endpoint (URL)
- **URL:** `ws://<domain>/ws` (or `wss://` in production).
- **Handshake subprotocol:** `whm.v1`.

### Token Acquisition
Before connecting, the client must obtain a short-lived (30s) JWT token:
- **Request:** `GET /api/auth/ws-token`
- **Response:** `{"token": "JWT_TOKEN_HERE"}`

### Handshake Authentication
The platform supports two methods for passing the token during the WebSocket upgrade:
1.  **Direct Header:** `Authorization: Bearer <token>`
2.  **Subprotocol String:** `Sec-WebSocket-Protocol: whm.v1, auth.<token>`

### Mandatory Post-Handshake Auth
After the TCP connection is established, the client **must** send an authentication message within **5 seconds**. Failure to do so results in the server closing the connection.
```json
{
  "type": "auth",
  "payload": { "token": "JWT_TOKEN_HERE" }
}
```

---

## 2. Message Schema
All WebSocket messages follow a unified JSON structure:
```typescript
interface WSMessage {
  type: string;
  payload: any;
}
```

### Client → Server Events
| Event (`type`) | Payload | Description |
| :--- | :--- | :--- |
| `auth` | `{"token": string}` | Required first message to authenticate. |
| `set_contact` | `{"contact_id": UUID}` | Subscribes the client to updates for a specific chat room. |
| `ping` | `{}` | Keepalive heartbeat. |

### Server → Client Events (Real-time Updates)
| Event (`type`) | Description | Data Context |
| :--- | :--- | :--- |
| `new_message` | Triggered on incoming/outgoing WhatsApp messages. | Contact & Message details. |
| `status_update` | Update on message status (sent, delivered, read, failed). | Message UUID & Status. |
| `contact_update` | Triggered when a contact is assigned, updated, or status changed. | Contact profile. |
| `campaign_stats_update` | Real-time counters for active bulk campaigns. | Campaign UUID & counters. |
| `instance_qr_code` | Raw string for QR code generation during instance pairing. | QR string. |
| `instance_connected` | Notification when an instance successfully connects to WhatsApp. | Instance ID & Phone. |
| `reaction_update` | Emoji reaction added or removed from a message. | Message ID & Reaction list. |
| `agent_transfer` | Incoming chat transfer request for the current user. | Transfer details. |
| `notification` | Generic system-wide or organization-wide notification. | Text & Severity. |

---

## 3. Room & Context Management

The server manages traffic isolation based on the authenticated session:
1.  **Organization Level:** All clients are grouped by `organization_id`. Broadcasts using `BroadcastToOrg` are visible to all authenticated users of that org.
2.  **User Level:** Events like `agent_transfer` utilize `BroadcastToUser` to target a specific employee across all their open tabs.
3.  **Contact Level:** When a client sends a `set_contact` event, the server tracks their current viewing context. Events like `reaction_update` or typing status use `BroadcastToContact` to update only relevant viewers.

---

## 4. Frontend Resilience (Reconnection)

The `WebSocketService` implementation in the frontend handles reliability:
- **Heartbeat:** Sends a `ping` every 30 seconds.
- **Reconnection Logic:**
    - Max Retries: 10.
    - Strategy: Exponential Backoff (1s, 2s, 4s, 8s...).
- **Data Synchronization:** Upon successful reconnection, the application triggers a state refresh (`refreshStaleData`) to pull any missed messages via REST.

---

## 5. Implementation Mapping

| Component | Code Reference |
| :--- | :--- |
| **Route Definition** | `cmd/whatomate/main.go` |
| **Hub Logic** | `internal/websocket/hub.go` |
| **Message Handlers** | `internal/handlers/websocket.go` & `internal/websocket/client.go` |
| **Frontend Service** | `frontend/src/services/websocket.ts` |
