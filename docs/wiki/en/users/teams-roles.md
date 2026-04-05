---
title: Teams & Roles
---

# Teams & Roles

Manage your organization's team structure, define roles with granular permissions, and control access to features through Role-Based Access Control (RBAC).

## Teams Management

### List Teams

**Endpoint:** `GET /api/teams`

View all teams in your organization. Each team entry includes the member count.

### Create Team

**Endpoint:** `POST /api/teams`

```json
{
  "name": "Support Team",
  "description": "Handles customer support inquiries",
  "member_ids": ["user-uuid-1", "user-uuid-2"]
}
```

**What happens:**

1. A team record is created
2. Team member records are created for each specified user
3. Members can be assigned chats as a group

### Update Team

**Endpoint:** `PUT /api/teams/{id}`

Modify team name, description, or membership.

### Delete Team

**Endpoint:** `DELETE /api/teams/{id}`

Remove a team. Member records are cleaned up automatically.

### Manage Team Members

#### List Members

**Endpoint:** `GET /api/teams/{id}/members`

#### Add Members

**Endpoint:** `POST /api/teams/{id}/members`

```json
{
  "user_ids": ["user-uuid-3", "user-uuid-4"]
}
```

#### Remove Members

**Endpoint:** `DELETE /api/teams/{id}/members`

```json
{
  "user_ids": ["user-uuid-1"]
}
```

## Roles & Permissions (RBAC)

Whatomate uses Role-Based Access Control to manage what users can do. Each role has a set of permissions defined as `resource:action` pairs.

### Built-in Roles

| Role | Description |
|------|-------------|
| **Admin** | Full access to all features and settings |
| **Manager** | Manage team members, view analytics, handle escalations |
| **Agent** | Handle assigned chats, send messages, use canned responses |

### List Roles

**Endpoint:** `GET /api/roles`

Returns all roles for your organization, including system roles and custom roles, with their permissions preloaded.

### Create Custom Role

**Endpoint:** `POST /api/roles`

```json
{
  "name": "Support Lead",
  "is_default": false,
  "permissions": [
    { "resource": "contacts", "action": "read" },
    { "resource": "contacts", "action": "write" },
    { "resource": "messages", "action": "read" },
    { "resource": "messages", "action": "write" },
    { "resource": "campaigns", "action": "read" },
    { "resource": "analytics", "action": "read" }
  ]
}
```

| Field | Description |
|-------|-------------|
| **name** | Role name (must be unique within the organization) |
| **is_default** | If true, new users are assigned this role by default |
| **permissions** | Array of resource:action permission pairs |

### Available Resources and Actions

| Resource | Actions |
|----------|---------|
| `users` | `read`, `write`, `delete` |
| `roles` | `read`, `write`, `delete` |
| `contacts` | `read`, `write`, `delete` |
| `messages` | `read`, `write` |
| `campaigns` | `read`, `write`, `delete` |
| `templates` | `read`, `write`, `delete` |
| `accounts` | `read`, `write`, `delete` |
| `api_keys` | `read`, `write`, `delete` |
| `canned_responses` | `read`, `write`, `delete` |
| `flows_chatbot` | `read`, `write` |
| `webhooks` | `read`, `write`, `delete` |
| `analytics` | `read` |

### Update Role

**Endpoint:** `PUT /api/roles/{id}`

Modify a role's name or permissions:

```json
{
  "name": "Senior Support Lead",
  "permissions": [
    { "resource": "contacts", "action": "read" },
    { "resource": "contacts", "action": "write" },
    { "resource": "messages", "action": "read" },
    { "resource": "messages", "action": "write" },
    { "resource": "analytics", "action": "read" },
    { "resource": "campaigns", "action": "read" },
    { "resource": "campaigns", "action": "write" }
  ]
}
```

**What happens:**

1. The role name is updated (if provided)
2. All existing permissions are deleted
3. New permissions are created from the request
4. The role permissions cache is invalidated

### Delete Role

**Endpoint:** `DELETE /api/roles/{id}`

Remove a custom role. **Restrictions:**

- System roles (admin, agent, manager) cannot be deleted
- Users assigned to the deleted role are reassigned to the default role

### List Permissions

**Endpoint:** `GET /api/permissions`

Returns all available `resource:action` pairs that can be assigned to roles.

## Permission System

### How Permissions Are Checked

1. When a user makes an API request, the auth middleware extracts their role
2. Role permissions are loaded from cache (via `GetRolePermissionsCached()`)
3. The handler checks if the user's role includes the required `resource:action` permission
4. If the permission is missing, a 403 Forbidden response is returned

### Permission Caching

Role permissions are cached for performance:

- Cache TTL: 10 minutes
- Cache is invalidated when a role is created, updated, or deleted
- Cache is loaded from the database on first access

### Agent Role Chat Scoping

Users with the **Agent** role have additional visibility restrictions beyond standard permissions:

| Restriction | Effect |
|-------------|--------|
| **Chat visibility** | Only see chats assigned to them, public chats, or chats where they are collaborators |
| **Message sending** | Must have access to the contact; cannot send to chats they cannot see |
| **Unclaimed chats** | Cannot view or send to unclaimed chats unless explicitly configured |

This means an agent with `contacts:read` permission still only sees their assigned contacts, not all contacts in the organization.

### Send Restrictions

Organizations can enforce additional send restrictions per user:

| Setting | Description |
|---------|-------------|
| **enabled** | Toggle send restrictions for this user |
| **include_all_contacts** | Allow sending to all contacts or only authorized numbers |
| **authorized_numbers** | Whitelist of phone numbers the user can message |
| **allowed_instance_ids** | Which WhatsApp instances the user can send from |
| **prefix_agent_name** | Auto-prefix messages with the agent's name |
| **allow_unclaimed_chat_view** | Allow viewing unassigned chats |
| **allow_unclaimed_chat_send** | Allow sending to unassigned chats |

**Update User Send Restrictions:**

**Endpoint:** `PUT /api/users/{id}/send-restrictions`

```json
{
  "send_restrictions": {
    "enabled": true,
    "include_all_contacts": false,
    "authorized_numbers": ["+1234567890", "+0987654321"],
    "allowed_instance_ids": ["instance-uuid-1"],
    "prefix_agent_name": true,
    "allow_unclaimed_chat_view": false,
    "allow_unclaimed_chat_send": false
  }
}
```

### Outbound Mode

Organizations can control outbound messaging at the organization level:

| Mode | Description |
|------|-------------|
| **inbound_only** | Users can only reply to inbound messages; proactive outbound is blocked |
| **mixed** | Both inbound replies and proactive outbound messages are allowed |

### Strict Rollout Mode

When enforcing send restrictions, organizations can choose:

| Mode | Behavior |
|------|----------|
| **audit** | Log violations but allow messages to send |
| **enforce** | Block messages that violate restrictions |

## See Also

- [Authentication & User Settings](authentication.md) — User management and settings
- [Tags & Organization](tags-organization.md) — Organization settings and members
- [Chat & Messaging](chat-messaging.md) — Agent chat scoping and restrictions
- [Analytics](analytics.md) — Agent performance metrics
