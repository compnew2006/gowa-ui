# `generate_reply(business_id, comment, customer_name)` — Generate Contextual Reply

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Generates a smart, contextual reply based on the comment content and the business's knowledge. Detects language and routes to the appropriate specialized reply generator.

## Signature

```python
def generate_reply(self, business_id: str, comment: str, customer_name: str = None) -> str
```

## Parameters

| Parameter       | Type    | Description                     |
|-----------------|---------|---------------------------------|
| `business_id`   | `str`   | The business identifier         |
| `comment`       | `str`   | The customer's comment text     |
| `customer_name` | `str`   | Optional customer name for personalization |

## Returns

`str` — The generated reply text.

## Reply Routing Logic

| Comment Contains              | Generator Used                  |
|-------------------------------|---------------------------------|
| `price`, `cost`, `how much`, `سعر`, `كم` | `generate_price_reply()`       |
| `hours`, `open`, `time`, `ساعة`, `مواعيد` | `generate_hours_reply()`       |
| `location`, `where`, `address`, `موقع`, `عنوان` | `generate_location_reply()`    |
| `service`, `offer`, `ماذا`, `خدمات` | `generate_services_reply()`    |
| None of the above             | Default friendly greeting       |

## Example

```python
reply = manager.generate_reply(
    "bakery",
    "كم سعر الكرواسون؟",
    "أحمد"
)
# "📍 أسعارنا في Bakery:\n• كرواسون: ٣ دولار\n..."
```
