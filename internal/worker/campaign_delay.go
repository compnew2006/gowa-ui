package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	campaignDelayKeyPrefix      = "whatomate:campaign:delay:"
	campaignDelayReservationTTL = 24 * time.Hour
)

var reserveCampaignDelaySlotScript = redis.NewScript(`
local key = KEYS[1]
local now_ms = tonumber(ARGV[1])
local gap_ms = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

local current_next = tonumber(redis.call("GET", key) or "0")
local send_at = now_ms
if current_next > send_at then
	send_at = current_next
end

local next_at = send_at + gap_ms
redis.call("PSETEX", key, ttl_ms, tostring(next_at))
return send_at
`)

func campaignDelayRedisKey(campaignID uuid.UUID) string {
	return campaignDelayKeyPrefix + campaignID.String()
}

func (w *Worker) applyCampaignSendDelay(ctx context.Context, campaignID uuid.UUID, minDelaySeconds, maxDelaySeconds int) error {
	if minDelaySeconds < 0 {
		minDelaySeconds = 0
	}
	if maxDelaySeconds < 0 {
		maxDelaySeconds = 0
	}
	if maxDelaySeconds < minDelaySeconds {
		maxDelaySeconds = minDelaySeconds
	}
	if minDelaySeconds == 0 && maxDelaySeconds == 0 {
		return nil
	}

	gapMs, err := randomDelayMilliseconds(minDelaySeconds, maxDelaySeconds)
	if err != nil {
		return err
	}
	if gapMs <= 0 {
		return nil
	}

	if w.Redis == nil {
		return sleepWithContext(ctx, time.Duration(gapMs)*time.Millisecond)
	}

	nowMs := time.Now().UnixMilli()
	ttlMs := int64(campaignDelayReservationTTL / time.Millisecond)
	rawSendAt, err := reserveCampaignDelaySlotScript.Run(
		ctx,
		w.Redis,
		[]string{campaignDelayRedisKey(campaignID)},
		nowMs,
		gapMs,
		ttlMs,
	).Result()
	if err != nil {
		w.Log.Warn("Failed to reserve campaign delay slot, falling back to local delay", "campaign_id", campaignID, "error", err)
		return sleepWithContext(ctx, time.Duration(gapMs)*time.Millisecond)
	}

	sendAtMs, err := parseScriptResultInt64(rawSendAt)
	if err != nil {
		w.Log.Warn("Failed to parse reserved campaign delay slot, falling back to local delay", "campaign_id", campaignID, "error", err)
		return sleepWithContext(ctx, time.Duration(gapMs)*time.Millisecond)
	}

	waitMs := sendAtMs - nowMs
	if waitMs <= 0 {
		return nil
	}

	return sleepWithContext(ctx, time.Duration(waitMs)*time.Millisecond)
}

func randomDelayMilliseconds(minDelaySeconds, maxDelaySeconds int) (int64, error) {
	if minDelaySeconds < 0 {
		minDelaySeconds = 0
	}
	if maxDelaySeconds < minDelaySeconds {
		maxDelaySeconds = minDelaySeconds
	}

	span := maxDelaySeconds - minDelaySeconds + 1
	if span <= 1 {
		return int64(minDelaySeconds) * int64(time.Second/time.Millisecond), nil
	}

	randomValue, err := rand.Int(rand.Reader, big.NewInt(int64(span)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random delay: %w", err)
	}

	selectedSeconds := minDelaySeconds + int(randomValue.Int64())
	return int64(selectedSeconds) * int64(time.Second/time.Millisecond), nil
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseScriptResultInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected script result type %T", value)
	}
}
