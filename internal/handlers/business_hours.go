package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/contactutil"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/zerodha/fastglue"
)

// Business hours: an account-level "we're closed" auto-reply. Inbound 1:1
// messages arriving outside the configured window get the away message once
// per contact per cooldown, so customers writing at night know when to expect
// an answer without the reply loop escalating.

// awayReplyCooldown bounds how often one contact receives the away message
// while the account is closed. 12h covers an overnight closure with one
// reply instead of one per message.
const awayReplyCooldown = 12 * time.Hour

// businessHoursSettings is the account's business_hours settings block.
type businessHoursSettings struct {
	Enabled bool `json:"enabled"`
	// StartTime / EndTime are "HH:MM" in the account's local time. A window
	// that crosses midnight (start > end, e.g. 20:00→06:00) is supported.
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	// Days lists open weekdays, 0=Sunday .. 6=Saturday (JS getDay order —
	// matches the frontend day pickers). Empty = every day.
	Days []int `json:"days"`
	// UtcOffsetMin is the account's local UTC offset in minutes
	// (e.g. Egypt standard time = 120, DST = 180).
	UtcOffsetMin int    `json:"utc_offset_min"`
	AwayMessage  string `json:"away_message"`
}

// businessHoursForAccount parses the account's business_hours block.
// Absent or unparsable blocks disable the feature.
func businessHoursForAccount(account *models.WhatsAppAccount) businessHoursSettings {
	bh := businessHoursSettings{}
	block, _ := account.Settings["business_hours"].(map[string]any)
	if block == nil {
		return bh
	}
	if v, ok := block["enabled"].(bool); ok {
		bh.Enabled = v
	}
	if v, ok := block["start_time"].(string); ok {
		bh.StartTime = v
	}
	if v, ok := block["end_time"].(string); ok {
		bh.EndTime = v
	}
	if days, ok := block["days"].([]any); ok {
		for _, d := range days {
			if n, ok := d.(float64); ok && n >= 0 && n <= 6 {
				bh.Days = append(bh.Days, int(n))
			}
		}
	}
	if v, ok := block["utc_offset_min"].(float64); ok {
		bh.UtcOffsetMin = int(v)
	}
	if v, ok := block["away_message"].(string); ok {
		bh.AwayMessage = strings.TrimSpace(v)
	}
	return bh
}

// parseClockMinutes converts "HH:MM" to minutes past midnight. ok=false when
// the value is not a valid 24h clock time.
func parseClockMinutes(s string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// isWithinBusinessHours reports whether the given UTC instant falls inside
// the open window. Pure function — table-testable. A window crossing
// midnight (start > end) spans two local days.
func isWithinBusinessHours(bh businessHoursSettings, nowUTC time.Time) bool {
	if !bh.Enabled {
		return false
	}
	start, okS := parseClockMinutes(bh.StartTime)
	end, okE := parseClockMinutes(bh.EndTime)
	if !okS || !okE || start == end {
		// Degenerate/invalid window: treat as in-hours (no away replies).
		// A misconfigured block must fail silent — never spam every customer
		// with away messages 24/7.
		return true
	}
	local := nowUTC.UTC().Add(time.Duration(bh.UtcOffsetMin) * time.Minute)
	if len(bh.Days) > 0 {
		open := false
		for _, d := range bh.Days {
			if int(local.Weekday()) == d {
				open = true
				break
			}
		}
		if !open {
			return false
		}
	}
	cur := local.Hour()*60 + local.Minute()
	if start < end {
		return cur >= start && cur < end
	}
	// Overnight window: [start..24:00) of today or [00:00..end) of tomorrow.
	// For the day-of-week check an overnight window counts on its start day.
	return cur >= start || cur < end
}

// maybeSendAwayReply sends the account's away message for an inbound 1:1
// message that arrived while closed. Guards: settings enabled + message set,
// outside the window, one reply per contact per cooldown (Redis; without
// Redis every inbound message gets one reply, which is still loop-safe —
// away replies are outgoing and never re-enter the inbound path).
func (a *App) maybeSendAwayReply(account *models.WhatsAppAccount, fromPhone, profileName string) {
	bh := businessHoursForAccount(account)
	if !bh.Enabled || bh.AwayMessage == "" || fromPhone == "" {
		return
	}
	if isWithinBusinessHours(bh, time.Now()) {
		return
	}
	contact, _, err := contactutil.GetOrCreateContact(a.DB, account.OrganizationID, fromPhone, profileName)
	if err != nil || contact == nil {
		a.Log.Error("Business hours: failed to load contact for away reply",
			"error", err, "phone", fromPhone)
		return
	}
	if contact.WhatsAppAccount == "" {
		if err := contactutil.StampAccountName(a.DB, contact, account.Name); err != nil {
			a.Log.Error("Business hours: failed to stamp account on contact",
				"error", err, "contact_id", contact.ID)
		}
	}
	if a.Redis != nil {
		key := fmt.Sprintf("away_reply:%s:%s", account.ID, contact.ID)
		res := a.Redis.SetNX(context.Background(), key, "1", awayReplyCooldown)
		if res.Err() != nil {
			// Fail open: still reply (bounded by direction — no loop).
			a.Log.Warn("Business hours: cooldown check failed; replying anyway",
				"error", res.Err(), "contact_id", contact.ID)
		} else if !res.Val() {
			return // already replied within the cooldown
		}
	}
	if err := a.sendAndSaveTextMessage(account, contact, bh.AwayMessage); err != nil {
		a.Log.Error("Business hours: failed to send away reply",
			"error", err, "contact_id", contact.ID, "account", account.Name)
	}
}

// GetBusinessHoursSettings returns the account's business-hours block.
// GET /api/accounts/{id}/business-hours
func (a *App) GetBusinessHoursSettings(r *fastglue.Request) error {
	account, ok := a.getAccountSettingsBlock(r)
	if !ok {
		return nil
	}
	bh := businessHoursForAccount(account)
	return r.SendEnvelope(bh)
}

// UpdateBusinessHoursSettings replaces the account's business-hours block.
// PUT /api/accounts/{id}/business-hours
func (a *App) UpdateBusinessHoursSettings(r *fastglue.Request) error {
	return a.updateAccountSettingsBlock(r, accountSettingsBlock{
		Key:      "business_hours",
		Resource: models.ResourceAccounts,
		Decode: func(body []byte) (map[string]any, error) {
			var req businessHoursSettings
			if err := decodeJSONSettingsBody(body, &req); err != nil {
				return nil, err
			}
			req.AwayMessage = strings.TrimSpace(req.AwayMessage)
			if _, ok := parseClockMinutes(req.StartTime); !ok {
				return nil, fmt.Errorf("start_time must be HH:MM (24h)")
			}
			if _, ok := parseClockMinutes(req.EndTime); !ok {
				return nil, fmt.Errorf("end_time must be HH:MM (24h)")
			}
			if req.UtcOffsetMin < -12*60 || req.UtcOffsetMin > 14*60 {
				return nil, fmt.Errorf("utc_offset_min must be between -720 and 840")
			}
			seen := map[int]bool{}
			days := make([]any, 0, len(req.Days))
			for _, d := range req.Days {
				if d < 0 || d > 6 || seen[d] {
					continue
				}
				seen[d] = true
				days = append(days, d)
			}
			return map[string]any{
				"enabled":        req.Enabled,
				"start_time":     req.StartTime,
				"end_time":       req.EndTime,
				"days":           days,
				"utc_offset_min": req.UtcOffsetMin,
				"away_message":   req.AwayMessage,
			}, nil
		},
	})
}
