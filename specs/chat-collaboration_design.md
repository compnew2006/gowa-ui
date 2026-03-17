# Chat Collaboration Feature (WhatsApp)

## Overview
Enable multi-agent collaboration inside a single WhatsApp chat. Agent1 can invite Agent2 (based on role permissions and instance access). Agent2 can view the full history and assist or take over.

## Requirements
- Invite collaborators scoped to an org + contact.
- Respect WhatsApp instance access restrictions for invitees.
- Invited/accepted collaborators can view chat history.
- Support accept/decline flows for invitees.
- Support removing collaborators (self or privileged roles).
- Update UI in real time (WebSocket events).

## Implementation Plan
1. Add collaborator data model and DB migration/indexes.
2. Extend access control filters to include collaborators.
3. Implement collaborator APIs (list, invite, accept/decline, remove).
4. Emit WebSocket events for invite/update.
5. Update frontend stores, API client, and chat UI panel.
6. Validate permissions and instance access; add security notes/tests.

## Data Model
**Table:** `contact_collaborators`
- `id` (UUID), `organization_id`, `contact_id`, `user_id`
- `role` (`viewer` | `assistant`)
- `status` (`invited` | `accepted` | `declined`)
- `invited_by_user_id`
- `accepted_at`, `declined_at`, soft delete via `deleted_at`

Indexes:
- Unique per contact/user (excluding soft deletes)
- Lookup by `user_id` + `status` and `contact_id` + `status`

## API Endpoints
- `GET /api/contacts/{id}/collaborators`
- `POST /api/contacts/{id}/collaborators` (invite)
- `PUT /api/contacts/{id}/collaborators/{user_id}/accept`
- `PUT /api/contacts/{id}/collaborators/{user_id}/decline`
- `DELETE /api/contacts/{id}/collaborators/{user_id}`

## Access Control
- **Invite / remove**: `chat.collaborators:write`
- **List**: any user who can access the contact (public, assigned, or collaborator)
- **Accept/decline**: invitee only
- **Instance restrictions**: invitee must have access to the contact's WhatsApp instance.

## WebSocket Events
- `chat_collaborator_invite` -> delivered to invitee
- `chat_collaborator_update` -> delivered to contact viewers

## Frontend UX
- Contact info panel shows collaborators list + status.
- Invite dialog filters eligible users by:
  - active status
  - not self
  - instance access
  - not already invited/accepted
- Invitee can accept/decline directly in the list.

## Security Notes
- Input validated server-side (UUIDs, role enum).
- No sensitive data exposed in collaborator list payload.
- Access checks enforced at API and contact visibility filters.
- WebSocket events scoped to user/contact.

## Open Questions
- Should **invited** collaborators be able to read messages before acceptance? Current design allows it to support immediate assistance. If acceptance should gate access, change visibility checks to only include `accepted` status.
