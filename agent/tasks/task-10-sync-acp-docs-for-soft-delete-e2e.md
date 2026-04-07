# Task 10: Sync ACP Docs for Soft Delete E2E Coverage

**Milestone**: [M6 - Chat Soft Delete End-to-End Coverage](../milestones/milestone-6-chat-soft-delete-e2e.md)
**Design Reference**: [Per-User Chat Soft Delete + Admin Notifications](../../specs/chat-soft-delete_design.md)
**Estimated Time**: 1 hour
**Dependencies**: Task 9
**Status**: Completed

---

## Objective

Update ACP progress, milestone notes, and requirements context after the soft-delete Playwright coverage has been hardened and locally verified.

## Context

Milestone 6 exists to turn the remaining soft-delete UI automation gap into tracked browser-level coverage. After Task 9, ACP needs to record what was actually verified and what still depends on runtime environment availability.

## Steps

### 1. Sync milestone and task records
- Marked Task 9 and Task 10 accurately in `agent/progress.yaml`, then closed M6 as completed.
- Updated milestone notes and recent-work entries with the discovery-validated E2E outcome.

### 2. Record residual gaps
- Documented that local verification was discovery-only (`playwright test --list`) rather than full runtime execution.
- Kept the missing local frontend/backend services explicit as the remaining runtime dependency for executing the browser tests end to end.

## Verification

- [x] ACP progress matches the M6 execution state.
- [x] Local verification scope is documented explicitly.
- [x] Remaining blockers or follow-up steps are actionable.

## Expected Output

ACP now records the soft-delete E2E coverage state as completed milestone history, with the discovery-only validation scope and runtime server dependency called out explicitly.

**Estimated Completion Date**: 2026-04-07T11:11:23+02:00
