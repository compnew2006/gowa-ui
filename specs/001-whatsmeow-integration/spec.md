# Feature Specification: Whatsmeow Integration

**Feature Branch**: `001-whatsmeow-integration`
**Created**: 2026-02-17
**Status**: Draft
**Input**: Replace Meta Cloud API with whatsmeow for QR-based WhatsApp Web protocol integration enabling zero-cost multi-instance WhatsApp connectivity.

## Clarifications

### Session 2026-02-17

- Q: What are the valid instance statuses and their allowed transitions? → A: 5 states: `disconnected`, `connecting`, `connected`, `banned`, `logged_out`. Ban and session expiry are tracked as separate terminal states.
- Q: How should the frontend detect which WhatsApp provider is active? → A: Backend config endpoint (`/api/config`) returns `{ "whatsapp_provider": "whatsmeow" }`. Frontend conditionally renders based on this response.
- Q: When WhatsApp bans or restricts a number, how should the system notify the admin? → A: WebSocket real-time event + persistent in-app notification (bell icon), visible on next login if admin was offline.
- Q: What should happen to outbound messages sent while the instance is disconnected? → A: Queue with 5-minute timeout. If instance reconnects within window, send queued messages with rate-limited delays. If timeout expires, mark as "failed".
- Q: What level of observability should the whatsmeow integration provide? → A: Structured JSON logs for all instance events + admin-facing health dashboard showing per-instance connection uptime, message counts, and error rates.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — QR Code Instance Pairing (Priority: P1)

An organization admin opens the Instances page and clicks "Create Instance". They enter a friendly name (e.g., "Sales Phone") and submit. The system creates the instance record. The admin clicks "Connect", and a QR code appears on screen in real-time. They scan it with their personal WhatsApp on their phone. Within seconds, the status badge turns green, the phone number is displayed, and the instance is ready to send and receive messages — with zero Meta account setup, zero API keys, and zero cost.

**Why this priority**: This is the fundamental new capability. Without QR pairing, nothing else in the integration works. It directly solves the core problem: eliminating Meta Business Account friction.

**Independent Test**: Can be fully tested by creating an instance, scanning a QR code, and verifying the connection status turns to "connected" with the phone number populated. Delivers immediate value — the user has a working WhatsApp connection.

**Acceptance Scenarios**:

1. **Given** an authenticated admin on the Instances page, **When** they create a new instance with a name and click "Connect", **Then** a QR code is displayed within 3 seconds via WebSocket.
2. **Given** a displayed QR code, **When** the user scans it with WhatsApp on their phone, **Then** the instance status updates to "connected" and the phone number is shown within 10 seconds.
3. **Given** a connected instance, **When** the server is restarted, **Then** the instance auto-reconnects without requiring a new QR scan.
4. **Given** an instance with status "connected", **When** the user clicks "Disconnect", **Then** the status changes to "disconnected" and the WhatsApp session is gracefully closed.

---

### User Story 2 — Send and Receive Messages (Priority: P1)

An agent opens a contact's chat and types a text message (or attaches an image, video, document, or audio file). They click send. The message is delivered to the contact's WhatsApp via the connected whatsmeow instance. When the contact replies, the incoming message appears in the agent's chat view in real-time. Read receipts and delivery confirmations are reflected in the UI.

**Why this priority**: Messaging is the core purpose of the platform. Without send/receive, the QR pairing from US1 has no practical value.

**Independent Test**: Can be tested by sending a text message to a real phone number and verifying delivery, then replying from that phone and verifying the reply appears in the UI in real-time.

**Acceptance Scenarios**:

1. **Given** a connected instance and a contact, **When** an agent sends a text message, **Then** the message arrives on the contact's phone as a WhatsApp message.
2. **Given** a connected instance, **When** an agent sends an image with a caption, **Then** the image and caption arrive on the contact's phone.
3. **Given** a connected instance, **When** the contact sends a reply, **Then** the reply appears in the agent's chat within 2 seconds via WebSocket.
4. **Given** a sent message, **When** the contact reads it, **Then** the message status updates to "read" (blue ticks) in the UI.
5. **Given** an agent viewing a message, **When** they react with an emoji, **Then** the reaction is delivered to the contact's phone.

---

### User Story 3 — Multi-Instance Management (Priority: P2)

An organization admin manages multiple WhatsApp numbers simultaneously. They create several instances (e.g., "Sales", "Support", "Marketing"), each connected to a different phone number. Each instance operates independently. The admin can view connection health, disconnect, reconnect, or delete instances individually. When sending a message, the system uses the organization's default instance or allows selecting a specific one.

**Why this priority**: Multi-instance is a key differentiator over the Meta Cloud API approach (which requires separate Meta Business accounts). However, single-instance messaging (US1 + US2) is a viable MVP on its own.

**Independent Test**: Can be tested by creating 3+ instances, connecting each via QR, and verifying all remain connected simultaneously with independent status tracking.

**Acceptance Scenarios**:

1. **Given** an organization with 3 connected instances, **When** the admin views the Instances page, **Then** all 3 show independent connection status badges.
2. **Given** multiple instances, **When** one is disconnected, **Then** the others continue operating normally.
3. **Given** multiple instances, **When** an agent sends a message, **Then** the system uses the organization's default instance (or the instance assigned to that contact).
4. **Given** 5 connected instances, **When** running sustained message traffic, **Then** all instances remain stable for 24+ hours without memory growth.

---

### User Story 4 — Graceful Feature Hiding (Priority: P2)

When the platform is running with whatsmeow (instead of Meta Cloud API), Meta-exclusive features are hidden from the UI. Template management, WhatsApp Flows builder, Catalog sync, and Business Profile editor are not visible — or display an informative banner explaining they are only available with Meta Cloud API. Users are not confused by broken or non-functional features.

**Why this priority**: Without this, users will encounter errors when trying to use Meta-only features. It's essential for a clean user experience but doesn't block core messaging functionality.

**Independent Test**: Can be tested by logging in and navigating to each Meta-only feature area, verifying it is either hidden or shows an appropriate informational message.

**Acceptance Scenarios**:

1. **Given** a whatsmeow deployment, **When** a user navigates the sidebar, **Then** Template management, Flows, Catalog, and Business Profile menu items are hidden based on the provider type returned by the config endpoint.
2. **Given** a whatsmeow deployment, **When** a user directly accesses a Meta-only feature URL, **Then** they see an informative message rather than an error.
3. **Given** a deployment, **When** the frontend loads, **Then** it fetches `/api/config` to determine the active provider and caches the result for the session.

---

### User Story 5 — Data Migration (Priority: P3)

An existing Whatomate deployment using Meta Cloud API has contacts, messages, and accounts data. The admin runs the migration to switch to whatsmeow. Existing contacts and message history are preserved. Foreign key references are updated from the old `whatsapp_accounts` table to the new `whatsapp_instances` table. The old data tables are retained for rollback safety.

**Why this priority**: Only relevant for existing deployments upgrading from Meta. New installations skip this entirely. Core functionality (US1-US4) is fully independent.

**Independent Test**: Can be tested by running the migration script against a database with existing Meta-based data and verifying all contacts and messages are accessible under the new instance model.

**Acceptance Scenarios**:

1. **Given** an existing database with `whatsapp_accounts` data, **When** the migration is run, **Then** all contacts are re-linked to new `whatsapp_instances` records.
2. **Given** a completed migration, **When** the admin views message history, **Then** all historical messages are intact and correctly attributed.
3. **Given** a migration failure, **When** the admin rolls back, **Then** the old `whatsapp_accounts` table is still intact and the system can revert to Meta Cloud API mode.

---

### Edge Cases

- What happens when the phone running WhatsApp goes offline for an extended period? The instance session may be invalidated by WhatsApp servers, requiring a new QR scan. The system MUST detect this, update status to `logged_out`, notify the admin via WebSocket `instance_logged_out` event, AND create a persistent in-app notification visible on next login.
- What happens when WhatsApp rate-limits the connection? The system MUST implement backoff delays and queue messages rather than dropping them. Human-like sending intervals (randomized delays) MUST be enforced.
- What happens when media upload fails? The system MUST retry once, and if it still fails, mark the message as "failed" with an actionable error for the agent.
- What happens when the QR code expires before scanning? The QR code refreshes automatically. If the WebSocket connection drops, the user can click "Connect" again to restart the flow.
- What happens when two instances attempt to use the same phone number? The system MUST detect duplicate JIDs and prevent connection, displaying a clear error.
- What happens when the agent sends a message while the instance is temporarily disconnected? The system MUST queue outbound messages for up to 5 minutes. If the instance reconnects within this window, queued messages are sent with rate-limited delays. If the timeout expires, messages are marked as "failed" with an actionable error for the agent.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow admins to create named WhatsApp instances without any Meta/Facebook credentials.
- **FR-002**: System MUST stream a QR code to the frontend via WebSocket when an instance connection is initiated.
- **FR-003**: System MUST persist WhatsApp sessions across server restarts without requiring re-scanning.
- **FR-004**: System MUST auto-reconnect all previously connected instances on server startup.
- **FR-005**: System MUST support sending text, image, video, audio, and document messages through the connected instance.
- **FR-006**: System MUST receive incoming messages and deliver them to the frontend in real-time via WebSocket.
- **FR-007**: System MUST support send/receive of emoji reactions.
- **FR-008**: System MUST support read receipt marking (blue ticks).
- **FR-009**: System MUST support at least 5 simultaneous connected instances per organization.
- **FR-010**: System MUST expose a config endpoint (`/api/config`) that returns the active WhatsApp provider type. Frontend MUST use this to hide or disable Meta-exclusive features (Templates, Flows, Catalog, Business Profile) when the provider is whatsmeow.
- **FR-011**: System MUST provide instance lifecycle endpoints: connect, disconnect, reconnect, status check.
- **FR-012**: System MUST provide a migration path from `whatsapp_accounts` to `whatsapp_instances` preserving all existing data.
- **FR-013**: System MUST preserve the old `whatsapp_accounts` table during migration for rollback safety.
- **FR-014**: System MUST implement rate limiting with human-like randomized delays to mitigate ban risk.
- **FR-015**: System MUST broadcast instance connection status changes (connected, disconnected, banned, logged_out) via WebSocket events.
- **FR-016**: System MUST support group message send and receive.
- **FR-017**: System MUST support reply-to-specific-message (quote context).
- **FR-018**: System MUST create persistent in-app notifications for critical instance events (banned, logged_out) that remain visible until dismissed, even if the admin was offline when the event occurred.
- **FR-019**: System MUST queue outbound messages for up to 5 minutes when the target instance is temporarily disconnected. On reconnection, queued messages MUST be sent with rate-limited delays. On timeout expiry, messages MUST be marked as "failed".
- **FR-020**: System MUST emit structured JSON logs for all instance lifecycle events (connect, disconnect, ban, logout, QR scan, message sent/received errors).
- **FR-021**: System MUST provide an admin-facing health dashboard showing per-instance connection uptime, message counts (sent/received/failed), and error rates.

### Key Entities

- **WhatsApp Instance**: Represents a connected WhatsApp phone number. Attributes: name, phone number (populated after QR scan), JID (WhatsApp identifier), connection status, organization ownership, session data, default flag. Status lifecycle: `disconnected` → `connecting` → `connected`. From `connected`: can transition to `disconnected` (user action or network loss), `banned` (WhatsApp enforcement), or `logged_out` (session expired/revoked). `banned` is a terminal state requiring a new phone number. `logged_out` requires admin intervention (new QR scan) and transitions back to `disconnected` to restart the pairing flow.
- **Message**: Existing entity extended with instance association (replacing account association). Supports text, media, reactions, and status tracking.
- **Contact**: Existing entity re-linked from account to instance. Each contact is associated with the instance through which they communicate.
- **Organization**: Existing entity. Each instance belongs to one organization. Multi-tenant isolation enforced.

### Assumptions

- Users have a dedicated phone (not their primary personal number) for connecting to WhatsApp via whatsmeow.
- Users understand that third-party WhatsApp APIs carry a risk of account bans and accept this tradeoff for zero cost.
- The whatsmeow library is stable enough for production use (current version pinned in `go.mod`).
- PostgreSQL is used as the session store for whatsmeow device data.
- Media files are stored locally or in the existing S3-compatible storage configured in `config.toml`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create an instance and complete QR code pairing in under 30 seconds end-to-end (instance creation + QR display + scan + connected status).
- **SC-002**: All existing message types (text, image, video, audio, document) work through whatsmeow with the same user experience as the Meta Cloud API path.
- **SC-003**: Incoming messages appear in the agent's chat within 2 seconds of being sent by the contact.
- **SC-004**: After a server restart, all previously connected instances auto-reconnect without user intervention within 60 seconds.
- **SC-005**: System sustains 100,000 messages per day per instance without memory leaks or crashes.
- **SC-006**: 5 instances connected simultaneously remain stable for 24+ hours under continuous message traffic.
- **SC-007**: Users encounter zero broken UI elements related to Meta-only features when running in whatsmeow mode.
- **SC-008**: Existing deployments can migrate from Meta Cloud API to whatsmeow with zero data loss of contacts and message history.
- **SC-009**: Each connected instance uses less than 50 MB of memory after 24 hours of uptime.
- **SC-010**: Core platform features (auth, contacts, campaigns, chatbot, canned responses, teams, roles, analytics) continue to function without regression after whatsmeow integration.
- **SC-011**: Admin health dashboard displays accurate per-instance metrics (uptime, message counts, error rates) updated within 30 seconds of events.
