# Task 14: Implement Issuer Persistence, Idempotent Issuance, and Internal APIs

**Milestone**: [M8 - Private Issuer Service](../milestones/milestone-8-hosted-licensing-private-issuer-service.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2-3 days
**Dependencies**: Task 13
**Status**: Not Started

---

## Objective

Build the issuer-owned persistence model plus the internal hosted issuance APIs with correct idempotency and nonce semantics.

## Context

The issuer must own `request_id` idempotency, atomically consume `bootstrap_nonce`, assign monotonic hosted revisions, and expose canonical read paths for hosted license state.

## Steps

### 1. Add issuer-owned persistence
- Create the issuer schema and tables for bootstrap sessions, issuer requests, issued licenses, audit events, and abuse signals.

### 2. Implement issuance APIs
- Build `POST /internal/licenses/issue-trial` and `POST /internal/licenses/issue-paid`.
- Add `GET /internal/licenses/{license_id}` for canonical hosted license reads.

### 3. Enforce transactional behavior
- Keep nonce consume, issuance persistence, and idempotency response persistence in one transaction.
- Reject revision conflicts and reused or expired nonces correctly.

## Verification

- [ ] Identical retries with the same `request_id` return the same cached response.
- [ ] Consumed or expired nonces cannot be reused.
- [ ] Paid issuance revisions stay monotonic per `license_family_id`.
- [ ] Canonical hosted license state is readable through the internal API.

## Expected Output

The issuer owns hosted issuance state durably and exposes the internal API contract required by the hosted provisioning flow.
