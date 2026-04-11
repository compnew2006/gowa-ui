package worker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/contactutil"
	"github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/internal/templateutil"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Worker processes jobs from the queue
type Worker struct {
	Config          *config.Config
	DB              *gorm.DB
	Redis           *redis.Client
	Log             logf.Logger
	WhatsApp        *whatsapp.Client
	MessageProvider provider.MessageProvider
	Consumer        queue.Consumer
	InboundConsumer queue.Consumer
	Publisher       *queue.Publisher
	License         *license.Service
}

// WorkerOptions configures which consumers a worker should start.
type WorkerOptions struct {
	CampaignOrganizationID uuid.UUID
	EnableCampaignConsumer bool
	EnableInboundMedia     bool
}

// Ensure Worker implements JobHandler interface
var _ queue.JobHandler = (*Worker)(nil)

type inboundMediaJobProcessor interface {
	ProcessInboundMediaJob(ctx context.Context, job *queue.InboundMediaJob) error
}

// New creates a new Worker instance
func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client, log logf.Logger, messageProvider provider.MessageProvider, licenseService *license.Service, opts WorkerOptions) (*Worker, error) {
	if !opts.EnableCampaignConsumer && !opts.EnableInboundMedia {
		opts.EnableCampaignConsumer = true
		opts.EnableInboundMedia = true
	}

	var (
		consumer        *queue.RedisConsumer
		inboundConsumer *queue.RedisConsumer
		err             error
	)

	if opts.EnableCampaignConsumer {
		if opts.CampaignOrganizationID != uuid.Nil {
			consumer, err = queue.NewOrganizationRedisConsumer(rdb, log, opts.CampaignOrganizationID)
		} else {
			consumer, err = queue.NewRedisConsumer(rdb, log)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer: %w", err)
		}
	}
	if opts.EnableInboundMedia {
		inboundConsumer, err = queue.NewRedisInboundMediaConsumer(rdb, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create inbound-media consumer: %w", err)
		}
	}

	publisher := queue.NewPublisher(rdb, log)

	return &Worker{
		Config:          cfg,
		DB:              db,
		Redis:           rdb,
		Log:             log,
		WhatsApp:        whatsapp.NewWithBaseURL(log, cfg.WhatsApp.BaseURL),
		MessageProvider: messageProvider,
		Consumer:        consumer,
		InboundConsumer: inboundConsumer,
		Publisher:       publisher,
		License:         licenseService,
	}, nil
}

// Run starts the worker and processes jobs until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	w.Log.Info("Worker starting")

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	startConsumer := func(name string, consumer queue.Consumer) {
		if consumer == nil {
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := consumer.Consume(runCtx, w); err != nil && !errors.Is(err, context.Canceled) {
				errCh <- fmt.Errorf("%s consumer error: %w", name, err)
			}
		}()
	}

	startConsumer("campaign", w.Consumer)
	startConsumer("inbound_media", w.InboundConsumer)

	var retErr error
	select {
	case <-ctx.Done():
		retErr = ctx.Err()
	case err := <-errCh:
		retErr = err
	}

	cancel()
	wg.Wait()

	if errors.Is(retErr, context.Canceled) {
		w.Log.Info("Worker stopped")
		return retErr
	}
	if retErr != nil {
		return retErr
	}

	w.Log.Info("Worker stopped")
	return nil
}

// WaitUntilOperational pauses queue consumption while the runtime is license-locked.
func (w *Worker) WaitUntilOperational(ctx context.Context) error {
	if w.License == nil {
		return nil
	}
	return w.License.WaitUntilOperational(ctx)
}

// HandleRecipientJob processes a single recipient message job
func (w *Worker) HandleRecipientJob(ctx context.Context, job *queue.RecipientJob) error {
	acquired, err := w.acquireRecipientLock(ctx, job.RecipientID)
	if err != nil {
		return err
	}
	if !acquired {
		w.Log.Warn("Skipping duplicate in-flight recipient job", "recipient_id", job.RecipientID, "campaign_id", job.CampaignID)
		return nil
	}
	defer w.releaseRecipientLock(ctx, job.RecipientID)

	var existingRecipient models.BulkMessageRecipient
	if err := w.DB.Where("id = ?", job.RecipientID).First(&existingRecipient).Error; err != nil {
		w.Log.Error("Failed to load recipient", "error", err, "recipient_id", job.RecipientID)
		return fmt.Errorf("failed to load recipient: %w", err)
	}
	if existingRecipient.Status != models.MessageStatusPending {
		w.Log.Info("Skipping already-processed recipient job",
			"recipient_id", job.RecipientID,
			"campaign_id", job.CampaignID,
			"status", existingRecipient.Status,
		)
		return nil
	}

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

	orgPolicy, err := w.loadOrganizationSendPolicy(job.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to load organization send policy: %w", err)
	}
	if w.isWhatsmeowProvider() && orgPolicy.CampaignDraftOnly {
		reason := "Campaign execution blocked by organization policy (POLICY_DRAFT_ONLY)"
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", reason)
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	// Get or create contact for this recipient
	contact, _, err := contactutil.GetOrCreateContact(w.DB, job.OrganizationID, job.PhoneNumber, job.RecipientName)
	if err != nil || contact == nil {
		w.Log.Error("Failed to get or create contact", "error", err, "phone", job.PhoneNumber)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "Failed to create contact")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil // Don't retry
	}

	if w.isWhatsmeowProvider() {
		if failureReason, validationErr := w.validateWhatsmeowCampaignInstance(job.OrganizationID, campaign.WhatsAppAccount); validationErr != nil {
			return fmt.Errorf("failed to validate campaign sender instance: %w", validationErr)
		} else if failureReason != "" {
			w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", failureReason)
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		if orgPolicy.ShouldEnforceInboundOnlyForSystemSends() {
			hasIncomingHistory, inboundErr := w.contactHasIncomingHistory(job.OrganizationID, contact.ID)
			if inboundErr != nil {
				return fmt.Errorf("failed to evaluate inbound history for contact: %w", inboundErr)
			}
			if !hasIncomingHistory {
				reason := "Message blocked by strict sending restrictions (POLICY_NO_INBOUND)"
				w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", reason)
				w.incrementCampaignCount(job.CampaignID, "failed_count")
				return nil
			}
		}
	}

	// Build recipient for sending
	templateBody := ""
	if campaign.Template != nil {
		templateBody = campaign.Template.BodyContent
	}
	mergedTemplateParams := w.resolveCampaignTemplateParams(
		ctx,
		job.OrganizationID,
		contact,
		job.PhoneNumber,
		job.RecipientName,
		templateBody,
		job.TemplateParams,
	)
	recipient := &models.BulkMessageRecipient{
		PhoneNumber:    job.PhoneNumber,
		RecipientName:  job.RecipientName,
		TemplateParams: mergedTemplateParams,
	}

	delayScopeKey := resolveCampaignDelayScopeKey(campaign.WhatsAppAccount, campaign.ID)
	if err := w.applyCampaignSendDelay(ctx, delayScopeKey, campaign.MinDelaySeconds, campaign.MaxDelaySeconds); err != nil {
		return fmt.Errorf("failed to apply campaign send delay: %w", err)
	}

	var (
		waMessageID       string
		sendErr           error
		messageInstanceID *uuid.UUID
	)

	if w.isWhatsmeowProvider() {
		if w.MessageProvider == nil {
			sendErr = fmt.Errorf("message provider is not configured")
			w.Log.Error("Failed to send message", "error", sendErr, "recipient", job.PhoneNumber)
			w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", sendErr.Error())
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		instanceID := strings.TrimSpace(campaign.WhatsAppAccount)
		if parsedInstanceID, parseErr := uuid.Parse(instanceID); parseErr == nil {
			messageInstanceID = &parsedInstanceID
		}

		waMessageID, sendErr = w.sendTemplateMessageViaProvider(ctx, instanceID, &campaign, campaign.Template, recipient)
	} else {
		// Get WhatsApp account
		var account models.WhatsAppAccount
		if err := w.DB.Where("name = ? AND organization_id = ?", campaign.WhatsAppAccount, job.OrganizationID).First(&account).Error; err != nil {
			w.Log.Error("Failed to load WhatsApp account", "error", err, "account_name", campaign.WhatsAppAccount)
			w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "WhatsApp account not found")
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil // Don't retry, mark as failed
		}
		if err := w.decryptAccountSecrets(&account); err != nil {
			w.Log.Error("Failed to decrypt WhatsApp account secrets", "error", err, "account_name", campaign.WhatsAppAccount)
			w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "Failed to decrypt WhatsApp account secrets")
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		waMessageID, sendErr = w.sendTemplateMessage(ctx, &account, campaign.Template, recipient, campaign.HeaderMediaID)
	}

	// Create Message record
	message := models.Message{
		OrganizationID:    job.OrganizationID,
		InstanceID:        messageInstanceID,
		WhatsAppAccount:   campaign.WhatsAppAccount,
		ContactID:         contact.ID,
		WhatsAppMessageID: waMessageID,
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeTemplate,
		TemplateParams:    recipient.TemplateParams,
		Metadata: models.JSONB{
			"campaign_id":    job.CampaignID.String(),
			"recipient_name": job.RecipientName,
		},
	}
	if campaign.Template != nil {
		message.TemplateName = campaign.Template.Name
		content := renderCampaignTemplateBody(campaign.Template.BodyContent, recipient.TemplateParams)
		message.Content = content
	}

	if sendErr != nil {
		w.Log.Error("Failed to send message", "error", sendErr, "recipient", job.PhoneNumber)
		message.Status = models.MessageStatusFailed
		message.ErrorMessage = sendErr.Error()
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", sendErr.Error())
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

// HandleInboundMediaJob processes a single inbound-media recovery job.
func (w *Worker) HandleInboundMediaJob(ctx context.Context, job *queue.InboundMediaJob) error {
	if job == nil {
		return queue.NewPermanentError(fmt.Errorf("inbound media job is nil"))
	}
	if job.MessageID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing message_id"))
	}
	if job.OrganizationID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing organization_id"))
	}
	if job.InstanceID == uuid.Nil {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing instance_id"))
	}
	if strings.TrimSpace(job.MediaKind) == "" {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing media_kind"))
	}
	if strings.TrimSpace(job.MediaPayloadBase64) == "" {
		return queue.NewPermanentError(fmt.Errorf("inbound media job missing media_payload_base64"))
	}

	var message models.Message
	if err := w.DB.WithContext(ctx).
		Select("id", "organization_id", "instance_id", "media_url").
		Where("id = ? AND organization_id = ?", job.MessageID, job.OrganizationID).
		First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return queue.NewPermanentError(fmt.Errorf("message %s not found in organization %s", job.MessageID, job.OrganizationID))
		}
		return fmt.Errorf("failed to load target message %s: %w", job.MessageID, err)
	}

	if strings.TrimSpace(message.MediaURL) != "" {
		w.Log.Debug("Skipping inbound media job because message already has media_url",
			"message_id", message.ID,
			"organization_id", job.OrganizationID,
		)
		return nil
	}

	if message.InstanceID == nil {
		return queue.NewPermanentError(fmt.Errorf("message %s has no instance_id", message.ID))
	}
	if *message.InstanceID != job.InstanceID {
		return queue.NewPermanentError(fmt.Errorf("message %s instance mismatch: message=%s job=%s", message.ID, message.InstanceID.String(), job.InstanceID.String()))
	}

	processor, ok := w.MessageProvider.(inboundMediaJobProcessor)
	if !ok || processor == nil {
		return queue.NewPermanentError(fmt.Errorf("message provider does not support inbound media recovery"))
	}

	if err := processor.ProcessInboundMediaJob(ctx, job); err != nil {
		return err
	}

	return nil
}

func (w *Worker) isWhatsmeowProvider() bool {
	return w.Config != nil && w.Config.WhatsApp.Provider == "whatsmeow"
}

// updateRecipientStatus updates the recipient's status in the database
func (w *Worker) updateRecipientStatus(recipientID uuid.UUID, status models.MessageStatus, waMessageID, errorMsg string) {
	updates := map[string]interface{}{
		"status":               status,
		"whats_app_message_id": waMessageID,
		"error_message":        errorMsg,
	}
	if status == models.MessageStatusSent {
		updates["sent_at"] = time.Now()
	}
	if err := w.DB.Model(&models.BulkMessageRecipient{}).Where("id = ?", recipientID).Updates(updates).Error; err != nil {
		w.Log.Error("Failed to update recipient status", "error", err, "recipient_id", recipientID, "status", status)
	}
}

// incrementCampaignCount increments a campaign counter atomically
func (w *Worker) incrementCampaignCount(campaignID uuid.UUID, column string) {
	if err := w.DB.Model(&models.BulkMessageCampaign{}).
		Where("id = ?", campaignID).
		Update(column, gorm.Expr(column+" + 1")).Error; err != nil {
		w.Log.Error("Failed to increment campaign count", "error", err, "campaign_id", campaignID, "column", column)
	}
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
	// Count pending recipients
	var pendingCount int64
	if err := w.DB.Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND status = ?", campaignID, models.MessageStatusPending).
		Count(&pendingCount).Error; err != nil {
		w.Log.Error("Failed to count pending campaign recipients", "error", err, "campaign_id", campaignID)
		return
	}

	// If no pending recipients, mark campaign as completed
	if pendingCount == 0 {
		now := time.Now()
		result := w.DB.Model(&models.BulkMessageCampaign{}).
			Where("id = ? AND status = ?", campaignID, models.CampaignStatusProcessing).
			Updates(map[string]interface{}{
				"status":       models.CampaignStatusCompleted,
				"completed_at": now,
			})
		if result.Error != nil {
			w.Log.Error("Failed to mark campaign as completed", "error", result.Error, "campaign_id", campaignID)
			return
		}
		if result.RowsAffected == 0 {
			// Another worker already completed or campaign no longer processing.
			return
		}

		var campaign models.BulkMessageCampaign
		if err := w.DB.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
			w.Log.Error("Failed to load completed campaign", "error", err, "campaign_id", campaignID)
			return
		}

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

// HandleContactRepairJob processes a direct-contact repair job.
func (w *Worker) HandleContactRepairJob(ctx context.Context, job *queue.ContactRepairJob) error {
	if job == nil {
		return nil
	}

	var contact models.Contact
	if err := w.DB.Where("id = ? AND organization_id = ?", job.ContactID, job.OrganizationID).First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return queue.NewPermanentError(fmt.Errorf("contact not found: %s", job.ContactID))
		}
		return err
	}

	if err := handlers.RepairDirectContactPhoneFromConversation(w.DB, &contact, job.ConversationID); err != nil {
		w.Log.Warn("Failed to repair direct contact phone", "contact_id", job.ContactID, "conversation_id", job.ConversationID, "error", err)
		return err
	}

	return nil
}

// sendTemplateMessage sends a template message via WhatsApp Cloud API
func (w *Worker) sendTemplateMessage(ctx context.Context, account *models.WhatsAppAccount, template *models.Template, recipient *models.BulkMessageRecipient, campaignHeaderMediaID string) (string, error) {
	waAccount := &whatsapp.Account{
		PhoneID:     account.PhoneID,
		BusinessID:  account.BusinessID,
		APIVersion:  account.APIVersion,
		AccessToken: account.AccessToken,
	}

	// Build template components with parameters
	var components []map[string]interface{}

	// Handle header component (for media templates)
	if template.HeaderType != "" && template.HeaderType != "TEXT" {
		// Use campaign's uploaded media ID if available
		if campaignHeaderMediaID != "" {
			headerParam := buildMediaParameter(template.HeaderType, "id", campaignHeaderMediaID)
			if headerParam != nil {
				components = append(components, map[string]interface{}{
					"type":       "header",
					"parameters": []map[string]interface{}{headerParam},
				})
			}
		} else if template.HeaderContent != "" {
			// Fall back to template's header content (URL)
			headerParam := buildMediaParameter(template.HeaderType, "link", template.HeaderContent)
			if headerParam != nil {
				components = append(components, map[string]interface{}{
					"type":       "header",
					"parameters": []map[string]interface{}{headerParam},
				})
			}
		}
	}

	// Resolve body parameters (supports both named and positional)
	resolvedParams := templateutil.ResolveParams(template.BodyContent, recipient.TemplateParams)
	if len(resolvedParams) > 0 {
		bodyParams := make([]map[string]interface{}, len(resolvedParams))
		for i, val := range resolvedParams {
			bodyParams[i] = map[string]interface{}{
				"type": "text",
				"text": val,
			}
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": bodyParams,
		})
	}

	return w.WhatsApp.SendTemplateMessage(ctx, waAccount, recipient.PhoneNumber, template.Name, template.Language, components)
}

func (w *Worker) sendTemplateMessageViaProvider(ctx context.Context, instanceID string, campaign *models.BulkMessageCampaign, template *models.Template, recipient *models.BulkMessageRecipient) (string, error) {
	if w.MessageProvider == nil {
		return "", fmt.Errorf("message provider is not configured")
	}
	sendCtx := provider.WithSkipTypingIndicator(ctx)

	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("campaign sender instance is required")
	}
	if _, err := uuid.Parse(instanceID); err != nil {
		return "", fmt.Errorf("invalid campaign sender instance")
	}
	if template == nil {
		return "", fmt.Errorf("campaign template not found")
	}

	body := strings.TrimSpace(renderCampaignTemplateBody(template.BodyContent, recipient.TemplateParams))
	if body == "" {
		body = fmt.Sprintf("[Campaign: %s]", template.DisplayName)
	}

	if campaign != nil && strings.TrimSpace(campaign.HeaderMediaLocalPath) != "" {
		mediaRef := strings.TrimSpace(campaign.HeaderMediaLocalPath)
		mediaFilename := strings.TrimSpace(campaign.HeaderMediaFilename)
		if mediaFilename == "" {
			mediaFilename = filepath.Base(mediaRef)
		}
		if mediaFilename == "" || mediaFilename == "." || mediaFilename == string(filepath.Separator) {
			mediaFilename = "attachment"
		}

		switch classifyCampaignMediaType(campaign.HeaderMediaMimeType, mediaFilename) {
		case "image":
			return w.MessageProvider.SendImage(sendCtx, instanceID, recipient.PhoneNumber, mediaRef, body)
		case "video":
			return w.MessageProvider.SendVideo(sendCtx, instanceID, recipient.PhoneNumber, mediaRef, body)
		case "audio":
			return w.MessageProvider.SendAudio(sendCtx, instanceID, recipient.PhoneNumber, mediaRef)
		default:
			return w.MessageProvider.SendDocument(sendCtx, instanceID, recipient.PhoneNumber, mediaRef, mediaFilename)
		}
	}

	return w.MessageProvider.SendText(sendCtx, instanceID, recipient.PhoneNumber, body)
}

func classifyCampaignMediaType(mimeType, filename string) string {
	normalizedMIME := strings.ToLower(strings.TrimSpace(mimeType))
	switch {
	case strings.HasPrefix(normalizedMIME, "image/"):
		return "image"
	case strings.HasPrefix(normalizedMIME, "video/"):
		return "video"
	case strings.HasPrefix(normalizedMIME, "audio/"):
		return "audio"
	case normalizedMIME != "":
		return "document"
	}

	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return "image"
	case ".mp4", ".3gp":
		return "video"
	case ".aac", ".m4a", ".mp3", ".ogg":
		return "audio"
	default:
		return "document"
	}
}

// buildMediaParameter creates a media parameter for WhatsApp template headers.
// keyName is "id" for Meta media IDs or "link" for external URLs.
func buildMediaParameter(headerType, keyName, value string) map[string]interface{} {
	var mediaType string
	switch headerType {
	case "IMAGE":
		mediaType = "image"
	case "VIDEO":
		mediaType = "video"
	case "DOCUMENT":
		mediaType = "document"
	default:
		return nil
	}
	return map[string]interface{}{
		"type": mediaType,
		mediaType: map[string]interface{}{
			keyName: value,
		},
	}
}

// decryptAccountSecrets decrypts encrypted account secrets for worker send paths.
func (w *Worker) decryptAccountSecrets(account *models.WhatsAppAccount) error {
	var key string
	allowLegacy := true
	if w.Config != nil {
		key = w.Config.App.EncryptionKey
		if w.Config.App.AllowLegacyEncryption != nil {
			allowLegacy = *w.Config.App.AllowLegacyEncryption
		}
	}
	if account.AccessToken != "" && !crypto.IsEncrypted(account.AccessToken) {
		return fmt.Errorf("access token is not encrypted")
	}
	if account.AppSecret != "" && !crypto.IsEncrypted(account.AppSecret) {
		return fmt.Errorf("app secret is not encrypted")
	}
	if dec, err := crypto.DecryptWithPolicy(account.AccessToken, key, allowLegacy); err != nil {
		return fmt.Errorf("failed to decrypt access token: %w", err)
	} else {
		account.AccessToken = dec
	}
	if dec, err := crypto.DecryptWithPolicy(account.AppSecret, key, allowLegacy); err != nil {
		return fmt.Errorf("failed to decrypt app secret: %w", err)
	} else {
		account.AppSecret = dec
	}
	return nil
}

// Close cleans up worker resources
func (w *Worker) Close() error {
	if w.Consumer != nil {
		if err := w.Consumer.Close(); err != nil {
			return err
		}
	}
	if w.InboundConsumer != nil {
		if err := w.InboundConsumer.Close(); err != nil {
			return err
		}
	}
	return nil
}
