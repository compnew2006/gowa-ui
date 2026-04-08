# Milestone 10: Abuse Controls & Observability

**Goal**: Prevent hosted trial abuse and give operations a complete, privacy-hardened view of hosted license state and suspicious activity.
**Duration**: 2 weeks
**Dependencies**: Milestone 8, Milestone 9
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 4 into ACP. It adds abuse gating before expensive provisioning, issuer-side velocity limits, canonical hosted license read paths, and privacy-hardened visibility for operations.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Abuse prevention
- Velocity checks on email, IP, ASN, and domain
- Cooldown evaluation for repeated hosted trials
- Early provisioning rejection for clearly blocked signups

### 2. Operations visibility
- Canonical hosted license lookup API
- Privacy-hardened audit and abuse signal views
- Internal operations workflows for reconciliation and review

## Success Criteria

- [ ] Abusive or duplicate hosted trial signups are blocked before provisioning expensive resources.
- [ ] Suspicious signups are logged with enough signal for operations review.
- [ ] Operations can fetch canonical hosted license state without direct DB reads.
- [ ] Audit exports avoid raw sensitive values and apply privacy hardening consistently.

## Tasks

1. [Task 18: Enforce Hosted Trial Abuse Gates Across Issuer and Provisioning](../tasks/task-18-enforce-hosted-trial-abuse-gates-across-issuer-and-provisioning.md) - Add issuer and provisioner abuse checks, cooldowns, and early rejection behavior.
2. [Task 19: Expose License Status, Audit Visibility, and Operations Workflows](../tasks/task-19-expose-license-status-audit-visibility-and-operations-workflows.md) - Surface canonical read paths and the internal operations views needed for hosted licensing support.

## Testing Requirements

- [ ] Coverage for blocked trial signup paths before infrastructure creation.
- [ ] Coverage for issuer rate limits and abnormal-issuance alert thresholds.
- [ ] Coverage for privacy filtering in audit export and status read paths.

## Documentation Requirements

- [ ] Document abuse thresholds and cooldown semantics.
- [ ] Document which service owns each audit and abuse signal.
- [ ] Capture operations workflows for review, reconciliation, and reissue decisions.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Abuse gates are too weak and expensive hosted resources are provisioned for bad actors | High | Medium | Evaluate abuse signals before provisioning and keep thresholds observable and tunable. |
| Operations visibility requires unsafe direct database access | Medium | Medium | Provide canonical internal read APIs and privacy-safe views instead of ad hoc DB inspection. |

**Next Milestone**: [Milestone 11: Hosted Control Plane](milestone-11-hosted-licensing-control-plane.md)
**Blockers**: Hosted issuance and provisioning flow from M8-M9 must exist first
**Notes**: Dashboard or tooling UX can evolve later, but the underlying APIs, audit model, and privacy contract must be established here.
