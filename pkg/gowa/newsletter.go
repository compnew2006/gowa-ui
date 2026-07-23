package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// NewsletterMessage represents a message posted in a newsletter/channel.
type NewsletterMessage struct {
	ServerID       int            `json:"server_id"`
	MessageID      string         `json:"message_id"`
	Type           string         `json:"type"`
	Timestamp      string         `json:"timestamp"`
	ViewsCount     int            `json:"views_count"`
	ReactionCounts map[string]int `json:"reaction_counts"`
	Text           string         `json:"text"`
}

// UnfollowNewsletter unsubscribes from a newsletter/channel.
// GOWA endpoint: POST /newsletter/unfollow
func (c *Client) UnfollowNewsletter(ctx context.Context, deviceID, newsletterID string) error {
	body := map[string]any{"newsletter_id": newsletterID}
	_, err := c.doJSON(ctx, "POST", "/newsletter/unfollow", deviceID, body)
	return err
}

// GetNewsletterMessages fetches recent messages from a newsletter/channel.
// GOWA endpoint: GET /newsletter/messages?newsletter_id={id}&count={n}
func (c *Client) GetNewsletterMessages(ctx context.Context, deviceID, newsletterID string, count int) ([]NewsletterMessage, error) {
	if count <= 0 {
		count = 50
	}
	path := fmt.Sprintf("/newsletter/messages?newsletter_id=%s&count=%d", url.QueryEscape(newsletterID), count)
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results struct {
			Data []NewsletterMessage `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse newsletter messages response: %w", err)
	}
	return resp.Results.Data, nil
}
