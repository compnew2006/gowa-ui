# Session Summery

## Date

- 2026-04-10

## Task

- Implement Phase 2 tenant-aware campaign queue refactoring for Whatomate using Redis Streams, including tenant queue routing, tenant-aware campaign consumption, and an explicit legacy campaign stream migration command.

## Skills Applied

- `golang-pro`
- `architecture-guardian`

## Audit Findings

- The existing queue implementation hardcoded global Redis stream keys in `internal/queue/redis.go`:
  - `whatomate:campaigns`
  - `whatomate:campaigns:dlq`
  - `whatomate:inbound_media`
- `RecipientJob` and `ContactRepairJob` both used the global `whatomate:campaigns` stream.
- `InboundMediaJob` used the separate global `whatomate:inbound_media` stream.
- There was no tenant-aware queue manager, no tenant campaign consumer, and no legacy campaign stream migration helper.
- Successfully processed legacy campaign jobs were `XACK`ed but not `XDEL`ed, so the old global stream could contain historical ACKed entries alongside real backlog.

## Changes Made

- Added `internal/queue/tenant.go`:
  - exported `TenantQueueManager`
  - exported `CampaignStreamName(orgID)` and `CampaignDeadLetterStreamName(orgID)`
  - per-organization routing for `RecipientJob` and `ContactRepairJob`
  - retained global routing for `InboundMediaJob`
  - exported `NewTenantCampaignConsumer` that:
    - discovers `whatomate:org:*:campaigns`
    - ensures `campaign-workers` exists on each discovered stream
    - consumes across multiple tenant streams
    - claims pending entries per source stream
    - ACKs and DLQs against the originating stream
- Added `internal/queue/migration.go`:
  - exported `CampaignMigrationOptions`
  - exported `CampaignMigrationSummary`
  - exported `MigrateLegacyCampaignStream(...)`
  - explicit lock-protected migration flow for legacy `whatomate:campaigns`
  - dry-run inspection mode
  - apply mode that migrates unread and pending backlog only
  - invalid/no-org messages are preserved in the legacy stream and reported
- Updated `internal/worker/worker.go`:
  - worker campaign consumer now depends on `queue.Consumer`
  - worker uses the new tenant campaign consumer
  - inbound-media consumer remains unchanged
- Updated `cmd/whatomate/main.go`:
  - server and worker bootstrap now use `TenantQueueManager`
  - added `queue-migrate-campaigns` CLI command
  - added usage/help text for the new CLI command
- Added `internal/queue/tenant_queue_test.go` covering:
  - tenant stream name usage
  - mixed-org enqueue fanout
  - missing-organization rejection
  - inbound-media staying global
  - tenant consumer across multiple streams
  - legacy migration scenarios

## Migration Rollout Notes

- Runtime campaign production and consumption now use per-org streams:
  - `whatomate:org:{org_id}:campaigns`
  - `whatomate:org:{org_id}:campaigns:dlq`
- Legacy `whatomate:campaigns` is not consumed by the new tenant campaign consumer.
- Backlog must be moved explicitly with:
  - `whatomate queue-migrate-campaigns -config config.toml -apply`
- Migration safety behavior:
  - acquires a Redis lock before apply
  - uses backlog state rather than copying the full stream
  - migrates unread and pending work
  - leaves ACKed historical entries untouched
  - preserves invalid legacy entries in place and reports them

## Verification

- Queue package tests:
  - `go test ./internal/queue -count=1`
  - passed
- Requested regression slice:
  - `go test ./internal/queue ./internal/worker ./pkg/whatsmeow -run 'Test(NewRedis|Enqueue|Consume|Reconcile)' -count=1`
  - passed
- Migration-focused slice:
  - `go test ./internal/queue -run 'TestMigrateLegacyCampaignStream' -count=1`
  - passed

## Notes

- Browser tooling was not used for verification because this task is backend-only and has no browser-facing behavior to validate.
- `go test ./cmd/whatomate -run '^$' -count=1` is blocked by a pre-existing asset issue in `internal/frontend/embed.go`:
  - `pattern all:dist: no matching files found`
  - this is unrelated to the queue refactor itself.
