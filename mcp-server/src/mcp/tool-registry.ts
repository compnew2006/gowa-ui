import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from './types.js';
import { registerContactTools } from '../tools/contacts.js';
import { registerMessageTools } from '../tools/messages.js';
import { registerCampaignTools } from '../tools/campaigns.js';
import { registerAnalyticsTools } from '../tools/analytics.js';
import { registerOpenAITools } from '../tools/openai.js';

export function registerAllTools(server: McpServer, deps: McpServerDependencies): void {
  registerContactTools(server, deps);
  registerMessageTools(server, deps);
  registerCampaignTools(server, deps);
  registerAnalyticsTools(server, deps);
  registerOpenAITools(server, deps);
}
