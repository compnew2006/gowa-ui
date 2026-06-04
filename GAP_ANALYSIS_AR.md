# تحليل الفجوات وخارطة التحسين — منصة Whatomate

**التاريخ:** يونيو 2026  
**الإصدار:** 1.0  
**النطاق:** البنية الخلفية (Go + fastglue/fasthttp) + البنية التحتية (PostgreSQL 17 + Redis 7 + S3) + طبقة WebSocket

---

## 1. الفجوات المعمارية (Architectural Gaps)

### 1.1 مركزية WebSocket Hub — عنق زجاجة التوسع الأفقي

**الملف:** `internal/websocket/hub.go:12–30`

الـ `Hub` عبارة عن singleton يعمل بـ goroutine واحد عبر `select` على ثلاث قنوات فقط (`register`, `unregister`, `broadcast`). الخريطة `clients map[uuid.UUID]map[uuid.UUID]map[*Client]struct{}` محمية بـ `sync.RWMutex` واحد.

- **المشكلة:** عند آلاف الاتصالات المتزامنة، يصبح الـ mutex نقطة تضارب (contention). عملية `broadcastMessage` تقفل القراءة على كامل الخريطة لكل رسالة (hub.go:117–173).
- **التأثير:** لا يمكن تشغيل أكثر من نسخة خلفية (instance) من الخادم لأن الحالة في الذاكرة فقط — لا يوجد Redis Pub/Sub bridge.
- **التوصية:** تقديم Sharded Hub (by orgID) + Redis Pub/Sub لـ cross-instance broadcast.

### 1.2 غياب Circuit Breaker للاتصالات الخارجية

**الملفات:** `internal/database/redis.go:14–28`, `internal/storage/object_storage.go:120–144`, `pkg/whatsapp/` (Cloud API)

- اتصال Redis (redis.go:14–19) لا يحتوي على إعدادات `PoolSize`, `MinIdleConns`, `ConnMaxIdleTime`, `DialTimeout`, `ReadTimeout`, `RetryBackoff`.
- اتصال MinIO/S3 (object_storage.go:130–135) لا يملك retry logic أو circuit breaker.
- لا يوجد sonde صحية دورية (health check) لأي اتصال خارجي.
- **التوصية:** إضافة `github.com/sony/gobreaker` أو `github.com/streadway/handy/breaker` + إعدادات pool كاملة لـ Redis.

### 1.3 غياب Abstracted Event Bus

- النظام يستخدم Redis Streams مباشرة في `internal/queue/redis.go` بدون واجهة (interface). هذا يربط الكود بـ Redis بشكل محكم.
- **التوصية:** تعريف `EventBus` interface يسمح بالتبديل بين Redis Streams و Kafka أو NATS مستقبلاً.

### 1.4 إدارة التخزين — غياب Lifecycle Management

**الملف:** `internal/storage/object_storage.go`

- لا توجد آلية لتنظيف الملفات القديمة (retention policy).
- `fileSystemObjectStorage` (object_storage.go:67–82) لا يتحقق من path traversal — `key` يُدمج مباشرة مع `filepath.Join`.
- لا يوجد تحقق من حجم الملف قبل الرفع (size limit enforcement).
- **التوصية:** إضافة sanitization لـ key + تحقق من المسار + حد أقصى للحجم + S3 lifecycle rules.

---

## 2. مخاطر الإنتاج (Production Risks)

### 2.1 Redis Configuration — أحادي النقطة (Single Point of Failure)

**الملف:** `internal/database/redis.go:14–19` + `internal/config/config.go:90–95`

```go
client := redis.NewClient(&redis.Options{
    Addr:     net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
    Password: cfg.Password,
    DB:       cfg.DB,
})
```

- **لا يوجد:** `PoolSize` (افتراضي = 10*GOMAXPROCS في go-redis v9 لكن بدون تحكم)، `MinIdleConns`، `ConnMaxIdleTime`، `DialTimeout`، `ReadTimeout`، `WriteTimeout`، retry policy.
- `RedisConfig` (config.go:90–95) لا تعرض أي إعدادات pool.
- **المخاطرة:** تحت حمل عالي، يتم استنفاد الاتصالات بدون إعادة استخدام كافية. انقطاع Redis يُسقط النظام بالكامل.
- **التوصية:** إضافة `RedisPoolConfig` كاملة مع قيم افتراضية آمنة + sentinel/cluster support.

### 2.2 Rate Limiter — Fixed-Window مع ثغرات

**الملف:** `internal/middleware/ratelimit.go`

- Fixed-window counting (INCR + EXPIRE) يسمح بـ burst عند حدود النافذة (2× الحد المسموح في transition period).
- Fails-closed عند خطأ Redis (رفض جميع الطلبات) — هذا جيد للأمان لكن سيء للتوفر.
- **التوصية:** الانتقال إلى sliding-window (sorted set) أو token bucket مع fail-open مؤقت مع logging.

### 2.3 غياب Graceful Shutdown موثوق

**الملف:** `cmd/whatomate/main.go`

- الـ signal handling يتوفر لكن لا يوجد drain واضح لـ Redis Streams consumers — رسائل قيد المعالجة قد تُفقد أو تُكرر.
- WebSocket clients لا يحصلون على إشعار `CloseGoingAway` قبل الإغلاق.
- **التوصية:** إضافة `Shutdown(ctx)` method لكل component (Hub, Worker, Queue consumer) مع timeout.

### 2.4 تسرب بيانات الاعتماد في Config

**الملف:** `internal/config/config.go:56–65`

```go
EncryptionKey string `koanf:"encryption_key"`
```

- `AIConfig` يحتوي على مفاتيح API مباشرة (config.go:139–143): `OpenAIKey`, `AnthropicKey`, `GoogleKey`.
- لا يوجد تحقق من عدم طباعة الـ config في logs (logf لا يطبعه لكن أي خطأ في التهيئة قد يكشفه).
- **التوصية:** استخدام secret store أو على الأقل mark الحقول كـ `***` في أي log output + تحقق من redaction في error paths.

### 2.5 غياب Health Check شامل

**الملف:** `internal/observability/observability.go`

- الـ observability manager يقدم مقاييس جيدة (DB pool, Redis pool, goroutines, heap) لكن:
  - لا يوجد `/health` endpoint يتحقق فعلياً من PostgreSQL + Redis + S3.
  - `/ready` غير مُعرَّف بشكل منفصل (readiness vs liveness).
- **التوصية:** إضافة readiness probe تتحقق من: DB ping + Redis ping + S3 head-bucket (optional).

---

## 3. تحسينات الأداء (Performance Optimization)

### 3.1 WebSocket Broadcast — تكرار Marshal

**الملف:** `internal/websocket/hub.go:126–127`

```go
data, err := json.Marshal(msg.Message)
```

- يتم `json.Marshal` لكل رسالة broadcast حتى لو أُرسلت لنفس المستخدم عدة مرات. لا يوجد caching للرسائل المتكررة.
- **التوصية:** pre-marshal مرة واحدة خارج القفل وتمرير `[]byte` بدلاً من `WSMessage`.

### 3.2 Observability Mutex Contention

**الملف:** `internal/observability/observability.go:184–215`

- `observeRequest` تقفل `m.mu` (mutex واحد) لكل طلب HTTP. هذا يعني كل request يتنافس على نفس القفل.
- **التوصية:** استخدام `sync.Map` أو sharded counters أو atomic counters مع periodic aggregation.

### 3.3 Redis Pool غير مُحسَّن

**الملف:** `internal/database/redis.go:15–19`

- إعدادات pool الافتراضية في go-redis v9 (`PoolSize = 10 * GOMAXPROCS`) قد تكون زائدة أو ناقصة حسب الحمل.
- **التوصية:** إضافة `PoolSize`, `MinIdleConns` (≈ 25% من PoolSize), `ConnMaxIdleTime` (5min), `ConnMaxLifetime` (30min) إلى config.

### 3.4 GORM Performance

**الملف:** `internal/database/postgres.go`

- يجب التحقق من:
  - استخدام `PrepareStmt: true` لprepared statements.
  - تجنب N+1 queries في handlers عبر `Preload()` أو joins.
  - إضافة فهرسة (indexing) على الأعمدة المستخدمة في `WHERE` المتكررة.

### 3.5 Queue Consumer — Batch Processing

**الملف:** `internal/queue/redis.go`

- `XREADGROUP` يستخدم `COUNT(1)` — معالجة رسالة واحدة في كل مرة.
- **التوصية:** استخدام `COUNT(10-50)` مع batch processing لتقليل round-trips إلى Redis.

---

## 4. تحسينات الاستقرار (Stability Improvements)

### 4.1 إعادة محاولات الاتصال (Connection Retries)

**الملف:** `internal/database/redis.go:22–25`

```go
if err := client.Ping(ctx).Err(); err != nil {
    return nil, fmt.Errorf("failed to connect to redis: %w", err)
}
```

- إذا فشل الاتصال الأول بـ Redis، يفشل التطبيق بالكامل بدون إعادة محاولة.
- **التوصية:** إضافة retry with exponential backoff (3-5 محاولات) أثناء التهيئة.

### 4.2 DLQ Monitoring

**الملف:** `internal/queue/redis.go` (DLQ streams)

- يوجد DLQ لكن لا يوجد:
  - تنبيه عند accumulation في DLQ.
  - آلية إعادة معالجة تلقائية من DLQ.
  - Dashboard لعرض الرسائل الفاشلة.
- **التوصية:** إضافة Prometheus counter لـ DLQ entries + periodic reprocessing job + admin UI.

### 4.3 WebSocket Client Cleanup

**الملف:** `internal/websocket/client.go:96–161`

- `ReadPump` يتعافى من panic (client.go:98–100) — جيد. لكن لا يوجد timeout على الاتصالات الصامتة (zombie connections).
- `pongWait = 60s` و `pingPeriod = 54s` — كافيان لكن يجب مراقبة عدد zombies.
- **التوصية:** إضافة periodic sweep للاتصالات التي لم تتجاوز ping/pong بنجاح + metric لعدد zombies.

### 4.4 Worker Crash Recovery

**الملف:** `internal/worker/worker.go`

- Pending message recovery يحدث كل 30s (queue/redis.go) بـ `ClaimMinIdleTime = 5min`.
- إذا تعطل الـ worker أثناء معالجة رسالة، ستنتظر 5 دقائق قبل أن يلتقطها worker آخر.
- **التوصية:** تقليل `ClaimMinIdleTime` إلى 2-3 دقائق + إضافة heartbeat mechanism.

### 4.5 File Upload Safety

**الملف:** `internal/storage/object_storage.go:67–82`

```go
func (s *fileSystemObjectStorage) PutObject(ctx context.Context, key string, body io.Reader, size int64, mimeType string) error {
    path := filepath.Join(s.rootPath, key)
```

- `key` لا يتم تطهيره (sanitize) — يمكن أن يحتوي على `../` مما يتيح path traversal.
- لا يوجد تحقق من `size` — يمكن رفع ملفات بحجم غير محدود.
- `io.Copy` بدون limit — استهلاك ذاكرة/قرص غير محدود.
- **التوصية:** إضافة `filepath.Clean` + تحقق من أن المسار يبدأ بـ `rootPath` + `io.LimitReader` + حد أقصى configurable.

---

## 5. مشاكل القابلية للصيانة (Maintainability Issues)

### 5.1 Tight Coupling بين Worker و Whatsmeow

**الملف:** `internal/worker/worker.go`

- الـ worker يحتوي على حقل `whatsmeowMgr` مباشرة — dependency injection محدود.
- **التوصية:** استخدام `MessageProvider` interface (المعرّف في `pkg/provider/interface.go`) بشكل كامل في worker.

### 5.2 غياب Structured Errors

- الأخطاء عبر الكود عبارة عن `fmt.Errorf` بسلاسل نصية — لا يوجد error codes أو error types.
- **التوصية:** تعريف error types مع codes (`ErrNotFound`, `ErrUnauthorized`, `ErrRateLimited`) + استخدام `errors.Is/As`.

### 5.3 Observability — مقاييس مفقودة

**الملف:** `internal/observability/observability.go`

- مقاييس HTTP متوفرة (request counts, latency histogram, DB pool, Redis pool) — ممتاز.
- **مفقود:**
  - WebSocket connections count per org
  - Queue depth per stream (pending messages)
  - Campaign send rate / error rate
  - WhatsApp API call latency + error rate
  - Worker job processing time
  - Storage operation latency
- **التوصية:** إضافة مقاييس لكل component رئيسي.

### 5.4 اختبارات التكامل

- الاختبارات تتطلب PostgreSQL + Redis حقيقيين (لا mocks لـ DB).
- `-p 1` مطلوب لتجنب تضارب قاعدة البيانات — بطيء في CI.
- **التوصية:** 
  - استخدام test containers مع isolated schemas لكل package.
  - إضافة unit tests مع interfaces/mockable dependencies.
  - فصل integration tests عن unit tests.

### 5.5 Documentation وCode Comments

- الكود يحتوي على minimal comments — معظم الدوال بدون doc comments.
- **التوصية:** إضافة godoc comments لكل exported function/type.

---

## 6. خارطة التحسين ذات الأولويات (Prioritized Roadmap)

### P0 — حرج (Critical) — يجب التنفيذ قبل الإنتاج

| # | التوصية | الجهد | الملفات المعنية |
|---|---------|-------|----------------|
| P0.1 | إضافة Redis pool tuning (PoolSize, MinIdleConns, Timeouts) | S | `internal/database/redis.go`, `internal/config/config.go` |
| P0.2 | Path traversal protection في file storage | S | `internal/storage/object_storage.go` |
| P0.3 | إضافة readiness/liveness health checks | S | `internal/observability/observability.go`, handlers |
| P0.4 | Secret redaction في logs وerror paths | S | `internal/config/config.go` |
| P0.5 | Graceful shutdown مع drain لكل component | M | `cmd/whatomate/main.go`, hub, worker, queue |
| P0.6 | تحديد حجم ملف الرفع الأقصى | S | `internal/storage/object_storage.go` |

### P1 — مهم (High) — يجب التنفيذ خلال الربع الأول

| # | التوصية | الجهد | الملفات المعنية |
|---|---------|-------|----------------|
| P1.1 | Circuit breaker لـ Redis + S3 + WhatsApp API | M | `internal/database/redis.go`, `internal/storage/object_storage.go` |
| P1.2 | إضافة DLQ monitoring + Prometheus counters | M | `internal/queue/redis.go`, `internal/observability/` |
| P1.3 | WebSocket Hub sharding (per-org) | L | `internal/websocket/hub.go` |
| P1.4 | Structured error types مع codes | M | عبر الكود |
| P1.5 | إضافة مقاييس لـ WebSocket + Queue + Worker + Storage | M | `internal/observability/observability.go` |
| P1.6 | Connection retry مع exponential backoff | S | `internal/database/redis.go` |

### P2 — متوسط (Medium) — خلال الربع الثاني

| # | التوصية | الجهد | الملفات المعنية |
|---|---------|-------|----------------|
| P2.1 | Redis Pub/Sub bridge لـ cross-instance WS broadcast | L | `internal/websocket/hub.go`, جديد |
| P2.2 | Sliding-window rate limiter | M | `internal/middleware/ratelimit.go` |
| P2.3 | Batch processing في Redis Streams consumer | M | `internal/queue/redis.go` |
| P2.4 | EventBus interface لفصل queue abstraction | L | `internal/queue/`, جديد |
| P2.5 | File retention / lifecycle management | M | `internal/storage/object_storage.go` |
| P2.6 | Pre-marshal optimization في WS broadcast | S | `internal/websocket/hub.go` |
| P2.7 | Observability atomic/sharded counters | M | `internal/observability/observability.go` |

### P3 — تحسين طويل الأمد (Low) — حسب الحاجة

| # | التوصية | الجهد | الملفات المعنية |
|---|---------|-------|----------------|
| P3.1 | Redis Sentinel/Cluster support | XL | `internal/database/redis.go`, `internal/config/` |
| P3.2 | Testcontainers isolation + unit/integration split | XL | `test/`, CI config |
| P3.3 | Godoc comments لكل exported symbol | M | عبر الكود |
| P3.4 | Worker heartbeat + faster claim | M | `internal/queue/redis.go`, `internal/worker/` |
| P3.5 | Zombie WebSocket connection sweeper | S | `internal/websocket/hub.go` |
| P3.6 | GORM prepared statements + N+1 audit | M | `internal/database/`, `internal/handlers/` |
| P3.7 | S3 lifecycle rules for media retention | S | Infrastructure (Terraform/CloudFormation) |

---

## مقياس الجهد

- **S (Small):** < يوم واحد — تغيير ملف واحد أو اثنين
- **M (Medium):** 1–3 أيام — تغيير عدة ملفات في نفس الحزمة
- **L (Large):** 3–7 أيام — تغيير عبر حزم متعددة أو تصميم جديد
- **XL (Extra Large):** 1–2 أسبوع — إعادة هيكلة رئيسية أو infrastructure changes

---

## ملخص تنفيذي

المنصة تتمتع بأساس معماري جيد مع:
- Observability مدمج مع Prometheus metrics + pprof
- Multi-tenancy مع tenant isolation
- Queue system مع DLQ و retry logic
- Dual provider support (Meta + Whatsmeow)

**الفجوات الحرجة** تتركز في:
1. **Redis configuration** — pool tuning مفقود بالكامل (redis.go:28 سطر فقط)
2. **File storage safety** — path traversal + حجم غير محدود
3. **WebSocket scalability** — singleton hub بدون path للتوسع الأفقي
4. **Missing health checks** — لا يوجد readiness probe حقيقي
5. **Graceful shutdown** — drain غير مكتمل

**الأولوية القصوى:** P0 items (6 عناصر بجهد S–M) يمكن إنجازها في أسبوع واحد وتغلق أهم الثغرات الإنتاجية.
