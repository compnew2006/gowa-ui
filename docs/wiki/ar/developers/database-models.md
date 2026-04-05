---
title: نماذج قواعد البيانات
rtl: true
lang: ar
---

<div dir="rtl">نماذج قواعد البيانات</div>

يستخدم Whatomate GORM كـ ORM مع PostgreSQL كقاعدة البيانات الأساسية. جميع النماذج تدعم الحذف المؤقت عبر عمود `deleted_at` في GORM.

## نمط الحذف المؤقت

جميع النماذج تضمّن `gorm.Model` الذي يوفر حقول `ID` و `CreatedAt` و `UpdatedAt` و `DeletedAt`. تستبعد الاستعلامات تلقائياً السجلات المحذوفة مؤقتاً ما لم يتم استدعاء `Unscoped()`.

```go
// حذف مؤقت
db.Delete(&user)          // يحدد deleted_at = NOW()
db.Unscoped().Delete(&user) // حذف نهائي

// الاستعلام يستبعد المحذوفين مؤقتاً افتراضياً
db.Find(&users)            // WHERE deleted_at IS NULL
db.Unscoped().Find(&users) // يشمل المحذوفين مؤقتاً
```

## النماذج الأساسية

### User

يمثل حساب مستخدم في النظام.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Email | string | عنوان بريد إلكتروني فريد |
| Password | string | كلمة مرور مشفرة بـ Bcrypt |
| FullName | string | اسم العرض |
| IsActive | bool | حالة تفعيل الحساب |
| OrganizationID | uint | المؤسسة النشطة الحالية |
| RoleID | uint | الدور الحالي داخل المؤسسة |
| Settings | JSONB | إعدادات خاصة بالمستخدم (قيود الإرسال، التفضيلات) |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type User struct {
    gorm.Model
    Email          string `gorm:"uniqueIndex:idx_users_email_org;not null"`
    Password       string `gorm:"not null"`
    FullName       string
    IsActive       bool `gorm:"default:true"`
    OrganizationID uint
    RoleID         uint
    Settings       datatypes.JSON
    Role           CustomRole
}
```

**العلاقات:**
- ينتمي إلى `CustomRole` عبر `RoleID`
- ينتمي إلى `Organization` عبر `OrganizationID`
- لديه العديد من عضويات `UserOrganization`
- لديه العديد من تعيينات `Contact` (assigned_to)
- لديه العديد من إدخالات `ConversationNote`

### Organization

حاوية المؤسسة متعددة المستأجرين.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم المؤسسة الفريد |
| Settings | JSONB | إعدادات على مستوى المؤسسة |
| StrictSendingRestrictionsEnabled | bool | مفتاح تقييد الإرسال الرئيسي |
| OutboundMode | string | "inbound_only" أو "mixed" |
| StrictRolloutMode | string | "audit" أو "enforce" |
| CampaignDraftOnly | bool | تقييد الحملات بالمسودة |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type Organization struct {
    gorm.Model
    Name                             string `gorm:"uniqueIndex;not null"`
    Settings                         datatypes.JSON
    StrictSendingRestrictionsEnabled bool
    OutboundMode                     string
    StrictRolloutMode                string
    StrictRolloutEnforceAt           *time.Time
    CampaignDraftOnly                bool
}
```

**العلاقات:**
- لديه العديد من سجلات `User`
- لديه العديد من تعريفات `CustomRole`
- لديه العديد من سجلات `WhatsAppAccount`
- لديه العديد من سجلات `WhatsAppInstance`
- حذف مؤقت متتالي عند حذف المؤسسة

### CustomRole

تعريفات الأدوار المخصصة داخل مؤسسة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم الدور (فريد داخل المؤسسة) |
| OrganizationID | uint | المؤسسة الأم |
| IsDefault | bool | الدور الافتراضي للمستخدمين الجدد |
| IsSystem | bool | دور نظام (admin, agent, manager) |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type CustomRole struct {
    gorm.Model
    Name           string `gorm:"uniqueIndex:idx_roles_name_org;not null"`
    OrganizationID uint   `gorm:"not null"`
    IsDefault      bool
    IsSystem       bool
    Permissions    []Permission `gorm:"foreignKey:RoleID"`
}
```

**العلاقات:**
- ينتمي إلى `Organization`
- لديه العديد من سجلات `Permission`
- لديه العديد من تعيينات `User`

### Permission

أزواج الصلاحية:المورد المرفقة بالأدوار.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| RoleID | uint | الدور الأم |
| Resource | string | اسم المورد (users, contacts, messages, إلخ) |
| Action | string | الإجراء (read, write, delete, admin) |

```go
type Permission struct {
    gorm.Model
    RoleID   uint   `gorm:"not null;index"`
    Resource string `gorm:"not null"`
    Action   string `gorm:"not null"`
}
```

## نماذج WhatsApp

### WhatsAppAccount

تكوين حساب Meta WhatsApp Business Cloud API.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم العرض |
| PhoneNumberID | string | معرّف هاتف Meta مشفر |
| AccessToken | string | رمز دخول Meta مشفر |
| BusinessAccountID | string | معرّف الحساب التجاري مشفر |
| WebhookVerifyToken | string | رمز التحقق من الويب هوك مشفر |
| OrganizationID | uint | المؤسسة الأم |
| Provider | string | "meta" |
| Status | string | حالة الاتصال |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type WhatsAppAccount struct {
    gorm.Model
    Name               string `gorm:"uniqueIndex:idx_accounts_name_org;not null"`
    PhoneNumberID      string // مشفر بالبادئة enc3:
    AccessToken        string // مشفر بالبادئة enc3:
    BusinessAccountID  string // مشفر بالبادئة enc3:
    WebhookVerifyToken string // مشفر بالبادئة enc3:
    OrganizationID     uint   `gorm:"not null;index"`
    Provider           string
    Status             string
}
```

**العلاقات:**
- ينتمي إلى `Organization`
- لديه العديد من سجلات `Message`
- لديه العديد من سجلات `BulkMessageCampaign`
- لديه العديد من ارتباطات `WhatsAppInstance`

### WhatsAppInstance

مثيل WhatsMeow المباشر لـ WhatsApp Web.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم المثيل (فريد داخل المؤسسة) |
| OrganizationID | uint | المؤسسة الأم |
| IsDefault | bool | المثيل الافتراضي للمؤسسة |
| AutoReadReceipt | bool | إرسال إيصالات القراءة تلقائياً |
| Settings | JSONB | إعدادات خاصة بالمثيل |
| Status | string | حالة الاتصال (disconnected, connecting, connected, qr, paired) |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type WhatsAppInstance struct {
    gorm.Model
    Name            string `gorm:"uniqueIndex:idx_instances_name_org;not null"`
    OrganizationID  uint   `gorm:"not null;index"`
    IsDefault       bool
    AutoReadReceipt bool
    Settings        datatypes.JSON
    Status          string
}
```

**العلاقات:**
- ينتمي إلى `Organization`
- لديه العديد من سجلات `Message`
- لديه العديد من سجلات `BulkMessageCampaign`

### Contact

جهة اتصال / محادثة WhatsApp.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| PhoneNumber | string | رقم هاتف WhatsApp |
| Name | string | اسم العرض |
| ProfileName | string | اسم الملف الشخصي في WhatsApp |
| OrganizationID | uint | المؤسسة الأم |
| AssignedUserID | uint | الوكيل المعيّن |
| WhatsAppAccountID | uint | حساب Meta المرتبط |
| WhatsAppInstanceID | uint | مثيل WhatsMeow المرتبط |
| Status | string | "open", "closed", "pending" |
| IsPublic | bool | رؤية المحادثة العامة |
| ClosedAt | *time.Time | وقت إغلاق المحادثة |
| ClosedByUserID | uint | المستخدم الذي أغلق المحادثة |
| LastMessageAt | *time.Time | طابع آخر رسالة |
| UnreadCount | int | عدد الرسائل غير المقروءة |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type Contact struct {
    gorm.Model
    PhoneNumber        string     `gorm:"index:idx_contacts_phone_org"`
    Name               string
    ProfileName        string
    OrganizationID     uint       `gorm:"index"`
    AssignedUserID     uint
    WhatsAppAccountID  uint
    WhatsAppInstanceID uint
    Status             string
    IsPublic           bool
    ClosedAt           *time.Time
    ClosedByUserID     uint
    LastMessageAt      *time.Time
    UnreadCount        int
    Tags               []Tag            `gorm:"many2many:contact_tags"`
    AssignedUser       User
    Collaborators      []ContactCollaborator
}
```

**العلاقات:**
- ينتمي إلى `Organization`
- ينتمي إلى `User` (معيّن)
- ينتمي إلى `WhatsAppAccount` أو `WhatsAppInstance`
- لديه العديد من سجلات `Message`
- لديه العديد من سجلات `ConversationNote`
- لديه العديد من `Tag` عبر جدول الربط `contact_tags`
- لديه العديد من سجلات `ContactCollaborator`

### Message

سجل رسالة WhatsApp (واردة وصادرة).

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| ContactID | uint | جهة الاتصال المرتبطة |
| WhatsAppMessageID | string | معرّف رسالة Meta/WhatsApp |
| Direction | string | "inbound" أو "outbound" |
| Type | string | "text", "image", "video", "audio", "document", "template", "interactive", "location", "contact" |
| Content | string | محتوى نص الرسالة |
| MediaURL | string | عنوان URL المحلي أو البعيد للوسائط |
| MediaType | string | نوع MIME |
| Status | string | "pending", "sent", "delivered", "read", "failed" |
| ActorType | string | "user", "system", "chatbot" |
| ReplyToMessageID | uint | الرسالة الأصلية للردود |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type Message struct {
    gorm.Model
    ContactID          uint   `gorm:"index"`
    WhatsAppMessageID  string `gorm:"index"`
    Direction          string
    Type               string
    Content            string
    MediaURL           string
    MediaType          string
    Status             string
    ActorType          string
    ReplyToMessageID   uint
    OrganizationID     uint   `gorm:"index"`
    Contact            Contact
    ReplyTo            *Message
    Reactions          []MessageReaction
}
```

## نماذج الحملات

### BulkMessageCampaign

تعريف الحملة لإرسال الرسائل بالجملة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم الحملة |
| OrganizationID | uint | المؤسسة الأم |
| WhatsAppAccountID | uint | حساب الإرسال |
| WhatsAppInstanceID | uint | مثيل الإرسال |
| TemplateID | uint | قالب الرسالة |
| BodyContent | string | جسم القالب مع النائبيين |
| HeaderMediaID | string | مقبض وسائط الرأس |
| MinDelaySeconds | int | الحد الأدنى للتأخير بين الرسائل |
| MaxDelaySeconds | int | الحد الأقصى للتأخير بين الرسائل |
| Status | string | "draft", "running", "paused", "completed", "cancelled" |
| TotalRecipients | int | إجمالي عدد المستلمين |
| SentCount | int | عدد الإرسالات الناجحة |
| FailedCount | int | عدد الإرسالات الفاشلة |
| ScheduledAt | *time.Time | وقت البدء المجدول |
| StartedAt | *time.Time | وقت البدء الفعلي |
| CompletedAt | *time.Time | وقت الانتهاء |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type BulkMessageCampaign struct {
    gorm.Model
    Name               string     `gorm:"not null"`
    OrganizationID     uint       `gorm:"index"`
    WhatsAppAccountID  uint
    WhatsAppInstanceID uint
    TemplateID         uint
    BodyContent        string
    HeaderMediaID      string
    MinDelaySeconds    int        `gorm:"default:20"`
    MaxDelaySeconds    int        `gorm:"default:45"`
    Status             string     `gorm:"default:draft"`
    TotalRecipients    int
    SentCount          int
    FailedCount        int
    ScheduledAt        *time.Time
    StartedAt          *time.Time
    CompletedAt        *time.Time
    Template           Template
    Account            WhatsAppAccount
    Recipients         []BulkMessageRecipient
}
```

### BulkMessageRecipient

مستلم فردي داخل حملة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| CampaignID | uint | الحملة الأم |
| PhoneNumber | string | رقم هاتف المستلم |
| Name | string | اسم المستلم |
| Params | JSONB | معاملات القالب |
| Status | string | "pending", "sent", "delivered", "failed", "cancelled" |
| ErrorMessage | string | سبب الفشل |
| SentAt | *time.Time | وقت إرسال الرسالة |

```go
type BulkMessageRecipient struct {
    gorm.Model
    CampaignID   uint           `gorm:"index"`
    PhoneNumber  string         `gorm:"not null"`
    Name         string
    Params       datatypes.JSON
    Status       string         `gorm:"default:pending;index"`
    ErrorMessage string
    SentAt       *time.Time
    Campaign     BulkMessageCampaign
}
```

## نماذج الشات بوت

### Template

قالب رسالة WhatsApp (محلي ومتزامن مع Meta).

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم القالب |
| Category | string | فئة القالب |
| Language | string | رمز لغة القالب |
| Components | JSONB | الرأس، الجسم، التذييل، الأزرار |
| Status | string | "approved", "pending", "rejected", "draft" |
| OrganizationID | uint | المؤسسة الأم |
| MetaTemplateID | string | معرّف قالب Meta |
| DeletedAt | gorm.DeletedAt | طابع الحذف المؤقت |

```go
type Template struct {
    gorm.Model
    Name           string         `gorm:"uniqueIndex:idx_templates_name_org_lang"`
    Category       string
    Language       string         `gorm:"default:en"`
    Components     datatypes.JSON
    Status         string         `gorm:"default:draft"`
    OrganizationID uint           `gorm:"index"`
    MetaTemplateID string
    Campaigns      []BulkMessageCampaign
}
```

### ChatbotSettings

تكوين أتمتة الشات بوت لكل مؤسسة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Enabled | bool | تفعيل الشات بوت |
| GreetingMessage | string | نص الترحيب التلقائي |
| FallbackMessage | string | نص البديل عند عدم التطابق |
| SessionTimeoutMinutes | int | انتهاء صلاحية الجلسة |
| BusinessHours | JSONB | الجدول اليومي |
| AIEnabled | bool | تفعيل ردود الذكاء الاصطناعي |
| AIProvider | string | اسم مزوّد الذكاء الاصطناعي |
| AIModel | string | معرّف نموذج الذكاء الاصطناعي |
| AIAPIKey | string | مفتاح API مشفر |
| AISystemPrompt | string | موجه النظام |
| AIMaxTokens | int | الحد الأقصى للرموز لكل رد |
| SLAResponseMinutes | int | عتبة SLA للاستجابة |
| SLAResolutionMinutes | int | عتبة SLA للحل |
| SLAEscalationMinutes | int | عتبة SLA للتصعيد |
| SLAAutoCloseHours | int | عتبة الإغلاق التلقائي |

```go
type ChatbotSettings struct {
    gorm.Model
    OrganizationID        uint       `gorm:"uniqueIndex;not null"`
    Enabled               bool
    GreetingMessage       string
    FallbackMessage       string
    SessionTimeoutMinutes int        `gorm:"default:30"`
    BusinessHours         datatypes.JSON
    AIEnabled             bool
    AIProvider            string
    AIModel               string
    AIAPIKey              string     // مشفر
    AISystemPrompt        string
    AIMaxTokens           int
    SLAResponseMinutes    int
    SLAResolutionMinutes  int
    SLAEscalationMinutes  int
    SLAAutoCloseHours     int
    SLAEscalationNotifyIDs datatypes.JSON
}
```

### KeywordRule

قواعد مطابقة الكلمات المفتاحية للشات بوت.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم القاعدة |
| Keywords | JSONB | مصفوفة كلمات مفتاحية |
| MatchType | string | "exact", "contains", "regex" |
| ResponseType | string | "text", "buttons", "flow" |
| ResponseContent | string | محتوى الرد مشفر |
| Priority | int | أولوية المطابقة (أقل = أولوية أعلى) |
| Enabled | bool | تفعيل القاعدة |

```go
type KeywordRule struct {
    gorm.Model
    Name            string         `gorm:"not null"`
    OrganizationID  uint           `gorm:"index"`
    Keywords        datatypes.JSON
    MatchType       string         `gorm:"default:contains"`
    ResponseType    string
    ResponseContent string         // مشفر
    Priority        int            `gorm:"default:0"`
    Enabled         bool           `gorm:"default:true"`
}
```

### ChatbotFlow

تدفقات محادثة متعددة الخطوات.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم التدفق |
| Description | string | وصف التدفق |
| TriggerKeywords | JSONB | مصفوفة كلمات مفتاحية مشغّلة |
| Steps | JSONB | تعريفات خطوات التدفق |
| Enabled | bool | تفعيل التدفق |

```go
type ChatbotFlow struct {
    gorm.Model
    Name            string         `gorm:"not null"`
    OrganizationID  uint           `gorm:"index"`
    Description     string
    TriggerKeywords datatypes.JSON
    Steps           datatypes.JSON
    Enabled         bool           `gorm:"default:true"`
}
```

### AIContext

سياق المعرفة لردود الذكاء الاصطناعي.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم السياق |
| ContentType | string | "static", "dynamic", "url" |
| Content | string | محتوى السياق أو عنوان URL |
| Keywords | JSONB | كلمات مفتاحية مشغّلة |
| Priority | int | أولوية السياق |
| Enabled | bool | تفعيل السياق |

```go
type AIContext struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    OrganizationID uint           `gorm:"index"`
    ContentType    string         `gorm:"default:static"`
    Content        string
    Keywords       datatypes.JSON
    Priority       int
    Enabled        bool           `gorm:"default:true"`
}
```

### AgentTransfer

طلبات التحويل من الشات بوت إلى وكيل بشري.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| ContactID | uint | جهة الاتصال المرتبطة |
| Reason | string | سبب التحويل |
| Priority | int | أولوية التحويل |
| Status | string | "pending", "assigned", "completed", "cancelled" |
| AssignedUserID | uint | الوكيل المعيّن |
| ResolvedAt | *time.Time | طابع الحل |

```go
type AgentTransfer struct {
    gorm.Model
    ContactID      uint       `gorm:"index"`
    Reason         string
    Priority       int        `gorm:"default:0"`
    Status         string     `gorm:"default:pending"`
    AssignedUserID uint
    ResolvedAt     *time.Time
    Contact        Contact
    AssignedUser   User
}
```

## نماذج الإنتاجية

### CannedResponse

قوالب ردود مكتوبة مسبقاً.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Shortcut | string | اختصار الإدراج السريع |
| Content | string | محتوى الرد |
| Category | string | تصنيف الفئة |
| MediaURL | string | وسائط مرتبطة |
| UsageCount | int | عدد مرات الاستخدام |

```go
type CannedResponse struct {
    gorm.Model
    Shortcut       string `gorm:"uniqueIndex:idx_canned_shortcut_org;not null"`
    Content        string `gorm:"not null"`
    Category       string
    MediaURL       string
    UsageCount     int    `gorm:"default:0"`
    OrganizationID uint   `gorm:"index"`
}
```

### Tag

نظام وسم جهات الاتصال.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم الوسم (فريد داخل المؤسسة) |
| Color | string | لون العرض |
| OrganizationID | uint | المؤسسة الأم |

```go
type Tag struct {
    gorm.Model
    Name           string `gorm:"uniqueIndex:idx_tags_name_org;not null"`
    Color          string
    OrganizationID uint   `gorm:"index"`
    Contacts       []Contact `gorm:"many2many:contact_tags"`
}
```

### Team

تجميعات فرق المستخدمين.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| Name | string | اسم الفريق |
| Description | string | وصف الفريق |
| OrganizationID | uint | المؤسسة الأم |

```go
type Team struct {
    gorm.Model
    Name           string `gorm:"not null"`
    Description    string
    OrganizationID uint   `gorm:"index"`
    Members        []TeamMember
}
```

### TeamMember

سجلات عضوية الفريق.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| TeamID | uint | الفريق الأم |
| UserID | uint | عضو الفريق |

```go
type TeamMember struct {
    gorm.Model
    TeamID uint `gorm:"index"`
    UserID uint `gorm:"index"`
    Team   Team
    User   User
}
```

## نماذج التكامل

### Webhook

تكوين نقطة نهاية الويب هوك الصادر.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| URL | string | عنوان نقطة نهاية الويب هوك |
| Secret | string | سر توقيع HMAC مشفر |
| Events | JSONB | أنواع الأحداث المشتركة |
| Enabled | bool | تفعيل الويب هوك |
| LastTriggeredAt | *time.Time | آخر محاولة تسليم |

```go
type Webhook struct {
    gorm.Model
    URL             string         `gorm:"not null"`
    Secret          string         // مشفر
    Events          datatypes.JSON
    Enabled         bool           `gorm:"default:true"`
    OrganizationID  uint           `gorm:"index"`
    LastTriggeredAt *time.Time
}
```

### CustomAction

تعريفات الإجراءات HTTP المخصصة لتدفقات الشات بوت.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم الإجراء |
| URL | string | نقطة النهاية المستهدفة |
| Method | string | طريقة HTTP |
| Headers | JSONB | رؤوس الطلب مشفرة |
| BodyTemplate | string | قالب جسم الطلب |
| Events | JSONB | الأحداث المشغّلة |

```go
type CustomAction struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    OrganizationID uint           `gorm:"index"`
    URL            string         `gorm:"not null"`
    Method         string         `gorm:"default:POST"`
    Headers        datatypes.JSON // مشفر
    BodyTemplate   string
    Events         datatypes.JSON
}
```

## نماذج التدقيق والتواصل

### ConversationNote

ملاحظات خاصة على جهات الاتصال/المحادثات.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| ContactID | uint | جهة الاتصال المرتبطة |
| UserID | uint | مؤلف الملاحظة |
| Content | string | محتوى الملاحظة |

```go
type ConversationNote struct {
    gorm.Model
    ContactID uint   `gorm:"index"`
    UserID    uint   `gorm:"index"`
    Content   string `gorm:"not null"`
    Contact   Contact
    User      User
}
```

### ActivityLog

سجل تدقيق للإجراءات المهمة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| UserID | uint | المستخدم الفاعل |
| Action | string | نوع الإجراء |
| Resource | string | نوع المورد |
| ResourceID | uint | المورد المتأثر |
| Details | JSONB | سياق إضافي |

```go
type ActivityLog struct {
    gorm.Model
    OrganizationID uint           `gorm:"index"`
    UserID         uint           `gorm:"index"`
    Action         string         `gorm:"not null"`
    Resource       string
    ResourceID     uint
    Details        datatypes.JSON
    User           User
}
```

### Widget

تعريفات أدوات التحليلات المخصصة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم الأداة |
| Type | string | نوع الأداة |
| Query | JSONB | تكوين استعلام البيانات |
| Config | JSONB | تكوين العرض |

```go
type Widget struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    Type           string
    Query          datatypes.JSON
    Config         datatypes.JSON
    OrganizationID uint           `gorm:"index"`
}
```

### LeadRequest

تقديمات نموذج التقاط العملاء المحتملين العام.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| WidgetID | uint | الأداة المصدر |
| Name | string | اسم العميل المحتمل |
| Email | string | بريد العميل المحتمل |
| Phone | string | هاتف العميل المحتمل |
| Message | string | رسالة العميل المحتمل |
| Status | string | "new", "contacted", "converted", "rejected" |

```go
type LeadRequest struct {
    gorm.Model
    Name           string
    Email          string
    Phone          string
    Message        string
    Status         string         `gorm:"default:new"`
    OrganizationID uint           `gorm:"index"`
    WidgetID       uint
}
```

### Notification

إشعارات المستخدم داخل التطبيق.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| UserID | uint | المستخدم المستهدف |
| OrganizationID | uint | المؤسسة الأم |
| Type | string | نوع الإشعار |
| Title | string | عنوان الإشعار |
| Message | string | جسم الإشعار |
| IsRead | bool | حالة القراءة |
| Data | JSONB | حمولة إضافية |

```go
type Notification struct {
    gorm.Model
    UserID         uint           `gorm:"index"`
    OrganizationID uint           `gorm:"index"`
    Type           string
    Title          string
    Message        string
    IsRead         bool           `gorm:"default:false"`
    Data           datatypes.JSON
}
```

## نماذج الربط والارتباط

### UserOrganization

سجلات عضوية المستخدم إلى المؤسسة.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| UserID | uint | المستخدم |
| OrganizationID | uint | المؤسسة |
| RoleID | uint | الدور داخل هذه المؤسسة |

```go
type UserOrganization struct {
    gorm.Model
    UserID         uint `gorm:"uniqueIndex:idx_user_org_user_org"`
    OrganizationID uint `gorm:"uniqueIndex:idx_user_org_user_org"`
    RoleID         uint
    User           User
    Organization   Organization
    Role           CustomRole
}
```

### ContactCollaborator

سجلات وصول التعاون لجهات الاتصال.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| ContactID | uint | جهة الاتصال المرتبطة |
| UserID | uint | المستخدم المتعاون |
| Status | string | "pending", "accepted", "declined" |

```go
type ContactCollaborator struct {
    gorm.Model
    ContactID uint   `gorm:"uniqueIndex:idx_collab_contact_user"`
    UserID    uint   `gorm:"uniqueIndex:idx_collab_contact_user"`
    Status    string `gorm:"default:pending"`
    Contact   Contact
    User      User
}
```

### SSOProvider

تكوينات مزوّدي SSO.

| الحقل | النوع | الوصف |
|-------|-------|-------|
| ID | uint | المفتاح الأساسي |
| OrganizationID | uint | المؤسسة الأم |
| Name | string | اسم المزوّد (google, azure, إلخ) |
| ClientID | string | معرّف عميل OAuth |
| ClientSecret | string | سر عميل OAuth مشفر |
| AuthorizationURL | string | نقطة نهاية تفويض OAuth |
| TokenURL | string | نقطة نهاية رمز OAuth |
| UserInfoURL | string | نقطة نهاية معلومات المستخدم |
| Scopes | JSONB | نطاقات OAuth |
| Enabled | bool | تفعيل المزوّد |

```go
type SSOProvider struct {
    gorm.Model
    Name             string         `gorm:"not null"`
    OrganizationID   uint           `gorm:"index"`
    ClientID         string         `gorm:"not null"`
    ClientSecret     string         // مشفر
    AuthorizationURL string
    TokenURL         string
    UserInfoURL      string
    Scopes           datatypes.JSON
    Enabled          bool           `gorm:"default:true"`
    DisplayName      string
}
```

## مخطط علاقات النماذج

```
Organization (1) ────< User (N)
Organization (1) ────< CustomRole (N) ────< Permission (N)
Organization (1) ────< WhatsAppAccount (N)
Organization (1) ────< WhatsAppInstance (N)
Organization (1) ────< Contact (N) ────< Message (N)
Organization (1) ────< BulkMessageCampaign (N) ────< BulkMessageRecipient (N)
Organization (1) ────< Template (N)
Organization (1) ────< ChatbotSettings (1)
Organization (1) ────< KeywordRule (N)
Organization (1) ────< ChatbotFlow (N)
Organization (1) ────< AIContext (N)
Organization (1) ────< CannedResponse (N)
Organization (1) ────< Tag (N)
Organization (1) ────< Team (N) ────< TeamMember (N) ────> User
Organization (1) ────< Webhook (N)
Organization (1) ────< CustomAction (N)
Organization (1) ────< ConversationNote (N)
Organization (1) ────< ActivityLog (N)
Organization (1) ────< Widget (N)
Organization (1) ────< LeadRequest (N)
Organization (1) ────< Notification (N)
Organization (1) ────< SSOProvider (N)
User (N) ────< UserOrganization (N) ────> Organization
Contact (1) ────< ContactCollaborator (N) ────> User
Contact (1) ────< AgentTransfer (N)
Contact (N) ────< Tag (N) [many-to-many via contact_tags]
```

## انظر أيضاً

- [نظرة عامة على البنية المعمارية](architecture.md) — بنية النظام وعلاقات المكونات
- [نظام التخزين المؤقت](caching.md) — أنماط التخزين المؤقت عبر Redis لبيانات النماذج
- [مرجع واجهة البرمجة (API)](api-reference.md) — نقاط نهاية REST API لكل نموذج
