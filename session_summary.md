# Session Summary - 2026-02-24 15:19

## Objective

Implement a production-grade TypeScript MCP sidecar (`mcp-server/`) for Whatomate with modular Tools, Resources, Prompts, stdio + HTTP transports, and OpenAI summarization support.

## Modules Touched

- `mcp-server/package.json`
- `mcp-server/tsconfig.json`
- `mcp-server/eslint.config.js`
- `mcp-server/vitest.config.ts`
- `mcp-server/.env.example`
- `mcp-server/Dockerfile`
- `mcp-server/README.md`
- `mcp-server/src/index.ts`
- `mcp-server/src/config.ts`
- `mcp-server/src/logger.ts`
- `mcp-server/src/errors.ts`
- `mcp-server/src/mcp/server.ts`
- `mcp-server/src/mcp/tool-registry.ts`
- `mcp-server/src/mcp/resource-registry.ts`
- `mcp-server/src/mcp/prompt-registry.ts`
- `mcp-server/src/transports/stdio.ts`
- `mcp-server/src/transports/streamable-http.ts`
- `mcp-server/src/transports/legacy-sse.ts`
- `mcp-server/src/clients/whatomate-client.ts`
- `mcp-server/src/clients/openai-client.ts`
- `mcp-server/src/tools/contacts.ts`
- `mcp-server/src/tools/messages.ts`
- `mcp-server/src/tools/campaigns.ts`
- `mcp-server/src/tools/analytics.ts`
- `mcp-server/src/tools/openai.ts`
- `mcp-server/src/resources/organization.ts`
- `mcp-server/src/resources/contacts.ts`
- `mcp-server/src/resources/campaigns.ts`
- `mcp-server/src/resources/analytics.ts`
- `mcp-server/src/prompts/draft-reply.ts`
- `mcp-server/src/prompts/campaign-brief.ts`
- `mcp-server/src/prompts/handoff-summary.ts`
- `mcp-server/tests/unit/config.test.ts`
- `mcp-server/tests/unit/errors.test.ts`
- `mcp-server/tests/unit/tool-schemas.test.ts`
- `mcp-server/tests/integration/clients.test.ts`
- `mcp-server/tests/integration/transports.test.ts`
- `mcp-server/tests/e2e/workflow.e2e.test.ts`
- `README.md`
- `docker/docker-compose.yml`
- `.github/workflows/test.yml`
- `MEMORY.md`
- `CHANGELOG.md`

## Technical Decisions

- Used `@modelcontextprotocol/sdk@1.27.0` and modularized the sidecar around registries and domain modules to keep blast radius low and testability high.
- Implemented streamable HTTP `/mcp` as primary remote transport and retained legacy SSE compatibility (`/sse` + `/messages`) behind `MCP_ENABLE_LEGACY_SSE`.
- Enforced deterministic config and payload validation with Zod, API timeout budgets, GET retries for idempotent Whatomate reads, and outbound host allowlisting.
- Normalized tool outputs to return both text content and object-form `structuredContent` for consistent MCP client consumption.
- Added CI checks for sidecar quality gates (`lint`, `typecheck`, `test`, `test:e2e`) without changing existing Go backend contracts.

## Next Steps

1. Wire staging secrets and bearer token into deployment environment for HTTP transport validation with a real MCP client.
2. Add production observability hooks (log shipping/metrics) for sidecar request and tool execution latency.
3. Optionally add integration tests against a running Whatomate backend instance (not mock services) for pre-release validation.

---

# Session Summary - 2026-02-22 01:07

## Objective

Upgrade canned responses to support WhatsApp-native rich text authoring and reusable photo/video attachments that can be sent together from chat.

## Modules Touched

- `internal/models/canned_responses.go`
- `internal/handlers/canned_responses.go`
- `internal/handlers/canned_response_media.go` (new)
- `internal/handlers/canned_response_send.go` (new)
- `cmd/whatomate/main.go`
- `frontend/src/services/api.ts`
- `frontend/src/views/settings/CannedResponsesView.vue`
- `frontend/src/components/chat/WhatsAppRichTextEditor.vue` (new)
- `frontend/src/components/chat/CannedResponsePicker.vue`
- `frontend/src/views/chat/ChatView.vue`
- `frontend/src/i18n/locales/en.json`
- `frontend/src/i18n/locales/ar.json`
- `MEMORY.md`
- `CHANGELOG.md`

## Technical Decisions

- Added typed canned-response attachments in model JSONB instead of introducing a new relational table to keep migration risk and query impact low.
- Reused existing outbound send and media-storage pipeline (`SendOutgoingMessage`, `saveMediaLocally`) so canned response dispatch stays provider-compatible (Meta + whatsmeow) without custom send logic per provider.
- Added a dedicated canned-response send endpoint to keep chat-side UX simple while preserving backend permission/lifecycle enforcement for closed/pending chat guards.
- Implemented a focused rich-text toolbar that inserts exact WhatsApp syntax wrappers to guarantee message rendering fidelity in WhatsApp clients.

## Next Steps

1. Add API tests for multipart canned response create/update flows with attachment keep/remove semantics.
2. Add attachment preview endpoint + thumbnails in settings edit dialog for previously uploaded media.
3. Add optional per-attachment captions if teams need media-specific text separate from the canned body.

---

# Session Summary - 2026-02-21 21:11

## Objective

Implement an automated daily reset that moves assigned chats back to pending, with admin-configurable reset time (default midnight or custom hour).

## Modules Touched

- `internal/handlers/chat_assignment_reset_settings.go` (new)
- `internal/handlers/chat_assignment_reset_worker.go` (new)
- `internal/handlers/chat_assignment_reset_worker_test.go` (new)
- `internal/handlers/organization.go`
- `internal/handlers/organization_test.go`
- `cmd/whatomate/main.go`
- `frontend/src/services/api.ts`
- `frontend/src/views/settings/SettingsView.vue`
- `MEMORY.md`
- `CHANGELOG.md`

## Technical Decisions

- Stored reset schedule as organization JSONB settings (`assigned_chat_reset_mode`, `assigned_chat_reset_hour`) to avoid schema migration and keep admin settings centralized.
- Implemented a dedicated background worker running every minute, evaluating each organization in its configured timezone, with once-per-day execution tracked via `assigned_chat_reset_last_date`.
- Added bootstrap guard for default midnight mode so organizations with no historical marker are initialized without an immediate daytime reset.
- Kept frontend media-grouping preference local while persisting reset schedule to organization settings from the same Chat Preferences panel.

## Next Steps

1. Add a user-visible activity/audit log entry when scheduled resets run.
2. Optionally add an admin endpoint/button to trigger a manual reset immediately.
3. Consider surfacing the next scheduled reset time in the settings UI.

---

# Session Summary - 2026-02-18 22:50

## Objective

Investigate the "[Unsupported message type]" error for messages containing "Mechanical" sent to a specific group, identifying the root cause without implementing a fix.

## Findings

- Confirmed that standard text messages ("Mechanical") are processed correctly.
- Identified that **PollUpdateMessage** (voting on a poll) explicitly returns `[Unsupported message type]` in `pkg/whatsmeow`.
- **Hypothesis**: The issue is caused when users vote on a poll option named "Mechanical". The vote update (`PollUpdateMessage`) is received but not handled, resulting in the generic error message.
- `SenderKeyDistributionMessage` also triggers this error, but `PollUpdateMessage` aligns better with the user report of "containing 'Mechanical'".

## Modules Touched (Investigation Only)

- `pkg/whatsmeow/repro_diag_mechanical_test.go` (Created and archived to `.repro_archive/2026-02-18_issue_investigation.go`)
- `diagnostics_report.md` (Created)

## Technical Decisions

- Used a comprehensive test suite covering Text, ExtendedText, Buttons, List, Interactive, Protocol, Reaction, and Poll messages to isolate the failure mode.
- Verified that simple text passes, ruling out basic encoding issues.
- Pinpointed `PollUpdateMessage` as the primary suspect for user-visible errors with context.

## Next Steps

- Implement handling for `PollUpdateMessage` in `pkg/whatsmeow/incoming_media.go` or `message_persist.go`.
- Implement filtering/silencing for `SenderKeyDistributionMessage`.
- Update frontend to handle unsupported types gracefully without showing error text if possible.

---

# Session Summary - 2026-02-19 05:37

## Objective

Eliminate silent catastrophic JWT auth failure modes by hardening secret validation and token issuance paths.

## Modules Touched

- `cmd/whatomate/main.go`
- `internal/config/jwt_validation.go`
- `internal/config/jwt_validation_test.go`
- `internal/middleware/middleware.go`
- `internal/middleware/middleware_test.go`
- `internal/handlers/jwt_secret.go`
- `internal/handlers/auth.go`
- `internal/handlers/websocket.go`
- `JWT_SECRET_HANDLING_REPORT.md`

## Technical Decisions

- Introduced centralized JWT config validation to avoid partial/duplicated checks and fail process startup on unsafe secret states.
- Enforced defense-in-depth at runtime by making middleware fail closed when secret is blank.
- Replaced ignored token-signing errors with explicit error handling in auth flows to remove silent failures.
- Centralized handler-side JWT secret retrieval to keep signing/parsing behavior consistent.

## Next Steps

- Optionally enforce stronger minimum JWT secret length in non-production environments for staging parity.
- Add operational startup check in deployment pipelines that validates `WHATOMATE_JWT_SECRET` before rollout.

---

# Session Summary - 2026-02-19 05:40

## Objective

Unblock local startup after JWT hardening by aligning config defaults/docs with fail-fast secret validation.

## Modules Touched

- `internal/config/jwt_validation.go`
- `config.example.toml`
- `config.toml` (local, untracked)
- `docs/src/content/docs/getting-started/configuration.mdx`
- `README.md`
- `JWT_SECRET_HANDLING_REPORT.md`
- `CHANGELOG.md`
- `MEMORY.md`

## Technical Decisions

- Kept strict fail-fast JWT validation (no rollback to warning-only behavior).
- Removed blocked placeholder secrets from setup templates to prevent false-start boot failures.
- Added direct remediation command in both docs and runtime error messages.
- Set a generated local secret in untracked `config.toml` so this workspace boots immediately.

## Next Steps

- Run production/staging with `WHATOMATE_JWT_SECRET` from secret manager instead of committed files.
- Add CI/static check that rejects placeholder/empty JWT values in deploy manifests.

---

# Session Summary - 2026-02-19 05:46

## Objective

Harden CORS and websocket origin defaults so cookie-authenticated requests do not rely on permissive cross-origin behavior.

## Modules Touched

- `internal/middleware/middleware.go`
- `cmd/whatomate/main.go`
- `internal/handlers/websocket.go`
- `internal/middleware/middleware_test.go`
- `internal/handlers/websocket_origin_test.go`
- `config.example.toml`
- `CORS_WEBSOCKET_ORIGIN_HARDENING_REPORT.md`

## Technical Decisions

- Introduced a shared origin evaluator (`IsOriginAllowedForRequest`) to keep CORS and WS checks consistent.
- Switched empty-allowlist fallback from allow-all to safe defaults (same-origin + loopback localhost only).
- Added origin normalization in config parsing to reduce mismatch bugs from default ports/trailing slash variants.
- Added focused regression tests for both middleware and websocket upgrader checks.

## Next Steps

- Set explicit `server.allowed_origins` in every non-local deployment.
- Optionally add startup validation/warning for production when `allowed_origins` is empty.

---

# Session Summary - 2026-02-19 06:12

## Objective

Close security and reliability gaps for tasks 5-9: secrets-at-rest enforcement, webhook trust hardening, queue/worker resiliency + idempotency, custom-action runtime safety, and endpoint routing consistency.

## Modules Touched

- `cmd/whatomate/main.go`
- `internal/config/encryption_validation.go`
- `internal/config/encryption_validation_test.go`
- `internal/crypto/crypto.go`
- `internal/crypto/crypto_test.go`
- `internal/handlers/accounts.go`
- `internal/handlers/cache.go`
- `internal/handlers/webhook.go`
- `internal/handlers/webhook_security.go`
- `internal/handlers/webhook_security_test.go`
- `internal/queue/redis.go`
- `internal/queue/queue_test.go`
- `internal/worker/worker.go`
- `internal/worker/idempotency.go`
- `internal/worker/worker_test.go`
- `internal/handlers/custom_action_runtime.go`
- `internal/handlers/custom_actions.go`
- `internal/handlers/custom_actions_test.go`
- `internal/handlers/whatsapp_client.go`
- `internal/handlers/flows.go`
- `ARCHITECTURE_RISK_REPORT.md`

## Technical Decisions

- Kept startup encryption validation strict in production, while preventing plaintext secret writes in all environments by making encryption fail when key is missing.
- Enforced webhook authenticity by requiring and validating Meta signatures against resolved app secrets (phone ID/business ID).
- Removed unbounded webhook goroutine fan-out and added payload-size guardrails.
- Added queue DLQ + periodic pending claim to avoid indefinite stalls and poison-message loops.
- Added recipient-level Redis lock + non-pending recipient skip checks to reduce duplicate sends under retries.
- Standardized worker and flow handlers to use configured WhatsApp base URL routing.
- Added bounded JS execution runtime and Redis-backed one-time redirect tokens for distributed-safe custom action behavior.

## Next Steps

1. Rotate/re-encrypt any legacy plaintext account secrets with a configured `app.encryption_key`.
2. Add metrics/alerts for DLQ growth and pending-claim recoveries.
3. Consider webhook replay protection (timestamp/nonce) as a second integrity layer beyond HMAC signature.

---

# Session Summary - 2026-02-20 00:14

## Objective

Implement explicit chat lifecycle handling (pending/assigned/closed) end-to-end in Go API + Vue frontend, including claim/close actions, pending chat privacy guards, and closed-chat read-only history view in settings.

## Modules Touched

- `cmd/whatomate/main.go`
- `internal/models/chat_status.go`
- `internal/models/models.go`
- `internal/database/postgres.go`
- `internal/handlers/chat_lifecycle.go`
- `internal/handlers/contacts.go`
- `internal/handlers/agent_transfers.go`
- `internal/handlers/contacts_test.go`
- `frontend/src/services/api.ts`
- `frontend/src/stores/contacts.ts`
- `frontend/src/views/chat/ChatView.vue` (large file; now near 2500+ lines, candidate for follow-up extraction)
- `frontend/src/router/index.ts`
- `frontend/src/components/layout/navigation.ts`
- `frontend/src/views/settings/ClosedChatsView.vue`
- `CHANGELOG.md`
- `MEMORY.md`

## Technical Decisions

- Reused `contacts` as the backing store for chats to avoid introducing a separate conversation table migration; added lifecycle columns and normalized status behavior in model helpers.
- Added dedicated lifecycle helper functions in a new handler module to keep status filter parsing/assignment transitions reusable across contacts and transfer flows.
- Enforced pending/unassigned message-read restrictions in backend handlers (`GetMessages`) so direct API calls cannot bypass UI restrictions.
- Kept closed chats readable but read-only by blocking send/media/reaction paths while allowing contact/message retrieval for historical views.
- Implemented frontend bucketed chat state in Pinia and tab-aware pagination using `/api/chats` with `status` and `assigned_to` filters.

## Next Steps

1. Add focused API tests for `ClaimChat` and `CloseChat` edge cases (double-claim, close by non-assignee, admin override).
2. Extract subcomponents from `ChatView.vue` (header/actions, restricted-state panel, input footer) to reduce maintenance risk.
3. Add i18n keys for new literal UI strings (`Pending`, `Assigned`, `Restricted View`, `Closed Chats`) to avoid mixed translation behavior.

---

# Session Summary - 2026-02-20 00:52

## Objective

Implement closed-chat reopen behavior end-to-end: auto-reopen on inbound message, manual reopen button in closed chat UI, forced unassignment on reopen, and settings pages for pending/assigned/closed queues.

## Modules Touched

- `cmd/whatomate/main.go`
- `internal/handlers/chat_lifecycle.go`
- `internal/handlers/contacts.go`
- `internal/handlers/messages.go`
- `internal/handlers/chatbot_processor.go`
- `internal/handlers/contacts_test.go`
- `internal/handlers/chatbot_processor_test.go`
- `pkg/whatsmeow/chat_lifecycle.go`
- `pkg/whatsmeow/message_persist.go`
- `frontend/src/services/api.ts`
- `frontend/src/services/websocket.ts`
- `frontend/src/stores/contacts.ts`
- `frontend/src/views/chat/ChatView.vue`
- `frontend/src/views/settings/ClosedChatsView.vue`
- `frontend/src/views/settings/PendingChatsView.vue`
- `frontend/src/views/settings/AssignedChatsView.vue`
- `frontend/src/views/settings/SettingsView.vue`
- `frontend/src/router/index.ts`
- `frontend/src/components/layout/navigation.ts`
- `CHANGELOG.md`
- `MEMORY.md`

## Technical Decisions

- Reopen semantics were made deterministic: reopening always applies `pending` status with `assigned_user_id = NULL`, clearing closed metadata.
- Auto-reopen was implemented in both inbound persistence paths (`saveIncomingMessage` for Meta and `persistParsedMessage` for Whatsmeow) so behavior is provider-independent.
- WebSocket `new_message` payloads now include `contact_status` and normalized assignment payloads, and frontend applies `patchContact` immediately to keep queue tabs synchronized without waiting for manual refresh.
- Added dedicated settings list views for `pending`, `assigned`, and `closed` queues to satisfy queue visibility from `/settings`.

## Next Steps

1. Add i18n keys for new queue/reopen labels currently hardcoded in settings/chat views.
2. Add API tests for auto-reopen under Whatsmeow inbound events (currently covered in handler-level saveIncomingMessage tests).
3. Consider throttling/debouncing websocket-triggered `fetchChats()` refreshes under high message volume.

---

# Session Summary - 2026-02-21 02:14

## Objective

Synchronize messages sent from the connected WhatsApp mobile device into the Whatomate chat thread as real-time outgoing messages with sent status.

## Modules Touched

- `pkg/whatsmeow/events_message.go`
- `pkg/whatsmeow/events_message_device_sent_test.go`
- `MEMORY.md`
- `CHANGELOG.md`

## Technical Decisions

- Kept the existing protection against duplicate self-origin runtime events by continuing to ignore generic `IsFromMe` events.
- Enabled persistence for cross-device sync events by gating `IsFromMe` handling on `DeviceSentMeta` (indicator for messages sent from another linked device, e.g. phone).
- Preserved existing message persistence and WebSocket payload flow so synced mobile messages reuse the same outgoing rendering and status semantics.
- Added focused regression tests for both paths: device-sent `IsFromMe` persists as outgoing, runtime-local `IsFromMe` remains skipped.

## Next Steps

1. Run the new DB-backed Whatsmeow tests in CI/local with `TEST_DATABASE_URL` set to validate persistence behavior end-to-end.
2. Optionally add an integration test asserting frontend WebSocket rendering for a device-sent outgoing event in `ChatView`.
