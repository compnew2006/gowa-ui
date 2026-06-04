package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	scheduledSendsKey         = "whatomate:scheduled_sends"
	scheduledSendsPollTimeout = 1 * time.Second
	scheduledSendsBatchSize   = 50
	scheduledSendLockTTL      = 30 * time.Minute
)

type scheduledSend struct {
	CampaignID     uuid.UUID `json:"campaign_id"`
	RecipientID    uuid.UUID `json:"recipient_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	PhoneNumber    string    `json:"phone_number"`
	RecipientName  string    `json:"recipient_name"`
	TemplateParams string    `json:"template_params"`
	RecipientType  string    `json:"recipient_type,omitempty"`
	GroupJID       string    `json:"group_jid,omitempty"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
	SendAt         time.Time `json:"send_at"`
}

func scheduledSendFromRecipientJob(job *queue.RecipientJob, sendAt time.Time) *scheduledSend {
	tp, _ := json.Marshal(job.TemplateParams)
	return &scheduledSend{
		CampaignID:     job.CampaignID,
		RecipientID:    job.RecipientID,
		OrganizationID: job.OrganizationID,
		PhoneNumber:    job.PhoneNumber,
		RecipientName:  job.RecipientName,
		TemplateParams: string(tp),
		RecipientType:  job.RecipientType,
		GroupJID:       job.GroupJID,
		EnqueuedAt:     job.EnqueuedAt,
		SendAt:         sendAt,
	}
}

func (s *scheduledSend) toRecipientJob() *queue.RecipientJob {
	var job queue.RecipientJob
	job.CampaignID = s.CampaignID
	job.RecipientID = s.RecipientID
	job.OrganizationID = s.OrganizationID
	job.PhoneNumber = s.PhoneNumber
	job.RecipientName = s.RecipientName
	_ = json.Unmarshal([]byte(s.TemplateParams), &job.TemplateParams)
	job.RecipientType = s.RecipientType
	job.GroupJID = s.GroupJID
	job.EnqueuedAt = s.EnqueuedAt
	return &job
}

func (w *Worker) scheduleRecipientSend(ctx context.Context, job *queue.RecipientJob, sendAt time.Time) error {
	if w.Redis == nil {
		return nil
	}
	ss := scheduledSendFromRecipientJob(job, sendAt)
	payload, err := json.Marshal(ss)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled send: %w", err)
	}
	score := float64(sendAt.UnixMilli())
	if err := w.Redis.ZAdd(ctx, scheduledSendsKey, redis.Z{
		Score:  score,
		Member: string(payload),
	}).Err(); err != nil {
		return fmt.Errorf("failed to schedule send in zset: %w", err)
	}
	w.Log.Info("Scheduled recipient send", "recipient_id", job.RecipientID, "campaign_id", job.CampaignID, "send_at", sendAt)
	return nil
}

func (w *Worker) runScheduledSendsPoller(ctx context.Context) {
	w.Log.Info("Scheduled sends poller starting")
	ticker := time.NewTicker(scheduledSendsPollTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Log.Info("Scheduled sends poller stopping")
			return
		case <-ticker.C:
			if err := w.pollScheduledSends(ctx); err != nil {
				w.Log.Error("Scheduled sends poll error", "error", err)
			}
		}
	}
}

func (w *Worker) pollScheduledSends(ctx context.Context) error {
	now := float64(time.Now().UnixMilli())
	results, err := w.Redis.ZRangeByScore(ctx, scheduledSendsKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: scheduledSendsBatchSize,
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to query scheduled sends: %w", err)
	}
	if len(results) == 0 {
		return nil
	}

	for _, raw := range results {
		var ss scheduledSend
		if err := json.Unmarshal([]byte(raw), &ss); err != nil {
			w.Log.Error("Failed to unmarshal scheduled send, removing corrupt entry", "error", err)
			w.Redis.ZRem(ctx, scheduledSendsKey, raw)
			continue
		}

		job := ss.toRecipientJob()
		token, claimed := w.claimScheduledSend(ctx, job.RecipientID)
		if !claimed {
			continue
		}

		removed, remErr := w.Redis.ZRem(ctx, scheduledSendsKey, raw).Result()
		if remErr != nil {
			w.Log.Error("Failed to remove scheduled send from ZSET before execution", "error", remErr, "recipient_id", job.RecipientID)
			w.releaseScheduledSendLock(ctx, job.RecipientID, token)
			continue
		}
		if removed == 0 {
			w.Log.Debug("Scheduled send already removed from ZSET by concurrent poller", "recipient_id", job.RecipientID)
			w.releaseScheduledSendLock(ctx, job.RecipientID, token)
			continue
		}

		w.Log.Info("Processing scheduled send", "recipient_id", job.RecipientID, "campaign_id", job.CampaignID)

		attemptID := uuid.New().String()
		stampResult := w.DB.Model(&models.BulkMessageRecipient{}).
			Where("id = ? AND status IN ?", job.RecipientID, []models.MessageStatus{models.MessageStatusPending, models.MessageStatusSending}).
			Updates(map[string]interface{}{
				"status":          models.MessageStatusSending,
				"send_attempt_id": attemptID,
			})
		if stampResult.Error != nil {
			w.Log.Error("Failed to stamp attempt ID", "error", stampResult.Error, "recipient_id", job.RecipientID)
			if reErr := w.scheduleRecipientSend(ctx, job, time.Now()); reErr != nil {
				w.Log.Error("Failed to re-insert scheduled send after attempt stamp failure", "error", reErr, "recipient_id", job.RecipientID)
			}
			w.releaseScheduledSendLock(ctx, job.RecipientID, token)
			continue
		}
		if stampResult.RowsAffected == 0 {
			w.Log.Debug("Recipient no longer pending or sending, skipping", "recipient_id", job.RecipientID)
			w.releaseScheduledSendLock(ctx, job.RecipientID, token)
			continue
		}

		execErr := w.executeRecipientSendWithAttempt(ctx, job, attemptID)

		if execErr != nil {
			w.Log.Error("Scheduled send execution failed, re-inserting for retry", "error", execErr, "recipient_id", job.RecipientID, "campaign_id", job.CampaignID)
			if reErr := w.scheduleRecipientSend(ctx, job, time.Now()); reErr != nil {
				w.Log.Error("Failed to re-insert scheduled send after execution failure", "error", reErr, "recipient_id", job.RecipientID)
			}
			w.releaseScheduledSendLock(ctx, job.RecipientID, token)
			continue
		}

		if !w.verifyScheduledSendLock(ctx, job.RecipientID, token) {
			w.Log.Warn("Scheduled send lock was lost during execution — another worker may have attempted send", "recipient_id", job.RecipientID)
		}
		w.releaseScheduledSendLock(ctx, job.RecipientID, token)
	}
	return nil
}

func (w *Worker) claimScheduledSend(ctx context.Context, recipientID uuid.UUID) (string, bool) {
	if w.Redis == nil {
		return "local", true
	}
	token := uuid.New().String()
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	res, err := w.Redis.SetArgs(ctx, key, token, redis.SetArgs{Mode: "NX", TTL: scheduledSendLockTTL}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false
		}
		w.Log.Warn("Failed to claim scheduled send lock", "error", err)
		return token, true
	}
	if res == "OK" {
		return token, true
	}
	return "", false
}

func (w *Worker) releaseScheduledSendLock(ctx context.Context, recipientID uuid.UUID, token string) {
	if w.Redis == nil || token == "" {
		return
	}
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	script := redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) else return 0 end`)
	_, err := script.Run(ctx, w.Redis, []string{key}, token).Result()
	if err != nil && err != redis.Nil {
		w.Log.Warn("Failed to release scheduled send lock", "error", err, "recipient_id", recipientID)
	}
}

func (w *Worker) verifyScheduledSendLock(ctx context.Context, recipientID uuid.UUID, expectedToken string) bool {
	if w.Redis == nil || expectedToken == "" {
		return true
	}
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	val, err := w.Redis.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return val == expectedToken
}

const (
	sendingStuckThreshold = 30 * time.Minute
	sendingRecoveryTick   = 5 * time.Minute
	sendingRecoveryBatch  = 100
)

func (w *Worker) runSendingRecoveryLoop(ctx context.Context) {
	w.Log.Info("Sending recovery loop starting")
	ticker := time.NewTicker(sendingRecoveryTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.Log.Info("Sending recovery loop stopping")
			return
		case <-ticker.C:
			if err := w.recoverStuckSendingRecipients(ctx); err != nil {
				w.Log.Error("Sending recovery error", "error", err)
			}
		}
	}
}

func (w *Worker) recoverStuckSendingRecipients(ctx context.Context) error {
	cutoff := time.Now().Add(-sendingStuckThreshold)
	var recipients []models.BulkMessageRecipient
	if err := w.DB.Where("status = ? AND updated_at < ?", models.MessageStatusSending, cutoff).
		Limit(sendingRecoveryBatch).
		Find(&recipients).Error; err != nil {
		return fmt.Errorf("failed to query stuck sending recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil
	}

	w.Log.Warn("Recovering stuck sending recipients", "count", len(recipients))
	for _, r := range recipients {
		if w.isScheduledSendLockHeld(ctx, r.ID) {
			w.Log.Debug("Skipping stuck recipient with active scheduled-send lock", "recipient_id", r.ID)
			continue
		}

		var campaign models.BulkMessageCampaign
		if err := w.DB.Where("id = ?", r.CampaignID).First(&campaign).Error; err != nil {
			w.Log.Error("Failed to load campaign for stuck recipient, marking as failed",
				"error", err, "recipient_id", r.ID, "campaign_id", r.CampaignID)
			w.updateRecipientStatusConditional(r.ID, models.MessageStatusSending, models.MessageStatusFailed, "", "Recovery failed: campaign not found")
			w.incrementCampaignCount(r.CampaignID, "failed_count")
			continue
		}

		if campaign.Status == models.CampaignStatusPaused || campaign.Status == models.CampaignStatusCancelled {
			w.Log.Info("Stuck recipient belongs to paused/cancelled campaign, marking as failed",
				"recipient_id", r.ID, "campaign_id", r.CampaignID, "campaign_status", campaign.Status)
			w.updateRecipientStatusConditional(r.ID, models.MessageStatusSending, models.MessageStatusFailed, "", fmt.Sprintf("Campaign %s during recovery", campaign.Status))
			w.incrementCampaignCount(r.CampaignID, "failed_count")
			continue
		}

		result := w.DB.Model(&models.BulkMessageRecipient{}).
			Where("id = ? AND status = ? AND (send_attempt_id = '' OR send_attempt_id = ?)", r.ID, models.MessageStatusSending, r.SendAttemptID).
			Updates(map[string]interface{}{
				"status":          models.MessageStatusPending,
				"send_attempt_id": "",
			})
		if result.Error != nil {
			w.Log.Error("Failed to reset stuck recipient", "error", result.Error, "recipient_id", r.ID)
			continue
		}
		if result.RowsAffected == 0 {
			w.Log.Debug("Stuck recipient already claimed by another worker or recovered", "recipient_id", r.ID)
			continue
		}
		w.Log.Info("Reset stuck sending recipient to pending", "recipient_id", r.ID, "campaign_id", r.CampaignID)

		job := &queue.RecipientJob{
			CampaignID:     campaign.ID,
			RecipientID:    r.ID,
			OrganizationID: campaign.OrganizationID,
			PhoneNumber:    r.PhoneNumber,
			RecipientName:  r.RecipientName,
			TemplateParams: r.TemplateParams,
			RecipientType:  r.RecipientType,
			GroupJID:       r.GroupJID,
			EnqueuedAt:     time.Now(),
		}

		if w.Redis != nil {
			q := queue.NewRedisQueue(w.Redis, w.Log)
			if qErr := q.EnqueueRecipient(ctx, job); qErr != nil {
				w.Log.Error("Failed to re-enqueue recovered recipient via stream, attempting ZSET fallback",
					"error", qErr, "recipient_id", r.ID, "campaign_id", r.CampaignID)
				if zErr := w.scheduleRecipientSend(ctx, job, time.Now()); zErr != nil {
					w.Log.Error("Failed to re-schedule recovered recipient, marking as failed",
						"error", zErr, "recipient_id", r.ID, "campaign_id", r.CampaignID)
					w.updateRecipientStatusConditional(r.ID, models.MessageStatusPending, models.MessageStatusFailed, "", "Recovery failed: unable to re-enqueue")
					w.incrementCampaignCount(r.CampaignID, "failed_count")
				} else {
					w.Log.Info("Re-scheduled recovered recipient via ZSET fallback", "recipient_id", r.ID, "campaign_id", r.CampaignID)
				}
			} else {
				w.Log.Info("Re-enqueued recovered recipient via stream", "recipient_id", r.ID, "campaign_id", r.CampaignID)
			}
		}
	}
	return nil
}

func (w *Worker) isScheduledSendLockHeld(ctx context.Context, recipientID uuid.UUID) bool {
	if w.Redis == nil {
		return false
	}
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	val, err := w.Redis.Exists(ctx, key).Result()
	if err != nil {
		w.Log.Warn("Failed to check scheduled send lock", "error", err, "recipient_id", recipientID)
		return false
	}
	return val == 1
}
