package handlers

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Chat-close CSAT rating cycle: prompt the customer after their conversation
// is closed, capture the reply deterministically (digits, stars, colloquial
// lexicon) inside the reply window.
//
// Placement rules (deliberate, do not move):
//   - The prompt is triggered from the CloseChat / LeaveChat HTTP handlers —
//     never from the chatlifecycle service, which stays free of messaging.
//   - Capture runs in processIncomingMessageFull STRICTLY BEFORE
//     ensureClaimableChatStatus, so a rating reply does not reopen the chat.

const (
	defaultCloseRatingWindowHours = 48
	maxCloseRatingWindowHours     = 720 // 30 days

	defaultCloseRatingPrompt = "قيّم خدمتنا: أرسل رقمًا من ١ إلى ٥ (٥ = ممتاز)\nRate our service: send a number from 1 to 5 (5 = excellent)"
	defaultCloseRatingThanks = "🙏 شكرًا لتقييمك! Thank you for your feedback!"
)

// defaultRatingLexicon maps common Arabic (incl. Egyptian colloquial) and
// English one-word replies to a 1-5 rating. Keys must be pre-normalized with
// normalizeRatingToken. Org-provided lexicon entries override these.
var defaultRatingLexicon = map[string]int{
	// 5 — enthusiastic praise
	"ممتاز": 5, "بطل": 5, "رائع": 5, "روعة": 5, "جامد": 5, "عظيم": 5,
	"تفوق التوقعات": 5, "فوق الممتاز": 5,
	"excellent": 5, "amazing": 5, "perfect": 5, "great": 5,
	// 4 — good
	"جيد": 4, "جيد جدا": 4, "جيد جداً": 4, "تمام": 4, "كويس": 4, "حلو": 4,
	"good": 4, "very good": 4, "nice": 4,
	// 3 — neutral
	"عادي": 3, "مقبول": 3, "ماشي": 3, "نص نص": 3,
	"ok": 3, "okay": 3, "average": 3,
	// 2 — poor
	"ضعيف": 2, "مش كويس": 2, "مش ولا بد": 2, "تحت المتوسط": 2,
	"bad": 2, "poor": 2,
	// 1 — very poor
	"سيء": 1, "سيئ": 1, "زفت": 1, "زبالة": 1,
	"terrible": 1, "awful": 1, "horrible": 1,
}

// closeRatingSettings is read from WhatsAppAccount.Settings["close_rating"]:
//
//	{"enabled": true, "window_hours": 48, "prompt": "...", "thanks": "...",
//	 "lexicon": {"يجنن": 5}}
//
// Per-account on purpose: each WhatsApp number belongs to a different branch
// with its own staff, address and wording, so the prompt/thanks/lexicon are
// configured on the account detail page, not organization-wide.
// Disabled by default; edited from Settings → Accounts → account → Rating
// (Get/UpdateCloseRatingSettings in chat_close_rating_settings.go).
type closeRatingSettings struct {
	Enabled     bool
	WindowHours int
	Prompt      string
	Thanks      string
	Lexicon     map[string]int
}

func closeRatingSettingsForAccount(account *models.WhatsAppAccount) closeRatingSettings {
	s := closeRatingSettings{
		WindowHours: defaultCloseRatingWindowHours,
		Prompt:      defaultCloseRatingPrompt,
		Thanks:      defaultCloseRatingThanks,
	}
	return parseCloseRatingSettings(account.Settings, s)
}

// parseCloseRatingSettings applies the "close_rating" block of the account
// settings JSONB on top of the defaults. Split out for table-driven tests.
func parseCloseRatingSettings(orgSettings models.JSONB, s closeRatingSettings) closeRatingSettings {
	raw, ok := orgSettings["close_rating"].(map[string]any)
	if !ok {
		return s
	}
	if v, ok := raw["enabled"].(bool); ok {
		s.Enabled = v
	}
	if v, ok := raw["window_hours"].(float64); ok && v >= 1 && v <= maxCloseRatingWindowHours {
		s.WindowHours = int(v)
	}
	if v, ok := raw["prompt"].(string); ok && strings.TrimSpace(v) != "" {
		s.Prompt = v
	}
	if v, ok := raw["thanks"].(string); ok {
		// An explicit empty string disables the thank-you message.
		s.Thanks = strings.TrimSpace(v)
	}
	if m, ok := raw["lexicon"].(map[string]any); ok {
		s.Lexicon = make(map[string]int, len(m))
		for word, rv := range m {
			if f, ok := rv.(float64); ok && f >= 1 && f <= 5 {
				if token := normalizeRatingToken(word); token != "" {
					s.Lexicon[token] = int(f)
				}
			}
		}
	}
	return s
}

// normalizeRatingDigits converts Arabic-Indic (٠-٩) and Extended Arabic-Indic
// (۰-۹) digits to ASCII so "٥" parses the same as "5".
func normalizeRatingDigits(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		}
		return r
	}, s)
}

// normalizeRatingToken lowercases and trims surrounding space, punctuation
// and symbols (emoji) so "ممتاز 👍!!" matches the lexicon key "ممتاز".
// Inner spaces survive, keeping multi-word entries like "تفوق التوقعات".
func normalizeRatingToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r) || r == '\ufe0f'
	})
}

// parseRatingReply extracts a 1-5 rating deterministically. Only whole-reply
// shapes match — a longer sentence that merely contains a digit is NOT a
// rating (it falls through to the normal message flow, and phase 2's LLM
// tail). Recognized shapes: "5" / "٥" / "4/5", star emoji runs, and lexicon
// words (org entries override the defaults).
func parseRatingReply(text string, lexicon map[string]int) (int, bool) {
	t := strings.TrimSpace(normalizeRatingDigits(text))
	if t == "" {
		return 0, false
	}
	if n, ok := matchNumericRating(t); ok {
		return n, true
	}
	if n, ok := matchStarRating(t); ok {
		return n, true
	}
	token := normalizeRatingToken(t)
	if token == "" {
		return 0, false
	}
	if lexicon != nil {
		if n, ok := lexicon[token]; ok {
			return n, true
		}
	}
	if n, ok := defaultRatingLexicon[token]; ok {
		return n, true
	}
	return 0, false
}

// matchNumericRating accepts exactly "N" or "N/5" (N in 1..5), nothing else.
func matchNumericRating(t string) (int, bool) {
	if i := strings.IndexRune(t, '/'); i >= 0 {
		if strings.TrimSpace(t[i+1:]) != "5" {
			return 0, false
		}
		t = strings.TrimSpace(t[:i])
	}
	if len(t) != 1 || t[0] < '1' || t[0] > '5' {
		return 0, false
	}
	return int(t[0] - '0'), true
}

// matchStarRating accepts a reply made exclusively of 1-5 star glyphs.
func matchStarRating(t string) (int, bool) {
	count := 0
	for _, r := range t {
		switch r {
		case '⭐', '🌟', '★':
			count++
		case ' ', '\ufe0f': // spaces and emoji variation selectors are noise
		default:
			return 0, false
		}
	}
	if count >= 1 && count <= 5 {
		return count, true
	}
	return 0, false
}

// maybeSendCloseRatingPrompt starts a CSAT cycle after a conversation closes.
// Runs in a goroutine from the CloseChat / LeaveChat handlers; the contact is
// passed by value so the handler's copy can't race.
//
// Semantics: supersede-then-create — a re-closed conversation gets a fresh
// prompt and the stale pending cycle is expired. The partial unique index
// (one pending cycle per contact) makes the insert race-safe: a concurrent
// close loses ON CONFLICT DO NOTHING and skips sending a second prompt.
func (a *App) maybeSendCloseRatingPrompt(orgID, closedBy uuid.UUID, contact models.Contact) {
	defer func() {
		if rv := recover(); rv != nil {
			a.Log.Error("Panic in maybeSendCloseRatingPrompt", "panic", rv, "contact_id", contact.ID)
		}
	}()

	// Groups and newsletters never get rating prompts.
	if contact.Metadata["is_group_chat"] == true || contact.Metadata["is_newsletter"] == true {
		return
	}

	// The settings live on the account, so it must be resolved before we know
	// whether the feature is even enabled for this number.
	account, err := a.resolveWhatsAppAccount(orgID, contact.WhatsAppAccount)
	if err != nil {
		return // no account, no prompt
	}
	settings := closeRatingSettingsForAccount(account)
	if !settings.Enabled {
		return
	}

	closedByCopy := closedBy
	cycle := models.ChatClosureRating{
		OrganizationID:  orgID,
		ContactID:       contact.ID,
		WhatsAppAccount: account.Name,
		ClosedByUserID:  &closedByCopy,
		Status:          models.RatingStatusPending,
		PromptKind:      models.RatingPromptText,
		ExpiresAt:       time.Now().Add(time.Duration(settings.WindowHours) * time.Hour),
	}

	created := false
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ChatClosureRating{}).
			Where("organization_id = ? AND contact_id = ? AND status = ?",
				orgID, contact.ID, models.RatingStatusPending).
			Update("status", models.RatingStatusExpired).Error; err != nil {
			return err
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&cycle)
		if res.Error != nil {
			return res.Error
		}
		created = res.RowsAffected > 0
		return nil
	})
	if err != nil {
		a.Log.Error("Failed to create close-rating cycle", "error", err, "contact_id", contact.ID)
		return
	}
	if !created {
		return // lost the race to a concurrent close — its prompt is on the way
	}

	if err := a.sendAndSaveTextMessage(account, &contact, settings.Prompt); err != nil {
		a.Log.Error("Failed to send close-rating prompt", "error", err, "contact_id", contact.ID)
		// Don't leave a cycle that would capture replies to a prompt that
		// never arrived.
		a.expireRatingCycle(cycle.ID)
		return
	}
	a.Log.Info("Close-rating prompt sent", "contact_id", contact.ID, "cycle_id", cycle.ID)
}

func (a *App) expireRatingCycle(cycleID uuid.UUID) {
	a.DB.Model(&models.ChatClosureRating{}).
		Where("id = ? AND status = ?", cycleID, models.RatingStatusPending).
		Update("status", models.RatingStatusExpired)
}

// maybeCaptureCloseRating intercepts a customer reply to a pending rating
// cycle. Returns true when the reply was consumed as a rating — the caller
// must then skip ensureClaimableChatStatus (no reopen) and further processing.
// Unmatched or late replies return false and flow through normally.
func (a *App) maybeCaptureCloseRating(account *models.WhatsAppAccount, contact *models.Contact, msg IncomingTextMessage) bool {
	if msg.Type != "text" || msg.Text == nil {
		return false
	}
	text := strings.TrimSpace(msg.Text.Body)
	if text == "" {
		return false
	}

	var cycle models.ChatClosureRating
	err := a.DB.Where("organization_id = ? AND contact_id = ? AND status = ?",
		account.OrganizationID, contact.ID, models.RatingStatusPending).
		Order("created_at DESC").First(&cycle).Error
	if err != nil {
		return false // no pending cycle — the overwhelmingly common path
	}

	// Lazy expiry: past the window the reply is a normal message again.
	if time.Now().After(cycle.ExpiresAt) {
		a.expireRatingCycle(cycle.ID)
		return false
	}

	settings := closeRatingSettingsForAccount(account)
	rating, ok := parseRatingReply(text, settings.Lexicon)
	if !ok {
		// Keep the first unmatched reply for later analysis (phase 2 LLM
		// tail), but let the message reopen the chat normally.
		if cycle.RawReply == "" {
			a.DB.Model(&models.ChatClosureRating{}).Where("id = ?", cycle.ID).
				Update("raw_reply", truncateString(text, 1000))
		}
		return false
	}

	// status='pending' guard: a double-delivered webhook can't rate twice.
	now := time.Now()
	res := a.DB.Model(&models.ChatClosureRating{}).
		Where("id = ? AND status = ?", cycle.ID, models.RatingStatusPending).
		Updates(map[string]any{
			"status":        models.RatingStatusRated,
			"rating":        rating,
			"rating_source": models.RatingSourceExplicit,
			"raw_reply":     truncateString(text, 1000),
			"rated_at":      now,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		return false
	}

	a.Log.Info("Chat close rating captured",
		"contact_id", contact.ID, "cycle_id", cycle.ID, "rating", rating)

	// Visible trace for agents in the conversation history.
	a.ChatLifecycle.CreateSystemMessage(account.OrganizationID, contact.ID,
		fmt.Sprintf("⭐ Customer rated the conversation %d/5", rating),
		models.JSONB{"system_type": "rating_received", "rating": rating})

	if settings.Thanks != "" {
		if err := a.sendAndSaveTextMessage(account, contact, settings.Thanks); err != nil {
			a.Log.Error("Failed to send rating thank-you", "error", err, "contact_id", contact.ID)
		}
	}
	return true
}
