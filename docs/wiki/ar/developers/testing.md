---
title: بنية الاختبار
rtl: true
lang: ar
---

<div dir="rtl">بنية الاختبار</div>

يستخدم Whatomate نهج اختبار متعدد الطبقات مع اختبارات وحدة Go لمنطق الخلفية واختبارات Playwright E2E للواجهة الأمامية.

## أنواع الاختبارات

| النوع | الموقع | الإطار | الغرض |
|-------|--------|--------|-------|
| اختبارات الوحدة | ملفات `*_test.go` | حزمة Go `testing` | منطق المعالجات، الأدوات، المساعدات |
| اختبارات التكامل | ملفات `*_test.go` | Go `testing` + قاعدة بيانات اختبار | تكامل المعالج مع قاعدة البيانات |
| اختبارات E2E | `frontend/e2e/` | Playwright (TypeScript) | سير عمل المستخدم كامل المكدس |

## اختبارات الوحدة

### تسمية ملفات الاختبار

تتبع ملفات الاختبار اتفاقية `*_test.go` بجانب ملفات المصدر:

```
internal/handlers/
├── auth_handlers.go
├── auth_handlers_test.go          # اختبارات معالج المصادقة
├── send_restriction_policy_helpers_test.go  # اختبارات قيود الإرسال
├── contacts_helpers_test.go       # اختبارات أدوات جهات الاتصال
├── organization_delete_test.go    # اختبارات حذف المؤسسة
├── sla_processor_test.go          # اختبارات معالج SLA
└── testhelpers_test.go            # أدوات اختبار مشتركة
```

### تشغيل اختبارات الوحدة

```bash
# تشغيل جميع الاختبارات
go test ./...

# تشغيل اختبارات لحزمة محددة
go test ./internal/handlers/

# تشغيل مع إخراج مفصّل
go test -v ./internal/handlers/

# تشغيل مع التغطية
go test -coverprofile=coverage.out ./...

# تشغيل دالة اختبار محددة
go test -run TestLogin ./internal/handlers/
```

### أدوات الاختبار

يوفر **`testhelpers_test.go`** أدوات اختبار شائعة:

```go
// إعداد اتصال قاعدة بيانات الاختبار
func setupTestDB() *gorm.DB { ... }

// إنشاء مستخدم اختبار مع دور
func createTestUser(db *gorm.DB, role string) *models.User { ... }

// إنشاء مؤسسة اختبار
func createTestOrg(db *gorm.DB) *models.Organization { ... }

// عميل Redis وهمي
func setupMockRedis() *redis.Client { ... }
```

يوفر **`stubs.go`** تطبيقات وهمية:

```go
// StubMessageProvider للاختبار بدون WhatsApp حقيقي
type StubMessageProvider struct {
    SentMessages []OutgoingMessage
}

func (s *StubMessageProvider) SendMessage(msg OutgoingMessage) error {
    s.SentMessages = append(s.SentMessages, msg)
    return nil
}
```

### تقارير التغطية

يتم إنشاء ملفات التغطية لكل تشغيل اختبار:

```
coverage.out              # تقرير التغطية الرئيسي
coverage_handlers.out     # تغطية حزمة المعالجات
coverage_worker.out       # تغطية حزمة العامل
coverage_crypto.out       # تغطية حزمة التشفير
```

عرض التغطية في المتصفح:

```bash
go tool cover -html=coverage.out
```

## اختبارات E2E

### الموقع

```
frontend/e2e/
├── auth.spec.ts           # تسجيل الدخول، التسجيل، تسجيل الخروج
├── contacts.spec.ts       # CRUD جهات الاتصال، البحث، التصفية
├── messaging.spec.ts      # إرسال/استلام الرسائل
├── campaigns.spec.ts      # إنشاء الحملات وإدارتها
├── chatbot.spec.ts        # تكوين الشات بوت
├── instances.spec.ts      # إدارة مثيلات WhatsApp
└── helpers/
    └── ApiHelper.ts       # أداة اختبار API بـ TypeScript
```

### تشغيل اختبارات E2E

```bash
# تثبيت Playwright
cd frontend && npx playwright install

# تشغيل اختبارات E2E
npx playwright test

# تشغيل مع واجهة المستخدم
npx playwright test --ui

# تشغيل ملف اختبار محدد
npx playwright test auth.spec.ts

# تشغيل مع تقرير
npx playwright test --reporter=html
```

### ApiHelper

توفر فئة `ApiHelper` بـ TypeScript وصولاً برمجياً لـ API لاختبارات E2E:

```typescript
class ApiHelper {
  private baseUrl: string;
  private authToken: string;

  async login(email: string, password: string): Promise<void> { ... }
  async get(path: string): Promise<Response> { ... }
  async post(path: string, body: any): Promise<Response> { ... }
  async put(path: string, body: any): Promise<Response> { ... }
  async delete(path: string): Promise<Response> { ... }

  // أدوات خاصة بالاختبار
  async createContact(data: ContactData): Promise<Contact> { ... }
  async sendMessage(contactId: number, content: string): Promise<Message> { ... }
  async createCampaign(data: CampaignData): Promise<Campaign> { ... }
}
```

## قاعدة بيانات الاختبار

تستخدم اختبارات التكامل قاعدة بيانات اختبار منفصلة:

```toml
# config.test.toml
[database]
host = "127.0.0.1"
port = 5432
user = "whatomate_test"
password = "test_password"
dbname = "whatomate_test"
ssl_mode = "disable"
```

تنشئ الاختبارات وتنظّف بياناتها الخاصة باستخدام المعاملات:

```go
func TestCreateUser(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)

    tx := db.Begin()
    defer tx.Rollback()

    // منطق الاختبار مع المعاملة
    user := createTestUser(tx, "admin")
    assert.NotNil(t, user.ID)
}
```

## أنماط الاختبار

### اختبار المعالجات

```go
func TestLogin(t *testing.T) {
    // الإعداد
    db := setupTestDB()
    app := setupTestApp(db)
    createTestUser(db, "admin")

    // التنفيذ
    req := createLoginRequest("admin@test.com", "password123")
    resp := app.Login(req)

    // التحقق
    assert.Equal(t, 200, resp.StatusCode())
    assert.Contains(t, string(resp.Body()), "access_token")
}
```

### اختبار المزوّد الوهمي

```go
func TestSendMessage(t *testing.T) {
    stub := &StubMessageProvider{}
    app := setupTestAppWithProvider(stub)

    app.SendMessage(createMessageRequest())

    assert.Len(t, stub.SentMessages, 1)
    assert.Equal(t, "Hello", stub.SentMessages[0].Content)
}
```

## اختبار CI

يتم تشغيل الاختبارات في CI بـ:

```bash
# بدء تبعيات الاختبار
docker compose -f docker-compose.test.yml up -d

# تشغيل اختبارات الخلفية
go test -coverprofile=coverage.out ./...

# تشغيل اختبارات E2E للواجهة الأمامية
cd frontend && npx playwright test

# رفع التغطية
go tool cover -func=coverage.out
```

## انظر أيضاً

- [دليل المساهمة](contributing.md) — نمط الكود ومتطلبات طلبات السحب
- [نظرة عامة على البنية المعمارية](architecture.md) — مكونات النظام لاختبارها
- [أنماط معالجة الأخطاء](architecture.md#error-handling) — أنماط الأخطاء لاختبارها
