package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/contactutil"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/internal/queue"
	"github.com/compnew2006/gowa-ui/internal/templateutil"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Worker processes jobs from the queue
type Worker struct {
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Log    logf.Logger
	// WARegistry resolves the GOWA provider per account.
	WARegistry *whatsapp.Registry
	Consumer   *queue.RedisConsumer
	Publisher  *queue.Publisher
}

// Ensure Worker implements JobHandler interface
var _ queue.JobHandler = (*Worker)(nil)

// New creates a new Worker instance. The registry resolves the GOWA
// provider per account.
func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client, log logf.Logger, registry *whatsapp.Registry) (*Worker, error) {
	consumer, err := queue.NewRedisConsumer(rdb, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	publisher := queue.NewPublisher(rdb, log)

	return &Worker{
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		Log:        log,
		WARegistry: registry,
		Consumer:   consumer,
		Publisher:  publisher,
	}, nil
}

// Run starts the worker and processes jobs until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	w.Log.Info("Worker starting")

	err := w.Consumer.Consume(ctx, w)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("consumer error: %w", err)
	}

	w.Log.Info("Worker stopped")
	return nil
}

// HandleRecipientJob processes a single recipient message job
func (w *Worker) HandleRecipientJob(ctx context.Context, job *queue.RecipientJob) error {
	// Check if campaign is still active before sending
	var campaign models.BulkMessageCampaign
	if err := w.DB.Where("id = ?", job.CampaignID).Preload("Template").First(&campaign).Error; err != nil {
		w.Log.Error("Failed to load campaign", "error", err, "campaign_id", job.CampaignID)
		return fmt.Errorf("failed to load campaign: %w", err)
	}

	// Skip if campaign is paused or cancelled
	if campaign.Status == models.CampaignStatusPaused || campaign.Status == models.CampaignStatusCancelled {
		w.Log.Info("Campaign not active, skipping recipient", "campaign_id", job.CampaignID, "status", campaign.Status, "recipient_id", job.RecipientID)
		return nil // Not an error, just skip
	}

	// Atomically claim the recipient (pending→sending). A paused-then-
	// restarted campaign leaves the original jobs unread in the Redis Stream
	// while StartCampaign enqueues a second copy — without this claim both
	// jobs would send. RowsAffected==0 means another job already claimed (or
	// finished) this recipient, so this one is a duplicate and must skip.
	claim := w.DB.Model(&models.BulkMessageRecipient{}).
		Where("id = ? AND status = ?", job.RecipientID, models.MessageStatusPending).
		Update("status", models.MessageStatusSending)
	if claim.Error != nil {
		w.Log.Error("Failed to claim recipient", "error", claim.Error, "recipient_id", job.RecipientID)
		return fmt.Errorf("failed to claim recipient: %w", claim.Error)
	}
	if claim.RowsAffected == 0 {
		w.Log.Info("Recipient already claimed by another job, skipping duplicate", "campaign_id", job.CampaignID, "recipient_id", job.RecipientID)
		return nil
	}

	// Get WhatsApp account
	var account models.WhatsAppAccount
	if err := w.DB.Where("name = ? AND organization_id = ?", campaign.WhatsAppAccount, job.OrganizationID).First(&account).Error; err != nil {
		w.Log.Error("Failed to load WhatsApp account", "error", err, "account_name", campaign.WhatsAppAccount)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "WhatsApp account not found")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil // Don't retry, mark as failed
	}
	w.decryptAccountSecrets(&account)

	// Get or create contact for this recipient
	contact, _, err := contactutil.GetOrCreateContact(w.DB, job.OrganizationID, job.PhoneNumber, job.RecipientName)
	if err != nil || contact == nil {
		w.Log.Error("Failed to get or create contact", "error", err, "phone", job.PhoneNumber)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "Failed to create contact")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil // Don't retry
	}

	// Check marketing opt-out
	if contact.MarketingOptOut && campaign.Template != nil && strings.EqualFold(campaign.Template.Category, "MARKETING") {
		w.Log.Info("Skipping marketing message for opted-out contact", "contact_id", contact.ID, "phone", job.PhoneNumber)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "Contact opted out of marketing messages")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	// Build recipient for sending
	recipient := &models.BulkMessageRecipient{
		PhoneNumber:    job.PhoneNumber,
		RecipientName:  job.RecipientName,
		TemplateParams: job.TemplateParams,
		HeaderParams:   job.HeaderParams,
	}

	// Pace the send: WhatsApp flags accounts that burst, so reserve a slot
	// from the account's per-minute budget before touching the provider.
	// No-ops when pacing is not configured (0 = unlimited, historical behavior).
	w.paceCampaignSend(ctx, job.OrganizationID, account.Name, w.accountPacePerMinute(account.Settings))

	// Send template message
	waMessageID, err := w.sendTemplateMessage(ctx, &account, campaign.Template, recipient, &campaign)

	// Create Message record
	message := models.Message{
		OrganizationID:    job.OrganizationID,
		WhatsAppAccount:   campaign.WhatsAppAccount,
		ContactID:         contact.ID,
		WhatsAppMessageID: waMessageID,
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeTemplate,
		TemplateParams:    job.TemplateParams,
		Metadata: models.JSONB{
			"campaign_id":    job.CampaignID.String(),
			"recipient_name": job.RecipientName,
		},
	}
	if campaign.Template != nil {
		message.TemplateName = campaign.Template.Name
		content := templateutil.ReplaceWithJSONBParams(campaign.Template.BodyContent, campaign.Template.BodyContent, job.TemplateParams)
		message.Content = content
		// Store campaign header media so it renders in the chat bubble
		if campaign.HeaderMediaLocalPath != "" {
			message.MediaURL = campaign.HeaderMediaLocalPath
			message.MediaMimeType = campaign.HeaderMediaMimeType
		}
	}

	if err != nil {
		w.Log.Error("Failed to send message", "error", err, "recipient", job.PhoneNumber)
		message.Status = models.MessageStatusFailed
		message.ErrorMessage = err.Error()
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", err.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
	} else {
		w.Log.Info("Message sent", "recipient", job.PhoneNumber, "message_id", waMessageID)
		message.Status = models.MessageStatusSent
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusSent, waMessageID, "")
		w.incrementCampaignCount(job.CampaignID, "sent_count")
	}

	// Save message record
	if err := w.DB.Create(&message).Error; err != nil {
		w.Log.Error("Failed to save message", "error", err, "recipient", job.PhoneNumber)
	}

	// Check if campaign is complete (all recipients processed)
	w.checkCampaignCompletion(ctx, job.CampaignID, job.OrganizationID)

	return nil
}

// updateRecipientStatus updates the recipient's status in the database
func (w *Worker) updateRecipientStatus(recipientID uuid.UUID, status models.MessageStatus, waMessageID, errorMsg string) {
	updates := map[string]any{
		"status":               status,
		"whats_app_message_id": waMessageID,
	}
	if status == models.MessageStatusSent {
		updates["sent_at"] = time.Now()
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	w.DB.Model(&models.BulkMessageRecipient{}).Where("id = ?", recipientID).Updates(updates)
}

// incrementCampaignCount increments a campaign counter atomically
func (w *Worker) incrementCampaignCount(campaignID uuid.UUID, column string) {
	w.DB.Model(&models.BulkMessageCampaign{}).
		Where("id = ?", campaignID).
		Update(column, gorm.Expr(column+" + 1"))
}

// publishCampaignStats publishes campaign stats for real-time updates
func (w *Worker) publishCampaignStats(ctx context.Context, campaignID, organizationID uuid.UUID) {
	var campaign models.BulkMessageCampaign
	if err := w.DB.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		return
	}

	_ = w.Publisher.PublishCampaignStats(ctx, &queue.CampaignStatsUpdate{
		CampaignID:     campaignID.String(),
		OrganizationID: organizationID,
		Status:         campaign.Status,
		SentCount:      campaign.SentCount,
		DeliveredCount: campaign.DeliveredCount,
		ReadCount:      campaign.ReadCount,
		FailedCount:    campaign.FailedCount,
	})
}

// checkCampaignCompletion checks if all recipients are processed and marks campaign as completed
func (w *Worker) checkCampaignCompletion(ctx context.Context, campaignID, organizationID uuid.UUID) {
	// Count recipients still awaiting an outcome. "sending" is the worker's
	// in-flight claim state — the campaign is only complete once no recipient
	// is pending OR being sent.
	var pendingCount int64
	w.DB.Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND status IN (?, ?)", campaignID, models.MessageStatusPending, models.MessageStatusSending).
		Count(&pendingCount)

	// If no pending recipients, mark campaign as completed
	if pendingCount == 0 {
		var campaign models.BulkMessageCampaign
		if err := w.DB.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
			return
		}

		// Only complete if currently processing
		if campaign.Status != models.CampaignStatusProcessing {
			return
		}

		now := time.Now()
		w.DB.Model(&campaign).Updates(map[string]any{
			"status":       models.CampaignStatusCompleted,
			"completed_at": now,
		})

		w.Log.Info("Campaign completed", "campaign_id", campaignID, "sent", campaign.SentCount, "failed", campaign.FailedCount)

		// Publish completion status
		_ = w.Publisher.PublishCampaignStats(ctx, &queue.CampaignStatsUpdate{
			CampaignID:     campaignID.String(),
			OrganizationID: organizationID,
			Status:         models.CampaignStatusCompleted,
			SentCount:      campaign.SentCount,
			DeliveredCount: campaign.DeliveredCount,
			ReadCount:      campaign.ReadCount,
			FailedCount:    campaign.FailedCount,
		})
	} else {
		// Publish current stats
		w.publishCampaignStats(ctx, campaignID, organizationID)
	}
}

// sendTemplateMessage renders the template locally and sends it via GOWA as
// text / media-with-caption / interactive buttons, using the shared render
// engine in internal/templateutil (the same path as the chat template send in
// internal/handlers).
func (w *Worker) sendTemplateMessage(ctx context.Context, account *models.WhatsAppAccount, template *models.Template, recipient *models.BulkMessageRecipient, campaign *models.BulkMessageCampaign) (string, error) {
	if template == nil {
		return "", fmt.Errorf("campaign template not found")
	}

	bodyParams := templateutil.ResolveNamedParams(template.BodyContent, recipient.TemplateParams)

	// TEXT-header variables prefer recipient.HeaderParams (populated by
	// AddRecipients) and fall back to TemplateParams for legacy recipient
	// rows persisted before HeaderParams existed.
	headerParams := make(map[string]string)
	for _, name := range templateutil.ExtParamNames(template.HeaderContent) {
		if raw, ok := recipient.HeaderParams[name]; ok {
			headerParams[name] = fmt.Sprintf("%v", raw)
		} else if raw, ok := recipient.TemplateParams[name]; ok {
			headerParams[name] = fmt.Sprintf("%v", raw)
		}
	}

	return templateutil.SendRenderedTemplate(ctx, templateutil.SendRequest{
		Provider:  w.resolveProvider(account),
		Account:   account.ToWAAccount(),
		Recipient: whatsapp.Recipient{Phone: recipient.PhoneNumber},
		Template:  template,
		Params: templateutil.TemplateParams{
			BodyParams:   bodyParams,
			HeaderParams: headerParams,
		},
		Media: templateutil.TemplateMedia{
			ID:       campaign.HeaderMediaID,
			MimeType: campaign.HeaderMediaMimeType,
			Filename: campaign.HeaderMediaFilename,
			// Campaign header media lives on local disk; load it lazily so
			// the file is only read when an upload is actually needed.
			Load: func() []byte {
				if campaign.HeaderMediaLocalPath == "" {
					return nil
				}
				basePath := "./media"
				if w.Config != nil && w.Config.Storage.LocalPath != "" {
					basePath = w.Config.Storage.LocalPath
				}
				if fileData, err := os.ReadFile(filepath.Join(basePath, campaign.HeaderMediaLocalPath)); err == nil {
					return fileData
				}
				return nil
			},
		},
	})
}

// resolveProvider returns the WhatsApp provider (GOWA) for the given account.
func (w *Worker) resolveProvider(account *models.WhatsAppAccount) whatsapp.Provider {
	var waAccount *whatsapp.Account
	if account != nil {
		waAccount = account.ToWAAccount()
	}
	return w.WARegistry.Get(waAccount)
}

// decryptAccountSecrets decrypts the encrypted secrets on a WhatsApp account.
func (w *Worker) decryptAccountSecrets(account *models.WhatsAppAccount) {
	var key string
	if w.Config != nil {
		key = w.Config.App.EncryptionKey
	}
	account.DecryptSecrets(key)
}

// Close cleans up worker resources
func (w *Worker) Close() error {
	if w.Consumer != nil {
		return w.Consumer.Close()
	}
	return nil
}
