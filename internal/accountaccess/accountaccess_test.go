package accountaccess

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseSendPolicy(t *testing.T) {
	tests := []struct {
		name     string
		settings models.JSONB
		want     SendPolicy
	}{
		{name: "nil settings use safe defaults", settings: nil, want: SendPolicy{Enforce: true}},
		{name: "empty settings use safe defaults", settings: models.JSONB{}, want: SendPolicy{Enforce: true}},
		{name: "cross-account enabled", settings: models.JSONB{SettingAllowCrossAccountSend: true},
			want: SendPolicy{Enforce: true, AllowCrossAccount: true}},
		{name: "audit-only mode", settings: models.JSONB{SettingEnforceSendAccountPolicy: false},
			want: SendPolicy{Enforce: false}},
		{name: "non-boolean values are ignored",
			settings: models.JSONB{SettingAllowCrossAccountSend: "true", SettingEnforceSendAccountPolicy: 0},
			want:     SendPolicy{Enforce: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseSendPolicy(tc.settings))
		})
	}
}

func TestAssignmentAllows(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		assigned []uuid.UUID
		account  uuid.UUID
		want     bool
	}{
		{name: "no assignment falls back to full access", assigned: nil, account: a, want: true},
		{name: "assigned account", assigned: []uuid.UUID{a}, account: a, want: true},
		{name: "unassigned account", assigned: []uuid.UUID{a}, account: b, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, AssignmentAllows(tc.assigned, tc.account))
		})
	}
}
