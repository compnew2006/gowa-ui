import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from './types.js';
import { registerOrganizationResources } from '../resources/organization.js';
import { registerContactResources } from '../resources/contacts.js';
import { registerCampaignResources } from '../resources/campaigns.js';
import { registerAnalyticsResources } from '../resources/analytics.js';

export function registerAllResources(server: McpServer, deps: McpServerDependencies): void {
  registerOrganizationResources(server, deps);
  registerContactResources(server, deps);
  registerCampaignResources(server, deps);
  registerAnalyticsResources(server, deps);
}
