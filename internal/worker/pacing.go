package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Campaign send pacing protects WhatsApp accounts from provider bans: the
// unofficial device API flags numbers that burst hundreds of messages, so
// every campaign send reserves a slot from a per-account fixed window before
// talking to the provider.
//
// Rate resolution (messages per minute):
//  1. the account's send_pacing settings block (messages_per_minute > 0),
//  2. the [campaigns].default_pacing_per_minute config value,
//  3. 0 = unlimited (the historical behavior — pacing is opt-in).
const (
	paceWindow           = time.Minute
	paceMaxWaitPerReturn = 30 * time.Second // sleep per turn while the window is full
	paceMaxWaitTotal     = 10 * time.Minute // hard cap: fail-open past this
)

// paceKey is the Redis fixed-window counter for one account's campaign sends.
func paceKey(orgID uuid.UUID, accountName string) string {
	return fmt.Sprintf("campaign_pace:%s:%s", orgID, accountName)
}

// accountPacePerMinute reads the per-minute send budget for an account from
// its settings block, falling back to the configured default. 0 = unlimited.
func (w *Worker) accountPacePerMinute(settings map[string]any) int {
	if block, ok := settings["send_pacing"].(map[string]any); ok {
		if v, ok := block["messages_per_minute"].(float64); ok && int(v) > 0 {
			return int(v)
		}
	}
	if w.Config != nil && w.Config.Campaigns.DefaultPacingPerMinute > 0 {
		return w.Config.Campaigns.DefaultPacingPerMinute
	}
	return 0
}

// paceCampaignSend blocks until the account's per-minute send budget has a
// free slot. It fails open: without a Redis client, on Redis errors, or after
// paceMaxWaitTotal of waiting, the send proceeds unpaced — a stalled campaign
// is preferable to a dead one, and the historical behavior was no pacing.
func (w *Worker) paceCampaignSend(ctx context.Context, orgID uuid.UUID, accountName string, perMinute int) {
	if perMinute <= 0 || w.Redis == nil {
		return
	}
	key := paceKey(orgID, accountName)
	deadline := time.Now().Add(paceMaxWaitTotal)
	for {
		ok, wait, err := w.reservePaceSlot(ctx, key, perMinute)
		if err != nil {
			// Fail open on Redis hiccups (matches the middleware convention).
			w.Log.Warn("Campaign pacing check failed; sending unpaced", "error", err, "account", accountName)
			return
		}
		if ok {
			return
		}
		if time.Now().Add(wait).After(deadline) {
			w.Log.Warn("Campaign pacing wait exceeded cap; sending unpaced",
				"account", accountName, "waited", paceMaxWaitTotal)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			// window rolled over — try to reserve again
		}
	}
}

// reservePaceSlot increments the fixed-window counter and reports whether a
// send slot was granted; when the window is full it returns how long to wait.
func (w *Worker) reservePaceSlot(ctx context.Context, key string, perMinute int) (ok bool, wait time.Duration, err error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	count, err := w.Redis.Incr(cctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := w.Redis.Expire(cctx, key, paceWindow).Err(); err != nil {
			return false, 0, err
		}
		return true, 0, nil
	}
	if count <= int64(perMinute) {
		return true, 0, nil
	}
	ttl, err := w.Redis.TTL(cctx, key).Result()
	if err != nil || ttl < 0 {
		ttl = paceWindow // unknown TTL — assume a full window
	}
	return false, ttl, nil
}
