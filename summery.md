# Prompt1 Session Summary

## Scope
- Phase 1 only: database foundation and tenant isolation.
- Worktree: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate-prompt1`
- Branch: `codex/prompt1-phase1-db-foundation`

## Audit Findings
- `OrganizationConfig` did not exist in `internal/models`.
- Auth already carried tenant identity via `organization_id`, `user_id`, and super-admin context values.
- Query isolation was still largely manual in handlers; there was no request-scoped GORM tenant seam enforcing org filtering centrally.

## Skills Applied
- `architecture-guardian`
  - blast-radius analysis
  - safe migration rollout
  - layered tenant-boundary design
- `golang-pro`
  - idiomatic Go/GORM implementation
  - table-driven regression tests
  - runtime verification

## Implemented Changes
- Added `internal/models/organization_config.go`
  - one-to-one org config model
  - fields: `organization_id`, `worker_count`, `max_queue_size`, `max_whatsapp_instances`
  - explicit GORM column tags to preserve the intended database names
- Updated `internal/models/models.go`
  - linked `Organization.Config` to `OrganizationConfig`
- Updated `internal/database/postgres.go`
  - added `OrganizationConfig` to migrations
  - added `BackfillOrganizationConfigs`
  - migration now backfills a config row for every existing organization
- Added `internal/tenant/scope.go`
  - shared `ResolveOrganizationID`
  - request-scoped `ScopedDB`
  - request-context storage and retrieval for scoped `*gorm.DB`
- Updated `internal/middleware/middleware.go`
  - added `TenantScope` middleware
- Updated `cmd/whatomate/main.go`
  - attached tenant middleware to authenticated tenant `/api` routes
  - excluded public/auth/webhook/SSO/custom-action redirect routes from tenant scoping
- Updated `internal/handlers/app.go`
  - added `requestDB(r)` helper
  - switched `getOrgID` to the shared tenant resolver
- Swept tenant-serving request handlers
  - replaced direct `a.DB` access in request-bound handlers with `requestDB := a.requestDB(r)`
  - preserved non-request/background/manual flows
- Updated `test/testutil/db.go`
  - included `OrganizationConfig` in shared test migrations and cleanup

## Verification
- Package compile checks
  - `go test ./internal/database -run '^$'`
  - `go test ./internal/middleware -run '^$'`
  - `go test ./internal/handlers -run '^$'`
- Focused regression suite
  - `go test ./internal/database ./internal/models ./internal/middleware ./internal/handlers -count=1`
- Production build
  - `make build-prod`
  - `go build -o whatomate ./cmd/whatomate`
- Runtime smoke
  - started isolated server on `127.0.0.1:18081` with throwaway config `/tmp/whatomate-prompt1-smoke.toml`
  - created isolated database `whatomate_prompt1_smoke`
  - login succeeded for seeded admin `prompt1-admin@example.com`
  - authenticated endpoint smoke succeeded:
    - `GET /api/accounts`
    - `GET /api/contacts`
    - `GET /api/instances`
  - health endpoint succeeded:
    - `GET /health`

## Tests Added
- `internal/database/postgres_test.go`
  - migration list includes `OrganizationConfig`
  - backfill execution is covered
- `internal/models/models_test.go`
  - `Organization` schema exposes the `Config` relation
  - `OrganizationConfig` schema uses the expected database column names
- `internal/middleware/middleware_test.go`
  - tenant middleware stores scoped DB for default org
  - membership-based org override works
  - super-admin org override works

## Runtime Notes
- Initial runtime smoke exposed two real regressions and both were fixed:
  - `MaxWhatsAppInstances` needed an explicit `gorm:"column:max_whatsapp_instances"` tag.
  - `TenantScope` initially ran on public auth routes and blocked `/api/auth/login`; route exclusions were added.
- Browser MCP was attempted, but Chrome/Playwright backends were unavailable in this desktop session due existing browser-profile contention/timeouts. Live HTTP smoke was completed against the running prompt1 binary instead.

---

# Prompt2 Session Summary

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
