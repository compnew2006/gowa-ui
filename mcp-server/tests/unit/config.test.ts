import { describe, expect, it } from 'vitest';
import { loadConfig } from '../../src/config.js';

function baseEnv(): NodeJS.ProcessEnv {
  return {
    MCP_TRANSPORT: 'stdio',
    WHATOMATE_BASE_URL: 'http://localhost:8080',
    WHATOMATE_API_KEY: 'whm_test',
    OPENAI_API_KEY: 'sk_test'
  };
}

describe('loadConfig', () => {
  it('loads defaults and normalizes Whatomate /api base URL', () => {
    const config = loadConfig(baseEnv());

    expect(config.transport).toBe('stdio');
    expect(config.whatomateApiBaseUrl).toBe('http://localhost:8080/api');
    expect(config.openAiModel).toBe('gpt-4o-mini');
    expect(config.logFile).toBe('/tmp/whatomate-mcp.log');
  });

  it('requires bearer token for http transport', () => {
    expect(() => loadConfig({
      ...baseEnv(),
      MCP_TRANSPORT: 'http'
    })).toThrow(/MCP_HTTP_BEARER_TOKEN/);
  });

  it('accepts explicit /api URL and custom retries', () => {
    const config = loadConfig({
      ...baseEnv(),
      WHATOMATE_BASE_URL: 'https://example.com/api',
      MCP_GET_RETRIES: '5'
    });

    expect(config.whatomateApiBaseUrl).toBe('https://example.com/api');
    expect(config.getRetries).toBe(5);
  });
});
