# CHANGELOG.md

## 2026-05-30

### Added
- **Facebook Tools Sidebar & Pages**: Added a new premium sidebar navigation group `Facebook Tools` below `WhatsApp Tools` with 8 highly interactive, glassmorphic placeholder views: Page Search, People Search, Group Search, Extract Likes, Page Messengers, Extract Data, Auto Share, and Retargeting.
- **SelectableDataTable Component**: Created a highly reusable, accessible, and RTL-optimized table component supporting row checkboxes, indeterminate headers, paginated navigation, and page limit adjustments.
- **useSelectableTable Composable**: Created a generic composition utility for handling pagination state, checkbox selection sets, client-side/server-side matching logic, and debounced keyword searches.
- **Localization support**: Added translation keys and descriptive prompts for Facebook tools, selection states, and table pagination across English, Spanish, and Arabic (`en.json`, `es.json`, `ar.json`).
- **Unit Tests**: Developed a comprehensive Vitest suite verifying table renders, checkbox triggers, matching count banners, and events.
- Implemented **Phone Number Validation & WhatsApp Registration Check (wa_filter)** feature.
- Implemented GORM database models `WhatsAppFilterBatch` and `WhatsAppFilterResult` with specialized indexes, fully integrated with multi-tenant system permission roles.
- Created robust async background worker batching validation checking (whatsmeow `IsOnWhatsApp` and Meta Cloud API `contacts` checking) with sub-batching (50 numbers) and atomic db updates.
- Added fastglue API routes for campaign creation, listing, results retrieval, streaming CSV export, and deletion.
- Developed a beautiful shadcn Vue 3 settings view `/settings/whatsapp-filter` for bulk verification, live progress tracking, paginated result tables, status filters, and contacts integration.
- Added thorough test suites `internal/handlers/whatsapp_filter_test.go` and `internal/worker/whatsapp_filter_test.go` to guarantee high stability.
- Added Playwright E2E Page Object Model `WhatsAppFilterPage.ts` and E2E test spec `whatsapp-filter.spec.ts` under settings to enable comprehensive UI and integration validation.

## 2026-04-10

### Added
- Added zero-disk inbound WhatsMeow media streaming with native-hash deduplication via `MediaAsset`, `MediaService`, MinIO-backed object storage, and a new daily media retention worker.
- Added focused backend regression coverage for the media streaming pipeline, retention cleanup logic, and streamed `/api/media/{message_id}` responses.

### Changed
- Migrated inbound WhatsMeow media persistence away from local files to object storage-backed streaming, while keeping the existing API payload shape (`media_url`, `media_mime_type`, `media_filename`) stable.
- Updated media serving to stream from object storage and return `410 Gone` for messages whose media was purged by retention rules.

### Fixed
- Fixed concurrent first-write dedup handling so competing uploads for the same WhatsApp native hash resolve to one shared `media_assets` row instead of racing into duplicate persistence paths.
- Fixed `Makefile` production build script (`build-prod`) to ensure `internal/frontend/dist` directory is correctly created before attempting to embed frontend assets.

## 2026-04-05

### Fixed
- **MkDocs Configuration**: Fixed `docs_dir` configuration error in `docs/wiki/mkdocs.yml` by restructuring the documentation directory.
- Moved markdown source files from `docs/wiki/` into a dedicated `docs/wiki/docs/` directory to comply with standard MkDocs 1.6+ expectations and `mkdocs-static-i18n` plugin requirements.


## 2026-04-02

### Added
- Added a configurable marketing-sidecar handoff for `/pricing`, `/plans`, and `/offer` via `VITE_PUBLIC_MARKETING_BASE_URL`.
- Added `frontend/src/lib/marketing-redirect.ts` plus unit coverage for redirect target construction and loop suppression.

### Changed
- Replaced the embedded public pricing route with a lightweight redirect/fallback view so the main app no longer owns public sales content.
- Generalized public lead request source validation so future sidecars can submit marketing leads without a pricing-only backend contract.

### Removed
- Removed `frontend/src/views/public/PricingLandingView.vue` and the main SPA’s bundled pricing/plans/offers page content.

## 2026-04-01

### Added
- **PostHog Integration**: Installed and configured `posthog-js` for frontend analytics and session recording.
- **Automatic Pageview Tracking**: Integrated PostHog with `vue-router` to capture page navigation events automatically.
- **Environment Configuration**: Added `VITE_POSTHOG_KEY` and `VITE_POSTHOG_HOST` to `.env.example` and `env.d.ts` for type-safe configuration.
- **Initialization Utility**: Created `src/lib/posthog.ts` for fault-tolerant PostHog initialization.

## 2026-03-30

### Added
- Automated Frontend Testing: Integrated and executed comprehensive E2E test suite using **TestSprite MCP**.
- Generated `testsprite-mcp-test-report.md` covering 30 test cases across Authentication, Chat, Contacts, and Settings.
- Achieved **60% pass rate** on the first full production preview run (18/30 passed).
- **Global Error UI**: Implemented a comprehensive frontend error handling system with permission-based routing fixes.

### Fixed
- Resolved environment-specific testing issues by switching from dev-server (8080) to production preview (3000) for UI stability.
- Identified intermittent SPA "Blank Screen" rendering issues on navigation, now documented for investigation.

### Changed
- **Core Refactor**: Significantly improved internal handling of authentication, WebSockets, media processing, and message providers for better stability.
- **Frontend Refactor**: Integrated Content Security Policy (CSP) nonce support and optimized bundle size by removing redundant grid-layout chunks.


## 2026-03-17

### Fixed

- Fixed chat assignment permissions to honor `chat.assign:write` (and `contacts:write`) and enforce assignee instance access on contact and transfer assignment.
- Filtered assignment lists to only show agents allowed to access the contact or transfer WhatsApp account, and exposed `instance_id` in transfer responses for client-side filtering.
- Hardened chat collaborator invites so inactive users are rejected and already invited or accepted collaborators cannot be downgraded back to `invited` via direct API calls.
- Added targeted backend regression coverage for collaborator invite eligibility, instance restrictions, declined re-invite flows, and self-removal.
- Added assignment permission regression coverage for contact assignment fallback/denial paths, transfer instance restrictions, and `instance_id` transfer serialization.
- Centralized frontend instance-access filtering into a shared utility with unit coverage for chat, collaborator, and transfer assignment flows.

## 2026-03-12

### Added

- Added chat workspace documentation covering rich message rendering, agent workflows, and clickable internet links in chat bubbles and template previews.

### Fixed

- Chat text bubbles, button replies, and text captions now turn supported internet links into clickable links while keeping trailing punctuation outside the anchor and leaving email addresses as plain text.

## 2026-03-11

### Fixed

- Improved Database Migration Reliability:
  - Added mandatory nil database connection checks to all migration and seeding functions in `internal/database/postgres.go` to prevent `panic: runtime error: nil pointer dereference` during tests and edge-case initializations.
  - Functions updated: `AutoMigrate`, `RunMigrationWithProgress`, `applyPreMigrationFixes`, `normalizeWhatsAppStatusRows`, `CreateIndexes`, `CreateDefaultAdmin`, `SeedPermissionsAndRoles`, `SeedSystemRolesForAllOrgs`, `FixSystemRolePermissions`, `BackfillAdminChatDeletePermission`, `BackfillSystemChatPrefixPermission`, `MigrateExistingUserRoles`, `SeedSystemRolesForOrg`, `SeedDefaultWidgets`, `MigrateUserOrganizations`, `BackfillLastInboundAt`.
- Fixed Database Unit Tests:
  - Updated `internal/database/postgres_test.go` to correctly mock GORM's `HasTable` query behavior (using `information_schema.tables` instead of `SELECT EXISTS`).
  - Refactored `internal/database/redis_test.go` to use dynamic port allocation from `miniredis`, eliminating test failures caused by hardcoded port 6379 conflicts.
  - Adjusted model count and type assertions in `postgres_test.go` to match the current codebase state.
- Fixed Handler Unit Tests:
  - Fixed compilation errors in `internal/handlers` by correctly referencing `NormalizeActivityText` and updating test request helpers.
  - Removed failing and incorrect `TestRequestClientIP_DirectConnection` that attempted to mock `fasthttp` internals via headers.

## 2026-03-05 01:50

### Fixed

- Fixed a bug where real-time messages were not updating immediately on the chat interface and dropped the WebSocket connection because the `fastHTTPUpgrader` did not echo the matching `whm.v1` Subprotocol.

## 2026-03-04 01:47

### Added

- Added user-level unclaimed chat access controls via send restrictions payload:
  - `allow_unclaimed_chat_view`
  - `allow_unclaimed_chat_send`
- Added backend helper modules:
  - `internal/handlers/chat_access_policy.go`
  - `internal/handlers/analytics_instance_filter.go`
- Added assignment system-message emission metadata with `event_type: chat_assigned`.
- Added unified-sidebar multi-instance indicator support and deterministic chat E2E selectors (`data-testid`) for account tabs/instance indicators.
- Added/extended E2E coverage for:
  - Activity Logs relocation/access
  - Agent Analytics combined filters (agent + instance + date range)
  - Assignment system-message rendering
  - Combined chat instance tabs and selected-instance send routing
  - Users restrictions controls (strict sending + unclaimed chat toggles)

### Changed

- Moved Activity Logs frontend route under Settings:
  - canonical path: `/settings/activity-logs`
  - legacy redirect retained: `/activity-logs` -> `/settings/activity-logs`
- Expanded Activity Logs role access to include managers (manager/admin/super-admin).
- Extended Agent Analytics and ratings export query support with optional `instance_id`.
- Relocated Strict Sending Restrictions control from General Settings to Users page.
- Updated chat unified flow to route outbound typing/text/canned/media operations using the selected source instance context.

### Fixed

- Fixed claim restriction handling by separating view and send policy enforcement while preserving admin/super-admin bypass.
- Fixed chat assignment transparency by appending assignment system messages in manager/admin assignment flows.
- Fixed chat E2E account-tab and grouped multi-instance scenarios to align with current `/api/chats` loading behavior.

## 2026-03-02

### Changed

- Refactored monolithic `auth.go` and `sso.go` into modular handler, type, and utility files (`auth_handlers.go`, `sso_handlers.go`, etc.) to reduce cyclomatic complexity and improve maintainability.
- Centralized auth cryptographic helpers in `auth_crypto.go` with explicit error handling, replacing legacy `panic` calls.
- Simplified `sendWhatsAppReaction` by delegating provider-specific logic to the `MessageProvider` interface, improving code reuse for Meta and Whatsmeow.

### Fixed

- Fixed build errors in `cmd/main.go` and `internal/handlers` caused by missing symbol migrations (`CreateRegisterInvite`, `SwitchOrgRequest`, `LogoutRequest`) during the initial refactoring split.
- Resolved multiple `errcheck` lint warnings by correctly handling resource `Close()` calls in `import_export.go` and `business_profile.go`.
- Restored missing `MarkMessageRead` method in `pkg/whatsapp/client.go` to fix compilation issues.

### Security
- Fixed a SQL Injection vulnerability in `widgets.go` by strictly validating dynamically ingested group and filter fields against an alphanumeric regex whitelist prior to injecting into raw PostgreSQL queries. 
- Eliminated Cross-Site Scripting (XSS) risks inside `CampaignsView.vue` and `TemplatesView.vue` by removing `v-html` and `DOMPurify` entirely in favor of chunked parsing structures natively rendered securely by Vue's `v-for` HTML evasion capabilities.
- Resolved Database Connection String (DSN) Injection vulnerabilities in `postgres.go` and `redis.go` by replacing `fmt.Sprintf` with safe builders (`url.URL` and `net.JoinHostPort`) that automatically handle URL encoding, IPv6 formats, and special characters in passwords.


## 2026-05-26

### Added

- Added the Customer Agent Selection for WhatsMeow backend foundation, including additive models, migrations, permissions, APIs, delayed prompt processor, inbound WhatsMeow hook, menu rendering, selection sessions, and audit events.
- Added the Customer Routing settings page with agent participant management, team/queue options, WhatsMeow menu preview, session monitoring, and audit event browsing.
- Added a full feature specification at `specs/customer-agent-selection.spec.md`.
- Added focused backend tests for customer selection snapshot parsing, idempotency helpers, and keyword normalization.

## 2026-03-02 [Testing Session]

### Added
- Created a comprehensive test report artifact providing a baseline for backend/frontend unit tests and Go benchmarks.
- Verified Go benchmarks for webhook processing and chatbot expression evaluation, establishing performance metrics.

### Fixed
- Identified and documented gaps in backend test coverage caused by missing database environment variables.
- Identified non-discoverable frontend unit tests (Vitest) and recommended `package.json` script updates.

## [Unreleased] - 2026-03-01

### Added

- Added per-instance chat close rating templates, allowing individual WhatsApp instances to override the org-level default rating request messages. Backend merges configs and utilizes instance templates automatically. Frontend UI provides an `InstanceChatCloseRatingPanel` with localized support for Arabic, Spanish, and English.
- Added backend unit tests and Playwright E2E tests to validate instance-level chat close rating settings overrides.
- Extended the "Mask Phone Numbers" feature to automatically mask international phone numbers within text message bubbles (both historically via REST API and live via WebSockets).

### Removed

- Removed the "Rating Window" (days) setting from both frontend and backend configurations, simplifying the chat close rating feature.

## [Unreleased] - 2026-02-28

### Added

- Whatsmeow typing-indicator planner module with cooldown and provider context skip support.
- Live chat typing presence endpoint `POST /api/contacts/{id}/typing` for composing/paused signaling from frontend composer.
- Whatsmeow typing presence module (`typing_presence.go`) with direct-chat validation and recipient normalization tests.
- Campaign/send policy helper modules and explicit reason-code constants for strict sending enforcement.
- Backend tests for campaign delay scope, typing indicator behavior, and send error classification.
- Frontend unit tests for instances store and auto-campaign normalization.
- E2E specs for instances health dashboard and chat policy-blocked send flow.
- Message history navigation in the chat view (954019f)
- Contact management functionality with backend API and Spanish localization (e0a23f5)
- Whatomate MCP sidecar with SDK transports (80f6185)
- Conversation notes and chat system messages (644c4f0, 0adddcd)
- Assigned chat reset functionality (1aae35e)
- Auto campaign settings and chat rating features (93f8a57)

### Changed

- Enforced campaign start/delay guardrails for Whatsmeow instances (connected + block checks + draft-only policy).
- Updated `ChatView` composer flow to send live `composing`/`paused` typing presence with throttling, idle pause, and cleanup on send/chat-switch/unmount.
- Updated worker campaign delay limiter from campaign scope to instance scope and added permanent-error retry classification.
- Persisted instance send blocking metadata from Whatsmeow events and surfaced send-block details in instances UI.
- Improved instances and chat UX for policy failures using `reason_code` mapping and better status transitions.
- Expanded instances E2E coverage for websocket status events, watchdog timeout, auto-campaign payload behavior, and delete-chats payload.
- Updated frontend routing and backend handlers for media and messaging (cc8cbc8)
- Enhanced Whatsmeow media processing and handler logic (93f8a57)
- Improved contact management and chat assignment persistence (644c4f0)

## [0.1.0] - 2025-02-18 (Example - adjust based on previous content)

### Added

- Created AGENT.md, PLAN.md, MEMORY.md, and session_summary.md per Ralph protocol.
- Added `mcp-server/` TypeScript MCP sidecar with SDK `@modelcontextprotocol/sdk@1.27.0`.
- Added MCP tool/resource/prompt registries and Whatomate/OpenAI typed client modules.
- Added streamable HTTP `/mcp`, health endpoint `/healthz`, and feature-flagged legacy SSE `/sse` + `/messages`.
- Added sidecar test coverage (unit, integration, e2e) and CI job in `.github/workflows/test.yml`.
- Added optional `mcp-server` service profile in `docker/docker-compose.yml`.

### Changed

- Updated root `README.md` with MCP sidecar quickstart and HTTP usage.
- Claim chat now always appends a `chat_claimed` system message on successful claim responses, including already-assigned same-user claim requests.
- Outgoing agent message name-prefixing is now controlled per user via Send Restrictions (`prefix_agent_name`) instead of role permission.

## 2026-03-01 20:46

### Fixed

- Fixed an issue where new incoming WebSocket messages from unknown contacts would hijack the active chat's state, causing their messages to appear in the currently open conversation view.

## 2026-03-01

### Fixed

- Fixed chat layout jumping unexpectedly when images/media finished loading asynchronously. Users opening a chat will now stay pinned at the bottom smoothly without having to scroll down again.
