# Whatomate MCP Server

TypeScript MCP sidecar for Whatomate.

## Features

- MCP Tools, Resources, and Prompts for Whatomate core operations
- Streamable HTTP transport (`/mcp`) and local `stdio` transport
- Optional legacy SSE compatibility endpoints (`/sse`, `/messages`)
- OpenAI-powered conversation summarization tool
- Zod-validated inputs and strict env configuration

## Requirements

- Node.js 20+
- Running Whatomate API endpoint
- Whatomate API key
- OpenAI API key

## Install

```bash
cd mcp-server
npm install
```

## Configuration

Copy and edit environment variables:

```bash
cp .env.example .env
```

Minimum required variables:

- `MCP_TRANSPORT=stdio|http|hybrid`
- `WHATOMATE_BASE_URL`
- `WHATOMATE_API_KEY`
- `OPENAI_API_KEY`

For HTTP mode, also set:

- `MCP_HTTP_BEARER_TOKEN`
- `MCP_HTTP_HOST` and `MCP_HTTP_PORT`

## Run

Build once:

```bash
npm run build
```

Run stdio:

```bash
npm run start:stdio
```

Run HTTP transport:

```bash
npm run start:http
```

Run both transports:

```bash
MCP_TRANSPORT=hybrid npm run start
```

## HTTP Endpoints

- `POST/GET/DELETE /mcp` - Streamable HTTP transport
- `GET /sse` and `POST /messages` - legacy SSE (when `MCP_ENABLE_LEGACY_SSE=true`)
- `GET /healthz` - health check

All MCP transport endpoints require `Authorization: Bearer <MCP_HTTP_BEARER_TOKEN>` in HTTP mode.

## Available MCP Tools

- `whatomate_list_contacts`
- `whatomate_get_contact`
- `whatomate_list_messages`
- `whatomate_send_text_message`
- `whatomate_create_campaign`
- `whatomate_start_campaign`
- `whatomate_get_campaign_status`
- `whatomate_get_dashboard_analytics`
- `whatomate_openai_summarize_conversation`

## Available MCP Resources

- `whatomate://organization/current`
- `whatomate://contacts/{contactId}`
- `whatomate://contacts/{contactId}/messages?limit={limit}`
- `whatomate://campaigns/{campaignId}`
- `whatomate://analytics/dashboard?period={period}&account_id={accountId}`

## Available MCP Prompts

- `whatomate_draft_reply`
- `whatomate_campaign_brief`
- `whatomate_handoff_summary`

## Development Commands

```bash
npm run lint
npm run typecheck
npm run test
npm run test:e2e
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
