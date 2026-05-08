import * as z from 'zod/v4';
import type { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from '../mcp/types.js';
import { wrapToolHandler } from './result.js';

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
    wrapToolHandler('whatomate_list_contacts', listContactsArgsSchema, deps.logger, (args) =>
      deps.whatomateClient.listContacts(args)
    )
  );

  server.registerTool(
    'whatomate_get_contact',
    {
      title: 'Get Contact',
      description: 'Fetch a single contact by ID.',
      inputSchema: getContactArgsSchema.shape
    },
    wrapToolHandler('whatomate_get_contact', getContactArgsSchema, deps.logger, (args) =>
      deps.whatomateClient.getContact(args.contact_id)
    )
  );
}
