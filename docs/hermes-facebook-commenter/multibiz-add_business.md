# `add_business(business_id, config)` — Add a New Business

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Adds a new business by saving its configuration as a JSON file to `~/.hermes/businesses/{business_id}.json` and creating a `BusinessConfig` instance in memory.

## Signature

```python
def add_business(self, business_id: str, config: dict) -> bool
```

## Parameters

| Parameter     | Type    | Description                              |
|---------------|---------|------------------------------------------|
| `business_id` | `str`   | Unique identifier (used as filename)     |
| `config`      | `dict`  | Business configuration (see BusinessConfig) |

## Returns

`True` on success, `False` on failure.

## Example

```python
config = {
    "name": "My New Shop",
    "page_id": "123",
    "page_access_token": "EAA...",
    "services": ["Printing"],
    "prices": {"Business Card": "50 EGP"},
    "location": {"address": "Cairo", "phone": "01000"},
    "hours": {"general": "Daily 9am-9pm"},
    "tone": "professional"
}
manager.add_business("my_shop", config)
```
