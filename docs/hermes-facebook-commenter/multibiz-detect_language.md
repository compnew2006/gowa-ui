# `detect_language(text)` — Detect Arabic or English

**File:** `multi_business_facebook.py`  
**Class:** `MultiBusinessFacebookManager`

## Description

Detects whether a text is primarily Arabic or English by counting Arabic Unicode characters (U+0600–U+06FF).

## Signature

```python
def detect_language(self, text: str) -> str
```

## Parameters

| Parameter | Type    | Description              |
|-----------|---------|--------------------------|
| `text`    | `str`   | The text to analyze      |

## Returns

`'ar'` if more than 30% of characters are Arabic, `'en'` otherwise.

## Example

```python
lang = manager.detect_language("مرحباً")
# 'ar'

lang = manager.detect_language("Hello")
# 'en'
```
