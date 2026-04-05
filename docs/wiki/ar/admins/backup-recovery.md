---
title: النسخ الاحتياطي والاستعادة
rtl: true
lang: ar
---

<div dir="rtl">النسخ الاحتياطي والاستعادة</div>

يغطي هذا الدليل استراتيجيات النسخ الاحتياطي وإجراءات الاستعادة لنشر Whatomate.

## مكونات النسخ الاحتياطي

يشمل النسخ الاحتياطي الكامل:

| المكون | النوع | طريقة النسخ الاحتياطي |
|-----------|------|---------------|
| قاعدة بيانات PostgreSQL | بيانات منظمة | `pg_dump` |
| Redis | التخزين المؤقت، الجلسات، الطوابير | لقطة RDB / AOF |
| تخزين الملفات | ملفات الوسائط، التحميلات | نسخ نظام الملفات |
| الإعدادات | ملفات الإعدادات، متغيرات البيئة | التحكم في الإصدار / مدير الأسرار |

## النسخ الاحتياطي لقاعدة البيانات (PostgreSQL)

### النسخ الاحتياطي الكامل

```bash
# النسخ الاحتياطي إلى ملف
pg_dump -h <host> -U whatomate -d whatomate -F c -f whatomate_$(date +%Y%m%d).dump

# النسخ الاحتياطي مع الضغط
pg_dump -h <host> -U whatomate -d whatomate -F c -Z 9 -f whatomate_$(date +%Y%m%d).dump.gz

# النسخ الاحتياطي لجداول محددة
pg_dump -h <host> -U whatomate -d whatomate -t contacts -t messages -F c -f partial.dump
```

### النسخ الاحتياطي الآلي

**مهمة Cron:**

```bash
# نسخ احتياطي يومي الساعة 2 صباحاً
0 2 * * * pg_dump -h localhost -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +\%Y\%m\%d).dump

# الاحتفاظ بآخر 30 يوماً
0 2 * * * pg_dump -h localhost -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +\%Y\%m\%d).dump && find /backups -name "whatomate_*.dump" -mtime +30 -delete
```

**Docker:**

```yaml
services:
  backup:
    image: postgres:15-alpine
    volumes:
      - ./backups:/backups
    environment:
      PGPASSWORD: ${DB_PASSWORD}
    command: >
      sh -c "pg_dump -h postgres -U whatomate -d whatomate -F c -f /backups/whatomate_$(date +%Y%m%d).dump"
```

### الاستعادة إلى نقطة زمنية

فعّل أرشفة WAL للاستعادة إلى نقطة زمنية:

```postgresql
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'
```

## النسخ الاحتياطي لـ Redis

### لقطة RDB

ينشئ Redis تلقائياً لقطات RDB بناءً على التكوين:

```redis
# redis.conf
save 900 1     # حفظ بعد 900 ثانية إذا تغير مفتاح واحد
save 300 10    # حفظ بعد 300 ثانية إذا تغيرت 10 مفاتيح
save 60 10000  # حفظ بعد 60 ثانية إذا تغيرت 10000 مفتاح
dbfilename dump.rdb
dir /data
```

لقطة يدوية:

```bash
redis-cli -h <host> -a <password> BGSAVE
```

### AOF (الملحق فقط)

لموثوقية أفضل، فعّل AOF:

```redis
# redis.conf
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
```

### النسخ الاحتياطي لملف RDB

```bash
# نسخ ملف RDB
cp /data/dump.rdb /backups/redis_$(date +%Y%m%d).rdb

# أو استخدام redis-cli
redis-cli -h <host> -a <password> --rdb /backups/redis_$(date +%Y%m%d).rdb
```

## النسخ الاحتياطي لتخزين الملفات

يتم تخزين ملفات الوسائط في مسار التخزين المحلي المُعدّد في `storage.local_path`:

```bash
# النسخ الاحتياطي لدليل التخزين
tar -czf /backups/storage_$(date +%Y%m%d).tar.gz -C /app/storage .

# أو استخدام rsync للنسخ الاحتياطي التزايدي
rsync -avz /app/storage/ /backups/storage/
```

### النسخ الاحتياطي لوحدة Docker

```bash
# النسخ الاحتياطي لوحدة Docker
docker run --rm -v whatomate_storage:/data -v $(pwd)/backups:/backup \
  alpine tar -czf /backup/storage_$(date +%Y%m%d).tar.gz -C /data .
```

## إجراءات الاستعادة

### استعادة قاعدة البيانات

```bash
# إيقاف التطبيق
docker compose stop whatomate

# الاستعادة من النسخ الاحتياطي
pg_restore -h <host> -U whatomate -d whatomate -c whatomate_20240101.dump

# إعادة تشغيل التطبيق
docker compose start whatomate
```

**مهم:** العلم `-c` يحذف الكائنات الموجودة قبل الاستعادة. استخدمه بحذر.

### استعادة Redis

```bash
# إيقاف Redis
docker compose stop redis

# استعادة ملف RDB
cp /backups/redis_20240101.rdb /data/dump.rdb

# تشغيل Redis
docker compose start redis
```

### استعادة تخزين الملفات

```bash
# الاستعادة من النسخ الاحتياطي
tar -xzf /backups/storage_20240101.tar.gz -C /app/storage/
```

### الاستعادة الكاملة

1. استعادة قاعدة بيانات PostgreSQL
2. استعادة بيانات Redis
3. استعادة تخزين الملفات
4. تشغيل الترحيلات للتأكد من حداثة المخطط:
   ```bash
   ./whatomate -migrate
   ```
5. تشغيل ترحيل التشفير إذا تغير مفتاح التشفير:
   ```bash
   ./whatomate crypto-migrate
   ```
6. تشغيل التطبيق
7. التحقق من الصحة:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

## خطة التعافي من الكوارث

### أهداف وقت الاستعادة

| المكون | RTO | RPO |
|-----------|-----|-----|
| قاعدة البيانات | 15 دقيقة | 24 ساعة (نسخ احتياطي يومي) |
| Redis | 5 دقائق | ساعة واحدة (لقطات RDB) |
| تخزين الملفات | 30 دقيقة | 24 ساعة |
| التطبيق | 5 دقائق | غير متاح (بدون حالة) |

### خطوات الاستعادة

1. **تقييم الفشل** — تحديد المكونات المتأثرة
2. **توفير البنية التحتية** — إعداد خوادم بديلة إذا لزم الأمر
3. **استعادة البيانات** — اتباع إجراءات الاستعادة أعلاه
4. **التحقق** — تشغيل فحوصات الصحة والجاهزية
5. **الإخطار** — إبلاغ المستخدمين باستعادة الخدمة

### اختبار الاستعادة

اختبر إجراءات النسخ الاحتياطي والاستعادة بانتظام:

1. أنشئ نسخة احتياطية من قاعدة بيانات الإنتاج
2. استعد إلى بيئة اختبار
3. تحقق من سلامة البيانات
4. اختبار وظائف التطبيق
5. توثيق أي مشاكل

## حذف المؤسسة

يستخدم Whatomate الحذف الناعم المتتالي لحذف المؤسسات. عند حذف مؤسسة:

1. يتم حذف سجل المؤسسة بنعومة (تعيين `deleted_at`)
2. يتم حذف جميع السجلات المتعلقة بنعومة بشكل متتالي:
   - المستخدمون
   - حسابات WhatsApp
   - نسخ WhatsApp
   - جهات الاتصال
   - الرسائل
   - الحملات
   - القوالب
   - إعدادات Chatbot
   - الأدوار والصلاحيات
   - الوسوم
   - الفرق
   - Webhooks
   - الإجراءات المخصصة
   - سجلات النشاط
   - الودجات
   - طلبات العملاء المحتملين
   - الإشعارات
   - مزودو SSO

### حذف مؤسسة

```bash
curl -X DELETE https://whatomate.example.com/api/organizations/{id} \
  -H "Authorization: Bearer <token>"
```

**ملاحظة:** يمكن لمسؤولي النظام فقط حذف المؤسسات. الحذف ناعم — يتم الحفاظ على البيانات ويمكن استعادتها عن طريق مسح الطابع الزمني `deleted_at`.

### استعادة مؤسسة محذوفة

```sql
-- استعادة المؤسسة
UPDATE organizations SET deleted_at = NULL WHERE id = <org_id>;

-- استعادة السجلات المتعلقة (متتالي)
UPDATE users SET deleted_at = NULL WHERE organization_id = <org_id>;
UPDATE whatsapp_accounts SET deleted_at = NULL WHERE organization_id = <org_id>;
-- ... كرر لجميع الجداول المتعلقة
```

## التحقق من النسخ الاحتياطي

تحقق من سلامة النسخ الاحتياطي بانتظام:

```bash
# التحقق من النسخ الاحتياطي لقاعدة البيانات
pg_restore -l whatomate_20240101.dump

# التحقق من النسخ الاحتياطي لـ Redis
redis-check-rdb /backups/redis_20240101.rdb

# التحقق من النسخ الاحتياطي للتخزين
tar -tzf /backups/storage_20240101.tar.gz | head
```

## انظر أيضاً

- [ترحيل البيانات](data-migration.md) — إجراءات ترحيل قاعدة البيانات
- [المراقبة](monitoring.md) — فحوصات الصحة للتحقق من النسخ الاحتياطي
- [استكشاف الأخطاء](troubleshooting.md) — مشاكل متعلقة بالاستعادة
- [الإعدادات](configuration.md) — تكوين مسار التخزين
