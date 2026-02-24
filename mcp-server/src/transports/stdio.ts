import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { Logger } from '../logger.js';

export interface RunningStdioServer {
  close: () => Promise<void>;
}

export async function startStdioServer(server: McpServer, logger: Logger): Promise<RunningStdioServer> {
  const transport = new StdioServerTransport();
  await server.connect(transport);

  logger.info('Stdio transport started');

  return {
    close: async () => {
      await transport.close();
      if (typeof (server as unknown as { close?: () => Promise<void> }).close === 'function') {
        await (server as unknown as { close: () => Promise<void> }).close();
      }
      logger.info('Stdio transport stopped');
    }
  };
}
