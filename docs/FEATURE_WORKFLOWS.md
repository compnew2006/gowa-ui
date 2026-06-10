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
101. [WhatsApp Poll Messages — see docs/POLL_MESSAGES_WORKFLOW.md](#101-whatsapp-poll-messages)
100. [Strict Rollout Mode](#100-strict-rollout-mode)
51. [Send Restriction Policies](#51-send-restriction-policies)
## 101. WhatsApp Poll Messages

**Source Files:** `pkg/whatsmeow/adapter_send.go`, `internal/handlers/contacts_messaging.go`, `frontend/src/views/chat/ChatView.vue`

See the dedicated feature doc for complete workflow details:

➡️ **[docs/POLL_MESSAGES_WORKFLOW.md](POLL_MESSAGES_WORKFLOW.md)**

### Quick Summary
- **Send Native Polls**: Via WhatsMeow provider using `client.BuildPollCreation()`
- **Vote on Polls**: Full LID resolution support for E2E-encrypted poll votes
- **Multi-Selection**: Supports both single-select and unlimited multi-select polls
- **LID Resolution**: Phone-number JIDs resolved to LID JIDs for correct encryption key derivation
- **E2E Encryption Workaround**: Temporary `Store.ID` override for LID sessions during `BuildPollVote()`
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

---

## 51. Send Restriction Policies

**Source Files:** `internal/handlers/send_restriction_policy.go`, `internal/handlers/user_send_restrictions.go`, `internal/handlers/send_restriction_policy_helpers_test.go`

### Overview
Send restriction policies control which users can send messages to which contacts, through which instances, and under what conditions. This is a critical security and compliance feature.

### Configuration Levels
1. **Organization-level settings:**
   - `strict_sending_restrictions_enabled`: Master toggle for strict mode
   - `outbound_mode`: "inbound_only" or "mixed"
   - `strict_sending_apply_to_system`: Whether restrictions apply to system/chatbot messages
   - `campaign_draft_only`: Restrict campaigns to draft mode only
   - `strict_rollout_mode`: "audit" (log violations) or "enforce" (block messages)
   - `strict_rollout_enforce_at`: Timestamp when enforcement begins

2. **User-level settings:**
   - `send_restrictions`: Per-user configuration including:
     - `enabled`: Toggle for this user
     - `include_all_contacts`: Allow all contacts or restrict to authorized numbers
     - `authorized_numbers`: Whitelist of phone numbers
     - `allowed_instance_id` / `allowed_instance_ids`: Which instances user can send from
     - `prefix_agent_name`: Auto-prefix messages with agent name
     - `allow_unclaimed_chat_view`: View unclaimed chats
     - `allow_unclaimed_chat_send`: Send to unclaimed chats

### Enforcement Flow
**Entry Point:** `enforceStrictSendRestrictions()` (called from `SendOutgoingMessage()`)
**Execution Path:**
1. Load organization settings
2. Load user send restrictions
3. Check if strict mode is enabled
4. If enabled:
   - Verify contact is in authorized numbers list (if not include_all_contacts)
   - Verify instance is in allowed instances list
   - Check outbound mode (inbound_only blocks proactive outbound)
   - Check if chat is claimed (if allow_unclaimed_chat_send is false)
5. If violation detected:
   - In "audit" mode: log warning, allow message
   - In "enforce" mode: return `restrictedSendViolationError`, block message
6. Apply agent name prefix if configured

### Update User Send Restrictions
**Entry Point:** `PUT /api/users/{id}/send-restrictions` → `App.UpdateUserSendRestrictions()`
**Inputs:** send_restrictions JSON object
**Execution Path:**
1. Authorize with `requirePermission(users, write)`
2. Load user, verify org membership
3. Validate restriction settings
4. Update user.settings.send_restrictions
5. Return updated settings

### Get User Send Restrictions
**Entry Point:** `GET /api/users/{id}/send-restrictions` → `App.GetUserSendRestrictions()`

### Chat Claim Enforcement
**Execution Path:**
1. When sending message, check if chat is claimed
2. If chat is restricted and unclaimed:
   - Check if user can send to unclaimed chats
   - If not, return 403 with message "This chat is currently unassigned. Claim it before sending messages."
3. Agent-role users have chat-scoped visibility even with contacts:read permission

### Outbound Mode Enforcement
**"inbound_only" mode:**
- Users can only reply to inbound messages
- Proactive outbound messages are blocked
- Campaign messages may be restricted based on campaign_draft_only setting

**"mixed" mode:**
- Both inbound replies and proactive outbound allowed
- Standard permission checks apply

---

## 52. Agent Chat Visibility Restrictions

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/contacts_messaging.go`, `internal/handlers/chat_access_policy.go`

### Overview
Agent-role users have restricted visibility into chats based on assignment status and access policies.

### Restriction Logic
**Execution Path:**
1. `shouldRestrictChatVisibilityToAgentScope()` checks if user is agent role
2. If restricted, apply `applyAgentVisibleChatAccessFilter()`:
   - Only show chats assigned to the user
   - Only show public chats (is_public = true)
   - Only show chats where user is a collaborator
3. When listing contacts, filter query based on user's access scope
4. When sending messages, verify agent has access to the contact

### Agent Message Sending
**Execution Path:**
1. Agent attempts to send message
2. Load contact with agent-scoped query
3. If contact not found in agent's scope, return 404
4. Check if chat is closed — reject if closed
5. Check if chat is restricted and unclaimed — require claim first
6. Proceed with message send

---

## 53. Contact Account Resolution

**Source Files:** `internal/handlers/contact_account_resolution.go`

### Overview
Resolves the correct WhatsApp account for a contact when sending messages, considering instance assignments and account mappings.

### Execution Path
1. Check if contact has an associated instance
2. If instance exists, find account linked to that instance
3. If no instance, use account from request or contact's whatsapp_account field
4. Validate account belongs to same organization
5. Return resolved account or error

---

## 54. Contact Repair

**Source Files:** `internal/handlers/contact_repair.go`

### Overview
Repairs orphaned or inconsistent contact records, typically after data migrations or account changes.

### Execution Path
1. Scan for contacts with missing or invalid references
2. Fix orphaned contacts by reassigning to correct account/instance
3. Update contact metadata to reflect current state
4. Log repair actions

---

## 55. Contact User Deletions

**Source Files:** `internal/handlers/contact_user_deletions.go`

### Overview
Handles cleanup when a user is deleted, reassigning or archiving their contacts and chats.

### Execution Path
1. Find all contacts assigned to deleted user
2. Reassign to team lead or unassign
3. Update chat assignment records
4. Notify affected users via WebSocket
5. Dispatch webhook for contact reassigned events

---

## 56. Closed Chat Filters

**Source Files:** `internal/handlers/closed_chat_filters.go`

### Overview
Provides filtering capabilities for closed chats in the contact list view.

### Filter Options
- Closed by user
- Closed date range
- Close reason
- Rating status

### Execution Path
1. Apply closed_at IS NOT NULL filter
2. Apply additional filters based on query params
3. Join with users table for closed_by_name
4. Paginate and return

---

## 57. Chat Lifecycle Management

**Source Files:** `internal/handlers/chat_lifecycle.go`, `internal/handlers/chat_system_messages.go`

### Chat States
- **open**: Active chat, agents can send messages
- **closed**: Chat closed, read-only
- **pending**: Awaiting agent assignment

### State Transitions
1. **open → closed**: Agent or system closes chat
   - Set closed_at, closed_by_user_id
   - Optionally send auto-close message
   - Optionally request rating
   - Broadcast via WebSocket
   - Dispatch webhook

2. **closed → open**: Agent reopens chat
   - Clear closed_at, closed_by_user_id
   - Set status back to open
   - Broadcast via WebSocket

3. **open → pending**: Chat unassigned
   - Clear assigned_user_id
   - Set status to pending
   - Notify available agents

4. **pending → open**: Chat claimed
   - Set assigned_user_id
   - Set status to open
   - Notify claiming agent

### System Messages
**Execution Path:**
1. Create message with actor_type = "system"
2. Content describes lifecycle event
3. Save to messages table
4. Broadcast via WebSocket
5. No provider send (internal only)

---

## 58. Reply Preview Helpers

**Source Files:** `internal/handlers/reply_preview_helpers.go`

### Overview
Extracts and formats reply context for messages that reference other messages.

### Execution Path
1. When message has reply_to_message_id
2. Load referenced message
3. Extract preview data:
   - Content (truncated for long messages)
   - Message type
   - Sender info
   - Media info (if media message)
4. Return formatted preview for API response

---

## 59. Message Template Placeholders

**Source Files:** `internal/handlers/message_template_placeholders.go`

### Overview
Resolves template placeholders in messages with actual values from contacts, users, or custom data.

### Supported Placeholders
- `{{contact.name}}` - Contact name
- `{{contact.phone}}` - Contact phone number
- `{{user.name}}` - Agent/sender name
- `{{organization.name}}` - Organization name
- Custom placeholders from template params

### Execution Path
1. Parse message content for placeholder patterns
2. Load context data (contact, user, organization)
3. Replace placeholders with actual values
4. Handle missing values (leave as-is or use defaults)
5. Return resolved message content

---

## 60. Campaign Policy Enforcement

**Source Files:** `internal/handlers/campaign_policy.go`

### Overview
Enforces policies on campaign creation and execution, including rate limits, template requirements, and scheduling constraints.

### Policy Checks
1. **Template approval**: Campaign template must be APPROVED
2. **Account status**: WhatsApp account must be active
3. **Delay validation**: Min/max delay within acceptable range
4. **Recipient validation**: Campaign must have at least one recipient
5. **Schedule validation**: Scheduled time must be in the future
6. **Rate limiting**: Check organization campaign rate limits

### Execution Path
1. `validateCampaignForCreate()` called during campaign creation
2. `validateCampaignForStart()` called before starting campaign
3. Return specific error messages for each policy violation
4. Block campaign operation if policy check fails

---

## 61. Flows Helpers

**Source Files:** `internal/handlers/flows_helpers_test.go`

### Overview
Helper functions for WhatsApp Flows operations, including JSON validation and Meta API integration.

### Functions
- Validate Flow JSON schema
- Transform Flow JSON for Meta API
- Parse Flow response from Meta
- Generate Flow tokens for tracking

---

## 62. Group Message Helpers

**Source Files:** `internal/handlers/group_message_helpers.go`

### Overview
Handles group message detection and processing for WhatsApp group chats.

### Execution Path
1. Detect if incoming message is from a group (JID contains @g.us)
2. Extract group metadata (group ID, sender JID)
3. Create or update group contact record
4. Store sender info in message metadata
5. Apply group-specific chatbot rules (if configured)

---

## 63. Instance Name Validation

**Source Files:** `internal/handlers/instance_name_validation.go`

### Overview
Validates WhatsApp instance names for uniqueness and format.

### Validation Rules
1. Name must be non-empty
2. Name must be unique within organization
3. Name must match pattern (alphanumeric, hyphens, underscores)
4. Name length limits (min 2, max 50 characters)

### Execution Path
1. `normalizeInstanceName()` - trim, lowercase, remove invalid chars
2. `isInstanceNameTaken()` - query database for existing name
3. Return validation error if invalid

---

## 64. Instance Selector

**Source Files:** `internal/handlers/instance_selector.go`

### Overview
Selects the appropriate WhatsApp instance for outbound messages based on configuration, availability, and load.

### Selection Strategies
1. **Default instance**: Use organization's default instance
2. **Contact-assigned instance**: Use instance linked to contact
3. **Request-specified instance**: Use instance from API request
4. **Round-robin**: Distribute across available instances (future)

### Execution Path
1. `resolveOutboundInstance()` called from message send handlers
2. Check request-specified instance ID first
3. Fall back to contact's instance
4. Fall back to organization default
5. Validate instance is connected and healthy
6. Return instance or error with reason code

---

## 65. Password Policy

**Source Files:** `internal/handlers/password_policy.go`

### Overview
Enforces password strength requirements during registration and password changes.

### Policy Rules
1. Minimum length: 8 characters
2. At least one uppercase letter
3. At least one lowercase letter
4. At least one digit
5. At least one special character
6. Not in common password list

### Execution Path
1. `validatePasswordStrength()` called during registration and password change
2. Check each policy rule
3. Return specific error message for first violated rule
4. Reject password if any rule fails

---

## 66. Provider Guard

**Source Files:** `internal/handlers/provider_guard.go`

### Overview
Middleware that restricts certain endpoints to specific WhatsApp providers (Meta or WhatsMeow).

### Execution Path
1. `ProviderGuard("meta", handler)` wraps handler
2. Check configured provider in app config
3. If provider doesn't match, return 400 with "Feature not available for current provider"
4. If provider matches, call wrapped handler

### Protected Features
- Templates (Meta only)
- WhatsApp Flows (Meta only)
- Catalogs (Meta only)

---

## 67. Reason Codes

**Source Files:** `internal/handlers/reason_codes.go`

### Overview
Provides standardized reason codes for API error responses, enabling frontend to handle errors programmatically.

### Common Reason Codes
- `instance_not_found`: Specified instance doesn't exist
- `instance_not_connected`: Instance is not connected
- `instance_not_allowed`: User not permitted to use this instance
- `chat_unclaimed`: Chat needs to be claimed before sending
- `chat_closed`: Chat is closed and read-only
- `restriction_violation`: Send restriction policy violated

### Execution Path
1. Create error with reason code using `asInstanceSelectionError()`
2. Return error envelope with `reason_code` field
3. Frontend uses reason code to display appropriate UI

---

## 68. Security Headers & CSRF Protection

**Source Files:** `internal/middleware/security.go`, `internal/middleware/csrf.go`

### Security Headers
**Applied to all responses:**
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`

### CSRF Protection
**Execution Path:**
1. Generate CSRF token on login
2. Store token in HTTP-only cookie (`whm_csrf`)
3. For mutating requests (POST/PUT/DELETE/PATCH):
   - Extract token from `X-CSRF-Token` header
   - Compare with cookie value
   - Reject if mismatch (403 Forbidden)
4. Safe methods (GET/HEAD/OPTIONS) skip CSRF check

---

## 69. Request Logging & Recovery

**Source Files:** `internal/middleware/logger.go`, `internal/middleware/recovery.go`

### Request Logger
**Execution Path:**
1. Log request method, path, remote address
2. Log response status and duration
3. Include request ID for tracing
4. Log user ID if authenticated

### Recovery Middleware
**Execution Path:**
1. Defer panic recovery at start of request
2. If panic occurs:
   - Log stack trace
   - Return 500 Internal Server Error
   - Don't expose panic details to client
3. Continue serving other requests

---

## 70. SSRF-Safe Dialer

**Source Files:** `internal/handlers/helpers.go`

### Overview
Prevents Server-Side Request Forgery (SSRF) attacks by blocking requests to internal IP ranges.

### Blocked Ranges
- 127.0.0.0/8 (loopback)
- 10.0.0.0/8 (private)
- 172.16.0.0/12 (private)
- 192.168.0.0/16 (private)
- 169.254.0.0/16 (link-local)
- ::1 (IPv6 loopback)
- fc00::/7 (IPv6 unique local)
- fe80::/10 (IPv6 link-local)

### Execution Path
1. `SSRFSafeDialer()` creates custom HTTP transport dialer
2. Before connecting, resolve target hostname
3. Check if resolved IP is in blocked ranges
4. If blocked, return connection refused error
5. If allowed, proceed with connection

---

## 71. Cache System

**Source Files:** `internal/handlers/cache.go`

### Cached Data
1. **WhatsApp Accounts**: Lookup by phone_number_id
2. **Role Permissions**: Permission lists by role ID
3. **Chatbot Settings**: Settings by organization ID
4. **Organization Settings**: Settings by organization ID

### Cache Operations
1. **Get**: Check Redis cache first
2. **Miss**: Load from database, store in cache with TTL
3. **Hit**: Return cached value
4. **Invalidate**: Delete cache key on update

### TTL Settings
- Accounts: 5 minutes
- Role permissions: 10 minutes
- Chatbot settings: 5 minutes
- Organization settings: 5 minutes

---

## 72. Cookie Management

**Source Files:** `internal/handlers/cookies.go`

### Auth Cookies
- `whm_access`: Access token (JWT), HTTP-only, Secure, SameSite=Strict
- `whm_refresh`: Refresh token (JWT), HTTP-only, Secure, SameSite=Strict
- `whm_csrf`: CSRF token, HTTP-only, Secure, SameSite=Strict

### Cookie Operations
1. `setAuthCookies()`: Set access, refresh, and CSRF cookies
2. `clearAuthCookies()`: Clear all auth cookies (set expired)
3. Cookie domain and path derived from request
4. Secure flag always enabled in production

---

## 73. JWT Secret Management

**Source Files:** `internal/handlers/jwt_secret.go`

### Overview
Manages JWT signing key with support for environment variable or config file.

### Execution Path
1. `jwtSecretBytes()` retrieves signing key
2. Check environment variable `WHATOMATE_JWT_SECRET` first
3. Fall back to config file value
4. Validate key meets minimum length requirement
5. Return key bytes for JWT signing

---

## 74. WhatsApp Client (Meta)

**Source Files:** `pkg/whatsapp/client.go`

### Overview
HTTP client for Meta WhatsApp Business Cloud API.

### Supported Operations
1. Send text message
2. Send media message (image, video, audio, document)
3. Send template message
4. Send interactive message (buttons, list, CTA URL)
5. Send location message
6. Send contact message
7. Mark message as read
8. Send typing indicator
9. Upload media
10. Download media
11. Fetch templates
12. Create/update/delete templates
13. Submit template for approval
14. Fetch Flows
15. Create/update/delete Flows
16. Publish/deprecate Flows
17. Fetch catalogs and products
18. Fetch business profile
19. Update business profile
20. Fetch analytics

### Execution Path
1. Build API URL from base URL and endpoint
2. Set Authorization header with access token
3. Build request body
4. Send HTTP request
5. Parse response
6. Handle errors (rate limits, invalid credentials, etc.)
7. Return result or error

---

## 75. Message Provider Abstraction

**Source Files:** `pkg/provider/provider.go`, `pkg/whatsapp/meta_adapter.go`, `pkg/whatsmeow/adapter.go`

### Overview
Provider interface abstracts differences between Meta and WhatsMeow providers.

### Interface Methods
1. `SendMessage()`: Send message to contact
2. `SendMediaMessage()`: Send media message
3. `SendTemplateMessage()`: Send template message
4. `MarkRead()`: Mark message as read
5. `SendTyping()`: Send typing indicator

### Meta Adapter
- Routes calls to Meta WhatsApp Client
- Handles Meta-specific error codes
- Transforms response formats

### WhatsMeow Adapter
- Routes calls to WhatsMeow Connection Manager
- Handles per-instance queuing
- Manages rate limiting per instance
- Handles WhatsMeow-specific errors

---

## 76. WhatsMeow Connection Manager

**Source Files:** `pkg/whatsmeow/manager.go`

### Overview
Manages WhatsApp Web connections for multiple instances.

### Connection Lifecycle
1. **Create**: Initialize new instance, generate QR code
2. **Connect**: Start WebSocket connection to WhatsApp
3. **Authenticated**: Session established, ready to send
4. **Disconnected**: Connection lost, attempt reconnect
5. **Logout**: Session terminated, need new QR code

### Connection Management
1. `GetClient()`: Get connected client for instance
2. `Connect()`: Start new connection
3. `Disconnect()`: Close connection
4. `Reconnect()`: Reconnect after disconnect
5. `ReconnectAll()`: Reconnect all active instances
6. `AutoConnectLinkedInstancesOnFirstRun()`: Auto-connect on startup

### Event Handling
1. **Message received**: Process inbound message, enqueue for media download
2. **Receipt received**: Update message status
3. **Presence update**: Update contact presence
4. **Connection status**: Broadcast status change via WebSocket
5. **QR code received**: Cache QR code for API retrieval

### Queue Depth Management
1. Each instance has per-instance message queue
2. Queue depth tracked and reported via API
3. Depth observer callback updates instance record
4. Rate limiting based on queue depth

---

## 77. WhatsMeow Queue Manager

**Source Files:** `pkg/whatsmeow/queue.go`

### Overview
Per-instance message queue for WhatsMeow provider with rate limiting.

### Queue Operations
1. **Enqueue**: Add message to instance queue
2. **Dequeue**: Get next message to send
3. **Depth**: Get current queue depth
4. **Wait**: Block until queue has capacity

### Rate Limiting
1. Configurable messages per minute per instance
2. Adaptive delay based on queue depth
3. Priority queue for high-priority messages

---

## 78. Redis Queue System

**Source Files:** `internal/queue/queue.go`, `internal/queue/consumer.go`, `internal/queue/publisher.go`

### Queue Types
1. **Campaign Queue**: Campaign message jobs
2. **Inbound Media Queue**: Media download jobs
3. **Pub/Sub Channels**: Campaign stats, notifications

### Job Types
1. **RecipientJob**: Campaign recipient message send
2. **InboundMediaJob**: Download media from Meta/WhatsMeow
3. **CampaignStatsUpdate**: Campaign progress update

### Consumer Operations
1. `Consume()`: Start consuming jobs from queue
2. `HandleJob()`: Process individual job
3. `Ack()`: Acknowledge successful job
4. `Nack()`: Negative acknowledge, requeue or dead-letter
5. `Retry()`: Retry failed job with backoff

### Publisher Operations
1. `Publish()`: Add job to queue
2. `PublishCampaignStats()`: Broadcast stats update

---

## 79. Campaign Stats Subscriber

**Source Files:** `internal/queue/subscriber.go`

### Overview
Subscribes to Redis pub/sub for campaign stats updates and broadcasts via WebSocket.

### Execution Path
1. Subscribe to campaign stats channel
2. On message received:
   - Parse stats update
   - Broadcast to organization via WebSocket
   - Log update
3. On disconnect:
   - Auto-resubscribe
   - Reconnect to channel

---

## 80. Database Migrations

**Source Files:** `internal/database/migrations.go`, `internal/handlers/migration_handler.go`

### Overview
Manages database schema migrations using GORM AutoMigrate.

### Migration Process
1. `RunMigrationWithProgress()`:
   - Run GORM AutoMigrate
   - Create default admin user if configured
   - Create default roles for organizations
   - Create default chatbot settings
   - Report migration progress
2. Migrations run on server startup with `-migrate` flag
3. Or triggered via API by super admin

### Default Admin Creation
1. Check if admin user exists
2. If not, create from config:
   - `default_admin.email`
   - `default_admin.password`
   - `default_admin.full_name`
3. Create organization for admin
4. Create default roles
5. Add admin to organization

---

## 81. Encryption System

**Source Files:** `internal/crypto/crypto.go`, `internal/crypto/migration.go`

### Overview
Encrypts sensitive data (access tokens, API keys, secrets) in database.

### Encryption Versions
1. **enc**: Original encryption format
2. **enc2**: Second generation format
3. **enc3**: Current format (AES-256-GCM)

### Encryption Process
1. `Encrypt()`: Encrypt plaintext with AES-256-GCM
2. `Decrypt()`: Decrypt ciphertext
3. Prefix encrypted values with `enc3:` for identification
4. Key derived from `app.encryption_key` config

### Crypto Migration
1. Scan database for legacy encrypted values (enc:, enc2:)
2. Decrypt with old format
3. Re-encrypt with enc3 format
4. Update records in batches
5. Report migration summary

### Encrypted Fields
- WhatsApp account access tokens
- WhatsApp account webhook verify tokens
- SSO client secrets
- Chatbot AI API keys
- Webhook secrets
- Custom action headers

---

## 82. Contact Utilities

**Source Files:** `internal/contactutil/contact.go`

### GetOrCreateContact
**Execution Path:**
1. Query contact by phone number and organization
2. If found, return existing contact
3. If not found:
   - Create new contact record
   - Set phone number, profile name
   - Set status to open
   - Return new contact and isNewContact=true
4. Update profile name if provided and different

---

## 83. Template Utilities

**Source Files:** `internal/templateutil/template.go`

### Overview
Helper functions for template rendering and placeholder resolution.

### Functions
1. `ResolvePlaceholders()`: Replace placeholders in template
2. `ValidateTemplateSyntax()`: Check template for valid placeholders
3. `ExtractPlaceholders()`: List all placeholders in template

---

## 84. WebSocket Message Types

**Source Files:** `internal/websocket/hub.go`, `internal/websocket/messages.go`

### Message Types
1. **message**: New message received
2. **message_status**: Message status updated
3. **contact_created**: New contact created
4. **contact_assigned**: Contact assigned to user
5. **chat_closed**: Chat closed
6. **chat_reopened**: Chat reopened
7. **campaign_stats_update**: Campaign progress update
8. **instance_status**: Instance connection status changed
9. **notification**: New notification
10. **typing**: Typing indicator
11. **presence**: Contact presence update
12. **instance_reconnect_failed**: Instance reconnection failed

### Message Format
```json
{
  "type": "message",
  "payload": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## 85. Frontend Embedded Build

**Source Files:** `internal/frontend/embed.go`, `frontend/`

### Overview
Frontend is built with React/Vite and embedded into Go binary.

### Build Process
1. Frontend built with `npm run build`
2. Output copied to `internal/frontend/dist/`
3. Go `embed` directive includes dist files
4. Server serves embedded files at runtime

### Development Mode
1. Frontend dev server runs separately
2. API proxy configured for development
3. CORS enabled for dev server origin

---

## 86. Configuration System

**Source Files:** `internal/config/config.go`

### Configuration Sections
1. **app**: Application settings (name, version, environment, debug, encryption_key)
2. **server**: Server settings (host, port, read/write timeouts, allowed_origins, max_request_body_size)
3. **database**: PostgreSQL connection (host, port, user, password, dbname, ssl_mode)
4. **redis**: Redis connection (host, port, password, db)
5. **whatsapp**: WhatsApp settings (provider, base_url, webhook_verify_token)
6. **whatsmeow**: WhatsMeow-specific settings (queue_depth, rate_limit)
7. **jwt**: JWT settings (secret, access_token_ttl, refresh_token_ttl)
8. **default_admin**: Default admin user (email, password, full_name)
9. **storage**: File storage settings (local_path)
10. **rate_limit**: Rate limiting settings (enabled, per-user, per-IP limits)

### Config Loading
1. Load from TOML file
2. Override with environment variables
3. Validate required fields
4. Set defaults for optional fields

---

## 87. Model Layer

**Source Files:** `internal/models/`

### Key Models
1. **User**: User accounts with roles and permissions
2. **Organization**: Multi-tenant organizations
3. **CustomRole**: Custom roles with permissions
4. **Permission**: Resource:action permission pairs
5. **WhatsAppAccount**: Meta WhatsApp Business accounts
6. **WhatsAppInstance**: WhatsMeow instances
7. **Contact**: WhatsApp contacts
8. **Message**: WhatsApp messages (inbound/outbound)
9. **BulkMessageCampaign**: Campaign definitions
10. **BulkMessageRecipient**: Campaign recipients
11. **Template**: WhatsApp message templates
12. **ChatbotSettings**: Chatbot configuration
13. **KeywordRule**: Chatbot keyword rules
14. **ChatbotFlow**: Chatbot conversation flows
15. **AIContext**: AI context definitions
16. **AgentTransfer**: Chatbot-to-agent transfers
17. **CannedResponse**: Pre-written responses
18. **Tag**: Contact tags
19. **Team**: User teams
20. **Webhook**: Outbound webhook definitions
21. **CustomAction**: Custom HTTP actions
22. **ConversationNote**: Chat notes
23. **ActivityLog**: Audit log entries
24. **Widget**: Custom analytics widgets
25. **LeadRequest**: Public lead submissions
26. **Notification**: User notifications
27. **UserOrganization**: User-org membership records
28. **TeamMember**: Team membership records
29. **ContactCollaborator**: Contact collaboration records
30. **SSOProvider**: SSO provider configurations

---

## 88. Middleware Chain

**Source Files:** `internal/middleware/`

### Request Processing Order
1. **CORS Wrapper**: Handle CORS headers (fasthttp level)
2. **Security Headers**: Add security headers
3. **Request Logger**: Log request details
4. **Recovery**: Catch panics
5. **CSRF Protection**: Validate CSRF tokens
6. **Activity Log Middleware**: Log significant actions
7. **Auth Middleware**: Validate JWT/API key (for protected routes)
8. **Role-based Access**: Check permissions (handler level)
9. **Provider Guard**: Check provider compatibility (handler level)
10. **Rate Limiting**: Check rate limits (endpoint level)

---

## 89. Error Handling Patterns

### Error Envelope Format
```json
{
  "error": {
    "message": "Human-readable error message",
    "code": "machine_readable_code",
    "field": "field_name_if_validation_error"
  }
}
```

### Common HTTP Status Codes
- **400**: Bad Request (validation errors)
- **401**: Unauthorized (auth failures)
- **403**: Forbidden (permission denied)
- **404**: Not Found (resource not found)
- **409**: Conflict (duplicate, closed chat)
- **413**: Payload Too Large
- **429**: Too Many Requests (rate limited)
- **500**: Internal Server Error

### Error Handling Strategy
1. Validate input early, return 400
2. Check auth, return 401 if invalid
3. Check permissions, return 403 if denied
4. Check resource exists, return 404 if not
5. Check business rules, return 409 if violated
6. Execute operation, return 500 on failure
7. Never expose internal errors to client

---

## 90. Testing Infrastructure

### Test Files
- Unit tests: `*_test.go` files alongside source
- E2E tests: `frontend/e2e/` with Playwright
- Integration tests: Test handlers with test database

### Test Helpers
- `testhelpers_test.go`: Common test utilities
- `stubs.go`: Stub implementations for testing
- `ApiHelper`: E2E API test helper (TypeScript)

### Coverage Reports
- Multiple coverage files for different packages
- `coverage.out`: Main coverage report
- `coverage_*.out`: Package-specific coverage

---

## 51. Send Restriction Policies

**Source Files:** `internal/handlers/send_restriction_policy.go`, `internal/handlers/user_send_restrictions.go`, `internal/handlers/send_restriction_policy_helpers_test.go`

### Overview
Send restriction policies control which users can send messages to which contacts, through which instances, and under what conditions. This is a critical security and compliance feature.

### Configuration Levels
1. **Organization-level settings:**
   - `strict_sending_restrictions_enabled`: Master toggle for strict mode
   - `outbound_mode`: "inbound_only" or "mixed"
   - `strict_sending_apply_to_system`: Whether restrictions apply to system/chatbot messages
   - `campaign_draft_only`: Restrict campaigns to draft mode only
   - `strict_rollout_mode`: "audit" (log violations) or "enforce" (block messages)
   - `strict_rollout_enforce_at`: Timestamp when enforcement begins

2. **User-level settings:**
   - `send_restrictions`: Per-user configuration including:
     - `enabled`: Toggle for this user
     - `include_all_contacts`: Allow all contacts or restrict to authorized numbers
     - `authorized_numbers`: Whitelist of phone numbers
     - `allowed_instance_id` / `allowed_instance_ids`: Which instances user can send from
     - `prefix_agent_name`: Auto-prefix messages with agent name
     - `allow_unclaimed_chat_view`: View unclaimed chats
     - `allow_unclaimed_chat_send`: Send to unclaimed chats

### Enforcement Flow
**Entry Point:** `enforceStrictSendRestrictions()` (called from `SendOutgoingMessage()`)
**Execution Path:**
1. Load organization settings
2. Load user send restrictions
3. Check if strict mode is enabled
4. If enabled:
   - Verify contact is in authorized numbers list (if not include_all_contacts)
   - Verify instance is in allowed instances list
   - Check outbound mode (inbound_only blocks proactive outbound)
   - Check if chat is claimed (if allow_unclaimed_chat_send is false)
5. If violation detected:
   - In "audit" mode: log warning, allow message
   - In "enforce" mode: return `restrictedSendViolationError`, block message
6. Apply agent name prefix if configured

### Update User Send Restrictions
**Entry Point:** `PUT /api/users/{id}/send-restrictions` → `App.UpdateUserSendRestrictions()`
**Inputs:** send_restrictions JSON object
**Execution Path:**
1. Authorize with `requirePermission(users, write)`
2. Load user, verify org membership
3. Validate restriction settings
4. Update user.settings.send_restrictions
5. Return updated settings

### Get User Send Restrictions
**Entry Point:** `GET /api/users/{id}/send-restrictions` → `App.GetUserSendRestrictions()`

### Chat Claim Enforcement
**Execution Path:**
1. When sending message, check if chat is claimed
2. If chat is restricted and unclaimed:
   - Check if user can send to unclaimed chats
   - If not, return 403 with message "This chat is currently unassigned. Claim it before sending messages."
3. Agent-role users have chat-scoped visibility even with contacts:read permission

### Outbound Mode Enforcement
**"inbound_only" mode:**
- Users can only reply to inbound messages
- Proactive outbound messages are blocked
- Campaign messages may be restricted based on campaign_draft_only setting

**"mixed" mode:**
- Both inbound replies and proactive outbound allowed
- Standard permission checks apply

---

## 52. Agent Chat Visibility Restrictions

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/contacts_messaging.go`, `internal/handlers/chat_access_policy.go`

### Overview
Agent-role users have restricted visibility into chats based on assignment status and access policies.

### Restriction Logic
**Execution Path:**
1. `shouldRestrictChatVisibilityToAgentScope()` checks if user is agent role
2. If restricted, apply `applyAgentVisibleChatAccessFilter()`:
   - Only show chats assigned to the user
   - Only show public chats (is_public = true)
   - Only show chats where user is a collaborator
3. When listing contacts, filter query based on user's access scope
4. When sending messages, verify agent has access to the contact

### Agent Message Sending
**Execution Path:**
1. Agent attempts to send message
2. Load contact with agent-scoped query
3. If contact not found in agent's scope, return 404
4. Check if chat is closed — reject if closed
5. Check if chat is restricted and unclaimed — require claim first
6. Proceed with message send

---

## 53. Contact Account Resolution

**Source Files:** `internal/handlers/contact_account_resolution.go`

### Overview
Resolves the correct WhatsApp account for a contact when sending messages, considering instance assignments and account mappings.

### Execution Path
1. Check if contact has an associated instance
2. If instance exists, find account linked to that instance
3. If no instance, use account from request or contact's whatsapp_account field
4. Validate account belongs to same organization
5. Return resolved account or error

---

## 54. Contact Repair

**Source Files:** `internal/handlers/contact_repair.go`

### Overview
Repairs orphaned or inconsistent contact records, typically after data migrations or account changes.

### Execution Path
1. Scan for contacts with missing or invalid references
2. Fix orphaned contacts by reassigning to correct account/instance
3. Update contact metadata to reflect current state
4. Log repair actions

---

## 55. Contact User Deletions

**Source Files:** `internal/handlers/contact_user_deletions.go`

### Overview
Handles cleanup when a user is deleted, reassigning or archiving their contacts and chats.

### Execution Path
1. Find all contacts assigned to deleted user
2. Reassign to team lead or unassign
3. Update chat assignment records
4. Notify affected users via WebSocket
5. Dispatch webhook for contact reassigned events

---

## 56. Closed Chat Filters

**Source Files:** `internal/handlers/closed_chat_filters.go`

### Overview
Provides filtering capabilities for closed chats in the contact list view.

### Filter Options
- Closed by user
- Closed date range
- Close reason
- Rating status

### Execution Path
1. Apply closed_at IS NOT NULL filter
2. Apply additional filters based on query params
3. Join with users table for closed_by_name
4. Paginate and return

---

## 57. Chat Lifecycle Management

**Source Files:** `internal/handlers/chat_lifecycle.go`, `internal/handlers/chat_system_messages.go`

### Chat States
- **open**: Active chat, agents can send messages
- **closed**: Chat closed, read-only
- **pending**: Awaiting agent assignment

### State Transitions
1. **open → closed**: Agent or system closes chat
   - Set closed_at, closed_by_user_id
   - Optionally send auto-close message
   - Optionally request rating
   - Broadcast via WebSocket
   - Dispatch webhook

2. **closed → open**: Agent reopens chat
   - Clear closed_at, closed_by_user_id
   - Set status back to open
   - Broadcast via WebSocket

3. **open → pending**: Chat unassigned
   - Clear assigned_user_id
   - Set status to pending
   - Notify available agents

4. **pending → open**: Chat claimed
   - Set assigned_user_id
   - Set status to open
   - Notify claiming agent

### System Messages
**Execution Path:**
1. Create message with actor_type = "system"
2. Content describes lifecycle event
3. Save to messages table
4. Broadcast via WebSocket
5. No provider send (internal only)

---

## 58. Reply Preview Helpers

**Source Files:** `internal/handlers/reply_preview_helpers.go`

### Overview
Extracts and formats reply context for messages that reference other messages.

### Execution Path
1. When message has reply_to_message_id
2. Load referenced message
3. Extract preview data:
   - Content (truncated for long messages)
   - Message type
   - Sender info
   - Media info (if media message)
4. Return formatted preview for API response

---

## 59. Message Template Placeholders

**Source Files:** `internal/handlers/message_template_placeholders.go`

### Overview
Resolves template placeholders in messages with actual values from contacts, users, or custom data.

### Supported Placeholders
- `{{contact.name}}` - Contact name
- `{{contact.phone}}` - Contact phone number
- `{{user.name}}` - Agent/sender name
- `{{organization.name}}` - Organization name
- Custom placeholders from template params

### Execution Path
1. Parse message content for placeholder patterns
2. Load context data (contact, user, organization)
3. Replace placeholders with actual values
4. Handle missing values (leave as-is or use defaults)
5. Return resolved message content

---

## 60. Campaign Policy Enforcement

**Source Files:** `internal/handlers/campaign_policy.go`

### Overview
Enforces policies on campaign creation and execution, including rate limits, template requirements, and scheduling constraints.

### Policy Checks
1. **Template approval**: Campaign template must be APPROVED
2. **Account status**: WhatsApp account must be active
3. **Delay validation**: Min/max delay within acceptable range
4. **Recipient validation**: Campaign must have at least one recipient
5. **Schedule validation**: Scheduled time must be in the future
6. **Rate limiting**: Check organization campaign rate limits

### Execution Path
1. `validateCampaignForCreate()` called during campaign creation
2. `validateCampaignForStart()` called before starting campaign
3. Return specific error messages for each policy violation
4. Block campaign operation if policy check fails

---

## 61. Flows Helpers

**Source Files:** `internal/handlers/flows_helpers_test.go`

### Overview
Helper functions for WhatsApp Flows operations, including JSON validation and Meta API integration.

### Functions
- Validate Flow JSON schema
- Transform Flow JSON for Meta API
- Parse Flow response from Meta
- Generate Flow tokens for tracking

---

## 62. Group Message Helpers

**Source Files:** `internal/handlers/group_message_helpers.go`

### Overview
Handles group message detection and processing for WhatsApp group chats.

### Execution Path
1. Detect if incoming message is from a group (JID contains @g.us)
2. Extract group metadata (group ID, sender JID)
3. Create or update group contact record
4. Store sender info in message metadata
5. Apply group-specific chatbot rules (if configured)

---

## 63. Instance Name Validation

**Source Files:** `internal/handlers/instance_name_validation.go`

### Overview
Validates WhatsApp instance names for uniqueness and format.

### Validation Rules
1. Name must be non-empty
2. Name must be unique within organization
3. Name must match pattern (alphanumeric, hyphens, underscores)
4. Name length limits (min 2, max 50 characters)

### Execution Path
1. `normalizeInstanceName()` - trim, lowercase, remove invalid chars
2. `isInstanceNameTaken()` - query database for existing name
3. Return validation error if invalid

---

## 64. Instance Selector

**Source Files:** `internal/handlers/instance_selector.go`

### Overview
Selects the appropriate WhatsApp instance for outbound messages based on configuration, availability, and load.

### Selection Strategies
1. **Default instance**: Use organization's default instance
2. **Contact-assigned instance**: Use instance linked to contact
3. **Request-specified instance**: Use instance from API request
4. **Round-robin**: Distribute across available instances (future)

### Execution Path
1. `resolveOutboundInstance()` called from message send handlers
2. Check request-specified instance ID first
3. Fall back to contact's instance
4. Fall back to organization default
5. Validate instance is connected and healthy
6. Return instance or error with reason code

---

## 65. Password Policy

**Source Files:** `internal/handlers/password_policy.go`

### Overview
Enforces password strength requirements during registration and password changes.

### Policy Rules
1. Minimum length: 8 characters
2. At least one uppercase letter
3. At least one lowercase letter
4. At least one digit
5. At least one special character
6. Not in common password list

### Execution Path
1. `validatePasswordStrength()` called during registration and password change
2. Check each policy rule
3. Return specific error message for first violated rule
4. Reject password if any rule fails

---

## 66. Provider Guard

**Source Files:** `internal/handlers/provider_guard.go`

### Overview
Middleware that restricts certain endpoints to specific WhatsApp providers (Meta or WhatsMeow).

### Execution Path
1. `ProviderGuard("meta", handler)` wraps handler
2. Check configured provider in app config
3. If provider doesn't match, return 400 with "Feature not available for current provider"
4. If provider matches, call wrapped handler

### Protected Features
- Templates (Meta only)
- WhatsApp Flows (Meta only)
- Catalogs (Meta only)

---

## 67. Reason Codes

**Source Files:** `internal/handlers/reason_codes.go`

### Overview
Provides standardized reason codes for API error responses, enabling frontend to handle errors programmatically.

### Common Reason Codes
- `instance_not_found`: Specified instance doesn't exist
- `instance_not_connected`: Instance is not connected
- `instance_not_allowed`: User not permitted to use this instance
- `chat_unclaimed`: Chat needs to be claimed before sending
- `chat_closed`: Chat is closed and read-only
- `restriction_violation`: Send restriction policy violated

### Execution Path
1. Create error with reason code using `asInstanceSelectionError()`
2. Return error envelope with `reason_code` field
3. Frontend uses reason code to display appropriate UI

---

## 68. Security Headers & CSRF Protection

**Source Files:** `internal/middleware/security.go`, `internal/middleware/csrf.go`

### Security Headers
**Applied to all responses:**
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`

### CSRF Protection
**Execution Path:**
1. Generate CSRF token on login
2. Store token in HTTP-only cookie (`whm_csrf`)
3. For mutating requests (POST/PUT/DELETE/PATCH):
   - Extract token from `X-CSRF-Token` header
   - Compare with cookie value
   - Reject if mismatch (403 Forbidden)
4. Safe methods (GET/HEAD/OPTIONS) skip CSRF check

---

## 69. Request Logging & Recovery

**Source Files:** `internal/middleware/logger.go`, `internal/middleware/recovery.go`

### Request Logger
**Execution Path:**
1. Log request method, path, remote address
2. Log response status and duration
3. Include request ID for tracing
4. Log user ID if authenticated

### Recovery Middleware
**Execution Path:**
1. Defer panic recovery at start of request
2. If panic occurs:
   - Log stack trace
   - Return 500 Internal Server Error
   - Don't expose panic details to client
3. Continue serving other requests

---

## 70. SSRF-Safe Dialer

**Source Files:** `internal/handlers/helpers.go`

### Overview
Prevents Server-Side Request Forgery (SSRF) attacks by blocking requests to internal IP ranges.

### Blocked Ranges
- 127.0.0.0/8 (loopback)
- 10.0.0.0/8 (private)
- 172.16.0.0/12 (private)
- 192.168.0.0/16 (private)
- 169.254.0.0/16 (link-local)
- ::1 (IPv6 loopback)
- fc00::/7 (IPv6 unique local)
- fe80::/10 (IPv6 link-local)

### Execution Path
1. `SSRFSafeDialer()` creates custom HTTP transport dialer
2. Before connecting, resolve target hostname
3. Check if resolved IP is in blocked ranges
4. If blocked, return connection refused error
5. If allowed, proceed with connection

---

## 71. Cache System

**Source Files:** `internal/handlers/cache.go`

### Cached Data
1. **WhatsApp Accounts**: Lookup by phone_number_id
2. **Role Permissions**: Permission lists by role ID
3. **Chatbot Settings**: Settings by organization ID
4. **Organization Settings**: Settings by organization ID

### Cache Operations
1. **Get**: Check Redis cache first
2. **Miss**: Load from database, store in cache with TTL
3. **Hit**: Return cached value
4. **Invalidate**: Delete cache key on update

### TTL Settings
- Accounts: 5 minutes
- Role permissions: 10 minutes
- Chatbot settings: 5 minutes
- Organization settings: 5 minutes

---

## 72. Cookie Management

**Source Files:** `internal/handlers/cookies.go`

### Auth Cookies
- `whm_access`: Access token (JWT), HTTP-only, Secure, SameSite=Strict
- `whm_refresh`: Refresh token (JWT), HTTP-only, Secure, SameSite=Strict
- `whm_csrf`: CSRF token, HTTP-only, Secure, SameSite=Strict

### Cookie Operations
1. `setAuthCookies()`: Set access, refresh, and CSRF cookies
2. `clearAuthCookies()`: Clear all auth cookies (set expired)
3. Cookie domain and path derived from request
4. Secure flag always enabled in production

---

## 73. JWT Secret Management

**Source Files:** `internal/handlers/jwt_secret.go`

### Overview
Manages JWT signing key with support for environment variable or config file.

### Execution Path
1. `jwtSecretBytes()` retrieves signing key
2. Check environment variable `WHATOMATE_JWT_SECRET` first
3. Fall back to config file value
4. Validate key meets minimum length requirement
5. Return key bytes for JWT signing

---

## 74. WhatsApp Client (Meta)

**Source Files:** `pkg/whatsapp/client.go`

### Overview
HTTP client for Meta WhatsApp Business Cloud API.

### Supported Operations
1. Send text message
2. Send media message (image, video, audio, document)
3. Send template message
4. Send interactive message (buttons, list, CTA URL)
5. Send location message
6. Send contact message
7. Mark message as read
8. Send typing indicator
9. Upload media
10. Download media
11. Fetch templates
12. Create/update/delete templates
13. Submit template for approval
14. Fetch Flows
15. Create/update/delete Flows
16. Publish/deprecate Flows
17. Fetch catalogs and products
18. Fetch business profile
19. Update business profile
20. Fetch analytics

### Execution Path
1. Build API URL from base URL and endpoint
2. Set Authorization header with access token
3. Build request body
4. Send HTTP request
5. Parse response
6. Handle errors (rate limits, invalid credentials, etc.)
7. Return result or error

---

## 75. Message Provider Abstraction

**Source Files:** `pkg/provider/provider.go`, `pkg/whatsapp/meta_adapter.go`, `pkg/whatsmeow/adapter.go`

### Overview
Provider interface abstracts differences between Meta and WhatsMeow providers.

### Interface Methods
1. `SendMessage()`: Send message to contact
2. `SendMediaMessage()`: Send media message
3. `SendTemplateMessage()`: Send template message
4. `MarkRead()`: Mark message as read
5. `SendTyping()`: Send typing indicator

### Meta Adapter
- Routes calls to Meta WhatsApp Client
- Handles Meta-specific error codes
- Transforms response formats

### WhatsMeow Adapter
- Routes calls to WhatsMeow Connection Manager
- Handles per-instance queuing
- Manages rate limiting per instance
- Handles WhatsMeow-specific errors

---

## 76. WhatsMeow Connection Manager

**Source Files:** `pkg/whatsmeow/manager.go`

### Overview
Manages WhatsApp Web connections for multiple instances.

### Connection Lifecycle
1. **Create**: Initialize new instance, generate QR code
2. **Connect**: Start WebSocket connection to WhatsApp
3. **Authenticated**: Session established, ready to send
4. **Disconnected**: Connection lost, attempt reconnect
5. **Logout**: Session terminated, need new QR code

### Connection Management
1. `GetClient()`: Get connected client for instance
2. `Connect()`: Start new connection
3. `Disconnect()`: Close connection
4. `Reconnect()`: Reconnect after disconnect
5. `ReconnectAll()`: Reconnect all active instances
6. `AutoConnectLinkedInstancesOnFirstRun()`: Auto-connect on startup

### Event Handling
1. **Message received**: Process inbound message, enqueue for media download
2. **Receipt received**: Update message status
3. **Presence update**: Update contact presence
4. **Connection status**: Broadcast status change via WebSocket
5. **QR code received**: Cache QR code for API retrieval

### Queue Depth Management
1. Each instance has per-instance message queue
2. Queue depth tracked and reported via API
3. Depth observer callback updates instance record
4. Rate limiting based on queue depth

---

## 77. WhatsMeow Queue Manager

**Source Files:** `pkg/whatsmeow/queue.go`

### Overview
Per-instance message queue for WhatsMeow provider with rate limiting.

### Queue Operations
1. **Enqueue**: Add message to instance queue
2. **Dequeue**: Get next message to send
3. **Depth**: Get current queue depth
4. **Wait**: Block until queue has capacity

### Rate Limiting
1. Configurable messages per minute per instance
2. Adaptive delay based on queue depth
3. Priority queue for high-priority messages

---

## 78. Redis Queue System

**Source Files:** `internal/queue/queue.go`, `internal/queue/consumer.go`, `internal/queue/publisher.go`

### Queue Types
1. **Campaign Queue**: Campaign message jobs
2. **Inbound Media Queue**: Media download jobs
3. **Pub/Sub Channels**: Campaign stats, notifications

### Job Types
1. **RecipientJob**: Campaign recipient message send
2. **InboundMediaJob**: Download media from Meta/WhatsMeow
3. **CampaignStatsUpdate**: Campaign progress update

### Consumer Operations
1. `Consume()`: Start consuming jobs from queue
2. `HandleJob()`: Process individual job
3. `Ack()`: Acknowledge successful job
4. `Nack()`: Negative acknowledge, requeue or dead-letter
5. `Retry()`: Retry failed job with backoff

### Publisher Operations
1. `Publish()`: Add job to queue
2. `PublishCampaignStats()`: Broadcast stats update

---

## 79. Campaign Stats Subscriber

**Source Files:** `internal/queue/subscriber.go`

### Overview
Subscribes to Redis pub/sub for campaign stats updates and broadcasts via WebSocket.

### Execution Path
1. Subscribe to campaign stats channel
2. On message received:
   - Parse stats update
   - Broadcast to organization via WebSocket
   - Log update
3. On disconnect:
   - Auto-resubscribe
   - Reconnect to channel

---

## 80. Database Migrations

**Source Files:** `internal/database/migrations.go`, `internal/handlers/migration_handler.go`

### Overview
Manages database schema migrations using GORM AutoMigrate.

### Migration Process
1. `RunMigrationWithProgress()`:
   - Run GORM AutoMigrate
   - Create default admin user if configured
   - Create default roles for organizations
   - Create default chatbot settings
   - Report migration progress
2. Migrations run on server startup with `-migrate` flag
3. Or triggered via API by super admin

### Default Admin Creation
1. Check if admin user exists
2. If not, create from config:
   - `default_admin.email`
   - `default_admin.password`
   - `default_admin.full_name`
3. Create organization for admin
4. Create default roles
5. Add admin to organization

---

## 81. Encryption System

**Source Files:** `internal/crypto/crypto.go`, `internal/crypto/migration.go`

### Overview
Encrypts sensitive data (access tokens, API keys, secrets) in database.

### Encryption Versions
1. **enc**: Original encryption format
2. **enc2**: Second generation format
3. **enc3**: Current format (AES-256-GCM)

### Encryption Process
1. `Encrypt()`: Encrypt plaintext with AES-256-GCM
2. `Decrypt()`: Decrypt ciphertext
3. Prefix encrypted values with `enc3:` for identification
4. Key derived from `app.encryption_key` config

### Crypto Migration
1. Scan database for legacy encrypted values (enc:, enc2:)
2. Decrypt with old format
3. Re-encrypt with enc3 format
4. Update records in batches
5. Report migration summary

### Encrypted Fields
- WhatsApp account access tokens
- WhatsApp account webhook verify tokens
- SSO client secrets
- Chatbot AI API keys
- Webhook secrets
- Custom action headers

---

## 82. Contact Utilities

**Source Files:** `internal/contactutil/contact.go`

### GetOrCreateContact
**Execution Path:**
1. Query contact by phone number and organization
2. If found, return existing contact
3. If not found:
   - Create new contact record
   - Set phone number, profile name
   - Set status to open
   - Return new contact and isNewContact=true
4. Update profile name if provided and different

---

## 83. Template Utilities

**Source Files:** `internal/templateutil/template.go`

### Overview
Helper functions for template rendering and placeholder resolution.

### Functions
1. `ResolvePlaceholders()`: Replace placeholders in template
2. `ValidateTemplateSyntax()`: Check template for valid placeholders
3. `ExtractPlaceholders()`: List all placeholders in template

---

## 84. WebSocket Message Types

**Source Files:** `internal/websocket/hub.go`, `internal/websocket/messages.go`

### Message Types
1. **message**: New message received
2. **message_status**: Message status updated
3. **contact_created**: New contact created
4. **contact_assigned**: Contact assigned to user
5. **chat_closed**: Chat closed
6. **chat_reopened**: Chat reopened
7. **campaign_stats_update**: Campaign progress update
8. **instance_status**: Instance connection status changed
9. **notification**: New notification
10. **typing**: Typing indicator
11. **presence**: Contact presence update
12. **instance_reconnect_failed**: Instance reconnection failed

### Message Format
```json
{
  "type": "message",
  "payload": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## 85. Frontend Embedded Build

**Source Files:** `internal/frontend/embed.go`, `frontend/`

### Overview
Frontend is built with React/Vite and embedded into Go binary.

### Build Process
1. Frontend built with `npm run build`
2. Output copied to `internal/frontend/dist/`
3. Go `embed` directive includes dist files
4. Server serves embedded files at runtime

### Development Mode
1. Frontend dev server runs separately
2. API proxy configured for development
3. CORS enabled for dev server origin

---

## 86. Configuration System

**Source Files:** `internal/config/config.go`

### Configuration Sections
1. **app**: Application settings (name, version, environment, debug, encryption_key)
2. **server**: Server settings (host, port, read/write timeouts, allowed_origins, max_request_body_size)
3. **database**: PostgreSQL connection (host, port, user, password, dbname, ssl_mode)
4. **redis**: Redis connection (host, port, password, db)
5. **whatsapp**: WhatsApp settings (provider, base_url, webhook_verify_token)
6. **whatsmeow**: WhatsMeow-specific settings (queue_depth, rate_limit)
7. **jwt**: JWT settings (secret, access_token_ttl, refresh_token_ttl)
8. **default_admin**: Default admin user (email, password, full_name)
9. **storage**: File storage settings (local_path)
10. **rate_limit**: Rate limiting settings (enabled, per-user, per-IP limits)

### Config Loading
1. Load from TOML file
2. Override with environment variables
3. Validate required fields
4. Set defaults for optional fields

---

## 87. Model Layer

**Source Files:** `internal/models/`

### Key Models
1. **User**: User accounts with roles and permissions
2. **Organization**: Multi-tenant organizations
3. **CustomRole**: Custom roles with permissions
4. **Permission**: Resource:action permission pairs
5. **WhatsAppAccount**: Meta WhatsApp Business accounts
6. **WhatsAppInstance**: WhatsMeow instances
7. **Contact**: WhatsApp contacts
8. **Message**: WhatsApp messages (inbound/outbound)
9. **BulkMessageCampaign**: Campaign definitions
10. **BulkMessageRecipient**: Campaign recipients
11. **Template**: WhatsApp message templates
12. **ChatbotSettings**: Chatbot configuration
13. **KeywordRule**: Chatbot keyword rules
14. **ChatbotFlow**: Chatbot conversation flows
15. **AIContext**: AI context definitions
16. **AgentTransfer**: Chatbot-to-agent transfers
17. **CannedResponse**: Pre-written responses
18. **Tag**: Contact tags
19. **Team**: User teams
20. **Webhook**: Outbound webhook definitions
21. **CustomAction**: Custom HTTP actions
22. **ConversationNote**: Chat notes
23. **ActivityLog**: Audit log entries
24. **Widget**: Custom analytics widgets
25. **LeadRequest**: Public lead submissions
26. **Notification**: User notifications
27. **UserOrganization**: User-org membership records
28. **TeamMember**: Team membership records
29. **ContactCollaborator**: Contact collaboration records
30. **SSOProvider**: SSO provider configurations

---

## 88. Middleware Chain

**Source Files:** `internal/middleware/`

### Request Processing Order
1. **CORS Wrapper**: Handle CORS headers (fasthttp level)
2. **Security Headers**: Add security headers
3. **Request Logger**: Log request details
4. **Recovery**: Catch panics
5. **CSRF Protection**: Validate CSRF tokens
6. **Activity Log Middleware**: Log significant actions
7. **Auth Middleware**: Validate JWT/API key (for protected routes)
8. **Role-based Access**: Check permissions (handler level)
9. **Provider Guard**: Check provider compatibility (handler level)
10. **Rate Limiting**: Check rate limits (endpoint level)

---

## 89. Error Handling Patterns

### Error Envelope Format
```json
{
  "error": {
    "message": "Human-readable error message",
    "code": "machine_readable_code",
    "field": "field_name_if_validation_error"
  }
}
```

### Common HTTP Status Codes
- **400**: Bad Request (validation errors)
- **401**: Unauthorized (auth failures)
- **403**: Forbidden (permission denied)
- **404**: Not Found (resource not found)
- **409**: Conflict (duplicate, closed chat)
- **413**: Payload Too Large
- **429**: Too Many Requests (rate limited)
- **500**: Internal Server Error

### Error Handling Strategy
1. Validate input early, return 400
2. Check auth, return 401 if invalid
3. Check permissions, return 403 if denied
4. Check resource exists, return 404 if not
5. Check business rules, return 409 if violated
6. Execute operation, return 500 on failure
7. Never expose internal errors to client

---

## 90. Testing Infrastructure

### Test Files
- Unit tests: `*_test.go` files alongside source
- E2E tests: `frontend/e2e/` with Playwright
- Integration tests: Test handlers with test database

### Test Helpers
- `testhelpers_test.go`: Common test utilities
- `stubs.go`: Stub implementations for testing
- `ApiHelper`: E2E API test helper (TypeScript)

### Coverage Reports
- Multiple coverage files for different packages
- `coverage.out`: Main coverage report
- `coverage_*.out`: Package-specific coverage

---

## 91. App Configuration Endpoint

**Source Files:** `internal/handlers/config_handler.go`

### Get App Config
**Entry Point:** `GET /api/config` → `App.GetAppConfig()`
**Execution Path:**
1. Read configured WhatsApp provider (default: "meta")
2. Determine feature flags based on provider:
   - Meta-only: templates, flows, catalog, business_profile, meta_insights
   - Both providers: campaigns
3. Return config with provider and features
**Outputs:** `{whatsapp_provider, features: {templates, flows, catalog, business_profile, campaigns, meta_insights}}`
**Dependencies:** None (no auth required)
**Purpose:** Frontend uses this to conditionally show/hide features

---

## 92. User Settings & Chat Background

**Source Files:** `internal/handlers/users.go`

### Update Current User Settings
**Entry Point:** `PUT /api/me/settings` → `App.UpdateCurrentUserSettings()`
**Inputs:** email_notifications, new_message_alerts, campaign_updates, notification_sound, chat_background
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Apply partial updates to settings JSONB:
   - Toggle notification preferences
   - Set notification sound (notification1, notification2, notification)
   - Update chat background (preset or custom)
4. Validate chat background:
   - Preset: must be in allowed list (aurora-veil, sunset-dunes, paper-garden, linen-grid, dot-orbit, ripple-lines)
   - Custom: validate MIME type (jpeg, png, webp), size limit (5MB)
5. Save and return updated user

### Upload User Chat Background
**Entry Point:** `POST /api/me/chat-background` → `App.UploadCurrentUserChatBackground()`
**Execution Path:**
1. Receive multipart file upload
2. Validate file size (max 5MB)
3. Validate MIME type (jpeg, png, webp)
4. Save to storage with user-specific path
5. Update user settings with custom_asset_id
6. Return background URL

### Get User Chat Background
**Entry Point:** `GET /api/me/chat-background` → `App.GetCurrentUserChatBackground()`

---

## 93. Availability Management

**Source Files:** `internal/handlers/users.go`

### Update Availability
**Entry Point:** `PUT /api/me/availability` → `App.UpdateAvailability()`
**Inputs:** is_available (boolean)
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Update is_available field
4. Broadcast availability change via WebSocket
5. Return updated user
**Purpose:** Controls whether user appears available for chat assignment

---

## 94. Change Password

**Source Files:** `internal/handlers/users.go`

### Change Password
**Entry Point:** `PUT /api/me/password` → `App.ChangePassword()`
**Inputs:** current_password, new_password
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Verify current_password with bcrypt
4. Validate new_password strength via `validatePasswordStrength()`
5. Hash new password with bcrypt
6. Update password_hash
7. Invalidate all refresh tokens (force re-login)
8. Return success

---

## 95. Contact Phone Start (WhatsMeow)

**Source Files:** `internal/handlers/contacts_chat_start.go`

### Overview
Allows agents to start a new direct chat with a phone number via WhatsMeow, without waiting for inbound message.

### Execution Path
1. Agent provides phone number
2. `ResolveDirectContact()` called:
   - Normalize phone number (strip non-digits except +)
   - Verify phone is on WhatsApp via `client.IsOnWhatsApp()`
   - Get canonical phone (JID user)
   - Resolve business name if verified
3. If phone not on WhatsApp, return error
4. Get or create contact with resolved details
5. Open chat for contact
**Dependencies:** Connected WhatsMeow instance
**Edge Cases:** Phone not on WhatsApp; instance disconnected; invalid phone format

---

## 96. Interactive Messages

**Source Files:** `internal/handlers/contacts_messaging.go`

### Overview
Supports sending interactive WhatsApp messages (buttons, lists, CTA URL).

### Interactive Types
1. **Button**: Up to 3 quick-reply buttons
2. **List**: Single-select list with sections
3. **CTA URL**: Call-to-action button with URL

### Execution Path
1. Parse `interactive` field from request
2. Validate interactive type and content
3. Build `OutgoingMessageRequest` with interactive fields
4. Route through `SendOutgoingMessage()`
5. Provider sends interactive message via WhatsApp API
**Dependencies:** Meta provider (WhatsMeow has limited interactive support)

---

## 97. Typing Presence

**Source Files:** `internal/handlers/contacts_messaging.go`

### Send Typing Presence
**Entry Point:** `POST /api/contacts/{id}/typing` → `App.SendTypingPresence()`
**Inputs:** state (composing/paused), instance_id
**Execution Path:**
1. Load contact and resolve instance
2. Route through provider:
   - Meta: Send typing indicator via Cloud API
   - WhatsMeow: Send chat presence update
3. Return success
**Purpose:** Shows "typing..." indicator to contact

---

## 98. Agent Role Chat Scoping

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/chat_access_policy.go`

### Overview
Agent-role users see only chats they have access to, not all organization contacts.

### Access Rules
1. **Assigned chats**: Chats where assigned_user_id = current user
2. **Public chats**: Chats where is_public = true
3. **Collaborator chats**: Chats where user is a collaborator
4. **Team-shared chats**: Chats shared with user's team (if configured)

### Execution Path (List Contacts)
1. Check if user is agent role via `shouldRestrictChatVisibilityToAgentScope()`
2. If restricted:
   - Apply `applyAgentVisibleChatAccessFilter()` to query
   - Filter by assigned_user_id OR is_public OR collaborator
3. If not restricted (admin/manager):
   - Show all contacts in organization
4. Apply additional filters (search, tags, status)
5. Paginate and return

### Execution Path (Send Message)
1. Load contact with agent-scoped query
2. If not found, return 404 (agent can't see this contact)
3. Check chat status (closed = read-only)
4. Check chat claim status (unclaimed = must claim first)
5. Proceed with send

---

## 99. Organization Outbound Mode

**Source Files:** `internal/handlers/send_restriction_policy.go`

### Overview
Organization-level setting that controls whether users can send outbound messages proactively.

### Modes
1. **inbound_only**: Users can only reply to inbound messages
2. **mixed**: Users can send both replies and proactive messages

### Enforcement
**Execution Path:**
1. Check organization outbound_mode setting
2. If inbound_only:
   - Check if message is a reply (reply_to_message_id set)
   - If not a reply, block with "Outbound messages are disabled. You can only reply to incoming messages."
3. If mixed: allow all messages (subject to other restrictions)
4. System/chatbot messages may be exempt based on `strict_sending_apply_to_system`

---

## 100. Strict Rollout Mode

**Source Files:** `internal/handlers/send_restriction_policy.go`

### Overview
Gradual enforcement mode for send restrictions, allowing organizations to transition from permissive to strict policies.

### Rollout Phases
1. **Audit mode**: Log violations but don't block messages
   - Warnings sent to admins
   - Analytics track violation frequency
2. **Enforce mode**: Block messages that violate restrictions
   - Activated at `strict_rollout_enforce_at` timestamp
   - Users receive error messages

### Execution Path
1. Check `strict_rollout_mode` setting
2. If "audit":
   - Log violation
   - Allow message to proceed
   - Notify admins of violation
3. If "enforce" and current time >= `strict_rollout_enforce_at`:
   - Block message
   - Return error to user
4. If "enforce" but before enforcement date:
   - Treat as audit mode
   - Warn about upcoming enforcement

---

## 91. App Configuration Endpoint

**Source Files:** `internal/handlers/config_handler.go`

### Get App Config
**Entry Point:** `GET /api/config` → `App.GetAppConfig()`
**Execution Path:**
1. Read configured WhatsApp provider (default: "meta")
2. Determine feature flags based on provider:
   - Meta-only: templates, flows, catalog, business_profile, meta_insights
   - Both providers: campaigns
3. Return config with provider and features
**Outputs:** `{whatsapp_provider, features: {templates, flows, catalog, business_profile, campaigns, meta_insights}}`
**Dependencies:** None (no auth required)
**Purpose:** Frontend uses this to conditionally show/hide features

---

## 92. User Settings & Chat Background

**Source Files:** `internal/handlers/users.go`

### Update Current User Settings
**Entry Point:** `PUT /api/me/settings` → `App.UpdateCurrentUserSettings()`
**Inputs:** email_notifications, new_message_alerts, campaign_updates, notification_sound, chat_background
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Apply partial updates to settings JSONB:
   - Toggle notification preferences
   - Set notification sound (notification1, notification2, notification)
   - Update chat background (preset or custom)
4. Validate chat background:
   - Preset: must be in allowed list (aurora-veil, sunset-dunes, paper-garden, linen-grid, dot-orbit, ripple-lines)
   - Custom: validate MIME type (jpeg, png, webp), size limit (5MB)
5. Save and return updated user

### Upload User Chat Background
**Entry Point:** `POST /api/me/chat-background` → `App.UploadCurrentUserChatBackground()`
**Execution Path:**
1. Receive multipart file upload
2. Validate file size (max 5MB)
3. Validate MIME type (jpeg, png, webp)
4. Save to storage with user-specific path
5. Update user settings with custom_asset_id
6. Return background URL

### Get User Chat Background
**Entry Point:** `GET /api/me/chat-background` → `App.GetCurrentUserChatBackground()`

---

## 93. Availability Management

**Source Files:** `internal/handlers/users.go`

### Update Availability
**Entry Point:** `PUT /api/me/availability` → `App.UpdateAvailability()`
**Inputs:** is_available (boolean)
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Update is_available field
4. Broadcast availability change via WebSocket
5. Return updated user
**Purpose:** Controls whether user appears available for chat assignment

---

## 94. Change Password

**Source Files:** `internal/handlers/users.go`

### Change Password
**Entry Point:** `PUT /api/me/password` → `App.ChangePassword()`
**Inputs:** current_password, new_password
**Execution Path:**
1. Extract user_id from context
2. Load user record
3. Verify current_password with bcrypt
4. Validate new_password strength via `validatePasswordStrength()`
5. Hash new password with bcrypt
6. Update password_hash
7. Invalidate all refresh tokens (force re-login)
8. Return success

---

## 95. Contact Phone Start (WhatsMeow)

**Source Files:** `internal/handlers/contacts_chat_start.go`

### Overview
Allows agents to start a new direct chat with a phone number via WhatsMeow, without waiting for inbound message.

### Execution Path
1. Agent provides phone number
2. `ResolveDirectContact()` called:
   - Normalize phone number (strip non-digits except +)
   - Verify phone is on WhatsApp via `client.IsOnWhatsApp()`
   - Get canonical phone (JID user)
   - Resolve business name if verified
3. If phone not on WhatsApp, return error
4. Get or create contact with resolved details
5. Open chat for contact
**Dependencies:** Connected WhatsMeow instance
**Edge Cases:** Phone not on WhatsApp; instance disconnected; invalid phone format

---

## 96. Interactive Messages

**Source Files:** `internal/handlers/contacts_messaging.go`

### Overview
Supports sending interactive WhatsApp messages (buttons, lists, CTA URL).

### Interactive Types
1. **Button**: Up to 3 quick-reply buttons
2. **List**: Single-select list with sections
3. **CTA URL**: Call-to-action button with URL

### Execution Path
1. Parse `interactive` field from request
2. Validate interactive type and content
3. Build `OutgoingMessageRequest` with interactive fields
4. Route through `SendOutgoingMessage()`
5. Provider sends interactive message via WhatsApp API
**Dependencies:** Meta provider (WhatsMeow has limited interactive support)

---

## 97. Typing Presence

**Source Files:** `internal/handlers/contacts_messaging.go`

### Send Typing Presence
**Entry Point:** `POST /api/contacts/{id}/typing` → `App.SendTypingPresence()`
**Inputs:** state (composing/paused), instance_id
**Execution Path:**
1. Load contact and resolve instance
2. Route through provider:
   - Meta: Send typing indicator via Cloud API
   - WhatsMeow: Send chat presence update
3. Return success
**Purpose:** Shows "typing..." indicator to contact

---

## 98. Agent Role Chat Scoping

**Source Files:** `internal/handlers/contacts.go`, `internal/handlers/chat_access_policy.go`

### Overview
Agent-role users see only chats they have access to, not all organization contacts.

### Access Rules
1. **Assigned chats**: Chats where assigned_user_id = current user
2. **Public chats**: Chats where is_public = true
3. **Collaborator chats**: Chats where user is a collaborator
4. **Team-shared chats**: Chats shared with user's team (if configured)

### Execution Path (List Contacts)
1. Check if user is agent role via `shouldRestrictChatVisibilityToAgentScope()`
2. If restricted:
   - Apply `applyAgentVisibleChatAccessFilter()` to query
   - Filter by assigned_user_id OR is_public OR collaborator
3. If not restricted (admin/manager):
   - Show all contacts in organization
4. Apply additional filters (search, tags, status)
5. Paginate and return

### Execution Path (Send Message)
1. Load contact with agent-scoped query
2. If not found, return 404 (agent can't see this contact)
3. Check chat status (closed = read-only)
4. Check chat claim status (unclaimed = must claim first)
5. Proceed with send

---

## 99. Organization Outbound Mode

**Source Files:** `internal/handlers/send_restriction_policy.go`

### Overview
Organization-level setting that controls whether users can send outbound messages proactively.

### Modes
1. **inbound_only**: Users can only reply to inbound messages
2. **mixed**: Users can send both replies and proactive messages

### Enforcement
**Execution Path:**
1. Check organization outbound_mode setting
2. If inbound_only:
   - Check if message is a reply (reply_to_message_id set)
   - If not a reply, block with "Outbound messages are disabled. You can only reply to incoming messages."
3. If mixed: allow all messages (subject to other restrictions)
4. System/chatbot messages may be exempt based on `strict_sending_apply_to_system`

---

## 100. Strict Rollout Mode

**Source Files:** `internal/handlers/send_restriction_policy.go`

### Overview
Gradual enforcement mode for send restrictions, allowing organizations to transition from permissive to strict policies.

### Rollout Phases
1. **Audit mode**: Log violations but don't block messages
   - Warnings sent to admins
   - Analytics track violation frequency
2. **Enforce mode**: Block messages that violate restrictions
   - Activated at `strict_rollout_enforce_at` timestamp
   - Users receive error messages

### Execution Path
1. Check `strict_rollout_mode` setting
2. If "audit":
   - Log violation
   - Allow message to proceed
   - Notify admins of violation
3. If "enforce" and current time >= `strict_rollout_enforce_at`:
   - Block message
   - Return error to user
4. If "enforce" but before enforcement date:
   - Treat as audit mode
   - Warn about upcoming enforcement
