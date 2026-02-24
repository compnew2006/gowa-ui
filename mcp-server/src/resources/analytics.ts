import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { ResourceTemplate } from '@modelcontextprotocol/sdk/server/mcp.js';
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

export function registerAnalyticsResources(server: McpServer, deps: McpServerDependencies): void {
  server.registerResource(
    'whatomate_dashboard_analytics',
    new ResourceTemplate('whatomate://analytics/dashboard', { list: undefined }),
    {
      title: 'Dashboard Analytics',
      description: 'Dashboard analytics snapshot from Whatomate.',
      mimeType: 'application/json'
    },
    async (uri) => {
      const periodValue = uri.searchParams.get('period') ?? 'week';
      const period = (periodValue === 'today' || periodValue === 'week' || periodValue === 'month' || periodValue === 'year')
        ? periodValue
        : 'week';

      const accountId = uri.searchParams.get('account_id') ?? undefined;
      const result = await deps.whatomateClient.getDashboardAnalytics({
        period,
        account_id: accountId
      });
      return asJsonResource(uri, result);
    }
  );
}
