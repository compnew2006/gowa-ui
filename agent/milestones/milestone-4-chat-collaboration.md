# Milestone 4: Chat Collaboration & Assignment Permissions

**Goal**: Make chat collaboration and agent assignment instance-aware so invites, transfer assignment, and eligibility filtering follow the same access rules.
**Duration**: 1 week
**Dependencies**: Milestones 2 and 3
**Status**: Completed

---

## Overview

This milestone formalizes the next feature slice after ACP context refinement. It covers the collaboration flow already described in the recent feature specs and turns it into a tracked milestone with concrete implementation checkpoints.

The focus is on keeping chat collaboration safe and predictable across backend authorization, frontend eligibility filtering, and transfer payloads. The work is driven by:

- [`specs/chat-collaboration_design.md`](../../specs/chat-collaboration_design.md)
- [`specs/assign-to-agent-permissions_design.md`](../../specs/assign-to-agent-permissions_design.md)

---

## Deliverables

### 1. Backend authorization alignment
- Enforce collaboration invite, accept, decline, and remove flows with explicit permission checks.
- Validate assignee instance access before assignment or invite actions are accepted.
- Expose `instance_id` consistently in transfer-related responses.

Current state:
- [x] Collaboration invites now reject inactive users.
- [x] Collaboration invites now reject duplicate invites for already invited or accepted collaborators.
- [x] Invite/accept/decline/remove flows now have targeted regression tests.

### 2. Frontend eligibility filtering
- Filter eligible users and agents by allowed instance IDs in chat and transfer UIs.
- Extend transfer-facing models to carry `instance_id` for client-side visibility rules.
- Keep collaboration and assignment affordances hidden when the current user lacks access.

Current state:
- [x] Chat and transfer assignment UIs already filter by allowed instance IDs.
- [x] Transfer-facing models already carry `instance_id`.
- [x] Shared frontend instance-access filtering logic now has unit coverage.

### 3. Regression coverage
- Add backend coverage for permission and instance restriction enforcement.
- Add frontend coverage for instance-aware filtering and response handling.
- Document the verification matrix for collaboration-related flows.

Current state:
- [x] Targeted backend collaborator invite/remove tests were added.
- [x] Assignment permission regression tests were added.
- [x] Frontend collaboration/assignment filtering logic now has shared unit coverage.

## Success Criteria

- [x] Collaboration and assignment docs reference the correct design sources.
- [x] Instance-aware authorization rules are clearly defined for backend implementation.
- [x] Frontend filtering requirements are explicit enough to drive implementation.
- [x] Test coverage expectations are mapped to the collaboration and assignment flows.
- [x] Progress tracking in `agent/progress.yaml` reflects the new milestone.

## Key Files to Create

```
agent/
├── milestones/
│   └── milestone-4-chat-collaboration.md
└── tasks/
    ├── task-3-align-collaboration-authorization.md
    ├── task-4-harden-assignment-permissions.md
    └── task-5-collaboration-regression-coverage.md
```

## Tasks

1. [Task 3: Align Collaboration Authorization](../tasks/task-3-align-collaboration-authorization.md) - Reconcile collaborator APIs with the chat collaboration design.
2. [Task 4: Harden Assignment Permissions](../tasks/task-4-harden-assignment-permissions.md) - Align transfer and assignment access rules with instance visibility.
3. [Task 5: Collaboration Regression Coverage](../tasks/task-5-collaboration-regression-coverage.md) - Define the test and verification matrix for the milestone.

## Testing Requirements

- [x] Backend permission checks are defined for invite, accept, decline, and remove paths.
- [x] Assignment eligibility rules are defined for both chat and transfer UIs.
- [x] Instance-scoped filtering expectations are captured for frontend behavior.
- [x] Regression coverage is explicitly tied to the two feature specs.

## Documentation Requirements

- [x] Update ACP progress tracking for the next milestone.
- [x] Keep the milestone linked to the two recent feature specs.
- [x] Preserve a clear handoff path from planning docs to implementation tasks.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Scope drift between collaboration and assignment rules | Medium | Medium | Keep the milestone anchored to the two named design specs. |
| Future implementation work outgrows a single milestone | Medium | Medium | Split follow-up work into later milestones instead of expanding M4 ad hoc. |

**Next Milestone**: TBD
**Blockers**: None
**Notes**: M4 implementation is complete. On 2026-04-07, targeted backend and frontend regression tests for the collaboration and assignment slice passed locally. The remaining limitation is environment-specific execution of DB-backed handler tests when `TEST_DATABASE_URL` and `TEST_REDIS_URL` are unavailable locally.
