# `add_business_interactive()` — Interactive Business Addition

**File:** `add_business.py`

## Description

Runs an interactive wizard that collects all business information via command-line prompts and saves it as a JSON configuration file.

## Signature

```python
def add_business_interactive()
```

## Interactive Prompts

| Prompt                    | Field              |
|---------------------------|--------------------|
| Business ID               | `business_id` (used as filename) |
| Business name             | `name`             |
| Facebook page name        | `page_name`        |
| Facebook Page ID          | `page_id`          |
| Facebook Page Access Token| `page_access_token`|
| Enable auto-reply?        | `auto_reply`       |
| Enable auto-post?         | `auto_post`        |
| Services (one per line)   | `services[]`       |
| Prices (item - price)     | `prices{}`         |
| Address                   | `location.address`  |
| Landmark                  | `location.landmark` |
| Phone                     | `location.phone`   |
| General hours             | `hours.general`    |

## Output

Saves config to `~/.hermes/businesses/{business_id}.json`

## Example Run

```
$ python3 add_business.py

🏪 Add New Business to Multi-Business Facebook Manager
============================================================

Business ID (unique identifier, e.g., maktabat_al_arkan): my_bakery
Business name (e.g., Maktabat Al-Arkan): My Bakery
...
✅ Business 'My Bakery' added successfully!
📁 Config saved to: /root/.hermes/businesses/my_bakery.json
```
