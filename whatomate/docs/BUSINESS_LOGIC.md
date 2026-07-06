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

## 5. Poll Vote Flow (WhatsMeow)
**Path**: `Chat UI -> Send Poll Vote -> E2E Encrypted Delivery`

### Overview
Whatomate supports casting votes on native WhatsApp polls (both single-select and multi-select) via the WhatsMeow provider. This feature required a significant E2E encryption fix involving LID (Linked Identifier) resolution.

### Flow Steps

1. **Initiation**:
   - User clicks a poll option in the Chat UI.
   - `buildNextPollSelection()` logic determines if the user is selecting or deselecting an option.
   - For multi-select polls: deselecting removes only the clicked option. For single-select: replaces current selection.
   - The UI distinguishes between multi-select (checkboxes, rounded border) and single-select (radio buttons, rounded-full border).

2. **Backend Vote Request**:
   - `POST /api/contacts/{id}/messages/{message_id}/poll-vote` sends the vote.
   - Body: `{selected_options: ["option1", "option2"]}`
   - Only available for WhatsMeow provider.

3. **Selection Limit Resolution**:
   - `pollVoteSelectionLimit()` queries the original poll message from the database.
   - Parses `max_selections` from the poll creation metadata.
   - If `max_selections = 0` (unlimited), returns `999`.
   - If `max_selections = 1` (single-select), returns `1`.

4. **Vote Encryption & Sending**:
   - `WhatsmeowAdapter.SendPollVote()` handles the actual vote:

   **Step 4a — Lookup Original Poll:**
   - Queries `messages` table by `organization_id`, `instance_id`, and `whats_app_message_id`.
   - Retrieves the original poll message including metadata (direction, conversation_id, is_group flag).

   **Step 4b — Resolve Chat JID:**
   - Parses `ConversationID` to get the chat JID.
   - Determines if the conversation is a group chat from `metadata.is_group`.
   - Gets the bot's own JID from `client.Store.GetJID()`.
   - **LID Resolution**: If the chat JID is a phone-number JID (`@s.whatsapp.net`), resolves it to a LID JID (`@lid`) via `client.Store.LIDs.GetLIDForPN()`.

   **Step 4c — Resolve Sender JID:**
   - `resolvePollSender()` determines the correct sender:
     - Outgoing poll (sent by us): uses the bot's own JID.
     - Incoming group poll: extracts sender from `metadata.sender_phone`.
     - Incoming direct poll: uses the chat partner's JID.
   - **LID Resolution**: Resolves the sender JID to LID if needed.

   **Step 4d — Build E2E Encrypted Vote:**
   - Normalizes both JIDs to non-AD form (`.ToNonAD()`).
   - Constructs `MessageInfo` with the resolved JIDs, original poll message ID, and timestamp.
   - **E2E Workaround**: If the chat is a LID chat (`HiddenUserServer`):
     - Saves `client.Store.ID` (bot's phone JID).
     - Temporarily sets `client.Store.ID` to the bot's **LID JID**.
     - Calls `client.BuildPollVote()` — ensuring correct encryption key derivation.
     - Restores the original `client.Store.ID`.
   - If not a LID chat, calls `BuildPollVote()` normally.

   **Step 4e — Send:**
   - Calls `client.SendMessage()` with the encrypted vote payload.
   - Returns the response message ID.

5. **Frontend Update**:
   - The vote is reflected immediately in the UI.
   - Multi-select polls show "(X/Y) votes" or "(unlimited)" based on max_selections.

### Key Files
| File | Purpose |
|------|---------|
| `pkg/whatsmeow/adapter_send.go` | `SendPollVote()` — core vote logic with LID resolution |
| `internal/handlers/contacts_messaging.go` | `pollVoteSelectionLimit()` — selection limit logic |
| `frontend/src/views/chat/ChatView.vue` | `buildNextPollSelection()`, `getPollSelectionLimit()` — UI selection logic |
| `internal/handlers/poll_vote_helpers_test.go` | Tests for selection limit resolution |

### Why LID Resolution is Required
WhatsApp assigns LID JIDs (`@lid`) to user accounts. When `BuildPollVote()` constructs the encrypted vote:
- It signs using the **sender's JID** for key derivation.
- It encrypts using the **chat JID** for key derivation.
- If these JIDs don't match what the recipient expects (phone-number vs LID), E2E decryption fails and the vote is silently discarded.
- The `Store.ID` override ensures the encryption identity matches what the LID recipient expects.

---

## 6. Multi-tenancy Enforcement
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
