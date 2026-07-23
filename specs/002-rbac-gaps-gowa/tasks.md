# Tasks: Close RBAC / User-Role Gaps in GOWA + Media Features

**Input**: Design documents from `/specs/002-rbac-gaps-gowa/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/
**Tests**: Included — the spec explicitly requests tests (Story 6, FR-016). Tests are Go integration tests (`testify` + `test/testutil/`) per constitution Principle 15.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `internal/` (handlers, models, config), `pkg/` (gowa, whatsapp), `cmd/whatomate/` (main)
- **Frontend**: `frontend/src/` (components, views, stores, composables, i18n)
- **Tests**: `internal/handlers/*_test.go` (Go integration), `frontend/e2e/` (Playwright)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify the existing project builds and tests pass before changes begin.

- [ ] T001 Verify Go backend compiles with `go build ./cmd/whatomate` in whatomate/
- [ ] T002 [P] Verify existing tests pass with `make test` in whatomate/ (baseline before changes)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared prerequisites that MUST be complete before any user story can be implemented. The `devices` permission catalog is needed by US2 (handlers call `requireAuth(ResourceDevices, ...)`) and US4 (frontend checks `hasPermission('devices', ...)`). The webhook-secret auto-generation is needed by US1 (accounts must have secrets for HMAC verification to work).

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 Add `ResourceDevices = "devices"` constant to the `PermissionResource` block in `internal/models/roles.go` (after `ResourceAccounts`, ~line 64)
- [ ] T004 Add `devices:read` and `devices:write` entries to `DefaultPermissions()` in `internal/models/roles.go` (~line 132, after the Accounts block)
- [ ] T005 Add `"devices:read", "devices:write"` to `managerPermissions` in `SystemRolePermissions()` in `internal/models/roles.go` (~line 256); do NOT add to `agentPermissions`
- [ ] T006 [P] Add `CheckReplay(timestamp int64, maxAge time.Duration) bool` pure function to `pkg/gowa/webhook.go` (research R2: 5-minute window, allows clock drift)
- [ ] T007 [P] Add `Organizations []string` field to `GOWAInstance` struct in `internal/config/config.go` (~line 166) and update `FindGOWAInstance` to filter by orgID (research R7: `["*"]` or empty = all)
- [ ] T008 Auto-generate `GowaWebhookSecret` via `gowa.GenerateWebhookSecret()` when a GOWA-type account is created without one — in `internal/handlers/accounts.go` `CreateAccount` (~line 130, after provider-type validation)
- [ ] T009 Auto-generate `GowaWebhookSecret` on update if the account is GOWA-type and the caller didn't supply one — in `internal/handlers/accounts.go` `UpdateAccount` (~line 290)
- [ ] T010 Add startup backfill: scan for GOWA accounts with empty `GowaWebhookSecret` and generate one — in `cmd/whatomate/main.go` after DB init (research R5)
- [ ] T011 [P] Add i18n keys for new error messages to all 5 locale files (`frontend/src/i18n/locales/{en,ar,es,hi,ta}.json`): `errors.media.tooLarge`, `errors.media.redownloadCooldown`, `errors.gowa.missingSignature`, `errors.gowa.invalidSignature`, `errors.gowa.noSecret`

**Checkpoint**: Foundation ready — `devices` permission exists and is seeded; `CheckReplay` helper available; org-scoped instance resolution available; GOWA accounts always have webhook secrets; i18n keys in place.

---

## Phase 3: User Story 1 — Inbound GOWA Webhooks Are Authenticated and Tenant-Isolated (Priority: P1) 🎯 MVP

**Goal**: Fail-close the GOWA webhook HMAC guard, add replay protection, and scope all downstream writes to the resolved account's organization — closing the unauthenticated cross-tenant contamination vector (review findings C4–C7, H3).

**Independent Test**: Send a POST to `/api/gowa/webhook` with (a) no signature header, (b) a tampered body, (c) a wrong secret, (d) a stale timestamp (>5 min). All are rejected with no DB writes. Then send a properly signed request and verify the message appears only in the target org.

### Tests for User Story 1

> **NOTE**: Write these tests FIRST, ensure they FAIL before implementation.

- [ ] T012 [P] [US1] Add `TestGowaWebhook_MissingSignature_Rejected` in `internal/handlers/gowa_webhook_test.go` — POST with no `X-Hub-Signature-256` header → assert 403, assert zero DB writes (no contact, no message created)
- [ ] T013 [P] [US1] Add `TestGowaWebhook_TamperedBody_Rejected` in `internal/handlers/gowa_webhook_test.go` — POST with valid header but altered body → assert 403, zero writes
- [ ] T014 [P] [US1] Add `TestGowaWebhook_EmptySecret_Rejected` in `internal/handlers/gowa_webhook_test.go` — POST for an account with no `GowaWebhookSecret` → assert 403, zero writes (after backfill, this should not occur for new accounts; test the fail-close path)
- [ ] T015 [P] [US1] Add `TestGowaWebhook_StaleTimestamp_Dropped` in `internal/handlers/gowa_webhook_test.go` — POST with valid signature but timestamp 6 min old → assert 200 but zero writes (silently dropped)
- [ ] T016 [P] [US1] Add `TestGowaWebhook_CrossOrgMutation_Ignored` in `internal/handlers/gowa_webhook_test.go` — forged webhook carrying a revoked/edit/reaction referencing another org's message ID → assert the target message is NOT mutated
- [ ] T017 [P] [US1] Add `TestCheckReplay` unit test in `pkg/gowa/webhook_test.go` — test fresh timestamp (pass), stale >5min (fail), future >5min (fail), zero timestamp (fail)

### Implementation for User Story 1

- [ ] T018 [US1] Fail-close the HMAC guard in `internal/handlers/gowa_webhook.go` ~line 72: replace `if sigHeader != "" && account.GowaWebhookSecret != ""` with fail-closed logic — reject if `secret == ""` (403 + log alert), reject if `header == ""` (403), reject if `!verify(...)` (403) (research R1, contracts/gowa-webhook-api.md)
- [ ] T019 [US1] Add replay check call in `internal/handlers/gowa_webhook.go` after HMAC verification, before event dispatch: call `gowa.CheckReplay(envelope.Timestamp, 5*time.Minute)` — if stale, log Warn and return 200 with no processing (research R2)
- [ ] T020 [US1] Move the `getGowaAccountByDeviceID` fallback path (`internal/handlers/gowa_webhook.go` ~lines 122-142, which iterates all tenants' accounts and makes outbound `GetAppStatus` calls) to AFTER the HMAC check — the fallback must never run on an unauthenticated request (research R4, finding M5)
- [ ] T021 [US1] Add `AND organization_id = ?` to `processGowaRevoked` query in `internal/handlers/gowa_webhook.go` ~line 654 — use `account.OrganizationID` (finding C7, research R4)
- [ ] T022 [US1] Add `AND organization_id = ?` to `processGowaEdited` query in `internal/handlers/gowa_webhook.go` ~line 702 — use `account.OrganizationID`
- [ ] T023 [US1] Change `updateMessageStatus` signature in `internal/handlers/webhook.go` ~line 399 to accept `orgID uuid.UUID` and add `AND organization_id = ?` to the query at ~line 407 (research R8); update all callers to pass the orgID from their resolved account
- [ ] T024 [US1] Add `AND organization_id = ?` to `handleIncomingReaction` LIKE query in `internal/handlers/chatbot_processor.go` ~line 1297/1327 — use `account.OrganizationID` (finding C7, L2)
- [ ] T025 [US1] Add structured logging (`a.Log.Warn`) for each rejection path in `internal/handlers/gowa_webhook.go`: missing signature, invalid signature, empty secret, stale timestamp (constitution Principle 16)

**Checkpoint**: GOWA webhooks are fail-closed, replay-protected, and all writes are org-scoped. Forged webhooks cannot inject messages, contacts, or mutations into any tenant. User Story 1 is independently testable.

---

## Phase 4: User Story 2 — Device Management Is Restricted to Authorized Roles (Priority: P1)

**Goal**: Add `requireAuth(r, ResourceDevices, ActionRead/Write)` to all five GOWA device handlers, and scope `GowaCreateDevice` instance selection to the caller's organization — closing the authenticated privilege-escalation vector (review findings C1, C2, C3, H6).

**Independent Test**: Log in as an agent and attempt all five device endpoints → all return 403. Log in as a manager → all succeed for own-org accounts. Attempt cross-org provisioning → refused.

### Tests for User Story 2

- [ ] T026 [P] [US2] Add `TestGowaDevice_AgentDenied_AllEndpoints` in `internal/handlers/gowa_device_test.go` — agent-role user gets 403 on QR, pair-code, status, instances, create-device
- [ ] T027 [P] [US2] Add `TestGowaDevice_ManagerSucceeds_OwnOrg` in `internal/handlers/gowa_device_test.go` — manager-role user succeeds on all five endpoints for accounts in their org
- [ ] T028 [P] [US2] Add `TestGowaDevice_CrossOrgProvisioning_Refused` in `internal/handlers/gowa_device_test.go` — manager from org A cannot provision on org B's instance → 400
- [ ] T029 [P] [US2] Add `TestGowaDevice_WebhookSecret_NotExposedToAgent` in `internal/handlers/gowa_device_test.go` — agent cannot obtain `webhook_secret` via create-device (403 before reaching the GOWA provider)

### Implementation for User Story 2

- [ ] T030 [US2] Replace `getOrgID` with `requireAuth(r, models.ResourceDevices, models.ActionWrite)` in `GowaLoginQR` in `internal/handlers/gowa_device.go` ~line 24/62 (contracts/gowa-device-api.md)
- [ ] T031 [US2] Replace `getOrgID` with `requireAuth(r, models.ResourceDevices, models.ActionWrite)` in `GowaPairCode` in `internal/handlers/gowa_device.go` ~line 24/103
- [ ] T032 [US2] Replace `getOrgID` with `requireAuth(r, models.ResourceDevices, models.ActionRead)` in `GowaDeviceStatus` in `internal/handlers/gowa_device.go` ~line 24/218
- [ ] T033 [US2] Replace `getOrgID` with `requireAuth(r, models.ResourceDevices, models.ActionRead)` in `GowaInstances` in `internal/handlers/gowa_device.go` ~line 133; also filter the returned instance list to org-allowed instances only (do NOT include `username`/`password` in the response)
- [ ] T034 [US2] Replace `getOrgID` with `requireAuth(r, models.ResourceDevices, models.ActionWrite)` in `GowaCreateDevice` in `internal/handlers/gowa_device.go` ~line 159; pass `orgID` to the org-scoped `FindGOWAInstance` (T007) so cross-org instances are rejected (finding C2)
- [ ] T035 [US2] Add `a.logAudit(orgID, userID, "devices", deviceID, "write", nil, device)` to `GowaCreateDevice` after successful device creation in `internal/handlers/gowa_device.go` (constitution Principle 17)

**Checkpoint**: All five GOWA device endpoints enforce `devices:read`/`devices:write`. Agents are denied. Cross-org provisioning is refused. Provisioning is audited. User Stories 1 AND 2 both work independently.

---

## Phase 5: User Story 3 — The Permission Catalog Knows About Device Management (Priority: P2)

**Goal**: Verify the `devices` permission (added in Foundational Phase 2) is seeded, visible in the role-settings UI, and correctly mapped to system roles. This story's implementation was done in T003–T005; this phase is verification + tests.

**Independent Test**: As admin, call `GET /api/permissions` and verify `devices:read`/`devices:write` appear. Check the manager role has both; the agent role has neither. Open `/settings/roles` and verify the "Device Management" group is visible.

### Tests for User Story 3

- [ ] T036 [P] [US3] Add `TestPermissions_DevicesSeeded` in `internal/handlers/roles_test.go` (or `permissions_test.go`) — `GET /api/permissions` returns entries with `resource == "devices"` for both `read` and `write` actions
- [ ] T037 [P] [US3] Add `TestSystemRoles_DevicesMapping` in `internal/handlers/roles_test.go` — manager role has `devices:read` + `devices:write`; agent role has neither

### Verification for User Story 3

- [ ] T038 [US3] Verify the `PermissionMatrix` UI at `/settings/roles` displays a "Device Management" group with read/write entries — manual check or Playwright E2E in `frontend/e2e/roles.spec.ts`
- [ ] T039 [US3] Verify a freshly initialized database (via `test/testutil/`) seeds `devices:read`/`devices:write` without manual migration — integration test asserting the permissions exist after `AutoMigrate` + seed

**Checkpoint**: The `devices` permission is a first-class catalog entry, seeded by default, visible in the UI, and correctly mapped to admin/manager (not agent).

---

## Phase 6: User Story 4 — The Frontend Hides Device-Management Actions From Unauthorized Users (Priority: P2)

**Goal**: Add `v-if` permission gates to the GOWA device-management controls in `AccountDetailView.vue` so agents don't see buttons they can't use. The "Connect Device" button, pair-code form, and provisioning controls require `devices:write`; the instance/status panel requires `devices:read`.

**Independent Test**: Log in as an agent, open a GOWA account's detail page → no Connect Device button, no pair form, no provisioning UI. Log in as a manager → all visible.

### Tests for User Story 4

- [ ] T040 [P] [US4] Add Playwright E2E test in `frontend/e2e/account-detail.spec.ts` — agent viewing GOWA account detail sees no device-management controls; manager sees all

### Implementation for User Story 4

- [ ] T041 [US4] Add `&& canWrite` (where `canWrite = authStore.hasPermission('devices', 'write')`) to the "Connect Device" button `v-if` in `frontend/src/views/settings/AccountDetailView.vue` ~line 490 (currently `v-if="!isNew && account && isGowa"`)
- [ ] T042 [US4] Add a `canManageDevices` computed (`authStore.hasPermission('devices', 'read')`) in `frontend/src/views/settings/AccountDetailView.vue` and gate the instance dropdown + device-status panel with `v-if="canManageDevices"`
- [ ] T043 [US4] Gate the provisioning block (instance dropdown + "Create Device" button) in `frontend/src/views/settings/AccountDetailView.vue` ~lines 599-615 with `v-if="canWrite"` (reuse the existing `canWrite` computed or add `canWriteDevices`)

**Checkpoint**: Agents see no device-management UI on the account-detail page. Managers see all controls. The Save/Delete buttons' existing gating is unchanged.

---

## Phase 7: User Story 5 — Media Export and Re-download Are Permission-Aware and Abuse-Resistant (Priority: P2)

**Goal**: Tier media access: bulk ZIP download requires `contacts:export`; re-download stays at `contacts:read` but adds a 60-second per-item cooldown. Add a total-size guard on ZIP generation. Hide frontend ZIP controls from users lacking `contacts:export`.

**Independent Test**: As an agent (lacks `contacts:export`), request `/api/media/zip` → 403. As a manager, request ZIP → 200. Trigger re-download twice rapidly → second gets 429. Request ZIP with >250MB of media → 413.

### Tests for User Story 5

- [ ] T044 [P] [US5] Add `TestMediaZip_ExportPermission_Required` in `internal/handlers/media_zip_test.go` — agent without `contacts:export` gets 403; manager with `contacts:export` gets 200
- [ ] T045 [P] [US5] Add `TestMediaRedownload_Cooldown` in a new `internal/handlers/media_redownload_test.go` — first re-download returns 200; immediate second returns 429; after 60s returns 200 again
- [ ] T046 [P] [US5] Add `TestMediaZip_TotalSizeGuard` in `internal/handlers/media_zip_test.go` — request ZIP with media totaling >250MB → 413

### Implementation for User Story 5

- [ ] T047 [US5] Change the permission gate in `ServeMediaZip` at `internal/handlers/media_zip.go` ~line 81 from `HasPermission(userID, ResourceContacts, ActionRead, orgID)` to `HasPermission(userID, ResourceContacts, ActionExport, orgID)` (research R6)
- [ ] T048 [US5] Add a total-size guard in `internal/handlers/media_zip.go`: before buffering the ZIP (~line 103), sum the media file sizes; if >250MB, return `413 "ZIP archive too large"` (FR-015)
- [ ] T049 [US5] Add a Redis cooldown in `internal/handlers/media_redownload.go` before the provider call (~line 81): `SET media:redownload:{msgID} 1 EX 60 NX` — if the key exists, return `429 "Re-download recently performed"` (FR-014, research R6)
- [ ] T050 [US5] Gate the "Collect files" toolbar button and floating chip in `frontend/src/views/chat/ChatView.vue` ~lines 1979, 2081 with `v-if="authStore.hasPermission('contacts', 'export')"` (FR-013)
- [ ] T051 [US5] Gate the ZIP/separate-download buttons in `frontend/src/components/chat/MediaBurstDialog.vue` ~lines 107, 111 with `v-if="authStore.hasPermission('contacts', 'export')"`
- [ ] T052 [US5] Add a permission check in `downloadAsZip` in `frontend/src/composables/useMediaExport.ts` ~line 44: return early with a toast if the user lacks `contacts:export` (defense-in-depth before the `fetch` call)

**Checkpoint**: Bulk ZIP download is gated on `contacts:export`. Re-download is rate-limited. ZIP has a size guard. Frontend hides ZIP controls from unauthorized users. The retry button stays visible to all `contacts:read` users.

---

## Phase 8: User Story 6 — Security-Critical Paths Have Tests (Priority: P3)

**Goal**: Comprehensive cross-story integration tests that verify the security fixes work together — no regressions when multiple stories interact. This phase covers test gaps not already filled by per-story tests in Phases 3–7.

**Independent Test**: Run `go test -race ./internal/handlers/...` — all security-path tests pass. Run `go test -race -run "Gowa|Media" ./internal/handlers/...` — zero failures.

### Tests for User Story 6

- [ ] T053 [P] [US6] Add `TestGowaWebhook_FullSignedFlow_OrgScoped` in `internal/handlers/gowa_webhook_test.go` — a properly signed webhook for org A's device creates a contact + message ONLY in org A; org B's DB is unchanged (end-to-end happy path with org isolation)
- [ ] T054 [P] [US6] Add `TestGowaWebhookSecret_AutoGenerated_OnCreate` in `internal/handlers/accounts_test.go` — creating a GOWA-type account without supplying a `GowaWebhookSecret` results in a non-empty secret stored (encrypted) on the account
- [ ] T055 [P] [US6] Add `TestGowaWebhookSecret_Backfill` in `internal/handlers/accounts_test.go` (or a startup test) — existing GOWA accounts with empty secrets get one generated after the backfill runs
- [ ] T056 [P] [US6] Add `TestGowaCreateDevice_AuditLogged` in `internal/handlers/gowa_device_test.go` — after a manager provisions a device, an `audit_logs` row exists with `resource_type = "devices"`, `action = "write"` (constitution Principle 17)
- [ ] T057 [P] [US6] Add `TestUpdateMessageStatus_OrgScoped` in `internal/handlers/webhook_test.go` — a status update for a message in org A does NOT affect a message with the same `whats_app_message_id` in org B (if such a collision exists in the test fixture)

**Checkpoint**: All security-critical paths have automated test coverage. The full test suite passes with `-race`. Zero untested permission or signature paths remain.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation, and cleanup across all stories.

- [ ] T058 Run full test suite: `make test` in whatomate/ — verify zero regressions across all existing + new tests
- [ ] T059 [P] Run `golangci-lint run ./...` and `gofmt -l .` — fix any lint/format issues introduced by the changes (constitution Principle 19: conventional commits)
- [ ] T060 [P] Run frontend type check: `cd frontend && npm run type-check` — verify no TypeScript errors from the new `v-if` gates and permission checks
- [ ] T061 Run the quickstart.md verification scripts manually (7 curl-based checks in `specs/002-rbac-gaps-gowa/quickstart.md`) — confirm each returns the expected HTTP status
- [ ] T062 Verify the `USER_ROLES_REVIEW_7509281.md` findings are resolved: all 8 CRITICAL + 6 HIGH findings have corresponding code changes; update the review file's status if desired

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — **BLOCKS all user stories**
- **User Stories (Phases 3–8)**: All depend on Foundational phase completion
  - US1 (Phase 3) and US2 (Phase 4) are both P1 — US1 first (unauthenticated vector is highest risk)
  - US3 (Phase 5) verifies the Foundational catalog work — can run in parallel with US1/US2
  - US4 (Phase 6) depends on US2 (frontend gates are meaningful only if backend gates exist)
  - US5 (Phase 7) is independent of US1–US4 — can run in parallel
  - US6 (Phase 8) depends on US1 + US2 + US5 being implemented (tests the integrated fixes)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational (T006 `CheckReplay`, T008–T010 webhook secret). No dependency on other stories.
- **US2 (P1)**: Depends on Foundational (T003–T005 `devices` permission, T007 org-scoped instances). No dependency on US1.
- **US3 (P2)**: Depends on Foundational (T003–T005 catalog addition). Implementation is already done; this phase is verification + tests.
- **US4 (P2)**: Depends on US2 (backend `devices:write` gate must exist for frontend hiding to be meaningful). Can implement frontend gates in parallel but tests require US2.
- **US5 (P2)**: Independent of US1–US4. Can start after Foundational.
- **US6 (P3)**: Depends on US1 + US2 + US5 (cross-story integration tests require all fixes in place).

### Within Each User Story

- Tests are written FIRST and must FAIL before implementation (TDD)
- Implementation tasks follow the contract file for that story
- Multiple tasks touching the SAME file are sequential (not [P])
- Tasks touching DIFFERENT files are marked [P] and can run in parallel

### Parallel Opportunities

- **Foundational**: T006 (`pkg/gowa/webhook.go`), T007 (`internal/config/config.go`), T011 (`frontend/src/i18n/`) can run in parallel with T003–T005 (`internal/models/roles.go`) and T008–T010 (`internal/handlers/accounts.go` + `cmd/whatomate/main.go`)
- **US1 tests**: T012–T017 all touch test files and can run in parallel
- **US1 implementation**: T018–T020, T021–T022, T023, T024 touch different files/regions but T018–T020 + T025 all modify `gowa_webhook.go` → sequential within that file
- **US2 tests**: T026–T029 all in `gowa_device_test.go` → can be written in parallel
- **US2 implementation**: T030–T035 all in `gowa_device.go` → sequential
- **US5**: T047–T048 (`media_zip.go`) and T049 (`media_redownload.go`) are different files → parallel; T050–T052 are frontend files → parallel with backend
- **Cross-story**: US1 (backend webhook) and US5 (backend media) touch entirely different files → can run in parallel after Foundational

---

## Parallel Example: User Story 1

```bash
# Launch all US1 test tasks together (different test functions, same file — write sequentially or split):
Task: "T012 TestGowaWebhook_MissingSignature_Rejected in internal/handlers/gowa_webhook_test.go"
Task: "T013 TestGowaWebhook_TamperedBody_Rejected in internal/handlers/gowa_webhook_test.go"
Task: "T017 TestCheckReplay unit test in pkg/gowa/webhook_test.go"  # different file — truly parallel

# After tests fail, launch implementation tasks:
# gowa_webhook.go tasks (sequential — same file):
Task: "T018 Fail-close HMAC guard in internal/handlers/gowa_webhook.go"
Task: "T019 Add replay check in internal/handlers/gowa_webhook.go"
Task: "T020 Move fallback after HMAC in internal/handlers/gowa_webhook.go"
# Different files (parallel with above):
Task: "T023 updateMessageStatus org-scope in internal/handlers/webhook.go"
Task: "T024 handleIncomingReaction org-scope in internal/handlers/chatbot_processor.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verify build + tests pass)
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (webhook fail-close + replay + org-scope)
4. **STOP and VALIDATE**: Run US1 tests + quickstart Verification 1 — forged webhooks are rejected with zero writes
5. Deploy if ready — the unauthenticated cross-tenant vector is closed

### Incremental Delivery

1. Setup + Foundational → Foundation ready (devices permission, webhook secrets, helpers)
2. Add US1 → Test independently → **Deploy** (MVP — closes the unauthenticated vector)
3. Add US2 → Test independently → **Deploy** (closes the authenticated privilege-escalation)
4. Add US3 → Verify catalog → **Deploy** (devices permission is visible/manageable)
5. Add US4 → Test frontend → **Deploy** (agents don't see device UI)
6. Add US5 → Test media → **Deploy** (export tiered, re-download throttled)
7. Add US6 → Run full suite → **Deploy** (all paths tested, regression-safe)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 (webhook auth — `gowa_webhook.go`, `webhook.go`, `chatbot_processor.go`)
   - Developer B: US2 (device RBAC — `gowa_device.go`)
   - Developer C: US5 (media — `media_zip.go`, `media_redownload.go`, frontend)
3. US3 (catalog verification) and US4 (frontend device gating) follow after US2
4. US6 (integration tests) after US1 + US2 + US5 merge

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (TDD)
- Commit after each task or logical group (constitution Principle 19: conventional commits, e.g. `fix(gowa): fail-close webhook HMAC verification`)
- Stop at any checkpoint to validate story independently
- The `devices` permission catalog (T003–T005) is in Foundational, not US3, because US2 (P1) depends on it existing — US3 is verification only
- `updateMessageStatus` (T023) is called from both Meta and GOWA webhook paths — the signature change must update ALL callers
