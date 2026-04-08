# Task 15: Add Hosted Runtime State and Protected Bootstrap Endpoint

**Milestone**: [M9 - Hosted Provisioning Path](../milestones/milestone-9-hosted-licensing-provisioning-path.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2 days
**Dependencies**: Milestone 8 completion
**Status**: Not Started

---

## Objective

Extend the managed instance with hosted deployment identity, hosted-mode state, and the protected internal bootstrap endpoint.

## Context

Hosted licensing requires the managed instance to know its `deployment_id`, accept hosted tokens only when that `deployment_id` matches, and expose a hosted-only bootstrap surface that is separate from the public self-hosted bootstrap path.

## Steps

### 1. Add hosted runtime state
- Persist hosted deployment identity locally.
- Add runtime fields for hosted mode, status, heartbeat timestamps, and suspension timestamps.

### 2. Extend local token validation
- Enforce hosted `deployment_id` checks only when hosted mode is enabled.
- Keep self-hosted tokens exempt from hosted-only checks.

### 3. Add the internal bootstrap endpoint
- Implement `POST /internal/license/bootstrap`.
- Protect it with internal auth and hosted-mode checks only.

## Verification

- [ ] Hosted tokens with the wrong `deployment_id` are rejected.
- [ ] Self-hosted activation remains unchanged.
- [ ] The internal bootstrap endpoint is unavailable to public traffic.

## Expected Output

The managed instance can participate in hosted onboarding safely without altering the public self-hosted licensing flow.
