import type { CallToolResult } from '@modelcontextprotocol/sdk/types.js';

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
