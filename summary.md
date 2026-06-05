# Whatomate — Analysis: Agent Selection vs. Chatbot Flows

This analysis investigates whether **Customer Agent Selection** (`/settings/agent-selection` or `/agent-selection`) and **Chatbot Flows** (`/chatbot/flows`) are the same feature or separate features in the Whatomate codebase.

## Summary of Findings

These are **two distinct, complementary features** with different scopes, database models, and execution lifecycles, though they both belong to the routing and automation domain and integrate with each other.

---

## 1. Chatbot Flows (`/chatbot/flows`)

- **Primary Models**: `ChatbotFlow`, `ChatbotFlowStep`, `ChatbotSession` (stored in `chatbot_flows`, `chatbot_flow_steps`, and `chatbot_sessions` tables).
- **Purpose**: A multi-step interactive conversation builder (dialog tree) used to automate communication with customers.
- **Features**:
  - Allows mapping multi-turn conversation trees.
  - Collects user inputs, validates them (e.g., regex, email, phone, numbers, dates), and stores responses in variables (via `StoreAs` fields in session data).
  - Integrates with external APIs (`api_fetch` steps) and formats templates.
  - Supports transferring the chat to a human agent/team queue at a designated step using the `FlowStepTypeTransfer` type.

---

## 2. Customer Agent Selection (`/settings/agent-selection`)

- **Primary Models**: `AgentSelectionSettings`, `AgentSelectionParticipant`, `AgentSelectionOption`, `AgentSelectionSession`, `AgentSelectionAuditEvent` (stored in `agent_selection_*` family of tables).
- **Purpose**: A specialized customer-driven routing setting that allows incoming customers to choose which agent, team, or queue should handle their chat.
- **Features**:
  - Automatically prompts incoming WhatsApp customers with a dynamic menu of available choices (e.g., `"1. Agent A"`, `"2. Agent B"`, `"3. Team C"`).
  - Can delay sending the menu by a configurable amount of time (`prompt_delay_minutes`) to allow staff to manually claim the chat first.
  - Dynamically builds the menu at send-time, checking if agents are active, marked as available (`IsAvailable`), within their maximum open chats capacity (`MaxOpenChats`), and allowed to access the WhatsApp instance.
  - Automatically manages timeouts, retries for invalid responses, and logs every event to a dedicated append-only audit trail (`agent_selection_audit_events`).

---

## 3. How They Integrate

While separate, the two features interact in the following way:
- **`chatbot_step` Trigger**: The Agent Selection menu trigger mode (`TriggerMode`) can be set to `chatbot_step`. This means that instead of triggering automatically on the first pending message, it can be launched directly from a specific step inside a Chatbot Flow.
- **Inbound Message Flow**: On incoming messages, the system first intercepts selection responses for active agent routing sessions. If none exist, it runs the standard chatbot/keyword rules logic.
