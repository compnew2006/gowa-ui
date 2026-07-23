package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Chat represents a single conversation returned by GET /chats.
type Chat struct {
	JID             string `json:"jid"`
	Name            string `json:"name"`
	LastMessageTime string `json:"last_message_time,omitempty"`
	EphemeralExpiry int    `json:"ephemeral_expiration,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	Archived        bool   `json:"archived,omitempty"`
}

// ListChatsOptions parameterizes ListChats. Zero values are omitted from the
// query string so GOWA applies its documented defaults (limit 25, offset 0).
type ListChatsOptions struct {
	Limit    int    // GOWA max is 100 per request; ListChats pages automatically up to MaxChats.
	Offset   int    // Starting offset for the first request.
	Search   string // Filter chats by name.
	Archived *bool  // nil = all, true = archived only, false = non-archived only.
}

// MaxChats caps the total number of chats ListChats will fetch across pages,
// guarding against an unbounded loop on a pathological server response.
const MaxChats = 1000

// chatPageLimit clamps the per-request page size to GOWA's documented maximum.
const chatPageLimit = 100

// ListChats retrieves the chat list for a device. It pages through the results
// (offset increments by the page size) until the reported total is reached or
// MaxChats is hit, then returns the aggregated slice alongside the server's
// reported total. Callers that want a single page can set Limit and Offset
// directly; ListChats still respects MaxChats as an upper bound.
//
// GOWA endpoint: GET /chats
func (c *Client) ListChats(ctx context.Context, deviceID string, opts ListChatsOptions) ([]Chat, int, error) {
	pageSize := opts.Limit
	if pageSize <= 0 || pageSize > chatPageLimit {
		pageSize = chatPageLimit
	}

	var all []Chat
	total := -1 // unknown until the first response
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	for {
		// Stop once we've fetched the reported total (or the safety cap).
		if total >= 0 && offset >= total {
			break
		}
		if len(all) >= MaxChats {
			break
		}

		rawBody, err := c.doRaw(ctx, "GET", buildChatsPath(opts.Search, pageSize, offset, opts.Archived), deviceID)
		if err != nil {
			return nil, 0, err
		}

		var resp struct {
			Results struct {
				Data       []Chat `json:"data"`
				Pagination struct {
					Limit  int `json:"limit"`
					Offset int `json:"offset"`
					Total  int `json:"total"`
				} `json:"pagination"`
			} `json:"results"`
		}
		if err := json.Unmarshal(rawBody, &resp); err != nil {
			return nil, 0, fmt.Errorf("parse chat list response: %w", err)
		}

		page := resp.Results.Data
		if len(page) == 0 {
			break
		}
		if total < 0 {
			total = resp.Results.Pagination.Total
			// If the server didn't report a total, fall back to "this is the last page".
			if total <= 0 {
				total = offset + len(page)
			}
		}
		all = append(all, page...)

		// If the server returned fewer than a full page, we've reached the end.
		if len(page) < pageSize {
			break
		}
		offset += pageSize
	}

	if total < 0 {
		total = len(all)
	}
	return all, total, nil
}

// buildChatsPath assembles the GET /chats query string from the options.
// Empty/zero options are omitted so GOWA applies its own defaults.
func buildChatsPath(search string, limit, offset int, archived *bool) string {
	q := url.Values{}
	if search != "" {
		q.Set("search", search)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if archived != nil {
		q.Set("archived", fmt.Sprintf("%t", *archived))
	}
	if encoded := q.Encode(); encoded != "" {
		return "/chats?" + encoded
	}
	return "/chats"
}

// ChatMessage is a single historical message returned by GET /chat/{jid}/messages.
type ChatMessage struct {
	ID         string `json:"id"`
	ChatJID    string `json:"chat_jid"`
	SenderJID  string `json:"sender_jid"`
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"` // RFC3339
	IsFromMe   bool   `json:"is_from_me"`
	MediaType  string `json:"media_type"`
	Filename   string `json:"filename"`
	URL        string `json:"url"`
	FileLength int64  `json:"file_length"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// ChatMessagesOptions parameterizes GetChatMessages.
type ChatMessagesOptions struct {
	Limit  int // max 100 per GOWA spec
	Offset int
	Search string
}

// GetChatMessages retrieves the message history for a specific chat from a
// device. It pages through results (offset increments by the page size) until
// the reported total is reached or MaxChats is hit. chatJID must be the full
// WhatsApp JID (e.g. "628123@s.whatsapp.net" or "groupid@g.us").
//
// GOWA endpoint: GET /chat/{chat_jid}/messages
func (c *Client) GetChatMessages(ctx context.Context, deviceID, chatJID string, opts ChatMessagesOptions) ([]ChatMessage, int, error) {
	pageSize := opts.Limit
	if pageSize <= 0 || pageSize > chatPageLimit {
		pageSize = chatPageLimit
	}

	encodedJID := url.PathEscape(chatJID)
	var all []ChatMessage
	total := -1
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	for {
		if total >= 0 && offset >= total {
			break
		}
		if len(all) >= MaxChats {
			break
		}

		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", pageSize))
		if offset > 0 {
			q.Set("offset", fmt.Sprintf("%d", offset))
		}
		if opts.Search != "" {
			q.Set("search", opts.Search)
		}
		path := fmt.Sprintf("/chat/%s/messages?%s", encodedJID, q.Encode())

		rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
		if err != nil {
			return nil, 0, err
		}

		var resp struct {
			Results struct {
				Data       []ChatMessage `json:"data"`
				Pagination struct {
					Limit  int `json:"limit"`
					Offset int `json:"offset"`
					Total  int `json:"total"`
				} `json:"pagination"`
			} `json:"results"`
		}
		if err := json.Unmarshal(rawBody, &resp); err != nil {
			return nil, 0, fmt.Errorf("parse chat messages response: %w", err)
		}

		page := resp.Results.Data
		if len(page) == 0 {
			break
		}
		if total < 0 {
			total = resp.Results.Pagination.Total
			if total <= 0 {
				total = offset + len(page)
			}
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		offset += pageSize
	}

	if total < 0 {
		total = len(all)
	}
	return all, total, nil
}
