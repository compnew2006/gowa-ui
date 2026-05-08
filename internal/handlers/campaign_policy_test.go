package handlers

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignPolicyViolationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *campaignPolicyViolationError
		want string
	}{
		{
			name: "empty message uses derived fallback",
			err:  newCampaignPolicyViolationError("", ""),
			want: "campaign policy violation",
		},
		{
			name: "blank message fallback",
			err:  newCampaignPolicyViolationError("   ", ""),
			want: "campaign policy violation",
		},
		{
			name: "returns message",
			err:  newCampaignPolicyViolationError("blocked by policy", ""),
			want: "blocked by policy",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.err.Error())
		})
	}
}

func TestAsCampaignPolicyViolation(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		message, reason, ok := asCampaignPolicyViolation(nil)
		assert.False(t, ok)
		assert.Empty(t, message)
		assert.Empty(t, reason)
	})

	t.Run("non policy error", func(t *testing.T) {
		t.Parallel()
		message, reason, ok := asCampaignPolicyViolation(errors.New("boom"))
		assert.False(t, ok)
		assert.Empty(t, message)
		assert.Empty(t, reason)
	})

	t.Run("policy error direct", func(t *testing.T) {
		t.Parallel()
		err := newCampaignPolicyViolationError("draft only", "  draft_only  ")
		message, reason, ok := asCampaignPolicyViolation(err)
		require.True(t, ok)
		assert.Equal(t, "draft only", message)
		assert.Equal(t, "draft_only", reason)
	})

	t.Run("policy error wrapped", func(t *testing.T) {
		t.Parallel()
		inner := newCampaignPolicyViolationError("instance blocked", ReasonCodeInstanceBlocked)
		wrapped := errors.Join(errors.New("transport layer"), inner)

		message, reason, ok := asCampaignPolicyViolation(wrapped)
		require.True(t, ok)
		assert.Equal(t, "instance blocked", message)
		assert.Equal(t, ReasonCodeInstanceBlocked, reason)
	})
}

func TestValidateCampaignDelayFloor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		minDelay  int
		maxDelay  int
		floor     int
		wantError bool
	}{
		{
			name:      "disabled floor allows any values",
			minDelay:  0,
			maxDelay:  1,
			floor:     0,
			wantError: false,
		},
		{
			name:      "valid delays at floor",
			minDelay:  10,
			maxDelay:  10,
			floor:     10,
			wantError: false,
		},
		{
			name:      "valid delays above floor",
			minDelay:  12,
			maxDelay:  24,
			floor:     10,
			wantError: false,
		},
		{
			name:      "min below floor fails",
			minDelay:  9,
			maxDelay:  20,
			floor:     10,
			wantError: true,
		},
		{
			name:      "max below floor fails",
			minDelay:  10,
			maxDelay:  9,
			floor:     10,
			wantError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCampaignDelayFloor(tc.minDelay, tc.maxDelay, tc.floor)
			if tc.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "at least")
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCampaignDelayFloorSeconds(t *testing.T) {
	t.Parallel()

	app := &App{}
	assert.Equal(t, strictCampaignDelayFloorSeconds, app.campaignDelayFloorSeconds(uuid.New()))
	assert.Equal(t, strictCampaignDelayFloorSeconds, app.campaignDelayFloorSeconds(uuid.Nil))
}
