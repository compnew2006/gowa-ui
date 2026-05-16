# `main()` — Entry Point with CLI Options

**File:** `add_business.py`

## Description

Entry point that supports three modes: interactive business addition, listing all businesses, and testing business auto-replies.

## Signature

```python
def main()
```

## CLI Usage

```bash
# Interactive mode (add a business)
python3 add_business.py

# List all businesses
python3 add_business.py --list

# Test a business's auto-replies
python3 add_business.py --test <business_id>
```

## `--list` Output Example

```
📊 Total businesses: 2

🏪 Maktabat Al-Arkan
   ID: maktabat_al_arkan
   Page: مكتبة الأركان
   Auto-reply: ✅
   Auto-post: ❌
...
```

## `--test` Behavior

1. Loads business from config
2. Shows business info (services, prices)
3. Tests auto-reply generation with sample queries:
   - "كم سعر البطاقة؟"
   - "متى تفتحون؟"
   - "أين تقعون؟"
4. Displays generated replies

## Example Test Output

```
🏪 Testing business: Maktabat Al-Arkan
   Services: طباعة بطاقات أعمال, طباعة بانرات...
   Prices: 2 items

🤖 Testing auto-replies:

   Question: كم سعر البطاقة؟
   Reply: 📍 أسعارنا في مكتبة الأركان:\n...
```
