package whatsapp

import (
	"context"
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// MetaAdapter implements the MessageProvider interface by wrapping the existing
// Meta Cloud API Client. This follows the Strangler Pattern — it delegates to
// existing Client methods without modifying them.
//
// For Meta, the "instanceID" parameter is interpreted as the WhatsApp account
// name (models.WhatsAppAccount.Name). The adapter resolves the account from
// the database to obtain credentials.
type MetaAdapter struct {
	client *Client
	db     *gorm.DB
	logger logf.Logger
}

// NewMetaAdapter creates a new MetaAdapter wrapping the existing Client.
func NewMetaAdapter(client *Client, db *gorm.DB, logger logf.Logger) *MetaAdapter {
	return &MetaAdapter{
		client: client,
		db:     db,
		logger: logger,
	}
}

// resolveAccount looks up a WhatsAppAccount by name (used as instanceID for Meta).
func (m *MetaAdapter) resolveAccount(instanceID string) (*Account, error) {
	var account models.WhatsAppAccount

	// Try to parse as UUID first (in case instanceID is a real UUID)
	if uid, err := uuid.Parse(instanceID); err == nil {
		if err := m.db.Where("id = ?", uid).First(&account).Error; err == nil {
			return m.toAccount(&account), nil
		}
	}

	// Fall back to name lookup
	if err := m.db.Where("name = ?", instanceID).First(&account).Error; err != nil {
		return nil, fmt.Errorf("WhatsApp account not found for identifier '%s': %w", instanceID, err)
	}

	return m.toAccount(&account), nil
}

func (m *MetaAdapter) toAccount(account *models.WhatsAppAccount) *Account {
	return &Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		AppID:       account.AppID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}
}

// SendText sends a text message via Meta Cloud API.
func (m *MetaAdapter) SendText(ctx context.Context, instanceID string, to string, text string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.SendTextMessage(ctx, account, to, text)
}

// SendImage sends an image message via Meta Cloud API.
func (m *MetaAdapter) SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	// For Meta, imageURL is expected to be a media ID (uploaded via UploadMedia).
	// The caller is responsible for uploading first.
	return m.client.SendImageMessage(ctx, account, to, imageURL, caption)
}

// SendDocument sends a document message via Meta Cloud API.
func (m *MetaAdapter) SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.SendDocumentMessage(ctx, account, to, docURL, filename, "")
}

// SendVideo sends a video message via Meta Cloud API.
func (m *MetaAdapter) SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.SendVideoMessage(ctx, account, to, videoURL, caption)
}

// SendAudio sends an audio message via Meta Cloud API.
func (m *MetaAdapter) SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.SendAudioMessage(ctx, account, to, audioURL)
}

// MarkRead marks a message as read via Meta Cloud API.
func (m *MetaAdapter) MarkRead(ctx context.Context, instanceID string, messageID string) error {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return err
	}
	return m.client.MarkMessageRead(ctx, account, messageID)
}

// SendReaction sends a reaction emoji via Meta Cloud API.
func (m *MetaAdapter) SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return err
	}

	// For Meta, we also need the recipient's phone number.
	// However, the SendReaction interface only provides messageID and emoji.
	// Since this adapter adheres to the MessageProvider interface, we might need
	// to resolve the message from the DB to get the recipient.
	// For now, we'll keep the previous behavior or implement the missing bits.
	// Actually, the handler SendReaction already has contact info.
	// Let's check if we can improve the interface or resolve it here.

	var msg models.Message
	if err := m.db.Where("whats_app_message_id = ?", messageID).First(&msg).Error; err != nil {
		return fmt.Errorf("message not found to send reaction: %w", err)
	}

	var contact models.Contact
	if err := m.db.Where("id = ?", msg.ContactID).First(&contact).Error; err != nil {
		return fmt.Errorf("contact not found to send reaction: %w", err)
	}

	return m.client.SendReactionMessage(ctx, account, contact.PhoneNumber, messageID, emoji)
}

// RevokeMessage is not supported for Meta provider in this adapter path.
func (m *MetaAdapter) RevokeMessage(ctx context.Context, instanceID string, messageID string) error {
	return fmt.Errorf("message revocation is not supported for the Meta provider adapter")
}

// GetMediaURL retrieves a media URL for a media ID via Meta Cloud API.
func (m *MetaAdapter) GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.GetMediaURL(ctx, mediaID, account)
}

// DownloadMedia downloads media content from a Meta CDN URL.
func (m *MetaAdapter) DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return nil, err
	}
	return m.client.DownloadMedia(ctx, mediaURL, account.AccessToken)
}

// UploadMedia uploads media to Meta's servers and returns a media ID.
func (m *MetaAdapter) UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error) {
	account, err := m.resolveAccount(instanceID)
	if err != nil {
		return "", err
	}
	return m.client.UploadMedia(ctx, account, data, mediaType, "upload")
}
