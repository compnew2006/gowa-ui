package gowa

import (
	"context"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
)

// All methods below are Meta Cloud API features that GOWA does not support.
// They return whatsapp.ErrNotSupported so that capability checks or error
// handling in handlers can degrade gracefully.

// --- Template methods (Meta-only) ---

func (c *Client) SendTemplateMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, templateName, languageCode string, components []map[string]any) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) SubmitTemplate(ctx context.Context, account *whatsapp.Account, template *whatsapp.TemplateSubmission) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) FetchTemplates(ctx context.Context, account *whatsapp.Account) ([]whatsapp.MetaTemplate, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) DeleteTemplate(ctx context.Context, account *whatsapp.Account, templateName string) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) ResumableUpload(ctx context.Context, account *whatsapp.Account, data []byte, mimeType, filename string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

// --- Flows (Meta-only) ---

func (c *Client) SendFlowMessage(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, flowID, headerText, bodyText, ctaText, flowToken, firstScreen string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

// --- Catalog (Meta-only) ---

func (c *Client) CreateCatalog(ctx context.Context, account *whatsapp.Account, name string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) ListCatalogs(ctx context.Context, account *whatsapp.Account) ([]whatsapp.CatalogInfo, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) DeleteCatalog(ctx context.Context, account *whatsapp.Account, catalogID string) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) ListCatalogProducts(ctx context.Context, account *whatsapp.Account, catalogID string) ([]whatsapp.ProductInfo, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) CreateProduct(ctx context.Context, account *whatsapp.Account, catalogID string, product *whatsapp.ProductInput) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) UpdateProduct(ctx context.Context, account *whatsapp.Account, productID string, product *whatsapp.ProductInput) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) DeleteProduct(ctx context.Context, account *whatsapp.Account, productID string) error {
	return whatsapp.ErrNotSupported
}

// --- Business profile (Meta-only) ---

func (c *Client) GetBusinessProfile(ctx context.Context, account *whatsapp.Account) (*whatsapp.BusinessProfile, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) UpdateBusinessProfile(ctx context.Context, account *whatsapp.Account, input whatsapp.BusinessProfileInput) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) UploadProfilePicture(ctx context.Context, account *whatsapp.Account, fileData []byte, mimeType string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

// --- Account setup / embedded signup (Meta-only) ---

func (c *Client) ValidateCredentials(ctx context.Context, phoneID, businessID, accessToken, apiVersion string) (*whatsapp.CredentialsValidationResult, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) SubscribeApp(ctx context.Context, account *whatsapp.Account) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) ExchangeCodeForToken(ctx context.Context, code, appID, appSecret, apiVersion string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) GetPhoneNumberInfo(ctx context.Context, phoneID, accessToken, apiVersion string) (*whatsapp.PhoneNumberInfo, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) RegisterPhoneNumber(ctx context.Context, phoneID, pin, accessToken, apiVersion string) error {
	return whatsapp.ErrNotSupported
}

func (c *Client) GetTokenDebugInfo(ctx context.Context, inputToken, accessToken string) (*whatsapp.TokenDebugInfo, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) GetSharedWABA(ctx context.Context, accessToken string) (*whatsapp.SharedWABAResponse, error) {
	return nil, whatsapp.ErrNotSupported
}

func (c *Client) GetWABAPhoneNumbers(ctx context.Context, wabaID, accessToken string) (*whatsapp.WABAPhoneNumbersResponse, error) {
	return nil, whatsapp.ErrNotSupported
}

// --- Voice calling (Meta-only) ---

func (c *Client) SendCallPermissionRequest(ctx context.Context, account *whatsapp.Account, rcpt whatsapp.Recipient, bodyText string) (string, error) {
	return "", whatsapp.ErrNotSupported
}

func (c *Client) GetCallPermission(ctx context.Context, account *whatsapp.Account, userPhone string) (string, error) {
	return "", whatsapp.ErrNotSupported
}
