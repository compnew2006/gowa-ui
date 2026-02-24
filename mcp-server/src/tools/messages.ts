import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { toToolErrorResult } from '../errors.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { toToolSuccessResult } from './result.js';

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
    async (rawArgs) => {
      try {
        const args = listMessagesArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.listMessages(args.contact_id, {
          page: args.page,
          limit: args.limit,
          before_id: args.before_id,
          account: args.account
        });
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_list_messages', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );

  server.registerTool(
    'whatomate_send_text_message',
    {
      title: 'Send Text Message',
      description: 'Send a text message to a contact.',
      inputSchema: sendTextMessageArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = sendTextMessageArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.sendTextMessage(args.contact_id, args.text, {
          reply_to_message_id: args.reply_to_message_id,
          instance_id: args.instance_id,
          whatsapp_account: args.whatsapp_account
        });
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_send_text_message', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );
}
