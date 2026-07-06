# Whatomate Feature Workflows

> **Updated:** 2026-06-18  
> Complete documentation of all features — backend handlers + frontend service/store/view per feature  
> **Exclusions:** Dashboard/ directory  
> **Total features documented:** 102

---

## Table of Contents

1. [Authentication & Authorization](#1-authentication--authorization)
2. [User Management](#2-user-management)
3. [Organization Management](#3-organization-management)
4. [Roles & Permissions (RBAC)](#4-roles--permissions-rbac)
5. [API Key Management](#5-api-key-management)
6. [WhatsApp Account Management](#6-whatsapp-account-management)
7. [WhatsApp Instance Management (WhatsMeow)](#7-whatsapp-instance-management-whatsmeow)
8. [Contact Management](#8-contact-management)
9. [Chat & Messaging](#9-chat--messaging)
10. [Message Sending (Unified)](#10-message-sending-unified)
11. [Media Handling](#11-media-handling)
12. [Webhook Processing (Meta)](#12-webhook-processing-meta)
13. [Bulk Campaign Management](#13-bulk-campaign-management)
14. [Campaign Worker Processing](#14-campaign-worker-processing)
15. [Chatbot Automation](#15-chatbot-automation)
16. [Chatbot AI Integration](#16-chatbot-ai-integration)
17. [Chatbot Keyword Rules](#17-chatbot-keyword-rules)
18. [Chatbot Flows](#18-chatbot-flows)
19. [Agent Transfers](#19-agent-transfers)
20. [SLA Processing](#20-sla-processing)
21. [Templates Management (Meta)](#21-templates-management-meta)
22. [WhatsApp Flows (Meta)](#22-whatsapp-flows-meta)
23. [Catalog & Products (Meta)](#23-catalog--products-meta)
24. [Canned Responses](#24-canned-responses)
25. [Tags Management](#25-tags-management)
26. [Teams Management](#26-teams-management)
27. [Analytics & Dashboard](#27-analytics--dashboard)
28. [Meta Analytics](#28-meta-analytics)
29. [Widgets (Custom Analytics)](#29-widgets-custom-analytics)
30. [Webhooks (Outbound)](#30-webhooks-outbound)
31. [Custom Actions](#31-custom-actions)
32. [Conversation Notes](#32-conversation-notes)
33. [Status Updates](#33-status-updates)
34. [SSO Authentication](#34-sso-authentication)
35. [WebSocket Real-time Communication](#35-websocket-real-time-communication)
36. [Import/Export Data](#36-importexport-data)
37. [Lead Requests](#37-lead-requests)
38. [Activity Logging & Retention](#38-activity-logging--retention)
39. [Data Migration](#39-data-migration)
40. [Crypto Migration](#40-crypto-migration)
41. [Chat Assignment & Routing](#41-chat-assignment--routing)
42. [Contact Collaborators](#42-contact-collaborators)
43. [Notifications](#43-notifications)
44. [Business Profile Management](#44-business-profile-management)
45. [Instance Auto-Campaign](#45-instance-auto-campaign)
46. [Chat Cleanup](#46-chat-cleanup)
47. [Chat Close Ratings](#47-chat-close-ratings)
48. [Health & Readiness Checks](#48-health--readiness-checks)
49. [Rate Limiting](#49-rate-limiting)
50. [Frontend Serving](#50-frontend-serving)
51. [Send Restriction Policies](#51-send-restriction-policies)
52. [Agent Chat Visibility Restrictions](#52-agent-chat-visibility-restrictions)
53. [Contact Account Resolution](#53-contact-account-resolution)
54. [Contact Repair](#54-contact-repair)
55. [Contact User Deletions](#55-contact-user-deletions)
56. [Closed Chat Filters](#56-closed-chat-filters)
57. [Chat Lifecycle Management](#57-chat-lifecycle-management)
58. [Reply Preview Helpers](#58-reply-preview-helpers)
59. [Message Template Placeholders](#59-message-template-placeholders)
60. [Campaign Policy Enforcement](#60-campaign-policy-enforcement)
61. [Flows Helpers](#61-flows-helpers)
62. [Group Message Helpers](#62-group-message-helpers)
63. [Instance Name Validation](#63-instance-name-validation)
64. [Instance Selector](#64-instance-selector)
65. [Password Policy](#65-password-policy)
66. [Provider Guard](#66-provider-guard)
67. [Reason Codes](#67-reason-codes)
68. [Security Headers & CSRF Protection](#68-security-headers--csrf-protection)
69. [Request Logging & Recovery](#69-request-logging--recovery)
70. [SSRF-Safe Dialer](#70-ssrf-safe-dialer)
71. [Cache System](#71-cache-system)
72. [Cookie Management](#72-cookie-management)
73. [JWT Secret Management](#73-jwt-secret-management)
74. [WhatsApp Client (Meta)](#74-whatsapp-client-meta)
75. [Message Provider Abstraction](#75-message-provider-abstraction)
76. [WhatsMeow Connection Manager](#76-whatsmeow-connection-manager)
77. [WhatsMeow Queue Manager](#77-whatsmeow-queue-manager)
78. [Redis Queue System](#78-redis-queue-system)
79. [Campaign Stats Subscriber](#79-campaign-stats-subscriber)
80. [Database Migrations](#80-database-migrations)
81. [Encryption System](#81-encryption-system)
82. [Contact Utilities](#82-contact-utilities)
83. [Template Utilities](#83-template-utilities)
84. [WebSocket Message Types](#84-websocket-message-types)
85. [Frontend Embedded Build](#85-frontend-embedded-build)
86. [Configuration System](#86-configuration-system)
87. [Model Layer](#87-model-layer)
88. [Middleware Chain](#88-middleware-chain)
89. [Error Handling Patterns](#89-error-handling-patterns)
90. [Testing Infrastructure](#90-testing-infrastructure)
91. [App Configuration Endpoint](#91-app-configuration-endpoint)
92. [User Settings & Chat Background](#92-user-settings--chat-background)
93. [Availability Management](#93-availability-management)
94. [Change Password](#94-change-password)
95. [Contact Phone Start (WhatsMeow)](#95-contact-phone-start-whatsmeow)
96. [Interactive Messages](#96-interactive-messages)
97. [Typing Presence](#97-typing-presence)
98. [Agent Role Chat Scoping](#98-agent-role-chat-scoping)
99. [Organization Outbound Mode](#99-organization-outbound-mode)
100. [Strict Rollout Mode](#100-strict-rollout-mode)
101. [WhatsApp Poll Messages](#101-whatsapp-poll-messages)
102. [Priority Event Ingestion with Sharded FIFO Queues](#102-priority-event-ingestion-with-sharded-fifo-queues)

---

## 1. Authentication & Authorization

**Backend:** `internal/handlers/auth_handlers.go`, `auth_crypto.go`, `auth_utils.go`, `auth_expiry.go`, `auth_types.go`  
**Frontend:** `frontend/src/services/auth.ts`, `frontend/src/stores/auth.ts`, `frontend/src/views/auth/*`

| Function | Backend Handler | Frontend Service | Store/View |
|----------|----------------|-----------------|------------|
| Login | `Login()` | `auth.login()` | `auth store` |
| Register | `Register()` | `auth.register()` | Auth views |
| Refresh token | `RefreshToken()` | `auth.refreshToken()` | `auth store` |
| Logout | `Logout()` | `auth.logout()` | `auth store` |
| Switch org | `SwitchOrg()` | `auth.switchOrg()` | `auth store` |
| Get WS token | `GetWSToken()` | — | WebSocket init |

**Flow:** Email/password → bcrypt verify → JWT issued → Middleware validates on every request.

---

## 2. User Management

**Backend:** `internal/handlers/users.go`  
**Frontend:** `frontend/src/services/users.ts`, `frontend/src/stores/*`, `frontend/src/views/users/`

| Function | Backend Handler | Frontend Service | View |
|----------|----------------|-----------------|------|
| List users | `ListUsers()` | `users.list()` | `UserList.vue` |
| Get user | `GetUser()` | `users.get()` | `UserDetail.vue` |
| Create user | `CreateUser()` | `users.create()` | `UserForm.vue` |
| Update user | `UpdateUser()` | `users.update()` | `UserForm.vue` |
| Delete user | `DeleteUser()` | `users.delete()` | User list |

---

## 3. Organization Management

**Backend:** `internal/handlers/organization.go`  
**Frontend:** `frontend/src/services/organizations.ts`, `frontend/src/views/settings/`

| Function | Backend Handler | Frontend Service |
|----------|----------------|-----------------|
| List orgs | `ListOrganizations()` | `organizations.list()` |
| Get current | `GetCurrentOrganization()` | `organizations.getCurrent()` |
| Create | `CreateOrganization()` | `organizations.create()` |
| Delete | `DeleteOrganization()` | `organizations.delete()` |
| Settings | `GetOrganizationSettings()` / `UpdateOrganizationSettings()` | `organizations.getSettings()` / `updateSettings()` |
| Members | `ListOrganizationMembers()` / `AddOrganizationMember()` / `RemoveOrganizationMember()` / `UpdateOrganizationMemberRole()` | `organizations.*` |

**Frontend Views:** `SettingsView.vue` — tabs for general, members, notifications

---

## 4. Roles & Permissions (RBAC)

**Backend:** `internal/handlers/roles.go`  
**Frontend:** `frontend/src/services/roles.ts`, `frontend/src/views/roles/`

| Function | Backend Handler | Frontend Service | View |
|----------|----------------|-----------------|------|
| List roles | `ListRoles()` | `roles.list()` | `RoleList.vue` |
| Get role | `GetRole()` | `roles.get()` | `RoleDetail.vue` |
| Create role | `CreateRole()` | `roles.create()` | `RoleForm.vue` |
| Update role | `UpdateRole()` | `roles.update()` | `RoleForm.vue` |
| Delete role | `DeleteRole()` | `roles.delete()` | Role list |
| Permissions | `ListPermissions()` | `roles.listPermissions()` | Role form |

**Permission Model:** `resource:action` — e.g. `contact:read`, `campaign:create`. Predefined roles: Admin, Manager, Agent.

---

## 5. API Key Management

**Backend:** `internal/handlers/apikeys.go`  
**Frontend:** `frontend/src/services/*`

| Function | Backend Handler | Description |
|----------|----------------|-------------|
| List API keys | `ListAPIKeys()` | Lists keys (prefix only, never full key) |
| Create key | `CreateAPIKey()` | Returns full key once |
| Delete key | `DeleteAPIKey()` | Revokes key |

**Usage:** `X-API-Key` header auth alternative to JWT.

---

## 6. WhatsApp Account Management

**Backend:** `internal/handlers/accounts.go`  
**Frontend:** `frontend/src/services/accounts.ts`, `frontend/src/views/accounts/`

| Function | Backend Handler | Frontend Service | View |
|----------|----------------|-----------------|------|
| List accounts | `ListAccounts()` | `accounts.list()` | `AccountList.vue` |
| Create account | `CreateAccount()` | `accounts.create()` | `AccountForm.vue` |
| Get account | `GetAccount()` | `accounts.get()` | `AccountDetail.vue` |
| Update | `UpdateAccount()` | `accounts.update()` | `AccountForm.vue` |
| Delete | `DeleteAccount()` | `accounts.delete()` | Account list |
| Test | `TestAccountConnection()` | `accounts.test()` | Account detail |
| Subscribe | `SubscribeApp()` | `accounts.subscribe()` | Subscribe to Meta webhook |

**Security:** AccessToken and AppSecret encrypted via `crypto.Encrypt()` before storage.

---

## 7. WhatsApp Instance Management (WhatsMeow)

**Backend:** `internal/handlers/instances.go`  
**Frontend:** `frontend/src/services/instances.ts`, `frontend/src/stores/instances.ts`, `frontend/src/views/instances/`

| Function | Backend Handler | Frontend Service | View |
|----------|----------------|-----------------|------|
| List | `ListInstances()` | `instances.list()` | `InstanceList.vue` |
| Create | `CreateInstance()` | `instances.create()` | `InstanceForm.vue` |
| Update | `UpdateInstance()` | `instances.update()` | Instance form |
| Delete | `DeleteInstance()` | `instances.delete()` | Instance list |
| Connect | `ConnectInstance()` | `instances.connect()` | QR code display |
| Disconnect | `DisconnectInstance()` | `instances.disconnect()` | Instance detail |
| Reconnect | `ReconnectInstance()` | `instances.reconnect()` | Instance detail |
| QR code | `GetInstanceQRCodeSnapshot()` | `instances.getQR()` | QR component |
| Pair phone | `PairPhoneInstance()` | `instances.pairPhone()` | Pairing form |
| Health | `GetInstanceHealth()` | `instances.getHealth()` | Instance detail |

**WhatsMeow Manager:** `pkg/whatsmeow/manager.go` — ConnectionManager manages multi-instance pool.

---

## 8. Contact Management

**Backend:** `internal/handlers/contacts.go`, `contacts_management.go`, `contacts_messaging.go`  
**Frontend:** `frontend/src/services/contacts.ts`, `frontend/src/stores/contacts.ts`, `frontend/src/views/contacts/`

| Function | Backend Handler | Frontend Service | Store/View |
|----------|----------------|-----------------|------------|
| List contacts | `ListContacts()` | `contacts.list()` | `contacts store` |
| Get contact | `GetContact()` | `contacts.get()` | Chat view |
| Create | `CreateContact()` | Contact form | Contact form |
| Update | `UpdateContact()` | Contact form | Contact form |
| Delete | `DeleteContact()` | | Contact list |

**Contact Store:** `frontend/src/stores/contacts.ts` — 25+ functions for contact/message state management.

---

## 9. Chat & Messaging

**Backend:** `internal/handlers/contacts.go`, `chat_lifecycle.go`  
**Frontend:** `frontend/src/services/contacts.ts`, `frontend/src/stores/contacts.ts`, `frontend/src/views/chat/`

| Operation | Backend | Frontend |
|-----------|---------|----------|
| List chats | `ListContacts()` | `contacts.list()` → `ChatList.vue` |
| Get messages | `GetMessages()` | `contacts.getMessages()` → `MessageList.vue` |
| Claim chat | Chat lifecycle | Contact assign action |
| Close chat | `closeChatUpdates()` | Chat close action |
| Reopen chat | `reopenChatUpdates()` | Chat reopen action |

**View:** `frontend/src/views/chat/ChatView.vue` — Main chat interface.

---

## 10. Message Sending (Unified)

**Backend:** `internal/handlers/messages.go`  
**Frontend:** `frontend/src/services/contacts.ts`

| Type | Backend Method | Provider Method |
|------|---------------|-----------------|
| Text | `SendOutgoingMessage()` → `sendViaProvider()` | `Provider.SendText()` |
| Image | Same path | `Provider.SendImage()` |
| Document | Same path | `Provider.SendDocument()` |
| Video | Same path | `Provider.SendVideo()` |
| Audio | Same path | `Provider.SendAudio()` |
| Template | `SendTemplateMessage()` | `Provider.SendTemplate()` |
| Reaction | `SendReaction()` | `Provider.SendReaction()` |
| Revoke | `RevokeMessage()` | `Provider.RevokeMessage()` |

**Provider Resolution:** `resolveProviderInstanceID()` chooses Cloud API or WhatsApp Web per message.

---

## 11. Media Handling

**Backend:** `internal/handlers/media.go`, `media_visibility.go`  
**Frontend:** Media display in `ChatView.vue`

| Operation | Backend | Description |
|-----------|---------|-------------|
| Download | `DownloadAndSaveMedia()` | Downloads from WhatsApp → stores to object/local storage |
| Serve | `ServeMedia()` | Serves with correct Content-Type, supports range requests |
| Retry | `RetryMediaDownload()` | Retries failed downloads |
| Visibility | Media visibility policy | Access control |

**Workers:** `uploads_cleanup_*.go` — Periodic cleanup of expired uploads.

---

## 12. Webhook Processing (Meta)

**Backend:** `internal/handlers/webhook.go`  
**Frontend:** Not applicable (server-to-server)

| Operation | Method | Description |
|-----------|--------|-------------|
| Verify | `WebhookVerify()` | GET handler for Meta challenge |
| Receive | `WebhookHandler()` | POST handler for inbound events |
| Deduplicate | `processIncomingMessageWithoutDuplicateCheck()` | Prevents duplicate processing |
| Status updates | `processStatusUpdate()` | Message delivery/read status |
| Template status | `processTemplateStatusUpdate()` | Template review updates |

---

## 13. Bulk Campaign Management

**Backend:** `internal/handlers/campaigns.go`, `campaign_policy.go`, `campaign_scheduler.go`, `campaign_start.go`  
**Frontend:** `frontend/src/services/campaigns.ts`, `frontend/src/stores/campaigns.ts`, `frontend/src/views/campaigns/`

| Operation | Backend Handler | Frontend Service | View |
|-----------|----------------|-----------------|------|
| List | `ListCampaigns()` | `campaigns.list()` | `CampaignList.vue` |
| Create | `CreateCampaign()` | `campaigns.create()` | `CampaignCreate.vue` |
| Update | `UpdateCampaign()` | `campaigns.update()` | Campaign form |
| Delete | `DeleteCampaign()` | `campaigns.delete()` | Campaign list |
| Start | `StartCampaign()` | `campaigns.start()` | Campaign detail |
| Pause | `PauseCampaign()` | `campaigns.pause()` | Campaign detail |
| Cancel | `CancelCampaign()` | `campaigns.cancel()` | Campaign detail |
| Retry | `RetryFailed()` | `campaigns.retry()` | Campaign detail |
| Recipients | `ImportRecipients()` / `GetCampaignRecipients()` / `DeleteCampaignRecipient()` | `campaigns.*` | Campaign detail |
| Media | `UploadCampaignMedia()` / `ServeCampaignMedia()` | `campaigns.uploadMedia()` | Campaign create |

---

## 14. Campaign Worker Processing

**Backend:** `internal/worker/worker.go`, `campaign_delay.go`, `send_policy.go`  
**Frontend:** Real-time stats via WebSocket

| Step | Backend Function | Description |
|------|-----------------|-------------|
| Enqueue | `StartCampaign()` → enqueue to Redis Stream | All recipients queued |
| Process | `HandleRecipientJob()` → `executeRecipientSend()` | Individual send |
| Delay | `computeCampaignDelayDuration()` | Random delay between sends |
| Status | `updateRecipientStatusConditional()` | Track per-recipient status |
| Stats | `publishCampaignStats()` → Redis Pub/Sub | Real-time progress |
| Complete | `checkCampaignCompletion()` | Auto-detect completion |

---

## 15. Chatbot Automation

**Backend:** `internal/handlers/chatbot.go`, `chatbot_processor.go`  
**Frontend:** `frontend/src/services/chatbot.ts`, `frontend/src/stores/chatbot.ts`, `frontend/src/views/chatbot/`

| Operation | Backend Handler | Frontend Service | View |
|-----------|----------------|-----------------|------|
| Settings | `GetChatbotSettings()` / `UpdateChatbotSettings()` | `chatbot.getSettings()` / `updateSettings()` | `ChatbotSettings.vue` |
| Keywords | CRUD on `/api/chatbot/keywords` | `chatbot.*` | `KeywordRules.vue` |
| Flows | CRUD on `/api/chatbot/flows` | `chatbot.*` | `FlowBuilder.vue` |
| AI Contexts | CRUD on `/api/chatbot/ai-contexts` | `chatbot.*` | `AIContexts.vue` |
| Sessions | `ListChatbotSessions()` / `GetChatbotSession()` | `chatbot.*` | Session list |

**Processing Pipeline:** Incoming message → keyword matching → flow execution → AI response → agent handoff.

---

## 16. Chatbot AI Integration

**Backend:** `internal/handlers/chatbot_processor.go`  
**Frontend:** `frontend/src/views/chatbot/AIContexts.vue`

| Component | Description |
|-----------|-------------|
| AI Provider | Configurable (OpenAI, etc.) |
| AI Model | Configurable per org |
| System Prompt | Customizable per org |
| AI Contexts | Trigger-specific context documents |
| Max Tokens | Configurable limit |

---

## 17. Chatbot Keyword Rules

**Backend:** `internal/handlers/chatbot.go`  
**Frontend:** `frontend/src/views/chatbot/KeywordRules.vue`

| Field | Description |
|-------|-------------|
| Keywords | Trigger words/phrases |
| Match Type | `exact`, `contains`, `regex` |
| Response Type | `text`, `flow`, `api`, `transfer` |
| Priority | Higher priority matched first |
| Enabled | Toggle on/off |

---

## 18. Chatbot Flows

**Backend:** `internal/handlers/chatbot.go`  
**Frontend:** `frontend/src/views/chatbot/FlowBuilder.vue`

| Step Type | Description |
|-----------|-------------|
| Send | Send message to user |
| Input | Wait for user input |
| API Call | External API integration |
| Condition | Branch based on input |
| Transfer | Transfer to agent |

**Composable:** `frontend/src/composables/useFlowSimulation.ts` — test flows in browser.

---

## 19. Agent Transfers

**Backend:** `internal/handlers/agent_transfers.go`  
**Frontend:** Chat interface — transfer button

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListAgentTransfers()` |
| Create | `CreateAgentTransfer()` |
| Pick next | `PickNextTransfer()` |
| Resume | `ResumeFromTransfer()` |
| Assign | `AssignAgentTransfer()` |

---

## 20. SLA Processing

**Backend:** `internal/handlers/sla_processor.go`  
**Frontend:** Chatbot settings

| Metric | Description |
|--------|-------------|
| Response time | Time until first agent response |
| Resolution time | Time until chat closed |
| Escalation time | Time before escalation |
| Auto-close | Close chat after inactivity |

---

## 21. Templates Management (Meta)

**Backend:** `internal/handlers/templates.go`  
**Frontend:** `frontend/src/services/templates.ts`, `frontend/src/views/templates/`

| Operation | Backend Handler | Frontend Service | View |
|-----------|----------------|-----------------|------|
| List | `ListTemplates()` | `templates.list()` | `TemplateList.vue` |
| Create | `CreateTemplate()` | `templates.create()` | `TemplateEditor.vue` |
| Update | `UpdateTemplate()` | `templates.update()` | Template editor |
| Delete | `DeleteTemplate()` | `templates.delete()` | Template list |
| Submit | `SubmitTemplate()` | `templates.submit()` | Submit to Meta |
| Sync | `SyncTemplates()` | `templates.sync()` | Sync from Meta |

**Template status flow:** Draft → Submitted → Under Review → Approved / Rejected.

---

## 22. WhatsApp Flows (Meta)

**Backend:** `internal/handlers/flows.go`  
**Frontend:** `frontend/src/services/flows.ts`, `frontend/src/views/*`

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListFlows()` |
| Create | `CreateFlow()` |
| Update | `UpdateFlow()` |
| Delete | `DeleteFlow()` |
| Save to Meta | `SaveFlowToMeta()` |
| Publish | `PublishFlow()` |
| Deprecate | `DeprecateFlow()` |
| Duplicate | `DuplicateFlow()` |
| Sync | `SyncFlows()` |

---

## 23. Catalog & Products (Meta)

**Backend:** `internal/handlers/catalog.go`  
**Frontend:** `frontend/src/services/catalogs.ts`

| Operation | Backend Handler |
|-----------|----------------|
| List catalogs | `ListCatalogs()` |
| Create catalog | `CreateCatalog()` |
| Delete catalog | `DeleteCatalog()` |
| Sync catalogs | `SyncCatalogs()` |
| List products | `ListCatalogProducts()` |
| CRUD products | `CreateCatalogProduct()`, `UpdateCatalogProduct()`, `DeleteCatalogProduct()` |

---

## 24. Canned Responses

**Backend:** `internal/handlers/canned_responses.go`, `canned_response_send.go`, `canned_response_media.go`  
**Frontend:** Chat interface — quick reply selector

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListCannedResponses()` |
| Create | `CreateCannedResponse()` |
| Update | `UpdateCannedResponse()` |
| Delete | `DeleteCannedResponse()` |
| Send | `SendCannedResponse()` |
| Usage | `IncrementCannedResponseUsage()` |

---

## 25. Tags Management

**Backend:** `internal/handlers/tags.go`  
**Frontend:** Contact detail — tag selector

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListTags()` |
| Create | `CreateTag()` |
| Update | `UpdateTag()` |
| Delete | `DeleteTag()` |

---

## 26. Teams Management

**Backend:** `internal/handlers/teams.go`  
**Frontend:** `frontend/src/services/teams.ts`, `frontend/src/views/teams/`

| Operation | Backend Handler | Frontend Service | View |
|-----------|----------------|-----------------|------|
| List | `ListTeams()` | `teams.list()` | `TeamList.vue` |
| Create | `CreateTeam()` | `teams.create()` | `TeamForm.vue` |
| Update | `UpdateTeam()` | `teams.update()` | Team form |
| Delete | `DeleteTeam()` | `teams.delete()` | Team list |
| Members | List/Add/Remove team members | `teams.*` | Team detail |

---

## 27. Analytics & Dashboard

**Backend:** `internal/handlers/analytics.go`  
**Frontend:** `frontend/src/services/analytics.ts`, `frontend/src/views/analytics/`

| Operation | Backend Handler | Frontend Service |
|-----------|----------------|-----------------|
| Dashboard stats | `GetDashboardStats()` | `analytics.getDashboardStats()` |
| Message analytics | — | Message metrics |
| Chatbot analytics | — | Bot performance |
| Campaign analytics | — | Campaign metrics |

**Widgets:** Custom dashboard widgets (`internal/handlers/widgets.go`) with data sources: messages, contacts, campaigns, transfers, sessions.

---

## 28. Meta Analytics

**Backend:** `internal/handlers/meta_analytics.go`  
**Frontend:** Analytics views

| Operation | Description |
|-----------|-------------|
| Get analytics | Fetch Meta-provided message analytics |
| List accounts | List accounts eligible for analytics |
| Refresh cache | Force refresh analytics cache |

---

## 29. Widgets (Custom Analytics)

**Backend:** `internal/handlers/widgets.go`  
**Frontend:** Analytics dashboard

| Operation | Backend Handler | Description |
|-----------|----------------|-------------|
| List | `ListWidgets()` | User's widgets |
| Create | `CreateWidget()` | New widget |
| Update | `UpdateWidget()` | Modify widget |
| Delete | `DeleteWidget()` | Remove widget |
| Layout | `SaveWidgetLayout()` | Grid position |
| Data | `GetWidgetData()` / `GetAllWidgetsData()` | Query data |

**Data Sources:** messages, contacts, campaigns, transfers, sessions  
**Display Types:** number, bar, line, pie, table

---

## 30. Webhooks (Outbound)

**Backend:** `internal/handlers/webhooks.go`, `webhook_dispatch.go`, `webhook_security.go`  
**Frontend:** `frontend/src/services/webhooks.ts`, `frontend/src/views/settings/`

| Operation | Backend Handler | Frontend Service |
|-----------|----------------|-----------------|
| List | `ListWebhooks()` | `webhooks.list()` |
| Create | `CreateWebhook()` | `webhooks.create()` |
| Update | `UpdateWebhook()` | `webhooks.update()` |
| Delete | `DeleteWebhook()` | `webhooks.delete()` |
| Test | `TestWebhook()` | `webhooks.test()` |

**Events:** message.sent, message.received, message.failed, message.read, contact.new, campaign.updated, chatbot.handoff

**Security:** Payload signed with secret, SSRFSafeDialer prevents SSRF.

---

## 31. Custom Actions

**Backend:** `internal/handlers/custom_actions.go`, `custom_action_runtime.go`  
**Frontend:** Chatbot flow builder

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListCustomActions()` |
| Create | `CreateCustomAction()` |
| Update | `UpdateCustomAction()` |
| Delete | `DeleteCustomAction()` |
| Execute | Runtime execution engine |

---

## 32. Conversation Notes

**Backend:** `internal/handlers/conversation_notes.go`  
**Frontend:** Chat view — notes panel

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListNotes()` |
| Create | `CreateNote()` |
| Update | `UpdateNote()` |
| Delete | `DeleteNote()` |

---

## 33. Status Updates

**Backend:** `internal/handlers/statuses.go`, `pkg/whatsmeow/statuses.go`  
**Frontend:** Status view

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListStatuses()` |
| Serve media | Status media handler |
| Reply | Reply to status |
| Seen | Mark status as seen |

---

## 34. SSO Authentication

**Backend:** `internal/handlers/sso_handlers.go`, `sso_types.go`, `sso_utils.go`, `sso_security.go`  
**Frontend:** Login page — SSO buttons

| Operation | Backend Handler | Description |
|-----------|----------------|-------------|
| List providers | `GetPublicSSOProviders()` | Available SSO options |
| Initiate | `InitSSO()` | Redirect to provider |
| Callback | `CallbackSSO()` | Handle provider callback |
| Settings | `GetSSOSettings()` / `UpdateSSOProvider()` / `DeleteSSOProvider()` | Org SSO config |

**Supported:** Generic OIDC / OAuth 2.0 providers.

---

## 35. WebSocket Real-time Communication

**Backend:** `internal/websocket/hub.go`, `internal/handlers/websocket.go`  
**Frontend:** `frontend/src/services/websocket.ts`

| Operation | Backend | Description |
|-----------|---------|-------------|
| Connect | `WebSocketHandler()` | Upgrade HTTP → WS |
| Broadcast org | `BroadcastToOrg()` | All org clients |
| Broadcast contact | `BroadcastToContact()` | Clients viewing a chat |
| Broadcast user | `BroadcastToUser()` | Specific user |
| Token | `validateWSToken()` | WS auth validation |

**Events:** new_message, message_status, contact_update, campaign_stats, instance_status, notification, chatbot_event

---

## 36. Import/Export Data

**Backend:** `internal/handlers/import_export.go`  
**Frontend:** Settings — import/export UI

| Operation | Backend Handler |
|-----------|----------------|
| Export | `ExportData()` with config-driven columns |
| Import | `ImportData()` with validation |
| Config | `GetExportConfig()` / `GetImportConfig()` |

**Supported:** Contacts, campaigns, templates (CSV format).

---

## 37. Lead Requests

**Backend:** Plugin or handler  
**Frontend:** Public lead capture forms

| Operation | Description |
|-----------|-------------|
| Create | Public form submission to create lead |
| List | List lead requests |
| Update status | Accept/decline leads |

---

## 38. Activity Logging & Retention

**Backend:** Activity middleware + retention worker  
**Frontend:** Activity log views

| Operation | Description |
|-----------|-------------|
| Log | Middleware logs API activity |
| List | Activity log query endpoint |
| Retention | Worker cleans old activity logs |

---

## 39. Data Migration

**Backend:** Database migration system  
**Frontend:** Not applicable

| Operation | Description |
|-----------|-------------|
| Trigger | Run migration |
| Status | Check migration progress |

---

## 40. Crypto Migration

**Backend:** `cmd/whatomate/main.go` — `runCryptoMigrate()`  
**Frontend:** Not applicable

| Version | Algorithm |
|---------|-----------|
| V2 | Legacy encryption |
| V3 | Current encryption with improved KDF |
| Migration | One-shot CLI command to re-encrypt all fields |

---

## 41. Chat Assignment & Routing

**Backend:** `internal/handlers/chat_assignment_reset_settings.go`, `chat_assignment_reset_worker.go`, `agent_selection.go`  
**Frontend:** Chat assignment UI

| Operation | Description |
|-----------|-------------|
| Settings | Configure auto-assignment rules |
| Strategy | Round-robin, least-busy, skill-based |
| Reset | Periodic reset of assignments |
| Selection | Agent selection algorithm |

---

## 42. Contact Collaborators

**Backend:** `internal/handlers/contact_collaborators.go`, `contact_collaborators_helpers.go`  
**Frontend:** Chat view — collaborator panel

| Operation | Backend Handler |
|-----------|----------------|
| List | `ListCollaborators()` |
| Invite | `InviteCollaborator()` |
| Accept/Decline | Collaboration response |
| Remove | `RemoveCollaborator()` |

---

## 43. Notifications

**Backend:** `internal/handlers/notifications.go`  
**Frontend:** `frontend/src/services/notifications.ts`, notification bell

| Operation | Backend Handler | Frontend Service |
|-----------|----------------|-----------------|
| List | `ListNotifications()` | `notifications.list()` |
| Dismiss | `DismissNotification()` | `notifications.dismiss()` |

**Types:** Campaign complete, new message, chatbot handoff, SLA breach.

---

## 44. Business Profile Management

**Backend:** `internal/handlers/business_profile.go`  
**Frontend:** Settings — business profile

| Operation | Backend Handler |
|-----------|----------------|
| Get | `GetBusinessProfile()` |
| Update | `UpdateBusinessProfile()` |
| Picture | `UpdateProfilePicture()` |

---

## 45. Instance Auto-Campaign

**Backend:** `internal/handlers/instance_auto_campaign_media.go`, `instance_auto_campaign_worker.go`  
**Frontend:** Instance settings — auto-campaign config

| Operation | Description |
|-----------|-------------|
| Worker | Periodic auto-campaign trigger |
| Media | Upload auto-campaign media |
| Settings | Per-instance auto-campaign config |

---

## 46. Chat Cleanup

**Backend:** `internal/handlers/chat_cleanup.go`  
**Frontend:** Not applicable (background worker)

| Operation | Description |
|-----------|-------------|
| Worker | Periodic cleanup of stale chats |
| Policy | Configurable auto-close timeout |

---

## 47. Chat Close Ratings

**Backend:** `internal/handlers/chat_close_ratings.go`, `pkg/chat_close_ratings/shared.go`  
**Frontend:** Rating UI after chat closes

| Operation | Backend Handler | Shared Package |
|-----------|----------------|----------------|
| Submit rating | `SubmitRating()` | `ChatCloseRating` struct |
| Get ratings | `GetChatRatings()` | Shared types |

---

## 48. Health & Readiness Checks

**Backend:** `internal/handlers/app.go` — `HealthCheck()`, `ReadyCheck()`  
**Frontend:** Not applicable

| Endpoint | Checks |
|----------|--------|
| `/health` | Liveness — process is running |
| `/ready` | Readiness — DB ping succeeds |

---

## 49. Rate Limiting

**Backend:** `internal/middleware/ratelimit.go`  
**Frontend:** Not applicable

| Type | Configuration |
|------|---------------|
| Global | Per-IP rate limit |
| Auth | Login attempt limit |
| Outbound | Per-user message send limit |

**Storage:** Redis with configurable TTL and max requests.

---

## 50. Frontend Serving

**Backend:** `internal/frontend/` with `//go:embed all:dist`  
**Frontend:** Embedded Vue SPA

| Build | Command | Description |
|-------|---------|-------------|
| Development | `make dev-frontend` | Vite dev server |
| Production | `make build-prod` | Build + embed in Go binary |

---

## 51. Send Restriction Policies

**Backend:** `internal/handlers/send_restriction_policy.go`, `user_send_restrictions.go`  
**Frontend:** Settings — send restrictions

| Level | Description |
|-------|-------------|
| Organization | Global outbound mode |
| User | Per-user send restrictions |
| Role | Role-based send limits |

---

## 52. Agent Chat Visibility Restrictions

**Backend:** `internal/handlers/chat_access_policy.go`  
**Frontend:** Chat list filtered by access

| Rule | Description |
|------|-------------|
| Own chats | Agent sees own assigned chats |
| Team chats | Agent sees team chats |
| Public chats | Agent sees unassigned chats |
| Admin | Full visibility |

---

## 53. Contact Account Resolution

**Backend:** `internal/handlers/contact_account_resolution.go`  
**Frontend:** Contact detail — account info

**Purpose:** Resolves which WhatsApp account/provider a contact belongs to.

---

## 54. Contact Repair

**Backend:** `internal/handlers/contact_repair.go`  
**Frontend:** Background process

**Purpose:** Repairs contact records with missing/inconsistent data (e.g., phone numbers, JIDs).

---

## 55. Contact User Deletions

**Backend:** `internal/handlers/contact_user_deletions.go`  
**Frontend:** Contact list

**Purpose:** Tracks which users deleted a contact to prevent re-assignment to the same user.

---

## 56. Closed Chat Filters

**Backend:** `internal/handlers/closed_chat_filters.go`  
**Frontend:** Chat list — closed tab

| Filter | Description |
|--------|-------------|
| Date range | Filter by close date |
| Closed by | Filter by closing user |
| Rating | Filter by close rating |

---

## 57. Chat Lifecycle Management

**Backend:** `internal/handlers/chat_lifecycle.go`  
**Frontend:** Chat state indicators

| State | Transitions |
|-------|-------------|
| Open | New → Agent assigned → ... |
| Pending | Waiting for agent claim |
| Closed | Resolved → Archived |

---

## 58. Reply Preview Helpers

**Backend:** `internal/handlers/reply_preview_helpers.go`  
**Frontend:** Chat — reply preview

**Purpose:** Generates preview of replied-to message in chat UI.

---

## 59. Message Template Placeholders

**Backend:** `internal/handlers/message_template_placeholders.go`  
**Frontend:** Campaign template editor

| Placeholder | Resolution |
|-------------|-----------|
| `{{1}}`, `{{2}}`, etc. | Per-recipient template params |
| `{{contact.name}}` | Contact profile name |
| `{{org.name}}` | Organization name |

---

## 60. Campaign Policy Enforcement

**Backend:** `internal/handlers/campaign_policy.go`  
**Frontend:** Campaign restrictions

| Check | Description |
|-------|-------------|
| Inbound-only | Require prior inbound message |
| Rate limit | Per-campaign send rate |
| Time window | Business hours only |

---

## 61. Flows Helpers

**Backend:** `internal/handlers/flows_helpers_test.go`  
**Frontend:** Flow builder

**Purpose:** Test helpers and utilities for WhatsApp Flow management.

---

## 62. Group Message Helpers

**Backend:** `internal/handlers/group_message_helpers.go`  
**Frontend:** Group campaigns

**Purpose:** Helper functions for sending messages to WhatsApp groups.

---

## 63. Instance Name Validation

**Backend:** `internal/handlers/instance_name_validation.go`  
**Frontend:** Instance create/edit form

| Rule | Description |
|------|-------------|
| Length | 3-50 characters |
| Pattern | Alphanumeric + hyphens |
| Uniqueness | No duplicate names per org |

---

## 64. Instance Selector

**Backend:** `internal/handlers/instance_selector.go`  
**Frontend:** Instance selection UI

| Strategy | Description |
|----------|-------------|
| Default outgoing | Use org default instance |
| Contact-bound | Use instance from contact's last message |
| Load-balanced | Distribute across instances |

---

## 65. Password Policy

**Backend:** `internal/handlers/password_policy.go`  
**Frontend:** Password change form

| Rule | Minimum |
|------|---------|
| Length | 8 characters |
| Complexity | Must include uppercase, lowercase, digit |

---

## 66. Provider Guard

**Backend:** `internal/handlers/provider_guard.go`  
**Frontend:** Feature gating

**Purpose:** Protects features that require specific provider capabilities (e.g., Cloud API vs Web).

---

## 67. Reason Codes

**Backend:** `internal/handlers/reason_codes.go`  
**Frontend:** Message status display

**Purpose:** Standardized error/reason codes for message status tracking.

---

## 68. Security Headers & CSRF Protection

**Backend:** `internal/middleware/middleware.go`  
**Frontend:** Not applicable

| Header | Value |
|--------|-------|
| Content-Security-Policy | Restricts sources |
| X-Content-Type-Options | nosniff |
| X-Frame-Options | DENY |
| Referrer-Policy | strict-origin-when-cross-origin |

---

## 69. Request Logging & Recovery

**Backend:** `internal/middleware/middleware.go`  
**Frontend:** Not applicable

| Middleware | Description |
|------------|-------------|
| Request Logger | Logs method, path, status, duration |
| Recovery | Catches panics, returns 500 |

---

## 70. SSRF-Safe Dialer

**Backend:** `internal/handlers/webhooks.go` — `SSRFSafeDialer()`  
**Frontend:** Not applicable

**Purpose:** Prevents Server-Side Request Forgery by blocking private IP ranges in webhook calls.

---

## 71. Cache System

**Backend:** `internal/handlers/cache.go`  
**Frontend:** Not applicable

| Cache | TTL | Purpose |
|-------|-----|---------|
| Permissions | 5 min | Role permission lookups |
| License | 1 min | License status |
| Config | 5 min | Org settings |

---

## 72. Cookie Management

**Backend:** `internal/handlers/cookies.go`  
**Frontend:** Not applicable

**Purpose:** HTTP cookie operations for auth token storage in cookie-based auth.

---

## 73. JWT Secret Management

**Backend:** `internal/handlers/jwt_secret.go`  
**Frontend:** Not applicable

**Purpose:** Dynamic JWT secret rotation with grace period for overlapping tokens.

---

## 74. WhatsApp Client (Meta)

**Backend:** `pkg/whatsapp/client.go`  
**Frontend:** Not applicable (SDK layer)

| Operation | Method |
|-----------|--------|
| Send text | `Client.SendText()` |
| Send template | `Client.SendTemplate()` |
| Media upload | `Client.UploadMedia()` |
| Media download | `Client.DownloadMedia()` |
| Template CRUD | Template API methods |
| Catalog API | Catalog methods |

---

## 75. Message Provider Abstraction

**Backend:** `pkg/provider/interface.go`  
**Frontend:** Not applicable

| Interface | Methods |
|-----------|---------|
| `MessageProvider` | SendText, SendImage, SendDocument, SendVideo, SendAudio, MarkRead, SendReaction, RevokeMessage, DownloadMedia, UploadMedia, GetMediaURL |
| `PollProvider` | SendPoll, SendPollVote |
| `GroupProvider` | GetGroups, VerifyGroupMembership, AddGroupParticipants, RemoveGroupParticipants |

**Adapters:** `pkg/whatsapp/adapter.go` (Cloud API), `pkg/whatsmeow/adapter.go` (WhatsApp Web)

---

## 76. WhatsMeow Connection Manager

**Backend:** `pkg/whatsmeow/manager.go`  
**Frontend:** Instance status indicators

| Operation | Method |
|-----------|--------|
| Create manager | `NewConnectionManager()` |
| Connect | `Connect()` — QR or pair |
| Disconnect | `Disconnect()` |
| Logout | `Logout()` |
| QR code | QR generation + caching |
| Pair phone | Pairing code flow |
| Health | `healthMonitor()` — periodic check |
| Reconnect | `ReconnectAll()` — batch reconnect |
| Pool | Multi-instance client pool |

---

## 77. WhatsMeow Queue Manager

**Backend:** `pkg/whatsmeow/queue.go`  
**Frontend:** Not applicable

| Operation | Description |
|-----------|-------------|
| Enqueue | Rate-limited send queue |
| Dequeue | Background send processor |
| Rate limit | Per-instance send rate |

---

## 78. Redis Queue System

**Backend:** `internal/queue/redis.go`, `pubsub.go`, `queue.go`  
**Frontend:** Not applicable

| Queue | Purpose |
|-------|---------|
| Campaign queue | Campaign recipient processing |
| Inbound media queue | Media download jobs |
| Pub/Sub | Campaign stats streaming |
| Consumer groups | Redis Streams consumer groups |

---

## 79. Campaign Stats Subscriber

**Backend:** `internal/handlers/app.go` — `StartCampaignStatsSubscriber()`  
**Frontend:** Campaign real-time progress

**Flow:** Worker publishes → Redis Pub/Sub → Subscriber receives → WSHub broadcasts to UI.

---

## 80. Database Migrations

**Backend:** `internal/database/postgres.go`  
**Frontend:** Not applicable

| Migration | Description |
|-----------|-------------|
| AutoMigrate | GORM AutoMigrate for core models |
| Plugin migrations | Per-plugin AutoMigrate |
| Default admin | Creates admin user on first run |

---

## 81. Encryption System

**Backend:** `internal/crypto/crypto.go`  
**Frontend:** Not applicable

| Version | Algorithm | Key Derivation |
|---------|-----------|----------------|
| V2 | AES-256-GCM | HMAC-SHA256 based KDF |
| V3 | AES-256-GCM | Improved KDF with salt |

**Encrypted Fields:** WhatsApp account credentials, webhook secrets, SSO client secrets.

---

## 82. Contact Utilities

**Backend:** `internal/contactutil/`  
**Frontend:** Not applicable

**Purpose:** WhatsApp JID parsing, group JID detection, phone number formatting.

---

## 83. Template Utilities

**Backend:** `internal/templateutil/templateutil.go`  
**Frontend:** Not applicable

**Purpose:** Template placeholder processing — replaces `{{n}}` with actual values.

---

## 84. WebSocket Message Types

**Backend:** `internal/websocket/messages.go`  
**Frontend:** `frontend/src/services/websocket.ts`

| Type | Direction | Payload |
|------|-----------|---------|
| `new_message` | Server→Client | Message object |
| `message_status` | Server→Client | Status update |
| `contact_update` | Server→Client | Contact changes |
| `campaign_stats` | Server→Client | Campaign progress |
| `instance_status` | Server→Client | Connection state |
| `notification` | Server→Client | In-app notification |
| `chatbot_event` | Server→Client | Bot interaction |
| `typing` | Client→Server | Typing indicator |
| `mark_read` | Client→Server | Read receipt |

---

## 85. Frontend Embedded Build

**Backend:** `internal/frontend/` — uses `//go:embed`  
**Frontend:** Full Vue 3 SPA

| Environment | Build Command | Description |
|-------------|--------------|-------------|
| Development | `make dev-frontend` | Vite dev server, HMR |
| Production | `make build-prod` | Builds + embeds in Go binary |

---

## 86. Configuration System

**Backend:** `internal/config/` — koanf-based TOML loader  
**Frontend:** Not applicable

| Section | Purpose |
|---------|---------|
| `[server]` | HTTP listen address, CORS |
| `[database]` | PostgreSQL connection |
| `[redis]` | Redis connection |
| `[jwt]` | JWT secret, expiry |
| `[encryption]` | Encryption key |
| `[license]` | License public key, mode |
| `[admin]` | Admin email, initial setup |

---

## 87. Model Layer

**Backend:** `internal/models/models.go` — 25+ GORM models  
**Frontend:** TypeScript types in `frontend/src/types/`

| Model Group | Models |
|-------------|--------|
| Core | Organization, User, UserOrganization |
| Auth | Role, Permission, APIKey, SSOProvider |
| WhatsApp | WhatsAppAccount, Instance, Contact, Message |
| Campaign | Campaign, CampaignRecipient |
| Content | Template, WhatsAppFlow, MediaAsset |
| Team | Team, TeamMember |
| Analytics | Widget, WidgetFilter |
| Webhook | Webhook, WebhookEvent |
| License | LicenseRecord, LicenseEvent |
| Misc | Tag, CannedResponse, CustomAction, Notification |

---

## 88. Middleware Chain

**Backend:** `internal/middleware/middleware.go`, `ratelimit.go`  
**Frontend:** Not applicable

| Order | Middleware | Purpose |
|-------|-----------|---------|
| 1 | CORS | Cross-origin headers |
| 2 | Recovery | Panic recovery |
| 3 | RateLimit | Per-IP rate limiting |
| 4 | Auth | JWT validation |
| 5 | TenantScope | Org-scoped queries |
| 6 | RBAC | Permission check |

---

## 89. Error Handling Patterns

**Backend:** Response envelope pattern  
**Frontend:** API client error handling

| Response | Format |
|----------|--------|
| Success | `{ "data": {...} }` |
| Error | `{ "error": { "message": "...", "code": "...", "field": "..." } }` |

**Frontend:** `api.ts` interceptors handle 401 (session expired) and 403 (license locked) globally.

---

## 90. Testing Infrastructure

**Backend:** `test/testutil/`, `internal/handlers/testhelpers_test.go`  
**Frontend:** `frontend/src/**/*.test.ts`, e2e tests in `frontend/e2e/`

| Layer | Framework | Pattern |
|-------|-----------|---------|
| Backend unit | Go `testing` | Table-driven tests |
| Backend DB | testutil.SetupTestDB | Ephemeral Postgres |
| Backend Redis | testutil.SetupTestRedis | Ephemeral Redis |
| Frontend unit | Vitest | Component tests |
| Frontend e2e | Playwright | E2E scenarios |

---

## 91. App Configuration Endpoint

**Backend:** `internal/handlers/config_handler.go`  
**Frontend:** App bootstrap

| Operation | Description |
|-----------|-------------|
| Get config | Public endpoint for app configuration (license status, features, etc.) |

---

## 92. User Settings & Chat Background

**Backend:** `internal/handlers/users.go`  
**Frontend:** `frontend/src/views/settings/SettingsView.vue`

| Operation | Backend Method | Frontend |
|-----------|---------------|----------|
| Update settings | `UpdateCurrentUserSettings()` | Settings form |
| Upload background | `UploadCurrentUserChatBackground()` | Background picker |
| Get background | `GetCurrentUserChatBackground()` | Apply background |

**Composable:** `frontend/src/composables/useColorMode.ts` — theme management (light/dark/system).

---

## 93. Availability Management

**Backend:** `internal/handlers/users.go` — `UpdateAvailability()`  
**Frontend:** Agent availability toggle

**Purpose:** Agents toggle online/offline for chat assignment.

---

## 94. Change Password

**Backend:** `internal/handlers/users.go` — `ChangePassword()`  
**Frontend:** Settings — password form

**Flow:** Current password verification → new password validation → hash → store.

---

## 95. Contact Phone Start (WhatsMeow)

**Backend:** `internal/handlers/contacts_chat_start.go`  
**Frontend:** Chat — start new conversation

**Purpose:** Initiates a WhatsApp Web chat by phone number.

---

## 96. Interactive Messages

**Backend:** `internal/handlers/messages.go` — `buildInteractiveData()`  
**Frontend:** Message composer — interactive UI

| Type | Description |
|------|-------------|
| Buttons | Quick reply buttons |
| List | Single-select list |
| Flow | Interactive flow |
| Product | Product message |

---

## 97. Typing Presence

**Backend:** `pkg/whatsmeow/typing_presence.go`, `typing_indicator.go`  
**Frontend:** Chat — typing indicator

| Direction | Description |
|-----------|-------------|
| Inbound | Display contact typing |
| Outbound | Send agent typing |

---

## 98. Agent Role Chat Scoping

**Backend:** `internal/handlers/chat_access_policy.go`  
**Frontend:** Chat list scoping

| Role | Access |
|------|--------|
| Admin | All chats |
| Manager | Team chats + unassigned |
| Agent | Own chats + public |

---

## 99. Organization Outbound Mode

**Backend:** `internal/handlers/organization.go` — outbound mode settings  
**Frontend:** Settings — outbound configuration

| Mode | Description |
|------|-------------|
| Open | All agents can send |
| Restricted | Only assigned agents |
| Closed | Outbound disabled |

---

## 100. Strict Rollout Mode

**Backend:** Campaign policy enforcement  
**Frontend:** Campaign restrictions

| Phase | Restriction |
|-------|-------------|
| Phase 1 | Admin only can send campaigns |
| Phase 2 | Manager + Admin |
| Phase 3 | All with approval |

---

## 101. WhatsApp Poll Messages

**Backend:** `pkg/whatsmeow/poll_vote.go`, `pkg/whatsmeow/adapter.go` — `SendPoll()`  
**Frontend:** Chat — poll message display

| Operation | Backend | Description |
|-----------|---------|-------------|
| Send poll | `Provider.SendPoll()` | Create and send poll |
| Vote | `Provider.SendPollVote()` | Submit vote |
| Results | Poll option tracking | Real-time results |

---

102. [Priority Event Ingestion with Sharded FIFO Queues](#102-priority-event-ingestion-with-sharded-fifo-queues)

---

## 102. Priority Event Ingestion with Sharded FIFO Queues

**Backend:** `pkg/whatsmeow/async_events.go`, `pkg/whatsmeow/metrics.go`, `internal/config/config.go` (WhatsmeowConfig)
**Frontend:** No direct UI — metrics visible via `/metrics` endpoint  
**Feature Flag:** `priority_queues_enabled` (default `false` — fully backward-compatible)

### Purpose
Replaces the single per-instance event channel with **chat-sharded high-priority lanes** and a **bounded drop-newest low-priority lane**, so that flood events (HistorySync, AppState, Presence) cannot starve incoming messages, receipts, or call events.

### Priority Classification

| Class | Events | Queue | Behavior |
|-------|--------|-------|----------|
| `eventClassMessage` | `*events.Message`, `*events.Receipt`, all `*events.Call*` variants | High (msg shards) | Sharded FIFO per chat/call. Bounded retry then `critical_overflow` |
| `eventClassLow` | `*events.HistorySync`, `*events.Contact`, `*events.AppState`, `*events.Presence`, `*events.ChatPresence`, `*events.DeleteForMe`, `*events.DeleteChat`, `*events.OfflineSyncCompleted`, `*events.PushName`, `*events.AppStateSyncComplete`, `*events.AppStateSyncError` | Low (single channel) | Drop-newest when full. When the circuit breaker is open, **droppable** events (`HistorySync`, `AppState*`, `Presence`, `ChatPresence`) are dropped as `circuit_open`; **important** events (`Contact`, `PushName`, `DeleteForMe`, `DeleteChat`, `OfflineSyncCompleted`) still enqueue |
| `eventClassLifecycle` | `*events.Connected`, `*events.Disconnected`, `*events.LoggedOut`, `*events.TemporaryBan`, `*events.PairSuccess`, `*events.QR` | Bypass | Handled synchronously, never queued |

### Shard Routing
`chatKeyForEvent()` → extracts Chat JID (Message/Receipt) or CallID (Call events) → `fnv.New32a()` hash → `shardIndex()` selection. Same chat/call always routes to same shard, preserving per-chat/call FIFO while allowing cross-chat parallelism.

### Enqueue Behavior
- **High priority** (`enqueueHigh`): Immediate non-blocking `select`, then 10ms bounded retry loop (100µs steps). If all fail → `critical_overflow` drop. Never blocks the websocket reader.
- **Low priority** (`enqueueLow`): Single non-blocking `select`. If full → drop-newest. Circuit breaker check first: if open, **droppable** low events (`HistorySync`, `AppState*`, `Presence`, `ChatPresence`) dropped as `circuit_open`; **important** low events (`Contact`, `PushName`, `DeleteForMe`, `DeleteChat`, `OfflineSyncCompleted`) still enqueue so names/avatars/deletions persist under load.

### Circuit Breaker
| Parameter | Default | Purpose |
|-----------|---------|---------|
| `event_circuit_breaker_rate_per_minute` | 60 | Low-priority events/min threshold per instance |
| `event_circuit_breaker_consecutive_windows` | 2 | Consecutive windows above threshold to trip |
| `event_circuit_breaker_cooldown_seconds` | 300 | Seconds to suppress HistorySync after trip |

Rolling window advanced every minute by `circuitBreakerTickerLoop()`. When all windows exceed rate, breaker trips for cooldown duration. High-priority events never affected.

### Config Fields (10 new in `[whatsmeow]`)
| Field | Default | Description |
|-------|---------|-------------|
| `priority_queues_enabled` | `false` | Enable priority lane routing |
| `event_msg_queue_size` | 2048 | Per-shard high-priority channel capacity |
| `event_low_queue_size` | 512 | Low-priority channel capacity |
| `event_msg_shards` | 4 | Number of high-priority shards per instance |
| `event_low_workers` | 2 | Goroutines draining low-priority channel |
| `event_high_enqueue_timeout_ms` | 10 | Max ms to retry high-priority enqueue |
| `event_shutdown_drain_timeout_seconds` | 5 | Max seconds to drain during shutdown |
| `event_circuit_breaker_rate_per_minute` | 60 | Low-priority events/min threshold |
| `event_circuit_breaker_consecutive_windows` | 2 | Consecutive windows to trip breaker |
| `event_circuit_breaker_cooldown_seconds` | 300 | Cooldown after breaker trips |

### Metrics (via `/metrics`)
| Metric | Type | Labels | Source |
|--------|------|--------|--------|
| `whatsmeow_queue_depth` | gauge | `instance`, `type` (msg/low) | `PriorityQueueDepth()` — no channel peeking |
| `whatsmeow_dropped_total` | counter | `instance`, `reason` (overflow) | `whatsmeowMetricsProvider()` |
| `whatsmeow_consumer_lag_seconds` | gauge | `instance`, `type` (msg/low) | `PriorityConsumerLag()` — atomic lag tracking |
| `whatsmeow_circuit_open` | gauge | `instance` | `IsCircuitBreakerOpen()` |

### Key Functions Overview
| File | Function | Purpose |
|------|----------|--------|
| `async_events.go` | `newAsyncEventDispatcher()` | Constructor with defaults (2048/512/4/2/10/60/2/300) |
| `async_events.go` | `Dispatch()` | Routes to priority or legacy path |
| `async_events.go` | `classifyEvent()` | Classifies as high/low/lifecycle |
| `async_events.go` | `chatKeyForEvent()` | Chat JID or CallID extraction |
| `async_events.go` | `enqueueHigh()` / `enqueueLow()` | Non-blocking enqueue |
| `async_events.go` | `priorityQueueFor()` | Lazy queue + worker creation |
| `async_events.go` | `msgWorker()` / `lowWorker()` | Event drain with lag tracking |
| `async_events.go` | `circuitBreakerOpen()` | Rolling window trip check |
| `async_events.go` | `stopInstancePriority()` | Graceful drain with timeout |
| `async_events.go` | `PriorityConsumerLag()` | Atomic lag — no channel peeking |
| `manager.go` | `buildPriorityQueueConfig()` | Config bridge (10 fields → dispatcher) |
| `metrics.go` | `GetPriorityMetricsSnapshot()` | Gathers all priority metrics |
| `main.go` | `whatsmeowMetricsProvider()` | Prometheus rendering |

### Tests (22 tests in `async_events_test.go`)
- `TestPriorityQueueNeverDropsMessageDueToLowFlood` — **Core guarantee**
- `TestCallEventsAreHighPriority`, `TestUnknownCallEventIsHighPriority`
- `TestProductionFloodEventsAreLowPriority`, `TestUnknownEventsDefaultLowPriority`
- `TestLifecycleEventsBypassDispatcher`
- `TestMessageOrderingPerChat`, `TestReceiptOrderingUsesChatKey`, `TestCallOrderingUsesCallID`
- `TestMessageShardParallelismAcrossChats`
- `TestLowPriorityDropsNewestWhenFull`
- `TestCriticalOverflowLogsAreSampledButMetricsCountAll`
- `TestCircuitBreakerTripsOnSustainedFlood`
- `TestStopInstanceDrainsBeforeClose`
- `TestHighPriorityEnqueueDoesNotDeadlockProducer`
- Plus 12 more covering dispatch, drop, panic, shutdown, independence

### Backward Compatibility
When `priority_queues_enabled = false` (default): legacy `asyncEventQueue` path, zero behavior change, no additional goroutines or memory.

---

*End of Feature Workflows — Each feature includes backend handler location + frontend service/store/view mapping.*
