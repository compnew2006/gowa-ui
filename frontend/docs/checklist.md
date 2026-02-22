# Pre-Flight Checklist — Whatsmeow Integration

## ✅ Before Starting (Prerequisites)
- [ ] Go 1.21+ installed
- [ ] PostgreSQL running with existing Whatomate database
- [ ] Redis running (for queue and caching)
- [ ] A test WhatsApp number (NOT your primary number)
- [ ] `whatsmeow` library version pinned in `go.mod`
- [ ] Read `RALPH_MEMORY.md` for past integration traps

## 🛡️ Code Quality Gates (Per Phase)
- [ ] `go build ./cmd/whatomate/` compiles cleanly
- [ ] `go vet ./...` no warnings
- [ ] `go test ./...` all pass
- [ ] No new lint errors introduced
- [ ] Blast radius documented for each modified file

## 🔄 Integration Test Scenarios
- [ ] **QR Pairing**: Create instance → Connect → Scan QR → Status = "connected"
- [ ] **Text Send**: Send text → Delivered on phone → Status = "sent"
- [ ] **Text Receive**: Receive text → Appears in UI in real-time
- [ ] **Image Send**: Upload image → Send → Delivered with caption
- [ ] **Image Receive**: Receive image → Media downloaded → Viewable in UI
- [ ] **Document Send/Receive**: PDF, DOCX handled correctly
- [ ] **Video/Audio**: Media handled with correct MIME types
- [ ] **Reactions**: Send/receive emoji reactions
- [ ] **Read Receipts**: Mark as read → Blue ticks on phone
- [ ] **Reply Context**: Reply to specific message → Quote shown
- [ ] **Group Messages**: Send/receive in group chats
- [ ] **Reconnection**: Kill server → Restart → Instance auto-reconnects
- [ ] **Session Persistence**: Restart server → No QR re-scan needed
- [ ] **Multi-Instance**: 3+ instances connected simultaneously

## 🚫 Regression Checks
- [ ] Auth (login/logout/refresh) still works
- [ ] Organizations (create/switch) still works
- [ ] Contacts CRUD still works
- [ ] Tags/Teams/Roles still work
- [ ] Canned Responses still work
- [ ] Campaigns still send (via new sender)
- [ ] Chatbot keyword matching still works
- [ ] Agent transfers still work
- [ ] Import/Export still works
- [ ] Custom Actions still work
- [ ] Widgets/Analytics still work
- [ ] SSO still works
- [ ] API Keys still work

## 📊 Performance Benchmarks
- [ ] Single instance: 100 messages/minute sustained
- [ ] Single instance: 1000 messages/day without memory growth
- [ ] 5 instances: All connected simultaneously, stable
- [ ] Media: 50MB file upload/download without timeout
- [ ] Memory: Instance uses < 50MB RAM after 24h uptime

## 📝 Documentation
- [ ] `api_spec.md` updated with `/instances` endpoints
- [ ] `CHANGELOG.md` entry added
- [ ] `RALPH_MEMORY.md` entry added
- [ ] Astro docs site updated (accounts → instances)
- [ ] README updated with new setup instructions
