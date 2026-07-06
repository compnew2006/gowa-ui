# GoWA — Product Requirements Document

**Version:** v8.10.0
**Date:** 2026-07-03
**Status:** Active

---

## 1. Product Overview

GoWA (Go WhatsApp Web Multi-Device) is a self-hosted WhatsApp gateway that provides programmatic control over WhatsApp accounts through two interfaces:

1. **REST API** — A full HTTP-based API with an embedded web dashboard for manual operation and integration with external systems.
2. **MCP Server** — A Model Context Protocol server using Server-Sent Events, enabling AI agents and automation tools (Cursor, n8n, custom agents) to interact with WhatsApp.

The product serves as a bridge between WhatsApp and external systems, enabling businesses and developers to send, receive, and manage WhatsApp messages at scale, integrate with customer support platforms, and automate communication workflows.

---

## 2. Target Users

| Persona | Description |
|---------|-------------|
| **Business Operators** | Small to medium businesses that need WhatsApp as a customer communication channel at scale. |
| **Customer Support Teams** | Teams using helpdesk platforms (e.g., Chatwoot) that need bidirectional WhatsApp messaging. |
| **Developers & Integrators** | Developers building custom WhatsApp integrations, chatbots, or automation pipelines. |
| **AI/Automation Engineers** | Engineers connecting AI agents to WhatsApp via MCP for conversational AI use cases. |
| **System Administrators** | Ops teams deploying and managing self-hosted WhatsApp gateway instances. |

---

## 3. Core Capabilities

### 3.1 Multi-Device Management

GoWA supports connecting and managing **multiple WhatsApp accounts** from a single server instance.

- **Device Registry** — Create, track, and manage multiple WhatsApp connections simultaneously. Each device has its own identity, session, and configuration.
- **Per-Device Configuration** — Each device can have its own webhook URL, webhook secret, event filter, and TLS settings.
- **Device Scoping** — All operations are scoped to a specific device, ensuring data and message isolation between accounts.
- **Connection Lifecycle** — Devices can be created, connected, disconnected, reconnected, and logged out independently.
- **Auto-Reconnect** — Devices automatically reconnect using stored sessions after server restarts or connection drops.
- **Custom Device Name** — The OS/device name shown on WhatsApp Mobile can be customized (e.g., "MyBusinessApp").

### 3.2 Authentication & Pairing

Multiple pairing methods accommodate different deployment environments:

- **QR Code Login** — Scan a QR code with the WhatsApp mobile app to pair the account. Suitable for interactive setups.
- **Phone Pairing Code** — Enter a phone number to receive a numeric code, enabling headless linking without camera access. Ideal for server deployments.
- **Passkey Pairing** — WebAuthn-based pairing flow with challenge/response/confirm steps for environments supporting passkeys.
- **Basic Authentication** — Multi-credential basic auth for securing the API server itself.

### 3.3 Presence & Availability

- **Connection Presence** — Configure how the account appears when connecting: "available" (online, suppresses phone notifications), "unavailable" (registers pushname without going online), or "none" (skips presence entirely).
- **Presence Pulse Scheduler** — Periodically marks devices as "available" for a configurable duration (e.g., 5 minutes every 24 hours), then returns to "unavailable." This enables receiving typing/presence events from contacts without staying permanently online.
- **Manual Presence Control** — Set account presence to "available" or "unavailable" on demand.
- **Chat Typing Indicators** — Send "typing" (composing) or "paused" indicators to specific chats. Receive incoming typing indicators from contacts.

---

## 4. Messaging

### 4.1 Message Types

GoWA supports sending the following message types to individual chats, groups, and status broadcasts:

| Message Type | Capabilities |
|--------------|-------------|
| **Text** | Plain text with optional reply-to quoting, forwarding flag, and disappearing message duration. |
| **Image** | Upload or URL-based. Caption, view-once mode, compression. Supports JPEG, JPG, PNG. |
| **Video** | Upload or URL-based. Caption, view-once mode, GIF playback toggle, compression (re-encode to H.264, max 720px). Supports MP4, MKV, AVI. |
| **Audio** | Upload or URL-based. Push-to-talk (voice note) mode with automatic OGG/Opus transcoding and waveform generation. Supports AAC, AMR, FLAC, M4A, M4R, MP3, MPEG, OGG, WMA, WAV. |
| **Sticker** | Upload or URL-based. Automatic conversion to WebP, resize to 512×512. Animated WebP supported (512×512, under 500KB, under 10 seconds). Input: JPEG, JPG, PNG, WebP, GIF. |
| **Document** | Upload or URL-based. Caption support. Automatic MIME type detection. Thumbnail generation for PDFs and images. |
| **Contact (vCard)** | Contact cards with name and phone number. |
| **Location** | Geographic coordinates (latitude/longitude). |
| **Link Preview** | URLs with automatic title, description, and thumbnail extraction. |
| **Poll** | Question with multiple configurable options. Single or multi-select voting. Minimum 2 options required. |

### 4.2 Advanced Text Features

- **Quoted Replies** — Reply to a specific existing message with a quote context.
- **Mentions** — Mention participants in messages using `@phone_number` syntax.
- **Ghost Mentions** — Mention participants without displaying `@phone` in the message text (invisible mentions).
- **@Everyone** — Automatically mention all group participants with a single keyword.

### 4.3 Message Actions

| Action | Description |
|--------|-------------|
| **React** | Add or change emoji reactions on any message. |
| **Edit** | Edit previously sent text messages (within WhatsApp's time limits). |
| **Revoke (Delete for Everyone)** | Permanently delete a message for all chat participants. |
| **Delete (for Me)** | Delete a message from the local account's view only. |
| **Mark as Read** | Explicitly send a read receipt for a specific message. |
| **Star / Unstar** | Bookmark or un-bookmark messages. |
| **Download Media** | Download media (images, videos, audio, documents, stickers) from received messages. |
| **Forward** | Forward messages from one chat to another. |

---

## 5. Chat Management

- **Chat Listing** — Browse recent chats with pagination and search filters. Filter by media presence and archived status.
- **Chat Messages** — Fetch messages from specific chats with time-range filtering, media-type filtering, sender filtering, and full-text search.
- **Pin Chats** — Pin or unpin conversations to the top of the chat list.
- **Archive Chats** — Archive or unarchive conversations.
- **Disappearing Messages** — Configure disappearing message timers on chats (disabled, 24 hours, 7 days, or 90 days).

---

## 6. Group Management

### 6.1 Group Lifecycle

| Action | Description |
|--------|-------------|
| **Create** | Create new groups with a subject/title and initial participant list. Participants are validated as registered WhatsApp users before creation. |
| **Join via Link** | Join groups using shared invitation links. |
| **Leave** | Leave any group the account is a member of. |
| **Info from Link** | Preview group details from an invite link without joining. |
| **List My Groups** | View all groups the account belongs to (subject to WhatsApp protocol limit of 500 groups). |

### 6.2 Participant Management

| Action | Description |
|--------|-------------|
| **List Participants** | View full participant list with phone numbers, display names, admin status, and role. |
| **Add Participants** | Add members to a group. |
| **Remove Participants** | Remove members from a group. |
| **Promote to Admin** | Elevate a member to group admin. |
| **Demote Admin** | Revert an admin to regular member. |
| **Export Participants** | Export participant list as CSV (JID, phone, display name, role). |

### 6.3 Group Settings

| Setting | Description |
|---------|-------------|
| **Group Name** | Change the group subject/title. |
| **Topic / Description** | Update the group topic or description text. |
| **Group Photo** | Upload or remove the group profile picture. |
| **Locked Status** | Toggle admin-only editing of group info (name, photo, description). |
| **Announce Mode** | Toggle announcement-only mode (only admins can send messages). |
| **Invite Link** | Retrieve or reset the group's invite link. |

### 6.4 Join Request Management

- **List Pending Requests** — View pending join requests for restricted groups.
- **Approve Requests** — Accept pending join requests.
- **Reject Requests** — Decline pending join requests.

---

## 7. Newsletter / Channel Support

- **List Subscriptions** — View all newsletters/channels the account is subscribed to, enriched with subscriber counts.
- **View Messages** — Retrieve messages posted in subscribed newsletters, with pagination and reaction counts.
- **Unsubscribe** — Leave/unfollow newsletters.

---

## 8. Call Management

| Feature | Description |
|---------|-------------|
| **Auto-Reject** | Automatically decline all incoming WhatsApp calls. |
| **Manual Reject** | Programmatically reject specific calls via the API. |
| **Call Events** | Receive webhook notifications for incoming calls with caller metadata (JID, call ID, platform, WhatsApp version). |

---

## 9. User Profile & Account

| Feature | Description |
|---------|-------------|
| **View Profile** | Retrieve account information including status, picture ID, devices, and verified name. |
| **Change Avatar** | Upload a new profile picture (auto-cropped to square, max 640px). |
| **Change Push Name** | Update the display name shown to contacts. |
| **Business Profile** | View business account details (email, address, categories, business hours). |
| **Privacy Settings** | View current privacy settings (group add, status visibility, read receipts, profile visibility). |
| **Contact List** | Retrieve all contacts from the WhatsApp account. |
| **Number Check** | Validate whether a phone number is registered on WhatsApp. |
| **LID Resolution** | Convert anonymous @lid JIDs to phone numbers and vice versa. |

---

## 10. Webhook & Event System

### 10.1 Supported Event Types

| Event | Description |
|-------|-------------|
| `message` | Incoming text, media, contact, location, and other message types. |
| `message.reaction` | Emoji reactions added or changed on messages. |
| `message.revoked` | Messages deleted for everyone (recalled). |
| `message.edited` | Messages edited by the sender. |
| `message.ack` | Delivery receipts (delivered, read, played) and read receipts. |
| `message.deleted` | Messages deleted for the current user (delete-for-me). |
| `chat_presence` | Typing and recording indicators from contacts. |
| `group.participants` | Group member join, leave, promote, demote, and info change events. |
| `group.joined` | The account was added to a group. |
| `call.offer` | Incoming call received. |
| `label.edit` | WhatsApp label metadata changes. |
| `label.association` | Label applied to or removed from a chat. |
| `newsletter.joined` | Subscribed to a newsletter/channel. |
| `newsletter.left` | Unsubscribed from a newsletter. |
| `newsletter.message` | New message posted in a newsletter. |
| `newsletter.mute` | Newsletter mute setting changed. |

### 10.2 Webhook Features

| Feature | Description |
|---------|-------------|
| **Global Webhooks** | Forward all events to one or more URLs. |
| **Per-Device Webhooks** | Each device has its own webhook URL, secret, event filter, and TLS configuration. Falls back to global webhooks when not configured. |
| **Event Filtering** | Whitelist specific event types to forward (e.g., only forward `message` and `message.ack`). |
| **HMAC Signature** | Every webhook includes an `X-Hub-Signature-256` header with HMAC SHA256 for payload verification. |
| **Retry with Backoff** | Failed deliveries are retried up to 5 times with exponential backoff (1s, 2s, 4s, 8s, 16s). |
| **TLS Configuration** | Optional TLS certificate verification skip for specific environments. |
| **Meta Ads Attribution** | When conversations originate from Meta Click-to-WhatsApp ads, the first inbound message includes ad metadata (source URL, ad title, ad body, click ID). |

### 10.3 Webhook Payload

Every webhook delivery includes:

- **Event type** — Identifies the category of the event.
- **Device identity** — The device that received the event.
- **Session ID** — Optional tenant/session correlation identifier.
- **Payload** — The full event-specific data.

---

## 11. Chatwoot Integration

GoWA provides a deep bidirectional integration with the Chatwoot customer support platform.

### 11.1 Core Integration Features

| Feature | Description |
|---------|-------------|
| **Bidirectional Messaging** | Receive WhatsApp messages in Chatwoot inboxes; reply from Chatwoot and deliver via WhatsApp. |
| **Multi-Format Support** | Text, images, audio, video, documents, stickers, locations, and contacts are bridged in both directions. |
| **Group Conversations** | Groups are auto-detected with group name used as the contact name; sender names are prefixed to group messages. |
| **Auto-Provisioning** | Automatically create or reuse a Chatwoot API-channel inbox on startup. |

### 11.2 Message History & Synchronization

| Feature | Description |
|---------|-------------|
| **REST History Import** | Import existing WhatsApp message history into Chatwoot via REST API. |
| **Direct Database Import** | For self-hosted Chatwoot, write directly to PostgreSQL for faster import with preserved original timestamps and group names. Idempotent and safe to re-run. |
| **Format Translation** | WhatsApp formatting (`*bold*`, `_italic_`, `~strike~`) is automatically translated to Chatwoot Markdown (`**bold**`, `*italic*`, `~~strike~~`) and vice versa. |
| **Edit Propagation** | WhatsApp message edits are mirrored into Chatwoot as threaded notes. |
| **Delete Propagation** | WhatsApp delete-for-everyone events are mirrored into Chatwoot as threaded notes. |

### 11.3 Conversation Management

| Feature | Description |
|---------|-------------|
| **Conversation Reopening** | Automatically reopen resolved conversations when a contact messages again. |
| **Read Sync** | Optionally synchronize read receipts between WhatsApp and Chatwoot. |
| **Delete Sync** | Optionally delete linked messages on both sides when deletion is reported. |
| **Agent Signature** | Optionally prefix Chatwoot agent replies with the agent's name. |

### 11.4 Filtering & Routing

| Feature | Description |
|---------|-------------|
| **JID Ignore List** | Exclude specific contacts or entire categories (all groups, all DMs) from Chatwoot forwarding. |
| **Retry Queue** | Failed Chatwoot deliveries are stored and retried in the background with exponential backoff. |
| **Multi-Device Routing** | Specify which WhatsApp device handles Chatwoot outbound messages. |

---

## 12. MCP (Model Context Protocol) Support

GoWA exposes a comprehensive set of MCP tools organized into four categories for AI agent integration:

### 12.1 Connection Management Tools
Check connection status, initiate QR login, generate pairing codes, logout, and reconnect.

### 12.2 Messaging & Communication Tools
Send text (with reply/forwarding), contacts, links, locations, images (with view-once, compression), videos (with view-once, GIF playback, compression), stickers, documents, audio (with PTT/voice note), and polls.

### 12.3 Chat & Contact Management Tools
List contacts, list chats with pagination and search, fetch chat messages with filtering, download media, archive chats, react to messages, edit messages, revoke/delete messages, mark as read, and star/unstar messages.

### 12.4 Group Management Tools
Create groups, join via links, leave groups, list participants, manage participants (add, remove, promote, demote), manage invite links (with reset), get group info, set group name, topic, locked, and announce modes, and manage join requests (list, approve, reject).

---

## 13. Automation Features

| Feature | Description |
|---------|-------------|
| **Auto-Reply** | Automatically reply to all incoming text messages with a configurable message. Skips groups, broadcasts, and self-messages. |
| **Auto Mark as Read** | Automatically send read receipts for all incoming messages. |
| **Auto Download Media** | Automatically download media from incoming messages. |
| **Auto Reject Calls** | Automatically decline all incoming WhatsApp calls. |

---

## 14. Web Dashboard

The embedded web dashboard provides a visual interface for all WhatsApp operations:

- **Device Overview** — View and manage connected devices, their states, and per-device webhook configuration.
- **Login & Pairing** — QR code scanning, pairing code entry, and passkey-based pairing.
- **Send Messages** — Dedicated forms for each message type: text, image, video, audio, sticker, document, contact, location, link, and poll.
- **Presence Control** — Set account availability and send typing indicators to specific chats.
- **Message Management** — Delete, revoke, react to, edit, and mark messages as read.
- **Group Management** — Create, join, leave, manage participants, update settings, and handle invite links.
- **Newsletter Browser** — View subscribed newsletters and their messages.
- **Contact Directory** — Browse account contacts and check phone number registration.
- **Account Settings** — View and update profile picture, push name, privacy settings, and business profile.
- **Chat Browser** — Browse chat list, view messages, pin and archive chats.
- **Real-Time Updates** — WebSocket-driven live updates for device state changes, login events, and notifications.
- **Call Management** — Reject incoming calls.

---

## 15. Configuration & Deployment

### 15.1 Configuration Options

| Category | Configurable Aspects |
|----------|---------------------|
| **Server** | Port, host binding, base path (for reverse proxy subpath deployment), basic authentication (multiple credentials). |
| **WhatsApp Behavior** | Auto-reply message, auto-mark-read, auto-download-media, auto-reject calls, presence on connect, presence pulse scheduler (interval and duration). |
| **Webhooks** | Global webhook URLs, secrets, event filters, TLS verification skip. |
| **Media Limits** | Maximum image, file, video, and download sizes. |
| **Chatwoot** | Server URL, API token, account/inbox/device IDs, history import settings, auto-provisioning, conversation behavior, JID filters, agent signatures, edit/delete propagation, read/delete sync. |
| **Trusted Proxies** | IP ranges for reverse proxy deployments. |

All settings can be configured via CLI flags, environment variables, or `.env` files.

### 15.2 Deployment Options

- **Docker** — Multi-stage containerized deployment with persistent volume mounts for storage and media.
- **Docker Compose** — Full stack deployment with volume configuration and environment file support.
- **Pre-Built Binaries** — Standalone binaries for Linux (x86_64, ARM64, ARMv7, 386), macOS (Intel, Apple Silicon), and Windows (amd64, 386).
- **Build from Source** — Compile from source for custom environments.

### 15.3 Cross-Platform Support

- Linux (x86_64, ARM64, ARMv7)
- macOS (Intel, Apple Silicon)
- Windows (WSL recommended)
- Raspberry Pi (ARMv6/ARMv7)

---

## 16. Health & Operational Monitoring

| Feature | Description |
|---------|-------------|
| **Health Check** — Public endpoint for infrastructure liveness/readiness probes. Returns degraded status if the device manager is unhealthy. |
| **Graceful Shutdown** — Properly drains connections and closes database pools on termination signals. |
| **Background Workers** — Presence pulse scheduler, webhook retry worker, and auto-reconnect monitor run as background processes. |

---

## 17. SDK & Client Libraries

SDK packages are available for integrating with the GoWA API in multiple programming languages:

- **Node.js / TypeScript** — `@aldinokemal/sdk-node-whatsapp-web-multidevice`
- **Go** — `SdkWhatsappWebMultiDevice`
- **PHP** — `SdkWhatsappWebMultiDevice`

---

## 18. n8n Integration

GoWA is available as an n8n community node package, enabling no-code/low-code workflow automation with WhatsApp through the n8n platform.

---

## 19. Constraints & Assumptions

- **Unofficial Client** — GoWA is not affiliated with WhatsApp/Meta. It uses the whatsmeow library as an unofficial WhatsApp Web client.
- **WhatsApp Protocol Limits** — The WhatsApp protocol imposes certain limits (e.g., maximum 500 groups retrievable per account).
- **Multi-Mode Exclusivity** — REST and MCP modes share the same underlying WhatsApp state and cannot run simultaneously in a single process for the same device.
- **Account Validation** — WhatsApp accounts may require periodic re-validation. GoWA supports configurable account validation.
- **Broadcast Filtering** — Status broadcast messages (`status@broadcast`) are intentionally mapped to display name "Status" in chat listings.
