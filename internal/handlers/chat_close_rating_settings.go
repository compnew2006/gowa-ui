package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// CloseRatingSettingsResponse is the effective close-rating configuration
// (account overrides merged over the built-in defaults) as shown in the UI.
type CloseRatingSettingsResponse struct {
	Enabled     bool           `json:"enabled"`
	WindowHours int            `json:"window_hours"`
	Prompt      string         `json:"prompt"`
	Thanks      string         `json:"thanks"`
	Lexicon     map[string]int `json:"lexicon"`
}

// GetCloseRatingSettings returns the effective close-rating settings for a
// WhatsApp account. Per-account because each number belongs to a different
// branch with its own staff, wording and address.
func (a *App) GetCloseRatingSettings(r *fastglue.Request) error {
	account, ok := a.getAccountSettingsBlock(r)
	if !ok {
		return nil
	}

	s := closeRatingSettingsForAccount(account)
	lexicon := s.Lexicon
	if lexicon == nil {
		lexicon = map[string]int{}
	}
	return r.SendEnvelope(CloseRatingSettingsResponse{
		Enabled:     s.Enabled,
		WindowHours: s.WindowHours,
		Prompt:      s.Prompt,
		Thanks:      s.Thanks,
		Lexicon:     lexicon,
	})
}

// UpdateCloseRatingSettings replaces the account's close_rating settings
// block. The frontend always sends the full block, so this is a full
// replacement — no per-field partial-update semantics.
func (a *App) UpdateCloseRatingSettings(r *fastglue.Request) error {
	return a.updateAccountSettingsBlock(r, accountSettingsBlock{
		Key:      "close_rating",
		Resource: models.ResourceSettingsCloseRating,
		Decode: func(body []byte) (map[string]any, error) {
			var req struct {
				Enabled     bool           `json:"enabled"`
				WindowHours int            `json:"window_hours"`
				Prompt      string         `json:"prompt"`
				Thanks      string         `json:"thanks"`
				Lexicon     map[string]int `json:"lexicon"`
			}
			if err := decodeJSONSettingsBody(body, &req); err != nil {
				return nil, err
			}

			if req.WindowHours < 1 || req.WindowHours > maxCloseRatingWindowHours {
				return nil, errors.New("window_hours must be between 1 and 720")
			}
			lexicon := map[string]any{}
			for word, rating := range req.Lexicon {
				if rating < 1 || rating > 5 {
					return nil, errors.New("Lexicon ratings must be between 1 and 5")
				}
				if w := strings.TrimSpace(word); w != "" {
					lexicon[w] = rating
				}
			}

			// Empty prompt falls back to the built-in default at send time;
			// an empty thanks explicitly disables the thank-you message
			// (parse-side contract).
			return map[string]any{
				"enabled":      req.Enabled,
				"window_hours": req.WindowHours,
				"prompt":       strings.TrimSpace(req.Prompt),
				"thanks":       strings.TrimSpace(req.Thanks),
				"lexicon":      lexicon,
			}, nil
		},
	})
}

// CloseRatingStatsResponse aggregates one account's rating cycles for the UI.
type CloseRatingStatsResponse struct {
	Total        int64            `json:"total"`
	Pending      int64            `json:"pending"`
	Rated        int64            `json:"rated"`
	Expired      int64            `json:"expired"`
	Average      float64          `json:"average"`
	ResponseRate float64          `json:"response_rate"`
	Distribution map[string]int64 `json:"distribution"`
}

// GetCloseRatingStats returns aggregate CSAT figures for one WhatsApp account.
// Cycles reference the account by name (the usual soft string reference), so
// the account row is loaded first to resolve its name.
func (a *App) GetCloseRatingStats(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceAccounts, models.ActionRead)
	if err != nil {
		return nil
	}

	id, err := parsePathUUID(r, "id", "account")
	if err != nil {
		return nil
	}

	account, err := findByIDAndOrg[models.WhatsAppAccount](a.DB, r, id, orgID, "Account")
	if err != nil {
		return nil
	}

	var statusCounts []struct {
		Status string
		Count  int64
	}
	if err := a.DB.Model(&models.ChatClosureRating{}).
		Select("status, COUNT(*) AS count").
		Where("organization_id = ? AND whats_app_account = ?", orgID, account.Name).
		Group("status").Scan(&statusCounts).Error; err != nil {
		a.Log.Error("Failed to load close-rating stats", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load stats", nil, "")
	}

	stats := CloseRatingStatsResponse{
		Distribution: map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
	}
	for _, sc := range statusCounts {
		stats.Total += sc.Count
		switch sc.Status {
		case models.RatingStatusPending:
			stats.Pending = sc.Count
		case models.RatingStatusRated:
			stats.Rated = sc.Count
		case models.RatingStatusExpired:
			stats.Expired = sc.Count
		}
	}

	var dist []struct {
		Rating int
		Count  int64
	}
	if err := a.DB.Model(&models.ChatClosureRating{}).
		Select("rating, COUNT(*) AS count").
		Where("organization_id = ? AND whats_app_account = ? AND status = ? AND rating IS NOT NULL",
			orgID, account.Name, models.RatingStatusRated).
		Group("rating").Scan(&dist).Error; err != nil {
		a.Log.Error("Failed to load close-rating distribution", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load stats", nil, "")
	}

	var sum, rated int64
	for _, d := range dist {
		if d.Rating < 1 || d.Rating > 5 {
			continue
		}
		stats.Distribution[string(rune('0'+d.Rating))] = d.Count
		sum += int64(d.Rating) * d.Count
		rated += d.Count
	}
	if rated > 0 {
		stats.Average = float64(sum) / float64(rated)
	}
	if stats.Total > 0 {
		stats.ResponseRate = float64(stats.Rated) / float64(stats.Total) * 100
	}

	return r.SendEnvelope(stats)
}
