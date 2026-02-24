import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from '../mcp/types.js';

function asJsonResource(uri: URL, payload: unknown) {
  return {
    contents: [
      {
        uri: uri.href,
        mimeType: 'application/json',
        text: JSON.stringify(payload, null, 2)
      }
    ]
  };
}

export function registerOrganizationResources(server: McpServer, deps: McpServerDependencies): void {
  server.registerResource(
    'whatomate_organization_current',
    'whatomate://organization/current',
    {
      title: 'Current Organization',
      description: 'Current organization context from Whatomate.',
      mimeType: 'application/json'
    },
    async (uri) => {
      const result = await deps.whatomateClient.getCurrentOrganization();
      return asJsonResource(uri, result);
    }
  );
}
