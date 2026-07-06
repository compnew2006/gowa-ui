package whatsapp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// VerifyWebhook verifies the webhook challenge from Meta
func VerifyWebhook(mode, token, challenge, expectedToken string) (string, error) {
	if mode != "subscribe" {
		return "", fmt.Errorf("invalid mode: %s", mode)
	}
	if token != expectedToken {
		return "", fmt.Errorf("token mismatch")
	}
	return challenge, nil
}

// ParseWebhook parses the incoming webhook payload from Meta
func ParseWebhook(body []byte) (*WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}
	return &payload, nil
}

// ExtractMessages extracts all messages from a webhook payload
func (p *WebhookPayload) ExtractMessages() []ParsedMessage {
	var messages []ParsedMessage

	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			phoneNumberID := change.Value.Metadata.PhoneNumberID

			// Get contact name
			contactName := ""
			if len(change.Value.Contacts) > 0 {
				contactName = change.Value.Contacts[0].Profile.Name
			}

			for _, msg := range change.Value.Messages {
				parsed := ParsedMessage{
					From:          msg.From,
					ID:            msg.ID,
					Type:          msg.Type,
					PhoneNumberID: phoneNumberID,
					ContactName:   contactName,
				}

				// Parse timestamp
				if ts, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
					parsed.Timestamp = time.Unix(ts, 0)
				}

				// Extract text content based on message type
				switch msg.Type {
				case "text":
					if msg.Text != nil {
						parsed.Text = msg.Text.Body
					}
				case "interactive":
					if msg.Interactive != nil {
						switch msg.Interactive.Type {
						case "button_reply":
							if msg.Interactive.ButtonReply != nil {
								parsed.ButtonReplyID = msg.Interactive.ButtonReply.ID
								parsed.Text = msg.Interactive.ButtonReply.Title
							}
						case "list_reply":
							if msg.Interactive.ListReply != nil {
								parsed.ListReplyID = msg.Interactive.ListReply.ID
								parsed.Text = msg.Interactive.ListReply.Title
							}
						case "nfm_reply":
							if msg.Interactive.NFMReply != nil {
								parsed.Text = msg.Interactive.NFMReply.Body
							}
						}
					}
				case "image":
					if msg.Image != nil {
						parsed.MediaID = msg.Image.ID
						parsed.MediaMimeType = msg.Image.MimeType
						parsed.Caption = msg.Image.Caption
					}
				case "document":
					if msg.Document != nil {
						parsed.MediaID = msg.Document.ID
						parsed.MediaMimeType = msg.Document.MimeType
						parsed.Caption = msg.Document.Caption
					}
				case "audio":
					if msg.Audio != nil {
						parsed.MediaID = msg.Audio.ID
						parsed.MediaMimeType = msg.Audio.MimeType
					}
				case "video":
					if msg.Video != nil {
						parsed.MediaID = msg.Video.ID
						parsed.MediaMimeType = msg.Video.MimeType
						parsed.Caption = msg.Video.Caption
					}
				}

				messages = append(messages, parsed)
			}
		}
	}

	return messages
}

// ExtractStatuses extracts all status updates from a webhook payload
func (p *WebhookPayload) ExtractStatuses() []ParsedStatus {
	var statuses []ParsedStatus

	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			for _, status := range change.Value.Statuses {
				parsed := ParsedStatus{
					MessageID:   status.ID,
					Status:      status.Status,
					RecipientID: status.RecipientID,
				}

				// Parse timestamp
				if ts, err := strconv.ParseInt(status.Timestamp, 10, 64); err == nil {
					parsed.Timestamp = time.Unix(ts, 0)
				}

				// Extract error info if present
				if len(status.Errors) > 0 {
					parsed.ErrorCode = status.Errors[0].Code
					parsed.ErrorTitle = status.Errors[0].Title
					parsed.ErrorMsg = status.Errors[0].Message
				}

				statuses = append(statuses, parsed)
			}
		}
	}

	return statuses
}

// GetPhoneNumberID returns the phone number ID from the webhook payload
func (p *WebhookPayload) GetPhoneNumberID() string {
	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			if change.Value.Metadata.PhoneNumberID != "" {
				return change.Value.Metadata.PhoneNumberID
			}
		}
	}
	return ""
}

// HasMessages returns true if the webhook contains messages
func (p *WebhookPayload) HasMessages() bool {
	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			if len(change.Value.Messages) > 0 {
				return true
			}
		}
	}
	return false
}

// HasStatuses returns true if the webhook contains status updates
func (p *WebhookPayload) HasStatuses() bool {
	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			if len(change.Value.Statuses) > 0 {
				return true
			}
		}
	}
	return false
}
