---
title: Tags & Organization
---

# Tags & Organization

Manage tags for organizing contacts, configure organization settings, manage members, handle lead requests, add conversation notes, and import/export data.

## Tags Management

Tags help you categorize and filter contacts (e.g., `vip`, `lead`, `support`, `billing`).

### List Tags

**Endpoint:** `GET /api/tags`

Returns all tags for your organization with usage counts:

```json
[
  { "name": "vip", "color": "#FF5733", "contact_count": 42 },
  { "name": "lead", "color": "#33C4FF", "contact_count": 128 }
]
```

Tags are sorted alphabetically by name.

### Create Tag

**Endpoint:** `POST /api/tags`

```json
{
  "name": "priority",
  "color": "#FF0000"
}
```

**Validation:**

- Tag names must be unique within the organization
- Color is optional; a default color is assigned if not provided

### Update Tag

**Endpoint:** `PUT /api/tags/{name}`

Change a tag's name or color. All contacts with the old tag name are updated automatically.

### Delete Tag

**Endpoint:** `DELETE /api/tags/{name}`

**What happens:**

1. The tag is removed from all contacts that have it
2. The tag record is deleted
3. This action cannot be undone

### Using Tags with Contacts

Tags are applied when creating or updating contacts:

```json
{
  "name": "John Doe",
  "phone_number": "+1234567890",
  "tags": ["vip", "customer"]
}
```

When filtering contacts, you can match by:

| Mode | Description |
|------|-------------|
| **any** | Contact has at least one of the specified tags |
| **all** | Contact has all of the specified tags |
| **has** | Contact has any tags at all |

## Organization Management

### List Organizations

**Endpoint:** `GET /api/organizations`

Returns all organizations where you have membership.

### Create Organization

**Endpoint:** `POST /api/organizations`

```json
{
  "name": "Acme Corp"
}
```

**What happens:**

1. The organization is created
2. Default roles (admin, agent, manager) are created automatically
3. The creator is added as an admin member
4. Default chatbot settings are initialized

### Current Organization

**Endpoint:** `GET /api/organizations/current`

Returns the organization you are currently active in.

### Delete Organization

**Endpoint:** `DELETE /api/organizations/{id}`

> **Warning:** Requires super admin privileges. This soft-deletes the organization and cascades to all related records (users, accounts, campaigns, etc.).

## Organization Settings

**Endpoints:** `GET/PUT /api/org/settings`

Configure organization-wide settings:

| Setting | Description |
|---------|-------------|
| **Timezone** | Default timezone for scheduling and reporting |
| **Business hours** | Operating hours for chatbot automation |
| **Default language** | Default language for templates and messages |
| **Strict sending restrictions** | Enable/disable send restriction enforcement |
| **Outbound mode** | `inbound_only` or `mixed` |
| **Campaign draft only** | Restrict campaigns to draft mode |
| **Strict rollout mode** | `audit` (log) or `enforce` (block) violations |

### Update Settings

```json
{
  "timezone": "America/New_York",
  "business_hours": {
    "monday": { "open": "09:00", "close": "17:00", "enabled": true },
    "tuesday": { "open": "09:00", "close": "17:00", "enabled": true }
  },
  "default_language": "en",
  "strict_sending_restrictions_enabled": true,
  "outbound_mode": "mixed"
}
```

## Organization Members

**Endpoints:** `GET/POST/PUT/DELETE /api/organizations/members`

Manage who belongs to your organization.

### List Members

View all members with their roles and membership status.

### Add Member

Add a user to the organization with a specific role:

```json
{
  "user_id": "user-uuid",
  "role_id": "role-uuid"
}
```

### Update Member Role

Change a member's role within the organization.

### Remove Member

Remove a user from the organization. The user account itself is not deleted — they simply lose access to this organization.

## Lead Requests

Collect leads from public-facing widgets on your website.

### Create Public Lead Request

**Endpoint:** `POST /api/public/lead-requests`

```json
{
  "name": "Jane Smith",
  "email": "jane@example.com",
  "phone": "+1234567890",
  "message": "I'd like to learn more about your products.",
  "widget_id": "widget-uuid"
}
```

**What happens:**

1. The lead request is created
2. Organization admins are notified via WebSocket
3. An outbound webhook is dispatched (if configured)

### List Lead Requests

**Endpoint:** `GET /api/lead-requests`

View all lead requests for your organization.

### Update Lead Request Status

**Endpoint:** `PUT /api/lead-requests/{id}/status`

```json
{
  "status": "contacted"
}
```

| Status | Description |
|--------|-------------|
| **new** | Fresh lead, not yet contacted |
| **contacted** | Lead has been reached out to |
| **converted** | Lead became a customer |
| **rejected** | Lead was not a fit |

## Notifications

**Endpoint:** `GET /api/notifications`

View your notifications, filtered by type and read status.

### Dismiss Notification

**Endpoint:** `PUT /api/notifications/{id}/dismiss`

Mark a notification as dismissed.

### Notification Types

| Type | Triggered By |
|------|--------------|
| **Contact assigned** | A contact is assigned to you |
| **Chat transfer** | A chatbot transfer is waiting |
| **SLA breach** | A chat has breached its SLA |
| **Lead request** | A new lead has been submitted |
| **Instance status** | An instance connection changed |

## Conversation Notes

Add internal notes to contacts that are visible to your team but not sent to the contact.

### List Notes

**Endpoint:** `GET /api/contacts/{id}/notes`

### Create Note

**Endpoint:** `POST /api/contacts/{id}/notes`

```json
{
  "content": "Customer prefers email communication over phone calls."
}
```

Notes are automatically tagged with the author's user ID and timestamp.

### Update Note

**Endpoint:** `PUT /api/contacts/{id}/notes/{note_id}`

Only the note author or an admin can modify a note.

### Delete Note

**Endpoint:** `DELETE /api/contacts/{id}/notes/{note_id}`

Only the note author or an admin can delete a note.

## Custom Actions

Define custom HTTP actions that can be triggered from chatbot flows or manually.

### List Custom Actions

**Endpoint:** `GET /api/custom-actions`

### Create Custom Action

**Endpoint:** `POST /api/custom-actions`

```json
{
  "name": "Create Support Ticket",
  "url": "https://api.example.com/tickets",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer secret-token"
  },
  "body_template": "{\"contact\": \"{{contact.phone}}\", \"message\": \"{{message.content}}\"}",
  "events": ["chatbot_no_match"]
}
```

| Field | Description |
|-------|-------------|
| **name** | Human-readable action name |
| **url** | Target URL (validated with SSRF protection) |
| **method** | HTTP method (GET, POST, PUT, DELETE) |
| **headers** | Custom headers (sensitive values are encrypted) |
| **body_template** | Request body with placeholder support |
| **events** | Events that trigger this action |

### Execute Custom Action

**Endpoint:** `POST /api/custom-actions/{id}/execute`

Manually trigger a custom action. Template variables are resolved and the HTTP request is sent.

### Custom Action Redirect

**Endpoint:** `GET /api/custom-actions/redirect/{token}`

Validate a one-time token and redirect to the configured URL. The token is invalidated after use.

## Import/Export Data

### Export Data

**Endpoint:** `POST /api/export`

```json
{
  "table": "contacts",
  "filters": { "tags": ["vip"] },
  "format": "csv"
}
```

| Field | Options |
|-------|---------|
| **table** | Data table to export (contacts, messages, campaigns, etc.) |
| **filters** | Query filters to narrow the export |
| **format** | `csv` or `json` |

Large exports are queued and available for download when ready.

### Import Data

**Endpoint:** `POST /api/import`

Upload a CSV or JSON file to bulk-import records:

1. The file is parsed and validated
2. Records are checked for errors
3. Valid records are bulk-inserted or updated
4. A summary is returned with created, updated, and error counts

### Get Import/Export Config

**Endpoints:** `GET /api/export/{table}/config`, `GET /api/import/{table}/config`

Returns available fields, required fields, and format information for a given table.

## See Also

- [Contacts](contacts.md) — Using tags with contacts
- [Teams & Roles](teams-roles.md) — Managing organization members and roles
- [Campaigns](campaigns.md) — Importing campaign recipients
- [Chatbot](chatbot.md) — Using custom actions in chatbot flows
- [Analytics](analytics.md) — Exporting analytics data
