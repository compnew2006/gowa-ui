import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { toToolErrorResult } from '../errors.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { toToolSuccessResult } from './result.js';

export const listContactsArgsSchema = z.object({
  page: z.number().int().min(1).default(1),
  limit: z.number().int().min(1).max(100).default(20),
  search: z.string().optional(),
  account_id: z.string().optional()
}).strict();

export const getContactArgsSchema = z.object({
  contact_id: z.string().min(1)
}).strict();

export function registerContactTools(server: McpServer, deps: McpServerDependencies): void {
  server.registerTool(
    'whatomate_list_contacts',
    {
      title: 'List Contacts',
      description: 'List contacts from Whatomate with pagination.',
      inputSchema: listContactsArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = listContactsArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.listContacts(args);
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_list_contacts', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );

  server.registerTool(
    'whatomate_get_contact',
    {
      title: 'Get Contact',
      description: 'Fetch a single contact by ID.',
      inputSchema: getContactArgsSchema.shape
    },
    async (rawArgs) => {
      try {
        const args = getContactArgsSchema.parse(rawArgs ?? {});
        const result = await deps.whatomateClient.getContact(args.contact_id);
        return toToolSuccessResult(result);
      } catch (error) {
        deps.logger.error('Tool failed: whatomate_get_contact', { error: String(error) });
        return toToolErrorResult(error);
      }
    }
  );
}
