# `save_business_memory(business_id, memory_type, data)` — Save Business Memory

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Appends a JSON line to the business-specific memory file. Each business has separate memory files per type (posts, replies, learnings).

## Signature

```python
def save_business_memory(self, business_id: str, memory_type: str, data: dict)
```

## Parameters

| Parameter     | Type    | Description                              |
|---------------|---------|------------------------------------------|
| `business_id` | `str`   | The business identifier                  |
| `memory_type` | `str`   | Memory category (e.g. `"post"`, `"reply"`, `"learning"`) |
| `data`        | `dict`  | The data to save (auto-appended as JSON line) |

## File Format

```
~/.hermes/businesses/memory/{business_id}_{memory_type}s.jsonl
```

Each line is a JSON object.

## Example

```python
manager.save_business_memory("bakery", "post", {
    "message": "Fresh bread available!",
    "timestamp": "2026-05-16T10:00:00"
})
```
