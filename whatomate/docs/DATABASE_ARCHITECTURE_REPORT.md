# تقرير تحليل هندسة قاعدة البيانات والتخزين — منصة Whatomate

## 1. هندسة قاعدة البيانات (Database Architecture)

### 1.1 الإعداد العام

تعتمد المنصة على **PostgreSQL 17** كقاعدة بيانات رئيسية مع **GORM** كـ ORM، و **Redis 7** للتخزين المؤقت وطوابير المهام.

**ملف الإعداد:** `internal/config/config.go`

```go
// internal/config/config.go:77-88
type DatabaseConfig struct {
    Host            string `koanf:"host"`
    Port            int    `koanf:"port"`
    User            string `koanf:"user"`
    Password        string `koanf:"password"`
    Name            string `koanf:"name"`
    SSLMode         string `koanf:"ssl_mode"`
    LogSQL          bool   `koanf:"log_sql"`
    MaxOpenConns    int    `koanf:"max_open_conns"`
    MaxIdleConns    int    `koanf:"max_idle_conns"`
    ConnMaxLifetime int    `koanf:"conn_max_lifetime"`
}
```

**القيم الافتراضية لحجم الاتصالات:**

```go
// internal/config/config.go:293-301
if cfg.Database.MaxOpenConns == 0 {
    cfg.Database.MaxOpenConns = 25      // أقصى عدد اتصالات مفتوحة
}
if cfg.Database.MaxIdleConns == 0 {
    cfg.Database.MaxIdleConns = 5       // اتصالات خاملة
}
if cfg.Database.ConnMaxLifetime == 0 {
    cfg.Database.ConnMaxLifetime = 300  // 5 دقائق
}
```

### 1.2 النماذج الأساسية (Core Models)

**الملف:** `internal/models/models.go`

#### BaseModel — الأساس المشترك

```go
// internal/models/models.go:80-100 (تقريباً)
type BaseModel struct {
    ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
```

**ملاحظات:**
- استخدام **UUID** كمعرف أساسي (حجم 128-bit)
- **Soft Delete** عبر `gorm.DeletedAt` في كل النماذج التي ترث من `BaseModel`
- الحقول الزمنية `CreatedAt`/`UpdatedAt` تُدار تلقائياً بواسطة GORM

#### النماذج الرئيسية:

| النموذج | الوصف | الفهرسة |
|---------|-------|----------|
| `User` | المستخدم مع `Email` كفهرس فريد | `uniqueIndex` على البريد |
| `Organization` | المؤسسة مع `Slug` فريد | `uniqueIndex` على الـ slug |
| `UserOrganization` | ربط المستخدم بالمؤسسة (علاقة N:N) | `uniqueIndex:idx_user_org` على (UserID, OrganizationID) |
| `Contact` | جهات الاتصال | `index` على OrganizationID |
| `Message` | الرسائل | `index` على OrganizationID, WhatsAppAccount, Status, AssignedUserID |
| `WhatsAppAccount` | حسابات واتساب | `uniqueIndex:idx_wa_org_name` على Name لكل مؤسسة |
| `Media` | الوسائط | `uniqueIndex` على FileHash |
| `Tag` | الوسوم | `index` على OrganizationID |

#### نوع JSONB المخصص

```go
// internal/models/models.go — نوع JSONB مخصص
type JSONB map[string]interface{}

// يدعم Scan و Value لواجهة sql.Scanner و driver.Valuer
```

يُستخدم بشكل مكثف في:
- `Organization.Settings`
- `WhatsAppInstance.Settings`
- `ChatbotSettings` (إعدادات البوت)
- `WhatsAppAccount.Settings`

### 1.3 تعدد المستأجرين (Multi-Tenancy)

**الملف:** `internal/tenant/scope.go`

#### آلية العزل

```go
// internal/tenant/scope.go:29-61
func ScopedDB(db *gorm.DB, orgID uuid.UUID) *gorm.DB {
    return db.Session(&gorm.Session{}).Scopes(func(tx *gorm.DB) *gorm.DB {
        // يبحث عن حقل OrganizationID في النموذج
        field := tx.Statement.Schema.LookUpField("OrganizationID")
        if field == nil {
            return tx  // إذا لم يوجد، يمرر بدون فلترة
        }
        // يضيف WHERE organization_id = ? تلقائياً
        return tx.Where(clause.Eq{
            Column: clause.Column{Table: tx.Statement.Schema.Table, Name: field.DBName},
            Value:  orgID,
        })
    })
}
```

**كيفية عمل العزل:**
1. كل نموذج يمتلك حقل `OrganizationID uuid.UUID` مع `gorm:"type:uuid;index;not null"`
2. الدالة `ScopedDB` تنشئ جلسة GORM جديدة مع Scope يضيف `WHERE organization_id = ?` تلقائياً
3. التحقق من وجود الحقل يتم ديناميكياً عبر `LookUpField("OrganizationID")`

#### تحديد المؤسسة

```go
// internal/tenant/scope.go:103-154
func ResolveOrganizationID(r *fastglue.Request, db *gorm.DB) (uuid.UUID, error) {
    // 1. أولاً: فحص الـ Host (subdomain)
    //    مثال: org1.example.com → slug = "org1"
    
    // 2. ثانياً: من سياق JWT (user_id, organization_id)
    
    // 3. ثالثاً: من header X-Organization-ID (تجاوز)
    //    - يتحقق أن المستخدم عضو في المؤسسة المطلوبة
    //    - Super Admin يمكنه تجاوز لأي مؤسسة موجودة
}
```

**دعم الـ Subdomain:**

```go
// internal/tenant/scope.go:174-199
func hostOrganizationSlug(host string) string {
    // يستخرج الـ slug من الـ hostname
    // org1.example.com → "org1"
    // org1.localhost → "org1"
    // localhost / IP addresses → "" (لا يوجد slug)
}
```

### 1.4 الفهارس (Indexes)

**عدد الفهارس الإجمالي عبر جميع ملفات النماذج:** ~200 فهرس

**أنواع الفهارس:**

1. **فهارس فردية:** `gorm:"type:uuid;index;not null"` على `OrganizationID` في كل نموذج
2. **فهارس فريدة:**
   - `User.Email` — `uniqueIndex`
   - `Organization.Slug` — `uniqueIndex`
   - `UserOrganization` — `uniqueIndex:idx_user_org` (UserID + OrganizationID)
   - `WhatsAppAccount.Name` — `uniqueIndex:idx_wa_org_name`
   - `Media.FileHash` — `uniqueIndex`
   - `Permission` — `uniqueIndex:idx_permission_resource_action` (Resource + Action)
3. **فهارس مركبة:**
   - `BulkMessageRecipient` — `uniqueIndex:idx_bulk_recipients_campaign_phone_normalized` (CampaignID + PhoneNormalized)
   - `AgentSelectionSettings` — `uniqueIndex:idx_agent_selection_settings_scope` (OrganizationID + InstanceID)
   - `AgentSelectionParticipant` — `uniqueIndex:idx_agent_selection_participant_user` مع `where:deleted_at IS NULL`
   - `FBComment` — `uniqueIndex:idx_fb_comment_external_org` (OrganizationID + ExternalID)
   - `ContactUserDeletion` — `uniqueIndex:idx_contact_user_deletions` (OrganizationID + ContactID + UserID)
4. **فهارس شرطية:** `where:deleted_at IS NULL` و `where:jid <> ''` و `where:whats_app_message_id <> ''`

### 1.5 العلاقات (Relationships)

```
Organization ──1:N──> User (via UserOrganization)
Organization ──1:N──> Contact
Organization ──1:N──> Message
Organization ──1:N──> WhatsAppAccount
Organization ──1:N──> WhatsAppInstance
Organization ──1:N──> ChatbotSettings
Organization ──1:N──> Tag
Organization ──1:N──> Webhook
Organization ──1:N──> BulkMessageCampaign
Organization ──1:N──> CustomRole
Organization ───1:1──> OrganizationConfig (uniqueIndex)

User ──N:N──> Organization (via UserOrganization)
User ──1:N──> Message (as sender/assigned)
User ──1:N──> Contact (as collaborator)
User ───M:1──> CustomRole (via RoleID)

Contact ──1:N──> Message
Contact ──N:N──> Tag (via contact_tags)
Contact ──N:N──> User (via ContactCollaborator)

WhatsAppAccount ──1:N──> Message
WhatsAppInstance ──1:N──> Message

BulkMessageCampaign ──1:N──> BulkMessageRecipient
CustomRole ──N:N──> Permission (via role_permissions)

ChatbotSettings ──1:N──> ChatbotFlow ──1:N──> ChatbotFlowStep
```

### 1.6 المعاملات (Transactions)

**استخدام المعاملات محدود — فقط في 8 ملفات:**

```
internal/handlers/legacy_media_restore.go  — 1
internal/handlers/agent_selection.go        — 1
internal/handlers/roles.go                  — 2
internal/handlers/chat_cleanup.go           — 2
internal/handlers/media_retention_worker.go — 1
internal/handlers/widgets.go                — 1
internal/handlers/contact_repair.go         — 1
internal/license/service.go                 — 1
```

**ملاحظة:** العديد من العمليات التي تعدّل عدة جداول لا تستخدم معاملات، مما قد يؤدي إلى حالة عدم اتساق عند الفشل.

---

## 2. طبقة التخزين (Storage Layer)

### 2.1 واجهة التخزين

**الملف:** `internal/storage/object_storage.go`

```go
type ObjectStorage interface {
    PutObject(ctx context.Context, key string, reader io.Reader, contentType string) error
    GetObject(ctx context.Context, key string) (io.ReadCloser, error)
    DeleteObject(ctx context.Context, key string) error
}
```

### 2.2 التنفيذات

#### تخزين محلي (Filesystem)

```go
// internal/storage/object_storage.go
type FileSystemStorage struct {
    basePath string
}
```

- يخزن الملفات في مجلد محلي (افتراضي: `./uploads`)
- ينشئ هيكل مجلدات فرعي من الـ key

#### تخزين S3/MinIO

```go
// internal/storage/object_storage.go
type S3Storage struct {
    client    *minio.Client
    bucket    string
}
```

- يدعم أي S3-compatible storage (MinIO, AWS S3, etc.)
- الإعداد عبر `config.toml`:

```go
// internal/config/config.go:145-154
type StorageConfig struct {
    Type       string `koanf:"type"`        // local, s3
    LocalPath  string `koanf:"local_path"`
    S3Bucket   string `koanf:"s3_bucket"`
    S3Region   string `koanf:"s3_region"`
    S3Key      string `koanf:"s3_key"`
    S3Secret   string `koanf:"s3_secret"`
    S3Endpoint string `koanf:"s3_endpoint"`
    S3UseSSL   bool   `koanf:"s3_use_ssl"`
}
```

### 2.3 نموذج الوسائط

```go
// internal/models/models.go
type Media struct {
    BaseModel
    OrganizationID uuid.UUID `gorm:"type:uuid;index;not null"`
    FileName       string
    FileType       string
    FileSize       int64
    URL            string
    StorageKey     string
    FileHash       string `gorm:"size:64;uniqueIndex;not null"` // SHA-256 hash
    WhatsAppURL    string   // رابط واتساب الأصلي
    MediaType      string   // image, video, audio, document, sticker
    MimeType       string
    Caption        string
}
```

---

## 3. طبقة التخزين المؤقت (Cache Layer)

### 3.1 إعداد Redis

**الملف:** `internal/database/redis.go` (28 سطر فقط)

```go
func NewRedisClient(cfg config.RedisConfig) *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password,
        DB:       cfg.DB,
    })
}
```

**إعدادات Redis:**

```go
// internal/config/config.go:90-95
type RedisConfig struct {
    Host     string `koanf:"host"`
    Port     int    `koanf:"port"`
    Password string `koanf:"password"`
    DB       int    `koanf:"db"`
}
```

**ملاحظة:** لا يوجد تكوين لحجم Pool أو Timeout أو أي إعدادات متقدمة.

### 3.2 أنماط التخزين المؤقت

**الملف:** `internal/handlers/cache.go` (837 سطر)

#### TTL موحد: 6 ساعات

```go
// internal/handlers/cache.go:18-28
const (
    settingsCacheTTL        = 6 * time.Hour
    flowsCacheTTL           = 6 * time.Hour
    keywordRulesCacheTTL    = 6 * time.Hour
    whatsappAccountCacheTTL = 6 * time.Hour
    webhooksCacheTTL        = 6 * time.Hour
    slaSettingsCacheTTL     = 6 * time.Hour
    aiContextsCacheTTL      = 6 * time.Hour
    userPermissionsCacheTTL = 6 * time.Hour
    rolePermissionsCacheTTL = 6 * time.Hour
    tagsCacheTTL            = 6 * time.Hour
)
```

#### مفاتيح التخزين المؤقت

| النمط | مثال | TTL |
|-------|------|-----|
| `chatbot:settings:{orgID}:{account}` | إعدادات البوت لكل حساب | 6h |
| `chatbot:flows:{orgID}` | تدفقات البوت | 6h |
| `chatbot:keywords:{orgID}:{account}` | قواعد الكلمات المفتاحية | 6h |
| `whatsapp:account:{phoneID}` | حساب واتساب (يحتوي أسرار مشفرة!) | 6h |
| `webhooks:{orgID}` | خطاطيف الويب | 6h |
| `chatbot:sla_enabled_settings` | إعدادات SLA (عالمي!) | 6h |
| `chatbot:ai_contexts:{orgID}:{account}` | سياقات الذكاء الاصطناعي | 6h |
| `permissions:user:{userID}` | صلاحيات المستخدم | 6h |
| `permissions:user:{userID}:{orgID}` | صلاحيات المستخدم في مؤسسة | 6h |
| `permissions:role:{roleID}` | صلاحيات الدور | 6h |
| `tags:{orgID}` | الوسوم | 6h |
| `ratelimit:{prefix}:{ip}` | عدادات تحديد المعدل | window seconds |

### 3.3 آلية الإبطال (Cache Invalidation)

#### إبطال مباشر

```go
// internal/handlers/cache.go:329-336
func (a *App) InvalidateWhatsAppAccountCache(phoneID string) {
    ctx := context.Background()
    cacheKey := fmt.Sprintf("%s%s", whatsappAccountCachePrefix, phoneID)
    a.Redis.Del(ctx, cacheKey)
}
```

#### إبطال بنمط (Pattern-based)

```go
// internal/handlers/cache.go:234-239
func (a *App) deleteKeysByPattern(ctx context.Context, pattern string) {
    iter := a.Redis.Scan(ctx, 0, pattern, 100).Iterator()
    for iter.Next(ctx) {
        a.Redis.Del(ctx, iter.Val())
    }
}
```

**تحذير:** استخدام `SCAN` + `DEL` في حلقة قد يكون بطيئاً مع عدد كبير من المفاتيح.

#### إبطال متسلسل (Cascading Invalidation)

```go
// internal/handlers/cache.go:734-755
func (a *App) InvalidateRolePermissionsCache(roleID uuid.UUID) {
    // 1. يحذف cache الدور
    a.Redis.Del(ctx, roleCacheKey)
    
    // 2. يبحث عن كل المستخدمين الذين لديهم هذا الدور
    var users []models.User
    a.DB.Select("id, organization_id").Where("role_id = ?", roleID).Find(&users)
    
    // 3. يبطل cache كل مستخدم
    for _, user := range users {
        a.Redis.Del(ctx, userCacheKey)
        a.notifyUserPermissionsChanged(user.ID) // WebSocket notification
    }
}
```

### 3.4 طوابير Redis Streams

**الملف:** `internal/queue/redis.go` (823 سطر)

#### أنواع الطوابير

| Stream | الوصف | نطاق |
|--------|-------|------|
| `whatomate:campaigns` | حملات المراسلة | عالمي |
| `whatomate:campaigns:{orgID}` | حملات المراسلة | لكل مؤسسة |
| `whatomate:campaigns:{orgID}:dlq` | رسائل ميتة | لكل مؤسسة |
| `whatomate:inbound_media` | استرداد الوسائط | عالمي |
| `whatomate:inbound_media:dlq` | رسائل ميتة للوسائط | عالمي |

#### أنواع المهام (8 أنواع)

```go
// internal/queue/queue.go
type RecipientJob struct {...}           // إرسال رسالة حملة
type InboundMediaJob struct {...}        // استرداد وسائط واردة
type ContactRepairJob struct {...}       // إصلاح جهات اتصال
type WhatsAppFilterJob struct {...}      // فلترة واتساب
type GroupJoinJob struct {...}           // الانضمام لمجموعة
type MessageExtractionJob struct {...}   // استخراج رسائل
type GroupExtractionJob struct {...}     // استخراج مجموعات
type MemberExtractionJob struct {...}    // استخراج أعضاء
```

#### آلية المعالجة

```go
// internal/queue/redis.go:531-604
func (c *RedisConsumer) Consume(ctx context.Context, handler JobHandler) error {
    // 1. يطالب بالرسائل المعلقة أولاً (من عمال متعطلين)
    c.claimPendingMessages(ctx, handler)
    
    // 2. حلقة رئيسية:
    for {
        // فحص Readiness Gate
        // فحص دوري للرسائل المعلقة (كل 30 ثانية)
        // XReadGroup مع Block: 5 ثواني
        // معالجة الرسالة
        // ACK + XDel
    }
}
```

**إعدادات الموثوقية:**

```go
// internal/queue/redis.go:17-47
BlockTimeout       = 5 * time.Minute    // وقت انتظار الرسائل
ClaimMinIdleTime   = 5 * time.Minute    // وقت خمول قبل المطالبة
PendingClaimInterval = 30 * time.Second // فاصل فحص المعلقة
MaxDeliveryAttempts = int64(5)          // محاولات قبل DLQ
```

---

## 4. التشفير (Encryption)

### 4.1 نظام التشفير

**الملف:** `internal/crypto/crypto.go`

#### ثلاثة إصدارات من التشفير

```go
// بادئات الإصدارات
const (
    legacyPrefix = "enc:"   // الإصدار القديم
    prefixV2     = "enc2:"  // الإصدار الثاني
    prefixV3     = "enc3:"  // الإصدار الحالي
)
```

| الإصدار | البادئة | الخوارزمية | اشتقاق المفتاح |
|---------|---------|-----------|---------------|
| V1 (Legacy) | `enc:` | AES-CBC | SHA-256 hash |
| V2 | `enc2:` | AES-GCM | SHA-256 hash |
| V3 (Current) | `enc3:` | AES-256-GCM | Argon2id |

#### التشفير الحالي (V3)

```go
// Argon2id لاشتقاق المفتاح من كلمة المرور
// AES-256-GCM للتشفير الفعلي
// nonce عشوائي 12 bytes
```

#### سياسة فك التشفير

```go
func DecryptWithPolicy(ciphertext, key string, allowLegacy bool) (string, error) {
    // إذا enc3: → فك تشفير V3
    // إذا enc2: → فك تشفير V2
    // إذا enc:  → فك تشفير V1 (legacy) — فقط إذا allowLegacy=true
    // بدون بادئة → نص عادي (legacy غير مشفر)
}
```

### 4.2 البيانات المشفرة

**الأعمدة المشفرة في قاعدة البيانات:**

```go
// internal/crypto/migrate_db.go:32-37
var migrationTargets = []migrationTarget{
    {Table: "whatsapp_accounts", Column: "access_token"},
    {Table: "whatsapp_accounts", Column: "app_secret"},
    {Table: "chatbot_settings", Column: "ai_api_key"},
    {Table: "sso_providers", Column: "client_secret"},
}
```

### 4.3 أداة الترحيل

**الملف:** `internal/crypto/migrate_db.go`

```go
func MigrateEncryptedColumns(db *gorm.DB, key string, opts MigrationOptions, logger logf.Logger) ([]MigrationSummary, error) {
    // يعالج على دفعات (BatchSize: 500 افتراضياً)
    // يدعم DryRun
    // يبحث عن enc: و enc2: (اختياري)
    // يفك التشفير القديم ثم يعيد التشفير بـ V3
}
```

**ملاحظة:** أداة الترحيل لا تستخدم معاملات — كل صف يُحدّث بشكل مستقل.

---

## 5. المخاوف المتعلقة بالأداء (Performance Concerns)

### 5.1 مشاكل محتملة

#### 5.1.1 تحميل مسبق (Eager Loading) محدود

**استخدام `Preload` في 18 ملف فقط:**

```
internal/handlers/contacts_management.go — 7 استدعاءات (Preload("ClosedByUser") مكرر)
internal/handlers/contacts.go             — 4
internal/handlers/teams.go                — 4
internal/handlers/saved_contents.go       — 3
internal/handlers/users.go                — 3
```

**استخدام `Joins` في 3 ملفات فقط:**

```
internal/handlers/agent_transfers.go — 9 (الأكثر)
internal/handlers/contacts.go        — 1
internal/handlers/teams.go           — 1
```

**الخطر:** العديد من الاستعلامات قد تعاني من مشكلة N+1 حيث يتم تحميل العلاقات بشكل منفصل.

#### 5.1.2 إعدادات Redis بسيطة جداً

```go
// internal/database/redis.go — 28 سطر فقط
// لا يوجد تكوين لـ:
// - PoolSize
// - MinIdleConns
// - ConnMaxIdleTime
// - ReadTimeout / WriteTimeout
// - MaxRetries
```

#### 5.1.3 تخزين أسرار مشفرة في Redis Cache

```go
// internal/handlers/cache.go:285-293
// يتم تخزين AccessToken و AppSecret في Redis cache
cacheData := whatsAppAccountCache{
    WhatsAppAccount: account,
    AccessToken:     account.AccessToken,  // مشفر!
    AppSecret:       account.AppSecret,    // مشفر!
}
// #nosec G117 — cached secrets are stored encrypted-at-rest
a.Redis.Set(ctx, cacheKey, data, whatsappAccountCacheTTL)
```

**ملاحظة:** الأسرار تُخزن مشفرة (بتنسيق enc3:) في Redis، ويتم فك التشفير عند القراءة فقط. لكن هذا يزيد من حجم البيانات المخزنة مؤقتاً ويتطلب عمليات تشفير/فك تشفير إضافية.

#### 5.1.4 SLA Settings Cache عالمي

```go
// internal/handlers/cache.go:37
slaSettingsCacheKey = "chatbot:sla_enabled_settings"  // مفتاح واحد عالمي!

// internal/handlers/cache.go:388
a.DB.Where("sla_enabled = ?", true).Find(&settings)  // بدون فلترة مؤسسة!
```

**الخطر:** هذا الاستعلام يجلب كل إعدادات SLA من كل المؤسسات في استعلام واحد ويخزنها في مفتاح واحد. مع نمو البيانات سيصبح بطيئاً جداً.

#### 5.1.5 إبطال الـ Cache عبر SCAN

```go
// internal/handlers/cache.go:234-239
func (a *App) deleteKeysByPattern(ctx context.Context, pattern string) {
    iter := a.Redis.Scan(ctx, 0, pattern, 100).Iterator()
    for iter.Next(ctx) {
        a.Redis.Del(ctx, iter.Val())  // DEL لكل مفتاح بشكل منفصل!
    }
}
```

**الخطر:** مع عدد كبير من المفاتيح، هذا بطيء. الأفضل استخدام `UNLINK` أو Lua script.

#### 5.1.6 معاملات محدودة

العمليات التي تعدّل عدة جداول لا تستخدم معاملات في معظم الحالات:
- إنشاء/تحديث جهات الاتصال مع الوسوم
- عمليات الترحيل (crypto migration)
- backfill البيانات

```go
// internal/database/assigned_chat_reset_backfill.go
// هذا الملف يعدّل عدة سجلات بدون معاملة
for _, org := range organizations {
    // يعدّل كل instance بشكل منفصل
    for _, instance := range instances {
        db.Model(&models.WhatsAppInstance{}).Where(...).Update(...)
    }
    // يعدّل المؤسسة
    db.Model(&models.Organization{}).Where(...).Update(...)
}
```

#### 5.1.7 إعدادات Pool لـ PostgreSQL

```go
// internal/config/config.go:293-301
MaxOpenConns:    25  // مناسب للتطبيقات المتوسطة
MaxIdleConns:    5   // قد يكون منخفضاً — 25% فقط من MaxOpen
ConnMaxLifetime: 300 // 5 دقائق — قصير نسبياً
```

**توصية:** رفع `MaxIdleConns` إلى ~10-15 لتقليل إنشاء الاتصالات الجديدة.

### 5.2 نقاط القوة

1. **فهرسة شاملة:** ~200 فهرس عبر جميع النماذج، بما في ذلك فهارس فريدة مركبة
2. **عزل متعدد المستأجرين آلي:** عبر GORM Scopes مع فحص ديناميكي للحقل
3. **تشفير متطور:** AES-256-GCM مع Argon2id ودعم ترحيل من إصدارات قديمة
4. **طوابير موثوقة:** Redis Streams مع DLQ و claim للرسائل المعلقة وحد أقصى للمحاولات
5. **Soft Delete:** في كل النماذج عبر `gorm.DeletedAt` مع فهارس شرطية `where:deleted_at IS NULL`
6. **UUID كمعرفات:** يمنع تخمين المعرفات ويصلح للأنظمة الموزعة

### 5.3 توصيات

1. **إضافة PoolSize لإعدادات Redis** لتجنب استنفاد الاتصالات
2. **فحص مشاكل N+1** في المعالجات التي تستخدم `Find` بدون `Preload`
3. **استخدام المعاملات** في العمليات التي تعدّل عدة جداول
4. **تحسين إبطال الـ Cache** باستخدام `UNLINK` بدل `DEL` و Lua scripts للإبطال الجماعي
5. **تقسيم SLA cache** لكل مؤسسة بدلاً من مفتاح عالمي واحد
6. **رفع MaxIdleConns** لـ PostgreSQL من 5 إلى 10-15
7. **إضافة ReadTimeout/WriteTimeout** لإعدادات Redis
