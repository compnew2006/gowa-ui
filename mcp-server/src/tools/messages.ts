import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { wrapToolHandler } from './result.js';

export const listMessagesArgsSchema = z.object({
  contact_id: z.string().min(1),
  page: z.number().int().min(1).default(1),
  limit: z.number().int().min(1).max(100).default(50),
  before_id: z.string().optional(),
  account: z.string().optional()
}).strict();

export const sendTextMessageArgsSchema = z.object({
  contact_id: z.string().min(1),
  text: z.string().min(1).max(4096),
  reply_to_message_id: z.string().optional(),
  instance_id: z.string().optional(),
  whatsapp_account: z.string().optional()
}).strict();

export function registerMessageTools(server: McpServer, deps: McpServerDependencies): void {
  server.registerTool(
    'whatomate_list_messages',
    {
      title: 'List Messages',
      description: 'List messages for a contact conversation.',
      inputSchema: listMessagesArgsSchema.shape
    },
    wrapToolHandler('whatomate_list_messages', listMessagesArgsSchema, deps.logger, (args) =>
      deps.whatomateClient.listMessages(args.contact_id, {
        page: args.page,
        limit: args.limit,
        before_id: args.before_id,
        account: args.account
      })
    )
  );

  server.registerTool(
    'whatomate_send_text_message',
    {
      title: 'Send Text Message',
      description: 'Send a text message to a contact.',
      inputSchema: sendTextMessageArgsSchema.shape
    },
    wrapToolHandler('whatomate_send_text_message', sendTextMessageArgsSchema, deps.logger, (args) =>
      deps.whatomateClient.sendTextMessage(args.contact_id, args.text, {
        reply_to_message_id: args.reply_to_message_id,
        instance_id: args.instance_id,
        whatsapp_account: args.whatsapp_account
      })
    )
  );
}
