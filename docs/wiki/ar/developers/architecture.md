---
title: البنية المعمارية للنظام
rtl: true
lang: ar
---

<div dir="rtl">البنية المعمارية للنظام</div>

تغطي هذه الصفحة البنية المعمارية العامة للنظام، ومجموعة التقنيات، وهيكل الدليل، وتصميم مكونات Whatomate.

## نظرة عامة على البنية المعمارية

يتبع Whatomate بنية طبقية مع فصل واضح للاهتمامات:

```
┌─────────────────────────────────────────────────┐
│              الواجهة الأمامية (React/Vite)       │
│         مضمّنة في ملف Go الثنائي عبر embed       │
├─────────────────────────────────────────────────┤
│              خادم HTTP (fasthttp)               │
│  ┌─────────┬──────────┬──────────┬───────────┐  │
│  │الوسيط    │ المعالجات │ الموجّه  │ WebSocket │  │
│  └─────────┴──────────┴──────────┴───────────┘  │
├─────────────────────────────────────────────────┤
│              طبقة منطق الأعمال                   │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │ حارس     │ نظام      │ محرك     │ عامل     │  │
│  │ المزوّد  │ الطابور   │ الشات بوت│ الحملة   │  │
│  └──────────┴───────────┴──────────┴──────────┘  │
├─────────────────────────────────────────────────┤
│              طبقة الوصول للبيانات                 │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │   GORM   │   Redis   │ نظام     │ طبقة     │  │
│  │(PostgreSQL)│ عميل    │ التشفير  │ التخزين  │  │
│  │          │           │          │ المؤقت   │  │
│  └──────────┴───────────┴──────────┴──────────┘  │
├─────────────────────────────────────────────────┤
│           التكاملات الخارجية                     │
│  ┌──────────┬───────────┬──────────┬──────────┐  │
│  │   Meta   │ WhatsMeow │ مزوّدو   │ أهداف    │  │
│  │  Cloud   │  عميل     │ الذكاء   │ الويب هوك│  │
│  └──────────┴───────────┴──────────┴──────────┘  │
└─────────────────────────────────────────────────┘
```

## مجموعة التقنيات

| المكوّن | التقنية | الغرض |
|---------|---------|-------|
| خادم HTTP | `valyala/fasthttp` | معالجة طلبات عالية الأداء |
| قاعدة البيانات | PostgreSQL + GORM | متجر البيانات الأساسي، ORM |
| التخزين المؤقت | Redis | رموز الجلسة، عمليات البحث المخزّنة، تحديد المعدل |
| الطابور | Redis | مهام الحملات، تنزيل الوسائط، pub/sub |
| WebSocket | `gorilla/websocket` | اتصال العميل في الوقت الفعلي |
| المصادقة | JWT (HS256) | مصادقة مبنية على الرموز |
| الواجهة الأمامية | React 18 + Vite | تطبيق صفحة واحدة مضمّن |
| التنسيق | TailwindCSS | CSS قائم على الأدوات |
| التشفير | AES-256-GCM | تشفير الحقول الحساسة |

## هيكل الدليل

```
whatomate/
├── cmd/whatomate/
│   └── main.go              # نقطة دخول التطبيق، إعداد الخادم
├── internal/
│   ├── config/
│   │   └── config.go        # محمّل تكوين TOML، تجاوزات env
│   ├── models/
│   │   └── *.go             # تعريفات نماذج GORM (أكثر من 30 نموذج)
│   ├── handlers/
│   │   ├── auth_handlers.go # نقاط نهاية المصادقة
│   │   ├── contacts.go      # إدارة جهات الاتصال/المحادثات
│   │   ├── messages.go      # إرسال الرسائل
│   │   ├── campaigns.go     # إدارة الحملات
│   │   ├── chatbot.go       # إعدادات/قواعد/تدفقات الشات بوت
│   │   ├── webhook.go       # الويب هوك الوارد من Meta
│   │   ├── webhooks.go      # إدارة الويب هوك الصادر
│   │   ├── websocket.go     # معالج WebSocket
│   │   ├── provider_guard.go# وسيط خاص بالمزوّد
│   │   ├── cache.go         # عمليات التخزين المؤقت Redis
│   │   └── ...              # معالجات إضافية
│   ├── middleware/
│   │   ├── auth.go          # مصادقة JWT/مفتاح API
│   │   ├── csrf.go          # حماية CSRF
│   │   ├── security.go      # رؤوس الأمان
│   │   ├── rate_limit.go    # تحديد المعدل
│   │   ├── logger.go        # تسجيل الطلبات
│   │   └── recovery.go      # استعادة الذعر
│   ├── worker/
│   │   ├── worker.go        # عامل الحملة
│   │   ├── campaign_delay.go# منطق التأخير
│   │   ├── send_policy.go   # فرض سياسة الإرسال
│   │   └── idempotency.go   # تفرد المهمة
│   ├── queue/
│   │   ├── queue.go         # تجريد طابور Redis
│   │   ├── consumer.go      # مستهلك المهام
│   │   ├── publisher.go     # ناشر المهام
│   │   └── subscriber.go    # مشترك Pub/Sub
│   ├── crypto/
│   │   ├── crypto.go        # تشفير AES-256-GCM
│   │   └── migration.go     # ترحيل تنسيق التشفير
│   ├── frontend/
│   │   └── embed.go         # نظام ملفات الواجهة الأمامية المضمّن
│   ├── websocket/
│   │   ├── hub.go           # محور WebSocket (org→connections)
│   │   └── messages.go      # أنواع رسائل WS
│   ├── contactutil/
│   │   └── contact.go       # أدوات جهات الاتصال
│   ├── templateutil/
│   │   └── template.go      # أدوات نائبي القالب
│   └── database/
│       └── migrations.go    # غلاف GORM AutoMigrate
├── pkg/
│   ├── provider/
│   │   └── provider.go      # واجهة MessageProvider
│   ├── whatsapp/
│   │   ├── client.go        # عميل Meta Cloud API
│   │   └── meta_adapter.go  # محوّل مزوّد Meta
│   └── whatsmeow/
│       ├── manager.go       # مدير الاتصال
│       ├── adapter.go       # محوّل مزوّد WhatsMeow
│       └── queue.go         # طابور رسائل لكل مثيل
├── frontend/                 # تطبيق React/Vite SPA
│   ├── src/
│   ├── e2e/                 # اختبارات Playwright E2E
│   └── vite.config.js
├── config.example.toml       # قالب التكوين
└── go.mod
```

## نظرة عامة على المكونات

### خادم HTTP (fasthttp)

يستخدم الخادم `valyala/fasthttp` لمعالجة الطلبات عالية الأداء. يتم تسجيل المسارات في `cmd/whatomate/main.go` عبر الدالة `setupRoutes()`.

```go
// نمط تسجيل المسارات
app.POST("/api/auth/login", app.Login)
app.GET("/api/contacts", app.RequireAuth(app.ListContacts))
app.POST("/api/contacts/:id/messages", app.RequireAuth(app.SendMessage))
```

### سلسلة الوسيط

تمر الطلبات عبر سلسلة وسيط محددة قبل الوصول إلى المعالجات:

1. **CORS** — رؤوس عبر النطاقات (مستوى fasthttp)
2. **رؤوس الأمان** — X-Content-Type-Options، X-Frame-Options، إلخ.
3. **مسجّل الطلبات** — الطريقة، المسار، الحالة، المدة
4. **الاستعادة** — التقاط الذعر، استجابة 500
5. **حماية CSRF** — التحقق من رمز CSRF للطلبات المعدّلة
6. **سجل النشاط** — تدوين الإجراءات المهمة
7. **المصادقة** — التحقق من JWT أو مفتاح API
8. **RBAC** — فحوصات الصلاحيات (مستوى المعالج)
9. **حارس المزوّد** — توافق المزوّد (مستوى المعالج)
10. **تحديد المعدل** — حدود لكل نقطة نهاية

### تجريد مزوّد الرسائل

تتيح واجهة `MessageProvider` التبديل بين Meta وWhatsMeow:

```go
type MessageProvider interface {
    SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error)
    SendMediaMessage(ctx context.Context, req *MediaMessageRequest) (*SendResult, error)
    SendTemplateMessage(ctx context.Context, req *TemplateMessageRequest) (*SendResult, error)
    MarkRead(ctx context.Context, messageID string) error
    SendTyping(ctx context.Context, contactID string, composing bool) error
}
```

راجع [تجريد المزوّد](./provider-abstraction) للتفاصيل.

### محور WebSocket

يحافظ محور WebSocket على خريطة من المؤسسة إلى الاتصالات للبث المستهدف:

```
العميل → /ws (مصادقة JWT) → Hub.Register() → حلقات القراءة/الكتابة
المحور → BroadcastToOrg() → جميع اتصالات أعضاء المؤسسة
```

راجع [أحداث WebSocket](./websocket-events) للتفاصيل.

### العوامل الخلفية

تعمل العوامل كـ goroutines تُبدأ في `main.go`:

| العامل | المُشغّل | الغرض |
|--------|---------|-------|
| معالج SLA | دقيقة واحدة | فحوصات خرق SLA، الإغلاق التلقائي |
| الاحتفاظ بالنشاط | ساعة واحدة | حذف سجلات النشاط القديمة |
| إعادة تعيين تعيين المحادثة | دقيقة واحدة | إعادة تعيين التعيينات القديمة |
| حملة المثيل التلقائية | دقيقة واحدة | إرسال رسائل آلية |
| عامل الحملة | مستمر | معالجة طابور حملة Redis |
| عامل الوسائط الواردة | مستمر | تنزيل الوسائط الواردة |
| مشترك إحصائيات الحملة | مستمر | بث الإحصائيات عبر WS |

راجع [العمليات الخلفية](./background-workers) للتفاصيل.

## مخططات تدفق البيانات

### تدفق إرسال الرسائل

```
طلب API → وسيط المصادقة → فحص الصلاحية → تحميل جهة الاتصال/الحساب
  → إنشاء سجل رسالة (قيد الانتظار) → الإرسال عبر المزوّد (غير متزامن)
  → تحديث الحالة (أُرسل/فشل) → بث WebSocket → إرسال الويب هوك
```

### تدفق الرسائل الواردة

```
ويب هوك Meta → التحقق من التوقيع → تحليل الحمولة
  → البحث عن الحساب → الحصول على/إنشاء جهة الاتصال → حفظ الرسالة
  → معالجة الشات بوت (الكلمات المفتاحية، التدفقات، الذكاء الاصطناعي، البديل)
  → إرسال الرد → بث WebSocket → إرسال الويب هوك
```

### تدفق الحملة

```
إنشاء حملة → استيراد المستلمين → بدء الحملة
  → النشر إلى طابور Redis → التقاط العمال للمهام
  → تطبيق التأخير → إرسال الرسالة → تحديث الإحصائيات
  → نشر الإحصائيات → بث WebSocket
```

### تدفق المصادقة

```
طلب تسجيل الدخول → البحث عن المستخدم → التحقق من كلمة المرور → إنشاء زوج JWT
  → تخزين رمز التحديث في Redis → تعيين ملفات تعريف الارتباط → إرجاع المستخدم
```

## نظام التكوين

يتم تحميل التكوين من TOML مع تجاوزات متغيرات البيئة:

```toml
[app]
name = "whatomate"
encryption_key = "your-32-byte-key-here"

[server]
host = "0.0.0.0"
port = 8080
allowed_origins = ["http://localhost:5173"]

[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "secret"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "localhost"
port = 6379

[whatsapp]
provider = "meta"  # أو "whatsmeow"
base_url = "https://graph.facebook.com/v18.0"

[jwt]
secret = "your-jwt-secret"
access_token_ttl = "15m"
refresh_token_ttl = "168h"
```

## انظر أيضاً

- [مرجع واجهة البرمجة (API)](./api-reference)
- [تجريد المزوّد](./provider-abstraction)
- [نماذج قواعد البيانات](./database-models)
- [العمليات الخلفية](./background-workers)
