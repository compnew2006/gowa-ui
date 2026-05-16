# `get_business_memory(business_id, memory_type)` — Retrieve Business Memory

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Reads back stored memories for a business. If a specific `memory_type` is provided, returns only that type's entries. Otherwise returns all memory types.

## Signature

```python
def get_business_memory(self, business_id: str, memory_type: str = None) -> Union[list, dict]
```

## Parameters

| Parameter      | Type    | Description                                  |
|----------------|---------|----------------------------------------------|
| `business_id`  | `str`   | The business identifier                      |
| `memory_type`  | `str`   | Optional filter (`"post"`, `"reply"`, `"learning"`) |

## Returns

- If `memory_type` specified: `list` of dicts
- If `memory_type` is `None`: `dict` of `{memory_type_plural: [entries]}`

## Example

```python
# Get all replies for a business
replies = manager.get_business_memory("bakery", "reply")
for r in replies:
    print(r['message'])

# Get all memory types
all_mem = manager.get_business_memory("bakery")
print(all_mem.keys())  # dict_keys(['posts', 'replies', 'learnings'])
```
