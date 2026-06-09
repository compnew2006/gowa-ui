package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/chat_close_ratings"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

const (
	organizationSettingChatCloseRatingEnabled               = "chat_close_rating_enabled"
	organizationSettingChatCloseRatingWindowDays            = "chat_close_rating_window_days"
	organizationSettingChatCloseRatingTemplates             = "chat_close_rating_templates"
	organizationSettingChatCloseRatingFollowupWindowMinutes = "chat_close_rating_followup_window_minutes"

	defaultChatCloseRatingWindowDays            = 2
	maxChatCloseRatingWindowDays                = 30
	defaultChatCloseRatingFollowupWindowMinutes = 15
	maxChatCloseRatingFollowupWindowMinutes     = 1440
	chatCloseRatingFollowupMessageLimit         = 3

	chatCloseRatingFollowupContextKey  = "followup"
	chatCloseRatingFollowupEntriesKey  = "entries"
	chatCloseRatingFollowupCommentsKey = "comments"
)

var defaultChatCloseRatingTemplates = map[string]string{
	"en": "Hi {customer_name}, your chat {chat_id} with {agent_name} at {organization_name} is now closed. Please reply with a number from 1 to 10 to rate your experience.",
	"ar": "مرحبًا {customer_name}، تم إغلاق المحادثة {chat_id} مع {agent_name} في {organization_name}. الرجاء الرد برقم من 1 إلى 10 لتقييم تجربتك.",
	"es": "Hola {customer_name}, tu chat {chat_id} con {agent_name} en {organization_name} se ha cerrado. Responde con un numero del 1 al 10 para calificar tu experiencia.",
}

type chatCloseRatingSettings struct {
	Enabled               bool
	WindowDays            int
	Templates             map[string]string
	FollowupWindowMinutes int
	UsePoll               bool
}

// AgentRatingSummary represents rating KPIs for the selected analytics period.
type AgentRatingSummary struct {
	TotalRatings   int64            `json:"total_ratings"`
	AverageRating  float64          `json:"average_rating"`
	RatingsByScore map[string]int64 `json:"ratings_by_score"`
}

// AgentRatingRecord represents a single rated close-cycle row.
type AgentRatingRecord struct {
	ID               string       `json:"id"`
	ChatID           string       `json:"chat_id"`
	ContactID        string       `json:"contact_id"`
	Contact          string       `json:"contact"`
	ContactPhone     string       `json:"contact_phone"`
	AgentID          string       `json:"agent_id,omitempty"`
	AgentName        string       `json:"agent_name"`
	ClosingAgentID   string       `json:"closing_agent_id"`
	ClosingAgentName string       `json:"closing_agent_name"`
	Rating           int          `json:"rating"`
	RatedAt          time.Time    `json:"rated_at"`
	RatingMessage    string       `json:"rating_message"`
	ContextMessages  models.JSONB `json:"context_messages"`
}

func cloneDefaultChatCloseRatingTemplates() map[string]string {
	out := make(map[string]string, len(defaultChatCloseRatingTemplates))
	for lang, template := range defaultChatCloseRatingTemplates {
		out[lang] = template
	}
	return out
}

func readInstanceChatCloseRatingSettings(instanceSettings models.JSONB) chatCloseRatingSettings {
	result := chatCloseRatingSettings{
		Enabled:               true,
		WindowDays:            defaultChatCloseRatingWindowDays,
		Templates:             cloneDefaultChatCloseRatingTemplates(),
		FollowupWindowMinutes: defaultChatCloseRatingFollowupWindowMinutes,
	}

	if instanceSettings != nil {
		applyChatCloseRatingSettingsToResult(&result, instanceSettings)
	}

	return result
}

// applyChatCloseRatingSettingsToResult applies settings from a JSONB object to the result struct.
// This helper function eliminates duplication between org and instance settings parsing.
func applyChatCloseRatingSettingsToResult(result *chatCloseRatingSettings, settings models.JSONB) {
	if rawEnabled, ok := settings[organizationSettingChatCloseRatingEnabled]; ok {
		if enabled, ok := rawEnabled.(bool); ok {
			result.Enabled = enabled
		}
	}

	if rawWindow, ok := settings[organizationSettingChatCloseRatingWindowDays]; ok {
		if parsed := parseChatCloseRatingWindowDays(rawWindow); parsed > 0 {
			result.WindowDays = parsed
		}
	}

	if rawTemplates, ok := settings[organizationSettingChatCloseRatingTemplates]; ok {
		for lang, template := range parseChatCloseRatingTemplates(rawTemplates) {
			result.Templates[lang] = template
		}
	}

	if rawFollowupWindow, ok := settings[organizationSettingChatCloseRatingFollowupWindowMinutes]; ok {
		if parsed := chat_close_ratings.ParseFollowupWindowMinutes(rawFollowupWindow); parsed > 0 {
			result.FollowupWindowMinutes = parsed
		}
	}

	if rawUsePoll, ok := settings["use_poll"]; ok {
		if usePoll, ok := rawUsePoll.(bool); ok {
			result.UsePoll = usePoll
		}
	}
}

func parseChatCloseRatingWindowDays(raw any) int {
	parsed := 0
	switch v := raw.(type) {
	case int:
		parsed = v
	case int32:
		parsed = int(v)
	case int64:
		parsed = int(v)
	case float64:
		parsed = int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			parsed = n
		}
	}

	if parsed < 1 {
		return defaultChatCloseRatingWindowDays
	}
	if parsed > maxChatCloseRatingWindowDays {
		return maxChatCloseRatingWindowDays
	}
	return parsed
}

func parseChatCloseRatingTemplates(raw any) map[string]string {
	templates := map[string]string{}

	switch typed := raw.(type) {
	case map[string]string:
		for lang, template := range typed {
			langKey := normalizeLanguageKey(lang)
			if langKey == "" || strings.TrimSpace(template) == "" {
				continue
			}
			templates[langKey] = strings.TrimSpace(template)
		}
	case map[string]any:
		for lang, rawTemplate := range typed {
			template, ok := rawTemplate.(string)
			if !ok {
				continue
			}
			langKey := normalizeLanguageKey(lang)
			if langKey == "" || strings.TrimSpace(template) == "" {
				continue
			}
			templates[langKey] = strings.TrimSpace(template)
		}
	}

	return templates
}

func normalizeLanguageKey(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	if idx := strings.Index(key, "-"); idx > 0 {
		return key[:idx]
	}
	if idx := strings.Index(key, "_"); idx > 0 {
		return key[:idx]
	}
	return key
}

func resolveChatCloseRatingLanguage(contact *models.Contact, orgSettings models.JSONB, templates map[string]string) string {
	candidates := make([]string, 0, 3)

	if contact != nil && contact.Metadata != nil {
		if rawLang, ok := contact.Metadata["language"].(string); ok {
			candidates = append(candidates, rawLang)
		}
		if rawLocale, ok := contact.Metadata["locale"].(string); ok {
			candidates = append(candidates, rawLocale)
		}
	}

	if orgSettings != nil {
		if rawLang, ok := orgSettings["language"].(string); ok {
			candidates = append(candidates, rawLang)
		}
		if rawLang, ok := orgSettings["default_language"].(string); ok {
			candidates = append(candidates, rawLang)
		}
	}

	for _, raw := range candidates {
		lang := normalizeLanguageKey(raw)
		if lang == "" {
			continue
		}
		if _, ok := templates[lang]; ok {
			return lang
		}
	}

	if _, ok := templates["en"]; ok {
		return "en"
	}

	for lang := range templates {
		return lang
	}

	return "en"
}

func renderChatCloseRatingPrompt(template string, organizationName string, contact *models.Contact, agentName string) string {
	customerName := "Customer"
	chatID := ""
	if contact != nil {
		chatID = contact.ID.String()
		if trimmed := strings.TrimSpace(contact.ProfileName); trimmed != "" {
			customerName = trimmed
		} else if trimmed := strings.TrimSpace(contact.PhoneNumber); trimmed != "" {
			customerName = trimmed
		}
	}

	replacer := strings.NewReplacer(
		"{agent_name}", agentName,
		"{customer_name}", customerName,
		"{chat_id}", chatID,
		"{organization_name}", organizationName,
	)

	return replacer.Replace(template)
}

func (a *App) handleManualChatCloseRatingPrompt(orgID, closingUserID uuid.UUID, contact *models.Contact) {
	if a == nil || contact == nil {
		return
	}

	var org models.Organization
	if err := a.DB.Select("id", "name", "settings").Where("id = ?", orgID).First(&org).Error; err != nil {
		a.Log.Error("Failed to load organization for close rating prompt", "error", err, "organization_id", orgID)
		return
	}

	var instanceSettings models.JSONB
	if contact.InstanceID != nil && *contact.InstanceID != uuid.Nil {
		var instance models.WhatsAppInstance
		if err := a.DB.Select("settings").Where("id = ?", *contact.InstanceID).First(&instance).Error; err == nil {
			instanceSettings = instance.Settings
		}
	}

	settings := readInstanceChatCloseRatingSettings(instanceSettings)
	if !settings.Enabled {
		return
	}

	if err := a.DB.Model(&models.ChatClosureRating{}).
		Where("organization_id = ? AND contact_id = ? AND state = ?", orgID, contact.ID, models.ChatClosureRatingStatePending).
		Updates(map[string]any{"state": models.ChatClosureRatingStateExpired}).Error; err != nil {
		a.Log.Error("Failed to expire pending close rating cycles", "error", err, "contact_id", contact.ID)
	}

	agentUserID := contact.AssignedUserID
	if agentUserID == nil {
		fallback := closingUserID
		agentUserID = &fallback
	}

	agentName := "Agent"
	if agentUserID != nil {
		if resolved := strings.TrimSpace(a.ResolveUserDisplayName(*agentUserID)); resolved != "" {
			agentName = resolved
		}
	}

	language := resolveChatCloseRatingLanguage(contact, org.Settings, settings.Templates)
	template := strings.TrimSpace(settings.Templates[language])
	if template == "" {
		template = strings.TrimSpace(settings.Templates["en"])
	}
	if template == "" {
		template = defaultChatCloseRatingTemplates["en"]
	}

	promptText := renderChatCloseRatingPrompt(template, org.Name, contact, agentName)

	cycle := models.ChatClosureRating{
		BaseModel:            models.BaseModel{ID: uuid.New()},
		OrganizationID:       orgID,
		ContactID:            contact.ID,
		ChatID:               contact.ID,
		AgentUserID:          agentUserID,
		ClosingAgentID:       closingUserID,
		ClosedAt:             time.Now().UTC(),
		State:                models.ChatClosureRatingStatePending,
		CloseMessage:         promptText,
		CloseMessageLanguage: language,
		ContextMessages:      models.JSONB{},
	}

	if err := a.DB.Create(&cycle).Error; err != nil {
		a.Log.Error("Failed to create close rating cycle", "error", err, "contact_id", contact.ID)
		return
	}

	promptMessageID, sendErr := a.sendChatCloseRatingPrompt(orgID, contact, promptText, settings)
	if sendErr != nil {
		a.Log.Error("Failed to send close rating prompt", "error", sendErr, "contact_id", contact.ID)
		return
	}
	if promptMessageID == nil {
		return
	}

	if err := a.DB.Model(&models.ChatClosureRating{}).
		Where("id = ?", cycle.ID).
		Update("close_message_id", *promptMessageID).Error; err != nil {
		a.Log.Error("Failed to store close rating prompt message id", "error", err, "cycle_id", cycle.ID)
	}
}

func (a *App) sendChatCloseRatingPrompt(orgID uuid.UUID, contact *models.Contact, promptText string, settings chatCloseRatingSettings) (*uuid.UUID, error) {
	account, err := a.resolveClosePromptAccount(orgID, contact)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("chat close rating prompt account is unavailable")
	}

	// If use_poll is enabled and the provider supports polls, send a native poll.
	if settings.UsePoll && contact.InstanceID != nil && *contact.InstanceID != uuid.Nil {
		if a.MessageProvider != nil {
			if pollSender, ok := a.MessageProvider.(provider.PollProvider); ok {
				instanceID := contact.InstanceID.String()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				ratingOptions := []string{
					"1 - Very Poor",
					"2 - Poor",
					"3 - Fair",
					"4 - Good",
					"5 - Excellent",
				}
				_, pollErr := pollSender.SendPoll(ctx, instanceID, contact.PhoneNumber, promptText, ratingOptions, 1)
				if pollErr != nil {
					a.Log.Error("Failed to send rating poll, falling back to text", "error", pollErr, "contact_id", contact.ID)
				} else {
					// Store a poll-type message so the prompt is tracked.
					msg, sendErr := a.SendOutgoingMessage(context.Background(), OutgoingMessageRequest{
						Account: account,
						Contact: contact,
						Type:    models.MessageTypePoll,
						Content: promptText,
					}, SLASendOptions())
					if sendErr != nil {
						a.Log.Error("Failed to persist rating poll message", "error", sendErr)
					}
					if msg != nil {
						msgID := msg.ID
						return &msgID, nil
					}
					return nil, nil
				}
			}
		}
	}

	if a.isWhatsmeowProvider() {
		if a.MessageProvider == nil {
			return nil, fmt.Errorf("message provider is not configured")
		}
	} else if a.WhatsApp == nil {
		return nil, fmt.Errorf("whatsapp client is not configured")
	}

	msg, err := a.SendOutgoingMessage(context.Background(), OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: promptText,
	}, SLASendOptions())
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	messageID := msg.ID
	return &messageID, nil
}

func (a *App) resolveClosePromptAccount(orgID uuid.UUID, contact *models.Contact) (*models.WhatsAppAccount, error) {
	accountName := ""
	if contact != nil {
		accountName = a.resolveContactMessageAccount(contact)
	}

	if a.isWhatsmeowProvider() {
		return &models.WhatsAppAccount{
			OrganizationID: orgID,
			Name:           accountName,
		}, nil
	}

	return a.resolveWhatsAppAccount(orgID, accountName)
}

func (a *App) loadChatCloseRatingSettings(instanceID *uuid.UUID) (chatCloseRatingSettings, error) {
	result := readInstanceChatCloseRatingSettings(nil)
	var instanceSettings models.JSONB
	if instanceID != nil && *instanceID != uuid.Nil {
		var instance models.WhatsAppInstance
		if err := a.DB.Select("settings").Where("id = ?", *instanceID).First(&instance).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return result, nil
			}
			return chatCloseRatingSettings{}, err
		}
		instanceSettings = instance.Settings
	}

	return readInstanceChatCloseRatingSettings(instanceSettings), nil
}

func (a *App) findActiveChatCloseRatingCycle(
	orgID uuid.UUID, contact *models.Contact,
	now time.Time,
) (*models.ChatClosureRating, chat_close_ratings.FollowupState, error) {
	settings, err := a.loadChatCloseRatingSettings(contact.InstanceID)
	if err != nil {
		return nil, chat_close_ratings.FollowupState{}, err
	}
	if !settings.Enabled {
		return nil, chat_close_ratings.FollowupState{}, nil
	}

	windowStart := now.UTC().Add(-time.Duration(settings.WindowDays) * 24 * time.Hour)

	var cycle models.ChatClosureRating
	if err := a.DB.Where("organization_id = ? AND contact_id = ? AND state IN ? AND closed_at >= ?",
		orgID,
		contact.ID,
		[]models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		},
		windowStart,
	).Order("closed_at DESC").First(&cycle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, chat_close_ratings.FollowupState{}, nil
		}
		return nil, chat_close_ratings.FollowupState{}, err
	}

	var closedAt time.Time
	if !cycle.ClosedAt.IsZero() {
		closedAt = cycle.ClosedAt
	} else {
		closedAt = now.UTC()
	}

	followup := chat_close_ratings.ReadFollowupState(closedAt, cycle.ContextMessages, settings.FollowupWindowMinutes)
	if followup.IsActive(now) {
		return &cycle, followup, nil
	}

	if cycle.State == models.ChatClosureRatingStatePending {
		if err := a.DB.Model(&models.ChatClosureRating{}).
			Where("id = ? AND state = ?", cycle.ID, models.ChatClosureRatingStatePending).
			Update("state", models.ChatClosureRatingStateExpired).Error; err != nil {
			return nil, followup, err
		}
	}

	return nil, followup, nil
}

func (a *App) maybeCaptureChatCloseRating(orgID uuid.UUID, contact *models.Contact, payload incomingMessagePayload, incomingMessage *models.Message) bool {
	if contact == nil || incomingMessage == nil {
		return false
	}

	// Handle poll votes as rating responses in addition to text.
	isPollVoteMsg := payload.MessageType == "poll" && incomingMessage.InteractiveData != nil
	var pollVoteOptions []string
	if isPollVoteMsg {
		if raw, ok := incomingMessage.InteractiveData["type"]; !ok || raw != "poll_vote" {
			isPollVoteMsg = false
		}
		if isPollVoteMsg {
			if opts, ok := incomingMessage.InteractiveData["selected_options"]; ok {
				if arr, ok := opts.([]interface{}); ok {
					for _, v := range arr {
						if s, ok := v.(string); ok {
							pollVoteOptions = append(pollVoteOptions, s)
						}
					}
				}
			}
		}
	}

	if payload.MessageType != "text" && !isPollVoteMsg {
		return false
	}

	now := time.Now().UTC()
	cycle, followup, err := a.findActiveChatCloseRatingCycle(orgID, contact, now)
	if err != nil {
		a.Log.Error("Failed to resolve close rating cycle", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	if cycle == nil {
		return false
	}

	trimmedContent := strings.TrimSpace(payload.MessageText)
	var ratingValue int
	var hasRating bool

	if isPollVoteMsg && len(pollVoteOptions) > 0 {
		for _, opt := range pollVoteOptions {
			if parsed := chat_close_ratings.ParseRatingFromPollOption(opt); parsed > 0 {
				ratingValue = parsed
				hasRating = true
				trimmedContent = opt
				break
			}
		}
	} else {
		ratingValue, hasRating = chat_close_ratings.ParseInboundRatingValue(payload.MessageText)
	}

	contextMessages := cycle.ContextMessages
	if hasRating {
		contextMessages = a.buildChatCloseRatingContext(contact.ID, incomingMessage)
	}
	if contextMessages == nil {
		contextMessages = models.JSONB{}
	}

	var ratingPointer *int
	entryKind := "comment"
	if hasRating {
		ratingPointer = &ratingValue
		entryKind = "rating"
	}
	followup.Entries = chat_close_ratings.AppendChatCloseRatingFollowupEntry(followup.Entries, incomingMessage, trimmedContent, entryKind, ratingPointer)
	if !hasRating && trimmedContent != "" {
		followup.Comments = append(followup.Comments, trimmedContent)
	}
	if followup.RemainingMessages > 0 {
		followup.RemainingMessages--
	}
	contextMessages = chat_close_ratings.WriteFollowupState(contextMessages, followup)

	updates := map[string]any{
		"context_messages": contextMessages,
	}
	if hasRating {
		ratedAt := now
		updates["state"] = models.ChatClosureRatingStateRated
		updates["rating"] = ratingValue
		updates["rated_at"] = &ratedAt
		updates["rating_message"] = trimmedContent
		updates["rating_message_id"] = incomingMessage.ID
	} else if cycle.State == models.ChatClosureRatingStateRated && trimmedContent != "" {
		updates["rating_message"] = trimmedContent
		updates["rating_message_id"] = incomingMessage.ID
	} else if cycle.State == models.ChatClosureRatingStatePending && !followup.IsActive(now) {
		updates["state"] = models.ChatClosureRatingStateExpired
	}

	result := a.DB.Model(&models.ChatClosureRating{}).
		Where("id = ? AND state IN ?", cycle.ID, []models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		}).
		Updates(updates)
	if result.Error != nil {
		a.Log.Error("Failed to persist chat close rating follow-up", "error", result.Error, "cycle_id", cycle.ID)
		return false
	}
	if result.RowsAffected == 0 {
		return false
	}

	if hasRating {
		a.Log.Info("Captured chat close rating", "cycle_id", cycle.ID, "contact_id", contact.ID, "rating", ratingValue)
	} else {
		a.Log.Info("Captured chat close rating follow-up comment", "cycle_id", cycle.ID, "contact_id", contact.ID)
	}
	return true
}

func (a *App) buildChatCloseRatingContext(contactID uuid.UUID, ratingMessage *models.Message) models.JSONB {
	if ratingMessage == nil {
		return models.JSONB{}
	}

	var before []models.Message
	a.DB.Where("contact_id = ? AND id <> ? AND created_at <= ?", contactID, ratingMessage.ID, ratingMessage.CreatedAt).
		Order("created_at DESC").
		Limit(2).
		Find(&before)

	var after []models.Message
	a.DB.Where("contact_id = ? AND id <> ? AND created_at >= ?", contactID, ratingMessage.ID, ratingMessage.CreatedAt).
		Order("created_at ASC").
		Limit(2).
		Find(&after)

	for i, j := 0, len(before)-1; i < j; i, j = i+1, j-1 {
		before[i], before[j] = before[j], before[i]
	}

	return models.JSONB{
		"before": mapMessagesForRatingContext(before),
		"rating": chat_close_ratings.MapSingleMessageForRatingContext(ratingMessage),
		"after":  mapMessagesForRatingContext(after),
	}
}

func mapMessagesForRatingContext(messages []models.Message) []any {
	entries := make([]any, 0, len(messages))
	for i := range messages {
		entries = append(entries, chat_close_ratings.MapSingleMessageForRatingContext(&messages[i]))
	}
	return entries
}

func extractFollowupCommentsForRatingMessage(contextMessages models.JSONB) []string {
	if len(contextMessages) == 0 {
		return nil
	}

	rawFollowup, ok := contextMessages[chatCloseRatingFollowupContextKey]
	if !ok || rawFollowup == nil {
		return nil
	}

	var followup map[string]any
	switch typed := rawFollowup.(type) {
	case map[string]any:
		followup = typed
	case models.JSONB:
		followup = map[string]any(typed)
	default:
		return nil
	}

	comments := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	appendUnique := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		comments = append(comments, trimmed)
	}

	for _, comment := range asStringSliceChatClose(followup[chatCloseRatingFollowupCommentsKey]) {
		appendUnique(comment)
	}

	if rawEntries, ok := followup[chatCloseRatingFollowupEntriesKey]; ok {
		if entries, ok := rawEntries.([]any); ok {
			for _, rawEntry := range entries {
				entry, ok := rawEntry.(map[string]any)
				if !ok {
					continue
				}
				kind, _ := entry["kind"].(string)
				if strings.TrimSpace(kind) != "comment" {
					continue
				}
				content, _ := entry["content"].(string)
				appendUnique(content)
			}
		}
	}

	return comments
}

func asStringSliceChatClose(val any) []string {
	if val == nil {
		return nil
	}
	switch typed := val.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

func mergeRatingMessageWithFollowupComments(base string, contextMessages models.JSONB) string {
	parts := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	appendUnique := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		parts = append(parts, trimmed)
	}

	appendUnique(base)
	for _, comment := range extractFollowupCommentsForRatingMessage(contextMessages) {
		appendUnique(comment)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " | ")
}

func decodeRatingContextMessages(raw json.RawMessage) models.JSONB {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil
	}
	return models.JSONB(decoded)
}

func parseRatingFilterBound(raw string) (*int, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ""
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value < 1 || value > 10 {
		return nil, "Rating filters must be integers between 1 and 10"
	}
	return &value, ""
}

func (a *App) calculateAgentRatingSummary(
	orgID uuid.UUID,
	start, end time.Time,
	filterAgentID *uuid.UUID,
	filterInstanceID *uuid.UUID,
	minRating, maxRating *int,
) (AgentRatingSummary, error) {
	summary := AgentRatingSummary{
		RatingsByScore: make(map[string]int64, 10),
	}
	for score := 1; score <= 10; score++ {
		summary.RatingsByScore[strconv.Itoa(score)] = 0
	}

	query := a.DB.Model(&models.ChatClosureRating{}).
		Where("organization_id = ? AND state = ? AND rated_at IS NOT NULL AND rated_at >= ? AND rated_at <= ?",
			orgID,
			models.ChatClosureRatingStateRated,
			start,
			end,
		)
	query = applyTransferAnalyticsInstanceFilter(query, orgID, filterInstanceID)
	if filterAgentID != nil {
		query = query.Where("agent_user_id = ?", *filterAgentID)
	}
	if minRating != nil {
		query = query.Where("rating >= ?", *minRating)
	}
	if maxRating != nil {
		query = query.Where("rating <= ?", *maxRating)
	}

	type aggregateRow struct {
		Total int64
		Avg   float64
	}
	var aggregate aggregateRow
	if err := query.Select("COUNT(*) AS total, COALESCE(AVG(rating), 0) AS avg").Scan(&aggregate).Error; err != nil {
		return summary, err
	}
	summary.TotalRatings = aggregate.Total
	summary.AverageRating = aggregate.Avg

	type bucketRow struct {
		Rating int
		Count  int64
	}
	var buckets []bucketRow
	if err := query.Select("rating, COUNT(*) AS count").Group("rating").Scan(&buckets).Error; err != nil {
		return summary, err
	}
	for _, bucket := range buckets {
		summary.RatingsByScore[strconv.Itoa(bucket.Rating)] = bucket.Count
	}

	return summary, nil
}

func (a *App) listAgentRatingRecords(
	orgID uuid.UUID,
	start, end time.Time,
	filterAgentID *uuid.UUID,
	filterInstanceID *uuid.UUID,
	minRating, maxRating *int,
	limit int,
) ([]AgentRatingRecord, error) {
	type row struct {
		ID               uuid.UUID
		ChatID           uuid.UUID
		ContactID        uuid.UUID
		ContactName      string
		ContactPhone     string
		AgentUserID      *uuid.UUID
		AgentName        string
		ClosingAgentID   uuid.UUID
		ClosingAgentName string
		Rating           int
		RatedAt          time.Time
		RatingMessage    string
		ContextMessages  json.RawMessage `gorm:"column:context_messages"`
	}

	query := a.DB.Table("chat_closure_ratings ccr").
		Select(`
			ccr.id,
			ccr.chat_id,
			ccr.contact_id,
			c.profile_name AS contact_name,
			c.phone_number AS contact_phone,
			ccr.agent_user_id,
			COALESCE(agent.full_name, '') AS agent_name,
			ccr.closing_agent_id,
			COALESCE(closing_agent.full_name, '') AS closing_agent_name,
			ccr.rating,
			ccr.rated_at,
			ccr.rating_message,
			ccr.context_messages
		`).
		Joins("LEFT JOIN contacts c ON c.id = ccr.contact_id").
		Joins("LEFT JOIN users agent ON agent.id = ccr.agent_user_id").
		Joins("LEFT JOIN users closing_agent ON closing_agent.id = ccr.closing_agent_id").
		Where("ccr.organization_id = ? AND ccr.state = ? AND ccr.rated_at IS NOT NULL AND ccr.rated_at >= ? AND ccr.rated_at <= ?",
			orgID,
			models.ChatClosureRatingStateRated,
			start,
			end,
		)
	query = applyRatingAnalyticsInstanceFilter(query, filterInstanceID, "c")

	if filterAgentID != nil {
		query = query.Where("ccr.agent_user_id = ?", *filterAgentID)
	}
	if minRating != nil {
		query = query.Where("ccr.rating >= ?", *minRating)
	}
	if maxRating != nil {
		query = query.Where("ccr.rating <= ?", *maxRating)
	}

	query = query.Order("ccr.rated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []row
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]AgentRatingRecord, 0, len(rows))
	for _, item := range rows {
		contactName := strings.TrimSpace(item.ContactName)
		if contactName == "" {
			contactName = strings.TrimSpace(item.ContactPhone)
		}
		agentID := ""
		if item.AgentUserID != nil {
			agentID = item.AgentUserID.String()
		}
		contextMessages := decodeRatingContextMessages(item.ContextMessages)

		records = append(records, AgentRatingRecord{
			ID:               item.ID.String(),
			ChatID:           item.ChatID.String(),
			ContactID:        item.ContactID.String(),
			Contact:          contactName,
			ContactPhone:     item.ContactPhone,
			AgentID:          agentID,
			AgentName:        strings.TrimSpace(item.AgentName),
			ClosingAgentID:   item.ClosingAgentID.String(),
			ClosingAgentName: strings.TrimSpace(item.ClosingAgentName),
			Rating:           item.Rating,
			RatedAt:          item.RatedAt,
			RatingMessage:    mergeRatingMessageWithFollowupComments(item.RatingMessage, contextMessages),
			ContextMessages:  contextMessages,
		})
	}

	return records, nil
}

// ExportAgentRatings exports chat close ratings as CSV.
func (a *App) ExportAgentRatings(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.HasPermission(userID, models.ResourceAnalytics, models.ActionRead, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Access denied", nil, "")
	}

	fromStr := string(r.RequestCtx.QueryArgs().Peek("from"))
	toStr := string(r.RequestCtx.QueryArgs().Peek("to"))
	agentIDStr := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("agent_id")))
	instanceIDStr := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id")))
	minRating, minErr := parseRatingFilterBound(string(r.RequestCtx.QueryArgs().Peek("min_rating")))
	if minErr != "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, minErr, nil, "")
	}
	maxRating, maxErr := parseRatingFilterBound(string(r.RequestCtx.QueryArgs().Peek("max_rating")))
	if maxErr != "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, maxErr, nil, "")
	}
	if minRating != nil && maxRating != nil && *minRating > *maxRating {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "min_rating cannot be greater than max_rating", nil, "")
	}

	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := now
	if fromStr != "" && toStr != "" {
		var errMsg string
		periodStart, periodEnd, errMsg = parseDateRange(fromStr, toStr)
		if errMsg != "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, errMsg, nil, "")
		}
	}

	var filterAgentID *uuid.UUID
	if agentIDStr != "" && !strings.EqualFold(agentIDStr, "all") {
		parsed, parseErr := uuid.Parse(agentIDStr)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid agent_id", nil, "")
		}
		filterAgentID = &parsed
	}

	filterInstanceID, instanceErr := a.parseAnalyticsInstanceID(orgID, instanceIDStr)
	if instanceErr != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, instanceErr.Error(), nil, "instance_id")
	}

	records, listErr := a.listAgentRatingRecords(orgID, periodStart, periodEnd, filterAgentID, filterInstanceID, minRating, maxRating, 0)
	if listErr != nil {
		a.Log.Error("Failed to export agent ratings", "error", listErr, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to export ratings", nil, "")
	}

	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	header := []string{"agent", "chat_id", "contact", "rating", "rated_at", "closing_agent", "rating_message", "context_messages"}
	_ = writer.Write(header)

	for _, record := range records {
		contextJSON := "{}"
		if len(record.ContextMessages) > 0 {
			if marshaled, marshalErr := json.Marshal(record.ContextMessages); marshalErr == nil {
				contextJSON = string(marshaled)
			}
		}

		row := []string{
			record.AgentName,
			record.ChatID,
			record.Contact,
			strconv.Itoa(record.Rating),
			record.RatedAt.UTC().Format(time.RFC3339),
			record.ClosingAgentName,
			record.RatingMessage,
			contextJSON,
		}

		for i, cell := range row {
			if len(cell) > 0 && (cell[0] == '=' || cell[0] == '@') {
				row[i] = "'" + cell
			}
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	filename := fmt.Sprintf("agent_ratings_%s.csv", time.Now().UTC().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "text/csv")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	r.RequestCtx.SetBody([]byte(builder.String()))

	return nil
}
