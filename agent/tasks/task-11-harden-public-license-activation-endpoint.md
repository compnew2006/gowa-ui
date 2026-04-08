# Task 11: Harden Public License Activation Endpoint

**Milestone**: [M7 - Operational Hardening](../milestones/milestone-7-hosted-licensing-operational-hardening.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 1-2 days
**Dependencies**: None
**Status**: Not Started

---

## Objective

Add rate limiting and privacy-safe failed-activation auditing to `POST /api/license/activate` without changing existing offline self-hosted activation behavior.

## Context

The finalized hosted licensing docs require this hardening as the first milestone because the public self-hosted activation surface is still canonical and must stay stable while hosted-only controls are added elsewhere.

## Steps

### 1. Audit the current activation path
- Review `internal/handlers/license.go`, `internal/license/service.go`, `cmd/whatomate/main.go`, and any relevant middleware.

### 2. Add activation rate limiting
- Apply a per-source-IP limit of 5 failed activation attempts per hour on `POST /api/license/activate`.
- Ensure `GET /api/license/bootstrap` remains unchanged.

### 3. Add failed-activation audit logging
- Emit structured audit events for failed activation attempts.
- Store token hashes only and mask or reduce IP detail for export-oriented paths.

## Verification

- [ ] More than 5 failed activation attempts per hour per IP return `429`.
- [ ] Failed activation events never store or emit raw token values.
- [ ] Self-hosted activation success behavior remains unchanged.

## Expected Output

The managed instance preserves its current offline activation contract while gaining rate limiting and privacy-safe failed-activation audit coverage on the public activation endpoint.
