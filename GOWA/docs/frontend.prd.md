# encantoWhatsapp — Product Requirements Document

> **Version:** 1.1  
> **Date:** 2026-07-03  
> **Scope:** Frontend SaaS layer — product features only (no technical architecture)

---

## 1. Product Overview

**encantoWhatsapp** is a multi-tenant WhatsApp Business engagement platform. It transforms raw WhatsApp multi-device connectivity into a team-ready customer engagement workspace with shared inbox, campaign automation, chatbot intelligence, contact management, real-time collaboration tools, and **multi-account media distribution**.

Organizations sign up, connect their WhatsApp accounts (one or many), invite their teams, and immediately gain a shared inbox where multiple agents can handle customer conversations simultaneously — with zero collisions, full audit trails, and built-in automation. When an organization connects multiple WhatsApp accounts, the platform intelligently distributes media sending across accounts based on capacity, ensuring no single account is overloaded and every file is traceable to the account that sent it.

The platform targets businesses that need to scale their WhatsApp-based customer communication beyond a single operator handling everything manually.

---

## 2. Problem Statement

| Problem | Impact |
|---------|--------|
| WhatsApp provides no native multi-agent collaboration | Only one person can manage a number at a time; scaling requires sharing passwords or phones |
| Manual outreach doesn't scale | Sending messages one by one to hundreds of contacts is slow, error-prone, and untrackable |
| No automation or intelligent routing | Every conversation requires a human to answer; no auto-replies, keyword handling, or round-robin distribution |
| No conversation continuity | When agents go offline, conversations stall with no visibility into pending work |
| Disconnected from business tooling | WhatsApp chats exist in isolation — no CRM, no tagging, no analytics, no integrations |

---

## 3. Goals & Objectives

- **Time-to-value under 10 minutes** — sign up, connect WhatsApp, start handling conversations
- **Dozens of concurrent agents** with zero message collisions across shared inboxes
- **Campaign automation** — send personalized bulk messages to hundreds of recipients with one action
- **99.9% uptime** for business-critical customer communication
- **Strong tenant growth** — support 1,000+ organizations, each with 50+ agents
- **Compliance & security** — role-based access, phone number masking, audit logging, tenant isolation

---

## 4. Target Personas

### 4.1 Organization Administrator
Manages the entire workspace: creates organizations, invites users, configures roles and permissions, manages WhatsApp account connections, sets business hours, controls chatbot behavior, and oversees analytics.

### 4.2 Support Agent
Handles live customer conversations in the shared inbox. Claims conversations, sends messages, uses canned responses, collaborates with teammates via notes, and transfers chats when needed.

### 4.3 Team Manager
Supervises a group of agents. Monitors team workload, reviews agent analytics, manages team assignment strategies, and handles escalated transfers from the chatbot.

### 4.4 Developer / Integrator
Connects encantoWhatsapp to external systems via webhooks and API keys. Configures event-driven integrations, automates workflows, and builds on top of the platform.

---

## 5. Success Metrics

| Category | Metric |
|----------|--------|
| Adoption | New organizations onboarded per week, agents active per organization |
| Operational Performance | Average conversation claim time, average first-response time, agent utilization rate |
| Campaign Effectiveness | Delivery rate, read rate, response rate per campaign |
| Platform Health | Uptime percentage, WebSocket reconnection rate, message delivery latency |
| Business | Messages sent per organization, campaign ROI, customer satisfaction scores |

---

## 6. Feature Inventory

### 6.1 Authentication & Account Management

| # | Feature | Description |
|---|---------|-------------|
| 1 | Login | Username/password authentication with session persistence |
| 2 | Registration | New account creation with email, name, and password |
| 3 | Single Sign-On (SSO) | Integration with external identity providers for seamless login using clerk authentication | 
| 4 | Session Management | Automatic session refresh and expiration handling |
| 5 | WebSocket Authentication | Secure token-based authentication for real-time connections |
| 6 | Password Management | Password change capability |
| 7 | Remember Me | Persistent login across browser sessions |
| 8 | User Profile | View and edit personal profile information |

### 6.2 Dashboard & Analytics

| # | Feature | Description |
|---|---------|-------------|
| 9 | Main Dashboard | Central overview with key metrics, recent activity, and quick actions |
| 10 | Agent Analytics | Per-agent performance metrics including response times, conversation counts, and resolution rates |
| 11 | Message Insights | Analytics for message delivery, read receipts, and response rates gathered from database logs and webhook payloads |
| 12 | Chart Visualizations | Interactive line charts and bar charts for trend analysis |

### 6.3 Multi-Tenant Organizations

| # | Feature | Description |
|---|---------|-------------|
| 14 | Organization Creation | Create new organizations with unique names and identifiers |
| 15 | Organization Switcher | Switch between organizations in a multi-org workspace |
| 16 | Organization Deletion | Remove organizations (protected default organization cannot be deleted) |
| 17 | Organization Settings | Configure timezone, date format, phone masking, outbound mode, upload cleanup, campaign controls, and rollout modes |
| 18 | Member Management | Invite users to organizations, assign roles, and manage memberships |

### 6.4 User Management

| # | Feature | Description |
|---|---------|-------------|
| 19 | User Creation | Create new user accounts with email, name, password, and role |
| 20 | User Listing | Search and browse all registered users |
| 21 | User Editing | Update user details, role, and active status |
| 22 | User Deletion | Remove user accounts from the system |
| 23 | Availability Status | Toggle between Available and Away with break timer tracking |
| 24 | Per-User Settings | Personal preferences including theme, notifications, chat background, and shortcuts |
| 25 | Send Restrictions | Per-user sending restriction configuration to control who can send what |
| 26 | Activity Tracking | Monitor user activity and engagement patterns |

### 6.5 Roles & Permissions

| # | Feature | Description |
|---|---------|-------------|
| 27 | System Roles | Built-in roles: Admin, Manager, Agent — protected from deletion |
| 28 | Custom Roles | Create roles with arbitrary names, descriptions, and permission sets |
| 29 | Fine-Grained Permissions | Granular permission strings (e.g., `contacts:read`, `campaigns:write`, `chat:assign`) |
| 30 | Default Role | Configurable default role assigned to new users |
| 31 | Role Editing | Modify role names, descriptions, default status, and permission sets |
| 32 | Role Deletion | Remove custom roles (system roles are protected) |

### 6.6 Team Management

| # | Feature | Description |
|---|---------|-------------|
| 33 | Team Creation | Create teams with name, description, and assignment strategy |
| 34 | Team Listing | Search and browse all teams |
| 35 | Assignment Strategies | Manual, Round-Robin, or Load-Balanced distribution of conversations to team members |
| 36 | Team Editing | Modify team details and active/inactive status |
| 37 | Team Deletion | Remove teams from the organization |
| 38 | Team Members | Add and manage team members with manager or agent roles |

### 6.7 Tag Management

| # | Feature | Description |
|---|---------|-------------|
| 39 | Tag Creation | Create color-coded tags for categorizing contacts and conversations |
| 40 | Tag Listing | Search and browse all tags |
| 41 | Tag Editing | Rename tags and change colors |
| 42 | Tag Deletion | Remove tags from the organization |
| 43 | Contact Tagging | Apply and remove tags on contacts for filtering and organization |

### 6.8 Live Chat Inbox

| # | Feature | Description |
|---|---------|-------------|
| 44 | Real-Time Inbox | Live conversation feed updated via WebSocket with instant message delivery |
| 45 | Conversation Claiming | Agents claim conversations to take ownership and prevent duplicate handling |
| 46 | Conversation Closing | Mark conversations as resolved/closed |
| 47 | Conversation Reopening | Reactivate closed conversations for further follow-up |
| 48 | Conversation Assignment | Assign conversations to specific agents |
| 49 | Public/Private Toggle | Make conversations visible to all agents or restricted to assigned agent |
| 50 | Soft Delete | Hide conversations from the inbox without permanent deletion |
| 51 | Filter by Status | Filter inbox by open, closed, or all conversations |
| 52 | Filter by Agent | Filter inbox by assigned agent |
| 53 | Filter by Tag | Filter conversations by applied tags |
| 54 | Filter by Instance | Filter conversations by connected WhatsApp account |
| 55 | Search | Full-text search across conversations |

### 6.9 Messaging

| # | Feature | Description |
|---|---------|-------------|
| 56 | Send Text Messages | Compose and send text messages to contacts |
| 57 | Send Images | Send image files with optional captions |
| 58 | Send Videos | Send video files |
| 59 | Send Audio | Send audio files and voice messages |
| 60 | Send Documents | Send PDF, DOCX, and other document types |
| 61 | Send Stickers | Send WhatsApp stickers |
| 62 | Send Contact Cards | Share contact information cards |
| 63 | Send Link Previews | Share links with generated preview cards |
| 64 | Send Location | Share current or pinned location with map preview |
| 65 | Send Polls | Create and send interactive polls with multiple options |
| 66 | Message Reactions | React to received messages with emoji |
| 67 | Message Editing | Edit previously sent messages |
| 68 | Message Revocation | Delete sent messages for all recipients |
| 69 | Message Starring | Star important messages for quick reference |
| 70 | Read Receipts | Track when messages have been delivered and read |
| 71 | Message Pinning | Pin important messages within conversations |
| 72 | Disappearing Messages | Configure auto-delete timers on conversations |
| 73 | Chat Archiving | Archive conversations to remove them from the main inbox |
| 74 | Quoted Replies | Reply to specific messages with quoted context |
| 75 | Rich Text Editor | Message composer with formatting support |
| 76 | Emoji Picker | Full emoji selector for message composition |
| 77 | Typing Indicators | See when contacts are typing and broadcast typing status |
| 78 | Online Presence | See when contacts come online or go offline |
| 79 | Merge & Batch Print | Select multiple files/images from chat and merge for printing |
| 80 | Chat Export | Export individual conversations |

### 6.10 Contacts & CRM

| # | Feature | Description |
|---|---------|-------------|
| 81 | Contact Listing | Browse all contacts sorted by recent activity with search |
| 82 | Contact Creation | Create new contact records manually |
| 83 | Contact Editing | Update contact name, email, status, and custom metadata |
| 84 | Contact Deletion | Soft-delete or permanently remove contacts |
| 85 | Internal Notes | Add private notes to conversations (never sent to customers) |
| 86 | Conversation Collaborators | Invite other agents to view or edit conversations with accept/decline workflow |
| 87 | Session Data | Store and retrieve per-conversation session information |
| 88 | Contact Import | Bulk import contacts from CSV files |
| 89 | Contact Export | Export contacts to CSV or PDF formats |

### 6.11 WhatsApp Instance Management

| # | Feature | Description |
|---|---------|-------------|
| 90 | Accounts Hub | Central view of all connected WhatsApp accounts |
| 91 | QR Code Pairing | Connect new WhatsApp accounts via QR code scanning |
| 92 | Phone Pairing Code | Connect via multi-device pairing code |
| 93 | Disconnect | Log out and disconnect WhatsApp accounts |
| 94 | Reconnect | Re-establish connection to disconnected accounts |
| 95 | Health Monitoring | Real-time status tracking including uptime, message queue, sent/received counts, error rates |
| 96 | Multi-Account Connection | Connect and manage multiple concurrent WhatsApp Web numbers/accounts per organization dashboard |
| 97 | Instance Tags | Tag and categorize connected accounts |
| 98 | Access Control | Control which agents can use which WhatsApp instances |
| 99 | Device Registry | Track all registered WhatsApp device sessions with connection status |
| 100 | Passkey Pairing | Connect new WhatsApp accounts via WebAuthn passkey pairing challenge/confirm flow |
| 101 | Presence & Pulse Config | Configure device presence mode (available, unavailable, none) and scheduled presence pulse (interval and duration) |
| 102 | WhatsApp Profile Settings | Expose settings to edit WhatsApp profile avatar, display push name, business details, and privacy settings |

### 6.12 Multi-Account Media Pool

| # | Feature | Description |
|---|---------|-------------|
| 103 | Media Pool Dashboard | Visual overview of all connected GOWA accounts in the media pool — showing each account's sending capacity, current load, queued files, and health status |
| 104 | Capacity-Aware Routing | Automatically route outbound media sends to the GOWA account with the most available capacity. When an account hits its rate limit or daily cap, the system transparently switches to the next available account |
| 105 | File→Account Mapping | Every media file sent through the platform is permanently mapped to the specific GOWA account/device that sent it. This mapping is queryable, exportable, and auditable |
| 106 | Automatic Failover | When a GOWA account goes offline, disconnects, or enters an error state, queued media sends for that account are automatically redistributed to other healthy accounts in the pool |
| 107 | Send Queue & Retry | When all accounts in the pool are at capacity, media sends are queued with configurable retry intervals. Sends resume automatically when capacity frees up. Admins are notified when the queue depth exceeds a threshold |
| 108 | R2 Media Registry | All media files (images, videos, documents, audio, stickers) are stored in Cloudflare R2 with metadata tracked in the database: original filename, MIME type, size, R2 object key, the GOWA account that sent it, delivery status, and timestamp |
| 109 | Pool Analytics | Per-account sending statistics: messages sent, media sent, failures, average latency, capacity utilization over time. Helps admins decide when to add more accounts to the pool |

### 6.13 Campaigns & Bulk Messaging

| # | Feature | Description |
|---|---------|-------------|
| 110 | Campaign Creation | Create bulk messaging campaigns targeting individual contacts |
| 111 | Template Variables | Personalize messages with per-recipient variable injection |
| 112 | Campaign Scheduling | Schedule campaigns to run at specific dates and times |
| 113 | Auto-Campaigns | Configure automated campaigns per WhatsApp instance |
| 114 | Media Attachments | Attach images, videos, or documents to campaigns. Media sends are routed through the multi-account media pool |
| 115 | Campaign Status Tracking | Monitor campaigns through Draft, Queued, Processing, Completed, Failed, and Cancelled states |
| 116 | Per-Recipient Variables | Customize message content for each recipient individually |
| 117 | Group Campaigns | Send bulk messages to WhatsApp groups |
| 118 | Group Join Campaigns | Mass-join WhatsApp groups with speed control and multi-account support |
| 119 | Campaign Analytics | Delivery statistics including sent, failed, and skipped counts — broken down by GOWA account used |

### 6.14 Canned Responses & Templates

| # | Feature | Description |
|---|---------|-------------|
| 120 | Quick Replies | Pre-written message templates triggered by keyboard shortcuts |
| 121 | Response Categories | Organize canned responses by category |
| 122 | Media Attachments | Attach images or videos to canned responses |
| 123 | Usage Tracking | Track how often each canned response is used |
| 124 | Active/Inactive Toggle | Enable or disable individual canned responses |
| 125 | Saved Contents | Library of reusable text and media templates with variable placeholders |

### 6.15 Chatbot & AI Automation

| # | Feature | Description |
|---|---------|-------------|
| 126 | Chatbot Overview | Central management dashboard for chatbot configuration |
| 127 | Greeting Messages | Automatic welcome messages sent when a customer starts a conversation |
| 128 | Fallback Messages | Default response when no keyword or flow matches |
| 129 | Session Timeout | Automatic chatbot reset after inactivity |
| 130 | Business Hours | Configure working hours for automated availability |
| 131 | SLA Settings | Service-level agreement targets for response times |
| 132 | Keyword Auto-Responses | Trigger actions based on exact, contains, or regex keyword matches |
| 133 | Conversation Flows | Multi-step conversational flow builder with JSON step definitions |
| 134 | AI Contexts | Configure AI providers, endpoints, models, and response paths for intelligent replies |
| 135 | Agent Transfers | Automatic chatbot-to-human handoff with transfer queue |
| 136 | Transfer Queue | Dashboard showing pending, assigned, and completed transfers |
| 137 | Auto-Pick Next Transfer | Atomically claim the oldest pending transfer (FIFO) |
| 138 | Transfer Assignment | Manually assign specific transfers to agents or teams |
| 139 | Transfer Resume | Reopen previously completed or cancelled transfers |
| 140 | Chatbot Pause/Resume | Toggle chatbot activity per conversation |

### 6.16 Customer Routing (Agent Selection)

| # | Feature | Description |
|---|---------|-------------|
| 141 | Interactive Agent Menu | Customers choose their preferred agent or department from a menu |
| 142 | Trigger Configuration | Configure when the menu appears (on first message, keyword trigger, or delay) |
| 143 | Agent Profiles | Configure display names, descriptions, availability, and max open chat limits |
| 144 | Team & Queue Options | Route customers to teams or general queues |
| 145 | Custom Actions | Define custom menu options (e.g., "keep pending" or route to a specific team) |
| 146 | Session Management | Track active selection sessions with timeout and invalid attempt handling |
| 147 | Audit Trail | Log all agent selection events for accountability |

### 6.17 Webhooks & Integrations

| # | Feature | Description |
|---|---------|-------------|
| 148 | Webhook Creation | Configure outgoing webhook endpoints |
| 149 | Event Filtering | Subscribe to specific events (15+ event types including message received, chat assigned, etc.) |
| 150 | Custom Headers | Attach custom HTTP headers to webhook payloads |
| 151 | HMAC Signing | Secure webhook deliveries with HMAC SHA256 signatures |
| 152 | Active Toggle | Enable or disable webhooks without deletion |
| 153 | Custom Actions | Create custom UI action buttons that trigger webhooks, open URLs, or execute JavaScript |

### 6.18 WhatsApp Number Filter

| # | Feature | Description |
|---|---------|-------------|
| 154 | Batch Validation | Validate whether phone numbers are registered on WhatsApp in bulk |
| 155 | Progress Tracking | Monitor batch validation progress with error handling |
| 156 | Result Classification | Separate valid numbers from invalid ones for targeted outreach |

### 6.19 Group Directory

| # | Feature | Description |
|---|---------|-------------|
| 157 | Group Catalog | Curated directory of WhatsApp groups with metadata |
| 158 | Group Discovery | Search groups by name, description, country, language, or category |
| 159 | Group Details | View participant counts, join links, images, and descriptions |

### 6.20 WhatsApp Group Manager

| # | Feature | Description |
|---|---------|-------------|
| 160 | Group Creator | Create new WhatsApp groups with subject, description, and initial participants |
| 161 | Group Invites & Join Links | Retrieve and reset the invite link of any group the account admins |
| 162 | Participant Manager | Add/remove participants, promote to admin, and demote admins |
| 163 | Group Settings Locker | Toggle locked status (who can edit info) and announcement mode (who can send messages) |
| 164 | Join Request Manager | View, approve, or reject pending join requests for restricted groups |
| 165 | Group Participants Export | Export group participants list to CSV with names, roles, and status |
| 166 | Leave & Join Actions | Join groups via invite link and leave active groups |

### 6.21 Calls

| # | Feature | Description |
|---|---------|-------------|
| 167 | Call Detection | Detect incoming WhatsApp voice and video calls |
| 168 | Call Rejection | Reject calls with configurable behavior |
| 169 | Auto-Reject | Automatically reject all incoming calls |

### 6.22 Notifications

| # | Feature | Description |
|---|---------|-------------|
| 170 | In-App Notification Center | Centralized notification feed with info, warning, and error types |
| 171 | Notification Dismissal | Mark notifications as read or dismissed |
| 172 | Notification Settings | Configure which notifications to receive |
| 173 | Notification Sounds | Choose from multiple notification sound options with preview |
| 174 | Escalation Alerts | Configurable alerts for SLA breaches and urgent events |

### 6.23 Appearance & Personalization

| # | Feature | Description |
|---|---------|-------------|
| 175 | Theme Selection | Light mode, Dark mode, or System-following theme |
| 176 | Theme Presets | Pre-built color schemes (Twitter, Ocean Breeze, Soft Pop, Amber Minimal) |
| 177 | Chat Background | Upload custom wallpaper for chat conversations |
| 178 | Language Support | 15+ languages including English, Spanish, French, German, Arabic (RTL), Japanese, Korean, Indonesian, Portuguese, Russian, Chinese, Hindi, Italian, Dutch, Malay, Polish |
| 179 | RTL Support | Full right-to-left layout support for Arabic and other RTL languages |

### 6.24 API & Developer Tools

| # | Feature | Description |
|---|---------|-------------|
| 180 | API Key Management | Create and manage programmatic access keys with usage tracking and expiry |

### 6.25 License & Quota Management

| # | Feature | Description |
|---|---------|-------------|
| 181 | License Overview | View license state, type (Trial, Paid, Lifetime), and activation details |
| 182 | Grace Period Alerts | Warnings when license is approaching expiry |
| 183 | Quota Tracking | Monitor organization limits (max orgs, users per org, endpoints per org, storage per org) |
| 184 | Quota Warnings | Alerts when approaching or exceeding quota limits |

### 6.26 Facebook Tools (Feature-Flagged)

| # | Feature | Description |
|---|---------|-------------|
| 185 | Facebook Account Connection | Connect and manage Facebook pages via OAuth |
| 186 | Comment Inbox | Handle page comments, public replies, and private follow-ups from one inbox |
| 187 | Page Search | Find and extract target Facebook pages by keywords, niches, or regions |
| 188 | People Search | Discover active Facebook users, influencers, and prospects |
| 189 | Group Search | Find Facebook groups matching target audience criteria |
| 190 | Extract Likes | Extract users who liked or engaged with posts |
| 191 | Page Messengers | Extract profiles of users who messaged official pages |
| 192 | Extract Data | Extract profile details, emails, and phone numbers from leads |
| 193 | Auto Share | Automate sharing of marketing content to authorized pages and groups |
| 194 | Retargeting | Retarget existing leads with campaigns |

### 6.27 WhatsApp Newsletters & Channels

| # | Feature | Description |
|---|---------|-------------|
| 195 | Newsletter Subscriptions | View all newsletters/channels the connected accounts follow with subscriber counts |
| 196 | Newsletter Viewer | Read posts and view reactions within subscribed newsletters |
| 197 | Unsubscribe Action | Unfollow/unsubscribe from newsletters directly from the dashboard |

---

## 7. Functional Requirements — Release Phases

### Phase 1 — Foundation (MVP)

**Milestone: Get organizations running with a shared inbox and multi-account media pool.**

- Authentication (login, registration, session management)
- Organization creation and member management
- User management with basic roles
- Live chat inbox with real-time messaging
- Core WhatsApp messaging (text, images, video, audio, documents, contacts, location)
- Message reactions, editing, revocation, starring, read receipts
- WhatsApp instance connection (QR pairing, **Passkey pairing**, disconnect, reconnect, **Presence/Pulse configuration**, and **WhatsApp profile settings**)
- **Multi-account media pool** (connect multiple GOWA accounts, capacity-aware routing, file→account mapping, R2 media registry)
- Basic contact management
- Per-user settings and theme preferences
- CSV import/export

### Phase 2 — Workspace & Automation

**Milestone: Scale team operations with automation and collaboration.**

- SSO integration
- Phone number masking
- Custom roles with fine-grained permissions
- Team management with assignment strategies
- Tag management and contact tagging
- Canned responses with keyboard shortcuts
- Chatbot auto-replies (keywords, greeting, fallback)
- Business hours and SLA configuration
- In-app notifications
- Agent availability and transfer queue
- Internal conversation notes
- Conversation collaborators
- Auto-reply for direct messages
- Call detection and auto-reject
- Message extraction and history export
- Extended messaging (stickers, link previews, polls, disappearing messages)
- **WhatsApp Group Manager** (Create groups, manage participants/admins, lock settings, announce mode, retrieve/reset invite links, approve/reject join requests, and CSV participants export)
- **Media pool analytics** (per-account sending stats, capacity utilization)

### Phase 3 — Advanced & Enterprise

**Milestone: Full-featured engagement platform.**

- Campaign management with scheduling and analytics (media routed through pool)
- Group campaigns and group join campaigns
- Advanced canned responses with media
- Webhook integrations with HMAC signing
- Custom actions
- SLA escalation and reminders
- Theme presets and chat background customization
- PDF export
- API key management
- Customer routing / agent selection menu
- License and quota management
- Facebook prospecting tools
- **WhatsApp Newsletters & Channels** (subscriptions listing, message viewing, unfollow actions)
- Message analytics (delivery, read, and response metrics from local database logs)

---

## 8. Non-Functional Requirements

### Performance
- Page first contentful paint under 1.5 seconds
- Interactive time under 3 seconds
- WebSocket event delivery under 2 seconds (p95)
- Data operations under 300ms response time (p95)

### Scalability
- Support 1,000+ organizations
- 50+ concurrent agents per organization
- 99.9% uptime target

### Security
- Role-based access control with granular permissions
- Phone number masking for privacy
- Tenant data isolation between organizations
- Audit logging across all operations
- SSO integration support
- HMAC-signed webhooks

### Reliability
- Automatic retry with idempotency for failed operations
- Webhook delivery with exponential backoff
- WebSocket auto-reconnection on disconnect
- Real-time presence pulse for connection monitoring

### Accessibility & Internationalization
- 15+ language support with RTL
- Keyboard navigation support
- Skip-to-content links
- Responsive design for mobile and desktop

### Deployment & Hosting
- **VPS Hosting**: Both GOWA (Go WhatsApp Web gateway) and the encantoWhatsapp server application are hosted on your Virtual Private Server (VPS) instance, allowing high-performance REST and websocket communication.
- **Media Storage**: All message attachment binaries (images, videos, audio, documents) are stored off-host in Cloudflare R2 object storage (S3-compatible, zero-egress fees).

---

## 9. Permission Model

The platform uses a capability-based permission system where each feature area has its own permission string. Key permission areas include:

| Area | Permissions |
|------|-------------|
| Users | `users` — full user management access |
| Teams | `teams` — create, edit, delete teams |
| Roles | `roles` — manage roles and permissions |
| Chat | `chat`, `chat.assign`, `chat.bypass_claim`, `chat.collaborators` |
| Contacts | `contacts` — manage contact records |
| Campaigns | `campaigns` — create and manage campaigns |
| Templates | `templates` — manage local canned responses and saved templates |
| Chatbot | `chatbot.keywords`, `chatbot.ai`, `transfers` |
| Settings | `settings.general`, `settings.chatbot`, `settings.sso`, `settings.uploads_cleanup` |
| Analytics | `analytics`, `analytics.agents` |
| Integrations | `webhooks`, `api_keys`, `custom_actions` |
| WhatsApp | `accounts`, `agent_selection`, `wa_filter`, `group_directory`, `group_participants` |
| Content | `canned_responses`, `saved_contents`, `tags`, `catalogs` |
| Organization | `organizations` |

---

## 10. Out of Scope

- WhatsApp account spam/ban prevention or compliance consulting
- Billing and payment processing
- Marketing website and landing pages
- Mobile native applications
- Email marketing campaigns
- Social media scheduling beyond Facebook tools
- Third-party CRM integration (Salesforce, HubSpot, etc.)

---

## 11. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| WhatsApp Web protocol changes by Meta may break GOWA connectivity | High | Monitor whatsmeow updates and keep GOWA backend binary updated to the latest release |
| Multi-tenant data leakage | Critical | Strict org-scoping on all data queries; audit logging; permission enforcement |
| WebSocket connection limits at scale | Medium | Connection pooling; presence pulse; automatic reconnection |
| Campaign delivery rate variability | Medium | Per-recipient status tracking; retry logic; delivery analytics |
| Agent collision on shared conversations | High | Conversation claiming workflow; real-time presence; lock semantics |
