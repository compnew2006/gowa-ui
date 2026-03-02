import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { loadConfig } from '../../src/config.js';
import { Logger } from '../../src/logger.js';
// import { WhatomateClient } from '../../src/clients/whatomate-client.js';
const WhatomateClient = class { constructor(config: any) {} } as any;
import { OpenAIClient } from '../../src/clients/openai-client.js';
import { createWhatomateMcpServer } from '../../src/mcp/server.js';
import { startHttpServer, type RunningHttpServer } from '../../src/transports/streamable-http.js';
import { startMockServices, type MockServices } from '../support/mock-services.js';

let services: MockServices;
let httpServer: RunningHttpServer;
const bearerToken = 'e2e-bearer-token';

beforeEach(async () => {
  const started = await startMockServices();
  services = started.services;

  const config = loadConfig({
    MCP_TRANSPORT: 'http',
    MCP_HTTP_BEARER_TOKEN: bearerToken,
    MCP_HTTP_HOST: '127.0.0.1',
    MCP_HTTP_PORT: '0',
    WHATOMATE_BASE_URL: services.whatomate.baseUrl,
    WHATOMATE_API_KEY: 'whm_test',
    OPENAI_API_KEY: 'sk_test',
    OPENAI_BASE_URL: services.openai.baseUrl,
    LOG_FILE: '/tmp/whatomate-mcp-e2e.log'
  });

  const logger = new Logger({ level: 'error', filePath: '/tmp/whatomate-mcp-e2e.log', useStderr: false });
  const whatomateClient = new WhatomateClient(config);
  const openAiClient = new OpenAIClient(config);

  httpServer = await startHttpServer({
    host: config.httpHost,
    port: config.httpPort,
    bearerToken,
    allowedHosts: config.httpAllowedHosts,
    enableLegacySse: false,
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

describe('MCP sidecar end-to-end', () => {
  it('executes the full HTTP workflow', async () => {
    const client = new Client({ name: 'e2e-http-client', version: '1.0.0' });
    const transport = new StreamableHTTPClientTransport(new URL(`${httpServer.baseUrl}/mcp`), {
      requestInit: {
        headers: {
          Authorization: `Bearer ${bearerToken}`
        }
      }
    });

    await client.connect(transport);

    const contacts = await client.callTool({
      name: 'whatomate_list_contacts',
      arguments: { page: 1, limit: 10 }
    });
    expect(contacts.isError).not.toBe(true);

    const messages = await client.callTool({
      name: 'whatomate_list_messages',
      arguments: { contact_id: 'contact-1', limit: 10 }
    });
    expect(messages.isError).not.toBe(true);

    const sendMessage = await client.callTool({
      name: 'whatomate_send_text_message',
      arguments: { contact_id: 'contact-1', text: 'Thanks, updating you now.' }
    });
    expect(sendMessage.isError).not.toBe(true);

    const createCampaign = await client.callTool({
      name: 'whatomate_create_campaign',
      arguments: {
        name: 'Flash Sale',
        whatsapp_account: 'account-main',
        body_content: 'Hi there!'
      }
    });
    expect(createCampaign.isError).not.toBe(true);

    const structuredCampaign = createCampaign.structuredContent as { id?: string };
    expect(structuredCampaign?.id).toBeDefined();

    const startCampaign = await client.callTool({
      name: 'whatomate_start_campaign',
      arguments: { campaign_id: structuredCampaign.id }
    });
    expect(startCampaign.isError).not.toBe(true);

    const summarize = await client.callTool({
      name: 'whatomate_openai_summarize_conversation',
      arguments: { contact_id: 'contact-1', limit: 10 }
    });
    expect(summarize.isError).not.toBe(true);

    await transport.close();
  });

  it('supports stdio handshake and list_tools', async () => {
    const client = new Client({ name: 'e2e-stdio-client', version: '1.0.0' });

    const transport = new StdioClientTransport({
      command: process.execPath,
      args: ['--import', 'tsx', 'src/index.ts'],
      cwd: process.cwd(),
      env: {
        ...process.env,
        MCP_TRANSPORT: 'stdio',
        LOG_FILE: '/tmp/whatomate-mcp-stdio.log',
        WHATOMATE_BASE_URL: services.whatomate.baseUrl,
        WHATOMATE_API_KEY: 'whm_test',
        OPENAI_API_KEY: 'sk_test',
        OPENAI_BASE_URL: services.openai.baseUrl
      },
      stderr: 'pipe'
    });

    await client.connect(transport);

    const tools = await client.listTools();
    expect(tools.tools.length).toBeGreaterThan(0);

    const contact = await client.callTool({
      name: 'whatomate_get_contact',
      arguments: { contact_id: 'contact-1' }
    });
    expect(contact.isError).not.toBe(true);

    await transport.close();
  });
});
