# `MultiBusinessFacebookManager.__init__()` — Initialize Manager

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Creates the multi-business manager. Sets up config and memory directories under `~/.hermes/businesses/`, then loads all existing business configurations from JSON files.

## Signature

```python
def __init__(self)
```

## What It Does

1. Creates `~/.hermes/businesses/` directory (if not exists)
2. Creates `~/.hermes/businesses/memory/` directory (if not exists)
3. Calls `load_all_businesses()` to load all `.json` files

## Directory Structure

```
~/.hermes/businesses/
├── maktabat_al_arkan.json       # Business config
├── restaurant.json              # Business config
└── memory/
    ├── maktabat_al_arkan_posts.jsonl
    ├── maktabat_al_arkan_replies.jsonl
    └── restaurant_posts.jsonl
```

## Example

```python
from multi_business_facebook import MultiBusinessFacebookManager
manager = MultiBusinessFacebookManager()
# ✓ Loaded business: Maktabat Al-Arkan
# ✓ Loaded business: Restaurant
```
