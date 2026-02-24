import { describe, expect, it } from 'vitest';
import { AppError, formatSafeErrorMessage, toToolErrorResult } from '../../src/errors.js';

describe('errors', () => {
  it('formats exposed AppError messages', () => {
    const error = new AppError({
      code: 'BAD_INPUT',
      message: 'Invalid value',
      exposeMessage: true,
      httpStatus: 400
    });

    expect(formatSafeErrorMessage(error)).toBe('[BAD_INPUT] Invalid value');
  });

  it('sanitizes unknown errors', () => {
    expect(formatSafeErrorMessage(new Error('secret detail'))).toBe('[INTERNAL_ERROR] Operation failed');
  });

  it('returns MCP tool error payloads', () => {
    const result = toToolErrorResult(new AppError({
      code: 'NOT_FOUND',
      message: 'Not found',
      exposeMessage: true,
      httpStatus: 404
    }));

    expect(result.isError).toBe(true);
    expect(result.content[0]?.type).toBe('text');
    expect((result.content[0] as { text?: string }).text).toContain('NOT_FOUND');
  });
});
