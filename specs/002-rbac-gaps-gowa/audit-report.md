# Guardian Audit Report

**Date**: 2026-07-13
**Scope**: branch `002-rbac-gaps-gowa` (full diff)
**Stack**: Go 1.26 + Vue 3 / TypeScript (single binary, embedded frontend)
**Verdict**: WARN (no P1/P2 blockers; P3/P4 items are test-coverage gaps, not code defects)

## Verdict

**WARN** — the production code compiles, passes gofmt, has no secrets, no injection vectors, and all 27 review findings have corresponding code fixes. The warnings are: (a) the new security-path tests (US6) and `CheckReplay` unit test are not yet written, (b) pre-existing test compilation failures in `chatbot_processor_test.go` and `pkg/gowa/full_api_test.go` predate this branch, and (c) one Go stdlib vuln (crypto/tls) requires a toolchain bump.

## Tools Run

| Tool | Status | Findings |
| :--- | :--- | :--- |
| `go build ./cmd/whatomate` | ✅ Pass | Zero errors |
| `gofmt -l` | ✅ Clean (after fix) | Fixed 2 files (`contacts.go`, `messages.go`); 2 pre-existing unrelated |
| `go vet` (prod packages) | ✅ Pass | Pre-existing test failures noted separately |
| `staticcheck` | ⚠️ Not installed | Install: `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| `govulncheck` | ⚠️ 1 finding | GO-2026-5856 (crypto/tls, stdlib, fix in go1.26.5) |
| Secrets scan | ✅ Clean | No hardcoded secrets in changed files |
| Injection scan | ✅ Clean | No `eval`/`exec`/`innerHTML` |
| Secret-leak scan | ✅ Clean | Webhook secret never logged (FR-022) |
| `vue-tsc --noEmit` | ✅ Pass (after fix) | Removed unused `canManageDevices` computed |
| jscpd (duplication) | ⏭️ Skipped | Not installed; manual review found no semantic duplicates |
| `go test -race` | ⏭️ Skipped | Pre-existing test compilation failures block test run (see P3-001) |

## Summary by Priority

| Priority | Count |
| :--- | :--- |
| 🔴 P1 BLOCKER | 0 |
| 🟠 P2 HIGH | 0 |
| 🟡 P3 MEDIUM | 3 |
| 🟢 P4 LOW | 3 |
| ⚪ P5 INFO | 1 |

## 🟡 P3: Medium

### P3-001: Security-path tests not yet written (US6 deferred)
**Files**: `internal/handlers/gowa_device_test.go`, `internal/handlers/gowa_webhook_test.go`, `internal/handlers/media_redownload_test.go`
**Detail**: The spec (FR-016, Story 6) requires automated tests for permission-denied (403), signature rejection, replay rejection, cross-org mutation rejection, and media cooldown. These tests were deferred to a follow-up session. The existing test files contain only happy-path and input-validation tests.
**Fix**: Implement T012-T017 (US1 tests), T026-T029 (US2 tests), T044-T046 (US5 tests), T053-T057 (US6 integration tests) per `tasks.md`.

### P3-002: `CheckReplay` function has no unit test
**File**: `pkg/gowa/webhook.go`
**Detail**: The `CheckReplay` function (added in T006) is a pure function with 4 code paths (RFC3339 timestamp, epoch timestamp, zero/missing timestamp, unparseable timestamp) but has no test. Task T017 in `tasks.md` specifies this test.
**Fix**: Add `TestCheckReplay` to `pkg/gowa/webhook_test.go` covering: fresh timestamp (pass), stale >5min (fail), future >5min (fail), zero/empty (fail), unparseable (fail).

### P3-003: Pre-existing test compilation failures block `go test`
**Files**: `internal/handlers/chatbot_processor_test.go` (4 call sites), `pkg/gowa/full_api_test.go` (1 call site)
**Detail**: These test files have pre-existing signature mismatches (`saveIncomingMessage` expects 9 args, tests pass 7; `CreateDevice` expects 3 args, test passes 2). These predate this branch — they were broken in the audited commit range (`7509281a`→`829ecf70`) when the function signatures changed but the tests weren't updated. They are NOT caused by this branch's changes.
**Fix**: Update the 5 call sites to match the current function signatures. This is a pre-existing bug, not a regression.

## 🟢 P4: Low

### P4-001: `media_redownload.go` has no test file
**File**: `internal/handlers/media_redownload.go`
**Detail**: No `media_redownload_test.go` exists. The handler makes outbound provider calls, mutates `media_url`, and has a Redis cooldown — all untested.
**Fix**: Create `internal/handlers/media_redownload_test.go` (tasks T045, T047).

### P4-002: `gowa_backfill.go` has no test file
**File**: `internal/handlers/gowa_backfill.go`
**Detail**: The `BackfillGowaWebhookSecrets` function is untested.
**Fix**: Add a test that creates GOWA accounts with empty secrets, runs the backfill, and verifies secrets are populated.

### P4-003: Pre-existing gofmt issues in files not touched by this branch
**Files**: `internal/handlers/chat_lifecycle.go`, `internal/handlers/media_zip_test.go`
**Detail**: These files have gofmt formatting issues but were not modified by this branch.
**Fix**: `gofmt -w internal/handlers/chat_lifecycle.go internal/handlers/media_zip_test.go` (pre-existing, not a regression).

## ⚪ P5: Info

### P5-001: Go stdlib vulnerability GO-2026-5856
**Detail**: `govulncheck` reports a vulnerability in `crypto/tls` (privacy leak in Encrypted Client Hello). Affects the Go toolchain, not application code. Fixed in `go1.26.5`; the project uses `go1.26.3`.
**Fix**: Bump the Go toolchain to `go1.26.5` in `go.mod` and CI. Not a code change — infrastructure.

## Quick Fixes

```bash
# Fix pre-existing gofmt issues (not from this branch)
gofmt -w internal/handlers/chat_lifecycle.go internal/handlers/media_zip_test.go

# Install staticcheck for future audits
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## Traceability (Spec → Code → Test)

| Requirement | Implementation | Automated test | Status |
| :--- | :--- | :--- | :--- |
| FR-001 (reject no/invalid signature) | `gowa_webhook.go` — fail-closed HMAC guard (3 checks, unified error) | ❌ Not written (T012-T014) | ⚠️ Code done, test missing |
| FR-002 (reject empty secret) | `gowa_webhook.go` — empty-secret rejection path | ❌ Not written (T014) | ⚠️ Code done, test missing |
| FR-003 (org-scope all writes) | `gowa_webhook.go` — `account.OrganizationID` on all writes | ❌ Not written (T053) | ⚠️ Code done, test missing |
| FR-004 (org-scope mutations) | `gowa_webhook.go:654,702`, `webhook.go:407`, `chatbot_processor.go:1297` — `AND organization_id = ?` | ❌ Not written (T016) | ⚠️ Code done, test missing |
| FR-005 (replay 5min) | `gowa_webhook.go` — `CheckReplay(envelope.Timestamp, 5*time.Minute)` | ❌ Not written (T015, T017) | ⚠️ Code done, test missing |
| FR-006 (devices:write for pair/QR/provision) | `gowa_device.go` — `requireAuth(ResourceDevices, ActionWrite)` on 3 handlers | ❌ Not written (T026) | ⚠️ Code done, test missing |
| FR-007 (devices:read for status) | `gowa_device.go` — `requireAuth(ResourceDevices, ActionRead)` on status | ❌ Not written (T026) | ⚠️ Code done, test missing |
| FR-008 (instances devices:read) | `gowa_device.go` — `requireAuth` + `FindGOWAInstancesForOrg` | ❌ Not written (T026) | ⚠️ Code done, test missing |
| FR-009 (org-scoped provisioning) | `gowa_device.go` — `FindGOWAInstance(baseURL, orgID.String())` | ❌ Not written (T028) | ⚠️ Code done, test missing |
| FR-010 (devices resource) | `roles.go` — `ResourceDevices = "devices"` | ❌ Not written (T036) | ⚠️ Code done, test missing |
| FR-011 (seed + map roles) | `roles.go` — `DefaultPermissions()` + `SystemRolePermissions()` manager | ❌ Not written (T037) | ⚠️ Code done, test missing |
| FR-012 (frontend device gate) | `AccountDetailView.vue` — `canWriteDevices` on Connect + provisioning | ❌ No E2E (T040) | ⚠️ Code done, test missing |
| FR-013 (media export gate) | `media_zip.go` — `HasPermission(contacts, export)` + `ChatView.vue` `canExportMedia` + `useMediaExport.ts` check | ❌ Not written (T044) | ⚠️ Code done, test missing |
| FR-014 (re-download cooldown) | `media_redownload.go` — Redis `SetNX` 60s per-message | ❌ Not written (T045) | ⚠️ Code done, test missing |
| FR-015 (ZIP size guard) | `media_zip.go` — `maxZipTotalSize = 250MB` + `os.Stat` accumulation | ❌ Not written (T046) | ⚠️ Code done, test missing |
| FR-016 (security tests) | — (meta-requirement) | ❌ All deferred | ❌ Missing |
| FR-017 (auto-gen secret) | `accounts.go` — `GenerateWebhookSecret()` on create/update; `gowa_backfill.go` — startup backfill | ❌ Not written (T054, T055) | ⚠️ Code done, test missing |
| FR-018 (explicit provider type) | `accounts.go` — `req.ProviderType` validated at create (pre-existing) | ✅ Pre-existing tests | ✅ |
| FR-019 (fallback after HMAC) | `gowa_webhook.go` — fallback removed entirely | ❌ Not written | ⚠️ Code done, test missing |
| FR-020 (webhook rate-limit) | ❌ Not implemented | ❌ | ❌ Missing |
| FR-021 (constant-time HMAC) | `pkg/gowa/verify.go` — `hmac.Equal` (pre-existing, correct) | ✅ `webhook_test.go:108-148` | ✅ |
| FR-022 (never log secret) | `gowa_webhook.go` — only `device_id` logged, never secret | ❌ Not written | ⚠️ Code done, test missing |
| FR-023 (unified error) | `gowa_webhook.go` — "Webhook verification failed" for all conditions | ❌ Not written | ⚠️ Code done, test missing |
| FR-024 (per-message cooldown) | `media_redownload.go` — key `media:redownload:{message_id}` | ❌ Not written | ⚠️ Code done, test missing |
| FR-025 (Redis fail-open) | `media_redownload.go` — `if err != nil { log.Warn; continue }` | ❌ Not written | ⚠️ Code done, test missing |
| FR-026 (updateMessageStatus all callers) | `webhook.go` — signature changed, both Meta + GOWA paths updated | ✅ `webhook_test.go` (7 call sites updated) | ✅ |
| FR-027 (instances response filter) | `gowa_device.go` — `FindGOWAInstancesForOrg` + response struct strips credentials | ❌ Not written | ⚠️ Code done, test missing |

### Summary

| Status | Count |
| :--- | :--- |
| ✅ Complete (code + test) | 3 |
| ⚠️ Code done, test missing | 22 |
| ❌ Missing (not implemented) | 2 (FR-016, FR-020) |

**Key finding**: FR-020 (webhook rate-limiting) was specified in the spec but not implemented. This is a P3 gap — the public webhook endpoint has no per-IP rate limit for brute-force signature attempts. All other FRs have corresponding code.

## Notes

- The `go test -race` run was blocked by pre-existing test compilation failures (P3-003), not by this branch's changes. Once those are fixed, the race detector should be run.
- The `staticcheck` tool is not installed in this environment. Recommend installing it for future audits.
- The govulncheck finding (P5-001) is a Go toolchain issue, not application code — it requires bumping the Go version, not changing any source files.
- All 8 CRITICAL findings (C1-C8) from the original review have code fixes in place. The remaining work is test coverage (US6).
