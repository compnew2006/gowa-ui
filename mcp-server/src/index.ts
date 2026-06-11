import { Logger } from './logger.js';
// import { WhatomateClient } from './clients/whatomate-client.js';
const WhatomateClient = class { constructor() {} } as any;
import { OpenAIClient } from './clients/openai-client.js';
import { createWhatomateMcpServer } from './mcp/server.js';
import { startStdioServer } from './transports/stdio.js';
import { startHttpServer } from './transports/streamable-http.js';

interface RunningComponent {
  close: () => Promise<void>;
}

async function main(): Promise<void> {
  const config = loadConfig();
  const logger = new Logger({
    level: config.logLevel,
    filePath: config.logFile,
    useStderr: config.transport !== 'stdio'
  });

  const whatomateClient = new WhatomateClient(config);
  const openAiClient = new OpenAIClient(config);

  const running: RunningComponent[] = [];

  const createServerFactory = () => createWhatomateMcpServer({
    logger,
    whatomateClient,
    openAiClient
  });

  if (config.transport === 'stdio' || config.transport === 'hybrid') {
    const stdio = await startStdioServer(createServerFactory(), logger);
    running.push(stdio);
  }

  if (config.transport === 'http' || config.transport === 'hybrid') {
    if (!config.httpBearerToken) {
      throw new Error('Missing MCP_HTTP_BEARER_TOKEN for HTTP transport');
    }

    const http = await startHttpServer({
      host: config.httpHost,
      port: config.httpPort,
      bearerToken: config.httpBearerToken,
      allowedHosts: config.httpAllowedHosts,
      enableLegacySse: config.enableLegacySse,
      createServer: createServerFactory,
      logger
    });
    running.push(http);
  }

  const shutdown = async (signal: string): Promise<void> => {
    logger.info('Shutting down MCP server', { signal });

    for (const component of running.reverse()) {
      await component.close();
    }

    process.exit(0);
  };

  process.on('SIGINT', () => {
    void shutdown('SIGINT');
  });

  process.on('SIGTERM', () => {
    void shutdown('SIGTERM');
  });

  logger.info('Whatomate MCP sidecar is running', {
    transport: config.transport,
    log_file: config.logFile ?? null
  });
}

main().catch((error) => {
  const message = error instanceof Error ? error.stack ?? error.message : String(error);
  process.stderr.write(`${message}\n`);
  process.exit(1);
});
