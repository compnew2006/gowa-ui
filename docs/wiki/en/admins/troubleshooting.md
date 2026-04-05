---
title: Troubleshooting
---

# Troubleshooting

Common issues and their resolutions for Whatomate deployments.

## Instance Connection Problems

### Symptom: WhatsApp instance shows "disconnected" or "qr" status

**WhatsMeow instances:**

1. Check instance status:
   ```bash
   curl https://whatomate.example.com/api/instances/{id}/health \
     -H "Authorization: Bearer <token>"
   ```

2. If status is "qr", scan the QR code:
   ```bash
   curl https://whatomate.example.com/api/instances/{id}/qr \
     -H "Authorization: Bearer <token>"
   ```

3. If status is "disconnected", reconnect:
   ```bash
   curl -X POST https://whatomate.example.com/api/instances/{id}/connect \
     -H "Authorization: Bearer <token>"
   ```

4. Check server logs for connection errors:
   ```bash
   docker compose logs whatomate | grep "whatsmeow"
   ```

**Common causes:**
- WhatsApp session expired (need to re-scan QR)
- Network connectivity issues to WhatsApp servers
- Rate limiting from WhatsApp
- Multiple instances using the same session

### Symptom: Instance reconnect loop

Check the status reconciliation worker logs. On startup, stale statuses are cleaned up within 30 seconds. If the loop persists:

1. Disconnect the instance:
   ```bash
   curl -X POST https://whatomate.example.com/api/instances/{id}/disconnect \
     -H "Authorization: Bearer <token>"
   ```

2. Wait 10 seconds, then reconnect

## Webhook Delivery Failures

### Symptom: Meta webhooks not being received

1. Verify webhook URL is accessible from the internet
2. Check webhook verification token matches:
   ```bash
   # In config.toml
   [whatsapp]
   webhook_verify_token = "your-token"
   ```

3. Verify the webhook subscription in Meta:
   ```bash
   curl -X POST https://whatomate.example.com/api/accounts/{id}/subscribe \
     -H "Authorization: Bearer <token>"
   ```

4. Check server logs for webhook errors:
   ```bash
   docker compose logs whatomate | grep "webhook"
   ```

### Symptom: Outbound webhooks not being delivered

1. Check webhook configuration:
   ```bash
   curl https://whatomate.example.com/api/webhooks \
     -H "Authorization: Bearer <token>"
   ```

2. Test webhook delivery:
   ```bash
   curl -X POST https://whatomate.example.com/api/webhooks/{id}/test \
     -H "Authorization: Bearer <token>"
   ```

3. Verify the target URL is accessible and responding to POST requests
4. Check if the webhook secret is correctly encrypted

## Campaign Processing Issues

### Symptom: Campaign stuck in "running" state

1. Check campaign stats:
   ```bash
   curl https://whatomate.example.com/api/campaigns/{id} \
     -H "Authorization: Bearer <token>"
   ```

2. Check Redis queue depth:
   ```bash
   redis-cli -h <redis-host> -a <password> LLEN campaign_queue
   ```

3. Check worker logs:
   ```bash
   docker compose logs whatomate | grep "campaign"
   ```

4. If workers are not processing, restart the application

### Symptom: Campaign messages failing

1. Check template status (must be "approved"):
   ```bash
   curl https://whatomate.example.com/api/templates/{id} \
     -H "Authorization: Bearer <token>"
   ```

2. Check account status (must be active):
   ```bash
   curl https://whatomate.example.com/api/accounts/{id} \
     -H "Authorization: Bearer <token>"
   ```

3. Check send restrictions:
   - Is `outbound_mode` set to "inbound_only"?
   - Are user send restrictions blocking the campaign?
   - Is `campaign_draft_only` enabled?

4. Retry failed recipients:
   ```bash
   curl -X POST https://whatomate.example.com/api/campaigns/{id}/retry-failed \
     -H "Authorization: Bearer <token>"
   ```

## Rate Limiting Errors

### Symptom: HTTP 429 Too Many Requests

1. Check rate limit configuration:
   ```toml
   [rate_limit]
   enabled = true
   per_user = 1000
   per_ip = 100
   ```

2. Identify the rate-limited endpoint from the response headers:
   ```
   X-RateLimit-Limit: 100
   X-RateLimit-Remaining: 0
   X-RateLimit-Reset: 1700000000
   ```

3. Wait for the rate limit window to reset, or increase limits in config

4. For auth endpoint rate limiting, check Redis:
   ```bash
   redis-cli -h <redis-host> -a <password> KEYS "ratelimit:*"
   ```

## Authentication Failures

### Symptom: Login returns 401 Unauthorized

1. Verify credentials are correct
2. Check if the user account is active (`is_active = true`)
3. Verify the user belongs to the target organization
4. Check database connectivity

### Symptom: Token refresh fails

1. Check Redis connectivity (refresh tokens are stored in Redis)
2. Verify JWT secret hasn't changed
3. Check if the refresh token has expired (7-day default)
4. Clear browser cookies and log in again

### Symptom: WebSocket authentication fails

1. Ensure the WebSocket token is obtained first:
   ```bash
   curl https://whatomate.example.com/api/auth/ws-token \
     -H "Authorization: Bearer <token>"
   ```

2. WebSocket tokens expire after 30 seconds — obtain a fresh one before connecting
3. Check that the token is passed as a query parameter: `?token=<ws_token>`

## Database Connection Problems

### Symptom: Application fails to start with database error

1. Verify database credentials in config:
   ```toml
   [database]
   host = "127.0.0.1"
   port = 5432
   user = "whatomate"
   password = "secure_password"
   dbname = "whatomate"
   ```

2. Test database connectivity:
   ```bash
   psql -h <host> -U <user> -d <dbname> -c "SELECT 1"
   ```

3. Check if PostgreSQL is running:
   ```bash
   docker compose ps postgres
   ```

4. Check PostgreSQL logs:
   ```bash
   docker compose logs postgres
   ```

5. Verify the database exists and the user has proper permissions

## Redis Connection Issues

### Symptom: Cache misses, rate limiting not working, refresh tokens failing

1. Verify Redis configuration:
   ```toml
   [redis]
   host = "127.0.0.1"
   port = 6379
   password = ""
   db = 0
   ```

2. Test Redis connectivity:
   ```bash
   redis-cli -h <host> -p <port> -a <password> ping
   ```

3. Check Redis is running:
   ```bash
   docker compose ps redis
   ```

4. Check Redis logs:
   ```bash
   docker compose logs redis
   ```

5. If Redis password changed, update config and restart

## Crypto Migration Guide

### When to Run Crypto Migration

Run the crypto migration when:
- Upgrading from an older version with legacy encryption
- Changing the encryption key
- Seeing `enc:` or `enc2:` prefixed values in the database

### Running the Migration

```bash
# Dry run (preview changes)
whatomate crypto-migrate -dry-run

# Include enc2 format
whatomate crypto-migrate -include-enc2

# Custom batch size
whatomate crypto-migrate -batch-size 500

# Execute migration
whatomate crypto-migrate
```

### Migration Output

```
Crypto Migration Report
======================
Total records scanned: 1500
Records updated (enc → enc3): 1200
Records updated (enc2 → enc3): 250
Records already enc3: 50
Failed: 0
```

### Troubleshooting Migration Failures

If migration fails:
1. Verify the current encryption key is correct
2. Check the `-include-enc2` flag if you have enc2 data
3. Run with `-dry-run` first to preview changes
4. Check database connectivity and permissions

## See Also

- [Monitoring](monitoring.md) — Health checks and metrics
- [Data Migration](data-migration.md) — Database and crypto migrations
- [Configuration](configuration.md) — Configuration troubleshooting
- [Deployment](deployment.md) — Deployment-related issues
