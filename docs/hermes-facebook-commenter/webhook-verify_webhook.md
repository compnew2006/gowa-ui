# `verify_webhook()` (GET) — Meta Webhook Verification

**File:** `facebook_webhook.py`  
**Endpoint:** `GET /webhook`

## Description

Handles the Meta webhook verification challenge. Meta sends a `GET` request with `hub.mode`, `hub.verify_token`, and `hub.challenge`. This endpoint validates the token and returns the challenge to confirm the endpoint is owned by you.

## Signature

```python
@app.route('/webhook', methods=['GET'])
def verify_webhook()
```

## Query Parameters

| Parameter           | Source     | Description                            |
|---------------------|------------|----------------------------------------|
| `hub.mode`          | Meta       | Must be `"subscribe"`                  |
| `hub.verify_token`  | Meta       | Compared against `FB_WEBHOOK_VERIFY_TOKEN` env var (default: `"hermes_facebook_verify"`) |
| `hub.challenge`     | Meta       | Random string to echo back             |

## Returns

- `200` with challenge string on success
- `403` on failure

## Example Meta Setup

In the Meta Developers dashboard, enter:
- **Callback URL**: `https://your-domain.com/webhook`
- **Verify Token**: `hermes_facebook_verify`
