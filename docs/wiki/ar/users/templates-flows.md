---
title: القوالب والتدفقات
rtl: true
lang: ar
---

<div dir="rtl">القوالب والتدفقات</div>

إدارة قوالب رسائل واتساب، والتدفقات التفاعلية، والكتالوجات، والمنتجات. مزامنة محتواك مع منصة Meta للموافقة والنشر.

> **ملاحظة:** تتطلب القوالب والتدفقات والكتالوجات مزود **Meta (Cloud API)**. هذه الميزات غير متوفرة عند استخدام WhatsMeow.

## إدارة القوالب

### سرد القوالب

**Endpoint:** `GET /api/templates`

عرض جميع قوالب الرسائل لمؤسستك:

```
GET /api/templates?status=approved&category=marketing&language=en
```

| الفلتر | الوصف |
|--------|-------------|
| **Status** | التصفية حسب حالة الموافقة: `approved`, `pending`, `rejected` |
| **Category** | التصفية حسب الفئة: `marketing`, `utility`, `authentication` |
| **Language** | التصفية حسب رمز اللغة |

يتضمن كل قالب إحصائيات الاستخدام توضح عدد مرات إرساله.

### إنشاء قالب

**Endpoint:** `POST /api/templates`

```json
{
  "name": "order_confirmation",
  "category": "utility",
  "language": "en",
  "components": {
    "header": {
      "type": "text",
      "text": "Order Confirmation"
    },
    "body": {
      "text": "Hi {{1}}, your order {{2}} has been confirmed. Expected delivery: {{3}}."
    },
    "footer": {
      "text": "Thank you for your purchase!"
    },
    "buttons": [
      {
        "type": "quick_reply",
        "text": "Track Order"
      },
      {
        "type": "url",
        "text": "View Details",
        "url": "https://example.com/orders/{{2}}"
      }
    ]
  }
}
```

### مكونات القالب

| المكون | الوصف |
|-----------|-------------|
| **Header** | رأس اختياري مع نص أو صورة أو فيديو أو مستند |
| **Body** | نص الرسالة المطلوب مع عناصر نائبة اختيارية (`{{1}}`, `{{2}}`, إلخ) |
| **Footer** | نص تذييل اختياري |
| **Buttons** | أزرار رد سريع أو URL أو رقم هاتف |

### تحديث قالب

**Endpoint:** `PUT /api/templates/{id}`

تعديل قالب موجود. لاحظ أن التغييرات على القوالب المعتمدة قد تتطلب إعادة التقديم إلى Meta للمراجعة.

### حذف قالب

**Endpoint:** `DELETE /api/templates/{id}`

إزالة قالب من سجلاتك المحلية ومن Meta اختيارياً.

### تقديم قالب للموافقة

**Endpoint:** `POST /api/templates/{id}/publish`

تقديم قالب إلى Meta للمراجعة:

1. يتم التحقق من صحة القالب للاكتمال
2. يتم تقديمه إلى نظام مراجعة Meta
3. تتغير الحالة إلى `pending`
4. سيتم إشعارك عندما توافق Meta أو ترفض القالب عبر webhook

### مزامنة القوالب

**Endpoint:** `POST /api/templates/sync`

مزامنة القوالب بين Whatomate و Meta:

1. جلب جميع القوالب من Meta API
2. المقارنة مع السجلات المحلية
3. إنشاء قوالب جديدة موجودة على Meta ولكن ليس محلياً
4. تحديث القوالب الموجودة مع تغييرات الحالة
5. إزالة القوالب المحلية التي لم تعد موجودة على Meta
6. إرجاع ملخص التغييرات

### تحميل وسائط القالب

**Endpoint:** `POST /api/templates/upload-media`

تحميل ملفات الوسائط (صور، فيديوهات، مستندات) للاستخدام في رؤوس القوالب:

1. يتم تحميل الملف إلى endpoint الوسائط الخاص بـ Meta
2. تُرجع Meta مقبض وسائط
3. استخدم المقبض في مكون الرأس الخاص بقالبك

## تدفقات واتساب

تتيح تدفقات واتساب تجارب تفاعلية منظمة داخل محادثات واتساب (مثل حجز المواعيد، نماذج الملاحظات).

### سرد التدفقات

**Endpoint:** `GET /api/flows`

عرض جميع تدفقات واتساب لمؤسستك.

### إنشاء تدفق

**Endpoint:** `POST /api/flows`

```json
{
  "name": "Appointment Booking",
  "categories": ["appointment_booking"],
  "json_payload": {
    "version": "3.0",
    "screens": [
      {
        "id": "SCREEN_1",
        "title": "Book Appointment",
        "terminal": true,
        "data": {},
        "layout": {
          "type": "SingleColumnLayout",
          "children": [
            {
              "type": "Form",
              "name": "appointment_form",
              "children": [
                {
                  "type": "TextInput",
                  "name": "service",
                  "label": "Service",
                  "required": true
                },
                {
                  "type": "DatePicker",
                  "name": "date",
                  "label": "Preferred Date",
                  "required": true
                }
              ]
            }
          ]
        }
      }
    ]
  }
}
```

### تحديث تدفق

**Endpoint:** `PUT /api/flows/{id}`

تعديل حمولة JSON أو بيانات تعريف تدفق موجود.

### حذف تدفق

**Endpoint:** `DELETE /api/flows/{id}`

إزالة تدفق من سجلاتك.

### حفظ التدفق إلى Meta

**Endpoint:** `POST /api/flows/{id}/save-to-meta`

دفع تدفقك إلى منصة Meta:

1. يتم التحقق من صحة JSON التدفق
2. يتم استدعاء Flow API الخاص بـ Meta لإنشاء أو تحديث التدفق
3. يتم تخزين معرف Meta Flow محلياً

### نشر تدفق

**Endpoint:** `POST /api/flows/{id}/publish`

جعل التدفق متاحاً للاستخدام في الرسائل والقوالب.

### إهمال تدفق

**Endpoint:** `POST /api/flows/{id}/deprecate`

وضع علامة على التدفق كمهمل. ستكتمل المحادثات الحالية التي تستخدم التدفق، لكن لا يمكن بدء محادثات جديدة به.

### تكرار تدفق

**Endpoint:** `POST /api/flows/{id}/duplicate`

إنشاء نسخة من تدفق موجود للتعديل دون التأثير على الأصلي.

### مزامنة التدفقات

**Endpoint:** `POST /api/flows/sync`

مزامنة التدفقات بين Whatomate و Meta، على غرار مزامنة القوالب.

## الكتالوجات والمنتجات

إدارة كتالوج المنتجات الخاص بك للاستخدام في رسائل واتساب.

### سرد الكتالوجات

**Endpoint:** `GET /api/catalogs`

عرض جميع الكتالوجات المرتبطة بحساب واتساب للأعمال الخاص بك.

### إنشاء كتالوج

**Endpoint:** `POST /api/catalogs`

إنشاء كتالوج منتجات جديد على Meta.

### حذف كتالوج

**Endpoint:** `DELETE /api/catalogs/{id}`

إزالة كتالوج.

### مزامنة الكتالوجات

**Endpoint:** `POST /api/catalogs/sync`

مزامنة الكتالوجات بين Whatomate و Meta.

### سرد منتجات الكتالوج

**Endpoint:** `GET /api/catalogs/{id}/products`

عرض جميع المنتجات داخل كتالوج محدد.

### إنشاء منتج

**Endpoint:** `POST /api/catalogs/{id}/products`

```json
{
  "name": "Premium Widget",
  "description": "High-quality widget for all your needs",
  "price": "29.99",
  "currency": "USD",
  "image_url": "https://example.com/widget.jpg",
  "url": "https://example.com/products/widget"
}
```

### تحديث منتج

**Endpoint:** `PUT /api/products/{id}`

تعديل تفاصيل المنتج بما في ذلك السعر والوصف والصور.

### حذف منتج

**Endpoint:** `DELETE /api/products/{id}`

إزالة منتج من الكتالوج.

## مزامنة Meta

### لماذا المزامنة؟

Meta هي المصدر الموثوق لحالة الموافقة على القوالب، وتوفر التدفقات، وبيانات الكتالوج. تضمن المزامنة المنتظمة تطابق سجلاتك المحلية مع منصة Meta.

### سلوك المزامنة

| المورد | إجراء المزامنة |
|----------|-------------|
| **Templates** | الجلب من Meta، إنشاء/تحديث/حذف السجلات المحلية |
| **Flows** | الجلب من Meta، مزامنة JSON التدفق والحالة |
| **Catalogs** | الجلب من Meta، مزامنة المنتجات والأسعار |

### المزامنة اليدوية مقابل التلقائية

- **المزامنة اليدوية:** يتم تشغيلها عبر endpoints المزامنة أعلاه
- **المزامنة التلقائية:** يتم استلام تحديثات حالة القوالب عبر webhooks الخاصة بـ Meta في الوقت الفعلي

## العناصر النائبة في القوالب

تدعم القوالب عناصر نائبة ديناميكية يتم حلها عند الإرسال:

| العنصر النائب | يتم حله من |
|-------------|---------------|
| `{{1}}`, `{{2}}`, إلخ. | المعلمات المقدمة عند الإرسال |
| `{{contact.name}}` | اسم جهة الاتصال |
| `{{contact.phone}}` | رقم هاتف جهة الاتصال |
| `{{user.name}}` | اسم الوكيل/المرسل |
| `{{organization.name}}` | اسم المؤسسة |

## انظر أيضاً

- [الحملات](campaigns.md) — استخدام القوالب في الحملات الجماعية
- [المحادثات والمراسلة](chat-messaging.md) — إرسال رسائل القوالب
- [روبوت الدردشة](chatbot.md) — استخدام القوالب في الردود الآلية
- [التحليلات](analytics.md) — إحصائيات استخدام القوالب
