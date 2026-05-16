# `get_comments(business_id, post_id)` — Get Comments for a Business Post

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Fetches all comments on a specific post for a specific business.

## Signature

```python
def get_comments(self, business_id: str, post_id: str) -> list
```

## Parameters

| Parameter     | Type    | Description                     |
|---------------|---------|---------------------------------|
| `business_id` | `str`   | The business identifier         |
| `post_id`     | `str`   | The Facebook post ID            |

## Returns

`list` — List of comment objects, or empty list if business not found.

## API Call

```
GET https://graph.facebook.com/v19.0/{post_id}/comments?fields=id,message,from,created_time
```

## Example

```python
comments = manager.get_comments("maktabat_al_arkan", "123_456")
for c in comments:
    print(c['message'])
```
