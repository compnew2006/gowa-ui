## Cross-Instance Chat Assignment — Gap Analysis (2026-05-18)

### User Request
Allow assigning a chat to an agent who does NOT have the WhatsApp instance in their `send_restrictions.allowed_instance_ids`. The assigned agent should be able to VIEW the chat (read messages) but NOT send messages (since they lack instance access).

### What Already Works (Bridge Rule)
`applyRestrictedInstanceVisibilityFilter` in `chat_access_policy.go` already has a "bridge rule":
```sql
(instance_id IN ? OR assigned_user_id = ?)
```
This means an assigned agent CAN already see the chat in contact lists and read messages, even without instance access.

### Gaps to Fix

#### Gap 1 — Frontend: Assignment dialog filters out users without instance access
**File**: `frontend/src/views/chat/ChatView.vue:1245-1250`
```js
const assignableUsers = computed(() => {
  const instanceId = contactsStore.currentContact?.instance_id?.trim();
  return usersStore.users
    .filter((u) => u.is_active !== false)
    .filter((u) => canUserAccessInstance(u, instanceId));  // <-- BLOCKS users
});
```
**Fix**: Remove or relax the `canUserAccessInstance` filter. Show ALL active org users. Optionally add a visual indicator (badge/icon) for users without instance access, so the admin knows they can only view (not send).

#### Gap 2 — Frontend: ContactInfoPanel collaborator invite also filters
**File**: `frontend/src/components/chat/ContactInfoPanel.vue:360-363`
Same `canUserAccessInstance` filter for collaborator invites. Same fix needed.

#### Gap 3 — Backend: `AssignContact` blocks agents without instance access (for non-admins)
**File**: `internal/handlers/contacts_management.go:215-220`
```go
if req.UserID != nil && *req.UserID != uuid.Nil && !a.canBypassPendingChatRestriction(userID, orgID) {
    allowed, err := a.canUserSeeContactInstance(orgID, *req.UserID, contact)
    // Returns 403 if agent doesn't have instance access
}
```
**Fix**: Remove the instance access check entirely, or only apply it when the assigner is also not an admin/manager. The bridge rule already handles read access; sending is already gated by `resolveOutboundInstance`.

#### Gap 4 — Backend: `AssignAgentTransfer` blocks agents without instance access
**File**: `internal/handlers/agent_transfers.go:890-896`
```go
allowed, err := a.canUserSeeContactInstance(orgID, *targetAgentID, transfer.Contact)
if !allowed {
    return 403 "Assignee does not have access to this WhatsApp account"
}
```
**Fix**: Same — remove or bypass the instance check. The bridge rule handles visibility.

#### Gap 5 — Backend: `CreateAgentTransfer` (transfer to agent) also blocks
**File**: `internal/handlers/agent_transfers.go:562-564`
Uses `validateTransferAssigneeAccess` which calls `canUserSeeContactInstance`. Same fix.

#### Gap 6 — Backend: Auto-assignment fallback in `assignToTeam` / webhook flow
**File**: `internal/handlers/agent_transfers.go:1315`
Auto-assignment via `validateTransferAssigneeAccess` silently falls back to queue when agent lacks instance access. Should allow the assignment.

#### Gap 7 — Backend: `InviteContactCollaborator` blocks by instance
**File**: `internal/handlers/contact_collaborators.go:175-180`
Same `canUserSeeContactInstance` check blocks inviting collaborators without instance access.

#### Gap 8 — Backend: Transfer list visibility has NO bridge rule
**File**: `internal/handlers/agent_transfers.go:104-109`
`applyTransferRestrictedInstanceVisibilityFilter` uses `contacts.instance_id IN ?` WITHOUT the bridge rule (`OR assigned_user_id = ?`). Agents without instance access won't see their assigned transfers in the transfers list.

### NOT a Gap (already handled)
- **Sending messages**: `resolveOutboundInstance` doesn't check user's send restrictions — it resolves the instance from contact's `instance_id`. This is correct; the send restriction is enforced separately via `send_restriction_policy.go`. Agents without instance access will still be able to send via the contact's instance. **This may need a separate discussion** — should agents without instance access be blocked from SENDING even if they can view?
- **Chat visibility after assignment**: The bridge rule in `applyRestrictedInstanceVisibilityFilter` already grants read access to assigned chats.
- **WebSocket updates**: `canSubscribeToContactUpdates` in `websocket.go:153` already uses the bridge rule.

---

## Multi-Agent Concurrent Access — Can 2 agents work on the same chat?

### Current Architecture

**1. `assigned_user_id` — SINGLE owner**
- `Contact` model has ONE `assigned_user_id` (nullable, single UUID)
- `chatAssignmentUpdates()` overwrites the field — assigning Agent B removes Agent A
- `applyAgentVisibleChatAccessFilter` shows chat to `assigned_user_id = ?` — only ONE user

**2. `contact_collaborators` — EXISTING multi-user system**
- `ContactCollaborator` model: many-to-many (contact_id + user_id), roles: `viewer` / `assistant`, statuses: `invited` / `accepted` / `declined`
- `applyAgentVisibleChatAccessFilter` already grants access: `EXISTS (SELECT 1 FROM contact_collaborators cc WHERE cc.contact_id = contacts.id AND cc.user_id = ? AND cc.status IN ('invited','accepted'))`
- Collaborators CAN see chat in sidebar, read messages, AND send messages
- No hard limit on number of collaborators per chat

### Answer: Partially yes — via collaborators

The collaborator system already supports multiple agents. Agent A = `assigned_user_id` (owner), Agent B/C/D = collaborators with `assistant` role. But it's blocked by instance access gaps and the UX is hidden.

### Additional Gaps for Multi-Agent

| # | Layer | File:Line | What's blocked |
|---|-------|-----------|----------------|
| **G9** | Frontend | `ChatView.vue:6031-6111` | Assignment dialog only replaces `assigned_user_id`. No "add co-agent" option. |
| **G10** | Frontend | `ChatView.vue` chat header | No visual indicator of multiple active agents. Only shows single `assigned_user_name`. |
| **G11** | Backend | `contacts_management.go:234-237` | `AssignContact` always overwrites `assigned_user_id`. No "co-assignment". |
| **G12** | Backend | `agent_transfers.go:930-937` | `AssignAgentTransfer` also overwrites `assigned_user_id`. |
| **G13** | Backend | `contact_collaborators.go` | Only `viewer` and `assistant` roles. No `co_owner` or `co_agent` role to distinguish from simple assistants. |
| **G14** | Frontend | `ContactInfoPanel.vue:657-751` | Collaborator system is buried in right sidebar panel. No prominent "add agent" action in chat header bar. |
| **G15** | Backend | `contact_collaborators.go:175-180` | Collaborator invite blocked by instance access (same as Gap 7). |
| **G16** | Frontend | `ContactInfoPanel.vue:363` | Collaborator invite dropdown filters by instance access (same as Gap 2). |
