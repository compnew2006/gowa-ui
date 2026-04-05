---
title: استكشاف الأخطاء
rtl: true
lang: ar
---

<div dir="rtl">استكشاف الأخطاء</div>

المشاكل الشائعة وحلولها لنشر Whatomate.

## مشاكل اتصال النسخ

### العرض: تظهر نسخة WhatsApp بحالة "disconnected" أو "qr"

**نسخ WhatsMeow:**

1. تحقق من حالة النسخ:
   ```bash
   curl https://whatomate.example.com/api/instances/{id}/health \
     -H "Authorization: Bearer <token>"
   ```

2. إذا كانت الحالة "qr"، امسح رمز QR:
   ```bash
   curl https://whatomate.example.com/api/instances/{id}/qr \
     -H "Authorization: Bearer <token>"
   ```

3. إذا كانت الحالة "disconnected"، أعد الاتصال:
   ```bash
   curl -X POST https://whatomate.example.com/api/instances/{id}/connect \
     -H "Authorization: Bearer <token>"
   ```

4. تحقق من سجلات الخادم للبحث عن أخطاء الاتصال:
   ```bash
   docker compose logs whatomate | grep "whatsmeow"
   ```

**الأسباب الشائعة:**
- انتهت صلاحية جلسة WhatsApp (تحتاج إلى إعادة مسح QR)
- مشاكل اتصال الشبكة بخوادم WhatsApp
- تحديد المعدل من WhatsApp
- نسخ متعددة تستخدم نفس الجلسة

### العرض: حلقة إعادة اتصال النسخ

تحقق من سجلات عامل التوفيق بين الحالات. عند البدء، يتم تنظيف الحالات القديمة خلال 30 ثانية. إذا استمرت الحلقة:

1. افصل النسخ:
   ```bash
   curl -X POST https://whatomate.example.com/api/instances/{id}/disconnect \
     -H "Authorization: Bearer <token>"
   ```

2. انتظر 10 ثوانٍ، ثم أعد الاتصال

## فشل تسليم Webhook

### العرض: عدم استلام webhooks من Meta

1. تحقق من إمكانية الوصول إلى عنوان URL لـ Webhook من الإنترنت
2. تحقق من تطابق رمز التحقق من Webhook:
   ```bash
   # في config.toml
   [whatsapp]
   webhook_verify_token = "your-token"
   ```

3. تحقق من اشتراك Webhook في Meta:
   ```bash
   curl -X POST https://whatomate.example.com/api/accounts/{id}/subscribe \
     -H "Authorization: Bearer <token>"
   ```

4. تحقق من سجلات الخادم للبحث عن أخطاء Webhook:
   ```bash
   docker compose logs whatomate | grep "webhook"
   ```

### العرض: عدم تسليم webhooks الصادرة

1. تحقق من تكوين Webhook:
   ```bash
   curl https://whatomate.example.com/api/webhooks \
     -H "Authorization: Bearer <token>"
   ```

2. اختبار تسليم Webhook:
   ```bash
   curl -X POST https://whatomate.example.com/api/webhooks/{id}/test \
     -H "Authorization: Bearer <token>"
   ```

3. تحقق من إمكانية الوصول إلى عنوان URL المستهدف واستجابته لطلبات POST
4. تحقق من تشفير سر Webhook بشكل صحيح

## مشاكل معالجة الحملة

### العرض: الحملة عالقة في حالة "running"

1. تحقق من إحصائيات الحملة:
   ```bash
   curl https://whatomate.example.com/api/campaigns/{id} \
     -H "Authorization: Bearer <token>"
   ```

2. تحقق من عمق طابور Redis:
   ```bash
   redis-cli -h <redis-host> -a <password> LLEN campaign_queue
   ```

3. تحقق من سجلات العامل:
   ```bash
   docker compose logs whatomate | grep "campaign"
   ```

4. إذا لم يكن العمال يعالجون، أعد تشغيل التطبيق

### العرض: فشل رسائل الحملة

1. تحقق من حالة القالب (يجب أن يكون "approved"):
   ```bash
   curl https://whatomate.example.com/api/templates/{id} \
     -H "Authorization: Bearer <token>"
   ```

2. تحقق من حالة الحساب (يجب أن يكون نشطاً):
   ```bash
   curl https://whatomate.example.com/api/accounts/{id} \
     -H "Authorization: Bearer <token>"
   ```

3. تحقق من قيود الإرسال:
   - هل `outbound_mode` مضبوط على "inbound_only"؟
   - هل قيود إرسال المستخدم تحظر الحملة؟
   - هل `campaign_draft_only` مفعّل؟

4. إعادة محاولة المستلمين الفاشلين:
   ```bash
   curl -X POST https://whatomate.example.com/api/campaigns/{id}/retry-failed \
     -H "Authorization: Bearer <token>"
   ```

## أخطاء تحديد المعدل

### العرض: HTTP 429 طلبات كثيرة جداً

1. تحقق من تكوين تحديد المعدل:
   ```toml
   [rate_limit]
   enabled = true
   per_user = 1000
   per_ip = 100
   ```

2. حدد نقطة النهاية محدودة المعدل من رؤوس الاستجابة:
   ```
   X-RateLimit-Limit: 100
   X-RateLimit-Remaining: 0
   X-RateLimit-Reset: 1700000000
   ```

3. انتظر حتى يتم إعادة تعيين نافذة تحديد المعدل، أو زد الحدود في الإعدادات

4. لتحديد معدل نقاط نهاية المصادقة، تحقق من Redis:
   ```bash
   redis-cli -h <redis-host> -a <password> KEYS "ratelimit:*"
   ```

## فشل المصادقة

### العرض: تسجيل الدخول يعيد 401 غير مصرح

1. تحقق من صحة بيانات الاعتماد
2. تحقق من أن حساب المستخدم نشط (`is_active = true`)
3. تحقق من انتماء المستخدم إلى المؤسسة المستهدفة
4. تحقق من اتصال قاعدة البيانات

### العرض: فشل تحديث الرمز

1. تحقق من اتصال Redis (يتم تخزين رموز التحديث في Redis)
2. تحقق من عدم تغيير سر JWT
3. تحقق من انتهاء صلاحية رمز التحديث (7 أيام افتراضياً)
4. امسح ملفات تعريف ارتباط المتصفح وسجّل الدخول مرة أخرى

### العرض: فشل مصادقة WebSocket

1. تأكد من الحصول على رمز WebSocket أولاً:
   ```bash
   curl https://whatomate.example.com/api/auth/ws-token \
     -H "Authorization: Bearer <token>"
   ```

2. تنتهي صلاحية رموز WebSocket بعد 30 ثانية — احصل على رمز جديد قبل الاتصال
3. تحقق من تمرير الرمز كمعامل استعلام: `?token=<ws_token>`

## مشاكل اتصال قاعدة البيانات

### العرض: فشل بدء التطبيق بخطأ قاعدة بيانات

1. تحقق من بيانات اعتماد قاعدة البيانات في الإعدادات:
   ```toml
   [database]
   host = "127.0.0.1"
   port = 5432
   user = "whatomate"
   password = "secure_password"
   dbname = "whatomate"
   ```

2. اختبار اتصال قاعدة البيانات:
   ```bash
   psql -h <host> -U <user> -d <dbname> -c "SELECT 1"
   ```

3. تحقق من تشغيل PostgreSQL:
   ```bash
   docker compose ps postgres
   ```

4. تحقق من سجلات PostgreSQL:
   ```bash
   docker compose logs postgres
   ```

5. تحقق من وجود قاعدة البيانات وأن المستخدم لديه الصلاحيات المناسبة

## مشاكل اتصال Redis

### العرض: فقدان التخزين المؤقت، عدم عمل تحديد المعدل، فشل رموز التحديث

1. تحقق من تكوين Redis:
   ```toml
   [redis]
   host = "127.0.0.1"
   port = 6379
   password = ""
   db = 0
   ```

2. اختبار اتصال Redis:
   ```bash
   redis-cli -h <host> -p <port> -a <password> ping
   ```

3. تحقق من تشغيل Redis:
   ```bash
   docker compose ps redis
   ```

4. تحقق من سجلات Redis:
   ```bash
   docker compose logs redis
   ```

5. إذا تغيرت كلمة مرور Redis، حدّث الإعدادات وأعد التشغيل

## دليل ترحيل التشفير

### متى يتم تشغيل ترحيل التشفير

شغّل ترحيل التشفير عند:
- الترقية من إصدار أقدم مع تشفير قديم
- تغيير مفتاح التشفير
- ظهور قيم بادئة `enc:` أو `enc2:` في قاعدة البيانات

### تشغيل الترحيل

```bash
# تشغيل تجريبي (معاينة التغييرات)
whatomate crypto-migrate -dry-run

# تضمين تنسيق enc2
whatomate crypto-migrate -include-enc2

# حجم دفعة مخصص
whatomate crypto-migrate -batch-size 500

# تنفيذ الترحيل
whatomate crypto-migrate
```

### مخرج الترحيل

```
Crypto Migration Report
======================
Total records scanned: 1500
Records updated (enc → enc3): 1200
Records updated (enc2 → enc3): 250
Records already enc3: 50
Failed: 0
```

### استكشاف أخطاء الترحيل

إذا فشل الترحيل:
1. تحقق من صحة مفتاح التشفير الحالي
2. تحقق من العلم `-include-enc2` إذا كانت لديك بيانات enc2
3. شغّل مع `-dry-run` أولاً لمعاينة التغييرات
4. تحقق من اتصال قاعدة البيانات والصلاحيات

## انظر أيضاً

- [المراقبة](monitoring.md) — فحوصات الصحة والمقاييس
- [ترحيل البيانات](data-migration.md) — ترحيلات قاعدة البيانات والتشفير
- [الإعدادات](configuration.md) — استكشاف أخطاء الإعدادات
- [النشر](deployment.md) — مشاكل متعلقة بالنشر
