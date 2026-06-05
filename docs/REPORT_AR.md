# تقرير تحليل تدفقات البيانات في منصة Whatomate

## دليل شامل لتدفق الرسائل ودورة حياة المحادثات والتحديثات الفورية

---

## 1. تدفق الرسائل من البداية إلى النهاية (Message Flow End-to-End)

### 1.1 واجهة مزود الرسائل (MessageProvider Interface)

تستخدم المنصة واجهة موحدة `MessageProvider` في الملف `pkg/provider/interface.go` لتجريد التعامل مع مزودي WhatsApp المختلفين. تدعم الواجهة نوعين من المزودين:

- **Meta Cloud API**: عبر بروتوكول WhatsApp Business API الرسمي
- **Whatsmeow**: عبر بروتوكول WhatsApp Web

```go
type MessageProvider interface {
    SendText(ctx, account, recipient, text) (msgID, error)
    SendImage(ctx, account, recipient, mediaURL, caption) (msgID, error)
    SendDocument(ctx, account, recipient, mediaURL, filename, caption) (msgID, error)
    SendVideo(ctx, account, recipient, mediaURL, caption) (msgID, error)
    SendAudio(ctx, account, recipient, mediaURL) (msgID, error)
    MarkRead(ctx, account, messageID) error
    SendReaction(ctx, account, messageID, emoji) error
    RevokeMessage(ctx, account, messageID) error
    GetMediaURL(ctx, account, mediaID) (string, error)
    DownloadMedia(ctx, mediaURL) ([]byte, error)
    UploadMedia(ctx, account, data, mimeType, filename) (mediaID, error)
    SendTemplate(ctx, account, recipient, template, params) (msgID, error)
    SendInteractive(ctx, account, recipient, content) (msgID, error)
}
```

يتم اختيار المزود عبر إعداد `[whatsapp].provider` في ملف `config.toml` ويتم توصيله في `cmd/whatomate/main.go`.

---

### 1.2 إرسال الرسائل الصادرة (Outbound Messages)

**الملف**: `internal/handlers/messages.go`

هيكل طلب الرسالة الصادرة:

```go
type OutgoingMessageRequest struct {
    ContactID       string `json:"contact_id"`
    WhatsAppAccount string `json:"whatsapp_account"`
    MessageType     string `json:"message_type"`     // text, image, document, video, audio, template, interactive
    Content         string `json:"content"`
    MediaURL        string `json:"media_url"`
    Caption         string `json:"caption"`
    Filename        string `json:"filename"`
    ReplyToID       string `json:"reply_to_id"`      // UUID للرسالة الأصلية
    TemplateID      string `json:"template_id"`
    TemplateParams  map[string]interface{} `json:"template_params"`
    InteractiveContent *InteractiveContent `json:"interactive_content"`
}
```

**مسار الإرسال الموحد**:
1. المصادقة عبر JWT أو API key (`internal/middleware/`)
2. التحقق من نطاق المؤسسة (TenantScope middleware)
3. تحليل الطلب والتحقق من صحة البيانات
4. استدعاء `MessageProvider` المناسب
5. حفظ الرسالة في قاعدة البيانات مع حالة `sent`
6. بث إشعار WebSocket من نوع `TypeNewMessage` إلى المؤسسة

**الملف**: `internal/handlers/contacts_messaging.go` — نقطة الدخول الثانية للإرسال:

```go
type SendMessageRequest struct {
    PhoneNumber      string                `json:"phone_number"`
    Message          string                `json:"message"`
    TemplateID       string                `json:"template_id"`
    TemplateParams   map[string]interface{} `json:"template_params"`
    InteractiveContent *InteractiveContent  `json:"interactive_content"`
}
```

---

### 1.3 استقبال الرسائل الواردة (Inbound Messages)

**الملف**: `internal/handlers/webhook.go`

#### لمزود Meta:
1. **GET `/webhook`**: تحقق من الاشتراك عبر `hub.challenge`
2. **POST `/webhook`**: استقبال إشعارات WhatsApp Cloud API

#### لمزود Whatsmeow:
- استقبال مباشر من اتصال WhatsApp Web عبر `pkg/whatsmeow/`

**الملف**: `internal/handlers/chatbot_processor.go` — معالجة الرسائل الواردة:

```go
type IncomingTextMessage struct {
    Text       string `json:"text"`
    Interactive *struct {
        Type   string `json:"type"`
        Button *struct {
            Title   string `json:"title"`
            Payload string `json:"payload"`
        } `json:"button"`
        ListReply *struct {
            ID    string `json:"id"`
            Title string `json:"title"`
        } `json:"list_reply"`
    } `json:"interactive"`
    Image    *MediaMessage `json:"image"`
    Document *MediaMessage `json:"document"`
    Audio    *MediaMessage `json:"audio"`
    Video    *MediaMessage `json:"video"`
    Sticker  *MediaMessage `json:"sticker"`
    Location *LocationMessage `json:"location"`
    Contacts []ContactInfo `json:"contacts"`
    Button   *ButtonMessage `json:"button"`
}
```

**مسار المعالجة**:
1. استقبال الرسالة من Webhook أو Whatsmeow
2. البحث عن جهة الاتصال أو إنشاؤها
3. حفظ الرسالة في قاعدة البيانات
4. معالجة Chatbot (إذا كان نشطاً)
5. فحص كلمات التحويل إلى وكيل (Keyword triggers)
6. بث إشعار WebSocket

---

### 1.4 تحديثات حالة الرسائل (Message Status Updates)

**الملف**: `internal/handlers/statuses.go`

حالات الرسائل بالترتيب:
```
pending → sent → delivered → read
                 ↘ failed
```

- يتم استقبال تحديثات الحالة عبر Webhook (Meta) أو Events (Whatsmeow)
- تحديث الرسالة في قاعدة البيانات
- بث إشعار WebSocket من نوع `TypeStatusUpdate`

---

### 1.5 الحملات التسويقية (Campaigns)

**الملف**: `internal/handlers/campaigns.go`

#### دورة حياة الحملة:
```
Draft → Queued → Processing → Completed
                   ↘ Paused → Processing (resume)
                   ↘ Cancelled
                   ↘ Failed → Processing (retry)
```

#### هيكل الحملة:

```go
type CampaignRequest struct {
    Name            string     `json:"name"`
    WhatsAppAccount string     `json:"whatsapp_account"`
    TemplateID      string     `json:"template_id"`
    BodyContent     string     `json:"body_content"`
    HeaderMediaID   string     `json:"header_media_id"`
    MinDelaySeconds *int       `json:"min_delay_seconds"`  // افتراضي: 20
    MaxDelaySeconds *int       `json:"max_delay_seconds"`  // افتراضي: 45
    ScheduledAt     *time.Time `json:"scheduled_at"`
}
```

#### استيراد المستلمين (`ImportRecipients`):
- حد أقصى: 10,000 مستلم (قابل للتكوين)
- تطبيع أرقام الهاتف (`normalizeCampaignRecipientPhone`)
- إزالة التكرار عبر `phone_normalized`
- فحص سياسة "الوارد فقط" (`shouldEnforceInboundOnlyForSystemSends`)

#### بدء الحملة (`StartCampaign`):
1. التحقق من الحالة (`Draft`)
2. إنشاء وظائف Redis Stream لكل مستلم (`RecipientJob`)
3. تغيير حالة الحملة إلى `Queued`/`Processing`

#### طوابير Redis:

**الملف**: `internal/queue/redis.go`

```go
// أسماء الطوابير
func CampaignStreamName(orgID uuid.UUID) string {
    return fmt.Sprintf("whatomate:campaigns:%s", orgID)
}
const InboundMediaStream = "whatomate:inbound_media"
```

- طوابير بنطاق المؤسسة: `whatomate:campaigns:{orgID}`
- DLQ: حد أقصى 5 محاولات تسليم
- مجموعات المستهلكين: `whatomate:campaigns_cg`

#### إيصالات الحملة:

**الملف**: `internal/campaignstats/receipts.go`

```go
func ApplyMessageReceipt(ctx, db, publisher, log, message, newStatus) {
    // 1. البحث عن الحملة من metadata["campaign_id"]
    // 2. تحديث حالة المستلم
    // 3. تحديث العدادات: delivered_count, read_count, failed_count
    // 4. نشر إحصائيات عبر Redis Pub/Sub
}
```

---

## 2. دورة حياة المحادثات (Conversation Lifecycle)

### 2.1 حالات المحادثة

**الملف**: `internal/handlers/chat_lifecycle.go`

حالات المحادثة الثلاث:

```
Pending  → محادثة جديدة بدون وكيل معين
Open     → محادثة مع وكيل معين
Closed   → محادثة مغلقة
```

### 2.2 تحويل الوكلاء (Agent Transfers)

**الملف**: `internal/handlers/agent_transfers.go`

هذا هو النظام الأساسي لإدارة تدفق المحادثات بين البوت والوكلاء البشريين.

#### هيكل التحويل:

```go
type AgentTransfer struct {
    ID                    uuid.UUID
    OrganizationID        uuid.UUID
    ContactID             uuid.UUID
    WhatsAppAccount       string
    PhoneNumber           string
    Status                TransferStatus  // active, resumed, completed
    Source                TransferSource  // manual, flow, keyword
    AgentID               *uuid.UUID
    TeamID                *uuid.UUID
    TransferredByUserID   *uuid.UUID
    Notes                 string
    TransferredAt         time.Time
    SLAResponseDeadline   *time.Time
    SLAResolutionDeadline *time.Time
    SLABreached           bool
    EscalationLevel       int
    ExpiresAt             *time.Time
}
```

#### مصادر التحويل:
- **manual**: تحويل يدوي من واجهة المستخدم
- **flow**: تحويل تلقائي من تدفق Chatbot
- **keyword**: تحويل ناتج عن كلمة مفتاحية

#### مسار التحويل (`CreateAgentTransfer`):

1. **التحقق من الصلاحيات** (`ResourceTransfers:ActionWrite`)
2. **منع التحويل المكرر**: التحقق من عدم وجود تحويل نشط
3. **تحديد الوكيل** (بالأولوية):
   - وكيل محدد صراحة في الطلب
   - تخصيص عبر استراتيجية الفريق (`assignToTeam`)
   - إعادة التعيين للوكيل السابق (إذا كان `AssignToSameAgent` مفعلًا)
   - بدون تعيين → يذهب إلى الطابور
4. **التحقق من وصول الوكيل** (`validateTransferAssigneeAccess`)
5. **إنشاء التحويل** مع تحديد SLA
6. **إنهاء جلسة Chatbot** النشطة
7. **بث إشعار WebSocket** (`broadcastTransferCreated`)
8. **إطلاق Webhook** (`WebhookEventTransferCreated`)

#### استراتيجيات التخصيص للفرق (`assignToTeam`):

```go
switch team.AssignmentStrategy {
case AssignmentStrategyRoundRobin:
    return assignToTeamRoundRobin(teamID, orgID)
    // يختار الوكيل الأقل تعييناً مؤخراً (ORDER BY last_assigned_at ASC)
case AssignmentStrategyLoadBalanced:
    return assignToTeamLoadBalanced(teamID, orgID)
    // يختار الوكيل بأقل عدد من التحويلات النشطة
case AssignmentStrategyManual:
    return nil  // بدون تخصيص تلقائي
}
```

#### استئناف المحادثة (`ResumeFromTransfer`):

1. التحقق من الصلاحيات
2. تحديث حالة التحويل إلى `resumed`
3. مسح تتبع Chatbot (`ClearContactChatbotTracking`)
4. إذا كان `AssignToSameAgent` معطلاً → إلغاء تعيين جهة الاتصال
5. بث إشعار WebSocket (`broadcastTransferResumed`)

#### التقاط من الطابور (`PickNextTransfer`):

يستخدم **قفل صفقة** لمنع ظروف السباق:

```go
tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
    Where("organization_id = ? AND status = ? AND agent_id IS NULL",
        orgID, TransferStatusActive).
    Order("transferred_at ASC")  // FIFO
```

- `FOR UPDATE SKIP LOCKED`: يمنع تحويلين من التقاط نفس العنصر
- ترتيب FIFO (الأقدم أولاً)
- فلترة حسب عضوية الفريق للمستخدمين العاديين

#### إعادة التحويلات عند عدم الاتاحة:

```go
func (a *App) ReturnAgentTransfersToQueue(userID, orgID uuid.UUID) int {
    // 1. البحث عن جميع التحويلات النشطة للوكيل
    // 2. إلغاء تعيين الوكيل (agentID = nil)
    // 3. إلغاء تعيين جهة الاتصال
    // 4. بث إشعارات WebSocket لكل تحويل
}
```

---

### 2.3 معالجة SLA (اتفاقية مستوى الخدمة)

**الملف**: `internal/handlers/sla_processor.go`

```go
type SLAProcessor struct {
    DB       *gorm.DB
    Hub      *websocket.Hub
    Interval time.Duration  // قابل للتكوين
}
```

- يعمل على **فاصل زمني قابل للتكوين** (Ticker)
- `processStaleTransfers`: يعالج التحويلات المتأخرة لكل مؤسسة
- `SetSLADeadlines`: يحدد مواعيد نهائية للاستجابة والحل
- `UpdateSLAOnPickup`: يسجل وقت التقاط التحويل
- `ClearContactChatbotTracking`: يمنع تفعيل SLA بعد إغلاق التحويل

---

### 2.4 معالجة Chatbot

**الملف**: `internal/handlers/chatbot_processor.go`

مسار معالجة الرسائل الواردة:

```
رسالة واردة
  ↓
البحث عن جلسة Chatbot نشطة
  ↓
┌─────────────────────┐
│ جلسة نشطة موجودة؟    │
├───── نعم ───────────┤
│ معالجة في التدفق الحالي│
│ فحص عقدة التحويل      │
│ (Transfer Node)       │
├───── لا ────────────┤
│ إنشاء جلسة جديدة      │
│ بدء التدفق الافتراضي   │
└─────────────────────┘
  ↓
فحص كلمات مفتاحية للتحويل
  ↓
┌─────────────────────┐
│ كلمة مفتاحية مطابقة؟  │
├───── نعم ───────────┤
│ createTransferFromKeyword│
├───── لا ────────────┤
│ رد Chatbot تلقائي     │
└─────────────────────┘
```

#### إنشاء تحويل من كلمة مفتاحية (`createTransferFromKeyword`):

```go
// 1. فحص ساعات العمل
if settings.BusinessHours.Enabled && !isWithinBusinessHours(settings.BusinessHours.Hours) {
    // إرسال رسالة خارج ساعات العمل
    sendAndSaveTextMessage(account, contact, outOfHoursMessage)
    return
}
// 2. محاولة تعيين الوكيل السابق (إذا كان AssignToSameAgent مفعلًا)
// 3. إنشاء التحويل
// 4. saveAndFinalizeTransfer → إنهاء جلسة Chatbot + بث WebSocket
```

---

### 2.5 إعدادات اختيار الوكلاء

**الملف**: `internal/handlers/agent_selection.go`

```go
type AgentSelectionSettings struct {
    TriggerMode       string    // auto, manual, keyword
    Keywords          []string  // كلمات التحويل
    PromptDelay       *int      // تأخير قبل المطالبة (ثواني)
    MinPromptDelay    int       // حد أدنى
    MaxPromptDelay    int       // حد أقصى
    SelectionTimeout  *int      // مهلة الاختيار (ثواني)
    RoundRobinEnabled bool      // تفعيل التوزيع الدائري
}
```

---

## 3. التحديثات الفورية (Real-time Updates)

### 3.1 بنية WebSocket Hub

**الملف**: `internal/websocket/hub.go`

```go
type Hub struct {
    clients    map[uuid.UUID]map[uuid.UUID]map[*Client]struct{}
    //            orgID       → userID    → *Client
    broadcast  chan *BroadcastMessage    // حجم المخزن المؤقت: 256
    register   chan *Client
    unregister chan *Client
}
```

**هيكل ثلاثية المستويات**:
- المستوى 1: `orgID` — المؤسسة
- المستوى 2: `userID` — المستخدم
- المستوى 3: `*Client` — الاتصال الفعلي (يدعم علامات تبويب متعددة)

طرق البث:
- `BroadcastToOrg(orgID, message)`: بث لجميع مستخدمي المؤسسة
- `BroadcastToUser(orgID, userID, message)`: بث لمستخدم محدد
- `BroadcastToContactViewers(orgID, contactID, message)`: بث لمشاهدي جهة اتصال

---

### 3.2 عميل WebSocket

**الملف**: `internal/websocket/client.go`

#### دورة الاتصال:
```
اتصال جديد (WS /ws)
  ↓
مصادقة عبر رسالة (TypeAuth)
  ↓ مهلة: 5 ثواني
┌──────────────┐
│ فشل المصادقة  │ → إغلاق الاتصال
├───── نجاح ───┤
│ Register(Hub) │
│ بدء ReadPump  │
│ بدء WritePump │
└──────────────┘
```

#### ثوابت التوقيت:
```go
writeWait      = 10 * time.Second  // مهلة الكتابة
pongWait       = 60 * time.Second  // مهلة انتظار Pong
pingPeriod     = 54 * time.Second  // فترة Ping (9/10 من pongWait)
maxMessageSize = 4096              // حد حجم الرسالة (بايت)
authTimeout    = 5 * time.Second   // مهلة المصادقة
```

#### ReadPump:
- يقرأ الرسائل من الاتصال
- يعالج رسائل `TypeSetContact` (تعيين جهة الاتصال الحالية)
- يعالج `TypePing` → يرسل `TypePong`
- يتحقق من صلاحية الوصول لجهة الاتصال (`ContactAccessFn`)

#### WritePump:
- يرسل الرسائل من قناة `send`
- يرسل Ping بشكل دوري
- يفرغ الرسائل المتراكمة عند كل كتابة

#### المخزن المؤقت:
```go
send chan []byte  // حجم: 256 رسالة
```

---

### 3.3 أنواع رسائل WebSocket

**الملف**: `internal/websocket/messages.go`

| النوع | الوصف | اتجاه البث |
|------|-------|-----------|
| `TypeNewMessage` | رسالة جديدة (واردة/صادرة) | مشاهدي جهة الاتصال |
| `TypeStatusUpdate` | تحديث حالة رسالة | مشاهدي جهة الاتصال |
| `TypeAgentTransfer` | تحويل وكيل جديد | المؤسسة بأكملها |
| `TypeAgentTransferResume` | استئناف من تحويل | المؤسسة بأكملها |
| `TypeAgentTransferAssign` | تخصيص تحويل | المؤسسة بأكملها |
| `TypeCampaignStatsUpdate` | إحصائيات حملة | المؤسسة بأكملها |
| `TypeAuth` | مصادقة | عميل → خادم |
| `TypeSetContact` | تعيين جهة اتصال | عميل → خادم |
| `TypePing`/`TypePong` | نبضات | ثنائي الاتجاه |

---

### 3.4 Pub/Sub لإحصائيات الحملات

**الملف**: `internal/queue/pubsub.go`

قناة Redis Pub/Sub: `whatomate:campaign_stats`

```go
type CampaignStatsUpdate struct {
    CampaignID     string    `json:"campaign_id"`
    OrganizationID uuid.UUID `json:"organization_id"`
    Status         CampaignStatus
    SentCount      int
    DeliveredCount int
    ReadCount      int
    FailedCount    int
}
```

المسار:
1. `ApplyMessageReceipt` (campaignstats) → تحديث العدادات
2. `Publisher.PublishCampaignStats` → نشر عبر Redis Pub/Sub
3. `Subscriber.SubscribeCampaignStats` → استقبال
4. بث عبر WebSocket Hub (`TypeCampaignStatsUpdate`)

---

## 4. نظام العمال والطوابير (Worker & Queue System)

### 4.1 أنواع الوظائف

**الملف**: `internal/queue/queue.go`

```go
const (
    JobTypeRecipient        = "recipient"          // إرسال حملة
    JobTypeInboundMedia      = "inbound_media"      // معالجة وسائط واردة
    JobTypeContactRepair     = "contact_repair"      // إصلاح بيانات جهة اتصال
    JobTypeWhatsAppFilter    = "whatsapp_filter"     // تصفية أرقام WhatsApp
    JobTypeGroupJoin         = "group_join"           // الانضمام لمجموعة
)

type RecipientJob struct {
    CampaignID     uuid.UUID
    RecipientID    uuid.UUID
    OrganizationID uuid.UUID
    PhoneNumber    string
    RecipientName  string
    TemplateParams JSONB
}
```

### 4.2 هيكل العامل

**الملف**: `internal/worker/worker.go`

```go
type Worker struct {
    CampaignConsumer  // مستهلك طوابير الحملات
    InboundConsumer   // مستهلك طوابير الوسائط الواردة
    SelfHealInterval  // 5 دقائق — معالجة الوسائط المتعثرة
}
```

- **InboundConsumer**: يعالج وسائط الحملات الواردة
- **Self-heal**: يعالج العناصر الأقدم من 15 دقيقة، حد أقصى 250 عنصر

### 4.3 موزع العمال الديناميكي (Worker Scaler)

**الملف**: `internal/worker/scaler.go`

```go
type WorkerScaler struct {
    globalBudget int                    // الحد الأقصى العالمي للعمال
    interval     time.Duration          // 15 ثانية
    runtimes     map[uuid.UUID]*TenantWorkerRuntime
}
```

#### خوارزمية التوسع:

1. **حساب العمق**: `XLEN` لطابور Redis لكل مؤسسة
2. **حساب العدد المطلوب**:
   ```go
   desired = ceil(depth / JobsPerWorker)
   desired = clamp(desired, MinWorkers, MaxWorkers)
   ```
3. **تطبيق حدود الترخيص** (`applyLicensedWorkerCap`)
4. **توزيع الميزانية العالمية** (`allocateWorkerBudget`):
   - ترتيب حسب نسبة التراكم (أعلى أولاً)
   - الحفاظ على العمال الحاليين قبل التوسع
5. **تطبيق تهدئة التصغير** (`applyScaleDownCooldown`)

#### حالة التجميد:
```go
type TenantWorkerRuntime struct {
    Frozen          bool
    FreezeReason    string     // "whatsapp_disconnected", "instance_send_blocked", "worker_start_failures"
    FailureStreak   int        // عتبة التجميد: 3 إخفاقات متتالية
    HealthySince    time.Time  // يجب أن يكون صحياً لدورة كاملة قبل إلغاء التجميد
}
```

---

## 5. مخطط تدفق البيانات الشامل

```
┌──────────────────────────────────────────────────────────────────────┐
│                          العميل (Vue SPA)                            │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────────┐      │
│  │ المحادثات │  │ الحملات  │  │ التحويلات│  │ WebSocket Client │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘      │
└───────┼─────────────┼─────────────┼──────────────────┼───────────────┘
        │ REST API     │ REST API    │ REST API         │ WS
        ▼              ▼             ▼                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│                       خادم Whatomate (fasthttp)                      │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    Middleware Layer                           │   │
│  │  Auth (JWT/API key) → CSRF → CORS → Rate Limit → TenantScope│   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌────────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Message Handler│  │Campaign Handler│  │ Transfer Handler       │ │
│  │ (messages.go)  │  │(campaigns.go) │  │ (agent_transfers.go)   │ │
│  └───────┬────────┘  └──────┬───────┘  └───────────┬────────────┘ │
│          │                  │                      │               │
│          ▼                  ▼                      ▼               │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              MessageProvider Interface                       │   │
│  │     ┌──────────────┐          ┌──────────────────┐          │   │
│  │     │ Meta Cloud   │          │   Whatsmeow      │          │   │
│  │     │ (whatsapp/)  │          │ (whatsmeow/)     │          │   │
│  │     └──────────────┘          └──────────────────┘          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────┐  ┌───────────────────────────────────────┐  │
│  │  WebSocket Hub   │  │         Redis Streams                │  │
│  │  (hub.go)        │  │  ┌──────────────────────────────┐   │  │
│  │  org→user→client │  │  │ whatomate:campaigns:{orgID}  │   │  │
│  └──────┬───────────┘  │  │ whatomate:inbound_media      │   │  │
│         │              │  │ whatomate:campaign_stats (Pub/Sub)│ │
│         ▼              │  └──────────────────────────────┘   │  │
│  ┌──────────────┐     └───────────────────────────────────────┘  │
│  │ Worker Pool  │              │                                   │
│  │ (worker.go)  │◄─────────────┘                                  │
│  │ (scaler.go)  │                                                   │
│  └──────┬───────┘                                                   │
│         │                                                           │
│         ▼                                                           │
│  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │ PostgreSQL   │  │ ChatbotProcessor │  │ SLAProcessor     │   │
│  │ (GORM)       │  │ (chatbot_*.go)   │  │ (sla_*.go)       │   │
│  └──────────────┘  └──────────────────┘  └──────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 6. ملخص التقنيات الرئيسية

| المكون | التكنولوجيا | الملف |
|--------|-------------|-------|
| خادم HTTP | fasthttp + fastglue | `cmd/whatomate/main.go` |
| قاعدة البيانات | PostgreSQL 17 + GORM | `internal/database/` |
| الطوابير | Redis Streams + Pub/Sub | `internal/queue/redis.go`, `internal/queue/pubsub.go` |
| WebSocket | fasthttp/websocket | `internal/websocket/` |
| المصادقة | JWT + API Key | `internal/middleware/` |
| تعدد المستأجرين | TenantScope middleware | `internal/middleware/` |
| التشفير | AES-256 | `internal/crypto/` |
| التكوين | TOML + koanf | `internal/config/` |
| الواجهة | Vue 3 + shadcn-vue | `frontend/` |
| MCP Sidecar | Node.js | `mcp-server/` |
