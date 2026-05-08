package chat_close_ratings

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFollowupWindowMinutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  any
		want int
	}{
		{name: "int within range", raw: 30, want: 30},
		{name: "int32", raw: int32(60), want: 60},
		{name: "int64", raw: int64(120), want: 120},
		{name: "float64", raw: float64(45), want: 45},
		{name: "string valid", raw: "20", want: 20},
		{name: "string with spaces", raw: "  25  ", want: 25},
		{name: "below minimum returns default", raw: 0, want: DefaultFollowupWindowMinutes},
		{name: "negative returns default", raw: -5, want: DefaultFollowupWindowMinutes},
		{name: "above maximum clamped", raw: 9999, want: MaxFollowupWindowMinutes},
		{name: "exactly max", raw: MaxFollowupWindowMinutes, want: MaxFollowupWindowMinutes},
		{name: "exactly 1", raw: 1, want: 1},
		{name: "nil returns default", raw: nil, want: DefaultFollowupWindowMinutes},
		{name: "invalid string returns default", raw: "abc", want: DefaultFollowupWindowMinutes},
		{name: "bool returns default", raw: true, want: DefaultFollowupWindowMinutes},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ParseFollowupWindowMinutes(tc.raw))
		})
	}
}

func TestParseJSONTime(t *testing.T) {
	t.Parallel()

	rfc3339 := "2024-06-15T10:30:00Z"
	rfc3339Nano := "2024-06-15T10:30:00.123456789Z"
	unixTS := float64(1718441400)
	parsedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		raw  any
		want time.Time
	}{
		{name: "time.Time", raw: parsedTime, want: parsedTime.UTC()},
		{name: "RFC3339 string", raw: rfc3339, want: parsedTime.UTC()},
		{name: "RFC3339Nano string", raw: rfc3339Nano, want: time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC)},
		{name: "float64 unix", raw: unixTS, want: time.Unix(int64(unixTS), 0).UTC()},
		{name: "int64 unix", raw: int64(1718441400), want: time.Unix(1718441400, 0).UTC()},
		{name: "invalid string", raw: "not-a-date", want: time.Time{}},
		{name: "nil", raw: nil, want: time.Time{}},
		{name: "bool", raw: true, want: time.Time{}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ParseJSONTime(tc.raw))
		})
	}
}

func TestParseJSONInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      any
		want     int
		wantBool bool
	}{
		{name: "int", raw: 42, want: 42, wantBool: true},
		{name: "int32", raw: int32(7), want: 7, wantBool: true},
		{name: "int64", raw: int64(99), want: 99, wantBool: true},
		{name: "float64", raw: float64(3.7), want: 3, wantBool: true},
		{name: "string valid", raw: "15", want: 15, wantBool: true},
		{name: "string with spaces", raw: "  8  ", want: 8, wantBool: true},
		{name: "string invalid", raw: "abc", want: 0, wantBool: false},
		{name: "nil", raw: nil, want: 0, wantBool: false},
		{name: "bool", raw: true, want: 0, wantBool: false},
		{name: "negative", raw: -5, want: -5, wantBool: true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseJSONInt(tc.raw)
			assert.Equal(t, tc.wantBool, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeInboundRatingText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "simple digit", raw: "5", want: "5"},
		{name: "with spaces", raw: "  5  ", want: "5"},
		{name: "Arabic-Indic digits", raw: "\u0665", want: "5"},
		{name: "Persian digits", raw: "\u06F5", want: "5"},
		{name: "with text", raw: "rating 5", want: "rating 5"},
		{name: "control chars stripped", raw: "\x005\x00", want: "5"},
		{name: "empty", raw: "", want: ""},
		{name: "only spaces", raw: "   ", want: ""},
		{name: "only ignorable runes", raw: "\u200b\u200c", want: ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, NormalizeInboundRatingText(tc.raw))
		})
	}
}

func TestParseInboundRatingValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		want     int
		wantBool bool
	}{
		{name: "simple 5", raw: "5", want: 5, wantBool: true},
		{name: "1", raw: "1", want: 1, wantBool: true},
		{name: "10", raw: "10", want: 10, wantBool: true},
		{name: "0 out of range", raw: "0", want: 0, wantBool: false},
		{name: "11 out of range", raw: "11", want: 0, wantBool: false},
		{name: "Arabic digit 5", raw: "\u0665", want: 5, wantBool: true},
		{name: "Persian digit 3", raw: "\u06F3", want: 3, wantBool: true},
		{name: "with punctuation after", raw: "5!", want: 5, wantBool: true},
		{name: "with space after", raw: "5 stars", want: 5, wantBool: true},
		{name: "with symbol after", raw: "5\u2605", want: 5, wantBool: true},
		{name: "letter after digit", raw: "5a", want: 0, wantBool: false},
		{name: "pure text", raw: "great", want: 0, wantBool: false},
		{name: "empty", raw: "", want: 0, wantBool: false},
		{name: "with leading text", raw: "rating: 5", want: 0, wantBool: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseInboundRatingValue(tc.raw)
			assert.Equal(t, tc.wantBool, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeRatingComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil returns empty", in: nil, want: []string{}},
		{name: "empty returns empty", in: []string{}, want: []string{}},
		{name: "trims whitespace", in: []string{"  hello  ", "  world  "}, want: []string{"hello", "world"}},
		{name: "filters empty strings", in: []string{"hello", "", "  ", "world"}, want: []string{"hello", "world"}},
		{name: "preserves valid", in: []string{"good service", "fast response"}, want: []string{"good service", "fast response"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, NormalizeRatingComments(tc.in))
		})
	}
}

func TestAppendChatCloseRatingFollowupEntry(t *testing.T) {
	t.Parallel()

	msgID := uuid.New()
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	msg := &models.Message{
		BaseModel:   models.BaseModel{ID: msgID, CreatedAt: now},
		MessageType: "text",
	}

	t.Run("nil message no rating", func(t *testing.T) {
		t.Parallel()
		result := AppendChatCloseRatingFollowupEntry(nil, nil, "thanks", "comment", nil)
		require.Len(t, result, 1)
		entry := result[0].(models.JSONB)
		assert.Equal(t, "comment", entry["kind"])
		assert.Equal(t, "thanks", entry["content"])
		_, hasMsgID := entry["message_id"]
		assert.False(t, hasMsgID)
		_, hasRating := entry["rating"]
		assert.False(t, hasRating)
	})

	t.Run("with message and rating", func(t *testing.T) {
		t.Parallel()
		rating := 5
		result := AppendChatCloseRatingFollowupEntry(nil, msg, "great", "rating", &rating)
		require.Len(t, result, 1)
		entry := result[0].(models.JSONB)
		assert.Equal(t, msgID.String(), entry["message_id"])
		assert.Equal(t, models.MessageType("text"), entry["message_type"])
		assert.Equal(t, now.UTC().Format(time.RFC3339), entry["created_at"])
		assert.Equal(t, 5, entry["rating"])
	})

	t.Run("appends to existing", func(t *testing.T) {
		t.Parallel()
		existing := []any{models.JSONB{"kind": "old"}}
		result := AppendChatCloseRatingFollowupEntry(existing, nil, "new", "comment", nil)
		assert.Len(t, result, 2)
	})
}

func TestMapSingleMessageForRatingContext(t *testing.T) {
	t.Parallel()

	t.Run("nil message", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, models.JSONB{}, MapSingleMessageForRatingContext(nil))
	})

	t.Run("with message", func(t *testing.T) {
		t.Parallel()
		msgID := uuid.New()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		msg := &models.Message{
			BaseModel:   models.BaseModel{ID: msgID, CreatedAt: now},
			Direction:   models.DirectionIncoming,
			MessageType: "text",
			Content:     "hello",
		}
		result := MapSingleMessageForRatingContext(msg)
		assert.Equal(t, msgID.String(), result["id"])
		assert.Equal(t, models.DirectionIncoming, result["direction"])
		assert.Equal(t, models.MessageType("text"), result["message_type"])
		assert.Equal(t, "hello", result["content"])
		assert.Equal(t, now.Format(time.RFC3339), result["created_at"])
	})
}

func TestFollowupState_IsActive(t *testing.T) {
	t.Parallel()

	t.Run("active within window", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := NewFollowupState(now, 15)
		assert.True(t, state.IsActive(now.Add(5*time.Minute)))
	})

	t.Run("expired past window", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := NewFollowupState(now, 15)
		assert.False(t, state.IsActive(now.Add(16*time.Minute)))
	})

	t.Run("exactly at expiry", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := NewFollowupState(now, 15)
		assert.True(t, state.IsActive(now.Add(15*time.Minute)))
	})

	t.Run("no remaining messages", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := FollowupState{
			ExpiresAt:         now.Add(15 * time.Minute),
			RemainingMessages: 0,
		}
		assert.False(t, state.IsActive(now.Add(5*time.Minute)))
	})

	t.Run("negative remaining messages", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := FollowupState{
			ExpiresAt:         now.Add(15 * time.Minute),
			RemainingMessages: -1,
		}
		assert.False(t, state.IsActive(now.Add(5*time.Minute)))
	})
}

func TestReadFollowupState_WriteFollowupState_Roundtrip(t *testing.T) {
	t.Parallel()

	t.Run("nil context returns default state", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := ReadFollowupState(now, nil, 15)
		assert.Equal(t, now.Add(15*time.Minute), state.ExpiresAt)
		assert.Equal(t, FollowupMessageLimit, state.RemainingMessages)
		assert.Empty(t, state.Entries)
		assert.Empty(t, state.Comments)
	})

	t.Run("empty context returns default state", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		state := ReadFollowupState(now, map[string]any{}, 15)
		assert.Equal(t, FollowupMessageLimit, state.RemainingMessages)
	})

	t.Run("roundtrip preserves data", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		original := NewFollowupState(now, 30)
		original.RemainingMessages = 1
		original.Entries = []any{models.JSONB{"kind": "rating", "content": "5"}}
		original.Comments = []string{"great"}

		context := make(map[string]any)
		written := WriteFollowupState(context, original)

		read := ReadFollowupState(now, written, 15)
		assert.Equal(t, original.ExpiresAt, read.ExpiresAt)
		assert.Equal(t, 1, read.RemainingMessages)
		assert.Len(t, read.Entries, 1)
		assert.Equal(t, []string{"great"}, read.Comments)
	})

	t.Run("negative remaining clamped to zero", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		context := map[string]any{
			FollowupContextKey: map[string]any{
				"expires_at":         now.Add(10 * time.Minute).Format(time.RFC3339),
				"remaining_messages": -3,
			},
		}
		state := ReadFollowupState(now, context, 15)
		assert.Equal(t, 0, state.RemainingMessages)
	})

	t.Run("entries from []map[string]any", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		context := map[string]any{
			FollowupContextKey: map[string]any{
				FollowupEntriesKey: []map[string]any{{"kind": "test"}},
			},
		}
		state := ReadFollowupState(now, context, 15)
		assert.Len(t, state.Entries, 1)
	})
}

func TestNewFollowupState(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	state := NewFollowupState(now, 30)

	assert.Equal(t, now.Add(30*time.Minute), state.ExpiresAt)
	assert.Equal(t, FollowupMessageLimit, state.RemainingMessages)
	assert.NotNil(t, state.Entries)
	assert.NotNil(t, state.Comments)
}
