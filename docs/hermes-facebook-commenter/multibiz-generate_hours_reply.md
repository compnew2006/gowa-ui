# `generate_hours_reply(business, language)` — Generate Business Hours Reply

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Generates a reply about the business's operating hours in the appropriate language.

## Signature

```python
def generate_hours_reply(self, business: BusinessConfig, language: str) -> str
```

## Parameters

| Parameter  | Type            | Description                |
|------------|-----------------|----------------------------|
| `business` | `BusinessConfig`| The business configuration |
| `language` | `str`           | `'ar'` or `'en'`          |

## Returns

`str` — Formatted hours reply.

## Example

```python
# Arabic
reply = manager.generate_hours_reply(business, 'ar')
# "⏰ ساعات العمل في مكتبة الأركان: يومياً ٩ ص - ٩ م"

# English
reply = manager.generate_hours_reply(business, 'en')
# "⏰ Business hours at Maktabat Al-Arkan: Daily 9am - 9pm"
```
