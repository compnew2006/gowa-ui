# Whatomate API Endpoints

This document lists all available API endpoints for the Whatomate platform.

## Base URL
The API base URL is typically `http://localhost:8080` (or as configured in `config.toml`).

## Authentication
Most endpoints require authentication. You can use either a JWT token in the `Authorization: Bearer <token>` header or an API key in the `X-API-Key: <key>` header.

---

## Health & System
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Basic health check |
| GET | `/ready` | Readiness check |
| GET | `/ws` | WebSocket connection (auth via message-based flow) |
| GET | `/api/config` | Get application configuration (provider & feature flags) |

## Authentication & SSO
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | User login |
| POST | `/api/auth/register` | User registration |
| POST | `/api/auth/refresh` | Refresh JWT token |
| POST | `/api/auth/logout` | User logout |
| POST | `/api/auth/switch-org` | Switch active organization |
| GET | `/api/auth/ws-token` | Get a short-lived token for WebSocket authentication |
| POST | `/api/auth/register/invite` | Generate a signed registration invite (Org admin only) |
| GET | `/api/auth/sso/providers` | List available SSO providers |
| GET | `/api/auth/sso/{provider}/init` | Initialize SSO flow |
| GET | `/api/auth/sso/{provider}/callback` | SSO callback handler |

## Current User
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/me` | Get current user profile |
| PUT | `/api/me/settings` | Update user settings |
| GET | `/api/me/organizations` | List organizations the user belongs to |
| PUT | `/api/me/password` | Change user password |
| PUT | `/api/me/availability` | Update user availability status |
| GET | `/api/me/chat-background` | Get user's custom chat background |
| POST | `/api/me/chat-background` | Upload a custom chat background |

## Organizations & Teams
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/organizations` | List organizations (Admin) |
| POST | `/api/organizations` | Create an organization |
| DELETE | `/api/organizations/{id}` | Delete an organization |
| GET | `/api/organizations/current` | Get current organization details |
| GET | `/api/organizations/members` | List members of the current organization |
| POST | `/api/organizations/members` | Add a member to the organization |
| PUT | `/api/organizations/members/{member_id}` | Update organization member role |
| DELETE | `/api/organizations/members/{member_id}` | Remove member from organization |
| GET | `/api/teams` | List teams |
| POST | `/api/teams` | Create a team |
| GET | `/api/teams/{id}` | Get team details |
| PUT | `/api/teams/{id}` | Update team details |
| DELETE | `/api/teams/{id}` | Delete a team |
| GET | `/api/teams/{id}/members` | List team members |
| POST | `/api/teams/{id}/members` | Add member to team |
| DELETE | `/api/teams/{id}/members/{member_user_id}` | Remove member from team |

## User Management (Admin)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create a new user |
| GET | `/api/users/{id}` | Get user details |
| PUT | `/api/users/{id}` | Update user details |
| DELETE | `/api/users/{id}` | Delete a user |
| GET | `/api/users/{id}/send-restrictions` | Get user's message sending restrictions |
| PUT | `/api/users/{id}/send-restrictions` | Update user's message sending restrictions |
| GET | `/api/roles` | List roles |
| POST | `/api/roles` | Create a role |
| GET | `/api/roles/{id}` | Get role details |
| PUT | `/api/roles/{id}` | Update role details |
| DELETE | `/api/roles/{id}` | Delete a role |
| GET | `/api/permissions` | List all available permissions |
| GET | `/api/api-keys` | List API keys |
| POST | `/api/api-keys` | Create an API key |
| DELETE | `/api/api-keys/{id}` | Delete an API key |

## Contacts & Chats
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/contacts` | List contacts |
| POST | `/api/contacts` | Create a contact |
| GET | `/api/contacts/{id}` | Get contact details |
| PUT | `/api/contacts/{id}` | Update contact details |
| DELETE | `/api/contacts/{id}` | Delete contact |
| POST | `/api/contacts/{id}/soft-delete` | Soft delete contact for user |
| PUT | `/api/contacts/{id}/assign` | Assign contact to a user/team |
| GET | `/api/contacts/{id}/collaborators` | List contact collaborators |
| POST | `/api/contacts/{id}/collaborators` | Invite a collaborator |
| PUT | `/api/contacts/{id}/collaborators/{user_id}/accept` | Accept collaboration |
| PUT | `/api/contacts/{id}/collaborators/{user_id}/decline` | Decline collaboration |
| DELETE | `/api/contacts/{id}/collaborators/{user_id}` | Remove collaborator |
| PUT | `/api/contacts/{id}/tags` | Update contact tags |
| GET | `/api/contacts/{id}/session-data` | Get contact session data |
| GET | `/api/chats` | List chats (alias for ListContacts) |
| PUT | `/api/chats/{id}/claim` | Claim an unassigned chat |
| PUT | `/api/chats/{id}/close` | Close a chat session |
| PUT | `/api/chats/{id}/reopen` | Reopen a closed chat |
| PUT | `/api/chats/{id}/public` | Set chat public/private |
| GET | `/api/chats/{id}/messages` | Get messages for a chat |
| GET | `/api/contacts/{id}/notes` | List notes for a conversation |
| POST | `/api/contacts/{id}/notes` | Create a conversation note |
| PUT | `/api/contacts/{id}/notes/{note_id}` | Update a note |
| DELETE | `/api/contacts/{id}/notes/{note_id}` | Delete a note |

## Messaging
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/contacts/{id}/messages` | Send a text message to a contact |
| POST | `/api/messages` | Send a text message (Legacy/Bulk) |
| POST | `/api/messages/template` | Send a template message |
| POST | `/api/messages/media` | Send a media message |
| POST | `/api/contacts/{id}/typing` | Send typing indicator |
| POST | `/api/contacts/{id}/messages/{message_id}/reaction` | Send message reaction |
| POST | `/api/contacts/{id}/messages/{message_id}/revoke` | Revoke/Delete a sent message |
| PUT | `/api/messages/{id}/read` | Mark message as read |
| GET | `/api/media/{message_id}` | Download/Serve media from a message |
| GET | `/api/statuses` | List WhatsApp statuses |
| GET | `/api/statuses/{id}/media` | Serve status media |
| POST | `/api/statuses/{id}/reply` | Reply to a status |
| POST | `/api/statuses/{id}/mark-seen` | Mark status as seen |

## WhatsApp Instances (Whatsmeow)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/instances` | List WhatsApp instances |
| POST | `/api/instances` | Create a new instance |
| GET | `/api/instances/{id}` | Get instance details |
| PUT | `/api/instances/{id}` | Update instance settings |
| DELETE | `/api/instances/{id}` | Delete an instance |
| GET | `/api/instances/{id}/health` | Get instance health status |
| GET | `/api/instances/{id}/qr` | Get QR code for pairing |
| POST | `/api/instances/{id}/connect` | Connect instance to WhatsApp |
| POST | `/api/instances/{id}/pair-phone` | Pair instance via phone number/code |
| POST | `/api/instances/{id}/disconnect` | Disconnect instance |
| POST | `/api/instances/{id}/reconnect` | Reconnect instance |
| POST | `/api/instances/{id}/status/send` | Post a status update from instance |
| POST | `/api/instances/{id}/auto-campaign/media` | Upload media for instance auto-campaign |

## Campaigns & Bulk Messaging
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/campaigns` | List campaigns |
| POST | `/api/campaigns` | Create a campaign |
| GET | `/api/campaigns/{id}` | Get campaign details |
| PUT | `/api/campaigns/{id}` | Update campaign |
| DELETE | `/api/campaigns/{id}` | Delete campaign |
| POST | `/api/campaigns/{id}/start` | Start campaign execution |
| POST | `/api/campaigns/{id}/pause` | Pause campaign execution |
| POST | `/api/campaigns/{id}/cancel` | Cancel campaign |
| POST | `/api/campaigns/{id}/retry-failed` | Retry failed messages in campaign |
| GET | `/api/campaigns/{id}/progress` | Get campaign progress/stats |
| POST | `/api/campaigns/{id}/recipients/import` | Import recipients for campaign |
| GET | `/api/campaigns/{id}/recipients` | List campaign recipients |
| DELETE | `/api/campaigns/{id}/recipients/{recipientId}` | Remove recipient from campaign |
| POST | `/api/campaigns/{id}/media` | Upload media for campaign |
| GET | `/api/campaigns/{id}/media` | Serve campaign media |

## Chatbot & AI
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/chatbot/settings` | Get chatbot settings |
| PUT | `/api/chatbot/settings` | Update chatbot settings |
| GET | `/api/chatbot/keywords` | List keyword rules |
| POST | `/api/chatbot/keywords` | Create a keyword rule |
| GET | `/api/chatbot/keywords/{id}` | Get keyword rule |
| PUT | `/api/chatbot/keywords/{id}` | Update keyword rule |
| DELETE | `/api/chatbot/keywords/{id}` | Delete keyword rule |
| GET | `/api/chatbot/flows` | List chatbot flows |
| POST | `/api/chatbot/flows` | Create a chatbot flow |
| GET | `/api/chatbot/flows/{id}` | Get chatbot flow |
| PUT | `/api/chatbot/flows/{id}` | Update chatbot flow |
| DELETE | `/api/chatbot/flows/{id}` | Delete chatbot flow |
| GET | `/api/chatbot/ai-contexts` | List AI knowledge contexts |
| POST | `/api/chatbot/ai-contexts` | Create an AI context |
| GET | `/api/chatbot/ai-contexts/{id}` | Get AI context |
| PUT | `/api/chatbot/ai-contexts/{id}` | Update AI context |
| DELETE | `/api/chatbot/ai-contexts/{id}` | Delete AI context |
| GET | `/api/chatbot/sessions` | List chatbot sessions (Debug) |
| GET | `/api/chatbot/sessions/{id}` | Get session details |

## Chatbot Transfers
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/chatbot/transfers` | List active agent transfers |
| POST | `/api/chatbot/transfers` | Create a manual transfer |
| POST | `/api/chatbot/transfers/pick` | Pick the next available transfer |
| PUT | `/api/chatbot/transfers/{id}/resume` | Resume chatbot from transfer |
| PUT | `/api/chatbot/transfers/{id}/assign` | Assign transfer to an agent |

## Canned Responses
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/canned-responses` | List canned responses |
| POST | `/api/canned-responses` | Create a canned response |
| GET | `/api/canned-responses/{id}` | Get canned response |
| PUT | `/api/canned-responses/{id}` | Update canned response |
| DELETE | `/api/canned-responses/{id}` | Delete canned response |
| POST | `/api/canned-responses/{id}/send` | Send a canned response to a contact |
| POST | `/api/canned-responses/{id}/use` | Increment usage counter |

## Analytics & Dashboards
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/analytics/dashboard` | Get main dashboard statistics |
| GET | `/api/analytics/messages` | Get detailed message analytics |
| GET | `/api/analytics/chatbot` | Get chatbot performance analytics |
| GET | `/api/analytics/agents` | Get agent performance analytics |
| GET | `/api/analytics/agents/comparison` | Compare agent performance |
| GET | `/api/analytics/agents/ratings/export` | Export agent satisfaction ratings |
| GET | `/api/analytics/agents/{id}` | Get details for specific agent |
| GET | `/api/analytics/meta` | Get Meta-specific analytics (Meta only) |
| GET | `/api/analytics/meta/accounts` | List Meta accounts for analytics |
| POST | `/api/analytics/meta/refresh` | Force refresh analytics cache |
| GET | `/api/widgets` | List dashboard widgets |
| POST | `/api/widgets` | Create a widget |
| GET | `/api/widgets/data-sources` | List available data sources for widgets |
| GET | `/api/widgets/data` | Get data for all widgets |
| GET | `/api/widgets/{id}` | Get widget configuration |
| PUT | `/api/widgets/{id}` | Update widget configuration |
| DELETE | `/api/widgets/{id}` | Delete a widget |
| GET | `/api/widgets/{id}/data` | Get data for a specific widget |
| POST | `/api/widgets/layout` | Save dashboard widget layout |

## Webhooks & Extensions
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/webhook` | Meta Webhook verification (GET) |
| POST | `/api/webhook` | Meta Webhook handler (POST) |
| GET | `/api/webhooks` | List outbound webhooks |
| POST | `/api/webhooks` | Create an outbound webhook |
| GET | `/api/webhooks/{id}` | Get webhook details |
| PUT | `/api/webhooks/{id}` | Update webhook |
| DELETE | `/api/webhooks/{id}` | Delete webhook |
| POST | `/api/webhooks/{id}/test` | Test an outbound webhook |
| GET | `/api/custom-actions` | List custom actions |
| POST | `/api/custom-actions` | Create a custom action |
| GET | `/api/custom-actions/{id}` | Get custom action details |
| PUT | `/api/custom-actions/{id}` | Update custom action |
| DELETE | `/api/custom-actions/{id}` | Delete custom action |
| POST | `/api/custom-actions/{id}/execute` | Manually execute custom action |
| GET | `/api/custom-actions/redirect/{token}` | Redirect endpoint for custom actions |

## Meta Specific (Templates, Flows, Catalogs)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/templates` | List Meta message templates |
| POST | `/api/templates` | Create a Meta message template |
| GET | `/api/templates/{id}` | Get template details |
| PUT | `/api/templates/{id}` | Update template |
| DELETE | `/api/templates/{id}` | Delete template |
| POST | `/api/templates/sync` | Sync templates from Meta |
| POST | `/api/templates/{id}/publish` | Submit template to Meta for review |
| POST | `/api/templates/upload-media` | Upload media for template headers |
| GET | `/api/flows` | List Meta Flows |
| POST | `/api/flows` | Create a Meta Flow |
| GET | `/api/flows/{id}` | Get Flow details |
| PUT | `/api/flows/{id}` | Update Flow |
| DELETE | `/api/flows/{id}` | Delete Flow |
| POST | `/api/flows/{id}/save-to-meta` | Save Flow configuration to Meta |
| POST | `/api/flows/{id}/publish` | Publish Flow |
| POST | `/api/flows/{id}/deprecate` | Deprecate Flow |
| POST | `/api/flows/{id}/duplicate` | Duplicate Flow |
| POST | `/api/flows/sync` | Sync Flows from Meta |
| GET | `/api/catalogs` | List product catalogs |
| POST | `/api/catalogs` | Create a catalog |
| GET | `/api/catalogs/{id}` | Get catalog details |
| DELETE | `/api/catalogs/{id}` | Delete catalog |
| POST | `/api/catalogs/sync` | Sync catalogs from Meta |
| GET | `/api/catalogs/{id}/products` | List products in catalog |
| POST | `/api/catalogs/{id}/products` | Add product to catalog |
| GET | `/api/products/{id}` | Get product details |
| PUT | `/api/products/{id}` | Update product details |
| DELETE | `/api/products/{id}` | Remove product |

## Data Management & Utilities
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/export` | Generic data export |
| POST | `/api/import` | Generic data import |
| GET | `/api/export/{table}/config` | Get export configuration for a table |
| GET | `/api/import/{table}/config` | Get import configuration for a table |
| GET | `/api/tags` | List all available tags |
| POST | `/api/tags` | Create a tag |
| PUT | `/api/tags/{name}` | Update a tag |
| DELETE | `/api/tags/{name}` | Delete a tag |
| GET | `/api/activity-logs` | List system activity logs |
| POST | `/api/activity-logs` | Create a manual activity log entry |
| GET | `/api/lead-requests` | List lead requests |
| PUT | `/api/lead-requests/{id}/status` | Update lead request status |
| POST | `/api/public/lead-requests` | Create a lead request (Public) |
| POST | `/api/admin/migrate` | Trigger data migration (Super Admin) |
| GET | `/api/admin/migrate/status` | Get data migration status |
| GET | `/api/notifications` | List system notifications |
| PUT | `/api/notifications/{id}/dismiss` | Dismiss a notification |
| GET | `/api/org/settings` | Get organization-level settings |
| PUT | `/api/org/settings` | Update organization-level settings |
| GET | `/api/settings/sso` | Get organization SSO settings |
| PUT | `/api/settings/sso/{provider}` | Update SSO provider configuration |
| DELETE | `/api/settings/sso/{provider}` | Remove SSO provider |
| GET | `/api/accounts` | List linked accounts |
| POST | `/api/accounts` | Link a new account |
| GET | `/api/accounts/{id}` | Get account details |
| PUT | `/api/accounts/{id}` | Update account details |
| DELETE | `/api/accounts/{id}` | Unlink account |
| POST | `/api/accounts/{id}/test` | Test account connection |
| POST | `/api/accounts/{id}/subscribe` | Subscribe to app events |
| GET | `/api/accounts/{id}/business_profile` | Get WhatsApp business profile |
| PUT | `/api/accounts/{id}/business_profile` | Update WhatsApp business profile |
| POST | `/api/accounts/{id}/business_profile/photo` | Update WhatsApp profile picture |
