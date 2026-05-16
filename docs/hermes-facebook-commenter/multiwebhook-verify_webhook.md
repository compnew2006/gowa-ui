# `verify_webhook()` (GET) — Webhook Verification (Multi-Business)

**File:** `multi_business_webhook.py`  
**Endpoint:** `GET /webhook`

## Description

Same webhook verification as the standard version but with multi-business verify token. Used by Meta to validate the webhook endpoint.

## Signature

```python
@app.route('/webhook', methods=['GET'])
def verify_webhook()
```

## Verification Token

Default: `"hermes_multi_business_verify"` (from `FB_WEBHOOK_VERIFY_TOKEN` env var)

## Returns

- `200` with challenge on success
- `403` on failure
