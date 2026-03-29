# Feature: Per-User Chat Soft Delete + Admin Notifications

## Requirements (EARS Format)
- While an authenticated user has the `contacts:soft_delete` permission, when they request to hide a chat, the system shall persist a per-user deletion timestamp for the contact and hide the chat from that user's list until new activity occurs.
- While a chat is soft-deleted for a user, when the user loads messages or unread counts, the system shall return/count only messages created after the stored deletion timestamp.
- While a user soft-deletes a chat, the system shall create an instance notification with `event_type=chat_deleted_by_user`, `contact_id`, and metadata identifying the actor and contact.
- While a user is not an admin or super admin, when they list or dismiss notifications, the system shall not expose `chat_deleted_by_user` notifications.
- While an admin views a notification that includes `contact_id`, when they click it, the UI shall open the corresponding chat.

## Architecture
- Frontend:
  - Add a soft-delete action in `ChatView` (sidebar) and `ContactInfoPanel`, gated by `contacts:soft_delete`.
  - Add `contactsService.softDelete` API client and show confirmation + toast states.
  - Make notifications clickable when a `contact_id` exists and localize the `chat_deleted_by_user` message.
  - Localize new strings in `en/ar/es`.
- Backend:
  - New `ContactUserDeletion` model/table keyed by `(organization_id, contact_id, user_id)`.
  - New endpoint `POST /api/contacts/{id}/soft-delete` to upsert deletion timestamp.
  - Filter contact list, unread counts, and message queries by deletion timestamp for the requesting user.
  - Extend `InstanceNotification` with `contact_id` + `metadata`, create/broadcast notifications on soft delete.
  - Seed and backfill `contacts:soft_delete` permission for system roles.
- Security:
  - Enforce authentication and `contacts:soft_delete` authorization for soft delete.
  - Validate contact UUIDs and scope all queries by organization/user.
  - Restrict notification visibility to admins/super admins.
  - Preserve message history (no hard delete) to avoid data loss.

## Implementation Plan
- [ ] Add `ContactUserDeletion` model + migration entry and extend `InstanceNotification` with `contact_id`/`metadata`.
- [ ] Seed/backfill the `contacts:soft_delete` permission for system roles.
- [ ] Implement `POST /api/contacts/{id}/soft-delete` with notification creation + websocket broadcast.
- [ ] Filter contact list, unread counts, and message retrieval by per-user deletion timestamp.
- [ ] Add frontend API call, UI actions, and notification click behavior.
- [ ] Add `en/ar/es` translations and update permission labels.
- [ ] Verify with Go tests and manual UI checks; document results.
