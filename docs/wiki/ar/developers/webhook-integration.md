---
title: تكامل الويب هوك
rtl: true
lang: ar
---

<div dir="rtl">تكامل الويب هوك</div>

يوفر Whatomate نظام ويب هوك صادر يُبلغ الخدمات الخارجية بالأحداث التي تحدث داخل المنصة.

## نظرة عامة

عند حدوث أحداث (إرسال/استلام رسائل، إنشاء جهات اتصال، تحديث حملات، إلخ)، يُرسل Whatomate طلبات HTTP POST إلى عناوين URL المُعدّة للويب هوك. يحدد كل اشتراك ويب هوك الأحداث التي يرغب في استلامها.

## أنواع الأحداث

| الحدث | المُشغّل |
|-------|---------|
| `message.received` | استلام رسالة واردة |
| `message.sent` | إرسال رسالة صادرة |
| `message.delivered` | تأكيد تسليم الرسالة |
| `message.read` | قراءة الرسالة من المستلم |
| `message.failed` | فشل إرسال الرسالة |
| `message.status_updated` | أي تغيير في حالة الرسالة |
| `contact.created` | إنشاء جهة اتصال جديدة |
| `contact.updated` | تغيير تفاصيل جهة الاتصال |
| `contact.assigned` | تعيين جهة اتصال لمستخدم |
| `contact.reassigned` | إعادة تعيين جهة اتصال لمستخدم مختلف |
| `chat.closed` | إغلاق جلسة المحادثة |
| `chat.reopened` | إعادة فتح جلسة المحادثة |
| `campaign.started` | بدء تنفيذ الحملة |
| `campaign.paused` | إيقاف الحملة مؤقتاً |
| `campaign.completed` | انتهاء الحملة |
| `campaign.cancelled` | إلغاء الحملة |
| `user.created` | إضافة مستخدم جديد للمؤسسة |
| `user.updated` | تغيير تفاصيل المستخدم |
| `lead_request.created` | تقديم طلب عميل محتمل جديد |
| `sla.breached` | تجاوز عتبة SLA |

## تنسيق الحمولة

تتبع جميع حمولات الويب هوك بنية متسقة:

```json
{
  "event": "message.received",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 123,
    "contact_id": 456,
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

### الحمولة حسب نوع الحدث

#### message.received

```json
{
  "event": "message.received",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 123,
    "contact_id": 456,
    "contact_name": "John Doe",
    "contact_phone": "+1234567890",
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "account_id": 1,
    "instance_id": null
  }
}
```

#### message.sent

```json
{
  "event": "message.sent",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 124,
    "contact_id": 456,
    "content": "Hi there!",
    "direction": "outbound",
    "status": "sent",
    "provider_message_id": "wamid.HBg...",
    "sent_by": {
      "id": 5,
      "name": "Agent Smith"
    }
  }
}
```

#### contact.created

```json
{
  "event": "contact.created",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 789,
    "phone_number": "+1234567890",
    "name": "New Contact",
    "status": "open",
    "tags": []
  }
}
```

#### campaign.started

```json
{
  "event": "campaign.started",
  "timestamp": "2026-01-01T00:00:00Z",
  "organization_id": 1,
  "data": {
    "id": 10,
    "name": "Holiday Promotion",
    "total_recipients": 500,
    "template_id": 5,
    "started_at": "2026-01-01T00:00:00Z"
  }
}
```

## التحقق من توقيع HMAC

يمكن تكوين الويب هوك بسري للتحقق من الحمولة. يوقّع Whatomate كل طلب باستخدام HMAC-SHA256:

```
X-Webhook-Signature: sha256=<hex-hmac>
```

### مثال التحقق (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(`sha256=${expected}`)
  );
}

// الاستخدام في Express
app.post('/webhook', (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const rawBody = JSON.stringify(req.body);
  
  if (!verifyWebhook(rawBody, signature, process.env.WEBHOOK_SECRET)) {
    return res.status(401).send('Invalid signature');
  }
  
  // معالجة الويب هوك
  handleEvent(req.body.event, req.body.data);
  res.status(200).send('OK');
});
```

### مثال التحقق (Go)

```go
func verifyWebhook(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := fmt.Sprintf("sha256=%x", mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

## محاولات التسليم

يحاول Whatomate تسليم الويب هوك بالسلوك التالي:

| المحاولة | التأخير | ملاحظات |
|----------|---------|---------|
| 1 | فوري | إرسال متزامن |
| 2 | 30 ثانية | إذا فشلت المحاولة الأولى |
| 3 | 5 دقائق | إذا فشلت المحاولة الثانية |
| 4 | 30 دقيقة | إذا فشلت المحاولة الثالثة |
| 5 | ساعتان | المحاولة الأخيرة |

بعد 5 محاولات فاشلة، يتم وضع علامة على الويب هوك كمتدهور ولا تتم محاولات إضافية حتى يتم إعادة تمكينه يدوياً.

### متطلبات الاستجابة

- يجب أن تستجيب نقطة نهاية الويب هوك برمز حالة `2xx`
- مهلة الاستجابة: 10 ثوانٍ
- تُحسب استجابات غير 2xx كفشل

## اختبار الويب هوك

يمكنك اختبار تكوين الويب هوك عبر API:

```
POST /api/webhooks/{id}/test
```

**الاستجابة (200):**
```json
{
  "success": true,
  "status_code": 200,
  "response_time_ms": 145,
  "message": "Webhook delivered successfully"
}
```

**الاستجابة (400):**
```json
{
  "success": false,
  "status_code": 500,
  "response_time_ms": 5023,
  "message": "Webhook delivery failed: Internal Server Error"
}
```

## API إدارة الويب هوك

### سرد الويب هوك

```
GET /api/webhooks
```

**الاستجابة (200):**
```json
{
  "webhooks": [
    {
      "id": 1,
      "url": "https://example.com/webhook",
      "events": ["message.received", "message.sent"],
      "enabled": true,
      "last_triggered": "2026-01-01T00:00:00Z",
      "last_status": "success",
      "created_at": "2025-06-01T00:00:00Z"
    }
  ]
}
```

### إنشاء ويب هوك

```
POST /api/webhooks
```

**جسم الطلب:**
```json
{
  "url": "https://example.com/webhook",
  "events": ["message.received", "message.sent", "contact.created"],
  "secret": "your-hmac-secret",
  "enabled": true
}
```

### تحديث ويب هوك

```
PUT /api/webhooks/{id}
```

### حذف ويب هوك

```
DELETE /api/webhooks/{id}
```

## تنفيذ الإرسال

يتم التعامل مع إرسال الويب هوك بشكل غير متزامن:

```go
func (app *App) DispatchWebhook(event string, data interface{}) {
    // البحث عن الويب هوك المفعّلة لهذا الحدث
    var webhooks []models.Webhook
    app.DB.Where("enabled = ? AND events @> ?", true, pq.Array([]string{event})).
        Find(&webhooks)
    
    for _, wh := range webhooks {
        go app.sendWebhook(wh, event, data)
    }
}

func (app *App) sendWebhook(wh models.Webhook, event string, data interface{}) {
    payload := WebhookPayload{
        Event:          event,
        Timestamp:      time.Now().UTC(),
        OrganizationID: wh.OrganizationID,
        Data:           data,
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    // التوقيع بـ HMAC إذا كان السري مُعدّاً
    if wh.Secret != "" {
        secret := decrypt(wh.Secret)
        mac := hmac.New(sha256.New, []byte(secret))
        mac.Write(body)
        req.Header.Set("X-Webhook-Signature", fmt.Sprintf("sha256=%x", mac.Sum(nil)))
    }
    
    // الإرسال مع منطق إعادة المحاولة
    app.sendWithRetry(req, body)
}
```

## اعتبارات الأمان

1. **تحقق دائماً من توقيعات HMAC** — لا تعالج الويب هوك بدون التحقق في الإنتاج
2. **استخدم نقاط نهاية HTTPS** — يجب أن تستخدم عناوين الويب هوك TLS
3. **تحقق من بنية الحمولة** — تحقق من أنواع الأحداث والحقول المطلوبة
4. **نفّذ التفرد** — استخدم طابع الحمولة الزمني وبيانات الحدث لاكتشاف التكرارات
5. **استجب بسرعة** — عالج بشكل غير متزامن وأعد 200 فوراً

## انظر أيضاً

- [مرجع واجهة البرمجة (API)](./api-reference) — نقاط نهاية الويب هوك
- [البنية المعمارية](./architecture) — مخططات تدفق البيانات
