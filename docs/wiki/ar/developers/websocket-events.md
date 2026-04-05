---
title: أحداث WebSocket
rtl: true
lang: ar
---

<div dir="rtl">أحداث WebSocket</div>

يتم التعامل مع الاتصال في الوقت الفعلي في Whatomate عبر اتصالات WebSocket. توثّق هذه الصفحة تدفق الاتصال، وعمليات المحور، وجميع أنواع الرسائل.

## تدفق الاتصال

### 1. الحصول على رمز WebSocket

قبل الاتصال، اطلب رمز JWT قصير العمر:

```
GET /api/auth/ws-token
```

**الاستجابة:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 30
}
```

الرمز هو JWT يحتوي على:
- `subject`: "ws"
- `user_id`: معرّف المستخدم المصادق عليه
- `org_id`: معرّف المؤسسة الحالية
- `exp`: 30 ثانية من الإصدار

### 2. إنشاء اتصال WebSocket

```
GET /ws?token=<ws-token>
```

**عملية الاتصال:**
1. يتحقق الخادم من رمز JWT
2. يستخرج `user_id` و `org_id` من المطالبات
3. يرقّي HTTP إلى WebSocket
4. يسجّل الاتصال مع المحور
5. يبدأ حلقات القراءة والكتابة

### 3. دورة حياة الاتصال

```
العميل                    الخادم
  │                         │
  │─── GET /ws?token=... ──▶│
  │◀─── 101 Switching ─────│
  │                         │
  │◀─── {type: "message",   │
  │      payload: {...}} ───│
  │                         │
  │─── {ping} ─────────────▶│
  │◀─── {pong} ────────────│
  │                         │
  │─── Close ──────────────▶│
  │                         │ Hub.Unregister()
```

## عمليات المحور

يدير محور WebSocket (`internal/websocket/hub.go`) الاتصالات:

```go
type Hub struct {
    // orgConnections تربط معرّف المؤسسة بمجموعة اتصالات
    orgConnections map[int64]map[*Connection]bool
    mu             sync.RWMutex
}

// Register يضيف اتصالاً إلى المحور
func (h *Hub) Register(conn *Connection)

// Unregister يزيل اتصالاً
func (h *Hub) Unregister(conn *Connection)

// BroadcastToOrg يرسل رسالة إلى جميع الاتصالات في مؤسسة
func (h *Hub) BroadcastToOrg(orgID int64, msg *WSMessage)
```

### بنية الاتصال

```go
type Connection struct {
    UserID  int64
    OrgID   int64
    Conn    *websocket.Conn
    Send    chan []byte  // قناة مخزنة مؤقتاً للرسائل الصادرة
}
```

## أنواع الرسائل

تتبع جميع رسائل WebSocket هذا التنسيق:

```json
{
  "type": "message_type",
  "payload": { ... },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 1. message

تم استلام أو إرسال رسالة جديدة.

```json
{
  "type": "message",
  "payload": {
    "id": 123,
    "contact_id": 456,
    "content": "Hello!",
    "direction": "inbound",
    "type": "text",
    "status": "delivered",
    "created_at": "2026-01-01T00:00:00Z",
    "sender": {
      "id": 456,
      "name": "John Doe",
      "phone": "+1234567890"
    }
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 2. message_status

تم تحديث حالة رسالة.

```json
{
  "type": "message_status",
  "payload": {
    "message_id": 123,
    "status": "read",
    "updated_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**قيم الحالة:** `pending`، `sent`، `delivered`، `read`، `failed`

### 3. contact_created

تم إنشاء جهة اتصال جديدة.

```json
{
  "type": "contact_created",
  "payload": {
    "id": 789,
    "phone_number": "+1234567890",
    "name": "New Contact",
    "status": "open",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 4. contact_assigned

تم تعيين جهة اتصال لمستخدم.

```json
{
  "type": "contact_assigned",
  "payload": {
    "contact_id": 789,
    "assigned_to": {
      "id": 5,
      "name": "Agent Smith"
    },
    "assigned_by": {
      "id": 1,
      "name": "Admin User"
    }
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 5. chat_closed

تم إغلاق محادثة.

```json
{
  "type": "chat_closed",
  "payload": {
    "contact_id": 789,
    "closed_by": {
      "id": 5,
      "name": "Agent Smith"
    },
    "closed_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 6. chat_reopened

تم إعادة فتح محادثة مغلقة.

```json
{
  "type": "chat_reopened",
  "payload": {
    "contact_id": 789,
    "reopened_by": {
      "id": 5,
      "name": "Agent Smith"
    },
    "reopened_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 7. campaign_stats_update

تم تحديث إحصائيات تقدم الحملة.

```json
{
  "type": "campaign_stats_update",
  "payload": {
    "campaign_id": 10,
    "total_recipients": 500,
    "sent": 150,
    "delivered": 120,
    "read": 80,
    "failed": 5,
    "pending": 345,
    "status": "running"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 8. instance_status

تغير حالة اتصال مثيل WhatsApp.

```json
{
  "type": "instance_status",
  "payload": {
    "instance_id": 3,
    "name": "support-instance",
    "status": "connected",
    "phone_number": "+1234567890",
    "queue_depth": 5
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**قيم الحالة:** `disconnected`، `connecting`، `qr_ready`، `connected`، `reconnecting`، `reconnect_failed`

### 9. notification

إشعار جديد للمستخدم.

```json
{
  "type": "notification",
  "payload": {
    "id": 42,
    "type": "sla_breach",
    "message": "Chat #789 has exceeded response SLA",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 10. typing

مؤشر الكتابة لجهة اتصال.

```json
{
  "type": "typing",
  "payload": {
    "contact_id": 789,
    "user_id": 5,
    "state": "composing"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**قيم الحالة:** `composing`، `paused`

### 11. presence

تحديث حضور جهة الاتصال.

```json
{
  "type": "presence",
  "payload": {
    "contact_id": 789,
    "status": "online",
    "last_seen": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

### 12. instance_reconnect_failed

فشل المثيل في إعادة الاتصال بعد محاولات متعددة.

```json
{
  "type": "instance_reconnect_failed",
  "payload": {
    "instance_id": 3,
    "name": "support-instance",
    "error": "connection timeout after 5 retries",
    "last_attempt": "2026-01-01T00:00:00Z"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

## الاستخدام من جانب العميل

### مثال JavaScript

```javascript
// 1. الحصول على رمز WebSocket
const tokenResp = await fetch('/api/auth/ws-token');
const { token } = await tokenResp.json();

// 2. الاتصال
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

// 3. معالجة الرسائل
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  switch (msg.type) {
    case 'message':
      appendMessage(msg.payload);
      break;
    case 'message_status':
      updateMessageStatus(msg.payload);
      break;
    case 'contact_assigned':
      showAssignmentNotification(msg.payload);
      break;
    case 'campaign_stats_update':
      updateCampaignProgress(msg.payload);
      break;
    case 'instance_status':
      updateInstanceIndicator(msg.payload);
      break;
    case 'typing':
      showTypingIndicator(msg.payload);
      break;
    case 'notification':
      showNotification(msg.payload);
      break;
  }
};

// 4. معالجة إعادة الاتصال
ws.onclose = () => {
  setTimeout(connect, 3000);
};
```

## البث

يتم بث الرسائل عبر المحور:

```go
// بث إلى المؤسسة بأكملها
app.Hub.BroadcastToOrg(orgID, &WSMessage{
    Type: "message",
    Payload: messageData,
    Timestamp: time.Now().UTC(),
})

// البث من Redis pub/sub (إحصائيات الحملة)
// يتم التعامل معه بواسطة CampaignStatsSubscriber
```

## انظر أيضاً

- [البنية المعمارية](./architecture)
- [مرجع واجهة البرمجة (API)](./api-reference) — نقطة نهاية رمز WebSocket
- [العمليات الخلفية](./background-workers) — مشترك إحصائيات الحملة
