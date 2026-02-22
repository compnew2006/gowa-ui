# Whatomate Security Hardening Report (Tasks 5-9)

Date: 2026-02-19
Scope:
- 5) Secrets-at-rest encryption optional no-op
- 6) Inbound webhook trust and processing vulnerabilities
- 7) Queue/worker reliability, idempotency, and endpoint consistency
- 8) Custom actions JS runtime safety and redirect token distribution
- 9) Worker/API external endpoint routing divergence

## Critical Risks Identified

### 1) Secrets-at-rest could silently become plaintext
- Risk: If `app.encryption_key` was empty, account secrets were effectively stored/handled without enforced encryption.
- Impact: Database compromise exposed WhatsApp access tokens and app secrets in recoverable form.

### 2) Inbound webhook trust model was fail-open under weak conditions
- Risk: Signature verification was conditional and could be bypassed in some paths (missing header, missing/indirect account mapping), with unbounded asynchronous processing fan-out.
- Impact: Forged payload acceptance risk and request-amplified processing pressure.

### 3) Queue processing could stall failed messages indefinitely
- Risk: Failed stream entries stayed pending without strong periodic reclaim + bounded retry lifecycle.
- Impact: Message backlog growth, unrecoverable stalls, and delayed campaign completion.

### 4) Duplicate send risk under retries/concurrency
- Risk: No per-recipient in-flight dedupe lock in worker path.
- Impact: Potential duplicate outbound messages for same recipient under retries or concurrent workers.

### 5) Worker/API endpoint routing drift
- Risk: Worker and some API flows code paths instantiated WhatsApp clients with defaults instead of configured `base_url`.
- Impact: Environment drift (prod/proxy/staging) and inconsistent behavior across components.

### 6) Custom action execution lacked hard runtime boundaries
- Risk: Server-side JavaScript execution had no timeout guard; redirect token model used process-local memory.
- Impact: Infinite-loop CPU abuse risk and non-distributed redirect behavior across multi-instance deployments.

## Architectural Flaws Corrected

### A) Encryption enforcement and fail-closed secret handling
Changes:
- Added strict encryption config validator (`internal/config/encryption_validation.go`).
- Added startup validation calls in server and worker boot (`cmd/whatomate/main.go`).
- Changed crypto behavior:
  - `Encrypt` now fails with `ErrMissingEncryptionKey` for non-empty plaintext when key is missing.
  - `Decrypt` now fails for encrypted (`enc:`) values when key is missing.
- Propagated decryption errors in account/cache paths (`internal/handlers/cache.go`, `internal/handlers/accounts.go`).
- Added explicit API feedback for account create/update when encryption key is absent.

Security effect:
- Prevents new plaintext-at-rest writes caused by empty key configuration.
- Converts silent decryption failures into explicit operational errors.

### B) Webhook authenticity and payload pressure hardening
Changes:
- Added webhook security module (`internal/handlers/webhook_security.go`).
- Webhook handler now:
  - Validates `object == whatsapp_business_account`.
  - Requires `X-Hub-Signature-256` and validates against resolved app secret(s).
  - Resolves secrets via phone ID and business ID fallback.
  - Rejects oversized event batches (`maxWebhookEventsPerRequest`).
  - Removes unbounded `go` fan-out for per-event processing in request path.

Security effect:
- Tightened trust boundary and reduced fail-open behavior.
- Reduced attacker leverage from payload-driven goroutine amplification.

### C) Queue resilience + dead-letter lifecycle
Changes:
- Extended Redis consumer (`internal/queue/redis.go`):
  - periodic pending reclaim loop,
  - bounded retry attempts (`MaxDeliveryAttempts`),
  - dead-letter stream (`whatomate:campaigns:dlq`),
  - permanent-message classification for malformed/unknown jobs.
- Added coverage test for permanent-failure DLQ behavior (`internal/queue/queue_test.go`).

Security/reliability effect:
- Failed jobs no longer remain pending forever.
- Poison messages are isolated rather than blocking queue health.

### D) Worker idempotency and duplicate-send reduction
Changes:
- Added per-recipient Redis lock module (`internal/worker/idempotency.go`).
- Worker now acquires/release recipient in-flight lock and skips non-pending recipients (`internal/worker/worker.go`).
- Added regression test for already-processed recipient skip (`internal/worker/worker_test.go`).

Effect:
- Reduces duplicate outbound sends during retries and parallel worker contention.

### E) Endpoint routing consistency between worker and API
Changes:
- Worker now uses configured `whatsapp.base_url` (`internal/worker/worker.go`).
- Added shared configured client helper (`internal/handlers/whatsapp_client.go`).
- Flows handler switched to configured client instead of defaults (`internal/handlers/flows.go`).

Effect:
- Worker/API align to same upstream routing policy.

### F) Custom action runtime safety + distributed token store
Changes:
- Added runtime/token module (`internal/handlers/custom_action_runtime.go`):
  - Redis-backed one-time redirect tokens (with in-memory fallback only when Redis absent),
  - JS execution timeout enforcement (default + capped max).
- Updated action execution/redirect paths (`internal/handlers/custom_actions.go`).
- Added JS timeout test (`internal/handlers/custom_actions_test.go`).

Effect:
- Prevents unbounded JS execution time.
- Makes redirect token handling multi-instance safe when Redis is present.

## Validation Results

Build and tests run:
- `go test ./...` ✅
- `npm --prefix frontend run typecheck` ✅
- `npm --prefix frontend run lint` ✅ (warnings only, no errors)
- `npm --prefix frontend run build` ✅
- `CGO_ENABLED=0 go build ./cmd/whatomate` ✅

Runtime smoke check:
- `./whatomate server -config config.toml -migrate` no longer fails on encryption config in development mode.

## Residual Risks / Follow-ups

1. Legacy plaintext secrets in existing databases should be rotated/re-encrypted with a configured key.
2. Consider adding explicit webhook replay protection (timestamp + nonce window) in addition to signature checks.
3. Add DLQ monitoring/alerting and operator tooling (requeue/inspect).
4. Consider stronger idempotency keys persisted in DB for cross-queue dedupe analytics.
5. Redis fallback for redirect tokens is intentionally in-memory; production deployments should treat Redis as required.
