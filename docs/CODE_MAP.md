# Whatomate — Code Map

> Auto-generated comprehensive index of the codebase architecture.
> Last updated: 2026-06-04

---

## Table of Contents

1. [Backend Entrypoints (CLI Commands)](#1-backend-entrypoints-cli-commands)
2. [API Handlers & Routes](#2-api-handlers--routes)
3. [Frontend Routes](#3-frontend-routes)
4. [Database Models](#4-database-models)
5. [Provider / Service Layer](#5-provider--service-layer)
6. [Queue & Worker Layer](#6-queue--worker-layer)
7. [WebSocket Layer](#7-websocket-layer)
8. [Middleware](#8-middleware)
9. [Shared Types, Interfaces & Enums](#9-shared-types-interfaces--enums)
10. [Configuration](#10-configuration)
11. [Supporting Modules](#11-supporting-modules)

---

## 1. Backend Entrypoints (CLI Commands)

**Binary**: `whatomate` | **Module**: `github.com/compnew2006/whatomate`
**Entry**: `cmd/whatomate/main.go:44`

| Command | Function | Description |
|---|---|---|
| `server` | `runServer` | Full API server (fastglue+fasthttp) with workers, WebSocket, providers |
| `worker` | `runWorker` | Background workers only (campaign scaler + inbound media) |
| `crypto-migrate` | `runCryptoMigrate` | Upgrade encrypted secrets to new format |
| `queue-migrate-campaigns` | `runQueueMigrateCampaigns` | Redistribute legacy campaign jobs to per-tenant streams |
| `admin-reset-password` | `runAdminResetPassword` | Reset admin password (`-email`, `-password`) |
| `inbound-media-reconcile` | `runInboundMediaReconcile` | Reconcile stale inbound-media queue entries |
| `legacy-media-reconcile` | `runLegacyMediaReconcile` | Mark missing local-media files as unavailable |
| `version` | (inline) | Print version + build time |

**Auxiliary binaries**:

| Binary | Command | Description |
|---|---|---|
| `whatomate-license-admin` | `inspect`, `verify` | License decode/verify CLI |
| `whatomate-license-issue` | (flag-based) | Issue signed license tokens |
| `whatomate-license-studio` | (flag-based) | Web UI for license management |
| `whatomate-license-vendor` | `keygen`, `issue` | Key generation + issuance for vendors |

---

## 2. API Handlers & Routes

**Route registration**: `cmd/whatomate/main.go:1178-1854` (`setupRoutes`)
**Handler container**: `internal/handlers/app.go:29` — `type App struct`
**Handler signature**: `func (a *App) MethodName(r *fastglue.Request) error`

### 2.1 System

| Method | Route | Handler | Auth |
|---|---|---|---|
| GET | `/health` | `HealthCheck` | No |
| GET | `/ready` | `ReadyCheck` | No |
| GET | `/metrics` | *(prometheus)* | No |

### 2.2 License

| Method | Route | Handler | Auth |
|---|---|---|---|
| GET | `/api/license/bootstrap` | `GetLicenseBootstrap` | No |
| POST | `/api/license/activate` | `ActivateLicense` | No |

### 2.3 Auth

| Method | Route | Handler |
|---|---|---|
| POST | `/api/auth/login` | `Login` |
| POST | `/api/auth/register` | `Register` |
| POST | `/api/auth/refresh` | `RefreshToken` |
| POST | `/api/auth/register/invite` | `CreateRegisterInvite` |
| POST | `/api/auth/logout` | `Logout` |
| POST | `/api/auth/switch-org` | `SwitchOrg` |
| GET | `/api/auth/ws-token` | `GetWSToken` |
| GET | `/api/auth/me` | `GetCurrentUser` |

### 2.4 SSO

| Method | Route | Handler |
|---|---|---|
| GET | `/api/auth/sso/providers` | `GetPublicSSOProviders` |
| GET | `/api/auth/sso/{provider}/init` | `InitSSO` |
| GET | `/api/auth/sso/{provider}/callback` | `CallbackSSO` |

### 2.5 Webhooks (Meta inbound)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/webhook` | `WebhookVerify` |
| POST | `/api/webhook` | `WebhookHandler` |

### 2.6 WebSocket

| Method | Route | Handler |
|---|---|---|
| GET | `/ws` | `WebSocketHandler` |

### 2.7 Current User

| Method | Route | Handler |
|---|---|---|
| GET | `/api/me` | `GetCurrentUser` |
| PUT | `/api/me/settings` | `UpdateCurrentUserSettings` |
| POST | `/api/me/chat-background` | `UploadCurrentUserChatBackground` |
| GET | `/api/me/chat-background` | `GetCurrentUserChatBackground` |
| PUT | `/api/me/password` | `ChangePassword` |
| PUT | `/api/me/availability` | `UpdateAvailability` |
| GET | `/api/me/organizations` | `ListMyOrganizations` |

### 2.8 Users (admin)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/users` | `ListUsers` |
| POST | `/api/users` | `CreateUser` |
| GET | `/api/users/{id}` | `GetUser` |
| PUT | `/api/users/{id}` | `UpdateUser` |
| DELETE | `/api/users/{id}` | `DeleteUser` |
| GET | `/api/users/{id}/send-restrictions` | `GetUserSendRestrictions` |
| PUT | `/api/users/{id}/send-restrictions` | `UpdateUserSendRestrictions` |

### 2.9 Roles & Permissions

| Method | Route | Handler |
|---|---|---|
| GET | `/api/roles` | `ListRoles` |
| POST | `/api/roles` | `CreateRole` |
| GET | `/api/roles/{id}` | `GetRole` |
| PUT | `/api/roles/{id}` | `UpdateRole` |
| DELETE | `/api/roles/{id}` | `DeleteRole` |
| GET | `/api/permissions` | `ListPermissions` |

### 2.10 API Keys

| Method | Route | Handler |
|---|---|---|
| GET | `/api/api-keys` | `ListAPIKeys` |
| POST | `/api/api-keys` | `CreateAPIKey` |
| DELETE | `/api/api-keys/{id}` | `DeleteAPIKey` |

### 2.11 Organizations

| Method | Route | Handler |
|---|---|---|
| GET | `/api/organizations` | `ListOrganizations` |
| POST | `/api/organizations` | `CreateOrganization` |
| DELETE | `/api/organizations/{id}` | `DeleteOrganization` |
| GET | `/api/organizations/current` | `GetCurrentOrganization` |
| GET | `/api/organizations/members` | `ListOrganizationMembers` |
| POST | `/api/organizations/members` | `AddOrganizationMember` |
| PUT | `/api/organizations/members/{member_id}` | `UpdateOrganizationMemberRole` |
| DELETE | `/api/organizations/members/{member_id}` | `RemoveOrganizationMember` |

### 2.12 Organization Settings

| Method | Route | Handler |
|---|---|---|
| GET | `/api/org/settings` | `GetOrganizationSettings` |
| PUT | `/api/org/settings` | `UpdateOrganizationSettings` |
| POST | `/api/org/uploads-cleanup/run` | `RunUploadsCleanupNow` |

### 2.13 WhatsApp Accounts

| Method | Route | Handler |
|---|---|---|
| GET | `/api/accounts` | `ListAccounts` |
| POST | `/api/accounts` | `CreateAccount` |
| GET | `/api/accounts/{id}` | `GetAccount` |
| PUT | `/api/accounts/{id}` | `UpdateAccount` |
| DELETE | `/api/accounts/{id}` | `DeleteAccount` |
| POST | `/api/accounts/{id}/test` | `TestAccountConnection` |
| POST | `/api/accounts/{id}/subscribe` | `SubscribeApp` |
| GET | `/api/accounts/{id}/business_profile` | `GetBusinessProfile` |
| PUT | `/api/accounts/{id}/business_profile` | `UpdateBusinessProfile` |
| POST | `/api/accounts/{id}/business_profile/photo` | `UpdateProfilePicture` |

### 2.14 WhatsApp Instances (whatsmeow)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/instances` | `ListInstances` |
| POST | `/api/instances` | `CreateInstance` |
| GET | `/api/instances/{id}` | `GetInstance` |
| PUT | `/api/instances/{id}` | `UpdateInstance` |
| DELETE | `/api/instances/{id}` | `DeleteInstance` |
| GET | `/api/instances/{id}/health` | `GetInstanceHealth` |
| GET | `/api/instances/{id}/qr` | `GetInstanceQRCodeSnapshot` |
| POST | `/api/instances/{id}/connect` | `ConnectInstance` |
| POST | `/api/instances/{id}/pair-phone` | `PairPhoneInstance` |
| POST | `/api/instances/{id}/disconnect` | `DisconnectInstance` |
| POST | `/api/instances/{id}/reconnect` | `ReconnectInstance` |
| POST | `/api/instances/{id}/status/send` | `SendStatus` |
| POST | `/api/instances/{id}/auto-campaign/media` | `UploadInstanceAutoCampaignMedia` |

### 2.15 Contacts

| Method | Route | Handler |
|---|---|---|
| GET | `/api/contacts` | `ListContacts` |
| POST | `/api/contacts` | `CreateContact` |
| GET | `/api/contacts/{id}` | `GetContact` |
| PUT | `/api/contacts/{id}` | `UpdateContact` |
| DELETE | `/api/contacts/{id}` | `DeleteContact` |
| POST | `/api/contacts/{id}/soft-delete` | `SoftDeleteContactForUser` |
| PUT | `/api/contacts/{id}/assign` | `AssignContact` |
| GET | `/api/contacts/{id}/collaborators` | `ListContactCollaborators` |
| POST | `/api/contacts/{id}/collaborators` | `InviteContactCollaborator` |
| PUT | `/api/contacts/{id}/collaborators/{user_id}/accept` | `AcceptContactCollaborator` |
| PUT | `/api/contacts/{id}/collaborators/{user_id}/decline` | `DeclineContactCollaborator` |
| DELETE | `/api/contacts/{id}/collaborators/{user_id}` | `RemoveContactCollaborator` |
| PUT | `/api/contacts/{id}/tags` | `UpdateContactTags` |
| GET | `/api/contacts/{id}/session-data` | `GetContactSessionData` |

### 2.16 Chats (lifecycle)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/chats` | `ListContacts` (alias) |
| PUT | `/api/chats/{id}/claim` | `ClaimChat` |
| PUT | `/api/chats/{id}/close` | `CloseChat` |
| PUT | `/api/chats/{id}/reopen` | `ReopenChat` |
| PUT | `/api/chats/{id}/public` | `SetChatPublic` |
| GET | `/api/chats/{id}/messages` | `GetMessages` |

### 2.17 Messages & Messaging

| Method | Route | Handler |
|---|---|---|
| POST | `/api/contacts/{id}/messages` | `SendMessage` |
| POST | `/api/contacts/{id}/typing` | `SendTypingPresence` |
| POST | `/api/contacts/{id}/messages/{message_id}/reaction` | `SendReaction` |
| POST | `/api/contacts/{id}/messages/{message_id}/revoke` | `RevokeMessage` |
| POST | `/api/messages` | `SendMessage` (legacy) |
| POST | `/api/messages/template` | `SendTemplateMessage` |
| POST | `/api/messages/media` | `SendMediaMessage` |
| PUT | `/api/messages/{id}/read` | `MarkMessageRead` |

### 2.18 WhatsApp Statuses (Stories)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/statuses` | `ListStatuses` |
| GET | `/api/statuses/{id}/media` | `ServeStatusMedia` |
| POST | `/api/statuses/{id}/reply` | `ReplyToStatus` |
| POST | `/api/statuses/{id}/mark-seen` | `MarkStatusSeen` |

### 2.19 Media

| Method | Route | Handler |
|---|---|---|
| GET | `/api/media/{message_id}` | `ServeMedia` |
| POST | `/api/media/{message_id}/retry-download` | `RetryMediaDownload` |

### 2.20 Templates (Meta only)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/templates` | `ListTemplates` |
| POST | `/api/templates` | `CreateTemplate` |
| GET | `/api/templates/{id}` | `GetTemplate` |
| PUT | `/api/templates/{id}` | `UpdateTemplate` |
| DELETE | `/api/templates/{id}` | `DeleteTemplate` |
| POST | `/api/templates/sync` | `SyncTemplates` |
| POST | `/api/templates/{id}/publish` | `SubmitTemplate` |
| POST | `/api/templates/upload-media` | `UploadTemplateMedia` |

### 2.21 WhatsApp Flows (Meta only)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/flows` | `ListFlows` |
| POST | `/api/flows` | `CreateFlow` |
| GET | `/api/flows/{id}` | `GetFlow` |
| PUT | `/api/flows/{id}` | `UpdateFlow` |
| DELETE | `/api/flows/{id}` | `DeleteFlow` |
| POST | `/api/flows/{id}/save-to-meta` | `SaveFlowToMeta` |
| POST | `/api/flows/{id}/publish` | `PublishFlow` |
| POST | `/api/flows/{id}/deprecate` | `DeprecateFlow` |
| POST | `/api/flows/{id}/duplicate` | `DuplicateFlow` |
| POST | `/api/flows/sync` | `SyncFlows` |

### 2.22 Campaigns (bulk messaging)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/campaigns` | `ListCampaigns` |
| POST | `/api/campaigns` | `CreateCampaign` |
| GET | `/api/campaigns/{id}` | `GetCampaign` |
| PUT | `/api/campaigns/{id}` | `UpdateCampaign` |
| DELETE | `/api/campaigns/{id}` | `DeleteCampaign` |
| POST | `/api/campaigns/{id}/start` | `StartCampaign` |
| POST | `/api/campaigns/{id}/pause` | `PauseCampaign` |
| POST | `/api/campaigns/{id}/cancel` | `CancelCampaign` |
| POST | `/api/campaigns/{id}/retry-failed` | `RetryFailed` |
| GET | `/api/campaigns/{id}/recipients` | `GetCampaignRecipients` |
| POST | `/api/campaigns/{id}/recipients/import` | `ImportRecipients` |
| DELETE | `/api/campaigns/{id}/recipients/{recipientId}` | `DeleteCampaignRecipient` |
| POST | `/api/campaigns/{id}/media` | `UploadCampaignMedia` |
| GET | `/api/campaigns/{id}/media` | `ServeCampaignMedia` |

### 2.23 Group Targeting (whatsmeow)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/accounts/{instanceId}/groups` | `ListInstanceGroups` |
| POST | `/api/campaigns/{id}/groups/validate` | `ValidateGroupJIDs` |
| POST | `/api/campaigns/{id}/groups` | `AddCampaignGroups` |
| GET | `/api/campaigns/{id}/groups` | `ListCampaignGroups` |
| DELETE | `/api/campaigns/{id}/groups/{recipientId}` | `DeleteCampaignGroup` |

### 2.24 Group Directory

| Method | Route | Handler |
|---|---|---|
| GET | `/api/groups/directory` | `SearchGroupDirectory` |
| POST | `/api/groups/directory` | `CreateGroupDirectory` |
| PUT | `/api/groups/directory/{id}` | `UpdateGroupDirectory` |
| DELETE | `/api/groups/directory/{id}` | `DeleteGroupDirectory` |
| GET | `/api/groups/directory/categories` | `GetGroupDirectoryCategories` |
| GET | `/api/groups/directory/countries` | `GetGroupDirectoryCountries` |
| POST | `/api/groups/directory/preview` | `PreviewGroupFromLink` |
| POST | `/api/groups/directory/import` | `ImportDirectoryGroupsToCampaign` |

### 2.25 Group Participants (whatsmeow)

| Method | Route | Handler |
|---|---|---|
| GET | `/api/groups/participants` | `ListGroupMembers` |
| POST | `/api/groups/participants/add` | `AddGroupMembers` |
| POST | `/api/groups/participants/remove` | `RemoveGroupMembers` |
| POST | `/api/groups/participants/promote` | `PromoteGroupMembers` |
| POST | `/api/groups/participants/demote` | `DemoteGroupMembers` |

### 2.26 Group Join Campaigns

| Method | Route | Handler |
|---|---|---|
| GET | `/api/group-join-campaigns` | `ListGroupJoinCampaigns` |
| POST | `/api/group-join-campaigns` | `CreateGroupJoinCampaign` |
| GET/PUT/DELETE | `/api/group-join-campaigns/{id}` | CRUD |
| POST | `/api/group-join-campaigns/{id}/start` | `StartGroupJoinCampaign` |
| POST | `/api/group-join-campaigns/{id}/pause` | `PauseGroupJoinCampaign` |
| GET | `/api/group-join-campaigns/{id}/stats` | `GroupJoinCampaignStats` |
| GET | `/api/group-join-campaigns/{id}/recipients` | `ListGroupJoinRecipients` |
| POST | `/api/group-join-campaigns/{id}/recipients` | `UploadGroupJoinRecipients` |
| DELETE | `/api/group-join-campaigns/{id}/recipients/{id}` | `DeleteGroupJoinRecipient` |

### 2.27 WhatsApp Filter (number validation)

| Method | Route | Handler |
|---|---|---|
| POST | `/api/whatsapp-filter/batches` | `CreateWhatsAppFilterBatch` |
| GET | `/api/whatsapp-filter/batches` | `ListWhatsAppFilterBatches` |
| GET/DELETE | `/api/whatsapp-filter/batches/{id}` | Get/Delete |
| GET | `/api/whatsapp-filter/batches/{id}/results` | `GetResults` |
| GET | `/api/whatsapp-filter/batches/{id}/export` | `ExportResults` |

### 2.28 Extraction Campaigns

| Prefix | Handler |
|---|---|---|
| `/api/message-extraction-campaigns` | Full CRUD + start/pause/stats/results/export |
| `/api/group-extraction-campaigns` | Full CRUD + start/pause/stats/results/export |
| `/api/member-extraction-campaigns` | Full CRUD + start/pause/stats/results/export |
| `/api/extract/contacts` | List/Export contacts + stats + sync trigger |

### 2.29 Chatbot

| Method | Route | Handler |
|---|---|---|
| GET/PUT | `/api/chatbot/settings` | Chatbot settings |
| GET/POST | `/api/chatbot/keywords` | Keyword rules CRUD |
| GET/POST | `/api/chatbot/flows` | Chatbot flows CRUD |
| GET/POST | `/api/chatbot/ai-contexts` | AI contexts CRUD |
| GET/POST | `/api/chatbot/transfers` | Agent transfers |
| POST | `/api/chatbot/transfers/pick` | `PickNextTransfer` |
| PUT | `/api/chatbot/transfers/{id}/resume` | `ResumeFromTransfer` |
| PUT | `/api/chatbot/transfers/{id}/assign` | `AssignAgentTransfer` |
| GET | `/api/chatbot/sessions` | Chatbot sessions |

### 2.30 Agent Selection

| Method | Route | Handler |
|---|---|---|
| GET/PUT/DELETE | `/api/agent-selection/settings` | Settings |
| GET/POST | `/api/agent-selection/participants` | Participants |
| GET/POST | `/api/agent-selection/options` | Options |
| POST | `/api/agent-selection/preview` | `PreviewAgentSelectionMenu` |
| POST | `/api/agent-selection/test-send` | `TestSendAgentSelectionMenu` |
| GET | `/api/agent-selection/audit` | Audit log |
| GET | `/api/agent-selection/sessions` | Sessions |
| POST | `/api/agent-selection/sessions/{id}/cancel` | Cancel session |

### 2.31 Teams

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/teams` | Teams |
| GET/PUT/DELETE | `/api/teams/{id}` | Team CRUD |
| GET/POST | `/api/teams/{id}/members` | Members |
| DELETE | `/api/teams/{id}/members/{member_user_id}` | Remove member |

### 2.32 Tags

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/tags` | Tags |
| PUT/DELETE | `/api/tags/{name}` | Update/Delete |

### 2.33 Canned Responses

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/canned-responses` | Canned responses |
| GET/PUT/DELETE | `/api/canned-responses/{id}` | CRUD |
| POST | `/api/canned-responses/{id}/send` | `SendCannedResponse` |
| POST | `/api/canned-responses/{id}/use` | `IncrementCannedResponseUsage` |

### 2.34 Saved Contents (Content Library)

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/saved-contents` | Content library |
| GET | `/api/saved-contents/categories` | Categories |
| GET/PUT/DELETE | `/api/saved-contents/{id}` | CRUD |
| GET | `/api/saved-contents/{id}/preview` | Preview |
| POST | `/api/saved-contents/import` | Bulk import |
| POST/GET | `/api/saved-contents/{id}/media` | Upload/Serve |

### 2.35 Conversation Notes

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/contacts/{id}/notes` | Notes |
| PUT/DELETE | `/api/contacts/{id}/notes/{note_id}` | Update/Delete |

### 2.36 Analytics

| Method | Route | Handler |
|---|---|---|
| GET | `/api/analytics/dashboard` | `GetDashboardStats` |
| GET | `/api/analytics/messages` | `GetMessageAnalytics` |
| GET | `/api/analytics/chatbot` | `GetChatbotAnalytics` |
| GET | `/api/analytics/agents` | `GetAgentAnalytics` |
| GET | `/api/analytics/agents/comparison` | `GetAgentComparison` |
| GET | `/api/analytics/agents/{id}` | `GetAgentDetails` |
| GET | `/api/analytics/agents/ratings/export` | `ExportAgentRatings` |
| GET | `/api/analytics/meta` | `GetMetaAnalytics` (meta) |
| POST | `/api/analytics/meta/refresh` | `RefreshMetaAnalyticsCache` (meta) |

### 2.37 Widgets

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/widgets` | Widgets |
| GET | `/api/widgets/data-sources` | Data sources |
| GET | `/api/widgets/data` | `GetAllWidgetsData` |
| GET/PUT/DELETE | `/api/widgets/{id}` | Widget CRUD |
| GET | `/api/widgets/{id}/data` | `GetWidgetData` |
| POST | `/api/widgets/layout` | `SaveWidgetLayout` |

### 2.38 Webhooks (outgoing)

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/webhooks` | Webhooks |
| GET/PUT/DELETE | `/api/webhooks/{id}` | CRUD |
| POST | `/api/webhooks/{id}/test` | `TestWebhook` |

### 2.39 Custom Actions

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/custom-actions` | Custom actions |
| GET/PUT/DELETE | `/api/custom-actions/{id}` | CRUD |
| POST | `/api/custom-actions/{id}/execute` | `ExecuteCustomAction` |
| GET | `/api/custom-actions/redirect/{token}` | `CustomActionRedirect` (public) |

### 2.40 Catalogs (Meta only)

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/catalogs` | Catalogs |
| GET/DELETE | `/api/catalogs/{id}` | Get/Delete |
| POST | `/api/catalogs/sync` | `SyncCatalogs` |
| GET/POST | `/api/catalogs/{id}/products` | Products |
| GET/PUT/DELETE | `/api/products/{id}` | Product CRUD |

### 2.41 Facebook

| Method | Route | Handler |
|---|---|---|
| GET/POST | `/api/facebook/accounts` | FB accounts |
| GET/PUT/DELETE | `/api/facebook/accounts/{id}` | Account CRUD |
| GET | `/api/facebook/accounts/{id}/oauth/renew` | `RenewFacebookOAuth` |
| POST | `/api/facebook/oauth/init` | `InitFacebookOAuth` |
| GET | `/api/facebook/oauth/callback` | `CallbackFacebookOAuth` |
| POST | `/api/facebook/accounts/{id}/pages/{page_id}/feed` | `PostFacebookPage` |
| GET | `/api/facebook/accounts/{id}/pages/{page_id}/insights` | `GetFacebookPageInsights` |
| POST | `/api/facebook/accounts/{id}/pages/{page_id}/messages` | `SendFacebookPageMessage` |
| GET | `/api/facebook/comments` | `ListFacebookComments` |
| GET | `/api/facebook/page-search` | `SearchFBPages` |
| GET/POST | `/api/facebook/people-search` | People search |
| POST | `/api/facebook/people-search/add-contacts` | `AddFBPeopleContacts` |
| POST | `/api/facebook/comments/sync` | `SyncFacebookComments` |
| GET/PUT | `/api/facebook/comments/settings` | Comment settings |
| POST | `/api/facebook/comments/{id}/reply` | `ReplyFacebookComment` |
| PUT | `/api/facebook/comments/{id}/status` | `UpdateCommentStatus` |
| GET/POST | `/api/facebook/comments/webhook` | Webhook (public) |

### 2.42 Notifications, Import/Export, SSO, Config

| Method | Route | Handler |
|---|---|---|
| GET/PUT | `/api/notifications/{id}/dismiss` | Notifications |
| POST | `/api/export` | `ExportData` |
| POST | `/api/import` | `ImportData` |
| GET | `/api/export/{table}/config` | `GetExportConfig` |
| GET | `/api/import/{table}/config` | `GetImportConfig` |
| GET/PUT/DELETE | `/api/settings/sso/{provider}` | SSO settings |
| GET | `/api/config` | `GetAppConfig` |

**Total**: ~230 registered routes across 30+ domains

---

## 3. Frontend Routes

**Router**: `frontend/src/router/index.ts` | **Layout**: `AppLayout.vue`
**Permissions**: Checked via `meta.permission` / `meta.adminOnly` / `meta.metaOnly`

### 3.1 Public Routes

| Path | Name | Component |
|---|---|---|
| `/login` | `login` | `views/auth/LoginView.vue` |
| `/register` | `register` | `views/auth/RegisterView.vue` |
| `/auth/sso/callback` | `sso-callback` | `views/auth/SSOCallbackView.vue` |
| `/activate` | `activate` | `views/public/ActivateLicenseView.vue` |
| `/pricing` | `marketing-redirect` | `views/public/MarketingRedirectView.vue` |

### 3.2 Authenticated Routes

| Path | Component | Permission |
|---|---|---|
| `/` | redirect → `/chat` | — |
| `/dashboard` | `DashboardView.vue` | `analytics` |
| `/chat` | `ChatView.vue` | `chat` |
| `/chat/:contactId` | `ChatView.vue` (props:true) | `chat` |
| `/profile` | `ProfileView.vue` | — |
| `/templates` | `TemplatesView.vue` | `templates` (meta) |
| `/flows` | `FlowsView.vue` | `flows.whatsapp` (meta) |
| `/campaigns` | `CampaignsView.vue` | `campaigns` |
| `/instances` | `InstancesView.vue` | `accounts` |
| `/instances/health` | `InstanceHealthView.vue` | `accounts` |
| `/chatbot` | `ChatbotView.vue` | `settings.chatbot` |
| `/chatbot/keywords` | `KeywordsView.vue` | `chatbot.keywords` |
| `/chatbot/flows` | `ChatbotFlowsView.vue` | `flows.chatbot` |
| `/chatbot/flows/new` | `ChatbotFlowBuilderView.vue` | `flows.chatbot` |
| `/chatbot/flows/:id/edit` | `ChatbotFlowBuilderView.vue` | `flows.chatbot` |
| `/chatbot/ai` | `AIContextsView.vue` | `chatbot.ai` |
| `/chatbot/transfers` | `AgentTransfersView.vue` | `transfers` |
| `/analytics/agents` | `AgentAnalyticsView.vue` | `analytics.agents` |
| `/analytics/meta-insights` | `MetaInsightsView.vue` | `analytics` (meta) |
| `/settings` | `SettingsView.vue` | `settings.general` or `settings.uploads_cleanup` |
| `/settings/chatbot` | `ChatbotSettingsView.vue` | `settings.chatbot` |
| `/settings/accounts` | `AccountsView.vue` | `accounts` (meta) |
| `/canned-responses` | `CannedResponsesView.vue` | `canned_responses` |
| `/saved-contents` | `SavedContentsView.vue` | `saved_contents` |
| `/contacts` | `ContactsView.vue` | `contacts` |
| `/closed-chats` | `ClosedChatsView.vue` | `chat` |
| `/tags` | `TagsView.vue` | `tags` |
| `/whatsapp-filter` | `WhatsAppFilter.vue` | `wa_filter` |
| `/group-search` | `GroupSearch.vue` | `campaigns` |
| `/group-join-campaigns` | `GroupJoinCampaignsView.vue` | `campaigns` |
| `/group-extraction` | `GroupExtractionView.vue` | `campaigns` |
| `/member-extraction` | `MemberExtractionView.vue` | `campaigns` |
| `/group-participants` | `GroupParticipantsView.vue` | `campaigns` |
| `/extract` | `ExtractView.vue` | `campaigns` |
| `/settings/users` | `UsersView.vue` | `users` |
| `/settings/roles` | `RolesView.vue` | `roles` |
| `/settings/teams` | `TeamsView.vue` | `teams` |
| `/settings/agent-selection` | `AgentSelectionView.vue` | `agent_selection` |
| `/settings/api-keys` | `APIKeysView.vue` | `api_keys` |
| `/settings/webhooks` | `WebhooksView.vue` | `webhooks` |
| `/settings/sso` | `SSOSettingsView.vue` | `settings.sso` |
| `/settings/license` | `LicenseSettingsView.vue` | admin only |
| `/settings/custom-actions` | `CustomActionsView.vue` | `custom_actions` |
| `/facebook/comments` | `FacebookCommentsView.vue` | `chat` |
| `/facebook/page-search` | `PageSearchView.vue` | `chat` |
| `/facebook/people-search` | `PeopleSearchView.vue` | `chat` |
| `/facebook/group-search` | `FacebookGroupSearchView.vue` | `chat` |
| `/facebook/extract-likes` | `ExtractLikesView.vue` | `chat` |
| `/facebook/page-messengers` | `PageMessengersView.vue` | `chat` |
| `/facebook/extract-data` | `ExtractDataView.vue` | `chat` |
| `/facebook/auto-share` | `AutoShareView.vue` | `chat` |
| `/facebook/retargeting` | `RetargetingView.vue` | `chat` |
| `/facebook/accounts` | `FacebookAccountsView.vue` | `accounts` |
| `/:pathMatch(.*)*` | `NotFoundView.vue` | — |

**Total**: 62 route definitions (5 public + 57 authenticated)

---

## 4. Database Models

**Base**: `internal/models/models.go:140` — `BaseModel` (UUID PK, timestamps, soft-delete)
**ORM**: GORM + PostgreSQL 17
**Custom types**: `JSONB`, `JSONBArray`, `StringArray`

### 4.1 Core (`models.go`)

| Model | Table | Key Fields |
|---|---|---|
| `Organization` | `organizations` | Name, Slug, Settings(JSONB) |
| `User` | `users` | Email, PasswordHash, RoleID, IsSuperAdmin |
| `UserOrganization` | `user_organizations` | UserID, OrganizationID, RoleID, IsDefault |
| `UserAvailabilityLog` | `user_availability_logs` | UserID, IsAvailable |
| `Team` | `teams` | Name, AssignmentStrategy |
| `TeamMember` | `team_members` | TeamID, UserID, Role |
| `APIKey` | `api_keys` | Name, KeyPrefix, KeyHash |
| `LicenseRecord` | `license_records` | ActivationToken, Tier, Status, HWIDHash |
| `LicenseEvent` | `license_events` | EventType, Reason |
| `SSOProvider` | `sso_providers` | Provider, ClientID, ClientSecret |
| `Webhook` | `webhooks` | Name, URL, Events(StringArray) |
| `CustomAction` | `custom_actions` | Name, ActionType, Config(JSONB) |
| `WhatsAppAccount` | `whatsapp_accounts` | PhoneID, AccessToken, Status |
| `Contact` | `contacts` | PhoneNumber, AssignedUserID, Tags |
| `MediaAsset` | `media_assets` | FileHash, S3Key, MimeType |
| `ContactUserDeletion` | `contact_user_deletions` | ContactID, UserID |
| `Message` | `messages` | Direction, MessageType, Status, ReplyToMessageID |
| `Template` | `templates` | MetaTemplateID, Category, BodyContent |
| `WhatsAppFlow` | `whatsapp_flows` | MetaFlowID, FlowJSON(JSONB) |
| `Widget` / `WidgetFilter` | `widgets` | DataSource, Metric, Filters |
| `NotificationRule` | `notification_rules` | TriggerType, TriggerConfig(JSONB) |

### 4.2 Instance (`instance.go`)

| Model | Table | Key Fields |
|---|---|---|
| `WhatsAppInstance` | `whatsapp_instances` | PhoneNumber, JID, Status, IsDefault |
| `InstanceNotification` | `instance_notifications` | EventType, Message, ContactID |

### 4.3 Chatbot (`chatbot.go`)

| Model | Table | Key Fields |
|---|---|---|
| `ChatbotSettings` | `chatbot_settings` | IsEnabled, DefaultResponse, AgentAssignment, SLA, AI |
| `KeywordRule` | `keyword_rules` | Keywords(StringArray), MatchType, ResponseType |
| `ChatbotFlow` | `chatbot_flows` | TriggerKeywords, InitialMessage, PanelConfig(JSONB) |
| `ChatbotFlowStep` | `chatbot_flow_steps` | StepOrder, MessageType, InputType, ConditionalNext |
| `ChatbotSession` | `chatbot_sessions` | Status, CurrentFlowID, SessionData(JSONB) |
| `ChatbotSessionMessage` | `chatbot_session_messages` | Direction, Message, StepName |
| `AIContext` | `ai_contexts` | ContextType, StaticContent, ApiConfig(JSONB) |
| `AgentTransfer` | `agent_transfers` | Status, Source, AgentID, TeamID, SLA |

### 4.4 Catalog (`catalog.go`)

| Model | Table | Key Fields |
|---|---|---|
| `Catalog` | `catalogs` | MetaCatalogID, Name |
| `CatalogProduct` | `catalog_products` | MetaProductID, Price, Currency |

### 4.5 Campaigns (`bulk.go`)

| Model | Table | Key Fields |
|---|---|---|
| `BulkMessageCampaign` | `bulk_message_campaigns` | TemplateID, Status, ScheduledAt |
| `BulkMessageRecipient` | `bulk_message_recipients` | PhoneNumber, TemplateParams(JSONB), Status |

### 4.6 Roles & Permissions (`roles.go`)

| Model | Table | Key Fields |
|---|---|---|
| `Permission` | `permissions` | Resource, Action |
| `CustomRole` | `custom_roles` | Name, IsSystem, Permissions |
| `RolePermission` | `role_permissions` | CustomRoleID, PermissionID |

### 4.7 Facebook (`fb_*.go`)

| Model | Table | Key Fields |
|---|---|---|
| `FacebookAccount` | `facebook_accounts` | Platform, AccountUID, Method |
| `FacebookOAuthState` | `facebook_oauth_states` | StateToken, ExpiresAt |
| `FacebookComment` | `facebook_comments` | PostID, Message, Direction |
| `FacebookCommentReply` | `facebook_comment_replies` | ReplyText, IsAuto |
| `FacebookCommentSettings` | `facebook_comment_settings` | SyncEnabled, AutoReplyEnabled |
| `FBPeopleSearch` | `fb_people_searches` | Name, FollowersCount |
| `FBPageSearch` | `fb_page_searches` | Name, FollowersCount |

### 4.8 WhatsApp Filter (`whatsapp_filter.go`)

| Model | Table | Key Fields |
|---|---|---|
| `WhatsAppFilterBatch` | `whatsapp_filter_batches` | Status, TotalNumbers, ValidNumbers |
| `WhatsAppFilterResult` | `whatsapp_filter_results` | PhoneNumber, IsValid |

### 4.9 Group/Extraction (`group_*.go`, `*_extraction.go`)

| Model | Table | Key Fields |
|---|---|---|
| `GroupDirectory` | `group_directories` | GroupJID, Country, Category |
| `GroupJoinCampaign` | `group_join_campaigns` | Speed, Status |
| `GroupJoinRecipient` | `group_join_recipients` | InviteLink, GroupJID, Status |
| `GroupExtractionCampaign` | `group_extraction_campaigns` | InstanceID, Status |
| `GroupExtractionResult` | `group_extraction_results` | GroupJID, ParticipantCount |
| `MemberExtractionCampaign` | `member_extraction_campaigns` | GroupJID, GroupName, Status |
| `MemberExtractionResult` | `member_extraction_results` | ParticipantJID, PhoneNumber |
| `MessageExtractionCampaign` | `message_extraction_campaigns` | Status |
| `MessageExtractionResult` | `message_extraction_results` | ChatJID, PhoneNumber |

### 4.10 Agent Selection (`agent_selection.go`)

| Model | Table | Key Fields |
|---|---|---|
| `AgentSelectionSettings` | `agent_selection_settings` | TriggerMode, PromptDelayMinutes |
| `AgentSelectionParticipant` | `agent_selection_participants` | UserID, MaxOpenChats |
| `AgentSelectionOption` | `agent_selection_options` | OptionType, UserID, TeamID |
| `AgentSelectionSession` | `agent_selection_sessions` | Status, ExpiresAt |
| `AgentSelectionAuditEvent` | `agent_selection_audit_events` | EventType, ActorType |

### 4.11 Other Models

| Model | Table | Key Fields |
|---|---|---|
| `SavedContent` | `saved_contents` | Name, Body, Category |
| `CannedResponse` | `canned_responses` | Shortcut, Content |
| `ContactCollaborator` | `contact_collaborators` | Role, Status |
| `OrganizationConfig` | `organization_configs` | WorkerCount, MaxQueueSize |
| `ConversationNote` | `conversation_notes` | Content |
| `WhatsAppStatus` | `whatsapp_statuses` | StatusType, Content |
| `ChatClosureRating` | `chat_closure_ratings` | State, Rating |
| `Tag` | `tags` | Name, Color |

**Total**: 70 GORM models across 20 files

---

## 5. Provider / Service Layer

### 5.1 MessageProvider Interface (`pkg/provider/interface.go`)

Core interface: `SendText`, `SendImage`, `SendDocument`, `SendVideo`, `SendAudio`, `MarkRead`, `SendReaction`, `RevokeMessage`, `GetMediaURL`, `DownloadMedia`, `UploadMedia`

**Extension interfaces**:
- `ReplyProvider` — `SendTextReply()`
- `GroupProvider` — `GetGroups()`, `VerifyGroupMembership()`
- `GroupParticipantProvider` — Add/Remove/Promote/Demote/Get
- `JoinGroupProvider` — `JoinGroupWithLink()`
- `GroupInfoProvider` — `GetGroupInfoFromLink()`

### 5.2 Meta Cloud API Adapter (`pkg/whatsapp/`)

| Component | File | Description |
|---|---|---|
| `Client` | `client.go` | Raw HTTP client for Meta Graph API |
| `MetaAdapter` | `adapter.go` | Wraps `Client` → implements `MessageProvider` + `ReplyProvider` |
| `types.go` | `types.go` | Meta API DTOs |
| `contacts.go` | Contacts | Contact management |
| `template.go` | Template | Template CRUD |
| `analytics.go` | Analytics | Analytics retrieval |
| `catalog.go` | Catalog | Catalog/product management |
| `flow.go` | Flow | WhatsApp Flow management |
| `profile_extras.go` | Profile | Business profile extras |

### 5.3 Whatsmeow Adapter (`pkg/whatsmeow/`)

| Component | File | Description |
|---|---|---|
| `WhatsmeowAdapter` | `adapter.go` | Implements all provider interfaces |
| `ConnectionManager` | `manager.go` | Multi-tenant connection lifecycle |
| `ConnectionPool` | `pool.go` | Thread-safe runtime registry |
| `QueueManager` | `queue.go` | Per-instance rate-limiting queue |
| `MediaService` | `media_service.go` | Inbound media → object storage |
| `MessagePersist` | `message_persist.go` | Inbound message persistence + contact sync |

### 5.4 Handler App Container (`internal/handlers/app.go`)

Central DI struct wiring: Config, DB, Redis, WhatsApp, WhatsmeowManager, ObjectStorage, WSHub, Queue, MessageProvider, License

---

## 6. Queue & Worker Layer

### 6.1 Queue (`internal/queue/`)

| Component | File | Description |
|---|---|---|
| `Queue` (interface) | `queue.go` | `Enqueue*` methods for all job types |
| `JobHandler` (interface) | `queue.go` | `Handle*Job` methods |
| `Consumer` (interface) | `queue.go` | `Consume(ctx, handler)` |
| `RedisQueue` | `redis.go` | Redis Streams implementation |
| `RedisConsumer` | `redis.go` | XREADGROUP consumer with dead-letter |
| `Publisher` | `pubsub.go` | Redis Pub/Sub campaign stats |
| `Subscriber` | `pubsub.go` | Redis Pub/Sub subscriber |

**Job types**: Recipient, InboundMedia, ContactRepair, WhatsAppFilter, GroupJoin, MessageExtraction, GroupExtraction, MemberExtraction

### 6.2 Worker (`internal/worker/`)

| Component | File | Description |
|---|---|---|
| `Worker` | `worker.go` | Core job processor (implements `JobHandler`) |
| `WorkerScaler` | `scaler.go` | Dynamic per-tenant autoscaling |
| Campaign helpers | `campaign_*.go` | Template placeholders, send delay |
| Domain workers | `whatsapp_filter.go`, `group_join.go`, `*_extraction.go` | Per-domain processing |
| `idempotency.go` | `idempotency.go` | Recipient lock (Redis) |
| `send_policy.go` | `send_policy.go` | Inbound-only enforcement |

---

## 7. WebSocket Layer

**Package**: `internal/websocket/`

| Component | File | Description |
|---|---|---|
| `Hub` | `hub.go` | Org→User→Client fan-out pub/sub |
| `Client` | `client.go` | Per-connection WS client |
| `WSMessage` | `messages.go` | Wire type: `{Type, Payload}` |
| `BroadcastMessage` | `messages.go` | Internal broadcast envelope |

**Message types**: `auth`, `new_message`, `status_update`, `contact_update`, `set_contact`, `ping/pong`, `agent_transfer*`, `campaign_stats_update`, `permissions_updated`, `conversation_note_*`, `chat_collaborator_*`, `instance_*`, `facebook_comment_*`

---

## 8. Middleware

**Package**: `internal/middleware/`

| Middleware | File | Description |
|---|---|---|
| `CORS` | `middleware.go` | Cross-origin resource sharing |
| `SecurityHeaders` | `middleware.go` | Security headers + CSP with nonce |
| `Recovery` | `middleware.go` | Panic recovery |
| `Auth` | `middleware.go` | JWT authentication |
| `AuthWithDB` | `middleware.go` | JWT auth with DB user lookup |
| `OrganizationContext` | `middleware.go` | Inject organization into context |
| `TenantScope` | `middleware.go` | GORM DB scoping per org |
| `PermissionChecker` | `middleware.go` | RBAC permission check |
| `RequirePermission` / `RequireAnyPermission` | `middleware.go` | Shorthand permission middleware |
| `RequestLogger` | `middleware.go` | HTTP request logging |
| `CSRFProtection` | `csrf.go` | Double-submit cookie CSRF |
| `RateLimit` | `ratelimit.go` | Rate limiting (Redis) |

---

## 9. Shared Types, Interfaces & Enums

### 9.1 Enums (`internal/models/constants.go`)

21 core enum types: `TeamRole`, `Direction`, `MessageType`, `MessageStatus`, `InstanceStatus`, `AIProvider`, `MatchType`, `ResponseType`, `FlowStepType`, `SessionStatus`, `TransferStatus`, `TransferSource`, `CampaignStatus`, `TemplateStatus`, `TemplateCategory`, `ContextType`, `InputType`, `AssignmentStrategy`, `SSOProviderType`, `WebhookEvent`, `ActionType`

Additional enums in domain files: `ChatStatus`, `CollaboratorRole`, `CollaboratorStatus`, `AgentSelectionTriggerMode`, `FacebookAccountStatus`, `GroupJoinSpeed`, etc.

### 9.2 DTOs / Request-Response Types

| File | Types |
|---|---|
| `internal/handlers/auth_types.go` | `LoginRequest`, `RegisterRequest`, `RefreshRequest`, `RegisterInviteClaims`, `SwitchOrgRequest`, `LogoutRequest` |
| `internal/handlers/sso_types.go` | `SSOState`, `SSOProviderPublic`, `SSOProviderRequest`, `SSOProviderResponse` |
| `pkg/whatsapp/types.go` | `WebhookPayload`, `ParsedMessage`, `MetaAPIError`, `MetaTemplate`, `TemplateComponent`, `CatalogInfo`, `BusinessProfile`, `ProductInfo`, etc. |

### 9.3 Helper Types

| File | Types |
|---|---|
| `pkg/provider/context.go` | `WithSkipTypingIndicator`, `ShouldSkipTypingIndicator` |
| `pkg/provider/interface.go` | `GroupInfo`, `GroupParticipant` |
| `internal/middleware/middleware.go` | `JWTClaims` (UserID, OrganizationID, Email, RoleID, IsSuperAdmin) |
| `internal/websocket/messages.go` | `WSMessage`, `BroadcastMessage` |

---

## 10. Configuration

| File | Format | Description |
|---|---|---|
| `config.example.toml` | TOML | Reference config (git-tracked) |
| `config.toml` | TOML | Actual config (gitignored) |
| `internal/config/` | Go | koanf-based config loading + validation |

Key config sections: `[server]`, `[database]`, `[redis]`, `[whatsapp]` (provider selection: `meta` or `whatsmeow`), `[whatsmeow]`, `[storage]`, `[license]`

---

## 11. Supporting Modules

| Package | Description |
|---|---|---|
| `internal/config/` | Config loading via koanf |
| `internal/crypto/` | AES-256 encrypt/decrypt for secrets at rest |
| `internal/database/` | GORM + PostgreSQL setup, migrations, seeding |
| `internal/frontend/` | Vue SPA embedding (`//go:embed all:dist`) |
| `internal/tenant/` | Multi-tenancy (HostOrganization, TenantScope) |
| `internal/license/` | Runtime license enforcement |
| `internal/licenseissuer/` | License issuance & key ring storage |
| `internal/licensestudio/` | License management HTTP server |
| `internal/campaignstats/` | Campaign receipt aggregation |
| `frontend/` | Vue 3 SPA (Vite + Pinia + TypeScript + shadcn-vue + Tailwind) |
| `mcp-server/` | MCP sidecar (Node.js) |