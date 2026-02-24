import { URL } from 'node:url';
import type { AppConfig } from '../config.js';
import { AppError } from '../errors.js';

const DEFAULT_ALLOWED_MODELS = new Set([
  'gpt-4o-mini',
  'gpt-4o',
  'gpt-4.1-mini'
]);

export interface SummarizeConversationInput {
  messages: string[];
  objective?: string;
}

export class OpenAIClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly model: string;

  constructor(private readonly config: AppConfig) {
    const host = new URL(config.openAiBaseUrl).hostname;
    if (!config.outboundAllowedHosts.includes(host)) {
      throw new Error(`OPENAI_BASE_URL host is not in outbound allowlist: ${host}`);
    }

    this.baseUrl = config.openAiBaseUrl;
    this.apiKey = config.openAiApiKey;
    this.model = config.openAiModel;
  }

  async summarizeConversation(input: SummarizeConversationInput): Promise<{ summary: string; model: string }> {
    if (!DEFAULT_ALLOWED_MODELS.has(this.model)) {
      throw new AppError({
        code: 'OPENAI_MODEL_NOT_ALLOWED',
        message: `Model is not allowed: ${this.model}`,
        exposeMessage: true,
        httpStatus: 400
      });
    }

    const joinedMessages = input.messages
      .map((message, index) => `Message ${index + 1}: ${message}`)
      .join('\n')
      .slice(0, 12000);

    const objective = input.objective ?? 'Summarize the conversation and highlight action items.';

    const response = await fetch(`${this.baseUrl}/v1/chat/completions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.apiKey}`
      },
      body: JSON.stringify({
        model: this.model,
        max_tokens: 500,
        temperature: 0.2,
        messages: [
          {
            role: 'system',
            content: 'You are a concise assistant generating operational summaries for WhatsApp support teams.'
          },
          {
            role: 'user',
            content: `${objective}\n\nConversation:\n${joinedMessages}`
          }
        ]
      }),
      signal: AbortSignal.timeout(this.config.requestTimeoutMs)
    });

    if (!response.ok) {
      const body = await response.text();
      throw new AppError({
        code: 'OPENAI_REQUEST_FAILED',
        message: `OpenAI request failed with status ${response.status}: ${body.slice(0, 200)}`,
        httpStatus: response.status,
        exposeMessage: response.status >= 400 && response.status < 500
      });
    }

    const payload = await response.json() as {
      choices?: Array<{ message?: { content?: string } }>;
    };

    const summary = payload.choices?.[0]?.message?.content?.trim();
    if (!summary) {
      throw new AppError({
        code: 'OPENAI_EMPTY_RESPONSE',
        message: 'OpenAI returned an empty response',
        httpStatus: 502,
        exposeMessage: false
      });
    }

    return {
      summary,
      model: this.model
    };
  }
}
