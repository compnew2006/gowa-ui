# Phase 3: Tenant-Aware Whatsmeow Connection Pool

## Applied Skills

- `golang-pro`: concurrent runtime registry, reconnect lifecycle, race-safe tests
- `architecture-guardian`: strangler-style refactor behind the existing `ConnectionManager` façade

## Audit Findings

- `whatsmeow.Client` was not a global singleton.
- Runtime clients were previously tracked in `pkg/whatsmeow/manager.go` with `map[uuid.UUID]*whatsmeow.Client` guarded by `sync.RWMutex`.
- There was no long-lived background health monitor for dropped runtime sessions.
- Startup reconnect existed in `cmd/whatomate/main.go`, but disconnect events only updated status and did not trigger ongoing recovery.

## Implemented Changes

- Added `pkg/whatsmeow/pool.go` with:
  - `InstanceKey { OrganizationID, AccountName }`
  - `ConnectionPool` using `sync.Map` for `byKey` and `byInstanceID`
  - per-entry reconnect guards, failure counters, and key reindexing support
- Refactored `pkg/whatsmeow/manager.go` to:
  - replace the old `clients` map with the new pool
  - preserve `Connect`, `Disconnect`, `Logout`, and `GetClient`
  - add `GetClientByKey`, `RegisterInstanceClient`, `ReindexInstance`
  - add explicit `StartHealthMonitor` / `StopHealthMonitor`
  - add bounded reconnect behavior with monitor interval + reconnect timeout config
- Updated `pkg/whatsmeow/events.go` so:
  - transient disconnects stay managed for reconnect attempts
  - `logged_out` and `banned` states evict runtime entries from the pool
  - connect/pair success events refresh pool state
- Updated `internal/handlers/instances.go` so instance rename reindexes the runtime key after DB update.
- Updated `cmd/whatomate/main.go` so server and worker flows start the health monitor when `whatsmeow` is the selected provider.
- Added config knobs in `internal/config/config.go` and `config.example.toml`:
  - `health_monitor_interval_seconds`
  - `reconnect_timeout_seconds`

## Verification

- Passed: `go test ./pkg/whatsmeow/...`
- Passed: `go test ./internal/handlers -run 'TestApp_UpdateInstance_(DuplicateNameConflict|ReindexesConnectedRuntimeKey)$' -count=1`
- Passed: `go test -race ./pkg/whatsmeow`
- Passed: `go test -race ./internal/handlers -run 'TestApp_UpdateInstance_(DuplicateNameConflict|ReindexesConnectedRuntimeKey)$' -count=1`

## Full Regression Blockers

- `go test ./...` is still blocked by pre-existing repository issues unrelated to this change:
  - missing embedded frontend build output for `internal/frontend/embed.go` (`all:dist`)
  - duplicate temporary root binaries in `tmp_encrypt.go` and `tmp_arabic.go`
  - pre-existing failures in `internal/license` and `internal/middleware`
- Browser smoke could not be completed in this worktree:
  - frontend build is blocked because `frontend/node_modules` does not contain `vite`
  - browser MCP sessions were unavailable because the local browser profile was already locked by another process
