package whatsmeow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

const (
	defaultInboundMediaReconcileOlderThan = 15 * time.Minute
	staleInboundMediaQueueReason          = "queued inbound media recovery record is stale after queue drained"
	inboundMediaPendingBatchSize          = int64(200)
)

var inboundMediaRecoverableTypes = []models.MessageType{
	models.MessageTypeImage,
	models.MessageTypeSticker,
	models.MessageTypeVideo,
	models.MessageTypeAudio,
	models.MessageTypeDocument,
}

type InboundMediaReconcileOptions struct {
	InstanceID       *uuid.UUID
	OlderThan        time.Duration
	Limit            int
	Apply            bool
	AllowActiveQueue bool
	Now              time.Time
}

type InboundMediaReconcileSummary struct {
	DryRun              bool
	Cutoff              time.Time
	QueuePending        int64
	QueueLag            int64
	ActivePendingIDs    int64
	SkippedActiveQueued int64
	TotalQueued         int64
	EligibleQueued      int64
	Updated             int64
	SampleIDs           []string
}

type inboundMediaQueueGroupState struct {
	Found   bool
	Pending int64
	Lag     int64
}

type inboundMediaQueuedMessage struct {
	ID        uuid.UUID    `gorm:"column:id"`
	Metadata  models.JSONB `gorm:"column:metadata;type:jsonb"`
	UpdatedAt time.Time    `gorm:"column:updated_at"`
}

func (s inboundMediaQueueGroupState) validate(allowActive bool) error {
	if !s.Found {
		return fmt.Errorf("redis consumer group %q not found on stream %q", queue.InboundMediaConsumerGroup, queue.InboundMediaStreamName)
	}
	if allowActive {
		return nil
	}
	if s.Lag > 0 {
		return fmt.Errorf("refusing reconciliation while inbound-media queue has lag=%d", s.Lag)
	}
	if s.Lag < 0 {
		return fmt.Errorf("refusing reconciliation because inbound-media queue lag is unknown")
	}
	return nil
}

func loadInboundMediaQueueGroupState(ctx context.Context, rdb *redis.Client) (inboundMediaQueueGroupState, error) {
	groups, err := rdb.XInfoGroups(ctx, queue.InboundMediaStreamName).Result()
	if err != nil {
		return inboundMediaQueueGroupState{}, fmt.Errorf("load inbound-media consumer groups: %w", err)
	}

	for _, group := range groups {
		if group.Name != queue.InboundMediaConsumerGroup {
			continue
		}
		return inboundMediaQueueGroupState{
			Found:   true,
			Pending: group.Pending,
			Lag:     group.Lag,
		}, nil
	}

	return inboundMediaQueueGroupState{}, nil
}

func buildStaleInboundMediaFailure(metadata models.JSONB) (models.JSONB, string, string) {
	nextMetadata := cloneJSONBMap(metadata)
	if nextMetadata == nil {
		nextMetadata = models.JSONB{}
	}

	reason := strings.TrimSpace(jsonBString(nextMetadata[inboundMediaAsyncLastErrorKey]))
	if reason == "" {
		reason = staleInboundMediaQueueReason
	}

	nextMetadata[inboundMediaAsyncStatusKey] = inboundMediaAsyncStatusFailed
	nextMetadata[inboundMediaAsyncLastErrorKey] = reason
	nextMetadata[inboundMediaAsyncRecoveredAtKey] = nil
	delete(nextMetadata, inboundMediaAsyncEnqueueErrorKey)

	return nextMetadata, reason, inboundMediaFailureErrorMessage(reason)
}

func inboundMediaFailureErrorMessage(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "Inbound media async recovery failed"
	}
	return fmt.Sprintf("Inbound media async recovery failed: %s", reason)
}

func jsonBString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		text := fmt.Sprint(v)
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func loadPendingInboundMediaMessageID(ctx context.Context, rdb *redis.Client, streamID string) (uuid.UUID, error) {
	streamMessages, err := rdb.XRangeN(ctx, queue.InboundMediaStreamName, streamID, streamID, 1).Result()
	if err != nil {
		return uuid.Nil, fmt.Errorf("read stream entry: %w", err)
	}
	if len(streamMessages) == 0 {
		return uuid.Nil, fmt.Errorf("stream entry %q not found", streamID)
	}

	payloadText := strings.TrimSpace(jsonBString(streamMessages[0].Values["payload"]))
	if payloadText == "" {
		return uuid.Nil, fmt.Errorf("stream entry %q has empty payload", streamID)
	}

	var job queue.InboundMediaJob
	if err := json.Unmarshal([]byte(payloadText), &job); err != nil {
		return uuid.Nil, fmt.Errorf("decode payload: %w", err)
	}
	if job.MessageID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("stream entry %q payload missing message_id", streamID)
	}

	return job.MessageID, nil
}

func loadActivePendingInboundMediaMessageIDs(
	ctx context.Context,
	rdb *redis.Client,
	queueState inboundMediaQueueGroupState,
) (map[uuid.UUID]struct{}, error) {
	activePendingIDs := make(map[uuid.UUID]struct{})
	if rdb == nil || queueState.Pending <= 0 {
		return activePendingIDs, nil
	}

	start := "-"
	for {
		pendingEntries, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: queue.InboundMediaStreamName,
			Group:  queue.InboundMediaConsumerGroup,
			Start:  start,
			End:    "+",
			Count:  inboundMediaPendingBatchSize,
		}).Result()
		if err != nil {
			return nil, fmt.Errorf("load pending inbound-media jobs: %w", err)
		}
		if len(pendingEntries) == 0 {
			return activePendingIDs, nil
		}

		for _, pendingEntry := range pendingEntries {
			messageID, err := loadPendingInboundMediaMessageID(ctx, rdb, pendingEntry.ID)
			if err != nil {
				return nil, fmt.Errorf("resolve pending inbound-media job %q: %w", pendingEntry.ID, err)
			}
			activePendingIDs[messageID] = struct{}{}
		}

		if len(pendingEntries) < int(inboundMediaPendingBatchSize) {
			return activePendingIDs, nil
		}
		start = "(" + pendingEntries[len(pendingEntries)-1].ID
	}
}

func filterQueuedInboundMediaRows(
	rows []inboundMediaQueuedMessage,
	activePendingIDs map[uuid.UUID]struct{},
) ([]inboundMediaQueuedMessage, int64) {
	if len(rows) == 0 || len(activePendingIDs) == 0 {
		return rows, 0
	}

	filtered := make([]inboundMediaQueuedMessage, 0, len(rows))
	var skipped int64
	for _, row := range rows {
		if _, ok := activePendingIDs[row.ID]; ok {
			skipped++
			continue
		}
		filtered = append(filtered, row)
	}

	return filtered, skipped
}

func ReconcileStaleQueuedInboundMedia(
	ctx context.Context,
	db *gorm.DB,
	rdb *redis.Client,
	opts InboundMediaReconcileOptions,
	logger logf.Logger,
) (*InboundMediaReconcileSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	if rdb == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	if opts.OlderThan <= 0 {
		opts.OlderThan = defaultInboundMediaReconcileOlderThan
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}

	queueState, err := loadInboundMediaQueueGroupState(ctx, rdb)
	if err != nil {
		return nil, err
	}
	if err := queueState.validate(opts.AllowActiveQueue); err != nil {
		return nil, err
	}

	summary := &InboundMediaReconcileSummary{
		DryRun:       !opts.Apply,
		Cutoff:       opts.Now.Add(-opts.OlderThan),
		QueuePending: queueState.Pending,
		QueueLag:     queueState.Lag,
	}

	activePendingIDs, err := loadActivePendingInboundMediaMessageIDs(ctx, rdb, queueState)
	if err != nil {
		return nil, err
	}
	summary.ActivePendingIDs = int64(len(activePendingIDs))

	baseQuery := db.WithContext(ctx).
		Model(&models.Message{}).
		Where("direction = ?", models.DirectionIncoming).
		Where("message_type IN ?", inboundMediaRecoverableTypes).
		Where("coalesce(media_url, '') = ''").
		Where(fmt.Sprintf("metadata ->> '%s' = ?", inboundMediaAsyncStatusKey), inboundMediaAsyncStatusQueued)

	if opts.InstanceID != nil {
		baseQuery = baseQuery.Where("instance_id = ?", *opts.InstanceID)
	}

	if err := baseQuery.Count(&summary.TotalQueued).Error; err != nil {
		return nil, fmt.Errorf("count queued inbound-media rows: %w", err)
	}

	eligibleQuery := baseQuery.
		Where("updated_at <= ?", summary.Cutoff).
		Order("updated_at ASC")

	if opts.Limit > 0 {
		eligibleQuery = eligibleQuery.Limit(opts.Limit)
	}

	var rows []inboundMediaQueuedMessage
	if err := eligibleQuery.Select("id", "metadata", "updated_at").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load queued inbound-media rows: %w", err)
	}

	rows, summary.SkippedActiveQueued = filterQueuedInboundMediaRows(rows, activePendingIDs)

	summary.EligibleQueued = int64(len(rows))
	for i := 0; i < len(rows) && i < 10; i++ {
		summary.SampleIDs = append(summary.SampleIDs, rows[i].ID.String())
	}

	if !opts.Apply || len(rows) == 0 {
		return summary, nil
	}

	now := opts.Now
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			nextMetadata, _, nextErrorMessage := buildStaleInboundMediaFailure(row.Metadata)
			if err := tx.Model(&models.Message{}).
				Where("id = ?", row.ID).
				Updates(map[string]any{
					"metadata":      nextMetadata,
					"error_message": nextErrorMessage,
					"updated_at":    now,
				}).Error; err != nil {
				return fmt.Errorf("update stale queued inbound-media row %s: %w", row.ID, err)
			}
			summary.Updated++
		}
		return nil
	}); err != nil {
		return nil, err
	}

	logger.Info(
		"Reconciled stale queued inbound-media rows",
		"updated", summary.Updated,
		"cutoff", summary.Cutoff.Format(time.RFC3339),
		"queue_pending", summary.QueuePending,
		"queue_lag", summary.QueueLag,
	)

	return summary, nil
}
