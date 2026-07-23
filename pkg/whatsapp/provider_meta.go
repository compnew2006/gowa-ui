package whatsapp

import (
	"context"
	"errors"
)

// ErrNotSupported is returned when a method is called on a provider that does
// not support the operation (e.g. SendTemplateMessage on GOWA).
var ErrNotSupported = errors.New("operation not supported by this provider")

// Capabilities advertises which optional features a provider implements.
// Handlers consult these flags before calling provider-specific methods and
// before rendering provider-specific UI sections.
type Capabilities struct {
	Templates       bool // SendTemplateMessage, SubmitTemplate, FetchTemplates, DeleteTemplate, ResumableUpload
	Flows           bool // SendFlowMessage and flow CRUD
	Calls           bool // voice_call interactive buttons, call permission, call events
	Catalog         bool // catalog and product CRUD
	BusinessProfile bool // GetBusinessProfile, UpdateBusinessProfile, UploadProfilePicture
	MediaUpload     bool // two-step upload-then-send (Meta) vs inline multipart (GOWA)
	Interactive     bool // interactive buttons, CTA URL, list messages
	AccountSetup    bool // SubscribeApp, ExchangeCodeForToken, RegisterPhoneNumber, embedded signup, token debug, WABA discovery
}

// Provider is the unified interface for WhatsApp messaging backends.
// Both the Meta Cloud API client (*Client) and the GOWA client implement this
// interface. The method signatures intentionally match the existing *Client
// methods 1:1 so the concrete Meta client satisfies the interface with no
// adapter wrappers. GOWA implements the same signatures, returning
// ErrNotSupported for Meta-only operations.
type Provider interface {
	// Capabilities returns the feature set supported by this provider.
	Capabilities() Capabilities

	// --- Core messaging (shared between Meta and GOWA) ---

	SendTextMessage(ctx context.Context, account *Account, rcpt Recipient, text string, replyToMsgID ...string) (string, error)
	SendImageMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, caption, replyMessageID string) (string, error)
	SendVideoMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, caption, replyMessageID string) (string, error)
	SendAudioMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, replyMessageID string) (string, error)
	SendDocumentMessage(ctx context.Context, account *Account, rcpt Recipient, mediaID, filename, caption, replyMessageID string) (string, error)
	UploadMedia(ctx context.Context, account *Account, data []byte, mimeType, filename string) (string, error)
	MarkMessageRead(ctx context.Context, account *Account, messageID string) error

	// --- Interactive messages ---

	SendInteractiveButtons(ctx context.Context, account *Account, rcpt Recipient, bodyText string, buttons []Button) (string, error)
	SendCTAURLButton(ctx context.Context, account *Account, rcpt Recipient, bodyText, buttonText, url string) (string, error)
	SendVoiceCallButton(ctx context.Context, account *Account, rcpt Recipient, bodyText, displayText string, ttlMinutes int, payload string) (string, error)

	// --- Media retrieval ---

	GetMediaURL(ctx context.Context, mediaID string, account *Account) (string, error)
	DownloadMedia(ctx context.Context, mediaURL string, accessToken string) ([]byte, error)

	// --- Template messages (Meta-only; GOWA returns ErrNotSupported) ---

	SendTemplateMessage(ctx context.Context, account *Account, rcpt Recipient, templateName, languageCode string, components []map[string]any) (string, error)
	SubmitTemplate(ctx context.Context, account *Account, template *TemplateSubmission) (string, error)
	FetchTemplates(ctx context.Context, account *Account) ([]MetaTemplate, error)
	DeleteTemplate(ctx context.Context, account *Account, templateName string) error
	ResumableUpload(ctx context.Context, account *Account, data []byte, mimeType, filename string) (string, error)

	// --- Flows (Meta-only) ---

	SendFlowMessage(ctx context.Context, account *Account, rcpt Recipient, flowID, headerText, bodyText, ctaText, flowToken, firstScreen string) (string, error)

	// --- Catalog (Meta-only) ---

	CreateCatalog(ctx context.Context, account *Account, name string) (string, error)
	ListCatalogs(ctx context.Context, account *Account) ([]CatalogInfo, error)
	DeleteCatalog(ctx context.Context, account *Account, catalogID string) error
	ListCatalogProducts(ctx context.Context, account *Account, catalogID string) ([]ProductInfo, error)
	CreateProduct(ctx context.Context, account *Account, catalogID string, product *ProductInput) (string, error)
	UpdateProduct(ctx context.Context, account *Account, productID string, product *ProductInput) error
	DeleteProduct(ctx context.Context, account *Account, productID string) error

	// --- Business profile (Meta-only) ---

	GetBusinessProfile(ctx context.Context, account *Account) (*BusinessProfile, error)
	UpdateBusinessProfile(ctx context.Context, account *Account, input BusinessProfileInput) error
	UploadProfilePicture(ctx context.Context, account *Account, fileData []byte, mimeType string) (string, error)

	// --- Account setup / embedded signup (Meta-only) ---

	ValidateCredentials(ctx context.Context, phoneID, businessID, accessToken, apiVersion string) (*CredentialsValidationResult, error)
	SubscribeApp(ctx context.Context, account *Account) error
	ExchangeCodeForToken(ctx context.Context, code, appID, appSecret, apiVersion string) (string, error)
	GetPhoneNumberInfo(ctx context.Context, phoneID, accessToken, apiVersion string) (*PhoneNumberInfo, error)
	RegisterPhoneNumber(ctx context.Context, phoneID, pin, accessToken, apiVersion string) error
	GetTokenDebugInfo(ctx context.Context, inputToken, accessToken string) (*TokenDebugInfo, error)
	GetSharedWABA(ctx context.Context, accessToken string) (*SharedWABAResponse, error)
	GetWABAPhoneNumbers(ctx context.Context, wabaID, accessToken string) (*WABAPhoneNumbersResponse, error)

	// --- Voice calling (Meta-only) ---

	SendCallPermissionRequest(ctx context.Context, account *Account, rcpt Recipient, bodyText string) (string, error)
	GetCallPermission(ctx context.Context, account *Account, userPhone string) (string, error)
}

// Compile-time assertion that the Meta Cloud API client satisfies Provider.
var _ Provider = (*Client)(nil)
