# Task 16: Persist Bootstrap Sessions and Orchestrate Trial Issuance

**Milestone**: [M9 - Hosted Provisioning Path](../milestones/milestone-9-hosted-licensing-provisioning-path.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2-3 days
**Dependencies**: Task 15
**Status**: Not Started

---

## Objective

Have the provisioning service persist hosted bootstrap sessions first and then orchestrate trial issuance safely with issuer-side idempotency.

## Context

The control plane must persist bootstrap session state before asking the issuer to sign anything. Retries must reuse the same `request_id`, and expired or stale nonces must trigger re-bootstrap rather than unsafe reuse.

## Steps

### 1. Add deployment bootstrap orchestration
- Create hosted deployment records and wait for managed-instance readiness.

### 2. Persist bootstrap sessions
- Call `POST /internal/license/bootstrap`.
- Persist `bootstrap_nonce`, `deployment_id`, `hwid_hash`, and expiry state before the issuer call.

### 3. Orchestrate trial issuance
- Call `POST /internal/licenses/issue-trial` with the correct hosted payload.
- Reuse the same `request_id` on retries and re-bootstrap when TTL is too close to expiry.

## Verification

- [ ] Bootstrap state is persisted before any issuer call.
- [ ] Expired bootstrap sessions are never sent to the issuer.
- [ ] Retry-after-timeout behavior does not create duplicate hosted licenses.

## Expected Output

The provisioning service becomes the reliable bridge between the managed-instance bootstrap step and the issuer’s idempotent hosted trial API.
