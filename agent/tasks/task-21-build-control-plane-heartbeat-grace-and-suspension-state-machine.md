# Task 21: Build Control-Plane Heartbeat, Grace, and Suspension State Machine

**Milestone**: [M11 - Hosted Control Plane](../milestones/milestone-11-hosted-licensing-control-plane.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2-3 days
**Dependencies**: Task 20
**Status**: Not Started

---

## Objective

Track hosted deployment liveness centrally and enforce the grace, suspension, and resume state machine in the control plane.

## Context

The provisioning control plane must ingest heartbeat signals, mark deployments as `grace` after 48 hours of missing heartbeat, suspend them later when required, and expose the resulting status back to the managed instance.

## Steps

### 1. Ingest and persist heartbeat activity
- Record liveness timestamps and deployment status changes in the control-plane store.

### 2. Implement lifecycle transitions
- Move deployments through `active`, `grace`, and `suspended`.
- Provide a resume path for payment recovery or operator action.

### 3. Reflect status to hosted instances
- Return clear runtime semantics so the managed instance can enforce the proper mode.

## Verification

- [ ] Missing heartbeat for 48 hours transitions the deployment into `grace`.
- [ ] Suspended deployments are reflected back to the instance.
- [ ] Resume behavior is possible through an explicit control-plane action.

## Expected Output

The hosted control plane becomes the source of truth for deployment liveness and suspension state, with explicit status transitions that the managed instance can honor.
