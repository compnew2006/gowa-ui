# `publish_post(business_id, message)` — Publish Post for a Business

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Publishes a post on a specific business's Facebook page using that business's access token. The post action is saved to the business's memory.

## Signature

```python
def publish_post(self, business_id: str, message: str) -> dict
```

## Parameters

| Parameter     | Type    | Description                     |
|---------------|---------|---------------------------------|
| `business_id` | `str`   | The business identifier         |
| `message`     | `str`   | Post text content               |

## Returns

`dict` — Graph API response (contains `id` on success) or `{"error": "Business not found"}`.

## Side Effects

- Saves a "post" memory entry to `{business_id}_posts.jsonl`

## Example

```python
result = manager.publish_post("maktabat_al_arkan", "Special offer today!")
if "id" in result:
    print("Post published!")
```
