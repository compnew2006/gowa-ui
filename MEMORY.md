# MEMORY.md

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
