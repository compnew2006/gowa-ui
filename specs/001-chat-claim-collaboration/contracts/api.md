# API Contracts: Chat Status, Claim & Collaboration

**Feature**: 001-chat-claim-collaboration  
**Date**: 2026-07-12

All endpoints follow the existing fastglue envelope pattern:
- Success: `{"status":"success","data":{...}}`
- Error: `{"status":"error","message":"...","data":null}`

---

## Endpoint 1: Claim Conversation

### `PUT /api/contacts/{id}/claim`

Claims an unassigned (pending) conversation. Assigns it to the requesting agent.

**Auth**: Requires `chat.assign:write` permission  
**Rate**: Subject to global rate limiting

**Path Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Contact ID |

**Request Body**: None (empty)

**Responses**:

#### 200 — Success
```json
{
  "status": "success",
  "data": {
    "contact_id": "uuid",
    "assigned": true,
    "agent_name": "Khaled Ahmed"
  }
}
```

#### 200 — Idempotent (already assigned to you)
```json
{
  "status": "success",
  "data": {
    "contact_id": "uuid",
    "assigned": true,
    "message": "Already assigned to you"
  }
}
```

#### 403 — Missing permission
```json
{
  "status": "error",
  "message": "Insufficient permissions",
  "data": null
}
```

#### 409 — Already assigned to another agent
```json
{
  "status": "error",
  "message": "This chat is already assigned to Omar Ali",
  "data": {
    "current_agent": "Omar Ali",
    "can_join": true
  }
}
```
> `can_join: true` signals the frontend to offer "Join as collaborator" instead.

#### 409 — Chat is closed
```json
{
  "status": "error",
  "message": "Cannot claim a closed chat. Reopen it first.",
  "data": null
}
```

#### 404 — Contact not found
```json
{
  "status": "error",
  "message": "Contact not found",
  "data": null
}
```

**WebSocket Events Emitted**:
- `chat_claimed` → broadcast to org

---

## Endpoint 2: Join as Collaborator

### `POST /api/contacts/{id}/join`

Joins an assigned conversation as a collaborator (helper). Does NOT take ownership.

**Auth**: Requires `chat.collaborate:write` permission

**Path Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Contact ID |

**Request Body**: None (empty)

**Responses**:

#### 200 — Success
```json
{
  "status": "success",
  "data": {
    "contact_id": "uuid",
    "collaborator": true,
    "user_name": "Sarah Ahmed"
  }
}
```

#### 200 — Already a collaborator
```json
{
  "status": "success",
  "data": {
    "message": "You are already a collaborator"
  }
}
```

#### 200 — Already the primary owner
```json
{
  "status": "success",
  "data": {
    "message": "You are the primary owner of this conversation"
  }
}
```

#### 403 — Missing permission
```json
{
  "status": "error",
  "message": "Insufficient permissions",
  "data": null
}
```

**WebSocket Events Emitted**:
- `collaborator_joined` → broadcast to org

---

## Endpoint 3: Leave Conversation

### `DELETE /api/contacts/{id}/join`

Removes the requesting user from the collaborators list.

**Auth**: Authenticated user (no specific permission — any collaborator can leave)

**Path Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Contact ID |

**Request Body**: None

**Responses**:

#### 200 — Success
```json
{
  "status": "success",
  "data": {
    "contact_id": "uuid",
    "left": true
  }
}
```

#### 400 — Not a collaborator
```json
{
  "status": "error",
  "message": "You are not a collaborator on this conversation",
  "data": null
}
```

#### 403 — Primary owner cannot leave
```json
{
  "status": "error",
  "message": "Primary owner cannot leave. Unassign the conversation instead.",
  "data": null
}
```

**WebSocket Events Emitted**:
- `collaborator_left` → broadcast to org

---

## Modified Endpoint: Get Messages (Privacy Guard)

### `GET /api/contacts/{id}/messages` (existing — modified)

New behavior: returns 403 if the conversation is pending, unassigned, and the user lacks access permissions.

**New 403 Response**:
```json
{
  "status": "error",
  "message": "Claim this chat to view messages",
  "data": {
    "pending_message_count": 3
  }
}
```

**Access Logic** (evaluated in order):
1. User has `contacts:read` → ✅ allow
2. `contact.assigned_user_id == userID` → ✅ allow
3. User is in `contact.Metadata["collaborators"]` → ✅ allow
4. User has `chat.collaborate:write` → ✅ allow (can join)
5. `chat_status == "pending"` && `assigned_user_id == nil` → ❌ deny (403)
6. Otherwise → ✅ allow (backward compat)

---

## Modified Endpoint: Contact List (new response fields)

### `GET /api/contacts` (existing — response extended)

Two new fields added to each `ContactResponse` object in the response:

```json
{
  "id": "uuid",
  "phone_number": "+966501234567",
  "name": "Ahmed Mohammed",
  "chat_status": "pending",
  "collaborators": [],
  "assigned_user_id": null,
  "unread_count": 1,
  "...": "..."
}
```

---

## Endpoint 4: Remove Collaborator (Manager Kick)

### `DELETE /api/contacts/{id}/collaborators/{user_id}`

Removes a specific collaborator from the conversation. Only managers/admins can do this.

**Auth**: Requires `chat.collaborate:write` permission

**Path Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Contact ID |
| `user_id` | UUID | The collaborator to remove |

**Responses**:

#### 200 — Success
```json
{
  "status": "success",
  "data": {
    "contact_id": "uuid",
    "removed_user_id": "uuid",
    "removed_by": "Manager Name"
  }
}
```

#### 400 — Target is not a collaborator
```json
{
  "status": "error",
  "message": "That user is not a collaborator on this conversation",
  "data": null
}
```

#### 403 — Cannot remove the primary owner
```json
{
  "status": "error",
  "message": "Cannot remove the primary owner. Unassign or close the conversation instead.",
  "data": null
}
```

**WebSocket Events Emitted**:
- `collaborator_left` → broadcast to org

---

## Endpoint 5: Close Conversation (Owner Leave as Last Participant)

When the primary owner leaves and no collaborators remain, the conversation closes automatically. This is handled by `LeaveChat` (`DELETE /api/contacts/{id}/join`) — no separate endpoint.

**Behavior**:
- If owner calls leave AND `len(collaborators) == 0` → set `chat_status = "closed"`, clear `assigned_user_id`, post "🔔 Conversation closed".
- If owner calls leave AND collaborators remain → clear `assigned_user_id`, collaborators stay, post "🔔 {Owner} left. {N} collaborators remain."
- If non-owner collaborator calls leave → standard self-leave (removed from collaborators).

---

## Auto-Revert (Background Worker — no API endpoint)

A background worker runs every 5 minutes and reverts inactive open conversations to pending.

**Config**: `Organization.Settings["chat_inactivity_timeout_hours"]` (default: 24)  
**Condition**: `chat_status = 'open'` AND `assigned_user_id IS NOT NULL` AND `last_message_at < NOW() - timeout`  
**Action**: Clear assignee + collaborators, set `pending`, post system message "🔔 Conversation released due to inactivity", broadcast WebSocket `chat_reverted`.

---

## WebSocket Message Types

### `chat_claimed`

Broadcast to org when an agent claims a conversation.

```json
{
  "type": "chat_claimed",
  "payload": {
    "contact_id": "uuid",
    "assigned_to": "user-uuid",
    "assigned_to_name": "Khaled Ahmed",
    "chat_status": "open"
  }
}
```

### `collaborator_joined`

Broadcast to org when a user joins as collaborator.

```json
{
  "type": "collaborator_joined",
  "payload": {
    "contact_id": "uuid",
    "user_id": "user-uuid",
    "user_name": "Sarah Ahmed",
    "user_role": "Accounting Staff"
  }
}
```

### `collaborator_left`

Broadcast to org when a collaborator leaves (self-leave OR manager kick).

```json
{
  "type": "collaborator_left",
  "payload": {
    "contact_id": "uuid",
    "user_id": "user-uuid",
    "user_name": "Sarah Ahmed",
    "removed_by": "Manager Name"
  }
}
```
> `removed_by` is present only when a manager kicked the collaborator. Absent for self-leave.

---

### `chat_reverted`

Broadcast to org when the inactivity worker reverts an open conversation to pending.

```json
{
  "type": "chat_reverted",
  "payload": {
    "contact_id": "uuid",
    "reason": "inactivity",
    "previous_assignee": "user-uuid"
  }
}
```

---

## Permission Matrix

| Permission Key | Resource | Action | agent | manager | admin | Custom Roles |
|----------------|----------|--------|-------|---------|-------|-------------|
| `chat:read` | `chat` | `read` | ✅ | ✅ | ✅ | configurable |
| `chat:write` | `chat` | `write` | ✅ | ✅ | ✅ | configurable |
| `chat.assign:write` | `chat.assign` | `write` | ✅ (NEW) | ✅ | ✅ | configurable |
| `chat.collaborate:write` | `chat.collaborate` | `write` | ❌ | ✅ (NEW) | ✅ | configurable |

New permissions appear automatically in `GET /api/permissions` and render in the PermissionMatrix component at `/settings/roles`.
