# Data Model: Chat Status, Claim & Collaboration

**Feature**: 001-chat-claim-collaboration  
**Date**: 2026-07-12

---

## No New Tables, No New Columns

This feature uses **existing JSONB Metadata fields** exclusively, per Constitution Principle 8. No GORM model changes, no migrations.

---

## Entity: ChatStatus (Metadata-stored enum)

Stored in `Contact.Metadata["chat_status"]` as a string.

| Value | Meaning | Set When |
|-------|---------|----------|
| `"pending"` | New/unassigned — awaiting agent claim | Incoming message arrives for unassigned contact |
| `"open"` | Assigned to an agent — actively handled | Agent claims, or manual assignment |
| `"closed"` | Ended — read-only | (Future: close action — guard only for now) |
| *(absent)* | Pre-existing contact — defaults to `"open"` | Backward compatibility |

**Helper methods on `Contact`** (new file `internal/models/chat_status.go`):

```go
func (c *Contact) EffectiveStatus() ChatStatus     // reads from Metadata, defaults to open
func (c *Contact) SetStatus(s ChatStatus)            // writes to Metadata
```

---

## Entity: Collaborator (Metadata-stored array element)

Stored in `Contact.Metadata["collaborators"]` as a JSON array.

```json
[
  {
    "user_id": "uuid-string",
    "name": "Sarah Ahmed",
    "role": "Accounting Staff",
    "joined_at": "2026-07-12T10:05:00Z"
  }
]
```

**Helper methods on `Contact`**:

```go
func (c *Contact) GetCollaborators() []Collaborator      // parse from Metadata
func (c *Contact) IsCollaborator(userID string) bool      // membership check
func (c *Contact) AddCollaborator(user Collaborator)      // append (dedup)
func (c *Contact) RemoveCollaborator(userID string)        // filter out
```

**Collaborator struct** (in `chat_status.go`):

```go
type Collaborator struct {
    UserID   string    `json:"user_id"`
    Name     string    `json:"name"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joined_at"`
}
```

---

## Entity: System Message (Message with metadata flag)

No schema change. Uses existing `Message` model with metadata markers:

```json
{
  "metadata": {
    "is_system_message": true,
    "system_type": "chat_claimed"
  }
}
```

**System types**:
| `system_type` | Content Pattern | Trigger |
|---------------|----------------|---------|
| `"chat_claimed"` | `"🔔 {Name} claimed this conversation"` | ClaimChat handler |
| `"collaborator_joined"` | `"🔔 {Name} joined the conversation"` | JoinChat handler |
| `"collaborator_left"` | `"🔔 {Name} left the conversation"` | LeaveChat handler |

---

## Entity: Permissions (2 new Permission records)

New resource constant + DefaultPermissions entries. Seeded automatically by `SeedPermissionsAndRoles`.

| Resource Constant | Value | New Permission Entry |
|-------------------|-------|---------------------|
| `ResourceChatCollaborate` | `"chat.collaborate"` | `{Resource: "chat.collaborate", Action: "write", Description: "Join assigned chats as a collaborator"}` |

Existing `ResourceChatAssign = "chat.assign"` is already seeded with `{Action: "write", Description: "Assign conversations to agents"}`.

**System role distribution** (in `SystemRolePermissions()`):

| Permission | agent | manager | admin |
|------------|-------|---------|-------|
| `chat.assign:write` | ✅ (ADD) | ✅ (exists) | ✅ (all) |
| `chat.collaborate:write` | ❌ | ✅ (ADD) | ✅ (all) |

---

## State Transition Diagram

```
                    ┌──────────────────────────────────────────┐
                    │                                          │
                    ▼                                          │
              ┌──────────┐     claim()      ┌──────────┐       │
  new msg ──► │ pending  │ ──────────────► │   open   │       │
              └──────────┘                  └──────────┘       │
                    ▲                           │              │
                    │                           │ collaborate  │
                    │                           ▼              │
                    │                    ┌──────────────┐      │
                    │                    │ open + collab│      │
                    │                    └──────────────┘      │
                    │                           │              │
                    │           ┌───────────────┤              │
                    │           │               │              │
                    │      unassign()      owner leaves        │
                    │      (or auto-       as LAST participant │
                    │       revert)              │             │
                    │           │               ▼             │
                    │           │          ┌──────────┐        │
                    └───────────┘          │  closed  │        │
                                              └──────────┘        │
                                              reopen() ──────────┘
                                              → pending (if unassigned)
```

**Transitions**:

| From | Event | To | Guard |
|------|-------|-----|-------|
| *(absent)* | incoming msg + unassigned | `pending` | `assigned_user_id == nil` |
| `pending` | claim() | `open` | `chat.assign:write` + not assigned to other |
| `open` | unassign() | `pending` | clears collaborators |
| `open` | join() | `open` + collaborator added | `chat.collaborate:write` |
| `open` + collab | collaborator self-leaves | `open` (collab removed) | must be collaborator |
| `open` + collab | manager kicks collaborator | `open` (collab removed) | `chat.collaborate:write` |
| `open` | owner leaves AS LAST participant | `closed` | no collaborators remain |
| `open` + collab | owner leaves with collabs remaining | `open` (owner cleared, collabs stay) | collaborators still present |
| `open` | **auto-revert (inactivity timeout)** | `pending` | `last_message_at < NOW() - timeout` |
| `closed` | reopen() | `pending` (if unassigned) or `open` | future — guard only |
| any | claim() on closed | *(rejected)* | 409 `chat_closed` |
| `open` | assign to agent | `open` | `AssignContact` sets status |


---

## Response DTO Changes

### ContactResponse (add 2 fields)

```go
// Existing struct at contacts.go:28 — add:
ChatStatus    string                `json:"chat_status,omitempty"`
Collaborators []models.Collaborator `json:"collaborators,omitempty"`
```

### MessageResponse (no change needed)

System messages flow through the existing `buildMessagesResponse` unchanged — the `metadata` is already exposed via the existing `Metadata any` path (indirectly, for reactions etc.). The frontend can read `metadata.is_system_message` from the raw payload.

---

## Validation Rules

| Rule | Enforcement Point |
|------|-------------------|
| `chat_status` must be one of: pending, open, closed | `SetStatus()` method (Go) |
| Collaborator `user_id` must be a valid UUID | `JoinChat` handler (DB lookup) |
| Collaborator `user_id` must be in same org | `JoinChat` handler (org-scoped query) |
| Cannot claim a closed conversation | `ClaimChat` handler guard |
| Cannot leave if you're the primary owner | `LeaveChat` handler guard |
| Cannot join without `chat.collaborate:write` | `JoinChat` → `requireAuth` |
| Cannot claim without `chat.assign:write` | `ClaimChat` → `requireAuth` |
| Cannot kick without `chat.collaborate:write` | `RemoveCollaborator` → `requireAuth` |
| Auto-revert timeout must be > 0 | Settings validation (min 1 hour) |

---

## Entity: Chat Inactivity Worker (Background goroutine)

A periodic background worker that checks for inactive open conversations and reverts them to pending.

**Configuration** (stored in `Organization.Settings` JSONB):
```json
{
  "chat_inactivity_timeout_hours": 24
}
```

**Behavior**:
- Runs every 5 minutes (hardcoded interval — not configurable).
- For each organization, reads `Settings["chat_inactivity_timeout_hours"]` (default: 24).
- Queries contacts where `metadata->>'chat_status' = 'open'` AND `assigned_user_id IS NOT NULL` AND `last_message_at < NOW() - (timeout * interval '1 hour')`.
- For each match: clears `assigned_user_id`, clears `collaborators`, sets `chat_status = "pending"`, posts system message, broadcasts WebSocket.
- **Silent**: no pre-warning. The system message appears after the revert.

**Started in**: `cmd/whatomate/main.go` alongside other workers:
```go
go app.StartChatInactivityWorker(ctx)
```

---

## Entity: Manager Kick Action

Allows managers/admins to remove any collaborator from a conversation.

**Route**: `DELETE /api/contacts/{id}/collaborators/{user_id}`  
**Permission**: `chat.collaborate:write` (managers and admins have this; agents do not)

**Behavior**:
- Removes the specified `user_id` from `contact.Metadata["collaborators"]`.
- Posts system message: "🔔 {Manager} removed {User} from the conversation".
- Broadcasts `collaborator_left` WebSocket event.
- Cannot remove the primary owner (must use unassign or close instead).
