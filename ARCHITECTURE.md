# Architecture

## Overview

Whatomate is primarily a Go monolith (`cmd/whatomate`) with embedded frontend assets. A new TypeScript MCP sidecar now lives in `mcp-server/` and integrates with the existing REST API via API keys.

## Primary Runtime Components

1. Go API/worker binary (`whatomate`)
- Serves REST endpoints under `/api`
- Handles auth, RBAC, campaign workers, chatbot, and websocket events

2. MCP sidecar (`mcp-server`)
- Exposes MCP Tools, Resources, and Prompts for LLM integrations
- Supports `stdio`, Streamable HTTP (`/mcp`), and optional legacy SSE (`/sse`, `/messages`)
- Uses strict env validation, request bounds, bearer auth (HTTP), and outbound host allowlist

## MCP Sidecar Internal Layout

- `src/config.ts`: env parsing and defaults
- `src/logger.ts`: structured logger with file/stderr sink
- `src/errors.ts`: safe error shaping for MCP responses
- `src/clients/whatomate-client.ts`: typed Whatomate envelope client with GET retries
- `src/clients/openai-client.ts`: OpenAI summarization connector
- `src/mcp/server.ts`: MCP server creation + registry wiring
- `src/tools/*`: tool handlers for contacts/messages/campaigns/analytics/openai
- `src/resources/*`: URI-based data resources
- `src/prompts/*`: reusable prompt templates
- `src/transports/*`: stdio + Streamable HTTP + legacy SSE transport adapters

## Deployment

- Local: run from `mcp-server` using npm scripts.
- Compose: optional `mcp-server` profile in `docker/docker-compose.yml`.
- Sidecar talks to Whatomate over internal network (`WHATOMATE_BASE_URL`).

## Marketing Sidecar Handoff

- The main frontend no longer serves bundled pricing/plans/offers content.
- `/pricing`, `/plans`, and `/offer` now resolve to a lightweight redirect handoff view in the SPA.
- The redirect destination is configured with `VITE_PUBLIC_MARKETING_BASE_URL`, which may point to an external origin or a same-origin path prefix.
- Public lead ingestion remains in the monolith through `POST /api/public/lead-requests`.
- Lead source validation is now generic enough for sidecar-owned marketing routes instead of hardcoding pricing-only values.

## Testing Strategy

- Unit tests: config, errors, schema validation
- Integration tests: Whatomate/OpenAI client behavior and transport auth
- E2E tests: HTTP workflow + stdio handshake against local mock services
