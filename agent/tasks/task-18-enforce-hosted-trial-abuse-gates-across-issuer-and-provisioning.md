# Task 18: Enforce Hosted Trial Abuse Gates Across Issuer and Provisioning

**Milestone**: [M10 - Abuse Controls & Observability](../milestones/milestone-10-hosted-licensing-abuse-controls-observability.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 1-2 days
**Dependencies**: Milestone 9 completion
**Status**: Not Started

---

## Objective

Block or slow suspicious hosted trial requests before resource provisioning and surface issuer-side abuse signals consistently.

## Context

The hosted licensing docs require velocity checks and cooldown evaluation keyed by email, IP, ASN, and domain history. The provisioning service must use these signals before booting expensive hosted infrastructure.

## Steps

### 1. Model abuse signals
- Persist the issuer and provisioning signals needed for cooldown and repeated-trial evaluation.

### 2. Enforce the gates
- Reject clearly abusive requests before provisioning starts.
- Flag suspicious requests for manual review when they should not be auto-approved.

### 3. Add service-level protection
- Apply issuer-side rate limits and alert thresholds to abnormal issuance volume.

## Verification

- [ ] Blocked hosted trial requests fail before infrastructure creation.
- [ ] Cooldown rules are enforced consistently.
- [ ] Suspicious requests are recorded for operations review.

## Expected Output

Hosted trial abuse checks operate across issuer and provisioning layers, reducing duplicate or fraudulent trial creation before resources are consumed.
