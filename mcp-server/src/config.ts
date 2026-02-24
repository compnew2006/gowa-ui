import * as z from 'zod/v4';
import { URL } from 'node:url';

export type TransportMode = 'stdio' | 'http' | 'hybrid';
export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const booleanFromEnv = z.preprocess((value) => {
  if (typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase();
    if (normalized === 'true' || normalized === '1' || normalized === 'yes') {
      return true;
    }
    if (normalized === 'false' || normalized === '0' || normalized === 'no') {
      return false;
    }
  }
  return value;
}, z.boolean());

const intFromEnv = z.preprocess((value) => {
  if (typeof value === 'number') {
    return value;
  }
  if (typeof value === 'string' && value.trim() !== '') {
    return Number.parseInt(value, 10);
  }
  return value;
}, z.number().int());

const envSchema = z.object({
  MCP_TRANSPORT: z.enum(['stdio', 'http', 'hybrid']).default('stdio'),
  MCP_HTTP_HOST: z.string().default('127.0.0.1'),
  MCP_HTTP_PORT: intFromEnv.default(3000),
  MCP_HTTP_BEARER_TOKEN: z.string().optional(),
  MCP_ENABLE_LEGACY_SSE: booleanFromEnv.default(false),
  MCP_HTTP_ALLOWED_HOSTS: z.string().optional(),

  WHATOMATE_BASE_URL: z.string().min(1),
  WHATOMATE_API_KEY: z.string().min(1),
  WHATOMATE_ORGANIZATION_ID: z.string().optional(),

  OPENAI_API_KEY: z.string().min(1),
  OPENAI_MODEL: z.string().default('gpt-4o-mini'),
  OPENAI_BASE_URL: z.string().default('https://api.openai.com'),

  LOG_LEVEL: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
  LOG_FILE: z.string().optional(),

  MCP_REQUEST_TIMEOUT_MS: intFromEnv.default(30000),
  MCP_GET_RETRIES: intFromEnv.default(2),
  MCP_OUTBOUND_HOST_ALLOWLIST: z.string().optional()
}).passthrough();

export interface AppConfig {
  transport: TransportMode;
  httpHost: string;
  httpPort: number;
  httpBearerToken?: string;
  enableLegacySse: boolean;
  httpAllowedHosts: string[];

  whatomateApiBaseUrl: string;
  whatomateApiKey: string;
  whatomateOrganizationId?: string;

  openAiApiKey: string;
  openAiModel: string;
  openAiBaseUrl: string;

  logLevel: LogLevel;
  logFile?: string;

  requestTimeoutMs: number;
  getRetries: number;
  outboundAllowedHosts: string[];
}

function parseCsv(input: string | undefined): string[] {
  if (!input) {
    return [];
  }

  return input
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function normalizeWhatomateApiBaseUrl(baseUrl: string): string {
  const trimmed = baseUrl.replace(/\/+$/, '');
  if (trimmed.endsWith('/api')) {
    return trimmed;
  }
  return `${trimmed}/api`;
}

function getHostFromUrl(urlValue: string): string {
  return new URL(urlValue).hostname;
}

export function loadConfig(rawEnv: NodeJS.ProcessEnv = process.env): AppConfig {
  const parsedEnv = envSchema.parse(rawEnv);

  if ((parsedEnv.MCP_TRANSPORT === 'http' || parsedEnv.MCP_TRANSPORT === 'hybrid') && !parsedEnv.MCP_HTTP_BEARER_TOKEN) {
    throw new Error('MCP_HTTP_BEARER_TOKEN is required when MCP_TRANSPORT is http or hybrid');
  }

  const whatomateApiBaseUrl = normalizeWhatomateApiBaseUrl(parsedEnv.WHATOMATE_BASE_URL);
  const openAiBaseUrl = parsedEnv.OPENAI_BASE_URL.replace(/\/+$/, '');

  const allowedHosts = new Set<string>([
    getHostFromUrl(whatomateApiBaseUrl),
    getHostFromUrl(openAiBaseUrl),
    ...parseCsv(parsedEnv.MCP_OUTBOUND_HOST_ALLOWLIST)
  ]);

  const httpAllowedHosts = parseCsv(parsedEnv.MCP_HTTP_ALLOWED_HOSTS);
  if (httpAllowedHosts.length === 0) {
    httpAllowedHosts.push(parsedEnv.MCP_HTTP_HOST, 'localhost', '127.0.0.1');
  }

  const logFile = parsedEnv.LOG_FILE ?? (parsedEnv.MCP_TRANSPORT === 'stdio' ? '/tmp/whatomate-mcp.log' : undefined);

  return {
    transport: parsedEnv.MCP_TRANSPORT,
    httpHost: parsedEnv.MCP_HTTP_HOST,
    httpPort: parsedEnv.MCP_HTTP_PORT,
    httpBearerToken: parsedEnv.MCP_HTTP_BEARER_TOKEN,
    enableLegacySse: parsedEnv.MCP_ENABLE_LEGACY_SSE,
    httpAllowedHosts,

    whatomateApiBaseUrl,
    whatomateApiKey: parsedEnv.WHATOMATE_API_KEY,
    whatomateOrganizationId: parsedEnv.WHATOMATE_ORGANIZATION_ID,

    openAiApiKey: parsedEnv.OPENAI_API_KEY,
    openAiModel: parsedEnv.OPENAI_MODEL,
    openAiBaseUrl,

    logLevel: parsedEnv.LOG_LEVEL,
    logFile,

    requestTimeoutMs: parsedEnv.MCP_REQUEST_TIMEOUT_MS,
    getRetries: Math.max(0, parsedEnv.MCP_GET_RETRIES),
    outboundAllowedHosts: Array.from(allowedHosts)
  };
}
