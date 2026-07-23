package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AppStatus represents the overall GOWA application connection status.
type AppStatus struct {
	IsConnected bool   `json:"is_connected"`
	IsLoggedIn  bool   `json:"is_logged_in"`
	DeviceID    string `json:"device_id"`
	JID         string `json:"jid"`
}

// LoginWithCodeResponse contains the phone pairing code.
type LoginWithCodeResponse struct {
	PairCode string `json:"pair_code"`
}

// PasskeyStatus represents the passkey pairing state.
type PasskeyStatus struct {
	DeviceID      string          `json:"device_id"`
	Status        string          `json:"status"`
	Challenge     json.RawMessage `json:"challenge,omitempty"`
	Code          string          `json:"code,omitempty"`
	SkipHandoffUX bool            `json:"skip_handoff_ux"`
}

// WebAuthnAssertion is the body for POST /app/passkey/response.
type WebAuthnAssertion struct {
	ID       string `json:"id"`
	RawID    string `json:"rawId"`
	Type     string `json:"type"`
	Response struct {
		ClientDataJSON    string `json:"clientDataJSON"`
		AuthenticatorData string `json:"authenticatorData"`
		Signature         string `json:"signature"`
		UserHandle        string `json:"userHandle,omitempty"`
	} `json:"response"`
}

// LoginWithCode initiates a phone-code pairing (alternative to QR).
// GOWA endpoint: GET /app/login-with-code?phone={phone}
func (c *Client) LoginWithCode(ctx context.Context, deviceID, phone string) (*LoginWithCodeResponse, error) {
	path := fmt.Sprintf("/app/login-with-code?phone=%s", url.QueryEscape(phone))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results LoginWithCodeResponse `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse login-with-code response: %w", err)
	}
	return &resp.Results, nil
}

// GetPasskeyStatus retrieves the pending passkey pairing state.
// GOWA endpoint: GET /app/passkey
func (c *Client) GetPasskeyStatus(ctx context.Context, deviceID string) (*PasskeyStatus, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/app/passkey", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results PasskeyStatus `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse passkey status response: %w", err)
	}
	return &resp.Results, nil
}

// SubmitPasskeyResponse submits the WebAuthn assertion for passkey pairing.
// GOWA endpoint: POST /app/passkey/response
func (c *Client) SubmitPasskeyResponse(ctx context.Context, deviceID string, assertion WebAuthnAssertion) error {
	_, err := c.doJSON(ctx, "POST", "/app/passkey/response", deviceID, assertion)
	return err
}

// ConfirmPasskey confirms the passkey pairing code.
// GOWA endpoint: POST /app/passkey/confirm
func (c *Client) ConfirmPasskey(ctx context.Context, deviceID string) error {
	_, err := c.doJSON(ctx, "POST", "/app/passkey/confirm", deviceID, nil)
	return err
}

// GetAppStatus retrieves the overall application connection status.
// GOWA endpoint: GET /app/status
func (c *Client) GetAppStatus(ctx context.Context, deviceID string) (*AppStatus, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/app/status", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results AppStatus `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse app status response: %w", err)
	}
	return &resp.Results, nil
}
