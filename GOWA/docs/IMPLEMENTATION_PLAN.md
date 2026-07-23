# GOWA Integration — Surgical Implementation Plan

**Based on:** `INTEGRATION_PRD.md` v1.0  
**Date:** 2026-07-11  
**Scope:** Add GOWA as a second WhatsApp provider alongside Meta Cloud API  
**Commit strategy:** One phase = one PR. Each phase compiles and passes existing tests before merging.

---

## Phase 0 — Preparation (Branch + Scaffold)

**Goal:** Create the workspace without breaking anything.

| Step | Action | File(s) |
|------|--------|---------|
| 0.1 | Create branch `feat/gowa-provider` off `main` | git |
| 0.2 | Create directory `pkg/gowa/` | new dir |
| 0.3 | Create directory `pkg/waprovider/` | new dir |
| 0.4 | Create file `pkg/waprovider/provider.go` (interface stub only — empty for now) | new file |
| 0.5 | Verify `go build ./...` still passes | — |

No existing code is changed. This phase only creates empty scaffolding.

---

## Phase 1 — Extract the Provider Interface

**Goal:** Define `waprovider.Provider` — the abstraction that both Meta and GOWA will implement. The interface is derived from every method on `*whatsapp.Client` that handlers actually call.

### Step 1.1 — Inventory all handler → `a.WhatsApp.*` call sites

These are the exact methods used across all handler files (from the codebase review):

| Method | File:Line (caller) | Signature |
|--------|--------------------|-----------|
| `SendTextMessage` | `messages.go:168` | `(ctx, account, recipient, text, replyTo) → (string, error)` |
| `UploadMedia` | `messages.go:175` | `(ctx, account, data, mime, filename) → (string, error)` |
| `SendImageMessage` | `messages.go:183` | `(ctx, account, recipient, mediaID, caption) → (string, error)` |
| `SendVideoMessage` | `messages.go:185` | `(ctx, account, recipient, mediaID, caption) → (string, error)` |
| `SendAudioMessage` | `messages.go:187` | `(ctx, account, recipient, mediaID) → (string, error)` |
| `SendDocumentMessage` | `messages.go:189` | `(ctx, account, recipient, mediaID, filename, caption) → (string, error)` |
| `SendCTAURLButton` | `messages.go:195` | `(ctx, account, recipient, bodyText, buttonText, url) → (string, error)` |
| `SendVoiceCallButton` | `messages.go:197` | `(ctx, account, recipient, bodyText, displayText, ttl, payload) → (string, error)` |
| `SendInteractiveButtons` | `messages.go:199` | `(ctx, account, recipient, bodyText, buttons) → (string, error)` |
| `SendTemplateMessage` | `messages.go:221` | `(ctx, account, recipient, name, lang, components) → (string, error)` |
| `SendFlowMessage` | `messages.go:227` | `(ctx, account, recipient, flowID, header, body, cta, token, screens) → (string, error)` |
| `MarkMessageRead` | `webhook.go` (read receipt) | `(ctx, account, messageID) → error` |
| `DownloadMedia` | `webhook.go` / media handlers | `(ctx, account, mediaID) → ([]byte, string, error)` |
| `ValidateCredentials` | account management | `(ctx, account) → error` |
| `SubscribeApp` | account onboarding | `(ctx, appID, appSecret, callbackURL, verifyToken, fields) → error` |
| `ExchangeCodeForToken` | OAuth flow | `(ctx, appID, appSecret, code) → (string, error)` |
| `GetPhoneNumberInfo` | account onboarding | `(ctx, accessToken, fields) → (PhoneNumberInfo, error)` |
| `RegisterPhoneNumber` | account onboarding | `(ctx, account, pin) → error` |

### Step 1.2 — Define the interface

**File: `pkg/waprovider/provider.go`**

```go
package waprovider

import (
    "context"
    "io"

    "github.com/shridarpatil/whatomate/pkg/whatsapp"
)

// Capability flags indicate which features a provider supports.
// Handlers check these before calling optional methods.
type Capabilities struct {
    Templates     bool // SendTemplateMessage, SubmitTemplate, FetchTemplates
    Flows         bool // SendFlowMessage, flow CRUD
    Calls         bool // voice_call interactive buttons, call events
    Catalog       bool // catalog CRUD
    Analytics     bool // GetAnalytics
    BusinessProfile bool // GetBusinessProfile, UpdateBusinessProfile
    MediaUpload   bool // two-step upload-then-send (Meta) vs inline (GOWA)
    Interactive   bool // interactive buttons, CTA URL, list messages
}

// Provider is the unified interface for WhatsApp messaging backends.
// Both Meta Cloud API and GOWA implement this interface.
type Provider interface {
    // Name returns the provider identifier (e.g., "meta", "gowa").
    Name() string

    // Capabilities returns the feature set supported by this provider.
    Capabilities() Capabilities

    // --- Core messaging (shared between Meta and GOWA) ---

    SendTextMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, text, replyTo string) (messageID string, err error)

    // Media: provider handles upload internally if needed.
    // For Meta, this is upload-then-send. For GOWA, this is inline multipart.
    SendImageMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename, mimeType, caption string) (messageID string, err error)
    SendVideoMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename, mimeType, caption string) (messageID string, err error)
    SendAudioMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename, mimeType string) (messageID string, err error)
    SendDocumentMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename, mimeType, caption string) (messageID string, err error)

    MarkMessageRead(ctx context.Context, account *whatsapp.Account, messageID string) error

    // --- Interactive messages (shared, GOWA may have limitations) ---

    SendInteractiveButtons(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, bodyText string, buttons []whatsapp.Button) (messageID string, err error)
    SendCTAURLButton(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, bodyText, buttonText, url string) (messageID string, err error)

    // --- Download media (shared) ---
    DownloadMedia(ctx context.Context, account *whatsapp.Account, mediaID string) (data []byte, mimeType string, err error)

    // --- Provider setup (Meta-only for most) ---
    ValidateCredentials(ctx context.Context, account *whatsapp.Account) error

    // --- Meta-only methods (GOWA returns ErrNotSupported) ---

    SendTemplateMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, name, language string, components []whatsapp.TemplateComponent) (messageID string, err error)
    SendFlowMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, flowID string, flowHeader whatsapp.FlowHeader, bodyText, flowCTA, flowToken string, flowFirstScreen map[string]any) (messageID string, err error)
    SendVoiceCallButton(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, bodyText, displayText string, ttlMinutes int, payload any) (messageID string, err error)
    SubscribeApp(ctx context.Context, appID, appSecret, callbackURL, verifyToken string, fields []string) error
    ExchangeCodeForToken(ctx context.Context, appID, appSecret, code string) (accessToken string, err error)
    GetPhoneNumberInfo(ctx context.Context, accessToken string, fields []string) (whatsapp.PhoneNumberInfo, error)
    RegisterPhoneNumber(ctx context.Context, account *whatsapp.Account, pin string) error

    // --- Template CRUD (Meta-only) ---
    SubmitTemplate(ctx context.Context, account *whatsapp.Account, template whatsapp.TemplateSubmit) (string, error)
    FetchTemplates(ctx context.Context, account *whatsapp.Account) ([]whatsapp.MetaTemplate, error)
    DeleteTemplate(ctx context.Context, account *whatsapp.Account, templateName string) error

    // --- Catalog CRUD (Meta-only) ---
    CreateCatalog(ctx context.Context, account *whatsapp.Account, name, address string) error
    ListCatalogs(ctx context.Context, account *whatsapp.Account) (any, error)
    DeleteCatalog(ctx context.Context, account *whatsapp.Account, catalogID string) error

    // --- Analytics (Meta-only) ---
    GetAnalytics(ctx context.Context, account *whatsapp.Account, start, end string, fields []string) (any, error)

    // --- Business Profile (Meta-only) ---
    GetBusinessProfile(ctx context.Context, account *whatsapp.Account) (any, error)
    UpdateBusinessProfile(ctx context.Context, account *whatsapp.Account, params any) error

    // --- GOWA-only methods (Meta returns ErrNotSupported) ---
    // These will be added in Phase 3 when GOWA client is implemented.
}

// ErrNotSupported is returned when a method is called on a provider
// that does not support the operation (e.g., SendTemplateMessage on GOWA).
var ErrNotSupported = errors.New("operation not supported by this provider")
```

### Step 1.3 — Make `*whatsapp.Client` implement `Provider`

**File: `pkg/whatsapp/client.go`**

Add adapter methods that satisfy the interface. Since `Client` already has all the methods, we only need:

```go
func (c *Client) Name() string                                    { return "meta" }
func (c *Client) Capabilities() waprovider.Capabilities { ... }    // return all-true
```

Plus adjust `SendImageMessage`, `SendVideoMessage`, `SendAudioMessage`, `SendDocumentMessage` signatures to accept `io.Reader` instead of `mediaID string` — OR keep the existing signatures and add thin wrappers. **Recommended: add wrapper methods** to avoid changing existing signatures in Phase 1:

```go
// SendImageMessageFromReader satisfies waprovider.Provider.
// It uploads the media then sends the image message.
func (c *Client) SendImageMessageFromReader(ctx context.Context, account *Account, recipient Recipient, media io.Reader, filename, mimeType, caption string) (string, error) {
    data, err := io.ReadAll(media)
    if err != nil {
        return "", fmt.Errorf("read media: %w", err)
    }
    mediaID, err := c.UploadMedia(ctx, account, data, mimeType, filename)
    if err != nil {
        return "", err
    }
    return c.SendImageMessage(ctx, account, recipient, mediaID, caption)
}
```

**Files changed:** `pkg/whatsapp/client.go` (add `Name()`, `Capabilities()`, 4 `*FromReader` wrappers)

### Step 1.4 — Change `App.WhatsApp` from concrete to interface

**File: `internal/handlers/app.go:31`**

```diff
- WhatsApp          *whatsapp.Client
+ WhatsApp          waprovider.Provider
```

**File: `cmd/whatomate/main.go:182,206`**

```diff
- waClient := whatsapp.NewWithBaseURL(lo, cfg.WhatsApp.BaseURL)
+ waClient := whatsapp.NewWithBaseURL(lo, cfg.WhatsApp.BaseURL)
+ var waProvider waprovider.Provider = waClient  // Meta provider
  
  app := &handlers.App{
      ...
-     WhatsApp:   waClient,
+     WhatsApp:   waProvider,
  }
```

### Step 1.5 — Fix all handler compilation errors

The sendFn closure in `messages.go:156-232` calls methods like `a.WhatsApp.UploadMedia` then `a.WhatsApp.SendImageMessage` (two-step). With the new interface, media methods accept `io.Reader`. Refactor the sendFn:

**File: `internal/handlers/messages.go:156-232`**

```go
sendFn := func(sendCtx context.Context) (string, error) {
    waAccount := a.toWhatsAppAccount(req.Account)
    rcpt := whatsapp.Recipient{Phone: req.Contact.PhoneNumber, BSUID: req.Contact.BSUID}
    var replyToMsgID string
    if req.ReplyToMessage != nil && req.ReplyToMessage.WhatsAppMessageID != "" {
        replyToMsgID = req.ReplyToMessage.WhatsAppMessageID
    }

    switch req.Type {
    case models.MessageTypeText:
        return a.WhatsApp.SendTextMessage(sendCtx, waAccount, rcpt, req.Content, replyToMsgID)

    case models.MessageTypeImage, models.MessageTypeVideo, models.MessageTypeAudio, models.MessageTypeDocument:
        // Build reader from MediaData or MediaURL
        var mediaReader io.Reader
        if len(req.MediaData) > 0 {
            mediaReader = bytes.NewReader(req.MediaData)
        } else if req.MediaURL != "" {
            resp, err := a.HTTPClient.Get(req.MediaURL)
            if err != nil {
                return "", fmt.Errorf("fetch media URL: %w", err)
            }
            defer resp.Body.Close()
            mediaReader = resp.Body
        }
        switch req.Type {
        case models.MessageTypeImage:
            return a.WhatsApp.SendImageMessage(sendCtx, waAccount, rcpt, mediaReader, req.MediaFilename, req.MediaMimeType, req.Caption)
        case models.MessageTypeVideo:
            return a.WhatsApp.SendVideoMessage(sendCtx, waAccount, rcpt, mediaReader, req.MediaFilename, req.MediaMimeType, req.Caption)
        case models.MessageTypeAudio:
            return a.WhatsApp.SendAudioMessage(sendCtx, waAccount, rcpt, mediaReader, req.MediaFilename, req.MediaMimeType)
        default: // document
            return a.WhatsApp.SendDocumentMessage(sendCtx, waAccount, rcpt, mediaReader, req.MediaFilename, req.MediaMimeType, req.Caption)
        }

    case models.MessageTypeInteractive:
        // ... same as before, these signatures don't change ...

    case models.MessageTypeTemplate:
        // ... same as before ...

    case models.MessageTypeFlow:
        // ... same as before ...

    default:
        return "", fmt.Errorf("unsupported message type: %s", req.Type)
    }
}
```

### Step 1.6 — Handle `UploadMedia` standalone calls

Search for other direct `a.WhatsApp.UploadMedia` calls (media handlers, profile upload, etc.) and convert them to either:
- Use the new `SendImageMessageFromReader` path (preferred), or
- Keep a `UploadMedia` method on the interface if needed for non-send contexts (profile picture, etc.)

### Step 1.7 — Handle Meta-only methods with capability checks

For methods that only Meta supports (templates, flows, calls, catalog, analytics, business profile), add capability guards in handlers:

```go
if !a.WhatsApp.Capabilities().Templates {
    return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Templates not supported by this provider", nil, "")
}
```

**Files touched:**
| File | Change |
|------|--------|
| `pkg/waprovider/provider.go` | NEW — interface + Capabilities + ErrNotSupported |
| `pkg/whatsapp/client.go` | ADD `Name()`, `Capabilities()`, 4 `*FromReader` wrappers |
| `internal/handlers/app.go:31` | CHANGE type from `*whatsapp.Client` → `waprovider.Provider` |
| `cmd/whatomate/main.go:182-206` | CAST `waClient` to `waprovider.Provider` |
| `internal/handlers/messages.go:156-232` | REFACTOR sendFn to use `io.Reader` media methods |
| template handlers | ADD capability guards |
| flow handlers | ADD capability guards |
| catalog handlers | ADD capability guards |
| analytics handlers | ADD capability guards |

### Step 1.8 — Verify

```bash
go build ./...
go test ./...
```

Both must pass with zero errors. **Phase 1 is the highest-risk phase** because it touches the most files.

---

## Phase 2 — Extend the Data Model

**Goal:** Add `provider_type` and GOWA-specific credential fields to `WhatsAppAccount`.

### Step 2.1 — Add fields to `WhatsAppAccount`

**File: `internal/models/models.go:293-322`**

```go
type WhatsAppAccount struct {
    BaseModel
    OrganizationID     uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
    Name               string    `gorm:"size:100;uniqueIndex:idx_wa_org_name;not null" json:"name"`
    
    // --- NEW: Provider discrimination ---
    ProviderType       string    `gorm:"size:20;default:'meta';index" json:"provider_type"` // "meta" or "gowa"
    
    // --- Meta credentials (nullable when provider_type = "gowa") ---
    AppID              string    `gorm:"size:100" json:"app_id,omitempty"`
    PhoneID            string    `gorm:"size:100" json:"phone_id,omitempty"`
    BusinessID         string    `gorm:"size:100" json:"business_id,omitempty"`
    AccessToken        string    `gorm:"type:text" json:"-"` // encrypted
    AppSecret          string    `gorm:"size:255" json:"-"`  // encrypted
    WebhookVerifyToken string    `gorm:"size:255" json:"webhook_verify_token,omitempty"`
    APIVersion         string    `gorm:"size:20;default:'v21.0'" json:"api_version,omitempty"`
    
    // --- NEW: GOWA credentials (nullable when provider_type = "meta") ---
    GowaBaseURL        string    `gorm:"size:255" json:"gowa_base_url,omitempty"`        // e.g. "http://gowa:8080"
    GowaDeviceID       string    `gorm:"size:100" json:"gowa_device_id,omitempty"`        // GOWA device UUID
    GowaWebhookSecret  string    `gorm:"size:255" json:"-"`                                // HMAC secret for GOWA webhooks (encrypted)
    
    // --- Shared fields ---
    IsDefaultIncoming  bool      `gorm:"default:false" json:"is_default_incoming"`
    IsDefaultOutgoing  bool      `gorm:"default:false" json:"is_default_outgoing"`
    AutoReadReceipt    bool      `gorm:"default:false" json:"auto_read_receipt"`
    BusinessCallingEnabled bool   `gorm:"default:false" json:"business_calling_enabled"`
    IsSMB              bool      `gorm:"default:false" json:"is_smb"`
    Status             string    `gorm:"size:20;default:'active'" json:"status"`
    Pin                string    `gorm:"size:255" json:"-"`
    CreatedByID        *uuid.UUID `gorm:"type:uuid" json:"created_by_id,omitempty"`
    UpdatedByID        *uuid.UUID `gorm:"type:uuid" json:"updated_by_id,omitempty"`

    Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
    CreatedBy    *User         `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
    UpdatedBy    *User         `gorm:"foreignKey:UpdatedByID" json:"updated_by,omitempty"`
}
```

### Step 2.2 — Update `DecryptSecrets`

**File: `internal/models/models.go:340-342`**

```go
func (a *WhatsAppAccount) DecryptSecrets(encryptionKey string) {
    crypto.DecryptFields(encryptionKey, &a.AccessToken, &a.AppSecret, &a.Pin, &a.GowaWebhookSecret)
}
```

### Step 2.3 — Update `ToWAAccount` to handle GOWA

**File: `internal/models/models.go:329-337`**

```go
func (a *WhatsAppAccount) ToWAAccount() *whatsapp.Account {
    return &whatsapp.Account{
        PhoneID:     a.PhoneID,
        BusinessID:  a.BusinessID,
        AppID:       a.AppID,
        APIVersion:  a.APIVersion,
        AccessToken: a.AccessToken,
        // NEW: provider context
        ProviderType:    a.ProviderType,
        GowaBaseURL:     a.GowaBaseURL,
        GowaDeviceID:    a.GowaDeviceID,
    }
}
```

### Step 2.4 — Add `ProviderType` and GOWA fields to `whatsapp.Account`

**File: `pkg/whatsapp/types.go:10-16`**

```go
type Account struct {
    PhoneID     string
    BusinessID  string
    AppID       string
    APIVersion  string
    AccessToken string
    // NEW
    ProviderType  string // "meta" or "gowa"
    GowaBaseURL  string
    GowaDeviceID string
}
```

### Step 2.5 — GORM AutoMigrate

The new columns will be auto-migrated by existing `db.AutoMigrate(...)` calls. No manual migration needed for dev. For production, provide a SQL migration:

```sql
ALTER TABLE whatsapp_accounts ADD COLUMN provider_type VARCHAR(20) DEFAULT 'meta';
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_base_url VARCHAR(255) DEFAULT '';
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_device_id VARCHAR(100) DEFAULT '';
ALTER TABLE whatsapp_accounts ADD COLUMN gowa_webhook_secret VARCHAR(255) DEFAULT '';
CREATE INDEX idx_whatsapp_accounts_provider_type ON whatsapp_accounts(provider_type);
```

### Step 2.6 — Add GOWA config section

**File: `internal/config/config.go`**

```go
type Config struct {
    // ... existing fields ...
    WhatsApp WhatsAppConfig `koanf:"whatsapp"`
    GOWA     GOWAConfig     `koanf:"gowa"`  // NEW
}

type GOWAConfig struct {
    BaseURL    string `koanf:"base_url"`    // default "http://localhost:8080"
    WebhookPath string `koanf:"webhook_path"` // default "/api/gowa/webhook"
}
```

### Step 2.7 — Verify

```bash
go build ./...
go test ./...
```

**Files touched:**
| File | Change |
|------|--------|
| `internal/models/models.go:293-342` | ADD `ProviderType`, `GowaBaseURL`, `GowaDeviceID`, `GowaWebhookSecret` |
| `pkg/whatsapp/types.go:10-16` | ADD `ProviderType`, `GowaBaseURL`, `GowaDeviceID` |
| `internal/config/config.go` | ADD `GOWAConfig` struct + field on `Config` |

---

## Phase 3 — Implement the GOWA Client

**Goal:** Build `pkg/gowa/client.go` that implements `waprovider.Provider`.

### Step 3.1 — GOWA client skeleton

**File: `pkg/gowa/client.go`** (NEW)

```go
package gowa

import (
    "context"
    "errors"
    "io"

    "github.com/shridarpatil/whatomate/pkg/waprovider"
    "github.com/shridarpatil/whatomate/pkg/whatsapp"
)

type Client struct {
    baseURL    string  // GOWA REST API base URL
    httpClient *http.Client
}

func New(baseURL string) *Client {
    return &Client{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) Name() string { return "gowa" }

func (c *Client) Capabilities() waprovider.Capabilities {
    return waprovider.Capabilities{
        Templates: false,      // GOWA has no template API
        Flows: false,
        Calls: false,
        Catalog: false,
        Analytics: false,
        BusinessProfile: false,
        MediaUpload: false,   // GOWA sends inline, no two-step upload
        Interactive: true,   // GOWA supports basic interactive
    }
}
```

### Step 3.2 — Core messaging methods

GOWA REST API endpoints (from `openapi.yaml`):

| Whatomate Method | GOWA Endpoint | Key Difference from Meta |
|-----------------|---------------|---------------------------|
| SendTextMessage | `POST /api/{device}/send/text` | Body: `{phone, message, reply_to?}` |
| SendImageMessage | `POST /api/{device}/send/image` | Multipart: image file + `{phone, caption?, reply_to?, view_once?}` |
| SendVideoMessage | `POST /api/{device}/send/video` | Multipart: video file + `{phone, caption?, reply_to?}` |
| SendAudioMessage | `POST /api/{device}/send/audio` | Multipart: audio file + `{phone, reply_to?}` |
| SendDocumentMessage | `POST /api/{device}/send/document` | Multipart: document + `{phone, filename, caption?, reply_to?}` |
| MarkMessageRead | `POST /api/{device}/send/read` | Body: `{messages: [id1, id2, ...]}` |
| DownloadMedia | `GET /api/{device}/media?id={message_id}` | Single-step, returns media bytes |

**File: `pkg/gowa/messages.go`** (NEW)

```go
// SendTextMessage sends a text message via GOWA REST API.
func (c *Client) SendTextMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, text, replyTo string) (string, error) {
    deviceID := account.GowaDeviceID
    body := map[string]any{
        "phone": recipient.Phone,
        "message": text,
    }
    if replyTo != "" {
        body["reply_to"] = replyTo
    }
    
    resp, err := c.doJSON(ctx, "POST", c.url("/api/%s/send/text", deviceID), body)
    if err != nil {
        return "", err
    }
    return resp.MessageID, nil
}

// SendImageMessage sends an image via GOWA multipart upload.
// GOWA accepts the image directly — no separate upload step.
func (c *Client) SendImageMessage(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename, mimeType, caption string) (string, error) {
    deviceID := account.GowaDeviceID
    
    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)
    
    part, _ := writer.CreateFormFile("image", filename)
    io.Copy(part, media)
    
    writer.WriteField("phone", recipient.Phone)
    if caption != "" {
        writer.WriteField("caption", caption)
    }
    writer.Close()
    
    // POST multipart to /api/{device}/send/image
    resp, err := c.doMultipart(ctx, "POST", c.url("/api/%s/send/image", deviceID), writer.FormDataContentType(), &buf)
    if err != nil {
        return "", err
    }
    return resp.MessageID, nil
}
// ... similar for Video, Audio, Document ...
```

### Step 3.3 — Interactive messages

GOWA supports:
- **Interactive buttons**: `POST /api/{device}/send/interactive` with `{phone, type: "button", body, buttons}`
- **CTA URL**: `POST /api/{device}/send/interactive` with `{phone, type: "cta_url", body, button_text, url}`

```go
func (c *Client) SendInteractiveButtons(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, bodyText string, buttons []whatsapp.Button) (string, error) {
    gowaButtons := make([]map[string]string, len(buttons))
    for i, b := range buttons {
        gowaButtons[i] = map[string]string{"id": b.ID, "title": b.Title}
    }
    body := map[string]any{
        "phone": recipient.Phone,
        "type": "button",
        "body": bodyText,
        "buttons": gowaButtons,
    }
    // ... POST /api/{device}/send/interactive
}
```

### Step 3.4 — Download media

```go
func (c *Client) DownloadMedia(ctx context.Context, account *whatsapp.Account, mediaID string) ([]byte, string, error) {
    // GOWA: GET /api/{device}/media?id={mediaID}
    // Response is the media bytes directly
}
```

### Step 3.5 — ErrNotSupported stubs for Meta-only methods

```go
func (c *Client) SendTemplateMessage(...) (string, error) {
    return "", waprovider.ErrNotSupported
}
func (c *Client) SendFlowMessage(...) (string, error) {
    return "", waprovider.ErrNotSupported
}
// ... same for all Meta-only methods ...
```

### Step 3.6 — GOWA-specific methods (interface extensions)

Add to `waprovider.Provider` interface OR handle via type assertion:

```go
// GOWA-only capabilities accessed via type assertion:
type GowaExtensions interface {
    SendGroupMessage(ctx context.Context, account *whatsapp.Account, groupJID, text string) (string, error)
    GetGroupList(ctx context.Context, account *whatsapp.Account) ([]Group, error)
    GetChatHistory(ctx context.Context, account *whatsapp.Account, jid string, count int) ([]Message, error)
    SendReaction(ctx context.Context, account *whatsapp.Account, messageID, emoji string) error
    SendSticker(ctx context.Context, account *whatsapp.Account, recipient whatsapp.Recipient, media io.Reader, filename string) (string, error)
    GetDeviceStatus(ctx context.Context, account *whatsapp.Account) (DeviceStatus, error)
}
```

### Step 3.7 — Verify

```bash
go build ./...
go test ./pkg/gowa/...
```

**Files touched:**
| File | Change |
|------|--------|
| `pkg/gowa/client.go` | NEW — Client struct, Name(), Capabilities(), doJSON(), doMultipart() |
| `pkg/gowa/messages.go` | NEW — SendTextMessage, SendImageMessage, SendVideoMessage, SendAudioMessage, SendDocumentMessage |
| `pkg/gowa/interactive.go` | NEW — SendInteractiveButtons, SendCTAURLButton |
| `pkg/gowa/media.go` | NEW — DownloadMedia |
| `pkg/gowa/stubs.go` | NEW — ErrNotSupported for all Meta-only methods |
| `pkg/gowa/extensions.go` | NEW — GOWA-specific methods (groups, reactions, stickers, history) |
| `pkg/gowa/types.go` | NEW — GOWA response types, DeviceStatus, Group |

---

## Phase 4 — Provider Registry + Dynamic Resolution

**Goal:** Route messages to the correct provider based on `WhatsAppAccount.ProviderType`.

### Step 4.1 — Create provider registry

**File: `pkg/waprovider/registry.go`** (NEW)

```go
package waprovider

import (
    "sync"

    "github.com/shridarpatil/whatomate/internal/config"
    "github.com/shridarpatil/whatomate/pkg/gowa"
    "github.com/shridarpatil/whatomate/pkg/whatsapp"
    "github.com/zerodha/logf"
)

type Registry struct {
    mu       sync.RWMutex
    meta     Provider  // singleton Meta client
    gowa     map[string]Provider // keyed by baseURL
    log      logf.Logger
}

func NewRegistry(metaClient *whatsapp.Client, gowaCfg config.GOWAConfig, log logf.Logger) *Registry {
    r := &Registry{
        meta: metaClient,
        gowa: make(map[string]Provider),
        log:  log,
    }
    // Pre-create default GOWA client
    if gowaCfg.BaseURL != "" {
        r.gowa[gowaCfg.BaseURL] = gowa.New(gowaCfg.BaseURL)
    }
    return r
}

func (r *Registry) Get(providerType string, gowaBaseURL string) Provider {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    switch providerType {
    case "meta":
        return r.meta
    case "gowa":
        if p, ok := r.gowa[gowaBaseURL]; ok {
            return p
        }
        // Auto-create on first use
        r.mu.RUnlock()
        r.mu.Lock()
        defer r.mu.RUnlock()
        p := gowa.New(gowaBaseURL)
        r.gowa[gowaBaseURL] = p
        return p
    default:
        return r.meta // fallback
    }
}
```

### Step 4.2 — Wire registry into App

**File: `internal/handlers/app.go`**

```go
type App struct {
    // ...
    WhatsApp    waprovider.Provider    // DEPRECATED: remove after Phase 4.3
    WARegistry  *waprovider.Registry   // NEW: multi-provider routing
    // ...
}
```

**File: `cmd/whatomate/main.go`**

```go
waClient := whatsapp.NewWithBaseURL(lo, cfg.WhatsApp.BaseURL)
waRegistry := waprovider.NewRegistry(waClient, cfg.GOWA, lo)

app := &handlers.App{
    // ...
    WhatsApp:   waClient,     // keep for backward compat during migration
    WARegistry: waRegistry,
}
```

### Step 4.3 — Refactor `toWhatsAppAccount` → provider resolution

**File: `internal/handlers/contacts.go:709`** (`resolveWhatsAppAccount`)

After resolving the `WhatsAppAccount` from DB, get the correct provider:

```go
func (a *App) resolveProvider(account *models.WhatsAppAccount) waprovider.Provider {
    return a.WARegistry.Get(account.ProviderType, account.GowaBaseURL)
}
```

### Step 4.4 — Update sendFn to use resolved provider

**File: `internal/handlers/messages.go`**

```go
sendFn := func(sendCtx context.Context) (string, error) {
    waAccount := a.toWhatsAppAccount(req.Account)
    provider := a.resolveProvider(req.Account)  // NEW: dynamic provider
    
    rcpt := whatsapp.Recipient{Phone: req.Contact.PhoneNumber, BSUID: req.Contact.BSUID}
    // ... use `provider` instead of `a.WhatsApp` ...
    switch req.Type {
    case models.MessageTypeText:
        return provider.SendTextMessage(sendCtx, waAccount, rcpt, req.Content, replyToMsgID)
    // ...
    }
}
```

### Step 4.5 — Remove `App.WhatsApp` (cleanup)

After all handlers use `a.resolveProvider(account)` or `a.WARegistry.Get(...)`:

```diff
  type App struct {
-     WhatsApp          waprovider.Provider
      WARegistry         *waprovider.Registry
      // ...
  }
```

**Files touched:**
| File | Change |
|------|--------|
| `pkg/waprovider/registry.go` | NEW — Registry struct with Get() |
| `internal/handlers/app.go` | ADD `WARegistry`, REMOVE `WhatsApp` |
| `internal/handlers/messages.go` | CHANGE `a.WhatsApp.*` → `provider.*` |
| `internal/handlers/webhook.go` | CHANGE incoming message processing to use registry |
| `internal/handlers/contacts.go` | ADD `resolveProvider()` helper |
| `cmd/whatomate/main.go` | CREATE registry, wire into App |

---

## Phase 5 — GOWA Webhook Handler

**Goal:** Receive GOWA webhooks, normalize to internal model, feed into existing chat pipeline.

### Step 5.1 — Register GOWA webhook routes

**File: `cmd/whatomate/main.go:504-505`**

```go
// Existing Meta webhook routes
g.GET("/api/webhook", app.WebhookVerify)
g.POST("/api/webhook", app.WebhookHandler)

// NEW: GOWA webhook routes
g.GET("/api/gowa/webhook", app.GowaWebhookVerify)
g.POST("/api/gowa/webhook", app.GowaWebhookHandler)
```

### Step 5.2 — GOWA webhook payload types

**File: `pkg/gowa/webhook.go`** (NEW)

```go
package gowa

// GOWA webhook payload (from webhook-payload.md)
type WebhookPayload struct {
    Event    string    `json:"event"`     // "message", "ack", "presence", etc.
    DeviceID string    `json:"device_id"`
    SessionID string   `json:"session_id"`
    Timestamp int64    `json:"timestamp,omitempty"` // top-level for non-message events
    Payload   Payload   `json:"payload"`
}

type Payload struct {
    // Message fields
    ID        string `json:"id,omitempty"`
    From      string `json:"from,omitempty"`       // phone number or group JID
    To        string `json:"to,omitempty"`
    Type      string `json:"type,omitempty"`       // "text", "image", "video", etc.
    Text      string `json:"text,omitempty"`
    Timestamp int64  `json:"timestamp,omitempty"`   // INSIDE payload for message events
    
    // Media (polymorphic: string or object)
    Image     *MediaPayload `json:"image,omitempty"`
    Video     *MediaPayload `json:"video,omitempty"`
    Audio     *MediaPayload `json:"audio,omitempty"`
    Document  *MediaPayload `json:"document,omitempty"`
    Sticker   *MediaPayload `json:"sticker,omitempty"`
    
    // Context
    ReplyTo   string `json:"reply_to,omitempty"`  // quoted message ID
    IsGroup   bool   `json:"is_group,omitempty"`
    GroupName string `json:"group_name,omitempty"`
    PushName  string `json:"push_name,omitempty"` // sender display name in groups
    
    // Interactive
    Interactive *InteractivePayload `json:"interactive,omitempty"`
    
    // Ack/receipt
    Status     string `json:"status,omitempty"`
    AckType    string `json:"ack_type,omitempty"`
    
    // Presence
    Presence   string `json:"presence,omitempty"`
    LastSeen   int64  `json:"last_seen,omitempty"`
}

// MediaPayload handles the polymorphic image field:
// - Plain string: auto-download ON, no caption → {url: "base64_or_url"}
// - Object: {url: "...", caption: "..."} or {id: "...", ...}
type MediaPayload struct {
    URL         string `json:"url,omitempty"`
    ID          string `json:"id,omitempty"`
    MimeType    string `json:"mime_type,omitempty"`
    FileName    string `json:"file_name,omitempty"`
    Caption     string `json:"caption,omitempty"`
    Width       int    `json:"width,omitempty"`
    Height      int    `json:"height,omitempty"`
}
```

### Step 5.3 — HMAC verification

**File: `pkg/gowa/verify.go`** (NEW)

```go
package gowa

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifyWebhookSignature(rawBody []byte, signatureHeader string, secret string) bool {
    if len(signatureHeader) < 7 || signatureHeader[:7] != "sha256=" {
        return false
    }
    sig, err := hex.DecodeString(signatureHeader[7:])
    if err != nil {
        return false
    }
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(rawBody)
    return hmac.Equal(mac.Sum(nil), sig)
}
```

**Critical:** Must use `rawBody` bytes (from `r.RequestCtx.PostBody()`), NOT re-serialized JSON.

### Step 5.4 — Webhook handler

**File: `internal/handlers/gowa_webhook.go`** (NEW)

```go
package handlers

// GowaWebhookVerify handles GET /api/gowa/webhook (verification challenge)
func (a *App) GowaWebhookVerify(r *fastglue.Request) error {
    mode := string(r.QueryArgs().Peek("hub.mode"))
    token := string(r.QueryArgs().Peek("hub.verify_token"))
    challenge := string(r.QueryArgs().Peek("hub.challenge"))
    
    if mode == "subscribe" && token != "" {
        // Verify against all GOWA accounts' webhook tokens
        var accounts []models.WhatsAppAccount
        a.DB.Where("provider_type = ?", "gowa").Find(&accounts)
        for _, acc := range accounts {
            a.decryptAccountSecrets(&acc)
            if acc.WebhookVerifyToken == token {
                r.SetBodyString(challenge)
                return nil
            }
        }
    }
    return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid verification", nil, "")
}

// GowaWebhookHandler handles POST /api/gowa/webhook
func (a *App) GowaWebhookHandler(r *fastglue.Request) error {
    body := r.RequestCtx.PostBody()
    signature := string(r.RequestCtx.Request.Header.Peek("X-Gowa-Signature"))
    
    var payload gowa.WebhookPayload
    if err := json.Unmarshal(body, &payload); err != nil {
        return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid payload", nil, "")
    }
    
    // Look up account by device_id
    account, err := a.getGowaAccountByDeviceID(payload.DeviceID)
    if err != nil {
        a.Log.Warn("Unknown GOWA device", "device_id", payload.DeviceID)
        return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Unknown device", nil, "")
    }
    
    // Verify HMAC if signature present
    if signature != "" && account.GowaWebhookSecret != "" {
        if !gowa.VerifyWebhookSignature(body, signature, account.GowaWebhookSecret) {
            return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid signature", nil, "")
        }
    }
    
    // Normalize and route based on event type
    switch payload.Event {
    case "message":
        go a.processGowaMessage(account, &payload)
    case "ack":
        go a.processGowaAck(account, &payload)
    case "presence":
        go a.processGowaPresence(account, &payload)
    // ... other event types ...
    }
    
    r.SetStatusCode(fasthttp.StatusOK)
    return nil
}
```

### Step 5.5 — Message normalization

**File: `internal/handlers/gowa_webhook.go`** (continued)

The key challenge: convert GOWA's flat payload into whatomate's existing `IncomingMessage` / `processIncomingMessage` flow.

```go
func (a *App) processGowaMessage(account *models.WhatsAppAccount, payload *gowa.WebhookPayload) {
    p := &payload.Payload
    
    // Map to whatomate's internal message model
    incoming := IncomingMessage{
        From:           p.From,
        To:             p.To,
        WhatsAppMsgID:  p.ID,
        Type:           mapGowaType(p.Type),
        Timestamp:      time.Unix(p.Timestamp, 0),
        PushName:       p.PushName,
        ReplyToMsgID:   p.ReplyTo,
    }
    
    switch p.Type {
    case "text":
        incoming.Text = p.Text
    case "image":
        incoming.MediaType = "image"
        incoming.MediaID = resolveMediaID(p.Image)
        incoming.Caption = resolveCaption(p.Image)
    case "video":
        incoming.MediaType = "video"
        incoming.MediaID = resolveMediaID(p.Video)
        incoming.Caption = resolveCaption(p.Video)
    case "audio":
        incoming.MediaType = "audio"
        incoming.MediaID = resolveMediaID(p.Audio)
    case "document":
        incoming.MediaType = "document"
        incoming.MediaID = resolveMediaID(p.Document)
    case "sticker":
        incoming.MediaType = "sticker"
        incoming.MediaID = resolveMediaID(p.Sticker)
    case "interactive":
        incoming.Text = p.Interactive.Body
        incoming.InteractiveType = p.Interactive.Type
        incoming.ButtonID = p.Interactive.ID
    }
    
    // Feed into existing chatbot processor
    a.processIncomingMessage(context.Background(), account, incoming)
}
```

### Step 5.6 — Handle polymorphic image/media resolution

```go
// resolveMediaID handles GOWA's polymorphic media field:
// - If ID is set → use it (media was auto-downloaded by GOWA, accessible via /media?id=ID)
// - If URL is set → it's base64 or direct URL
func resolveMediaID(m *gowa.MediaPayload) string {
    if m == nil {
        return ""
    }
    if m.ID != "" {
        return m.ID
    }
    return m.URL
}

func resolveCaption(m *gowa.MediaPayload) string {
    if m == nil {
        return ""
    }
    return m.Caption
}
```

**Files touched:**
| File | Change |
|------|--------|
| `pkg/gowa/webhook.go` | NEW — GOWA webhook payload types |
| `pkg/gowa/verify.go` | NEW — HMAC verification |
| `internal/handlers/gowa_webhook.go` | NEW — GowaWebhookVerify, GowaWebhookHandler, processGowaMessage, normalization |
| `cmd/whatomate/main.go:504+` | ADD GOWA webhook routes |

---

## Phase 6 — Frontend + Account Management

**Goal:** Allow users to create/manage GOWA accounts from the UI.

### Step 6.1 — Account API updates

Add `provider_type` field to account creation/update endpoints. The frontend needs:

1. Provider type selector (Meta / GOWA) on account creation form
2. When GOWA is selected, show GOWA-specific fields (Base URL, Device ID, Webhook Secret) and hide Meta fields (App ID, Phone ID, Business ID, App Secret)
3. Disable Meta-only features (Templates, Flows, Catalog, Analytics) in the UI for GOWA accounts

### Step 6.2 — Account creation endpoint

**File: `internal/handlers/accounts.go`** (or wherever account CRUD lives)

```go
// In account creation handler:
account.ProviderType = r.QueryArgs().PeekString("provider_type")
if account.ProviderType == "" {
    account.ProviderType = "meta" // backward compatible default
}
if account.ProviderType == "gowa" {
    account.GowaBaseURL = r.QueryArgs().PeekString("gowa_base_url")
    account.GowaDeviceID = r.QueryArgs().PeekString("gowa_device_id")
    account.GowaWebhookSecret = crypto.EncryptField(encryptionKey, r.QueryArgs().PeekString("gowa_webhook_secret"))
} else {
    // Meta credential setup (existing logic)
}
```

### Step 6.3 — Frontend changes

In the React SPA (`web/src/`):
- Add provider type toggle to account creation modal
- Conditionally render Meta vs GOWA credential fields
- Hide/disable Meta-only UI sections (Templates tab, Flows, Catalog, Analytics) when `provider_type === "gowa"`

### Step 6.4 — Add GOWA-specific settings page

- Device connection status indicator (connected/disconnected/reconnecting)
- QR code display for device pairing (if GOWA supports it)
- Session list per device
- Auto-read receipt toggle

**Files touched:**
| File | Change |
|------|--------|
| Account handler files | ADD `provider_type` handling |
| React SPA components | ADD provider toggle, conditional fields |

---

## Phase 7 — Testing + Edge Cases

### Step 7.1 — Unit tests

| Test | File | What to verify |
|------|------|----------------|
| Meta adapter | `pkg/whatsapp/client_test.go` | `Name() == "meta"`, all `Capabilities() == true` |
| GOWA adapter | `pkg/gowa/client_test.go` | `Name() == "gowa"`, correct capabilities flags |
| GOWA text send | `pkg/gowa/messages_test.go` | Correct HTTP call to GOWA REST |
| GOWA image send | `pkg/gowa/messages_test.go` | Multipart upload, no separate upload step |
| Registry routing | `pkg/waprovider/registry_test.go` | Meta account → Meta provider, GOWA account → GOWA provider |
| Webhook verification | `pkg/gowa/verify_test.go` | HMAC correct/incorrect, edge cases |
| Webhook normalization | `internal/handlers/gowa_webhook_test.go` | All message types normalize correctly |
| Polymorphic media | `internal/handlers/gowa_webhook_test.go` | String vs object image field |
| Timestamp handling | `internal/handlers/gowa_webhook_test.go` | Payload-embedded vs top-level |

### Step 7.2 — Integration tests

- End-to-end: create GOWA account → send text → verify GOWA received → simulate webhook → verify message stored
- End-to-end: send image via GOWA → verify multipart format → simulate webhook with media ID → download media
- Provider fallback: if GOWA is down, verify error propagation (not silent Meta fallback)

### Step 7.3 — Edge cases

1. **GOWA auto-download polymorphism**: image field is `string` (base64) when auto-download ON and no caption; is `object` when caption exists or auto-download OFF. Test both.
2. **Timestamp inconsistency**: messages have `payload.timestamp`, other events have top-level `timestamp`. Handler must check both.
3. **Group messages**: `is_group=true` + `group_name` + `push_name` (sender). whatomate needs to handle group JIDs differently from individual phone numbers.
4. **Media URL vs ID**: GOWA may return base64 data URL or server-stored media ID. Download path must handle both.
5. **HMAC on raw bytes**: webhook verification MUST use `r.RequestCtx.PostBody()` before any JSON parsing, not re-serialized bytes.
6. **Per-device webhook routing**: GOWA v8.10.0 supports device-specific webhook URLs. whatomate should register each device with its own callback path: `/api/gowa/webhook/{device_id}`.
7. **GOWA reconnection**: when a GOWA device disconnects, it sends a `connection` event. whatomate should update account status and notify the frontend.

---

## Execution Order & Dependencies

```
Phase 0 (scaffold)
    ↓
Phase 1 (provider interface) ← HIGHEST RISK, touches most files
    ↓
Phase 2 (data model)        ← Independent of Phase 1 code, but needs Phase 1 types
    ↓
Phase 3 (GOWA client)       ← Needs Phase 1 interface + Phase 2 model
    ↓
Phase 4 (registry + routing) ← Needs Phase 3 client
    ↓
Phase 5 (webhook handler)   ← Needs Phase 2 model + Phase 4 registry
    ↓
Phase 6 (frontend + accounts) ← Needs Phase 2 model
    ↓
Phase 7 (testing)            ← Parallel with Phases 3-6, finalize after
```

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Phase 1 breaks existing Meta flow | **HIGH** | GOWA client implements same interface; Meta client adapts with thin wrappers. Run full test suite after Phase 1. |
| `io.Reader` media signature change breaks callers | **MEDIUM** | Keep original Meta methods unchanged; add `*FromReader` wrappers for the interface. Only the `sendFn` closure needs refactoring. |
| GOWA webhook payload edge cases (polymorphic media, timestamp inconsistency) | **MEDIUM** | Write exhaustive test cases before implementing handler. Reference `webhook-payload.md` examples. |
| GOWA REST API unavailability / breaking changes | **LOW** | GOWA is self-hosted. Pin to specific version. Add health check endpoint. |
| Database migration in production | **LOW** | New columns are nullable with defaults. AutoMigrate handles it. Backward compatible. |

## Estimated Effort

| Phase | Files Changed | New Files | Risk | Time Estimate |
|-------|--------------|------------|------|---------------|
| Phase 0 | 0 | 2 | None | 15 min |
| Phase 1 | 8-10 | 1 | HIGH | 4-6 hrs |
| Phase 2 | 3 | 0 | LOW | 1-2 hrs |
| Phase 3 | 0 | 7 | MEDIUM | 6-8 hrs |
| Phase 4 | 5 | 1 | MEDIUM | 3-4 hrs |
| Phase 5 | 1 | 3 | MEDIUM | 4-6 hrs |
| Phase 6 | 3-4 | 0 | LOW | 3-4 hrs |
| Phase 7 | 0 | 4-6 | LOW | 4-6 hrs |
| **Total** | **~20** | **~18** | | **~26-37 hrs** |

---

*This plan is designed for sequential execution. Each phase produces a compilable, testable state. Phase 1 is the critical gate — once the provider interface is extracted and Meta adapts cleanly, the rest is additive.*
