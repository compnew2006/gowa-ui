# Task 22: Implement Hosted Key Rotation and Renewal Operations

**Milestone**: [M11 - Hosted Control Plane](../milestones/milestone-11-hosted-licensing-control-plane.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 2 days
**Dependencies**: Task 21
**Status**: Not Started

---

## Objective

Add hosted key-rotation operations and renewal flows that support current and previous `kid` handling without changing the offline verifier contract.

## Context

Hosted licensing requires a current-key signing policy, a previous-key verify-only grace window, and forced hosted renewal semantics during rotation. Self-hosted offline binaries remain an accepted limitation and must be documented rather than overpromised.

## Steps

### 1. Add hosted key-ring support
- Support current and previous `kid` values operationally.

### 2. Enforce hosted rotation rules
- Sign only with the current key.
- Allow previous-key verification during the hosted grace window.

### 3. Define renewal and rollout workflows
- Implement the hosted renewal path and document the operator-run rotation procedure.

## Verification

- [ ] Hosted signing uses the current key only.
- [ ] Previous-key verification is limited to the hosted grace window.
- [ ] Hosted renewal and rotation flows are defined and testable.

## Expected Output

Hosted licensing supports controlled key rotation and renewal without promising impossible instant revocation for self-hosted offline binaries.
