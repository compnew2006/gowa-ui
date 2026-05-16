# `generate_location_reply(business, language)` — Generate Location Reply

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Generates a reply about the business's location/address in the appropriate language.

## Signature

```python
def generate_location_reply(self, business: BusinessConfig, language: str) -> str
```

## Parameters

| Parameter  | Type            | Description                |
|------------|-----------------|----------------------------|
| `business` | `BusinessConfig`| The business configuration |
| `language` | `str`           | `'ar'` or `'en'`          |

## Returns

`str` — Formatted location reply.

## Example

```python
# Arabic
reply = manager.generate_location_reply(business, 'ar')
# "📍 موقعنا: دشنا، قنا، مصر"

# English
reply = manager.generate_location_reply(business, 'en')
# "📍 Our location: Dashna, Qena, Egypt"
```
