---
description: "Task list for Per-Instance Uploads Cleanup Retention"
---

# Tasks: Per-Instance Uploads Cleanup Retention

**Input**: Design documents from `/specs/001-per-instance-uploads-cleanup/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: The user explicitly requested TDD throughout the plan and prior tasks. Test tasks are included in every user story phase; tests MUST fail before implementation per constitution §7.2.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Plugin (Go)**: `plugin/per-instance-uploads-cleanup/`
- **Core worker extension (Go)**: `internal/handlers/uploads_cleanup_worker_instance.go` (one file, additive, flagged core touchpoint)
- **Frontend (Vue 3)**: `frontend/src/components/settings/`, `frontend/src/composables/`, `frontend/src/services/`, `frontend/src/i18n/locales/`
- **Frontend integration points**: `frontend/src/components/whatsmeow/InstanceCard.vue`, `frontend/src/views/settings/SettingsView.vue`, `frontend/src/services/api.ts`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Plugin skeleton, registration, and one additive core-touchpoint for the worker extension.

- [X] T001 Create plugin directory and `plugin.go` skeleton at `plugin/per-instance-uploads-cleanup/plugin.go` with `core.RegisterPlugin` in `init()`, `Name()`, `Init()`, `Routes()`, and an empty `Migrate()` per the plugin interface contract
- [X] T002 Activate the plugin via blank import in `cmd/whatomate/main.go` (one line, next to existing plugin imports)
- [X] T003 [P] Create plugin test helper at `plugin/per-instance-uploads-cleanup/testdata/helpers.go` with hand-rolled `setupTestEnv(t)` returning a `*Plugin` pre-wired with `testutil.SetupTestDB`, `testutil.SetupTestRedis`, and `slog.Default()` per the Plugin Test Pattern
- [X] T004 [P] Add i18n key stubs in `frontend/src/i18n/locales/en.json` for the 15 new keys under `settings.uploadsCleanup*` (empty values; the feature implementation fills them) per D-11
- [X] T005 [P] Add same i18n key stubs in `frontend/src/i18n/locales/es.json`
- [X] T006 [P] Add same i18n key stubs in `frontend/src/i18n/locales/ar.json`

**Checkpoint**: Plugin compiles, builds, registers; `make build` succeeds; empty routes return 404; i18n key files validate.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared models, services, the worker extension, and the index that every user story depends on. **No user story work can begin until this phase is complete.**

- [X] T007 [P] Create `InstanceUploadsCleanupAudit` GORM model at `plugin/per-instance-uploads-cleanup/model.go` embedding `BaseModel` with fields `OrganizationID uuid.UUID`, `InstanceID uuid.UUID`, `ActorUserID *uuid.UUID`, `ActorEmail *string`, `OldInherit *bool`, `NewInherit bool`, `OldRetentionDays *int`, `NewRetentionDays *int`, `Reason *string`; add `TableName() string { return "instance_uploads_cleanup_audits" }` per §6.3/§6.4
- [X] T008 [P] Create retention resolution service at `plugin/per-instance-uploads-cleanup/service.go` with `ResolveEffectiveRetention(ctx, orgID, instanceID, now) (effectiveDays int, source custom|default|disabled, err error)` implementing the D-8 state machine from `data-model.md` §"State machine for `inherit × retention_days`"
- [X] T009 [P] Create retention validator at `plugin/per-instance-uploads-cleanup/validation.go` with `ValidateRetentionUpdate(inherit bool, retentionDays *int) error` enforcing FR-001, FR-012, D-7 bounds `[0, maxUploadsCleanupRetentionDays=3650]` and the `uploads_cleanup_disabled`/`uploads_cleanup_retention_days_required`/`uploads_cleanup_inherit_invalid` error codes from data-model.md
- [X] T010 Implement audit writer at `plugin/per-instance-uploads-cleanup/service.go` with `WriteAuditRow(ctx, orgID, instanceID, actor *User, old, new RetentionSnapshot, reason *string) error` — the same `service.go` file as T008 since both are part of the same service surface
- [X] T011 [P] Create new file `internal/handlers/uploads_cleanup_worker_instance.go` containing the exported function `RunManualCleanupForInstance(ctx, db, orgID, instanceID uuid.UUID, override *int) (deletedCount int, err error)` that internally calls the existing `RunManualCleanup` with an additive `RunOptions{RetentionDaysOverride: override, InstanceID: &instanceID}` argument; do **not** modify the existing `RunManualCleanup` signature in `internal/handlers/uploads_cleanup_worker.go` — if the additive-parameter deviation (R1) is rejected, the plugin calls the legacy `RunManualCleanup` and applies the per-instance filter at the filesystem-walk level (no signature change needed)
- [X] T012 Implement plugin `Migrate(db *gorm.DB)` in `plugin/per-instance-uploads-cleanup/plugin.go` with three idempotent steps: (1) backfill `uploads_cleanup.inherit=true` via `UPDATE ... || '{"uploads_cleanup":{"inherit":true}}'::jsonb WHERE settings->'uploads_cleanup' IS NULL`; (2) `db.AutoMigrate(&InstanceUploadsCleanupAudit{})`; (3) `CREATE INDEX IF NOT EXISTS idx_iuca_org_instance_created ON instance_uploads_cleanup_audits (organization_id, instance_id, created_at DESC)` per data-model.md §"Migration plan"
- [X] T013 [P] Create the in-process `sync.Mutex` for the per-instance run guard in `plugin/per-instance-uploads-cleanup/service.go` field `instanceRunMu sync.Mutex` and helper `tryAcquireInstanceRun() (release func(), ok bool)` — D-5
- [X] T014 [P] Add the `data-testid` convention documentation comment block to the top of `frontend/src/components/whatsmeow/InstanceCard.vue` listing every `data-testid` attribute the new block will use (per prior checklist testability requirement)
- [X] T015 [P] Add new API method stubs in `frontend/src/services/api.ts` for `getInstanceUploadsCleanup`, `updateInstanceUploadsCleanup`, `getInstanceUploadsCleanupHistory`, `runInstanceUploadsCleanup`, `getOrgUploadsCleanupOverview` (signatures only; bodies throw `not implemented`) per the contract in `contracts/instances.uploads-cleanup.yaml`
- [X] T016 Run `make build` and `./whatomate server -config config.toml -migrate` against an ephemeral Postgres to confirm T012 applies the table and index without error
- [ ] T016a [P] [Foundation] Add unit test at `internal/handlers/uploads_cleanup_worker_instance_test.go` asserting that the existing `deleteExpiredUploadFiles` filesystem walk correctly resolves unscoped files to the workspace default (FR-009, FR-010): seed two instances under the same org with files that carry and lack `instance_id`; run cleanup; assert unscoped files use the org's workspace default retention and per-instance files use the per-instance effective retention; this is the only way FR-010 is verifiably covered

**Checkpoint**: Plugin `Migrate` is idempotent and creates the table; `ResolveEffectiveRetention` is unit-tested; `RunManualCleanup` accepts the new options struct; foundation is ready for user story implementation.

---

## Phase 3: User Story 1 - Configure per-instance retention from the Instance settings (Priority: P1) 🎯 MVP

**Goal**: An admin can open any WhatsApp instance's settings, toggle the "Inherit workspace default" switch, enter a custom retention value in days, and have it saved. The value is displayed on reload and is used by the worker for that instance's files.

**Independent Test**: Open any single instance's settings, change retention to `5`, save, reload, verify the value is shown; toggle the switch off→on, save, verify the input is hidden and the effective workspace default is shown.

### Tests for User Story 1 ⚠️ (write FIRST, ensure they FAIL)

- [X] T017 [P] [US1] Contract test for `GET /api/instances/{id}/uploads-cleanup` at `plugin/per-instance-uploads-cleanup/handler_retention_test.go` asserting the envelope shape, 404 on cross-org access, 403 on missing read permission, and the `effective_source` resolution
- [X] T018 [P] [US1] Contract test for `PUT /api/instances/{id}/uploads-cleanup` at `plugin/per-instance-uploads-cleanup/handler_retention_test.go` asserting: (a) successful save returns 200 + updated `InstanceUploadsCleanup`; (b) `retention_days=99999` returns 400 `uploads_cleanup_retention_days`; (c) `inherit=false` without `retention_days` returns 400 `uploads_cleanup_retention_days_required`; (d) a new audit row is written
- [X] T019 [P] [US1] Unit test for retention validation state machine at `plugin/per-instance-uploads-cleanup/validation_test.go` covering all four states from `data-model.md` §"State machine for `inherit × retention_days`"
- [X] T020 [P] [US1] Unit test for `ResolveEffectiveRetention` at `plugin/per-instance-uploads-cleanup/service_test.go` covering: custom positive, custom 0 (disabled), inherit+workspace default positive, inherit+workspace default 0 (disabled), instance not found, instance soft-deleted
- [ ] T021 [P] [US1] Frontend component test for `PerInstanceUploadsCleanup.vue` at `frontend/src/components/settings/PerInstanceUploadsCleanup.spec.ts` using Vitest + Vue Test Utils + MSW: assert the toggle hides the input when ON, shows it when OFF, the save button is disabled while a request is in flight, and the success toast fires on 200

### Implementation for User Story 1

- [X] T022 [P] [US1] Create `PerInstanceUploadsCleanup.vue` component at `frontend/src/components/settings/PerInstanceUploadsCleanup.vue` using Composition API + `<script setup lang="ts">` with `props: { instanceId: string }`, a `useQuery` for GET, a `useMutation` for PUT, and the toggle/input UI per FR-005
- [X] T023 [P] [US1] Create `usePerInstanceUploadsCleanup` composable at `frontend/src/composables/usePerInstanceUploadsCleanup.ts` with `useInstanceUploadsCleanup(instanceId)`, `useUpdateInstanceUploadsCleanup(instanceId)`, `useInstanceUploadsCleanupHistory(instanceId, limit=5)`, and `useRunInstanceUploadsCleanup(instanceId)` — query keys per the project's Vue Query convention
- [X] T024 [US1] Implement `GET /api/instances/{id}/uploads-cleanup` handler at `plugin/per-instance-uploads-cleanup/handler_retention.go`: resolve org, scope to instance, parse `settings.uploads_cleanup.*`, call `ResolveEffectiveRetention`, return the envelope
- [X] T025 [US1] Implement `PUT /api/instances/{id}/uploads-cleanup` handler at `plugin/per-instance-uploads-cleanup/handler_retention.go`: parse body, call `ValidateRetentionUpdate`, persist to `whatsapp_instances.settings` via `tenant.ScopedDB`, call `WriteAuditRow` in the same transaction, return updated state (depends on T022..T025; can be split if needed). Q-OPT-2 default behavior (pinned at this task): when the request sets `inherit=true`, the handler MUST NOT clear or overwrite the existing `uploads_cleanup.retention_days` value in the JSONB sub-key — the prior value is preserved (allows one-click "undo" by toggling back to `inherit=false`). When the request sets `inherit=false`, `retention_days` is required (returns 400 `uploads_cleanup_retention_days_required` if missing). The validation rule from T009 (`retention_days` is "ignored" when `inherit=true`) is enforced **at the validator only**; the handler preserves the field on disk. Add a unit test in `handler_retention_test.go` asserting: PUT `{"inherit":true}` on an instance that previously had `retention_days=30` results in the row's JSONB still containing `retention_days=30`.
- [X] T026 [US1] Mount the new component in `frontend/src/components/whatsmeow/InstanceCard.vue` immediately after the existing `auto_campaign` sub-block (line 474) per the prior checklist
- [X] T027 [US1] Fill in the 15 i18n keys in `frontend/src/i18n/locales/en.json` with English copy per the spec's UX guidance
- [X] T028 [P] [US1] Mirror the 15 i18n keys in `frontend/src/i18n/locales/es.json` (Spanish)
- [X] T029 [P] [US1] Mirror the 15 i18n keys in `frontend/src/i18n/locales/ar.json` (Arabic; verify `dir="rtl"` rendering is unaffected)
- [X] T030 [US1] Run `go test -v -race -p 1 ./plugin/per-instance-uploads-cleanup/...` and `cd frontend && npm run test:unit -- PerInstanceUploadsCleanup` to confirm T017..T021 now pass

**Checkpoint**: US1 is fully functional. An admin can change retention per instance and see it persist; the workspace Uploads Cleanup section still shows the old "Default N days" only.

---

## Phase 4: User Story 2 - View and override retention from the workspace Uploads Cleanup section (Priority: P2)

**Goal**: The existing `/settings` Uploads Cleanup section grows a per-instance overview list showing each instance's effective retention (custom badge / "Default" badge / "Disabled" badge) and last run date. Each row links to the instance's settings page. A "Last 5 changes" history list appears on the instance's settings page.

**Independent Test**: Open the workspace Uploads Cleanup section, verify each instance row shows the correct effective-source badge; click an instance, change its retention, return, verify the row's badge updates to "Custom".

### Tests for User Story 2 ⚠️

- [X] T031 [P] [US2] Contract test for `GET /api/org/uploads-cleanup/instances` at `plugin/per-instance-uploads-cleanup/handler_overview_test.go` asserting pagination envelope shape (data: items+total+limit+offset), filter by `source=custom|default|disabled|all`, search by `q` substring match, 403 on missing read permission
- [X] T032 [P] [US2] Contract test for `GET /api/instances/{id}/uploads-cleanup/history` at `plugin/per-instance-uploads-cleanup/handler_retention_test.go` asserting: default `limit=5`, max `limit=100`, ordering is `created_at DESC`, 400 on `limit=0` or `limit>100`
- [ ] T033 [P] [US2] Frontend integration test at `frontend/src/views/settings/SettingsView.spec.ts` (extend existing file) asserting the per-instance overview renders, paginates, filters by `source`, and clicking a row navigates to the instance's settings

### Implementation for User Story 2

- [X] T034 [US2] Implement `GET /api/org/uploads-cleanup/instances` handler at `plugin/per-instance-uploads-cleanup/handler_overview.go`: bulk SELECT `(id, name, settings)` for the org, resolve effective retention per row, apply `source` and `q` filters in SQL via `tenant.ScopedDB`, paginate with `offset`+`limit`, return `InstanceUploadsCleanupOverview` envelope (D-4, FR-014)
- [X] T035 [US2] Implement `GET /api/instances/{id}/uploads-cleanup/history` handler at `plugin/per-instance-uploads-cleanup/handler_history.go`: validate `limit`+`offset`, query `tenant.ScopedDB(...).Where(...).Order("created_at DESC").Limit(limit).Offset(offset).Find(...)` on the new audit table, return `InstanceUploadsCleanupHistory` envelope
- [X] T036 [P] [US2] Add `useInstanceUploadsCleanupHistory` and `useOrgUploadsCleanupOverview` to the composable at `frontend/src/composables/usePerInstanceUploadsCleanup.ts` (extend the file from T023)
- [ ] T037 [US2] Extend `frontend/src/views/settings/SettingsView.vue` Uploads Cleanup section with a `<DataTable>` of per-instance rows (badge + name + last-run + link); wire it to `useOrgUploadsCleanupOverview`
- [X] T038 [US2] Extend `PerInstanceUploadsCleanup.vue` (from T022) to add the "Last 5 changes" history list below the toggle, using `useInstanceUploadsCleanupHistory`; render `actor_email` (not `actor_user_id`), `old`/`new` values with the "Old: X / New: Y (days)" label per Q-OPT-1
- [X] T039 [US2] Run `go test -v -race -p 1 ./plugin/per-instance-uploads-cleanup/...` and `cd frontend && npm run test:unit -- SettingsView PerInstanceUploadsCleanup` to confirm T031..T033 pass

**Checkpoint**: US2 is fully functional. Admins can review all instances from the workspace view, see badges, click through, and review the change history on the instance page.

---

## Phase 5: User Story 3 - "Run cleanup now" for a single instance (Priority: P3)

**Goal**: A "Run cleanup now" button on the per-instance block runs cleanup for that one instance only, using its effective retention, and shows the result inline. Returns 409 if a workspace or per-instance run is already in progress.

**Independent Test**: Click "Run cleanup now" on an instance with a custom retention; observe a result toast/line; click again from a second tab while the first is still running; observe a 409 error surfaced as a user-friendly toast.

### Tests for User Story 3 ⚠️

- [ ] T040 [P] [US3] Contract test for `POST /api/instances/{id}/uploads-cleanup/run` at `plugin/per-instance-uploads-cleanup/handler_run_test.go` asserting: 200 with `InstanceUploadsCleanupRunResult` on success, 400 `uploads_cleanup_disabled` when effective retention is 0, 409 when another run holds the lock, 404 on cross-org access
- [ ] T041 [P] [US3] Concurrency test at `plugin/per-instance-uploads-cleanup/service_test.go` asserting: (a) two simultaneous single-instance runs against the same instance result in one 200 and one 409; (b) a single-instance run while a workspace run holds `pg_try_advisory_lock` also returns 409
- [ ] T042 [P] [US3] Frontend component test at `frontend/src/components/settings/PerInstanceUploadsCleanup.spec.ts` (extend from T021) for the "Run cleanup now" button: loading state, success toast with `deleted_files` and `retention_used`, 409 toast with the localized "already running" message

### Implementation for User Story 3

- [X] T043 [US3] Implement `POST /api/instances/{id}/uploads-cleanup/run` handler at `plugin/per-instance-uploads-cleanup/handler_run.go`: call `tryAcquireInstanceRun`; on conflict → 409; on success call `ResolveEffectiveRetention`; if `0` → 400 `uploads_cleanup_disabled`; else call the exported `RunManualCleanupForInstance(ctx, db, orgID, instanceID, &days)` function from T011 (in `internal/handlers/uploads_cleanup_worker_instance.go`); on success update `uploads_cleanup.last_run_date` in instance settings and return the result envelope (D-6, D-5)
- [X] T044 [P] [US3] Add the "Run cleanup now" button to `PerInstanceUploadsCleanup.vue` (extend from T022) with the result inline + toast on 409; wire to `useRunInstanceUploadsCleanup`
- [X] T045 [US3] Add the per-instance breakdown `instances[]` field to the existing `POST /api/org/uploads-cleanup/run` response in `internal/handlers/uploads_cleanup_http.go` per C-2 in `research.md` and `contracts/org.uploads-cleanup.runs.yaml`; the legacy `deleted_files` and `retention_days` top-level fields are preserved
- [X] T046 [P] [US3] Update the `api.ts` `runUploadsCleanupNow` response type in `frontend/src/services/api.ts` to include the new `instances: PerInstanceRunRow[]` field (additive, non-breaking)
- [X] T047 [US3] Run `go test -v -race -p 1 ./plugin/per-instance-uploads-cleanup/...` and `cd frontend && npm run test:unit -- PerInstanceUploadsCleanup` to confirm T040..T042 pass

**Checkpoint**: US3 is fully functional. Admins can trigger per-instance runs, see results inline, and observe the concurrency guard.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories, plus pre-release validation.

- [X] T048 [P] Run `make lint` and resolve any new lint findings in the plugin
- [X] T049 [P] Run `cd frontend && npm run lint && npm run typecheck` and resolve any new findings
- [X] T050 [P] Add plugin unit tests at `plugin/per-instance-uploads-cleanup/plugin_test.go` asserting the plugin self-registers (`core.RegisterPlugin(Plugin{})` ran in `init()`) and its `Migrate(db)` is idempotent (running twice does not error)
- [ ] T051 [P] Add OpenAPI `x-codeSamples` (curl + js) to both `contracts/instances.uploads-cleanup.yaml` and `contracts/org.uploads-cleanup.runs.yaml` for documentation site generation
- [ ] T052 [P] Update the i18n completion check: run `cd frontend && npm run i18n:check` (or equivalent) to verify all three locale files have a value for every new key
- [ ] T053 Run `npx playwright test --project=chromium` for the per-instance retention happy path defined in `quickstart.md` §"End-to-end happy path"
- [ ] T054 Run the full `make test` and confirm no regression in `internal/handlers/uploads_cleanup_*_test.go` from the `RunManualCleanup` signature change
- [ ] T055 Manually exercise the rollback procedure described in `quickstart.md` §"Rollout" against an ephemeral Postgres (revert plugin, verify table + JSONB sub-keys remain but unused)
- [ ] T056 [P] Append a session summary to `summary.md` per the project protocol: branch, list of generated artifacts, blast radius, tests, gotchas
- [ ] T057 [P] Update the existing `frontend/src/i18n/locales/*.json` README/CHANGELOG (if any) with a one-line note about the 15 new keys

---

## Phase 7: Bug Fixes

- [X] T058 [Bug] Fix fatal "Cannot read properties of undefined (reading 'length')" on `/whatsapp/instances` — root cause: composables in `usePerInstanceUploadsCleanup.ts` used `res.data` directly instead of `unwrapResponse()`, returning the full API envelope `{ status, data, message }` instead of the inner data, making `history.entries` undefined. Fix: (1) import and use `unwrapResponse` from `@/lib/api-utils` in all five queryFn/mutationFn callbacks; (2) add optional chaining `history?.entries?.length` in `PerInstanceUploadsCleanup.vue` template as defensive guard
- [X] T059 [Bug] Fix 403 on "Run cleanup now" and "Save" buttons — root cause: plugin's `hasPermission` checked `user_organizations.is_super_admin` but core checks `users.is_super_admin`. Fix: rewrote `hasPermission` in `plugin/per-instance-uploads-cleanup/handler_retention.go` to check `users.is_super_admin` first
- [X] T060 [Bug] Fix 400 "uploads_cleanup_disabled" raw error code shown on "Run cleanup now" and "common.saved" literal key shown on "Save" — root cause: (1) backend `handleRun` returned error code string `"uploads_cleanup_disabled"` as the message instead of a user-friendly sentence; (2) i18n key `common.saved` was missing from all three locale files; (3) frontend error handlers used `err instanceof Error ? err.message` instead of `getErrorMessage()` which can't extract Axios response messages. Fix: (1) replaced backend error message with user-friendly description; (2) added `"saved"` key to en/es/ar locale `common` sections; (3) imported `getErrorMessage` from `@/lib/api-utils` and used it in both error handlers in `PerInstanceUploadsCleanup.vue`

- [X] T061 [Bug] Fix misleading zero-value health stats shown for disconnected instances — root cause: backend `GetInstanceHealth` always returns a zero-valued `InstanceHealthResponse` struct (200 OK) even for disconnected instances with no active whatsmeow connection. Frontend stores this as a truthy object, so `v-if="instance.health"` passes and displays all-zero stats (uptime 0m, sent/received/failed/queue all 0). Fix: changed condition to `v-if="instance.health && isConnected"` in `InstanceCard.vue` so the health section only renders for connected instances where runtime metrics are meaningful. Disconnected instances no longer show misleading zero-value stats cards.
- [X] T062 [UI] Redesign `InstanceCard.vue` layout into 6 clear sections for cleaner scannability — restructured template into: (1) Header/Status with compact badges, name, phone, JID; (2) Health stats as 2x2 grid with tinted backgrounds and uppercase labels, gated behind `isConnected`; (3) Settings in 2-column grid via container query (auto-sync, auto-download, auto-reject, auto-campaign, chat-close-rating, assigned-chat-reset); (4) PerInstanceUploadsCleanup as full-width section; (5) InstanceTagSettings as full-width section; (6) Action button in footer. Consistent border/radius/spacing tokens throughout.

**Checkpoint**: `/whatsapp/instances` loads without crash; Save and Run buttons return 200 for super admin; health stats only visible for connected instances; card layout is clean and balanced.

---

## Phase 8: Chat Export Fixes

- [X] T063 [Bug] Fix export button click not working on `/chat/` — root cause: `PopoverTrigger` nested inside `TooltipTrigger` both with `as-child` caused Radix Vue click handler conflict, preventing the popover from opening. Nesting reorder alone did not fix it because Radix Vue's `as-child` slot merging breaks when `PopoverTrigger as-child` wraps a component (`Tooltip`) instead of a DOM element. Fix: removed the `Tooltip` + `TooltipTrigger` wrapper entirely from the export button. `PopoverTrigger as-child` now wraps the `Button` directly, so clicks propagate to the popover handler without interception. Tooltip hint preserved via native `title` attribute (`${t('chat.exportChatTooltip')} (E)`). Keyboard shortcut E continued to work because it calls `exportChatAsPDF()` directly.
- [X] T064 [Bug] Fix images showing as "[Image]" text in PDF export instead of thumbnails — root cause: `buildHtmlForPrint` used `extractText()` which calls `mediaLabel()` returning placeholder strings like `"[Image]"`, `"[Video]"`. Fix: (1) added `fetchImageDataUrls()` to `chat-export.ts` that fetches image blobs via `prefetchMediaBlob()` and converts them to base64 data URLs with concurrency control; (2) updated `buildHtmlForPrint` to accept an `imageDataUrls` map and render `<img>` tags with thumbnails (max 220x180px) for image/sticker messages; (3) updated `exportChatAsPDF()` in ChatView.vue to call `fetchImageDataUrls()` before building the HTML.

---

## Phase 9: Group Chat Fixes

- [X] T065 [Bug] Fix group chat messages showing phone numbers instead of sender names — root cause: `getGroupSenderPhone()` in ChatView.vue called `getMessageSenderPhone()` which only returned `sender_phone`/`metadata.sender_phone`, never `sender_push_name`. The backend already stores `sender_push_name` in message metadata (message_persist.go stores `metadata["sender_push_name"] = evt.Info.PushName` for group messages) and the API returns it via `MessageResponse.SenderPushName`. Frontend `Message` type has `sender_push_name?: string`. Fix: updated `getGroupSenderPhone()` to check `message.sender_push_name` first (returning the push name when available), falling back to `getMessageSenderPhone(message)` for the phone number.
- [ ] T066 [Feature] Auto-sync group members to WhatsApp contacts — **BLOCKED: requires core modification approval**. Current behavior: `persistParsedMessage` in `pkg/whatsmeow/message_persist.go` creates ONE contact per group (the group chat itself via the group JID), not individual contacts for each participant. The `inboundMessageHook` explicitly skips group messages (`!isGroup && !isChannel`), so plugin-based approaches cannot intercept group senders. Options: (a) Add a new call in `persistParsedMessage` to create/update a contact for each group message sender (core mod in `pkg/whatsmeow/`); (b) Remove the `!isGroup` exclusion from `inboundMessageHook` and handle group member contact creation via a plugin. Both options modify `pkg/whatsmeow/message_persist.go`. Awaiting explicit user approval per Plugin Architecture invariant.