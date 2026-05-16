# `forward_to_hermes(business_id, payload)` — Forward Event to Hermes (Multi-Business)

**File:** `multi_business_webhook.py`

## Description

Forwards a business-scoped webhook event to the Hermes API for additional processing.

## Signature

```python
def forward_to_hermes(business_id, payload)
```

## Parameters

| Parameter     | Type    | Description                    |
|---------------|---------|--------------------------------|
| `business_id` | `str`   | Business identifier for logging|
| `payload`     | `dict`  | Event data with business context|

## Forwarded Payload Includes

- `business_id` and `business_name` for context
- All standard comment/event fields
