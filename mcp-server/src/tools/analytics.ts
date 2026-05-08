import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { wrapToolHandler } from './result.js';

export const getDashboardAnalyticsArgsSchema = z.object({
  account_id: z.string().optional(),
  period: z.enum(['today', 'week', 'month', 'year']).default('week')
}).strict();

export function registerAnalyticsTools(server: McpServer, deps: McpServerDependencies): void {
  server.registerTool(
    'whatomate_get_dashboard_analytics',
    {
      title: 'Get Dashboard Analytics',
      description: 'Read dashboard analytics from Whatomate.',
      inputSchema: getDashboardAnalyticsArgsSchema.shape
    },
    wrapToolHandler('whatomate_get_dashboard_analytics', getDashboardAnalyticsArgsSchema, deps.logger, (args) =>
      deps.whatomateClient.getDashboardAnalytics(args)
    )
  );
}
