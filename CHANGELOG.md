# CHANGELOG.md

## Unreleased
### Added
- Created AGENT.md, PLAN.md, MEMORY.md, and session_summary.md per Ralph protocol.
- Added `mcp-server/` TypeScript MCP sidecar with SDK `@modelcontextprotocol/sdk@1.27.0`.
- Added MCP tool/resource/prompt registries and Whatomate/OpenAI typed client modules.
- Added streamable HTTP `/mcp`, health endpoint `/healthz`, and feature-flagged legacy SSE `/sse` + `/messages`.
- Added sidecar test coverage (unit, integration, e2e) and CI job in `.github/workflows/test.yml`.
- Added optional `mcp-server` service profile in `docker/docker-compose.yml`.

### Changed
- Updated root `README.md` with MCP sidecar quickstart and HTTP usage.
