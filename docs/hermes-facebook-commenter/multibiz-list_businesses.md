# `list_businesses()` — List All Configured Businesses

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Returns a summary list of all configured businesses with key metadata.

## Signature

```python
def list_businesses(self) -> List[dict]
```

## Returns

`list` of dicts, each containing:
- `id` — Business identifier
- `name` — Business name
- `page_name` — Facebook page name
- `page_id` — Facebook Page ID
- `auto_reply` — Auto-reply enabled?
- `auto_post` — Auto-post enabled?

## Example

```python
for b in manager.list_businesses():
    print(f"{b['name']} (auto-reply: {b['auto_reply']})")
```
