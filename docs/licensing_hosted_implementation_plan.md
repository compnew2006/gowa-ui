# Hosted Licensing Implementation Plan

## Purpose

This plan converts the hosted licensing contract into executable work across:

- `Managed Whatomate Instance`
- `Private Issuer Service`
- `Provisioning Service`

It is aligned with:

- [licensing_hosted_api_contract.md](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/licensing_hosted_api_contract.md)
- [licensing_hosted_PRD&Milestone.md](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docs/licensing_hosted_PRD&Milestone.md)

## Guardrails

- Do not break the current offline self-hosted activation core.
- Keep the current public endpoints and semantics intact:
  - `GET /api/license/bootstrap`
  - `POST /api/license/activate`
- Implement hosted features as additive layers, not a rewrite of the current verifier.
- Keep the private signing key only inside the `Private Issuer Service`.

## Delivery Sequence

Recommended execution order:

1. `Managed Whatomate Instance`: public activation hardening
2. `Private Issuer Service`: internal signing service and control-plane persistence
3. `Managed Whatomate Instance`: hosted bootstrap and hosted token validation
4. `Provisioning Service`: automated hosted onboarding
5. `Private Issuer Service`: abuse controls, status APIs, and audit privacy hardening
6. `Managed Whatomate Instance` + `Provisioning Service`: outbound heartbeat and suspension flow

This sequence matches the PRD milestone intent while keeping integration risk low.

## Shared Design Decisions

### 1. Control-Plane Data Store

Hosted lifecycle state needs one source of truth outside the managed instance.

Use one shared control-plane Postgres cluster in v1 with explicit service-owned schemas.

Reason:

- simpler initial deployment
- fewer moving parts during Milestones 2 and 3
- still allows ownership boundaries at the schema and migration level

Service ownership:

- `issuer` schema is owned by `Private Issuer Service`
- `provisioner` schema is owned by `Provisioning Service`

No service should run migrations for the other service's schema.

Use the shared cluster for:

- `customers`
- `deployments`
- `bootstrap_sessions`
- `issuer_requests`
- `issued_licenses`
- `activation_events`
- `heartbeat_events`
- `suspension_events`
- `abuse_signals`

This database is owned by the hosted control plane and used by:

- `Private Issuer Service`
- `Provisioning Service`

The managed instance does not need direct write access to this database.

### 2. Nonce Ownership

`bootstrap_nonce` is generated during hosted bootstrap and persisted in the control-plane database as a bootstrap session record.

Required states:

- `CREATED`
- `CONSUMED`
- `EXPIRED`

Implementation rule:

- `Provisioning Service` receives the nonce from the managed instance
- `Provisioning Service` immediately persists the bootstrap session in the control-plane DB
- `Private Issuer Service` atomically consumes the nonce from that DB during issuance

This gives the issuer a real source of truth for `single-use` enforcement.

### 3. Idempotency Ownership

`request_id` idempotency is owned by the `Private Issuer Service`.

Required behavior:

- same `request_id` + same payload => same response
- same `request_id` + different payload => `409`
- response cache TTL => `24h`

### 4. Hosted Deployment Identity

Hosted instances need a stable local deployment identity to validate hosted tokens.

Store locally:

- `deployment_id`
- `hosted_mode`
- `hosted_status`
- `last_heartbeat_at`
- `suspended_at`

This can live in:

- a small local DB table, preferred
- or a dedicated hosted config section if bootstrapping is simpler

Preferred approach: local DB table for runtime mutability and auditability.

## Managed Whatomate Instance Changes

### Scope

Add hosted-only runtime surfaces and public activation hardening without changing the current offline verifier contract.

### Workstream A: Public Activation Hardening

#### Goals

- satisfy PRD Milestone 1
- harden the existing self-hosted path

#### Tasks

1. Add rate limiting to `POST /api/license/activate`.
2. Add structured audit logging for failed activation attempts.
3. Store token hashes, not raw tokens, in audit events.
4. Mask or reduce source-IP detail in exported audit data.
5. Keep `GET /api/license/bootstrap` behavior unchanged.

#### Code Touchpoints

- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/license.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/cmd/whatomate/main.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/middleware/`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/service.go`

#### Acceptance

- self-hosted activation still works exactly as before
- more than `5` activation failures per hour per IP returns `429`
- every failed activation writes an audit event with a token hash only

### Workstream B: Hosted Runtime Identity

#### Goals

- allow the managed instance to know whether it is in hosted mode
- allow local validation of `deployment_id`

#### Tasks

1. Add a local persistence model for hosted deployment state.
2. Add `hosted_mode` config and runtime wiring.
3. Persist `deployment_id` locally during hosted provisioning bootstrap.
4. Extend the license activation path to validate hosted `deployment_id` when hosted mode is enabled.
5. Keep self-hosted tokens exempt from `deployment_id`.
6. Treat revision as issuer-owned metadata only.

#### Proposed Local Table

`hosted_instance_state`

Fields:

- `id`
- `deployment_id`
- `hosted_mode`
- `status`
- `last_heartbeat_at`
- `suspended_at`
- `created_at`
- `updated_at`

#### Code Touchpoints

- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/service.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/token.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/database/`

#### Acceptance

- hosted tokens with the wrong `deployment_id` are rejected
- self-hosted activation remains unchanged

### Workstream C: Hosted Bootstrap Endpoint

#### Goals

- satisfy PRD Milestone 3
- expose a private bootstrap endpoint separate from the public self-hosted bootstrap

#### Tasks

1. Add `POST /internal/license/bootstrap`.
2. Restrict it to hosted mode only.
3. Authenticate with `mTLS` or short-lived internal JWT.
4. Return:
   - `deployment_id`
   - `hwid_full`
   - `hwid_hash`
   - `bootstrap_nonce`
   - `nonce_expires_at`
5. Ensure repeat bootstrap for the same deployment invalidates older unconsumed nonce material.

#### Implementation Note

The endpoint may generate the nonce locally, but the control plane remains the lifecycle authority after the provisioning service persists the bootstrap session.

#### Code Touchpoints

- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/license.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/cmd/whatomate/main.go`
- new hosted auth middleware under `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/middleware/`

#### Acceptance

- endpoint is unreachable from public traffic
- endpoint returns bootstrap payload only when hosted mode is enabled

### Workstream D: Hosted Heartbeat Sender

#### Goals

- satisfy PRD Milestone 5 from the instance side

#### Tasks

1. Add a lightweight outbound heartbeat sender in hosted mode only.
2. Send heartbeat to the control plane every `24h`.
3. Cache the last control-plane status response locally.
4. Respect hosted control-plane `grace` status by:
   - continuing operation
   - showing admin-only hosted billing warning
   - increasing heartbeat frequency
5. Respect hosted control-plane `suspended` status by entering hosted lock mode.
6. Do not enable this path for self-hosted offline mode.

#### Code Touchpoints

- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/service.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/cmd/whatomate/main.go`

#### Acceptance

- hosted instance sends outbound heartbeat only in hosted mode
- suspension state from control plane is enforced locally
- no outbound heartbeat exists in self-hosted mode

### Managed-Instance Test Plan

- unit tests for rate limit behavior on `/api/license/activate`
- unit tests for hosted `deployment_id` validation
- handler tests for `/internal/license/bootstrap`
- integration test for hosted token activation success/failure
- integration test that self-hosted activation remains unchanged

## Private Issuer Service

### Scope

Create the internal trust anchor that signs hosted licenses, owns idempotency, and records issuance/audit events.

### Workstream A: Service Skeleton and Key Isolation

#### Goals

- satisfy PRD Milestone 2

#### Tasks

1. Create a standalone issuer service binary or repository module.
2. Load the Ed25519 private signing key only inside this service.
3. Add `mTLS` and/or signed internal JWT authentication.
4. Restrict ingress to the provisioning service via network policy.

#### Acceptance

- issuer is not public
- no other service or client binary holds the private key

### Workstream B: Control-Plane Persistence

#### Goals

- support bootstrap session lifecycle, issuance history, and idempotency

#### Required Tables

`bootstrap_sessions`

- `bootstrap_nonce`
- `deployment_id`
- `customer_id`
- `hwid_hash`
- `status`
- `expires_at`
- `created_at`
- `consumed_at`

`issuer_requests`

- `request_id`
- `request_hash`
- `response_blob`
- `status_code`
- `created_at`
- `expires_at`

`issued_licenses`

- `license_id`
- `license_family_id`
- `deployment_id`
- `customer_id`
- `revision`
- `kid`
- `jti`
- `license_kind`
- `status`
- `issued_at`
- `expires_at`

`audit_events`

- `event_type`
- `customer_id`
- `deployment_id`
- `token_hash`
- `masked_ip`
- `details_json`
- `created_at`

`abuse_signals`

- `id`
- `signal_type`
- `entity_type`
- `entity_value_hash`
- `count`
- `window_start`
- `window_end`
- `action_taken`
- `created_at`

### Workstream C: `issue-trial`

#### Goals

- satisfy PRD Milestone 3

#### Tasks

1. Implement `POST /internal/licenses/issue-trial`.
2. Validate service authentication.
3. Validate `request_id` idempotency.
4. Load and atomically consume the `bootstrap_nonce`.
5. Ensure nonce state is `CREATED`.
6. Sign a hosted trial token with:
   - current `kid`
   - `deployment_id`
   - existing offline claims
7. Persist issued-license record.
8. Persist issuance audit event.

#### Atomicity Requirement

The following must happen in one transaction:

- bootstrap session lookup
- nonce consume transition
- issued-license persistence
- idempotency response persistence

#### Acceptance

- duplicate retries with same `request_id` return the same response
- consumed nonce cannot be reused

### Workstream D: `issue-paid`

#### Goals

- support hosted renewals and upgrades

#### Tasks

1. Implement `POST /internal/licenses/issue-paid`.
2. Validate `request_id` idempotency.
3. Validate `license_family_id`, `deployment_id`, and optional `expected_previous_revision`.
4. Load `last_revision` for the license family and assign `next_revision = last_revision + 1` atomically.
5. Sign paid token with current `kid`.
6. Persist issuance history and audit event.

#### Acceptance

- stale revisions are rejected
- revision gaps are impossible because the issuer computes the next revision
- repeat renewals with same `request_id` are idempotent

### Workstream E: License Status and Admin Read Paths

#### Goals

- support Milestone 4 admin tooling and provisioning reconciliation

#### Tasks

1. Implement `GET /internal/licenses/{license_id}`.
2. Return canonical license state for internal tooling.
3. Add indexed lookup by:
   - `license_id`
   - `license_family_id`
   - `deployment_id`
4. Reuse this endpoint for dashboard drill-down and provisioning status checks.

#### Acceptance

- provisioning can fetch canonical license state after issuance
- internal admin tooling does not need direct DB reads for license detail

### Workstream F: Abuse Controls and Audit Privacy

#### Goals

- satisfy PRD Milestone 4

#### Tasks

1. Add per-minute and per-hour service rate limits.
2. Add alerting thresholds for abnormal issuance volume.
3. Add velocity counters for:
   - email
   - IP
   - ASN
   - domain
4. Add cooldown evaluation hooks for hosted trials.
5. Mask source IP in audit export paths.
6. Store peppered HWID hashes for audit-only views.
7. Encrypt audit storage at rest.

#### Acceptance

- issuer rejects abusive request patterns
- audit export avoids raw sensitive fields

### Workstream G: Hosted Suspension and Key Rotation

#### Goals

- satisfy PRD Milestone 5

#### Tasks

1. Implement `POST /internal/licenses/suspend`.
2. Add hosted deployment status transitions:
   - `active`
   - `grace`
   - `suspended`
3. Add current/previous key-ring support.
4. Enforce `current` key for signing only.
5. Allow `previous` key in verify-only mode for hosted grace window operations.
6. Document operational limit that offline self-hosted binaries are not forcibly updated by issuer-side rotation.

#### Acceptance

- hosted suspension is visible to heartbeat responses
- key rotation supports hosted rollout without breaking current verifier assumptions

### Issuer Test Plan

- unit tests for `request_id` idempotency
- unit tests for nonce state transitions
- unit tests for hosted trial issuance payloads
- unit tests for revision conflict handling
- unit tests for issuer-owned monotonic revision assignment
- integration tests for retry-after-timeout returning same result
- integration tests for audit event writes and privacy filtering
- integration tests for `GET /internal/licenses/{license_id}`

## Provisioning Service

### Scope

Automate hosted onboarding end-to-end using the managed instance and the issuer service.

### Workstream A: Deployment Creation Flow

#### Goals

- satisfy PRD Milestone 3 onboarding target

#### Tasks

1. Create hosted deployment record with `deployment_id`.
2. Boot infrastructure and wait for instance readiness.
3. Store customer/deployment relationship in the control-plane DB.
4. Generate a fresh `request_id` for every issuance attempt group.

#### Acceptance

- deployment record exists before bootstrap starts
- each onboarding run has traceable `request_id`

### Workstream B: Hosted Bootstrap Orchestration

#### Goals

- bridge the managed instance bootstrap endpoint to the control-plane nonce registry

#### Tasks

1. Call `POST /internal/license/bootstrap` on the managed instance.
2. Persist the returned bootstrap session in control-plane DB:
   - `bootstrap_nonce`
   - `deployment_id`
   - `hwid_hash`
   - `expires_at`
3. Refuse to continue if bootstrap TTL is too close to expiry.
4. Re-bootstrap instead of retrying with a stale nonce.
5. If bootstrap returns `bootstrap_already_completed`, branch by deployment state:
   - if deployment is already active, reuse existing activation state
   - if no active license exists, treat this as a contract violation and alert immediately

#### Acceptance

- bootstrap response is persisted before issuer call
- expired bootstrap sessions are never sent for issuance

### Workstream C: Trial Issuance Orchestration

#### Goals

- automate hosted `14d` trial activation

#### Tasks

1. Call `POST /internal/licenses/issue-trial`.
2. Pass:
   - `request_id`
   - `customer_id`
   - `deployment_id`
   - `bootstrap_nonce`
   - `hwid_hash`
   - `tier`
   - quota limits
3. Handle timeout by retrying with the same `request_id`.
4. On success, store issuance metadata in deployment state.
5. If issuer succeeded but response delivery is uncertain, resolve by replaying the same `request_id` or reading canonical state from `GET /internal/licenses/{license_id}` once known.

#### Acceptance

- retry after timeout does not create a second license
- successful issue returns token and revision metadata

### Workstream D: Managed Instance Activation

#### Goals

- finish onboarding without exposing manual activation to the hosted customer

#### Tasks

1. Call existing `POST /api/license/activate` on the managed instance.
2. Treat activation as complete only after managed instance returns active state.
3. Persist activation result in control-plane deployment state.
4. Mark onboarding complete.
5. On activation failure:
   - retry activation with the same already-issued token using bounded exponential backoff
   - after the retry budget is exhausted, mark the deployment `activation_pending_manual`
   - alert operations instead of issuing a fresh token automatically

#### Acceptance

- hosted onboarding finishes without operator intervention
- customer receives ready instance, not a manual activation screen

### Workstream E: Abuse Controls

#### Goals

- satisfy PRD Milestone 4 from the orchestration side

#### Tasks

1. Evaluate trial eligibility before provisioning:
   - email history
   - IP history
   - ASN history
   - domain history
2. Enforce cooldown rules before infrastructure creation.
3. Flag suspicious signups for manual review.
4. Avoid booting expensive infrastructure for obviously blocked trials.

#### Acceptance

- blocked trial signups fail before resource provisioning
- suspicious signups are logged for operations review

### Workstream F: Heartbeat and Suspension Loop

#### Goals

- satisfy PRD Milestone 5 from the control-plane side

#### Tasks

1. Receive heartbeat from hosted instances.
2. Update deployment liveness timestamps.
3. Enter control-plane grace after `48h` missing heartbeat.
4. Trigger suspension workflow after grace timeout.
5. Allow resume after payment or operator action.
6. Expose clear runtime semantics:
   - `grace`: instance may continue operating until `grace_until`
   - `suspended`: instance must lock normal app usage

#### Acceptance

- deployment status transitions are visible in control-plane records
- suspended deployments are reflected back to the instance on heartbeat

### Provisioning Test Plan

- integration test for full hosted onboarding happy path
- integration test for bootstrap timeout + same `request_id` retry
- integration test for expired nonce requiring re-bootstrap
- integration test for activation failure retry with same token
- integration test for abuse gate blocking before provisioning
- integration test for activation failure rollback or alerting

## Milestone Mapping

### Milestone 1: Operational Hardening

Primary owner:

- `Managed Whatomate Instance`

Deliverables:

- activation rate limiting
- activation audit events
- SOP documentation outside code

### Milestone 2: Private Issuer Service

Primary owner:

- `Private Issuer Service`

Deliverables:

- isolated signing service
- control-plane DB
- idempotency and issuance persistence
- network and auth hardening

### Milestone 3: Hosted Provisioning Path

Primary owners:

- `Managed Whatomate Instance`
- `Provisioning Service`
- `Private Issuer Service`

Deliverables:

- `/internal/license/bootstrap`
- bootstrap session persistence
- hosted `issue-trial`
- automatic activation

### Milestone 4: Abuse Controls & Observability

Primary owners:

- `Private Issuer Service`
- `Provisioning Service`

Deliverables:

- velocity checks
- cooldown enforcement
- `GET /internal/licenses/{license_id}`
- privacy-hardened audit output
- operations visibility

### Milestone 5: Hosted Control Plane

Primary owners:

- `Provisioning Service`
- `Managed Whatomate Instance`
- `Private Issuer Service`

Deliverables:

- outbound heartbeat loop
- suspension flow
- hosted key-rotation operations

### Milestone 6: Hardening & Future-Proofing

Primary owners:

- all three services

Deliverables:

- end-to-end failure drills
- key-rotation runbooks
- documented self-hosted limitations
- future `install_id` exploration without promising clone prevention

## Risks and Open Decisions

### 1. Where to Persist Hosted Deployment Identity Locally

Recommendation:

- DB table on the managed instance

Reason:

- cleaner runtime updates and auditability than config-file mutation

### 2. Hosted Auth Mechanism

Recommendation:

- `mTLS` first
- signed internal JWT only as fallback

Reason:

- simpler trust boundary and easier network-policy enforcement

### 3. Issuer and Provisioning Repository Boundaries

Recommendation:

- separate deployables at minimum
- separate private repositories if you want stronger operational isolation

## Definition of Done

The hosted licensing plan is complete when:

- self-hosted manual activation remains fully functional
- hosted onboarding can provision, issue, and activate a `14d` trial automatically
- hosted retries do not duplicate issuance
- private signing key exists only in the issuer service
- hosted deployments can be suspended from the control plane
- audit logs provide full issuance visibility without leaking raw sensitive data
