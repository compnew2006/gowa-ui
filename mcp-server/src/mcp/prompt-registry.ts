import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { registerDraftReplyPrompt } from '../prompts/draft-reply.js';
import { registerCampaignBriefPrompt } from '../prompts/campaign-brief.js';
import { registerHandoffSummaryPrompt } from '../prompts/handoff-summary.js';

export function registerAllPrompts(server: McpServer): void {
  registerDraftReplyPrompt(server);
  registerCampaignBriefPrompt(server);
  registerHandoffSummaryPrompt(server);
}
