# CHANGELOG.md

## [Unreleased] - 2026-02-28

### Added
- Message history navigation in the chat view (954019f)
- Contact management functionality with backend API and Spanish localization (e0a23f5)
- Whatomate MCP sidecar with SDK transports (80f6185)
- Conversation notes and chat system messages (644c4f0, 0adddcd)
- Assigned chat reset functionality (1aae35e)
- Auto campaign settings and chat rating features (93f8a57)

### Changed
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
