import type { CallToolResult } from '@modelcontextprotocol/sdk/types.js';
import { toToolErrorResult } from '../errors.js';

export function toToolSuccessResult(structuredContent: unknown): CallToolResult {
  const normalizedStructuredContent =
    structuredContent && typeof structuredContent === 'object' && !Array.isArray(structuredContent)
      ? (structuredContent as Record<string, unknown>)
      : { value: structuredContent };

  return {
    content: [
      {
        type: 'text',
        text: JSON.stringify(structuredContent, null, 2)
      }
    ],
    structuredContent: normalizedStructuredContent
  };
}

type Logger = { error: (msg: string, meta?: Record<string, unknown>) => void };

export function wrapToolHandler<TArgs>(
  toolName: string,
  schema: { parse: (v: unknown) => TArgs },
  logger: Logger,
  handler: (args: TArgs) => Promise<unknown>
): (rawArgs: unknown) => Promise<CallToolResult> {
  return async (rawArgs) => {
    try {
      const args = schema.parse(rawArgs ?? {});
      const result = await handler(args);
      return toToolSuccessResult(result);
    } catch (error) {
      logger.error(`Tool failed: ${toolName}`, { error: String(error) });
      return toToolErrorResult(error);
    }
  };
}
