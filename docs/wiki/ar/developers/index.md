---
title: دليل المطوّرين
rtl: true
lang: ar
---

<div dir="rtl">دليل المطوّرين</div>

مرحباً بك في وثائق Whatomate الخاصة بالمطوّرين. يغطي هذا الدليل كل ما تحتاجه لبناء وتوسيع وصيانة منصة Whatomate لواجهة برمجة تطبيقات WhatsApp Business.

## ما هو Whatomate؟

Whatomate هي منصة متعددة المستأجرين لواجهة برمجة تطبيقات WhatsApp Business تدعم مزوّدي خلفيتين:

- **Meta Cloud API** — واجهة WhatsApp Business Cloud API الرسمية مع القوالب والتدفقات والكتالوجات
- **WhatsMeow** — اتصال مباشر ببروتوكول WhatsApp Web عبر `go.mau.fi/whatsmeow`

## مجموعة التقنيات

| الطبقة | التقنية |
|-------|---------|
| الخلفية | Go 1.21+، `valyala/fasthttp` |
| قاعدة البيانات | PostgreSQL (عبر GORM) |
| التخزين المؤقت/الطابور | Redis |
| الواجهة الأمامية | React 18، Vite، TailwindCSS |
| الوقت الفعلي | WebSocket (gorilla/websocket) |
| المصادقة | JWT (HS256)، ملفات تعريف ارتباط HTTP-only |
| التشفير | AES-256-GCM |

## البدء السريع

```bash
# استنساخ وتكوين
git clone https://github.com/whatomate/whatomate.git
cd whatomate
cp config.example.toml config.toml

# تشغيل الترحيلات
go run cmd/whatomate/main.go -migrate

# بدء الخادم
go run cmd/whatomate/main.go
```

## هيكل الوثائق

| الصفحة | الوصف |
|--------|-------|
| [البنية المعمارية](./architecture) | بنية النظام، مجموعة التقنيات، هيكل الدليل، تدفقات البيانات |
| [مرجع API](./api-reference) | مرجع كامل لواجهة REST API منظمة حسب المورد |
| [تجريد المزوّد](./provider-abstraction) | واجهة MessageProvider، محوّلات Meta/WhatsMeow |
| [أحداث WebSocket](./websocket-events) | أنواع الأحداث في الوقت الفعلي وتنسيقات الرسائل |
| [تكامل الويب هوك](./webhook-integration) | نظام الويب هوك الصادر، أنواع الأحداث، توقيع HMAC |
| [العمليات الخلفية](./background-workers) | نظام طابور Redis، أنواع المهام، عمليات المستهلك |
| [نماذج قواعد البيانات](./database-models) | أكثر من 30 نموذج GORM، العلاقات، المخطط |
| [نظام التخزين المؤقت](./caching) | نظام التخزين المؤقت عبر Redis، إعدادات TTL، أنماط المفاتيح |
| [بنية الاختبار](./testing) | اختبارات الوحدة، اختبارات E2E، التغطية، أدوات الاختبار |
| [دليل المساهمة](./contributing) | نمط الكود، عملية طلبات السحب، إضافة الميزات |

## المفاهيم الأساسية

### تعدد المستأجرين

كل مورد مرتبط بمؤسسة معينة. ينتمي المستخدمون إلى المؤسسات من خلال سجلات العضوية `user_organizations` ويمكنهم التبديل بينها.

### تجريد المزوّد

تجرّد واجهة `MessageProvider` الاختلافات بين Meta وWhatsMeow. يتم إرسال جميع الرسائل عبر `SendOutgoingMessage()` الذي يوجّه إلى المزوّد الصحيح.

### بنية الوقت الفعلي

- **WebSocket** لتحديثات الوقت الفعلي المواجهة للعميل (الرسائل، الحالة، الإشعارات)
- **Redis Pub/Sub** للاتصال بين العمليات (إحصائيات الحملات)
- **طوابير Redis** لمعالجة المهام الخلفية (الحملات، تنزيل الوسائط)

### نموذج الأمان

- مصادقة مبنية على JWT مع تدوير رموز الدخول والتحديث
- RBAC مع أزواج الصلاحية:المورد
- تشفير AES-256-GCM للحقول الحساسة
- حماية CSRF عبر إرسال مزدوج للملف
- Dialer آمن ضد SSRF لطلبات HTTP الصادرة

## انظر أيضاً

- [نظرة عامة على البنية المعمارية](./architecture)
- [مرجع واجهة البرمجة (API)](./api-reference)
- [تجريد المزوّد](./provider-abstraction)
