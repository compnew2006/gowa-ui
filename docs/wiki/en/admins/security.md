---
title: Security
---

# Security

Whatomate implements multiple layers of security to protect data in transit, at rest, and during processing.

## Authentication

### JWT Authentication

Whatomate uses JWT (JSON Web Tokens) with HS256 signing for API authentication.

**Token Types:**

| Token | TTL | Purpose |
|-------|-----|---------|
| Access Token | 15 minutes | API request authentication |
| Refresh Token | 7 days | Obtain new access tokens |
| WebSocket Token | 30 seconds | WebSocket connection authentication |

**Token Claims:**

```json
{
  "sub": "refresh",
  "user_id": 1,
  "org_id": 1,
  "role_id": 1,
  "permissions": ["contacts:read", "messages:write"],
  "jti": "unique-token-id",
  "exp": 1700000000,
  "iat": 1699999990
}
```

### Refresh Token Rotation

Refresh tokens use a rotation pattern to prevent replay attacks:

1. On login, a refresh token JTI (JWT ID) is stored in Redis
2. On refresh, the JTI is checked and **deleted** from Redis (single-use)
3. A new token pair is generated with a new JTI
4. If a deleted JTI is presented again, it indicates a replay attack and is rejected

```
Login → Store JTI-A in Redis
Refresh → Check JTI-A exists → Delete JTI-A → Generate JTI-B → Store JTI-B
Replay  → Check JTI-A → Not found → Reject (replay attack detected)
```

### Cookie Security

Auth tokens are stored in HTTP-only, Secure, SameSite=Strict cookies:

| Cookie | Purpose | Flags |
|--------|---------|-------|
| `whm_access` | Access token | HTTP-only, Secure, SameSite=Strict |
| `whm_refresh` | Refresh token | HTTP-only, Secure, SameSite=Strict |
| `whm_csrf` | CSRF token | HTTP-only, Secure, SameSite=Strict |

### SSO Authentication

SSO providers (Google, Azure AD, etc.) use OAuth2 with state token CSRF protection:

1. Generate state token (CSRF protection)
2. Redirect to provider's authorization URL
3. Exchange authorization code for tokens
4. Fetch user info and create/find local user
5. Generate JWT pair and set cookies

## CSRF Protection

Cross-Site Request Forgery protection is enforced on all mutating requests:

```
POST/PUT/DELETE/PATCH → Extract X-CSRF-Token header
                      → Compare with whm_csrf cookie
                      → Match → Proceed
                      → Mismatch → 403 Forbidden

GET/HEAD/OPTIONS → Skip CSRF check
```

## Data Encryption

### AES-256-GCM Encryption

Sensitive data is encrypted at rest using AES-256-GCM:

```go
// Encryption
ciphertext := crypto.Encrypt(plaintext)  // Returns "enc3:<base64>"

// Decryption
plaintext, err := crypto.Decrypt(ciphertext)
```

**Encrypted Fields:**

| Model | Fields |
|-------|--------|
| WhatsAppAccount | access_token, phone_number_id, business_account_id, webhook_verify_token |
| SSOProvider | client_secret |
| ChatbotSettings | ai_api_key |
| Webhook | secret |
| CustomAction | headers |

### Encryption Versions

| Version | Prefix | Status |
|---------|--------|--------|
| enc1 | `enc:` | Legacy — should be migrated |
| enc2 | `enc2:` | Legacy — should be migrated |
| enc3 | `enc3:` | Current (AES-256-GCM) |

Run crypto migration to upgrade legacy data:

```bash
whatomate crypto-migrate
```

## SSRF-Safe Dialer

Outbound HTTP requests use an SSRF-safe dialer that blocks requests to internal IP ranges:

**Blocked Ranges:**

| Range | Type |
|-------|------|
| `127.0.0.0/8` | Loopback |
| `10.0.0.0/8` | Private |
| `172.16.0.0/12` | Private |
| `192.168.0.0/16` | Private |
| `169.254.0.0/16` | Link-local |
| `::1` | IPv6 loopback |
| `fc00::/7` | IPv6 unique local |
| `fe80::/10` | IPv6 link-local |

## Send Restriction Policies

### Organization-Level Settings

| Setting | Description |
|---------|-------------|
| `strict_sending_restrictions_enabled` | Master toggle for strict sending mode |
| `outbound_mode` | "inbound_only" (reply only) or "mixed" (full outbound) |
| `strict_sending_apply_to_system` | Apply restrictions to chatbot/system messages |
| `campaign_draft_only` | Restrict campaigns to draft mode only |
| `strict_rollout_mode` | "audit" (log violations) or "enforce" (block messages) |
| `strict_rollout_enforce_at` | Timestamp when enforcement begins |

### User-Level Settings

| Setting | Description |
|---------|-------------|
| `enabled` | Toggle restrictions for this user |
| `include_all_contacts` | Allow all contacts or restrict to whitelist |
| `authorized_numbers` | Whitelist of allowed phone numbers |
| `allowed_instance_id` | Specific instance user can send from |
| `allowed_instance_ids` | Multiple instances user can send from |
| `prefix_agent_name` | Auto-prefix messages with agent's name |
| `allow_unclaimed_chat_view` | Allow viewing unclaimed chats |
| `allow_unclaimed_chat_send` | Allow sending to unclaimed chats |

### Enforcement Modes

**Audit Mode:** Log violations but allow messages to proceed.

**Enforce Mode:** Block messages that violate restrictions with a `restrictedSendViolationError`.

## Agent Chat Visibility Restrictions

Agent-role users have restricted visibility:

- Only see chats assigned to them
- Only see public chats (`is_public = true`)
- Only see chats where they are a collaborator

This applies even if the agent has `contacts:read` permission.

## Security Headers

All HTTP responses include security headers:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME type sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-XSS-Protection` | `1; mode=block` | Enable browser XSS filter |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer information |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Restrict browser features |

## Rate Limiting

Rate limiting is enforced at multiple levels:

| Level | Key | Default Limit |
|-------|-----|---------------|
| Per-user | `user_id:org_id` | 1000 requests/window |
| Per-IP | IP address | 100 requests/window |
| Auth endpoints | IP address | Lower limit |
| Outbound messages | User | Configurable |
| Webhooks | Endpoint | Configurable |

Rate limits use Redis counters with sliding windows. Exceeded limits return HTTP 429.

## Password Policy

Passwords must meet these requirements:

| Rule | Requirement |
|------|-------------|
| Minimum length | 8 characters |
| Uppercase | At least 1 uppercase letter |
| Lowercase | At least 1 lowercase letter |
| Digit | At least 1 digit |
| Special character | At least 1 special character |
| Common passwords | Not in common password list |

Validation is performed during registration and password changes.

## Security Checklist

- [ ] Strong encryption key (32 bytes, unique per environment)
- [ ] Strong JWT secret (32+ characters)
- [ ] Database credentials rotated regularly
- [ ] Redis password set (not empty)
- [ ] CORS `allowed_origins` restricted to known domains
- [ ] `debug = false` in production
- [ ] SSL/TLS termination at reverse proxy
- [ ] Rate limiting enabled
- [ ] Send restrictions configured for compliance
- [ ] Activity logging enabled for audit trail
- [ ] Regular security advisor checks (run `get_advisors`)

## See Also

- [Configuration](configuration.md) — Security-related configuration options
- [Deployment](deployment.md) — Production deployment checklist
- [Monitoring](monitoring.md) — Security event monitoring
- [Troubleshooting](troubleshooting.md) — Authentication failure resolution
