# P0-6 Async WhatsMeow Event Ingestion

## Task
Implemented asynchronous WhatsMeow event ingestion so WhatsMeow reader callbacks no longer run DB, media, webhook, campaign, or WebSocket work inline.

## Approach and Key Decisions
- Added a per-instance FIFO async event dispatcher with bounded buffers and one worker per instance.
- Changed `ConnectionManager.newClient` so `AddEventHandler` dispatches events non-blockingly by default.
- Kept `ConnectionManager.handleEvent` as the single source of truth for event behavior; the worker now calls it asynchronously.
- Added `event_buffer_size` and `event_dispatch_enabled` WhatsMeow config fields. Defaults are `4096` and `true`; setting dispatch enabled to `false` restores synchronous handling for emergency rollback.
- Overflow policy is fail-closed/non-blocking: full or stopped queues drop the event, log a warning, and increment per-instance dropped-event metrics.
- Added shutdown hooks so server and worker paths stop dispatcher workers on process shutdown.

## Files Modified or Created
- Created `pkg/whatsmeow/async_events.go`
- Created `pkg/whatsmeow/async_events_test.go`
- Modified `pkg/whatsmeow/manager.go`
- Modified `pkg/whatsmeow/events.go`
- Modified `pkg/whatsmeow/metrics.go`
- Modified `pkg/whatsmeow/metrics_unit_test.go`
- Modified `internal/config/config.go`
- Modified `internal/config/config_test.go`
- Modified `internal/handlers/instances.go`
- Modified `cmd/whatomate/main.go`
- Modified `config.example.toml`

## Tests Added
- Async dispatcher returns immediately while a worker handler is blocked.
- Per-instance FIFO ordering is preserved.
- Different instances process independently.
- Full buffers drop without blocking and increment the drop counter.
- Stop/shutdown drains queued events and closes the dispatcher.
- Config defaults and environment overrides cover `event_buffer_size` and `event_dispatch_enabled`.

## Verification
- `go test ./pkg/whatsmeow -run 'Async|ConnectionManager|Event|Message|Receipt'` passed.
- `go test ./internal/config` passed.
- `go test -p 1 ./pkg/whatsmeow ./internal/config` passed.
- `go test ./cmd/whatomate` passed.
- `go test ./internal/handlers` passed.
- `git diff --check` passed.

Playwright was not run because this task only changes backend WhatsMeow ingestion/config/metrics behavior and does not touch UI, routing, forms, or browser behavior.

## Known Limitations
- If a queue is full, events are intentionally dropped instead of blocking WhatsMeow's reader goroutine. Drops are visible through logs and `events_dropped_today` health metrics.
- `ruvector.db` was already/externally dirtied by tooling state during the session and was left untouched.
