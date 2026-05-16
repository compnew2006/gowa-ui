# `verify_webhook()` (GET) — Webhook Verification (Production)

**File:** `facebook_webhook_gunicorn.py`  
**Endpoint:** `GET /webhook`

## Description

Production-ready Meta webhook verification. Same logic as the basic version but uses structured logging instead of `print`.

## Signature

```python
@app.route('/webhook', methods=['GET'])
def verify_webhook()
```

## Query Parameters

| Parameter           | Description                                  |
|---------------------|----------------------------------------------|
| `hub.mode`          | Must be `"subscribe"`                        |
| `hub.verify_token`  | Compared against `FB_WEBHOOK_VERIFY_TOKEN` env var |
| `hub.challenge`     | Echoed back on success                       |

## Returns

- `200` with challenge on success
- `403` with `"Forbidden"` on failure
- Logs verification success/failure via `logger.info` / `logger.warning`

## Environment Variables

- `FB_WEBHOOK_VERIFY_TOKEN` — Custom verify token (optional, default: `"hermes_facebook_verify"`)
