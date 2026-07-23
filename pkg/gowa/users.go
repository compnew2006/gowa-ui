package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// UserInfo represents user/account information from GOWA.
type UserInfo struct {
	VerifiedName string   `json:"verified_name"`
	Status       string   `json:"status"`
	PictureID    string   `json:"picture_id"`
	Devices      []string `json:"devices"`
}

// UserAvatar represents a user's profile picture URL.
type UserAvatar struct {
	URL  string `json:"url"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ContactEntry represents a contact in the device's contact list.
type ContactEntry struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

// GroupListItem represents a group the connected account belongs to.
type GroupListItem struct {
	JID               string `json:"JID"`
	Name              string `json:"Name"`
	IsLocked          bool   `json:"IsLocked"`
	IsAnnounce        bool   `json:"IsAnnounce"`
	IsEphemeral       bool   `json:"IsEphemeral"`
	DisappearingTimer int    `json:"DisappearingTimer"`
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

// GetUserInfo retrieves the connected account's profile info.
// GOWA endpoint: GET /user/info
func (c *Client) GetUserInfo(ctx context.Context, deviceID string) (*UserInfo, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/user/info", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results UserInfo `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse user info response: %w", err)
	}
	return &resp.Results, nil
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

// SetPushName changes the connected account's display name.
// GOWA endpoint: POST /user/pushname
func (c *Client) SetPushName(ctx context.Context, deviceID, pushName string) error {
	body := map[string]any{"push_name": pushName}
	_, err := c.doJSON(ctx, "POST", "/user/pushname", deviceID, body)
	return err
}

// SetUserAvatar changes the connected account's profile picture.
// photoData is the raw image bytes (JPEG recommended).
// GOWA endpoint: POST /user/avatar (multipart/form-data)
func (c *Client) SetUserAvatar(ctx context.Context, deviceID string, photoData []byte, filename string) error {
	if filename == "" {
		filename = "avatar.jpg"
	}
	_, err := c.doMultipart(ctx, "POST", "/user/avatar", deviceID, nil, "avatar", filename, photoData)
	return err
}

// CheckUser verifies whether a phone number is registered on WhatsApp.
// GOWA endpoint: GET /user/check?phone={phone}
func (c *Client) CheckUser(ctx context.Context, deviceID, phone string) (bool, error) {
	path := fmt.Sprintf("/user/check?phone=%s", url.QueryEscape(phone))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return false, err
	}
	var resp struct {
		Results struct {
			IsOnWhatsApp bool `json:"is_on_whatsapp"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return false, fmt.Errorf("parse user check response: %w", err)
	}
	return resp.Results.IsOnWhatsApp, err
}

// GetMyContacts returns the connected account's contact list.
// GOWA endpoint: GET /user/my/contacts
func (c *Client) GetMyContacts(ctx context.Context, deviceID string) ([]ContactEntry, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/user/my/contacts", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results struct {
			Data []ContactEntry `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse contacts response: %w", err)
	}
	return resp.Results.Data, nil
}

// GetMyGroups returns groups the connected account belongs to (max 500).
// GOWA endpoint: GET /user/my/groups
func (c *Client) GetMyGroups(ctx context.Context, deviceID string) ([]GroupListItem, error) {
	rawBody, err := c.doRaw(ctx, "GET", "/user/my/groups", deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results struct {
			Data []GroupListItem `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse groups response: %w", err)
	}
	return resp.Results.Data, nil
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
