# Hosted Licensing Design

**Status**: Finalized source-doc consolidation
**Last Updated**: 2026-04-08
**Purpose**: Fast ACP session-read summary of the hosted licensing design. This file is a consolidation of the finalized docs under `docs/`; it is not the source of truth and must stay aligned with them.

---

## Authoritative Source Documents

- [`docs/licensing_hosted_api_contract.md`](../../docs/licensing_hosted_api_contract.md)
- [`docs/licensing_hosted_implementation_plan.md`](../../docs/licensing_hosted_implementation_plan.md)
- [`docs/licensing_hosted_PRD&Milestone.md`](../../docs/licensing_hosted_PRD&Milestone.md)

## Design Intent

Hosted licensing is an additive control-plane layer around the existing offline licensing core. The current self-hosted activation path remains canonical for offline use, while hosted deployments gain:

- internal bootstrap for newly provisioned deployments
- private issuer-backed hosted trial and paid issuance
- control-plane visibility for issuance, abuse detection, heartbeat, and suspension

The existing offline verifier remains the center of trust on the managed instance. Hosted control-plane behavior must extend it without breaking self-hosted operation.

## Critical Constraints (Must Never Violate)

- `private.key` lives in the Issuer service only.
- `GET /api/license/bootstrap` stays public.
- `POST /api/license/activate` keeps its current behavior; only additive hardening such as rate limiting and failed-attempt auditing may be added.
- `internal/license/token.go` and `internal/license/service.go` must not be rewritten to change the core offline verification model.
- Self-hosted offline mode must not require heartbeat.
- `bootstrap_nonce` must be single-use, must expire after `5 minutes`, and must be consumed atomically with hosted issuance state changes.

## Existing Code Boundaries

- `internal/handlers/license.go`
- `internal/license/service.go`
- `internal/license/token.go`
- `cmd/whatomate/main.go`

These files define the current licensing entry points and verifier contract. Hosted work should layer around them rather than replace them.

## Service Roles

### Managed Whatomate Instance

- keeps the current public self-hosted licensing endpoints
- validates and installs signed licenses locally
- stores hosted deployment identity locally when running in hosted mode
- sends heartbeat outbound only in hosted mode

### Private Issuer Service

- holds the Ed25519 private signing key
- owns `request_id` idempotency
- owns hosted issuance history, nonce consumption, and revision assignment
- exposes internal APIs for trial issuance, paid issuance, suspension, and canonical hosted-license reads

### Provisioning Service

- creates hosted deployments and tracks deployment lifecycle
- calls the managed instance bootstrap endpoint
- persists bootstrap session state before issuer calls
- activates the managed instance automatically after issuance
- owns heartbeat-derived deployment lifecycle from the control-plane side

## Architecture Boundaries

### Self-Hosted Offline Path

- public `GET /api/license/bootstrap`
- public `POST /api/license/activate`
- local token verification on the managed instance
- no mandatory online check, heartbeat, or hosted control-plane dependency

### Hosted Path

- internal bootstrap handshake between provisioning and the managed instance
- issuer-backed hosted trial and paid issuance
- outbound heartbeat from the managed instance to the control plane
- suspension and grace driven by hosted control-plane state only

## Canonical API Surface

### Existing Public Endpoints

- `GET /api/license/bootstrap`
  - remains public
  - remains the self-hosted bootstrap surface

- `POST /api/license/activate`
  - remains public
  - keeps current activation semantics
  - gains rate limiting and failed-attempt audit logging only

### Hosted-Only Internal Endpoints

- `POST /internal/license/bootstrap`
  - hosted mode only
  - internal auth via `mTLS` or short-lived internal JWT
  - returns `deployment_id`, `hwid_full`, `hwid_hash`, `bootstrap_nonce`, and `nonce_expires_at`

- `POST /internal/licenses/issue-trial`
  - issuer-owned hosted trial issuance
  - idempotent by `request_id`
  - consumes `bootstrap_nonce` atomically

- `POST /internal/licenses/issue-paid`
  - issuer-owned hosted renewal and upgrade issuance
  - revision is issuer-owned and monotonic per `license_family_id`

- `GET /internal/licenses/{license_id}`
  - canonical hosted-license read path for provisioning and operations

- `POST /internal/licenses/suspend`
  - hosted control-plane suspension state change

## Control-Plane Persistence Model

### Shared Postgres Cluster, Service-Owned Schemas

Use one hosted control-plane Postgres cluster in v1 with explicit ownership:

- `issuer` schema owned by the Private Issuer Service
- `provisioner` schema owned by the Provisioning Service

No service should run migrations for the other service's schema.

### Core Hosted Records

- `bootstrap_sessions`
- `issuer_requests`
- `issued_licenses`
- `audit_events`
- `abuse_signals`
- deployment and activation lifecycle records owned by provisioning

### Managed-Instance Local Hosted State

Persist locally on the managed instance:

- `deployment_id`
- `hosted_mode`
- `status`
- `last_heartbeat_at`
- `suspended_at`

Preferred shape: a dedicated local DB table such as `hosted_instance_state`.

## Hosted Flow Summary

1. Provisioning creates a hosted deployment record.
2. Provisioning calls `POST /internal/license/bootstrap` on the managed instance.
3. Provisioning persists the returned `bootstrap_nonce` session before calling the issuer.
4. Provisioning calls `POST /internal/licenses/issue-trial` with a fresh `request_id`.
5. Issuer validates idempotency, consumes the nonce atomically, signs the hosted token, and persists issuance history.
6. Provisioning activates the managed instance through the existing public activation endpoint using the issued token.
7. Hosted instances later send outbound heartbeat; the control plane manages `active`, `grace`, and `suspended` lifecycle state.

## Operational Rules

- Hosted deployments must validate `deployment_id` locally.
- Self-hosted tokens must remain exempt from hosted-only `deployment_id` checks.
- Repeat bootstrap for the same deployment invalidates older unconsumed nonces.
- Expired bootstrap sessions must cause re-bootstrap, not unsafe nonce reuse.
- Retry after timeout must reuse the same `request_id` and must not create a second hosted license.
- Issuer signing uses the current `kid` only; previous `kid` is verify-only during the hosted grace window.

## Milestone Mapping

- ACP `M7` -> PRD Milestone 1: Operational hardening on the current public self-hosted activation path.
- ACP `M8` -> PRD Milestone 2: Private issuer service, key isolation, control-plane persistence, and idempotent issuance.
- ACP `M9` -> PRD Milestone 3: Hosted bootstrap, trial issuance orchestration, and automatic activation.
- ACP `M10` -> PRD Milestone 4: Abuse controls, audit privacy, and operations observability.
- ACP `M11` -> PRD Milestone 5: Heartbeat, grace/suspension lifecycle, and hosted key rotation.
- ACP `M12` -> PRD Milestone 6: Final hardening, `install_id`, runbooks, and end-to-end validation.

## Accepted Limitations and Non-Goals

- No replacement of the current offline verification core.
- No mandatory heartbeat or revocation path for self-hosted offline deployments.
- No TPM-backed anti-cloning guarantee in this phase.
- No multi-region hosted control plane in v1.
- No integrated payment processing in this phase.

## Review Checklist For Future Sessions

- Confirm all new hosted work stays additive to the public self-hosted path.
- Confirm the issuer remains the only holder of the private signing key.
- Confirm hosted-only runtime behavior is gated behind hosted mode.
- Confirm nonce lifecycle and idempotency remain issuer-owned and transactional.
- Confirm docs in `docs/` remain the authority if any ACP summary ever drifts.
