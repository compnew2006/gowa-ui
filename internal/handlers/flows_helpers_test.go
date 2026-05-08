package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHasCompleteAction(t *testing.T) {
	tests := []struct {
		name     string
		children []interface{}
		expected bool
	}{
		{
			name:     "empty children list",
			children: []interface{}{},
			expected: false,
		},
		{
			name: "single child with complete action",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "single child with different action",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
			},
			expected: false,
		},
		{
			name: "single child without on-click-action",
			children: []interface{}{
				map[string]any{
					"type": "text",
				},
			},
			expected: false,
		},
		{
			name: "single child with non-map type",
			children: []interface{}{
				"string",
			},
			expected: false,
		},
		{
			name: "multiple children, none with complete action",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
				map[string]any{
					"type": "text",
				},
			},
			expected: false,
		},
		{
			name: "multiple children, one with complete action",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple children, complete action is first",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
			},
			expected: true,
		},
		{
			name: "on-click-action is not a map",
			children: []interface{}{
				map[string]any{
					"type":            "button",
					"on-click-action": "string",
				},
			},
			expected: false,
		},
		{
			name: "on-click-action name is not a string",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": 123,
					},
				},
			},
			expected: false,
		},
		{
			name: "child is not a map",
			children: []interface{}{
				"not a map",
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "complete action with different casing",
			children: []interface{}{
				map[string]any{
					"type": "button",
					"on-click-action": map[string]any{
						"name": "Complete",
					},
				},
			},
			expected: false,
		},
		{
			name: "complete action with extra properties",
			children: []interface{}{
				map[string]any{
					"type":  "button",
					"label": "Submit",
					"on-click-action": map[string]any{
						"name": "complete",
						"payload": map[string]any{
							"screen": "next",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCompleteAction(tt.children)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateFlowStructure(t *testing.T) {
	tests := []struct {
		name        string
		screens     []interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty screens",
			screens:     []interface{}{},
			expectError: true,
			errorMsg:    "flow must have at least one screen",
		},
		{
			name: "single screen with complete action",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "single screen without complete action",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "text",
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "two screens, complete action on last screen only",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "text",
							},
						},
					},
				},
				map[string]any{
					"id": "SCREEN_2",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "two screens, complete action on first screen",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
				map[string]any{
					"id": "SCREEN_2",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "text",
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "should only be on the last screen",
		},
		{
			name: "three screens, complete action on middle and last",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{},
					},
				},
				map[string]any{
					"id": "SCREEN_2",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
				map[string]any{
					"id": "SCREEN_3",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "should only be on the last screen",
		},
		{
			name: "screen without layout",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "screen with non-map layout",
			screens: []interface{}{
				map[string]any{
					"id":     "SCREEN_1",
					"layout": "invalid",
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "screen with layout but no children",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{},
					},
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "screen with non-array children",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": "not an array",
					},
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "multiple screens, none have complete action",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{},
					},
				},
				map[string]any{
					"id": "SCREEN_2",
					"layout": map[string]any{
						"children": []interface{}{},
					},
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "screen with nested components, complete action deep",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "container",
							},
							map[string]any{
								"type": "footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "non-map screen item",
			screens: []interface{}{
				"not a map",
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlowStructure(tt.screens)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ID with digit gets converted",
			input:    "SCREEN_1",
			expected: "SCREEN_B", // 1 -> B
		},
		{
			name:     "already valid ID with lowercase",
			input:    "screen_one",
			expected: "screen_one",
		},
		{
			name:     "already valid ID with mixed case and numbers",
			input:    "Screen_One_123",
			expected: "Screen_One_BCD", // 1->B, 2->C, 3->D
		},
		{
			name:     "ID with numbers converted",
			input:    "id_1234_abc",
			expected: "id_BCDE_abc", // 1->B, 2->C, 3->D, 4->E
		},
		{
			name:     "ID with hyphens dropped",
			input:    "screen-one",
			expected: "screenone", // hyphens are dropped
		},
		{
			name:     "ID with spaces dropped",
			input:    "screen one",
			expected: "screenone", // spaces are dropped
		},
		{
			name:     "ID with special characters dropped",
			input:    "screen@#$",
			expected: "screen", // special chars are dropped
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single digit converted",
			input:    "1",
			expected: "B", // 'A' + (1 - '0') = 'B'
		},
		{
			name:     "mixed valid and invalid",
			input:    "SCREEN_1-test",
			expected: "SCREEN_Btest", // 1->B, hyphen dropped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeScreensForMeta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		screens  []interface{}
		validate func(t *testing.T, result []interface{})
	}{
		{
			name:    "nil returns empty",
			screens: nil,
			validate: func(t *testing.T, result []interface{}) {
				assert.Empty(t, result)
			},
		},
		{
			name:    "empty returns empty",
			screens: []interface{}{},
			validate: func(t *testing.T, result []interface{}) {
				assert.Empty(t, result)
			},
		},
		{
			name: "screen with no components gets sanitized IDs",
			screens: []interface{}{
				map[string]any{
					"id":     "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				screen := result[0].(map[string]any)
				assert.Equal(t, "SCREEN_B", screen["id"])
			},
		},
		{
			name: "screens with components get sanitized",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "field_1",
								"id":  "comp_1",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				screen := result[0].(map[string]any)
				assert.Equal(t, "SCREEN_B", screen["id"])
				layout := screen["layout"].(map[string]any)
				children := layout["children"].([]interface{})
				comp := children[0].(map[string]any)
				assert.Equal(t, "field_B", comp["name"])
				assert.NotContains(t, comp, "id")
			},
		},
		{
			name: "terminal screen gets marked",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "Footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				screen := result[0].(map[string]any)
				assert.Equal(t, true, screen["terminal"])
			},
		},
		{
			name: "multi-screen data model propagated",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
				map[string]any{
					"id": "SCREEN_2",
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "Footer",
								"on-click-action": map[string]any{
									"name": "complete",
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				screen2 := result[1].(map[string]any)
				data := screen2["data"].(map[string]any)
				assert.Contains(t, data, "email")
				emailEntry := data["email"].(map[string]any)
				assert.Equal(t, "string", emailEntry["type"])
			},
		},
		{
			name: "non-map screen passes through unchanged",
			screens: []interface{}{
				"not a map",
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				assert.Equal(t, "not a map", result[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeScreensForMeta(tt.screens)
			tt.validate(t, result)
		})
	}
}

func TestCollectFormFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		screens  []interface{}
		expected []string
	}{
		{
			name:     "nil returns empty",
			screens:  nil,
			expected: nil,
		},
		{
			name:     "empty returns empty",
			screens:  []interface{}{},
			expected: nil,
		},
		{
			name: "collects field names from components",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
							map[string]any{
								"type": "Dropdown",
								"name": "country",
							},
						},
					},
				},
			},
			expected: []string{"email", "country"},
		},
		{
			name: "sanitizes field names with numbers",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "field_1",
							},
						},
					},
				},
			},
			expected: []string{"field_B"},
		},
		{
			name: "skips components without name",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextBody",
							},
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
			},
			expected: []string{"email"},
		},
		{
			name: "handles screens without layout",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
				},
			},
			expected: nil,
		},
		{
			name: "handles empty name",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "",
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "does not deduplicate across screens",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
			},
			expected: []string{"email", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := collectFormFieldNames(tt.screens)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectFormFieldsPerScreen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		screens  []interface{}
		expected map[int][]string
	}{
		{
			name:     "nil returns empty map",
			screens:  nil,
			expected: map[int][]string{},
		},
		{
			name:     "empty returns empty map",
			screens:  []interface{}{},
			expected: map[int][]string{},
		},
		{
			name: "maps each screen to its fields",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "Dropdown",
								"name": "country",
							},
						},
					},
				},
			},
			expected: map[int][]string{
				0: {"email"},
				1: {"country"},
			},
		},
		{
			name: "screen with no fields is omitted from map",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextBody",
							},
						},
					},
				},
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "email",
							},
						},
					},
				},
			},
			expected: map[int][]string{
				1: {"email"},
			},
		},
		{
			name: "handles screen without layout",
			screens: []interface{}{
				map[string]any{
					"id": "SCREEN_1",
				},
			},
			expected: map[int][]string{},
		},
		{
			name: "sanitizes field names",
			screens: []interface{}{
				map[string]any{
					"layout": map[string]any{
						"children": []interface{}{
							map[string]any{
								"type": "TextInput",
								"name": "field_1",
							},
						},
					},
				},
			},
			expected: map[int][]string{
				0: {"field_B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := collectFormFieldsPerScreen(tt.screens)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeComponentsWithPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		children []interface{}
		allFields []string
		prevFields []string
		validate  func(t *testing.T, result []interface{})
	}{
		{
			name:      "nil returns empty",
			children:  nil,
			allFields: nil,
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				assert.Empty(t, result)
			},
		},
		{
			name: "removes id from components without ID support",
			children: []interface{}{
				map[string]any{
					"type": "TextInput",
					"name": "email",
					"id":   "comp_1",
				},
			},
			allFields:  []string{"email"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				assert.NotContains(t, comp, "id")
			},
		},
		{
			name: "preserves id on components with ID support",
			children: []interface{}{
				map[string]any{
					"type": "Button",
					"name": "btn_1",
					"id":   "btn_comp",
				},
			},
			allFields:  []string{"email"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				assert.Equal(t, "btn_comp", comp["id"])
			},
		},
		{
			name: "sanitizes component name with numbers",
			children: []interface{}{
				map[string]any{
					"type": "TextInput",
					"name": "field_1",
				},
			},
			allFields:  []string{"field_B"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				assert.Equal(t, "field_B", comp["name"])
			},
		},
		{
			name: "complete action gets auto-populated payload with data refs when no current-screen fields",
			children: []interface{}{
				map[string]any{
					"type": "Footer",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			allFields:  []string{"email", "phone"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				action := comp["on-click-action"].(map[string]any)
				payload := action["payload"].(map[string]any)
				assert.Equal(t, "${data.email}", payload["email"])
				assert.Equal(t, "${data.phone}", payload["phone"])
			},
		},
		{
			name: "complete action uses form refs for current-screen fields",
			children: []interface{}{
				map[string]any{
					"type": "TextInput",
					"name": "email",
				},
				map[string]any{
					"type": "TextInput",
					"name": "phone",
				},
				map[string]any{
					"type": "Footer",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			allFields:  []string{"email", "phone"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[2].(map[string]any)
				action := comp["on-click-action"].(map[string]any)
				payload := action["payload"].(map[string]any)
				assert.Equal(t, "${form.email}", payload["email"])
				assert.Equal(t, "${form.phone}", payload["phone"])
			},
		},
		{
			name: "complete action uses data reference for previous screen fields",
			children: []interface{}{
				map[string]any{
					"type": "TextInput",
					"name": "address",
				},
				map[string]any{
					"type": "Footer",
					"on-click-action": map[string]any{
						"name": "complete",
					},
				},
			},
			allFields:  []string{"email", "address"},
			prevFields: []string{"email"},
			validate: func(t *testing.T, result []interface{}) {
				comp := result[1].(map[string]any)
				action := comp["on-click-action"].(map[string]any)
				payload := action["payload"].(map[string]any)
				assert.Equal(t, "${data.email}", payload["email"])
				assert.Equal(t, "${form.address}", payload["address"])
			},
		},
		{
			name: "navigate action gets payload with current and previous fields",
			children: []interface{}{
				map[string]any{
					"type": "TextInput",
					"name": "email",
				},
				map[string]any{
					"type": "Footer",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
			},
			allFields:  []string{"email", "phone"},
			prevFields: []string{"phone"},
			validate: func(t *testing.T, result []interface{}) {
				comp := result[1].(map[string]any)
				action := comp["on-click-action"].(map[string]any)
				payload := action["payload"].(map[string]any)
				assert.Equal(t, "${data.phone}", payload["phone"])
				assert.Equal(t, "${form.email}", payload["email"])
			},
		},
		{
			name: "navigate action with no fields has no payload",
			children: []interface{}{
				map[string]any{
					"type": "Footer",
					"on-click-action": map[string]any{
						"name": "navigate",
					},
				},
			},
			allFields:  []string{},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				action := comp["on-click-action"].(map[string]any)
				assert.NotContains(t, action, "payload")
			},
		},
		{
			name: "non-map child passes through",
			children: []interface{}{
				"not a map",
			},
			allFields:  nil,
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				assert.Equal(t, "not a map", result[0])
			},
		},
		{
			name: "sanitizes data-source option IDs",
			children: []interface{}{
				map[string]any{
					"type": "Dropdown",
					"name": "country",
					"data-source": []interface{}{
						map[string]any{"id": "opt_1", "title": "US"},
						map[string]any{"id": "opt_2", "title": "UK"},
					},
				},
			},
			allFields:  []string{"country"},
			prevFields: nil,
			validate: func(t *testing.T, result []interface{}) {
				comp := result[0].(map[string]any)
				ds := comp["data-source"].([]interface{})
				assert.Equal(t, "opt_B", ds[0].(map[string]any)["id"])
				assert.Equal(t, "opt_C", ds[1].(map[string]any)["id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeComponentsWithPayload(tt.children, tt.allFields, tt.prevFields)
			tt.validate(t, result)
		})
	}
}

func TestFlowToResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		flow     models.WhatsAppFlow
		expected FlowResponse
	}{
		{
			name: "zero UUID defaults",
			flow: models.WhatsAppFlow{},
			expected: FlowResponse{
				CreatedAt: "0001-01-01T00:00:00Z",
				UpdatedAt: "0001-01-01T00:00:00Z",
			},
		},
		{
			name: "all fields mapped correctly",
			flow: models.WhatsAppFlow{
				BaseModel: models.BaseModel{
					ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				OrganizationID:  uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
				WhatsAppAccount: "test-account",
				MetaFlowID:      "meta-123",
				Name:            "Test Flow",
				Status:          "DRAFT",
				Category:        "UTILITY",
				JSONVersion:     "6.0",
				FlowJSON:        models.JSONB{"key": "value"},
				Screens:         models.JSONBArray{map[string]any{"id": "SCREEN_1"}},
				PreviewURL:      "https://example.com/preview",
				HasLocalChanges: true,
			},
			expected: FlowResponse{
				ID:              uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				WhatsAppAccount: "test-account",
				MetaFlowID:      "meta-123",
				Name:            "Test Flow",
				Status:          "DRAFT",
				Category:        "UTILITY",
				JSONVersion:     "6.0",
				FlowJSON:        map[string]any{"key": "value"},
				Screens:         []interface{}{map[string]any{"id": "SCREEN_1"}},
				PreviewURL:      "https://example.com/preview",
				HasLocalChanges: true,
				CreatedAt:       "2025-06-15T10:30:00Z",
				UpdatedAt:       "2025-06-15T10:30:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := flowToResponse(tt.flow)
			assert.Equal(t, tt.expected, result)
		})
	}
}
