# Task 19: Expose License Status, Audit Visibility, and Operations Workflows

**Milestone**: [M10 - Abuse Controls & Observability](../milestones/milestone-10-hosted-licensing-abuse-controls-observability.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 1-2 days
**Dependencies**: Task 18
**Status**: Not Started

---

## Objective

Surface canonical hosted license status and privacy-hardened audit data to the internal operations workflows that support onboarding, abuse review, and reconciliation.

## Context

Operations should not need direct database reads to understand issued-license state. The hosted licensing plan calls for canonical internal read APIs plus privacy-hardened audit output and visibility into suspicious patterns.

## Steps

### 1. Expose canonical license reads
- Implement or finish `GET /internal/licenses/{license_id}` and indexed lookups for related identifiers.

### 2. Expose privacy-safe audit views
- Provide operations-oriented visibility into issued licenses, abuse signals, and suspicious patterns without raw sensitive fields.

### 3. Define the support workflows
- Capture how operations reconcile onboarding, review suspicious signups, and handle revoke or reissue decisions.

## Verification

- [ ] Operations can retrieve canonical hosted license state without direct DB access.
- [ ] Audit views mask or hash sensitive values according to the finalized docs.
- [ ] Reconciliation and abuse-review workflows are defined and actionable.

## Expected Output

Internal operations have the hosted license visibility they need through supported APIs and privacy-hardened views rather than ad hoc manual inspection.
