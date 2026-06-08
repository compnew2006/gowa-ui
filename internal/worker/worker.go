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
	waprovider "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	inboundMediaSelfHealInterval   = 5 * time.Minute
	inboundMediaSelfHealOlderThan  = 15 * time.Minute
	inboundMediaSelfHealBatchLimit = 250
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
	whatsmeowMgr     *waprovider.ConnectionManager
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
		consumer        queue.Consumer
		inboundConsumer queue.Consumer
		err             error
	)

	if opts.EnableCampaignConsumer {
		var redisConsumer *queue.RedisConsumer
		if opts.CampaignOrganizationID != uuid.Nil {
			redisConsumer, err = queue.NewOrganizationRedisConsumer(rdb, log, opts.CampaignOrganizationID)
		} else {
			redisConsumer, err = queue.NewRedisConsumer(rdb, log)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer: %w", err)
		}
		consumer = redisConsumer
	}
	if opts.EnableInboundMedia {
		var redisConsumer *queue.RedisConsumer
		redisConsumer, err = queue.NewRedisInboundMediaConsumer(rdb, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create inbound-media consumer: %w", err)
		}
		inboundConsumer = redisConsumer
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

func (w *Worker) SetWhatsmeowManager(mgr *waprovider.ConnectionManager) {
	w.whatsmeowMgr = mgr
}

// Run starts the worker and processes jobs until context is cancelled
func (w *Worker) Run(ctx context.Context) error {
	w.Log.Info("Worker starting")

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

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
	if w.shouldRunInboundMediaSelfHeal() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runInboundMediaSelfHealLoop(runCtx)
		}()
	}

	if w.Redis != nil && w.Consumer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runScheduledSendsPoller(runCtx)
		}()
	}

	if w.DB != nil && w.Redis != nil && w.Consumer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runSendingRecoveryLoop(runCtx)
		}()
	}

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

	var campaign models.BulkMessageCampaign
	if err := w.DB.Where("id = ?", job.CampaignID).Preload("Template").First(&campaign).Error; err != nil {
		w.Log.Error("Failed to load campaign", "error", err, "campaign_id", job.CampaignID)
		return fmt.Errorf("failed to load campaign: %w", err)
	}

	if campaign.Status == models.CampaignStatusPaused || campaign.Status == models.CampaignStatusCancelled {
		w.Log.Info("Campaign not active, skipping recipient", "campaign_id", job.CampaignID, "status", campaign.Status, "recipient_id", job.RecipientID)
		return nil
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

	if existingRecipient.RecipientType == models.RecipientTypeGroup {
		return w.handleGroupRecipientJob(ctx, job, existingRecipient, campaign)
	}

	contact, _, err := contactutil.GetOrCreateContact(w.DB, job.OrganizationID, job.PhoneNumber, job.RecipientName)
	if err != nil || contact == nil {
		w.Log.Error("Failed to get or create contact", "error", err, "phone", job.PhoneNumber)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", "Failed to create contact")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
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

	delayScopeKey := resolveCampaignDelayScopeKey(campaign.WhatsAppAccount, campaign.ID)
	delayDuration, err := w.computeCampaignDelayDuration(ctx, delayScopeKey, campaign.MinDelaySeconds, campaign.MaxDelaySeconds)
	if err != nil {
		return fmt.Errorf("failed to compute campaign send delay: %w", err)
	}

	if delayDuration > 0 && w.Redis != nil {
		sendAt := time.Now().Add(delayDuration)
		if err := w.scheduleRecipientSend(ctx, job, sendAt); err != nil {
			w.Log.Error("Failed to schedule send, falling back to immediate send", "error", err, "recipient_id", job.RecipientID)
		} else {
			if err := w.transitionRecipientToSending(ctx, job.RecipientID, uuid.New().String()); err != nil {
				w.Log.Warn("Recipient already being sent by another worker (scheduled send will be skipped at poll time)", "recipient_id", job.RecipientID, "error", err)
			}
			return nil
		}
	}

	if delayDuration > 0 && w.Redis == nil {
		if err := sleepWithContext(ctx, delayDuration); err != nil {
			return fmt.Errorf("campaign delay interrupted: %w", err)
		}
	}

	return w.executeRecipientSend(ctx, job)
}

func (w *Worker) executeRecipientSendWithAttempt(ctx context.Context, job *queue.RecipientJob, attemptID string) error {
	var existingRecipient models.BulkMessageRecipient
	if err := w.DB.Where("id = ?", job.RecipientID).First(&existingRecipient).Error; err != nil {
		return fmt.Errorf("failed to load recipient for attempt check: %w", err)
	}
	if existingRecipient.SendAttemptID != attemptID {
		w.Log.Info("Skipping recipient with mismatched attempt ID — recovery or another worker owns it",
			"recipient_id", job.RecipientID,
			"expected_attempt", attemptID,
			"actual_attempt", existingRecipient.SendAttemptID,
		)
		return nil
	}
	return w.executeRecipientSend(ctx, job)
}

func (w *Worker) executeRecipientSend(ctx context.Context, job *queue.RecipientJob) error {
	var existingRecipient models.BulkMessageRecipient
	if err := w.DB.Where("id = ?", job.RecipientID).First(&existingRecipient).Error; err != nil {
		w.Log.Error("Failed to load recipient for send", "error", err, "recipient_id", job.RecipientID)
		return fmt.Errorf("failed to load recipient: %w", err)
	}
	if existingRecipient.Status != models.MessageStatusPending && existingRecipient.Status != models.MessageStatusSending {
		w.Log.Info("Skipping already-processed recipient in executeRecipientSend",
			"recipient_id", job.RecipientID,
			"campaign_id", job.CampaignID,
			"status", existingRecipient.Status,
		)
		return nil
	}

	var campaign models.BulkMessageCampaign
	if err := w.DB.Where("id = ?", job.CampaignID).Preload("Template").First(&campaign).Error; err != nil {
		w.Log.Error("Failed to load campaign for send", "error", err, "campaign_id", job.CampaignID)
		return fmt.Errorf("failed to load campaign: %w", err)
	}

	if campaign.Status == models.CampaignStatusPaused || campaign.Status == models.CampaignStatusCancelled {
		w.Log.Info("Campaign no longer active at send time", "campaign_id", job.CampaignID, "status", campaign.Status)
		w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", "Campaign paused or cancelled")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	isGroup := existingRecipient.RecipientType == models.RecipientTypeGroup || job.RecipientType == models.RecipientTypeGroup

	if isGroup {
		return w.executeGroupRecipientSend(ctx, job, existingRecipient, campaign)
	}

	contact, _, err := contactutil.GetOrCreateContact(w.DB, job.OrganizationID, job.PhoneNumber, job.RecipientName)
	if err != nil || contact == nil {
		w.Log.Error("Failed to get or create contact for send", "error", err, "phone", job.PhoneNumber)
		w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", "Failed to create contact")
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

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

	var (
		waMessageID       string
		sendErr           error
		messageInstanceID *uuid.UUID
	)

	if w.isWhatsmeowProvider() {
		if w.MessageProvider == nil {
			sendErr = fmt.Errorf("message provider is not configured")
			w.Log.Error("Failed to send message", "error", sendErr, "recipient", job.PhoneNumber)
			w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", sendErr.Error())
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		instanceID := strings.TrimSpace(campaign.WhatsAppAccount)
		if parsedInstanceID, parseErr := uuid.Parse(instanceID); parseErr == nil {
			messageInstanceID = &parsedInstanceID
		}

		waMessageID, sendErr = w.sendTemplateMessageViaProvider(ctx, instanceID, &campaign, campaign.Template, recipient)
	} else {
		var account models.WhatsAppAccount
		if err := w.DB.Where("name = ? AND organization_id = ?", campaign.WhatsAppAccount, job.OrganizationID).First(&account).Error; err != nil {
			w.Log.Error("Failed to load WhatsApp account", "error", err, "account_name", campaign.WhatsAppAccount)
			w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", "WhatsApp account not found")
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}
		if err := w.decryptAccountSecrets(&account); err != nil {
			w.Log.Error("Failed to decrypt WhatsApp account secrets", "error", err, "account_name", campaign.WhatsAppAccount)
			w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", "Failed to decrypt WhatsApp account secrets")
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}

		waMessageID, sendErr = w.sendTemplateMessage(ctx, &account, campaign.Template, recipient, campaign.HeaderMediaID)
	}

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
		w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusFailed, "", sendErr.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
	} else {
		w.Log.Info("Message sent", "recipient", job.PhoneNumber, "message_id", waMessageID)
		message.Status = models.MessageStatusSent
		w.updateRecipientStatusConditional(job.RecipientID, existingRecipient.Status, models.MessageStatusSent, waMessageID, "")
		w.incrementCampaignCount(job.CampaignID, "sent_count")
	}

	if err := w.DB.Create(&message).Error; err != nil {
		w.Log.Error("Failed to save message", "error", err, "recipient", job.PhoneNumber)
	}

	w.checkCampaignCompletion(ctx, job.CampaignID, job.OrganizationID)
	return nil
}

func (w *Worker) executeGroupRecipientSend(ctx context.Context, job *queue.RecipientJob, recipient models.BulkMessageRecipient, campaign models.BulkMessageCampaign) error {
	groupJID := recipient.GroupJID
	if groupJID == "" {
		groupJID = job.GroupJID
	}

	if !contactutil.IsValidGroupJID(groupJID) {
		reason := fmt.Sprintf("Invalid group JID format: %s", groupJID)
		w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusFailed, "", reason)
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	if !w.isWhatsmeowProvider() {
		reason := "Group messaging is not supported by this provider (Meta Cloud API). Use a whatsmeow instance."
		w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusFailed, "", reason)
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	instanceID := strings.TrimSpace(campaign.WhatsAppAccount)

	if gp, ok := w.MessageProvider.(provider.GroupProvider); ok {
		if _, err := gp.VerifyGroupMembership(ctx, instanceID, groupJID); err != nil {
			reason := fmt.Sprintf("Group not found or inaccessible: %s", groupJID)
			w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusFailed, "", reason)
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}
	}

	recipient.PhoneNumber = groupJID

	templateBody := ""
	if campaign.Template != nil {
		templateBody = campaign.Template.BodyContent
	}
	mergedTemplateParams := w.resolveCampaignTemplateParams(
		ctx,
		job.OrganizationID,
		nil,
		groupJID,
		recipient.GroupName,
		templateBody,
		job.TemplateParams,
	)
	recipient.TemplateParams = mergedTemplateParams

	if w.MessageProvider == nil {
		sendErr := fmt.Errorf("message provider is not configured")
		w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusFailed, "", sendErr.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	waMessageID, sendErr := w.sendTemplateMessageViaProvider(ctx, instanceID, &campaign, campaign.Template, &recipient)

	var messageInstanceID *uuid.UUID
	if parsedInstanceID, parseErr := uuid.Parse(instanceID); parseErr == nil {
		messageInstanceID = &parsedInstanceID
	}

	message := models.Message{
		OrganizationID:    job.OrganizationID,
		InstanceID:        messageInstanceID,
		WhatsAppAccount:   campaign.WhatsAppAccount,
		WhatsAppMessageID: waMessageID,
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeTemplate,
		TemplateParams:    recipient.TemplateParams,
		Metadata: models.JSONB{
			"campaign_id":    job.CampaignID.String(),
			"recipient_name": job.RecipientName,
			"group_jid":      groupJID,
			"group_name":     recipient.GroupName,
		},
	}
	if campaign.Template != nil {
		message.TemplateName = campaign.Template.Name
		message.Content = renderCampaignTemplateBody(campaign.Template.BodyContent, recipient.TemplateParams)
	}

	if sendErr != nil {
		w.Log.Error("Failed to send group message", "error", sendErr, "group_jid", groupJID)
		message.Status = models.MessageStatusFailed
		message.ErrorMessage = sendErr.Error()
		w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusFailed, "", sendErr.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
	} else {
		w.Log.Info("Group message sent", "group_jid", groupJID, "message_id", waMessageID)
		message.Status = models.MessageStatusSent
		w.updateRecipientStatusConditional(job.RecipientID, recipient.Status, models.MessageStatusSent, waMessageID, "")
		w.incrementCampaignCount(job.CampaignID, "sent_count")
	}

	if err := w.DB.Create(&message).Error; err != nil {
		w.Log.Error("Failed to save group message", "error", err, "group_jid", groupJID)
	}

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

func (w *Worker) computeCampaignDelayDuration(ctx context.Context, delayScopeKey string, minDelaySeconds, maxDelaySeconds int) (time.Duration, error) {
	minDelaySeconds, maxDelaySeconds = normalizeCampaignDelaySeconds(minDelaySeconds, maxDelaySeconds)
	if minDelaySeconds == 0 && maxDelaySeconds == 0 {
		return 0, nil
	}

	gapMs, err := randomDelayMilliseconds(minDelaySeconds, maxDelaySeconds)
	if err != nil {
		return 0, err
	}
	if gapMs <= 0 {
		return 0, nil
	}

	if w.Redis == nil {
		return time.Duration(gapMs) * time.Millisecond, nil
	}

	nowMs := time.Now().UnixMilli()
	ttlMs := int64(campaignDelayReservationTTL / time.Millisecond)
	rawSendAt, err := reserveCampaignDelaySlotScript.Run(
		ctx,
		w.Redis,
		[]string{campaignDelayRedisKey(delayScopeKey)},
		nowMs,
		gapMs,
		ttlMs,
	).Result()
	if err != nil {
		w.Log.Warn("Failed to reserve campaign delay slot, using local delay", "delay_scope", strings.TrimSpace(delayScopeKey), "error", err)
		return time.Duration(gapMs) * time.Millisecond, nil
	}

	sendAtMs, err := parseScriptResultInt64(rawSendAt)
	if err != nil {
		w.Log.Warn("Failed to parse reserved campaign delay slot, using local delay", "delay_scope", strings.TrimSpace(delayScopeKey), "error", err)
		return time.Duration(gapMs) * time.Millisecond, nil
	}

	waitMs := sendAtMs - nowMs
	if waitMs <= 0 {
		return 0, nil
	}

	return time.Duration(waitMs) * time.Millisecond, nil
}

func (w *Worker) transitionRecipientToSending(ctx context.Context, recipientID uuid.UUID, attemptID string) error {
	result := w.DB.Model(&models.BulkMessageRecipient{}).
		Where("id = ? AND status = ?", recipientID, models.MessageStatusPending).
		Updates(map[string]interface{}{
			"status":          models.MessageStatusSending,
			"send_attempt_id": attemptID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("recipient %s is no longer pending (RowsAffected=0)", recipientID)
	}
	return nil
}

func (w *Worker) updateRecipientStatusConditional(recipientID uuid.UUID, expectedCurrentStatus models.MessageStatus, newStatus models.MessageStatus, waMessageID, errorMsg string) {
	updates := map[string]interface{}{
		"status":               newStatus,
		"whats_app_message_id": waMessageID,
		"error_message":        errorMsg,
	}
	if newStatus == models.MessageStatusSent {
		updates["sent_at"] = time.Now()
	}
	result := w.DB.Model(&models.BulkMessageRecipient{}).
		Where("id = ? AND status IN ?", recipientID, []models.MessageStatus{expectedCurrentStatus, models.MessageStatusSending}).
		Updates(updates)
	if result.Error != nil {
		w.Log.Error("Failed to conditionally update recipient status", "error", result.Error, "recipient_id", recipientID, "new_status", newStatus)
	}
	if result.RowsAffected == 0 {
		w.Log.Warn("Recipient status update skipped (already processed)", "recipient_id", recipientID, "expected_status", expectedCurrentStatus, "new_status", newStatus)
	}
}

func (w *Worker) isWhatsmeowProvider() bool {
	return w.Config != nil && w.Config.WhatsApp.Provider == "whatsmeow"
}

func (w *Worker) shouldRunInboundMediaSelfHeal() bool {
	return w != nil &&
		w.InboundConsumer != nil &&
		w.DB != nil &&
		w.Redis != nil &&
		w.isWhatsmeowProvider()
}

func (w *Worker) runInboundMediaSelfHealLoop(ctx context.Context) {
	w.reconcileStaleInboundMedia(ctx)

	ticker := time.NewTicker(inboundMediaSelfHealInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.reconcileStaleInboundMedia(ctx)
		}
	}
}

func (w *Worker) reconcileStaleInboundMedia(ctx context.Context) {
	summary, err := waprovider.ReconcileStaleQueuedInboundMedia(
		ctx,
		w.DB,
		w.Redis,
		waprovider.InboundMediaReconcileOptions{
			OlderThan: inboundMediaSelfHealOlderThan,
			Limit:     inboundMediaSelfHealBatchLimit,
			Apply:     true,
		},
		w.Log,
	)
	if err != nil {
		w.Log.Warn("Inbound media self-heal skipped", "error", err)
		return
	}
	if summary == nil {
		return
	}
	if summary.Requeued == 0 && summary.MarkedFailed == 0 {
		return
	}

	w.Log.Info(
		"Inbound media self-heal pass completed",
		"requeued", summary.Requeued,
		"marked_failed", summary.MarkedFailed,
		"eligible_queued", summary.EligibleQueued,
	)
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
	var unfinishedCount int64
	if err := w.DB.Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND status IN ?", campaignID, []models.MessageStatus{models.MessageStatusPending, models.MessageStatusSending}).
		Count(&unfinishedCount).Error; err != nil {
		w.Log.Error("Failed to count unfinished campaign recipients", "error", err, "campaign_id", campaignID)
		return
	}

	if unfinishedCount == 0 {
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
			return
		}

		var campaign models.BulkMessageCampaign
		if err := w.DB.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
			w.Log.Error("Failed to load completed campaign", "error", err, "campaign_id", campaignID)
			return
		}

		w.Log.Info("Campaign completed", "campaign_id", campaignID, "sent", campaign.SentCount, "failed", campaign.FailedCount)

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
			return w.MessageProvider.SendDocument(sendCtx, instanceID, recipient.PhoneNumber, mediaRef, mediaFilename, body)
		}
	}

	// If the campaign has a poll configured, send a native WhatsApp poll instead of text.
	if campaign != nil && strings.TrimSpace(campaign.PollQuestion) != "" {
		if pollSender, ok := w.MessageProvider.(provider.PollProvider); ok {
			options := pollOptionsToStrings(campaign.PollOptions)
			if len(options) >= 2 {
				return pollSender.SendPoll(sendCtx, instanceID, recipient.PhoneNumber, campaign.PollQuestion, options, campaign.PollMaxSelections)
			}
		}
	}

	return w.MessageProvider.SendText(sendCtx, instanceID, recipient.PhoneNumber, body)
}

// pollOptionsToStrings extracts a []string from JSONBArray poll options.
func pollOptionsToStrings(raw models.JSONBArray) []string {
	if raw == nil {
		return nil
	}
	var out []string
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
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
