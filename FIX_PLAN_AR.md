# خطة الإصلاح الشاملة — مشروع Whatomate

> **تاريخ الإعداد:** يونيو 2026  
> **الإصدار:** 1.0  
> **الهدف:** تحويل المنصة من حالة "مثيل واحد فقط" إلى بنية قابلة للتوسع الأفقي تدعم 50,000+ مستخدم متزامن

---

## ملخص المراحل

| المرحلة | المدة | الأولوية | عدد الإصلاحات | الهدف |
|---------|-------|----------|---------------|-------|
| **P0 — حرج** | أسبوع واحد | 🔴 حرج | 6 | استقرار الإنتاج ومنع الانهيار |
| **P1 — مهم** | 4 أسابيع | 🟠 عالي | 6 | إزالة حواجز التوسع الأفقي |
| **P2 — تحسين** | 8 أسابيع | 🟡 متوسط | 7 | أداء الإنتاج وتجربة المستخدم |
| **P3 — استراتيجي** | 12+ أسبوع | 🟢 منخفض | 7 | Enterprise-Grade ومراقبة متقدمة |

---

# المرحلة P0 — إصلاحات حرجة (أسبوع واحد)

> **الهدف:** منع انهيار الإنتاج تحت الحمل. كل إصلاح هنا يمثل خطرًا فوريًا على الاستقرار.

---

## P0-1: تهيئة Redis Connection Pool

**الملف:** `internal/database/redis.go:14-28`  
**التأثير:** 🔴 حرج — بدون PoolSize، Redis يستخدم القيمة الافتراضية (10 اتصالات فقط × CPU core)  
**الجهد:** ساعة واحدة

### المشكلة
```go
// الوضع الحالي — بدون أي تهيئة للـ pool
func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
        Password: cfg.Password,
        DB:       cfg.DB,
    })
```
- `PoolSize` = القيمة الافتراضية = `10 × runtime.GOMAXPROCS`
- `MinIdleConns` = 0 → إنشاء اتصالات باردة عند كل طلب مفاجئ
- لا `ReadTimeout` / `WriteTimeout` → قد يتجمد العامل إلى الأبد
- لا `MaxRetries` → فشل من أول محاولة

### الإصلاح المطلوب
```go
func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
    poolSize := cfg.PoolSize
    if poolSize == 0 {
        poolSize = 100
    }
    minIdle := cfg.MinIdleConns
    if minIdle == 0 {
        minIdle = 20
    }

    client := redis.NewClient(&redis.Options{
        Addr:         net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     poolSize,
        MinIdleConns: minIdle,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolTimeout:  4 * time.Second,
        MaxRetries:   3,
        MinRetryBackoff: 100 * time.Millisecond,
        MaxRetryBackoff: 500 * time.Millisecond,
    })
    // ... ping test
}
```

### تغييرات إضافية مطلوبة
**الملف:** `internal/config/config.go:90-95`
```go
type RedisConfig struct {
    Host         string `koanf:"host"`
    Port         int    `koanf:"port"`
    Password     string `koanf:"password"`
    DB           int    `koanf:"db"`
    PoolSize     int    `koanf:"pool_size"`      // جديد
    MinIdleConns int    `koanf:"min_idle_conns"` // جديد
}
```
**الملف:** `internal/config/config.go:302-304` — إضافة القيم الافتراضية:
```go
if cfg.Redis.PoolSize == 0 {
    cfg.Redis.PoolSize = 100
}
if cfg.Redis.MinIdleConns == 0 {
    cfg.Redis.MinIdleConns = 20
}
```

### كيفية التحقق
```bash
# مراقبة Redis بعد التغيير
redis-cli INFO clients | grep connected_clients
redis-cli INFO clients | grep blocked_clients
```

---

## P0-2: تقليم Redis Streams بـ MAXLEN

**الملف:** `internal/queue/redis.go:104-110` (وكل `XAdd` في الملف)  
**التأثير:** 🔴 حرج — بدون MAXLEN، الـ Streams تنمو بلا حدود → OOM  
**الجهد:** ساعتان

### المشكلة
كل `XAdd` في الملف لا يحدد `MaxLen`:
```go
q.client.XAdd(ctx, &redis.XAddArgs{
    Stream: CampaignStreamName(job.OrganizationID),
    Values: map[string]interface{}{
        "type":    string(JobTypeRecipient),
        "payload": string(payload),
    },
}).Result()
```
في حمل عالٍ (10,000 رسالة حمل ≈ 1KB لكل رسالة = 10MB/حمل)، الـ Stream تنمو بدون أي حد.

### الإصلاح المطلوب
إضافة ثابت جديد + تطبيقه على كل `XAdd`:
```go
const (
    // StreamMaxLen is the approximate maximum length for campaign streams.
    StreamMaxLen = 50000
)

// استبدال كل XAdd بإضافة MaxLen + Approximate trimming
q.client.XAdd(ctx, &redis.XAddArgs{
    Stream: CampaignStreamName(job.OrganizationID),
    MaxLen: StreamMaxLen,
    Approx: true, // استخدام ~ لتجنب حظر Redis
    Values: map[string]interface{}{
        "type":    string(JobTypeRecipient),
        "payload": string(payload),
    },
}).Result()
```

### المواقع المطلوب تعديلها (كل `XAdd` في الملف):
1. `EnqueueRecipient` — السطر 104
2. `EnqueueRecipients` (pipeline) — السطر 150
3. `EnqueueInboundMedia` — السطر 180
4. `EnqueueContactRepair` — السطر 211
5. `EnqueueWhatsAppFilter` — السطر 242
6. `EnqueueGroupJoin` — السطر 273
7. `EnqueueGroupJoins` (pipeline) — السطر 312
8. `EnqueueMessageExtraction` — السطر 344
9. `EnqueueGroupExtraction` — السطر 372
10. `EnqueueMemberExtraction` — السطر 400
11. `moveToDeadLetter` — السطر 780

### كيفية التحقق
```bash
# التأكد من أن الـ Stream لا يتجاوز MAXLEN
redis-cli XLEN whatomate:campaigns:<org-id>
redis-cli XINFO STREAM whatomate:campaigns:<org-id>
```

---

## P0-3: ضمان تسليم Pub/Sub لإحصائيات الحملات

**الملف:** `internal/queue/pubsub.go:44-57`  
**التأثير:** 🔴 حرج — إحصائيات الحملات قد تُفقد بدون أي تنبيه  
**الجهد:** 3 ساعات

### المشكلة
```go
func (p *Publisher) PublishCampaignStats(ctx context.Context, update *CampaignStatsUpdate) error {
    payload, err := json.Marshal(update)
    if err != nil {
        return err
    }
    if err := p.client.Publish(ctx, CampaignStatsChannel, payload).Err(); err != nil {
        p.log.Error("Failed to publish campaign stats", "error", err, "campaign_id", update.CampaignID)
        return err
    }
    return nil
}
```
Redis Pub/Sub = fire-and-forget:
- إذا لم يكن هناك subscriber نشط → الرسالة تضيع
- لا يوجد تخزين مؤقت للرسائل
- لا يوجد ACK من المستلم

### الإصلاح المطلوب (نهج مزدوج)

**الخيار A — إضافة Redis Stream احتياطي (موصى به):**
```go
const CampaignStatsStream = "whatomate:campaign_stats_stream"

func (p *Publisher) PublishCampaignStats(ctx context.Context, update *CampaignStatsUpdate) error {
    payload, err := json.Marshal(update)
    if err != nil {
        return err
    }

    // 1. نشر فوري عبر Pub/Sub (للتحديثات اللحظية)
    pubErr := p.client.Publish(ctx, CampaignStatsChannel, payload).Err()
    if pubErr != nil {
        p.log.Warn("Pub/Sub publish failed, falling back to stream", "error", pubErr)
    }

    // 2. كتابة إلى Stream (ضمان عدم الفقدان)
    if err := p.client.XAdd(ctx, &redis.XAddArgs{
        Stream: CampaignStatsStream,
        MaxLen: 1000,
        Approx: true,
        Values: map[string]interface{}{
            "campaign_id": update.CampaignID,
            "payload":     string(payload),
        },
    }).Err(); err != nil {
        p.log.Error("Failed to persist campaign stats to stream", "error", err)
        return err
    }

    return nil
}
```

**الخيار B — كتابة مباشرة لقاعدة البيانات (للموثوقية القصوى):**
تحديث `campaigns` table مباشرة بعد كل إرسال ناجح، واستخدام Pub/Sub فقط كقناة إشعار.

---

## P0-4: إضافة سجل إرسال محسّن للأخطاء (Send Policy Caching)

**الملف:** `internal/worker/send_policy.go:35-59`  
**التأثير:** 🔴 حرج — استعلام قاعدة بيانات لكل رسالة حملة = قفل القاعدة  
**الجهد:** 4 ساعات

### المشكلة
```go
func (w *Worker) loadOrganizationSendPolicy(orgID uuid.UUID) (organizationSendPolicy, error) {
    // ...
    var org models.Organization
    if err := w.DB.Select("id", "settings").Where("id = ?", orgID).First(&org).Error; err != nil {
        // ...
    }
    // ...
}
```
هذه الدالة تُستدعى لكل رسالة في الحملة. لحملة بـ 10,000 مستلم = 10,000 استعلام إلى PostgreSQL.

### الإصلاح المطلوب
إضافة طبقة تخزين مؤقت باستخدام Redis مع TTL:
```go
func (w *Worker) loadOrganizationSendPolicy(orgID uuid.UUID) (organizationSendPolicy, error) {
    cacheKey := fmt.Sprintf("whatomate:send_policy:%s", orgID)

    // محاولة القراءة من Redis أولاً
    cached, err := w.Redis.Get(context.Background(), cacheKey).Result()
    if err == nil {
        var policy organizationSendPolicy
        if json.Unmarshal([]byte(cached), &policy) == nil {
            return policy, nil
        }
    }

    // Fallback إلى قاعدة البيانات
    policy := organizationSendPolicy{
        StrictEnabled:     false,
        OutboundMode:      "mixed",
        ApplyToSystem:     true,
        CampaignDraftOnly: false,
    }
    // ... نفس الكود الحالي للقراءة من DB ...

    // حفظ في Redis لمدة 5 دقائق
    if data, err := json.Marshal(policy); err == nil {
        w.Redis.Set(context.Background(), cacheKey, data, 5*time.Minute)
    }

    return policy, nil
}
```

كذلك نفس المعالجة لـ `validateWhatsmeowCampaignInstance` (السطر 73-108) و `contactHasIncomingHistory` (السطر 62-71).

---

## P0-5: سياسة كلمات مرور أقوى

**الملف:** `internal/handlers/password_policy.go:1-35`  
**التأثير:** 🟠 عالي — لا يوجد فحص للأحرف الخاصة  
**الجهد:** ساعة واحدة

### المشكلة
```go
var hasLower, hasUpper, hasDigit bool
// ...
if !hasLower || !hasUpper || !hasDigit {
    return fmt.Errorf("password must include at least one uppercase letter, one lowercase letter, and one number")
}
```
لا يطلب أحرفًا خاصة (!@#$%^&*) — كلمات مرور مثل `Password123` تُقبل.

### الإصلاح المطلوب
```go
const (
    minPasswordLength = 12
    maxPasswordLength = 128
    minSpecialChars   = 1
)

func validatePasswordStrength(password string) error {
    if len(password) > maxPasswordLength {
        return fmt.Errorf("password must be at most %d characters", maxPasswordLength)
    }
    if len(password) < minPasswordLength {
        return fmt.Errorf("password must be at least %d characters", minPasswordLength)
    }

    var hasLower, hasUpper, hasDigit, hasSpecial bool
    for _, ch := range password {
        switch {
        case ch >= 'a' && ch <= 'z':
            hasLower = true
        case ch >= 'A' && ch <= 'Z':
            hasUpper = true
        case ch >= '0' && ch <= '9':
            hasDigit = true
        case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;':\",./<>?`~", ch):
            hasSpecial = true
        }
    }

    if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
        return fmt.Errorf("password must include at least one uppercase letter, one lowercase letter, one number, and one special character")
    }

    return nil
}
```

---

## P0-6: منع Webhook URLs غير مشفرة (HTTP)

**الملف:** `config.example.toml` + منطق التحقق  
**التأثير:** 🟠 عالي — بيانات حساسة عبر HTTP  
**الجهد:** ساعتان

### المشكلة
config يسمح بـ HTTP URLs في webhook_verify_token و base_url و redirect_uri.

### الإصلاح المطلوب
إضافة دالة تحقق في `internal/config/config.go`:
```go
func (c *Config) Validate() error {
    if c.App.Environment == "production" {
        if c.WhatsApp.BaseURL != "" && !strings.HasPrefix(c.WhatsApp.BaseURL, "https://") {
            return fmt.Errorf("whatsapp.base_url must use HTTPS in production")
        }
        if c.FacebookOAuth.RedirectURI != "" && !strings.HasPrefix(c.FacebookOAuth.RedirectURI, "https://") {
            return fmt.Errorf("facebook_oauth.redirect_uri must use HTTPS in production")
        }
        if c.Storage.Type == "s3" && !c.Storage.S3UseSSL {
            return fmt.Errorf("storage.s3_use_ssl must be true in production")
        }
    }
    return nil
}
```
استدعاء `Validate()` في نهاية `Load()`.

---

# المرحلة P1 — إزالة حواجز التوسع (4 أسابيع)

> **الهدف:** تمكين تشغيل مثيلات متعددة خلف Load Balancer

---

## P1-1: Redis-backed WebSocket Hub (الحاجز الأكبر)

**الملف:** `internal/websocket/hub.go:1-243`  
**التأثير:** 🔴 حرج — يمنع التوسع الأفقي تمامًا  
**الجهد:** 5 أيام

### المشكلة
```go
type Hub struct {
    clients map[uuid.UUID]map[uuid.UUID]map[*Client]struct{} // خريطة في الذاكرة فقط
    broadcast chan BroadcastMessage
    register  chan *Client
    unregister chan *Client
    mu        sync.RWMutex // قفل واحد لكل شيء
}
```
- الـ Hub يعمل في الذاكرة فقط — لا يمكن مشاركته بين مثيلات
- `sync.RWMutex` واحد يحمي كل العمليات
- لا يمكن تشغيل أكثر من مثيل واحد

### حل معماري مقترح (3 مراحل)

#### المرحلة 1: فصل البنية (أسبوع)
```go
// Hub interface — للسماح بالتبديل بين implementations
type HubBackend interface {
    BroadcastToOrg(ctx context.Context, orgID uuid.UUID, msg WSMessage) error
    BroadcastToUser(ctx context.Context, orgID, userID uuid.UUID, msg WSMessage) error
    BroadcastToContact(ctx context.Context, orgID, contactID uuid.UUID, msg WSMessage) error
}
```

#### المرحلة 2: Redis Pub/Sub Backend (أسبوعان)
```go
type RedisHubBackend struct {
    client *redis.Client
    localHub *LocalHub // يدير العملاء المحليين فقط
}

func (r *RedisHubBackend) BroadcastToOrg(ctx context.Context, orgID uuid.UUID, msg WSMessage) error {
    data, _ := json.Marshal(msg)
    return r.client.Publish(ctx, "ws:org:"+orgID.String(), data).Err()
}
```

#### المرحلة 3: Redis Streams Backend (أسبوعان)
للرسائل الحرجة (إشعارات المحادثات الجديدة)، استبدال Pub/Sub بـ Streams مع:
- Consumer Group لكل مثيل
- ACK + إعادة معالجة
- `MAXLEN ~ 10000`

---

## P1-2: Object Storage Retry + Circuit Breaker

**الملف:** `internal/storage/object_storage.go:1-233`  
**التأثير:** 🟠 عالي — فشل رفع الملفات = رسائل مفقودة  
**الجهد:** 3 أيام

### المشكلة
```go
func (s *s3ObjectStorage) PutObject(...) error {
    // لا يوجد retry
    // لا يوجد circuit breaker
    // فشل واحد = فقدان الملف
}
```

### الإصلاح المطلوب
إضافة retry مع exponential backoff:
```go
type retryableObjectStorage struct {
    inner    ObjectStorage
    maxRetry int
    baseDelay time.Duration
}

func (r *retryableObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
    var lastErr error
    for attempt := 0; attempt <= r.maxRetry; attempt++ {
        if attempt > 0 {
            delay := r.baseDelay * time.Duration(1<<(attempt-1))
            jitter := time.Duration(rand.Int63n(int64(delay) / 2))
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay + jitter):
            }
        }

        // إعادة إنشاء Reader لكل محاولة
        seekableBody, err := toSeekableReader(body, size)
        if err != nil {
            return err
        }

        lastErr = r.inner.PutObject(ctx, key, seekableBody, size, mimeType)
        if lastErr == nil {
            return nil
        }
    }
    return fmt.Errorf("PutObject failed after %d attempts: %w", r.maxRetry+1, lastErr)
}
```

---

## P1-3: إضافة Readiness Gate للتأكد من جاهزية العامل

**الملف:** `internal/worker/scaler.go:1-657`  
**التأثير:** 🟡 متوسط  
**الجهد:** 3 أيام

### الإصلاح
تأكد من أن كل عامل يمر بـ readiness check قبل استقبال مهام:
- الاتصال بقاعدة البيانات نشط
- اتصال Redis نشط
- مثيل WhatsApp متصل (للعاملين whatsmeow)

---

## P1-4: إضافة Observability شامل (Prometheus Metrics)

**الملفات الجديدة:** `internal/metrics/metrics.go`, `internal/metrics/prometheus.go`  
**التأثير:** 🟡 متوسط — بدون مقاييس، لا يمكن تشخيص المشاكل  
**الجهد:** 5 أيام

### المقاييس المطلوبة
```
whatomate_messages_sent_total{provider, org_id, direction}
whatomate_messages_failed_total{provider, org_id, error_type}
whatomate_campaign_recipients_total{campaign_id, status}
whatomate_websocket_clients_connected{org_id}
whatomate_redis_pool_active_connections
whatomate_redis_pool_idle_connections
whatomate_worker_jobs_processed_total{job_type, status}
whatomate_worker_job_duration_seconds{job_type}
whatomate_queue_depth{stream_name}
whatomate_queue_dead_letter_total{stream_name}
```

---

## P1-5: إضافة معالجة أخطاء محسّنة في ObjectStorage S3

**الملف:** `internal/storage/object_storage.go` (جزء MinIO)  
**التأثير:** 🟡 متوسط  
**الجهد:** يومان

### الإصلاح
```go
func newMinIOObjectStorage(cfg *config.StorageConfig) (ObjectStorage, error) {
    // ... existing code ...
    minioClient := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.S3Key, cfg.S3Secret, ""),
        Secure: cfg.S3UseSSL,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 100,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    })
    // ...
}
```

---

## P1-6: إضافة Health Check Endpoints شاملة

**الملفات:** `internal/handlers/health.go` (جديد أو تعديل)  
**التأثير:** 🟡 متوسط  
**الجهد:** يومان

### Endpoints المطلوبة
```
GET /health          → حالة عامة (للموازن)
GET /health/ready    → فحص جاهزية (DB + Redis + WhatsApp)
GET /health/live     → فحص حيوية (goroutine count + memory)
```

---

# المرحلة P2 — تحسين الأداء (8 أسابيع)

---

## P2-1: تحسين WebSocket Hub — sharding حسب المؤسسة

**الملف:** `internal/websocket/hub.go`  
**الجهد:** أسبوعان

### الإصلاح
```go
type ShardedHub struct {
    shards    [16]*LocalHub // 16 shard حسب orgID % 16
    shardFunc func(uuid.UUID) int
}

func (s *ShardedHub) getShard(orgID uuid.UUID) *LocalHub {
    hash := fnv.New32a()
    hash.Write(orgID[:])
    return s.shards[hash.Sum32()%16]
}
```
كل shard لديه `sync.RWMutex` خاص → تقليل المنافسة بنسبة 16×.

---

## P2-2: تحسين استعلامات قاعدة البيانات

**الملفات:** `internal/models/models.go`, `internal/handlers/*.go`  
**الجهد:** 3 أسابيع

### الفهارس المطلوبة
```sql
CREATE INDEX idx_messages_org_contact_created ON messages(organization_id, contact_id, created_at DESC);
CREATE INDEX idx_messages_org_direction ON messages(organization_id, direction, created_at DESC);
CREATE INDEX idx_conversations_org_status ON conversations(organization_id, status, updated_at DESC);
CREATE INDEX idx_campaign_recipients_campaign_status ON campaign_recipients(campaign_id, status);
```

### تحسين الاستعلامات
- استخدام `SELECT` محدد بدل `SELECT *`
- إضافة `LIMIT` لكل استعلام قائمة
- استخدام `Preload` بذكاء في GORM

---

## P2-3: إضافة Rate Limiting دقيق لكل مؤسسة

**الملف:** `internal/middleware/` (جديد أو تعديل)  
**الجهد:** أسبوعان

### الإصلاح
```go
type TenantRateLimiter struct {
    client *redis.Client
}

func (t *TenantRateLimiter) Allow(ctx context.Context, orgID uuid.UUID, limit int, window time.Duration) (bool, error) {
    key := fmt.Sprintf("ratelimit:org:%s:%d", orgID, time.Now().Unix()/int64(window.Seconds()))
    count, err := t.client.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    if count == 1 {
        t.client.Expire(ctx, key, window)
    }
    return count <= int64(limit), nil
}
```

---

## P2-4: إضافة Graceful Degradation للخدمات

**الملفات:** عبر `internal/handlers/app.go`  
**الجهد:** أسبوعان

### الإصلاح
- إذا Redis غير متاح → العمل في وضع degraded (بدون cache)
- إذا ObjectStorage غير متاح → تخزين مؤقت محلي + إعادة محاولة لاحقة
- إذا WhatsApp API غير متاح → طوابير تأخير تلقائية

---

## P2-5: تحسين حمل الدُفعات (Batch Operations)

**الملفات:** `internal/queue/redis.go`, `internal/handlers/campaign_scheduler.go`  
**الجهد:** أسبوع

### الإصلاح
- استخدام `XReadGroup` مع `Count: 50` بدل `Count: 1`
- معالجة دُفعات متوازية باستخدام goroutine pool
- Pipeline لكتابات قاعدة البيانات

---

## P2-6: إضافة Connection Draining عند الإيقاف

**الملف:** `cmd/whatomate/main.go`  
**الجهد:** 3 أيام

### الإصلاح
```go
// ترتيب الإيقاف:
// 1. التوقف عن قبول اتصالات جديدة
// 2. إكمال المهام الجارية (timeout 30s)
// 3. إغلاق اتصالات WebSocket بلطف
// 4. Flush أي طوابير متبقية
// 5. إغلاق DB + Redis
```

---

## P2-7: تحسين إدارة ذاكرة Whatsmeow Pool

**الملف:** `pkg/whatsmeow/pool.go`  
**الجهد:** أسبوع

### الإصلاح
- إضافة idle eviction للمثيلات غير المستخدمة
- تحسين حجم Pool حسب حمل المؤسسة
- إضافة metrics لاستخدام Pool

---

# المرحلة P3 — Enterprise Grade (12+ أسبوع)

---

## P3-1: نقل الأسرار إلى Environment Variables أو Vault

**الملفات:** `config.example.toml`, `internal/config/config.go`  
**الجهد:** أسبوعان

### الإصلاح
- جميع الحقول الحساسة يجب أن تُقرأ من env vars فقط في الإنتاج
- إضافة `config.Validate()` يمنع تشغيل الإنتاج بأسرار في config file
- دعم HashiCorp Vault أو AWS Secrets Manager (اختياري)

---

## P3-2: إضافة Distributed Tracing (OpenTelemetry)

**الملفات الجديدة:** `internal/tracing/`  
**الجهد:** 3 أسابيع

### الإصلاح
```go
import "go.opentelemetry.io/otel"

func (q *RedisQueue) EnqueueRecipient(ctx context.Context, job *RecipientJob) error {
    ctx, span := otel.Tracer("queue").Start(ctx, "EnqueueRecipient")
    defer span.End()
    span.SetAttributes(attribute.String("org_id", job.OrganizationID.String()))
    // ...
}
```

---

## P3-3: إضافة Audit Logging

**الملفات الجديدة:** `internal/audit/`  
**الجهد:** أسبوعان

### الإصلاح
تسجيل كل عملية حساسة:
- تسجيل دخول / خروج
- تغيير إعدادات المؤسسة
- إنشاء / تعديل / حذف حملات
- تغيير صلاحيات المستخدمين
- تصدير البيانات

---

## P3-4: إضافة Database Connection Pool Monitoring

**الملفات:** `internal/database/`  
**الجهد:** أسبوع

### الإصلاح
```go
db.DB().Stats() // مقاييس GORM
// نشر كمقاييس Prometheus
whatomate_db_max_open_connections
whatomate_db_open_connections
whatomate_db_in_use
whatomate_db_idle
whatomate_db_wait_count
whatomate_db_wait_duration_seconds
```

---

## P3-5: إضافة End-to-End Encryption للمحادثات

**الملفات:** `internal/crypto/`  
**الجهد:** 4 أسابيع

### الإصلاح
- تشفير محتوى الرسائل في قاعدة البيانات (استخدام AES-256-GCM الموجود)
- إضافة حقل `encrypted_content` لجدول messages
- فك التشفير عند العرض فقط للمستخدم المصرح له

---

## P3-6: إضافة Multi-Region Support

**الملفات:** عبر المشروع  
**الجهد:** 6 أسابيع

### الإصلاح
- Redis Cluster بدل instance واحد
- PostgreSQL read replicas
- CDN للملفات المرفوعة
- DNS-based routing

---

## P3-7: إضافة API Versioning

**الملفات:** `internal/handlers/`  
**الجهد:** أسبوعان

### الإصلاح
```
/api/v1/... — النسخة الحالية
/api/v2/... — النسخة المحسّنة (pagination، filtering أفضل)
```

---

# ملخص الملفات المتأثرة

| الملف | المراحل |
|-------|---------|
| `internal/database/redis.go` | P0-1 |
| `internal/config/config.go` | P0-1, P0-6, P3-1 |
| `internal/queue/redis.go` | P0-2 |
| `internal/queue/pubsub.go` | P0-3 |
| `internal/worker/send_policy.go` | P0-4 |
| `internal/handlers/password_policy.go` | P0-5 |
| `internal/websocket/hub.go` | P1-1, P2-1 |
| `internal/storage/object_storage.go` | P1-2, P1-5 |
| `internal/worker/scaler.go` | P1-3 |
| `cmd/whatomate/main.go` | P2-6 |
| `pkg/whatsmeow/pool.go` | P2-7 |
| `config.example.toml` | P3-1 |
| `internal/handlers/app.go` | P2-4 |

---

# معايير القبول لكل مرحلة

## P0 — حرج
- [ ] Redis pool يتعامل مع 500+ اتصال متزامن بدون خطأ
- [ ] Redis Streams لا تتجاوز 50,000 رسالة
- [ ] إحصائيات الحملات لا تُفقد عند إعادة تشغيل العامل
- [ ] Send policy cache يقلل استعلامات DB بنسبة 95%+
- [ ] كلمات المرور تتطلب حرفًا خاصًا واحدًا على الأقل
- [ ] لا يوجد HTTP URLs في تكوين الإنتاج

## P1 — مهم
- [ ] يمكن تشغيل مثيلين خلف Load Balancer مع WebSocket يعمل
- [ ] فشل S3 لا يفقد ملفات (retry 3 مرات)
- [ ] Prometheus metrics تُصدَر لكل مكون
- [ ] Health endpoints تعمل بشكل صحيح

## P2 — تحسين
- [ ] WebSocket Hub يتعامل مع 10,000 اتصال متزامن
- [ ] استعلامات الرسائل تستخدم فهارس محسّنة
- [ ] Rate limiting لكل مؤسسة يعمل
- [ ] Graceful shutdown لا يفقد أي رسالة

## P3 — استراتيجي
- [ ] لا أسرار في ملفات التكوين
- [ ] Distributed tracing يتبع رسالة من الاستلام إلى التسليم
- [ ] Audit log يسجل كل عملية حساسة
