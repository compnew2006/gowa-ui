# Task 6: Add Chat Soft Delete Regression Coverage

**Milestone**: [M5 - Per-User Chat Soft Delete Validation](../milestones/milestone-5-chat-soft-delete-validation.md)
**Design Reference**: [Per-User Chat Soft Delete + Admin Notifications](../../specs/chat-soft-delete_design.md)
**Estimated Time**: 2 hours
**Dependencies**: None
**Status**: Completed

---

## Objective

Add active backend regression coverage for the shipped per-user chat soft-delete flow so permission checks, deletion persistence, admin notification visibility, and message/contact filtering are protected.

## Context

The feature already exists in production code across `contacts_management.go`, `contacts.go`, and `notifications.go`, but active tests were missing. That left the behavior vulnerable to regression even though the design in `specs/chat-soft-delete_design.md` is explicit about authorization, filtering, and admin-only notification access.

## Steps

### 1. Cover the endpoint contract
- Add a forbidden test for users without `contacts:soft_delete`.
- Add a happy-path test for `SoftDeleteContactForUser` that verifies:
  - a `contact_user_deletions` row is persisted
  - open chats are closed as part of the hide flow
  - an admin-visible `chat_deleted_by_user` notification is created with `contact_id` and actor metadata

### 2. Cover filtered visibility
- Add a contact-list regression test showing the hidden chat stays out of the requesting user's list until new activity occurs.
- Verify unread counts only include messages created after the stored deletion timestamp.

### 3. Cover message-history filtering
- Add a messages regression test showing `GetMessages` excludes messages created before the requesting user's deletion timestamp.

## Verification

- [x] Users without `contacts:soft_delete` receive a forbidden response.
- [x] Soft delete persists a `contact_user_deletions` record.
- [x] Admin-only `chat_deleted_by_user` notifications are created and non-admin listing excludes them.
- [x] Contact list visibility reappears only after newer activity than the deletion timestamp.
- [x] Message history excludes pre-delete messages for the deleting user.
- [x] `go test ./internal/handlers -run 'TestApp_(SoftDeleteContactForUser|ListContacts_HidesSoftDeleted|GetMessages_ExcludesSoftDeleted)'` passes.

## Expected Output

- Active Go regression tests for the chat soft-delete backend behavior.
- ACP progress tracking updated to reflect the new M5 validation milestone and completed Task 6.

## Notes

- Production code under test:
  - `internal/handlers/contacts_management.go`
  - `internal/handlers/contacts.go`
  - `internal/handlers/notifications.go`
- The endpoint and filters were already implemented; this task adds protection, not new product behavior.

**Next Task**: [Task 7: Verify Chat Soft Delete Frontend and Notifications](task-7-verify-chat-soft-delete-frontend-and-notifications.md)
**Estimated Completion Date**: 2026-04-07T10:32:13+02:00
