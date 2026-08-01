package handlers

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParseRatingReply(t *testing.T) {
	orgLexicon := map[string]int{
		"يجنن": 5,
		"تمام": 2, // org override beats the default (تمام=4)
	}

	tests := []struct {
		name       string
		text       string
		lexicon    map[string]int
		wantRating int
		wantOK     bool
	}{
		// Latin digits
		{"latin digit 5", "5", nil, 5, true},
		{"latin digit 1", "1", nil, 1, true},
		{"latin digit with spaces", "  3  ", nil, 3, true},
		{"zero rejected", "0", nil, 0, false},
		{"six rejected", "6", nil, 0, false},
		{"ten rejected", "10", nil, 0, false},

		// Arabic-Indic digits
		{"arabic-indic 5", "٥", nil, 5, true},
		{"arabic-indic 1", "١", nil, 1, true},
		{"extended arabic-indic 4", "۴", nil, 4, true},
		{"arabic-indic zero rejected", "٠", nil, 0, false},

		// N/5 shape
		{"four out of five", "4/5", nil, 4, true},
		{"five out of five spaced", "5 / 5", nil, 5, true},
		{"arabic-indic out of five", "٣/٥", nil, 3, true},
		{"wrong denominator rejected", "4/10", nil, 0, false},

		// Digits inside longer sentences must NOT be captured
		{"digit in sentence rejected", "ممكن اطلب 3 قطع", nil, 0, false},
		{"digit in english sentence rejected", "call me at 5", nil, 0, false},

		// Star emoji
		{"five stars", "⭐⭐⭐⭐⭐", nil, 5, true},
		{"three stars spaced", "⭐ ⭐ ⭐", nil, 3, true},
		{"two glyph stars", "★★", nil, 2, true},
		{"glowing star", "🌟", nil, 1, true},
		{"stars with variation selector", "⭐\ufe0f⭐\ufe0f", nil, 2, true},
		{"six stars rejected", "⭐⭐⭐⭐⭐⭐", nil, 0, false},
		// Not a star-count — but the edge-trimmed lexicon word still matches,
		// same as "ممتاز 👍" below.
		{"star plus lexicon word", "⭐ great", nil, 5, true},

		// Default lexicon — Arabic and Egyptian colloquial
		{"mumtaz", "ممتاز", nil, 5, true},
		{"batal", "بطل", nil, 5, true},
		{"exceeded expectations", "تفوق التوقعات", nil, 5, true},
		{"tamam default", "تمام", nil, 4, true},
		{"3adi", "عادي", nil, 3, true},
		{"mesh wala bod", "مش ولا بد", nil, 2, true},
		{"zift", "زفت", nil, 1, true},
		{"sayye2", "سيء", nil, 1, true},

		// Default lexicon — English, case-insensitive
		{"excellent", "Excellent", nil, 5, true},
		{"ok upper", "OK", nil, 3, true},
		{"bad", "bad", nil, 2, true},

		// Trailing punctuation / emoji around lexicon words
		{"mumtaz with emoji", "ممتاز 👍", nil, 5, true},
		{"mumtaz with punctuation", "ممتاز!!", nil, 5, true},

		// Org lexicon overrides and extends the defaults
		{"org custom word", "يجنن", orgLexicon, 5, true},
		{"org override wins", "تمام", orgLexicon, 2, true},
		{"default still reachable with org lexicon", "ممتاز", orgLexicon, 5, true},

		// Unmatched free text
		{"free text rejected", "الخدمة كانت بطيئة شوية بس الموظف محترم", nil, 0, false},
		{"empty rejected", "", nil, 0, false},
		{"whitespace rejected", "   ", nil, 0, false},
		{"emoji only rejected", "👍", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rating, ok := parseRatingReply(tt.text, tt.lexicon)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantRating, rating)
		})
	}
}

func TestNormalizeRatingDigits(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"arabic-indic", "٠١٢٣٤٥٦٧٨٩", "0123456789"},
		{"extended arabic-indic", "۰۱۲۳۴۵۶۷۸۹", "0123456789"},
		{"ascii untouched", "12345", "12345"},
		{"mixed text", "تقييمي ٥", "تقييمي 5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeRatingDigits(tt.in))
		})
	}
}

func TestParseCloseRatingSettings(t *testing.T) {
	defaults := closeRatingSettings{
		WindowHours: defaultCloseRatingWindowHours,
		Prompt:      defaultCloseRatingPrompt,
		Thanks:      defaultCloseRatingThanks,
	}

	tests := []struct {
		name  string
		in    models.JSONB
		check func(t *testing.T, s closeRatingSettings)
	}{
		{
			name: "missing block keeps defaults and stays disabled",
			in:   models.JSONB{},
			check: func(t *testing.T, s closeRatingSettings) {
				assert.False(t, s.Enabled)
				assert.Equal(t, defaultCloseRatingWindowHours, s.WindowHours)
				assert.Equal(t, defaultCloseRatingPrompt, s.Prompt)
			},
		},
		{
			name: "full block applied",
			in: models.JSONB{"close_rating": map[string]any{
				"enabled":      true,
				"window_hours": float64(24),
				"prompt":       "قيّمنا من ١ إلى ٥",
				"thanks":       "شكرًا",
				"lexicon":      map[string]any{"يجنن": float64(5)},
			}},
			check: func(t *testing.T, s closeRatingSettings) {
				assert.True(t, s.Enabled)
				assert.Equal(t, 24, s.WindowHours)
				assert.Equal(t, "قيّمنا من ١ إلى ٥", s.Prompt)
				assert.Equal(t, "شكرًا", s.Thanks)
				assert.Equal(t, 5, s.Lexicon["يجنن"])
			},
		},
		{
			name: "window outside bounds keeps default",
			in: models.JSONB{"close_rating": map[string]any{
				"window_hours": float64(100000),
			}},
			check: func(t *testing.T, s closeRatingSettings) {
				assert.Equal(t, defaultCloseRatingWindowHours, s.WindowHours)
			},
		},
		{
			name: "explicit empty thanks disables the thank-you message",
			in: models.JSONB{"close_rating": map[string]any{
				"thanks": "",
			}},
			check: func(t *testing.T, s closeRatingSettings) {
				assert.Equal(t, "", s.Thanks)
			},
		},
		{
			name: "lexicon ratings outside 1-5 are dropped",
			in: models.JSONB{"close_rating": map[string]any{
				"lexicon": map[string]any{"غش": float64(0), "حلو": float64(9), "يجنن": float64(5)},
			}},
			check: func(t *testing.T, s closeRatingSettings) {
				assert.Len(t, s.Lexicon, 1)
				assert.Equal(t, 5, s.Lexicon["يجنن"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, parseCloseRatingSettings(tt.in, defaults))
		})
	}
}
