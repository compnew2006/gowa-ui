# Implementation Plan: Whatsmeow Integration

**Branch**: `001-whatsmeow-integration` | **Date**: 2026-02-17 | **Spec**: [spec.md](file:///Users/noiemany/Downloads/whatomate_GOWA/specs/001-whatsmeow-integration/spec.md)
**Input**: Feature specification from `/specs/001-whatsmeow-integration/spec.md`

## Summary

Replace Meta Cloud API dependency with whatsmeow (WhatsApp Web multi-device protocol) to enable zero-cost, QR-based WhatsApp connectivity. The approach introduces a `MessageProvider` interface behind which both the existing Meta adapter and a new whatsmeow adapter implement the same contract. Handler code references the interface only. A new `WhatsAppInstance` entity models per-device connections with a 5-state lifecycle. The existing `WhatsAppAccount` table and `pkg/whatsapp/` package remain untouched (Strangler Pattern).

## Technical Context

**Language/Version**: Go 1.24 (pinned in `go.mod`)
**Primary Dependencies**: Fastglue (HTTP), GORM (ORM), whatsmeow (new), logf (logging), fasthttp/websocket (real-time)
**Storage**: PostgreSQL (existing), whatsmeow sqlstore (new — shared DB)
**Testing**: Go `testing` + testify (backend), Playwright (frontend)
**Target Platform**: Linux server (Docker single binary)
**Project Type**: Web application (Go backend + Vue 3 frontend)
**Performance Goals**: <500ms message send latency (p95), <3s QR display, 5 concurrent instances per org
**Constraints**: Zero mandatory external deps (Principle VI), <500 lines per file (Principle VII)
**Scale/Scope**: 50 orgs, 5 instances/org, ~10K messages/day

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| # | Principle | Status | Evidence |
|:--|:----------|:-------|:---------|
| I | Adapter-First Architecture | ✅ PASS | New `MessageProvider` interface in `pkg/provider/`; both Meta and whatsmeow implement it |
| II | Zero-Regression / Strangler Pattern | ✅ PASS | `pkg/whatsapp/` untouched; new `pkg/whatsmeow/` created separately; old `whatsapp_accounts` table preserved |
| III | Test-First Verification | ✅ PASS | Contract tests for `MessageProvider` interface; integration tests per adapter |
| IV | Multi-Tenant Isolation | ✅ PASS | `WhatsAppInstance` scoped by `organization_id`; all queries filtered; whatsmeow sessions isolated per instance+org |
| V | Real-Time First | ✅ PASS | QR codes, status changes, incoming messages all via WebSocket; existing Hub pattern reused |
| VI | Single-Binary Simplicity | ✅ PASS | whatsmeow sqlstore shares existing PostgreSQL; no new external services required |
| VII | Modular File Structure | ✅ PASS | New files: `pkg/provider/interface.go`, `pkg/whatsmeow/adapter.go`, `pkg/whatsmeow/manager.go`, `internal/models/instance.go`, `internal/handlers/instances.go` — all under 500 lines |
| VIII | Surgical Impact Analysis | ✅ PASS | Blast radius documented per-component below |
| IX | Ralph Method | ✅ DEFERRED | Applied post-implementation |
| X | Skeptical Self-Review | ✅ PASS | See section below |

## 🛑 SKEPTICAL REVIEW (Self-Correction)

- **The Plan:** Extract a `MessageProvider` interface, build whatsmeow adapter behind it, create `WhatsAppInstance` model with 5-state lifecycle, route messages through provider-aware handlers.
- **The Critic:** "The `contacts` and `messages` tables currently FK on `WhatsAppAccount.Name` (a string). Adding a new `instance_id` UUID column creates a messy dual-FK situation. Queries will need `COALESCE(instance_id, NULL)` logic everywhere. This is technical debt."
- **The Defense/Fix:** Good point. The dual-FK is intentional as a *transition mechanism*, not permanent debt. The data migration task (US5/P3) populates `instance_id` for existing records. Once migration is complete and Meta accounts are deprecated for an org, a future cleanup task drops the string FK. This is documented in data-model.md migration plan. The alternative — rewriting all existing queries in one shot — is far riskier.

## Project Structure

### Documentation (this feature)

```text
specs/001-whatsmeow-integration/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research decisions
├── data-model.md        # Entity schema and migrations
├── quickstart.md        # Developer setup guide
├── contracts/
│   ├── instances-api.md # Instance CRUD + lifecycle API
│   └── messaging-api.md # Message send/receive API
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
whatomate/
├── cmd/server/main.go           # MODIFY: Wire provider + instance manager at startup
├── config.toml                  # MODIFY: Add [whatsapp] and [whatsmeow] sections
├── go.mod                       # MODIFY: Add whatsmeow dependency
├── internal/
│   ├── config/                  # MODIFY: Add WhatsmeowConfig struct
│   ├── handlers/
│   │   ├── instances.go         # NEW: Instance CRUD + lifecycle handlers
│   │   ├── config_handler.go    # NEW: /api/config endpoint
│   │   ├── notifications.go     # NEW: Notification list + dismiss
│   │   ├── messages.go          # MODIFY: Route through MessageProvider interface
│   │   └── webhook.go           # MODIFY: Guard Meta webhook when provider=whatsmeow
│   ├── models/
│   │   ├── instance.go          # NEW: WhatsAppInstance + InstanceNotification models
│   │   └── constants.go         # MODIFY: Add instance status constants
│   ├── middleware/               # NO CHANGE
│   ├── websocket/
│   │   └── messages.go          # MODIFY: Add new WS event types (qr_code, instance_*)
│   └── database/                # MODIFY: AutoMigrate new tables
├── pkg/
│   ├── provider/
│   │   └── interface.go         # NEW: MessageProvider interface definition
│   ├── whatsapp/                # NO CHANGE (Strangler Pattern)
│   ├── whatsmeow/
│   │   ├── adapter.go           # NEW: MessageProvider implementation
│   │   ├── manager.go           # NEW: Multi-instance connection manager
│   │   ├── queue.go             # NEW: Per-instance message queue with rate limiting
│   │   └── events.go            # NEW: whatsmeow event handler (incoming msg, receipts)
├── frontend/
│   ├── src/
│   │   ├── views/
│   │   │   └── InstancesView.vue    # NEW: Instance management page
│   │   ├── components/
│   │   │   ├── QRCodeModal.vue      # NEW: QR code display modal
│   │   │   ├── InstanceCard.vue     # NEW: Instance status card
│   │   │   └── HealthDashboard.vue  # NEW: Per-instance health metrics
│   │   ├── composables/
│   │   │   ├── useInstances.ts      # NEW: Instance API composable
│   │   │   └── useConfig.ts         # NEW: Config API composable
│   │   └── router/
│   │       └── index.ts             # MODIFY: Add /instances route
└── test/
    ├── contract/
    │   └── provider_test.go         # NEW: Interface compliance tests
    ├── integration/
    │   ├── instance_test.go         # NEW: Instance lifecycle tests
    │   └── messaging_test.go        # MODIFY: Test through MessageProvider
    └── unit/
        ├── queue_test.go            # NEW: Rate limiter + queue tests
        └── adapter_test.go          # NEW: whatsmeow adapter unit tests
```

**Structure Decision**: Web application (Go backend + Vue 3 frontend). All new backend code follows handler → service → repository layering. New packages `pkg/provider/` and `pkg/whatsmeow/` created. No existing packages modified in-place.

### Blast Radius Summary

| Component Modified | Files Affected |
|:-------------------|:---------------|
| `cmd/server/main.go` (wire provider) | Startup only — no downstream |
| `internal/handlers/messages.go` (route via interface) | `pkg/provider/interface.go`, `pkg/whatsmeow/adapter.go`, `pkg/whatsapp/client.go` (wrapping) |
| `internal/websocket/messages.go` (add event types) | `internal/websocket/hub.go` (broadcast, no changes needed) |
| `internal/models/constants.go` (add status consts) | `internal/models/instance.go` (new file, uses consts) |
| `internal/database/` (add AutoMigrate) | New tables only — existing tables untouched |
| `config.toml` + `internal/config/` | Additive — new sections, no breaking changes |

## Complexity Tracking

No constitution violations requiring justification. All changes follow established patterns.

| Decision | Rationale | Alternative Rejected |
|:---------|:----------|:---------------------|
| Dual FK (instance_id + whatsapp_account) | Transition safety — rollback without data loss | Single FK replacement (breaking change to all queries) |
| Per-instance goroutine for queue | Simple, no external deps | Redis queue (Principle VI violation) |
