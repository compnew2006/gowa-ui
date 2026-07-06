# PRD: Whatsmeow Integration — From Meta Cloud API to WhatsApp Web Protocol

**Version**: 1.0
**Date**: 2026-02-17
**Status**: Draft

---

## 1. Problem Statement

The current Whatomate backend uses the **Meta Cloud API** (`graph.facebook.com`) for all WhatsApp communication. This approach requires:
- A registered **Meta Business Account** with paid API access.
- Each phone number to be **registered through Meta's official onboarding**.
- Webhook verification via Meta's developer portal.

This creates significant friction for users who want to:
- Connect **personal/unofficial WhatsApp numbers** via QR code scan.
- Run **multiple instances** (phones) from a single deployment.
- Avoid per-message or per-conversation fees.

## 2. Proposed Solution

Replace the `pkg/whatsapp` adapter layer with **[whatsmeow](https://github.com/tulir/whatsmeow)**, a Go library implementing the WhatsApp Web multi-device protocol. This enables:

1. **QR Code Authentication**: Users scan a QR code from the UI to link their WhatsApp account.
2. **Multi-Instance Support**: Each "Account" becomes a persistent `whatsmeow.Client` session.
3. **Zero API Cost**: Direct WebSocket connection to WhatsApp servers.
4. **Full Feature Parity**: Text, images, videos, documents, reactions, read receipts.

## 3. Scope

### 3.1 In Scope

| Area | Change |
|:-----|:-------|
| **Backend: `pkg/whatsapp/`** | Replace Meta HTTP client with `whatsmeow.Client` adapter |
| **Backend: `internal/handlers/accounts.go`** | New "Instance" model with QR code pairing flow |
| **Backend: `internal/handlers/messages.go`** | `SendOutgoingMessage` calls `whatsmeow` instead of Meta API |
| **Backend: `internal/models/`** | New `WhatsAppInstance` model replacing `WhatsAppAccount` |
| **Backend: `pkg/whatsmeow/`** | New adapter package wrapping `whatsmeow` library |
| **Backend: `internal/websocket/`** | New `qr_code` event type for streaming QR to frontend |
| **Frontend: Accounts Page** | Replace "Add Account" form with "Create Instance" + QR scanner |
| **Frontend: `api_spec.md`** | Update all endpoint definitions |

### 3.2 Out of Scope (Phase 1)

- WhatsApp Channels / Newsletter support
- Multi-device linking (one phone → multiple sessions)
- Group admin operations (create group, change settings)
- End-to-end encrypted backups
- Template Messages (Meta Cloud API exclusive feature)
- WhatsApp Flows (Meta Cloud API exclusive feature)

> [!WARNING]
> **Breaking Change**: Template Messages and WhatsApp Flows are **Meta Cloud API exclusive** features. They will NOT be available with `whatsmeow`. The UI must gracefully hide these features.

### 3.3 Features Lost vs. Gained

| Lost (Meta-only) | Gained (whatsmeow) |
|:---|:---|
| Template Messages (pre-approved) | QR Code Auth (no Meta account needed) |
| WhatsApp Flows | Zero API cost |
| Business Profile API | Multiple instance support |
| Catalog Sync from Meta | Group message support |
| Webhook Subscriptions | Faster message delivery (direct WS) |
| Analytics API | Contact/Group metadata access |

## 4. Architecture

### 4.1 Current Architecture (Meta Cloud API)

```
Frontend → Go Backend → pkg/whatsapp/client.go → HTTP → graph.facebook.com
                                                          ↓
                                       Meta Webhook → /api/webhook/whatsapp
```

### 4.2 Target Architecture (whatsmeow)

```
Frontend → Go Backend → pkg/whatsmeow/adapter.go → WebSocket → WhatsApp Servers
                ↑                                        ↓
                └──── Event Handler (incoming msgs) ─────┘
                
           ConnectionManager (map[instanceID]*whatsmeow.Client)
                ↑
         PostgreSQL (whatsmeow session store via container/sql)
```

### 4.3 Key Components

1. **`ConnectionManager`**: Singleton that manages all active `whatsmeow.Client` instances. Keyed by `instanceID` (UUID). Handles lifecycle (connect, disconnect, reconnect).

2. **`WhatsmeowAdapter`**: Implements the same interface as the current `pkg/whatsapp.Client` so that `internal/handlers/messages.go` requires **minimal changes**.

3. **`QR Pairing Flow`**: Uses the existing `internal/websocket/` hub to stream QR code data to the frontend in real-time.

4. **`EventRouter`**: Converts `whatsmeow` events (`events.Message`, `events.Receipt`, `events.HistorySync`) into the existing `WebhookMessage` / `ParsedMessage` types for seamless integration.

## 5. Data Model Changes

### 5.1 WhatsAppInstance (replaces WhatsAppAccount)

```go
type WhatsAppInstance struct {
    BaseModel
    OrganizationID  uuid.UUID `gorm:"type:uuid;index;not null"`
    Name            string    `gorm:"size:100;not null"`
    PhoneNumber     string    `gorm:"size:50"`         // Populated after QR scan
    JID             string    `gorm:"size:100;index"`  // WhatsApp JID (user@s.whatsapp.net)
    Status          string    `gorm:"size:20;default:'disconnected'"` // disconnected, connecting, connected, banned
    IsDefault       bool      `gorm:"default:false"`
    AutoReadReceipt bool      `gorm:"default:false"`
    DeviceStore     []byte    `gorm:"type:bytea"`      // Serialized whatsmeow device store
    LastConnectedAt *time.Time
}
```

### 5.2 Migration Strategy
- Keep `whatsapp_accounts` table for rollback safety.
- Create `whatsapp_instances` table.
- Update `contacts.whatsapp_account` → `contacts.instance_id` (FK).
- Update `messages.whatsapp_account` → `messages.instance_id` (FK).

## 6. API Changes

### 6.1 Removed Endpoints

| Endpoint | Reason |
|:---------|:-------|
| `POST /accounts` (with PhoneID/BusinessID/AccessToken) | Replaced by instance creation |
| `POST /accounts/{id}/test` | No Meta credential validation |
| `POST /accounts/{id}/subscribe` | No Meta webhook subscription |
| `GET/PUT /accounts/{id}/business_profile` | Meta-only feature |
| `POST /accounts/{id}/business_profile/photo` | Meta-only feature |
| `POST /templates/sync` | Meta-only feature |
| `POST /templates` (submit to Meta) | Meta-only feature |
| `POST /catalogs/sync` | Meta-only feature |
| `POST /api/webhook/whatsapp` | Replaced by internal event router |

### 6.2 New Endpoints

| Endpoint | Method | Description |
|:---------|:-------|:------------|
| `/instances` | `GET` | List all WhatsApp instances for org |
| `/instances` | `POST` | Create a new instance (name only) |
| `/instances/{id}` | `GET/PUT/DELETE` | CRUD for instance |
| `/instances/{id}/connect` | `POST` | Start QR pairing (streams QR via WebSocket) |
| `/instances/{id}/disconnect` | `POST` | Gracefully disconnect |
| `/instances/{id}/reconnect` | `POST` | Force reconnect |
| `/instances/{id}/status` | `GET` | Connection health check |

### 6.3 Modified Endpoints

| Endpoint | Change |
|:---------|:-------|
| `POST /contacts/{id}/messages` | Uses `whatsmeow` sender instead of Meta API |
| `POST /messages/media` | Uploads via `whatsmeow.Upload()` |
| `GET /media/{id}` | Downloads via `whatsmeow.Download()` |

### 6.4 WebSocket Events (New)

| Event | Direction | Payload |
|:------|:----------|:--------|
| `instance_qr` | Server→Client | `{ instance_id, qr_data }` |
| `instance_status` | Server→Client | `{ instance_id, status, phone_number }` |
| `instance_logged_out` | Server→Client | `{ instance_id, reason }` |

## 7. Frontend Changes

### 7.1 Accounts Page → Instances Page
- Replace "Add Account" form fields (PhoneID, BusinessID, AccessToken) with a simple "Name" input.
- After creation, show a **QR Code** (rendered from `instance_qr` WebSocket event).
- Show connection status badge (🟢 Connected, 🟡 Connecting, 🔴 Disconnected).
- Show phone number (populated after successful QR scan).

### 7.2 Features to Hide
- Template management UI (or mark as "Meta Cloud API only").
- WhatsApp Flows builder.
- Catalog sync button.
- Business Profile editor.

## 8. Risk Assessment

| Risk | Impact | Mitigation |
|:-----|:-------|:-----------|
| WhatsApp bans numbers used with unofficial APIs | High | Rate limiting, human-like delays, educate users |
| `whatsmeow` library breaking changes | Medium | Pin version, wrap in adapter interface |
| Session invalidation (phone offline too long) | Medium | Auto-reconnect logic, user notification |
| No template messages | Medium | Document clearly, offer canned responses as alternative |
| Data migration from old accounts | Low | Keep old table, parallel migration path |

## 9. Success Metrics

- [ ] All existing text/media message types work via `whatsmeow`.
- [ ] QR code pairing completes in < 10 seconds.
- [ ] Reconnection after network drop succeeds automatically.
- [ ] 100k messages/day sustained without memory leak or crash.
- [ ] Frontend hides Meta-only features gracefully.
