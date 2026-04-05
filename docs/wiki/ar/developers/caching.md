---
title: نظام التخزين المؤقت
rtl: true
lang: ar
---

<div dir="rtl">نظام التخزين المؤقت</div>

يستخدم Whatomate Redis لتخزين البيانات التي يتم الوصول إليها بشكل متكرر لتقليل الحمل على قاعدة البيانات وتحسين أوقات الاستجابة.

## بنية التخزين المؤقت

تقع طبقة التخزين المؤقت بين المعالجات وقاعدة البيانات. عند عدم العثور في التخزين المؤقت، يتم تحميل البيانات من قاعدة البيانات وتخزينها في Redis مع TTL ثم إرجاعها. عند التحديث، يتم إبطال مفاتيح التخزين المؤقت ذات الصلة.

```
الطلب → GET من التخزين المؤقت → موجود؟ → إرجاع
                          → غير موجود؟ → استعلام قاعدة البيانات → SET في التخزين المؤقت → إرجاع
التحديث → كتابة قاعدة البيانات → إبطال التخزين المؤقت
```

## أنواع البيانات المخزّنة

| نوع البيانات | نمط مفتاح التخزين المؤقت | TTL | المصدر |
|-------------|------------------------|-----|--------|
| حسابات WhatsApp | `account:phone_number_id:{pnid}` | 5 دقائق | جدول `whatsapp_accounts` |
| صلاحيات الأدوار | `role:permissions:{role_id}` | 10 دقائق | جدول `permissions` |
| إعدادات الشات بوت | `chatbot:settings:{org_id}` | 5 دقائق | جدول `chatbot_settings` |
| إعدادات المؤسسة | `org:settings:{org_id}` | 5 دقائق | جدول `organizations` |

## عمليات التخزين المؤقت

### GET

التحقق من Redis للقيمة المخزّنة. يُرجع القيمة المُحلّلة إذا وُجدت.

```go
func GetAccountByPhoneNumberIDCached(phoneNumberID string) (*WhatsAppAccount, error) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    cached, err := redisClient.Get(ctx, key).Result()
    if err == nil {
        // موجود في التخزين المؤقت
        var account WhatsAppAccount
        json.Unmarshal([]byte(cached), &account)
        return &account, nil
    }
    // غير موجود — تحميل من قاعدة البيانات
    return loadAccountFromDB(phoneNumberID)
}
```

### عدم الوجود (Miss)

عند عدم الوجود في التخزين المؤقت، يتم التحميل من قاعدة البيانات والتخزين مع TTL:

```go
func cacheAccount(account *WhatsAppAccount, phoneNumberID string) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    data, _ := json.Marshal(account)
    redisClient.Set(ctx, key, data, 5*time.Minute)
}
```

### الوجود (Hit)

تُرجع عمليات الوجود فوراً دون الوصول إلى قاعدة البيانات، مما يقلل زمن الانتقال من ~10 مللي ثانية (استعلام قاعدة بيانات) إلى ~1 مللي ثانية (Redis GET).

### الإبطال

يتم حذف مفاتيح التخزين المؤقت عند تغيير البيانات الأساسية:

```go
func InvalidateAccountCache(phoneNumberID string) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    redisClient.Del(ctx, key)
}

func InvalidateRolePermissionsCache(roleID uint) {
    key := fmt.Sprintf("role:permissions:%d", roleID)
    redisClient.Del(ctx, key)
}

func InvalidateChatbotSettingsCache(orgID uint) {
    key := fmt.Sprintf("chatbot:settings:%d", orgID)
    redisClient.Del(ctx, key)
}
```

## إعدادات TTL

| نوع البيانات | TTL | السبب |
|-------------|-----|-------|
| الحسابات | 5 دقائق | تتغير بيانات اعتماد الحساب بشكل غير متكرر؛ TTL قصير يضمن الدوران في الوقت المناسب |
| صلاحيات الأدوار | 10 دقائق | نادراً ما تتغير الصلاحيات؛ TTL أطول يقلل الحمل على قاعدة البيانات أثناء فحوصات المصادقة |
| إعدادات الشات بوت | 5 دقائق | قد تتغير الإعدادات أثناء التكوين؛ TTL متوسط يوازن بين الحداثة والأداء |
| إعدادات المؤسسة | 5 دقائق | تتغير إعدادات المؤسسة أحياناً؛ TTL متوسط |

## أنماط مفاتيح التخزين المؤقت

```
account:phone_number_id:{phone_number_id}   → WhatsAppAccount JSON
role:permissions:{role_id}                  → []Permission JSON
chatbot:settings:{org_id}                   → ChatbotSettings JSON
org:settings:{org_id}                       → Organization settings JSON
```

## مُشغّلات إبطال التخزين المؤقت

| الإجراء | التخزين المؤقت المُبطَل |
|---------|------------------------|
| إنشاء/تحديث/حذف حساب | `account:phone_number_id:*` |
| إنشاء/تحديث/حذف دور | `role:permissions:{role_id}` |
| تحديث إعدادات الشات بوت | `chatbot:settings:{org_id}` |
| تحديث إعدادات المؤسسة | `org:settings:{org_id}` |

## تفاصيل التنفيذ

**ملفات المصدر:** `internal/handlers/cache.go`

```go
// GetRolePermissionsCached يحمّل الصلاحيات من التخزين المؤقت أو قاعدة البيانات
func GetRolePermissionsCached(roleID uint) ([]Permission, error) {
    key := fmt.Sprintf("role:permissions:%d", roleID)

    // محاولة التخزين المؤقت أولاً
    cached, err := RedisClient.Get(ctx, key).Result()
    if err == nil {
        var perms []Permission
        if err := json.Unmarshal([]byte(cached), &perms); err == nil {
            return perms, nil
        }
    }

    // غير موجود — تحميل من قاعدة البيانات
    var perms []Permission
    db.Where("role_id = ?", roleID).Find(&perms)

    // تخزين في التخزين المؤقت
    if data, err := json.Marshal(perms); err == nil {
        RedisClient.Set(ctx, key, data, 10*time.Minute)
    }

    return perms, nil
}
```

## اتصال Redis

يتم تكوين Redis عبر `config.toml` أو متغيرات البيئة:

```toml
[redis]
host = "127.0.0.1"
port = 6379
password = ""
db = 0
```

يتم إنشاء الاتصال عند بدء التشغيل وإعادة استخدامه عبر جميع عمليات التخزين المؤقت. يتم التعامل مع تجمع الاتصالات بواسطة مكتبة عميل Redis.

## التخزين المؤقت وتدفق المصادقة

أثناء المصادقة، يتم تحميل صلاحيات الأدوار من التخزين المؤقت:

```
تسجيل الدخول → تحميل المستخدم من قاعدة البيانات
           → GetRolePermissionsCached(user.RoleID)
             → موجود: إرجاع الصلاحيات
             → غير موجود: استعلام قاعدة البيانات، تخزين لمدة 10 دقائق، إرجاع
           → إنشاء JWT مع الصلاحيات
```

## التخزين المؤقت ومعالجة الويب هوك

أثناء معالجة الويب هوك الواردة، يستخدم البحث عن الحساب التخزين المؤقت:

```
الويب هوك → استخراج phone_number_id من الحمولة
         → GetAccountByPhoneNumberIDCached(pnid)
           → موجود: إرجاع الحساب (سريع)
           → غير موجود: استعلام قاعدة البيانات، تخزين لمدة 5 دقائق، إرجاع
         → معالجة الرسالة مع سياق الحساب
```

## انظر أيضاً

- [نظام التكوين](../admins/configuration.md) — تكوين اتصال Redis
- [نماذج قواعد البيانات](database-models.md) — النماذج المخزّنة مؤقتاً
- [المراقبة](../admins/monitoring.md) — مراقبة صحة Redis
