import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { SSEClientTransport } from '@modelcontextprotocol/sdk/client/sse.js';
import { loadConfig } from '../../src/config.js';
import { Logger } from '../../src/logger.js';
import { WhatomateClient } from '../../src/clients/whatomate-client.js';
import { OpenAIClient } from '../../src/clients/openai-client.js';
import { createWhatomateMcpServer } from '../../src/mcp/server.js';
import { startHttpServer, type RunningHttpServer } from '../../src/transports/streamable-http.js';
import { startMockServices, type MockServices } from '../support/mock-services.js';

let services: MockServices;
let httpServer: RunningHttpServer;
const bearerToken = 'test-bearer-token';

beforeEach(async () => {
  const started = await startMockServices();
  services = started.services;

  const config = loadConfig({
    MCP_TRANSPORT: 'http',
    MCP_HTTP_BEARER_TOKEN: bearerToken,
    MCP_HTTP_HOST: '127.0.0.1',
    MCP_HTTP_PORT: '0',
    MCP_ENABLE_LEGACY_SSE: 'true',
    WHATOMATE_BASE_URL: services.whatomate.baseUrl,
    WHATOMATE_API_KEY: 'whm_test',
    OPENAI_API_KEY: 'sk_test',
    OPENAI_BASE_URL: services.openai.baseUrl,
    LOG_FILE: '/tmp/whatomate-mcp-test.log'
  });

  const logger = new Logger({ level: 'error', filePath: '/tmp/whatomate-mcp-test.log', useStderr: false });
  const whatomateClient = new WhatomateClient(config);
  const openAiClient = new OpenAIClient(config);

  httpServer = await startHttpServer({
    host: config.httpHost,
    port: config.httpPort,
    bearerToken,
    allowedHosts: config.httpAllowedHosts,
    enableLegacySse: true,
    logger,
    createServer: () => createWhatomateMcpServer({
      logger,
      whatomateClient,
      openAiClient
    })
  });
});

afterEach(async () => {
  await httpServer.close();
  await services.whatomate.close();
  await services.openai.close();
});

describe('HTTP transports', () => {
  it('supports Streamable HTTP list_tools and call_tool', async () => {
    const client = new Client({ name: 'integration-client', version: '1.0.0' });
    const transport = new StreamableHTTPClientTransport(new URL(`${httpServer.baseUrl}/mcp`), {
      requestInit: {
        headers: {
          Authorization: `Bearer ${bearerToken}`
        }
      }
    });

    await client.connect(transport);

    const tools = await client.listTools();
    expect(tools.tools.some((tool) => tool.name === 'whatomate_get_contact')).toBe(true);

    const result = await client.callTool({
      name: 'whatomate_get_contact',
      arguments: { contact_id: 'contact-1' }
    });

    expect(result.isError).not.toBe(true);

    await transport.close();
  });

  it('supports legacy SSE endpoints when enabled', async () => {
    const client = new Client({ name: 'legacy-client', version: '1.0.0' });
    const transport = new SSEClientTransport(new URL(`${httpServer.baseUrl}/sse`), {
      requestInit: {
        headers: {
          Authorization: `Bearer ${bearerToken}`
        }
      }
    });

    await client.connect(transport);

    const tools = await client.listTools();
    expect(tools.tools.some((tool) => tool.name === 'whatomate_list_contacts')).toBe(true);

    await transport.close();
  });

  it('rejects unauthorized requests', async () => {
    const response = await fetch(`${httpServer.baseUrl}/mcp`, {
      method: 'GET'
    });

    expect(response.status).toBe(401);
  });
});
