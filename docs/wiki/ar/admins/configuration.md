---
title: الإعدادات
rtl: true
lang: ar
---

<div dir="rtl">الإعدادات</div>

يتم إعداد Whatomate عبر ملف TOML مع إمكانية تجاوز جميع الإعدادات عبر متغيرات البيئة.

## ملف الإعدادات

أنشئ `config.toml` في جذر المشروع أو حدده باستخدام العلم `-config`:

```bash
./whatomate serve -config /etc/whatomate/config.toml
```

## مثال كامل للإعدادات

```toml
[app]
name = "Whatomate"
version = "1.0.0"
environment = "production"
debug = false
encryption_key = "your-32-byte-encryption-key-here!!"

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
allowed_origins = ["https://whatomate.example.com"]
max_request_body_size = 52428800  # 50MB

[database]
host = "127.0.0.1"
port = 5432
user = "whatomate"
password = "secure_db_password"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "127.0.0.1"
port = 6379
password = ""
db = 0

[whatsapp]
provider = "meta"  # "meta" أو "whatsmeow"
base_url = "https://graph.facebook.com/v17.0"
webhook_verify_token = "your-webhook-verify-token"

[whatsmeow]
queue_depth = 100
rate_limit = 60  # رسائل في الدقيقة

[jwt]
secret = "your-jwt-secret-min-32-characters!!"
access_token_ttl = "15m"
refresh_token_ttl = "7d"

[default_admin]
email = "admin@whatomate.example.com"
password = "Admin@1234"
full_name = "System Administrator"

[storage]
local_path = "./storage"

[rate_limit]
enabled = true
per_user = 1000
per_ip = 100
```

## أقسام الإعدادات

### app

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `name` | نص | لا | اسم عرض التطبيق |
| `version` | نص | لا | إصدار التطبيق |
| `environment` | نص | لا | "development" أو "production" |
| `debug` | منطقي | لا | تمكين تسجيل التصحيح |
| `encryption_key` | نص | **نعم** | مفتاح 32 بايت لتشفير AES-256-GCM |

**encryption_key**: يُستخدم لتشفير البيانات الحساسة (الرموز، مفاتيح API، الأسرار). يجب أن يكون 32 بايت بالضبط. أنشئه باستخدام:

```bash
openssl rand -base64 32
```

### server

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `host` | نص | لا | عنوان الربط (الافتراضي: `0.0.0.0`) |
| `port` | عدد صحيح | لا | منفذ HTTP (الافتراضي: `8080`) |
| `read_timeout` | مدة | لا | مهلة قراءة الطلب (الافتراضي: `30s`) |
| `write_timeout` | مدة | لا | مهلة كتابة الاستجابة (الافتراضي: `30s`) |
| `allowed_origins` | []نص | لا | أصول CORS المسموحة |
| `max_request_body_size` | عدد صحيح | لا | الحد الأقصى لجسم الطلب بالبايت (الافتراضي: 50MB) |

### database

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `host` | نص | **نعم** | مضيف PostgreSQL |
| `port` | عدد صحيح | لا | منفذ PostgreSQL (الافتراضي: `5432`) |
| `user` | نص | **نعم** | مستخدم قاعدة البيانات |
| `password` | نص | **نعم** | كلمة مرور قاعدة البيانات |
| `dbname` | نص | **نعم** | اسم قاعدة البيانات |
| `ssl_mode` | نص | لا | وضع SSL (الافتراضي: `disable`) |

### redis

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `host` | نص | **نعم** | مضيف Redis |
| `port` | عدد صحيح | لا | منفذ Redis (الافتراضي: `6379`) |
| `password` | نص | لا | كلمة مرور Redis |
| `db` | عدد صحيح | لا | رقم قاعدة بيانات Redis (الافتراضي: `0`) |

### whatsapp

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `provider` | نص | **نعم** | "meta" أو "whatsmeow" |
| `base_url` | نص | لا | عنوان URL الأساسي لـ Meta API |
| `webhook_verify_token` | نص | لا | رمز اشتراك Webhook |

### whatsmeow

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `queue_depth` | عدد صحيح | لا | عمق الطابور لكل نسخة (الافتراضي: `100`) |
| `rate_limit` | عدد صحيح | لا | رسائل في الدقيقة لكل نسخة (الافتراضي: `60`) |

### jwt

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `secret` | نص | **نعم** | سر توقيع JWT (32 حرفاً كحد أدنى) |
| `access_token_ttl` | مدة | لا | عمر رمز الوصول (الافتراضي: `15m`) |
| `refresh_token_ttl` | مدة | لا | عمر رمز التحديث (الافتراضي: `7d`) |

### default_admin

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `email` | نص | لا | البريد الإلكتروني الافتراضي للمدير |
| `password` | نص | لا | كلمة المرور الافتراضية للمدير |
| `full_name` | نص | لا | اسم العرض الافتراضي للمدير |

يتم إنشاء المدير الافتراضي أثناء الترحيل الأول إذا لم يكن موجوداً بالفعل.

### storage

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `local_path` | نص | لا | مسار تخزين الملفات المحلي (الافتراضي: `./storage`) |

### rate_limit

| المفتاح | النوع | مطلوب | الوصف |
|-----|------|----------|-------------|
| `enabled` | منطقي | لا | تمكين تحديد المعدل (الافتراضي: `true`) |
| `per_user` | عدد صحيح | لا | طلبات لكل نافذة لكل مستخدم (الافتراضي: `1000`) |
| `per_ip` | عدد صحيح | لا | طلبات لكل نافذة لكل عنوان IP (الافتراضي: `100`) |

## تجاوزات متغيرات البيئة

يمكن تجاوز جميع قيم الإعدادات عبر متغيرات البيئة باستخدام البادئة `WHATOMATE_` وأسماء الأقسام/المفاتيح بأحرف كبيرة:

```bash
# قاعدة البيانات
export WHATOMATE_DATABASE_HOST=prod-db.example.com
export WHATOMATE_DATABASE_PASSWORD=super_secret

# Redis
export WHATOMATE_REDIS_HOST=prod-redis.example.com
export WHATOMATE_REDIS_PASSWORD=redis_secret

# JWT
export WHATOMATE_JWT_SECRET=$(openssl rand -base64 64)

# التشفير
export WHATOMATE_APP_ENCRYPTION_KEY=$(openssl rand -base64 32)

# الخادم
export WHATOMATE_SERVER_PORT=443
export WHATOMATE_SERVER_ALLOWED_ORIGINS='["https://app.example.com"]'

# WhatsApp
export WHATOMATE_WHATSAPP_PROVIDER=whatsmeow
```

تتمتع متغيرات البيئة بأسبقية على ملف TOML.

## إعداد مفتاح التشفير

مفتاح التشفير حاسم لأمان البيانات. يقوم بتشفير:

- رموز الوصول لحسابات WhatsApp
- رموز التحقق من Webhook
- أسرار عملاء SSO
- مفاتيح API لذكاء Chatbot
- أسرار Webhook
- رؤوس الإجراءات المخصصة

**مهم**: إذا غيّرت مفتاح التشفير، يجب تشغيل ترحيل التشفير لإعادة تشفير جميع البيانات الموجودة:

```bash
whatomate crypto-migrate
```

راجع [ترحيل البيانات](data-migration.md) للتفاصيل.

## إدارة سر JWT

يُستخدم سر JWT لتوقيع رموز الوصول والتحديث. يمكن تعيينه عبر:

1. متغير البيئة: `WHATOMATE_JWT_SECRET`
2. ملف الإعدادات: `jwt.secret`

**تدوير المفتاح**: يؤدي تغيير سر JWT إلى إبطال جميع الرموز الموجودة. سيحتاج المستخدمون إلى تسجيل الدخول مرة أخرى. خطط للتدوير خلال نوافذ الصيانة.

```bash
# إنشاء سر JWT قوي
openssl rand -base64 64
```

## التحقق من صحة الإعدادات

عند البدء، يتحقق Whatomate من الإعدادات المطلوبة:

- يجب أن يكون `encryption_key` 32 بايت بالضبط
- يجب أن يكون `jwt.secret` 32 حرفاً على الأقل
- يجب تعيين `database.host` و`database.user` و`database.dbname`
- يجب تعيين `redis.host`
- يجب أن يكون `whatsapp.provider` "meta" أو "whatsmeow"

تسبب الإعدادات غير الصالحة خروج الخادم مع رسالة خطأ.

## انظر أيضاً

- [النشر](deployment.md) — النشر في الإنتاج مع الإعدادات
- [الأمان](security.md) — الإعدادات المتعلقة بالأمان
- [ترحيل البيانات](data-migration.md) — ترحيل التشفير لتغييرات مفتاح التشفير
- [استكشاف الأخطاء](troubleshooting.md) — مشاكل متعلقة بالإعدادات
