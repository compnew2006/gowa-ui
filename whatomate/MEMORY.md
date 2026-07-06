# MEMORY.md

## 2026-05-26 23:37

### Work Summary
- Implemented the first backend slice of Customer Agent Selection for WhatsMeow.
- Added additive database models for routing settings, customer-visible participants, non-agent options, delayed selection sessions, and immutable audit events.
- Added settings, participant, option, preview, session, cancel, and audit APIs under `/api/agent-selection`.
- Added a delayed processor that sends a text menu after the configured delay only if the chat is still pending and unassigned.
- Added a WhatsMeow inbound hook that handles active menu replies before normal chatbot routing and schedules delayed sessions for eligible pending chats.

### Architectural Decisions
- Preserved existing chat lifecycle behavior by reusing `chatAssignmentUpdates`, existing transfer helpers, existing WhatsMeow send path, and existing websocket lifecycle broadcasts.
- Kept the feature disabled by default and scoped to WhatsMeow to avoid changing Meta provider behavior.
- Stored customer-visible agent labels in `agent_selection_participants` so admins can control the public list without changing user identity.
- Added `customer_selection` as a transfer source for team/queue choices.

### Current Project State
- Go verification passes:
  - `go test ./internal/handlers -run 'TestSelectedRenderedOption|TestSessionHasProcessedInbound|TestNormalizeStringArray'`
  - `go test ./cmd/whatomate ./internal/models ./internal/database ./internal/handlers -run TestNonExistent`
  - `go test ./...`
- Frontend UI for the settings page is not implemented yet; the backend API and preview endpoint are ready for it.

## 2026-05-27 00:22

### Work Summary
- Added the frontend Customer Routing settings page at `/settings/agent-selection`.
- Added `agentSelectionService` and `useAgentSelectionStore` to centralize Customer Agent Selection API calls.
- Added navigation and route entries gated by `agent_selection` permission.
- Added localized navigation labels for English, Arabic, and Spanish.
- The page now supports settings save, dynamic WhatsMeow menu preview, adding/removing customer-visible agents, adding/removing team/queue options, viewing sessions, cancelling active sessions, and browsing audit events.

### Architectural Decisions
- Kept Customer Routing UI as a standalone settings page instead of expanding the existing chatbot settings surface.
- Reused existing users and teams stores for selectable agents and teams.
- Kept API response handling inside a dedicated Pinia store to avoid direct endpoint coupling throughout the view.

### Current Project State
- Frontend build passes: `cd frontend && npm run build`.
- Targeted eslint for Customer Routing files passes.
- Frontend `npm run typecheck` still fails on pre-existing project-wide TypeScript errors outside the new files.
- Backend tests still pass: `go test ./...`.

## 2026-04-10 22:10

### Work Summary
- Implemented a zero-disk inbound WhatsMeow media pipeline that streams decrypted media directly into S3-compatible object storage using `io.Pipe`, with no temp files and no `[]byte` media buffering in the application layer.
- Added native WhatsApp `FileSha256` deduplication through the new `MediaAsset` model and linked inbound `Message` rows via `MediaAssetID`.
- Reworked inbound async media recovery to reuse the same streaming/object-storage path and added a retention worker that marks expired media, appends the retention note, and deletes shared objects only when all referencing messages have expired.

### Architectural Decisions
- Introduced a narrow `ObjectStorage` seam for inbound media and media serving so the WhatsMeow pipeline, HTTP handler, and retention worker share one storage contract without disturbing existing local-storage flows outside this scope.
- Kept the public message/media API stable by continuing to populate `media_url`, `media_mime_type`, and `media_filename`, while changing the backing implementation to object storage plus `/api/media/{message_id}` streaming.
- Soft-deleted fully expired `media_assets` rows and taught dedup recovery to restore them on reappearance, avoiding unique-hash collisions after retention-driven physical object deletion.

### Current Project State
- Focused Go verification passes:
  - `go test ./pkg/whatsmeow -run 'TestMediaService_|TestPersistParsedMessage_' -count=1`
  - `go test ./internal/handlers -run 'TestMediaRetentionWorker_|TestServeMedia_StreamsFromObjectStorage|TestSensitiveRBAC_ServeMedia_ForbiddenWithoutPermission' -count=1`
- Browser verification was completed with Chrome DevTools against a temporary `/api/media/...` contract harness, confirming streamed `200` responses and `Content-Type` behavior.
- Full repo-wide `go test ./... -run '^$'` remains blocked by pre-existing unrelated issues:
  - `internal/frontend/embed.go` expects missing `dist` assets
  - `tmp_encrypt.go` and `tmp_arabic.go` both define `main`

## 2026-04-02 19:08

### Work Summary
- Replaced the public pricing landing route group with a marketing-sidecar handoff view and removed the embedded pricing/offers page implementation from the frontend.
- Added a reusable redirect helper plus targeted unit coverage for sidecar URL construction.
- Generalized backend lead-request source validation so the future sidecar can keep using the monolith lead inbox without a pricing-only payload contract.

### Architectural Decisions
- Kept `/pricing`, `/plans`, and `/offer` as stable public entry URLs, but transferred ownership to a redirect seam instead of leaving sales content inside the main SPA.
- Preserved `POST /api/public/lead-requests` and the authenticated `/settings/lead-requests` workflow in the monolith for the first migration phase.
- Used `VITE_PUBLIC_MARKETING_BASE_URL` as the explicit extension point so the sidecar can live on an external origin or a same-origin prefixed path.

### Current Project State
- Frontend unit test passes: `npx vitest run src/lib/marketing-redirect.test.ts`
- Frontend build passes: `npm run build`
- Go production build passes: `go build ./cmd/whatomate`
- `npm run typecheck` is still failing due to pre-existing frontend typing issues in contacts/auth/chatbot modules unrelated to this change.
- `go test ./internal/handlers -run 'TestApp_(CreatePublicLeadRequest|ListLeadRequests|UpdateLeadRequestStatus)$' -count=1` is still blocked by the pre-existing `testutil.MockQueue` / `EnqueueContactRepair` compile failure in `internal/handlers/campaigns_test.go`.

## 2026-03-17 20:27

### Work Summary
- Hardened the collaborator invite API to reject inactive users and reject duplicate invites for already invited or accepted collaborators.
- Added targeted backend regression tests for collaborator invite eligibility, instance restriction enforcement, declined re-invite, and self-removal flows.
- Added assignment permission regression tests plus a shared frontend instance-access utility with unit coverage.
- Added ACP M4 milestone tracking for chat collaboration and assignment permissions and marked current task state accurately.

### Architectural Decisions
- Preserved the current policy that invited collaborators may still see chats immediately; only invite eligibility and state-corruption gaps were fixed.
- Replaced duplicated frontend instance-filter helpers with a shared utility so chat, collaborator, and transfer assignment flows use the same rule implementation.
- Kept assignment permission implementation tracking separate from regression coverage so existing production behavior could be marked complete without hiding remaining tests.

### Current Project State
- Handler package compiles: `go test ./internal/handlers -run '^$'`.
- New collaborator handler tests are present but DB-backed execution was skipped locally because `TEST_DATABASE_URL` and `TEST_REDIS_URL` are not configured.

## 2026-03-17 00:00

### Work Summary
- Fixed chat assignment authorization to honor `chat.assign:write` and enforce assignee instance access.
- Filtered assignment lists by allowed instance IDs and surfaced transfer `instance_id` for UI filtering.
- Added a design spec for assign-to-agent permissions and instance visibility.

### Architectural Decisions
- Reused send restriction instance allowlist for assignment visibility checks to keep access enforcement consistent across chat and transfers.

### Current Project State
- Backend handlers tests (targeted) pass: `go test ./internal/handlers -run AssignAgentTransfer -count=1`.

## 2026-03-04 01:47

### Work Summary
- Completed the Whatomate 7-enhancement bundle across backend, frontend, unit tests, and E2E tests.
- Added user-level unclaimed chat controls with split view/send policy and backend normalization (`send => view`).
- Moved Activity Logs under Settings route with legacy redirect and manager/admin access handling.
- Added `instance_id` filtering across agent analytics summary/details/comparison and ratings export.
- Added assignment system-message emission for manager/admin assignment actions.
- Implemented combined-chat multi-instance indicator and selected-instance outbound routing.
- Moved Strict Sending Restrictions control from General Settings to Users page.

### Architectural Decisions
- Kept claim access policy user-scoped in send restrictions payload (`allow_unclaimed_chat_view`, `allow_unclaimed_chat_send`).
- Implemented reusable backend helpers for policy and analytics instance filter parsing:
  - `internal/handlers/chat_access_policy.go`
  - `internal/handlers/analytics_instance_filter.go`
- Preserved backward compatibility for Activity Logs via `/activity-logs` -> `/settings/activity-logs` redirect.
- For unified multi-instance chats, tab selection resolves source contact/instance and explicitly passes `instance_id` on outbound actions.

### Current Project State
- Backend tests pass (`go test ./...`).
- Frontend unit tests pass (`npm run test:unit`).
- Frontend build passes (`npm run build`).
- Feature-scoped E2E suites for this bundle pass (activity, analytics, chat routing/system messages, unified sidebar, users restrictions controls).
