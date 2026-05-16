# `get_business(business_id)` — Get a Business by ID

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Retrieves a `BusinessConfig` instance by its unique business identifier.

## Signature

```python
def get_business(self, business_id: str) -> Optional[BusinessConfig]
```

## Parameters

| Parameter     | Type   | Description                |
|---------------|--------|----------------------------|
| `business_id` | `str`  | Unique business identifier |

## Returns

`BusinessConfig` if found, `None` otherwise.

## Example

```python
business = manager.get_business("maktabat_al_arkan")
if business:
    print(business.name)
    # "مكتبة الأركان"
```
