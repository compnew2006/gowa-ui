# Implementation Plan: Chat Status, Claim & Collaboration

**Branch**: `001-chat-claim-collaboration` | **Date**: 2026-07-12 (updated post-clarification) | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/001-chat-claim-collaboration/spec.md`

---

## Summary

Implement a three-state conversation lifecycle (pending → open → closed) stored in `Contact.Metadata` JSONB, with a privacy gate hiding message content for unclaimed conversations, a claim action, a collaboration system allowing multiple agents to participate, a manager-only kick capability, owner-leave auto-close, and an auto-revert worker that releases inactive conversations back to pending. Two new granular permissions (`chat.assign:write`, `chat.collaborate:write`) are configurable from `/settings/roles`. The inactivity timeout is configurable from Settings → General via `Organization.Settings` JSONB. No database migrations — all data lives in existing JSONB fields.

---

## Technical Context

**Language/Version**: Go 1.25 (backend) + TypeScript 5.3 / Vue 3.4 (frontend)  
**Primary Dependencies**: fastglue/fasthttp, GORM, Redis, Pinia, vue-i18n, shadcn-vue  
**Storage**: PostgreSQL 17 (JSONB Metadata) + Redis 7 (permission cache)  
**Testing**: Go integration tests (testify + real Postgres/Redis) + Playwright E2E  
**Target Platform**: Linux server (single binary with embedded frontend)  
**Project Type**: Web (Go backend + embedded Vue SPA)  
**Performance Goals**: < 2s message-to-pending, < 5s claim flow, < 2s WebSocket latency  
**Constraints**: No migrations (JSONB only), backward-compatible with existing contacts  
**Scale/Scope**: ~5 new files, ~10 modified files, 4 new API endpoints, 4 new WS types, 1 background worker

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Single-Binary Architecture | ✅ Pass | No new binaries; worker is a goroutine in the existing server process |
| 2 | Fastglue + Fasthttp | ✅ Pass | All new handlers use `func (a *App) X(r *fastglue.Request) error` |
| 3 | Global Auth, Handler Permissions | ✅ Pass | `requireAuth(r, resource, action)` on all new endpoints |
| 4 | Multi-Tenancy | ✅ Pass | Worker queries are org-scoped; all queries filter by `organization_id` |
| 5 | Response Envelopes | ✅ Pass | `SendEnvelope` / `SendErrorEnvelope` on all responses |
| 6 | Explicit Response Builders | ✅ Pass | Extends `ContactResponse` via `buildContactResponse` |
| 7 | GORM AutoMigrate | ✅ Pass | No new models — only JSONB keys in existing Metadata |
| 8 | JSONB for Flexible Data | ✅ Pass | `chat_status`, `collaborators`, `chat_inactivity_timeout_hours` all in JSONB |
| 9 | WebSocket Typed Messages | ✅ Pass | 4 new typed constants + explicit frontend field mapping |
| 10 | Provider Abstraction | ✅ Pass | No provider changes; pending state set in `processIncomingMessageFull` |
| 11 | Vue 3 `<script setup>` + Pinia | ✅ Pass | New computed/actions added to existing setup stores |
| 12 | Cookie Auth + CSRF | ✅ Pass | No auth changes; uses existing cookie/session infrastructure |
| 13 | shadcn-vue + Tailwind | ✅ Pass | Dark-first CSS, `light:` overrides for claim/collab UI |
| 14 | i18n `$t()` | ✅ Pass | All new UI text via `$t()` in 5 locales |
| 15 | Testing | ✅ Pass | Go integration tests + manual E2E per quickstart |
| 16 | Structured Logging | ✅ Pass | `a.Log.Error/Debug` in all new handlers + worker |
| 17 | Audit + Cache | ✅ Pass | `logAudit` on claim/join/leave/kick/revert mutations |
| 18 | TOML + Env Config | ✅ Pass | No config.toml changes; timeout in `Organization.Settings` JSONB |
| 19 | Conventional Commits | ✅ Pass | `feat(chat): add claim, collaboration, and auto-revert system` |

**Gate Result**: ✅ ALL PASS — no violations, no complexity tracking needed.

---

## Project Structure

### Documentation (this feature)

```text
specs/001-chat-claim-collaboration/
├── spec.md              # Feature specification (updated with 4 clarifications)
├── plan.md              # This file (updated post-clarification)
├── research.md          # 10 technical decisions (updated with R8, R9, R10)
├── data-model.md        # Entities, state transitions (updated with auto-revert, kick)
├── quickstart.md        # 10-step testing guide (updated with kick, auto-close, auto-revert)
├── contracts/
│   └── api.md           # 5 endpoints + 4 WS types (updated with kick, chat_reverted)
└── checklists/
    └── requirements.md  # Spec quality validation
```

### Source Code (repository root)

```text
# ─── NEW FILES ───
internal/models/
└── chat_status.go                      # ChatStatus type + Collaborator + Contact helpers

internal/handlers/
└── chat_lifecycle.go                   # ClaimChat + JoinChat + LeaveChat + RemoveCollaborator
└── chat_inactivity_worker.go           # Background worker (auto-revert open→pending)

# ─── MODIFIED FILES (Backend) ───
internal/models/roles.go                # + ResourceChatCollaborate, + DefaultPermissions,
                                        #   + agent gets chat.assign, + manager gets chat.collaborate
internal/handlers/contacts.go           # + ContactResponse fields, + buildContactResponse,
                                        #   + GetMessages privacy guard, + AssignContact status
internal/handlers/chatbot_processor.go  # + Set chat_status=pending after GetOrCreateContact
internal/websocket/messages.go          # + TypeChatClaimed, TypeCollaboratorJoined,
                                        #   TypeCollaboratorLeft, TypeChatReverted
cmd/whatomate/main.go                   # + 4 routes + start inactivity worker goroutine

# ─── MODIFIED FILES (Frontend) ───
frontend/src/stores/contacts.ts         # + types, + computed, + claimChat/joinChat/leaveChat,
                                        #   + 403 handling
frontend/src/services/websocket.ts      # + 4 switch cases for new WS events
frontend/src/views/chat/ChatView.vue    # + Claim screen, + Join screen, + Collaborators bar,
                                        #   + Kick button (managers), + Leave/close UI
frontend/src/views/settings/SettingsView.vue # + chat_inactivity_timeout_hours field
frontend/src/i18n/locales/en.json       # + i18n keys
frontend/src/i18n/locales/ar.json       # + Arabic translations
frontend/src/i18n/locales/es.json       # + Spanish translations
frontend/src/i18n/locales/hi.json       # + Hindi translations
frontend/src/i18n/locales/ta.json       # + Tamil translations
```

**Structure Decision**: Extends existing Go `internal/handlers/` + `internal/models/` and Vue `src/stores/` + `src/views/`. No new directories. Worker is a goroutine in the server process (Constitution Principle 1 — single binary).

---

## Complexity Tracking

> No violations — table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |

---

## Implementation Phases

### Phase A: Backend Models & Permissions

| Task | File | Details |
|------|------|---------|
| Create ChatStatus type | `internal/models/chat_status.go` (NEW) | `ChatStatus` type, `EffectiveStatus()`, `SetStatus()`, `Collaborator` struct, collaborator CRUD helpers |
| Add permission resource | `internal/models/roles.go:72` | `ResourceChatCollaborate = "chat.collaborate"` |
| Add default permission | `internal/models/roles.go:169` | `{Resource: ResourceChatCollaborate, Action: ActionWrite, ...}` |
| Update agent role | `internal/models/roles.go:296` | Add `"chat.assign:write"` |
| Update manager role | `internal/models/roles.go:268` | Add `"chat.collaborate:write"` |

### Phase B: Backend Handlers (Claim + Join + Leave + Kick)

| Task | File | Details |
|------|------|---------|
| ClaimChat handler | `internal/handlers/chat_lifecycle.go` (NEW) | `requireAuth(chat.assign, write)` + 4 guards + assign + system msg + WS |
| JoinChat handler | `internal/handlers/chat_lifecycle.go` | `requireAuth(chat.collaborate, write)` + add collaborator + WS |
| LeaveChat handler | `internal/handlers/chat_lifecycle.go` | Self-leave: remove collaborator. Owner-leave: if last participant → close; if collabs remain → clear owner only |
| RemoveCollaborator handler | `internal/handlers/chat_lifecycle.go` | `requireAuth(chat.collaborate, write)` + remove target from collaborators. Reject if target is owner |
| createSystemMessage helper | `internal/handlers/chat_lifecycle.go` | Shared helper for system message creation |
| WS message constants | `internal/websocket/messages.go:30` | `TypeChatClaimed`, `TypeCollaboratorJoined`, `TypeCollaboratorLeft`, `TypeChatReverted` |
| Register routes | `cmd/whatomate/main.go:687` | `PUT /claim`, `POST /join`, `DELETE /join`, `DELETE /collaborators/{user_id}` |

### Phase C: Backend Integration (Privacy Guard + Pending + Open)

| Task | File | Details |
|------|------|---------|
| Privacy guard in GetMessages | `internal/handlers/contacts.go:276` | Access check: owner/collaborator/contacts:read/collaborate:write → 403 if denied + pending |
| Extend ContactResponse | `internal/handlers/contacts.go:48` | `ChatStatus`, `Collaborators` fields |
| Extend buildContactResponse | `internal/handlers/contacts.go:1654` | Populate from Metadata |
| Set pending on incoming | `internal/handlers/chatbot_processor.go` | After GetOrCreateContact, set pending if unassigned |
| Set open on assignment | `internal/handlers/contacts.go:1204` | AssignContact also sets chat_status=open |

### Phase D: Auto-Revert Worker

| Task | File | Details |
|------|------|---------|
| Create worker | `internal/handlers/chat_inactivity_worker.go` (NEW) | `StartChatInactivityWorker(ctx)` — ticker every 5 min |
| Worker logic | `internal/handlers/chat_inactivity_worker.go` | Per-org: read timeout from `Settings`, query open+inactive contacts, revert to pending, post system msg, broadcast WS |
| Start worker | `cmd/whatomate/main.go` | `go app.StartChatInactivityWorker(ctx)` alongside other workers |
| Settings field | `frontend/src/views/settings/SettingsView.vue` | Add `chat_inactivity_timeout_hours` input (number, default 24) |

### Phase E: Frontend (Claim + Join + Collaborators + Kick)

| Task | File | Details |
|------|------|---------|
| Types + computed + actions | `frontend/src/stores/contacts.ts` | Contact interface, isPendingClaim, claimChat/joinChat/leaveChat/removeCollaborator |
| 403 handling in fetchMessages | `frontend/src/stores/contacts.ts` | Catch `chat_not_claimed`, store pendingMessageCount |
| WS handlers | `frontend/src/services/websocket.ts` | 4 new switch cases: chat_claimed, collaborator_joined, collaborator_left, chat_reverted |
| Claim + Join screen UI | `frontend/src/views/chat/ChatView.vue` | Lock icon, pending count, buttons (v-if conditionals) |
| Collaborators bar | `frontend/src/views/chat/ChatView.vue` | Header: avatars + names + Remove button (managers only, via `hasPermission('chat.collaborate', 'write')`) |
| Leave button | `frontend/src/views/chat/ChatView.vue` | Self-leave for collaborators, owner-leave with close confirmation |
| i18n keys × 5 locales | `frontend/src/i18n/locales/*.json` | All new strings including kick/remove/auto-revert messages |

---

## Key Design Decisions

1. **No GORM columns** — `chat_status`, `collaborators`, and `chat_inactivity_timeout_hours` all in JSONB (Principle 8)
2. **Privacy guard in GetMessages** — 403 with pending count, frontend shows claim screen
3. **System messages = regular Message records** — reuse pipeline, flagged via `metadata.is_system_message`
4. **Two distinct permissions** — `chat.assign:write` (claim) vs `chat.collaborate:write` (join + kick)
5. **Owner leave = close** — when last participant leaves, conversation closes (not orphaned)
6. **Manager kick** — `DELETE /collaborators/{user_id}` reuses `chat.collaborate:write` permission
7. **Auto-revert worker** — goroutine ticker (5 min), reads org settings, reverts open→pending after timeout
8. **Silent revert** — no pre-warning notification; system message posted after the fact
9. **Backward compatible** — absent `chat_status` defaults to `open`, existing contacts unaffected
