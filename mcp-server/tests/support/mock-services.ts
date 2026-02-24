import type { AddressInfo } from 'node:net';
import type { Server as HttpServer } from 'node:http';
import express, { type Express } from 'express';

interface MockServer {
  baseUrl: string;
  close: () => Promise<void>;
}

export interface MockServices {
  whatomate: MockServer;
  openai: MockServer;
}

export interface MockState {
  sentMessages: Array<{ contactId: string; body: string }>;
  campaigns: Map<string, Record<string, unknown>>;
}

function ok<T>(data: T) {
  return {
    status: 'success',
    data
  };
}

function listen(app: Express): Promise<{ server: HttpServer; baseUrl: string }> {
  return new Promise((resolve, reject) => {
    const server = app.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        reject(new Error('Failed to read server address'));
        return;
      }
      const addr = address as AddressInfo;
      resolve({ server, baseUrl: `http://127.0.0.1:${addr.port}` });
    });

    server.on('error', reject);
  });
}

async function startMockWhatomateServer(state: MockState): Promise<MockServer> {
  const app = express();
  app.use(express.json());

  const contacts = [
    {
      id: 'contact-1',
      phone_number: '+12025550100',
      name: 'Alice'
    }
  ];

  const messages = [
    {
      id: 'msg-1',
      direction: 'incoming',
      content: {
        body: 'Hello, I need an update on my order.'
      }
    },
    {
      id: 'msg-2',
      direction: 'outgoing',
      content: {
        body: 'Sure, I can help with that.'
      }
    }
  ];

  app.use('/api', (req, res, next) => {
    const apiKey = req.header('X-API-Key');
    if (!apiKey) {
      res.status(401).json({ status: 'error', message: 'Missing API key' });
      return;
    }
    next();
  });

  app.get('/api/organizations/current', (_req, res) => {
    res.json(ok({ id: 'org-1', name: 'Demo Org' }));
  });

  app.get('/api/contacts', (req, res) => {
    const page = Number.parseInt(String(req.query.page ?? '1'), 10);
    const limit = Number.parseInt(String(req.query.limit ?? '20'), 10);
    res.json(ok({ contacts, total: contacts.length, page, limit }));
  });

  app.get('/api/contacts/:id', (req, res) => {
    const contact = contacts.find((item) => item.id === req.params.id);
    if (!contact) {
      res.status(404).json({ status: 'error', message: 'Contact not found' });
      return;
    }
    res.json(ok(contact));
  });

  app.get('/api/contacts/:id/messages', (_req, res) => {
    res.json(ok({ messages, total: messages.length, page: 1, limit: messages.length, has_more: false }));
  });

  app.post('/api/contacts/:id/messages', (req, res) => {
    const body = String(req.body?.content?.body ?? '');
    state.sentMessages.push({ contactId: req.params.id, body });

    res.json(ok({
      id: `msg-${state.sentMessages.length + 2}`,
      status: 'sent',
      content: {
        body
      }
    }));
  });

  app.post('/api/campaigns', (req, res) => {
    const campaignId = `campaign-${state.campaigns.size + 1}`;
    const campaign = {
      id: campaignId,
      name: req.body?.name,
      status: 'draft',
      whatsapp_account: req.body?.whatsapp_account
    };
    state.campaigns.set(campaignId, campaign);
    res.json(ok(campaign));
  });

  app.post('/api/campaigns/:id/start', (req, res) => {
    const campaign = state.campaigns.get(req.params.id);
    if (!campaign) {
      res.status(404).json({ status: 'error', message: 'Campaign not found' });
      return;
    }
    campaign.status = 'sending';
    res.json(ok({ id: req.params.id, status: 'sending' }));
  });

  app.get('/api/campaigns/:id', (req, res) => {
    const campaign = state.campaigns.get(req.params.id);
    if (!campaign) {
      res.status(404).json({ status: 'error', message: 'Campaign not found' });
      return;
    }
    res.json(ok(campaign));
  });

  app.get('/api/analytics/dashboard', (req, res) => {
    res.json(ok({
      period: req.query.period ?? 'week',
      messages: {
        total: 2,
        sent: state.sentMessages.length
      }
    }));
  });

  const { server, baseUrl } = await listen(app);
  return {
    baseUrl,
    close: async () => {
      await new Promise<void>((resolve, reject) => {
        server.close((error) => {
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      });
    }
  };
}

async function startMockOpenAiServer(): Promise<MockServer> {
  const app = express();
  app.use(express.json());

  app.post('/v1/chat/completions', (req, res) => {
    const prompt = String(req.body?.messages?.[1]?.content ?? '');
    res.json({
      choices: [
        {
          message: {
            content: `Summary generated for prompt length ${prompt.length}`
          }
        }
      ]
    });
  });

  const { server, baseUrl } = await listen(app);
  return {
    baseUrl,
    close: async () => {
      await new Promise<void>((resolve, reject) => {
        server.close((error) => {
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      });
    }
  };
}

export async function startMockServices(): Promise<{ services: MockServices; state: MockState }> {
  const state: MockState = {
    sentMessages: [],
    campaigns: new Map<string, Record<string, unknown>>()
  };

  const whatomate = await startMockWhatomateServer(state);
  const openai = await startMockOpenAiServer();

  return {
    state,
    services: {
      whatomate,
      openai
    }
  };
}
