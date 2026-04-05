---
title: Provider Abstraction
---

# Provider Abstraction

Whatomate supports two WhatsApp providers through a unified interface. This page documents the `MessageProvider` interface, the Meta and WhatsMeow adapters, and the provider guard middleware.

## Overview

The provider abstraction allows Whatomate to switch between Meta Cloud API and WhatsMeow (direct WhatsApp Web) without changing the message sending logic.

```
┌──────────────────────┐
│  SendOutgoingMessage │
│  (Unified Handler)   │
└──────────┬───────────┘
           │
    ┌──────▼───────┐
    │MessageProvider│
    │   Interface   │
    └──────┬───────┘
           │
    ┌──────┴──────┐
    ▼             ▼
┌─────────┐  ┌────────────┐
│  Meta   │  │  WhatsMeow │
│ Adapter │  │  Adapter   │
└────┬────┘  └─────┬──────┘
     │             │
     ▼             ▼
┌──────────┐  ┌────────────┐
│Meta Cloud│  │WhatsApp Web│
│   API    │  │  Protocol  │
└──────────┘  └────────────┘
```

## MessageProvider Interface

The interface is defined in `pkg/provider/provider.go`:

```go
type MessageProvider interface {
    // SendMessage sends a text message to a contact
    SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error)
    
    // SendMediaMessage sends a media message (image, video, audio, document)
    SendMediaMessage(ctx context.Context, req *MediaMessageRequest) (*SendResult, error)
    
    // SendTemplateMessage sends a template message
    SendTemplateMessage(ctx context.Context, req *TemplateMessageRequest) (*SendResult, error)
    
    // MarkRead marks a message as read
    MarkRead(ctx context.Context, messageID string) error
    
    // SendTyping sends typing indicator
    SendTyping(ctx context.Context, contactID string, composing bool) error
}
```

## SendOutgoingMessage

All message sending flows through `SendOutgoingMessage()` in the handlers. This function orchestrates:

```go
func (app *App) SendOutgoingMessage(ctx context.Context, req *OutgoingMessageRequest, opts *SendOptions) (*Message, error) {
    // 1. Enforce send restrictions
    if err := enforceStrictSendRestrictions(ctx, req, opts); err != nil {
        return nil, err
    }
    
    // 2. Apply agent name prefix if configured
    if opts.PrefixAgentName {
        req.Content = fmt.Sprintf("[%s] %s", opts.AgentName, req.Content)
    }
    
    // 3. Create message record (status=pending)
    msg := &models.Message{
        ContactID:   req.ContactID,
        Content:     req.Content,
        Direction:   "outbound",
        Status:      "pending",
        AccountID:   req.AccountID,
        InstanceID:  req.InstanceID,
    }
    app.DB.Create(msg)
    
    // 4. Determine provider
    provider := app.ResolveProvider(req)
    
    // 5. Send asynchronously
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
        
        // 6. Broadcast via WebSocket
        app.BroadcastMessage(msg)
        
        // 7. Dispatch webhook
        app.DispatchWebhook("message.sent", msg)
    }()
    
    return msg, nil
}
```

## Meta Adapter

**Source:** `pkg/whatsapp/meta_adapter.go`

The Meta adapter routes calls to the Meta WhatsApp Cloud API client:

```go
type MetaAdapter struct {
    Client *whatsapp.Client  // Meta Cloud API HTTP client
}

func (a *MetaAdapter) SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error) {
    // Build Meta API payload
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to": req.PhoneNumber,
        "type": "text",
        "text": map[string]interface{}{
            "body": req.Content,
            "preview_url": req.PreviewURL,
        },
    }
    
    // Add reply context if present
    if req.ReplyToMessageID != "" {
        payload["context"] = map[string]interface{}{
            "message_id": req.ReplyToMessageID,
        }
    }
    
    // Send via Meta API
    resp, err := a.Client.SendMessage(ctx, req.PhoneNumberID, payload)
    if err != nil {
        return nil, a.transformMetaError(err)
    }
    
    return &SendResult{
        MessageID: resp.Messages[0].ID,
    }, nil
}
```

### Meta-Specific Error Handling

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

## WhatsMeow Adapter

**Source:** `pkg/whatsmeow/adapter.go`

The WhatsMeow adapter routes calls through the per-instance connection manager:

```go
type WhatsMeowAdapter struct {
    Manager *whatsmeow.Manager  // Connection manager
}

func (a *WhatsMeowAdapter) SendMessage(ctx context.Context, req *OutgoingMessageRequest) (*SendResult, error) {
    // Get connected client for instance
    client, err := a.Manager.GetClient(req.InstanceID)
    if err != nil {
        return nil, fmt.Errorf("instance not connected: %w", err)
    }
    
    // Build JID
    jid := types.NewJID(req.PhoneNumber, types.DefaultUserServer)
    
    // Build message
    msg := &waE2E.Message{
        Conversation: proto.String(req.Content),
    }
    
    // Add reply context if present
    if req.ReplyToMessageID != "" {
        msg.ContextInfo = &waE2E.ContextInfo{
            StanzaID:      proto.String(req.ReplyToMessageID),
            Participant:   proto.String(req.PhoneNumber + "@s.whatsapp.net"),
            QuotedMessage: &waE2E.Message{},
        }
    }
    
    // Send via WhatsMeow (enqueues if rate limited)
    resp, err := client.SendMessage(ctx, jid, msg)
    if err != nil {
        return nil, a.transformWhatsMeowError(err)
    }
    
    return &SendResult{
        MessageID: resp.ID,
    }, nil
}
```

### Per-Instance Queuing

WhatsMeow uses a per-instance message queue with rate limiting:

```go
// Queue operations
queue := a.Manager.GetQueue(instanceID)
queue.Enqueue(message)  // Add to queue
queue.Dequeue()         // Get next message
queue.Depth()           // Current queue depth
queue.Wait()            // Block until capacity available
```

## Provider Resolution

The provider is resolved based on the account/instance configuration:

```go
func (app *App) ResolveProvider(req *OutgoingMessageRequest) MessageProvider {
    if req.InstanceID != 0 {
        // WhatsMeow instance
        return app.WhatsMeowAdapter
    }
    if req.AccountID != 0 {
        // Meta account
        return app.MetaAdapter
    }
    // Default based on config
    if app.Config.WhatsApp.Provider == "whatsmeow" {
        return app.WhatsMeowAdapter
    }
    return app.MetaAdapter
}
```

## Provider Guard Middleware

The `ProviderGuard` middleware restricts certain endpoints to specific providers:

```go
// Usage in route setup
app.GET("/api/templates", app.RequireAuth(
    ProviderGuard("meta", app.ListTemplates),
))
```

**Source:** `internal/handlers/provider_guard.go`

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

### Protected Features

| Feature | Required Provider |
|---------|------------------|
| Templates | Meta |
| WhatsApp Flows | Meta |
| Catalogs | Meta |
| Business Profile | Meta |
| Meta Analytics | Meta |
| Campaigns | Both |

## Provider Comparison

| Feature | Meta Cloud API | WhatsMeow |
|---------|---------------|-----------|
| Message Sending | HTTP API | WebSocket |
| Templates | Yes (approval required) | No (free-form) |
| Flows | Yes | No |
| Catalogs | Yes | No |
| QR Code Auth | N/A | Yes |
| Phone Pairing | N/A | Yes |
| Rate Limiting | Meta-enforced | Per-instance queue |
| Media Upload | Meta CDN | Direct |
| Group Support | Yes | Yes |
| Business Profile API | Yes | No |

## See Also

- [Architecture](./architecture)
- [API Reference](./api-reference) — Accounts and Instances endpoints
- [WebSocket Events](./websocket-events) — Instance status events
