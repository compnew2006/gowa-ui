# PRD: GOWA Provider Integration into Whatomate

**Version:** 1.0  
**Date:** 2026-07-11  
**Status:** Draft  
**Author:** Integration Architecture Review  
**Dependencies:** whatomate (Go, fasthttp, GORM, PostgreSQL, Redis, React SPA), GOWA v8.10.0 (Go, whatsmeow, SQLite)

---

## 1. Executive Summary

Whatomate is a mature multi-tenant WhatsApp Business engagement platform built exclusively on Meta Cloud API. This PRD defines the integration of **GOWA (Go WhatsApp Web Multi-Device)** as a second provider, running side by side with Meta Cloud API, giving organizations the choice between official (Meta) and unofficial (GOWA) WhatsApp connectivity within the same platform.

The two providers are **complementary, not competing**: Meta excels at template-based outbound messaging, commerce, analytics, and enterprise compliance. GOWA excels at free-form messaging, rich media, group management, chat history, and zero-cost operation. Together they create a single platform that serves both enterprise customers (Meta) and SMBs in emerging markets (GOWA).

This integration requires three architectural phases: extracting a provider interface from the monolithic Meta client, implementing a GOWA client behind that interface, and extending the data model to support multi-provider account management.

---

## 2. Current Architecture Analysis

### 2.1 Whatomate Tech Stack

| Layer | Technology |
|-------|-----------|
| HTTP | `valyala/fasthttp` + `zerodha/fastglue` |
| Database | PostgreSQL via `gorm` |
| Cache/Queue | Redis |
| WhatsApp | Meta Cloud API via `graph.facebook.com` |
| Storage | Local filesystem or AWS S3 |
| Auth | JWT + Basic Auth + SSO (Clerk) |
| Real-time | WebSocket hub |
| Frontend | Embedded React SPA (built into Go binary) |
| Deployment | Docker multi-stage build |

### 2.2 Provider Architecture (Current State)

**Critical finding: there is NO provider abstraction.** The system is built around a single concrete type:

```go
// pkg/whatsapp/client.go
type Client struct {
    HTTPClient *http.Client
    Log        logf.Logger
    baseURL    string // https://graph.facebook.com
}
```

This client is created once at startup and stored directly on the App struct:

```go
// internal/handlers/app.go
type App struct {
    WhatsApp *whatsapp.Client  // concrete type, no interface
    // ...
}
```

**Account credentials** are stored in the `WhatsAppAccount` database model with Meta-specific fields:

```go
// internal/models/models.go (line ~329)
func (a *WhatsAppAccount) ToWAAccount() *whatsapp.Account {
    return &whatsapp.Account{
        PhoneID:     a.PhoneID,
        BusinessID:  a.BusinessID,
        AppID:       a.AppID,
        APIVersion:  a.APIVersion,
        AccessToken: a.AccessToken,
    }
}
```

**Account resolution** is database-driven: incoming webhooks match by `phone_number_id`, outgoing messages select by account name or default-outgoing flag.

### 2.3 Client Method Surface (45+ methods)

| File | Methods |
|------|---------|
| `message.go` | `SendTextMessage`, `SendInteractiveButtons`, `SendCTAURLButton`, `SendVoiceCallButton`, `SendFlowMessage`, `SendTemplateMessage` + template component builders |
| `client.go` | `SendImageMessage`, `SendDocumentMessage`, `SendVideoMessage`, `SendAudioMessage`, `MarkMessageRead`, `GetMediaURL`, `DownloadMedia`, `UploadMedia`, `ResumableUpload`, `ValidateCredentials`, `SubscribeApp`, `ExchangeCodeForToken`, `GetPhoneNumberInfo`, `RegisterPhoneNumber` |
| `template.go` | `SubmitTemplate`, `FetchTemplates`, `DeleteTemplate` |
| `catalog.go` | `CreateCatalog`, `ListCatalogs`, `DeleteCatalog`, `ListCatalogProducts`, `CreateProduct`, `UpdateProduct`, `DeleteProduct` |
| `flow.go` | `CreateFlow`, `GetFlow`, `ListFlows`, `DeleteFlow`, `UpdateFlowJSON`, `PublishFlow`, `DeprecateFlow`, `GetFlowAssets` |
| `call.go` | `PreAcceptCall`, `AcceptCall`, `RejectCall`, `TerminateCall`, `SendCallPermissionRequest`, `GetCallPermission`, `InitiateCall` |
| `analytics.go` | `GetAnalytics` |
| `profile_extras.go` | `GetBusinessProfile`, `UpdateBusinessProfile`, `UploadProfilePicture`, `GetTokenDebugInfo`, `GetSharedWABA`, `GetWABAPhoneNumbers` |

### 2.4 Incoming Message Flow

```
POST /api/webhook (Meta payload)
  → WebhookHandler (webhook.go)
    → Verify HMAC-SHA256
    → Route by field type (messages, statuses, calls, etc.)
    → processIncomingMessage(phoneNumberID, msg, profileName)
      → Deduplication check (by WhatsApp message ID)
      → processIncomingMessageFull (chatbot_processor.go:137)
        → getWhatsAppAccountCached(phoneID) // Redis-cached DB lookup
        → Handle reactions (special case)
        → GetOrCreateContact(orgID, phone, profileName)
        → extractMessageContent(msg, account) // download media if needed
        → Save Message to DB
        → Run chatbot keyword/flow processing
        → Broadcast via WebSocket
        → Dispatch webhook events
```

### 2.5 Outgoing Message Flow

```
SendOutgoingMessage(ctx, req, opts) (messages.go:145)
  → createOutgoingMessage(req, opts) → models.Message (pending)
  → Save to DB
  → sendFn closure:
      → toWhatsAppAccount() converts DB model → whatsapp.Account
      → Switch on message type:
        text        → WhatsApp.SendTextMessage()
        image       → UploadMedia + SendImageMessage
        video       → UploadMedia + SendVideoMessage
        audio       → UploadMedia + SendAudioMessage
        document    → UploadMedia + SendDocumentMessage
        interactive → SendInteractiveButtons / SendCTAURLButton
        template    → BuildTemplateComponents + SendTemplateMessage
        flow        → SendFlowMessage
  → Execute sendFn (async or sync)
  → finalizeMessageSend(msg, wamid, err)
    → Update DB status
    → Broadcast via WebSocket
    → Dispatch webhook events
```

---

## 3. GOWA Architecture Analysis

### 3.1 What GOWA Is

GOWA (Go WhatsApp Web Multi-Device) is a self-hosted WhatsApp gateway built in Go using the `whatsmeow` library. It acts as an unofficial WhatsApp Web client, providing REST API, MCP server, and web dashboard access to WhatsApp accounts. It is NOT affiliated with WhatsApp/Meta.

**Key characteristics:**
- Multi-device: multiple WhatsApp accounts per server instance
- Free and self-hosted: no per-conversation pricing
- Unofficial: uses whatsmeow protocol, account ban risk exists
- Comprehensive: 64 REST endpoints, 17 webhook event types, MCP support

### 3.2 GOWA API Surface (64 endpoints across 10 categories)

| Category | Endpoints | Key Operations |
|----------|-----------|---------------|
| **App** (10) | `/health`, `/app/login`, `/app/login-with-code`, `/app/passkey*`, `/app/logout`, `/app/reconnect`, `/app/devices`, `/app/status` | QR/code/passkey login, connection lifecycle |
| **Device** (11) | `/devices` CRUD, `/devices/{id}/logout`, `/reconnect`, `/status`, `/webhook` | Multi-device management, per-device webhook config |
| **User** (10) | `/user/info`, `/user/avatar`, `/user/pushname`, `/user/my/privacy`, `/user/my/groups`, `/user/my/contacts`, `/user/check`, `/user/business-profile` | Profile, contacts, privacy, number validation |
| **Send** (12) | `/send/message`, `/send/image`, `/send/video`, `/send/audio`, `/send/file`, `/send/sticker`, `/send/contact`, `/send/link`, `/send/location`, `/send/poll`, `/send/presence`, `/send/chat-presence` | All message types (multipart or JSON) |
| **Message** (9) | `/message/{id}/revoke`, `/delete`, `/reaction`, `/update`, `/read`, `/star`, `/unstar`, `/forward`, `/download` | Message actions |
| **Call** (1) | `/call/reject` | Call rejection |
| **Chat** (5) | `/chats`, `/chat/{jid}/messages`, `/pin`, `/disappearing`, `/archive` | Chat management with pagination |
| **Group** (20) | `/group` CRUD, `/group/participants*`, `/group/join-with-link`, `/group/leave`, `/group/photo`, `/group/name`, `/group/topic`, `/group/locked`, `/group/announce`, `/group/invite-link`, `/group/participant-requests*`, `/group/info-from-link`, `/group/participants/export` | Full group lifecycle |
| **Newsletter** (2) | `/newsletter/unfollow`, `/newsletter/messages` | Channel management |
| **Chatwoot** (3) | `/chatwoot/webhook`, `/chatwoot/sync`, `/chatwoot/sync/status` | Chatwoot bridge |

### 3.3 GOWA Authentication Model

| Mechanism | Details |
|-----------|---------|
| API Auth | HTTP Basic Auth (multi-credential: `user1:pass1,user2:pass2`) |
| Device Scoping | `X-Device-Id` header or `?device_id=` query parameter |
| WhatsApp Pairing | QR code, phone pairing code, WebAuthn passkey |
| Webhook Security | HMAC-SHA256 via `X-Hub-Signature-256` header |

### 3.4 GOWA Webhook System (17 event types)

**Envelope:**
```json
{
  "event": "<event_type>",
  "device_id": "<jid>@s.whatsapp.net",
  "session_id": "org_2",
  "payload": { ... }
}
```

**Event types:**

| Event | Category | Key Payload Fields |
|-------|----------|-------------------|
| `message` | Core | `id`, `chat_id`, `from`, `from_name`, `timestamp`, `body`, `image`, `video`, `audio`, `document`, `sticker`, `video_note`, `contact`, `contacts_array`, `location`, `live_location`, `replied_to_id`, `quoted_body`, `view_once`, `forwarded`, `referral` |
| `message.reaction` | Action | `reaction`, `reacted_message_id` |
| `message.revoked` | Action | `revoked_message_id`, `revoked_from_me` |
| `message.edited` | Action | `original_message_id`, `body` (new text) |
| `message.ack` | Receipt | `ids[]`, `receipt_type` (delivered/read), `receipt_type_description` |
| `message.deleted` | Action | `deleted_message_id`, `original_content`, `original_media_type` |
| `chat_presence` | Presence | `state` (composing/paused), `media` (""/audio), `is_group` |
| `group.participants` | Group | `type` (join/leave/promote/demote), `jids[]` |
| `group.joined` | Group | Device added to group |
| `call.offer` | Call | `call_id`, `from`, `auto_rejected`, `remote_platform`, `group_jid` |
| `label.edit` | Label | `label_id`, `name`, `color`, `deleted`, `type` |
| `label.association` | Label | `label_id`, `labeled` (true/false), `chat_id` |
| `newsletter.joined` | Newsletter | `newsletter_id`, `name`, `description` |
| `newsletter.left` | Newsletter | `newsletter_id`, `role` |
| `newsletter.message` | Newsletter | `newsletter_id`, `messages[]` (array of message objects) |
| `newsletter.mute` | Newsletter | `newsletter_id`, `mute` (on/off) |

**Critical implementation notes:**
- Timestamp placement is inconsistent: `message` events have `timestamp` inside `payload`; `message.ack`, `chat_presence`, `label.*`, `group.participants`, `call.offer`, and newsletter events have `timestamp` at the top level (sibling of `payload`)
- Image field is polymorphic: plain string when auto-download ON without caption; object when caption present or auto-download OFF
- HMAC must be computed over **raw body bytes**, not re-serialized JSON
- Retry: 5 attempts, exponential backoff (1s, 2s, 4s, 8s, 16s)
- `newsletter.message` is the only event carrying an array of messages

### 3.5 GOWA Per-Device Webhook Routing

**Status: IMPLEMENTED** in GOWA v8.10.0.

Two-tier routing:
1. **Per-device (primary):** Each device can have its own `webhook_url`, `webhook_secret`, `webhook_events` (whitelist), `webhook_insecure_skip_verify`
2. **Global (fallback):** If device has no custom webhook, falls back to global `WHATSAPP_WEBHOOK` config

This is mutually exclusive per event — no fan-out to both device-specific and global.

### 3.6 GOWA Media Handling

| Media Type | Upload Mode | Auto-Download ON (no caption) | Auto-Download ON (with caption) | Auto-Download OFF |
|-----------|-------------|-------------------------------|-------------------------------|-------------------|
| Image | multipart or URL | `"image": "path/file.jpeg"` (string) | `"image": {"path": "...", "caption": "..."}` | `"image": {"url": "...", "caption": "..."}` |
| Video | multipart or URL | `"video": {"path": "...", "caption": ""}` | `"video": {"path": "...", "caption": "..."}` | `"video": {"url": "...", "caption": "..."}` |
| Audio | multipart or URL | `"audio": "path/file.ogg"` (string) | N/A | Inferred: `{"url": "..."}` |
| Document | multipart or URL | `"document": {"path": "...", "caption": ""}` | `"document": {"path": "...", "caption": "..."}` | `"document": {"url": "...", "filename": "..."}` |
| Sticker | multipart or URL | `"sticker": "path/file.webp"` (string) | N/A | Inferred: `{"url": "..."}` |

Transcoding: FFmpeg for video/audio, libwebp for stickers. Audio auto-transcodes to OGG/Opus for PTT mode.

---

## 4. Feature Comparison Matrix

### 4.1 Whatomate Meta Client vs GOWA API

| Capability | Meta (whatomate) | GOWA | Integration Strategy |
|-----------|-----------------|------|---------------------|
| **Send Text** | ✅ `SendTextMessage` | ✅ `/send/message` | Map to interface method |
| **Send Image** | ✅ (upload + send) | ✅ (direct multipart/URL) | Different upload flow — GOWA uploads inline |
| **Send Video** | ✅ (upload + send) | ✅ (direct multipart/URL) | Different upload flow |
| **Send Audio** | ✅ (upload + send) | ✅ (direct + PTT flag) | GOWA has PTT/voice note mode |
| **Send Document** | ✅ (upload + send) | ✅ (direct multipart/URL) | Different upload flow |
| **Send Sticker** | ❌ | ✅ (auto WebP conversion) | **GOWA-only feature** — gate behind provider check |
| **Send Contact** | ❌ | ✅ (vCard) | **GOWA-only feature** |
| **Send Location** | ❌ | ✅ | **GOWA-only feature** |
| **Send Poll** | ❌ | ✅ | **GOWA-only feature** |
| **Send Link Preview** | ❌ | ✅ | **GOWA-only feature** |
| **Send Typing Indicator** | ❌ | ✅ `/send/chat-presence` | **GOWA-only feature** |
| **Send Presence** | ❌ | ✅ `/send/presence` | **GOWA-only feature** |
| **Send Template (HSM)** | ✅ | ❌ | **Meta-only** — disable for GOWA accounts |
| **Send Interactive Buttons** | ✅ | ❌ | **Meta-only** |
| **Send CTA URL Button** | ✅ | ❌ | **Meta-only** |
| **Send Flow Message** | ✅ | ❌ | **Meta-only** |
| **Send Voice Call Button** | ✅ | ❌ | **Meta-only** |
| **Template Management** | ✅ (CRUD) | ❌ | **Meta-only** |
| **Catalog/Commerce** | ✅ (CRUD) | ❌ | **Meta-only** |
| **Analytics** | ✅ | ❌ | **Meta-only** |
| **Mark as Read** | ✅ | ✅ `/message/{id}/read` | Map to interface |
| **Upload Media** | ✅ (ResumableUpload) | ✅ (inline with send) | Different mechanism |
| **Download Media** | ✅ (via media ID) | ✅ `/message/{id}/download` | Different mechanism |
| **React to Message** | ❌ | ✅ | **GOWA-only feature** |
| **Edit Message** | ❌ | ✅ | **GOWA-only feature** |
| **Revoke/Delete Message** | ❌ | ✅ | **GOWA-only feature** |
| **Star/Unstar** | ❌ | ✅ | **GOWA-only feature** |
| **Forward Message** | ❌ | ✅ | **GOWA-only feature** |
| **QR/Code/Passkey Login** | ❌ (OAuth) | ✅ | **GOWA-only** — device management in UI |
| **Device Management** | ❌ | ✅ (full CRUD) | **GOWA-only** — device status in UI |
| **Chat History** | ❌ | ✅ (paginated) | **GOWA-only** — optional chat browser |
| **Group Management** | ❌ | ✅ (full lifecycle) | **GOWA-only** — group manager module |
| **Newsletter/Channel** | ❌ | ✅ | **GOWA-only** |
| **Call Rejection** | ❌ | ✅ | **GOWA-only** |
| **User Profile** | ✅ (read+update) | ✅ (read+partial update) | Partial overlap |
| **Webhook Verification** | ✅ GET challenge + HMAC | ✅ HMAC only | Different verify flow |
| **Webhook Payload Format** | Meta nested format | GOWA flat format | Separate parsers needed |
| **Phone Number Validation** | ❌ | ✅ `/user/check` | **GOWA-only** |

### 4.2 Capability Classification

| Category | Description | Examples |
|----------|-------------|---------|
| **Shared (both)** | Core messaging both support | Send text, image, video, audio, document, mark read |
| **Meta-only (enterprise)** | Official business features | Templates, interactive buttons, flows, catalog, analytics |
| **GOWA-only (consumer)** | Unofficial WhatsApp client features | Stickers, contacts, location, polls, groups, chat history, reactions, typing indicators |
| **GOWA-only (operations)** | GOWA-specific infrastructure | Device management, QR login, connection lifecycle, per-device webhooks |

---

## 5. Integration Architecture

### 5.1 Design Principles

1. **Interface-driven:** Extract a `WhatsAppProvider` interface; both Meta and GOWA clients implement it
2. **Provider-agnostic business logic:** `processIncomingMessageFull` and `SendOutgoingMessage` must NEVER contain `if provider == "gowa"` logic
3. **Boundary normalization:** All provider-specific differences are absorbed at the provider boundary (client implementation + webhook handler), not in the core
4. **Feature gating:** Capabilities that only one provider supports are gated at the UI/API layer based on the account's `provider_type`
5. **Data model extension:** Extend `WhatsAppAccount` with a `provider_type` discriminator and provider-specific credential fields
6. **Zero regression:** Every existing Meta Cloud API feature must continue working unchanged

### 5.2 Provider Interface

```go
// pkg/whatsapp/provider.go

type WhatsAppProvider interface {
    // Core Messaging
    SendTextMessage(ctx context.Context, account Account, phone string, 
        text string, replyTo *ReplyContext) (string, error)
    SendImageMessage(ctx context.Context, account Account, phone string, 
        mediaID string, caption string, replyTo *ReplyContext) (string, error)
    SendVideoMessage(ctx context.Context, account Account, phone string, 
        mediaID string, caption string, replyTo *ReplyContext) (string, error)
    SendAudioMessage(ctx context.Context, account Account, phone string, 
        mediaID string, replyTo *ReplyContext) (string, error)
    SendDocumentMessage(ctx context.Context, account Account, phone string, 
        mediaID string, caption string, replyTo *ReplyContext) (string, error)
    
    // Interactive (Meta-only; GOWA returns ErrNotSupported)
    SendInteractiveButtons(ctx context.Context, account Account, phone string, 
        bodyText string, buttons []Button) (string, error)
    SendCTAURLButton(ctx context.Context, account Account, phone string, 
        bodyText, buttonText, url string) (string, error)
    SendFlowMessage(ctx context.Context, account Account, phone string, 
        flowID, headerText, bodyText, ctaText, flowToken, firstScreen string) (string, error)
    
    // Templates (Meta-only; GOWA returns ErrNotSupported)
    SendTemplateMessage(ctx context.Context, account Account, phone string, 
        templateName, languageCode string, components []TemplateComponent) (string, error)
    
    // Media Management
    UploadMedia(ctx context.Context, account Account, reader io.Reader, 
        mediaType string) (string, error)
    DownloadMedia(ctx context.Context, account Account, mediaID string) ([]byte, string, error)
    
    // Read Receipts
    MarkMessageRead(ctx context.Context, account Account, messageID string) error
    
    // Provider Info
    ProviderType() string
    SupportedFeatures() ProviderFeatures
}

type ProviderFeatures struct {
    Templates      bool
    Interactive    bool
    Flows          bool
    Stickers       bool
    Contacts       bool
    Location       bool
    Polls          bool
    Typing         bool
    Groups         bool
    Reactions      bool
    MessageEdit    bool
    MessageRevoke  bool
    ChatHistory    bool
    Newsletters    bool
    Catalog        bool
    Analytics      bool
    Calling        bool
}
```

### 5.3 Account Data Model Extension

```sql
ALTER TABLE whatsapp_accounts ADD COLUMN provider_type VARCHAR(20) 
    DEFAULT 'meta_cloud_api' 
    CHECK (provider_type IN ('meta_cloud_api', 'gowa'));

-- GOWA-specific credential fields (nullable — only used when provider_type = 'gowa')
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_server_url TEXT;
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_username TEXT;
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_password TEXT;
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_device_id TEXT;
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_webhook_secret TEXT DEFAULT 'secret';
```

The Go model:

```go
type WhatsAppAccount struct {
    // ... existing fields ...
    
    // Provider discriminator
    ProviderType string `gorm:"type:varchar(20);default:'meta_cloud_api'"`
    
    // Meta credentials (used when provider_type = 'meta_cloud_api')
    PhoneID     string
    BusinessID  string
    AppID       string
    APIVersion  string
    AccessToken string
    
    // GOWA credentials (used when provider_type = 'gowa')
    GowaServerURL    string
    GowaUsername     string
    GowaPassword     string
    GowaDeviceID     string
    GowaWebhookSecret string
}
```

### 5.4 Provider Registry

```go
// internal/handlers/provider_registry.go

type ProviderRegistry struct {
    providers map[string]map[string]WhatsAppProvider // orgID -> accountName -> provider
    mu        sync.RWMutex
}

func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        providers: make(map[string]map[string]WhatsAppProvider),
    }
}

func (r *ProviderRegistry) Register(orgID, accountName string, p WhatsAppProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if r.providers[orgID] == nil {
        r.providers[orgID] = make(map[string]WhatsAppProvider)
    }
    r.providers[orgID][accountName] = p
}

func (r *ProviderRegistry) Get(orgID, accountName string) (WhatsAppProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    org, ok := r.providers[orgID]
    if !ok {
        return nil, fmt.Errorf("no providers for organization %s", orgID)
    }
    p, ok := org[accountName]
    if !ok {
        return nil, fmt.Errorf("provider %s not found for organization %s", accountName, orgID)
    }
    return p, nil
}
```

### 5.5 GOWA Client Implementation

```go
// pkg/gowa/client.go

type Client struct {
    httpClient *http.Client
    serverURL  string
    username   string
    password   string
    deviceID   string
    log        logf.Logger
}

func New(serverURL, username, password, deviceID string, log logf.Logger) *Client {
    return &Client{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        serverURL:  strings.TrimSuffix(serverURL, "/"),
        username:   username,
        password:   password,
        deviceID:   deviceID,
        log:        log,
    }
}

func (c *Client) ProviderType() string { return "gowa" }

func (c *Client) SupportedFeatures() whatsapp.ProviderFeatures {
    return whatsapp.ProviderFeatures{
        Templates:     false, // GOWA cannot send HSM templates
        Interactive:   false, // GOWA has no interactive button messages
        Flows:         false,
        Stickers:      true,
        Contacts:      true,
        Location:      true,
        Polls:         true,
        Typing:        true,
        Groups:        true,
        Reactions:     true,
        MessageEdit:   true,
        MessageRevoke: true,
        ChatHistory:   true,
        Newsletters:   true,
        Catalog:       false,
        Analytics:     false,
        Calling:       false,
    }
}

func (c *Client) doRequest(ctx context.Context, method, path string, 
    body io.Reader, contentType string) ([]byte, error) {
    url := c.serverURL + path
    req, err := http.NewRequestWithContext(ctx, method, url, body)
    if err != nil {
        return nil, err
    }
    req.SetBasicAuth(c.username, c.password)
    if c.deviceID != "" {
        req.Header.Set("X-Device-Id", c.deviceID)
    }
    if contentType != "" {
        req.Header.Set("Content-Type", contentType)
    }
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return nil, parseGowaError(resp.StatusCode, respBody)
    }
    return respBody, nil
}

func (c *Client) SendTextMessage(ctx context.Context, account whatsapp.Account,
    phone, text string, replyTo *whatsapp.ReplyContext) (string, error) {
    
    payload := map[string]interface{}{
        "phone":   phone,
        "message": text,
    }
    if replyTo != nil {
        payload["reply_message_id"] = replyTo.MessageID
    }
    
    resp, err := c.doJSON(ctx, "POST", "/send/message", payload)
    if err != nil {
        return "", fmt.Errorf("failed to send text message via GOWA: %w", err)
    }
    
    var result struct {
        Code    string `json:"code"`
        Results struct {
            MessageID string `json:"message_id"`
        } `json:"results"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return "", err
    }
    return result.Results.MessageID, nil
}

// Image sending — GOWA uses direct multipart upload, no pre-upload step
func (c *Client) SendImageMessage(ctx context.Context, account whatsapp.Account,
    phone, mediaPath string, caption string, replyTo *whatsapp.ReplyContext) (string, error) {
    
    // Use multipart form: no pre-upload to get media ID like Meta
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    
    writer.WriteField("phone", phone)
    if caption != "" {
        writer.WriteField("caption", caption)
    }
    if replyTo != nil {
        writer.WriteField("reply_message_id", replyTo.MessageID)
    }
    
    part, _ := writer.CreateFormFile("image", filepath.Base(mediaPath))
    f, _ := os.Open(mediaPath)
    io.Copy(part, f)
    f.Close()
    writer.Close()
    
    resp, err := c.doRequest(ctx, "POST", "/send/image", &body, writer.FormDataContentType())
    if err != nil {
        return "", fmt.Errorf("failed to send image via GOWA: %w", err)
    }
    
    var result struct {
        Results struct {
            MessageID string `json:"message_id"`
        } `json:"results"`
    }
    json.Unmarshal(resp, &result)
    return result.Results.MessageID, nil
}

// Meta-only methods return ErrNotSupported
func (c *Client) SendInteractiveButtons(...) (string, error) {
    return "", whatsapp.ErrNotSupported
}
func (c *Client) SendTemplateMessage(...) (string, error) {
    return "", whatsapp.ErrNotSupported
}
func (c *Client) SendFlowMessage(...) (string, error) {
    return "", whatsapp.ErrNotSupported
}
```

### 5.6 GOWA Webhook Handler

```go
// internal/handlers/gowa_webhook.go

// GOWA webhook payload envelope
type GowaWebhookEnvelope struct {
    Event     string          `json:"event"`
    DeviceID  string          `json:"device_id"`
    SessionID string          `json:"session_id,omitempty"`
    Payload   json.RawMessage `json:"payload"`
    // Some events have timestamp at top level
    Timestamp string          `json:"timestamp,omitempty"`
}

// GOWA message payload
type GowaMessagePayload struct {
    ID        string `json:"id"`
    ChatID    string `json:"chat_id"`
    From      string `json:"from"`
    FromLID   string `json:"from_lid,omitempty"`
    FromName  string `json:"from_name"`
    Timestamp string `json:"timestamp"`
    IsFromMe  bool   `json:"is_from_me"`
    Body      string `json:"body,omitempty"`
    
    // Media fields (polymorphic — see §3.6)
    Image       interface{} `json:"image,omitempty"`
    Video       *GowaMediaObject `json:"video,omitempty"`
    Audio       interface{} `json:"audio,omitempty"`
    Document    *GowaDocObject `json:"document,omitempty"`
    Sticker     interface{} `json:"sticker,omitempty"`
    VideoNote   interface{} `json:"video_note,omitempty"`
    Contact     *GowaContactObject `json:"contact,omitempty"`
    ContactsArray []GowaContactObject `json:"contacts_array,omitempty"`
    Location    *GowaLocation `json:"location,omitempty"`
    LiveLocation *GowaLocation `json:"live_location,omitempty"`
    
    // Context
    RepliedToID string `json:"replied_to_id,omitempty"`
    QuotedBody  string `json:"quoted_body,omitempty"`
    ViewOnce    bool   `json:"view_once,omitempty"`
    Forwarded   bool   `json:"forwarded,omitempty"`
    Referral   *GowaReferral `json:"referral,omitempty"`
}

// Handler registration in main.go
g.POST("/api/webhook/gowa", app.GowaWebhookHandler)
g.GET("/api/webhook/gowa", app.GowaWebhookVerify) // optional: if GOWA adds verify

func (a *App) GowaWebhookHandler(ctx *fastglue.Request) error {
    // 1. Read raw body for HMAC verification
    rawBody := ctx.Request.Body()
    
    // 2. Parse envelope
    var envelope GowaWebhookEnvelope
    json.Unmarshal(rawBody, &envelope)
    
    // 3. Look up GOWA account by device_id
    account, err := a.resolveGowaAccount(envelope.DeviceID)
    if err != nil {
        a.Log.Error("gowa account not found", "device_id", envelope.DeviceID)
        return ctx.SendStatus(200) // acknowledge to prevent retry
    }
    
    // 4. Verify HMAC-SHA256
    signature := ctx.Request.Header.Get("X-Hub-Signature-256")
    if !verifyGowaSignature(rawBody, signature, account.GowaWebhookSecret) {
        return ctx.SendStatus(401)
    }
    
    // 5. Route by event type
    switch envelope.Event {
    case "message":
        go a.processGowaMessage(account, envelope.Payload)
    case "message.ack":
        go a.processGowaAck(account, envelope)
    case "message.reaction":
        go a.processGowaReaction(account, envelope.Payload)
    case "message.edited":
        go a.processGowaEdit(account, envelope.Payload)
    case "message.revoked":
        go a.processGowaRevoke(account, envelope.Payload)
    case "message.deleted":
        go a.processGowaDelete(account, envelope.Payload)
    case "chat_presence":
        go a.processGowaPresence(account, envelope.Payload)
    case "call.offer":
        go a.processGowaCall(account, envelope.Payload)
    case "group.participants":
        go a.processGowaGroupEvent(account, envelope.Payload)
    // ... other events
    }
    
    return ctx.SendStatus(200)
}

// Normalize GOWA message → whatomate internal Message model
func (a *App) processGowaMessage(account *models.WhatsAppAccount, rawPayload json.RawMessage) {
    var payload GowaMessagePayload
    json.Unmarshal(rawPayload, &payload)
    
    // Deduplication (same as Meta path)
    // ...
    
    // Extract contact info
    phone := extractPhone(payload.From)
    orgID := account.OrganizationID
    
    // Get or create contact (same as Meta path)
    contact, _ := contactutil.GetOrCreateContact(a.DB, orgID, phone, payload.FromName)
    
    // Normalize to models.Message
    msg := &models.Message{
        OrganizationID: orgID,
        ContactID:      contact.ID,
        WhatsAppAccountID: account.ID,
        Direction:      "inbound",
        MessageID:      payload.ID,
        Status:         "received",
        Source:         "gowa",
        // ... common fields
    }
    
    // Extract message content (GOWA-specific media handling)
    msg = extractGowaMessageContent(msg, &payload, account)
    
    // Save to DB (same as Meta path)
    a.DB.Create(msg)
    
    // Run chatbot processing (same as Meta path)
    a.processIncomingChatbot(account, msg, contact)
    
    // Broadcast via WebSocket (same as Meta path)
    a.broadcastMessage(msg)
    
    // Dispatch webhook events (same as Meta path)
    a.dispatchWebhookEvent("message.received", msg)
}
```

---

## 6. GOWA-Specific Features (Phase 2+)

These features have no Meta equivalent and require dedicated UI/API modules:

### 6.1 Device Management

GOWA accounts require device lifecycle management (QR pairing, code pairing, passkey, connection monitoring). This maps to a new UI section:

- **Device dashboard:** Show connected GOWA devices with connection status (disconnected/connecting/connected/logged_in)
- **QR pairing flow:** Generate QR code, display for scanning, poll for connection
- **Pairing code flow:** Enter phone number, display code for headless linking
- **Connection health:** WebSocket-based real-time status updates, auto-reconnect monitoring
- **Per-device webhook config:** URL, secret, event filter (maps to GOWA's per-device webhook system)

### 6.2 Group Management

Full group lifecycle through whatomate's UI:
- Create groups with participants
- Manage participants (add, remove, promote, demote)
- Group settings (name, topic, photo, locked, announce mode)
- Invite link management (get, reset)
- Join request management (approve, reject)
- Participant CSV export
- Group info preview from link (without joining)

### 6.3 Extended Message Actions

UI additions for GOWA-only message actions:
- Emoji reactions (picker + display)
- Message editing (within 15-minute window)
- Message revocation (delete for everyone)
- Message starring
- Message forwarding from chat storage

### 6.4 Chat History Browser

GOWA provides paginated chat history — a feature Meta lacks:
- Browse all chats (with search, media filter, archived filter)
- View chat message history (with time range, media type, sender filters)
- Pin/unpin and archive/unarchive chats
- Set disappearing message timers

### 6.5 Newsletter/Channel Support

- List followed channels
- View channel messages
- Unfollow channels

### 6.6 Additional Send Capabilities

UI additions to the message composer for GOWA accounts:
- Sticker sending (auto WebP conversion)
- Contact card sending (vCard)
- Location sharing (map picker)
- Poll creation (question + options)
- Typing indicator toggling
- @mentions and @everyone (in groups)
- Ephemeral message timer per send

---

## 7. Implementation Plan

### Phase 1: Provider Interface Extraction

**Goal:** Extract a `WhatsAppProvider` interface without changing any behavior.

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `feat: define WhatsAppProvider interface` | Create `pkg/whatsapp/provider.go` with interface and `ProviderFeatures` struct | New file |
| `refactor: make Meta Client implement Provider interface` | Add `ProviderType()`, `SupportedFeatures()`, and ensure all existing methods satisfy the interface | `pkg/whatsapp/client.go` |
| `refactor: change App.WhatsApp to interface type` | Change `App.WhatsApp` from `*whatsapp.Client` to `WhatsAppProvider`; update `main.go` instantiation | `internal/handlers/app.go`, `cmd/whatomate/main.go` |
| `test: verify all existing tests pass` | Ensure zero regression — all handlers, webhooks, chatbot, campaigns work unchanged | Test files |

**Risk:** `App.WhatsApp` is referenced in 13+ handler files. The type change is mechanical but must be done carefully.

### Phase 2: GOWA Client Implementation

**Goal:** A working GOWA client that can send and receive messages.

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `feat: add GOWA client package` | `pkg/gowa/client.go` implementing `WhatsAppProvider` for text, image, video, audio, document sends + media upload/download | New package |
| `feat: GOWA webhook handler` | `/api/webhook/gowa` endpoint, envelope parsing, HMAC verification, message normalization to `models.Message` | `internal/handlers/gowa_webhook.go` |
| `feat: extend WhatsAppAccount model` | `provider_type` column + GOWA credential fields + DB migration | `internal/models/models.go`, migration file |
| `feat: provider registry` | `ProviderRegistry` type, registration on startup, resolution by org+account name | `internal/handlers/provider_registry.go` |
| `feat: startup provider initialization` | In `main.go`: create Meta client as before, create GOWA clients for each GOWA account, register in registry | `cmd/whatomate/main.go` |
| `feat: GOWA account resolution` | Extend `resolveWhatsAppAccount` to handle GOWA accounts, pass correct provider from registry | `internal/handlers/contacts.go` |
| `test: GOWA client unit tests` | Mock GOWA server tests for all send methods + webhook parsing tests | `pkg/gowa/client_test.go` |

### Phase 3: Account Management UI

**Goal:** Users can add and manage GOWA accounts from the whatomate UI.

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `feat: GOWA account creation API` | REST endpoints for creating GOWA accounts (server URL, credentials, device ID) | New API routes |
| `feat: GOWA account UI form` | Frontend form for adding GOWA accounts (provider type selector, credential fields) | Frontend |
| `feat: provider type indicator` | Account list shows Meta vs GOWA badge, feature availability indicators | Frontend |
| `feat: feature gating` | Disable template/interactive/flow/catalog/analytics UI for GOWA accounts | Frontend |
| `feat: GOWA device status widget` | Real-time connection status display for GOWA devices | Frontend |

### Phase 4: GOWA-Specific Features

**Goal:** Expose GOWA-only capabilities in whatomate.

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `feat: sticker/contact/location/poll sending` | Extend message composer for GOWA accounts with new message types | Frontend + API |
| `feat: message actions (react, edit, revoke, star)` | UI for GOWA-only message actions | Frontend + API |
| `feat: typing indicators` | Real-time typing status via WebSocket (bidirectional with GOWA) | Frontend + WebSocket |
| `feat: QR pairing flow` | QR code generation and polling for GOWA device pairing | Frontend + API |
| `feat: pairing code flow` | Phone number input → code display for headless pairing | Frontend + API |

### Phase 5: Advanced Features (GOWA-Only)

**Goal:** Group management, chat history, newsletters.

| Commit | Description |
|--------|-------------|
| `feat: group management module` | Full group CRUD, participant management, settings, invite links |
| `feat: chat history browser` | Chat listing, message history viewing, pin/archive, disappearing messages |
| `feat: newsletter support` | Channel listing, message viewing, unfollow |
| `feat: message forwarding` | Forward messages from GOWA chat storage |
| `feat: ephemeral messages` | Per-message disappearing timer in composer |
| `feat: @mentions in groups` | Ghost mentions, @everyone support in group chats |

---

## 8. Webhook Payload Mapping: GOWA → Whatomate

### 8.1 Message Normalization

| GOWA Field | Whatomate `models.Message` Field | Transformation |
|-----------|--------------------------------|---------------|
| `payload.id` | `MessageID` | Direct |
| `payload.chat_id` | `ChatID` | Direct |
| `payload.from` | Contact phone (derived) | Strip `@s.whatsapp.net` |
| `payload.from_name` | Contact name | `GetOrCreateContact` |
| `payload.timestamp` | `CreatedAt` | Parse RFC3339 |
| `payload.body` | `Content` | Direct |
| `payload.image` | `MessageType: "image"`, `MediaURL: path/url` | Handle polymorphism (string vs object) |
| `payload.video` | `MessageType: "video"`, `MediaURL: path/url` | Extract from object |
| `payload.audio` | `MessageType: "audio"`, `MediaURL: path` | Extract string or object |
| `payload.document` | `MessageType: "document"`, `MediaURL: path/url` | Extract from object |
| `payload.sticker` | `MessageType: "sticker"`, `MediaURL: path` | Extract string |
| `payload.location` | `MessageType: "location"`, `Latitude`, `Longitude` | Extract coordinates |
| `payload.contact` | `MessageType: "contact"`, `Content: vCard` | Extract vCard text |
| `payload.replied_to_id` | `ReplyToMessageID` | Direct |
| `payload.is_from_me` | `Direction: "outbound"` | Boolean to string |
| `payload.forwarded` | `IsForwarded: true` | Direct |

### 8.2 Status/ACK Normalization

| GOWA `receipt_type` | Whatomate Message Status |
|--------------------|-----------------------|
| `delivered` | `status: "delivered"` |
| `read` | `status: "read"` |

### 8.3 Event-to-Whatomate Webhook Mapping

| GOWA Event | Whatomate Dispatch Event | Notes |
|-----------|------------------------|-------|
| `message` | `message.received` | After normalization |
| `message.ack` (delivered) | `message.status_update` | Status: delivered |
| `message.ack` (read) | `message.status_update` | Status: read |
| `message.reaction` | `message.reaction` | New event type |
| `message.edited` | `message.edited` | New event type |
| `message.revoked` | `message.deleted` | Map to existing delete event |
| `message.deleted` | `message.deleted_for_me` | New event type |
| `chat_presence` (composing) | `contact.typing` | New event type |
| `chat_presence` (paused) | `contact.idle` | New event type |
| `call.offer` | `call.incoming` | Map to existing call event |
| `group.participants` | `group.participants_changed` | New event type |

---

## 9. Configuration

### 9.1 New Configuration Fields

```go
// internal/config/config.go — additions

type GowaConfig struct {
    Enabled bool   `env:"GOWA_ENABLED" default:"false"`
    // No global GOWA server URL — each account configures its own
}
```

GOWA accounts are configured per-organization, not globally. Each `WhatsAppAccount` record with `provider_type = "gowa"` carries its own server URL and credentials.

### 9.2 Webhook Configuration

For GOWA to forward events to whatomate:

```
GOWA per-device webhook URL: https://whatomate.example.com/api/webhook/gowa
GOWA per-device webhook secret: <account.GowaWebhookSecret>
GOWA per-device webhook events: message,message.ack,message.reaction,message.edited,message.revoked,chat_presence,call.offer
```

This is configured either:
- Via GOWA's API: `PATCH /devices/{device_id}/webhook`
- Or by whatomate automatically configuring it during device pairing

---

## 10. Risk Assessment

### 10.1 High Risk

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **WhatsApp account ban** | Critical — loss of WhatsApp number for the organization | Prominent disclaimer in UI, default to Meta, monitor whatsmeow updates, implement rate limiting on GOWA sends |
| **whatsmeow protocol breakage** | High — GOWA stops working after WhatsApp updates | Monitor GOWA GitHub for updates, version-pin GOWA binary, implement health check monitoring, rapid rollback plan |
| **No interface abstraction (current)** | High — Phase 1 requires touching 13+ files | Comprehensive test coverage before refactoring, feature flags for rollback |

### 10.2 Medium Risk

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **REST/MCP exclusivity** | Medium — cannot run both REST and MCP for same device in one GOWA process | Always use REST mode for whatomate integration (MCP is for AI agent tools, not platform use) |
| **Webhook payload inconsistency** | Medium — timestamp placement varies by event type, image field is polymorphic | Comprehensive webhook parsing tests, document edge cases |
| **GOWA session persistence** | Medium — sessions can become invalid, requiring re-pairing | Implement device health monitoring, alert users on disconnect, provide easy re-pairing flow |
| **Media storage divergence** | Medium — GOWA stores media locally, Meta on CDN | GOWA media paths are server-relative; whatomate must handle both URL-based (Meta) and path-based (GOWA) media references |

### 10.3 Low Risk

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Feature gating complexity** | Low — some features only work on one provider | Clear UI indicators, API returns `ProviderFeatures` per account |
| **500-group protocol limit** | Low — WhatsApp protocol constraint | Document limitation, implement pagination |
| **Webhook retry exhaustion** | Low — GOWA retries 5 times with max 16s backoff | Ensure whatomate webhook endpoints respond within 10s, implement async processing |

---

## 11. Testing Strategy

### 11.1 Unit Tests

| Component | Test Focus |
|-----------|-----------|
| `pkg/gowa/client.go` | Each send method against mock HTTP server, error handling, media upload |
| `internal/handlers/gowa_webhook.go` | Envelope parsing, HMAC verification, message normalization for all 16 event types |
| `pkg/whatsapp/provider.go` | Interface contract (compile-time check) |

### 11.2 Integration Tests

| Test | Focus |
|------|-------|
| GOWA webhook → message processing | Full pipeline: webhook → normalize → DB save → WebSocket broadcast → chatbot |
| Outgoing message via GOWA | Full pipeline: UI → API → provider registry → GOWA client → GOWA server |
| Provider registry | Multi-provider scenario: Meta account + GOWA account in same org |
| Feature gating | Verify Meta-only features return errors for GOWA accounts |

### 11.3 End-to-End Tests

| Test | Focus |
|------|-------|
| Full conversation (GOWA) | Send message from WhatsApp → receive in whatomate → reply from whatomate → receive in WhatsApp |
| Account switching | Send from Meta account → send from GOWA account in same org → verify correct routing |
| Device reconnection | Disconnect GOWA device → verify status update → reconnect → verify messages flow again |

---

## 12. Out of Scope

- **MCP Server integration:** GOWA's MCP mode is for AI agent tools (Cursor, n8n), not whatomate platform use. Always use REST mode.
- **Chatwoot bridge:** Whatomate has its own shared inbox. GOWA's Chatwoot integration is not used.
- **WhatsApp Calling:** Whatomate's WebRTC calling feature remains Meta-only.
- **WhatsApp Flows:** Meta-only interactive feature.
- **Catalog/Commerce:** Meta-only.
- **Campaign management via GOWA:** Campaigns initially remain Meta-only; GOWA campaign support is Phase 5+.
- **WhatsApp Business Profile update via GOWA:** GOWA can only read business profiles, not update them.
- **Direct PostgreSQL Chatwoot import:** Not relevant to whatomate.
- **Meta Ads attribution:** GOWA supports it but whatomate doesn't currently use it.

---

## 13. Success Metrics

| Metric | Target |
|--------|--------|
| **Zero regression** | All existing Meta Cloud API features pass without modification |
| **Message delivery** | GOWA messages delivered in < 3s (p95) |
| **Webhook processing** | GOWA webhook processed in < 500ms |
| **Provider switch** | Adding/removing GOWA accounts without restarting whatomate |
| **Feature gating accuracy** | 0% of Meta-only features accessible on GOWA accounts |
| **Test coverage** | > 80% for new GOWA client and webhook handler code |

---

## 14. Glossary

| Term | Definition |
|------|-----------|
| **GOWA** | Go WhatsApp Web Multi-Device — self-hosted WhatsApp gateway using whatsmeow |
| **whatsmeow** | Go library implementing WhatsApp Web Multi-Device protocol |
| **Provider** | A WhatsApp backend implementation (Meta Cloud API or GOWA) |
| **Device** | A WhatsApp account registered in a GOWA instance |
| **Session ID** | Tenant correlation identifier in GOWA webhook payloads |
| **HSM** | Highly Structured Message — Meta's pre-approved message template system |
| **WABA** | WhatsApp Business Account — Meta's business-level container |
| **PhoneID** | Meta's identifier for a phone number in the Business API |
| **MCP** | Model Context Protocol — GOWA's SSE-based AI agent interface |
| **JID** | Jabber ID — WhatsApp's internal identifier format (`number@s.whatsapp.net`, `groupid@g.us`) |

---

## 15. References

| Document | Location |
|----------|----------|
| GOWA PRD v8.10.0 | `GOWA/docs/gowa.prd.md` |
| GOWA OpenAPI Specification | `GOWA/docs/openapi.yaml` (5087 lines) |
| GOWA Webhook Payloads | `GOWA/docs/webhook-payload.md` |
| GOWA Per-Device Webhook Plan | `GOWA/docs/per-device-webhook-plan.md` |
| GOWA Chatwoot Integration | `GOWA/docs/chatwoot.md` |
| encantoWhatsapp Frontend PRD | `GOWA/docs/frontend.prd.md` |
| GOWA SDK Configuration | `GOWA/docs/sdk/config.yaml` |
| GOWA Agent Progress | `GOWA/docs/agent-progress.md` |
| Whatomate Meta Client | `pkg/whatsapp/` (8 files) |
| Whatomate Handlers | `internal/handlers/` (13+ files) |
| Whatomate Models | `internal/models/models.go` |
| Whatomate Config | `internal/config/config.go` |
| Whatomate Main | `cmd/whatomate/main.go` |
| GOWA README | `GOWA/whatsapp_8.10.0_darwin_arm64/readme.md` |
| GOWA API Reference | `gowa_api.html` (Arabic) |
