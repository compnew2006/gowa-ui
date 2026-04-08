# Milestone 8: Private Issuer Service

**Goal**: Establish the isolated internal issuer service that owns the private signing key, control-plane persistence, and idempotent hosted issuance.
**Duration**: 3 weeks
**Dependencies**: Milestone 7
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 2 into ACP. It creates the trust anchor for hosted licensing by moving all Ed25519 private-key usage into a dedicated internal issuer service and adding durable control-plane persistence for nonce lifecycle, request idempotency, and license issuance history.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Isolated issuer boundary
- Dedicated internal issuer deployable
- Private signing key loaded only inside the issuer
- Service-to-service authentication via `mTLS` or short-lived internal JWT

### 2. Control-plane data ownership
- `issuer` schema and issuer-owned tables
- Nonce lifecycle persistence
- Request idempotency and issuance history

## Success Criteria

- [ ] No public endpoint exposes the issuer service.
- [ ] The private signing key does not exist in any other service or client binary.
- [ ] `request_id` idempotency returns the same response for 24 hours on identical retries.
- [ ] Trial and paid issuance APIs persist canonical license history and nonce state transitions.

## Tasks

1. [Task 13: Stand Up Private Issuer Service and Isolate Signing Keys](../tasks/task-13-stand-up-private-issuer-service-and-isolate-signing-keys.md) - Create the deployable boundary and move the signing key into it exclusively.
2. [Task 14: Implement Issuer Persistence, Idempotent Issuance, and Internal APIs](../tasks/task-14-implement-issuer-persistence-idempotent-issuance-and-internal-apis.md) - Build the control-plane schema, issuance APIs, and idempotent request handling.

## Testing Requirements

- [ ] Unit coverage for `request_id` idempotency and nonce transitions.
- [ ] Revision-conflict and monotonic revision tests for paid issuance.
- [ ] Integration coverage for issuer audit writes and canonical read paths.

## Documentation Requirements

- [ ] Document issuer schema ownership and migration boundaries.
- [ ] Document key isolation and internal-auth requirements.
- [ ] Record the control-plane persistence model used by issuer and provisioner.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Key material leaks into another service during bootstrap or deployment | High | Low | Centralize key loading inside the issuer service and audit startup/config paths explicitly. |
| Idempotency or nonce handling is split across services inconsistently | High | Medium | Keep nonce consumption and response caching issuer-owned and transactional. |

**Next Milestone**: [Milestone 9: Hosted Provisioning Path](milestone-9-hosted-licensing-provisioning-path.md)
**Blockers**: None
**Notes**: The issuer and provisioning services may share one Postgres cluster in v1, but schema ownership must remain explicit and isolated.
