package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// UserAvatar represents a user's profile picture URL.
type UserAvatar struct {
	URL  string `json:"url"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Newsletter represents a subscribed newsletter/channel.
type Newsletter struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Thread string `json:"thread,omitempty"`
}

// PrivacySettings represents the account's privacy settings.
type PrivacySettings struct {
	GroupAdd     string `json:"group_add"`
	LastSeen     string `json:"last_seen"`
	Status       string `json:"status"`
	Profile      string `json:"profile"`
	ReadReceipts string `json:"read_receipts"`
}

// BusinessProfileInfo represents a WhatsApp Business profile.
type BusinessProfileInfo struct {
	JID                   string   `json:"jid"`
	Email                 string   `json:"email"`
	Address               string   `json:"address"`
	Categories            []string `json:"categories"`
	ProfileOptions        string   `json:"profile_options"`
	BusinessHoursTimezone string   `json:"business_hours_timezone"`
}

// GetUserAvatar retrieves the profile picture for a phone number.
// GOWA endpoint: GET /user/avatar?phone={phone}
func (c *Client) GetUserAvatar(ctx context.Context, deviceID, phone string) (*UserAvatar, error) {
	path := fmt.Sprintf("/user/avatar?phone=%s", url.QueryEscape(phone))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results UserAvatar `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse user avatar response: %w", err)
	}
	return &resp.Results, nil
}

// GetMyNewsletters returns subscribed newsletters.
// GOWA endpoint: GET /user/my/newsletters
func (c *Client) GetMyNewsletters(ctx context.Context, deviceID string) ([]Newsletter, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/user/my/newsletters", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results struct {
			Data []Newsletter `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse newsletters response: %w", err)
	}
	return resp.Results.Data, nil
}

// GetPrivacySettings returns the account's privacy settings.
// GOWA endpoint: GET /user/my/privacy
func (c *Client) GetPrivacySettings(ctx context.Context, deviceID string) (*PrivacySettings, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/user/my/privacy", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results PrivacySettings `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse privacy settings response: %w", err)
	}
	return &resp.Results, nil
}

// GetUserBusinessProfile retrieves the business profile for a phone number.
// GOWA endpoint: GET /user/business-profile?phone={phone}
func (c *Client) GetUserBusinessProfile(ctx context.Context, deviceID, phone string) (*BusinessProfileInfo, error) {
	path := fmt.Sprintf("/user/business-profile?phone=%s", url.QueryEscape(phone))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results BusinessProfileInfo `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse business profile response: %w", err)
	}
	return &resp.Results, nil
}
