import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';

const handoffSummaryArgsSchema = {
  contact_name: z.string().optional(),
  issue_summary: z.string().min(1),
  latest_messages: z.string().min(1),
  next_action: z.string().min(1)
};

export function registerHandoffSummaryPrompt(server: McpServer): void {
  server.registerPrompt(
    'whatomate_handoff_summary',
    {
      title: 'Handoff Summary',
      description: 'Create an agent handoff summary.',
      argsSchema: handoffSummaryArgsSchema
    },
    (args) => {
      const contactName = args.contact_name ?? 'contact';
      const text = [
        `Produce a concise handoff summary for ${contactName}.`,
        `Issue summary: ${args.issue_summary}`,
        'Latest messages:',
        args.latest_messages,
        `Next action: ${args.next_action}`,
        'Output fields:',
        '- Situation',
        '- Critical facts',
        '- Pending tasks',
        '- Recommended immediate reply'
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
