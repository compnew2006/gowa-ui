---
title: دليل البدء السريع
rtl: true
lang: ar
---

<div dir="rtl">

# دليل البدء السريع

</div>

شغّل واتومات في دقائق. يرشدك هذا الدليل خلال التثبيت والإعداد وتسجيل الدخول الأول.

## المتطلبات المسبقة

قبل البدء، تأكد من تثبيت المكونات التالية:

| المكون | الإصدار | الغرض |
|--------|---------|-------|
| **Go** | 1.21+ | بيئة وخلفية البناء |
| **PostgreSQL** | 13+ | قاعدة البيانات الرئيسية |
| **Redis** | 6+ | التخزين المؤقت والطوابير وإدارة الجلسات |
| **Node.js** | 18+ (اختياري) | تطوير/بناء الواجهة الأمامية |

## الخطوة 1: استنساخ المستودع

```bash
git clone https://github.com/whatomate/whatomate.git
cd whatomate
```

## الخطوة 2: إعداد التطبيق

انسخ ملف الإعدادات المثال وعدّله:

```bash
cp config.example.toml config.toml
```

عدّل `config.toml` بإعداداتك:

```toml
[app]
name = "Whatomate"
environment = "development"
encryption_key = "your-32-byte-encryption-key-here!!"

[server]
host = "0.0.0.0"
port = 8080
allowed_origins = ["http://localhost:5173"]

[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "your_db_password"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "localhost"
port = 6379
password = ""
db = 0

[whatsapp]
provider = "meta"  # or "whatsmeow"
base_url = "https://graph.facebook.com/v18.0"
webhook_verify_token = "your-verify-token"

[jwt]
secret = "your-jwt-secret-min-32-characters!!"
access_token_ttl = "15m"
refresh_token_ttl = "7d"

[default_admin]
email = "admin@whatomate.local"
password = "Admin@1234"
full_name = "System Administrator"
```

> **مهم:** ولّد قيمًا آمنة لـ `encryption_key` و`jwt.secret`:
>
> ```bash
> openssl rand -base64 32  # encryption_key (use first 32 bytes)
> openssl rand -base64 64  # jwt.secret
> ```

## الخطوة 3: بناء الواجهة الأمامية (اختياري)

إذا كنت تريد الواجهة الأمامية المضمنة:

```bash
cd frontend
npm install
npm run build
cd ..
```

هذا يُخرج ملفات ثابتة إلى `internal/frontend/dist/`، والتي تُضمّن في ملف Go الثنائي.

للتطوير، يمكنك تشغيل الواجهة الأمامية بشكل منفصل على المنفذ 5173 وتوجيه طلبات API.

## الخطوة 4: تشغيل ترحيلات قاعدة البيانات

```bash
go run cmd/whatomate/main.go -migrate
```

سيقوم هذا بـ:

1. إنشاء جميع جداول قاعدة البيانات باستخدام GORM AutoMigrate
2. إنشاء المستخدم المدير الافتراضي (من إعدادات `default_admin`)
3. إنشاء الأدوار الافتراضية (admin وmanager وagent)
4. إنشاء إعدادات الدردشة الآلية الافتراضية لمؤسسة المدير

## الخطوة 5: تشغيل الخادم

```bash
go run cmd/whatomate/main.go
```

أو مع ملف إعدادات محدد:

```bash
go run cmd/whatomate/main.go -config /path/to/config.toml
```

سيبدأ الخادم على المنفذ المُعدّ (الافتراضي: `8080`).

## الخطوة 6: الوصول للواجهة الأمامية

- **الواجهة الأمامية المضمنة:** افتح `http://localhost:8080` في متصفحك
- **الواجهة الأمامية للتطوير:** افتح `http://localhost:5173` (إذا كنت تشغّل خادم Vite للتطوير بشكل منفصل)

## الخطوة 7: تسجيل الدخول الأول

سجّل الدخول باستخدام بيانات المدير الافتراضية من `config.toml`:

- **البريد الإلكتروني:** `admin@whatomate.local` (أو ما قمت بإعداده)
- **كلمة المرور:** `Admin@1234` (أو ما قمت بإعداده)

> **ملاحظة أمنية:** غيّر كلمة مرور المدير الافتراضية فورًا بعد تسجيل الدخول الأول.

## الخطوة 8: مهام ما بعد الإعداد

بعد تسجيل الدخول:

1. **غيّر كلمة مرور المدير الافتراضية** — انتقل إلى ملفك الشخصي وحدّث كلمة المرور
2. **أعدّ مزود واتساب** — أنشئ حساب Meta أو مثيل WhatsMeow الخاص بك
3. **أنشئ مؤسستك** — إذا لم تُنشأ تلقائيًا أثناء الترحيل
4. **ادعُ أعضاء الفريق** — أرسل رموز الدعوة لفريقك
5. **أعدّ الدردشة الآلية** — هيّئ رسائل الترحيب وساعات العمل وقواعد الكلمات المفتاحية

## البدء السريع عبر Docker

لأسرع إعداد، استخدم Docker Compose:

```bash
# Create environment file with secure secrets
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
ENCRYPTION_KEY=$(openssl rand -base64 32 | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
EOF

# Start all services
docker compose up -d

# Run migrations
docker compose exec whatomate ./whatomate -migrate
```

افتح التطبيق على `http://localhost:8080`.

## استكشاف الأخطاء

| المشكلة | الحل |
|---------|------|
| فشل الاتصال بقاعدة البيانات | تحقق من تشغيل PostgreSQL وصحة بيانات الاعتماد |
| فشل الاتصال بـ Redis | تحقق من تشغيل Redis وإمكانية الوصول إليه |
| أخطاء الترحيل | تأكد من أن مستخدم قاعدة البيانات لديه أذونات CREATE TABLE |
| المنفذ مستخدم بالفعل | غيّر `server.port` في config.toml |
| الواجهة الأمامية لا تُحمّل | ابنِ الواجهة الأمامية أو شغّل خادم Vite للتطوير |

## الخطوات التالية

- اقرأ [نظرة عامة على المنصة](overview.md) لفهم بنية واتومات
- استكشف [دليل المستخدم](users/index.md) لوثائق الميزات
- راجع [دليل المدير](admins/index.md) للنشر والعمليات
- تحقق من [الأسئلة الشائعة](faq.md) للأسئلة الشائعة

## انظر أيضًا

- [مرجع الإعدادات](admins/configuration.md) — جميع خيارات الإعدادات
- [دليل النشر](admins/deployment.md) — النشر للإنتاج
- [دليل الأمان](admins/security.md) — أفضل ممارسات الأمان
