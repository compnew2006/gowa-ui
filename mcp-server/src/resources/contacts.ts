import * as z from 'zod/v4';
import { ResourceTemplate, type McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import type { McpServerDependencies } from '../mcp/types.js';

const resourceLimitSchema = z.coerce.number().int().min(1).max(100).default(25);

function asJsonResource(uri: URL, payload: unknown) {
  return {
    contents: [
      {
        uri: uri.href,
        mimeType: 'application/json',
        text: JSON.stringify(payload, null, 2)
      }
    ]
  };
}

export function registerContactResources(server: McpServer, deps: McpServerDependencies): void {
  server.registerResource(
    'whatomate_contact_by_id',
    new ResourceTemplate('whatomate://contacts/{contactId}', { list: undefined }),
    {
      title: 'Contact',
      description: 'A single contact record from Whatomate.',
      mimeType: 'application/json'
    },
    async (uri, { contactId }) => {
      const normalizedContactId = Array.isArray(contactId) ? contactId[0] : contactId;
      const result = await deps.whatomateClient.getContact(normalizedContactId);
      return asJsonResource(uri, result);
    }
  );

  server.registerResource(
    'whatomate_contact_messages',
    new ResourceTemplate('whatomate://contacts/{contactId}/messages', { list: undefined }),
    {
      title: 'Contact Messages',
      description: 'Recent messages for a specific contact.',
      mimeType: 'application/json'
    },
    async (uri, { contactId }) => {
      const normalizedContactId = Array.isArray(contactId) ? contactId[0] : contactId;
      const parsedLimit = resourceLimitSchema.parse(uri.searchParams.get('limit') ?? '25');
      const account = uri.searchParams.get('account') ?? undefined;
      const result = await deps.whatomateClient.listMessages(normalizedContactId, {
        page: 1,
        limit: parsedLimit,
        account
      });
      return asJsonResource(uri, result);
    }
  );
}
