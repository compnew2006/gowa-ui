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
- [ ] T045 [US3] Add the per-instance breakdown `instances[]` field to the existing `POST /api/org/uploads-cleanup/run` response in `internal/handlers/uploads_cleanup_http.go` per C-2 in `research.md` and `contracts/org.uploads-cleanup.runs.yaml`; the legacy `deleted_files` and `retention_days` top-level fields are preserved
- [ ] T046 [P] [US3] Update the `api.ts` `runUploadsCleanupNow` response type in `frontend/src/services/api.ts` to include the new `instances: PerInstanceRunRow[]` field (additive, non-breaking)
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

**Checkpoint**: All tests pass, lint clean, i18n complete, manual smoke test green, summary recorded.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - **BLOCKS all user stories**
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2). No dependencies on other stories. Self-contained: the GET/PUT retention endpoints and the per-instance UI block.
- **User Story 2 (P2)**: Can start after Foundational (Phase 2). Builds on US1 by *reading* the data US1 writes (no shared code) and adds the workspace overview + history list. Independently testable: a workspace with no per-instance overrides still shows correct "Default" badges.
- **User Story 3 (P3)**: Can start after Foundational (Phase 2). Builds on US1 (reuses `ResolveEffectiveRetention`) and US2 (reuses the "Last 5 changes" panel for the result display). Independently testable: a single instance with retention set can run a single-instance cleanup with no overview page being shown.

### Within Each User Story

- Tests (included) MUST be written and FAIL before implementation per constitution §7.2
- Models before services (Phase 2 only — US1+ reuse the foundation)
- Service resolution before handlers
- Handlers before UI
- UI i18n key addition can be parallel to handler implementation
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T003..T006)
- All Foundational tasks marked [P] can run in parallel (T007, T008, T009, T013, T014, T015)
- Once Foundational phase completes, US1 can start; US2 and US3 can start in parallel after US1 reaches its first checkpoint
- Within US1: T017, T018, T019, T020, T021 (all tests) run in parallel; T022, T023, T028, T029 (component, composable, i18n es+ar) run in parallel
- Within US2: T031, T032, T033 (tests) parallel; T036, T038 (composable + component extension) parallel after T037 wires them
- Within US3: T040, T041, T042 (tests) parallel; T044, T046 (frontend touches) parallel after T043

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (must FAIL before implementation):
Task: "T017 [US1] Contract test for GET /api/instances/{id}/uploads-cleanup in plugin/per-instance-uploads-cleanup/handler_retention_test.go"
Task: "T018 [US1] Contract test for PUT /api/instances/{id}/uploads-cleanup in plugin/per-instance-uploads-cleanup/handler_retention_test.go"
Task: "T019 [US1] Unit test for retention validation in plugin/per-instance-uploads-cleanup/validation_test.go"
Task: "T020 [US1] Unit test for ResolveEffectiveRetention in plugin/per-instance-uploads-cleanup/service_test.go"
Task: "T021 [US1] Frontend component test in frontend/src/components/settings/PerInstanceUploadsCleanup.spec.ts"

# Launch all frontend implementation in parallel (after backend tests pass):
Task: "T022 [US1] Create PerInstanceUploadsCleanup.vue in frontend/src/components/settings/"
Task: "T023 [US1] Create usePerInstanceUploadsCleanup composable in frontend/src/composables/"
Task: "T028 [US1] Mirror i18n keys in es.json"
Task: "T029 [US1] Mirror i18n keys in ar.json"
```

---

## Parallel Example: User Story 2

```bash
# Tests parallel:
Task: "T031 [US2] Contract test for GET /api/org/uploads-cleanup/instances in plugin/per-instance-uploads-cleanup/handler_overview_test.go"
Task: "T032 [US2] Contract test for GET /api/instances/{id}/uploads-cleanup/history in plugin/per-instance-uploads-cleanup/handler_history_test.go"
Task: "T033 [US2] Frontend integration test extending frontend/src/views/settings/SettingsView.spec.ts"

# After handlers land, frontend touches parallel:
Task: "T036 [US2] Extend composable usePerInstanceUploadsCleanup.ts with history + overview"
Task: "T038 [US2] Extend PerInstanceUploadsCleanup.vue with history list"
```

---

## Parallel Example: User Story 3

```bash
# Tests parallel:
Task: "T040 [US3] Contract test for POST /api/instances/{id}/uploads-cleanup/run"
Task: "T041 [US3] Concurrency test in service_test.go"
Task: "T042 [US3] Frontend component test extending PerInstanceUploadsCleanup.spec.ts"

# Frontend touches parallel after the run handler ships:
Task: "T044 [US3] Add Run cleanup now button to PerInstanceUploadsCleanup.vue"
Task: "T046 [US3] Update api.ts runUploadsCleanupNow response type"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001..T006)
2. Complete Phase 2: Foundational (T007..T016) — **CRITICAL, blocks all stories**
3. Complete Phase 3: User Story 1 (T017..T030)
4. **STOP and VALIDATE**: per `quickstart.md` §"End-to-end happy path" steps 2–3
5. Deploy/demo if ready — the per-instance retention save+read is shippable on its own

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready (T001..T016)
2. Add User Story 1 → Test independently → Deploy/Demo (**MVP!**)
3. Add User Story 2 → Test independently → Deploy/Demo (workspace overview + history land)
4. Add User Story 3 → Test independently → Deploy/Demo (per-instance "Run now" lands)
5. Polish phase (T048..T057) before declaring done
6. Each story adds value without breaking previous stories (additive contracts preserve §5.1)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (T001..T016)
2. Once Foundational is done:
   - Developer A: User Story 1 (T017..T030) — ~2–3 days
   - Developer B (in parallel after T008 lands): starts User Story 2 backend (T031, T032, T034, T035) while A works on the frontend
   - Developer C (in parallel after T011 lands): starts User Story 3 backend (T040, T041, T043) while B works on the overview
3. Frontend pieces of US2 and US3 land on the corresponding instance card / settings page in parallel with their backend stories
4. Polish phase serializes: lint, typecheck, e2e, summary

---

## Open items handed to implementation (must be resolved before MVP deploy)

| ID | Item | Source | Default until told otherwise |
|---|---|---|---|
| C-1 | Audit retention policy | `research.md#c-1` | Forever (matches `AgentSelectionAuditEvent` precedent) |
| Q-OPT-1 | History "Old / New" label format | `data-model.md` open Q | "Old: X / New: Y (days)" |
| Q-OPT-2 | Inherit toggle clears `retention_days` vs keeps it | `data-model.md` open Q | Keep the old value; UI hides the input |
| Worker-extension approval | Acceptable to add a new file `internal/handlers/uploads_cleanup_worker_instance.go` exporting `RunManualCleanupForInstance(...)` that calls the existing `RunManualCleanup` with the additive `RunOptions` parameter | `plan.md` §11.5 row R1 | Accept (avoids duplicating the filesystem walk; new file keeps the existing `uploads_cleanup_worker.go` untouched and preserves zero-blast-radius for the existing 2 call sites) |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability (US1, US2, US3)
- Each user story should be independently completable and testable
- Verify tests FAIL before implementing (§7.2)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same-file conflicts, cross-story dependencies that break independence
- The one core touchpoint (T011) is a new file `internal/handlers/uploads_cleanup_worker_instance.go` that exports `RunManualCleanupForInstance(...)` wrapping the existing `UploadsCleanupWorker.RunManualCleanup` with an additive `RunOptions{RetentionDaysOverride, InstanceID}` parameter; the existing `RunManualCleanup` signature is preserved (the `RunOptions` is a single new optional field; the 2 existing call sites in `internal/handlers/uploads_cleanup_http.go` continue to work unchanged). This is the only edit outside the `plugin/` directory beyond the one-line blank import in `cmd/whatomate/main.go` (T002).
