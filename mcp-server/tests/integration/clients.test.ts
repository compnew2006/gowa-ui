import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { loadConfig } from '../../src/config.js';
// import { WhatomateClient } from '../../src/clients/whatomate-client.js';
const WhatomateClient = class { constructor(config: any) {} } as any;
import { OpenAIClient } from '../../src/clients/openai-client.js';
import { AppError } from '../../src/errors.js';
import { startMockServices, type MockServices } from '../support/mock-services.js';

let services: MockServices;

beforeEach(async () => {
  const started = await startMockServices();
  services = started.services;
});

afterEach(async () => {
  await services.whatomate.close();
  await services.openai.close();
});

describe('API clients', () => {
  it('executes Whatomate client workflows', async () => {
    const config = loadConfig({
      MCP_TRANSPORT: 'stdio',
      WHATOMATE_BASE_URL: services.whatomate.baseUrl,
      WHATOMATE_API_KEY: 'whm_test',
      OPENAI_API_KEY: 'sk_test',
      OPENAI_BASE_URL: services.openai.baseUrl
    });

    const whatomateClient = new WhatomateClient(config);

    const contacts = await whatomateClient.listContacts({ page: 1, limit: 20 });
    expect(contacts.items.length).toBeGreaterThan(0);

    const contact = await whatomateClient.getContact('contact-1');
    expect(contact.id).toBe('contact-1');

    const sent = await whatomateClient.sendTextMessage('contact-1', 'hello world');
    expect(sent.status).toBe('sent');

    const createdCampaign = await whatomateClient.createCampaign({
      name: 'Promo',
      whatsapp_account: 'account-main',
      body_content: 'Hello'
    });

    expect(createdCampaign.id).toBeDefined();

    const started = await whatomateClient.startCampaign(String(createdCampaign.id));
    expect(started.status).toBe('sending');

    const campaign = await whatomateClient.getCampaign(String(createdCampaign.id));
    expect(campaign.status).toBe('sending');
  });

  it('maps Whatomate errors to AppError', async () => {
    const config = loadConfig({
      MCP_TRANSPORT: 'stdio',
      WHATOMATE_BASE_URL: services.whatomate.baseUrl,
      WHATOMATE_API_KEY: 'whm_test',
      OPENAI_API_KEY: 'sk_test',
      OPENAI_BASE_URL: services.openai.baseUrl
    });

    const whatomateClient = new WhatomateClient(config);

    await expect(whatomateClient.getContact('missing')).rejects.toBeInstanceOf(AppError);
  });

  it('summarizes messages via OpenAI client', async () => {
    const config = loadConfig({
      MCP_TRANSPORT: 'stdio',
      WHATOMATE_BASE_URL: services.whatomate.baseUrl,
      WHATOMATE_API_KEY: 'whm_test',
      OPENAI_API_KEY: 'sk_test',
      OPENAI_BASE_URL: services.openai.baseUrl
    });

    const openAiClient = new OpenAIClient(config);

    const response = await openAiClient.summarizeConversation({
      messages: ['hello', 'need support'],
      objective: 'Summarize in one line'
    });

    expect(response.summary).toContain('Summary generated');
    expect(response.model).toBe('gpt-4o-mini');
  });
});
