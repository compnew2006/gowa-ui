# `reply_to_comment(business_id, comment_id, message)` — Reply as a Business

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Replies to a specific comment using a specific business's Facebook page identity and access token. The reply is saved to the business's memory.

## Signature

```python
def reply_to_comment(self, business_id: str, comment_id: str, message: str) -> dict
```

## Parameters

| Parameter     | Type    | Description                     |
|---------------|---------|---------------------------------|
| `business_id` | `str`   | The business identifier         |
| `comment_id`  | `str`   | The comment ID to reply to      |
| `message`     | `str`   | The reply text                  |

## Returns

`dict` — Graph API response or `{"error": "Business not found"}`.

## Side Effects

- Saves a "reply" memory entry to `{business_id}_replies.jsonl`

## Example

```python
result = manager.reply_to_comment(
    "maktabat_al_arkan",
    "987654321",
    "شكراً لتواصلك معنا!"
)
```
