# Task Checklist — Whatsmeow Integration

## Phase 0: Foundation
- [ ] Add `whatsmeow` Go dependency (`go get go.mau.fi/whatsmeow`)
- [ ] Create `pkg/whatsmeow/types.go` (Instance types, status enums)
- [ ] Create `pkg/whatsmeow/store.go` (SQL session store wrapper)
- [ ] Create `pkg/whatsmeow/connection.go` (ConnectionManager singleton)
- [ ] Create `pkg/whatsmeow/adapter.go` (MessageSender interface implementation)
- [ ] Create `pkg/whatsmeow/events.go` (Event → internal type converter)
- [ ] Create `internal/models/instance.go` (WhatsAppInstance model)
- [ ] Create DB migration for `whatsapp_instances` table
- [ ] Unit tests for `pkg/whatsmeow/` adapter

## Phase 1: Instance Management
- [ ] Create `internal/handlers/instances.go` (CRUD handlers)
- [ ] Register `/instances` routes in `cmd/whatomate/main.go`
- [ ] Wire `ConnectionManager` into `App` struct
- [ ] Implement QR pairing flow (`POST /instances/{id}/connect`)
- [ ] Add `instance_qr` WebSocket event type
- [ ] Add `instance_status` WebSocket event type
- [ ] Implement `POST /instances/{id}/disconnect`
- [ ] Implement `POST /instances/{id}/reconnect`
- [ ] Implement `GET /instances/{id}/status`
- [ ] Auto-reconnect on server startup for active instances

## Phase 2: Message Sending
- [ ] Implement `SendTextMessage` via whatsmeow
- [ ] Implement `SendImageMessage` via whatsmeow (with upload)
- [ ] Implement `SendVideoMessage` via whatsmeow
- [ ] Implement `SendAudioMessage` via whatsmeow
- [ ] Implement `SendDocumentMessage` via whatsmeow
- [ ] Implement `MarkMessageRead` via whatsmeow
- [ ] Implement `SendReaction` via whatsmeow
- [ ] Update `handlers/messages.go` to use new sender
- [ ] Update `handlers/contacts.go` to resolve instance
- [ ] Update campaign worker to use new sender
- [ ] Update chatbot processor to use new sender

## Phase 3: Message Receiving
- [ ] Handle `events.Message` → create incoming `models.Message`
- [ ] Handle `events.Receipt` → update message status
- [ ] Handle media messages (download + store locally)
- [ ] Handle interactive message replies (button/list)
- [ ] Handle reactions (`events.Message` with reaction)
- [ ] Wire events to WebSocket hub (`new_message`, `status_update`)
- [ ] Wire events to webhook dispatcher (outbound webhooks)
- [ ] Handle group messages

## Phase 4: Frontend
- [ ] Create `InstancesView.vue` (replaces AccountsView)
- [ ] Create `QRCodeDialog.vue` (QR pairing UI)
- [ ] Create `InstanceStatusBadge.vue` component
- [ ] Update `services/api.ts` (replace `/accounts` → `/instances`)
- [ ] Update WebSocket handler for new event types
- [ ] Hide Template management UI (or show "Meta-only" banner)
- [ ] Hide Flows builder UI
- [ ] Hide Catalog sync UI
- [ ] Hide Business Profile editor
- [ ] Update sidebar navigation
- [ ] Update `api_spec.md` with new endpoints

## Phase 5: Migration & Cleanup
- [ ] Create data migration script (`accounts → instances`)
- [ ] Update `contacts` table FK (`whatsapp_account → instance_id`)
- [ ] Update `messages` table FK (`whatsapp_account → instance_id`)
- [ ] Archive `pkg/whatsapp/` (rename to `pkg/whatsapp_meta_archived/`)
- [ ] Remove Meta webhook handler from routes
- [ ] Clean up `config.toml` (Meta-specific config)
- [ ] Update Astro docs site
- [ ] Update `CHANGELOG.md`
- [ ] Update `RALPH_MEMORY.md`
- [ ] Final `go vet` and `go test ./...`
