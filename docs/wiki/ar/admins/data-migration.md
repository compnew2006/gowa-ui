---
title: ترحيل البيانات
rtl: true
lang: ar
---

<div dir="rtl">ترحيل البيانات</div>

يستخدم Whatomate GORM AutoMigrate لإدارة المخطط وترحيل تشفير مخصص لترقيات التشفير.

## ترحيلات قاعدة البيانات

### GORM AutoMigrate

يستخدم Whatomate AutoMigrate من GORM لإدارة مخطط قاعدة البيانات. عند الترحيل، يقوم GORM بـ:

- إنشاء جداول لجميع النماذج المحددة
- إضافة الأعمدة المفقودة
- إنشاء الفهارس والقيود
- **لا** يحذف الأعمدة أو يغير أنواع الأعمدة (لمنع فقدان البيانات)

### تشغيل الترحيلات

**سطر الأوامر:**

```bash
# تشغيل الترحيلات والخروج
./whatomate -migrate

# مع إعداد مخصص
./whatomate -migrate -config /etc/whatomate/config.toml
```

**نقطة نهاية API (لمسؤول النظام فقط):**

```bash
# تشغيل الترحيل
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>"

# التحقق من حالة الترحيل
curl https://whatomate.example.com/api/admin/migrate/status \
  -H "Authorization: Bearer <token>"
```

### عملية الترحيل

عند تشغيل الترحيلات:

1. **GORM AutoMigrate** — إنشاء/تحديث جميع جداول النماذج
2. **إنشاء المدير الافتراضي** — إنشاء مستخدم مدير من الإعدادات إذا لم يكن موجوداً
3. **الأدوار الافتراضية** — إنشاء أدوار المدير والوكيل والمدير للمؤسسات
4. **إعدادات chatbot الافتراضية** — إنشاء تكوين chatbot افتراضي لكل مؤسسة

### إنشاء المدير الافتراضي

يتم إنشاء المدير الافتراضي من التكوين أثناء الترحيل الأول:

```toml
[default_admin]
email = "admin@whatomate.example.com"
password = "Admin@1234"
full_name = "System Administrator"
```

يتحقق الترحيل من وجود مستخدم بالبريد الإلكتروني المُعدّد. إذا لم يكن موجوداً:
1. ينشئ المستخدم بكلمة مرور مشفرة بـ bcrypt
2. ينشئ المؤسسة
3. ينشئ الأدوار الافتراضية (مدير، وكيل، مشرف)
4. يضيف المستخدم إلى المؤسسة بدور المدير

## ترحيل التشفير

### نظرة عامة

يعيد ترحيل التشفير تشفير البيانات الحساسة من تنسيقات التشفير القديمة إلى تنسيق AES-256-GCM الحالي (enc3).

### إصدارات التشفير

| الإصدار | البادئة | الخوارزمية | الحالة |
|---------|--------|-----------|--------|
| enc1 | `enc:` | قديم | مهمل — رحّل إلى enc3 |
| enc2 | `enc2:` | قديم v2 | مهمل — رحّل إلى enc3 |
| enc3 | `enc3:` | AES-256-GCM | الحالي |

### تشغيل ترحيل التشفير

```bash
# معاينة التغييرات (تشغيل تجريبي)
./whatomate crypto-migrate -dry-run

# تضمين تنسيق enc2 في الفحص
./whatomate crypto-migrate -include-enc2

# حجم دفعة مخصص
./whatomate crypto-migrate -batch-size 500

# تنفيذ
./whatomate crypto-migrate
```

### ما يتم ترحيله

| الجدول | الأعمدة المشفرة |
|-------|-------------------|
| `whatsapp_accounts` | access_token, phone_number_id, business_account_id, webhook_verify_token |
| `sso_providers` | client_secret |
| `chatbot_settings` | ai_api_key |
| `webhooks` | secret |
| `custom_actions` | headers |

### عملية الترحيل

1. تحميل الإعدادات والتحقق من مفتاح التشفير
2. الاتصال بقاعدة البيانات
3. فحص الأسرار المشفرة القديمة (بادئات `enc:` و`enc2:`)
4. لكل سجل:
   - فك التشفير بالتنسيق القديم
   - إعادة التشفير بتنسيق `enc3:`
   - تحديث السجل
5. المعالجة في دفعات (قابل للتكوين، الافتراضي 1000)
6. تقرير ملخص بالسجلات المحدثة

### مخرج الترحيل

```
Crypto Migration Report
======================
Total records scanned: 1500
Records updated (enc → enc3): 1200
Records updated (enc2 → enc3): 250
Records already enc3: 50
Failed: 0
Duration: 12.5s
```

### أمان الترحيل

- **التشغيل التجريبي أولاً**: شغّل دائماً مع `-dry-run` لمعاينة التغييرات
- **النسخ الاحتياطي أولاً**: أنشئ نسخة احتياطية لقاعدة البيانات قبل التشغيل
- **معالجة الدفعات**: تتم معالجة مجموعات البيانات الكبيرة في دفعات قابلة للتكوين
- **قابل للتكرار**: تشغيل الترحيل عدة مرات آمن — يتم تخطي السجلات التي تم ترحيلها بالفعل

### استكشاف الأخطاء

**فشل الترحيل بخطأ فك تشفير:**
- قد يكون مفتاح التشفير قد تغير
- تحقق من أن `encryption_key` الحالي في الإعدادات يطابق المفتاح المستخدم للتشفير الأصلي
- إذا فُقد المفتاح، لا يمكن فك تشفير البيانات القديمة

**الترحيل لا يجد سجلات:**
- قد تكون جميع البيانات بتنسيق enc3 بالفعل
- جرّب مع `-include-enc2` إذا كنت تشك في وجود بيانات enc2

**الترحيل بطيء:**
- زد حجم الدفعة: `-batch-size 2000`
- تحقق من أداء قاعدة البيانات وزمن انتقال الشبكة

## API حالة الترحيل

تحقق من حالة الترحيل عبر API:

```bash
curl https://whatomate.example.com/api/admin/migrate/status \
  -H "Authorization: Bearer <token>"
```

الاستجابة:

```json
{
  "status": "completed",
  "last_run": "2024-01-01T12:00:00Z",
  "models_migrated": 30,
  "errors": []
}
```

## تغييرات المخطط

عند إضافة نماذج جديدة أو تعديل النماذج الحالية:

1. حدّث تعريف النموذج في `internal/models/`
2. شغّل `whatomate -migrate` لتطبيق التغييرات
3. سينشئ GORM جداول جديدة ويضيف الأعمدة المفقودة

**ملاحظة:** لا يقوم GORM AutoMigrate بـ:
- حذف الأعمدة التي لم تعد موجودة في النموذج
- تغيير أنواع الأعمدة
- حذف الفهارس التي لم تعد موجودة

للتغييرات التدميرية، اكتب ترحيلات SQL خام:

```bash
# تطبيق ترحيل SQL خام
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query": "ALTER TABLE contacts ADD COLUMN priority INT DEFAULT 0"}'
```

## انظر أيضاً

- [نماذج قاعدة البيانات](../developers/database-models.md) — جميع تعريفات النماذج
- [الإعدادات](configuration.md) — تكوين مفتاح التشفير
- [النسخ الاحتياطي والاستعادة](backup-recovery.md) — النسخ الاحتياطي قبل الترحيلات
- [استكشاف الأخطاء](troubleshooting.md) — استكشاف أخطاء الترحيل
