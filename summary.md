# ARCH-08 + ARCH-09 — ContactInfoPanel Split & Dead WebSocket Code Cleanup

**Date**: 2026-05-09
**Branch**: `agent/arch-08-09-contact-panel-split-ws-cleanup`
**Commit**: `f797131`

## Summary

Split the 963-line `ContactInfoPanel.vue` mega-component into 4 focused sub-components (parent reduced to 271 lines). Removed dead WebSocket unauthenticated client path (function, types, tests, and frontend constant).

## ARCH-08: ContactInfoPanel Decomposition

**Before**: 1 file, 963 lines
**After**: 5 files, 1010 total lines (parent: 271 lines)

| File | Lines | Purpose |
|------|-------|---------|
| `ContactInfoPanel.vue` | 271 | Shell: header, resize handle, contact avatar, delete actions |
| `ContactTagsPanel.vue` | 152 | Tag display, add/remove via popover + command palette |
| `ContactCollaboratorsPanel.vue` | 319 | Collaborator list, accept/decline/remove, invite dialog |
| `ContactSessionDataPanel.vue` | 224 | Dynamic session data with collapsible sections |
| `ContactMetadataPanel.vue` | 44 | Contact metadata primitives and object sections |

### Props Interface

- **ContactTagsPanel**: `contactId`, `contactTags`, `canEditTags` → emits `tagsUpdated`
- **ContactCollaboratorsPanel**: `contactId`, `instanceId` (self-contained WS subscriptions)
- **ContactSessionDataPanel**: `sessionData`
- **ContactMetadataPanel**: `metadata`

### Backward Compatibility

`ChatView.vue` imports `ContactInfoPanel` unchanged — same props (`contact`, `sessionData`), same events (`close`, `tagsUpdated`, `deleted`).

## ARCH-09: Dead WebSocket Code Removal

### Analysis Findings

Contrary to the task description, most WS types ARE used in production:
- `TypeAgentTransfer/Resume/Assign` — used in `agent_transfers.go`
- `TypePermissionsUpdated` — used in `cache.go`
- `StatusUpdatePayload` — used in `whatsmeow/events.go` and `stubs.go`

The ONLY truly dead code was the **unauthenticated client path**:
- `NewUnauthenticatedClient()` — never called in production, only in tests
- `AuthenticateFn` type — only for the dead unauthenticated path
- `handleAuthMessage()` — only for message-based auth (superseded by HTTP handshake auth)
- `AuthPayload` type — only used by `handleAuthMessage`
- `TypeAuth` constant — only used by `handleAuthMessage`
- `authTimeout` constant — only for the auth timeout in ReadPump

### What was removed

| File | Change |
|------|--------|
| `internal/websocket/client.go` | Removed `NewUnauthenticatedClient`, `AuthenticateFn`, `authFn` field, `handleAuthMessage`, `authTimeout`, simplified `ReadPump` |
| `internal/websocket/messages.go` | Removed `TypeAuth` constant, `AuthPayload` type |
| `internal/websocket/export_test.go` | Removed `ClientHandleAuthMessage` export |
| `internal/websocket/websocket_test.go` | Removed 12 dead test functions + `successAuthFn`/`failAuthFn` helpers |
| `frontend/src/services/websocket.ts` | Removed unused `WS_TYPE_AUTH` constant |

## Files Modified/Created

| File | Action |
|------|--------|
| `frontend/src/components/chat/ContactTagsPanel.vue` | **Created** |
| `frontend/src/components/chat/ContactCollaboratorsPanel.vue` | **Created** |
| `frontend/src/components/chat/ContactSessionDataPanel.vue` | **Created** |
| `frontend/src/components/chat/ContactMetadataPanel.vue` | **Created** |
| `frontend/src/components/chat/ContactInfoPanel.vue` | **Modified** (963→271 lines) |
| `internal/websocket/client.go` | **Modified** (removed dead auth path) |
| `internal/websocket/messages.go` | **Modified** (removed `TypeAuth`, `AuthPayload`) |
| `internal/websocket/export_test.go` | **Modified** (removed `ClientHandleAuthMessage`) |
| `internal/websocket/websocket_test.go` | **Modified** (removed 12 dead tests) |
| `frontend/src/services/websocket.ts` | **Modified** (removed `WS_TYPE_AUTH`) |

## Verification

- `go build ./cmd/whatomate/... ./internal/websocket/... ./pkg/...` — **passes** (0 errors)
- `go vet ./internal/websocket/...` — **passes** (0 warnings)
- `npm run build` (frontend) — **passes** (production build succeeds)
- `npm run typecheck` — pre-existing errors only (none from our changes)
- Net change: **862 insertions, 1161 deletions** (−299 lines)

## Key Design Decisions

1. **Tags fetch stays in parent**: `tagsStore.fetchTags()` remains in `ContactInfoPanel` on mount so it's fetched once and shared. The `ContactTagsPanel` reads from the store.
2. **Collaborators are self-contained**: `ContactCollaboratorsPanel` manages its own WS subscriptions, fetch, and invite dialog — no prop drilling of collaborator state.
3. **SessionDataPanel owns collapsed state**: The `collapsedSections` ref moved entirely into the session data panel — it's only relevant there.
4. **Kept `authenticated` guard in WritePump**: Even though all production clients are pre-authenticated, the `if !c.authenticated` check in `WritePump` is retained as defense-in-depth for test clients created with `uuid.Nil`.

## Known Limitations

- Cannot run full Go test suite (requires PostgreSQL 17 + Redis 7)
- `WritePump` still has the `authenticated` check — could be removed in a future pass for full simplification
