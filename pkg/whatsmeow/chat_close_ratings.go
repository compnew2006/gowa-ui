package whatsmeow

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	organizationSettingChatCloseRatingEnabled               = "chat_close_rating_enabled"
	organizationSettingChatCloseRatingFollowupWindowMinutes = "chat_close_rating_followup_window_minutes"
	defaultChatCloseRatingFollowupWindowMinutes = 15
	maxChatCloseRatingFollowupWindowMinutes     = 1440
	chatCloseRatingFollowupMessageLimit         = 3

	chatCloseRatingFollowupContextKey  = "followup"
	chatCloseRatingFollowupEntriesKey  = "entries"
	chatCloseRatingFollowupCommentsKey = "comments"
)

var localizedRatingDigitReplacer = strings.NewReplacer(
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
	"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
	"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
)

type chatCloseRatingSettings struct {
	Enabled               bool
	FollowupWindowMinutes int
}

func readChatCloseRatingSettings(settings models.JSONB) chatCloseRatingSettings {
	result := chatCloseRatingSettings{
		Enabled:               true,
		FollowupWindowMinutes: defaultChatCloseRatingFollowupWindowMinutes,
	}

	if settings == nil {
		return result
	}

	if rawEnabled, ok := settings[organizationSettingChatCloseRatingEnabled]; ok {
		if enabled, ok := rawEnabled.(bool); ok {
			result.Enabled = enabled
		}
	}
	if rawFollowupWindow, ok := settings[organizationSettingChatCloseRatingFollowupWindowMinutes]; ok {
		result.FollowupWindowMinutes = parseChatCloseRatingFollowupWindowMinutes(rawFollowupWindow)
	}

	return result
}

func parseChatCloseRatingFollowupWindowMinutes(raw any) int {
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
		return defaultChatCloseRatingFollowupWindowMinutes
	}
	if parsed > maxChatCloseRatingFollowupWindowMinutes {
		return maxChatCloseRatingFollowupWindowMinutes
	}
	return parsed
}

type chatCloseRatingFollowupState struct {
	ExpiresAt         time.Time
	RemainingMessages int
	Entries           []any
	Comments          []string
}

func newChatCloseRatingFollowupState(closedAt time.Time, settings chatCloseRatingSettings) chatCloseRatingFollowupState {
	return chatCloseRatingFollowupState{
		ExpiresAt:         closedAt.Add(time.Duration(settings.FollowupWindowMinutes) * time.Minute),
		RemainingMessages: chatCloseRatingFollowupMessageLimit,
		Entries:           make([]any, 0, chatCloseRatingFollowupMessageLimit),
		Comments:          []string{},
	}
}

func readChatCloseRatingFollowupState(cycle *models.ChatClosureRating, settings chatCloseRatingSettings) chatCloseRatingFollowupState {
	if cycle == nil {
		return newChatCloseRatingFollowupState(time.Now().UTC(), settings)
	}

	state := newChatCloseRatingFollowupState(cycle.ClosedAt, settings)
	if cycle.ContextMessages == nil {
		return state
	}

	rawFollowup, ok := cycle.ContextMessages[chatCloseRatingFollowupContextKey]
	if !ok || rawFollowup == nil {
		return state
	}

	var payload map[string]any
	switch typed := rawFollowup.(type) {
	case map[string]any:
		payload = typed
	case models.JSONB:
		payload = map[string]any(typed)
	default:
		return state
	}

	if rawExpiresAt, ok := payload["expires_at"]; ok {
		if parsed := parseJSONTime(rawExpiresAt); !parsed.IsZero() {
			state.ExpiresAt = parsed
		}
	}
	if rawRemaining, ok := payload["remaining_messages"]; ok {
		if parsed, ok := parseJSONInt(rawRemaining); ok {
			state.RemainingMessages = parsed
		}
	}
	if rawEntries, ok := payload[chatCloseRatingFollowupEntriesKey]; ok {
		switch typed := rawEntries.(type) {
		case []any:
			state.Entries = append([]any{}, typed...)
		case []map[string]any:
			for _, entry := range typed {
				state.Entries = append(state.Entries, entry)
			}
		}
	}
	if rawComments, ok := payload[chatCloseRatingFollowupCommentsKey]; ok {
		state.Comments = normalizeRatingComments(asStringSlice(rawComments))
	}

	if state.RemainingMessages < 0 {
		state.RemainingMessages = 0
	}
	return state
}

func (s chatCloseRatingFollowupState) IsActive(now time.Time) bool {
	if s.RemainingMessages <= 0 {
		return false
	}
	return !now.After(s.ExpiresAt)
}

func normalizeRatingComments(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func appendChatCloseRatingFollowupEntry(
	state *chatCloseRatingFollowupState,
	incomingMessage *models.Message,
	content string,
	kind string,
	ratingValue *int,
) {
	if state == nil {
		return
	}

	entry := models.JSONB{
		"kind":    kind,
		"content": strings.TrimSpace(content),
	}
	if incomingMessage != nil {
		entry["message_id"] = incomingMessage.ID.String()
		entry["message_type"] = incomingMessage.MessageType
		entry["created_at"] = incomingMessage.CreatedAt.UTC().Format(time.RFC3339)
	}
	if ratingValue != nil {
		entry["rating"] = *ratingValue
	}

	state.Entries = append(state.Entries, entry)
}

func writeChatCloseRatingFollowupState(contextMessages models.JSONB, state chatCloseRatingFollowupState) models.JSONB {
	if contextMessages == nil {
		contextMessages = models.JSONB{}
	}

	contextMessages[chatCloseRatingFollowupContextKey] = models.JSONB{
		"expires_at":                       state.ExpiresAt.UTC().Format(time.RFC3339),
		"remaining_messages":               state.RemainingMessages,
		chatCloseRatingFollowupEntriesKey:  state.Entries,
		chatCloseRatingFollowupCommentsKey: state.Comments,
	}
	return contextMessages
}

func parseJSONTime(raw any) time.Time {
	switch typed := raw.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(typed))
		if err == nil {
			return parsed.UTC()
		}
	case time.Time:
		return typed.UTC()
	}
	return time.Time{}
}

func parseJSONInt(raw any) (int, bool) {
	switch typed := raw.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func asStringSlice(raw interface{}) []string {
	switch values := raw.(type) {
	case []string:
		return append([]string{}, values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, rawValue := range values {
			value, ok := rawValue.(string)
			if !ok {
				continue
			}
			out = append(out, value)
		}
		return out
	default:
		return nil
	}
}

func normalizeInboundRatingText(raw string) string {
	normalized := strings.TrimSpace(localizedRatingDigitReplacer.Replace(raw))
	if normalized == "" {
		return ""
	}

	runes := []rune(normalized)
	start := 0
	for start < len(runes) && (unicode.IsSpace(runes[start]) || isIgnorableRatingRune(runes[start])) {
		start++
	}
	if start >= len(runes) {
		return ""
	}

	end := len(runes)
	for end > start && (unicode.IsSpace(runes[end-1]) || isIgnorableRatingRune(runes[end-1])) {
		end--
	}

	return string(runes[start:end])
}

func isIgnorableRatingRune(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}
	return unicode.In(r, unicode.Cf)
}

func parseInboundRatingValue(raw string) (int, bool) {
	trimmed := normalizeInboundRatingText(raw)
	if trimmed == "" {
		return 0, false
	}

	runes := []rune(trimmed)
	end := 0
	for end < len(runes) && runes[end] >= '0' && runes[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}

	nextIndex := end
	for nextIndex < len(runes) && isIgnorableRatingRune(runes[nextIndex]) {
		nextIndex++
	}

	if nextIndex < len(runes) {
		next := runes[nextIndex]
		if !unicode.IsSpace(next) && !unicode.IsPunct(next) && !unicode.IsSymbol(next) {
			return 0, false
		}
	}

	rating, err := strconv.Atoi(string(runes[:end]))
	if err != nil {
		return 0, false
	}
	if rating < 1 || rating > 10 {
		return 0, false
	}
	return rating, true
}

func (cm *ConnectionManager) loadChatCloseRatingSettings(ctx context.Context, orgID uuid.UUID) (chatCloseRatingSettings, error) {
	var org models.Organization
	if err := cm.db.WithContext(ctx).
		Select("settings").
		Where("id = ?", orgID).
		First(&org).Error; err != nil {
		return chatCloseRatingSettings{}, err
	}
	return readChatCloseRatingSettings(org.Settings), nil
}

func (cm *ConnectionManager) findActiveChatCloseRatingCycle(
	ctx context.Context,
	orgID, contactID uuid.UUID,
	now time.Time,
) (*models.ChatClosureRating, chatCloseRatingFollowupState, error) {
	settings, err := cm.loadChatCloseRatingSettings(ctx, orgID)
	if err != nil {
		return nil, chatCloseRatingFollowupState{}, err
	}
	if !settings.Enabled {
		return nil, chatCloseRatingFollowupState{}, nil
	}

	windowStart := now.UTC().Add(-2 * 24 * time.Hour)

	var cycle models.ChatClosureRating
	if err := cm.db.WithContext(ctx).Where(
		"organization_id = ? AND contact_id = ? AND state IN ? AND closed_at >= ?",
		orgID,
		contactID,
		[]models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		},
		windowStart,
	).Order("closed_at DESC").First(&cycle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, chatCloseRatingFollowupState{}, nil
		}
		return nil, chatCloseRatingFollowupState{}, err
	}

	followup := readChatCloseRatingFollowupState(&cycle, settings)
	if followup.IsActive(now) {
		return &cycle, followup, nil
	}

	if cycle.State == models.ChatClosureRatingStatePending {
		if err := cm.db.WithContext(ctx).
			Model(&models.ChatClosureRating{}).
			Where("id = ? AND state = ?", cycle.ID, models.ChatClosureRatingStatePending).
			Update("state", models.ChatClosureRatingStateExpired).Error; err != nil {
			return nil, followup, err
		}
	}

	return nil, followup, nil
}

func (cm *ConnectionManager) shouldSkipClosedChatAutoReopenForIncomingMessage(
	ctx context.Context,
	orgID uuid.UUID,
	contact *models.Contact,
	msgType models.MessageType,
	content string,
) bool {
	if contact == nil {
		return false
	}
	if contact.EffectiveStatus() != models.ChatStatusClosed {
		return false
	}
	if msgType != models.MessageTypeText {
		return false
	}
	_ = content

	cycle, _, err := cm.findActiveChatCloseRatingCycle(ctx, orgID, contact.ID, time.Now().UTC())
	if err != nil {
		cm.logger.Error("Failed to resolve pending close rating cycle before auto-reopen", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	return cycle != nil
}

func (cm *ConnectionManager) maybeCaptureChatCloseRating(
	ctx context.Context,
	orgID uuid.UUID,
	contact *models.Contact,
	incomingMessage *models.Message,
) bool {
	if contact == nil || incomingMessage == nil {
		return false
	}
	if incomingMessage.Direction != models.DirectionIncoming || incomingMessage.MessageType != models.MessageTypeText {
		return false
	}

	now := time.Now().UTC()
	cycle, followup, err := cm.findActiveChatCloseRatingCycle(ctx, orgID, contact.ID, now)
	if err != nil {
		cm.logger.Error("Failed to resolve close rating cycle", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	if cycle == nil {
		return false
	}

	trimmedContent := strings.TrimSpace(incomingMessage.Content)
	ratingValue, hasRating := parseInboundRatingValue(incomingMessage.Content)
	contextMessages := cycle.ContextMessages
	if hasRating {
		contextMessages = cm.buildChatCloseRatingContext(ctx, contact.ID, incomingMessage)
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
	appendChatCloseRatingFollowupEntry(&followup, incomingMessage, incomingMessage.Content, entryKind, ratingPointer)
	if !hasRating && trimmedContent != "" {
		followup.Comments = append(followup.Comments, trimmedContent)
	}
	if followup.RemainingMessages > 0 {
		followup.RemainingMessages--
	}
	contextMessages = writeChatCloseRatingFollowupState(contextMessages, followup)

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

	result := cm.db.WithContext(ctx).
		Model(&models.ChatClosureRating{}).
		Where("id = ? AND state IN ?", cycle.ID, []models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		}).
		Updates(updates)
	if result.Error != nil {
		cm.logger.Error("Failed to persist chat close rating follow-up", "error", result.Error, "cycle_id", cycle.ID)
		return false
	}
	if result.RowsAffected == 0 {
		return false
	}

	if hasRating {
		cm.logger.Info("Captured chat close rating", "cycle_id", cycle.ID, "contact_id", contact.ID, "rating", ratingValue)
	} else {
		cm.logger.Info("Captured chat close rating follow-up comment", "cycle_id", cycle.ID, "contact_id", contact.ID)
	}
	return true
}

func (cm *ConnectionManager) buildChatCloseRatingContext(
	ctx context.Context,
	contactID uuid.UUID,
	ratingMessage *models.Message,
) models.JSONB {
	if ratingMessage == nil {
		return models.JSONB{}
	}

	var before []models.Message
	cm.db.WithContext(ctx).
		Where("contact_id = ? AND id <> ? AND created_at <= ?", contactID, ratingMessage.ID, ratingMessage.CreatedAt).
		Order("created_at DESC").
		Limit(2).
		Find(&before)

	var after []models.Message
	cm.db.WithContext(ctx).
		Where("contact_id = ? AND id <> ? AND created_at >= ?", contactID, ratingMessage.ID, ratingMessage.CreatedAt).
		Order("created_at ASC").
		Limit(2).
		Find(&after)

	for i, j := 0, len(before)-1; i < j; i, j = i+1, j-1 {
		before[i], before[j] = before[j], before[i]
	}

	return models.JSONB{
		"before": mapMessagesForRatingContext(before),
		"rating": mapSingleMessageForRatingContext(ratingMessage),
		"after":  mapMessagesForRatingContext(after),
	}
}

func mapMessagesForRatingContext(messages []models.Message) []any {
	entries := make([]any, 0, len(messages))
	for i := range messages {
		entries = append(entries, mapSingleMessageForRatingContext(&messages[i]))
	}
	return entries
}

func mapSingleMessageForRatingContext(message *models.Message) models.JSONB {
	if message == nil {
		return models.JSONB{}
	}
	return models.JSONB{
		"id":           message.ID.String(),
		"direction":    message.Direction,
		"message_type": message.MessageType,
		"content":      message.Content,
		"created_at":   message.CreatedAt.Format(time.RFC3339),
	}
}
