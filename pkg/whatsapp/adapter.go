package whatsapp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
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
	// Meta reactions are handled inline by the contacts.go handler
	// (sendWhatsAppReaction). This is a no-op at the adapter level
	// since the handler already constructs the raw API call.
	// TODO: Move reaction sending logic from handler into this adapter.
	m.logger.Warn("MetaAdapter.SendReaction is not yet fully wired — reactions still use direct API call in handler")
	return nil
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
