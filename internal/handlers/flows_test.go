package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlowsCollectFormFieldNames(t *testing.T) {
	screens := []interface{}{
		map[string]interface{}{
			"layout": map[string]interface{}{
				"children": []interface{}{
					map[string]interface{}{"type": "TextInput", "name": "email"},
					map[string]interface{}{"type": "TextInput", "name": "email"},
					map[string]interface{}{"type": "TextArea", "name": "comment"},
				},
			},
		},
		map[string]interface{}{
			"layout": map[string]interface{}{
				"children": []interface{}{
					map[string]interface{}{"type": "Dropdown", "name": "country"},
					map[string]interface{}{"type": "Body", "text": "hello"},
				},
			},
		},
	}
	names := collectFormFieldNames(screens)
	assert.Equal(t, []string{"email", "email", "comment", "country"}, names)
}

func TestFlowsSanitizeID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "numbers converted to letters", input: "abc123", want: "abcBCD"},
		{name: "spaces trimmed", input: "  abc  ", want: "abc"},
		{name: "empty", input: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitizeID(tc.input))
		})
	}
}
