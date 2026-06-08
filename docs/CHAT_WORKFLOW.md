# Chat Workflow Documentation

## Overview

The `/chat` route in Whatomate provides a comprehensive real-time messaging interface for managing WhatsApp conversations. It handles both direct 1-on-1 chats and group conversations, supports multiple WhatsApp accounts per contact, and integrates with chatbot automation, agent transfers, and collaboration features.

## Route Structure

### Frontend Routes
- `/chat` - Main chat interface with sidebar showing contact list
- `/chat/:contactId` - Specific conversation view with message history

### Backend API Routes
All chat routes are prefixed with `/api/chats` (these are aliases to the underlying contacts API):

| Method | Route | Purpose | Handler |
|--------|-------|---------|---------|
| GET | `/api/chats` | List contacts/chats with filters | `ListContacts` |
| GET | `/api/chats/{id}/messages` | Get message history for a chat | `GetMessages` |
| PUT | `/api/chats/{id}/claim` | Claim/unclaim a chat | `ClaimChat` |
| PUT | `/api/chats/{id}/close` | Close a conversation | `CloseChat` |
| PUT | `/api/chats/{id}/reopen` | Reopen a closed conversation | `ReopenChat` |
| PUT | `/api/chats/{id}/public` | Toggle public/private visibility | `SetChatPublic` |

## Chat Lifecycle States

A chat (contact) can be in one of three states:

### 1. **Pending** (`ChatStatusPending`)
- Chat has incoming messages but no assigned agent
- Visible in "Pending" queue to agents with appropriate permissions
- Auto-assignment rules may apply based on chatbot settings

### 2. **Open** (`ChatStatusOpen`)
- Chat is assigned to an agent (`assigned_user_id` is set)
- Agent can view and send messages
- Visible in assigned user's active chats list

### 3. **Closed** (`ChatStatusClosed`)
- Conversation is marked as resolved
- `closed_at` and `closed_by_user_id` are populated
- Can be reopened later
- Moved to closed chats view

**Note**: The effective status is determined by both the `status` field and `assigned_user_id`. A contact with an assignment is effectively "open" regardless of the stored status value.

## Core Features

### 1. Multi-Account Support

A single contact can have conversations across multiple WhatsApp instances/accounts:

**Frontend Behavior:**
- Sidebar shows account toggles when a contact has multiple accounts
- Each account has its own message history
- User can switch between accounts to view different conversation threads
- Account filter persists in sidebar state via `ChatSidebarUnifier`

**Backend Implementation:**
- `Contact` model has `instance_id` linking to a specific WhatsApp instance
- `Message` model includes `whatsapp_account` field
- Message history queries can be filtered by `whatsapp_account` parameter
- `/api/chats/{id}/messages?account=<account_name>` returns messages for specific account

### 2. Real-Time Updates via WebSocket

The chat interface uses WebSocket for real-time synchronization:

**WebSocket Message Types:**
- `new_message` - New message received/sent
- `message_media_updated` - Media upload/download status
- `contact_update` - Contact metadata changes (assignment, status, tags)
- `set_contact` - User viewing specific contact
- `typing` - Typing indicators (if supported)

**Broadcast Scope:**
- Organization-wide broadcasts for new messages
- Contact-scoped broadcasts for message updates
- User-scoped broadcasts for personal notifications

### 3. Message Types Supported

| Type | Description | Supported |
|------|-------------|-----------|
| Text | Plain text messages | ✅ |
| Image | Images with optional captions | ✅ |
| Video | Video files with optional captions | ✅ |
| Audio | Voice notes and audio files | ✅ |
| Document | PDF, DOC, etc. with filename | ✅ |
| Interactive | Button and list responses | ✅ |
| Template | WhatsApp Business templates | ✅ |
| Flow | WhatsApp Flows (interactive flows) | ✅ |
| Reaction | Emoji reactions to messages | ✅ |
| Status | WhatsApp Status replies | ✅ |
| Poll | Native WhatsApp polls with vote rendering | ✅ |

### 4. Message Pagination and Loading

Two pagination strategies are implemented:

**Cursor-based (Recommended):**
- Uses `before_id` parameter to load older messages
- Efficient for infinite scroll
- Maintains cursor position in message history
- Example: `GET /api/chats/{id}/messages?before_id=<message_id>&limit=50`

**Page-based (Legacy):**
- Uses `page` and `limit` parameters
- Loads from the end (most recent) backwards
- Example: `GET /api/chats/{id}/messages?page=1&limit=50`

**Auto-mark as read:**
- Messages are automatically marked as read when fetched
- Respects user deletion timestamps (messages deleted by user are hidden)
- Group chat messages marked read at conversation level

### 5. Collaboration Features

**Collaborator Roles:**
- Owner: Full access to chat
- Viewer: Read-only access to messages
- Editor: Can send messages and manage chat

**Collaborator Actions:**
- `POST /api/contacts/{id}/collaborators` - Invite user to collaborate
- `PUT /api/contacts/{id}/collaborators/{userId}/accept` - Accept invitation
- `PUT /api/contacts/{id}/collaborators/{userId}/decline` - Decline invitation
- `DELETE /api/contacts/{id}/collaborators/{userId}` - Remove collaborator

**Visibility:**
- Collaborators see chat in their sidebar
- IsCollaborator flag in ContactResponse indicates collaboration status
- WebSocket events notify collaborators of changes

### 6. Chat Actions

**Claim Chat:**
```
PUT /api/chats/{id}/claim
```
- Assigns chat to current user
- Requires `chat_assign:write` or `contacts:write` or `chat:write` permission
- Cannot claim closed chats
- Cannot claim chats assigned to other users
- Emits `contact_update` WebSocket event
- Adds system message indicating chat was claimed

**Close Chat:**
```
PUT /api/chats/{id}/close
```
- Marks conversation as resolved
- Requires same permissions as claim
- Only assigned user (or admin) can close
- Triggers chat close rating prompt if configured
- Emits `contact_update` WebSocket event
- Adds system message indicating chat was closed

**Reopen Chat:**
```
PUT /api/chats/{id}/reopen
```
- Reopens a closed conversation
- Requires same permissions as claim
- Clears `closed_at` and `closed_by_user_id`
- Emits `contact_update` WebSocket event
- No system message added on reopen

**Set Public/Private:**
```
PUT /api/chats/{id}/public
Body: { is_public: boolean }
```
- Toggles chat visibility to all agents
- Public chats are pinned at top of list
- Emits `contact_update` WebSocket event
- Adds system message indicating visibility change

### 7. Sending Messages

**Send Text/Interactive/Template:**
```
POST /api/contacts/{id}/messages
Body: {
  type: "text" | "image" | "video" | "audio" | "document" | "interactive" | "template" | "flow",
  content: string,
  reply_to_message_id?: string,
  instance_id?: string,
  whatsapp_account?: string,
  // Type-specific fields...
}
```

**Send Media:**
```
POST /api/messages/media
Body (multipart/form-data):
  - file: File
  - contact_id: string
  - type: "image" | "video" | "audio" | "document"
  - caption?: string
  - instance_id?: string
  - whatsapp_account?: string
```

**Send Typing Indicator:**
```
POST /api/contacts/{id}/typing
Body: {
  state: "composing" | "paused",
  instance_id?: string
}
```

**Send Reaction:**
```
POST /api/contacts/{id}/messages/{message_id}/reaction
Body: {
  emoji: string
}
```

**Revoke Message:**
```
POST /api/contacts/{id}/messages/{message_id}/revoke
```

### 8. Filtering and Search

**List Chats with Filters:**
```
GET /api/chats?search={query}&status={pending|open|closed}&assigned_to={me|unassigned|userId}&tags={tag1,tag2}&instance_id={id}&chat_types={direct,group}&page={1}&limit={50}
```

**Filter Parameters:**
- `search` - Search by phone number or profile name (case-insensitive)
- `status` - Filter by chat status
- `assigned_to` - Filter by assignment ("me", "unassigned", or user ID)
- `tags` - Filter by tags (comma-separated, matches ANY)
- `instance_id` - Filter by WhatsApp instance
- `chat_types` - Filter by chat type ("direct", "group")
- `date_basis` - Date filter basis ("created" or "incoming_any")
- `date_from` - Start date for filtering
- `date_to` - End date for filtering
- `created_from` - Contact creation start date
- `created_to` - Contact creation end date
- `closed_by` - Filter by who closed the chat
- `closed_from` - Closed chats start date
- `closed_to` - Closed chats end date

**Sort Order:**
- Public chats first
- Then by `last_message_at` DESC (most recent activity)
- Then by `created_at` DESC

### 9. Unread Message Counting

Unread messages are calculated per user and respect:

**Direct Chats:**
- Counted at contact level
- Filters by `contact_id`
- Excludes messages marked as read
- Respects user deletion timestamps

**Group Chats:**
- Counted at conversation level
- Uses `conversation_id` + `instance_id` as key
- Filters by `conversation_id`
- Excludes messages marked as read
- Respects user deletion timestamps

**Display:**
- `UnreadCount` field in ContactResponse
- Updated in real-time via WebSocket
- Badge shown in sidebar

### 10. User Deletion Support

Users can soft-delete chats, hiding them from their view:

**Soft Delete:**
```
POST /api/contacts/{id}/soft-delete
```
- Creates entry in `contact_user_deletions` table
- Stores deletion timestamp
- Contact hides from lists for that user
- Messages after deletion timestamp still appear

**Query Behavior:**
- List queries exclude deleted chats unless newer messages exist
- Message queries hide messages older than deletion timestamp
- Collaboration overrides deletion (collaborators always see chat)

### 11. Phone Number Masking

Organizations can enable phone number masking for privacy:

**When Enabled:**
- Phone numbers are partially hidden
- Profile names are also masked if they are phone numbers
- Format: `+1**********23` (country code + asterisks + last 2 digits)

**Exceptions:**
- Admin users see unmasked numbers
- Users with `mask_phone_numbers: false` in settings
- Masking only affects display, not storage or WhatsApp API calls

### 12. Service Window Indicator

Shows if a contact is within the 24-hour service window:

**Calculation:**
- Based on `last_inbound_at` timestamp
- Window = 24 hours from last inbound message
- `ServiceWindowOpen` field in ContactResponse
- Visual indicator in UI (green dot or similar)

**Purpose:**
- Helps agents prioritize recent conversations
- Complies with WhatsApp business messaging policies
- Useful for follow-up timing

## Access Control and Permissions

### Required Permissions

| Action | Permission(s) |
|--------|--------------|
| View chats | `chat:read` or `contacts:read` |
| View all contacts (not just assigned) | `contacts:read` |
| View unassigned chats | `allow_unclaimed_chat_view` (user restriction) |
| Send messages | `chat:write` or `contacts:write` |
| Send to unassigned chats | `allow_unclaimed_chat_send` (user restriction) |
| Claim chats | `chat_assign:write` or `contacts:write` or `chat:write` |
| Close chats | `chat_assign:write` or `contacts:write` or `chat:write` |
| Reopen chats | `chat_assign:write` or `contacts:write` or `chat:write` |
| Set public/private | `chat_assign:write` or `contacts:write` or `chat:write` |
| Soft delete chats | `contacts:soft_delete` |
| Revoke messages | `chat:delete` |
| Manage transfers | `transfers:write` |

### Role-Based Visibility

**Agent Role:**
- Limited to assigned chats by default
- Can see unassigned chats if `allow_unclaimed_chat_view` is true
- Can send to unassigned chats if `allow_unclaimed_chat_send` is true
- Cannot see other agents' assigned chats (unless collaborator)

**Admin/Manager Role:**
- Full visibility across all chats
- Can view and manage any conversation
- Can override assignment restrictions

### Instance Restrictions

Users can be restricted to specific WhatsApp instances:

**Implementation:**
- `user_instance_restrictions` table maps users to allowed instances
- Queries automatically filter by allowed instances
- Frontend hides restricted instances in UI
- Message sending validates instance permissions

### Send Restrictions

Additional send restrictions can be configured per user:

**Restriction Types:**
- Authorized phone numbers only
- Specific instances only
- Agent name prefix on messages
- Time-based restrictions (via chatbot settings)

## Frontend Components

### Main Chat View (`ChatView.vue`)

**Key Sections:**
1. **Sidebar** - Contact/chat list with filters and search
2. **Message List** - Scrollable message history with pagination
3. **Message Input** - Text area with media upload and emoji picker
4. **Contact Info Panel** - Contact details, tags, metadata
5. **Conversation Notes** - Internal notes for agents
6. **Media Group Bar** - Batch media download/view
7. **Status Stories Bar** - WhatsApp Status integration

**State Management:**
- Uses Pinia stores: `contacts`, `auth`, `users`, `transfers`, `config`, `tags`, `notes`, `instances`
- WebSocket service handles real-time updates
- Infinite scroll for message history
- Message grouping for media files

**Key Features:**
- Multi-account toggles in sidebar
- Real-time typing indicators
- Message reactions
- Reply to messages
- Forward/share messages
- Print single messages or merge media for printing
- Download media files
- Canned response picker
- Custom action triggers
- Agent availability toggle

## Backend Implementation

### Handler Files

| File | Purpose |
|------|---------|
| `internal/handlers/contacts.go` | Contact list, message retrieval, read marking |
| `internal/handlers/contacts_management.go` | Claim, close, reopen, set public |
| `internal/handlers/messages.go` | Send messages, media upload, reactions, revoke |
| `internal/handlers/contacts_messaging.go` | Message sending logic, typing indicators |
| `internal/handlers/contact_collaborators.go` | Collaboration management |
| `internal/handlers/conversation_notes.go` | Internal notes system |
| `internal/handlers/chat_access_policy.go` | Access control logic |
| `internal/handlers/send_restriction_policy.go` | Send validation and restrictions |

### Database Models

**Contact Model:**
```go
type Contact struct {
    ID                  uuid.UUID
    OrganizationID      uuid.UUID
    InstanceID          *uuid.UUID
    PhoneNumber         string
    ProfileName         string
    WhatsAppAccount     string
    Status              ChatStatus
    AssignedUserID      *uuid.UUID
    IsPublic            bool
    ClosedAt            *time.Time
    ClosedByUserID      *uuid.UUID
    LastMessageAt       *time.Time
    LastMessagePreview  string
    LastInboundAt       *time.Time
    Tags                JSONB
    Metadata            JSONB
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

**Message Model:**
```go
type Message struct {
    ID                uuid.UUID
    OrganizationID     uuid.UUID
    InstanceID         *uuid.UUID
    WhatsAppAccount    string
    ContactID          uuid.UUID
    ConversationID     string  // For group chats
    Direction          string  // "incoming" or "outgoing"
    MessageType        string  // "text", "image", "video", "poll", etc.
    Content            string
    MediaURL           string
    MediaMimeType      string
    MediaFilename      string
    Status             string  // "sent", "delivered", "read", "failed"
    WAMID              string  // WhatsApp message ID
    Error              string
    IsReply            bool
    ReplyToMessageID   *uuid.UUID
    InteractiveData    JSONB
    Reactions          JSONB
    Metadata           JSONB
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

## WebSocket Events

### Client → Server

**Auth:**
```json
{
  "type": "auth",
  "payload": { "token": "jwt_token" }
}
```

**Set Contact:**
```json
{
  "type": "set_contact",
  "payload": { "contact_id": "uuid" }
}
```

### Server → Client

**New Message:**
```json
{
  "type": "new_message",
  "payload": { /* Message object */ }
}
```

**Contact Update:**
```json
{
  "type": "contact_update",
  "payload": { /* Contact object */ }
}
```

**Typing Indicator:**
```json
{
  "type": "typing",
  "payload": {
    "contact_id": "uuid",
    "state": "composing" | "paused"
  }
}
```

## Integration Points

### Chatbot Integration

**Automatic Transfers:**
- Chatbot can create agent transfers
- Triggered by keywords, flow steps, or timeout
- Transfer includes context and notes

**Session Handoff:**
- When agent claims chat, chatbot session is cancelled
- Agent sees full conversation history
- Bot messages remain visible

**Current Conversation Only:**
- Chatbot setting limits agent view to messages from session start
- Useful for privacy in multi-tenant scenarios

### Analytics Integration

**Agent Analytics:**
- Message send/receive counts
- Response times
- Chat closure rates
- SLA compliance

**Meta Insights:**
- Message delivery rates
- Read receipt percentages
- Media engagement metrics

### Campaign Integration

**Campaign Replies:**
- Campaign messages appear in chat
- Replies link back to campaign
- Agent can see campaign context

**Auto-Campaign Media:**
- Media uploaded to campaigns can be reused in chat
- Shared media library

## Testing

### Unit Tests

**Backend:**
- `internal/handlers/contacts_test.go`
- `internal/handlers/messages_test.go`
- `internal/handlers/contacts_management_test.go`
- `internal/handlers/contact_collaborators_test.go`

**Frontend:**
- `frontend/src/views/chat/ChatView.test.ts`
- `frontend/src/components/chat/*/*.test.ts`

### E2E Tests

**Playwright Tests:**
- `frontend/e2e/tests/chat.spec.ts`
- Tests for sending messages, media upload, chat lifecycle
- Multi-account switching tests
- Real-time message delivery tests

## Performance Considerations

### Database Queries

**Indexed Fields:**
- `contacts.organization_id`, `contacts.instance_id`, `contacts.assigned_user_id`
- `contacts.status`, `contacts.is_public`
- `messages.contact_id`, `messages.organization_id`, `messages.conversation_id`
- `messages.direction`, `messages.status`
- `contact_user_deletions.contact_id`, `contact_user_deletions.user_id`

**Query Optimization:**
- Pagination limits result sets (max 500 for list, 100 for messages)
- Cursor-based pagination for efficient history loading
- Preloading of related entities (assigned user, closed by user)
- JSONB containment queries for tags

**Caching:**
- Chatbot settings cached in memory
- User permissions cached in context
- Contact avatar refresh scheduled asynchronously

### WebSocket Performance

**Broadcast Optimization:**
- Org-wide broadcasts for new messages
- Contact-scoped updates for message changes
- User-scoped notifications for personal events
- Connection pooling and reuse

### Frontend Performance

**Virtual Scrolling:**
- Long message lists use virtual scrolling
- Media lazy-loading
- Thumbnail generation for images

**State Management:**
- Pinia stores for efficient state updates
- Computed properties for derived data
- Watchers for reactive updates

## Security Considerations

### Authentication

- JWT-based authentication
- CSRF token validation for mutating requests
- Session refresh on token expiry

### Authorization

- Permission-based access control
- Role-based visibility restrictions
- Instance-level restrictions
- Send restrictions per user

### Data Privacy

- Phone number masking
- User-specific deletion timestamps
- Collaboration access controls
- Audit logging for sensitive actions

### Input Validation

- Message content validation
- Media type and size limits
- Phone number format validation
- SQL injection prevention (parameterized queries)

### Rate Limiting

- Message sending rate limits
- API endpoint rate limits
- WebSocket message rate limits

## Troubleshooting

### Common Issues

**Messages not appearing:**
- Check WebSocket connection status
- Verify user has `chat:read` permission
- Check contact assignment and visibility filters
- Review user deletion timestamps

**Cannot send messages:**
- Verify `chat:write` permission
- Check send restrictions
- Validate WhatsApp instance status
- Check if chat is closed

**Unread count incorrect:**
- Check message read status
- Verify deletion timestamps
- Review group chat conversation logic
- Clear browser cache and reload

**Multi-account issues:**
- Verify contact has multiple instances
- Check account filter in frontend
- Review `whatsapp_account` field in messages
- Validate instance permissions

### Debug Logging

**Backend:**
- Enable debug logging in config
- Check application logs for errors
- Review WebSocket connection logs
- Monitor database query performance

**Frontend:**
- Open browser DevTools Console
- Check Network tab for API errors
- Review WebSocket messages in Network tab
- Check Vue DevTools for state issues

## Future Enhancements

### Planned Features

1. **Message Search** - Full-text search within conversations
2. **Message Forwarding** - Forward messages to other chats
3. **Scheduled Messages** - Schedule messages for later delivery
4. **Message Templates** - Quick reply templates in chat
5. **Voice Messages** - Record and send voice notes
6. **Location Sharing** - Send and view location data
7. **Contact Merging** - Merge duplicate contacts
8. **Chat Export** - Export conversation history
9. **Message Encryption** - End-to-end encryption indicators
10. **Multi-language Support** - Real-time translation

### Performance Improvements

1. **Database Read Replicas** - Offload read queries
2. **Message Archiving** - Archive old messages
3. **CDN for Media** - Cache media files
4. **WebSocket Clustering** - Horizontal scaling
5. **Frontend Code Splitting** - Reduce bundle size

## Related Documentation

- [Workflow Guide](./workflow.md) - General workflow documentation
- [Chat Workflow Guide](./chat-workflow-guide.html) - Detailed chatbot workflow
- [API Endpoints](./API_ENDPOINTS.md) - Complete API reference
- [WebSocket Protocol](./WEBSOCKET_PROTOCOL.md) - WebSocket message formats
- [Database Schema](./schema.sql) - Database structure
- [ERD Diagram](./ERD.md) - Entity relationship diagram
