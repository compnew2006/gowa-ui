# Whatomate Reverse Spec (Chatbot AI + Transfers)

## 1) Technology Stack & Architecture (Observed)
- Backend: Go services with handlers under `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers` and models in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models`.
- Frontend: Vue 3 app under `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src` with views and services.
- Data access: GORM models with PostgreSQL in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models` and DB initialization in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/database`.
- Cache: Redis-based caching in `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/cache.go`.
- HTTP routing: `cmd/whatomate/main.go` registers REST routes, including chatbot endpoints.

## 2) Module/Directory Structure (Observed)
- Chatbot API endpoints: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot.go`.
- Chatbot runtime/processing: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go`.
- Agent transfers: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/agent_transfers.go`.
- Chatbot models: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/chatbot.go`.
- AI provider constants: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/constants.go`.
- Frontend Chatbot settings: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src/views/settings/ChatbotSettingsView.vue`.
- Frontend AI contexts: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src/views/chatbot/AIContextsView.vue`.

## 3) Observed Requirements (EARS)
- When a chatbot message arrives and an active agent transfer exists, the system shall skip chatbot processing. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (function `processIncomingMessageFull`).
- When chatbot is disabled for an account, the system shall create an agent transfer to the queue. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (check for `settings.IsEnabled` and `createTransferToQueue`).
- When a transfer keyword is matched, the system shall send the transfer message and create a transfer, subject to business hours. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (keyword transfer flow).
- When a flow is triggered by keywords, the system shall start a chatbot flow session. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (`matchFlowTrigger` and `startFlow`).
- When an AI response is enabled and configured, the system shall call the configured provider to generate a response. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (`generateAIResponse`, `generateOpenAIResponse`, `generateAnthropicResponse`, `generateGoogleResponse`).
- When an AI system prompt is configured, the system shall include it in provider-specific system instructions. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (`systemPrompt` usage in OpenAI/Anthropic/Google handlers).
- When AI contexts are enabled, the system shall append all enabled AI context content to the system prompt. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (`buildAIContext`) and `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/cache.go` (`getAIContextsCached`).
- When an AI API key is provided, the system shall encrypt it before storage. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot.go` (Encrypt on update) and `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/cache.go` (Decrypt on use).
- When a flow step type is `transfer`, the system shall create a transfer to a team or general queue and end the flow session. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (transfer step handling).
- When a keyword transfer is created and `assign_to_same_agent` is enabled, the system shall assign the transfer to the contact’s existing agent if available. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/agent_transfers.go` (`createTransferFromKeyword`).

## 4) Non-Functional Observations (Security/Reliability)
- AI API keys are encrypted at rest and decrypted before use. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot.go` and `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/cache.go`.
- Chatbot settings are cached in Redis with a TTL and invalidated on update. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/cache.go`.
- Chatbot AI settings require permissions via `authorizeChatbotRequest`. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot.go`.

## 5) Inferred Acceptance Criteria (Derived from Observed Behavior)
- The system should provide consistent AI responses using a single system prompt plus appended context data. Inference based on `buildAIContext` and provider-specific system prompt usage. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go`.
- The chatbot should stop automated responses after any transfer is active. Inference based on transfer gating at the start of processing. Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go`.

## 6) Uncertainties & Questions
- Trigger keywords in AI contexts are stored but are not applied in `buildAIContext`; is keyword-based inclusion intended? Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/chatbot.go` (TriggerKeywords) vs `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` (no filter).
- There is no observed AI-driven routing to a specific agent; only flow/team or queue transfer exists. Is AI-based agent routing planned? Evidence: `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go` and `/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/agent_transfers.go`.

## 7) Recommendations (High-Level)
- Add a structured “instruction set” layer to support multiple user-configurable instruction blocks with scoping and precedence.
- Add agent-routing detection and direct assignment to specific agents when explicitly requested by the user, with permission checks.
- Add provider abstractions to support local LLMs (Ollama, LM Studio) without duplicating logic.

