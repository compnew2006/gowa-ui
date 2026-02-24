import { randomUUID } from 'node:crypto';
import type { Server as HttpServer } from 'node:http';
import type { AddressInfo } from 'node:net';
import type { Express, NextFunction, Request, RequestHandler, Response } from 'express';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { createMcpExpressApp } from '@modelcontextprotocol/sdk/server/express.js';
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import { isInitializeRequest } from '@modelcontextprotocol/sdk/types.js';
import type { Logger } from '../logger.js';
import { registerLegacySseRoutes } from './legacy-sse.js';

interface StreamableEntry {
  transport: StreamableHTTPServerTransport;
  server: McpServer;
}

export interface StartHttpServerOptions {
  host: string;
  port: number;
  bearerToken: string;
  allowedHosts: string[];
  enableLegacySse: boolean;
  createServer: () => McpServer;
  logger: Logger;
}

export interface RunningHttpServer {
  app: Express;
  baseUrl: string;
  close: () => Promise<void>;
}

function createBearerAuthMiddleware(token: string): RequestHandler {
  return (req: Request, res: Response, next: NextFunction) => {
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      res.status(401).json({ error: 'Missing bearer token' });
      return;
    }

    const providedToken = authHeader.slice('Bearer '.length).trim();
    if (providedToken !== token) {
      res.status(403).json({ error: 'Invalid bearer token' });
      return;
    }

    next();
  };
}

async function closeMcpServer(server: McpServer): Promise<void> {
  if (typeof (server as unknown as { close?: () => Promise<void> }).close === 'function') {
    await (server as unknown as { close: () => Promise<void> }).close();
  }
}

export async function startHttpServer(options: StartHttpServerOptions): Promise<RunningHttpServer> {
  const app = createMcpExpressApp({
    host: options.host,
    allowedHosts: options.allowedHosts
  });

  app.get('/healthz', (_req, res) => {
    res.status(200).json({ status: 'ok' });
  });

  const requireAuth = createBearerAuthMiddleware(options.bearerToken);

  const streamableEntries = new Map<string, StreamableEntry>();

  app.all('/mcp', requireAuth, async (req: Request, res: Response) => {
    try {
      const sessionIdHeader = req.headers['mcp-session-id'];
      const sessionId = typeof sessionIdHeader === 'string' ? sessionIdHeader : undefined;

      if (sessionId) {
        const existing = streamableEntries.get(sessionId);
        if (!existing) {
          res.status(404).json({
            jsonrpc: '2.0',
            error: {
              code: -32001,
              message: 'Session not found'
            },
            id: null
          });
          return;
        }

        await existing.transport.handleRequest(req, res, req.body);
        return;
      }

      if (req.method !== 'POST' || !isInitializeRequest(req.body)) {
        res.status(400).json({
          jsonrpc: '2.0',
          error: {
            code: -32000,
            message: 'Session not initialized'
          },
          id: null
        });
        return;
      }

      const server = options.createServer();
      const transport = new StreamableHTTPServerTransport({
        sessionIdGenerator: () => randomUUID(),
        onsessioninitialized: (sessionIdValue) => {
          streamableEntries.set(sessionIdValue, {
            transport,
            server
          });
          options.logger.info('Streamable HTTP session initialized', { sessionId: sessionIdValue });
        }
      });

      transport.onclose = () => {
        const sessionKey = transport.sessionId;
        if (sessionKey) {
          streamableEntries.delete(sessionKey);
          options.logger.info('Streamable HTTP session closed', { sessionId: sessionKey });
        }
      };

      await server.connect(transport);
      await transport.handleRequest(req, res, req.body);
    } catch (error) {
      options.logger.error('Failed to process /mcp request', { error: String(error) });
      if (!res.headersSent) {
        res.status(500).json({
          jsonrpc: '2.0',
          error: {
            code: -32603,
            message: 'Internal server error'
          },
          id: null
        });
      }
    }
  });

  let closeLegacyRoutes: (() => Promise<void>) | undefined;
  if (options.enableLegacySse) {
    closeLegacyRoutes = registerLegacySseRoutes({
      app,
      createServer: options.createServer,
      requireAuth,
      logger: options.logger
    });
  }

  const httpServer = await new Promise<HttpServer>((resolve, reject) => {
    const server = app.listen(options.port, options.host, () => resolve(server));
    server.on('error', reject);
  });

  const address = httpServer.address();
  if (!address || typeof address === 'string') {
    throw new Error('Failed to resolve HTTP server address');
  }
  const { port: boundPort } = address as AddressInfo;
  const baseUrl = `http://${options.host}:${boundPort}`;

  options.logger.info('HTTP transport started', {
    host: options.host,
    port: boundPort,
    legacy_sse: options.enableLegacySse
  });

  return {
    app,
    baseUrl,
    close: async () => {
      if (closeLegacyRoutes) {
        await closeLegacyRoutes();
      }

      const entries = Array.from(streamableEntries.values());
      streamableEntries.clear();
      for (const entry of entries) {
        await entry.transport.close();
        await closeMcpServer(entry.server);
      }

      await new Promise<void>((resolve, reject) => {
        httpServer.close((error) => {
          if (error) {
            reject(error);
            return;
          }
          resolve();
        });
      });

      options.logger.info('HTTP transport stopped');
    }
  };
}
