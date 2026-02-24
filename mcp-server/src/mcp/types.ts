import type { Logger } from '../logger.js';
import type { WhatomateClient } from '../clients/whatomate-client.js';
import type { OpenAIClient } from '../clients/openai-client.js';

export interface McpServerDependencies {
  logger: Logger;
  whatomateClient: WhatomateClient;
  openAiClient: OpenAIClient;
}
