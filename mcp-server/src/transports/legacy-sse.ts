import type { Express, NextFunction, Request, RequestHandler, Response } from 'express';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { SSEServerTransport } from '@modelcontextprotocol/sdk/server/sse.js';
import type { Logger } from '../logger.js';

interface LegacySseEntry {
  transport: SSEServerTransport;
  server: McpServer;
}

interface LegacySseOptions {
  app: Express;
  logger: Logger;
  requireAuth: RequestHandler;
  createServer: () => McpServer;
}

async function closeMcpServer(server: McpServer): Promise<void> {
  if (typeof (server as unknown as { close?: () => Promise<void> }).close === 'function') {
    await (server as unknown as { close: () => Promise<void> }).close();
  }
}

export function registerLegacySseRoutes(options: LegacySseOptions): () => Promise<void> {
  const entries = new Map<string, LegacySseEntry>();

  options.app.get('/sse', options.requireAuth, async (_req: Request, res: Response, next: NextFunction) => {
    try {
      const server = options.createServer();
      const transport = new SSEServerTransport('/messages', res);
      const sessionId = transport.sessionId;

      entries.set(sessionId, { transport, server });
      transport.onclose = () => {
        options.logger.info('Legacy SSE session closed', { sessionId });
        entries.delete(sessionId);
      };

      await server.connect(transport);
      options.logger.info('Legacy SSE session started', { sessionId });
    } catch (error) {
      next(error);
    }
  });

  options.app.post('/messages', options.requireAuth, async (req: Request, res: Response, next: NextFunction) => {
    try {
      const sessionId = typeof req.query.sessionId === 'string' ? req.query.sessionId : undefined;
      if (!sessionId) {
        res.status(400).json({ error: 'Missing sessionId query parameter' });
        return;
      }

      const entry = entries.get(sessionId);
      if (!entry) {
        res.status(404).json({ error: 'Session not found' });
        return;
      }

      await entry.transport.handlePostMessage(req, res, req.body);
    } catch (error) {
      next(error);
    }
  });

  return async () => {
    const allEntries = Array.from(entries.values());
    entries.clear();

    for (const entry of allEntries) {
      await entry.transport.close();
      await closeMcpServer(entry.server);
    }

    options.logger.info('Legacy SSE routes stopped');
  };
}
