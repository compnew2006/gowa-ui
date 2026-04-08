# Task 20: Add Hosted Heartbeat Sender and Local Suspension Enforcement

**Milestone**: [M11 - Hosted Control Plane](../milestones/milestone-11-hosted-licensing-control-plane.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md), [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)
**Estimated Time**: 2 days
**Dependencies**: Milestone 10 completion
**Status**: Not Started

---

## Objective

Add hosted-only outbound heartbeat behavior to the managed instance and enforce control-plane grace or suspension responses locally.

## Context

Heartbeat and suspension are hosted-only capabilities. The managed instance must not introduce any outbound dependency for self-hosted offline mode, but hosted deployments must regularly report health and honor control-plane status changes.

## Steps

### 1. Add the outbound heartbeat loop
- Send heartbeat every 24 hours in hosted mode only.

### 2. Cache and interpret status
- Persist the last control-plane response locally and react to `active`, `grace`, and `suspended` status.

### 3. Enforce local runtime behavior
- Keep operating during `grace` with an admin-only warning.
- Enter hosted lock mode on `suspended`.

## Verification

- [ ] No heartbeat behavior exists in self-hosted offline mode.
- [ ] Hosted deployments send heartbeat on the expected cadence.
- [ ] `grace` and `suspended` states are enforced locally as designed.

## Expected Output

The managed instance behaves like a hosted client of the control plane without compromising self-hosted offline deployment behavior.
