# `health_check()` (GET) — Health Check (Multi-Business)

**File:** `multi_business_webhook.py`  
**Endpoint:** `GET /health`

## Description

Returns service health status including the number of configured businesses and their details.

## Signature

```python
@app.route('/health', methods=['GET'])
def health_check()
```

## Returns

```json
{
  "status": "healthy",
  "service": "multi-business-facebook-webhook",
  "total_businesses": 2,
  "businesses": [
    {"id": "bakery", "name": "Bakery", "auto_reply": true},
    {"id": "restaurant", "name": "Restaurant", "auto_reply": true}
  ]
}
```
