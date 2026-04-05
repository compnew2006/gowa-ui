---
title: دليل المساهمة
rtl: true
lang: ar
---

<div dir="rtl">

# دليل المساهمة

يغطي هذا الدليل نمط الكود، وأنماط التطوير، وعملية طلبات السحب (PR) في واتومات.

## نمط الكود

### اصطلاحات Go

- اتبع تنسيق `gofmt` (شغّل `gofmt -w .` قبل الالتزام)
- استخدم `ruff check .` لملفات Python إذا كان ذلك ممكناً
- أسماء الحزم بأحرف صغيرة وكلمة واحدة
- الأسماء المُصدَّرة تستخدم PascalCase؛ غير المُصدَّرة تستخدم camelCase
- أسماء الواجهات تنتهي بـ `-er` عندما يكون ذلك ممكناً (`Provider`، `Handler`)
- متغيرات الخطأ تبدأ بـ `Err` (`ErrNotFound`، `ErrUnauthorized`)

```go
// جيد
type MessageProvider interface {
    SendMessage(msg OutgoingMessage) error
}

var ErrAccountNotFound = errors.New("account not found")

// سيء
type message_provider interface {
    send_message(msg outgoingMessageRequest) error
}
```

### تنظيم الملفات

```
internal/
├── handlers/        # معالجات طلبات HTTP
├── middleware/      # وسائط HTTP
├── models/          # تعريفات نماذج GORM
├── config/          # تحميل التكوين
├── database/        # اتصال قاعدة البيانات والترحيلات
├── crypto/          # أدوات التشفير
├── queue/           # نظام طوابير Redis
├── worker/          # العمالة الخلفية
└── websocket/       # محور WebSocket والرسائل

pkg/
├── whatsapp/        # عميل Meta WhatsApp
├── whatsmeow/       # مدير اتصال WhatsMeow
└── provider/        # واجهة تجريد المزود
```

### اصطلاحات التسمية

| الكيان | الاصطلاح | مثال |
|--------|-----------|---------|
| المعالجات | `App.ActionName()` | `App.ListContacts()` |
| الوسائط | اسم وصفي | `AuthMiddleware`، `CSRFProtection` |
| النماذج | PascalCase مفرد | `User`، `WhatsAppAccount` |
| هياكل التكوين | PascalCase | `AppConfig`، `DatabaseConfig` |
| الثوابت | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT` |
| رموز الأخطاء | camelCase | `instance_not_found`، `chat_closed` |

## ترتيب سلسلة الوسائط

يتم تطبيق الوسائط بهذا الترتيب لكل طلب:

```
1. CORS Wrapper          (مستوى fasthttp — يتعامل مع preflight)
2. رؤوس الأمان          (X-Content-Type-Options، X-Frame-Options، إلخ)
3. مسجل الطلبات        (يسجل الطريقة، المسار، المدة، الحالة)
4. الاسترداد              (استرداد من الذعر — يُرجع 500)
5. حماية CSRF       (التحقق من رمز CSRF للطلبات المتحولة)
6. سجل النشاط          (يسجل الإجراءات المهمة)
7. وسيط المصادقة       (التحقق من JWT أو مفتاح API)
8. [المعالج]             (منطق الأعمال)
   ├── التحقق من الصلاحية  (requirePermission على مستوى المعالج)
   ├── حراسة المزود    (التحقق من توافق المزود)
   └── تحديد المعدل     (حدود المعدل لكل نقطة نهاية)
```

عند إضافة وسيط جديد، فكر في مكانه في هذه السلسلة. يجب أن تكون الوسائط المتعلقة بالأمان مبكرة؛ ووسائط منطق الأعمال بعد المصادقة.

## أنماط معالجة الأخطاء

### مغلف الخطأ

تتبع جميع أخطاء API مغلف JSON متسق:

```json
{
  "error": {
    "message": "رسالة خطأ مقروءة للبشر",
    "code": "machine_readable_code",
    "field": "field_name_if_validation_error"
  }
}
```

### رموز حالة HTTP

| الحالة | المعنى | متى تُستخدم |
|--------|---------|-------------|
| 400 | طلب غير صالح | أخطاء التحقق، إدخال غير صحيح |
| 401 | غير مصرح | مصادقة مفقودة أو غير صالحة |
| 403 | ممنوع | مصادق لكن بدون صلاحية |
| 404 | غير موجود | المورد غير موجود |
| 409 | تعارض | مورد مكرر، محادثة مغلقة |
| 413 | الحمولة كبيرة جداً | جسم الطلب يتجاوز الحد |
| 429 | طلبات كثيرة جداً | تجاوز حد المعدل |
| 500 | خطأ داخلي في الخادم | فشل غير متوقع في الخادم |

### استراتيجية معالجة الأخطاء

```go
func (app *App) CreateUser(c *fasthttp.RequestCtx) {
    // 1. التحقق من الإدخال مبكراً
    var req CreateUserRequest
    if err := json.Unmarshal(c.PostBody(), &req); err != nil {
        sendError(c, 400, "invalid_request", "Invalid JSON body")
        return
    }

    // 2. التحقق من المصادقة (يتم التعامل معه بواسطة الوسيط)
    userID := getUserIDFromContext(c)

    // 3. التحقق من الصلاحيات
    if !app.requirePermission(c, "users", "write") {
        sendError(c, 403, "permission_denied", "Insufficient permissions")
        return
    }

    // 4. التحقق من وجود المورد
    if app.isEmailTaken(req.Email) {
        sendError(c, 409, "duplicate_email", "Email already exists")
        return
    }

    // 5. تنفيذ العملية
    user, err := app.createUser(req)
    if err != nil {
        sendError(c, 500, "internal_error", "Failed to create user")
        return
    }

    // 6. إرجاع النجاح
    sendJSON(c, 201, user)
}
```

### رموز السبب

تتيح رموز السبب الموحدة معالجة أخطاء برمجية في الواجهة الأمامية:

```go
// internal/handlers/reason_codes.go
const (
    ReasonInstanceNotFound    = "instance_not_found"
    ReasonInstanceNotConnected = "instance_not_connected"
    ReasonInstanceNotAllowed  = "instance_not_allowed"
    ReasonChatUnclaimed       = "chat_unclaimed"
    ReasonChatClosed          = "chat_closed"
    ReasonRestrictionViolation = "restriction_violation"
)

// خطأ برمز سبب
func asInstanceSelectionError(code, message string) error {
    return &InstanceSelectionError{
        ReasonCode: code,
        Message:    message,
    }
}
```

## إضافة ميزات جديدة

### 1. تعريف النموذج

أضف النموذج إلى `internal/models/`:

```go
type NewFeature struct {
    gorm.Model
    Name           string `gorm:"not null"`
    OrganizationID uint   `gorm:"index"`
    Settings       datatypes.JSON
}
```

### 2. تشغيل الترحيل

تستخدم الترحيلات GORM AutoMigrate. سيتم إنشاء النموذج تلقائياً عند بدء التشغيل باستخدام العلم `-migrate`:

```bash
./whatomate -migrate
```

أو التشغيل عبر API (مشرف فائق فقط):

```bash
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>"
```

### 3. إضافة إدخال التخزين المؤقت (إذا كان ذلك ممكناً)

إذا كانت البيانات يتم الوصول إليها بشكل متكرر، أضف دعم التخزين المؤقت في `internal/handlers/cache.go`:

```go
func GetNewFeatureCached(id uint) (*NewFeature, error) {
    key := fmt.Sprintf("feature:%d", id)
    // ... نمط جلب/عدم وجود/تخزين من التخزين المؤقت
}
```

### 4. إنشاء المعالجات

أضف المعالجات في `internal/handlers/`:

```go
func (app *App) ListNewFeatures(c *fasthttp.RequestCtx) {
    if !app.requirePermission(c, "features", "read") {
        sendError(c, 403, "permission_denied", "Insufficient permissions")
        return
    }
    // ... استعلام وإرجاع
}
```

### 5. تسجيل المسارات

أضف المسارات في `cmd/whatomate/main.go`:

```go
api.GET("/features", app.AuthMiddleware(app.ListNewFeatures))
api.POST("/features", app.AuthMiddleware(app.CreateNewFeature))
api.PUT("/features/{id}", app.AuthMiddleware(app.UpdateNewFeature))
api.DELETE("/features/{id}", app.AuthMiddleware(app.DeleteNewFeature))
```

### 6. إضافة الاختبارات

أنشئ ملف `*_test.go` بجانب المعالجات:

```go
func TestListNewFeatures(t *testing.T) {
    db := setupTestDB()
    app := setupTestApp(db)

    // اختبار بمصادقة وصلاحية صحيحة
    // اختبار بدون صلاحية
    // اختبار بنتائج فارغة
}
```

### 7. إضافة أحداث Webhook (إذا كان ذلك ممكناً)

إذا كانت الميزة يجب أن تُشغّل Webhooks، أضف أنواع الأحداث واستدعاءات الإرسال:

```go
app.DispatchWebhook("feature_created", map[string]interface{}{
    "feature_id": feature.ID,
    "name":       feature.Name,
})
```

### 8. إضافة أحداث WebSocket (إذا كان ذلك ممكناً)

إذا كانت الميزة تحتاج تحديثات في الوقت الفعلي:

```go
websocket.BroadcastToOrg(orgID, websocket.WSMessage{
    Type: "feature_updated",
    Payload: feature,
})
```

## متطلبات الاختبار

- يجب أن تحتوي جميع المعالجات الجديدة على اختبارات وحدة
- يجب أن تحتوي المسارات الحرجة على اختبارات E2E
- يجب ألا تنخفض التغطية (تحقق من `coverage.out`)
- شغّل `go test ./...` قبل تقديم PR

## عملية طلب السحب (PR)

1. أنشئ فرع ميزة: `git checkout -b feature/description`
2. أجرِ التغييرات متبعاً اصطلاحات نمط الكود
3. أضف اختبارات للوظائف الجديدة
4. شغّل الاختبارات: `go test ./...`
5. شغّل المدقق: `gofmt -w .`
6. التزم برسالة وصفية
7. ادفع وأنشئ طلب سحب
8. عالج ملاحظات المراجعة
9. ادمج بعد الموافقة

## تسمية الفروع

| النوع | النمط | مثال |
|------|---------|---------|
| ميزة | `feature/description` | `feature/campaign-scheduling` |
| إصلاح خطأ | `fix/description` | `fix/webhook-signature` |
| إعادة هيكلة | `refactor/description` | `refactor/cache-layer` |
| وثائق | `docs/description` | `docs/api-reference` |
| صحة | `desloppify/description` | `desloppify/code-health` |

## انظر أيضاً

- [نظرة عامة على البنية المعمارية](architecture.md) — تصميم النظام وعلاقات المكونات
- [نماذج قاعدة البيانات](database-models.md) — أنماط النماذج الموجودة
- [بنية الاختبار](testing.md) — إعداد الاختبار والأنماط
- [مرجع API](api-reference.md) — اصطلاحات REST API
- [تكامل Webhook](webhook-integration.md) — أنماط Webhook الصادرة

</div>
