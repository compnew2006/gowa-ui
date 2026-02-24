import { ResourceTemplate, type McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
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

export function registerCampaignResources(server: McpServer, deps: McpServerDependencies): void {
  server.registerResource(
    'whatomate_campaign_by_id',
    new ResourceTemplate('whatomate://campaigns/{campaignId}', { list: undefined }),
    {
      title: 'Campaign',
      description: 'Campaign details and counters from Whatomate.',
      mimeType: 'application/json'
    },
    async (uri, { campaignId }) => {
      const normalizedCampaignId = Array.isArray(campaignId) ? campaignId[0] : campaignId;
      const result = await deps.whatomateClient.getCampaign(normalizedCampaignId);
      return asJsonResource(uri, result);
    }
  );
}
