# Task 23: Add install_id Guardrail and Document Self-Hosted Limits

**Milestone**: [M12 - Hardening & Future-Proofing](../milestones/milestone-12-hosted-licensing-hardening-future-proofing.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 1 day
**Dependencies**: Milestone 11 completion
**Status**: Not Started

---

## Objective

Add `install_id` as an extra self-hosted signal and document the accepted technical limits of self-hosted offline licensing precisely.

## Context

The PRD treats `install_id` as an additive hardening layer only. It is not a guarantee against VM cloning, and the self-hosted offline path still carries accepted limitations around rotation and revocation.

## Steps

### 1. Introduce the additive signal
- Add `install_id` in a way that does not break the existing offline licensing core.

### 2. Document the limits
- Capture the accepted VM cloning and offline rotation limitations explicitly.

### 3. Align the operational story
- Ensure docs and runbooks do not overstate what `install_id` actually guarantees.

## Verification

- [ ] `install_id` is additive and non-breaking.
- [ ] Self-hosted limitations are documented explicitly.
- [ ] No contract text claims clone prevention that the system cannot enforce.

## Expected Output

The self-hosted licensing path gains one more guardrail, and the remaining accepted limitations are documented honestly for engineering and operations.
