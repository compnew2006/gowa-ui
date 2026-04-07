# Milestone 6: Chat Soft Delete End-to-End Coverage

**Goal**: Turn the soft-delete frontend verification into durable browser-level regression coverage and close the remaining UI automation gap for the shipped feature.
**Duration**: 1 week
**Dependencies**: Milestone 5
**Status**: Completed

---

## Overview

Milestone 5 validated the shipped chat soft-delete feature with backend tests, targeted frontend unit coverage, and ACP documentation. The main remaining gap is browser-level coverage for the actual sidebar and contact-panel flows, especially the inline sidebar confirmation model that was only source-reviewed.

This milestone formalizes that follow-up work so the repo has tracked, maintainable E2E coverage instead of loose ad hoc verification files.

The work is driven by:

- [`specs/chat-soft-delete_design.md`](../../specs/chat-soft-delete_design.md)

---

## Deliverables

### 1. Stable Playwright coverage
- Add or harden a Playwright soft-delete spec that matches the current shipped UI behavior.
- Use reliable selectors and test scaffolding for permission-aware users and chat fixtures.
- Validate discovery or execution locally as far as the available environment allows.

### 2. ACP closeout
- Record the E2E coverage status in ACP milestone/task tracking.
- Document any remaining environment-dependent limitations for full execution.

## Success Criteria

- [x] Soft-delete browser coverage is tracked in ACP with a concrete task document.
- [x] The Playwright spec matches the actual sidebar confirmation and contact-panel behavior.
- [x] Local verification is recorded honestly, including any server/environment gaps.
- [x] ACP progress tracking reflects the E2E verification state cleanly.

## Tasks

1. [Task 9: Harden Soft Delete Playwright Coverage](../tasks/task-9-harden-soft-delete-playwright-coverage.md) - Align the soft-delete browser test with the shipped UI and validate it locally.
2. [Task 10: Sync ACP Docs for Soft Delete E2E Coverage](../tasks/task-10-sync-acp-docs-for-soft-delete-e2e.md) - Close the milestone by updating ACP progress, notes, and residual gaps after the E2E pass.

## Testing Requirements

- [x] Permission-gated soft-delete affordances are covered at the browser level.
- [x] Sidebar inline confirmation behavior is covered at the browser level.
- [x] Contact info panel soft-delete flow is covered at the browser level.
- [x] Admin notification click-through behavior is covered or its residual gap is documented explicitly.

## Documentation Requirements

- [x] Add milestone/task tracking for the E2E follow-up slice.
- [x] Keep the milestone linked to the soft-delete design spec.
- [x] Record whether local verification was discovery-only or full runtime execution.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Draft E2E coverage drifts from the shipped UI selectors or confirmation model | High | High | Align the spec directly to the current Vue templates and existing E2E helper patterns before treating it as verified. |
| Full runtime execution remains blocked by missing local servers or test data | Medium | High | Validate spec discovery locally, then document the runtime dependency explicitly in ACP instead of overstating coverage. |

**Next Milestone**: TBD
**Blockers**: None
**Notes**: M6 is a follow-up automation milestone for an already shipped feature, not a net-new product launch. The hardened Playwright spec is discovery-validated locally, while full runtime execution still depends on bringing up the local frontend and backend services.
