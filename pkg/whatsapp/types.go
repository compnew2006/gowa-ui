package whatsapp

import (
	"encoding/json"
	"fmt"
)

// Account represents WhatsApp Business Account credentials.
// Meta-specific fields (PhoneID, BusinessID, AppID, APIVersion, AccessToken)
// are populated when ProviderType == "meta". GOWA-specific fields
// (GowaBaseURL, GowaDeviceID) are populated when ProviderType == "gowa".
type Account struct {
	PhoneID     string
	BusinessID  string
	AppID       string
	APIVersion  string
	AccessToken string

	// ProviderType discriminates the backend: "meta" (default) or "gowa".
	ProviderType string
	// GOWA credentials — used by the GOWA provider to route API calls.
	GowaBaseURL  string // GOWA REST API base URL
	GowaDeviceID string // GOWA device UUID
}

// Button represents an interactive button
type Button struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"` // "reply" (default) or "url"
	URL   string `json:"url,omitempty"`  // URL for type="url" buttons
}

// MetaAPIResponse represents a successful API response from Meta
type MetaAPIResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// MetaAPIError represents an error response from Meta API
type MetaAPIError struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		ErrorUserMsg string `json:"error_user_msg"`
		ErrorData    struct {
			Details string `json:"details"`
		} `json:"error_data"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// ParseError attempts to parse respBody as a Meta API error. If successful,
// it returns a formatted error including code, message, details, and user message.
// If parsing fails, it returns a generic error with the status code and raw body.
func ParseMetaAPIError(statusCode int, respBody []byte) error {
	var apiErr MetaAPIError
	if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
		errMsg := fmt.Sprintf("API error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
		if apiErr.Error.ErrorData.Details != "" {
			errMsg += " - Details: " + apiErr.Error.ErrorData.Details
		}
		if apiErr.Error.ErrorUserMsg != "" {
			errMsg += " - " + apiErr.Error.ErrorUserMsg
		}
		return fmt.Errorf("%s", errMsg)
	}
	return fmt.Errorf("API returned status %d: %s", statusCode, string(respBody))
}

// TemplateResponse represents response from template submission
type TemplateResponse struct {
	ID string `json:"id"`
}

// MetaQualityScore represents quality score information from Meta
type MetaQualityScore struct {
	Score string `json:"score"`
}

// MetaTemplate represents a template fetched from Meta
type MetaTemplate struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Language      string              `json:"language"`
	Category      string              `json:"category"`
	Status        string              `json:"status"`
	QualityRating string              `json:"quality_rating,omitempty"`
	QualityScore  *MetaQualityScore   `json:"quality_score,omitempty"`
	Components    []TemplateComponent `json:"components"`
}

// TemplateComponent represents a component of a template
type TemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
	Example *TemplateExample `json:"example,omitempty"`
}

// TemplateButton represents a button in a template.
// FlowID uses json.Number because Meta returns it as a numeric ID.
type TemplateButton struct {
	Type           string      `json:"type"`
	Text           string      `json:"text"`
	URL            string      `json:"url,omitempty"`
	PhoneNumber    string      `json:"phone_number,omitempty"`
	Example        any         `json:"example,omitempty"`
	FlowID         json.Number `json:"flow_id,omitempty"`
	FlowAction     string      `json:"flow_action,omitempty"`
	NavigateScreen string      `json:"navigate_screen,omitempty"`
	OTPType        string      `json:"otp_type,omitempty"`       // "COPY_CODE", "ONE_TAP", "ZERO_TAP"
	AutofillText   string      `json:"autofill_text,omitempty"`  // For ONE_TAP OTP
	PackageName    string      `json:"package_name,omitempty"`   // For ONE_TAP/ZERO_TAP OTP
	SignatureHash  string      `json:"signature_hash,omitempty"` // For ONE_TAP/ZERO_TAP OTP
}

// TemplateExample represents example values for template variables
type TemplateExample struct {
	HeaderText   []string   `json:"header_text,omitempty"`
	HeaderHandle []string   `json:"header_handle,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
}

// TemplateListResponse represents response from fetching templates
type TemplateListResponse struct {
	Data []MetaTemplate `json:"data"`
}

// CatalogInfo represents a catalog from Meta API
type CatalogInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CatalogListResponse represents response from listing catalogs
type CatalogListResponse struct {
	Data []CatalogInfo `json:"data"`
}

// ProductInput represents input for creating/updating a product
type ProductInput struct {
	Name        string `json:"name"`
	Price       int64  `json:"price"` // Price in cents
	Currency    string `json:"currency"`
	URL         string `json:"url"`
	ImageURL    string `json:"image_url"`
	RetailerID  string `json:"retailer_id"` // SKU
	Description string `json:"description"`
}

// ProductInfo represents a product from Meta API
type ProductInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Price       string `json:"price"`
	Currency    string `json:"currency"`
	URL         string `json:"url"`
	ImageURL    string `json:"image_url"`
	RetailerID  string `json:"retailer_id"`
	Description string `json:"description"`
}

// ProductListResponse represents response from listing products
type ProductListResponse struct {
	Data []ProductInfo `json:"data"`
}

// ProductCreateResponse represents response from creating a product
type ProductCreateResponse struct {
	ID string `json:"id"`
}

// BusinessProfile represents the business profile of a phone number
type BusinessProfile struct {
	MessagingProduct string   `json:"messaging_product"`
	Address          string   `json:"address"`
	Description      string   `json:"description"`
	Vertical         string   `json:"vertical"`
	Email            string   `json:"email"`
	Websites         []string `json:"websites"`
	ProfilePicture   string   `json:"profile_picture_url"`
	About            string   `json:"about"` // Status text
}

// BusinessProfileInput represents the input for updating a business profile
type BusinessProfileInput struct {
	MessagingProduct     string   `json:"messaging_product"`
	Address              string   `json:"address,omitempty"`
	Description          string   `json:"description,omitempty"`
	Vertical             string   `json:"vertical,omitempty"`
	Email                string   `json:"email,omitempty"`
	Websites             []string `json:"websites,omitempty"`
	ProfilePictureHandle string   `json:"profile_picture_handle,omitempty"`
	About                string   `json:"about,omitempty"`
}
