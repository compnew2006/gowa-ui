# Milestone 9: Hosted Provisioning Path

**Goal**: Deliver the end-to-end hosted bootstrap, trial issuance, and automatic activation path for newly provisioned hosted deployments.
**Duration**: 3 weeks
**Dependencies**: Milestone 7, Milestone 8
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 3 into ACP. It adds the hosted-only bootstrap surface on the managed instance, persists bootstrap session state in the control plane, and completes trial issuance plus activation automatically through the provisioning service.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Managed instance hosted bootstrap
- Hosted runtime identity storage
- `deployment_id` validation for hosted tokens
- `POST /internal/license/bootstrap` with hosted-only auth

### 2. Provisioning orchestration
- Bootstrap session persistence before issuer calls
- Trial issuance with retry-safe `request_id`
- Automatic activation with bounded retry logic

## Success Criteria

- [ ] `POST /internal/license/bootstrap` is unavailable to public traffic and works only in hosted mode.
- [ ] `bootstrap_nonce` follows the `CREATED -> CONSUMED -> EXPIRED` lifecycle.
- [ ] Hosted trial onboarding completes without manual customer activation steps.
- [ ] End-to-end hosted onboarding finishes in under two minutes for the happy path.

## Tasks

1. [Task 15: Add Hosted Runtime State and Protected Bootstrap Endpoint](../tasks/task-15-add-hosted-runtime-state-and-protected-bootstrap-endpoint.md) - Extend the managed instance with hosted identity and the internal bootstrap surface.
2. [Task 16: Persist Bootstrap Sessions and Orchestrate Trial Issuance](../tasks/task-16-persist-bootstrap-sessions-and-orchestrate-trial-issuance.md) - Have the provisioning service persist bootstrap state and call the issuer safely.
3. [Task 17: Automate Hosted Activation and Deployment State Finalization](../tasks/task-17-automate-hosted-activation-and-deployment-state-finalization.md) - Finish hosted onboarding by activating the managed instance and recording final deployment state.

## Testing Requirements

- [ ] Handler tests for the hosted bootstrap endpoint.
- [ ] Integration tests for hosted activation success and wrong-`deployment_id` rejection.
- [ ] Retry-path coverage for stale nonce, timeout replay, and activation retry behavior.

## Documentation Requirements

- [ ] Record hosted bootstrap auth requirements and nonce semantics.
- [ ] Record deployment-state ownership across managed instance and provisioning service.
- [ ] Link every ACP task back to the finalized hosted licensing docs.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Provisioning retries accidentally create duplicate hosted licenses | High | Medium | Persist bootstrap sessions before issuer calls and always retry with the same `request_id`. |
| Hosted token validation regresses self-hosted activation | High | Medium | Make `deployment_id` validation conditional on hosted mode only and re-run offline activation regression coverage. |

**Next Milestone**: [Milestone 10: Abuse Controls & Observability](milestone-10-hosted-licensing-abuse-controls-observability.md)
**Blockers**: Milestone 8 must define the issuer APIs and persistence model first
**Notes**: This milestone spans all three hosted actors: the managed instance, the private issuer, and the provisioning service.
