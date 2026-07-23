# WhatsApp API MultiDevice - API Endpoints

**Version:** 8.10.0  
**Base URL:** `http://localhost:3000`  
**Authentication:** Basic Auth (`basicAuth`)

> Most device-scoped endpoints accept `X-Device-Id` header or `device_id` query parameter.

---

## Table of Contents

- [App](#app)
- [Device](#device)
- [User](#user)
- [Send](#send)
- [Message](#message)
- [Call](#call)
- [Chat](#chat)
- [Group](#group)
- [Newsletter](#newsletter)
- [Chatwoot](#chatwoot)

---

## App

Initial Connection to WhatsApp server.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `GET` | `/health` | Health check | No |
| `GET` | `/app/login` | Login to WhatsApp server (QR) | Yes |
| `GET` | `/app/login-with-code` | Login with pairing code | Yes |
| `GET` | `/app/passkey` | Get pending passkey pairing state | Yes |
| `POST` | `/app/passkey/response` | Submit WebAuthn assertion for passkey pairing | Yes |
| `POST` | `/app/passkey/confirm` | Confirm passkey pairing code | Yes |
| `GET` | `/app/logout` | Logout active device (keeps device slot) | Yes |
| `GET` | `/app/reconnect` | Reconnecting to WhatsApp server | Yes |
| `GET` | `/app/devices` | Get list connected devices | Yes |
| `GET` | `/app/status` | Get connection status | Yes |

### `GET /health`
Health probe (not prefixed by `APP_BASE_PATH`). Returns `OK` (200) or `Service Unavailable` (503).

### `GET /app/login`
Login to WhatsApp server via QR code.

**Header:** `X-Device-Id` (optional)  
**Response:** `LoginResponse` — includes `qr_link` and `qr_duration`.

### `GET /app/login-with-code`
Login with pairing code instead of QR.

**Header:** `X-Device-Id` (optional)  
**Query:** `phone` — your phone number  
**Response:** `LoginWithCodeResponse` — includes `pair_code`.

### `GET /app/passkey`
Get pending passkey pairing state. Returns `none`, `awaiting_response`, or `awaiting_confirmation`.

**Header:** `X-Device-Id` (optional)  
**Response:** `PasskeyStatusResponse`.

### `POST /app/passkey/response`
Submit the WebAuthn assertion for passkey pairing.

**Header:** `X-Device-Id` (optional)  
**Body:** `WebAuthnAssertion` (JSON)  
**Response:** `GenericResponse`.

### `POST /app/passkey/confirm`
Confirm the passkey pairing code from the PASSKEY_CONFIRMATION event.

**Header:** `X-Device-Id` (optional)  
**Response:** `GenericResponse`.

### `GET /app/logout`
Logout the active device. Keeps the device slot; use `DELETE /devices/{device_id}` to remove the slot.

**Header:** `X-Device-Id` (optional)  
**Response:** `GenericResponse`.

### `GET /app/reconnect`
Reconnect to WhatsApp server.

**Header:** `X-Device-Id` (optional)  
**Response:** `GenericResponse`.

### `GET /app/devices`
Get list of all connected devices (persisted across restarts).

**Response:** `DeviceResponse`.

### `GET /app/status`
Get connection status (`is_connected`, `is_logged_in`, `device_id`, `jid`).

**Header:** `X-Device-Id` (optional)  
**Response:** object with `status`, `code`, `message`, `results`.

---

## Device

Device management for multi-device support.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `GET` | `/devices` | List all devices | Yes |
| `POST` | `/devices` | Add a new device | Yes |
| `GET` | `/devices/{device_id}` | Get device info | Yes |
| `DELETE` | `/devices/{device_id}` | Remove a device | Yes |
| `GET` | `/devices/{device_id}/login` | Reserved device QR login (deprecated) | Yes |
| `POST` | `/devices/{device_id}/login/code` | Reserved device pairing-code login (deprecated) | Yes |
| `POST` | `/devices/{device_id}/logout` | Logout device | Yes |
| `POST` | `/devices/{device_id}/reconnect` | Reconnect device | Yes |
| `GET` | `/devices/{device_id}/status` | Get device connection status | Yes |
| `GET` | `/devices/{device_id}/webhook` | Get device webhook configuration | Yes |
| `PATCH` | `/devices/{device_id}/webhook` | Set device webhook configuration | Yes |

### `GET /devices`
Returns all registered devices with their connection status.

**Response:** `DeviceListResponse`.

### `POST /devices`
Create a new device slot.

**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `device_id` | string | No | Custom device ID (auto-generated if omitted) |
| `webhook_url` | string | No | Device-specific webhook URL |
| `webhook_secret` | string | No | HMAC secret for webhook payloads |
| `webhook_events` | string | No | Comma-separated event whitelist |
| `webhook_insecure_skip_verify` | boolean | No | Skip TLS verification |

**Response:** `DeviceAddResponse`.

### `GET /devices/{device_id}`
Get detailed information about a specific device.

**Path:** `device_id`  
**Response:** `DeviceInfoResponse`.

### `DELETE /devices/{device_id}`
Remove a device slot (also logs out and clears session data).

**Path:** `device_id`  
**Response:** `GenericResponse`.

### `POST /devices/{device_id}/logout` ⚠️ Deprecated → use `GET /app/logout` with `X-Device-Id`
Logout a specific device (keeps the slot).

### `POST /devices/{device_id}/reconnect`
Reconnect a specific device to WhatsApp.

### `GET /devices/{device_id}/status`
Get the current connection status of a specific device.

### `GET /devices/{device_id}/webhook`
Get the webhook configuration for a specific device.

### `PATCH /devices/{device_id}/webhook`
Set or update the webhook configuration for a specific device.

**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `webhook_url` | string | Yes | Webhook URL (empty string to clear) |
| `webhook_secret` | string | No | HMAC signing secret |
| `webhook_events` | string | No | Comma-separated event whitelist |
| `webhook_insecure_skip_verify` | boolean | No | Skip TLS verification |

---

## User

Getting user information.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `GET` | `/user/info` | User Info | Yes |
| `GET` | `/user/avatar` | Get User Avatar | Yes |
| `POST` | `/user/avatar` | Change User Avatar | Yes |
| `POST` | `/user/pushname` | Change display name (push name) | Yes |
| `GET` | `/user/my/privacy` | Get privacy settings | Yes |
| `GET` | `/user/my/groups` | Get list of joined groups (max 500) | Yes |
| `GET` | `/user/my/newsletters` | Get list of newsletters | Yes |
| `GET` | `/user/my/contacts` | Get list of contacts | Yes |
| `GET` | `/user/check` | Check if user is on WhatsApp | Yes |
| `GET` | `/user/business-profile` | Get Business Profile | Yes |

### `GET /user/info`
**Header:** `X-Device-Id` (optional)  
**Query:** `phone` — phone with country code (e.g. `6289685028129@s.whatsapp.net`)  
**Response:** `UserInfoResponse`.

### `GET /user/avatar`
**Query:** `phone`, `is_preview` (bool), `is_community` (bool)  
**Response:** `UserAvatarResponse`.

### `POST /user/avatar`
**Body:** `multipart/form-data` with `avatar` (binary).  
**Response:** `GenericResponse`.

### `POST /user/pushname`
**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `push_name` | string | Yes | New display name |

**Response:** `GenericResponse`.

### `GET /user/my/privacy`
Returns privacy settings: `group_add`, `last_seen`, `status`, `profile`, `read_receipts`.

### `GET /user/my/groups`
Returns all joined groups (max 500 due to WhatsApp protocol limitation).

### `GET /user/my/newsletters`
Returns list of followed newsletters/channels.

### `GET /user/my/contacts`
Returns list of saved contacts with `jid` and `name`.

### `GET /user/check`
**Query:** `phone`  
**Response:** `UserCheckResponse` with `is_on_whatsapp` boolean.

### `GET /user/business-profile`
**Query:** `phone` (required)  
**Response:** `BusinessProfileResponse` — email, address, categories, business hours, timezone.

---

## Send

Send Message (Text/Image/File/Video/Audio/Sticker/Contact/Link/Location/Poll/Presence).

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `POST` | `/send/message` | Send text message | Yes |
| `POST` | `/send/image` | Send image | Yes |
| `POST` | `/send/audio` | Send audio | Yes |
| `POST` | `/send/file` | Send file/document | Yes |
| `POST` | `/send/sticker` | Send sticker | Yes |
| `POST` | `/send/video` | Send video | Yes |
| `POST` | `/send/contact` | Send contact | Yes |
| `POST` | `/send/link` | Send link | Yes |
| `POST` | `/send/location` | Send location | Yes |
| `POST` | `/send/poll` | Send poll/vote | Yes |
| `POST` | `/send/presence` | Send presence status | Yes |
| `POST` | `/send/chat-presence` | Send typing indicator | Yes |

### `POST /send/message`
**Header:** `X-Device-Id` (optional)  
**Body (JSON):**
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `message` | string | Message text |
| `reply_message_id` | string | Message ID to reply to |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer (0/86400/604800/7776000) |
| `mentions` | array | Phone numbers to mention; use `@everyone` for all |

**Response:** `SendResponse`.

### `POST /send/image`
**Body:** `multipart/form-data`
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `caption` | string | Caption text |
| `reply_message_id` | string | Message ID to reply to |
| `view_once` | boolean | View once |
| `image` | binary | Image file |
| `image_url` | string | Image URL |
| `compress` | boolean | Compress image |
| `duration` | integer | Disappearing timer |
| `is_forwarded` | boolean | Forwarded flag |

**Response:** `SendResponse`.

### `POST /send/audio`
**Body:** `multipart/form-data`
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `audio` | binary | Audio file |
| `audio_url` | string | Audio URL |
| `ptt` | boolean | Send as voice note (requires ffmpeg) |
| `reply_message_id` | string | Message ID to reply to |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/file`
**Body:** `multipart/form-data`
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `caption` | string | Caption text |
| `reply_message_id` | string | Message ID to reply to |
| `file` | binary | File to send |
| `file_url` | string | File URL to download and send |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/sticker`
Send sticker with automatic WebP conversion.

**Body:** `multipart/form-data`
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `sticker` | binary | Sticker image (jpg/jpeg/png/webp/gif) |
| `sticker_url` | string | Sticker image URL |
| `duration` | integer | Disappearing timer |
| `is_forwarded` | boolean | Forwarded flag |

**Response:** `SendResponse`.

### `POST /send/video`
**Body:** `multipart/form-data`
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `caption` | string | Caption text |
| `reply_message_id` | string | Message ID to reply to |
| `view_once` | boolean | View once |
| `video` | binary | Video file |
| `video_url` | string | Video URL |
| `compress` | boolean | Compress video |
| `gif_playback` | boolean | Display as GIF |
| `duration` | integer | Disappearing timer |
| `is_forwarded` | boolean | Forwarded flag |

**Response:** `SendResponse`.

### `POST /send/contact`
**Body (JSON):**
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `contact_name` | string | Contact display name |
| `contact_phone` | string | Contact phone number |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/link`
**Body (JSON):**
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `link` | string | Link URL |
| `caption` | string | Caption text |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/location`
**Body (JSON):**
| Field | Type | Description |
|-------|------|-------------|
| `phone` | string | Phone with country code |
| `latitude` | string | Latitude coordinate |
| `longitude` | string | Longitude coordinate |
| `is_forwarded` | boolean | Forwarded flag |
| `duration` | integer | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/poll`
**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Phone with country code |
| `question` | string | Yes | Poll question |
| `options` | array | Yes | Array of option strings |
| `max_answer` | integer | Yes | Max answers allowed |
| `duration` | integer | No | Disappearing timer |

**Response:** `SendResponse`.

### `POST /send/presence`
Send presence status (`available` / `unavailable`).

**Body (JSON):** `{ "type": "available" | "unavailable" }`

### `POST /send/chat-presence`
Send typing indicator.

**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Phone with country code |
| `action` | string | Yes | `start` or `stop` |

---

## Message

Message manipulation (revoke, react, update, delete, read, star, forward, download).

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `POST` | `/message/{message_id}/revoke` | Revoke (delete for everyone) | Yes |
| `POST` | `/message/{message_id}/delete` | Delete message (for self) | Yes |
| `POST` | `/message/{message_id}/reaction` | React to a message | Yes |
| `POST` | `/message/{message_id}/update` | Edit message (within 15 min) | Yes |
| `POST` | `/message/{message_id}/read` | Mark as read | Yes |
| `POST` | `/message/{message_id}/star` | Star message | Yes |
| `POST` | `/message/{message_id}/unstar` | Unstar message | Yes |
| `POST` | `/message/{message_id}/forward` | Forward message | Yes |
| `GET` | `/message/{message_id}/download` | Download media from message | Yes |

### `POST /message/{message_id}/revoke`
Revoke (delete for everyone). Bot's own messages by default; can revoke others' messages if bot is group admin.

**Body (JSON):** `{ "phone": "..." }`

### `POST /message/{message_id}/delete`
Delete message for self.

**Body (JSON):** `{ "phone": "..." }`

### `POST /message/{message_id}/reaction`
**Body (JSON):** `{ "phone": "...", "emoji": "🙏" }`

### `POST /message/{message_id}/update`
Edit message text (within 15 minutes).

**Body (JSON):** `{ "phone": "...", "message": "new text" }`

### `POST /message/{message_id}/read`
Mark a message as read.

**Body (JSON):** `{ "phone": "..." }`

### `POST /message/{message_id}/star`
Star a message.

**Body (JSON):** `{ "phone": "..." }`

### `POST /message/{message_id}/unstar`
Unstar a message.

**Body (JSON):** `{ "phone": "..." }`

### `POST /message/{message_id}/forward`
Forward a message from local chat storage.

**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Destination phone/group JID |
| `duration` | integer | No | Disappearing timer override |
| `force_reupload` | boolean | No | Force re-upload media |

### `GET /message/{message_id}/download`
Download media content from a message.

**Path:** `message_id`  
**Query:** `phone` (required)  
**Response:** object with `file_path`, `file_url`, `file_size`, `media_type`, `filename`.

---

## Call

Call management.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `POST` | `/call/reject` | Reject an incoming call | Yes |

### `POST /call/reject`
Reject a specific incoming call. Call must still be ringing.

**Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `caller_jid` | string | Yes | Caller JID (from webhook `payload.from`) |
| `call_id` | string | Yes | Call ID (from webhook `payload.call_id`) |

**Response:** `GenericResponse`.

---

## Chat

Chat conversations and messaging.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `GET` | `/chats` | Get list of chats | Yes |
| `GET` | `/chat/{chat_jid}/messages` | Get messages from a chat | Yes |
| `POST` | `/chat/{chat_jid}/pin` | Pin or unpin a chat | Yes |
| `POST` | `/chat/{chat_jid}/disappearing` | Set disappearing messages timer | Yes |
| `POST` | `/chat/{chat_jid}/archive` | Archive or unarchive a chat | Yes |

### `GET /chats`
**Query:** `limit` (default 25, max 100), `offset`, `search`, `has_media` (bool), `archived` (bool)  
**Response:** `ChatListResponse` with pagination.

### `GET /chat/{chat_jid}/messages`
**Path:** `chat_jid`  
**Query:** `limit` (default 50, max 100), `offset`, `start_time`, `end_time`, `media_only` (bool), `is_from_me` (bool), `search`  
**Response:** `ChatMessagesResponse` with pagination and `chat_info`.

### `POST /chat/{chat_jid}/pin`
**Body (JSON):** `{ "pinned": true }`

### `POST /chat/{chat_jid}/disappearing`
Set disappearing messages timer (0 = off, 86400 = 24h, 604800 = 7d, 7776000 = 90d).

**Body (JSON):** `{ "timer_seconds": 86400 }`

### `POST /chat/{chat_jid}/archive`
**Body (JSON):** `{ "archived": true }`

---

## Group

Group setting and management.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `GET` | `/group/info` | Get group info | Yes |
| `POST` | `/group` | Create group | Yes |
| `GET` | `/group/participants` | Get group participants | Yes |
| `POST` | `/group/participants` | Add participants to group | Yes |
| `GET` | `/group/participants/export` | Export participants as CSV | Yes |
| `POST` | `/group/participants/remove` | Remove participants | Yes |
| `POST` | `/group/participants/promote` | Promote to admin | Yes |
| `POST` | `/group/participants/demote` | Demote to member | Yes |
| `POST` | `/group/join-with-link` | Join group with link | Yes |
| `GET` | `/group/info-from-link` | Get group info from invite link | Yes |
| `GET` | `/group/participant-requests` | Get participant requests | Yes |
| `POST` | `/group/participant-requests/approve` | Approve participant request | Yes |
| `POST` | `/group/participant-requests/reject` | Reject participant request | Yes |
| `POST` | `/group/leave` | Leave group | Yes |
| `POST` | `/group/photo` | Set group photo | Yes |
| `POST` | `/group/name` | Set group name | Yes |
| `POST` | `/group/locked` | Set group locked status | Yes |
| `POST` | `/group/announce` | Set group announce mode | Yes |
| `POST` | `/group/topic` | Set group topic/description | Yes |
| `GET` | `/group/invite-link` | Get group invite link | Yes |

### `GET /group/info`
**Query:** `group_id`  
**Response:** `GroupInfoResponse`.

### `POST /group`
Create group and add participants.

**Body (JSON):**
| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Group name |
| `participants` | array | Array of phone numbers |

### `GET /group/participants`
**Query:** `group_id` (required)  
**Response:** `GroupParticipantsResponse`.

### `POST /group/participants`
**Body (JSON):** `{ "group_id": "...", "participants": ["...", "..."] }`

### `GET /group/participants/export`
Export participants as CSV.

**Query:** `group_id` (required)  
**Response:** `text/csv` stream.

### `POST /group/participants/remove`
Remove participants from group.

**Body (JSON):** `ManageParticipantRequest`.

### `POST /group/participants/promote`
Promote participants to admin.

**Body (JSON):** `ManageParticipantRequest`.

### `POST /group/participants/demote`
Demote participants to member.

**Body (JSON):** `ManageParticipantRequest`.

### `POST /group/join-with-link`
**Body (JSON):** `{ "link": "https://chat.whatsapp.com/..." }`

### `GET /group/info-from-link`
Get group info without joining.

**Query:** `link` (required)  
**Response:** `GroupInfoFromLinkResponse` — group_id, name, topic, created_at, participant_count, etc.

### `GET /group/participant-requests`
**Query:** `group_id` (required)  
**Response:** `GroupParticipantRequestListResponse`.

### `POST /group/participant-requests/approve`
**Body (JSON):** `{ "group_id": "...", "participants": ["..."] }`

### `POST /group/participant-requests/reject`
**Body (JSON):** `{ "group_id": "...", "participants": ["..."] }`

### `POST /group/leave`
**Body (JSON):** `{ "group_id": "..." }`

### `POST /group/photo`
**Body:** `multipart/form-data` with `group_id` and `photo` (binary, JPEG recommended).

### `POST /group/name`
**Body (JSON):** `{ "group_id": "...", "name": "New Group Name" }` (max 25 chars).

### `POST /group/locked`
Lock/unlock group (only admins can modify group info).

**Body (JSON):** `{ "group_id": "...", "locked": true }`

### `POST /group/announce`
Enable/disable announce mode (only admins can send messages).

**Body (JSON):** `{ "group_id": "...", "announce": true }`

### `POST /group/topic`
Set or remove group topic/description.

**Body (JSON):** `{ "group_id": "...", "topic": "..." }` (empty to remove).

### `GET /group/invite-link`
**Query:** `group_id` (required), `reset` (bool, default false)  
**Response:** `GetGroupInviteLinkResponse`.

---

## Newsletter

Newsletter (channel) management.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `POST` | `/newsletter/unfollow` | Unfollow newsletter | Yes |
| `GET` | `/newsletter/messages` | Get latest newsletter messages | Yes |

### `POST /newsletter/unfollow`
**Body (JSON):** `{ "newsletter_id": "120363024512399999@newsletter" }`

### `GET /newsletter/messages`
**Query:** `newsletter_id` (required), `count` (default 50, max 100), `before` (server ID for pagination)  
**Response:** `NewsletterMessagesResponse`.

---

## Chatwoot

Chatwoot integration for customer support.

| Method | Endpoint | Summary | Auth |
|--------|----------|---------|------|
| `POST` | `/chatwoot/sync` | Sync message history to Chatwoot | No |
| `GET` | `/chatwoot/sync/status` | Get sync progress | No |
| `POST` | `/chatwoot/webhook` | Chatwoot webhook endpoint | No |

### `POST /chatwoot/sync`
Initiates background sync of WhatsApp history to Chatwoot.

**Body (JSON):**
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `device_id` | string | — | Device ID to sync |
| `days_limit` | integer | 3 | Days of history to sync |
| `include_media` | boolean | true | Include media attachments |
| `include_groups` | boolean | true | Include group chats |

**Response:** `ChatwootSyncResponse`.

### `GET /chatwoot/sync/status`
**Query:** `device_id`  
**Response:** `ChatwootSyncStatusResponse` — includes progress, total/synced chats and messages.

### `POST /chatwoot/webhook`
Receives webhook events from Chatwoot and sends messages to WhatsApp.

**Body:** Chatwoot webhook payload.

---

## Common Parameters

| Parameter | Location | Description |
|-----------|----------|-------------|
| `X-Device-Id` | Header | Device identifier for multi-device support |
| `device_id` | Query | Alternative to `X-Device-Id` header |

## Common Response Schemas

| Schema | Fields |
|--------|--------|
| `GenericResponse` | `code`, `message`, `results` |
| `SendResponse` | `code`, `message`, `results.message_id`, `results.status` |
| `ErrorBadRequest` | `code` (400), `message`, `results` |
| `ErrorUnauthorized` | `code` (401), `message`, `results` |
| `ErrorNotFound` | `code` (404), `message`, `results` |
| `ErrorInternalServer` | `code`, `message`, `results` |
