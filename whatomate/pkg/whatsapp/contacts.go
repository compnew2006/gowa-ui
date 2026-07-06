package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ContactCheckRequest represents a WhatsApp Contact Check API request
type ContactCheckRequest struct {
	Blocking         string   `json:"blocking"`
	Contacts         []string `json:"contacts"`
	MessagingProduct string   `json:"messaging_product"`
}

// ContactCheckResponse represents a WhatsApp Contact Check API response
type ContactCheckResponse struct {
	Contacts []ContactCheckItem `json:"contacts"`
}

// ContactCheckItem represents a checked contact response item
type ContactCheckItem struct {
	Input  string `json:"input"`
	Status string `json:"status"` // valid or invalid
	WaID   string `json:"wa_id"`
}

// CheckContacts checks if a list of phone numbers are registered on WhatsApp using Meta's contacts API
func (c *Client) CheckContacts(ctx context.Context, account *Account, phones []string) ([]ContactCheckItem, error) {
	url := fmt.Sprintf("%s/%s/%s/contacts", c.getBaseURL(), account.APIVersion, account.PhoneID)
	
	payload := ContactCheckRequest{
		Blocking:         "wait",
		Contacts:         phones,
		MessagingProduct: "whatsapp",
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, url, payload, account.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to check contacts: %w", err)
	}

	var resp ContactCheckResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse contacts check response: %w", err)
	}

	return resp.Contacts, nil
}
