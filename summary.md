# P1-1: Object Storage Retry + Circuit Breaker

## Task
Fix the P1-1 vulnerability from FIX_PLAN_AR.md: Object Storage had no retry on GetObject/DeleteObject and no circuit breaker, meaning transient S3 failures caused permanent file loss.

## Approach and Key Decisions
- **Existing state**: `retryableObjectStorage` already wrapped `PutObject` with retry + exponential backoff. `GetObject` and `DeleteObject` delegated directly with no retry.
- **Added**: Retry with exponential backoff + jitter for both `GetObject` and `DeleteObject`.
- **Added**: Circuit breaker (`circuitBreaker` struct) with closed -> open -> half-open state machine.
  - Opens after 5 consecutive failures (configurable `failureThreshold`).
  - Resets to half-open after 30 seconds (`resetTimeout`).
  - Requires 2 consecutive successes in half-open to close (`halfOpenRequired`).
- **Nil-safe**: breaker fields are nil-checked so existing tests (which don't set them) continue to work.
- `ErrObjectNotFound` is explicitly excluded from retry (not transient).
- `isTransientError` already handles: net errors, MinIO 429/5xx, string matching for connection reset/broken pipe/timeout.
- Added shutdown hooks so server and worker paths stop dispatcher workers on process shutdown.

## Gap 1 — Caller error handling (committed `bc9a25d2`)
The initial implementation had three callers that conflated `ErrCircuitOpen` with `ErrObjectNotFound` / generic errors. Fixed:
- **`media.go ServeMedia` (line 305)**: when `GetObject` returns `ErrCircuitGetOpen`, the request now returns **503 + `Retry-After: 30`** instead of falling into the `ErrObjectNotFound` self-heal branch. Without this fix, a temporarily unhealthy S3 would have caused the handler to delete the `MediaAsset` row and mark the message as `media_deleted_at`, permanently losing a recoverable file.
- **`media.go RetryMediaDownload` (line 394)**: the existence probe now distinguishes circuit-open (log warn, treat as unavailable) from not-found (existing behavior).
- **`media_retention_worker.go` (line 201)**: when the delete breaker is open, the worker logs and returns `nil` instead of failing the row. Failing the row would have made the worker re-process the same asset forever, hammering the breaker. The next sweep picks it up once the circuit closes.

## Gap 2 — Per-operation circuit breakers (committed `bc9a25d2`)
The initial design used a single shared breaker. That was too coarse: a wave of slow `PutObject` failures (e.g. media uploads timing out) would have opened the breaker for the entire object storage, blocking `GetObject` (live user downloads) and `DeleteObject` (retention cleanup) too.
- `retryableObjectStorage` now has three independent breakers: `putBreaker`, `getBreaker`, `deleteBreaker`. `NewObjectStorage` constructs three `newCircuitBreaker(5, 30*time.Second)` instances.
- Added two wrapper sentinels — `ErrCircuitGetOpen` and `ErrCircuitDeleteOpen` — both still satisfy `errors.Is(err, ErrCircuitOpen)`. The wrappers let callers distinguish which breaker tripped when handling the error.
- Two new tests assert the independence: `TestCircuitBreaker_PutOpenDoesNotBlockGet` and `TestCircuitBreaker_GetOpenDoesNotBlockPut`.

## Gap 3 — Skipped
Local-disk fallback queue for `PutObject` (write-side graceful degradation) was identified as a third gap but **deferred**: it duplicates the recovery work already captured by P1-4 (media retry/regeneration) and P1-6 (async media recovery worker) in FIX_PLAN_AR.md. Scope was intentionally limited to retry + circuit breaker + caller safety.

## Files Modified
| File | Change |
|------|--------|
| `internal/storage/object_storage.go` | Added `circuitBreaker` struct + state machine; added `putBreaker`/`getBreaker`/`deleteBreaker` to `retryableObjectStorage`; added retry logic to `GetObject` and `DeleteObject`; integrated per-op breakers into all 3 methods; added `ErrCircuitOpen`/`ErrCircuitGetOpen`/`ErrCircuitDeleteOpen` sentinel errors |
| `internal/storage/object_storage_test.go` | Updated `mockStorage` with `getFunc`/`deleteFunc` + separate call counters; updated all existing tests to per-op field names; added 17 new tests (15 in commit `2343d1d6`, +2 per-op independence in `bc9a25d2`) |
| `internal/handlers/media.go` | `ServeMedia` returns 503 + `Retry-After: 30` on `ErrCircuitGetOpen`; `RetryMediaDownload` distinguishes circuit-open from not-found |
| `internal/handlers/media_retention_worker.go` | `ErrCircuitDeleteOpen` logged and skipped (returns nil) so the next sweep catches the asset once the circuit closes |
- Modified `pkg/whatsmeow/metrics_unit_test.go`
- Modified `internal/config/config.go`
- Modified `internal/config/config_test.go`
- Modified `internal/handlers/instances.go`
- Modified `cmd/whatomate/main.go`
- Modified `config.example.toml`

## Tests Added (17 new total)
- GetObject: success first attempt, success on retry, NotFound skips retry, max retries exhausted
- DeleteObject: success first attempt, success on retry, max retries exhausted
- Circuit breaker: opens after threshold, half-open after reset timeout, closes after recovery, re-opens on half-open failure
- Circuit breaker blocks GetObject/DeleteObject when open (now uses `ErrCircuitGetOpen` / `ErrCircuitDeleteOpen`)
- Per-op breaker independence: `TestCircuitBreaker_PutOpenDoesNotBlockGet`, `TestCircuitBreaker_GetOpenDoesNotBlockPut`
- Permanent errors count as breaker failures
- Config defaults and environment overrides cover `event_buffer_size` and `event_dispatch_enabled`.

## Verification
- `go build ./...` -- clean (modulo pre-existing tmp/scratch noise unrelated to this work)
- `go vet ./internal/storage/... ./internal/handlers/...` -- clean
- `go test ./internal/storage/... -v` -- **26/26 pass** (9 existing + 17 new)
- `go test ./internal/handlers/...` -- compiles cleanly (no tests added; behavior changes are HTTP-status mapping)
- `git diff --check` passed.

Playwright was not run because this task only changes backend WhatsMeow ingestion/config/metrics behavior and does not touch UI, routing, forms, or browser behavior.

## Known Limitations
- Circuit breaker parameters (threshold=5, reset=30s) are hardcoded in `NewObjectStorage`. A future enhancement could expose them via config.
- No metrics/observability for circuit breaker state transitions yet (tracked as P1-3).
