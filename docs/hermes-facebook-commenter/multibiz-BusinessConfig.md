# `class BusinessConfig` — Single Business Configuration

**File:** `multi_business_facebook.py`

## Description

Holds configuration data for one business/page pair. Used by `MultiBusinessFacebookManager` to store per-business settings, services, prices, location, hours, and FAQs.

## Constructor

```python
def __init__(self, business_id: str, config: dict)
```

## Parameters

| Parameter     | Type    | Description                               |
|---------------|---------|-------------------------------------------|
| `business_id` | `str`   | Unique identifier for the business        |
| `config`      | `dict`  | Configuration dictionary (from JSON file) |

## Config Fields

| Field               | Type      | Default                | Description                        |
|---------------------|-----------|------------------------|------------------------------------|
| `name`              | `str`     | `'Unknown Business'`   | Business display name              |
| `page_id`           | `str`     | `None`                 | Facebook Page ID                   |
| `page_access_token` | `str`     | `None`                 | Page-scoped access token           |
| `page_name`         | `str`     | `''`                   | Facebook page name                 |
| `auto_reply`        | `bool`    | `True`                 | Enable auto-reply on comments      |
| `auto_post`         | `bool`    | `False`                | Enable scheduled auto-posting      |
| `post_schedule`     | `dict`    | `{}`                   | Cron-style posting schedule        |
| `reply_language`    | `str`     | `'auto'`               | Reply language (`auto`, `en`, `ar`)|
| `services`          | `list`    | `[]`                   | List of service names              |
| `prices`            | `dict`    | `{}`                   | `{service_name: price}` map        |
| `location`          | `dict`    | `{}`                   | `{address, phone, landmark}`       |
| `hours`             | `dict`    | `{}`                   | `{general: "hours string"}`        |
| `faqs`              | `list`    | `[]`                   | FAQ entries                        |
| `tone`              | `str`     | `'professional'`       | Reply tone (professional, friendly)|

## Example

```python
config = BusinessConfig("bakery", {
    "name": "Al-Arkan Bakery",
    "page_id": "123456789",
    "services": ["Croissant", "Baguette"],
    "prices": {"Croissant": "$3", "Baguette": "$5"},
    "tone": "friendly"
})
```
