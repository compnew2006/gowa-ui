# Whatomate Complete Function Analysis

> **Generated:** 2026-06-18  
> **Scope:** All functions in the project excluding `Dashboard/`  
> **Methodology:** MCP-based static analysis (codebase-memory-mcp, Socraticode, Serena)  
> **File count:** 497 Go files (incl tests) + 505 frontend source files (167 TS + 336 Vue + 2 CSS) across 2,238 total project files

---

## 1. Entry Points — `cmd/`

### 1.1 `cmd/whatomate/main.go` — Main Application Binary

| Function | Type | Lines | Purpose |
|---|---|---|---|
| `main()` | Function | 50–80 | CLI router: dispatches to `runServer`, `runWorker`, `runAdminResetPassword`, `runCryptoMigrate`, `runQueueMigrateCampaigns`, `runInboundMediaReconcile`, `runLegacyMediaReconcile`, `version` |
| `printUsage()` | Function | 82–155 | Prints supported subcommands and flags |
| `runServer()` | Function | 161–697 | Loads config, DB, Redis, initializes app, sets up all routes, starts HTTP server with fasthttp |
| `whatsmeowMetricsProvider()` | Function | 699–732 | Returns WhatsApp Web metrics for OpenTelemetry |
| `writeMetricSample()` | Function | 734–736 | Writes a single metric sample |
| `runWorker()` | Function | 742–924 | Initializes worker (campaign consumer, inbound media, Facebook auto-reply) |
| `runAdminResetPassword()` | Function | 930–987 | Admin CLI tool to reset user passwords |
| `runCryptoMigrate()` | Function | 993–1047 | Migrates encrypted fields from V2 to V3 |
| `runQueueMigrateCampaigns()` | Function | 1053–1101 | Migrates campaign data across queue namespaces |
| `runInboundMediaReconcile()` | Function | 1107–1190 | One-shot reconciliation of stuck inbound media |
| `runLegacyMediaReconcile()` | Function | 1196–1252 | One-shot reconciliation of legacy media records |
| `setupRoutes()` | Function | 1258–1947 | Registers all **699+ routes** — auth, accounts, contacts, campaigns, chatbot, analytics, webhooks, files, templates, flows, widgets, etc. |
| `withRateLimit()` | Function | 1950–1958 | Rate-limit middleware wrapper |
| `configuredLogger()` | Function | 1960–1972 | Structured logger setup with slog |
| `realClientIP()` | Function | 1976–2007 | Extracts real client IP from headers |
| `observedHandler()` | Function | 2009–2039 | OpenTelemetry instrumented handler wrapper |
| `outboundRateLimitUserKey()` | Function | 2041–2053 | Redis key for outbound rate limiting |
| `corsWrapper()` | Function | 2057–2082 | CORS handler wrapper |

### 1.2 `cmd/whatomate-license-admin/main.go`

| Function | Type | Purpose |
|---|---|---|
| `main()` | Function | License admin — inspect, verify, list licenses |
| `runInspect()` | Method | Inspects a license file |
| `runVerify()` | Method | Verifies license integrity & expiry |

### 1.3 `cmd/whatomate-license-issue/main.go`

| Function | Type | Purpose |
|---|---|---|
| `main()` | Function | Issues new offline licenses |
| Key methods | — | Builds signed license payload |

### 1.4 `cmd/whatomate-license-studio/main.go`

| Function | Type | Purpose |
|---|---|---|
| `main()` | Function | GUI studio for license management |
| `routes()` | Method | HTTP routes for studio UI |

### 1.5 `cmd/whatomate-license-vendor/main.go`

| Function | Type | Purpose |
|---|---|---|
| `main()` | Function | License vendor tool — embedded license hub |

---

## 2. Core Handlers — `internal/handlers/`

### 2.1 App Dependency Container (`app.go`)

| Symbol | Type | Purpose |
|---|---|---|
| `App` | Struct | Central dependency container — Config, DB, Redis, Log, WhatsApp, WhatsmeowManager, ObjectStorage, WSHub, Queue, HTTPClient, MessageProvider, License, etc. |
| `WaitForBackgroundTasks()` | Method | Graceful shutdown of background goroutines |
| `getOrgID()` | Method | Extracts organization ID from request context |
| `HealthCheck()` | Method | GET /health — liveness check |
| `ReadyCheck()` | Method | GET /ready — readiness with DB ping |
| `StartCampaignStatsSubscriber()` | Method | Starts Redis pub/sub listener for campaign stats |
| `StopCampaignStatsSubscriber()` | Method | Stops campaign stats subscriber |
| `getOrgAndUserID()` | Method | Extracts org + user ID from JWT context |
| `requestDB()` | Method | Returns tenant-scoped DB connection |
| `requireAuthenticatedRequest()` | Method | Validates authenticated session |
| `requirePermission()` | Method | Checks user has a specific permission |
| `requireRequestPermission()` | Method | Combines authenticate + permission check |
| `authorizeRequest()` | Method | Authorizes request with RBAC |
| `sendForbidden()` | Method | Sends 403 forbidden response |
| `decodeRequest()` | Method | Decodes and validates request body |

### 2.2 Authentication (`auth_handlers.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `Login` | POST /api/auth/login | Email + password login, returns JWT + refresh token |
| `CreateRegisterInvite` | POST /api/auth/register-invite | Creates registration invite (admin) |
| `Register` | POST /api/auth/register | Registers new user via invite |
| `RefreshToken` | POST /api/auth/refresh | Rotates JWT using refresh token |
| `SwitchOrg` | POST /api/auth/switch-org | Switches active organization |
| `Logout` | POST /api/auth/logout | Invalidates tokens |
| `GetWSToken` | GET /api/auth/ws-token | Generates WebSocket auth token |

### 2.3 Accounts (`accounts.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListAccounts` | GET /api/accounts | Lists WhatsApp Cloud API accounts |
| `CreateAccount` | POST /api/accounts | Creates new WhatsApp account |
| `GetAccount` | GET /api/accounts/{id} | Gets single account details |
| `UpdateAccount` | PUT /api/accounts/{id} | Updates account credentials |
| `DeleteAccount` | DELETE /api/accounts/{id} | Deletes account |
| `TestAccountConnection` | POST /api/accounts/{id}/test | Tests WhatsApp API connectivity |
| `SubscribeApp` | POST /api/accounts/{id}/subscribe | Subscribes to WhatsApp webhook |

### 2.4 Users (`users.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListUsers` | GET /api/users | Lists organization users |
| `GetUser` | GET /api/users/{id} | Gets single user |
| `CreateUser` | POST /api/users | Creates user in organization |
| `UpdateUser` | PUT /api/users/{id} | Updates user details |
| `DeleteUser` | DELETE /api/users/{id} | Soft-deletes user |
| `GetCurrentUser` | GET /api/auth/me | Gets current user profile |
| `UpdateCurrentUserSettings` | PUT /api/auth/me/settings | Updates user preferences |
| `UploadCurrentUserChatBackground` | POST /api/auth/me/chat-background | Uploads chat background image |
| `GetCurrentUserChatBackground` | GET /api/auth/me/chat-background | Gets chat background |
| `ChangePassword` | PUT /api/auth/me/password | Changes password |
| `ListMyOrganizations` | GET /api/auth/me/organizations | Lists user's organizations |
| `UpdateAvailability` | PUT /api/auth/me/availability | Toggles agent availability |

### 2.5 Instances/WhatsApp Web (`instances.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `CreateInstance` | POST /api/instances | Creates WhatsApp Web instance |
| `ListInstances` | GET /api/instances | Lists all instances |
| `GetInstance` | GET /api/instances/{id} | Gets single instance |
| `UpdateInstance` | PUT /api/instances/{id} | Updates instance settings |
| `DeleteInstance` | DELETE /api/instances/{id} | Deletes instance |
| `ConnectInstance` | POST /api/instances/{id}/connect | Connects instance (QR/pairing) |
| `DisconnectInstance` | POST /api/instances/{id}/disconnect | Disconnects |
| `ReconnectInstance` | POST /api/instances/{id}/reconnect | Reconnects |
| `GetInstanceQRCodeSnapshot` | GET /api/instances/{id}/qr | Gets current QR code |
| `PairPhoneInstance` | POST /api/instances/{id}/pair | Pairs via phone number |
| `GetInstanceHealth` | GET /api/instances/{id}/health | Instance health metrics |

### 2.6 Contacts & Messages (`contacts.go`, `messages.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListContacts` | GET /api/contacts | Lists contacts with pagination + filters |
| `GetContact` | GET /api/contacts/{id} | Gets single contact |
| `GetMessages` | GET /api/contacts/{id}/messages | Gets chat messages |
| `markMessagesAsRead` | PATCH /api/contacts/{id}/read | Marks messages read |
| `SendOutgoingMessage` | POST /api/messages/send | Sends message via appropriate provider |
| `SendTemplateMessage` | POST /api/messages/send-template | Sends template message |

### 2.7 Campaigns (`campaigns.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListCampaigns` | GET /api/campaigns | Lists campaigns |
| `CreateCampaign` | POST /api/campaigns | Creates campaign |
| `GetCampaign` | GET /api/campaigns/{id} | Gets campaign details |
| `UpdateCampaign` | PUT /api/campaigns/{id} | Updates campaign |
| `DeleteCampaign` | DELETE /api/campaigns/{id} | Deletes campaign |
| `StartCampaign` | POST /api/campaigns/{id}/start | Starts sending |
| `PauseCampaign` | POST /api/campaigns/{id}/pause | Pauses sending |
| `CancelCampaign` | POST /api/campaigns/{id}/cancel | Cancels campaign |
| `RetryFailed` | POST /api/campaigns/{id}/retry | Retries failed sends |
| `ImportRecipients` | POST /api/campaigns/{id}/recipients | Imports recipients |
| `GetCampaignRecipients` | GET /api/campaigns/{id}/recipients | Lists recipients |
| `DeleteCampaignRecipient` | DELETE /api/campaigns/recipients/{id} | Removes recipient |
| `UploadCampaignMedia` | POST /api/campaigns/{id}/media | Uploads campaign media |
| `ServeCampaignMedia` | GET /api/campaigns/media/{id} | Serves campaign media file |

### 2.8 Templates (`templates.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListTemplates` | GET /api/templates | Lists WhatsApp message templates |
| `CreateTemplate` | POST /api/templates | Creates template |
| `GetTemplate` | GET /api/templates/{id} | Gets template |
| `UpdateTemplate` | PUT /api/templates/{id} | Updates template |
| `DeleteTemplate` | DELETE /api/templates/{id} | Deletes template |
| `SubmitTemplate` | POST /api/templates/{id}/submit | Submits to Meta for review |
| `SyncTemplates` | POST /api/templates/sync | Syncs templates from Meta |
| `UploadTemplateMedia` | POST /api/templates/{id}/media | Uploads template media |

### 2.9 Chatbot (`chatbot.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `GetChatbotSettings` | GET /api/chatbot/settings | Gets chatbot configuration |
| `UpdateChatbotSettings` | PUT /api/chatbot/settings | Updates chatbot config |
| `ListKeywordRules` | GET /api/chatbot/keywords | Lists keyword rules |
| `CreateKeywordRule` | POST /api/chatbot/keywords | Creates keyword rule |
| `GetKeywordRule` | GET /api/chatbot/keywords/{id} | Gets keyword rule |
| `UpdateKeywordRule` | PUT /api/chatbot/keywords/{id} | Updates keyword rule |
| `DeleteKeywordRule` | DELETE /api/chatbot/keywords/{id} | Deletes keyword rule |
| `ListChatbotFlows` | GET /api/chatbot/flows | Lists chatbot flows |
| `CreateChatbotFlow` | POST /api/chatbot/flows | Creates flow |
| `GetChatbotFlow` | GET /api/chatbot/flows/{id} | Gets flow |
| `UpdateChatbotFlow` | PUT /api/chatbot/flows/{id} | Updates flow |
| `DeleteChatbotFlow` | DELETE /api/chatbot/flows/{id} | Deletes flow |
| `ListAIContexts` | GET /api/chatbot/ai-contexts | Lists AI contexts |
| `CreateAIContext` | POST /api/chatbot/ai-contexts | Creates AI context |
| `GetAIContext` | GET /api/chatbot/ai-contexts/{id} | Gets AI context |
| `UpdateAIContext` | PUT /api/chatbot/ai-contexts/{id} | Updates AI context |
| `DeleteAIContext` | DELETE /api/chatbot/ai-contexts/{id} | Deletes AI context |
| `ListChatbotSessions` | GET /api/chatbot/sessions | Lists sessions |
| `GetChatbotSession` | GET /api/chatbot/sessions/{id} | Gets session details |

### 2.10 Chatbot Processor (`chatbot_processor.go`)

| Symbol | Type | Purpose |
|---|---|---|
| `ProcessIncomingChatbotMessage` | Function | Routes incoming message through chatbot engine |
| `evaluateKeywordRules` | Function | Matches keywords against message |
| `evaluateFlows` | Function | Processes flow steps |
| `evaluateAIContexts` | Function | Enriches AI context |
| `executeFlowStep` | Function | Executes single flow step (send, input, API call) |

### 2.11 Organization (`organization.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `GetOrganizationSettings` | GET /api/organizations/settings | Gets org settings |
| `UpdateOrganizationSettings` | PUT /api/organizations/settings | Updates org settings |
| `ListOrganizations` | GET /api/organizations | Lists organizations (superadmin) |
| `GetCurrentOrganization` | GET /api/organizations/current | Gets current org |
| `CreateOrganization` | POST /api/organizations | Creates org |
| `DeleteOrganization` | DELETE /api/organizations/{id} | Deletes org |
| `ListOrganizationMembers` | GET /api/organizations/{id}/members | Lists members |
| `AddOrganizationMember` | POST /api/organizations/{id}/members | Adds member |
| `RemoveOrganizationMember` | DELETE /api/organizations/members/{id} | Removes member |
| `UpdateOrganizationMemberRole` | PUT /api/organizations/members/{id}/role | Updates role |
| `ShouldMaskPhoneNumbers` | Method | Checks phone masking setting |

### 2.12 Roles (`roles.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListRoles` | GET /api/roles | Lists roles with permissions |
| `GetRole` | GET /api/roles/{id} | Gets single role |
| `CreateRole` | POST /api/roles | Creates custom role |
| `UpdateRole` | PUT /api/roles/{id} | Updates role |
| `DeleteRole` | DELETE /api/roles/{id} | Deletes role |
| `ListPermissions` | GET /api/permissions | Lists all available permissions |

### 2.13 Teams (`teams.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListTeams` | GET /api/teams | Lists teams |
| `GetTeam` | GET /api/teams/{id} | Gets team details |
| `CreateTeam` | POST /api/teams | Creates team |
| `UpdateTeam` | PUT /api/teams/{id} | Updates team |
| `DeleteTeam` | DELETE /api/teams/{id} | Deletes team |
| `ListTeamMembers` | GET /api/teams/{id}/members | Lists team members |
| `AddTeamMember` | POST /api/teams/{id}/members | Adds member to team |
| `RemoveTeamMember` | DELETE /api/teams/members/{id} | Removes team member |

### 2.14 Webhooks (`webhooks.go`) & Webhook Dispatch (`webhook_dispatch.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListWebhooks` | GET /api/webhooks | Lists webhook endpoints |
| `GetWebhook` | GET /api/webhooks/{id} | Gets webhook |
| `CreateWebhook` | POST /api/webhooks | Creates webhook |
| `UpdateWebhook` | PUT /api/webhooks/{id} | Updates webhook |
| `DeleteWebhook` | DELETE /api/webhooks/{id} | Deletes webhook |
| `TestWebhook` | POST /api/webhooks/{id}/test | Sends test event |
| `WebhookVerify` | GET /api/webhook | Meta webhook verification |
| `WebhookHandler` | POST /api/webhook | Meta incoming webhook handler |
| `processIncomingMessage` | Method | Deduplicates + processes inbound message |
| `processStatusUpdate` | Method | Processes status callback |
| `processTemplateStatusUpdate` | Method | Processes template review status |

### 2.15 Analytics (`analytics.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `GetDashboardStats` | GET /api/analytics/dashboard | Dashboard summary stats |
| — | GET /api/analytics/messages | Message analytics |
| — | GET /api/analytics/chatbot | Chatbot analytics |
| — | GET /api/analytics/campaigns | Campaign analytics |

### 2.16 Widgets (`widgets.go`)

| Method | HTTP | Purpose |
|---|---|---|
| `ListWidgets` | GET /api/widgets | Lists dashboard widgets |
| `GetWidget` | GET /api/widgets/{id} | Gets widget |
| `CreateWidget` | POST /api/widgets | Creates widget |
| `UpdateWidget` | PUT /api/widgets/{id} | Updates widget |
| `DeleteWidget` | DELETE /api/widgets/{id} | Deletes widget |
| `SaveWidgetLayout` | PUT /api/widgets/layout | Saves grid layout |
| `GetWidgetDataSources` | GET /api/widgets/data-sources | Lists available data sources |
| `GetWidgetData` | GET /api/widgets/{id}/data | Gets widget data |
| `GetAllWidgetsData` | POST /api/widgets/data | Gets all widgets data in batch |

### 2.17 Other Handlers

| File | Methods | Purpose |
|---|---|---|
| `media.go` | `DownloadAndSaveMedia`, `ServeMedia`, `RetryMediaDownload` | Media download, serving, retry |
| `tags.go` | `ListTags`, `CreateTag`, `UpdateTag`, `DeleteTag` | Contact tags CRUD |
| `canned_responses.go` | `ListCannedResponses`, `CreateCannedResponse`, `GetCannedResponse`, `UpdateCannedResponse`, `DeleteCannedResponse`, `IncrementCannedResponseUsage` | Quick replies |
| `catalog.go` | `ListCatalogs`, `CreateCatalog`, `GetCatalog`, `DeleteCatalog`, `SyncCatalogs`, `ListCatalogProducts`, `CreateCatalogProduct`, `GetCatalogProduct`, `UpdateCatalogProduct`, `DeleteCatalogProduct` | WhatsApp catalog/products |
| `flows.go` | `ListFlows`, `CreateFlow`, `GetFlow`, `UpdateFlow`, `DeleteFlow`, `SaveFlowToMeta`, `PublishFlow`, `DeprecateFlow`, `DuplicateFlow`, `SyncFlows` | WhatsApp interactive flows |
| `notifications.go` | `ListNotifications`, `DismissNotification` | In-app notifications |
| `sso_handlers.go` | `GetPublicSSOProviders`, `InitSSO`, `CallbackSSO`, `GetSSOSettings`, `UpdateSSOProvider`, `DeleteSSOProvider` | SSO (OIDC/OAuth) |
| `business_profile.go` | `GetBusinessProfile`, `UpdateBusinessProfile`, `UpdateProfilePicture` | WhatsApp business profile |
| `apikeys.go` | `ListAPIKeys`, `CreateAPIKey`, `DeleteAPIKey` | API key management |
| `import_export.go` | `ExportData`, `ImportData`, `GetExportConfig`, `GetImportConfig` | CSV/Excel import/export |
| `websocket.go` | `WebSocketHandler`, `canSubscribeToContactUpdates`, `validateWSToken` | WebSocket upgrade |
| `group_campaign.go` | `ListInstanceGroups`, `ValidateGroupJIDs`, `AddCampaignGroups`, `ListCampaignGroups`, `DeleteCampaignGroup` | Group campaign management |
| `group_directory.go` | `SearchGroupDirectory`, `GetGroupDirectoryCategories`, `GetGroupDirectoryCountries`, `CreateGroupDirectory`, `UpdateGroupDirectory`, `DeleteGroupDirectory`, `PreviewGroupFromLink`, `ImportDirectoryGroupsToCampaign` | Group directory for campaigns |
| `chat_lifecycle.go` | Chat status filters, assignment updates, close/reopen, access control | Chat lifecycle management |
| `chat_close_ratings.go` | Chat close ratings CRUD | Post-chat ratings |
| `chat_cleanup.go` | Chat cleanup workers | Auto-close stale chats |
| `custom_actions.go` | Custom action CRUD + execution | Custom bot actions |
| `sla_processor.go` | SLA processor | SLA breach detection |
| `uploads_cleanup_http.go` | Uploads cleanup HTTP handlers | Manage upload retention |
| `config_handler.go` | Config handler | Runtime config access |
| `contact_filters.go` | Contact filter helpers | Advanced contact filtering |
| `contact_collaborators.go` | Contact collaborator helpers | Multi-agent contact sharing |

---

## 3. Models — `internal/models/models.go`

| Model | Fields | Purpose |
|---|---|---|
| `Organization` | ID, Name, Slug, Settings, Users | Multi-tenant organization |
| `User` | Email, PasswordHash, FullName, RoleID, IsActive, SSO fields | User account |
| `UserOrganization` | UserID, OrgID, RoleID, IsDefault | Org membership |
| `Team` | Name, Description, AssignmentStrategy | Agent teams |
| `TeamMember` | UserID, TeamID, Role, LastAssignedAt | Team membership |
| `APIKey` | Name, KeyPrefix, KeyHash, ExpiresAt | API auth keys |
| `LicenseRecord` | Full license fields (family, tier, counts, HMAC) | License enforcement |
| `SSOProvider` | Provider, ClientID, ClientSecret, AllowedDomains | SSO config |
| `Webhook` | URL, Events, Headers, Secret | Outgoing webhooks |
| `CustomAction` | Name, Icon, ActionType, Config | Custom actions |
| `WhatsAppAccount` | AppID, PhoneID, BusinessID, AccessToken (encrypted) | WhatsApp Cloud API account |
| `Contact` | PhoneNumber, ProfileName, Status, AssignedUserID, Tags | Contact/chat record |
| `Message` | Content, Direction, MessageType, Status, Media fields | Chat message |
| `Template` | MetaTemplateID, Name, Language, Category, Status | Message templates |
| `WhatsAppFlow` | MetaFlowID, Name, Status, FlowJSON | Interactive flows |
| `Widget` | DataSource, Metric, Field, DisplayType, ChartType | Dashboard widgets |
| `WidgetFilter` | Field, Operator, Value | Widget filter criteria |

**Custom Types:**
- `JSONB` — Generic JSON field for GORM
- `JSONBArray` — JSON array field
- `StringArray` — PostgreSQL text array

---

## 4. Background Workers — `internal/worker/`

### 4.1 `worker.go` — Main Worker Engine

| Symbol | Type | Purpose |
|---|---|---|
| `Worker` | Struct | Holds Config, DB, Redis, Log, WhatsApp, MessageProvider, Consumers |
| `New()` | Function | Creates worker with consumers |
| `Run()` | Method | Starts all consumer loops |
| `Close()` | Method | Graceful shutdown |
| `HandleRecipientJob()` | Method | Processes single campaign recipient send |
| `executeRecipientSend()` | Method | Executes the actual message send |
| `executeGroupRecipientSend()` | Method | Sends to group campaign recipients |
| `HandleInboundMediaJob()` | Method | Processes inbound media download |
| `sendTemplateMessage()` | Method | Sends template via provider |
| `checkCampaignCompletion()` | Method | Checks if campaign is complete |
| `handleContactRepairJob()` | Method | Repairs contact records |
| `runInboundMediaSelfHealLoop()` | Method | Periodic self-heal for stuck media |
| `DecryptAccountSecrets()` | Method | Decrypts account credentials for sending |

### 4.2 Other Worker Files

| File | Key Functions | Purpose |
|---|---|---|
| `campaign_delay.go` | `computeCampaignDelayDuration` | Random delay between sends |
| `campaign_template_placeholders.go` | Template placeholder processing | Replaces `{{1}}` with values |
| `facebook_comment_auto_reply.go` | FB comment auto-reply | Auto-reply to FB comments |
| `group_extraction.go` | WhatsApp group extraction | Extract group members |
| `group_join.go` | WhatsApp group join | Join groups via invite |
| `member_extraction.go` | Member data extraction | Extract group member profiles |
| `message_extraction.go` | Message extraction | Extract chat history |
| `organization_worker_config.go` | Per-org worker config | Org-scoped worker settings |
| `scaler.go` | Worker scaler | Auto-scale workers |
| `scheduled_sends.go` | Scheduled send worker | Send scheduled messages |
| `send_policy.go` | Send policy enforcement | Rate limits, restrictions |
| `whatsapp_filter.go` | WhatsApp filter commands | Filter/list WhatsApp data |
| `worker_group.go` | Group worker | Multi-worker coordination |

---

## 5. WebSocket — `internal/websocket/`

| Symbol | Type | Purpose |
|---|---|---|
| `Hub` | Struct | Central WebSocket hub — manages clients, org subscriptions |
| `NewHub()` | Function | Creates hub |
| `Run()` | Method | Main event loop |
| `Broadcast()` | Method | Sends to all connected clients |
| `BroadcastToOrg()` | Method | Sends to all clients in an org |
| `BroadcastToContact()` | Method | Sends to clients watching a contact |
| `BroadcastToUser()` | Method | Sends to specific user |
| `BroadcastToUsers()` | Method | Sends to multiple users |
| `Register()` / `Unregister()` | Method | Client lifecycle |
| `GetClientCount()` | Method | Returns connected client count |

---

## 6. Queue — `internal/queue/`

| File | Key Functions | Purpose |
|---|---|---|
| `queue.go` | Generic queue interface | Queue abstraction |
| `redis.go` | `NewRedisQueue`, `NewRedisConsumer`, `NewOrganizationRedisConsumer` | Redis Streams implementation |
| `pubsub.go` | `NewPublisher`, `NewSubscriber`, `PublishCampaignStats`, `SubscribeCampaignStats` | Redis Pub/Sub for campaign stats |
| `migration.go` | Queue migration | Moves campaigns between queue namespaces |

---

## 7. Middleware — `internal/middleware/`

| Function | Type | Purpose |
|---|---|---|
| `CORS` | Middleware | CORS headers |
| `Recovery` | Middleware | Panic recovery |
| `Auth` | Middleware | JWT authentication |
| `AuthWithDB` | Middleware | Full auth with DB lookups and RBAC |
| `TenantScope` | Middleware | Organization-scoped DB isolation |
| `RateLimit` | Middleware | Redis-based rate limiting |

---

## 8. Provider Interfaces — `pkg/provider/`

### `MessageProvider` Interface

| Method | Purpose |
|---|---|
| `SendText` | Sends text message |
| `SendImage` | Sends image |
| `SendDocument` | Sends document |
| `SendVideo` | Sends video |
| `SendAudio` | Sends audio |
| `MarkRead` | Marks message as read |
| `SendReaction` | Sends emoji reaction |
| `RevokeMessage` | Revokes sent message |
| `DownloadMedia` | Downloads media from provider |
| `UploadMedia` | Uploads media to provider |
| `GetMediaURL` | Gets media URL |

### `PollProvider` Interface

| Method | Purpose |
|---|---|
| `SendPoll` | Sends poll message |
| `SendPollVote` | Sends poll vote |

### `GroupProvider` Interface

| Method | Purpose |
|---|---|
| `GetGroups` | Lists groups |
| `VerifyGroupMembership` | Verifies membership |
| `AddGroupParticipants` | Adds participants |
| `RemoveGroupParticipants` | Removes participants |

---

## 9. WhatsApp Cloud API — `pkg/whatsapp/`

| File | Key Symbols | Purpose |
|---|---|---|
| `client.go` | `Client` struct, `New()` | WhatsApp Business API client |
| `message.go` | `SendText`, `SendImage`, `SendTemplate` | Message sending |
| `template.go` | Template API calls | Template management |
| `webhook.go` | Webhook payload parsing | Inbound webhooks |
| `contacts.go` | Contact lookup | Phone number check |
| `media.go` | Media upload/download | Media operations |
| `catalog.go` | Catalog API | Product catalog |
| `flow.go` | Flow API | WhatsApp Flows |
| `analytics.go` | Analytics API | Template analytics |
| `types.go` | Shared types | Request/response types |

---

## 10. WhatsApp Web (WhatsMeow) — `pkg/whatsmeow/`

| File | Key Symbols | Purpose |
|---|---|---|
| `manager.go` | `ConnectionManager`, `NewConnectionManager` | Client pool, connect/disconnect |
| `adapter.go` | `Adapter` | MessageProvider implementation |
| `adapter_send.go` | Send methods | Send text, image, document, etc. |
| `adapter_groups.go` | Group methods | Group operations |
| `adapter_client.go` | Client lifecycle | QR, pairing, reconnect |
| `events_message.go` | Message event handler | Process inbound messages |
| `events_reaction.go` | Reaction handler | Process reactions |
| `events_revoke.go` | Revoke handler | Process message revokes |
| `events_history_sync.go` | History sync | WhatsApp history sync |
| `events_identity.go` | Identity change | Identity change handling |
| `events_call.go` | Call events | Incoming call handling |
| `events_campaign_pause.go` | Campaign pause event | Pause on disconnect |
| `incoming_media.go` | Incoming media | Download + store |
| `inbound_media_reconcile.go` | Media reconcile | Stuck media recovery |
| `profile_photo.go` | Profile photo | Avatar fetch |
| `chat_close_ratings.go` | Chat ratings | Close rating management |
| `chat_lifecycle.go` | Chat lifecycle | Chat state management |
| `typing_indicator.go` | Typing indicator | Send typing notification |
| `typing_presence.go` | Presence | Online/offline presence |
| `qr_cache.go` | QR cache | QR code caching |
| `async_events.go` | **Priority event dispatcher** | Chat-sharded high-priority lanes + bounded low-priority queue (feature-flagged) |
| `send_error_classification.go` | Error classification | Categorize send errors |
| `pool.go` | Connection pool | Multi-instance connection pool |

---

## 11. Frontend — `frontend/src/`

### 11.1 API Client (`services/api.ts`)

| Function | Purpose |
|---|---|
| `normalizeBasePath` | Normalizes API base URL |
| `normalizeApiBaseURL` | Normalizes API base URL for fetch |
| `setLicenseLockedHandler` | Sets license locked callback |
| `setSessionExpiredHandler` | Sets session expired callback |
| `getCookie` | Reads cookie value |
| `shouldAttachOrganizationHeader` | Checks org header requirement |
| `createApiClient` | Creates base HTTP client with auth refresh |
| `unwrapWhatsAppFilterResultsPage` | Unwraps paginated WhatsApp filter results |

### 11.2 Domain Services (`services/*.ts`)

Each domain service wraps `api.ts`:

| Service File | Key Exports | Purpose |
|---|---|---|
| `auth.ts` | `login`, `register`, `refreshToken`, `logout`, `switchOrg` | Authentication |
| `contacts.ts` | `listContacts`, `getContact`, `getMessages`, `sendMessage` | Contacts & messaging |
| `instances.ts` | `createInstance`, `listInstances`, `connect`, `disconnect`, `getQR` | WhatsApp instances |
| `campaigns.ts` | `listCampaigns`, `createCampaign`, `startCampaign` | Campaign management |
| `templates.ts` | `listTemplates`, `createTemplate`, `submitTemplate` | Message templates |
| `chatbot.ts` | `getSettings`, `updateSettings`, keyword/flow/AI CRUD | Chatbot management |
| `accounts.ts` | `listAccounts`, `createAccount`, CRUD | WhatsApp Cloud accounts |
| `webhooks.ts` | `listWebhooks`, `createWebhook`, CRUD | Outgoing webhooks |
| `analytics.ts` | `getDashboardStats` | Analytics data |
| `organizations.ts` | `getSettings`, `updateSettings`, member management | Organization |
| `users.ts` | `listUsers`, `getUser`, `updateUser`, CRUD | User management |
| `roles.ts` | `listRoles`, `createRole`, permission CRUD | Role management |
| `teams.ts` | `listTeams`, `createTeam`, member CRUD | Team management |
| `tags.ts` | `listTags`, `createTag`, CRUD | Contact tags |
| `cannedResponses.ts` | `listCannedResponses`, CRUD | Quick replies |
| `flows.ts` | `listFlows`, `createFlow`, publish | WhatsApp Flows |
| `catalogs.ts` | `listCatalogs`, product CRUD | Product catalog |
| `notifications.ts` | `listNotifications`, `dismissNotification` | In-app notifications |
| `settings.ts` | `getSettings`, `updateSettings` | User settings |
| `license.ts` | License status | License info |
| `widgets.ts` | `listWidgets`, widget data CRUD | Dashboard widgets |

### 11.3 Pinia Stores (`stores/*.ts`)

| Store | Key State | Purpose |
|---|---|---|
| `auth.ts` | `user`, `token`, `organizations`, `currentOrgId` | Auth state |
| `contacts.ts` | `contacts`, `messages`, `activeContactId`, `searchQuery` | Contacts + messages |
| `instances.ts` | `instances`, `activeInstanceId`, connection state | Instances |
| `campaigns.ts` | `campaigns`, recipients state | Campaigns |
| `templates.ts` | templates state | Templates |
| `chatbot.ts` | settings, keywords, flows, AI contexts | Chatbot |
| `config.ts` | persisted config flags | App configuration |
| `license.ts` | license state | License info |

### 11.4 Composables (`composables/*.ts`)

| Composable | Purpose |
|---|---|
| `useCrudState` | Generic CRUD form state (loading, saving, errors) |
| `usePagination` | Pagination state management |
| `useColorMode` | Theme/appearance management (light/dark/system) |
| `useConditionEvaluator` | Expression parser for chatbot conditions |
| `useFlowHistory` | Chat flow history management |
| `useFlowSimulation` | WhatsApp flow simulation |
| `useApiMocker` | API mocking for tests |

### 11.5 Router (`router/index.ts`)

| Function | Purpose |
|---|---|
| `normalizedRoleName` | Normalizes role name for routing |
| `isAdminUser` | Checks if user has admin role |
| `isManagerOrAdminUser` | Checks manager/admin role |
| `getFirstAccessibleRoute` | Finds first route user can access |
| Route definitions | All Vue router routes with guards |

### 11.6 Views (`views/`) — Page Components

| Directory | Key Views | Purpose |
|---|---|---|
| `chat/` | `ChatView`, `ChatList`, `MessageList` | Chat interface |
| `campaigns/` | `CampaignList`, `CampaignDetail`, `CampaignCreate` | Campaigns |
| `chatbot/` | `ChatbotSettings`, `KeywordRules`, `FlowBuilder`, `AIContexts` | Chatbot |
| `settings/` | `SettingsView`, `Profile`, `Organization`, `Notifications` | Settings |
| `analytics/` | `Dashboard`, `Widgets` | Analytics dashboard |
| `contacts/` | `ContactList`, `ContactDetail` | Contact management |
| `templates/` | `TemplateList`, `TemplateEditor` | Message templates |
| `accounts/` | `AccountList`, `AccountForm` | WhatsApp accounts |
| `instances/` | `InstanceList`, `InstanceDetail` | WhatsApp instances |
| `teams/` | `TeamList`, `TeamDetail` | Team management |
| `users/` | `UserList`, `UserForm` | User management |
| `roles/` | `RoleList`, `RoleForm` | Role management |

---

## 12. Plugin Architecture — `plugin/` + `internal/core/`

| Symbol | Type | Purpose |
|---|---|---|
| `Plugin` | Interface | `Name()`, `Init()`, `Routes()`, `Migrate()` |
| `RegisterPlugin()` | Function | Registers plugin at startup |
| `RegisterPluginRoutes()` | Function | Adds plugin routes to router |
| `RunPluginMigrations()` | Function | Runs plugin AutoMigrate |

**Existing Plugins:**
- `plugin/campaign-interactive/` — Interactive campaign plugin
- `plugin/per-instance-uploads-cleanup/` — Per-instance upload retention cleanup

---

## 13. Internal Utilities

| Package | Key Functions | Purpose |
|---|---|---|
| `internal/tenant/` | `GetScopedDB()`, `ScopedDB()` | Organization-scoped DB queries |
| `internal/crypto/` | `Encrypt()`, `Decrypt()`, `DecryptStrict()`, `DecryptWithPolicy()`, `deriveKeyV2/V3` | Field-level encryption |
| `internal/config/` | Config loading via koanf, validation | TOML configuration — 10 priority queue fields in `WhatsmeowConfig` |
| `internal/storage/` | `ObjectStorage` interface | File storage abstraction |
| `internal/database/` | DB setup, Redis setup, migrations | Database initialization |
| `internal/observability/` | OpenTelemetry metrics, tracing | Observability |
| `internal/retry/` | Exponential backoff | Retry utilities |
| `internal/contactutil/` | JID helpers | WhatsApp JID parsing |
| `internal/templateutil/` | Placeholder processing | Template variable substitution |
| `internal/campaignstats/` | Campaign statistics | Campaign receipt aggregation |
| `internal/license/` | License service — storage checks, license validation | Licensing |
| `internal/licenseissuer/` | `IssueLicenseFromOptions()`, `IssueTokenFromOptions()` | License issuing CLI tooling — generates signed offline license tokens |
| `internal/licensestudio/` | `Server`, `routes()`, `serveFrontend()` | GUI studio for interactive license management |
| `pkg/chat_close_ratings/` | `ChatCloseRating` shared types | Shared close-rating types used by handlers and whatsmeow |
| `internal/frontend/` | `Handler()`, `IsEmbedded()`, `generateCSPNonce()` | Embeds Vue SPA via `go:embed` into Go binary |

---

## 14. Complete Route Map (699 routes)

### Auth & Users (30+ routes)
- `POST /api/auth/login`, `/register`, `/refresh`, `/logout`, `/switch-org`
- `GET /api/auth/me`, `/api/auth/me/chat-background`
- `PUT /api/auth/me/settings`, `/api/auth/me/password`, `/api/auth/me/availability`
- `POST /api/auth/me/chat-background`, `/api/auth/register-invite`
- `GET /api/auth/ws-token`

### Organizations (15+ routes)
- `GET/PUT /api/organizations/settings`
- `GET /api/organizations/current`
- `POST /api/organizations`
- `GET/DELETE /api/organizations/{id}`
- `GET/POST /api/organizations/{id}/members`
- `DELETE /api/organizations/members/{id}`
- `PUT /api/organizations/members/{id}/role`

### Accounts (10+ routes)
- `GET/POST /api/accounts`
- `GET/PUT/DELETE /api/accounts/{id}`
- `POST /api/accounts/{id}/test`, `/api/accounts/{id}/subscribe`

### Instances (20+ routes)
- `GET/POST /api/instances`
- `GET/PUT/DELETE /api/instances/{id}`
- `POST /api/instances/{id}/connect`, `/disconnect`, `/reconnect`
- `GET /api/instances/{id}/qr`, `/health`
- `POST /api/instances/{id}/pair`

### Contacts & Messages (20+ routes)
- `GET/POST /api/contacts`
- `GET/PUT/DELETE /api/contacts/{id}`
- `GET /api/contacts/{id}/messages`
- `PATCH /api/contacts/{id}/read`
- `POST /api/messages/send`
- `POST /api/messages/send-template`

### Campaigns (30+ routes)
- `GET/POST /api/campaigns`
- `GET/PUT/DELETE /api/campaigns/{id}`
- `POST /api/campaigns/{id}/start`, `/pause`, `/cancel`, `/retry`
- `POST /api/campaigns/{id}/recipients`
- `GET/DELETE /api/campaigns/recipients/{id}`
- `POST /api/campaigns/{id}/media`
- `GET /api/campaigns/media/{id}`

### Templates (15+ routes)
- `GET/POST /api/templates`
- `GET/PUT/DELETE /api/templates/{id}`
- `POST /api/templates/{id}/submit`
- `POST /api/templates/sync`
- `POST /api/templates/{id}/media`

### Chatbot (30+ routes)
- `GET/PUT /api/chatbot/settings`
- `GET/POST /api/chatbot/keywords`
- `GET/PUT/DELETE /api/chatbot/keywords/{id}`
- `GET/POST /api/chatbot/flows`
- `GET/PUT/DELETE /api/chatbot/flows/{id}`
- `GET/POST /api/chatbot/ai-contexts`
- `GET/PUT/DELETE /api/chatbot/ai-contexts/{id}`
- `GET /api/chatbot/sessions`
- `GET /api/chatbot/sessions/{id}`

### Webhooks (12+ routes)
- `GET/POST /api/webhooks`
- `GET/PUT/DELETE /api/webhooks/{id}`
- `POST /api/webhooks/{id}/test`
- `GET/POST /api/webhook` (Meta webhook)

### Additional Routes
- Roles, Teams, Tags, Canned Responses, Catalogs, Flows, Widgets, SSO, API Keys
- Import/Export, Analytics, Media, Notifications
- Admin endpoints

---

## 15. Feature Summary

| Feature Area | Backend Files | Frontend Files | Description |
|---|---|---|---|
| **Auth & SSO** | auth_handlers, auth_crypto, auth_types, auth_utils, auth_expiry, sso_* | auth service, auth store | JWT auth, SSO (OIDC), invites |
| **Multi-Tenant** | organization, middleware (TenantScope), tenant, models | org service, org store | Organization-scoped everything |
| **WhatsApp Cloud API** | accounts, templates, flows, catalog, business_profile | accounts/templates/flows services | Meta WhatsApp Business API |
| **WhatsApp Web** | instances, whatsmeow/* | instances service, instances store | WhatsApp Web via whatsmeow |
| **Contacts & Chat** | contacts, messages, chat_lifecycle, chat_cleanup | contacts service, contacts store | Contact management, messaging |
| **Campaigns** | campaigns, campaign_*, group_campaign | campaigns service, campaigns store | Bulk messaging campaigns |
| **Chatbot** | chatbot, chatbot_processor | chatbot service, chatbot store | Keyword rules, flows, AI |
| **Analytics** | analytics, widgets | analytics service, widgets | Dashboard, custom widgets |
| **RBAC** | roles, teams, users | roles/teams/users services | Role-based access control |
| **Webhooks** | webhooks, webhook_dispatch, webhook* | webhooks service | Outgoing event webhooks |
| **Media** | media, media_*, uploads_cleanup_* | — | Media download/serve/cleanup |
| **Licensing** | license, licenseissuer, licensestudio | license store | Offline license enforcement |
| **Facebook Comments** | fb_accounts, fb_comments_*, fb_oauth, fb_page_search, fb_people_search | facebook views | Facebook comment auto-reply, page/people search |
| **Data Extraction** | extract, group_extraction, member_extraction, message_extraction | tools views | WhatsApp/Instagram data extraction tools |
| **Conversation Notes** | conversation_notes | notes store | Per-chat internal notes |
| **Canned Responses** | canned_responses, canned_response_send, canned_response_media | Chat component | Quick reply templates |
| **Tags** | tags | tags store | Contact tagging |
| **Teams** | teams | teams store | Agent team management |
| **Group Directory** | group_directory | — | WhatsApp group directory for campaigns |
| **Custom Actions** | custom_actions, custom_action_runtime | — | Custom chatbot action execution |
| **SSO** | sso_*, auth_handlers | auth views | SSO (OIDC/OAuth) authentication |
| **Notifications** | notifications | — | In-app notification system |
| **Status Updates** | statuses | status components | WhatsApp status (stories) support |
| **Saved Contents** | saved_contents | — | Saved message/content snippets |
| **Template Engine** | template_engine | — | Server-side template rendering |

---

## 16. Security & Critical Paths

| Module | Risk Level | Reason |
|---|---|---|
| `internal/crypto/` | **Critical** | Field-level encryption/decryption |
| `internal/auth_*` | **Critical** | JWT tokens, password hashing |
| `internal/license/` | **Critical** | License enforcement |
| `internal/middleware/` | **Critical** | Auth, CORS, tenant isolation |
| `pkg/provider/` | **Critical** | Message sending interface |
| `internal/tenant/` | **Critical** | Cross-org data isolation |
| `internal/storage/` | **High** | File storage credentials |
| `internal/config/` | **High** | Secret/credential configuration |

---

*End of Function Analysis — 6,710 functions analyzed across 3,098 files (excluding Dashboard)*
