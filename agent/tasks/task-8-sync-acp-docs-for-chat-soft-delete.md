# Task 8: Sync ACP Docs for Chat Soft Delete

**Milestone**: [M5 - Per-User Chat Soft Delete Validation](../milestones/milestone-5-chat-soft-delete-validation.md)
**Design Reference**: [Per-User Chat Soft Delete + Admin Notifications](../../specs/chat-soft-delete_design.md)
**Estimated Time**: 1 hour
**Dependencies**: Task 7
**Status**: Completed

---

## Objective

Update ACP requirements, milestone notes, and progress tracking so the validated chat soft-delete feature is represented cleanly in the project roadmap.

## Context

The feature spec lives in `specs/`, while ACP currently tracks only the collaboration slice through M4. Once backend and frontend verification are complete, ACP needs to capture the feature without relying on out-of-band repo knowledge.

## Steps

### 1. Sync high-level project docs
- Updated `agent/design/requirements.md` to include the validated per-user chat hide and admin notification capability in the functional requirements summary.
- Synced milestone notes and recent-work tracking so the verified implementation state is captured inside ACP instead of only in `specs/` and tests.

### 2. Sync progress tracking
- Marked all M5 task statuses accurately in `agent/progress.yaml` and closed the milestone.
- Reset `current_milestone` to `null` and updated the follow-up steps now that M5 is complete.

### 3. Record residual gaps
- Documented the remaining manual/environment-dependent follow-up:
  - the `ChatView.vue` sidebar soft-delete affordance was source-reviewed in Task 7 rather than mounted in a component test
  - the broader DB-backed handler suite still depends on `TEST_DATABASE_URL` and `TEST_REDIS_URL`

## Verification

- [x] ACP requirements mention the validated soft-delete behavior if needed.
- [x] Progress tracking matches the completed M5 tasks.
- [x] Remaining gaps are explicit and actionable.

## Expected Output

ACP now records the validated soft-delete feature as completed project history, with the remaining manual/environment-dependent follow-up called out explicitly instead of being left as implicit repo knowledge.

**Estimated Completion Date**: 2026-04-07
