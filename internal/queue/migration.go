package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/logf"
)

const (
	campaignMigrationGroup   = "campaign-migration"
	campaignMigrationLockKey = StreamName + ":migration:lock"
	campaignMigrationTTL     = 5 * time.Minute
	maxMigrationSamples      = 10
)

// CampaignMigrationOptions controls how legacy campaign stream migration runs.
type CampaignMigrationOptions struct {
	Apply     bool
	BatchSize int64
	LockTTL   time.Duration
}

// CampaignMigrationSummary describes a legacy campaign stream migration run.
type CampaignMigrationSummary struct {
	DryRun             bool
	LegacyStreamExists bool
	ConsumerGroupFound bool
	TemporaryGroupUsed bool
	Unread             int64
	Pending            int64
	Migrated           int64
	Invalid            int64
	Skipped            int64
	InvalidMessageIDs  []string
	MigratedMessageIDs []string
}

// MigrateLegacyCampaignStream redistributes legacy global campaign jobs into tenant streams.
func MigrateLegacyCampaignStream(ctx context.Context, client *redis.Client, log logf.Logger, opts CampaignMigrationOptions) (*CampaignMigrationSummary, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if opts.LockTTL <= 0 {
		opts.LockTTL = campaignMigrationTTL
	}

	summary := &CampaignMigrationSummary{DryRun: !opts.Apply}

	exists, err := client.Exists(ctx, StreamName).Result()
	if err != nil {
		return nil, fmt.Errorf("check legacy campaign stream existence: %w", err)
	}
	if exists == 0 {
		return summary, nil
	}
	summary.LegacyStreamExists = true

	groupInfo, found, err := loadLegacyConsumerGroupInfo(ctx, client)
	if err != nil {
		return nil, err
	}
	summary.ConsumerGroupFound = found

	if !opts.Apply {
		if found {
			if err := inspectUnreadLegacyMessages(ctx, client, groupInfo.LastDeliveredID, opts.BatchSize, summary); err != nil {
				return nil, err
			}
			if err := inspectPendingLegacyMessages(ctx, client, ConsumerGroup, opts.BatchSize, summary); err != nil {
				return nil, err
			}
		} else {
			if err := inspectEntireLegacyStream(ctx, client, opts.BatchSize, summary); err != nil {
				return nil, err
			}
		}
		return summary, nil
	}

	lockToken, err := acquireCampaignMigrationLock(ctx, client, opts.LockTTL)
	if err != nil {
		return nil, err
	}
	defer releaseCampaignMigrationLock(context.Background(), client, lockToken)

	groupName := ConsumerGroup
	if !found {
		if err := client.XGroupCreateMkStream(ctx, StreamName, campaignMigrationGroup, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return nil, fmt.Errorf("create migration group on legacy stream: %w", err)
		}
		groupName = campaignMigrationGroup
		summary.TemporaryGroupUsed = true
		defer func() {
			_ = client.XGroupDestroy(context.Background(), StreamName, campaignMigrationGroup).Err()
		}()
	}

	migrationConsumerID := fmt.Sprintf("campaign-migrator-%s", uuid.NewString())

	if err := applyUnreadLegacyMessages(ctx, client, log, groupName, migrationConsumerID, opts.BatchSize, summary); err != nil {
		return nil, err
	}
	if found {
		if err := applyPendingLegacyMessages(ctx, client, log, groupName, migrationConsumerID, opts.BatchSize, summary); err != nil {
			return nil, err
		}
	}

	return summary, nil
}

func loadLegacyConsumerGroupInfo(ctx context.Context, client *redis.Client) (*redis.XInfoGroup, bool, error) {
	groups, err := client.XInfoGroups(ctx, StreamName).Result()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such key") {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("load legacy campaign consumer groups: %w", err)
	}

	for _, group := range groups {
		if group.Name == ConsumerGroup {
			groupCopy := group
			return &groupCopy, true, nil
		}
	}
	return nil, false, nil
}

func inspectEntireLegacyStream(ctx context.Context, client *redis.Client, batchSize int64, summary *CampaignMigrationSummary) error {
	start := "-"
	for {
		messages, err := client.XRangeN(ctx, StreamName, start, "+", batchSize).Result()
		if err != nil {
			return fmt.Errorf("inspect legacy campaign stream: %w", err)
		}
		if len(messages) == 0 {
			return nil
		}

		for _, msg := range messages {
			summary.Unread++
			classifyLegacyMessage(msg, summary)
		}

		if len(messages) < int(batchSize) {
			return nil
		}
		start = "(" + messages[len(messages)-1].ID
	}
}

func inspectUnreadLegacyMessages(ctx context.Context, client *redis.Client, lastDeliveredID string, batchSize int64, summary *CampaignMigrationSummary) error {
	start := "(" + lastDeliveredID
	if strings.TrimSpace(lastDeliveredID) == "" {
		start = "-"
	}

	for {
		messages, err := client.XRangeN(ctx, StreamName, start, "+", batchSize).Result()
		if err != nil {
			return fmt.Errorf("inspect unread legacy campaign messages: %w", err)
		}
		if len(messages) == 0 {
			return nil
		}

		for _, msg := range messages {
			summary.Unread++
			classifyLegacyMessage(msg, summary)
		}

		if len(messages) < int(batchSize) {
			return nil
		}
		start = "(" + messages[len(messages)-1].ID
	}
}

func inspectPendingLegacyMessages(ctx context.Context, client *redis.Client, groupName string, batchSize int64, summary *CampaignMigrationSummary) error {
	start := "-"
	for {
		pendingEntries, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: StreamName,
			Group:  groupName,
			Start:  start,
			End:    "+",
			Count:  batchSize,
		}).Result()
		if err != nil {
			return fmt.Errorf("inspect pending legacy campaign messages: %w", err)
		}
		if len(pendingEntries) == 0 {
			return nil
		}

		for _, entry := range pendingEntries {
			summary.Pending++
			msg, found, err := loadLegacyMessageByID(ctx, client, entry.ID)
			if err != nil {
				return err
			}
			if !found {
				summary.Skipped++
				continue
			}
			classifyLegacyMessage(msg, summary)
		}

		if len(pendingEntries) < int(batchSize) {
			return nil
		}
		start = "(" + pendingEntries[len(pendingEntries)-1].ID
	}
}

func applyUnreadLegacyMessages(ctx context.Context, client *redis.Client, log logf.Logger, groupName, consumerName string, batchSize int64, summary *CampaignMigrationSummary) error {
	for {
		results, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{StreamName, ">"},
			Count:    batchSize,
			Block:    -1,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				return nil
			}
			return fmt.Errorf("read unread legacy campaign messages: %w", err)
		}
		if len(results) == 0 {
			return nil
		}

		for _, streamResult := range results {
			for _, msg := range streamResult.Messages {
				summary.Unread++
				if err := migrateLegacyMessage(ctx, client, log, groupName, msg, summary); err != nil {
					return err
				}
			}
		}
	}
}

func applyPendingLegacyMessages(ctx context.Context, client *redis.Client, log logf.Logger, groupName, consumerName string, batchSize int64, summary *CampaignMigrationSummary) error {
	start := "-"
	for {
		pendingEntries, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: StreamName,
			Group:  groupName,
			Start:  start,
			End:    "+",
			Count:  batchSize,
		}).Result()
		if err != nil {
			return fmt.Errorf("load pending legacy campaign messages: %w", err)
		}
		if len(pendingEntries) == 0 {
			return nil
		}

		ids := make([]string, 0, len(pendingEntries))
		for _, entry := range pendingEntries {
			ids = append(ids, entry.ID)
		}

		messages, err := client.XClaim(ctx, &redis.XClaimArgs{
			Stream:   StreamName,
			Group:    groupName,
			Consumer: consumerName,
			MinIdle:  0,
			Messages: ids,
		}).Result()
		if err != nil {
			return fmt.Errorf("claim pending legacy campaign messages: %w", err)
		}

		messageByID := make(map[string]redis.XMessage, len(messages))
		for _, msg := range messages {
			messageByID[msg.ID] = msg
		}

		for _, entry := range pendingEntries {
			summary.Pending++
			msg, ok := messageByID[entry.ID]
			if !ok {
				if err := client.XAck(ctx, StreamName, groupName, entry.ID).Err(); err != nil {
					return fmt.Errorf("ack missing pending legacy message %s: %w", entry.ID, err)
				}
				summary.Skipped++
				continue
			}
			if err := migrateLegacyMessage(ctx, client, log, groupName, msg, summary); err != nil {
				return err
			}
		}

		if len(pendingEntries) < int(batchSize) {
			return nil
		}
		start = "(" + pendingEntries[len(pendingEntries)-1].ID
	}
}

func migrateLegacyMessage(ctx context.Context, client *redis.Client, log logf.Logger, groupName string, msg redis.XMessage, summary *CampaignMigrationSummary) error {
	orgID, err := legacyCampaignMessageOrganizationID(msg)
	if err != nil {
		if ackErr := client.XAck(ctx, StreamName, groupName, msg.ID).Err(); ackErr != nil {
			return fmt.Errorf("ack invalid legacy message %s: %w", msg.ID, ackErr)
		}
		summary.Invalid++
		appendMigrationSample(&summary.InvalidMessageIDs, msg.ID)
		log.Warn("Preserved invalid legacy campaign message during migration", "message_id", msg.ID, "error", err)
		return nil
	}

	values := cloneLegacyValues(msg.Values)
	pipe := client.TxPipeline()
	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: CampaignStreamName(orgID),
		Values: values,
	})
	pipe.XAck(ctx, StreamName, groupName, msg.ID)
	pipe.XDel(ctx, StreamName, msg.ID)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("migrate legacy campaign message %s: %w", msg.ID, err)
	}

	summary.Migrated++
	appendMigrationSample(&summary.MigratedMessageIDs, msg.ID)
	return nil
}

func loadLegacyMessageByID(ctx context.Context, client *redis.Client, messageID string) (redis.XMessage, bool, error) {
	messages, err := client.XRangeN(ctx, StreamName, messageID, messageID, 1).Result()
	if err != nil {
		return redis.XMessage{}, false, fmt.Errorf("load legacy campaign message %s: %w", messageID, err)
	}
	if len(messages) == 0 {
		return redis.XMessage{}, false, nil
	}
	return messages[0], true, nil
}

func classifyLegacyMessage(msg redis.XMessage, summary *CampaignMigrationSummary) {
	if _, err := legacyCampaignMessageOrganizationID(msg); err != nil {
		summary.Invalid++
		appendMigrationSample(&summary.InvalidMessageIDs, msg.ID)
	}
}

func legacyCampaignMessageOrganizationID(msg redis.XMessage) (uuid.UUID, error) {
	jobType, ok := streamStringValue(msg.Values["type"])
	if !ok {
		return uuid.Nil, fmt.Errorf("legacy message missing type")
	}

	payload, ok := streamStringValue(msg.Values["payload"])
	if !ok {
		return uuid.Nil, fmt.Errorf("legacy message missing payload")
	}

	switch JobType(jobType) {
	case JobTypeRecipient:
		var job RecipientJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return uuid.Nil, fmt.Errorf("decode legacy recipient job: %w", err)
		}
		if job.OrganizationID == uuid.Nil {
			return uuid.Nil, fmt.Errorf("legacy recipient job missing organization_id")
		}
		return job.OrganizationID, nil

	case JobTypeContactRepair:
		var job ContactRepairJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			return uuid.Nil, fmt.Errorf("decode legacy contact repair job: %w", err)
		}
		if job.OrganizationID == uuid.Nil {
			return uuid.Nil, fmt.Errorf("legacy contact repair job missing organization_id")
		}
		return job.OrganizationID, nil

	default:
		return uuid.Nil, fmt.Errorf("unsupported legacy job type %q", jobType)
	}
}

func cloneLegacyValues(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func appendMigrationSample(dst *[]string, value string) {
	if len(*dst) >= maxMigrationSamples {
		return
	}
	*dst = append(*dst, value)
}

func acquireCampaignMigrationLock(ctx context.Context, client *redis.Client, ttl time.Duration) (string, error) {
	token := uuid.NewString()
	result, err := client.SetArgs(ctx, campaignMigrationLockKey, token, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	if errors.Is(err, redis.Nil) || result == "" {
		return "", fmt.Errorf("campaign migration lock is already held")
	}
	if err != nil {
		return "", fmt.Errorf("acquire campaign migration lock: %w", err)
	}
	return token, nil
}

func releaseCampaignMigrationLock(ctx context.Context, client *redis.Client, token string) {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`
	_ = client.Eval(ctx, script, []string{campaignMigrationLockKey}, token).Err()
}
