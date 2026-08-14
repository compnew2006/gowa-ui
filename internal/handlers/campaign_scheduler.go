package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CampaignSchedulerProcessor fires due scheduled bulk campaigns and keeps the
// recipient claim state machine healthy. It ticks every minute and:
//
//  1. Recovers recipients stranded in "sending" by a crashed worker — a
//     worker claims a recipient pending→sending before sending; if it dies
//     mid-send the row would otherwise block campaign completion forever.
//     Stale rows are marked failed (never auto-re-sent — a surprise re-send
//     is worse than a manual retry from the campaign page).
//  2. Completes "processing" campaigns whose recipients are all terminal
//     (self-heal for campaigns whose last completion check never ran).
//  3. Atomically claims campaigns whose scheduled_at has passed
//     (scheduled→processing UPDATE ... RETURNING with FOR UPDATE SKIP LOCKED,
//     so concurrent replicas can never double-fire) and enqueues their
//     pending recipients on the same Redis Stream the manual start uses.
//
// Lifecycle mirrors ScheduledMessageProcessor / GowaWebhookProcessor
// (Start/Stop with a context + stopCh).
type CampaignSchedulerProcessor struct {
	app      *App
	interval time.Duration
	stopCh   chan struct{}
}

// campaignRecipientStaleAge is how long a recipient may sit in "sending"
// before the scheduler assumes the claiming worker crashed mid-send and
// fails the row. Generous enough that no live send exceeds it.
const campaignRecipientStaleAge = 10 * time.Minute

// NewCampaignSchedulerProcessor creates a new scheduled-campaign dispatcher.
// The interval controls the due-campaign poll frequency; the default is one
// minute.
func NewCampaignSchedulerProcessor(app *App, interval time.Duration) *CampaignSchedulerProcessor {
	return &CampaignSchedulerProcessor{
		app:      app,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the dispatch loop. Blocks until the context is cancelled or
// Stop is called.
func (p *CampaignSchedulerProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Campaign scheduler processor started", "interval", p.interval)

	// Catch up shortly after startup: fire anything that came due while the
	// server was down.
	time.AfterFunc(15*time.Second, func() {
		p.RecoverStaleRecipients(time.Now())
		p.CompleteIdleCampaigns()
		p.ProcessDue(time.Now())
	})

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Campaign scheduler processor stopped by context")
			return
		case <-p.stopCh:
			p.app.Log.Info("Campaign scheduler processor stopped")
			return
		case now := <-ticker.C:
			p.RecoverStaleRecipients(now)
			p.CompleteIdleCampaigns()
			p.ProcessDue(now)
		}
	}
}

// Stop signals the processor to exit. Idempotent.
func (p *CampaignSchedulerProcessor) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

// RecoverStaleRecipients fails recipients claimed (pending→sending) by a
// worker that crashed before recording an outcome, and keeps the campaigns'
// failed counters consistent.
func (p *CampaignSchedulerProcessor) RecoverStaleRecipients(now time.Time) {
	type staleAgg struct {
		CampaignID uuid.UUID
		Count      int64
	}
	var agg []staleAgg
	if err := p.app.DB.Model(&models.BulkMessageRecipient{}).
		Select("campaign_id, COUNT(*) as count").
		Where("status = ? AND updated_at < ?", models.MessageStatusSending, now.Add(-campaignRecipientStaleAge)).
		Group("campaign_id").Scan(&agg).Error; err != nil {
		p.app.Log.Error("Failed to find stale sending recipients", "error", err)
		return
	}
	if len(agg) == 0 {
		return
	}

	const staleErrMsg = "send interrupted (worker crash); retry from the campaign page"
	res := p.app.DB.Model(&models.BulkMessageRecipient{}).
		Where("status = ? AND updated_at < ?", models.MessageStatusSending, now.Add(-campaignRecipientStaleAge)).
		Updates(map[string]any{
			"status":        models.MessageStatusFailed,
			"error_message": staleErrMsg,
		})
	if res.Error != nil {
		p.app.Log.Error("Failed to recover stale sending recipients", "error", res.Error)
		return
	}
	p.app.Log.Warn("Failed stale sending recipients (worker crash)",
		"count", res.RowsAffected)

	for _, a := range agg {
		p.app.DB.Model(&models.BulkMessageCampaign{}).
			Where("id = ?", a.CampaignID).
			Update("failed_count", gorm.Expr("failed_count + ?", a.Count))
	}
}

// CompleteIdleCampaigns marks "processing" campaigns completed once no
// recipient is pending or sending. The worker's per-job completion check
// normally does this; this covers the case where the last check never ran
// (e.g. after stale-recipient recovery).
func (p *CampaignSchedulerProcessor) CompleteIdleCampaigns() {
	now := time.Now()
	res := p.app.DB.Model(&models.BulkMessageCampaign{}).
		Where(`status = ? AND NOT EXISTS (
			SELECT 1 FROM bulk_message_recipients r
			WHERE r.campaign_id = bulk_message_campaigns.id
			  AND r.status IN (?, ?) AND r.deleted_at IS NULL)`,
			models.CampaignStatusProcessing, models.MessageStatusPending, models.MessageStatusSending).
		Updates(map[string]any{
			"status":       models.CampaignStatusCompleted,
			"completed_at": now,
		})
	if res.Error != nil {
		p.app.Log.Error("Failed to complete idle campaigns", "error", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		p.app.Log.Info("Completed idle campaigns", "count", res.RowsAffected)
	}
}

// ProcessDue atomically claims all due scheduled campaigns and enqueues their
// recipients. The claim (UPDATE ... RETURNING over FOR UPDATE SKIP LOCKED) is
// the idempotency barrier: a campaign can only transition scheduled→processing
// once, no matter how many replicas tick simultaneously.
func (p *CampaignSchedulerProcessor) ProcessDue(now time.Time) {
	var due []models.BulkMessageCampaign
	res := p.app.DB.Model(&due).Clauses(clause.Returning{}).
		Where(`id IN (SELECT id FROM bulk_message_campaigns
			WHERE status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?
			AND deleted_at IS NULL
			FOR UPDATE SKIP LOCKED)`,
			models.CampaignStatusScheduled, now).
		Updates(map[string]any{
			"status":     models.CampaignStatusProcessing,
			"started_at": now,
		})
	if res.Error != nil {
		p.app.Log.Error("Failed to claim due scheduled campaigns", "error", res.Error)
		return
	}
	if len(due) == 0 {
		return
	}

	p.app.Log.Info("Firing due scheduled campaigns", "count", len(due))
	for i := range due {
		p.launchOne(&due[i])
	}
}

// launchOne enqueues a claimed campaign's recipients. Permanent config errors
// (missing template, no recipients) fail the campaign; a queue outage returns
// it to "scheduled" so the next tick retries.
func (p *CampaignSchedulerProcessor) launchOne(campaign *models.BulkMessageCampaign) {
	var template models.Template
	if err := p.app.DB.Where("id = ? AND organization_id = ?", campaign.TemplateID, campaign.OrganizationID).
		First(&template).Error; err != nil {
		p.app.Log.Error("Scheduled campaign template no longer exists",
			"campaign_id", campaign.ID, "template_id", campaign.TemplateID)
		p.failCampaign(campaign, "campaign template no longer exists")
		return
	}

	enqueued, err := p.app.enqueuePendingRecipients(context.Background(), campaign.OrganizationID, campaign.ID)
	if err != nil {
		p.app.Log.Error("Failed to enqueue scheduled campaign recipients; will retry next tick",
			"error", err, "campaign_id", campaign.ID)
		p.app.DB.Model(campaign).Update("status", models.CampaignStatusScheduled)
		return
	}
	if enqueued == 0 {
		p.app.Log.Error("Scheduled campaign has no pending recipients",
			"campaign_id", campaign.ID)
		p.failCampaign(campaign, "campaign has no pending recipients")
		return
	}

	p.app.Log.Info("Scheduled campaign fired",
		"campaign_id", campaign.ID, "recipients", enqueued)
}

// failCampaign marks a claimed campaign failed with the reason recorded in
// the log (the model has no error field).
func (p *CampaignSchedulerProcessor) failCampaign(campaign *models.BulkMessageCampaign, reason string) {
	if err := p.app.DB.Model(campaign).Updates(map[string]any{
		"status": models.CampaignStatusFailed,
	}).Error; err != nil {
		p.app.Log.Error("Failed to mark scheduled campaign failed",
			"error", err, "campaign_id", campaign.ID)
	}
	p.app.Log.Warn("Scheduled campaign failed", "campaign_id", campaign.ID, "reason", reason)
}
