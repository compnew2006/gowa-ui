# `load_all_businesses()` — Load All Business Configurations

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Scans the config directory (`~/.hermes/businesses/`) for all `.json` files and loads each one as a `BusinessConfig` instance into the `self.businesses` dictionary.

## Signature

```python
def load_all_businesses(self)
```

## Behavior

- Iterates over `*.json` files in `~/.hermes/businesses/`
- Uses the filename stem (without `.json`) as `business_id`
- On success: prints `✓ Loaded business: {name}`
- On failure: prints `✗ Failed to load {business_id}: {error}`

## Example

```python
manager.load_all_businesses()
# ✓ Loaded business: Maktabat Al-Arkan
# ✓ Loaded business: Restaurant
```
