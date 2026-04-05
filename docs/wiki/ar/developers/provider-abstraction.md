---
title: تجريد المزوّد
rtl: true
lang: ar
---

<div dir="rtl">تجريد المزوّد</div>

يدعم Whatomate مزوّدي WhatsApp من خلال واجهة موحّدة. توثّق هذه الصفحة واجهة `MessageProvider`، ومحوّلات Meta وWhatsMeow، ووسيط حارس المزوّد.

## نظرة عامة

يتيح تجريد المزوّد لـ Whatomate التبديل بين Meta Cloud API وWhatsMeow (WhatsApp Web المباشر) دون تغيير منطق إرسال الرسائل.

```
┌──────────────────────┐
│  SendOutgoingMessage │
│  (معالج موحّد)       │
└──────────┬───────────┘
           │
    ┌──────▼───────┐
    │MessageProvider│
    │   الواجهة    │
    └──────┬───────┘
           │
    ┌──────┴──────┐
    ▼             ▼
┌─────────┐  ┌────────────┐
│  Meta   │  │  WhatsMeow │
│ محوّل   │  │  محوّل     │
└────┬────┘  └─────┬──────┘
     │             │
     ▼             ▼
┌──────────┐  ┌────────────┐
│Meta Cloud│  │بروتوكول    │
│   API    │  │WhatsApp Web│
└──────────┘  └────────────┘
```

## واجهة MessageProvider

الواجهة معرّفة في `pkg/provider/provider.go`:

```go
type MessageProvider interface {
    // SendMessage يرسل رسالة نصية إلى جهة اتصال
    SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error)
    
    // SendMediaMessage يرسل رسالة وسائط (صورة، فيديو، صوت، مستند)
    SendMediaMessage(ctx context.Context, req *MediaMessageRequest) (*SendResult, error)
    
    // SendTemplateMessage يرسل رسالة قالب
    SendTemplateMessage(ctx context.Context, req *TemplateMessageRequest) (*SendResult, error)
    
    // MarkRead يحدد الرسالة كمقروءة
    MarkRead(ctx context.Context, messageID string) error
    
    // SendTyping يرسل مؤشر الكتابة
    SendTyping(ctx context.Context, contactID string, composing bool) error
}
```

## SendOutgoingMessage

جميع عمليات إرسال الرسائل تمر عبر `SendOutgoingMessage()` في المعالجات. هذه الدالة تنسّق:

```go
func (app *App) SendOutgoingMessage(ctx context.Context, req *OutgoingMessageRequest, opts *SendOptions) (*Message, error) {
    // 1. فرض قيود الإرسال
    if err := enforceStrictSendRestrictions(ctx, req, opts); err != nil {
        return nil, err
    }
    
    // 2. تطبيق بادئة اسم الوكيل إذا كانت مُعدّة
    if opts.PrefixAgentName {
        req.Content = fmt.Sprintf("[%s] %s", opts.AgentName, req.Content)
    }
    
    // 3. إنشاء سجل رسالة (الحالة=pending)
    msg := &models.Message{
        ContactID:   req.ContactID,
        Content:     req.Content,
        Direction:   "outbound",
        Status:      "pending",
        AccountID:   req.AccountID,
        InstanceID:  req.InstanceID,
    }
    app.DB.Create(msg)
    
    // 4. تحديد المزوّد
    provider := app.ResolveProvider(req)
    
    // 5. الإرسال بشكل غير متزامن
    go func() {
        result, err := provider.SendMessage(ctx, req)
        if err != nil {
            msg.Status = "failed"
            msg.ErrorMessage = err.Error()
        } else {
            msg.Status = "sent"
            msg.ProviderMessageID = result.MessageID
        }
        app.DB.Save(msg)
        
        // 6. البث عبر WebSocket
        app.BroadcastMessage(msg)
        
        // 7. إرسال الويب هوك
        app.DispatchWebhook("message.sent", msg)
    }()
    
    return msg, nil
}
```

## محوّل Meta

**المصدر:** `pkg/whatsapp/meta_adapter.go`

يوجّه محوّل Meta الاستدعاءات إلى عميل Meta WhatsApp Cloud API:

```go
type MetaAdapter struct {
    Client *whatsapp.Client  // عميل Meta Cloud API HTTP
}

func (a *MetaAdapter) SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error) {
    // بناء حمولة Meta API
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to": req.PhoneNumber,
        "type": "text",
        "text": map[string]interface{}{
            "body": req.Content,
            "preview_url": req.PreviewURL,
        },
    }
    
    // إضافة سياق الرد إذا كان موجوداً
    if req.ReplyToMessageID != "" {
        payload["context"] = map[string]interface{}{
            "message_id": req.ReplyToMessageID,
        }
    }
    
    // الإرسال عبر Meta API
    resp, err := a.Client.SendMessage(ctx, req.PhoneNumberID, payload)
    if err != nil {
        return nil, a.transformMetaError(err)
    }
    
    return &SendResult{
        MessageID: resp.Messages[0].ID,
    }, nil
}
```

### معالجة أخطاء Meta

```go
func (a *MetaAdapter) transformMetaError(err error) error {
    var metaErr *MetaAPIError
    if errors.As(err, &metaErr) {
        switch metaErr.Code {
        case 131051:
            return ErrRateLimited
        case 131026:
            return ErrInvalidCredentials
        case 131047:
            return ErrTemplateNotFound
        default:
            return fmt.Errorf("meta api error: %s", metaErr.Message)
        }
    }
    return err
}
```

## محوّل WhatsMeow

**المصدر:** `pkg/whatsmeow/adapter.go`

يوجّه محوّل WhatsMeow الاستدعاءات عبر مدير الاتصال لكل مثيل:

```go
type WhatsMeowAdapter struct {
    Manager *whatsmeow.Manager  // مدير الاتصال
}

func (a *WhatsMeowAdapter) SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error) {
    // الحصول على العميل المتصل للمثيل
    client, err := a.Manager.GetClient(req.InstanceID)
    if err != nil {
        return nil, fmt.Errorf("instance not connected: %w", err)
    }
    
    // بناء JID
    jid := types.NewJID(req.PhoneNumber, types.DefaultUserServer)
    
    // بناء الرسالة
    msg := &waE2E.Message{
        Conversation: proto.String(req.Content),
    }
    
    // إضافة سياق الرد إذا كان موجوداً
    if req.ReplyToMessageID != "" {
        msg.ContextInfo = &waE2E.ContextInfo{
            StanzaID:      proto.String(req.ReplyToMessageID),
            Participant:   proto.String(req.PhoneNumber + "@s.whatsapp.net"),
            QuotedMessage: &waE2E.Message{},
        }
    }
    
    // الإرسال عبر WhatsMeow (يضع في الطابور إذا تم تحديد المعدل)
    resp, err := client.SendMessage(ctx, jid, msg)
    if err != nil {
        return nil, a.transformWhatsMeowError(err)
    }
    
    return &SendResult{
        MessageID: resp.ID,
    }, nil
}
```

### الطابور لكل مثيل

يستخدم WhatsMeow طابور رسائل لكل مثيل مع تحديد المعدل:

```go
// عمليات الطابور
queue := a.Manager.GetQueue(instanceID)
queue.Enqueue(message)  // إضافة إلى الطابور
queue.Dequeue()         // الحصول على الرسالة التالية
queue.Depth()           // عمق الطابور الحالي
queue.Wait()            // حظر حتى تتوفر السعة
```

## حل المزوّد

يتم حل المزوّد بناءً على تكوين الحساب/المثيل:

```go
func (app *App) ResolveProvider(req *OutgoingMessageRequest) MessageProvider {
    if req.InstanceID != 0 {
        // مثيل WhatsMeow
        return app.WhatsMeowAdapter
    }
    if req.AccountID != 0 {
        // حساب Meta
        return app.MetaAdapter
    }
    // الافتراضي بناءً على التكوين
    if app.Config.WhatsApp.Provider == "whatsmeow" {
        return app.WhatsMeowAdapter
    }
    return app.MetaAdapter
}
```

## وسيط حارس المزوّد

يقيّد وسيط `ProviderGuard` نقاط نهاية معينة لمزوّدات محددة:

```go
// الاستخدام في إعداد المسار
app.GET("/api/templates", app.RequireAuth(
    ProviderGuard("meta", app.ListTemplates),
))
```

**المصدر:** `internal/handlers/provider_guard.go`

```go
func ProviderGuard(requiredProvider string, handler fasthttp.RequestHandler) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        app := getApp(ctx)
        if app.Config.WhatsApp.Provider != requiredProvider {
            ctx.SetStatusCode(fasthttp.StatusBadRequest)
            ctx.SetContentType("application/json")
            ctx.Write([]byte(`{"error":{"message":"Feature not available for current provider","code":"provider_not_supported"}}`))
            return
        }
        handler(ctx)
    }
}
```

### الميزات المحمية

| الميزة | المزوّد المطلوب |
|--------|-----------------|
| القوالب | Meta |
| تدفقات WhatsApp | Meta |
| الكتالوجات | Meta |
| الملف التجاري | Meta |
| تحليلات Meta | Meta |
| الحملات | كلاهما |

## مقارنة المزوّدين

| الميزة | Meta Cloud API | WhatsMeow |
|--------|---------------|-----------|
| إرسال الرسائل | HTTP API | WebSocket |
| القوالب | نعم (يتطلب موافقة) | لا (نص حر) |
| التدفقات | نعم | لا |
| الكتالوجات | نعم | لا |
| مصادقة رمز QR | غير متاح | نعم |
| إقران الهاتف | غير متاح | نعم |
| تحديد المعدل | مفروض من Meta | طابور لكل مثيل |
| تحميل الوسائط | Meta CDN | مباشر |
| دعم المجموعات | نعم | نعم |
| واجهة الملف التجاري | نعم | لا |

## انظر أيضاً

- [البنية المعمارية](./architecture)
- [مرجع واجهة البرمجة (API)](./api-reference) — نقاط نهاية الحسابات والمثيلات
- [أحداث WebSocket](./websocket-events) — أحداث حالة المثيل
