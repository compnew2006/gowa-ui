# 🏪 Multi-Business Facebook Manager - Complete Guide

## 🎯 What This Does

Manage **multiple Facebook pages** for **different businesses**, where each business:
- ✅ Has its own Facebook page and access token
- ✅ Auto-replies to comments with **business-specific knowledge**
- ✅ Auto-posts content on schedule
- ✅ **Learns independently** from each business's interactions
- ✅ Maintains **separate memory** for each business
- ✅ Replies in the right language (Arabic/English) automatically

## 🚀 Quick Setup

### 1. Add Your First Business

```bash
cd ~/.hermes/plugins
python3 add_business.py
```

Follow the prompts to add:
- Business name
- Facebook Page ID
- Page Access Token
- Services
- Prices
- Location
- Hours

### 2. Add More Businesses

Run the same command again for each business:

```bash
python3 add_business.py
```

Each business gets its own configuration file at `~/.hermes/businesses/BUSINESS_ID.json`

### 3. List All Businesses

```bash
python3 add_business.py --list
```

### 4. Test a Business

```bash
python3 add_business.py --test maktabat_al_arkan
```

## 📁 Business Configuration Structure

Each business has a JSON config file:

```json
{
  "name": "مكتبة الأركان",
  "page_id": "123456789",
  "page_name": "Maktabat Al-Arkan",
  "page_access_token": "your_token_here",
  "auto_reply": true,
  "auto_post": true,
  "services": [
    "طباعة بطاقات أعمال",
    "طباعة بانرات"
  ],
  "prices": {
    "بطاقة أعمال": "٥٠ جنيه",
    "بانر للمتر": "٣٥ جنيه"
  },
  "location": {
    "address": "دشنا، قنا، مصر",
    "phone": "٠١٠٠٠٠٠٠٠٠٠"
  },
  "hours": {
    "general": "يومياً ٩ ص - ٩ م"
  },
  "tone": "friendly_professional"
}
```

## 🤖 How It Works

### Auto-Reply Logic

When someone comments on ANY of your business pages:

1. **Detect which business** the page belongs to
2. **Read the comment** and detect language (Arabic/English)
3. **Generate contextual reply** based on that business's knowledge:
   - Prices query → Reply with that business's prices
   - Hours query → Reply with that business's hours
   - Location query → Reply with that business's address
   - Service query → Reply with that business's services
4. **Post the reply** as that business
5. **Learn from interaction** → Save to that business's memory

### Example Interactions

**Business 1: Maktabat Al-Arkan (Printing Services)**
```
Customer: "كم سعر البطاقة؟"
Bot: "📍 أسعارنا في مكتبة الأركان:
      • بطاقة أعمال: ٥٠ جنيه
      • بانر للمتر: ٣٥ جنيه"
```

**Business 2: Al-Arkan Restaurant**
```
Customer: "كم سعر الوجبة؟"
Bot: "📍 أسعارنا في مطعم الأركان:
      • وجبة فردية: من ٨٠ جنيه
      • مشويات: ١٥٠ جنيه"
```

Each business replies with **its own** information! 🎯

## 🧠 Independent Learning

Each business **learns separately**:

```
~/.hermes/businesses/memory/
├── maktabat_al_arkan_posts.jsonl      # Business 1 posts
├── maktabat_al_arkan_replies.jsonl    # Business 1 replies
├── maktabat_al_arkan_learnings.jsonl  # Business 1 learnings
├── restaurant_posts.jsonl              # Business 2 posts
├── restaurant_replies.jsonl            # Business 2 replies
└── restaurant_learnings.jsonl          # Business 2 learnings
```

No cross-contamination! Each business has its own brain 🧠

## 📡 Webhook Setup

### Single Webhook for All Pages

You need **ONE webhook** for all your businesses:

```
https://fbwebhook.ofuqalmadenah.com/webhook
```

This single webhook:
- Receives events from ALL your Facebook pages
- Automatically routes to the correct business
- Uses the correct page token for replies
- Maintains separate context for each business

### Meta Developer Setup

**For each Facebook page you manage:**

1. Go to Meta Developers → Your App → Webhooks
2. Add the SAME callback URL for all pages:
   - Callback URL: `https://fbwebhook.ofuqalmadenah.com/webhook`
   - Verify Token: `hermes_multi_business_verify`
3. Subscribe each page to the webhook

### Subscribe All Pages

Run this for **each** Facebook page:

```bash
# Page 1: Maktabat Al-Arkan
curl -X POST \
  "https://graph.facebook.com/v19.0/PAGE_1_ID/subscribed_apps" \
  -d "access_token=PAGE_1_TOKEN" \
  -d "subscribed_fields=comments,feed"

# Page 2: Restaurant
curl -X POST \
  "https://graph.facebook.com/v19.0/PAGE_2_ID/subscribed_apps" \
  -d "access_token=PAGE_2_TOKEN" \
  -d "subscribed_fields=comments,feed"
```

## 🗓️ Scheduled Posting

Each business can have its own posting schedule:

```json
"post_schedule": {
  "morning_greeting": "09:00",
  "evening_post": "18:00",
  "friday_special": "10:00"
}
```

Add to crontab to post automatically:

```bash
# Business 1 - Morning post
0 9 * * * python3 -c "from multi_business_facebook import publish_post; publish_post('maktabat_al_arkan', 'صباح الخير! مكتبة الأركان ترحب بكم 🌅')"

# Business 2 - Lunch post
0 12 * * * python3 -c "from multi_business_facebook import publish_post; publish_post('restaurant', 'وجبة الغداء جاهزة! 🍽️')"
```

## 🔧 Management Commands

### Add Business Interactively
```bash
python3 add_business.py
```

### List All Businesses
```bash
python3 add_business.py --list
```

### Test Business Replies
```bash
python3 add_business.py --test maktabat_al_arkan
```

### Manual Post
```bash
python3 -c "from multi_business_facebook import publish_post; publish_post('business_id', 'Your message here')"
```

### Manual Reply
```bash
python3 -c "from multi_business_facebook import reply_to_comment; reply_to_comment('business_id', 'comment_id', 'Your reply')"
```

### Get Business Info
```bash
python3 -c "from multi_business_facebook import get_business_info; import json; print(json.dumps(get_business_info('business_id'), indent=2))"
```

## 📊 Monitoring

### Check Webhook Status
```bash
curl https://fbwebhook.ofuqalmadenah.com/health
```

Returns:
```json
{
  "status": "healthy",
  "total_businesses": 3,
  "businesses": [
    {"id": "maktabat_al_arkan", "name": "مكتبة الأركان", "auto_reply": true},
    {"id": "restaurant", "name": "مطعم الأركان", "auto_reply": true}
  ]
}
```

### View Business Events
```bash
# Business 1 events
tail -f ~/.hermes/webhook_events/business_maktabat_al_arkan_*.json

# Business 2 events
tail -f ~/.hermes/webhook_events/business_restaurant_*.json
```

### View Service Logs
```bash
sudo journalctl -u hermes-facebook-webhook -f
```

## 🎨 Customization Examples

### Retail Business
```json
{
  "name": "Al-Arkan Electronics",
  "services": ["بيع أجهزة منزلية", "صيانة", "توصيل وتركيب"],
  "prices": {"تلفزيون": "من ٥٠٠٠ جنيه", "ثلاجة": "من ٨٠٠٠ جنيه"},
  "tone": "professional"
}
```

### Restaurant
```json
{
  "name": "مطعم الأركان",
  "services": ["مشويات", "مأكولات بحرية", "حجز عائلي"],
  "prices": {"وجبة فردية": "من ٨٠ جنيه", "عشاء عائلي": "من ٣٠٠ جنيه"},
  "tone": "friendly"
}
```

### Service Provider
```json
{
  "name": "Al-Arkan Laundry",
  "services": ["غسيل مكوي", "غسيل جاف", "توصيل"],
  "prices": {"قطعة عادية": "٢٠ جنيه", "بدلة": "٦٠ جنيه"},
  "hours": {"general": "يومياً ٨ ص - ١٠ م"}
}
```

## 🌍 Multi-Language Support

Each business can:
- **Auto-detect** language (Arabic/English)
- Reply in the **same language** as the comment
- Have **different prices/content** per language

```json
"reply_language": "auto"  // auto, en, ar
```

## 🔒 Security

- ✅ Each business has its own access token
- ✅ Tokens stored in separate config files
- ✅ Webhook uses single verify token
- ✅ HTTPS only (SSL required)
- ✅ No cross-business data leakage

## 📈 Scaling

You can manage:
- ✅ **Unlimited businesses** (each with own config)
- ✅ **Unlimited pages** (one or many per business)
- ✅ **Unlimited comments** (auto-reply to all)
- ✅ **Unlimited memory** (continuous learning)

## 🎯 Example Use Cases

### 1. Chain Stores
Same business, multiple locations:
```
maktabat_dishna    → Page 1
maktabat_qena      → Page 2
maktabat_safaga    → Page 3
```

### 2. Different Businesses
Unrelated businesses:
```
printing_services  → Printing shop
restaurant         → Restaurant
laundry            → Laundry service
```

### 3. Client Management
Manage for multiple clients:
```
client1_business   → Client 1's page
client2_business   → Client 2's page
client3_business   → Client 3's page
```

## 🆘 Troubleshooting

### Business Not Replying
```bash
# Check if business is loaded
python3 -c "from multi_business_facebook import manager; print(manager.get_business('business_id'))"

# Check auto-reply is enabled
python3 add_business.py --list
```

### Wrong Business Replying
```bash
# Check page_id mapping
python3 -c "from multi_business_facebook import manager; b = manager.get_business_by_page_id('PAGE_ID'); print(b.name if b else 'Not found')"
```

### Memory Not Saving
```bash
# Check directory permissions
ls -la ~/.hermes/businesses/memory/
```

### Webhook Not Receiving
```bash
# Test webhook
curl -X POST https://fbwebhook.ofuqalmadenah.com/webhook \
  -H "Content-Type: application/json" \
  -d '{"object": "page", "entry": []}'
```

## 📞 Support

Each business operates independently, so issues with one business don't affect others!

**Perfect for:**
- 🏪 Multiple business owners
- 🏢 Chain stores
- 🤝 Client management
- 📈 Social media agencies
