# Whatomate MCP Server

Production TypeScript MCP sidecar for Whatomate, built on `@modelcontextprotocol/sdk@1.27.0`.

## Implementation Overview

- SDK: `@modelcontextprotocol/sdk@1.27.0`
- Runtime: Node.js 20+
- Core primitives:
  - Tools (action execution)
  - Resources (structured context)
  - Prompts (workflow templates)
- Transports:
  - `stdio` for local MCP hosts
  - Streamable HTTP at `/mcp` for remote clients
  - Optional legacy SSE compatibility (`/sse`, `/messages`)
- Security:
  - Bearer auth on MCP HTTP transport routes
  - Zod validation for config and tool/prompt inputs
  - Outbound host allowlisting
  - Sanitized error responses

Main modules:

- `src/mcp/*` server + registries
- `src/transports/*` stdio, streamable HTTP, legacy SSE
- `src/clients/*` typed Whatomate + OpenAI connectors
- `src/tools/*`, `src/resources/*`, `src/prompts/*` modular feature surface

## Exposed MCP Interfaces

Tools:

- `whatomate_list_contacts`
- `whatomate_get_contact`
- `whatomate_list_messages`
- `whatomate_send_text_message`
- `whatomate_create_campaign`
- `whatomate_start_campaign`
- `whatomate_get_campaign_status`
- `whatomate_get_dashboard_analytics`
- `whatomate_openai_summarize_conversation`

Resources:

- `whatomate://organization/current`
- `whatomate://contacts/{contactId}`
- `whatomate://contacts/{contactId}/messages?limit={limit}`
- `whatomate://campaigns/{campaignId}`
- `whatomate://analytics/dashboard?period={period}&account_id={accountId}`

Prompts:

- `whatomate_draft_reply`
- `whatomate_campaign_brief`
- `whatomate_handoff_summary`

## HTTP Endpoints

- `POST/GET/DELETE /mcp` (Streamable HTTP transport, auth required)
- `GET /sse` (legacy compatibility mode, auth required)
- `POST /messages` (legacy compatibility mode, auth required)
- `GET /healthz` (health check, no auth)

Legacy SSE endpoints are available only when `MCP_ENABLE_LEGACY_SSE=true`.

## Installation

```bash
cd mcp-server
npm install
cp .env.example .env
```

## Configuration

Required:

- `WHATOMATE_BASE_URL`
- `WHATOMATE_API_KEY`
- `OPENAI_API_KEY`

Transport selection:

- `MCP_TRANSPORT=stdio|http|hybrid` (default: `stdio`)
- `MCP_HTTP_BEARER_TOKEN` required for `http` and `hybrid`

Key optional settings:

- `MCP_HTTP_HOST` (default `127.0.0.1`)
- `MCP_HTTP_PORT` (default `3000`)
- `MCP_ENABLE_LEGACY_SSE` (default `false`)
- `MCP_HTTP_ALLOWED_HOSTS` (CSV)
- `WHATOMATE_ORGANIZATION_ID`
- `OPENAI_MODEL` (default `gpt-4o-mini`)
- `LOG_LEVEL` (default `info`)
- `LOG_FILE` (stdio defaults to `/tmp/whatomate-mcp.log`)
- `MCP_REQUEST_TIMEOUT_MS` (default `30000`)
- `MCP_GET_RETRIES` (default `2`, GET only)
- `MCP_OUTBOUND_HOST_ALLOWLIST` (CSV extension)

## Run

Build:

```bash
npm run build
```

Start in stdio mode:

```bash
npm run start:stdio
```

Start in HTTP mode:

```bash
MCP_TRANSPORT=http \
MCP_HTTP_BEARER_TOKEN=replace_me \
WHATOMATE_BASE_URL=http://localhost:8080 \
WHATOMATE_API_KEY=whm_replace_me \
OPENAI_API_KEY=sk_replace_me \
npm run start:http
```

Start hybrid mode:

```bash
MCP_TRANSPORT=hybrid npm run start
```

## How To Use It

### 1) Use with an MCP host over stdio

Point your MCP host to run:

```bash
node /absolute/path/to/whatomate/mcp-server/dist/index.js
```

With environment:

- `MCP_TRANSPORT=stdio`
- `WHATOMATE_BASE_URL`
- `WHATOMATE_API_KEY`
- `OPENAI_API_KEY`

### 2) Use with an MCP client over HTTP

Connect to:

- `http://<host>:<port>/mcp`

Set header:

- `Authorization: Bearer <MCP_HTTP_BEARER_TOKEN>`

Minimal TypeScript client example:

```ts
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const client = new Client({ name: "demo-client", version: "1.0.0" });
const transport = new StreamableHTTPClientTransport(
  new URL("http://127.0.0.1:3000/mcp"),
  {
    requestInit: {
      headers: {
        Authorization: "Bearer replace_me"
      }
    }
  }
);

await client.connect(transport);

const tools = await client.listTools();
console.log(tools.tools.map((t) => t.name));

const result = await client.callTool({
  name: "whatomate_list_contacts",
  arguments: { page: 1, limit: 20 }
});

console.log(result.structuredContent);
await transport.close();
```

## Development and Validation

```bash
npm run lint
npm run typecheck
npm run test
npm run test:e2e
npm run build
```

## Docker

Build image:

```bash
docker build -t whatomate-mcp ./mcp-server
```

Run container:

```bash
docker run --rm -p 3000:3000 \
  -e MCP_TRANSPORT=http \
  -e MCP_HTTP_HOST=0.0.0.0 \
  -e MCP_HTTP_PORT=3000 \
  -e MCP_HTTP_BEARER_TOKEN=replace_me \
  -e WHATOMATE_BASE_URL=http://host.docker.internal:8080 \
  -e WHATOMATE_API_KEY=whm_replace_me \
  -e OPENAI_API_KEY=sk_replace_me \
  whatomate-mcp
```

Compose profile from repository root:

```bash
docker compose -f docker/docker-compose.yml --profile mcp up -d --build
```
