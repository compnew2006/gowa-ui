# Whatomate Feature Workflows

Complete documentation of all distinct features, their execution paths, inputs, processing steps, outputs, and dependencies.

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

---

## 1. Authentication & Authorization

**Source Files:** `internal/handlers/auth_handlers.go`, `internal/handlers/auth_utils.go`, `internal/handlers/auth_crypto.go`, `internal/handlers/cookies.go`, `internal/middleware/auth.go`

### Login Flow
**Inputs:** Email, Password (JSON body)
**Entry Point:** `POST /api/auth/login` → `App.Login()`

**Execution Path:**
1. Decode JSON request into `LoginRequest`
2. Query database for user by email with role preloaded (`internal/handlers/auth_handlers.go:28`)
3. If user not found, perform dummy bcrypt comparison (timing attack prevention)
4. Load role permissions from cache via `GetRolePermissionsCached()`
5. Check `user.IsActive` flag — reject if disabled
6. Verify password via `bcrypt.CompareHashAndPassword()`
7. Generate access token (JWT with HS256, 15-min expiry) via `generateAccessToken()`
8. Generate refresh token (JWT with HS256, 7-day expiry) via `generateRefreshToken()`
9. Store refresh token JTI in Redis for rotation tracking
10. Set HTTP-only, Secure cookies for both tokens via `setAuthCookies()`
11. Return user object with expiresIn

**Outputs:** `{expires_in, user}` with auth cookies set
**Dependencies:** PostgreSQL (user store), Redis (refresh token store), JWT secret
**Edge Cases:** Timing-attack prevention via dummy bcrypt; disabled accounts rejected; invalid credentials return generic message

### Registration Flow
**Entry Point:** `POST /api/auth/register` → `App.Register()`

**Execution Path:**
1. Decode `RegisterRequest` (email, password, full_name, invitation_token)
2. Validate invitation token via `validateRegisterInviteToken()` — checks expiry and org
3. Verify organization exists
4. Find org's default role (is_default=true or name="agent")
5. Validate password strength via `validatePasswordStrength()`
6. If email already exists, return success without creating duplicate
7. Hash password with bcrypt
8. Begin transaction:
   - Create user record
   - Create user_organizations membership record
9. Commit transaction
10. Return success message

**Outputs:** `{message: "Registration submitted..."}`
**Dependencies:** JWT invite token validation, org default role
**Edge Cases:** Duplicate email silently succeeds; password strength enforced

### Refresh Token Flow
**Entry Point:** `POST /api/auth/refresh` → `App.RefreshToken()`

**Execution Path:**
1. Extract refresh token from cookie or JSON body
2. Parse and validate JWT (HS256, subject="refresh")
3. Check Redis for token JTI — if deleted, token was already used (replay attack prevention)
4. Delete JTI from Redis (single-use rotation)
5. Load user from database
6. Verify user is active
7. Apply organization context from token via `applyRefreshTokenOrgContext()`
8. Populate role permissions via `populateUserRolePermissions()`
9. Generate new access token and refresh token pair
10. Set new cookies
11. Return response

**Outputs:** New token pair with cookies
**Dependencies:** Redis (token revocation check), PostgreSQL
**Edge Cases:** Token rotation prevents replay attacks; org context preserved

### Logout Flow
**Entry Point:** `POST /api/auth/logout` → `App.Logout()`

**Execution Path:**
1. Extract refresh token from cookie or body
2. Parse JWT and extract JTI
3. Delete JTI from Redis (revoke token)
4. Clear auth cookies via `clearAuthCookies()`
5. Return success status

### Organization Switch
**Entry Point:** `POST /api/auth/switch-org` → `App.SwitchOrg()`

**Execution Path:**
1. Extract user_id from context
2. Decode `SwitchOrgRequest` with target organization_id
3. Verify organization exists
4. Load user record
5. If not super admin, verify user_organizations membership exists
6. Update user's organization_id and role_id from membership
7. Regenerate both tokens with new org context
8. Set new cookies

### WebSocket Token
**Entry Point:** `GET /api/auth/ws-token` → `App.GetWSToken()`

**Execution Path:**
1. Extract user_id and org_id from context
2. Create short-lived JWT (30-second expiry, subject="ws")
3. Sign with HS256
4. Return token for WebSocket authentication

---

## 2. User Management

**Source Files:** `internal/handlers/users.go`, `internal/handlers/users_helpers_test.go`

### List Users
**Entry Point:** `GET /api/users` → `App.ListUsers()`
**Inputs:** org_id, user_id (from auth context), pagination, search, status filter
**Execution Path:**
1. Authorize with `requirePermission(users, read)`
2. Query users filtered by organization_id
3. Apply search (email, full_name ILIKE), status filter
4. Preload role relationships
5. Paginate and return
**Outputs:** Paginated user list with roles

### Create User
**Entry Point:** `POST /api/users` → `App.CreateUser()`
**Inputs:** email, full_name, role_id, is_active, organization context
**Execution Path:**
1. Authorize with `requirePermission(users, write)`
2. Validate email uniqueness within org
3. Generate temporary password or require invitation flow
4. Hash password with bcrypt
5. Create user record
6. Create user_organizations membership
7. Dispatch webhook for user_created event
**Outputs:** Created user object

### Update User
**Entry Point:** `PUT /api/users/{id}` → `App.UpdateUser()`
**Execution Path:**
1. Authorize with `requirePermission(users, write)`
2. Load user, verify org membership
3. Apply partial updates (name, role, active status)
4. If role changed, update user_organizations.role_id
5. Save and return

### Delete User
**Entry Point:** `DELETE /api/users/{id}` → `App.DeleteUser()`
**Execution Path:**
1. Authorize with `requirePermission(users, delete)`
2. Soft-delete user (set deleted_at)
3. Remove from user_organizations
4. Reassign any active chats if needed
**Edge Cases:** Cannot delete self; last admin protection

### User Send Restrictions
**Entry Point:** `GET/PUT /api/users/{id}/send-restrictions`
**Execution Path:**
1. Load user send restriction config
2. Update allowed instances, rate limits
3. Persist to database

---

## 3. Organization Management

**Source Files:** `internal/handlers/organization.go`, `internal/handlers/organization_delete_test.go`

### List Organizations
**Entry Point:** `GET /api/organizations` → `App.ListOrganizations()`
**Execution Path:**
1. Extract user_id from context
2. Query organizations where user has user_organizations membership
3. Return list with membership details

### Create Organization
**Entry Point:** `POST /api/organizations` → `App.CreateOrganization()`
**Execution Path:**
1. Authorize (admin only)
2. Validate name uniqueness
3. Create organization record
4. Create default roles (admin, agent, manager)
5. Add creator as admin member
6. Create default chatbot settings

### Delete Organization
**Entry Point:** `DELETE /api/organizations/{id}` → `App.DeleteOrganization()`
**Execution Path:**
1. Authorize (super admin only)
2. Soft-delete organization
3. Cascade soft-delete to all related records (users, accounts, campaigns, etc.)

### Current Organization
**Entry Point:** `GET /api/organizations/current` → `App.GetCurrentOrganization()`

### Organization Members
**Entry Points:** `GET/POST/PUT/DELETE /api/organizations/members`
**Execution Path:**
1. Manage user_organizations records
2. Add/remove members with role assignments
3. Update member roles

### Organization Settings
**Entry Points:** `GET/PUT /api/org/settings`
**Execution Path:**
1. Load/update organization-level settings
2. Settings include: timezone, business hours, default language, etc.

---

## 4. Roles & Permissions (RBAC)

**Source Files:** `internal/handlers/roles.go`

### List Roles
**Entry Point:** `GET /api/roles` → `App.ListRoles()`
**Execution Path:**
1. Authorize with `requirePermission(roles, read)`
2. Query custom_roles filtered by organization_id
3. Include system roles (admin, agent, manager)
4. Preload permissions

### Create Role
**Entry Point:** `POST /api/roles` → `App.CreateRole()`
**Inputs:** name, permissions array, is_default flag
**Execution Path:**
1. Authorize with `requirePermission(roles, write)`
2. Validate role name uniqueness within org
3. Create custom_role record
4. Create permission records (resource:action pairs)
5. Invalidate role permissions cache

### Update Role
**Entry Point:** `PUT /api/roles/{id}` → `App.UpdateRole()`
**Execution Path:**
1. Load role, verify org ownership
2. Update name if provided
3. Delete existing permissions
4. Create new permissions from request
5. Invalidate cache

### Delete Role
**Entry Point:** `DELETE /api/roles/{id}` → `App.DeleteRole()`
**Edge Cases:** Cannot delete system roles; reassign users to default role

### List Permissions
**Entry Point:** `GET /api/permissions` → `App.ListPermissions()`
**Outputs:** All available resource:action pairs

---

## 5. API Key Management

**Source Files:** `internal/handlers/apikeys.go`

### List API Keys
**Entry Point:** `GET /api/api-keys` → `App.ListAPIKeys()`
**Execution Path:**
1. Authorize with `requirePermission(api_keys, read)`
2. Query api_keys filtered by organization_id
3. Mask secret keys in response (show prefix only)

### Create API Key
**Entry Point:** `POST /api/api-keys` → `App.CreateAPIKey()`
**Inputs:** name, permissions, expiry
**Execution Path:**
1. Authorize with `requirePermission(api_keys, write)`
2. Generate random API key (prefix + secret)
3. Hash secret with SHA-256
4. Store hashed key with metadata
5. Return full key (only shown once)

### Delete API Key
**Entry Point:** `DELETE /api/api-keys/{id}` → `App.DeleteAPIKey()`
**Execution Path:**
1. Soft-delete API key record
2. Invalidate any cached lookups

---

## 6. WhatsApp Account Management

**Source Files:** `internal/handlers/accounts.go`

### List Accounts
**Entry Point:** `GET /api/accounts` → `App.ListAccounts()`
**Execution Path:**
1. Authorize with `requirePermission(accounts, read)`
2. Query whatsapp_accounts by organization_id
3. Apply filters: status, provider, search
4. Decrypt encrypted fields (access_token, phone_number_id)
5. Return with connection status

### Create Account
**Entry Point:** `POST /api/accounts` → `App.CreateAccount()`
**Inputs:** name, phone_number_id, access_token, business_account_id, webhook_verify_token
**Execution Path:**
1. Authorize with `requirePermission(accounts, write)`
2. Validate name uniqueness
3. Encrypt sensitive fields (access_token, verify_token)
4. Create whatsapp_account record
5. Cache account lookup by phone_number_id
6. Verify connection to Meta API

### Update Account
**Entry Point:** `PUT /api/accounts/{id}` → `App.UpdateAccount()`
**Execution Path:**
1. Load account, verify org ownership
2. Apply partial updates
3. Re-encrypt changed sensitive fields
4. Update cache

### Delete Account
**Entry Point:** `DELETE /api/accounts/{id}` → `App.DeleteAccount()`
**Execution Path:**
1. Verify no active campaigns or instances
2. Soft-delete account
3. Clear cache

### Test Account Connection
**Entry Point:** `POST /api/accounts/{id}/test` → `App.TestAccountConnection()`
**Execution Path:**
1. Make test API call to Meta
2. Verify credentials and permissions
3. Return connection status

### Subscribe App
**Entry Point:** `POST /api/accounts/{id}/subscribe` → `App.SubscribeApp()`
**Execution Path:**
1. Call Meta API to subscribe app to WhatsApp Business Account
2. Enable webhook events

### Business Profile
**Entry Points:** `GET/PUT /api/accounts/{id}/business_profile`, `POST /api/accounts/{id}/business_profile/photo`
**Execution Path:**
1. Fetch/update business profile via Meta API
2. Upload profile picture

---

## 7. WhatsApp Instance Management (WhatsMeow)

**Source Files:** `internal/handlers/instances.go`, `internal/handlers/instance_pairing.go`, `pkg/whatsmeow/`

### List Instances
**Entry Point:** `GET /api/instances` → `App.ListInstances()`
**Execution Path:**
1. Authorize with `requirePermission(accounts, read)`
2. Apply instance access restrictions for user
3. Query instances by organization_id
4. Include connection status, queue depth, health metrics

### Create Instance
**Entry Point:** `POST /api/instances` → `App.CreateInstance()`
**Inputs:** name, is_default, auto_read_receipt, settings (JSONB)
**Execution Path:**
1. Authorize with `requirePermission(accounts, write)`
2. Validate instance name uniqueness
3. Validate instance settings schema
4. Create instance record
5. Initialize whatsmeow store entry

### Update Instance
**Entry Point:** `PUT /api/instances/{id}` → `App.UpdateInstance()`

### Delete Instance
**Entry Point:** `DELETE /api/instances/{id}` → `App.DeleteInstance()`
**Execution Path:**
1. Disconnect active session if connected
2. Soft-delete instance
3. Clean up whatsmeow store data

### Get Instance Health
**Entry Point:** `GET /api/instances/{id}/health` → `App.GetInstanceHealth()`
**Outputs:** uptime, messages sent/received/failed today, error rate, queue depth

### Get QR Code
**Entry Point:** `GET /api/instances/{id}/qr` → `App.GetInstanceQRCodeSnapshot()`
**Execution Path:**
1. Check cached QR code for instance
2. Return base64 QR image if available
3. Include timeout and expiry info

### Connect Instance
**Entry Point:** `POST /api/instances/{id}/connect` → `App.ConnectInstance()`
**Execution Path:**
1. Load instance record
2. Initialize whatsmeow client
3. Start QR code generation or restore existing session
4. Broadcast status via WebSocket

### Pair Phone Instance
**Entry Point:** `POST /api/instances/{id}/pair-phone` → `App.PairPhoneInstance()`
**Inputs:** phone_number, show_push_notification, client_type, client_display_name
**Execution Path:**
1. Generate pairing code via whatsmeow
2. Return code for user to enter on phone
3. Monitor for successful pairing

### Disconnect Instance
**Entry Point:** `POST /api/instances/{id}/disconnect` → `App.DisconnectInstance()`

### Reconnect Instance
**Entry Point:** `POST /api/instances/{id}/reconnect` → `App.ReconnectInstance()`

### Send Status
**Entry Point:** `POST /api/instances/{id}/status/send` → `App.SendStatus()`

### Auto-Campaign Media
**Entry Point:** `POST /api/instances/{id}/auto-campaign/media` → `App.UploadInstanceAutoCampaignMedia()`

---

## 8. Contact Management

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/contacts_helpers_test.go`, `internal/handlers/contact_filters.go`

### List Contacts
**Entry Point:** `GET /api/contacts` → `App.ListContacts()`
**Inputs:** org_id, user_id, pagination, search, tags, assigned_to, status filters
**Execution Path:**
1. Authorize with `requirePermission(contacts, read)`
2. Build query with filters:
   - Search (phone, name, profile_name)
   - Tags (has/all/any)
   - Assigned user
   - Status (open, closed, pending)
   - Date range
3. Join with messages for last_message_at, unread_count
4. Apply instance access restrictions
5. Paginate and return with assigned user names

### Create Contact
**Entry Point:** `POST /api/contacts` → `App.CreateContact()`
**Inputs:** phone_number, name, tags, metadata
**Execution Path:**
1. Authorize with `requirePermission(contacts, write)`
2. Validate phone number format
3. Check for existing contact by phone
4. Create contact record
5. Dispatch webhook for contact_created

### Get Contact
**Entry Point:** `GET /api/contacts/{id}` → `App.GetContact()`

### Update Contact
**Entry Point:** `PUT /api/contacts/{id}` → `App.UpdateContact()`
**Execution Path:**
1. Load contact, verify org access
2. Apply updates (name, tags, metadata)
3. Save and return

### Delete Contact
**Entry Point:** `DELETE /api/contacts/{id}` → `App.DeleteContact()`

### Soft Delete Contact
**Entry Point:** `POST /api/contacts/{id}/soft-delete` → `App.SoftDeleteContactForUser()`
**Execution Path:**
1. Mark contact as deleted for specific user
2. Preserve for other collaborators

### Assign Contact
**Entry Point:** `PUT /api/contacts/{id}/assign` → `App.AssignContact()`
**Execution Path:**
1. Set assigned_user_id on contact
2. Notify assigned user via WebSocket
3. Dispatch webhook for contact_assigned

### Contact Session Data
**Entry Point:** `GET /api/contacts/{id}/session-data` → `App.GetContactSessionData()`
**Outputs:** Recent messages, tags, notes, assignment history

---

## 9. Chat & Messaging

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/contacts_messaging.go`, `internal/handlers/chat_lifecycle.go`

### List Chats
**Entry Point:** `GET /api/chats` → `App.ListContacts()` (alias to contacts)

### Get Messages
**Entry Point:** `GET /api/chats/{id}/messages` → `App.GetMessages()`
**Inputs:** contact_id, pagination, before/after cursor, message type filter
**Execution Path:**
1. Authorize with `requirePermission(messages, read)`
2. Query messages by contact_id
3. Filter by direction (inbound/outbound), type
4. Include reply context, reactions
5. Serve media URLs for media messages
6. Paginate (cursor-based)
**Outputs:** Paginated message list with media URLs

### Claim Chat
**Entry Point:** `PUT /api/chats/{id}/claim` → `App.ClaimChat()`
**Execution Path:**
1. Verify chat is unassigned or in queue
2. Assign to current user
3. Notify via WebSocket
4. Dispatch webhook

### Close Chat
**Entry Point:** `PUT /api/chats/{id}/close` → `App.CloseChat()`
**Execution Path:**
1. Set closed_at, closed_by_user_id
2. Update contact status
3. Notify participants via WebSocket
4. Dispatch webhook for chat_closed

### Reopen Chat
**Entry Point:** `PUT /api/chats/{id}/reopen` → `App.ReopenChat()`
**Execution Path:**
1. Clear closed_at
2. Set status back to open
3. Notify via WebSocket

### Set Chat Public
**Entry Point:** `PUT /api/chats/{id}/public` → `App.SetChatPublic()`
**Execution Path:**
1. Toggle is_public flag
2. Allow collaborator access

### Send Typing Presence
**Entry Point:** `POST /api/contacts/{id}/typing` → `App.SendTypingPresence()`
**Execution Path:**
1. Route through provider (Meta or WhatsMeow)
2. Send typing indicator to WhatsApp

---

## 10. Message Sending (Unified)

**Source Files:** `internal/handlers/messages.go`, `internal/handlers/send_template_test.go`

### Send Message
**Entry Point:** `POST /api/contacts/{id}/messages` → `App.SendMessage()`
**Inputs:** contact_id, content (text), account/instance, reply_to_message_id
**Execution Path:**
1. Authorize with `requirePermission(messages, write)`
2. Load contact and account
3. Build `OutgoingMessageRequest`
4. Call `SendOutgoingMessage()` with `DefaultSendOptions()`:
   - Enforce send restrictions
   - Apply agent name prefix if configured
   - Create message record (status=pending)
   - Determine provider (Meta or WhatsMeow)
   - Send via provider asynchronously
   - Update status (sent/failed)
   - Broadcast via WebSocket
   - Dispatch webhook
**Outputs:** Message object with pending status
**Dependencies:** WhatsApp provider, Redis (rate limiting)

### Send Media Message
**Entry Point:** `POST /api/messages/media` → `App.SendMediaMessage()`
**Inputs:** media file, caption, contact, account
**Execution Path:**
1. Upload media to storage
2. Build media message request
3. Route through `SendOutgoingMessage()`

### Send Template Message
**Entry Point:** `POST /api/messages/template` → `App.SendTemplateMessage()`
**Inputs:** template_id, parameters, contact, account
**Execution Path:**
1. Load template
2. Resolve template parameters
3. Build template message request
4. Route through `SendOutgoingMessage()`

### Send Reaction
**Entry Point:** `POST /api/contacts/{id}/messages/{message_id}/reaction` → `App.SendReaction()`

### Revoke Message
**Entry Point:** `POST /api/contacts/{id}/messages/{message_id}/revoke` → `App.RevokeMessage()`

### Mark Message Read
**Entry Point:** `PUT /api/messages/{id}/read` → `App.MarkMessageRead()`

---

## 11. Media Handling

**Source Files:** `internal/handlers/media.go`, `internal/handlers/whatsapp_media_policy.go`

### Serve Media
**Entry Point:** `GET /api/media/{message_id}` → `App.ServeMedia()`
**Inputs:** message_id, auth token
**Execution Path:**
1. Authenticate request
2. Load message record
3. Verify user has access to contact/org
4. If media is local, serve from storage
5. If media is remote (Meta), proxy download
6. Set appropriate Content-Type headers
**Outputs:** Media file stream
**Edge Cases:** Expired media URLs; deleted messages

### Media Upload
**Entry Point:** Via message send endpoints
**Execution Path:**
1. Receive multipart form data
2. Validate file size and type
3. Store in local storage or upload to Meta
4. Return media ID/URL

---

## 12. Webhook Processing (Meta)

**Source Files:** `internal/handlers/webhook.go`, `internal/handlers/webhook_dispatch.go`, `internal/handlers/chatbot_processor.go`

### Webhook Verification
**Entry Point:** `GET /api/webhook` → `App.WebhookVerify()`
**Inputs:** hub.mode, hub.verify_token, hub.challenge
**Execution Path:**
1. Verify mode == "subscribe"
2. Match verify_token against config or account tokens
3. Return challenge string
**Outputs:** Challenge string (200 OK)

### Webhook Handler
**Entry Point:** `POST /api/webhook` → `App.WebhookHandler()`
**Inputs:** Meta webhook payload (JSON), X-Hub-Signature-256 header
**Execution Path:**
1. Verify HMAC-SHA256 signature
2. Validate payload size and event count
3. Parse webhook payload
4. For each entry/change:
   - If field == "message_template_status_update": update template status
   - If field == "messages": process each message via `processIncomingMessageFull()`
   - If field == "statuses": update message delivery status
5. Return 200 OK immediately (async processing)
**Outputs:** 200 OK
**Dependencies:** Chatbot processor, message status updater

### Message Status Updates
**Execution Path:**
1. Match WhatsApp message ID to internal message
2. Update status (sent → delivered → read, or failed)
3. Update campaign recipient stats if applicable
4. Broadcast via WebSocket
5. Dispatch webhook for message.status_updated

---

## 13. Bulk Campaign Management

**Source Files:** `internal/handlers/campaigns.go`

### List Campaigns
**Entry Point:** `GET /api/campaigns` → `App.ListCampaigns()`
**Inputs:** org_id, status filter, account filter, search, date range
**Execution Path:**
1. Authorize with `requirePermission(campaigns, read)`
2. Query campaigns by org
3. Preload template, account
4. Calculate progress stats
5. Paginate and return

### Create Campaign
**Entry Point:** `POST /api/campaigns` → `App.CreateCampaign()`
**Inputs:** name, whatsapp_account, template_id, body_content, header_media_id, min/max_delay_seconds, scheduled_at
**Execution Path:**
1. Authorize with `requirePermission(campaigns, write)`
2. Validate template exists and is approved
3. Validate account is active
4. Validate delay settings (min 20s, max 45s default)
5. Create campaign record
6. Return campaign with stats

### Update Campaign
**Entry Point:** `PUT /api/campaigns/{id}` → `App.UpdateCampaign()`
**Edge Cases:** Cannot update active/running campaigns

### Delete Campaign
**Entry Point:** `DELETE /api/campaigns/{id}` → `App.DeleteCampaign()`
**Edge Cases:** Cannot delete active campaigns

### Start Campaign
**Entry Point:** `POST /api/campaigns/{id}/start` → `App.StartCampaign()`
**Execution Path:**
1. Verify campaign has recipients
2. Verify template is approved
3. Set status = "running"
4. Set started_at timestamp
5. Publish recipients to Redis queue
6. Broadcast status via WebSocket
7. Publish campaign stats update to Redis pub/sub

### Pause Campaign
**Entry Point:** `POST /api/campaigns/{id}/pause` → `App.PauseCampaign()`
**Execution Path:**
1. Set status = "paused"
2. Workers will skip paused campaign recipients

### Cancel Campaign
**Entry Point:** `POST /api/campaigns/{id}/cancel` → `App.CancelCampaign()`
**Execution Path:**
1. Set status = "cancelled"
2. Set completed_at timestamp
3. Mark pending recipients as cancelled

### Retry Failed
**Entry Point:** `POST /api/campaigns/{id}/retry-failed` → `App.RetryFailed()`
**Execution Path:**
1. Find failed recipients
2. Reset status to pending
3. Re-publish to queue

### Import Recipients
**Entry Point:** `POST /api/campaigns/{id}/recipients/import` → `App.ImportRecipients()`
**Inputs:** CSV/JSON file with phone numbers, names, template params
**Execution Path:**
1. Parse uploaded file
2. Validate phone numbers
3. Bulk insert recipients
4. Update campaign total_recipients count

### Get Campaign Recipients
**Entry Point:** `GET /api/campaigns/{id}/recipients` → `App.GetCampaignRecipients()`

### Delete Campaign Recipient
**Entry Point:** `DELETE /api/campaigns/{id}/recipients/{recipientId}`

### Upload Campaign Media
**Entry Point:** `POST /api/campaigns/{id}/media` → `App.UploadCampaignMedia()`

### Serve Campaign Media
**Entry Point:** `GET /api/campaigns/{id}/media` → `App.ServeCampaignMedia()`

---

## 14. Campaign Worker Processing

**Source Files:** `internal/worker/worker.go`, `internal/worker/campaign_delay.go`, `internal/worker/campaign_template_placeholders.go`, `internal/worker/send_policy.go`, `internal/worker/idempotency.go`

### Worker Job Processing
**Entry Point:** Redis queue consumer → `Worker.HandleRecipientJob()`
**Inputs:** RecipientJob from Redis queue (campaign_id, recipient_id, message template)
**Execution Path:**
1. Acquire distributed lock on recipient_id (prevent duplicates)
2. Load recipient record, verify status = pending
3. Load campaign, verify status = running
4. Apply campaign delay (random between min/max)
5. Resolve template placeholders with recipient params
6. Build message payload
7. Send via MessageProvider
8. Update recipient status (sent/delivered/failed)
9. Update campaign stats (sent_count, failed_count)
10. Publish stats update to Redis pub/sub
11. Release recipient lock
**Outputs:** Updated recipient status, campaign stats
**Dependencies:** Redis (queue, locks, pub/sub), MessageProvider, PostgreSQL

### Idempotency
**Execution Path:**
1. Check Redis for processed job ID
2. Skip if already processed
3. Mark as processing

### Send Policy Enforcement
**Execution Path:**
1. Check business hours restrictions
2. Check user send restrictions
3. Check rate limits
4. Block if outside allowed window

---

## 15. Chatbot Automation

**Source Files:** `internal/handlers/chatbot.go`, `internal/handlers/chatbot_processor.go`

### Get Chatbot Settings
**Entry Point:** `GET /api/chatbot/settings` → `App.GetChatbotSettings()`
**Execution Path:**
1. Authorize with `requirePermission(flows_chatbot, read)`
2. Load or create default settings for org
3. Gather stats (sessions, messages, transfers)
4. Return settings with stats

### Update Chatbot Settings
**Entry Point:** `PUT /api/chatbot/settings` → `App.UpdateChatbotSettings()`
**Inputs:** enabled, greeting_message, fallback_message, session_timeout, business_hours, AI config, SLA settings
**Execution Path:**
1. Authorize with `requirePermission(flows_chatbot, write)`
2. Validate settings
3. Encrypt AI API key if provided
4. Upsert settings record
5. Clear settings cache

### Incoming Message Processing
**Entry Point:** `processIncomingMessageFull()` (called from webhook handler)
**Execution Path:**
1. Find WhatsApp account by phone_number_id
2. Handle reactions separately
3. Get or create contact
4. Dispatch webhook for new contact
5. Parse message payload (text, media, interactive, flow)
6. Save message to database
7. Check if chatbot is enabled
8. If enabled:
   - Check business hours
   - Check for active session
   - Match keyword rules
   - Match flow triggers
   - If AI enabled, call AI provider
   - Apply fallback if no match
   - Send response
   - Update session
9. Broadcast message via WebSocket
10. Dispatch webhook

### Business Hours Check
**Execution Path:**
1. Load business hours config (per day, open/close times)
2. Check current time against schedule
3. If outside hours and automation disabled, queue for agent

### Session Management
**Execution Path:**
1. Check for active session (contact_id, not expired)
2. If expired, create new session
3. Update session last_activity_at
4. Apply session_timeout_minutes

---

## 16. Chatbot AI Integration

**Source Files:** `internal/handlers/chatbot_processor.go`

### AI Response Generation
**Execution Path:**
1. Check if AI is enabled in settings
2. Load AI config (provider, model, max_tokens, system_prompt)
3. Build conversation history (last N messages)
4. Load relevant AI contexts (keyword-matched or static)
5. Construct prompt:
   - System prompt
   - AI context information
   - Conversation history
   - Current message
6. Call AI provider API (OpenAI, etc.)
7. Parse response
8. Send as WhatsApp message
9. Log AI response for analytics
**Dependencies:** External AI API, HTTP client
**Edge Cases:** API rate limits, token limits, network errors

### AI Contexts
**Entry Points:** `GET/POST/PUT/DELETE /api/chatbot/ai-contexts`
**Execution Path:**
1. Manage AI context records
2. Context types: static, dynamic, URL-based
3. Trigger by keywords or always active
4. Priority ordering

---

## 17. Chatbot Keyword Rules

**Source Files:** `internal/handlers/chatbot.go`

### List Keyword Rules
**Entry Point:** `GET /api/chatbot/keywords` → `App.ListKeywordRules()`

### Create Keyword Rule
**Entry Point:** `POST /api/chatbot/keywords` → `App.CreateKeywordRule()`
**Inputs:** name, keywords array, match_type (exact, contains, regex), response_type (text, buttons, flow), response_content, priority, enabled
**Execution Path:**
1. Authorize
2. Create keyword_rule record
3. Store keywords as array
4. Encrypt response content if needed

### Update/Delete Keyword Rule
**Entry Points:** `PUT/DELETE /api/chatbot/keywords/{id}`

### Keyword Matching
**Execution Path (during message processing):**
1. Load enabled rules for org, ordered by priority
2. For each rule:
   - Check match type (exact, contains, regex)
   - If matched, return response
3. Return first match or continue to AI/fallback

---

## 18. Chatbot Flows

**Source Files:** `internal/handlers/chatbot.go`

### List Chatbot Flows
**Entry Point:** `GET /api/chatbot/flows` → `App.ListChatbotFlows()`

### Create Chatbot Flow
**Entry Point:** `POST /api/chatbot/flows` → `App.CreateChatbotFlow()`
**Inputs:** name, description, trigger_keywords, steps (JSON), enabled
**Execution Path:**
1. Authorize
2. Validate flow steps schema
3. Create chatbot_flow record
4. Store steps as JSON

### Update/Delete Flow
**Entry Points:** `PUT/DELETE /api/chatbot/flows/{id}`

### Flow Execution
**Execution Path (during message processing):**
1. Check if message matches flow trigger keywords
2. Load flow steps
3. Track current step in session
4. Execute step (send message, wait for response)
5. Navigate to next step based on response
6. Complete flow or transfer to agent

---

## 19. Agent Transfers

**Source Files:** `internal/handlers/agent_transfers.go`

### List Agent Transfers
**Entry Point:** `GET /api/chatbot/transfers` → `App.ListAgentTransfers()`

### Create Agent Transfer
**Entry Point:** `POST /api/chatbot/transfers` → `App.CreateAgentTransfer()`
**Inputs:** contact_id, reason, priority
**Execution Path:**
1. Create agent_transfer record
2. Set status = pending
3. Notify available agents via WebSocket
4. Update contact assignment

### Pick Next Transfer
**Entry Point:** `POST /api/chatbot/transfers/pick` → `App.PickNextTransfer()`
**Execution Path:**
1. Find oldest pending transfer
2. Assign to current user
3. Set status = assigned
4. Notify via WebSocket

### Resume From Transfer
**Entry Point:** `PUT /api/chatbot/transfers/{id}/resume` → `App.ResumeFromTransfer()`

### Assign Agent Transfer
**Entry Point:** `PUT /api/chatbot/transfers/{id}/assign` → `App.AssignAgentTransfer()`

---

## 20. SLA Processing

**Source Files:** `internal/handlers/sla_processor.go`, `internal/handlers/sla_processor_test.go`

### SLA Processor
**Entry Point:** Background goroutine (started in main.go) → `SLAProcessor.Start()`
**Execution Path (runs every minute):**
1. Load SLA settings for each org
2. For each open chat:
   - Check response SLA (time since last inbound message)
   - Check resolution SLA (time since chat opened)
   - Check escalation SLA (time since first response)
3. If SLA breached:
   - Send warning message to contact (if configured)
   - Notify escalation users via WebSocket
   - Escalate to manager if escalation_minutes exceeded
4. Auto-close chats exceeding auto_close_hours
5. Update SLA metrics

### SLA Settings
Configured in chatbot settings:
- sla_response_minutes
- sla_resolution_minutes
- sla_escalation_minutes
- sla_auto_close_hours
- sla_escalation_notify_ids

---

## 21. Templates Management (Meta)

**Source Files:** `internal/handlers/templates.go`, `internal/handlers/template_engine.go`

### List Templates
**Entry Point:** `GET /api/templates` → `App.ListTemplates()`
**Execution Path:**
1. Guard: provider must be "meta"
2. Authorize with `requirePermission(templates, read)`
3. Query templates by org
4. Apply filters: status, category, language
5. Return with usage stats

### Create Template
**Entry Point:** `POST /api/templates` → `App.CreateTemplate()`
**Inputs:** name, category, language, components (header, body, footer, buttons)
**Execution Path:**
1. Guard: provider must be "meta"
2. Validate template structure
3. Create local template record
4. Optionally sync to Meta

### Update Template
**Entry Point:** `PUT /api/templates/{id}` → `App.UpdateTemplate()`

### Delete Template
**Entry Point:** `DELETE /api/templates/{id}` → `App.DeleteTemplate()`

### Sync Templates
**Entry Point:** `POST /api/templates/sync` → `App.SyncTemplates()`
**Execution Path:**
1. Fetch templates from Meta API
2. Compare with local records
3. Create/update/delete as needed
4. Return sync summary

### Submit Template
**Entry Point:** `POST /api/templates/{id}/publish` → `App.SubmitTemplate()`
**Execution Path:**
1. Validate template is complete
2. Submit to Meta for review
3. Update status to "pending"

### Upload Template Media
**Entry Point:** `POST /api/templates/upload-media` → `App.UploadTemplateMedia()`
**Execution Path:**
1. Upload media file to Meta
2. Return media handle for template use

---

## 22. WhatsApp Flows (Meta)

**Source Files:** `internal/handlers/flows.go`

### List Flows
**Entry Point:** `GET /api/flows` → `App.ListFlows()`
**Guard:** provider must be "meta"

### Create Flow
**Entry Point:** `POST /api/flows` → `App.CreateFlow()`
**Inputs:** name, categories, json_payload (Flow JSON)
**Execution Path:**
1. Validate Flow JSON schema
2. Create local flow record
3. Return flow with Meta ID (if synced)

### Update Flow
**Entry Point:** `PUT /api/flows/{id}` → `App.UpdateFlow()`

### Delete Flow
**Entry Point:** `DELETE /api/flows/{id}` → `App.DeleteFlow()`

### Save Flow to Meta
**Entry Point:** `POST /api/flows/{id}/save-to-meta` → `App.SaveFlowToMeta()`
**Execution Path:**
1. Validate flow JSON
2. Call Meta Flows API to create/update
3. Store Meta flow_id

### Publish Flow
**Entry Point:** `POST /api/flows/{id}/publish` → `App.PublishFlow()`

### Deprecate Flow
**Entry Point:** `POST /api/flows/{id}/deprecate` → `App.DeprecateFlow()`

### Duplicate Flow
**Entry Point:** `POST /api/flows/{id}/duplicate` → `App.DuplicateFlow()`

### Sync Flows
**Entry Point:** `POST /api/flows/sync` → `App.SyncFlows()`

---

## 23. Catalog & Products (Meta)

**Source Files:** `internal/handlers/catalog.go`

### List Catalogs
**Entry Point:** `GET /api/catalogs` → `App.ListCatalogs()`
**Guard:** provider must be "meta"

### Create Catalog
**Entry Point:** `POST /api/catalogs` → `App.CreateCatalog()`

### Delete Catalog
**Entry Point:** `DELETE /api/catalogs/{id}` → `App.DeleteCatalog()`

### Sync Catalogs
**Entry Point:** `POST /api/catalogs/sync` → `App.SyncCatalogs()`

### List Catalog Products
**Entry Point:** `GET /api/catalogs/{id}/products` → `App.ListCatalogProducts()`

### Create/Update/Delete Product
**Entry Points:** `POST/PUT/DELETE /api/catalogs/{id}/products`, `/api/products/{id}`

---

## 24. Canned Responses

**Source Files:** `internal/handlers/canned_responses.go`, `internal/handlers/canned_response_media.go`, `internal/handlers/canned_response_send.go`

### List Canned Responses
**Entry Point:** `GET /api/canned-responses` → `App.ListCannedResponses()`
**Execution Path:**
1. Authorize with `requirePermission(canned_responses, read)`
2. Query by org, filter by category, search
3. Return with usage counts

### Create Canned Response
**Entry Point:** `POST /api/canned-responses` → `App.CreateCannedResponse()`
**Inputs:** shortcut, content, category, media (optional)
**Execution Path:**
1. Authorize with `requirePermission(canned_responses, write)`
2. Validate shortcut uniqueness
3. Create record

### Update/Delete Canned Response
**Entry Points:** `PUT/DELETE /api/canned-responses/{id}`

### Send Canned Response
**Entry Point:** `POST /api/canned-responses/{id}/send` → `App.SendCannedResponse()`
**Execution Path:**
1. Load canned response
2. Build message with content and media
3. Route through `SendOutgoingMessage()`
4. Increment usage count

### Increment Usage
**Entry Point:** `POST /api/canned-responses/{id}/use` → `App.IncrementCannedResponseUsage()`

---

## 25. Tags Management

**Source Files:** `internal/handlers/tags.go`

### List Tags
**Entry Point:** `GET /api/tags` → `App.ListTags()`
**Execution Path:**
1. Query tags by org
2. Include usage count (contacts with tag)
3. Return sorted by name

### Create Tag
**Entry Point:** `POST /api/tags` → `App.CreateTag()`
**Inputs:** name, color
**Execution Path:**
1. Validate name uniqueness within org
2. Create tag record

### Update Tag
**Entry Point:** `PUT /api/tags/{name}` → `App.UpdateTag()`

### Delete Tag
**Entry Point:** `DELETE /api/tags/{name}` → `App.DeleteTag()`
**Execution Path:**
1. Remove tag from all contacts
2. Delete tag record

---

## 26. Teams Management

**Source Files:** `internal/handlers/teams.go`

### List Teams
**Entry Point:** `GET /api/teams` → `App.ListTeams()`
**Execution Path:**
1. Authorize (admin/manager)
2. Query teams by org
3. Include member count

### Create Team
**Entry Point:** `POST /api/teams` → `App.CreateTeam()`
**Inputs:** name, description, member_ids
**Execution Path:**
1. Create team record
2. Create team_members records

### Update/Delete Team
**Entry Points:** `PUT/DELETE /api/teams/{id}`

### List/Add/Remove Team Members
**Entry Points:** `GET/POST/DELETE /api/teams/{id}/members`

---

## 27. Analytics & Dashboard

**Source Files:** `internal/handlers/analytics.go`, `internal/handlers/agent_analytics.go`, `internal/handlers/analytics_instance_filter.go`

### Dashboard Stats
**Entry Point:** `GET /api/analytics/dashboard` → `App.GetDashboardStats()`
**Execution Path:**
1. Aggregate counts: total contacts, messages, open chats
2. Calculate response times
3. Get today's activity
4. Return summary stats

### Message Analytics
**Entry Point:** `GET /api/analytics/messages` → `App.GetMessageAnalytics()`
**Execution Path:**
1. Query messages by date range
2. Group by type, direction, status
3. Calculate delivery rates
4. Return time-series data

### Chatbot Analytics
**Entry Point:** `GET /api/analytics/chatbot` → `App.GetChatbotAnalytics()`
**Execution Path:**
1. Count sessions, AI responses, transfers
2. Calculate resolution rates
3. Return keyword/flow usage stats

### Agent Analytics
**Entry Point:** `GET /api/analytics/agents` → `App.GetAgentAnalytics()`
**Execution Path:**
1. Aggregate per-agent metrics
2. Response times, resolution times
3. Chat volumes, ratings

### Agent Comparison
**Entry Point:** `GET /api/analytics/agents/comparison` → `App.GetAgentComparison()`

### Agent Details
**Entry Point:** `GET /api/analytics/agents/{id}` → `App.GetAgentDetails()`

### Export Agent Ratings
**Entry Point:** `GET /api/analytics/agents/ratings/export` → `App.ExportAgentRatings()`

---

## 28. Meta Analytics

**Source Files:** `internal/handlers/meta_analytics.go`

### Get Meta Analytics
**Entry Point:** `GET /api/analytics/meta` → `App.GetMetaAnalytics()`
**Guard:** provider must be "meta"
**Execution Path:**
1. Fetch analytics from Meta API
2. Cache results in Redis
3. Return conversation and message metrics

### List Meta Accounts for Analytics
**Entry Point:** `GET /api/analytics/meta/accounts` → `App.ListMetaAccountsForAnalytics()`

### Refresh Meta Analytics Cache
**Entry Point:** `POST /api/analytics/meta/refresh` → `App.RefreshMetaAnalyticsCache()`

---

## 29. Widgets (Custom Analytics)

**Source Files:** `internal/handlers/widgets.go`

### List Widgets
**Entry Point:** `GET /api/widgets` → `App.ListWidgets()`

### Create Widget
**Entry Point:** `POST /api/widgets` → `App.CreateWidget()`
**Inputs:** name, type, query, config
**Execution Path:**
1. Authorize
2. Validate widget configuration
3. Create widget record

### Get Widget Data
**Entry Point:** `GET /api/widgets/{id}/data` → `App.GetWidgetData()`
**Execution Path:**
1. Load widget query config
2. Execute query against database
3. Return formatted data

### Get All Widgets Data
**Entry Point:** `GET /api/widgets/data` → `App.GetAllWidgetsData()`

### Save Widget Layout
**Entry Point:** `POST /api/widgets/layout` → `App.SaveWidgetLayout()`

### Get Data Sources
**Entry Point:** `GET /api/widgets/data-sources` → `App.GetWidgetDataSources()`

---

## 30. Webhooks (Outbound)

**Source Files:** `internal/handlers/webhooks.go`, `internal/handlers/webhooks_mgmt_test.go`

### List Webhooks
**Entry Point:** `GET /api/webhooks` → `App.ListWebhooks()`
**Execution Path:**
1. Query webhooks by org
2. Return with event subscriptions, last triggered

### Create Webhook
**Entry Point:** `POST /api/webhooks` → `App.CreateWebhook()`
**Inputs:** url, events array, secret, enabled
**Execution Path:**
1. Authorize
2. Validate URL format
3. Encrypt secret
4. Create webhook record

### Update/Delete Webhook
**Entry Points:** `PUT/DELETE /api/webhooks/{id}`

### Test Webhook
**Entry Point:** `POST /api/webhooks/{id}/test` → `App.TestWebhook()`
**Execution Path:**
1. Build test payload
2. Send POST to webhook URL
3. Verify response
4. Return test result

### Webhook Dispatch
**Entry Point:** `App.DispatchWebhook()` (called from various handlers)
**Execution Path:**
1. Find enabled webhooks for event type
2. Build payload with event data
3. Sign with HMAC-SHA256 (if secret configured)
4. Send POST request asynchronously
5. Log delivery attempt

---

## 31. Custom Actions

**Source Files:** `internal/handlers/custom_actions.go`, `internal/handlers/custom_action_runtime.go`

### List Custom Actions
**Entry Point:** `GET /api/custom-actions` → `App.ListCustomActions()`

### Create Custom Action
**Entry Point:** `POST /api/custom-actions` → `App.CreateCustomAction()`
**Inputs:** name, url, method, headers, body_template, events
**Execution Path:**
1. Authorize
2. Validate URL (SSRF protection)
3. Encrypt sensitive headers
4. Create custom_action record

### Update/Delete Custom Action
**Entry Points:** `PUT/DELETE /api/custom-actions/{id}`

### Execute Custom Action
**Entry Point:** `POST /api/custom-actions/{id}/execute` → `App.ExecuteCustomAction()`
**Execution Path:**
1. Load action config
2. Resolve body template variables
3. Send HTTP request
4. Log execution result
5. Return response

### Custom Action Redirect
**Entry Point:** `GET /api/custom-actions/redirect/{token}` → `App.CustomActionRedirect()`
**Execution Path:**
1. Validate one-time token
2. Redirect to configured URL
3. Invalidate token

---

## 32. Conversation Notes

**Source Files:** `internal/handlers/conversation_notes.go`

### List Notes
**Entry Point:** `GET /api/contacts/{id}/notes` → `App.ListConversationNotes()`

### Create Note
**Entry Point:** `POST /api/contacts/{id}/notes` → `App.CreateConversationNote()`
**Inputs:** content, contact_id
**Execution Path:**
1. Authorize
2. Create note with user_id, timestamp
3. Return note

### Update/Delete Note
**Entry Points:** `PUT/DELETE /api/contacts/{id}/notes/{note_id}`
**Edge Cases:** Only note author or admin can modify

---

## 33. Status Updates

**Source Files:** `internal/handlers/statuses.go`

### List Statuses
**Entry Point:** `GET /api/statuses` → `App.ListStatuses()`

### Serve Status Media
**Entry Point:** `GET /api/statuses/{id}/media` → `App.ServeStatusMedia()`

### Reply to Status
**Entry Point:** `POST /api/statuses/{id}/reply` → `App.ReplyToStatus()`

### Mark Status Seen
**Entry Point:** `POST /api/statuses/{id}/mark-seen` → `App.MarkStatusSeen()`

---

## 34. SSO Authentication

**Source Files:** `internal/handlers/sso_handlers.go`, `internal/handlers/sso_utils.go`

### Get SSO Providers
**Entry Point:** `GET /api/auth/sso/providers` → `App.GetPublicSSOProviders()`
**Execution Path:**
1. Query enabled SSO providers
2. Return provider names and display info

### Initiate SSO
**Entry Point:** `GET /api/auth/sso/{provider}/init` → `App.InitSSO()`
**Execution Path:**
1. Load provider config
2. Generate state token (CSRF protection)
3. Build OAuth2 authorization URL
4. Redirect user to provider

### SSO Callback
**Entry Point:** `GET /api/auth/sso/{provider}/callback` → `App.CallbackSSO()`
**Execution Path:**
1. Validate state token
2. Exchange code for tokens
3. Fetch user info from provider
4. Find or create local user
5. Generate access/refresh tokens
6. Set cookies
7. Redirect to app

### SSO Settings
**Entry Points:** `GET/PUT/DELETE /api/settings/sso`
**Execution Path:**
1. Manage SSO provider configurations
2. Encrypt client secrets
3. Validate provider connectivity

---

## 35. WebSocket Real-time Communication

**Source Files:** `internal/handlers/websocket.go`, `internal/websocket/hub.go`

### WebSocket Handler
**Entry Point:** `GET /ws` → `App.WebSocketHandler()`
**Execution Path:**
1. Authenticate via query param token (short-lived JWT)
2. Validate token (user_id, org_id, subject="ws")
3. Upgrade HTTP to WebSocket
4. Register connection with Hub
5. Start read loop for client messages
6. Start write loop for server messages
**Outputs:** Persistent bidirectional connection

### Hub Operations
**Execution Path:**
1. Hub maintains org→connections map
2. `BroadcastToOrg()` sends message to all org members
3. Messages include: new messages, status updates, campaign stats, notifications
4. Connection cleanup on disconnect

### WebSocket Token Generation
**Entry Point:** `GET /api/auth/ws-token` → `App.GetWSToken()`
**Outputs:** 30-second single-use JWT

---

## 36. Import/Export Data

**Source Files:** `internal/handlers/import_export.go`

### Export Data
**Entry Point:** `POST /api/export` → `App.ExportData()`
**Inputs:** table, filters, format (CSV/JSON)
**Execution Path:**
1. Authorize
2. Query data with filters
3. Format as CSV or JSON
4. Stream download or queue for large exports

### Import Data
**Entry Point:** `POST /api/import` → `App.ImportData()`
**Inputs:** table, file (CSV/JSON), mapping
**Execution Path:**
1. Parse file
2. Validate records
3. Bulk insert/update
4. Return import summary (created, updated, errors)

### Get Export/Import Config
**Entry Points:** `GET /api/export/{table}/config`, `GET /api/import/{table}/config`
**Outputs:** Available fields, required fields, format info

---

## 37. Lead Requests

**Source Files:** `internal/handlers/lead_requests.go`

### Create Public Lead Request
**Entry Point:** `POST /api/public/lead-requests` → `App.CreatePublicLeadRequest()`
**Inputs:** name, email, phone, message, widget_id
**Execution Path:**
1. Validate input
2. Create lead_request record
3. Notify org admins via WebSocket
4. Dispatch webhook
5. Return success

### List Lead Requests
**Entry Point:** `GET /api/lead-requests` → `App.ListLeadRequests()`

### Update Lead Request Status
**Entry Point:** `PUT /api/lead-requests/{id}/status` → `App.UpdateLeadRequestStatus()`
**Inputs:** status (new, contacted, converted, rejected)

---

## 38. Activity Logging & Retention

**Source Files:** `internal/handlers/activity_logs.go`, `internal/handlers/activity_middleware.go`, `internal/handlers/activity_service.go`, `internal/handlers/activity_retention.go`

### Create Activity Log
**Entry Point:** `POST /api/activity-logs` → `App.CreateActivityLog()`
**Inputs:** action, resource, resource_id, details
**Execution Path:**
1. Extract user from context
2. Create activity_log record
3. Return log entry

### List Activity Logs
**Entry Point:** `GET /api/activity-logs` → `App.ListActivityLogs()`
**Execution Path:**
1. Query logs by org, user, date range
2. Filter by action, resource
3. Paginate and return

### Activity Middleware
**Execution Path:**
1. Intercepts all API requests
2. Logs significant actions (create, update, delete)
3. Records user, path, method, timestamp
4. Persists to activity_logs table

### Activity Retention Worker
**Entry Point:** Background goroutine (hourly) → `ActivityRetentionWorker.Start()`
**Execution Path:**
1. Find logs older than retention period (90 days default)
2. Bulk delete old logs
3. Log retention stats

---

## 39. Data Migration

**Source Files:** `internal/handlers/migration_handler.go`

### Trigger Migration
**Entry Point:** `POST /api/admin/migrate` → `App.TriggerMigration()`
**Execution Path:**
1. Authorize (super admin only)
2. Run database migrations
3. Return migration status

### Get Migration Status
**Entry Point:** `GET /api/admin/migrate/status` → `App.GetMigrationStatus()`

---

## 40. Crypto Migration

**Source Files:** `internal/crypto/`, `cmd/whatomate/main.go:runCryptoMigrate()`

### Crypto Migration Command
**Entry Point:** `whatomate crypto-migrate [-dry-run] [-batch-size N] [-include-enc2]`
**Execution Path:**
1. Load config, validate encryption key
2. Connect to database
3. Scan for legacy encrypted secrets (enc:/enc2: prefixes)
4. Decrypt with old format
5. Re-encrypt with enc3: format
6. Update records in batches
7. Report summary (updated, failed)
**Outputs:** Migration summary
**Dependencies:** Encryption key, PostgreSQL

---

## 41. Chat Assignment & Routing

**Source Files:** `internal/handlers/chat_assignment_reset_settings.go`, `internal/handlers/chat_assignment_reset_worker.go`

### Chat Assignment Reset Settings
**Entry Points:** `GET/PUT /api/chat-assignment-reset-settings`
**Execution Path:**
1. Manage auto-assignment reset schedule
2. Configure reset intervals, conditions

### Chat Assignment Reset Worker
**Entry Point:** Background goroutine (every minute) → `ChatAssignmentResetWorker.Start()`
**Execution Path:**
1. Check schedule for reset rules
2. Find chats matching reset conditions
3. Reset assignments
4. Notify affected users

---

## 42. Contact Collaborators

**Source Files:** `internal/handlers/contact_collaborators.go`

### List Collaborators
**Entry Point:** `GET /api/contacts/{id}/collaborators` → `App.ListContactCollaborators()`

### Invite Collaborator
**Entry Point:** `POST /api/contacts/{id}/collaborators` → `App.InviteContactCollaborator()`
**Inputs:** user_id
**Execution Path:**
1. Create pending collaboration record
2. Notify invited user

### Accept/Decline Collaboration
**Entry Points:** `PUT /api/contacts/{id}/collaborators/{user_id}/accept`, `/decline`

### Remove Collaborator
**Entry Point:** `DELETE /api/contacts/{id}/collaborators/{user_id}` → `App.RemoveContactCollaborator()`

---

## 43. Notifications

**Source Files:** `internal/handlers/notifications.go`

### List Notifications
**Entry Point:** `GET /api/notifications` → `App.ListNotifications()`
**Execution Path:**
1. Query notifications for user/org
2. Filter by type, read status
3. Paginate and return

### Dismiss Notification
**Entry Point:** `PUT /api/notifications/{id}/dismiss` → `App.DismissNotification()`

---

## 44. Business Profile Management

**Source Files:** `internal/handlers/business_profile.go`

### Get Business Profile
**Entry Point:** `GET /api/accounts/{id}/business_profile` → `App.GetBusinessProfile()`
**Execution Path:**
1. Fetch from Meta API
2. Return profile data

### Update Business Profile
**Entry Point:** `PUT /api/accounts/{id}/business_profile` → `App.UpdateBusinessProfile()`

### Update Profile Picture
**Entry Point:** `POST /api/accounts/{id}/business_profile/photo` → `App.UpdateProfilePicture()`

---

## 45. Instance Auto-Campaign

**Source Files:** `internal/handlers/instance_auto_campaign_worker.go`, `internal/handlers/instance_auto_campaign_media.go`

### Instance Auto-Campaign Worker
**Entry Point:** Background goroutine (every minute) → `InstanceAutoCampaignWorker.Start()`
**Execution Path:**
1. Check instances with auto-campaign enabled
2. Find contacts matching criteria
3. Send automated messages
4. Track results

### Upload Auto-Campaign Media
**Entry Point:** `POST /api/instances/{id}/auto-campaign/media` → `App.UploadInstanceAutoCampaignMedia()`

---

## 46. Chat Cleanup

**Source Files:** `internal/handlers/chat_cleanup.go`

### Chat Cleanup Worker
**Entry Point:** Background process → cleanup functions
**Execution Path:**
1. Find expired/closed chats
2. Clean up old messages
3. Archive contact data
4. Update stats

---

## 47. Chat Close Ratings

**Source Files:** `internal/handlers/chat_close_ratings.go`

### Chat Close Ratings
**Execution Path:**
1. When chat is closed, optionally send rating request
2. Collect customer satisfaction rating
3. Store rating with chat
4. Include in agent analytics

---

## 48. Health & Readiness Checks

**Source Files:** `internal/handlers/app.go`

### Health Check
**Entry Point:** `GET /health` → `App.HealthCheck()`
**Outputs:** `{status: "ok", service: "whatomate"}`

### Readiness Check
**Entry Point:** `GET /ready` → `App.ReadyCheck()`
**Execution Path:**
1. Ping database
2. Ping Redis
3. Return ready status or error
**Outputs:** `{status: "ready"}` or 500 error

---

## 49. Rate Limiting

**Source Files:** `internal/middleware/rate_limit.go`, `cmd/whatomate/main.go`

### Rate Limiting Middleware
**Applied To:** Auth endpoints, outbound messages, webhooks
**Execution Path:**
1. Extract key (user_id:org_id or IP)
2. Check Redis counter for window
3. If exceeded, return 429 Too Many Requests
4. Otherwise, increment counter and proceed
**Configuration:** Per-user, per-IP, per-endpoint limits

---

## 50. Frontend Serving

**Source Files:** `internal/frontend/`, `cmd/whatomate/main.go:setupRoutes()`

### Frontend Handler
**Execution Path:**
1. Check if frontend is embedded (build-time flag)
2. If embedded, serve SPA from embedded filesystem
3. Catch-all route returns index.html for client-side routing
4. Static assets served with proper caching headers
**Outputs:** React/Vite frontend application

---

## Background Processes Summary

| Process | Interval | Source | Purpose |
|---------|----------|--------|---------|
| SLA Processor | 1 minute | `sla_processor.go` | Check SLA breaches, auto-close |
| Activity Retention | 1 hour | `activity_retention.go` | Delete old activity logs |
| Chat Assignment Reset | 1 minute | `chat_assignment_reset_worker.go` | Reset stale assignments |
| Instance Auto-Campaign | 1 minute | `instance_auto_campaign_worker.go` | Send automated messages |
| Campaign Worker | Continuous | `worker/worker.go` | Process campaign queue |
| Inbound Media Worker | Continuous | `worker/worker.go` | Download inbound media |
| Campaign Stats Subscriber | Continuous | `app.go` | Broadcast campaign stats via WS |
| WhatsMeow Reconnect | Startup | `main.go` | Reconnect all instances |
| WhatsMeow Auto-Connect | Startup | `main.go` | Connect linked sessions |
| Status Reconciliation | Startup (30s timeout) | `main.go` | Clean stale instance statuses |

---

## Data Flow Diagrams

### Message Sending Flow
```
API Request → Auth Middleware → Permission Check → Load Contact/Account
  → Create Message Record (pending) → Send via Provider (async)
  → Update Status (sent/failed) → WebSocket Broadcast → Webhook Dispatch
```

### Incoming Message Flow
```
Meta Webhook → Signature Verification → Parse Payload
  → Find Account → Get/Create Contact → Save Message
  → Chatbot Processing (keywords, flows, AI, fallback)
  → Send Response → WebSocket Broadcast → Webhook Dispatch
```

### Campaign Flow
```
Create Campaign → Import Recipients → Start Campaign
  → Publish to Redis Queue → Workers Pick Up Jobs
  → Apply Delay → Send Message → Update Stats
  → Publish Stats → WebSocket Broadcast
```

### Authentication Flow
```
Login Request → Find User → Verify Password → Generate JWT Pair
  → Store Refresh Token in Redis → Set Cookies → Return User
```

---

## Provider Abstraction

Whatomate supports two WhatsApp providers:

### Meta (Cloud API)
- Uses official Meta WhatsApp Business Cloud API
- Templates, Flows, Catalogs supported
- Webhook-based message receiving

### WhatsMeow (Direct WhatsApp Web)
- Uses `go.mau.fi/whatsmeow` library
- QR code or phone-code pairing
- Direct WebSocket connection to WhatsApp
- No template approval needed
- Per-instance message queuing

Provider selection is configured in `config.toml` under `whatsapp.provider`.
