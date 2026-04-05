---
title: Chatbot
---

# Chatbot

Automate WhatsApp conversations with keyword rules, conversation flows, AI-powered responses, and seamless agent transfers.

## Chatbot Settings

**Endpoint:** `GET /api/chatbot/settings`

View your organization's chatbot configuration:

```json
{
  "enabled": true,
  "greeting_message": "Welcome! How can we help you?",
  "fallback_message": "I didn't understand that. Please type 'help' or wait for an agent.",
  "session_timeout_minutes": 30,
  "business_hours": {
    "monday": { "open": "09:00", "close": "17:00", "enabled": true },
    "tuesday": { "open": "09:00", "close": "17:00", "enabled": true },
    "wednesday": { "open": "09:00", "close": "17:00", "enabled": true },
    "thursday": { "open": "09:00", "close": "17:00", "enabled": true },
    "friday": { "open": "09:00", "close": "17:00", "enabled": true },
    "saturday": { "open": "10:00", "close": "14:00", "enabled": false },
    "sunday": { "enabled": false }
  },
  "ai_enabled": false,
  "sla_response_minutes": 15,
  "sla_resolution_minutes": 120,
  "sla_escalation_minutes": 60,
  "sla_auto_close_hours": 24,
  "stats": {
    "total_sessions": 1250,
    "ai_responses": 890,
    "agent_transfers": 45
  }
}
```

### Update Settings

**Endpoint:** `PUT /api/chatbot/settings`

```json
{
  "enabled": true,
  "greeting_message": "Hello! How can I assist you today?",
  "fallback_message": "Sorry, I couldn't understand. An agent will be with you shortly.",
  "session_timeout_minutes": 45,
  "ai_enabled": true,
  "ai_provider": "openai",
  "ai_model": "gpt-4",
  "ai_max_tokens": 500,
  "ai_system_prompt": "You are a helpful customer support assistant.",
  "ai_api_key": "sk-...",
  "sla_response_minutes": 10,
  "sla_resolution_minutes": 60,
  "sla_escalation_minutes": 30,
  "sla_auto_close_hours": 12,
  "sla_escalation_notify_ids": ["user-uuid-1", "user-uuid-2"]
}
```

> **Security note:** AI API keys are encrypted before storage.

## Business Hours

Business hours control when the chatbot is active. Outside business hours:

- Automation can be disabled, queuing messages for agents
- A custom out-of-hours message can be sent
- Incoming messages are still received and stored

Configure per-day schedules with open/close times and enable/disable toggles.

## Session Management

Each conversation with the chatbot creates a session:

1. **Session creation:** A new session is created when a contact sends their first message or when the previous session has expired
2. **Session timeout:** Configured via `session_timeout_minutes` (default: 30 minutes). If no activity occurs within this window, the session expires
3. **Activity tracking:** Each message updates the session's `last_activity_at` timestamp
4. **Session context:** The session tracks the current flow step, AI conversation history, and matched keyword rules

## Keyword Rules

Automate responses based on keywords in incoming messages.

### List Keyword Rules

**Endpoint:** `GET /api/chatbot/keywords`

### Create Keyword Rule

**Endpoint:** `POST /api/chatbot/keywords`

```json
{
  "name": "Order Status",
  "keywords": ["order", "tracking", "status"],
  "match_type": "contains",
  "response_type": "text",
  "response_content": "Please provide your order number and I'll check the status for you.",
  "priority": 1,
  "enabled": true
}
```

| Field | Options | Description |
|-------|---------|-------------|
| **match_type** | `exact`, `contains`, `regex` | How keywords are matched against incoming messages |
| **response_type** | `text`, `buttons`, `flow` | Type of response to send |
| **priority** | Integer (lower = higher priority) | Order in which rules are evaluated |

### Update Keyword Rule

**Endpoint:** `PUT /api/chatbot/keywords/{id}`

### Delete Keyword Rule

**Endpoint:** `DELETE /api/chatbot/keywords/{id}`

### How Keyword Matching Works

1. Enabled rules are loaded, ordered by priority (lowest number first)
2. Each rule is checked against the incoming message using its match type:
   - **Exact:** The message must exactly match one of the keywords
   - **Contains:** The message must contain any of the keywords
   - **Regex:** The message is tested against each keyword as a regular expression
3. The first matching rule's response is sent
4. If no rule matches, processing continues to AI or fallback

## Chatbot Flows

Create multi-step conversation flows for complex interactions.

### List Flows

**Endpoint:** `GET /api/chatbot/flows`

### Create Flow

**Endpoint:** `POST /api/chatbot/flows`

```json
{
  "name": "Appointment Booking",
  "description": "Guide customers through booking an appointment",
  "trigger_keywords": ["appointment", "book", "schedule"],
  "steps": [
    {
      "id": "step_1",
      "type": "message",
      "content": "What service would you like to book?",
      "next": "step_2"
    },
    {
      "id": "step_2",
      "type": "input",
      "prompt": "Please tell me the service name",
      "next": "step_3"
    },
    {
      "id": "step_3",
      "type": "message",
      "content": "Your appointment has been requested. An agent will confirm shortly.",
      "next": null
    }
  ],
  "enabled": true
}
```

### Flow Execution

1. When a message matches a flow's trigger keywords, the flow is activated
2. The current step is tracked in the user's session
3. Each step sends a message and waits for the user's response
4. The flow navigates to the next step based on the response
5. The flow completes when the last step is reached, or can transfer to an agent

### Update Flow

**Endpoint:** `PUT /api/chatbot/flows/{id}`

### Delete Flow

**Endpoint:** `DELETE /api/chatbot/flows/{id}`

## AI Integration

Enable AI-powered responses using external providers like OpenAI.

### Configuration

| Setting | Description |
|---------|-------------|
| **ai_enabled** | Toggle AI responses on/off |
| **ai_provider** | AI provider name (e.g., `openai`) |
| **ai_model** | Model identifier (e.g., `gpt-4`) |
| **ai_max_tokens** | Maximum tokens per response |
| **ai_system_prompt** | System instructions for the AI |
| **ai_api_key** | API key (encrypted on save) |

### How AI Responses Work

1. If no keyword rule or flow matches, the AI is consulted (if enabled)
2. The conversation history (last N messages) is included as context
3. Relevant AI contexts (keyword-matched or static) are loaded and included
4. The prompt is constructed from: system prompt + AI context + conversation history + current message
5. The AI provider generates a response
6. The response is sent as a WhatsApp message and logged for analytics

### AI Contexts

**Endpoints:** `GET/POST/PUT/DELETE /api/chatbot/ai-contexts`

AI contexts provide additional information to the AI model:

| Context Type | Description |
|--------------|-------------|
| **Static** | Always included in the prompt |
| **Dynamic** | Triggered by specific keywords |
| **URL-based** | Fetched from a URL and included |

Contexts are ordered by priority and matched against incoming messages.

## Agent Transfers

Seamlessly transfer conversations from the chatbot to human agents.

### List Transfers

**Endpoint:** `GET /api/chatbot/transfers`

### Create Transfer

**Endpoint:** `POST /api/chatbot/transfers`

```json
{
  "contact_id": "contact-uuid",
  "reason": "Customer requested human agent",
  "priority": "normal"
}
```

**What happens:**

1. A transfer record is created with `pending` status
2. Available agents are notified via WebSocket
3. The contact's assignment is updated

### Pick Up a Transfer

**Endpoint:** `POST /api/chatbot/transfers/pick`

An agent claims the oldest pending transfer:

1. The oldest pending transfer is found
2. It is assigned to the current user
3. Status changes to `assigned`
4. The agent and contact are notified

### Assign Transfer

**Endpoint:** `PUT /api/chatbot/transfers/{id}/assign`

Manually assign a transfer to a specific user.

### Resume From Transfer

**Endpoint:** `PUT /api/chatbot/transfers/{id}/resume`

Resume a chatbot session after an agent transfer has been resolved.

## SLA Processing

Service Level Agreement (SLA) monitoring runs automatically in the background every minute.

### SLA Settings

| Setting | Description |
|---------|-------------|
| **sla_response_minutes** | Max time before first response is considered breached |
| **sla_resolution_minutes** | Max time to resolve a chat |
| **sla_escalation_minutes** | Time before escalating to a manager |
| **sla_auto_close_hours** | Auto-close chats after this many hours of inactivity |
| **sla_escalation_notify_ids** | User IDs to notify on escalation |

### SLA Breach Actions

When an SLA is breached:

1. A warning message can be sent to the contact (if configured)
2. Escalation users are notified via WebSocket
3. If escalation time is exceeded, the chat is escalated to a manager
4. Chats exceeding `sla_auto_close_hours` are automatically closed

### Auto-Close

Chats that have been open longer than `sla_auto_close_hours` are automatically closed with a system message recording the reason.

## Incoming Message Processing

When a message arrives, the chatbot processes it in this order:

1. Find the WhatsApp account
2. Get or create the contact
3. Save the message to the database
4. Check if the chatbot is enabled
5. Check business hours
6. Check for an active session
7. Match keyword rules (by priority)
8. Match flow triggers
9. If AI is enabled and no rule matched, call the AI provider
10. Apply fallback message if nothing matched
11. Send the response
12. Update the session
13. Broadcast the message via WebSocket
14. Dispatch outbound webhooks

## See Also

- [Chat & Messaging](chat-messaging.md) — Manual chat operations
- [Templates & Flows](templates-flows.md) — WhatsApp Flows for Meta
- [Teams & Roles](teams-roles.md) — Managing agent roles
- [Analytics](analytics.md) — Chatbot performance metrics
