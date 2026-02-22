# CORS & WebSocket Origin Hardening Report

## Date
2026-02-19

## Problem Statement
Origin validation defaults were over-permissive for cookie-authenticated traffic:

- Empty `server.allowed_origins` effectively allowed any origin for CORS.
- WebSocket upgrade origin checks used the same permissive fallback.
- Combined with cookie auth and `/api/auth/ws-token`, this created unnecessary cross-origin exposure risk.

## Critical Risks
1. **Cross-origin data exposure risk**: permissive CORS + `Access-Control-Allow-Credentials: true` can expose authenticated API responses when browser cookie policy permits sending cookies.
2. **WebSocket hijack surface expansion**: permissive WS `CheckOrigin` increases risk of unauthorized browser-initiated socket upgrades from untrusted origins.
3. **Config drift ambiguity**: comments/documentation indicated allow-all fallback, encouraging insecure defaults in deployments.

## Architectural Flaws
- Origin policy behavior depended on an implicit allow-all fallback when no whitelist was configured.
- CORS and WS checks were not backed by a single explicit safe-default policy contract.
- Origin matching lacked normalization, so trailing slash/default-port variants could be brittle.

## Implemented Fixes

### 1. Unified Safe Origin Evaluator
- Added `IsOriginAllowedForRequest` in `internal/middleware/middleware.go`.
- Behavior:
  - If explicit allowlist exists: only exact normalized allowlist origins are accepted.
  - If allowlist is empty: only **same-origin** and **loopback localhost origins** are accepted.
  - Invalid origins are rejected.

### 2. Origin Normalization
- `ParseAllowedOrigins` now normalizes configured origins and skips invalid entries.
- Handles scheme/host normalization and default ports.

### 3. CORS Wrapper Hardening
- Updated `cmd/whatomate/main.go` `corsWrapper`:
  - removed permissive allow-all fallback when allowlist is empty.
  - now sets `Access-Control-Allow-Origin` only for allowed origins.
  - sets `Access-Control-Allow-Credentials: true` and `Vary: Origin` only when origin is allowed.

### 4. WebSocket CheckOrigin Hardening
- Updated `internal/handlers/websocket.go` upgrader `CheckOrigin` to use the same strict evaluator (`IsOriginAllowedForRequest`).

### 5. Regression Tests Added
- Extended `internal/middleware/middleware_test.go` with:
  - allowlist accept/deny coverage
  - default same-origin and localhost fallback coverage
  - default cross-origin deny coverage
  - normalization and invalid-origin parsing coverage
- Added `internal/handlers/websocket_origin_test.go` for WS `CheckOrigin` policy coverage.

### 6. Config Guidance Updated
- Updated comment in `config.example.toml`:
  - empty `allowed_origins` now documented as `same-origin + localhost loopback only`.

## Verification
- `go test ./internal/middleware ./internal/handlers` ✅
- `go test ./...` ✅
- `npm --prefix frontend run typecheck` ✅
- `npm --prefix frontend run lint` ✅ (existing warnings only)
- `npm --prefix frontend run build` ✅

## Result
- Origin policy is now fail-closed for non-loopback cross-origin requests when allowlist is not set.
- CORS and WebSocket origin checks are consistent and centralized.
- Deployment guidance now reflects secure default behavior.

## Operational Recommendation
Set `server.allowed_origins` explicitly in all non-local environments, for example:

```toml
[server]
allowed_origins = "https://app.example.com"
```

For multiple frontends:

```toml
allowed_origins = "https://app.example.com,https://admin.example.com"
```
