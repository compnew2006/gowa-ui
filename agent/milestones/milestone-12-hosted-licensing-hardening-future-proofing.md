# Milestone 12: Hardening & Future-Proofing

**Goal**: Close the hosted licensing program with final hardening, documented self-hosted limits, runbooks, and end-to-end validation.
**Duration**: 2 weeks
**Dependencies**: Milestone 7, Milestone 8, Milestone 9, Milestone 10, Milestone 11
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 6 into ACP. It is the final hardening pass across all three hosted licensing actors and the self-hosted offline path. It adds the optional `install_id` layer, formalizes accepted limitations, and validates the complete hosted licensing surface through runbooks and security review.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Final technical hardening
- `install_id` added as an additive self-hosted layer
- End-to-end hosted rotation and renewal validation
- Final security review of every hosted licensing surface

### 2. Operational readiness
- Runbooks for key leak, hosted cancellation, and HWID migration
- Explicit documentation of VM cloning and offline rotation limitations
- Final cross-service failure drills and recovery expectations

## Success Criteria

- [ ] `install_id` exists as an additive guardrail and its limits are documented honestly.
- [ ] Key rotation is tested end to end across hosted and self-hosted flows where applicable.
- [ ] Runbooks exist for the critical hosted licensing incident scenarios.
- [ ] Final security review is completed across the managed instance, issuer, and provisioning control plane.

## Tasks

1. [Task 23: Add install_id Guardrail and Document Self-Hosted Limits](../tasks/task-23-add-install-id-guardrail-and-document-self-hosted-limits.md) - Add the extra self-hosted layer and capture the accepted technical limitations precisely.
2. [Task 24: Execute End-to-End Hardening, Runbooks, and Final Security Review](../tasks/task-24-execute-end-to-end-hardening-runbooks-and-final-security-review.md) - Close the program with failure drills, runbooks, and final validation.

## Testing Requirements

- [ ] End-to-end rotation and renewal drills across hosted flows.
- [ ] Regression coverage proving the self-hosted offline path still works after `install_id` addition.
- [ ] Validation of failure and recovery scenarios from the runbooks.

## Documentation Requirements

- [ ] Publish runbooks for key leak, hosted cancellation, and HWID migration.
- [ ] Document accepted VM cloning and offline revocation limits contractually and technically.
- [ ] Sync final ACP progress and milestone notes after implementation closes.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Final hardening work is deferred and the hosted licensing stack ships without incident readiness | High | Medium | Treat runbooks and failure drills as required milestone outputs, not optional cleanup. |
| `install_id` is misunderstood as clone prevention | Medium | High | Document it explicitly as an additive signal, not a complete anti-cloning guarantee. |

**Next Milestone**: TBD
**Blockers**: Hosted lifecycle control and rotation behavior from M11 must be available before final hardening can close
**Notes**: This milestone should end with the hosted licensing roadmap fully documented, validated, and ready for long-term maintenance rather than just feature-complete code.
