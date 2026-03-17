# Task 5: Collaboration Regression Coverage

**Milestone**: [M4 - Chat Collaboration & Assignment Permissions](../milestones/milestone-4-chat-collaboration.md)
**Estimated Time**: 1 hour
**Dependencies**: Tasks 3 and 4
**Status**: Completed

---

## Objective

Define and implement the regression coverage needed before collaboration and assignment permissions are considered functionally complete.

## Context

The collaboration and assignment specs are now tracked under M4. This task closed the major coverage gaps by adding backend assignment assertions and shared frontend filtering tests alongside the collaborator coverage.

## Steps

### 1. Backend checks
- [x] Added collaborator-handler regression tests for inactive invitees, duplicate invited/accepted collaborators, instance-restricted invitees, declined re-invite, and self-removal.
- [x] Added assignment permission tests for `AssignContact`, transfer instance restrictions, and transfer response `instance_id` assertions.

### 2. Frontend checks
- [x] Added unit coverage for shared instance-aware filtering logic used by chat and transfer assignment flows.
- [x] Moved duplicated instance filtering helpers into a shared utility to keep future UI coverage centered in one place.

### 3. Documentation checks
- [x] Ensure the milestone links to the two feature specs.
- [x] Verify the ACP progress state matches the new milestone structure.
- [x] Record the current environment limitation for DB-backed handler execution.

## Verification

- [x] Regression coverage expectations are documented for backend and frontend.
- [x] Negative-path authorization cases are included.
- [x] Progress tracking is aligned with the milestone and task files.

## Expected Output

The task now records the completed coverage work and the remaining environment-specific validation caveat.

## Notes

- Backend coverage added in this turn lives in `internal/handlers/contact_collaborators_test.go` and `internal/handlers/assignment_permissions_test.go`.
- Shared frontend coverage added in this turn lives in `frontend/src/lib/instance-access.test.ts`.
- DB-backed handler tests could not be executed locally because `TEST_DATABASE_URL` and `TEST_REDIS_URL` are not configured in this environment.
- If the implementation scope changes later, update the milestone instead of overloading this checklist.

**Next Task**: None
**Related Design Docs**: [`specs/chat-collaboration_design.md`](../../specs/chat-collaboration_design.md), [`specs/assign-to-agent-permissions_design.md`](../../specs/assign-to-agent-permissions_design.md)
**Estimated Completion Date**: 2026-03-17
