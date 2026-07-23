package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DeviceInfo represents a registered GOWA device.
type DeviceInfo struct {
	ID            string    `json:"id"`
	PhoneNumber   string    `json:"phone_number,omitempty"`
	DisplayName   string    `json:"display_name"`
	State         string    `json:"state"`
	JID           string    `json:"jid"`
	CreatedAt     time.Time `json:"created_at"`
	WebhookURL    string    `json:"webhook_url,omitempty"`
	WebhookEvents string    `json:"webhook_events,omitempty"`
}

// DeviceStatus represents the connection state of a device.
type DeviceStatus struct {
	DeviceID    string `json:"device_id"`
	IsConnected bool   `json:"is_connected"`
	IsLoggedIn  bool   `json:"is_logged_in"`
}

// WebhookConfig represents the per-device webhook configuration.
type WebhookConfig struct {
	WebhookURL                string `json:"webhook_url"`
	WebhookSecret             string `json:"webhook_secret,omitempty"`
	WebhookEvents             string `json:"webhook_events,omitempty"`
	WebhookInsecureSkipVerify bool   `json:"webhook_insecure_skip_verify,omitempty"`
}

// LoginResponse represents a QR code login response.
type LoginResponse struct {
	QRDuration int    `json:"qr_duration"`
	QRLink     string `json:"qr_link"`
}

// ListDevices returns all registered devices.
// GOWA endpoint: GET /devices
func (c *Client) ListDevices(ctx context.Context) ([]DeviceInfo, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/devices", "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Results []DeviceInfo    `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse device list response: %w", err)
	}
	return resp.Results, nil
}

// CreateDevice registers a new device on the GOWA instance.
// GOWA endpoint: POST /devices
func (c *Client) CreateDevice(ctx context.Context, deviceID string, cfg WebhookConfig) (*DeviceInfo, error) {
	body := map[string]any{
		"device_id":                    deviceID,
		"webhook_url":                  cfg.WebhookURL,
		"webhook_secret":               cfg.WebhookSecret,
		"webhook_events":               cfg.WebhookEvents,
		"webhook_insecure_skip_verify": cfg.WebhookInsecureSkipVerify,
	}
	rawBody, err := c.doJSONRaw(ctx, "POST", "/devices", "", body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results DeviceInfo `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse create device response: %w", err)
	}
	return &resp.Results, nil
}

// DeleteDevice removes a device from the GOWA instance.
// GOWA endpoint: DELETE /devices/{device_id}
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	_, err := c.doRaw(ctx, "DELETE", fmt.Sprintf("/devices/%s", deviceID), "")
	return err
}

// GetDeviceStatus checks the connection state of a device.
// GOWA endpoint: GET /devices/{device_id}/status
func (c *Client) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	rawBody, err := c.doRaw(ctx, "GET", fmt.Sprintf("/devices/%s/status", deviceID), "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results DeviceStatus `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse device status response: %w", err)
	}
	return &resp.Results, nil
}

// LogoutDevice logs out a device (keeps the device slot).
// GOWA endpoint: POST /devices/{device_id}/logout
func (c *Client) LogoutDevice(ctx context.Context, deviceID string) error {
	_, err := c.doRaw(ctx, "POST", fmt.Sprintf("/devices/%s/logout", deviceID), "")
	return err
}

// ReconnectDevice triggers a reconnection for a device.
// GOWA endpoint: POST /devices/{device_id}/reconnect
func (c *Client) ReconnectDevice(ctx context.Context, deviceID string) error {
	_, err := c.doRaw(ctx, "POST", fmt.Sprintf("/devices/%s/reconnect", deviceID), "")
	return err
}

// GetDeviceWebhook retrieves the webhook config for a device.
// GOWA endpoint: GET /devices/{device_id}/webhook
func (c *Client) GetDeviceWebhook(ctx context.Context, deviceID string) (*WebhookConfig, error) {
	rawBody, err := c.doRaw(ctx, "GET", fmt.Sprintf("/devices/%s/webhook", deviceID), "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results WebhookConfig `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse webhook config response: %w", err)
	}
	return &resp.Results, nil
}

// SetDeviceWebhook configures the webhook for a device.
// GOWA endpoint: PATCH /devices/{device_id}/webhook
func (c *Client) SetDeviceWebhook(ctx context.Context, deviceID string, cfg WebhookConfig) (*WebhookConfig, error) {
	body := map[string]any{
		"webhook_url":                  cfg.WebhookURL,
		"webhook_secret":               cfg.WebhookSecret,
		"webhook_events":               cfg.WebhookEvents,
		"webhook_insecure_skip_verify": cfg.WebhookInsecureSkipVerify,
	}
	rawBody, err := c.doJSONRaw(ctx, "PATCH", fmt.Sprintf("/devices/%s/webhook", deviceID), "", body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results WebhookConfig `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse webhook config response: %w", err)
	}
	return &resp.Results, nil
}

// GetLoginQR retrieves a QR code for pairing a device.
// Uses the recommended GET /app/login with X-Device-Id header.
// GOWA endpoint: GET /app/login
func (c *Client) GetLoginQR(ctx context.Context, deviceID string) (*LoginResponse, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/app/login", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results LoginResponse `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse login QR response: %w", err)
	}
	return &resp.Results, nil
}
