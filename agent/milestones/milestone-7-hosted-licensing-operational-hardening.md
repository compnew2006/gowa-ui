# Milestone 7: Operational Hardening

**Goal**: Harden the existing self-hosted activation path and surrounding operational controls without changing the offline activation contract.
**Duration**: 2 weeks
**Dependencies**: None
**Status**: Not Started

---

## Overview

This milestone maps the hosted licensing PRD Milestone 1 into ACP. It is intentionally additive: the public self-hosted bootstrap and activation endpoints stay intact, while the managed instance gains rate limiting, privacy-hardened failed-activation audit events, and operational controls for manual issuance.

The source of truth for this milestone is:

- [Hosted Licensing Design](../design/hosted-licensing-design.md)
- [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md)
- [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
- [Hosted Licensing PRD & Milestones](../../docs/licensing_hosted_PRD&Milestone.md)

## Deliverables

### 1. Public activation hardening
- Rate limiting on `POST /api/license/activate`
- Structured failed-activation audit events
- Token hashing instead of raw token storage in audit paths

### 2. Operational controls
- Manual issuance and reissue SOP with approval requirements
- Privacy rules for source IP and audit export data
- Explicit preservation of existing self-hosted bootstrap behavior

## Success Criteria

- [ ] More than 5 failed activation attempts per hour per source IP return `429`.
- [ ] Every failed activation is recorded with a token hash only, never the raw token.
- [ ] `GET /api/license/bootstrap` remains unchanged for self-hosted users.
- [ ] Manual issuance and reissue controls are documented and enforceable operationally.

## Tasks

1. [Task 11: Harden Public License Activation Endpoint](../tasks/task-11-harden-public-license-activation-endpoint.md) - Add rate limiting and privacy-safe failed-activation logging while preserving current endpoint semantics.
2. [Task 12: Establish Licensing Audit Trail and Manual Issuance SOP](../tasks/task-12-establish-licensing-audit-trail-and-manual-issuance-sop.md) - Define the operational and audit controls that close the self-hosted hardening gap.

## Testing Requirements

- [ ] Unit coverage for activation rate-limit behavior.
- [ ] Regression coverage proving offline self-hosted activation still succeeds.
- [ ] Verification that audit events contain token hashes instead of raw tokens.

## Documentation Requirements

- [ ] Link ACP tracking back to the finalized hosted licensing docs in `docs/`.
- [ ] Document the manual issuance SOP and reissue approval trail.
- [ ] Record privacy rules for audit exports.

## Risks and Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Activation hardening accidentally changes the existing offline path | High | Medium | Keep all changes additive around `POST /api/license/activate` and re-run self-hosted activation regression coverage. |
| Audit logging leaks sensitive token or IP detail | High | Medium | Hash tokens, mask IPs in exports, and review every new audit field before rollout. |

**Next Milestone**: [Milestone 8: Private Issuer Service](milestone-8-hosted-licensing-private-issuer-service.md)
**Blockers**: None
**Notes**: This milestone is the gate for all hosted licensing follow-on work because the public self-hosted surface must remain stable before new hosted-only flows are layered on top.
