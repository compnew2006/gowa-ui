# Task 4: Harden Assignment Permissions

**Milestone**: [M4 - Chat Collaboration & Assignment Permissions](../milestones/milestone-4-chat-collaboration.md)
**Estimated Time**: 2 hours
**Dependencies**: Task 3
**Status**: Completed

---

## Objective

Verify that chat assignment and transfer assignment already align with the instance-aware assignment design and capture what remains.

## Context

The recent assignment permissions design describes the concrete rule set for filtering eligible assignees and enforcing server-side restrictions. Existing code already implements the production behavior, so this task records that alignment and leaves the remaining test gaps to Task 5.

## Steps

### 1. Capture authorization boundaries
- Confirmed contact assignment accepts `chat.assign:write` or `contacts:write`.
- Confirmed contact and transfer assignment reject assignees who cannot access the relevant WhatsApp instance.

### 2. Define instance-aware filtering
- Confirmed the chat and transfer assignment dialogs already filter by allowed instance IDs.
- Confirmed the instance-scoped filtering behavior is implemented in frontend state/view code.

### 3. Record transfer payload expectations
- Confirmed transfer responses surface `instance_id` and the transfer store model carries it through to the UI.
- Left remaining serialization/assertion coverage as explicit follow-up work in Task 5.

## Verification

- [x] Assignment permission rules are aligned to the design spec.
- [x] Instance-aware filtering requirements are captured for the UI.
- [x] Transfer payload expectations are documented clearly.

## Expected Output

The assignment permission behavior is already implemented; the remaining work is regression coverage rather than production logic changes.

## Notes

- Verified implementation areas:
  - `internal/handlers/contacts_management.go`
  - `internal/handlers/agent_transfers.go`
  - `frontend/src/views/chat/ChatView.vue`
  - `frontend/src/views/chatbot/AgentTransfersView.vue`
  - `frontend/src/stores/transfers.ts`
- Coverage gaps roll forward to Task 5.

**Next Task**: [Task 5: Collaboration Regression Coverage](task-5-collaboration-regression-coverage.md)
**Related Design Docs**: [`specs/assign-to-agent-permissions_design.md`](../../specs/assign-to-agent-permissions_design.md)
**Estimated Completion Date**: 2026-03-17
