package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Chat represents a conversation summary.
type Chat struct {
	JID                 string `json:"jid"`
	Name                string `json:"name"`
	LastMessageTime     string `json:"last_message_time"`
	EphemeralExpiration int    `json:"ephemeral_expiration"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
	Archived            bool   `json:"archived"`
}

// ChatPagination represents pagination metadata for chat listing.
type ChatPagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// ChatMessage represents a single message in a chat's history.
type ChatMessage struct {
	ID         string `json:"id"`
	ChatJID    string `json:"chat_jid"`
	SenderJID  string `json:"sender_jid"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
	IsFromMe   bool   `json:"is_from_me"`
	MediaType  string `json:"media_type"`
	Filename   string `json:"filename,omitempty"`
	URL        string `json:"url,omitempty"`
	FileLength int64  `json:"file_length,omitempty"`
}

// ChatListParams controls chat listing pagination and filtering.
type ChatListParams struct {
	Limit    int
	Offset   int
	Search   string
	HasMedia bool
	Archived *bool // nil = all, true = archived only, false = unarchived only
}

// ListChats returns a paginated list of all chats.
// GOWA endpoint: GET /chats
func (c *Client) ListChats(ctx context.Context, deviceID string, params ChatListParams) ([]Chat, *ChatPagination, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", params.Offset))
	}
	if params.Search != "" {
		q.Set("search", params.Search)
	}
	if params.HasMedia {
		q.Set("has_media", "true")
	}
	if params.Archived != nil {
		q.Set("archived", fmt.Sprintf("%v", *params.Archived))
	}

	path := "/chats"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, nil, err
	}
	var resp struct {
		Results struct {
			Data       []Chat         `json:"data"`
			Pagination ChatPagination `json:"pagination"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse chat list response: %w", err)
	}
	return resp.Results.Data, &resp.Results.Pagination, nil
}

// ChatHistoryParams controls chat message history retrieval.
type ChatHistoryParams struct {
	Limit     int
	Offset    int
	StartTime string // RFC3339
	EndTime   string // RFC3339
	MediaOnly bool
	IsFromMe  *bool
	Search    string
}

// GetChatHistory returns paginated message history for a chat.
// GOWA endpoint: GET /chat/{chat_jid}/messages
func (c *Client) GetChatHistory(ctx context.Context, deviceID, chatJID string, params ChatHistoryParams) ([]ChatMessage, *ChatPagination, error) {
	q := url.Values{}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", params.Offset))
	}
	if params.StartTime != "" {
		q.Set("start_time", params.StartTime)
	}
	if params.EndTime != "" {
		q.Set("end_time", params.EndTime)
	}
	if params.MediaOnly {
		q.Set("media_only", "true")
	}
	if params.IsFromMe != nil {
		q.Set("is_from_me", fmt.Sprintf("%v", *params.IsFromMe))
	}
	if params.Search != "" {
		q.Set("search", params.Search)
	}

	path := fmt.Sprintf("/chat/%s/messages", url.PathEscape(chatJID))
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, nil, err
	}
	var resp struct {
		Results struct {
			Data       []ChatMessage  `json:"data"`
			Pagination ChatPagination `json:"pagination"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse chat history response: %w", err)
	}
	return resp.Results.Data, &resp.Results.Pagination, nil
}

// PinChat pins or unpins a chat.
// GOWA endpoint: POST /chat/{chat_jid}/pin
func (c *Client) PinChat(ctx context.Context, deviceID, chatJID string, pinned bool) error {
	path := fmt.Sprintf("/chat/%s/pin", url.PathEscape(chatJID))
	body := map[string]any{"pinned": pinned}
	_, err := c.doJSON(ctx, "POST", path, deviceID, body)
	return err
}

// SetDisappearingTimer sets the disappearing message timer for a chat.
// Valid values: 0, 86400, 604800, 7776000.
// GOWA endpoint: POST /chat/{chat_jid}/disappearing
func (c *Client) SetDisappearingTimer(ctx context.Context, deviceID, chatJID string, timerSeconds int) error {
	path := fmt.Sprintf("/chat/%s/disappearing", url.PathEscape(chatJID))
	body := map[string]any{"timer_seconds": timerSeconds}
	_, err := c.doJSON(ctx, "POST", path, deviceID, body)
	return err
}

// ArchiveChat archives or unarchives a chat.
// GOWA endpoint: POST /chat/{chat_jid}/archive
func (c *Client) ArchiveChat(ctx context.Context, deviceID, chatJID string, archived bool) error {
	path := fmt.Sprintf("/chat/%s/archive", url.PathEscape(chatJID))
	body := map[string]any{"archived": archived}
	_, err := c.doJSON(ctx, "POST", path, deviceID, body)
	return err
}
