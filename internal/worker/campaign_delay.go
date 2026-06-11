package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	campaignDelayKeyPrefix      = "whatomate:instance:delay:"
	campaignDelayReservationTTL = 24 * time.Hour
	campaignDelayFloorSeconds   = 10
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

func campaignDelayRedisKey(scopeKey string) string {
	normalized := strings.TrimSpace(scopeKey)
	if normalized == "" {
		normalized = "default"
	}
	return campaignDelayKeyPrefix + normalized
}

func resolveCampaignDelayScopeKey(instanceID string, fallbackCampaignID uuid.UUID) string {
	normalizedInstanceID := strings.TrimSpace(instanceID)
	if normalizedInstanceID != "" {
		return normalizedInstanceID
	}
	if fallbackCampaignID != uuid.Nil {
		return fallbackCampaignID.String()
	}
	return "default"
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

func normalizeCampaignDelaySeconds(minDelaySeconds, maxDelaySeconds int) (int, int) {
	if minDelaySeconds < 0 {
		minDelaySeconds = 0
	}
	if maxDelaySeconds < 0 {
		maxDelaySeconds = 0
	}
	if minDelaySeconds == 0 && maxDelaySeconds == 0 {
		return 0, 0
	}
	if minDelaySeconds < campaignDelayFloorSeconds {
		minDelaySeconds = campaignDelayFloorSeconds
	}
	if maxDelaySeconds < campaignDelayFloorSeconds {
		maxDelaySeconds = campaignDelayFloorSeconds
	}
	if maxDelaySeconds < minDelaySeconds {
		maxDelaySeconds = minDelaySeconds
	}
	return minDelaySeconds, maxDelaySeconds
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
