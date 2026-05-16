# `generate_price_reply(business, language)` — Generate Price Reply

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Generates a formatted reply listing all prices from the business's config, in the detected language.

## Signature

```python
def generate_price_reply(self, business: BusinessConfig, language: str) -> str
```

## Parameters

| Parameter  | Type            | Description                |
|------------|-----------------|----------------------------|
| `business` | `BusinessConfig`| The business configuration |
| `language` | `str`           | `'ar'` for Arabic, `'en'` for English |

## Returns

`str` — Formatted price list.

## Example Output (Arabic)

```
📍 أسعارنا في مكتبة الأركان:

• بطاقة أعمال: ٥٠ جنيه
• بانر للمتر: ٣٥ جنيه

للاستفسار عن خدمات أخرى، تواصل معنا! 📞
```

## Example Output (English)

```
📍 Our prices at Maktabat Al-Arkan:

• Business Card: 50 EGP
• Banner per meter: 35 EGP

For other services, contact us! 📞
```
