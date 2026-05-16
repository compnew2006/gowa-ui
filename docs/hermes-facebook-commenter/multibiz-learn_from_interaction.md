# `learn_from_interaction(business_id, interaction)` — Learn from Customer Interactions

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Processes a customer interaction to extract business knowledge. Strengthens associations between services mentioned in comments and stores learning data.

## Signature

```python
def learn_from_interaction(self, business_id: str, interaction: dict)
```

## Parameters

| Parameter      | Type    | Description                                  |
|----------------|---------|----------------------------------------------|
| `business_id`  | `str`   | The business identifier                      |
| `interaction`  | `dict`  | Interaction data containing `message`, `from`, `timestamp` |

## Behavior

- Checks if any known services appear in the message (intent to strengthen association)
- Checks if any known price keywords appear in the message (intent to confirm pricing)
- Saves a "learning" memory entry

## Example

```python
manager.learn_from_interaction("bakery", {
    "message": "How much is the croissant?",
    "from": "John",
    "timestamp": "2026-05-16T10:00:00"
})
```
