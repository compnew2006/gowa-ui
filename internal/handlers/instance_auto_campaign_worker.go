package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	waManager "github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceAutoCampaignWorker periodically creates campaigns based on per-instance settings.
type InstanceAutoCampaignWorker struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

type autoCampaignContactSeed struct {
	PhoneNumber string `gorm:"column:phone_number"`
	ProfileName string `gorm:"column:profile_name"`
}

func NewInstanceAutoCampaignWorker(app *App, interval time.Duration) *InstanceAutoCampaignWorker {
	return &InstanceAutoCampaignWorker{
		app:      app,
		interval: interval,
	}
}

func (w *InstanceAutoCampaignWorker) Start(ctx context.Context) {
	w.mu.Lock()
	w.ticker = time.NewTicker(w.interval)
	ticker := w.ticker
	w.mu.Unlock()
	defer ticker.Stop()

	w.runOnce(time.Now().UTC())

	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			w.runOnce(tick.UTC())
		}
	}
}

func (w *InstanceAutoCampaignWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ticker != nil {
		w.ticker.Stop()
		w.ticker = nil
	}
}

func (w *InstanceAutoCampaignWorker) runOnce(nowUTC time.Time) {
	if w.app == nil || !w.app.isWhatsmeowProvider() {
		return
	}

	var instances []models.WhatsAppInstance
	if err := w.app.DB.Select("id", "organization_id", "settings", "status", "send_blocked_until", "send_block_reason").Find(&instances).Error; err != nil {
		w.app.Log.Error("Auto campaign worker failed to load instances", "error", err)
		return
	}
	if len(instances) == 0 {
		return
	}

	orgTimezones := w.loadOrganizationTimezones(instances)
	for idx := range instances {
		instance := instances[idx]
		timezone := orgTimezones[instance.OrganizationID]
		if timezone == "" {
			timezone = "UTC"
		}

		if err := w.processInstance(nowUTC, instance, timezone); err != nil {
			w.app.Log.Error(
				"Auto campaign worker failed to process instance",
				"instance_id", instance.ID,
				"org_id", instance.OrganizationID,
				"timezone", timezone,
				"error", err,
			)
		}
	}
}

func (w *InstanceAutoCampaignWorker) loadOrganizationTimezones(instances []models.WhatsAppInstance) map[uuid.UUID]string {
	orgIDSet := make(map[uuid.UUID]struct{}, len(instances))
	for _, instance := range instances {
		orgIDSet[instance.OrganizationID] = struct{}{}
	}
	orgIDs := make([]uuid.UUID, 0, len(orgIDSet))
	for orgID := range orgIDSet {
		orgIDs = append(orgIDs, orgID)
	}

	timezones := make(map[uuid.UUID]string, len(orgIDs))
	if len(orgIDs) == 0 {
		return timezones
	}

	var organizations []models.Organization
	if err := w.app.DB.Select("id", "settings").Where("id IN ?", orgIDs).Find(&organizations).Error; err != nil {
		w.app.Log.Warn("Auto campaign worker failed to load organization settings; defaulting to UTC", "error", err)
		return timezones
	}
	for _, org := range organizations {
		timezones[org.ID] = parseOrganizationTimezone(org.Settings)
	}

	return timezones
}

func (w *InstanceAutoCampaignWorker) processInstance(nowUTC time.Time, instance models.WhatsAppInstance, timezone string) error {
	settings := waManager.AutoCampaignSettingsFromSettings(instance.Settings)
	if !settings.Enabled {
		return nil
	}
	if settings.IntervalDays < 1 {
		settings.IntervalDays = 1
	}
	if strings.TrimSpace(settings.Message) == "" {
		w.app.Log.Warn("Auto campaign skipped due to empty message", "instance_id", instance.ID, "org_id", instance.OrganizationID)
		return nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	localNow := nowUTC.In(location)

	if !isAutoCampaignDue(localNow.UTC(), settings.LastGeneratedAt, settings.IntervalDays) {
		return nil
	}

	windowStartLocal := localNow.AddDate(0, 0, -settings.IntervalDays)
	windowStartUTC := windowStartLocal.UTC()
	windowEndUTC := localNow.UTC()

	contacts, err := w.loadInstanceContactsInWindow(instance.OrganizationID, instance.ID, windowStartUTC, windowEndUTC)
	if err != nil {
		return err
	}

	if len(contacts) == 0 {
		if err := w.persistLastGeneratedAt(instance.ID, nowUTC); err != nil {
			return err
		}
		w.app.Log.Info("Auto campaign skipped; no contacts in window", "instance_id", instance.ID, "org_id", instance.OrganizationID, "days", settings.IntervalDays)
		return nil
	}

	campaignName := buildAutoCampaignName(settings.NamePrefix, windowStartLocal, localNow)
	duplicate, err := w.campaignNameExists(instance.OrganizationID, instance.ID.String(), campaignName, windowStartUTC)
	if err != nil {
		return err
	}
	if duplicate {
		if err := w.persistLastGeneratedAt(instance.ID, nowUTC); err != nil {
			return err
		}
		w.app.Log.Info("Auto campaign skipped; campaign already exists", "instance_id", instance.ID, "campaign_name", campaignName)
		return nil
	}

	campaign, recipients, err := w.createAutoCampaign(instance, settings, campaignName, contacts)
	if err != nil {
		return err
	}

	if settings.TargetStatus == waManager.AutoCampaignTargetStatusRun {
		if err := w.enforceAutoCampaignRunPolicy(instance); err != nil {
			if message, reasonCode, ok := asCampaignPolicyViolation(err); ok {
				w.app.Log.Warn(
					"Auto campaign kept as draft by policy",
					"instance_id", instance.ID,
					"campaign_id", campaign.ID,
					"reason_code", reasonCode,
					"message", message,
				)
			} else {
				return err
			}
		} else if err := w.startAutoCampaign(instance.OrganizationID, campaign, recipients); err != nil {
			return err
		}
	}

	if err := w.persistLastGeneratedAt(instance.ID, nowUTC); err != nil {
		return err
	}

	w.app.Log.Info(
		"Auto campaign generated",
		"instance_id", instance.ID,
		"org_id", instance.OrganizationID,
		"campaign_id", campaign.ID,
		"campaign_name", campaign.Name,
		"recipient_count", len(recipients),
		"target_status", settings.TargetStatus,
	)
	return nil
}

func (w *InstanceAutoCampaignWorker) enforceAutoCampaignRunPolicy(instance models.WhatsAppInstance) error {
	if err := w.app.enforceCampaignStartPolicy(instance.OrganizationID, instance.ID.String()); err != nil {
		return err
	}
	if instance.Status != models.InstanceStatusConnected {
		return &campaignPolicyViolationError{
			message:    "Campaign sender instance is not connected",
			reasonCode: ReasonCodeInstanceNotConn,
		}
	}
	if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
		return &campaignPolicyViolationError{
			message:    blockReason,
			reasonCode: ReasonCodeInstanceBlocked,
		}
	}
	return nil
}

func (w *InstanceAutoCampaignWorker) loadInstanceContactsInWindow(
	orgID, instanceID uuid.UUID,
	windowStartUTC, windowEndUTC time.Time,
) ([]autoCampaignContactSeed, error) {
	var seeds []autoCampaignContactSeed
	if err := w.app.DB.Model(&models.Contact{}).
		Select("phone_number", "profile_name").
		Where("organization_id = ? AND instance_id = ?", orgID, instanceID).
		Where("COALESCE(metadata->>'is_group_chat', 'false') <> 'true'").
		Where("COALESCE(metadata->>'is_channel_chat', 'false') <> 'true'").
		Where("COALESCE(last_inbound_at, created_at) >= ? AND COALESCE(last_inbound_at, created_at) < ?", windowStartUTC, windowEndUTC).
		Order("COALESCE(last_inbound_at, created_at) ASC").
		Find(&seeds).Error; err != nil {
		return nil, err
	}

	unique := make([]autoCampaignContactSeed, 0, len(seeds))
	seen := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		normalizedPhone := normalizeAutoCampaignPhone(seed.PhoneNumber)
		if normalizedPhone == "" {
			continue
		}
		if _, exists := seen[normalizedPhone]; exists {
			continue
		}
		seen[normalizedPhone] = struct{}{}
		seed.PhoneNumber = normalizedPhone
		unique = append(unique, seed)
	}

	return unique, nil
}

func (w *InstanceAutoCampaignWorker) createAutoCampaign(
	instance models.WhatsAppInstance,
	settings waManager.AutoCampaignSettings,
	campaignName string,
	recipients []autoCampaignContactSeed,
) (*models.BulkMessageCampaign, []models.BulkMessageRecipient, error) {
	minDelaySeconds, maxDelaySeconds := autoCampaignDelaySeconds(
		settings.MinDelayMinutes,
		settings.MaxDelayMinutes,
	)
	bodyContent := normalizeAutoCampaignMessageTemplate(settings.Message)

	template, err := w.app.resolveCampaignTemplate(instance.OrganizationID, CampaignRequest{
		Name:            campaignName,
		WhatsAppAccount: instance.ID.String(),
		BodyContent:     bodyContent,
	})
	if err != nil {
		return nil, nil, err
	}

	creatorID, err := w.resolveCampaignCreator(instance.OrganizationID)
	if err != nil {
		return nil, nil, err
	}

	campaign := &models.BulkMessageCampaign{
		OrganizationID:       instance.OrganizationID,
		WhatsAppAccount:      instance.ID.String(),
		Name:                 campaignName,
		TemplateID:           template.ID,
		HeaderMediaFilename:  strings.TrimSpace(settings.MediaFilename),
		HeaderMediaMimeType:  strings.TrimSpace(settings.MediaMimeType),
		HeaderMediaLocalPath: strings.TrimSpace(settings.MediaLocalPath),
		MinDelaySeconds:      minDelaySeconds,
		MaxDelaySeconds:      maxDelaySeconds,
		Status:               models.CampaignStatusDraft,
		CreatedBy:            creatorID,
	}

	if err := w.app.DB.Create(campaign).Error; err != nil {
		return nil, nil, err
	}

	campaignRecipients := make([]models.BulkMessageRecipient, 0, len(recipients))
	for _, seed := range recipients {
		recipientName := strings.TrimSpace(seed.ProfileName)
		if recipientName == "" {
			recipientName = seed.PhoneNumber
		}
		campaignRecipients = append(campaignRecipients, models.BulkMessageRecipient{
			CampaignID:    campaign.ID,
			PhoneNumber:   seed.PhoneNumber,
			RecipientName: recipientName,
			TemplateParams: models.JSONB{
				"contact_name": recipientName,
				"phone_number": seed.PhoneNumber,
			},
			Status: models.MessageStatusPending,
		})
	}

	if len(campaignRecipients) == 0 {
		return nil, nil, fmt.Errorf("no valid recipients found for auto campaign")
	}

	if err := w.app.DB.Create(&campaignRecipients).Error; err != nil {
		return nil, nil, err
	}
	if err := w.app.DB.Model(campaign).Update("total_recipients", len(campaignRecipients)).Error; err != nil {
		return nil, nil, err
	}

	return campaign, campaignRecipients, nil
}

func (w *InstanceAutoCampaignWorker) resolveCampaignCreator(orgID uuid.UUID) (uuid.UUID, error) {
	var user models.User
	if err := w.app.DB.Select("id").
		Where("organization_id = ?", orgID).
		Order("created_at ASC").
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, fmt.Errorf("organization has no users")
		}
		return uuid.Nil, err
	}
	return user.ID, nil
}

func (w *InstanceAutoCampaignWorker) startAutoCampaign(
	orgID uuid.UUID,
	campaign *models.BulkMessageCampaign,
	recipients []models.BulkMessageRecipient,
) error {
	if w.app.Queue == nil {
		return fmt.Errorf("queue is not initialized")
	}

	now := time.Now().UTC()
	if err := w.app.DB.Model(campaign).Updates(map[string]any{
		"status":     models.CampaignStatusProcessing,
		"started_at": now,
	}).Error; err != nil {
		return err
	}

	jobs := make([]*queue.RecipientJob, len(recipients))
	for i, recipient := range recipients {
		jobs[i] = &queue.RecipientJob{
			CampaignID:     campaign.ID,
			RecipientID:    recipient.ID,
			OrganizationID: orgID,
			PhoneNumber:    recipient.PhoneNumber,
			RecipientName:  recipient.RecipientName,
			TemplateParams: recipient.TemplateParams,
		}
	}

	if err := w.app.Queue.EnqueueRecipients(context.Background(), jobs); err != nil {
		_ = w.app.DB.Model(campaign).Updates(map[string]any{
			"status":     models.CampaignStatusDraft,
			"started_at": nil,
		}).Error
		return err
	}

	return nil
}

func (w *InstanceAutoCampaignWorker) campaignNameExists(
	orgID uuid.UUID,
	whatsappAccount, campaignName string,
	createdAfter time.Time,
) (bool, error) {
	var total int64
	if err := w.app.DB.Model(&models.BulkMessageCampaign{}).
		Where("organization_id = ? AND whats_app_account = ? AND name = ?", orgID, whatsappAccount, campaignName).
		Where("created_at >= ?", createdAfter).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func isAutoCampaignDue(nowUTC time.Time, lastGeneratedAt *time.Time, intervalDays int) bool {
	if intervalDays < 1 {
		intervalDays = 1
	}
	if lastGeneratedAt == nil {
		return true
	}
	nextRun := lastGeneratedAt.UTC().AddDate(0, 0, intervalDays)
	return !nowUTC.Before(nextRun)
}

func buildAutoCampaignName(prefix string, windowStart, windowEnd time.Time) string {
	_, week := windowEnd.ISOWeek()
	baseName := fmt.Sprintf(
		"week%d-%d/%d-%d/%d",
		week,
		windowStart.Day(),
		int(windowStart.Month()),
		windowEnd.Day(),
		int(windowEnd.Month()),
	)

	trimmedPrefix := strings.TrimSpace(prefix)
	if trimmedPrefix == "" {
		return baseName
	}
	return trimmedPrefix + baseName
}

func normalizeAutoCampaignMessageTemplate(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return normalized
	}

	normalized = strings.ReplaceAll(normalized, "{{contact_name}}", "__whm_contact_name__")
	normalized = strings.ReplaceAll(normalized, "{{phone_number}}", "__whm_phone_number__")
	normalized = strings.ReplaceAll(normalized, "{contact_name}", "{{contact_name}}")
	normalized = strings.ReplaceAll(normalized, "{phone_number}", "{{phone_number}}")
	normalized = strings.ReplaceAll(normalized, "__whm_contact_name__", "{{contact_name}}")
	normalized = strings.ReplaceAll(normalized, "__whm_phone_number__", "{{phone_number}}")

	return normalized
}

func autoCampaignDelaySeconds(minDelayMinutes, maxDelayMinutes int) (int, int) {
	if minDelayMinutes < 0 {
		minDelayMinutes = 0
	}
	if maxDelayMinutes < 0 {
		maxDelayMinutes = 0
	}
	if maxDelayMinutes < minDelayMinutes {
		maxDelayMinutes = minDelayMinutes
	}

	const secondsPerMinute = 60
	return minDelayMinutes * secondsPerMinute, maxDelayMinutes * secondsPerMinute
}

func normalizeAutoCampaignPhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func timePtr(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}

func (w *InstanceAutoCampaignWorker) persistLastGeneratedAt(instanceID uuid.UUID, generatedAt time.Time) error {
	var instance models.WhatsAppInstance
	if err := w.app.DB.Select("id", "settings").Where("id = ?", instanceID).First(&instance).Error; err != nil {
		return err
	}

	settings := waManager.AutoCampaignSettingsFromSettings(instance.Settings)
	settings.LastGeneratedAt = timePtr(generatedAt)
	return w.app.persistInstanceAutoCampaignSettings(instanceID, settings)
}
