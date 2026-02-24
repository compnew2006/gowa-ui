import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from './types.js';
import { registerAllTools } from './tool-registry.js';
import { registerAllResources } from './resource-registry.js';
import { registerAllPrompts } from './prompt-registry.js';

export function createWhatomateMcpServer(deps: McpServerDependencies): McpServer {
  const server = new McpServer(
    {
      name: 'whatomate-mcp-server',
      version: '0.1.0'
    },
    {
      capabilities: {
        logging: {}
      }
    }
  );

  registerAllTools(server, deps);
  registerAllResources(server, deps);
  registerAllPrompts(server);

  deps.logger.info('MCP server initialized', {
    tools: 9,
    resources: 5,
    prompts: 3
  });

  return server;
}
