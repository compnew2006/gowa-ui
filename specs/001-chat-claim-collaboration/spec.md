# Feature Specification: Chat Status, Claim & Collaboration System

**Feature Branch**: `001-chat-claim-collaboration`  
**Created**: 2026-07-12  
**Status**: Draft  
**Input**: User description: "implement المرحلة 1: نظام حالة المحادثة + الاستلام (Claim) + التعاون (Collaboration) from phase1.md respecting user roles"

---

## Clarifications

### Session 2026-07-12

- Q: Should the primary agent (owner) be able to remove (kick) a collaborator, or is self-leave the only removal method? → A: **Only managers/admins can remove any collaborator. Agents cannot remove each other. Each agent can self-leave. When the last participant (owner) leaves, the conversation must CLOSE (not remain open).**
- Q: Should pending conversations auto-close after inactivity? → A: **Pending conversations NEVER auto-close — they stay pending forever until claimed. However, CLAIMED (open) conversations MUST auto-revert to pending and release the assigned agent after X hours of inactivity. Collaborators are also cleared on revert.**
- Q: What is the default inactivity timeout? → A: **The timeout value MUST be configurable from the settings page (Settings → General), inheriting the existing `Organization.Settings` JSONB pattern (like `mask_phone_numbers`). The settings UI exposes a field for `chat_inactivity_timeout_hours`. A background worker checks periodically. Default value: 24 hours.**
- Q: Should agents receive a warning before auto-revert? → A: **Silent revert — no warning message or notification. The conversation simply reverts to pending and posts a system message after the fact ("🔔 Conversation released due to inactivity"). Agents discover it when they look at their list.**

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Incoming Message Becomes Pending Chat (Priority: P1)

When a customer sends a WhatsApp message to the business for the first time (or after a chat has been reopened), the system creates a contact record (if new) and marks the conversation as "pending" — meaning it is awaiting an agent to take ownership. All agents in the organization see this pending chat appear in their sidebar list, showing the customer's name, phone number, timestamp, and a red badge with the count of unread incoming messages. However, the **content** of the messages is deliberately withheld — agents cannot read what the customer wrote until they formally "claim" the conversation. This privacy gate ensures that no agent can silently read customer messages without taking responsibility for the conversation.

**Why this priority**: This is the foundational trigger for the entire claim-and-collaboration workflow. Without pending state, there is no privacy gate and no claim action. It must work first because every subsequent story depends on it.

**Independent Test**: Send a real WhatsApp message from a test phone to a connected account → verify a new contact appears in the agents' sidebar with `chat_status = pending`, the message count badge is correct, but opening the conversation shows a "Claim to view" screen instead of message content.

**Acceptance Scenarios**:

1. **Given** a customer who has never messaged before, **When** they send a WhatsApp message, **Then** a new contact is created with `chat_status = "pending"` in its metadata, `assigned_user_id` is null, and all agents see it in the sidebar with a red unread badge showing "1".
2. **Given** an existing contact whose conversation was previously closed, **When** the customer sends a new message, **Then** the conversation reverts to `chat_status = "pending"` (if no agent is currently assigned) so it re-enters the queue.
3. **Given** a contact with `chat_status = "pending"` and `assigned_user_id = null`, **When** an agent without `contacts:read` permission opens it, **Then** the system returns a 403 error with `code: "chat_not_claimed"` and a `pending_message_count` field, and the frontend displays the claim screen instead of message content.
4. **Given** a pre-existing contact created before this feature was deployed (no `chat_status` in metadata), **When** the system evaluates its status, **Then** it defaults to `"open"` so backward compatibility is preserved — no existing conversations break.

---

### User Story 2 — Agent Claims a Pending Conversation (Priority: P1)

An agent sees a pending conversation in their sidebar and wants to take ownership of it. They click on the conversation, see the "Claim this chat" screen with the count of waiting messages, and press the "Claim" button. The system assigns the conversation to them (sets `assigned_user_id` to their user ID), changes the status from `pending` to `open`, posts a system message ("🔔 {Agent Name} claimed this conversation") into the chat, and broadcasts a WebSocket event so all other agents' sidebars update in real time. The claiming agent can now read all messages and reply. No other ordinary agent can claim the same conversation — it disappears from their pending queue immediately.

**Why this priority**: Claim is the core action that unlocks the conversation. Without it, messages remain permanently hidden. This must work second because collaboration (Story 3) only makes sense once someone owns the conversation.

**Independent Test**: With a pending conversation visible, click "Claim" → verify `assigned_user_id` changes to the current user, `chat_status` changes to `"open"`, a system message appears, and messages become readable. Then log in as a second agent and verify the conversation no longer appears as claimable.

**Acceptance Scenarios**:

1. **Given** a pending, unassigned conversation and an agent with `chat.assign:write` permission, **When** the agent presses "Claim this chat", **Then** the system sets `assigned_user_id` to the agent, sets `chat_status` to `"open"`, creates a system message, broadcasts a `chat_claimed` WebSocket event, and returns success.
2. **Given** a conversation already assigned to Agent A, **When** Agent B (who lacks `chat.collaborate:write`) attempts to claim it, **Then** the system returns 409 with `code: "already_assigned"` and Agent B sees an error message naming the current owner.
3. **Given** a closed conversation (`chat_status = "closed"`), **When** any agent attempts to claim it, **Then** the system returns 409 with `code: "chat_closed"` instructing them to reopen first.
4. **Given** a conversation that is already assigned to the current agent, **When** they press "Claim" again, **Then** the system returns a success response with `message: "Already assigned to you"` (idempotent) without creating a duplicate system message.
5. **Given** two agents who press "Claim" on the same pending conversation simultaneously, **When** both requests reach the server, **Then** only the first one succeeds; the second receives 409 `already_assigned`. The database ensures atomicity.

---

### User Story 3 — Collaborator Joins an Assigned Conversation (Priority: P2)

After an agent has claimed a conversation, they may need help from a specialist (e.g., an accounting staff member for pricing). A user who holds the `chat.collaborate:write` permission — which is configurable per role from the `/settings/roles` page — can join that conversation as a collaborator without taking ownership away from the primary agent. The collaborator can read all messages, send replies, and see the conversation in their own sidebar. Multiple collaborators can join the same conversation simultaneously. The primary agent sees who has joined via a collaborators bar in the conversation header. Either the collaborator or the primary agent can end the collaboration at any time by pressing "Leave".

**Why this priority**: Collaboration is the key differentiator from a simple claim system. It enables real-world workflows (agent + accountant, agent + supervisor) but depends on Stories 1 and 2 being functional first since collaboration requires an assigned conversation.

**Independent Test**: As Agent A, claim a conversation. Then log in as User B (with `chat.collaborate:write`), open the same conversation, press "Join as collaborator" → verify User B appears in the collaborators list, can read messages, can send replies, and a system message "🔔 {Name} joined the conversation" appears. Then press "Leave" and verify User B is removed.

**Acceptance Scenarios**:

1. **Given** a conversation assigned to Agent A, **When** User B (with `chat.collaborate:write`) opens it and presses "Join as collaborator", **Then** User B is added to the `collaborators` array in the contact's metadata, a system message "🔔 {Name} joined the conversation" is posted, and a `collaborator_joined` WebSocket event is broadcast.
2. **Given** a conversation where User B is already a collaborator, **When** User B opens it again, **Then** they see the messages directly without needing to re-join (they already have access).
3. **Given** a conversation assigned to Agent A with User B as a collaborator, **When** User B presses "Leave", **Then** User B is removed from the `collaborators` array, a system message "🔔 {Name} left the conversation" is posted, and a `collaborator_left` WebSocket event is broadcast. User B can no longer see the conversation in their sidebar (unless they have `contacts:read`).
4. **Given** a conversation assigned to Agent A, **When** Agent A (the primary owner) attempts to "Leave" and they are the last remaining participant, **Then** the system closes the conversation (`chat_status = "closed"`) rather than leaving it orphaned. If other collaborators are still present, the owner can leave and the conversation remains open with remaining collaborators.
5. **Given** a conversation with Agent A as owner and User B as collaborator, **When** a manager/admin removes User B, **Then** User B is removed from the collaborators list and a system message is posted. Agents (non-managers) cannot remove other collaborators.
6. **Given** a conversation assigned to Agent A, **When** a regular agent without `chat.collaborate:write` attempts to join, **Then** the system returns 403 Forbidden and the user sees no "Join" button.
7. **Given** User B has joined as a collaborator and their `chat.collaborate:write` permission is later revoked by an admin, **When** User B tries to join a *new* conversation, **Then** they are denied. However, they remain a collaborator on conversations they already joined until they leave or are removed.

---

### User Story 4 — Admin Configures Collaboration Permissions per Role (Priority: P2)

An administrator manages the organization's roles and permissions from the `/settings/roles` page. They need to create custom roles (e.g., "Accounting Staff", "Senior Agent") and grant or revoke the chat-related permissions: `chat.assign:write` (claim pending chats) and `chat.collaborate:write` (join assigned chats as collaborator). These new permissions must appear automatically in the permission matrix UI alongside existing chat permissions, and changes must take effect immediately for affected users (via the existing permission cache invalidation mechanism).

**Why this priority**: Without role-based control of collaboration, the feature cannot be deployed safely — every agent would be able to join any conversation. This story makes the feature production-ready for real organizational structures.

**Independent Test**: As admin, go to `/settings/roles`, create a new role "Accounting Staff", enable `chat.collaborate:write` but NOT `chat.assign:write`, assign a user to this role → verify that user can join assigned conversations but cannot claim pending ones.

**Acceptance Scenarios**:

1. **Given** the system has been updated with the new permissions, **When** an admin opens `/settings/roles` and edits any role, **Then** the permission matrix displays `chat.assign:write` ("Claim unassigned chats") and `chat.collaborate:write` ("Join assigned chats as a collaborator") under the "Chat" group, alongside the existing `chat:read` and `chat:write`.
2. **Given** an admin creates a custom role "Accounting Staff", **When** they enable `chat.collaborate:write` and `chat:write` but disable `chat.assign:write`, and assign user Sarah to this role, **Then** Sarah can join and reply in assigned conversations but cannot claim pending ones.
3. **Given** the default system role "agent", **When** the system is deployed, **Then** the agent role includes `chat.assign:write` (can claim) but NOT `chat.collaborate:write` (cannot self-join — must be invited).
4. **Given** the default system roles "manager" and "admin", **When** the system is deployed, **Then** both include `chat.collaborate:write` (managers and admins can join any conversation).
5. **Given** an admin changes a role's permissions, **When** the change is saved, **Then** all users with that role experience the updated permissions immediately (within seconds) without needing to log out — the existing Redis-based permission cache invalidation handles this.

---

### Edge Cases

- **Concurrent claims**: Two agents press "Claim" on the same pending conversation within milliseconds. Only the first request succeeds; the second receives a 409 error. The database transaction ensures atomicity.
- **Collaborator permission revoked mid-session**: A user who joined as collaborator has their `chat.collaborate:write` permission revoked. They remain a collaborator on conversations they already joined (the collaborators list persists in metadata) but cannot join new ones. If they leave, they cannot rejoin.
- **Pre-existing contacts (backward compatibility)**: Contacts created before this feature have no `chat_status` in their metadata. The system treats absent status as `"open"` so all existing conversations continue to work exactly as before — no migration needed.
- **Network disconnect during claim**: If the WebSocket broadcast fails after a successful claim, the claiming agent still has local access (their frontend optimistically updates). Other agents will see the updated state on their next page refresh or reconnection.
- **Agent attempts to send a message on a pending (unclaimed) conversation**: The system must prevent this — the send endpoint should check `chat_status` and reject with 403 if the conversation is pending and unassigned.
- **All collaborators leave**: Collaborators can leave freely. If all collaborators leave but the primary agent is still assigned, the conversation stays `open`. If the primary agent (last participant) leaves or unassigns, the conversation MUST close (`chat_status = "closed"`), not remain orphaned. Managers/admins can force-remove any collaborator; agents cannot remove each other.
- **Closed conversation**: A closed conversation (`chat_status = "closed"`) cannot be claimed or joined. It must be reopened first. (Reopen is a separate lifecycle action not part of this feature's MVP scope but the guard must exist.)
- **Customer messages a closed conversation**: When a customer sends a new WhatsApp message to a conversation with `chat_status = "closed"`, the system MUST automatically reopen it: set `chat_status = "pending"`, clear `assigned_user_id` and `collaborators`, and post a system message ("🔔 Conversation reopened by customer"). The message is stored normally. This ensures customers can always reach the business even after a conversation was closed.
- **Customer messages an auto-reverted (pending) conversation**: When a customer sends a new message to a conversation that auto-reverted to `pending`, the conversation stays `pending` (it is already in the correct state). The message is stored normally and the `pending_message_count` increments. No status change needed — the conversation is already awaiting a new claim.
- **Manager removes last collaborator and owner simultaneously**: A manager cannot remove the primary owner via the RemoveCollaborator endpoint — the system MUST reject with 400 `code: "cannot_remove_owner"`. The manager must use the existing `AssignContact` endpoint (with `user_id = nil`) to unassign the owner, which triggers the close-or-revert logic. If a manager removes the last collaborator and the owner is still assigned, the conversation stays `open` with just the owner.
- **AssignContact vs ClaimChat conflict resolution**: The existing `AssignContact` endpoint (`PUT /api/contacts/{id}/assign`) requires `contacts:write` permission (not `chat.assign:write`) and is intended for managerial assignment (assigning a conversation TO someone else). The new `ClaimChat` endpoint requires `chat.assign:write` and is for self-assignment (claiming FOR yourself). Agents with only `chat.assign:write` can claim but cannot assign to others. Agents with `contacts:write` (managers/admins) can do both. This is NOT a conflict — the two endpoints serve different purposes with different permission gates. Both set `chat_status = "open"`.
- **Collaborator account deleted/deactivated**: If a user account is deleted or deactivated while they are listed as a collaborator, their entry remains in the `collaborators` metadata array (it is a snapshot, not a live reference). The frontend SHOULD handle gracefully by showing "Unknown User" if the user no longer exists. A manager can remove stale entries via the RemoveCollaborator endpoint.
- **Inactivity timeout set to 0**: If `chat_inactivity_timeout_hours` is set to 0 in organization settings, the auto-revert worker MUST skip that organization entirely (auto-revert is disabled). This allows organizations that want manual-only lifecycle management to opt out.
- **WebSocket broadcast privacy**: All WebSocket broadcasts for chat lifecycle events (`chat_claimed`, `collaborator_joined`, `collaborator_left`) MUST use `BroadcastToOrg` but payloads MUST contain only: `contact_id`, user IDs/names of involved agents, and `chat_status`. Payloads MUST NOT contain customer phone numbers, message content, or any PII. This prevents uninvolved agents in the same org from learning customer identities via WebSocket events.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST track a three-state lifecycle for each conversation: `pending` (new/unassigned), `open` (assigned to an agent), and `closed` (ended). This status MUST be stored in the contact's existing metadata field (JSONB) — no new database columns or migrations required.
- **FR-002**: When a new incoming WhatsApp message arrives for a contact that has no `assigned_user_id` and no existing `chat_status`, the system MUST automatically set `chat_status` to `"pending"` so the conversation enters the agent queue.
- **FR-003**: The system MUST withhold message content from users who do not have access. Specifically, a user can read a conversation's messages only if at least ONE of these conditions is true: (a) they have the `contacts:read` permission, (b) the conversation is assigned to them (`assigned_user_id` matches their user ID), (c) they are listed as a collaborator in the contact's metadata, or (d) they have the `chat.collaborate:write` permission. Otherwise, if the conversation is `pending` and unassigned, the system MUST return a 403 error with the count of pending messages.
- **FR-004**: The system MUST provide a "Claim" action (`PUT /api/contacts/{id}/claim`) that assigns the conversation to the requesting agent, changes status to `"open"`, creates a system message, and broadcasts a real-time WebSocket event. The requesting user MUST have the `chat.assign:write` permission. The WebSocket broadcast MUST be scoped to the organization but the payload MUST NOT include customer message content or phone number — only `contact_id`, `assigned_to`, `assigned_to_name`, and `chat_status` (to prevent information leakage to uninvolved agents).
- **FR-005**: The Claim action MUST reject with 409 if the conversation is closed, or if it is already assigned to a different agent and the requester lacks `chat.collaborate:write`.
- **FR-006**: The Claim action MUST be idempotent — claiming a conversation already assigned to the same agent returns success without creating a duplicate system message.
- **FR-007**: The system MUST provide a "Join as collaborator" action (`POST /api/contacts/{id}/join`) that adds the requesting user to the conversation's `collaborators` list in metadata, creates a system message, and broadcasts a WebSocket event. The requesting user MUST have the `chat.collaborate:write` permission. A maximum of 10 collaborators are allowed per conversation — joining beyond this limit MUST return 409 with `code: "collaborator_limit_reached"`.
- **FR-008**: The system MUST provide a "Leave" action (`DELETE /api/contacts/{id}/join`) that removes the requesting user from the collaborators list, creates a system message, and broadcasts a WebSocket event. Agents cannot remove each other — each agent self-leaves only. Managers and admins can remove any collaborator via a "Remove" action (`DELETE /api/contacts/{id}/collaborators/{user_id}`). When the last remaining participant (the primary owner) leaves or unassigns, the conversation MUST be set to `chat_status = "closed"` (not left as orphaned open).
- **FR-009**: The system MUST expose two new granular permissions in the roles system: `chat.assign:write` (claim pending conversations) and `chat.collaborate:write` (join assigned conversations as collaborator). These MUST appear automatically in the `/settings/roles` permission matrix UI.
- **FR-010**: The default system role "agent" MUST include `chat.assign:write` but NOT `chat.collaborate:write`. The default roles "manager" and "admin" MUST include both.
- **FR-011**: The contact list API response (`ContactResponse`) MUST include a `chat_status` field and a `collaborators` array so the frontend can render appropriate UI (claim screen, join button, collaborators bar).
- **FR-012**: When a conversation is manually assigned to an agent via the existing `AssignContact` endpoint, the system MUST also set `chat_status` to `"open"` to maintain lifecycle consistency.
- **FR-013**: System messages (claim/join/leave notifications) MUST be stored as regular message records with metadata marking them as system messages (`is_system_message: true`, `system_type: "chat_claimed" | "collaborator_joined" | "collaborator_left"`), so they render naturally in the message timeline.
- **FR-014**: The frontend MUST display three distinct UI states when opening a conversation: (a) "Claim this chat" screen for pending unassigned conversations (showing pending message count), (b) "Join as collaborator" screen for conversations assigned to others when the user has `chat.collaborate:write`, and (c) the normal message view for conversations the user owns or has joined.
- **FR-015**: The frontend MUST display a collaborators bar in the conversation header showing avatars/names of all current collaborators, with a "Leave" button visible to collaborators who are not the primary owner.
- **FR-016**: Pending (unclaimed) conversations MUST remain pending indefinitely — no auto-close, no auto-expiry.
- **FR-017**: Claimed (open) conversations MUST auto-revert to `pending` after a configurable period of inactivity. On revert: `assigned_user_id` is cleared, `collaborators` are cleared, `chat_status` is set to `"pending"`, and a system message is posted ("🔔 Conversation released due to inactivity"). The inactivity timer resets on any incoming message, outgoing message, claim, or collaboration action. The timeout duration MUST be configurable per-organization via the existing Settings page (`Organization.Settings["chat_inactivity_timeout_hours"]` JSONB field, default: 24), inheriting the same settings pattern as `mask_phone_numbers`. A background worker checks periodically (every 5 minutes) and reverts expired conversations. If the timeout is set to 0, auto-revert is disabled for that organization.
- **FR-018**: All lifecycle mutations (claim, join, leave, remove collaborator, auto-revert) MUST be logged via the existing `a.logAudit()` mechanism with the resource type `"chat"`, the contact ID as resource ID, and the action (claimed/joined/left/removed/reverted) for compliance and debugging purposes.
- **FR-019**: The claim screen, join screen, and collaborators bar MUST be accessible: claim/join buttons MUST have `aria-label` attributes, the collaborators bar MUST use `aria-label="Collaborators"`, and keyboard focus MUST move to the claim button when the claim screen appears.

### Key Entities *(include if feature involves data)*

- **ChatStatus**: A three-value enum (`pending`, `open`, `closed`) representing the lifecycle state of a conversation. Stored as a string in the contact's metadata JSONB field under the key `chat_status`. Absent value defaults to `open` for backward compatibility.
- **Collaborator**: A lightweight record representing a non-primary participant in a conversation. Attributes: `user_id` (UUID), `name` (display name), `role` (role name at time of joining), `joined_at` (timestamp). Stored as an array in the contact's metadata JSONB under the key `collaborators`.
- **System Message**: A message record that is auto-generated by the system (not sent by a human or chatbot) to indicate lifecycle events. Identified by `metadata.is_system_message = true` and a `metadata.system_type` value. Rendered visually distinct from regular messages.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of new incoming WhatsApp messages from unassigned contacts result in a conversation with `chat_status = "pending"` within 2 seconds of receipt.
- **SC-002**: An agent can claim a pending conversation and begin reading/replying to messages in under 3 clicks (sidebar click → claim button → messages visible), completing the entire flow in under 5 seconds.
- **SC-003**: Zero unauthorized message reads — no agent without proper permissions (ownership, collaboration, or `contacts:read`) can view the content of a pending conversation's messages.
- **SC-004**: Two or more agents can simultaneously participate in the same conversation (one owner + N collaborators) with all participants seeing messages and replies in real time (under 2 seconds latency via WebSocket).
- **SC-005**: An administrator can create a custom role with collaboration permissions and assign it to a user in under 2 minutes from the `/settings/roles` page, with the permission taking effect immediately without requiring the user to log out.
- **SC-006**: 100% of pre-existing conversations (created before this feature) continue to function identically after deployment — no data migration, no broken chats, no changed behavior.
- **SC-007**: The system correctly handles concurrent claim attempts — when two agents press "Claim" simultaneously, exactly one succeeds and the other receives a clear error message within 1 second.

---

## Assumptions

- The project already has a granular resource-action permission system (`Permission`, `CustomRole`, `role_permissions` many-to-many) with Redis-based caching and WebSocket-based invalidation. New permissions are added as new `Permission` records and resource constants.
- The project already has a WebSocket hub (`WSHub.BroadcastToOrg`) for real-time communication, and the frontend already handles WebSocket messages via a switch-case in `websocket.ts`.
- The project already stores contact data in a `Metadata` JSONB field (used for `is_group_chat`, campaign data, etc.) and this field defaults to `{}` — making it suitable for storing `chat_status` and `collaborators` without schema changes.
- The project already has `assigned_user_id` on the Contact model and a `scopeAssignedContact` function that limits which contacts a user can see. This feature builds on top of that existing scoping.
- The existing `AssignContact` endpoint and `processGowaMessage` webhook handler will be modified (not replaced) to integrate with the new lifecycle.
- The frontend uses Vue 3 + Pinia with an existing `contactsStore` that manages contact/message state. New computed properties and actions will be added to this store.
- Internationalization keys will be added to all 5 supported locales (en, ar, es, and others as configured).
