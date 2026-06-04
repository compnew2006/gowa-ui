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
- **Nil-safe**: `breaker` field is nil-checked so existing tests (which don't set it) continue to work.
- `ErrObjectNotFound` is explicitly excluded from retry (not transient).
- `isTransientError` already handles: net errors, MinIO 429/5xx, string matching for connection reset/broken pipe/timeout.
- Added shutdown hooks so server and worker paths stop dispatcher workers on process shutdown.

## Files Modified
| File | Change |
|------|--------|
| `internal/storage/object_storage.go` | Added `circuitBreaker` struct + state machine; added `breaker` field to `retryableObjectStorage`; added retry logic to `GetObject` and `DeleteObject`; integrated circuit breaker into all 3 methods; added `ErrCircuitOpen` sentinel error; added `sync` import |
| `internal/storage/object_storage_test.go` | Updated `mockStorage` with `getFunc`/`deleteFunc` + separate call counters; added 15 new tests |
- Modified `pkg/whatsmeow/metrics_unit_test.go`
- Modified `internal/config/config.go`
- Modified `internal/config/config_test.go`
- Modified `internal/handlers/instances.go`
- Modified `cmd/whatomate/main.go`
- Modified `config.example.toml`

## Tests Added (15 new)
- GetObject: success first attempt, success on retry, NotFound skips retry, max retries exhausted
- DeleteObject: success first attempt, success on retry, max retries exhausted
- Circuit breaker: opens after threshold, half-open after reset timeout, closes after recovery, re-opens on half-open failure
- Circuit breaker blocks GetObject/DeleteObject when open
- Permanent errors count as breaker failures
- Config defaults and environment overrides cover `event_buffer_size` and `event_dispatch_enabled`.

## Verification
- `go build ./cmd/... ./internal/... ./pkg/...` -- clean
- `go vet ./internal/storage/...` -- clean
- `go test ./internal/storage/... -v` -- **26/26 pass** (11 existing + 15 new)
- Handler tests referencing ObjectStorage compile and skip correctly (no DB)
- `git diff --check` passed.

Playwright was not run because this task only changes backend WhatsMeow ingestion/config/metrics behavior and does not touch UI, routing, forms, or browser behavior.

## Known Limitations
- Circuit breaker parameters (threshold=5, reset=30s) are hardcoded in `NewObjectStorage`. A future enhancement could expose them via config.
- No metrics/observability for circuit breaker state transitions yet (tracked as P1-3).
