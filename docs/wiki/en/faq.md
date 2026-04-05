---
title: Frequently Asked Questions
---

# Frequently Asked Questions

Find answers to common questions about Whatomate.

<!-- TODO: Add more FAQs -->

## General

### What is Whatomate?

Whatomate is a self-hosted, multi-tenant WhatsApp Business API platform. It enables organizations to manage customer communications at scale through messaging, campaigns, chatbot automation, and team collaboration.

### What providers are supported?

Whatomate supports two WhatsApp providers:

- **Meta Cloud API** — The official Meta WhatsApp Business Cloud API with template approval, Flows, and catalogs
- **WhatsMeow** — Direct WhatsApp Web protocol connection via `go.mau.fi/whatsmeow`, no template approval needed

You choose the provider in your `config.toml` under `whatsapp.provider`.

### What is the difference between Meta and WhatsMeow?

| Feature | Meta Cloud API | WhatsMeow |
|---------|---------------|-----------|
| Connection | Official API via HTTP | Direct WhatsApp Web protocol |
| Authentication | Access tokens | QR code or phone-code pairing |
| Templates | Requires Meta approval | No approval needed |
| Flows/Catalogs | Supported | Not available |
| Rate limits | Meta-enforced | Per-instance configurable |
| Reliability | High (official) | Depends on WhatsApp Web stability |

### Can I use both providers at the same time?

No. The provider is configured globally in `config.toml`. You can switch between them, but only one is active at a time.

## Authentication & Users

### How does authentication work?

Whatomate uses JWT (HS256) with a two-token system:

1. **Access token** — Short-lived (15 minutes), used for API requests
2. **Refresh token** — Long-lived (7 days), used to generate new access tokens

Both tokens are stored as HTTP-only, Secure, SameSite=Strict cookies. The refresh token uses single-use rotation to prevent replay attacks.

### Can I use SSO?

Yes. Whatomate supports OAuth2-based Single Sign-On. Configure SSO providers in the admin settings or via the API at `/api/settings/sso`.

### How do I invite team members?

Admins can invite users through the registration flow. An invitation token (JWT) is generated and shared with the new user, who uses it during registration at `POST /api/auth/register`.

## Messaging & Campaigns

### How do campaigns work?

Campaigns send template messages to a list of recipients:

1. Create a campaign with a template and settings
2. Import recipients via CSV/JSON
3. Start the campaign — recipients are published to a Redis queue
4. Background workers process jobs with randomized delays
5. Progress is tracked and broadcast via WebSocket in real time

### Can I schedule campaigns?

Yes. Set the `scheduled_at` field when creating a campaign. The campaign will start automatically at the specified time.

### How are message delays handled?

Each campaign has configurable `min_delay_seconds` and `max_delay_seconds` (default: 20-45 seconds). The actual delay for each message is randomly selected within this range to create natural-looking sending patterns.

### Can I retry failed campaign messages?

Yes. Use `POST /api/campaigns/{id}/retry-failed` to re-attempt sending to all recipients that previously failed.

## Chatbot

### How do I set up a chatbot?

1. Go to Chatbot Settings (`GET /api/chatbot/settings`)
2. Enable the chatbot and configure:
   - Greeting message
   - Fallback message
   - Business hours
   - Session timeout
3. Add keyword rules for automated responses
4. Optionally enable AI integration with an API key

### Does the chatbot support AI?

Yes. Whatomate supports AI-powered responses through external providers like OpenAI. Configure the AI provider, model, API key, and system prompt in the chatbot settings. The AI API key is encrypted before storage.

### What happens outside business hours?

When outside configured business hours, you can choose to:
- Disable automation and queue messages for agents
- Send a custom out-of-hours message
- Continue automation as normal

### Can the chatbot transfer to a human agent?

Yes. Chatbot flows and keyword rules can trigger agent transfers. Transfers are queued and available agents are notified via WebSocket.

## Security

### How are messages encrypted?

Whatomate uses AES-256-GCM encryption for sensitive data stored in the database, including:

- WhatsApp account access tokens
- Webhook verify tokens
- SSO client secrets
- Chatbot AI API keys
- Webhook secrets

The encryption key is configured in `config.toml` under `app.encryption_key`.

### How do I rotate the encryption key?

1. Generate a new 32-byte encryption key
2. Update `config.toml` or set `WHATOMATE_APP_ENCRYPTION_KEY`
3. Run `whatomate crypto-migrate` to re-encrypt all existing data

### Can I restrict who can send messages?

Yes. Whatomate has a comprehensive send restriction system:

- **Organization-level:** Enable strict mode, set outbound mode (inbound_only/mixed), enforce campaign draft only
- **User-level:** Restrict authorized phone numbers, allowed instances, unclaimed chat access
- **Enforcement modes:** Audit (log violations) or Enforce (block messages)

### What CSRF protection is used?

Whatomate uses the double-submit cookie pattern. A CSRF token is stored in an HTTP-only cookie (`whm_csrf`) and must be sent in the `X-CSRF-Token` header for all mutating requests (POST, PUT, DELETE, PATCH).

## Webhooks

### How do webhooks work?

Whatomate supports outbound webhooks that notify external systems about events:

1. Create a webhook with a URL and event subscriptions
2. When an event occurs (message sent, contact created, etc.), Whatomate sends a POST request to your URL
3. Payloads are signed with HMAC-SHA256 if a secret is configured
4. Delivery attempts are logged

### What events can I subscribe to?

Common events include: `message.received`, `message.sent`, `message.status_updated`, `contact.created`, `contact.assigned`, `chat.closed`, `campaign.stats_update`, and more.

### How do I verify webhook signatures?

If you configured a webhook secret, verify the `X-Webhook-Signature-256` header using HMAC-SHA256 with your secret.

## Monitoring & Operations

### How do I monitor system health?

Whatomate provides two endpoints:

- **`GET /health`** — Basic health check, returns `{"status": "ok"}`
- **`GET /ready`** — Readiness check, verifies database and Redis connectivity

Use these for load balancer health checks or Kubernetes probes.

### How do I backup my data?

Backup these components:

1. **PostgreSQL database** — Use `pg_dump` or your database provider's backup tool
2. **Redis** — Use `redis-cli BGSAVE` or enable RDB/AOF persistence
3. **Media files** — Backup the storage directory configured in `storage.local_path`
4. **Configuration** — Keep a copy of `config.toml` and environment variables

### How do I restore from backup?

1. Restore the PostgreSQL database: `psql -d whatomate < backup.sql`
2. Restore Redis data from RDB/AOF files
3. Restore media files to the storage directory
4. Restart the Whatomate server

## Development

### How do I contribute to Whatomate?

See the [Contributing Guide](developers/contributing.md) for code style, PR process, and how to add new features.

### How do I run tests?

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run E2E tests
cd frontend && npm run test:e2e
```

### What is the API base URL?

All API endpoints are prefixed with `/api`. For example: `http://localhost:8080/api/auth/login`.

## See Also

- [Platform Overview](overview.md) — Whatomate architecture and capabilities
- [Quick Start](quickstart.md) — Get up and running
- [Troubleshooting](admins/troubleshooting.md) — Common issues and solutions
- [Release Notes](release-notes.md) — Version history
