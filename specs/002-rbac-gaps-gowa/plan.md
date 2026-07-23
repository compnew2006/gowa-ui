# Implementation Plan: Close RBAC / User-Role Gaps in GOWA + Media Features

**Branch**: `002-rbac-gaps-gowa` | **Date**: 2026-07-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-rbac-gaps-gowa/spec.md`

## Summary

The GOWA provider, media burst/zip, and group-routing drop (commits `7509281a`→`829ecf70`) introduced 27 RBAC/user-role gaps. This plan closes them by: (1) fail-closing the GOWA webhook HMAC guard and adding replay protection + org-scoped device resolution, (2) adding `requireAuth` with a new `devices` permission to all five GOWA device handlers, (3) extending the RBAC catalog with a `devices` resource, (4) gating the frontend GOWA + media controls, (5) tiering media export behind `contacts:export`, and (6) adding security-path tests. All changes follow the project's 19-principle constitution — no new frameworks, no new patterns.

## Technical Context

**Language/Version**: Go 1.22+ (backend), Vue 3 + TypeScript (frontend)
**Primary Dependencies**: fastglue + fasthttp (HTTP), GORM (ORM), logf (logging), Pinia + shadcn-vue (frontend)
**Storage**: PostgreSQL (GORM AutoMigrate), Redis (caching/rate-limit)
**Testing**: Go integration tests (`test/testutil/` + testify), Playwright E2E (`frontend/e2e/`)
**Target Platform**: Linux server (single binary, embedded frontend)
**Project Type**: Single binary web app (Go + embedded Vue)
**Performance Goals**: Webhook verification < 10ms; no added latency on non-GOWA paths
**Constraints**: No new HTTP framework; no `net/http`; no SQL migration files; every query org-scoped
**Scale/Scope**: 6 stories, ~18 functional requirements, 27 review findings; touches ~12 Go files + ~6 Vue/TS files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| 1 | Single-binary, embedded frontend | ✅ Pass | No new binary/deployment changes |
| 2 | Fastglue + fasthttp only | ✅ Pass | All handlers keep `func (a *App) Name(r *fastglue.Request) error`; no `net/http` |
| 3 | Global auth middleware, handler-level permissions | ✅ **Core fix** | Adding `requireAuth` to GOWA handlers IS the fix for Principle 3 compliance |
| 4 | Every query scoped by `organization_id` | ✅ **Core fix** | Adding org-scope to device lookup + reaction/revoked/edited/ack updates |
| 5 | Response envelopes | ✅ Pass | All new/modified handlers use `SendEnvelope`/`SendErrorEnvelope` |
| 6 | Explicit response builders | ✅ Pass | `GowaCreateDevice` already returns a response map; no raw model exposure |
| 7 | GORM AutoMigrate, no SQL files | ✅ Pass | New `ResourceDevices` is a constant + `DefaultPermissions` entry, no new table |
| 8 | JSONB for flexible data | ✅ Pass | No new ad-hoc columns; existing `GowaWebhookSecret` field already on the model |
| 9 | WebSocket: typed messages, explicit field mapping | ✅ Pass | No new WS message types; existing GOWA broadcasts unchanged |
| 10 | `whatsapp.Provider` interface | ✅ Pass | No new providers; existing GOWA provider stays behind the interface |
| 11 | Vue 3 `<script setup>` + Pinia setup stores | ✅ Pass | Frontend gating uses existing `authStore.hasPermission` + `v-if` |
| 12 | Cookie auth + CSRF + Web Locks | ✅ Pass | No auth changes; reuses existing cookie/session infrastructure |
| 13 | shadcn-vue + Tailwind, dark-first | ✅ Pass | No new components; gating adds `v-if` to existing buttons |
| 14 | `$t()` for all user-facing text | ✅ Pass | New error messages use existing i18n keys or add keys to all 5 locales |
| 15 | Go integration tests + Playwright E2E | ✅ Pass | New tests use `newTestApp` builder + testify; Story 6 |
| 16 | `logf` structured logging | ✅ Pass | Webhook rejection/replay logs use `a.Log.Warn`/`Debug` with key-value pairs |
| 17 | Audit mutations + cache hot reads | ✅ Pass | Device provisioning is audited via existing `logAudit`; permission cache invalidated via `InvalidateRolePermissionsCache` |
| 18 | TOML + env config via koanf | ✅ Pass | No config changes; existing `GOWAInstance` config unchanged |
| 19 | Conventional commits | ✅ Pass | Commits: `fix(gowa): fail-close webhook HMAC verification`, etc. |

**Gate result: PASS — zero violations.** This feature IS the remediation of Principles 3 and 4 violations found in the review.

## Project Structure

### Documentation (this feature)

```text
specs/002-rbac-gaps-gowa/
├── plan.md              # This file
├── research.md          # Phase 0: technical decisions
├── data-model.md        # Phase 1: entity + permission catalog changes
├── quickstart.md        # Phase 1: local verification guide
├── contracts/           # Phase 1: API contracts
│   ├── gowa-device-api.md
│   ├── gowa-webhook-api.md
│   └── media-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```text
whatomate/
├── internal/
│   ├── models/
│   │   └── roles.go                    # ADD ResourceDevices + DefaultPermissions entries + SystemRolePermissions mappings
│   ├── handlers/
│   │   ├── gowa_device.go              # FIX: add requireAuth + org-scope instance resolution
│   │   ├── gowa_webhook.go             # FIX: fail-close HMAC + org-scope device lookup + replay window + org-scope mutations
│   │   ├── media_zip.go                # FIX: gate ZIP on contacts:export
│   │   ├── media_redownload.go         # ADD: cooldown enforcement
│   │   ├── webhook.go                  # FIX: org-scope updateMessageStatus query
│   │   ├── chatbot_processor.go        # FIX: org-scope handleIncomingReaction LIKE query
│   │   ├── accounts.go                 # FIX: auto-generate GowaWebhookSecret on create/update
│   │   └── *_test.go                   # ADD: authz/IDOR/spoofing tests (Story 6)
│   ├── config/
│   │   └── config.go                   # FIX: org-scope FindGOWAInstance (if instances map to orgs)
│   └── database/
│       └── postgres.go                 # NO CHANGE (ResourceDevices is a permission, not a model)
├── frontend/src/
│   ├── views/settings/
│   │   └── AccountDetailView.vue       # FIX: add canWrite/canRead gates to GOWA buttons
│   ├── views/chat/
│   │   └── ChatView.vue                # FIX: add export-permission gate to collect-files
│   ├── components/chat/
│   │   ├── MediaBurstDialog.vue        # FIX: gate ZIP buttons on export permission
│   │   └── MediaRetryButton.vue        # OK (stays visible to contacts:read users)
│   ├── composables/
│   │   └── useMediaExport.ts           # FIX: add permission check before zip download
│   └── i18n/locales/
│       └── *.json (×5)                 # ADD: new error message keys
├── pkg/gowa/
│   └── verify.go                       # NO CHANGE (VerifyWebhookSignature is already correct)
└── cmd/whatomate/
    └── main.go                         # NO CHANGE to routes; GOWA routes stay in authed group
```

**Structure Decision**: Single-binary web app (constitution Principle 1). All changes are in-place fixes to existing files — no new packages, no new modules. The `devices` permission is a constant + catalog entry in the existing `roles.go`, not a new model/table.

## Complexity Tracking

> No constitution violations — table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
