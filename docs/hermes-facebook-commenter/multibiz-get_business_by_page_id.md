# `get_business_by_page_id(page_id)` — Find Business by Facebook Page ID

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Looks up which business owns a given Facebook Page ID. Critical for routing incoming webhook events to the correct business context.

## Signature

```python
def get_business_by_page_id(self, page_id: str) -> Optional[BusinessConfig]
```

## Parameters

| Parameter | Type    | Description                     |
|-----------|---------|---------------------------------|
| `page_id` | `str`   | The Facebook Page ID to look up |

## Returns

`BusinessConfig` if a business with this `page_id` is found, `None` otherwise.

## Example

```python
business = manager.get_business_by_page_id("123456789")
if business:
    print(f"Found: {business.name}")
```
