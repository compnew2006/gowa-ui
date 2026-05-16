# `list_businesses()` (GET) — List All Businesses Endpoint

**File:** `multi_business_webhook.py`  
**Endpoint:** `GET /businesses`

## Description

Lists all configured businesses with their IDs and metadata.

## Signature

```python
@app.route('/businesses', methods=['GET'])
def list_businesses()
```

## Returns

```json
{
  "total": 2,
  "businesses": [
    {"id": "bakery", "name": "Bakery", "page_name": "...", "auto_reply": true, ...}
  ]
}
```
