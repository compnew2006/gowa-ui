# نظام تعليقات Facebook بقواعد بيانات SQLite

## 🎯 ما تم إنجازه

تم تحويل نظام Facebook Webhook لحفظ التعليقات في **قواعد بيانات SQLite منفصلة لكل صفحة Facebook** بدلاً من استخدام Hermes memory.

## 📊 الهيكل الجديد

### 1. قواعد البيانات
- **كل صفحة Facebook لها قاعدة بياناتها الخاصة**
- المسار: `/opt/hermes-webhook/databases/page_<PAGE_ID>.db`
- مثال: `/opt/hermes-webhook/databases/page_895247390337022.db`

### 2. الجداول في قاعدة البيانات

#### جدول `comments`
يحفظ جميع التعليقات المستلمة:
- `comment_id`: معرف التعليق الفريد
- `post_id`: معرف المنشور
- `message`: نص التعليق
- `from_name`: اسم صاحب التعليق
- `from_id`: معرف صاحب التعليق
- `replied`: هل تم الرد أم لا
- `reply_text`: نص الرد التلقائي
- `created_time`: وقت استلام التعليق

#### جدول `replies`
يحفظ جميع الردود التلقائية:
- `comment_id`: معرف التعليق
- `reply_id`: معرف الرد
- `reply_text`: نص الرد
- `ai_model`: نموذج الذكاء الاصطناعي المستخدم
- `created_time`: وقت الرد

#### جدول `stats`
إحصائيات يومية:
- `date`: التاريخ
- `total_comments`: عدد التعليقات
- `total_replies`: عدد الردود
- `unique_commenters`: عدد المعلقين الفريدين

## 🔧 الأدوات المتاحة

### 1. عرض التعليقات (سطر الأوامر)
```bash
# عرض جميع الصفحات
python3 /opt/hermes-webhook/view_comments.py list

# عرض تعليقات صفحة محددة
python3 /opt/hermes-webhook/view_comments.py view 895247390337022

# البحث في التعليقات
python3 /opt/hermes-webhook/view_comments.py search 895247390337022 "يوتيوب"

# عرض الردود الأخيرة
python3 /opt/hermes-webhook/view_comments.py replies 895247390337022
```

### 2. Dashboard الويب
🌐 **الرابط**: `http://your-server:5001?page=895247390337022`

المميزات:
- ✅ عرض إحصائيات حية
- ✅ عرض التعليقات الأخيرة
- 🔍 بحث في التعليقات
- 🔄 تحديث تلقائي كل 30 ثانية
- 📱 تصميم متجاوب (موبايل/تابلت/ديسكتوب)

### 3. الاستعلام المباشر من SQLite
```bash
# الاتصال بقاعدة البيانات
sqlite3 /opt/hermes-webhook/databases/page_895247390337022.db

# استعلامات مفيدة
SELECT * FROM comments ORDER BY created_time DESC LIMIT 10;
SELECT * FROM comments WHERE replied = FALSE;
SELECT COUNT(*) FROM comments WHERE DATE(created_time) = DATE('now');
```

## 📁 الملفات المهمة

### 1. ملف قاعدة البيانات
```python
/opt/hermes-webhook/facebook_db.py
```
- إدارة قواعد البيانات SQLite
- دالة `get_page_db(page_id)` للحصول على قاعدة بيانات صفحة

### 2. ملف الويبوك (مُحدَّث)
```python
/opt/hermes-webhook/facebook_webhook_gunicorn.py
```
- يستخدم `forward_to_hermes()` لحفظ التعليقات في SQLite
- يستخدم `save_reply_to_sqlite()` لحفظ الردود

### 3. أدوات العرض
```bash
/opt/hermes-webhook/view_comments.py      # سطر الأوامر
/opt/hermes-webhook/web_dashboard.py      # واجهة الويب
```

## 🚀 الخدمات (Systemd Services)

### 1. Facebook Webhook
```bash
sudo systemctl status hermes-facebook-webhook
sudo systemctl restart hermes-facebook-webhook
sudo journalctl -u hermes-facebook-webhook -f  # مشاهدة السجلات
```

### 2. Dashboard الويب
```bash
sudo systemctl status hermes-facebook-dashboard
sudo systemctl restart hermes-facebook-dashboard
```

## 📊 مثال الاستخدام

### استيراد تعليقات موجودة من السجلات
```bash
# استخراج التعليقات من سجلات webhook وحفظها في SQLite
grep "COMMENT_EVENT" /opt/hermes-webhook/logs/webhook_events.log | \
  python3 /opt/hermes-webhook/import_from_logs.py
```

### عرض إحصائيات سريعة
```python
import sys
sys.path.append('/opt/hermes-webhook')
from facebook_db import get_page_db

db = get_page_db('895247390337022')
stats = db.get_stats()
print(f"Total: {stats['total_comments']}")
print(f"Today: {stats['today_comments']}")
print(f"Replied: {stats['replied_comments']}")
```

## 🔒 الأمان والصلاحيات

- جميع قواعد البيانات في `/opt/hermes-webhook/databases/`
- المالك: `www-data:www-data`
- الصلاحيات: `755` (قراءة/كتابة للمالك والمجموعة)

## 🎉 المزايا

### مقارنة بالنظام القديم:
❌ **قديم (Hermes Memory)**:
- ملفات Markdown متناثرة
- لا يمكن البحث أو الفرز
- لا إحصائيات فورية

✅ **جديد (SQLite)**:
- قاعدة بيانات منظمة
- بحث متقدم
- إحصائيات فورية
- واجهة ويب حية
- علاقة بين التعليقات والردود
- تصدير البيانات بسهولة

## 📝 ملاحظات

- النظام **مستقل تماماً** عن Hermes
- كل صفحة Facebook لها قاعدة بيانات منفصلة
- يمكن إضافة صفحات Facebook جديدة بدون تعديل الكود
- الردود التلقائية تعمل بشكل طبيعي
- جميع الردات تُحفظ في SQLite

---

تم الإنشاء: 2026-05-23
الإصدار: 1.0
