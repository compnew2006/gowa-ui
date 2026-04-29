package handlers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
)

const campaignSchedulerBatchSize = 100

// CampaignScheduler starts due scheduled campaigns.
type CampaignScheduler struct {
	app      *App
	interval time.Duration
	mu       sync.Mutex
	ticker   *time.Ticker
}

func NewCampaignScheduler(app *App, interval time.Duration) *CampaignScheduler {
	return &CampaignScheduler{
		app:      app,
		interval: interval,
	}
}

func (s *CampaignScheduler) Start(ctx context.Context) {
	if s == nil || s.app == nil {
		return
	}

	s.mu.Lock()
	s.ticker = time.NewTicker(s.interval)
	ticker := s.ticker
	s.mu.Unlock()
	defer ticker.Stop()

	s.RunOnce(ctx, time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			s.RunOnce(ctx, tick.UTC())
		}
	}
}

func (s *CampaignScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

func (s *CampaignScheduler) RunOnce(ctx context.Context, nowUTC time.Time) {
	if s == nil || s.app == nil || s.app.DB == nil {
		return
	}

	if err := s.app.DB.WithContext(ctx).
		Model(&models.BulkMessageCampaign{}).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at > ?", models.CampaignStatusDraft, nowUTC).
		Update("status", models.CampaignStatusScheduled).Error; err != nil {
		s.app.Log.Error("Failed to mark future scheduled campaigns", "error", err)
	}

	var due []models.BulkMessageCampaign
	if err := s.app.DB.WithContext(ctx).
		Select("id", "organization_id").
		Where("status IN ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
			[]models.CampaignStatus{models.CampaignStatusDraft, models.CampaignStatusScheduled}, nowUTC).
		Order("scheduled_at ASC").
		Limit(campaignSchedulerBatchSize).
		Find(&due).Error; err != nil {
		s.app.Log.Error("Failed to load due scheduled campaigns", "error", err)
		return
	}

	for _, campaign := range due {
		if _, err := s.app.StartCampaignByID(ctx, s.app.DB, campaign.OrganizationID, campaign.ID); err != nil {
			var startErr *campaignStartError
			if errors.As(err, &startErr) && startErr.kind == campaignStartConflict {
				s.app.Log.Debug("Scheduled campaign was already claimed", "campaign_id", campaign.ID, "organization_id", campaign.OrganizationID)
				continue
			}
			s.app.Log.Warn("Scheduled campaign did not start", "campaign_id", campaign.ID, "organization_id", campaign.OrganizationID, "error", err)
			continue
		}
		s.app.Log.Info("Scheduled campaign started", "campaign_id", campaign.ID, "organization_id", campaign.OrganizationID)
	}
}
