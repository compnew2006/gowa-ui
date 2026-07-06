# Hosted Licensing API Contract

## Purpose

This document defines the hosted control-plane API contract for Whatomate licensing without changing the existing offline self-hosted activation core.

It applies to three services:

- `Provisioning Service`
- `Private Issuer Service`
- `Managed Whatomate Instance`

It explicitly does **not** replace the current offline activation flow:

- `GET /api/license/bootstrap`
- `POST /api/license/activate`

Those existing endpoints remain the canonical self-hosted activation surface.

## Architecture Boundaries

### Existing Core That Must Stay Stable

The current offline licensing core remains unchanged:

- the managed instance verifies signed tokens locally
- the managed instance binds the token to local host identity
- the managed instance does not need online validation for self-hosted offline mode

Current code boundaries:

- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/license.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/service.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/license/token.go`
- `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/cmd/whatomate/main.go`

### Hosted Extension Model

Hosted licensing is an additive layer on top of the existing core:

- internal bootstrap for a newly provisioned hosted instance
- internal issuer API for short-lived hosted trial issuance
- outbound heartbeat and suspension logic in the hosted control plane

Hosted-only controls must not break offline self-hosted activation.

## Service Roles

### Provisioning Service

Responsibilities:

- create a hosted Whatomate deployment
- obtain bootstrap identity from the managed instance
- request a signed hosted trial from the private issuer
- activate the managed instance using the returned token
- track deployment lifecycle, heartbeat state, and suspension state

### Private Issuer Service

Responsibilities:

- hold the Ed25519 private signing key
- validate internal issuance requests
- enforce nonce and request idempotency
- issue signed hosted trial and hosted paid tokens
- record issuance audit events

### Managed Whatomate Instance

Responsibilities:

- expose self-hosted public activation endpoints unchanged
- expose a separate hosted-only internal bootstrap endpoint
- verify and install signed licenses locally
- send outbound hosted heartbeat signals when running in hosted mode

## Trust Model

### Self-Hosted Offline

- customer uses public `GET /api/license/bootstrap`
- customer sends HWID to vendor
- vendor issues token out of band
- customer activates using public `POST /api/license/activate`

No heartbeat, revocation, or hosted bootstrap dependency is required.

### Hosted

- `Provisioning Service` talks to the managed instance over internal network only
- `Provisioning Service` talks to the `Private Issuer Service` over internal authenticated channel only
- hosted-specific bootstrap is unavailable to the public internet
- hosted heartbeat is outbound from the instance to the control plane

## Authentication and Transport

### Provisioning Service -> Managed Instance

Transport:

- HTTPS
- private network only

Authentication:

- preferred: `mTLS`
- acceptable fallback: `Authorization: Bearer <internal_jwt>`

Required request headers for internal endpoints:

- `X-Request-Id`
- `X-Provisioning-Instance-Id`
- `Authorization`

### Provisioning Service -> Private Issuer Service

Transport:

- HTTPS
- private network only

Authentication:

- preferred: `mTLS`
- acceptable fallback: signed internal JWT with short TTL

Required request headers:

- `X-Request-Id`
- `Authorization`

### Public Client -> Managed Instance

Transport:

- HTTPS

Public endpoints:

- `GET /api/license/bootstrap`
- `POST /api/license/activate`

`POST /api/license/activate` must be rate limited and audited.

## Canonical Identifiers

### `customer_id`

Stable vendor-side customer identifier.

### `deployment_id`

Stable hosted deployment identifier assigned by the provisioning service.

### `bootstrap_nonce`

Single-use bootstrap token tied to one `deployment_id`.

### `request_id`

Idempotency key for internal hosted issuance requests.

### `license_id`

Unique license instance identifier carried inside the signed token.

### `license_family_id`

Stable entitlement family identifier used across renewal, reissue, and revision changes.

### `jti`

JWT unique token identifier.

## Managed Whatomate Instance API

### Existing Public Endpoint: `GET /api/license/bootstrap`

Purpose:

- self-hosted activation bootstrap

Authentication:

- none

Response `200`:

```json
{
  "data": {
    "state": {
      "enabled": true,
      "status": "unlicensed",
      "hwid_full": "4333a6135a7f4dc52a4e908a7490c84a8c269f6504d7e31cf0abac7750d4af53",
      "hwid_short": "4333a6135a7f",
      "license_kind": "",
      "tier": "",
      "days_until_expiry": 0,
      "expiring_soon": false,
      "quota_overages": {}
    },
    "usage": {
      "organizations": 0,
      "users_per_org": {},
      "whatsapp_endpoints_per_org": {}
    }
  }
}
```

Notes:

- this remains public for self-hosted
- no IP whitelist on this endpoint

### Existing Public Endpoint: `POST /api/license/activate`

Purpose:

- install a signed offline license on the current host

Authentication:

- none

Rate limit:

- `5` attempts per hour per source IP

Audit rules:

- every failed activation attempt must be logged
- store `sha256(token)` or equivalent hash, not the raw token

Request:

```json
{
  "token": "<signed_security_key>"
}
```

Success `200`:

```json
{
  "data": {
    "state": {
      "status": "active",
      "license_id": "lic_01J...",
      "license_kind": "trial",
      "trial_days": 14,
      "tier": "starter",
      "days_until_expiry": 14,
      "expiring_soon": true
    },
    "usage": {
      "organizations": 0,
      "users_per_org": {},
      "whatsapp_endpoints_per_org": {}
    }
  }
}
```

Failure `423` example:

```json
{
  "error": "A valid license is required to use Whatomate",
  "details": {
    "code": "license_locked",
    "activate_url": "/activate"
  }
}
```

Hosted note:

- `Provisioning Service` may call this existing endpoint after successful hosted issuance
- for hosted tokens, the managed instance must additionally validate `deployment_id`

### New Hosted-Only Endpoint: `POST /internal/license/bootstrap`

Purpose:

- internal bootstrap handshake for a newly provisioned hosted deployment

Authentication:

- `mTLS` or short-lived internal JWT

Request:

```json
{
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1"
}
```

Success `200`:

```json
{
  "data": {
    "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
    "hwid_full": "4333a6135a7f4dc52a4e908a7490c84a8c269f6504d7e31cf0abac7750d4af53",
    "hwid_hash": "6c1208d343c6c8459c8fc8a22126f9ca6f25f4de0d0f65cbfd88c7fd1d7f8c68",
    "bootstrap_nonce": "btn_01J9YQAKMN2Z6M7WN8ER3QJ7D2",
    "nonce_expires_at": "2026-04-08T12:05:00Z"
  }
}
```

Rules:

- available only in hosted mode
- `bootstrap_nonce` is `single-use`
- nonce TTL is `5 minutes`
- nonce is bound to `deployment_id`
- a new bootstrap call invalidates any older unconsumed nonce for the same deployment
- return `bootstrap_already_completed` only when the deployment already has an active hosted license installed
- if the previous nonce expired without a successful activation, the endpoint must return a fresh bootstrap payload instead of `409`

Failure `409`:

```json
{
  "error": "Hosted bootstrap already completed for this deployment",
  "details": {
    "code": "bootstrap_already_completed",
    "hint": "deployment already has an active hosted license"
  }
}
```

## Private Issuer Service API

### Endpoint: `POST /internal/licenses/issue-trial`

Purpose:

- issue a hosted trial token for a just-provisioned deployment

Authentication:

- `mTLS` or short-lived internal JWT

Headers:

- `X-Request-Id: <request_id>`

Request:

```json
{
  "request_id": "req_01J9YQBGZW3BGQ6ZK0X2M5PS2N",
  "customer_id": "cus_01J9YQ8BY52KCMF10S7M5PHVYJ",
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
  "bootstrap_nonce": "btn_01J9YQAKMN2Z6M7WN8ER3QJ7D2",
  "hwid_hash": "6c1208d343c6c8459c8fc8a22126f9ca6f25f4de0d0f65cbfd88c7fd1d7f8c68",
  "tier": "starter",
  "trial_days": 14,
  "limits": {
    "max_organizations": 1,
    "max_users_per_org": 5,
    "max_whatsapp_endpoints_per_org": 5,
    "max_workers": 2
  }
}
```

Success `200`:

```json
{
  "data": {
    "request_id": "req_01J9YQBGZW3BGQ6ZK0X2M5PS2N",
    "license_id": "lic_01J9YQCC5Y4S4N8XPZQ8A6B6C2",
    "license_family_id": "fam_01J9YQ8C9Q8CP5MGC4W54Z2V9A",
    "revision": 1,
    "token": "<signed_security_key>",
    "issued_at": "2026-04-08T12:00:00Z",
    "expires_at": "2026-04-22T12:00:00Z"
  }
}
```

Rules:

- `request_id` is an idempotency key
- repeated requests with the same `request_id` must return the same stored result for `24 hours`
- `bootstrap_nonce` must be in state `CREATED`
- on successful issuance, `bootstrap_nonce` moves to `CONSUMED`
- issuer must reject `EXPIRED` or `CONSUMED` nonces

Failure `409`:

```json
{
  "error": "Bootstrap nonce already consumed",
  "details": {
    "code": "bootstrap_nonce_consumed"
  }
}
```

### Endpoint: `POST /internal/licenses/issue-paid`

Purpose:

- issue a hosted paid license for renewals, upgrades, or post-trial activation

Authentication:

- `mTLS` or short-lived internal JWT

Headers:

- `X-Request-Id: <request_id>`

Request:

```json
{
  "request_id": "req_01J9YQDZ80V7B0N2EB1HZT7WNQ",
  "customer_id": "cus_01J9YQ8BY52KCMF10S7M5PHVYJ",
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
  "hwid_hash": "6c1208d343c6c8459c8fc8a22126f9ca6f25f4de0d0f65cbfd88c7fd1d7f8c68",
  "license_family_id": "fam_01J9YQ8C9Q8CP5MGC4W54Z2V9A",
  "expected_previous_revision": 1,
  "tier": "starter",
  "duration_days": 365,
  "limits": {
    "max_organizations": 1,
    "max_users_per_org": 5,
    "max_whatsapp_endpoints_per_org": 5,
    "max_workers": 2
  }
}
```

Success `200`:

```json
{
  "data": {
    "request_id": "req_01J9YQDZ80V7B0N2EB1HZT7WNQ",
    "license_id": "lic_01J9YQFDHED9D4Q3Y7HBPFV9XF",
    "license_family_id": "fam_01J9YQ8C9Q8CP5MGC4W54Z2V9A",
    "revision": 2,
    "token": "<signed_security_key>",
    "issued_at": "2026-04-22T12:00:00Z",
    "expires_at": "2027-04-22T12:00:00Z"
  }
}
```

Rules:

- paid issuance for hosted renewals does not need a bootstrap nonce when bound to an existing `deployment_id` and `license_family_id`
- revision is issuer-owned, not caller-owned

Revision rules:

- revision is monotonically increasing per `license_family_id`
- issuer stores `last_revision` per license family
- next issued revision must equal `last_revision + 1`
- if `expected_previous_revision` is present and does not match the stored `last_revision`, reject with `409`
- issuer must never skip revision numbers

### Endpoint: `POST /internal/licenses/suspend`

Purpose:

- mark a hosted deployment as suspended in the control plane

Authentication:

- `mTLS` or short-lived internal JWT

Request:

```json
{
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
  "reason": "billing_past_due"
}
```

Success `200`:

```json
{
  "data": {
    "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
    "status": "suspended"
  }
}
```

Notes:

- this is hosted control-plane state, not offline self-hosted token revocation

### Endpoint: `GET /internal/licenses/{license_id}`

Purpose:

- fetch the canonical hosted-license record for provisioning reconciliation and internal admin tooling

Authentication:

- `mTLS` or short-lived internal JWT

Success `200`:

```json
{
  "data": {
    "license_id": "lic_01J9YQFDHED9D4Q3Y7HBPFV9XF",
    "license_family_id": "fam_01J9YQ8C9Q8CP5MGC4W54Z2V9A",
    "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
    "customer_id": "cus_01J9YQ8BY52KCMF10S7M5PHVYJ",
    "revision": 2,
    "license_kind": "paid",
    "tier": "starter",
    "status": "active",
    "issued_at": "2026-04-22T12:00:00Z",
    "expires_at": "2027-04-22T12:00:00Z"
  }
}
```

## Provisioning Service Control-Plane API

### Endpoint: `POST /internal/deployments/heartbeat`

Purpose:

- allow hosted instances to push liveness and local license state to the control plane

Caller:

- `Managed Whatomate Instance`

Authentication:

- `mTLS` or short-lived internal JWT

Request:

```json
{
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1",
  "license_id": "lic_01J9YQCC5Y4S4N8XPZQ8A6B6C2",
  "license_jti": "jti_01J9YQCC6WQ8Q4R7DTRQJKJEDP",
  "hwid_hash": "6c1208d343c6c8459c8fc8a22126f9ca6f25f4de0d0f65cbfd88c7fd1d7f8c68",
  "status": "active",
  "sent_at": "2026-04-08T12:00:00Z"
}
```

Success `200`:

```json
{
  "data": {
    "status": "active",
    "grace_until": null,
    "next_heartbeat_after_seconds": 86400
  }
}
```

Grace `200`:

```json
{
  "data": {
    "status": "grace",
    "grace_until": "2026-04-10T12:00:00Z",
    "next_heartbeat_after_seconds": 3600
  }
}
```

Suspended `200`:

```json
{
  "data": {
    "status": "suspended",
    "grace_until": null,
    "next_heartbeat_after_seconds": 3600
  }
}
```

Rules:

- heartbeat is hosted-only
- the direction is outbound from instance to control plane
- no heartbeat dependency is introduced for self-hosted offline mode
- outbound is preferred because it is more robust through NAT and firewall boundaries

Instance behavior:

- `status=active`: continue normal operation
- `status=grace`: continue operating until `grace_until`, show an admin-only hosted billing warning, and retry heartbeat more frequently
- `status=suspended`: enter the existing lock behavior for normal app usage while preserving already-allowed health and webhook surfaces

## Hosted Token Requirements

Hosted tokens must carry all current offline claims plus the following hosted-only claim:

```json
{
  "deployment_id": "dep_01J9YQ8K0AW7R6N9B8M4S2F3D1"
}
```

Managed hosted instances must reject activation when:

- `deployment_id` is missing for a hosted-issued token
- `deployment_id` does not match the local hosted deployment identity

Self-hosted tokens must not require `deployment_id`.

## Nonce Lifecycle

State machine:

- `CREATED`
- `CONSUMED`
- `EXPIRED`

Rules:

- nonce starts in `CREATED`
- nonce becomes `CONSUMED` only after a successful issuance commit
- nonce becomes `EXPIRED` automatically at `created_at + 5 minutes`
- consumed or expired nonces are never reusable
- nonce is bound to one `deployment_id`

## Idempotency Contract

Idempotency applies to hosted issuer operations only.

Rules:

- `request_id` is required on all internal issuance endpoints
- the issuer stores the full successful result by `request_id` for `24 hours`
- same `request_id` and same payload returns the original response
- same `request_id` with different payload returns `409`

Failure `409`:

```json
{
  "error": "Request id already used with different payload",
  "details": {
    "code": "request_id_payload_conflict"
  }
}
```

## Audit Requirements

### Shared

Audit on:

- hosted bootstrap creation
- hosted issuance success
- hosted issuance failure
- public activation failure
- public activation success
- hosted suspension and hosted resume events

### Privacy Rules

Store:

- `customer_id`, not raw email
- masked source IP where possible
- `sha256(token)` or equivalent hash, not raw token
- `hwid_hash` with an additional application pepper where audit storage is separate from license verification storage

Audit log storage must be encrypted at rest.

## Error Catalog

Common codes:

- `license_disabled`
- `missing_token`
- `invalid_signature`
- `invalid_claims`
- `hwid_mismatch`
- `deployment_id_mismatch`
- `expired`
- `not_yet_valid`
- `stale_revision`
- `bootstrap_already_completed`
- `bootstrap_nonce_expired`
- `bootstrap_nonce_consumed`
- `request_id_payload_conflict`
- `revision_conflict`
- `rate_limited`
- `suspended`

## Rate Limits

### Public

- `POST /api/license/activate`: `5` attempts per hour per IP

### Internal

- issuer endpoints should also enforce service-level caps:
  - example: `10` per minute
  - example: `100` per hour

Those internal limits are operational safeguards, not protocol semantics.

## Heartbeat Policy

Hosted-only policy:

- heartbeat interval: `24 hours`
- if no heartbeat for `48 hours`, deployment enters control-plane grace state
- if no heartbeat beyond the grace window, control plane may suspend the hosted deployment

This policy must not be enforced for self-hosted offline mode.

## Key Rotation Policy

Rules:

- all tokens carry explicit `kid`
- verifier trusts `current` and `previous` public keys only
- issuer signs with `current` only
- previous key remains `verify-only` during a fixed operational grace period

Important limit:

- for offline self-hosted, key rotation policy alone cannot force old binaries to distrust an old embedded key ring
- removal of trust in an old key requires an updated binary or updated trusted key material on the instance

## Non-Goals

This contract does not attempt to guarantee:

- immediate remote revocation for offline self-hosted customers
- perfect prevention of full-image VM cloning in offline self-hosted mode
- mandatory payment-method enforcement for all hosted trials at day one

## Implementation Guidance

Build hosted support as new layers and services, not by rewriting the existing offline verification core.

Recommended implementation order:

1. add `rate limit + audit` to public activation
2. add hosted internal bootstrap endpoint
3. add issuer nonce + idempotency handling
4. add hosted `deployment_id` claim enforcement
5. add heartbeat and suspension state
6. add hosted abuse controls and audit privacy hardening
