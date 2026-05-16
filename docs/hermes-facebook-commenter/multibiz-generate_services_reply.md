# `generate_services_reply(business, language)` — Generate Services Reply

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Generates a formatted reply listing all services from the business's configuration.

## Signature

```python
def generate_services_reply(self, business: BusinessConfig, language: str) -> str
```

## Parameters

| Parameter  | Type            | Description                |
|------------|-----------------|----------------------------|
| `business` | `BusinessConfig`| The business configuration |
| `language` | `str`           | `'ar'` or `'en'`          |

## Returns

`str` — Formatted services list.

## Example Output (Arabic)

```
✨ خدماتنا في مكتبة الأركان:

• طباعة بطاقات أعمال
• طباعة بانرات
• تصميم جرافيك
```

## Example Output (English)

```
✨ Our services at Maktabat Al-Arkan:

• Business Card Printing
• Banner Printing
• Graphic Design
```
