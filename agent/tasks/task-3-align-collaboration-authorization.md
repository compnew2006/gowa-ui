# Task 3: Align Collaboration Authorization

**Milestone**: [M4 - Chat Collaboration & Assignment Permissions](../milestones/milestone-4-chat-collaboration.md)
**Estimated Time**: 2 hours
**Dependencies**: Task 2
**Status**: Completed

---

## Objective

Align the backend collaboration authorization flow with the design spec and close the most important server-side enforcement gaps.

## Context

The collaboration UI already filtered active, eligible users, but the API still accepted inactive invitees and allowed accepted collaborators to be downgraded back to `invited` through direct calls.

## Steps

### 1. Reconcile backend rules
- Compared the collaborator API behavior with [`specs/chat-collaboration_design.md`](../../specs/chat-collaboration_design.md).
- Confirmed the server must enforce active-user eligibility and reject duplicate invites for already invited or accepted collaborators.

### 2. Preserve observable behavior
- Kept the invite endpoint response contract stable while tightening validation.
- Left the invited-collaborator visibility policy unchanged because it matches the current spec default.

### 3. Land the backend hardening
- Updated `internal/handlers/contact_collaborators.go` so the invite API now rejects inactive users.
- Prevented the API from mutating already invited or accepted collaborators back to `invited`.

## Verification

- [x] Collaboration API authorization rules are written down in one place.
- [x] The task references the correct design specification.
- [x] Remaining backend work is easy to hand off to implementation.

## Expected Output

The collaboration invite API now enforces the missing eligibility rules, and the remaining work is isolated to regression coverage.

## Notes

- Production files touched:
  - `internal/handlers/contact_collaborators.go`
  - `internal/handlers/contact_collaborators_test.go`
- Remaining work stays in Task 5.

**Next Task**: [Task 4: Harden Assignment Permissions](task-4-harden-assignment-permissions.md)
**Related Design Docs**: [`specs/chat-collaboration_design.md`](../../specs/chat-collaboration_design.md)
**Estimated Completion Date**: 2026-03-17
