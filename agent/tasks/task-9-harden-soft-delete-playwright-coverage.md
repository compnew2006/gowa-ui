# Task 9: Harden Soft Delete Playwright Coverage

**Milestone**: [M6 - Chat Soft Delete End-to-End Coverage](../milestones/milestone-6-chat-soft-delete-e2e.md)
**Design Reference**: [Per-User Chat Soft Delete + Admin Notifications](../../specs/chat-soft-delete_design.md)
**Estimated Time**: 2 hours
**Dependencies**: Milestone 5 completion
**Status**: Completed

---

## Objective

Turn the draft soft-delete Playwright coverage into a deterministic browser-level regression spec that matches the shipped UI and can be validated locally.

## Context

The repo already contains draft E2E work in `frontend/e2e/tests/chat/soft-delete.spec.ts` and `frontend/e2e/helpers/api.ts`, but it does not yet line up with the actual implementation details:

- the sidebar uses inline two-click confirmation rather than a modal dialog
- sidebar entries are keyed by `data-testid="chat-sidebar-entry"` and visible contact text, not `data-contact-id`
- the contact info panel uses `window.confirm(...)` rather than the sidebar’s inline confirm state
- the local environment may not have the frontend/backend servers needed for full runtime execution

This task hardens that coverage and records the honest local verification status.

## Steps

### 1. Stabilize the test scaffolding
- Reworked `frontend/e2e/tests/chat/soft-delete.spec.ts` to use the shipped `ChatView.vue` sidebar entry test IDs, inline two-click confirmation flow, `#info-button`, and the contact-panel browser confirm dialog.
- Reused the existing E2E API helper pattern to create deterministic contacts plus temporary roles/users for explicit permission coverage instead of relying on ambiguous seeded defaults.

### 2. Cover the shipped browser flows
- Added a browser test for the negative permission case across both the sidebar and contact info panel affordances.
- Added a browser test for the sidebar inline confirm flow plus admin notification click-through navigation.
- Added a browser test for the contact info panel browser-confirm flow and post-delete redirect back to `/chat`.

### 3. Validate locally
- Ran Playwright discovery mode to prove the spec parses and registers cleanly:
  - `npm run test:e2e -- --list e2e/tests/chat/soft-delete.spec.ts`
- Confirmed the local frontend (`http://localhost:3000`) and backend (`http://localhost:8080`) are both down in this environment, so full runtime execution remains explicitly environment-dependent.

## Verification

- [x] The soft-delete E2E spec matches the current shipped selectors and confirmation behavior.
- [x] Permission coverage does not rely on ambiguous default seeded roles.
- [x] Local Playwright validation is recorded honestly.
- [x] Any runtime-only blocker is documented explicitly.

## Expected Output

The repo now contains a stable, ACP-tracked Playwright soft-delete spec plus an honest local verification result: discovery succeeded, while full runtime execution remains blocked on local server availability.

**Next Task**: [Task 10: Sync ACP Docs for Soft Delete E2E Coverage](task-10-sync-acp-docs-for-soft-delete-e2e.md)
**Estimated Completion Date**: 2026-04-07T11:03:43+02:00
