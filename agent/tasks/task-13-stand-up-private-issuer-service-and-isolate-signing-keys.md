# Task 13: Stand Up Private Issuer Service and Isolate Signing Keys

**Milestone**: [M8 - Private Issuer Service](../milestones/milestone-8-hosted-licensing-private-issuer-service.md)
**Design Reference**: [Hosted Licensing Design](../design/hosted-licensing-design.md), [Hosted Licensing API Contract](../../docs/licensing_hosted_api_contract.md), [Hosted Licensing Implementation Plan](../../docs/licensing_hosted_implementation_plan.md)
**Estimated Time**: 2 days
**Dependencies**: Milestone 7 completion
**Status**: Not Started

---

## Objective

Create the dedicated internal issuer boundary and move all private signing-key usage into it exclusively.

## Context

The hosted licensing design requires the Ed25519 private key to exist only inside the issuer service. No public traffic may reach this service, and all calls must be authenticated via `mTLS` or short-lived internal JWT.

## Steps

### 1. Define the issuer deployable boundary
- Create the standalone service module or binary and define its config surface.

### 2. Move private-key loading into the issuer
- Ensure no other service or client binary loads or stores the private key.

### 3. Harden ingress and identity
- Restrict access to provisioning-service callers only.
- Apply network-policy and internal-auth requirements from the source docs.

## Verification

- [ ] The private key is loaded only by the issuer service.
- [ ] The issuer is not reachable from public traffic.
- [ ] Internal callers authenticate with the approved service-to-service mechanism.

## Expected Output

An isolated issuer service boundary that becomes the single signing authority for hosted licensing.
