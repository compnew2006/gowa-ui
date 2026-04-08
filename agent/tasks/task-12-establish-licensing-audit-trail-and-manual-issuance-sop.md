# Task 12: Establish Licensing Audit Trail and Manual Issuance SOP

**Milestone**: [M7 - Operational Hardening](../milestones/milestone-7-hosted-licensing-operational-hardening.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 1 day
**Dependencies**: Task 11
**Status**: Not Started

---

## Objective

Define the operational controls and audit requirements for manual license issuance and reissue handling in the self-hosted path.

## Context

The PRD requires every manual issuance or reissue request to be tied to an existing `customer_id`, registered email, documented reason, and named approver, with explicit visibility into reissue counts and privacy-safe audit retention.

## Steps

### 1. Define the SOP contract
- Capture required fields, approvals, and reissue escalation rules.

### 2. Define the audit event shape
- Use `customer_id` rather than direct email where possible.
- Mask exported IP data and keep token or HWID material hashed according to the source docs.

### 3. Wire the operational outputs
- Identify where the SOP lives and how audit events are retained or forwarded externally.

## Verification

- [ ] The manual issuance flow requires `customer_id`, email, reason, and approver identity.
- [ ] Reissue counting and escalation conditions are defined.
- [ ] Audit guidance aligns with the privacy constraints in the finalized docs.

## Expected Output

A documented and enforceable manual issuance process plus a privacy-safe audit trail contract that can be applied consistently to self-hosted licensing operations.
