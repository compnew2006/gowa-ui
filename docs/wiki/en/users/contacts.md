---
title: Contacts
---

# Contacts

Manage your WhatsApp contacts, organize them with filters and tags, assign them to team members, and collaborate across your team.

## Listing Contacts

**Endpoint:** `GET /api/contacts`

View all contacts in your organization. The list includes each contact's phone number, name, tags, last message time, and unread count.

### Filtering

Narrow down contacts using these filters:

| Filter | Description | Example |
|--------|-------------|---------|
| **Search** | Match by phone, name, or profile name | `?search=john` |
| **Tags** | Filter by assigned tags | `?tags=vip,lead` |
| **Tag mode** | Match all or any tags | `?tag_mode=all` or `?tag_mode=any` |
| **Assigned to** | Filter by assigned user | `?assigned_to=user-id` |
| **Status** | Open, closed, or pending chats | `?status=open` |
| **Date range** | Filter by last activity | `?from=2024-01-01&to=2024-12-31` |

### Agent View

If you have the **Agent** role, you only see contacts you have access to:

- Chats assigned to you
- Public chats (`is_public = true`)
- Chats where you are a collaborator

Admins and managers see all contacts in the organization.

## Creating a Contact

**Endpoint:** `POST /api/contacts`

```json
{
  "phone_number": "+1234567890",
  "name": "John Doe",
  "tags": ["vip", "customer"],
  "metadata": {
    "source": "website"
  }
}
```

**Validation:**

- Phone number format is validated automatically
- Duplicate phone numbers within the same organization are prevented
- A `contact_created` webhook is dispatched on success

### Starting a New Chat (WhatsMeow)

If your organization uses the WhatsMeow provider, you can start a chat directly with any phone number:

1. Enter the phone number in the chat start dialog
2. The system verifies the number is on WhatsApp
3. If verified, the contact is created and the chat opens
4. If the number is a verified business, the business name is resolved automatically

## Viewing a Contact

**Endpoint:** `GET /api/contacts/{id}`

View full contact details including profile information, tags, assignment history, and metadata.

### Session Data

**Endpoint:** `GET /api/contacts/{id}/session-data`

Returns a consolidated view of the contact's recent activity:

- Recent messages
- Applied tags
- Conversation notes
- Assignment history

## Updating a Contact

**Endpoint:** `PUT /api/contacts/{id}`

Update any editable field:

```json
{
  "name": "Jane Doe",
  "tags": ["vip", "priority"],
  "metadata": {
    "source": "referral"
  }
}
```

## Assigning Contacts

**Endpoint:** `PUT /api/contacts/{id}/assign`

Assign a contact to a specific team member:

```json
{
  "user_id": "user-uuid"
}
```

**What happens:**

1. The contact is assigned to the specified user
2. The user receives a real-time notification via WebSocket
3. A `contact_assigned` webhook is dispatched

## Collaborators

Invite other team members to collaborate on a contact.

### List Collaborators

**Endpoint:** `GET /api/contacts/{id}/collaborators`

### Invite Collaborator

**Endpoint:** `POST /api/contacts/{id}/collaborators`

```json
{
  "user_id": "user-uuid"
}
```

The invited user receives a notification and can accept or decline.

### Accept / Decline

- **Accept:** `PUT /api/contacts/{id}/collaborators/{user_id}/accept`
- **Decline:** `PUT /api/contacts/{id}/collaborators/{user_id}/decline`

### Remove Collaborator

**Endpoint:** `DELETE /api/contacts/{id}/collaborators/{user_id}`

## Soft Delete

**Endpoint:** `POST /api/contacts/{id}/soft-delete`

Hide a contact from your view without deleting it for other collaborators. The contact remains visible to other team members who have access.

## Deleting a Contact

**Endpoint:** `DELETE /api/contacts/{id}`

Permanently removes the contact and all associated data. This action requires `contacts:delete` permission.

## Contact Repair

Orphaned or inconsistent contact records can be repaired automatically. This typically happens after data migrations or account changes:

1. The system scans for contacts with missing or invalid references
2. Orphaned contacts are reassigned to the correct account or instance
3. Contact metadata is updated to reflect the current state
4. All repair actions are logged

## See Also

- [Chat & Messaging](chat-messaging.md) — Send messages to contacts
- [Tags & Organization](tags-organization.md) — Managing tags
- [Teams & Roles](teams-roles.md) — User roles and permissions
