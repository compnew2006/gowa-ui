# Whatomate Business Logic Documentation

This document describes the technical implementation and step-by-step logic for the core flows in the Whatomate platform.

---

## 1. Inbound Message Flow
**Path**: `WhatsApp Webhook -> Chat UI`

1. **Entry Point**: Meta sends a POST request to `/api/whatsapp/webhook`.
2. **Verification**: 
   - `handlers.App.WebhookHandler` verifies the `X-Hub-Signature-256` header using the `AppSecret` (HMAC-SHA256).
3. **Parsing**:
   - The JSON payload is parsed into `whatsapp.WebhookPayload`.
   - The system checks for `messages` or `statuses` updates.
4. **Message Ingestion**:
   - `processIncomingMessageFull` is called.
   - **Contact Resolution**: The system looks up the phone number in the `contacts` table. If not found, a new contact is created using the profile name from the payload.
   - **Duplicate Prevention**: Checks `whats_app_message_id` against existing records.
5. **Persistence**:
   - The message is saved to the `messages` table via GORM.
   - For media (images, voice, etc.), the system enqueues a recovery job in Redis (`inbound_media` stream) for background download from Meta.
6. **Real-time Dispatch**:
   - `wsHub.BroadcastToOrg` sends a `TypeMessage` event to all active WebSocket clients belonging to the message's `organization_id`.
7. **Chatbot Trigger**:
   - If the message is text, it is passed to the `Chatbot Processor`.

---

## 2. Campaign Execution Flow
**Path**: `User Starts Campaign -> Final Message Sent`

1. **Initiation**:
   - User calls `POST /api/campaigns/{id}/start`.
   - `handlers.App.StartCampaign` validates:
     - Organization active status.
     - License worker quotas.
     - WhatsApp account health.
     - Sending policies (e.g., `POLICY_NO_INBOUND`).
2. **Queueing**:
   - Campaign status changes to `Processing`.
   - All recipients are loaded from `bulk_message_recipients`.
   - Recipients are enqueued in Redis Streams (tenant-specific streams: `whatomate:campaigns:{org_id}`).
3. **Worker Processing**:
   - `worker.Worker.HandleRecipientJob` consumes the job.
   - **Idempotency**: Acquires a Redis lock for the recipient ID.
   - **Template Rendering**: Merges contact placeholders (e.g., `{{name}}`) into the template body.
   - **Anti-Spam Delay**: Applies a random delay between `min_delay` and `max_delay` defined in the campaign.
4. **Sending**:
   - **Meta Flow**: Calls WhatsApp Cloud API.
   - **Whatsmeow Flow**: Directly sends via the linked instance.
5. **Success/Failure**:
   - Updates `sent_count` or `failed_count` in the `bulk_message_campaigns` table.
   - Updates individual recipient status to `sent`.
6. **Statistics**:
   - Worker publishes a `TypeCampaignStatsUpdate` event via Redis Pub/Sub, which the API server broadcasts to UI clients via WebSocket.

---

## 3. Chatbot Decision Flow
**Path**: `Incoming Message -> Automated Response or Human Transfer`

The decision logic follows a strict priority matrix in `chatbot_processor.go`:

1. **Active Transfer Check**: If the contact has a row in `agent_transfers` that is not closed, the chatbot is bypassed.
2. **Transfer Keyword Match**: If the message matches a reserved "Transfer" keyword, an agent transfer is initiated immediately.
3. **Active Flow State**: If the user is currently in the middle of a "Flow" (multi-step dialog), the system processes their response against the expected input for the current step.
4. **Keyword Rule Match**: 
   - Exact Match -> Contains Match -> Regex Match.
   - If a match is found, the system sends the associated response or triggers an action (e.g., "Add Tag").
5. **Session Greeting**: If no active session exists (timeout configurable), the system sends a greeting message if one is defined.
6. **AI Query**: 
   - If enabled for the organization, the message + conversation history are sent to an LLM provider (OpenAI, Gemini, or Anthropic).
   - The AI response is streamed or sent back to the user.
7. **Fallback**: If all the above fail, the system sends a "Fallback Message" or does nothing if disabled.

---

## 4. Authentication Flow
**Path**: `Login -> JWT Issuance -> Org Switching`

1. **Login**:
   - `POST /api/auth/login` checks the password (bcrypt).
   - Loads the user's `user_organizations` memberships.
2. **Token Generation**:
   - **Access Token**: Short-lived (15-60m) JWT containing `user_id`, `organization_id`, and `role`.
   - **Refresh Token**: Long-lived token stored in Redis with a rotation policy.
3. **WebSocket Handshake**:
   - To avoid sending main JWTs over WebSocket URLs, the client calls `GET /api/auth/ws-token`.
   - Server returns a 30-second one-time use JWT.
   - Client connects to `/ws` with this token.
4. **Organization Switching**:
   - `POST /api/auth/switch-org`.
   - Server validates the user belongs to the target `org_id`.
   - Sets a new secure cookie and returns updated JWTs scope to the new org.

---

## 5. Multi-tenancy Enforcement
**Path**: `Data Isolation & Query Injection`

1. **Global Scoping**:
   - Primarily implemented via GORM Scopes in `internal/tenant/scope.go`.
   - Every `db` call in a request handler uses `requestDB(r)` which injects `WHERE organization_id = ?`.
2. **Resolution Logic**:
   - `ResolveOrganizationID` extracts the ID from:
     - Subdomain (mapping slugs to orgs).
     - JWT Payload claims.
     - `X-Organization-ID` header (only for `is_super_admin`).
3. **Schema Level Isolation**:
   - Tables without the `organization_id` column are ignored by the auto-scoping logic (e.g., `organizations`, `plans`).
4. **Data Integrity**:
   - Unique constraints in the DB almost always include `organization_id` (e.g., `UNIQUE(organization_id, phone_number)` for contacts).
