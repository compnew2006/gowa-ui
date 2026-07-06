package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestParseDeleteChatsQueryFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		queryValue  string
		expectBool  bool
		expectError bool
	}{
		{
			name:        "empty query returns false",
			queryValue:  "",
			expectBool:  false,
			expectError: false,
		},
		{
			name:        "true lowercase",
			queryValue:  "true",
			expectBool:  true,
			expectError: false,
		},
		{
			name:        "false lowercase",
			queryValue:  "false",
			expectBool:  false,
			expectError: false,
		},
		{
			name:        "True capitalized",
			queryValue:  "True",
			expectBool:  true,
			expectError: false,
		},
		{
			name:        "False capitalized",
			queryValue:  "False",
			expectBool:  false,
			expectError: false,
		},
		{
			name:        "TRUE uppercase",
			queryValue:  "TRUE",
			expectBool:  true,
			expectError: false,
		},
		{
			name:        "FALSE uppercase",
			queryValue:  "FALSE",
			expectBool:  false,
			expectError: false,
		},
		{
			name:        "1",
			queryValue:  "1",
			expectBool:  true,
			expectError: false,
		},
		{
			name:        "0",
			queryValue:  "0",
			expectBool:  false,
			expectError: false,
		},
		{
			name:        "invalid value",
			queryValue:  "yes",
			expectBool:  false,
			expectError: true,
		},
		{
			name:        "whitespace is trimmed",
			queryValue:  "  true  ",
			expectBool:  true,
			expectError: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := testutil.NewRequest(t)
			if tc.queryValue != "" {
				testutil.SetQueryParam(req, "delete_chats", tc.queryValue)
			}

			result, err := parseDeleteChatsQueryFlag(req)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectBool, result)
			}
		})
	}
}
