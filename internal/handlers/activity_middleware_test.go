package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParseContextUUID(t *testing.T) {
	t.Parallel()

	validUUID := uuid.New()

	tests := []struct {
		name      string
		value     any
		expectOK  bool
		expectNil bool
	}{
		{
			name:      "valid uuid.UUID",
			value:     validUUID,
			expectOK:  true,
			expectNil: false,
		},
		{
			name:      "valid uuid string",
			value:     validUUID.String(),
			expectOK:  true,
			expectNil: false,
		},
		{
			name:      "invalid string",
			value:     "not-a-uuid",
			expectOK:  false,
			expectNil: true,
		},
		{
			name:      "empty string",
			value:     "",
			expectOK:  false,
			expectNil: true,
		},
		{
			name:      "nil value",
			value:     nil,
			expectOK:  false,
			expectNil: true,
		},
		{
			name:      "integer",
			value:     123,
			expectOK:  false,
			expectNil: true,
		},
		{
			name:      "uuid.Nil",
			value:     uuid.Nil,
			expectOK:  true,
			expectNil: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, ok := parseContextUUID(tc.value)
			assert.Equal(t, tc.expectOK, ok)
			if tc.expectNil {
				assert.Equal(t, uuid.Nil, result)
			} else {
				assert.NotEqual(t, uuid.Nil, result)
			}
		})
	}
}
