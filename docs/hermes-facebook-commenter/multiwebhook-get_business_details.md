# `get_business(business_id)` (GET) — Get Business Details

**File:** `multi_business_webhook.py`  
**Endpoint:** `GET /businesses/<business_id>`

## Description

Returns detailed information for a specific business.

## Signature

```python
@app.route('/businesses/<business_id>', methods=['GET'])
def get_business(business_id)
```

## Parameters

| Parameter      | Type    | Source    | Description                |
|----------------|---------|-----------|----------------------------|
| `business_id`  | `str`   | URL path  | Unique business identifier |

## Returns

- `200` with business details (name, services, prices, location, hours, tone)
- `404` with `{"error": "Business not found"}` if not found
