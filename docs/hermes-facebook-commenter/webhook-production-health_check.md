# `health_check()` (GET) — Health Check Endpoint

**File:** `facebook_webhook_gunicorn.py`  
**Endpoint:** `GET /health`

## Description

Production health check endpoint. Returns service status and configuration state.

## Signature

```python
@app.route('/health', methods=['GET'])
def health_check()
```

## Returns

```json
{
  "status": "healthy",
  "service": "facebook-webhook",
  "page_configured": true
}
```

- Uses `logged` instead of `print` (production logging)
- Returns `200` always
