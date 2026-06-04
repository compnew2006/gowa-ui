package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	scheduledSendsKey         = "whatomate:scheduled_sends"
	scheduledSendsPollTimeout = 1 * time.Second
	scheduledSendsBatchSize   = 50
	scheduledSendLockTTL      = 5 * time.Minute
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
		Min: "-inf",
		Max: fmt.Sprintf("%f", now),
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
			w.Log.Error("Failed to unmarshal scheduled send, removing", "error", err)
			w.Redis.ZRem(ctx, scheduledSendsKey, raw)
			continue
		}

		removed, err := w.Redis.ZRem(ctx, scheduledSendsKey, raw).Result()
		if err != nil || removed == 0 {
			continue
		}

		job := ss.toRecipientJob()
		w.Log.Info("Processing scheduled send", "recipient_id", job.RecipientID, "campaign_id", job.CampaignID)
		if err := w.executeRecipientSend(ctx, job); err != nil {
			w.Log.Error("Failed to execute scheduled send", "error", err, "recipient_id", job.RecipientID, "campaign_id", job.CampaignID)
		}
	}
	return nil
}

func (w *Worker) claimScheduledSend(ctx context.Context, recipientID uuid.UUID) bool {
	if w.Redis == nil {
		return true
	}
	key := fmt.Sprintf("whatomate:scheduled_send_lock:%s", recipientID)
	res, err := w.Redis.SetArgs(ctx, key, "1", redis.SetArgs{Mode: "NX", TTL: scheduledSendLockTTL}).Result()
	if err != nil {
		if err == redis.Nil {
			return false
		}
		w.Log.Warn("Failed to claim scheduled send lock", "error", err)
		return true
	}
	return res == "OK"
}
