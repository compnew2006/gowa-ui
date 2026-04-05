---
title: Chat & Messaging
---

# Chat & Messaging

Send and receive WhatsApp messages, manage chat lifecycles, and use rich messaging features like reactions, read receipts, and reply previews.

## Chat Listing

Chats are displayed in your contacts list with real-time updates. Each chat shows:

- Contact name and phone number
- Last message preview and timestamp
- Unread message count
- Assigned agent (if any)
- Tags and status indicators

### Chat States

| State | Description |
|-------|-------------|
| **Open** | Active conversation, messages can be sent and received |
| **Closed** | Conversation ended, read-only until reopened |
| **Pending** | Awaiting agent assignment |

## Messages

### Viewing Messages

**Endpoint:** `GET /api/chats/{id}/messages`

Messages are loaded with cursor-based pagination for smooth scrolling:

```
GET /api/chats/{id}/messages?limit=50&before=cursor-token
```

**Features:**

- Messages are grouped by date
- Inbound and outbound messages are visually distinguished
- Reply context is shown inline when a message references another
- Media messages display thumbnails with download links
- System messages (e.g., "Chat closed by Agent") appear as inline notifications

### Message Types

| Type | Description |
|------|-------------|
| Text | Plain text messages |
| Media | Images, videos, audio, documents |
| Template | Pre-approved WhatsApp template messages |
| Interactive | Buttons, lists, CTA URL messages |
| Reaction | Emoji reactions to existing messages |
| System | Internal lifecycle events |

## Sending Messages

### Text Message

**Endpoint:** `POST /api/contacts/{id}/messages`

```json
{
  "content": "Hello! How can I help you today?",
  "reply_to_message_id": "msg-uuid"
}
```

Including `reply_to_message_id` creates a threaded reply with a preview of the original message.

### Media Message

**Endpoint:** `POST /api/messages/media`

Upload and send media in one step:

- **Supported types:** Image, video, audio, document
- **Process:** File is uploaded to storage, then sent via WhatsApp
- **Caption:** Optional text accompanying the media

### Template Message

**Endpoint:** `POST /api/messages/template`

```json
{
  "template_id": "template-uuid",
  "contact_id": "contact-uuid",
  "parameters": {
    "1": "John",
    "2": "your order #12345"
  }
}
```

Template messages use pre-approved Meta templates with dynamic placeholder resolution.

### Interactive Messages

Send rich interactive messages with buttons or lists:

| Type | Description | Limit |
|------|-------------|-------|
| **Button** | Quick-reply buttons | Up to 3 buttons |
| **List** | Single-select list with sections | Up to 10 options |
| **CTA URL** | Call-to-action button with URL | 1 button |

```json
{
  "interactive": {
    "type": "button",
    "header": "Choose an option",
    "body": "How would you like to proceed?",
    "buttons": [
      { "id": "1", "title": "Support" },
      { "id": "2", "title": "Sales" }
    ]
  }
}
```

> **Note:** Interactive messages require the Meta provider. WhatsMeow has limited support.

## Chat Lifecycle

### Claim a Chat

**Endpoint:** `PUT /api/chats/{id}/claim`

Take ownership of an unassigned or pending chat:

1. The chat is assigned to you
2. Other agents are notified via WebSocket
3. You can now send messages to this contact

### Close a Chat

**Endpoint:** `PUT /api/chats/{id}/close`

End a conversation:

1. The chat status changes to `closed`
2. A system message records who closed it and when
3. Optionally, a satisfaction rating request is sent
4. All participants are notified via WebSocket

### Reopen a Chat

**Endpoint:** `PUT /api/chats/{id}/reopen`

Reopen a closed conversation:

1. The `closed_at` timestamp is cleared
2. Status returns to `open`
3. A system message records the reopening
4. All participants are notified

### Set Chat Public

**Endpoint:** `PUT /api/chats/{id}/public`

Toggle the `is_public` flag to allow collaborator access. Public chats are visible to all agents, not just the assigned user.

## Reactions

**Endpoint:** `POST /api/contacts/{id}/messages/{message_id}/reaction`

Send emoji reactions to any message:

```json
{
  "emoji": "\uD83D\uDC4D"
}
```

Reactions appear inline on the original message in the chat view.

## Revoke Message

**Endpoint:** `POST /api/contacts/{id}/messages/{message_id}/revoke`

Delete a previously sent message for everyone. The message is replaced with a "This message was deleted" notice.

## Read Receipts

**Endpoint:** `PUT /api/messages/{id}/read`

Mark inbound messages as read. This sends a read receipt to the contact via WhatsApp and updates the message status in the system.

## Typing Indicators

**Endpoint:** `POST /api/contacts/{id}/typing`

Show or hide the "typing..." indicator:

```json
{
  "state": "composing"
}
```

Use `composing` to show typing and `paused` to hide it. The indicator is routed through your configured provider (Meta or WhatsMeow).

## Reply Preview

When a message references another message, a reply preview is displayed showing:

- The original message content (truncated for long messages)
- Message type indicator (text, image, etc.)
- Sender name
- Media thumbnail (if applicable)

## Agent Chat Scoping

If you have the **Agent** role, your chat visibility is restricted:

| You Can See | You Cannot See |
|-------------|----------------|
| Chats assigned to you | Chats assigned to other agents |
| Public chats (`is_public = true`) | Private chats assigned to others |
| Chats where you are a collaborator | Unassigned chats (unless configured) |

When sending a message, the system verifies you have access to the contact. If the chat is closed or unclaimed, you will receive an appropriate error message.

## Real-Time Updates

All chat events are broadcast via WebSocket in real time:

| Event | Description |
|-------|-------------|
| `message` | New message received |
| `message_status` | Message delivery status updated |
| `chat_closed` | Chat was closed |
| `chat_reopened` | Chat was reopened |
| `contact_assigned` | Contact assigned to a user |
| `typing` | Typing indicator |
| `presence` | Contact online/offline status |

## See Also

- [Contacts](contacts.md) — Managing contacts
- [Canned Responses](canned-responses.md) — Quick reply templates
- [Chatbot](chatbot.md) — Automated chat responses
- [Templates & Flows](templates-flows.md) — Message templates
