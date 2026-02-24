import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { toToolErrorResult } from '../errors.js';
import type { MessageRecord } from '../clients/whatomate-client.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { toToolSuccessResult } from './result.js';

export const summarizeConversationArgsSchema = z.object({
  contact_id: z.string().min(1),
  account: z.string().optional(),
  limit: z.number().int().min(1).max(100).default(25),
  objective: z.string().max(1000).optional()
}).strict();

function extractMessageBody(message: MessageRecord): string {
  const content = message.content;
  if (typeof content === 'string') {
    return content;
  }

  if (content && typeof content === 'object') {
    if ('body' in content && typeof (content as { body?: unknown }).body === 'string') {
      return (content as { body: string }).body;
    }

    return JSON.stringify(content);
  }

  return '';
}

export function registerOpenAITools(server: McpServer, deps: McpServerDependencies): void {
  server.registerTool(
    'whatomate_openai_summarize_conversation',
    {
      title: 'Summarize Conversation (OpenAI)',
      description: 'Summarize a contact conversation using OpenAI.',
      inputSchema: summarizeConversationArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = summarizeConversationArgsSchema.parse(rawArgs ?? {});
        const messages = await deps.whatomateClient.listMessages(args.contact_id, {
          page: 1,
          limit: args.limit,
          account: args.account
        });

        const normalized = messages.items
          .map(extractMessageBody)
          .map((body) => body.trim())
          .filter((body) => body.length > 0);

        const summary = await deps.openAiClient.summarizeConversation({
          messages: normalized,
          objective: args.objective
        });

        return toToolSuccessResult({
          contact_id: args.contact_id,
          messages_considered: normalized.length,
          summary: summary.summary,
          model: summary.model
        });
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_openai_summarize_conversation', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );
}
