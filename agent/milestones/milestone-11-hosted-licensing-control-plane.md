# Milestone 11: Hosted Control Plane

**Goal**: Manage hosted deployment lifecycle centrally through heartbeat, grace, suspension, and key rotation operations.
**Duration**: 4 weeks
**Dependencies**: Milestone 8, Milestone 9, Milestone 10
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 5 into ACP. It connects the managed instance heartbeat loop to the provisioning control plane, adds hosted deployment grace and suspension state, and operationalizes hosted key rotation without rewriting the offline verifier.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Hosted lifecycle control
- Hosted-only outbound heartbeat from the managed instance
- Control-plane liveness tracking and grace transitions
- Suspension and resume behavior reflected back to the instance

### 2. Key rotation operations
- Current and previous `kid` support
- Hosted verify-only grace window for previous keys
- Forced hosted renewal path during rotation

## Success Criteria

- [ ] Hosted instances send heartbeat every 24 hours in hosted mode only.
- [ ] Missing heartbeat for 48 hours enters control-plane grace and later suspension.
- [ ] Suspended hosted deployments are enforced locally by the managed instance.
- [ ] Hosted signing uses the current key only while previous-key verification remains available during the grace window.

## Tasks

1. [Task 20: Add Hosted Heartbeat Sender and Local Suspension Enforcement](../tasks/task-20-add-hosted-heartbeat-sender-and-local-suspension-enforcement.md) - Implement the hosted-only outbound heartbeat loop and local response handling in the managed instance.
2. [Task 21: Build Control-Plane Heartbeat, Grace, and Suspension State Machine](../tasks/task-21-build-control-plane-heartbeat-grace-and-suspension-state-machine.md) - Track deployment liveness and enforce hosted lifecycle transitions in the control plane.
3. [Task 22: Implement Hosted Key Rotation and Renewal Operations](../tasks/task-22-implement-hosted-key-rotation-and-renewal-operations.md) - Add hosted rotation semantics and renewal flows that coexist with offline self-hosted limits.

## Testing Requirements

- [ ] Hosted heartbeat happy path and missing-heartbeat state transition coverage.
- [ ] Suspension and resume coverage between control plane and managed instance.
- [ ] End-to-end hosted rotation tests covering current and previous key behavior.

## Documentation Requirements

- [ ] Document hosted heartbeat cadence, grace-window semantics, and suspension rules.
- [ ] Document hosted rotation policy, including the 90-day previous-key verify-only window.
- [ ] Document the accepted self-hosted offline limitations around rotation and revocation.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Hosted suspension logic leaks into self-hosted offline mode | High | Low | Gate all heartbeat and suspension behavior behind hosted runtime state only. |
| Rotation rollout breaks existing hosted deployments or overpromises offline control | High | Medium | Keep signing on the current key only, verify previous keys during grace, and document offline limitations explicitly. |

**Next Milestone**: [Milestone 12: Hardening & Future-Proofing](milestone-12-hosted-licensing-hardening-future-proofing.md)
**Blockers**: The hosted bootstrap, issuance, and observability layers must exist before lifecycle control is reliable
**Notes**: Revocation and heartbeat remain hosted-only features. Self-hosted offline mode is explicitly outside the forced-lifecycle contract.
