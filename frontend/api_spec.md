# Whatomate - API Specification Guide

This document provides a comprehensive map of the API endpoints, data models, and communication patterns required by the Whatomate frontend.

## Global Configuration

- **Base URL**: `/api` (configurable via `VITE_API_URL`)
- **Headers**:
  - `Content-Type: application/json`
  - `X-CSRF-Token`: Required for mutations. Read from `whm_csrf` cookie.
- **Auth Strategy**: Cookie-based JWT (`whm_token`, `whm_refresh`).

---

## 🔐 Authentication & Profile (`/auth`, `/me`)

| Endpoint              | Method | Payload               | Description                         |
| :-------------------- | :----- | :-------------------- | :---------------------------------- |
| `/auth/login`         | `POST` | `{ email, password }` | Login.                              |
| `/auth/register`      | `POST` | `{ ... }`             | Registration.                       |
| `/auth/logout`        | `POST` | `{}`                  | Logout.                             |
| `/auth/refresh`       | `POST` | `{}`                  | Token refresh.                      |
| `/auth/me`            | `GET`  | -                     | Current user profile.               |
| `/auth/switch-org`    | `POST` | `{ organization_id }` | Switch org context.                 |
| `/auth/ws-token`      | `GET`  | -                     | Short-lived token for WS.           |
| `/auth/sso/providers` | `GET`  | -                     | List available SSO providers.       |
| `/me`                 | `GET`  | -                     | Detailed profile data.              |
| `/me/settings`        | `PUT`  | `UserSettings`        | Update personal settings.           |
| `/me/password`        | `PUT`  | `{ current, new }`    | Change password.                    |
| `/me/availability`    | `PUT`  | `{ is_available }`    | Toggle agent availability.          |
| `/me/organizations`   | `GET`  | -                     | List organizations user belongs to. |

---

## 👥 Contacts & Messages (`/contacts`)

| Endpoint                      | Method           | Payload                      | Description                                           |
| :---------------------------- | :--------------- | :--------------------------- | :---------------------------------------------------- |
| `/contacts`                   | `GET`            | `?search, page, limit, tags` | Search/List contacts.                                 |
| `/contacts/{id}`              | `GET/PUT/DELETE` | `Contact`                    | CRUD for contacts.                                    |
| `/contacts/{id}/assign`       | `PUT`            | `{ user_id }`                | Assign to agent.                                      |
| `/contacts/{id}/tags`         | `PUT`            | `{ tags: string[] }`         | Update tags.                                          |
| `/contacts/{id}/session-data` | `GET`            | -                            | Internal state/vars.                                  |
| `/contacts/{cid}/messages`    | `GET/POST`       | `Message`                    | message history / send message.                       |
| `.../messages/template`       | `POST`           | `{ template, components }`   | Send WhatsApp Template.                               |
| `.../messages/{mid}/reaction` | `POST`           | `{ emoji }`                  | Send reaction.                                        |
| `/api/media/{message_id}`     | `GET`            | -                            | Retrieve/Download media content for a stored message. |
| `/api/messages/media`         | `POST`           | `FormData`                   | Upload media for messaging (returns internal URL).    |

### Group Chats

Group chats are treated as **Contacts** within the system. There are no separate "Groups" endpoints for messaging.

- **Identification**: A group chat is a contact where the `phone_number` (or JID) field contains a WhatsApp Group JID (e.g., `120363422675615917@g.us`).
- **Endpoints**: Use the standard `/api/contacts/{contactId}/messages` endpoints for both fetching and sending messages within a group.
- **Permissions**: Interacting with group messages requires the same `contacts:read` and `contacts:write` permissions as private chats.

---

## 🤖 Automations & Chatbot (`/chatbot`, `/flows`)

| Endpoint                         | Method    | Payload    | Description                 |
| :------------------------------- | :-------- | :--------- | :-------------------------- |
| `/chatbot/settings`              | `GET/PUT` | `Settings` | Global bot config.          |
| `/chatbot/keywords`              | `G/P/U/D` | `Rule`     | Keyword-based replies.      |
| `/chatbot/flows`                 | `G/P/U/D` | `Flow`     | List/Manage chatbot flows.  |
| `/chatbot/sessions`              | `GET`     | `?params`  | List active bot sessions.   |
| `/chatbot/transfers`             | `G/P`     | `Transfer` | Manage agent handovers.     |
| `/chatbot/transfers/pick`        | `POST`    | -          | Auto-pick next in queue.    |
| `/chatbot/transfers/{id}/resume` | `PUT`     | -          | Resume bot for the contact. |

---

### 📊 Dashboard & Widgets (`/widgets`)

| Endpoint                | Method           | Payload    | Description                     |
| :---------------------- | :--------------- | :--------- | :------------------------------ |
| `/widgets`              | `GET/POST`       | `Widget`   | List/Create dashboard widgets.  |
| `/widgets/data-sources` | `GET`            | -          | List allowed metrics/sources.   |
| `/widgets/data`         | `GET`            | `?period`  | Fetch data for ALL widgets.     |
| `/widgets/layout`       | `POST`           | `Layout[]` | Save grid positions.            |
| `/widgets/{id}`         | `GET/PUT/DELETE` | `Widget`   | CRUD for specific widget.       |
| `/widgets/{id}/data`    | `GET`            | `?period`  | Fetch data for a single widget. |

- **Data Metrics**: Supports `count`, `sum`, `avg` over `messages`, `contacts`, `transfers`, and `sessions`.
- **Grouping**: Server-side grouping by `status`, `direction`, `type`, or `agent`.

---

### 🛒 Catalog Management (`/catalogs`)

| Endpoint                  | Method           | Payload         | Description                    |
| :------------------------ | :--------------- | :-------------- | :----------------------------- |
| `/catalogs`               | `GET/POST`       | `Catalog`       | List/Create Product Catalogs.  |
| `/catalogs/sync`          | `POST`           | -               | Pull catalogs from Meta.       |
| `/catalogs/{id}`          | `GET/DELETE`     | -               | Manage catalog metadata.       |
| `/catalogs/{id}/products` | `GET/POST`       | `Product`       | List/Add products to catalog.  |
| `/products/{id}`          | `GET/PUT/DELETE` | `ProductUpdate` | Individual product management. |

---

### ⚡ Custom Actions & Webhooks (`/webhooks`, `/custom-actions`)

| Endpoint                         | Method     | Payload                | Description                           |
| :------------------------------- | :--------- | :--------------------- | :------------------------------------ |
| `/webhooks`                      | `GET/POST` | `Webhook`              | Manage outbound webhooks.             |
| `/webhooks/{id}/test`            | `POST`     | -                      | Send test payload.                    |
| `/custom-actions`                | `GET/POST` | `Action`               | List/Create UI action buttons.        |
| `/custom-actions/{id}/execute`   | `POST`     | `{ contact_id, vars }` | Trigger action logic.                 |
| `/custom-actions/redirect/{tok}` | `GET`      | -                      | Handle action URL redirects (Public). |

- **Safe Execution**: JavaScript actions run in a restricted `goja` sandbox.
- **Variable Injection**: Supports `{{contact.name}}`, `{{user.name}}`, and `{{current_time}}` in URLs and payloads.

---

### 📥 Import/Export (`/import`, `/export`)

| Endpoint                 | Method | Payload              | Description                         |
| :----------------------- | :----- | :------------------- | :---------------------------------- |
| `/export`                | `POST` | `{ table, filters }` | Generate CSV/Excel download.        |
| `/import`                | `POST` | `FormData (CSV)`     | Bulk import (Contacts/Tags).        |
| `/export/{table}/config` | `GET`  | -                    | Fetch allowed columns and mappings. |
| `/import/{table}/config` | `GET`  | -                    | Fetch validation rules for import.  |

---

## 🔌 WebSocket Events (`/ws`)

Clients authenticate via an `auth` message using the `/auth/ws-token`.

Clients authenticate via an `auth` message using the `/auth/ws-token`.

### Server -> Client (Incoming)

- `new_message`: Real-time chat updates.
- `status_update`: Message delivery status (sent/delivered/read).
- `reaction_update`: Emoji reactions on messages.
- `agent_transfer`: Notification of new handovers.
- `agent_transfer_resume/assign`: Sync transfer ownership.
- `transfer_escalation`: Critical SLA breach alerts.
- `campaign_stats_update`: Live campaign progress.
- `permissions_updated`: Triggers forced frontend reload.
- `conversation_note_created/updated/deleted`: Internal agent notes sync.

### Client -> Server (Outgoing)

- `auth`: Initial token verification.
- `set_contact`: Notifies server which chat the agent is currently viewing.
- `ping`: Keep-alive heartbeat.

---

## 📦 Data Models (Core Interfaces)

### Contact

```typescript
interface Contact {
  id: string;
  phone_number: string;
  name: string;
  profile_name?: string;
  avatar_url?: string;
  status: string;
  tags: string[];
  metadata: Record<string, any>;
  last_message_at?: string;
  unread_count: number;
  assigned_user_id?: string;
}
```

### Message

```typescript
interface Message {
  id: string;
  contact_id: string;
  direction: "incoming" | "outgoing";
  message_type: string; // text, image, video, interactive, template
  content: any;
  media_url?: string;
  status: string; // sent, delivered, read, failed
  wamid?: string; // WhatsApp Message ID
  reply_to_message_id?: string;
  reactions?: Array<{ emoji: string; from_phone?: string }>;
}
```

### User

```typescript
interface User {
  id: string;
  email: string;
  full_name: string;
  role: {
    name: string;
    permissions: Array<{ resource: string; action: string }>;
  };
  organization_id: string;
  is_available: boolean;
}
```

---

## 🚀 Response Wrapping Pattern

The frontend expects a consistent response structure.

**Success**:

```json
{
  "status": "success",
  "data": { ... } // or [...]
}
```

**Error**:

```json
{
  "status": "error",
  "message": "Human readable error description",
  "errors": [{ "field": "email", "message": "Invalid email format" }]
}
```
