---
title: مرجع واجهة البرمجة (API)
rtl: true
lang: ar
---

<div dir="rtl">مرجع واجهة البرمجة (API)</div>

مرجع كامل لواجهة REST API في Whatomate، منظم حسب المورد. جميع نقاط النهاية مسبوقة بعنوان URL الأساسي (مثال: `http://localhost:8080/api`).

## المصادقة

تتطلب جميع نقاط النهاية المصادقة إما:
- ملفات تعريف ارتباط HTTP-only (`whm_access` لرمز الدخول)
- رمز Bearer في رأس `Authorization`
- مفتاح API في رأس `X-API-Key`

### تسجيل الدخول

```
POST /api/auth/login
```

**جسم الطلب:**
```json
{
  "email": "admin@example.com",
  "password": "securepassword"
}
```

**الاستجابة (200):**
```json
{
  "expires_in": 900,
  "user": {
    "id": 1,
    "email": "admin@example.com",
    "full_name": "Admin User",
    "role": "admin",
    "is_active": true
  }
}
```

**رموز الأخطاء:**
| الرمز | الحالة | الوصف |
|-------|--------|-------|
| `invalid_credentials` | 401 | البريد الإلكتروني أو كلمة المرور غير صحيحة |
| `account_disabled` | 403 | حساب المستخدم غير نشط |

### التسجيل

```
POST /api/auth/register
```

**جسم الطلب:**
```json
{
  "email": "user@example.com",
  "password": "Str0ng!Pass",
  "full_name": "New User",
  "invitation_token": "jwt-token-here"
}
```

**الاستجابة (200):**
```json
{
  "message": "تم إرسال التسجيل. يرجى التحقق من بريدك الإلكتروني."
}
```

### تحديث الرمز

```
POST /api/auth/refresh
```

**الاستجابة (200):** زوج رموز جديد مع تعيين ملفات تعريف الارتباط.

### تسجيل الخروج

```
POST /api/auth/logout
```

### تبديل المؤسسة

```
POST /api/auth/switch-org
```

**جسم الطلب:**
```json
{
  "organization_id": 2
}
```

### الحصول على رمز WebSocket

```
GET /api/auth/ws-token
```

**الاستجابة (200):**
```json
{
  "token": "short-lived-jwt-for-ws"
}
```

## المستخدمون

### سرد المستخدمين

```
GET /api/users
```

**معلمات الاستعلام:**
| المعلمة | النوع | الوصف |
|---------|-------|-------|
| `page` | int | رقم الصفحة (الافتراضي: 1) |
| `per_page` | int | العناصر لكل صفحة (الافتراضي: 20) |
| `search` | string | البحث بالبريد/الاسم |
| `status` | string | التصفية: active, inactive |

**الاستجابة (200):**
```json
{
  "users": [...],
  "total": 50,
  "page": 1,
  "per_page": 20
}
```

### إنشاء مستخدم

```
POST /api/users
```

**جسم الطلب:**
```json
{
  "email": "newuser@example.com",
  "full_name": "New User",
  "role_id": 3,
  "is_active": true
}
```

### تحديث مستخدم

```
PUT /api/users/{id}
```

### حذف مستخدم

```
DELETE /api/users/{id}
```

### قيود إرسال المستخدم

```
GET  /api/users/{id}/send-restrictions
PUT  /api/users/{id}/send-restrictions
```

## المؤسسات

### سرد المؤسسات

```
GET /api/organizations
```

### إنشاء مؤسسة

```
POST /api/organizations
```

### الحصول على المؤسسة الحالية

```
GET /api/organizations/current
```

### حذف مؤسسة

```
DELETE /api/organizations/{id}
```

### أعضاء المؤسسة

```
GET    /api/organizations/members
POST   /api/organizations/members
PUT    /api/organizations/members/{id}
DELETE /api/organizations/members/{id}
```

### إعدادات المؤسسة

```
GET /api/org/settings
PUT /api/org/settings
```

## الأدوار والصلاحيات

### سرد الأدوار

```
GET /api/roles
```

### إنشاء دور

```
POST /api/roles
```

**جسم الطلب:**
```json
{
  "name": "Senior Agent",
  "is_default": false,
  "permissions": [
    {"resource": "contacts", "action": "read"},
    {"resource": "contacts", "action": "write"},
    {"resource": "messages", "action": "read"},
    {"resource": "messages", "action": "write"}
  ]
}
```

### تحديث دور

```
PUT /api/roles/{id}
```

### حذف دور

```
DELETE /api/roles/{id}
```

### سرد الصلاحيات

```
GET /api/permissions
```

## مفاتيح API

### سرد مفاتيح API

```
GET /api/api-keys
```

### إنشاء مفتاح API

```
POST /api/api-keys
```

**جسم الطلب:**
```json
{
  "name": "Integration Key",
  "permissions": ["contacts:read", "messages:write"],
  "expiry": "2026-12-31T23:59:59Z"
}
```

**الاستجابة (200):**
```json
{
  "id": 1,
  "name": "Integration Key",
  "key": "sk_live_abc123...",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### حذف مفتاح API

```
DELETE /api/api-keys/{id}
```

## الحسابات (WhatsApp Business)

### سرد الحسابات

```
GET /api/accounts
```

**معلمات الاستعلام:**
| المعلمة | النوع | الوصف |
|---------|-------|-------|
| `status` | string | التصفية حسب الحالة |
| `provider` | string | التصفية حسب المزوّد |
| `search` | string | البحث بالاسم |

### إنشاء حساب

```
POST /api/accounts
```

**جسم الطلب:**
```json
{
  "name": "Production Account",
  "phone_number_id": "123456789",
  "access_token": "EAAB...",
  "business_account_id": "987654321",
  "webhook_verify_token": "my-secret-token"
}
```

### تحديث حساب

```
PUT /api/accounts/{id}
```

### حذف حساب

```
DELETE /api/accounts/{id}
```

### اختبار اتصال الحساب

```
POST /api/accounts/{id}/test
```

### اشتراك التطبيق

```
POST /api/accounts/{id}/subscribe
```

### الملف التجاري

```
GET  /api/accounts/{id}/business_profile
PUT  /api/accounts/{id}/business_profile
POST /api/accounts/{id}/business_profile/photo
```

## المثيلات (WhatsMeow)

### سرد المثيلات

```
GET /api/instances
```

### إنشاء مثيل

```
POST /api/instances
```

**جسم الطلب:**
```json
{
  "name": "support-instance",
  "is_default": true,
  "auto_read_receipt": true,
  "settings": {}
}
```

### تحديث مثيل

```
PUT /api/instances/{id}
```

### حذف مثيل

```
DELETE /api/instances/{id}
```

### الحصول على حالة المثيل

```
GET /api/instances/{id}/health
```

**الاستجابة (200):**
```json
{
  "uptime": "24h",
  "messages_sent_today": 150,
  "messages_received_today": 200,
  "messages_failed_today": 2,
  "error_rate": 0.01,
  "queue_depth": 5
}
```

### الحصول على رمز QR

```
GET /api/instances/{id}/qr
```

### توصيل المثيل

```
POST /api/instances/{id}/connect
```

### إقران الهاتف

```
POST /api/instances/{id}/pair-phone
```

**جسم الطلب:**
```json
{
  "phone_number": "+1234567890",
  "show_push_notification": true,
  "client_type": "android",
  "client_display_name": "Whatomate"
}
```

### فصل المثيل

```
POST /api/instances/{id}/disconnect
```

### إعادة توصيل المثيل

```
POST /api/instances/{id}/reconnect
```

## جهات الاتصال

### سرد جهات الاتصال

```
GET /api/contacts
```

**معلمات الاستعلام:**
| المعلمة | النوع | الوصف |
|---------|-------|-------|
| `page` | int | رقم الصفحة |
| `per_page` | int | العناصر لكل صفحة |
| `search` | string | البحث بالهاتف/الاسم |
| `tags` | string | التصفية حسب الوسوم (has/all/any) |
| `assigned_to` | int | التصفية حسب المستخدم المعيّن |
| `status` | string | التصفية: open, closed, pending |

### إنشاء جهة اتصال

```
POST /api/contacts
```

**جسم الطلب:**
```json
{
  "phone_number": "+1234567890",
  "name": "John Doe",
  "tags": ["vip", "enterprise"],
  "metadata": {}
}
```

### الحصول على جهة اتصال

```
GET /api/contacts/{id}
```

### تحديث جهة اتصال

```
PUT /api/contacts/{id}
```

### حذف جهة اتصال

```
DELETE /api/contacts/{id}
```

### حذف مؤقت لجهة اتصال

```
POST /api/contacts/{id}/soft-delete
```

### تعيين جهة اتصال

```
PUT /api/contacts/{id}/assign
```

**جسم الطلب:**
```json
{
  "user_id": 5
}
```

### بيانات جلسة جهة الاتصال

```
GET /api/contacts/{id}/session-data
```

## الرسائل

### سرد الرسائل

```
GET /api/chats/{id}/messages
```

**معلمات الاستعلام:**
| المعلمة | النوع | الوصف |
|---------|-------|-------|
| `before` | string | مؤشر لترقيم الصفحات |
| `after` | string | مؤشر لترقيم الصفحات |
| `type` | string | التصفية حسب نوع الرسالة |
| `direction` | string | التصفية: inbound, outbound |

### إرسال رسالة

```
POST /api/contacts/{id}/messages
```

**جسم الطلب:**
```json
{
  "content": "Hello, how can I help you?",
  "account_id": 1,
  "reply_to_message_id": 123
}
```

**الاستجابة (200):**
```json
{
  "id": 456,
  "content": "Hello, how can I help you?",
  "direction": "outbound",
  "status": "pending",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### إرسال رسالة وسائط

```
POST /api/messages/media
```

### إرسال رسالة قالب

```
POST /api/messages/template
```

**جسم الطلب:**
```json
{
  "template_id": 1,
  "contact_id": 10,
  "parameters": {
    "name": "John",
    "order_id": "ORD-123"
  }
}
```

### إرسال تفاعل

```
POST /api/contacts/{id}/messages/{message_id}/reaction
```

### إلغاء رسالة

```
POST /api/contacts/{id}/messages/{message_id}/revoke
```

### تحديد الرسالة كمقروءة

```
PUT /api/messages/{id}/read
```

### إرسال الكتابة

```
POST /api/contacts/{id}/typing
```

## الحملات

### سرد الحملات

```
GET /api/campaigns
```

### إنشاء حملة

```
POST /api/campaigns
```

**جسم الطلب:**
```json
{
  "name": "Holiday Promotion",
  "whatsapp_account": 1,
  "template_id": 5,
  "body_content": "Hi {{1}}, check out our deals!",
  "min_delay_seconds": 20,
  "max_delay_seconds": 45,
  "scheduled_at": "2026-12-01T09:00:00Z"
}
```

### تحديث حملة

```
PUT /api/campaigns/{id}
```

### حذف حملة

```
DELETE /api/campaigns/{id}
```

### بدء حملة

```
POST /api/campaigns/{id}/start
```

### إيقاف حملة مؤقتاً

```
POST /api/campaigns/{id}/pause
```

### إلغاء حملة

```
POST /api/campaigns/{id}/cancel
```

### إعادة المحاولة الفاشلة

```
POST /api/campaigns/{id}/retry-failed
```

### استيراد المستلمين

```
POST /api/campaigns/{id}/recipients/import
```

### الحصول على المستلمين

```
GET /api/campaigns/{id}/recipients
```

### تحميل وسائط الحملة

```
POST /api/campaigns/{id}/media
```

## الشات بوت

### الحصول على الإعدادات

```
GET /api/chatbot/settings
```

### تحديث الإعدادات

```
PUT /api/chatbot/settings
```

**جسم الطلب:**
```json
{
  "enabled": true,
  "greeting_message": "Welcome! How can I help?",
  "fallback_message": "I didn't understand. Let me connect you to an agent.",
  "session_timeout_minutes": 30,
  "business_hours": {
    "monday": {"open": "09:00", "close": "17:00"}
  },
  "ai_enabled": true,
  "ai_provider": "openai",
  "ai_model": "gpt-4",
  "ai_api_key": "sk-...",
  "ai_system_prompt": "You are a helpful assistant.",
  "sla_response_minutes": 15,
  "sla_resolution_minutes": 60,
  "sla_auto_close_hours": 24
}
```

### قواعد الكلمات المفتاحية

```
GET    /api/chatbot/keywords
POST   /api/chatbot/keywords
PUT    /api/chatbot/keywords/{id}
DELETE /api/chatbot/keywords/{id}
```

### تدفقات الشات بوت

```
GET    /api/chatbot/flows
POST   /api/chatbot/flows
PUT    /api/chatbot/flows/{id}
DELETE /api/chatbot/flows/{id}
```

### سياقات الذكاء الاصطناعي

```
GET    /api/chatbot/ai-contexts
POST   /api/chatbot/ai-contexts
PUT    /api/chatbot/ai-contexts/{id}
DELETE /api/chatbot/ai-contexts/{id}
```

### تحويلات الوكيل

```
GET    /api/chatbot/transfers
POST   /api/chatbot/transfers
POST   /api/chatbot/transfers/pick
PUT    /api/chatbot/transfers/{id}/assign
PUT    /api/chatbot/transfers/{id}/resume
```

## القوالب (Meta)

### سرد القوالب

```
GET /api/templates
```

### إنشاء قالب

```
POST /api/templates
```

### تحديث قالب

```
PUT /api/templates/{id}
```

### حذف قالب

```
DELETE /api/templates/{id}
```

### مزامنة القوالب

```
POST /api/templates/sync
```

### إرسال القالب

```
POST /api/templates/{id}/publish
```

### تحميل وسائط القالب

```
POST /api/templates/upload-media
```

## التدفقات (Meta)

### سرد التدفقات

```
GET /api/flows
```

### إنشاء تدفق

```
POST /api/flows
```

### تحديث تدفق

```
PUT /api/flows/{id}
```

### حذف تدفق

```
DELETE /api/flows/{id}
```

### حفظ التدفق في Meta

```
POST /api/flows/{id}/save-to-meta
```

### نشر التدفق

```
POST /api/flows/{id}/publish
```

### إهمال التدفق

```
POST /api/flows/{id}/deprecate
```

### تكرار التدفق

```
POST /api/flows/{id}/duplicate
```

### مزامنة التدفقات

```
POST /api/flows/sync
```

## الكتالوجات (Meta)

### سرد الكتالوجات

```
GET /api/catalogs
```

### إنشاء كتالوج

```
POST /api/catalogs
```

### حذف كتالوج

```
DELETE /api/catalogs/{id}
```

### مزامنة الكتالوجات

```
POST /api/catalogs/sync
```

### سرد المنتجات

```
GET /api/catalogs/{id}/products
```

### إنشاء/تحديث/حذف منتج

```
POST   /api/catalogs/{id}/products
PUT    /api/products/{id}
DELETE /api/products/{id}
```

## الردود الجاهزة

### سرد الردود الجاهزة

```
GET /api/canned-responses
```

### إنشاء رد جاهز

```
POST /api/canned-responses
```

**جسم الطلب:**
```json
{
  "shortcut": "/greeting",
  "content": "Hello! How can I help you today?",
  "category": "greetings"
}
```

### تحديث رد جاهز

```
PUT /api/canned-responses/{id}
```

### حذف رد جاهز

```
DELETE /api/canned-responses/{id}
```

### إرسال رد جاهز

```
POST /api/canned-responses/{id}/send
```

### زيادة الاستخدام

```
POST /api/canned-responses/{id}/use
```

## الوسوم

### سرد الوسوم

```
GET /api/tags
```

### إنشاء وسم

```
POST /api/tags
```

**جسم الطلب:**
```json
{
  "name": "vip",
  "color": "#FFD700"
}
```

### تحديث وسم

```
PUT /api/tags/{name}
```

### حذف وسم

```
DELETE /api/tags/{name}
```

## الفرق

### سرد الفرق

```
GET /api/teams
```

### إنشاء فريق

```
POST /api/teams
```

**جسم الطلب:**
```json
{
  "name": "Support Team",
  "description": "Primary support team",
  "member_ids": [1, 2, 3]
}
```

### تحديث فريق

```
PUT /api/teams/{id}
```

### حذف فريق

```
DELETE /api/teams/{id}
```

### أعضاء الفريق

```
GET    /api/teams/{id}/members
POST   /api/teams/{id}/members
DELETE /api/teams/{id}/members/{user_id}
```

## التحليلات

### إحصائيات لوحة التحكم

```
GET /api/analytics/dashboard
```

### تحليلات الرسائل

```
GET /api/analytics/messages
```

### تحليلات الشات بوت

```
GET /api/analytics/chatbot
```

### تحليلات الوكيل

```
GET /api/analytics/agents
```

### مقارنة الوكلاء

```
GET /api/analytics/agents/comparison
```

### تفاصيل الوكيل

```
GET /api/analytics/agents/{id}
```

### تصدير تقييمات الوكلاء

```
GET /api/analytics/agents/ratings/export
```

### تحليلات Meta

```
GET    /api/analytics/meta
POST   /api/analytics/meta/refresh
GET    /api/analytics/meta/accounts
```

## الويب هوك (الصادر)

### سرد الويب هوك

```
GET /api/webhooks
```

### إنشاء ويب هوك

```
POST /api/webhooks
```

**جسم الطلب:**
```json
{
  "url": "https://example.com/webhook",
  "events": ["message.received", "message.sent", "contact.created"],
  "secret": "hmac-secret",
  "enabled": true
}
```

### تحديث ويب هوك

```
PUT /api/webhooks/{id}
```

### حذف ويب هوك

```
DELETE /api/webhooks/{id}
```

### اختبار الويب هوك

```
POST /api/webhooks/{id}/test
```

## الإجراءات المخصصة

### سرد الإجراءات المخصصة

```
GET /api/custom-actions
```

### إنشاء إجراء مخصص

```
POST /api/custom-actions
```

### تحديث إجراء مخصص

```
PUT /api/custom-actions/{id}
```

### حذف إجراء مخصص

```
DELETE /api/custom-actions/{id}
```

### تنفيذ إجراء مخصص

```
POST /api/custom-actions/{id}/execute
```

## ملاحظات المحادثة

### سرد الملاحظات

```
GET /api/contacts/{id}/notes
```

### إنشاء ملاحظة

```
POST /api/contacts/{id}/notes
```

**جسم الطلب:**
```json
{
  "content": "Customer requested a refund."
}
```

### تحديث ملاحظة

```
PUT /api/contacts/{id}/notes/{note_id}
```

### حذف ملاحظة

```
DELETE /api/contacts/{id}/notes/{note_id}
```

## SSO

### الحصول على مزوّدي SSO

```
GET /api/auth/sso/providers
```

### بدء SSO

```
GET /api/auth/sso/{provider}/init
```

### رد نداء SSO

```
GET /api/auth/sso/{provider}/callback
```

### إعدادات SSO

```
GET    /api/settings/sso
PUT    /api/settings/sso
DELETE /api/settings/sso
```

## WebSockets

### الاتصال

```
GET /ws?token=<ws-token>
```

راجع [أحداث WebSocket](./websocket-events) لأنواع الرسائل.

## الاستيراد/التصدير

### تصدير البيانات

```
POST /api/export
```

**جسم الطلب:**
```json
{
  "table": "contacts",
  "filters": {"status": "open"},
  "format": "csv"
}
```

### استيراد البيانات

```
POST /api/import
```

### تصدير/استيراد التكوين

```
GET /api/export/{table}/config
GET /api/import/{table}/config
```

## طلبات العملاء المحتملين

### إنشاء طلب عميل محتمل عام

```
POST /api/public/lead-requests
```

**جسم الطلب:**
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "phone": "+1234567890",
  "message": "Interested in your product.",
  "widget_id": 1
}
```

### سرد طلبات العملاء المحتملين

```
GET /api/lead-requests
```

### تحديث حالة طلب العميل المحتمل

```
PUT /api/lead-requests/{id}/status
```

**جسم الطلب:**
```json
{
  "status": "contacted"
}
```

## سجلات النشاط

### سرد سجلات النشاط

```
GET /api/activity-logs
```

**معلمات الاستعلام:**
| المعلمة | النوع | الوصف |
|---------|-------|-------|
| `user_id` | int | التصفية حسب المستخدم |
| `action` | string | التصفية حسب الإجراء |
| `resource` | string | التصفية حسب المورد |
| `from` | string | تاريخ البدء |
| `to` | string | تاريخ الانتهاء |

### إنشاء سجل نشاط

```
POST /api/activity-logs
```

## الفحص الجاهزية والصحة

### فحص الصحة

```
GET /health
```

**الاستجابة (200):**
```json
{
  "status": "ok",
  "service": "whatomate"
}
```

### فحص الجاهزية

```
GET /ready
```

**الاستجابة (200):**
```json
{
  "status": "ready"
}
```

**الاستجابة (500):**
```json
{
  "status": "not ready",
  "error": "database connection failed"
}
```

## تنسيق استجابة الأخطاء

تتبع جميع الأخطاء تنسيقاً متسقاً:

```json
{
  "error": {
    "message": "رسالة خطأ مقروءة",
    "code": "machine_readable_code",
    "field": "field_name"
  }
}
```

### رموز حالة HTTP

| الرمز | المعنى |
|-------|--------|
| 400 | طلب غير صالح — خطأ تحقق |
| 401 | غير مصرّح — مصادقة غير صالحة أو مفقودة |
| 403 | ممنوع — رفض الصلاحية |
| 404 | غير موجود — المورد غير موجود |
| 409 | تعارض — تكرار أو انتهاك قاعدة عمل |
| 413 | الحمولة كبيرة جداً |
| 429 | طلبات كثيرة جداً — تم تحديد المعدل |
| 500 | خطأ داخلي في الخادم |

### رموز الأسباب

| الرمز | الوصف |
|-------|-------|
| `instance_not_found` | المثيل غير موجود |
| `instance_not_connected` | المثيل غير متصل |
| `instance_not_allowed` | لا يمكن للمستخدم استخدام هذا المثيل |
| `chat_unclaimed` | يجب المطالبة بالمحادثة قبل الإرسال |
| `chat_closed` | المحادثة مغلقة وتقبل القراءة فقط |
| `restriction_violation` | انتهاك سياسة قيود الإرسال |

## انظر أيضاً

- [البنية المعمارية](./architecture)
- [تجريد المزوّد](./provider-abstraction)
- [أحداث WebSocket](./websocket-events)
- [تكامل الويب هوك](./webhook-integration)
