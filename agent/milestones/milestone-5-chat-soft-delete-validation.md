# Milestone 5: Per-User Chat Soft Delete Validation

**Goal**: Verify and harden the shipped per-user chat soft-delete feature with active regression coverage, frontend/admin behavior validation, and ACP backfill.
**Duration**: 1 week
**Dependencies**: Milestone 4
**Status**: Completed

---

## Overview

The per-user chat soft-delete feature already exists in the codebase and is described in [`specs/chat-soft-delete_design.md`](../../specs/chat-soft-delete_design.md). This milestone backfilled ACP tracking, active regression coverage, and frontend/admin verification so the feature is now both documented and protected.

The work centers on three layers:

- backend enforcement and filtering behavior
- frontend affordances and admin notification visibility
- ACP synchronization so future sessions can resume from tracked milestone/task state instead of ad hoc repo knowledge

## Deliverables

### 1. Backend regression coverage
- Active tests for `POST /api/contacts/{id}/soft-delete`
- Coverage for per-user deletion rows and admin notifications
- Coverage for list/message filtering after soft delete

### 2. Frontend and notification validation
- Validation of chat hide affordances in the chat sidebar and contact info panel
- Validation of admin-only notification visibility and click-through behavior
- Clear statement of remaining UI/manual verification gaps

### 3. ACP synchronization
- Milestone and task docs for the feature
- Progress tracking aligned with the new validation work
- Requirements/design references updated once the validation pass is complete

## Success Criteria

- [x] ACP milestone/task tracking exists for the chat soft-delete feature.
- [x] Active backend regression coverage protects the soft-delete endpoint and filter behavior.
- [x] Frontend behavior and admin notification interaction are verified and captured in ACP.
- [x] ACP requirements/progress docs are fully synced to the validated feature state.

## Key Files to Create

```
agent/
├── milestones/
│   └── milestone-5-chat-soft-delete-validation.md
└── tasks/
    ├── task-6-add-chat-soft-delete-regression-coverage.md
    ├── task-7-verify-chat-soft-delete-frontend-and-notifications.md
    └── task-8-sync-acp-docs-for-chat-soft-delete.md
```

## Tasks

1. [Task 6: Add Chat Soft Delete Regression Coverage](../tasks/task-6-add-chat-soft-delete-regression-coverage.md) - Protect backend endpoint, notification, and filtering behavior with active tests.
2. [Task 7: Verify Chat Soft Delete Frontend and Notifications](../tasks/task-7-verify-chat-soft-delete-frontend-and-notifications.md) - Validate UI affordances and admin notification interaction against the feature design.
3. [Task 8: Sync ACP Docs for Chat Soft Delete](../tasks/task-8-sync-acp-docs-for-chat-soft-delete.md) - Fold the validated feature state back into ACP requirements and progress tracking.

## Testing Requirements

- [x] Backend soft-delete happy path is covered.
- [x] Authorization and visibility edge cases are covered.
- [x] Contact list and message filtering behavior is covered.
- [x] Frontend soft-delete affordances have explicit verification coverage.
- [x] Admin notification click behavior is verified.

## Documentation Requirements

- [x] Link the milestone to the source feature spec.
- [x] Add ACP task tracking for the validation work.
- [x] Update requirements/progress summaries after the verification sweep is complete.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Shipped soft-delete behavior drifts from the spec unnoticed | High | Medium | Add targeted regression tests before broader refactors touch contacts, notifications, or chat history. |
| ACP remains behind the real codebase state | Medium | Medium | Track the backfill work as a real milestone instead of burying it in recent_work notes. |

**Next Milestone**: TBD
**Blockers**: None
**Notes**: This milestone is a validation/backfill slice, not a net-new feature launch. Residual follow-up is limited to optional stronger UI automation for the `ChatView.vue` sidebar path and broader DB-backed handler execution when external test environment variables are available.
