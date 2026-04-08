# Task 17: Automate Hosted Activation and Deployment State Finalization

**Milestone**: [M9 - Hosted Provisioning Path](../milestones/milestone-9-hosted-licensing-provisioning-path.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2 days
**Dependencies**: Task 16
**Status**: Not Started

---

## Objective

Complete hosted onboarding by activating the managed instance automatically and persisting the final hosted deployment state.

## Context

Hosted customers must not see a manual activation step. The provisioning service must use the already-issued token with the existing activation endpoint, record success or bounded retry results, and escalate safely if activation still fails.

## Steps

### 1. Call the existing activation endpoint
- Use the already-issued hosted token with `POST /api/license/activate`.

### 2. Finalize deployment state
- Persist the activation outcome and canonical hosted license metadata in control-plane deployment records.

### 3. Handle failure safely
- Retry activation with the same token using bounded exponential backoff.
- Mark the deployment `activation_pending_manual` and alert operations if retries are exhausted.

## Verification

- [ ] Hosted onboarding finishes without manual customer activation in the happy path.
- [ ] Activation retries reuse the same token rather than issuing a second hosted license.
- [ ] Failure state is visible to operations when bounded retries are exhausted.

## Expected Output

Hosted onboarding ends with either a ready licensed instance or an explicit operations-handled failure state, never an ambiguous partial activation.
