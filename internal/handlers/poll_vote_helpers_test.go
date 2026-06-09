package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNormalizePollVoteSelectedOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{name: "nil input", input: nil, expected: []string{}},
		{name: "empty input", input: []string{}, expected: []string{}},
		{name: "single option", input: []string{"Yes"}, expected: []string{"Yes"}},
		{name: "trims whitespace", input: []string{" Yes ", " No "}, expected: []string{"Yes", "No"}},
		{name: "removes duplicates", input: []string{"Yes", "Yes", "No"}, expected: []string{"Yes", "No"}},
		{name: "skips empty strings", input: []string{"Yes", "", "  ", "No"}, expected: []string{"Yes", "No"}},
		{name: "preserves order", input: []string{"C", "B", "A"}, expected: []string{"C", "B", "A"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := normalizePollVoteSelectedOptions(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPollVoteIntValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{name: "int", input: 5, expected: 5},
		{name: "int32", input: int32(3), expected: 3},
		{name: "int64", input: int64(7), expected: 7},
		{name: "float32", input: float32(2.0), expected: 2},
		{name: "float64", input: float64(4.0), expected: 4},
		{name: "string numeric", input: "6", expected: 6},
		{name: "string with spaces", input: " 3 ", expected: 3},
		{name: "string non-numeric", input: "abc", expected: 0},
		{name: "nil", input: nil, expected: 0},
		{name: "bool", input: true, expected: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, pollVoteIntValue(tc.input))
		})
	}
}

func TestPollVoteSelectionLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     models.JSONB
		expected int
	}{
		{
			name:     "max_selections present",
			data:     models.JSONB{"max_selections": 3},
			expected: 3,
		},
		{
			name:     "falls back to selectable_options_count",
			data:     models.JSONB{"selectable_options_count": 2},
			expected: 2,
		},
		{
			name:     "max_selections takes precedence",
			data:     models.JSONB{"max_selections": 1, "selectable_options_count": 5},
			expected: 1,
		},
		{
			name:     "defaults to 1 when neither set",
			data:     models.JSONB{},
			expected: 1,
		},
		{
			name:     "zero max_selections falls back",
			data:     models.JSONB{"max_selections": 0, "selectable_options_count": 4},
			expected: 4,
		},
		{
			name:     "both zero defaults to 999",
			data:     models.JSONB{"max_selections": 0, "selectable_options_count": 0},
			expected: 999,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, pollVoteSelectionLimit(tc.data))
		})
	}
}

func TestPollVoteSelectionVoters(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns empty map", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, map[string][]string{}, pollVoteSelectionVoters(nil))
	})

	t.Run("map[string][]string passthrough", func(t *testing.T) {
		t.Parallel()
		input := map[string][]string{"user1": {"A", "B"}}
		assert.Equal(t, map[string][]string{"user1": {"A", "B"}}, pollVoteSelectionVoters(input))
	})

	t.Run("map[string]interface with interface slices", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"user1": []any{"A", "B"}, "user2": []any{"C"}}
		assert.Equal(t, map[string][]string{"user1": {"A", "B"}, "user2": {"C"}}, pollVoteSelectionVoters(input))
	})

	t.Run("filters empty strings from interface slice", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"user1": []any{"A", "", "B"}}
		assert.Equal(t, map[string][]string{"user1": {"A", "B"}}, pollVoteSelectionVoters(input))
	})

	t.Run("unrecognized type returns empty", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, map[string][]string{}, pollVoteSelectionVoters("not a map"))
	})
}

func TestPollVoteSelectionCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		voters   map[string][]string
		expected map[string]int
	}{
		{name: "empty", voters: map[string][]string{}, expected: map[string]int{}},
		{name: "single voter single option", voters: map[string][]string{"u1": {"A"}}, expected: map[string]int{"A": 1}},
		{name: "multiple voters same option", voters: map[string][]string{"u1": {"A"}, "u2": {"A"}}, expected: map[string]int{"A": 2}},
		{name: "multiple options", voters: map[string][]string{"u1": {"A", "B"}, "u2": {"A"}}, expected: map[string]int{"A": 2, "B": 1}},
		{name: "skips empty strings", voters: map[string][]string{"u1": {"A", "  ", ""}}, expected: map[string]int{"A": 1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, pollVoteSelectionCounts(tc.voters))
		})
	}
}

func TestApplyPollVoteSelectionToInteractive(t *testing.T) {
	t.Parallel()

	t.Run("adds new voter", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{"type": "poll", "options": []any{"A", "B"}}
		result := applyPollVoteSelectionToInteractive(existing, "user1", []string{"A"})

		voters := result["voters"].(map[string][]string)
		assert.Equal(t, []string{"A"}, voters["user1"])
		assert.Equal(t, map[string]int{"A": 1}, result["votes"])
		assert.Equal(t, 1, result["total_votes"])
		assert.Equal(t, []string{"A"}, result["last_selected_options"])
		assert.Equal(t, "user1", result["last_voter"])
	})

	t.Run("removes voter on empty selection", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{
			"type":   "poll",
			"voters": map[string]any{"user1": []any{"A"}},
			"votes":  map[string]any{"A": 1},
		}
		result := applyPollVoteSelectionToInteractive(existing, "user1", []string{})

		voters := result["voters"].(map[string][]string)
		_, exists := voters["user1"]
		assert.False(t, exists, "voter should be removed on empty selection")
		assert.Equal(t, 0, result["total_votes"])
	})

	t.Run("preserves existing fields", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{"type": "poll", "question": "Pick one", "options": []any{"A"}}
		result := applyPollVoteSelectionToInteractive(existing, "u1", []string{"A"})
		assert.Equal(t, "Pick one", result["question"])
		assert.Equal(t, "poll", result["type"])
	})

	t.Run("sets type to poll when missing", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{"options": []any{"A"}}
		result := applyPollVoteSelectionToInteractive(existing, "u1", []string{"A"})
		assert.Equal(t, "poll", result["type"])
	})

	t.Run("replaces existing voter selection", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{
			"type":   "poll",
			"voters": map[string]any{"user1": []any{"A"}},
			"votes":  map[string]any{"A": 1},
		}
		result := applyPollVoteSelectionToInteractive(existing, "user1", []string{"B"})

		voters := result["voters"].(map[string][]string)
		assert.Equal(t, []string{"B"}, voters["user1"])
		assert.Equal(t, map[string]int{"B": 1}, result["votes"])
	})

	t.Run("does not mutate original", func(t *testing.T) {
		t.Parallel()
		existing := models.JSONB{"type": "poll", "options": []any{"A"}}
		originalVoters := existing["voters"]
		_ = applyPollVoteSelectionToInteractive(existing, "u1", []string{"A"})
		assert.Equal(t, originalVoters, existing["voters"])
	})
}

func TestBroadcastPollMessageUpdate_NilHub_NilMessage(t *testing.T) {
	t.Parallel()
	app := &App{WSHub: nil}
	// Should not panic with nil hub or nil message
	app.broadcastPollMessageUpdate(uuid.Nil, nil)
}
