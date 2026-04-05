---
title: العمليات الخلفية
rtl: true
lang: ar
---

<div dir="rtl">العمليات الخلفية</div>

يستخدم Whatomate عمال خلفيين مدعومين من Redis للمعالجة غير المتزامنة. توثّق هذه الصفحة نظام الطابور، وأنواع المهام، وعمليات العمال، وgoroutines الخلفية.

## نظام طابور Redis

**المصدر:** `internal/queue/queue.go`، `internal/queue/consumer.go`، `internal/queue/publisher.go`

### أنواع الطوابير

| الطابور | الغرض | نمط المفتاح |
|---------|-------|-------------|
| طابور الحملات | مهام رسائل الحملات | `queue:campaign` |
| طابور الوسائط الواردة | مهام تنزيل الوسائط | `queue:inbound_media` |
| Pub/Sub: إحصائيات الحملات | بث تقدم الحملات | `channel:campaign_stats` |
| Pub/Sub: الإشعارات | إشعارات النظام | `channel:notifications` |

### أنواع المهام

#### RecipientJob

```go
type RecipientJob struct {
    CampaignID    int64             `json:"campaign_id"`
    RecipientID   int64             `json:"recipient_id"`
    PhoneNumber   string            `json:"phone_number"`
    TemplateID    int64             `json:"template_id"`
    TemplateName  string            `json:"template_name"`
    BodyContent   string            `json:"body_content"`
    HeaderMediaID string            `json:"header_media_id,omitempty"`
    Params        map[string]string `json:"params,omitempty"`
    AccountID     int64             `json:"account_id"`
    MinDelay      int               `json:"min_delay"`
    MaxDelay      int               `json:"max_delay"`
}
```

#### InboundMediaJob

```go
type InboundMediaJob struct {
    MessageID   int64  `json:"message_id"`
    MediaURL    string `json:"media_url"`
    AccountID   int64  `json:"account_id"`
    PhoneNumber string `json:"phone_number"`
}
```

#### CampaignStatsUpdate

```go
type CampaignStatsUpdate struct {
    CampaignID int64 `json:"campaign_id"`
    Sent       int   `json:"sent"`
    Delivered  int   `json:"delivered"`
    Read       int   `json:"read"`
    Failed     int   `json:"failed"`
    Pending    int   `json:"pending"`
    Status     string `json:"status"`
}
```

### عمليات الناشر

```go
// نشر مهمة إلى طابور
func (p *Publisher) Publish(queue string, job interface{}) error {
    data, _ := json.Marshal(job)
    return p.client.LPush(ctx, queue, data).Err()
}

// نشر إحصائيات الحملة عبر pub/sub
func (p *Publisher) PublishCampaignStats(update *CampaignStatsUpdate) error {
    data, _ := json.Marshal(update)
    return p.client.Publish(ctx, "channel:campaign_stats", data).Err()
}
```

### عمليات المستهلك

```go
// بدء استهلاك المهام
func (c *Consumer) Consume(queue string, handler JobHandler) {
    for {
        result, err := c.client.BRPop(ctx, 0, queue).Result()
        if err != nil {
            continue
        }
        
        go func(jobData string) {
            if err := handler(jobData); err != nil {
                // إعادة محاولة مع تأخير متزايد
                c.Retry(queue, jobData)
            }
        }(result[1])
    }
}

// تأكيد المعالجة الناجحة
func (c *Consumer) Ack(jobID string) error {
    return c.client.Del(ctx, "job:processing:"+jobID).Err()
}

// رفض سلبي (إعادة إلى الطابور أو dead-letter)
func (c *Consumer) Nack(queue string, jobData string, requeue bool) error {
    if requeue {
        return c.client.RPush(ctx, queue, jobData).Err()
    }
    return c.client.LPush(ctx, "queue:dead_letter", jobData).Err()
}
```

## معالجة عامل الحملة

**المصدر:** `internal/worker/worker.go`

### معالجة مهام العامل

```
طابور Redis → Worker.HandleRecipientJob()
  1. الحصول على قفل موزّع على recipient_id
  2. تحميل سجل المستلم، التحقق من الحالة = pending
  3. تحميل الحملة، التحقق من الحالة = running
  4. تطبيق تأخير الحملة (عشوائي بين min/max)
  5. حل نائبي القالب
  6. بناء حمولة الرسالة
  7. الإرسال عبر MessageProvider
  8. تحديث حالة المستلم (sent/delivered/failed)
  9. تحديث إحصائيات الحملة
  10. نشر الإحصائيات إلى Redis pub/sub
  11. إطلاق قفل المستلم
```

### التفرد

```go
func (w *Worker) checkIdempotency(jobID string) (bool, error) {
    // التحقق مما إذا كانت قد عولجت بالفعل
    exists, err := w.redis.Exists(ctx, "job:processed:"+jobID).Result()
    if err != nil {
        return false, err
    }
    return exists > 0, nil
}

func (w *Worker) markProcessed(jobID string) error {
    // وضع علامة كمُعالَج مع انتهاء صلاحية 24 ساعة
    return w.redis.Set(ctx, "job:processed:"+jobID, "1", 24*time.Hour).Err()
}
```

### تأخير الحملة

```go
func (w *Worker) applyDelay(minDelay, maxDelay int) {
    delay := time.Duration(rand.Intn(maxDelay-minDelay+1)+minDelay) * time.Second
    time.Sleep(delay)
}
```

## فرض سياسة الإرسال

**المصدر:** `internal/worker/send_policy.go`

قبل إرسال رسالة حملة، يفرض العامل السياسات:

```go
func (w *Worker) enforceSendPolicy(ctx context.Context, job *RecipientJob) error {
    // 1. التحقق من ساعات العمل
    if !isWithinBusinessHours(job.OrgID) {
        return ErrOutsideBusinessHours
    }
    
    // 2. التحقق من قيود إرسال المستخدم
    if err := checkUserRestrictions(job.UserID, job.ContactID); err != nil {
        return err
    }
    
    // 3. التحقق من حدود المعدل
    if err := checkRateLimit(job.OrgID); err != nil {
        return err
    }
    
    return nil
}
```

## Goroutines الخلفية

يتم بدء goroutines التالية في `main.go`:

### معالج SLA

**المصدر:** `internal/handlers/sla_processor.go`

**الفاصل:** كل دقيقة واحدة

```go
func (p *SLAProcessor) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        p.checkSLABreaches()
        p.autoCloseExpiredChats()
    }
}
```

**المعالجة:**
1. تحميل إعدادات SLA لكل مؤسسة
2. لكل محادثة مفتوحة:
   - التحقق من SLA الاستجابة (الوقت منذ آخر رسالة واردة)
   - التحقق من SLA الحل (الوقت منذ فتح المحادثة)
   - التحقق من SLA التصعيد (الوقت منذ الرد الأول)
3. إذا تم خرق SLA:
   - إرسال رسالة تحذير لجهة الاتصال (إذا مُعدّة)
   - إشعار المستخدمين المتصعدين عبر WebSocket
   - التصعيد للمدير إذا تجاوز `sla_escalation_minutes`
4. الإغلاق التلقائي للمحادثات التي تجاوزت `sla_auto_close_hours`

**إعدادات SLA:**
| الإعداد | الوصف |
|---------|-------|
| `sla_response_minutes` | الحد الأقصى للوقت لأول رد |
| `sla_resolution_minutes` | الحد الأقصى لحل المحادثة |
| `sla_escalation_minutes` | الوقت قبل تصعيد المدير |
| `sla_auto_close_hours` | الساعات قبل الإغلاق التلقائي |
| `sla_escalation_notify_ids` | معرّفات المستخدمين للإشعار عند التصعيد |

### عامل الاحتفاظ بالنشاط

**المصدر:** `internal/handlers/activity_retention.go`

**الفاصل:** كل ساعة واحدة

```go
func (w *ActivityRetentionWorker) Start() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cutoff := time.Now().Add(-90 * 24 * time.Hour) // 90 يوم افتراضي
        w.DB.Where("created_at < ?", cutoff).Delete(&models.ActivityLog{})
    }
}
```

### عامل إعادة تعيين تعيين المحادثة

**المصدر:** `internal/handlers/chat_assignment_reset_worker.go`

**الفاصل:** كل دقيقة واحدة

```go
func (w *ChatAssignmentResetWorker) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        w.checkResetRules()
    }
}
```

**المعالجة:**
1. التحقق من الجدول لقواعد إعادة التعيين
2. البحث عن المحادثات المطابقة لشروط إعادة التعيين
3. إعادة تعيين التعيينات (مسح `assigned_user_id`)
4. إشعار المستخدمين المتأثرين عبر WebSocket

### عامل حملة المثيل التلقائية

**المصدر:** `internal/handlers/instance_auto_campaign_worker.go`

**الفاصل:** كل دقيقة واحدة

```go
func (w *InstanceAutoCampaignWorker) Start() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        w.processAutoCampaigns()
    }
}
```

**المعالجة:**
1. التحقق من المثيلات المفعّلة للحملة التلقائية
2. البحث عن جهات الاتصال المطابقة للمعايير
3. إرسال رسائل آلية
4. تتبع النتائج

### مشترك إحصائيات الحملة

**المصدر:** `internal/queue/subscriber.go`

**المُشغّل:** مستمر (Redis pub/sub)

```go
func (s *CampaignStatsSubscriber) Start() {
    pubsub := s.redis.Subscribe(ctx, "channel:campaign_stats")
    ch := pubsub.Channel()
    
    for msg := range ch {
        var update CampaignStatsUpdate
        json.Unmarshal([]byte(msg.Payload), &update)
        s.hub.BroadcastToOrg(update.OrgID, &WSMessage{
            Type:    "campaign_stats_update",
            Payload: update,
        })
    }
}
```

### إعادة توصيل WhatsMeow

**المُشغّل:** بدء تشغيل الخادم

```go
// إعادة توصيل جميع المثيلات النشطة
manager.ReconnectAll()

// ربط الجلسات المرتبطة تلقائياً عند التشغيل الأول
manager.AutoConnectLinkedInstancesOnFirstRun()
```

### التوفيق بين الحالات

**المُشغّل:** بدء تشغيل الخادم (مهلة 30 ثانية)

ينظّف حالات المثيلات القديمة من جلسات الخادم السابقة.

## ملخص العمليات الخلفية

| العملية | الفاصل | ملف المصدر | الغرض |
|---------|--------|------------|-------|
| معالج SLA | دقيقة واحدة | `sla_processor.go` | فحص خرق SLA، الإغلاق التلقائي |
| الاحتفاظ بالنشاط | ساعة واحدة | `activity_retention.go` | حذف سجلات النشاط القديمة |
| إعادة تعيين تعيين المحادثة | دقيقة واحدة | `chat_assignment_reset_worker.go` | إعادة تعيين التعيينات القديمة |
| حملة المثيل التلقائية | دقيقة واحدة | `instance_auto_campaign_worker.go` | إرسال رسائل آلية |
| عامل الحملة | مستمر | `worker/worker.go` | معالجة طابور الحملات |
| عامل الوسائط الواردة | مستمر | `worker/worker.go` | تنزيل الوسائط الواردة |
| مشترك إحصائيات الحملة | مستمر | `app.go` | بث إحصائيات الحملات عبر WS |
| إعادة توصيل WhatsMeow | بدء التشغيل | `main.go` | إعادة توصيل جميع المثيلات |
| التوفيق بين الحالات | بدء التشغيل (30 ثانية) | `main.go` | تنظيف حالات المثيلات القديمة |

## انظر أيضاً

- [البنية المعمارية](./architecture)
- [نظام التخزين المؤقت](./caching) — نظام التخزين المؤقت عبر Redis
- [أحداث WebSocket](./websocket-events) — بث إحصائيات الحملات
