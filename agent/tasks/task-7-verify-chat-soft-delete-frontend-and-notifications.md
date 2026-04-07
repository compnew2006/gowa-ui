# Task 7: Verify Chat Soft Delete Frontend and Notifications

**Milestone**: [M5 - Per-User Chat Soft Delete Validation](../milestones/milestone-5-chat-soft-delete-validation.md)
**Design Reference**: [Per-User Chat Soft Delete + Admin Notifications](../../specs/chat-soft-delete_design.md)
**Estimated Time**: 2 hours
**Dependencies**: Task 6
**Status**: Completed

---

## Objective

Validate the frontend soft-delete affordances and the admin notification interaction path against the feature design.

## Context

The chat sidebar, contact info panel, notification bell, and locale files already contain soft-delete behavior, but ACP does not yet capture whether those pieces fully match the design and whether explicit UI coverage exists.

## Steps

### 1. Review the frontend affordances
- Verified `frontend/src/views/chat/ChatView.vue` uses the same `canSoftDeleteChats` permission gate for the sidebar affordance and the same soft-delete endpoint/toast path as the design.
- Added explicit unit coverage for the contact-level hide action in `frontend/src/components/chat/ContactInfoPanel.test.ts`.

### 2. Review admin notification behavior
- Added explicit unit coverage for `chat_deleted_by_user` rendering and click-through behavior in `frontend/src/components/NotificationBell.test.ts`.
- Added locale coverage assertions for `en`, `ar`, and `es` in `frontend/src/i18n/chat-soft-delete-locales.test.ts`.

### 3. Capture the verification result
- Added targeted frontend coverage for the contact info panel, notification bell, and locale keys.
- Recorded the remaining manual-only check: `ChatView.vue` sidebar behavior was source-verified in this slice rather than mounted in a component test because of the view's broader dependency surface.

## Verification

- [x] Frontend hide-chat affordances match the permission gate from the design.
- [x] Admin notification messaging and navigation behavior are verified.
- [x] Localization coverage for the soft-delete flow is confirmed.
- [x] Any remaining manual-only checks are documented explicitly.

## Expected Output

ACP records that the contact-level hide flow, admin notification rendering/navigation, and locale coverage now have explicit verification, with the chat sidebar path documented as a manual source review in this task.

**Next Task**: [Task 8: Sync ACP Docs for Chat Soft Delete](task-8-sync-acp-docs-for-chat-soft-delete.md)
**Estimated Completion Date**: 2026-04-07T10:32:13+02:00
