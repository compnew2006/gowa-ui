import type { CallToolResult } from '@modelcontextprotocol/sdk/types.js';

export class AppError extends Error {
  readonly code: string;
  readonly httpStatus: number;
  readonly exposeMessage: boolean;
  readonly details?: unknown;

  constructor(options: {
    code: string;
    message: string;
    httpStatus?: number;
    exposeMessage?: boolean;
    details?: unknown;
  }) {
    super(options.message);
    this.name = 'AppError';
    this.code = options.code;
    this.httpStatus = options.httpStatus ?? 500;
    this.exposeMessage = options.exposeMessage ?? false;
    this.details = options.details;
  }
}

export function toAppError(error: unknown, fallbackCode = 'INTERNAL_ERROR'): AppError {
  if (error instanceof AppError) {
    return error;
  }

  if (error instanceof Error) {
    return new AppError({
      code: fallbackCode,
      message: error.message,
      httpStatus: 500,
      exposeMessage: false,
      details: error
    });
  }

  return new AppError({
    code: fallbackCode,
    message: 'Unexpected error',
    httpStatus: 500,
    exposeMessage: false,
    details: error
  });
}

export function formatSafeErrorMessage(error: unknown): string {
  const appError = toAppError(error);
  if (appError.exposeMessage) {
    return `[${appError.code}] ${appError.message}`;
  }
  return `[${appError.code}] Operation failed`;
}

export function toToolErrorResult(error: unknown): CallToolResult {
  return {
    isError: true,
    content: [
      {
        type: 'text',
        text: formatSafeErrorMessage(error)
      }
    ]
  };
}
