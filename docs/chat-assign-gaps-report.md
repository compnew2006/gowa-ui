# Chat & Assignment System — Comprehensive Gaps Report

> **Project**: whatomate  
> **Date**: 2026-05-08  
> **Scope**: Entire chat feature (inbox, conversations, messages, WebSocket, handlers, UI) + All assignment types (chat assignment, agent transfers, campaigns, RBAC, teams, instances)  
> **Analysis Method**: 5 specialized agents (backend, frontend, assignment, security, workflow tracer) using Serena MCP, codebase-memory-mcp, and ruflo MCP  

---

## Executive Summary

| Category | CRITICAL | HIGH | MEDIUM | LOW | Total |
|----------|----------|------|--------|-----|-------|
| Security | 1 | 4 | 4 | 1 | 10 |
| Performance | 0 | 3 | 4 | 2 | 9 |
| Architecture | 2 | 5 | 8 | 3 | 18 |
| Missing Features | 3 | 6 | 8 | 6 | 23 |
| Assignment System | 1 | 3 | 6 | 5 | 15 |
| Workflow Gaps | 1 | 4 | 5 | 2 | 12 |
| **Total** | **8** | **25** | **35** | **19** | **87** |

### Top 10 Priority Fixes

| # | ID | Severity | Summary |
|---|-----|----------|---------|
| 1 | SEC-001 | CRITICAL | WebSocket token replay via Sec-WebSocket-Protocol header |
| 2 | C1 | CRITICAL | WebSocket broadcast silently drops messages on full buffer |
| 3 | SEC-002 | HIGH | No rate limiting on message sending endpoints |
| 4 | SEC-004 | HIGH | No input sanitization on chat message content (XSS risk) |
| 5 | SEC-005 | HIGH | No upload size limits on media messages (DoS risk) |
| 6 | FE-01 | CRITICAL | ChatView.vue is 6,193 lines — unmaintainable mega-component |
| 7 | FE-02 | CRITICAL | 79% of chat i18n missing in Spanish (174 keys untranslated) |
| 8 | ASGN-01 | HIGH | Transfer expiration cleanup never runs — stale transfers accumulate |
| 9 | ASGN-03 | HIGH | Race condition in CreateAgentTransfer — duplicate transfers possible |
| 10 | WF-01 | HIGH | No dead-letter queue for inbound messages on DB failure |

---

## 1. Architecture Gaps

### ARCH-01 [CRITICAL] — ChatView.vue is a 6,193-line mega-component
**File**: `frontend/src/views/chat/ChatView.vue`  
**Impact**: Impossible to maintain, test, or reason about. Any change risks regressions.  
**Details**: Contains sidebar logic, message rendering, media handling, typing presence, account switching, transfer handling, custom actions, canned responses, profile photos, batch printing — all in one file.  
**Recommendation**: Decompose into ~15-20 focused components:
- `ChatSidebar.vue` — contact list, tabs, search
- `ChatMessageList.vue` — message rendering with virtual scrolling
- `ChatMessageBubble.vue` — individual message (incoming/outgoing variants)
- `ChatInputBar.vue` — message composition, media upload, reply quote
- `ChatHeader.vue` — contact info, actions, status
- `ChatTransferDialog.vue` — transfer-to-agent flow
- `ChatAssignmentDialog.vue` — assign-to-agent flow
- `ChatCannedResponses.vue` — canned response picker
- `ChatTypingIndicator.vue` — typing presence display
- `ChatEmptyState.vue` — no conversation selected

### ARCH-02 [CRITICAL] — Contacts store is 1,422 lines doing too much
**File**: `frontend/src/stores/contacts.ts`  
**Impact**: Violates single-responsibility; hard to test individual concerns.  
**Recommendation**: Split into:
- `useContactsStore` — contact CRUD, pagination, filtering
- `useMessagesStore` — message list, send, status updates
- `useChatFiltersStore` — search, tags, instance filter, status filter

### ARCH-03 [HIGH] — processIncomingMessageFull is 1000+ lines
**File**: `internal/handlers/chatbot_processor.go:113`  
**Impact**: Monolithic function handling contact creation, message persistence, chatbot evaluation, SLA tracking, close ratings, and WebSocket broadcast. Extremely hard to test.  
**Recommendation**: Extract into focused functions:
- `persistIncomingMessage()`
- `evaluateChatbotRules()`
- `trackChatSLA()`
- `processCloseRatings()`
- `notifyNewMessage()`

### ARCH-04 [HIGH] — No message search endpoint
**File**: `cmd/whatomate/server.go`  
**Impact**: Users cannot search within conversations or across contacts.  
**Recommendation**: Add `GET /api/contacts/{id}/messages/search?q=` with PostgreSQL full-text search or `LIKE` with trigram index.

### ARCH-05 [HIGH] — No cursor-based pagination for messages
**File**: `internal/handlers/messages.go`  
**Impact**: Offset pagination degrades on large message tables. Virtual scrolling impossible without cursors.  
**Recommendation**: Implement cursor-based pagination using `created_at + message_id` as cursor.

### ARCH-06 [HIGH] — N+1 query in ListContacts
**File**: `internal/handlers/contacts.go:370+`  
**Impact**: With 500 contacts per page, hundreds of extra queries for per-contact hydration (assigned user name, closed-by name, avatar, conversation context).  
**Recommendation**: Use batch queries or subquery joins. Preload all related users in a single query.

### ARCH-07 [HIGH] — Missing composite database indexes
**File**: `internal/models/models.go`  
**Impact**: Full index scans on hottest query paths.  
**Missing indexes**:
```sql
CREATE INDEX idx_contacts_org_status_lastmsg ON contacts (organization_id, status, last_message_at DESC NULLS LAST);
CREATE INDEX idx_messages_org_contact_created ON messages (organization_id, contact_id, created_at DESC);
CREATE INDEX idx_contacts_org_assigned ON contacts (organization_id, assigned_user_id) WHERE assigned_user_id IS NOT NULL;
```

### ARCH-08 [HIGH] — ContactInfoPanel is 963 lines
**File**: `frontend/src/components/chat/ContactInfoPanel.vue`  
**Impact**: Second mega-component, should be split.  
**Recommendation**: Extract assignment panel, collaborator panel, tags panel, session data panel, and notes panel as separate components.

### ARCH-09 [MEDIUM] — Dead WebSocket message types
**File**: `internal/websocket/messages.go`  
**Details**: `TypeAgentTransfer`, `TypeAgentTransferResume`, `TypeAgentTransferAssign`, `TypePermissionsUpdated`, `StatusUpdatePayload` are defined but some have no sender. `NewUnauthenticatedClient()` exists but is unused since WS auth moved to HTTP handshake.  
**Recommendation**: Remove dead code or implement the intended functionality.

### ARCH-10 [MEDIUM] — Inconsistent route naming
**File**: `cmd/whatomate/server.go`  
**Details**: `/api/chats` and `/api/contacts` are aliased to the same handler. `/api/chat/sessions` is a legacy alias for `/api/chatbot/sessions`.  
**Recommendation**: Standardize on `/api/contacts` for CRUD and `/api/chats` for lifecycle (claim, close, reopen). Remove legacy aliases.

### ARCH-11 [MEDIUM] — Group chat metadata uses raw JSONB
**File**: `internal/handlers/contacts.go`  
**Details**: Group/channel detection relies on `metadata->>'is_group_chat'` JSONB keys with no typed model fields. Fragile.  
**Recommendation**: Add typed fields to the `Contact` model: `IsGroupChat bool`, `GroupName string`, `GroupParticipantCount int`.

### ARCH-12 [MEDIUM] — No caching layer for contact/chat data
**Details**: Every ListContacts hit queries PostgreSQL. Redis is only used for rate limiting, CSRF, and pub/sub.  
**Recommendation**: Cache contact list data in Redis with short TTL (5-10s), invalidated on contact update.

### ARCH-13 [MEDIUM] — Offset pagination degrades at scale
**File**: `internal/handlers/contacts.go`  
**Details**: Uses `parsePaginationWithDefaults` (offset-based). With high message volumes, offset pagination becomes slow on large tables.  
**Recommendation**: Implement keyset/cursor pagination for inbox.

### ARCH-14 [MEDIUM] — resolveContactConversationContext queries DB per-contact
**File**: `internal/handlers/contacts.go:300-400`  
**Details**: Runs `SELECT ... ORDER BY created_at DESC LIMIT 1` on every inbound message.  
**Recommendation**: Cache context or derive from contact record.

### ARCH-15 [MEDIUM] — Chat close rating context builds 4 extra queries
**File**: `internal/handlers/chat_close_ratings.go:buildChatCloseRatingContext`  
**Details**: Two separate queries for "before" and "after" messages, each with LIMIT 2.  
**Recommendation**: Combine into a single query with window functions.

### ARCH-16 [LOW] — Magic strings for system_event metadata
**File**: `internal/handlers/chat_system_messages.go:11`  
**Recommendation**: Define as typed constants.

### ARCH-17 [LOW] — No message edit endpoint
**File**: `cmd/whatomate/server.go`  
**Recommendation**: Add `PUT /api/messages/{id}` for editing sent messages within a time window.

### ARCH-18 [LOW] — No message forwarding endpoint
**File**: `cmd/whatomate/server.go`  
**Recommendation**: Add `POST /api/messages/{id}/forward` for forwarding messages.

---

## 2. Security Gaps

### SEC-001 [CRITICAL] — WebSocket Token Replay via Sec-WebSocket-Protocol Header
**OWASP**: A07:2021 — Identification and Authentication Failures  
**File**: `internal/handlers/websocket.go:110-125`  
**Description**: Tokens can be extracted from `Sec-WebSocket-Protocol` header (visible in browser dev tools). When Redis is unavailable, replay protection is bypassed (line 188 logs warning but continues).  
**Impact**: Session hijacking if Redis is unavailable.  
**Fix**: Remove Sec-WebSocket-Protocol token path; hard-fail when Redis unavailable.

### SEC-002 [HIGH] — No Rate Limiting on Message Sending
**OWASP**: A04:2021 — Insecure Design  
**File**: `internal/handlers/contacts_messaging.go:61`  
**Description**: `SendMessage` endpoint has no rate limiting. An abusive agent could flood contacts, risking WhatsApp API bans.  
**Fix**: Apply per-user rate limiting using `KeyFunc` based on orgID + userID.

### SEC-003 [HIGH] — API Key Brute-Force via Timing Side-Channel
**OWASP**: A07:2021 — Identification and Authentication Failures  
**File**: `internal/middleware/middleware.go:376-424`  
**Description**: No rate limiting on failed API key attempts. All active keys matching prefix are iterated with bcrypt.  
**Fix**: Add rate limiting for API key auth failures per key prefix.

### SEC-004 [HIGH] — No Input Sanitization on Chat Message Content
**OWASP**: A03:2021 — Injection  
**File**: `internal/handlers/contacts_messaging.go:134-141`, `internal/handlers/messages.go:549-628`  
**Description**: Message content stored and broadcast without HTML/sanitization. Risk for consuming systems (webhooks, integrations).  
**Fix**: Sanitize at input with `bluemonday`. Encode in webhook payloads.

### SEC-005 [HIGH] — Missing Upload Size Limits on Media Messages
**OWASP**: A04:2021 — Insecure Design  
**File**: `internal/handlers/media.go`  
**Description**: No `MaxBytesReader` on media upload handlers. Arbitrary file sizes can be uploaded.  
**Fix**: Apply `MaxBytesReader` with configurable limit (20MB default).

### SEC-006 [MEDIUM] — JWT Secret Has No Rotation Mechanism
**OWASP**: A07:2021 — Identification and Authentication Failures  
**File**: `internal/handlers/jwt_secret.go:8-24`  
**Fix**: Support list of valid secrets during rotation period.

### SEC-007 [MEDIUM] — Contact Assignment Without Ownership Verification
**OWASP**: A01:2021 — Broken Access Control  
**File**: `internal/handlers/contacts_management.go:182-238`  
**Description**: Any user with `contacts:write` can reassign ANY contact. No check that user has relationship to the contact.  
**Fix**: Require current assignee, collaborator, or admin/manager role for the contact's team.

### SEC-008 [MEDIUM] — CSRF Protection Bypassed by API Key
**OWASP**: A01:2021 — Broken Access Control  
**File**: `internal/middleware/csrf.go:23-25`  
**Description**: CSRF entirely bypassed when `Authorization` or `X-API-Key` present.  
**Fix**: Enforce `SameSite=Strict` on cookies; validate CORS origin on API-key requests.

### SEC-009 [MEDIUM] — No Rate Limiting on Chat Lifecycle Endpoints
**File**: `cmd/whatomate/server.go:471-474`  
**Description**: `ClaimChat`, `CloseChat`, `ReopenChat`, `SetChatPublic` have no rate limiting. Spam claim/release can manipulate assignment.  
**Fix**: Apply rate limiting to all chat lifecycle endpoints.

### SEC-010 [LOW] — SQL WHERE Clause Uses Standard String Comparison for Key Prefix
**File**: `internal/middleware/middleware.go:384`  
**Impact**: Minor timing difference; low severity given bcrypt overhead dominates.

---

## 3. Performance Gaps

### PER-001 [HIGH] — WebSocket Broadcast Channel Buffer Undersized
**File**: `internal/websocket/hub.go:36`  
**Description**: Buffer of 256 messages. During burst activity, messages silently dropped.  
**Fix**: Increase to 4096+. Implement per-user broadcast channels.

### PER-002 [HIGH] — N+1 in Transfer List with Queue Counts
**File**: `internal/handlers/agent_transfers.go:264-351`  
**Description**: 5+ separate DB queries per ListAgentTransfers request.  
**Fix**: Use CTE/window function to combine. Cache queue counts in Redis with 5-15s TTL.

### PER-003 [HIGH] — No Virtual Scrolling for Message List
**File**: `frontend/src/views/chat/ChatView.vue`  
**Description**: All messages rendered in DOM. Long conversations cause memory pressure and slow rendering.  
**Fix**: Implement cursor-based pagination + virtual scrolling (e.g., `vue-virtual-scroller`).

### PER-004 [MEDIUM] — Broadcast to All Org Users Instead of Targeted
**File**: `internal/handlers/messages.go:830-833`  
**Description**: `BroadcastToOrg` sends every message to ALL connected clients. Agents receive messages for contacts they cannot access.  
**Fix**: Use `BroadcastToContact` for new messages; implement room/subscription model.

### PER-005 [MEDIUM] — Missing Composite Index on Messages Table
**File**: `internal/models/models.go:431-438`  
**Fix**: `CREATE INDEX idx_messages_org_contact_created ON messages (organization_id, contact_id, created_at DESC)`

### PER-006 [MEDIUM] — Transfer Queue Count Queries Without Caching
**File**: `internal/handlers/agent_transfers.go:304-351`  
**Fix**: Cache in Redis with short TTL, invalidated on transfer create/assign/resume.

### PER-007 [MEDIUM] — Client Send Buffer (256) Causes Silent Message Loss
**File**: `internal/websocket/client.go:73`  
**Fix**: Larger buffer or priority queue that drops older messages. Add "missed messages" notification.

### PER-008 [LOW] — Per-User DB Query in WebSocket Contact Subscription
**File**: `internal/handlers/websocket.go:127-158`  
**Fix**: Cache contact access decisions per user/org session.

### PER-009 [LOW] — BroadcastToUsers Sequential Channel Sends
**File**: `internal/websocket/hub.go:211-214`  
**Fix**: Batch user-targeted broadcasts.

---

## 4. Assignment System Gaps

### ASGN-01 [HIGH] — Transfer Expiration Cleanup Never Runs
**File**: `internal/models/constants.go:130`  
**Description**: `TransferStatusExpired` constant exists but no worker/cron processes expired transfers.  
**Impact**: Stale transfers accumulate forever, polluting queue views.  
**Fix**: Add periodic worker to expire transfers past SLA deadline.

### ASGN-02 [HIGH] — Round-Robin LastAssignedAt Not Updated on Transfer Pick
**File**: `internal/handlers/agent_transfers.go:968` vs `1480`  
**Description**: `PickNextTransfer` does NOT update `TeamMember.LastAssignedAt`. Self-picked transfers bypass round-robin tracking.  
**Impact**: Round-robin strategy is broken for self-pick flow.  
**Fix**: Update `LastAssignedAt` in `PickNextTransfer`.

### ASGN-03 [HIGH] — Race Condition in CreateAgentTransfer
**File**: `internal/handlers/agent_transfers.go:500-512`  
**Description**: Checks `existingCount > 0` then creates without transaction. Two concurrent requests can create duplicate active transfers.  
**Fix**: Wrap check + create in a single database transaction with `SELECT FOR UPDATE`.

### ASGN-04 [MEDIUM] — No Bulk Assignment
**Description**: Cannot assign multiple contacts at once.  
**Fix**: Add `PUT /api/contacts/bulk-assign` endpoint.

### ASGN-05 [MEDIUM] — No Assignment Timeout/Auto-Escalation
**Description**: SLA breach tracked but no auto-escalation (reassign to manager).  
**Fix**: Add worker that auto-escalates expired transfers.

### ASGN-06 [MEDIUM] — No Skill-Based Routing
**Description**: Team assignment is purely round-robin or load-balanced by count — no skill/capability matching.  
**Fix**: Add skill tags to agents; match transfer topic to agent skills.

### ASGN-07 [MEDIUM] — Campaign Assignment Gap
**Description**: No way to assign campaign ownership or restrict which campaigns a user can execute beyond global `campaigns:execute`.  
**Fix**: Add `campaign_owner_id` field; scope execution permission to owned campaigns.

### ASGN-08 [MEDIUM] — ReturnAgentTransfersToQueue Not Called on WebSocket Disconnect
**File**: `internal/handlers/agent_transfers.go:1589`  
**Description**: Only called when user explicitly sets unavailable, not on connection drop.  
**Fix**: Hook into WebSocket disconnect handler.

### ASGN-09 [MEDIUM] — No Transfer Priority Queue
**Description**: All queue picks are FIFO; no priority for VIP customers.  
**Fix**: Add `priority` field to AgentTransfer; order by priority then transferred_at.

### ASGN-10 [LOW] — No Assignment History/Audit Trail
**Description**: No table tracking who assigned what to whom over time.  
**Fix**: Add `assignment_history` table with timestamp, from_user, to_user, contact_id.

### ASGN-11 [LOW] — Load-Balanced Strategy Ignores Transfer Complexity
**Description**: Counts active transfers equally regardless of difficulty.  
**Fix**: Add weight/complexity scoring to transfers.

### ASGN-12 [LOW] — No Push/Email Notifications for Queue Items
**Description**: Only WebSocket broadcast for new transfers. No push notification.  
**Fix**: Add email/push notification when transfer enters queue.

### ASGN-13 [LOW] — Manual Strategy Teams Have No Fallback
**Description**: If all agents on a manual team are unavailable, transfer sits forever.  
**Fix**: Add escalation to admin/manager after N minutes.

### ASGN-14 [LOW] — Chat Assignment Reset Per-Instance, Not Per-Team
**File**: `internal/handlers/chat_assignment_reset_worker.go`  
**Description**: Cannot set different reset schedules per team.  
**Fix**: Add team-level reset configuration.

### ASGN-15 [LOW] — No Assignment Analytics
**Description**: No metrics on assignment distribution, time-to-assign, agent workload.  
**Fix**: Add dashboard with assignment analytics.

---

## 5. Missing Features

### MF-01 [CRITICAL] — No Dead-Letter Queue for Inbound Messages
**Impact**: If `saveIncomingMessage` fails (DB error), message is permanently lost.  
**Fix**: Add DLQ in Redis for failed inbound processing; add retry worker.

### MF-02 [HIGH] — No Retry on Outgoing Send Failure
**File**: `internal/handlers/messages.go:672`  
**Description**: `finalizeMessageSend` marks as failed immediately. No exponential backoff or retry queue.  
**Fix**: Add retry queue with exponential backoff (3 retries, 30s/5min/30min).

### MF-03 [HIGH] — No Idempotency Key on Outgoing Messages
**Description**: Frontend retry can create duplicate messages.  
**Fix**: Accept client-generated idempotency key; check before insert.

### MF-04 [HIGH] — No Incoming Typing Indicator Display
**Description**: Backend sends typing events but frontend doesn't render them.  
**Fix**: Add typing indicator component; subscribe to typing WS events.

### MF-05 [HIGH] — No Incoming Read Receipts (Auto-Read)
**Description**: `auto_read_receipt` setting exists but no handler implementation.  
**Fix**: Implement auto-read receipt marking for inbound messages.

### MF-06 [HIGH] — No Reconnect Message Replay
**Description**: When client disconnects and reconnects, messages during gap are lost.  
**Fix**: Store last-seen message timestamp; on reconnect, fetch missed messages.

### MF-07 [HIGH] — No Chat Transfer Between Agents (API)
**Description**: WS message types for transfer exist but no REST API endpoint.  
**Fix**: Add `POST /api/contacts/{id}/transfer` endpoint.

### MF-08 [HIGH] — No Bulk Chat Operations
**Description**: Cannot bulk close/assign/reopen chats.  
**Fix**: Add batch endpoints with input validation.

### MF-09 [MEDIUM] — No Contact Merge/Deduplication
**Description**: No mechanism to merge duplicate phone numbers.  
**Fix**: Add contact merge API with conversation consolidation.

### MF-10 [MEDIUM] — No Agent Online/Offline Presence
**Description**: No agent online/offline WebSocket broadcast.  
**Fix**: Track agent WebSocket connections; broadcast presence changes.

### MF-11 [MEDIUM] — No Message Threading
**Description**: No first-class threading; relies on `reply_to_message_id`.  
**Fix**: Add thread view that groups replies together.

### MF-12 [MEDIUM] — No Voice Message Recording
**Description**: `MessageTypeAudio` exists but no recording capability.  
**Fix**: Add MediaRecorder API integration in frontend; audio upload handler.

### MF-13 [MEDIUM] — No Location Message Handling
**Description**: `MessageTypeLocation` constant exists but no specific UI.  
**Fix**: Add map display for location messages.

### MF-14 [MEDIUM] — No E2E Encryption Indicators
**Description**: No encryption status tracking or UI.  
**Fix**: Display lock icon for encrypted messages.

### MF-15 [MEDIUM] — No Message Pinning
**Description**: No ability to pin important messages.  
**Fix**: Add `is_pinned` field + pin UI.

### MF-16 [MEDIUM] — No Rich Link Previews
**Description**: No URL preview generation for shared links.  
**Fix**: Add link unfurling service; render preview cards.

### MF-17 [LOW] — No Message Star/Favorite
**Fix**: Add star toggle with filtered view.

### MF-18 [LOW] — No Drag-and-Drop File Upload
**Fix**: Add drag-and-drop zone to chat input area.

### MF-19 [LOW] — No Chat Background Image Upload
**Description**: `useChatBackground` composable exists but usage is basic.  
**Fix**: Implement full background image selection/upload UI.

### MF-20 [LOW] — No File Type Icons
**Description**: No distinct icons for different document types.  
**Fix**: Map MIME types to icons.

### MF-21 [LOW] — No Emoji Reactions UI
**Description**: Backend supports reactions but frontend rendering is limited.  
**Fix**: Add reaction picker and display component.

### MF-22 [LOW] — No Internal Approval Workflow for Templates
**Description**: Templates approved externally by Meta only.  
**Fix**: Add internal review step before Meta submission.

### MF-23 [LOW] — No Dark Mode-Specific Chat Bubble Styling
**Fix**: Add dark mode variants for chat bubbles.

---

## 6. Workflow Gaps

### WF-01 [HIGH] — No Dead-Letter Queue for Inbound Messages
**Flow**: Incoming Message → `processIncomingMessageFull` → `saveIncomingMessage`  
**Gap**: If DB write fails, message is silently lost. No retry, no DLQ.  
**Impact**: Permanent message loss.  
**Fix**: Add Redis DLQ; retry worker with exponential backoff.

### WF-02 [HIGH] — Outgoing Message Race Between Broadcast and Finalize
**Flow**: `broadcastNewMessage` (before send) → `finalizeMessageSend` (after send)  
**Gap**: Two WS messages for one send — frontend shows pending then status update.  
**Fix**: Broadcast only after send completes, or use optimistic update with reconciliation.

### WF-03 [HIGH] — Auto-Assignment Not Triggered on Contact Creation
**Flow**: Incoming message → contact created → chatbot disabled → transfer created  
**Gap**: `createTransferToQueue` creates unassigned transfer. Round-robin/load-balanced functions exist but are NOT called in the incoming message flow.  
**Fix**: Call team assignment strategy when creating transfers from incoming messages.

### WF-04 [HIGH] — Duplicate Incoming Message Race
**Flow**: Two concurrent webhook deliveries → both pass `fetchExistingIncomingMessageIDs` → both insert  
**Gap**: Batch dedup check has a race window between SELECT and INSERT.  
**Fix**: Use `INSERT ... ON CONFLICT DO NOTHING` with unique constraint on `whats_app_message_id`.

### WF-05 [MEDIUM] — No Reconnect Message Replay
**Flow**: Client disconnects → messages sent during gap → client reconnects  
**Gap**: Messages during gap are permanently lost to that client.  
**Fix**: Track last-seen message ID per client; fetch missed messages on reconnect.

### WF-06 [MEDIUM] — Broadcast Channel Drops Messages Under Load
**Flow**: Hub receives broadcast → channel full → message dropped  
**Gap**: Non-blocking send with only a log warning.  
**Fix**: Increase buffer; implement backpressure or priority queue.

### WF-07 [MEDIUM] — No Push Notification for Queue Transfers
**Flow**: Transfer created → WS broadcast → agents may not be online  
**Gap**: Offline agents miss new transfer notifications entirely.  
**Fix**: Add push/email notification on transfer creation.

### WF-08 [MEDIUM] — Inbox Loading 5-Second Timeout Too Tight
**Flow**: `ListContacts` with complex filters → 5s timeout  
**Gap**: Complex queries with many active filters can exceed this.  
**Fix**: Increase to 10-15s or optimize query paths.

### WF-09 [MEDIUM] — Transfer Broadcasts to Entire Org
**Flow**: Transfer created → `BroadcastToOrg`  
**Gap**: All org users receive transfer events, not just agents with appropriate permissions.  
**Fix**: Scope broadcasts to users with `transfers:read` permission.

### WF-10 [LOW] — LID Contact Migration Fire-and-Forget
**Flow**: Whatsmeow inbound → `migrateContactPhoneFromLID` → error logged only  
**Gap**: Migration errors are not retried.  
**Fix**: Add retry queue for failed migrations.

### WF-11 [LOW] — Hub.Run() Has No Shutdown Mechanism
**File**: `internal/websocket/hub.go:42-51`  
**Gap**: Goroutine leaks on shutdown.  
**Fix**: Add context cancellation or stop channel.

### WF-12 [LOW] — Dual WebSocket Authentication is Redundant
**Flow**: HTTP header auth + WS message auth  
**Gap**: Both paths exist; WS message auth is legacy.  
**Fix**: Remove WS message auth path; document HTTP-only auth.

---

## 7. Frontend-Specific Gaps

### FE-01 [CRITICAL] — ChatView.vue is 6,193 lines
(See ARCH-01)

### FE-02 [CRITICAL] — 79% of Chat i18n Missing in Spanish
**File**: `frontend/src/i18n/locales/es.json`  
**Details**: 174 chat keys untranslated out of 220 total.  
**Fix**: Complete Spanish translation for all chat keys.

### FE-03 [HIGH] — No Keyboard Navigation
**Description**: Only `@keydown.enter` for send. No arrow keys for sidebar, no Escape for panels.  
**Fix**: Add full keyboard navigation (arrow keys, Escape, Tab, Enter).

### FE-04 [HIGH] — Minimal ARIA Labels
**Description**: Only 5 `aria-label` attributes in 6,193 lines. No `aria-live` for message updates.  
**Fix**: Add comprehensive ARIA labels; use `aria-live="polite"` for new messages.

### FE-05 [HIGH] — No Error State for Message Load Failure
**File**: `frontend/src/stores/contacts.ts:1094`  
**Description**: Only `console.error`, no retry UI.  
**Fix**: Add error component with retry button.

### FE-06 [HIGH] — No Error Boundary
**Description**: Any runtime error in ChatView crashes entire chat.  
**Fix**: Add Vue error boundary wrapper around ChatView.

### FE-07 [HIGH] — WebSocket Disconnect Has No UI Indicator
**File**: `frontend/src/services/websocket.ts`  
**Description**: Max 5 reconnect attempts with no visible feedback. User sees stale data silently.  
**Fix**: Add connection status banner (connecting/reconnected/offline).

### FE-08 [MEDIUM] — No Virtual Scrolling for Messages
**Fix**: Implement `vue-virtual-scroller` or similar.

### FE-09 [MEDIUM] — No Message Search Within Conversation
**Fix**: Add search bar in message area with backend support.

### FE-10 [MEDIUM] — Media Blob URLs Not Cleaned on Error
**File**: `frontend/src/views/chat/ChatView.vue:663-671`  
**Fix**: Add cleanup in error handler and `onUnmounted`.

### FE-11 [MEDIUM] — Deep Watch on Messages Array
**File**: `frontend/src/views/chat/ChatView.vue:1879-1898`  
**Fix**: Replace with computed properties or specific watchers.

### FE-12 [MEDIUM] — `any` Type Used Extensively in WS Handling
**File**: `frontend/src/services/websocket.ts`  
**Fix**: Define proper TypeScript interfaces for all WS payload types.

### FE-13 [MEDIUM] — No Debouncing on Media Prefetch
**Fix**: Debounce media prefetch queue; limit concurrent requests.

### FE-14 [LOW] — ContactInfoPanel.vue.bak Left in Repo
**Fix**: Remove backup file.

### FE-15 [LOW] — markConversationAsRead is a Hack
**File**: `frontend/src/stores/contacts.ts:1318-1327`  
**Description**: Fetches 1 message to trigger server-side read.  
**Fix**: Add dedicated `POST /api/contacts/{id}/read` endpoint.

---

## 8. Data Flow Diagrams

### Incoming Message Flow
```
Meta Cloud API ───────┐                     Whatsmeow Protocol ───────┐
  POST /api/webhook   │                     handleEvent()              │
  validateWebhook()   │                     handleMessage()            │
  dedup check         │                     normalizeMessage()         │
       │              │                          │                    │
       ▼              │                          ▼                    │
  processIncoming     │                     persistParsedMessage()    │
  MessageFull()       │                     findOrCreateContact()     │
       │              │                     download media            │
       ▼              │                          │                    │
  saveIncomingMessage()│                          ▼                    │
  (DB: INSERT msg)    │                     (DB: INSERT msg)          │
       │              │                          │                    │
       ▼              │                          ▼                    │
  Chatbot processing? │◄─────────────────────────┘                    │
  ├── Yes → keyword/flow/AI                                          │
  └── No  → continue                                                │
       │                                                             │
       ▼                                                             │
  broadcastNewMessage()                                              │
       │                                                             │
       ▼                                                             │
  Hub.BroadcastToOrg() ──► All connected clients                     │
```

### Conversation Assignment Flow
```
Incoming Message
     │
     ├─ Chatbot keyword/flow triggers
     │       │
     │       ▼
     │   createTransferToQueue()
     │       │
     │       ├─ Team strategy → auto-assign (round_robin / load_balanced)
     │       ├─ AssignToSameAgent → reassign if available
     │       └─ No match → stays in queue (agent_id=NULL)
     │
     ├─ Agent picks from queue
     │       │
     │       ▼
     │   PickNextTransfer() ──► SELECT FOR UPDATE SKIP LOCKED (FIFO)
     │       │
     │       ▼
     │   Contact.AssignedUserID updated → status = "open"
     │
     ├─ Agent resolves → ResumeFromTransfer → back to chatbot
     │
     └─ Direct assignment
             │
             ▼
         AssignContact() → Contact.AssignedUserID set → status = "open"
```

### WebSocket Lifecycle
```
Browser ──► GET /ws (Authorization: Bearer <token>)
              │
              ▼
         WebSocketHandler()
              │
              ├─ License check
              ├─ JWT validation → userID, orgID
              ├─ Redis DEL token (replay protection)
              │
              ▼
         ws.NewClient(hub, conn, userID, orgID)
              │
              ▼
         hub.Register(client)
              │
         ┌────┴────┐
         │         │
    ReadPump   WritePump
    (ping/    (messages from
     auth,    client.send
     set_     channel)
     contact)
```

---

## 9. Middleware Stack (Chat Routes)

Applied in order for all chat endpoints:

1. **SecurityHeaders** — CSP, X-Frame-Options (production only)
2. **RequestLogger** — Request/response logging
3. **Recovery** — Panic recovery
4. **License gate** — Blocks if license locked/overage
5. **CSRFProtection** — Double-submit cookie (skipped for Bearer/API-key)
6. **AuthWithDB** — JWT + API key authentication
7. **TenantScope** — Scopes DB queries to organization
8. **Rate limiting** — Per-endpoint (NOT applied to chat lifecycle endpoints)

---

## 10. Permission Matrix (Assignment-Related)

| Resource | read | write | delete | execute | pickup |
|----------|------|-------|--------|---------|--------|
| contacts | ✅ | ✅ | ✅ | | |
| chat.assign | | ✅ | | | |
| chat.collaborators | ✅ | ✅ | | | |
| transfers | ✅ | ✅ | | | ✅ |
| campaigns | ✅ | ✅ | ✅ | ✅ | |
| templates | ✅ | ✅ | ✅ | | ✅ |

---

## 11. Recommendations by Priority

### Immediate (Week 1)
1. Fix SEC-001: Remove WS protocol token path; hard-fail without Redis
2. Fix SEC-002: Add rate limiting to message send endpoints
3. Fix SEC-005: Add MaxBytesReader to media uploads
4. Fix ASGN-03: Wrap CreateAgentTransfer in transaction
5. Fix WF-04: Add unique constraint on whats_app_message_id

### Short-term (Week 2-3)
6. Fix C1: Increase WS broadcast buffer; add backpressure
7. Fix ARCH-01: Begin ChatView decomposition (extract 5 most critical components)
8. Fix ARCH-06: Resolve N+1 in ListContacts
9. Fix ARCH-07: Add missing composite indexes
10. Fix ASGN-01: Add transfer expiration worker
11. Fix ASGN-02: Update LastAssignedAt on transfer pick
12. Fix MF-01: Add DLQ for inbound messages

### Medium-term (Month 2)
13. Fix FE-02: Complete Spanish i18n
14. Fix ARCH-03: Decompose processIncomingMessageFull
15. Fix PER-004: Implement targeted contact-scoped broadcasts
16. Fix MF-02: Add outgoing message retry queue
17. Fix MF-06: Add reconnect message replay
18. Fix FE-03/04: Add keyboard navigation + ARIA labels

### Long-term (Month 3+)
19. Fix ARCH-01/02: Complete frontend decomposition
20. Fix PER-003: Implement virtual scrolling
21. Fix ARCH-13: Migrate to cursor-based pagination
22. Fix MF-04/05/07: Typing indicators, auto-read, transfer API
23. Fix ASGN-06: Skill-based routing
24. Fix SEC-006: JWT key rotation

---

## 12. Files Analyzed

### Backend
- `cmd/whatomate/server.go` — Route definitions
- `internal/handlers/contacts.go` — ListContacts, GetContact
- `internal/handlers/contacts_management.go` — AssignContact, soft delete
- `internal/handlers/contacts_messaging.go` — SendMessage, typing, reactions
- `internal/handlers/messages.go` — SendOutgoingMessage, broadcast
- `internal/handlers/chatbot_processor.go` — processIncomingMessageFull
- `internal/handlers/chat_lifecycle.go` — Claim, close, reopen, status
- `internal/handlers/chat_system_messages.go` — System event messages
- `internal/handlers/chat_close_ratings.go` — Close rating context
- `internal/handlers/chat_access_policy.go` — Agent visibility filters
- `internal/handlers/agent_transfers.go` — Transfer CRUD, queue, strategies
- `internal/handlers/chat_assignment_reset_worker.go` — Scheduled unassignment
- `internal/handlers/webhook.go` — Meta webhook handler
- `internal/handlers/websocket.go` — WebSocket handler
- `internal/handlers/media.go` — Media upload/serve
- `internal/handlers/jwt_secret.go` — JWT secret validation
- `internal/middleware/middleware.go` — Auth, rate limiting, CSRF
- `internal/middleware/csrf.go` — CSRF protection
- `internal/models/models.go` — Contact, Message, User models
- `internal/models/roles.go` — RBAC model
- `internal/models/constants.go` — Status constants
- `internal/models/chatbot.go` — AgentTransfer model
- `internal/models/chat_assignment_reset_settings.go` — Reset config
- `internal/models/collaboration.go` — ContactCollaborator model
- `internal/models/conversation_notes.go` — ConversationNote model
- `internal/models/chat_closure_rating.go` — Rating model
- `internal/models/instance.go` — WhatsAppInstance model
- `internal/websocket/hub.go` — Hub, broadcast logic
- `internal/websocket/client.go` — Client, read/write pumps
- `internal/websocket/messages.go` — WS message types
- `internal/queue/redis.go` — Redis Streams setup
- `internal/worker/` — Job processing workers
- `pkg/provider/interface.go` — MessageProvider interface
- `pkg/whatsapp/` — Meta Cloud API adapter
- `pkg/whatsmeow/` — WhatsApp Web protocol adapter

### Frontend
- `frontend/src/views/chat/ChatView.vue` — Main chat view (6,193 lines)
- `frontend/src/components/chat/ContactInfoPanel.vue` — Contact panel (963 lines)
- `frontend/src/components/chat/ConversationNotes.vue` — Notes (413 lines)
- `frontend/src/components/chat/MediaGroupBar.vue` — Batch media (258 lines)
- `frontend/src/components/chat/CannedResponsePicker.vue` — Canned responses (205 lines)
- `frontend/src/components/chat/MetadataSection.vue` — Session data (141 lines)
- `frontend/src/components/chat/WhatsAppRichTextEditor.vue` — Rich text (123 lines)
- `frontend/src/components/chat/InstanceTag.vue` — Instance badge (79 lines)
- `frontend/src/components/chat/LinkifiedMessageText.vue` — Linkified text (27 lines)
- `frontend/src/components/chat/status/` — Status viewer/composer/stories
- `frontend/src/stores/contacts.ts` — Contacts store (1,422 lines)
- `frontend/src/stores/notes.ts` — Notes store (114 lines)
- `frontend/src/stores/transfers.ts` — Transfers store (333 lines)
- `frontend/src/services/websocket.ts` — WebSocket client (958 lines)
- `frontend/src/services/api.ts` — API service layer
- `frontend/src/router/index.ts` — Route definitions
- `frontend/src/types/contacts.ts` — TypeScript types
- `frontend/src/i18n/locales/en.json` — English translations
- `frontend/src/i18n/locales/es.json` — Spanish translations (incomplete)
- `frontend/src/i18n/locales/ar.json` — Arabic translations
- `frontend/src/lib/chat-sidebar-unifier.ts` — Sidebar utilities (260 lines)
- `frontend/src/lib/chat-backgrounds.ts` — Backgrounds (364 lines)
- `frontend/src/lib/message-history-navigator.ts` — History nav (79 lines)
- `frontend/src/lib/mention-contact-resolver.ts` — Mentions (200 lines)
- `frontend/src/lib/media-actions.ts` — Media actions (205 lines)
- `frontend/src/lib/media_prefetch_cache.ts` — Media cache (209 lines)
- `frontend/src/composables/useInfiniteScroll.ts` — Infinite scroll (160 lines)
- `frontend/src/composables/useMediaGroups.ts` — Media groups (186 lines)
- `frontend/src/composables/useChatBackground.ts` — Chat background (210 lines)

---

*Report generated by 5-agent analysis team using Serena MCP, codebase-memory-mcp, and ruflo MCP.*
