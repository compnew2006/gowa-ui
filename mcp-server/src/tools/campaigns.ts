import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { toToolErrorResult } from '../errors.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { toToolSuccessResult } from './result.js';

export const createCampaignArgsSchema = z.object({
  name: z.string().min(1),
  whatsapp_account: z.string().min(1),
  template_id: z.string().optional(),
  body_content: z.string().optional(),
  header_media_id: z.string().optional(),
  min_delay_seconds: z.number().int().min(0).optional(),
  max_delay_seconds: z.number().int().min(0).optional(),
  scheduled_at: z.string().optional()
}).strict();

export const startCampaignArgsSchema = z.object({
  campaign_id: z.string().min(1)
}).strict();

export const getCampaignStatusArgsSchema = z.object({
  campaign_id: z.string().min(1)
}).strict();

export function registerCampaignTools(server: McpServer, deps: McpServerDependencies): void {
  server.registerTool(
    'whatomate_create_campaign',
    {
      title: 'Create Campaign',
      description: 'Create a campaign draft in Whatomate.',
      inputSchema: createCampaignArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = createCampaignArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.createCampaign(args);
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_create_campaign', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );

  server.registerTool(
    'whatomate_start_campaign',
    {
      title: 'Start Campaign',
      description: 'Start an existing campaign.',
      inputSchema: startCampaignArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = startCampaignArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.startCampaign(args.campaign_id);
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_start_campaign', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );

  server.registerTool(
    'whatomate_get_campaign_status',
    {
      title: 'Get Campaign Status',
      description: 'Get campaign details and delivery counters.',
      inputSchema: getCampaignStatusArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = getCampaignStatusArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.getCampaign(args.campaign_id);
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_get_campaign_status', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );
}
