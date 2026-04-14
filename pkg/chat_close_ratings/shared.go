package chat_close_ratings

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	DefaultFollowupWindowMinutes = 15
	MaxFollowupWindowMinutes     = 1440
	FollowupMessageLimit         = 3

	FollowupContextKey  = "followup"
	FollowupEntriesKey  = "entries"
	FollowupCommentsKey = "comments"
)

var LocalizedRatingDigitReplacer = strings.NewReplacer(
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
	"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
	"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
)

func ParseFollowupWindowMinutes(raw any) int {
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
		return DefaultFollowupWindowMinutes
	}
	if parsed > MaxFollowupWindowMinutes {
		return MaxFollowupWindowMinutes
	}
	return parsed
}

func ParseJSONTime(raw any) time.Time {
	switch v := raw.(type) {
	case time.Time:
		return v.UTC()
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}
	case float64:
		return time.Unix(int64(v), 0).UTC()
	case int64:
		return time.Unix(v, 0).UTC()
	}
	return time.Time{}
}

func ParseJSONInt(raw any) (int, bool) {
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

func AppendChatCloseRatingFollowupEntry(
	entries []any,
	incomingMessage *models.Message,
	content string,
	kind string,
	ratingValue *int,
) []any {
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

	return append(entries, entry)
}

func NormalizeInboundRatingText(raw string) string {
	normalized := strings.TrimSpace(LocalizedRatingDigitReplacer.Replace(raw))
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

func ParseInboundRatingValue(raw string) (int, bool) {
	trimmed := NormalizeInboundRatingText(raw)
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

func NormalizeRatingComments(values []string) []string {
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

func MapSingleMessageForRatingContext(message *models.Message) models.JSONB {
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

type FollowupState struct {
	ExpiresAt         time.Time
	RemainingMessages int
	Entries           []any
	Comments          []string
}

func NewFollowupState(closedAt time.Time, followupWindowMinutes int) FollowupState {
	return FollowupState{
		ExpiresAt:         closedAt.Add(time.Duration(followupWindowMinutes) * time.Minute),
		RemainingMessages: FollowupMessageLimit,
		Entries:           make([]any, 0, FollowupMessageLimit),
		Comments:          []string{},
	}
}

func (s FollowupState) IsActive(now time.Time) bool {
	if s.RemainingMessages <= 0 {
		return false
	}
	return !now.After(s.ExpiresAt)
}

func ReadFollowupState(cycleClosedAt time.Time, contextMessages map[string]any, followupWindowMinutes int) FollowupState {
	state := NewFollowupState(cycleClosedAt, followupWindowMinutes)
	if contextMessages == nil {
		return state
	}

	rawFollowup, ok := contextMessages[FollowupContextKey]
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
		if parsed := ParseJSONTime(rawExpiresAt); !parsed.IsZero() {
			state.ExpiresAt = parsed
		}
	}
	if rawRemaining, ok := payload["remaining_messages"]; ok {
		if parsed, ok := ParseJSONInt(rawRemaining); ok {
			state.RemainingMessages = parsed
		}
	}
	if rawEntries, ok := payload[FollowupEntriesKey]; ok {
		switch typed := rawEntries.(type) {
		case []any:
			state.Entries = append([]any{}, typed...)
		case []map[string]any:
			for _, entry := range typed {
				state.Entries = append(state.Entries, entry)
			}
		}
	}
	if rawComments, ok := payload[FollowupCommentsKey]; ok {
		state.Comments = NormalizeRatingComments(asStringSlice(rawComments))
	}

	if state.RemainingMessages < 0 {
		state.RemainingMessages = 0
	}
	return state
}

func WriteFollowupState(contextMessages map[string]any, state FollowupState) map[string]any {
	if contextMessages == nil {
		contextMessages = make(map[string]any)
	}

	contextMessages[FollowupContextKey] = map[string]any{
		"expires_at":         state.ExpiresAt.UTC().Format(time.RFC3339),
		"remaining_messages": state.RemainingMessages,
		FollowupEntriesKey:   state.Entries,
		FollowupCommentsKey:  state.Comments,
	}
	return contextMessages
}

func asStringSlice(val any) []string {
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
