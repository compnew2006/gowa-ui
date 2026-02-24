import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';

const draftReplyArgsSchema = {
  contact_name: z.string().optional(),
  tone: z.enum(['professional', 'friendly', 'direct']).default('professional'),
  objective: z.string().min(1),
  conversation_summary: z.string().min(1)
};

export function registerDraftReplyPrompt(server: McpServer): void {
  server.registerPrompt(
    'whatomate_draft_reply',
    {
      title: 'Draft Reply',
      description: 'Draft a WhatsApp reply using conversation context.',
      argsSchema: draftReplyArgsSchema
    },
    (args) => {
      const contactName = args.contact_name ?? 'the customer';
      const text = [
        `Draft a ${args.tone} reply for ${contactName}.`,
        `Objective: ${args.objective}`,
        'Conversation summary:',
        args.conversation_summary,
        'Constraints:',
        '- Keep it concise and actionable.',
        '- Preserve factual accuracy.',
        '- Use WhatsApp-friendly wording.'
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
