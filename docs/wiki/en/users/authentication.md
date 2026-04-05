---
title: Authentication & User Settings
---

# Authentication & User Settings

This page covers logging in, registering, managing your account, and personalizing your Whatomate experience.

## Login

**Endpoint:** `POST /api/auth/login`

To log in, provide your email and password:

```json
{
  "email": "you@example.com",
  "password": "your-password"
}
```

**What happens:**

1. Your credentials are verified securely
2. An access token (15-minute expiry) and refresh token (7-day expiry) are generated
3. Tokens are stored as secure, HTTP-only cookies
4. Your user profile and role permissions are loaded

**Response:**

```json
{
  "expires_in": 900,
  "user": {
    "id": "uuid",
    "email": "you@example.com",
    "full_name": "Your Name",
    "role": "admin",
    "is_active": true
  }
}
```

> **Security note:** Failed login attempts use a dummy bcrypt comparison to prevent timing attacks. Error messages are generic to avoid revealing whether an email exists.

## Registration

**Endpoint:** `POST /api/auth/register`

Registration requires a valid invitation token from your organization admin:

```json
{
  "email": "newuser@example.com",
  "password": "SecurePass123!",
  "full_name": "New User",
  "invitation_token": "jwt-token-from-invite"
}
```

**Requirements:**

- A valid, non-expired invitation token
- Password must meet the strength policy (see [Password Policy](#password-policy))
- If the email already exists, the request succeeds silently without creating a duplicate

## Refresh Token

**Endpoint:** `POST /api/auth/refresh`

Access tokens expire after 15 minutes. The refresh token flow automatically generates a new token pair:

1. The system checks that the refresh token hasn't been used before (replay attack prevention)
2. A new access token and refresh token are issued
3. Your organization context and permissions are preserved

This happens automatically in the background — you don't need to trigger it manually.

## Logout

**Endpoint:** `POST /api/auth/logout`

Logging out revokes your refresh token and clears all authentication cookies. All active sessions are terminated.

## Switching Organizations

**Endpoint:** `POST /api/auth/switch-org`

If you belong to multiple organizations, switch between them:

```json
{
  "organization_id": "org-uuid"
}
```

Both tokens are regenerated with the new organization context. Your role in the target organization is applied automatically.

## SSO Authentication

Whatomate supports Single Sign-On (SSO) via OAuth2 providers.

### Available Providers

**Endpoint:** `GET /api/auth/sso/providers`

Returns a list of enabled SSO providers for your organization.

### Initiating SSO

**Endpoint:** `GET /api/auth/sso/{provider}/init`

You are redirected to your identity provider (e.g., Google, Microsoft) to authenticate.

### SSO Callback

**Endpoint:** `GET /api/auth/sso/{provider}/callback`

After authentication, Whatomate:

1. Validates the OAuth2 state token (CSRF protection)
2. Exchanges the authorization code for user info
3. Finds or creates your local user account
4. Generates session tokens and redirects you to the app

## Password Policy

All passwords must meet these requirements:

| Rule | Requirement |
|------|------------|
| Minimum length | 8 characters |
| Uppercase | At least one uppercase letter |
| Lowercase | At least one lowercase letter |
| Digit | At least one number |
| Special character | At least one special character |
| Common passwords | Must not be in the common password list |

### Change Password

**Endpoint:** `PUT /api/me/password`

```json
{
  "current_password": "old-password",
  "new_password": "NewSecurePass456!"
}
```

After changing your password, all existing sessions are invalidated and you must log in again.

## User Settings

**Endpoint:** `PUT /api/me/settings`

Personalize your experience:

```json
{
  "email_notifications": true,
  "new_message_alerts": true,
  "campaign_updates": true,
  "notification_sound": "notification1",
  "chat_background": "aurora-veil"
}
```

### Notification Sounds

Choose from three built-in sounds:

- `notification1`
- `notification2`
- `notification`

### Chat Background

Customize your chat window background with a preset or upload your own image.

**Available presets:**

| Preset | Description |
|--------|-------------|
| `aurora-veil` | Soft gradient aurora |
| `sunset-dunes` | Warm desert tones |
| `paper-garden` | Light botanical pattern |
| `linen-grid` | Subtle grid on linen |
| `dot-orbit` | Dotted orbital pattern |
| `ripple-lines` | Concentric ripple lines |

### Upload Custom Background

**Endpoint:** `POST /api/me/chat-background`

Upload your own image:

- **Allowed formats:** JPEG, PNG, WebP
- **Maximum size:** 5 MB

### Get Current Background

**Endpoint:** `GET /api/me/chat-background`

Returns your currently active chat background settings.

## Availability

**Endpoint:** `PUT /api/me/availability`

Toggle your availability status:

```json
{
  "is_available": true
}
```

When set to `false`, you will not receive new chat assignments. Your team can see your availability status in real time via WebSocket updates.

## WebSocket Authentication

**Endpoint:** `GET /api/auth/ws-token`

Generates a short-lived (30-second) JWT token for WebSocket connections. This token is single-use and specifically scoped for real-time communication.

## See Also

- [Teams & Roles](teams-roles.md) — Understanding roles and permissions
- [Tags & Organization](tags-organization.md) — Organization-level settings
- [Chat & Messaging](chat-messaging.md) — Using the chat interface
