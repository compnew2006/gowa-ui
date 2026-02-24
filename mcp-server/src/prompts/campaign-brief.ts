import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';

const campaignBriefArgsSchema = {
  campaign_goal: z.string().min(1),
  audience: z.string().min(1),
  offer: z.string().min(1),
  tone: z.enum(['professional', 'friendly', 'urgent']).default('friendly')
};

export function registerCampaignBriefPrompt(server: McpServer): void {
  server.registerPrompt(
    'whatomate_campaign_brief',
    {
      title: 'Campaign Brief',
      description: 'Generate a campaign brief for WhatsApp outreach.',
      argsSchema: campaignBriefArgsSchema
    },
    (args) => {
      const text = [
        'Create a WhatsApp campaign brief with the following inputs:',
        `Goal: ${args.campaign_goal}`,
        `Audience: ${args.audience}`,
        `Offer: ${args.offer}`,
        `Tone: ${args.tone}`,
        'Output format:',
        '1. One-line campaign thesis',
        '2. Key message points (3 bullets)',
        '3. Risks and mitigations (2 bullets)',
        '4. Suggested CTA variants (3 options)'
      ].join('\n');

      return {
        messages: [
          {
            role: 'user',
            content: {
              type: 'text',
              text
            }
          }
        ]
      };
    }
  );
}
