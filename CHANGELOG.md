# CHANGELOG.md

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
