# PLAN: Whatsmeow Integration — Step-by-Step Execution

**Last Updated**: 2026-02-17

---

## Phase 0: Foundation (No Breaking Changes)

### 0.1 Add `whatsmeow` dependency
```bash
go get go.mau.fi/whatsmeow@latest
go get go.mau.fi/whatsmeow/store/sqlstore
```

### 0.2 Create `pkg/whatsmeow/` adapter package
Create a new adapter that wraps `whatsmeow.Client` behind an interface compatible with the existing `pkg/whatsapp.Client` methods. This is the **Strangler Pattern** — the old `pkg/whatsapp/` stays untouched.

**Files to create:**
- `pkg/whatsmeow/adapter.go` — Core adapter implementing send/receive methods
- `pkg/whatsmeow/connection.go` — ConnectionManager (singleton, holds all active clients)
- `pkg/whatsmeow/events.go` — Event handler converting whatsmeow events → internal types
- `pkg/whatsmeow/store.go` — SQL store wrapper for session persistence
- `pkg/whatsmeow/types.go` — Instance-specific types (QR data, status enums)

### 0.3 Create `WhatsAppInstance` model
- Add new model in `internal/models/instance.go`.
- Create migration to add `whatsapp_instances` table.
- DO NOT touch `whatsapp_accounts` table yet.

---

## Phase 1: Instance Management (Backend)

### 1.1 Instance CRUD handler
Create `internal/handlers/instances.go` with:
- `ListInstances`, `CreateInstance`, `GetInstance`, `UpdateInstance`, `DeleteInstance`
- No QR logic yet, just database CRUD.

### 1.2 Connection Manager integration
- Initialize `ConnectionManager` in `cmd/whatomate/main.go`.
- Wire it into the `App` struct in `internal/handlers/app.go`.
- On server startup, auto-connect all instances with `status = "connected"`.

### 1.3 QR Pairing endpoint
- `POST /instances/{id}/connect` → Starts `whatsmeow` login.
- QR code data streamed via WebSocket (`instance_qr` event).
- On successful scan → update `WhatsAppInstance.JID`, `PhoneNumber`, `Status`.

### 1.4 Connection lifecycle endpoints
- `POST /instances/{id}/disconnect` → `client.Disconnect()`
- `POST /instances/{id}/reconnect` → Disconnect + Connect
- `GET /instances/{id}/status` → Return connection state

---

## Phase 2: Message Sending (Backend)

### 2.1 Define `WhatsmeowSender` interface
```go
type MessageSender interface {
    SendTextMessage(ctx, instanceID, phone, text, replyTo) (msgID, error)
    SendImageMessage(ctx, instanceID, phone, data, mime, caption) (msgID, error)
    SendDocumentMessage(ctx, instanceID, phone, data, mime, filename, caption) (msgID, error)
    SendVideoMessage(ctx, instanceID, phone, data, mime, caption) (msgID, error)
    SendAudioMessage(ctx, instanceID, phone, data, mime) (msgID, error)
    MarkMessageRead(ctx, instanceID, msgID) error
}
```

### 2.2 Implement sender in `pkg/whatsmeow/adapter.go`
- Map each method to `whatsmeow.Client.SendMessage()` with appropriate protobuf message types.
- Media: Use `whatsmeow.Upload()` for media before sending.

### 2.3 Switch `handlers/messages.go`
- Replace `a.WhatsApp.SendTextMessage(...)` → `a.Whatsmeow.SendTextMessage(...)`.
- The `SendOutgoingMessage` unified method stays, only the `sendFn` closure changes.

### 2.4 Update `handlers/contacts.go` (send message handler)
- Change `whatsapp_account` references to `instance_id`.
- Resolve default instance per org.

---

## Phase 3: Message Receiving (Backend)

### 3.1 Event router in `pkg/whatsmeow/events.go`
Convert whatsmeow events to existing internal types:
- `events.Message` → Create `models.Message` (incoming) + update `Contact.LastMessageAt`
- `events.Receipt` → Update `models.Message.Status` (delivered/read)
- `events.HistorySync` → Optional: import chat history on first connect

### 3.2 Wire event handler to WebSocket hub
- Incoming messages → `WSHub.BroadcastToOrg()` (existing `new_message` event)
- Status updates → `WSHub.BroadcastToOrg()` (existing `status_update` event)

### 3.3 Media download handling
- Replace Meta CDN download (`client.DownloadMedia`) with `whatsmeow.Download()`.
- Store media locally or in S3 (existing `internal/handlers/media.go` logic).

---

## Phase 4: Frontend Updates

### 4.1 Instances Page (replaces Accounts)
- New component: `InstancesView.vue` (replaces `AccountsView.vue`).
- "Create Instance" → Simple name input → POST `/instances`.
- After creation → "Connect" button → Start QR pairing.
- QR Code rendered using `qrcode.vue` component (use `qrcode` npm package).
- Status badge: Real-time via WebSocket `instance_status` events.

### 4.2 API Client Updates (`services/api.ts`)
- Replace all `/accounts` calls with `/instances`.
- Remove Meta-specific fields (PhoneID, BusinessID, AccessToken).
- Add new methods: `connectInstance()`, `disconnectInstance()`, `getInstanceStatus()`.

### 4.3 Hide Meta-only Features
- Template management → Hide or show "Not available" banner.
- Flows builder → Hide.
- Catalog sync → Hide.
- Business Profile → Hide.
- Conditionally render based on a feature flag or config.

### 4.4 Update `api_spec.md`
- Document all new `/instances` endpoints.
- Remove deprecated Meta-only endpoints.
- Update WebSocket events list.

---

## Phase 5: Migration & Cleanup

### 5.1 Data migration
- Script to migrate `whatsapp_accounts` → `whatsapp_instances` (for existing data).
- Update foreign keys in `contacts` and `messages` tables.

### 5.2 Remove Meta dependencies
- Archive `pkg/whatsapp/` (rename to `pkg/whatsapp_meta_archived/`).
- Remove Meta webhook handler from `cmd/whatomate/main.go`.
- Clean up `config.toml` (remove `[whatsapp]` section with Meta URLs).

### 5.3 Update documentation
- `frontend/docs/` Astro docs site.
- `frontend/api_spec.md`.
- `CHANGELOG.md`.

---

## Verification Plan

### Automated
- `go test ./pkg/whatsmeow/...` — Unit tests for adapter.
- `go test ./internal/handlers/...` — Integration tests for instance CRUD.
- `go build ./cmd/whatomate/` — Ensure clean compilation.

### Manual
- Create instance → Scan QR → Verify connection status in UI.
- Send text message → Verify delivery on phone.
- Receive message → Verify real-time display in UI.
- Send image/video/document → Verify media handling.
- Disconnect → Reconnect → Verify session persistence.
- 1000 messages/minute sustained test.
