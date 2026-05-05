package handlers

import (
	"testing"

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
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "single child with different action",
			children: []interface{}{
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "navigate",
					},
				},
			},
			expected: false,
		},
		{
			name: "single child without on-click-action",
			children: []interface{}{
				map[string]interface{}{
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
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "navigate",
					},
				},
				map[string]interface{}{
					"type": "text",
				},
			},
			expected: false,
		},
		{
			name: "multiple children, one with complete action",
			children: []interface{}{
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "navigate",
					},
				},
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple children, complete action is first",
			children: []interface{}{
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "complete",
					},
				},
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "navigate",
					},
				},
			},
			expected: true,
		},
		{
			name: "on-click-action is not a map",
			children: []interface{}{
				map[string]interface{}{
					"type":            "button",
					"on-click-action": "string",
				},
			},
			expected: false,
		},
		{
			name: "on-click-action name is not a string",
			children: []interface{}{
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
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
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "complete",
					},
				},
			},
			expected: true,
		},
		{
			name: "complete action with different casing",
			children: []interface{}{
				map[string]interface{}{
					"type": "button",
					"on-click-action": map[string]interface{}{
						"name": "Complete",
					},
				},
			},
			expected: false,
		},
		{
			name: "complete action with extra properties",
			children: []interface{}{
				map[string]interface{}{
					"type":  "button",
					"label": "Submit",
					"on-click-action": map[string]interface{}{
						"name": "complete",
						"payload": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "text",
							},
						},
					},
				},
				map[string]interface{}{
					"id": "SCREEN_2",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
									"name": "complete",
								},
							},
						},
					},
				},
				map[string]interface{}{
					"id": "SCREEN_2",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{},
					},
				},
				map[string]interface{}{
					"id": "SCREEN_2",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
									"name": "complete",
								},
							},
						},
					},
				},
				map[string]interface{}{
					"id": "SCREEN_3",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
				},
			},
			expectError: true,
			errorMsg:    "Complete Flow",
		},
		{
			name: "screen with non-map layout",
			screens: []interface{}{
				map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{},
					},
				},
				map[string]interface{}{
					"id": "SCREEN_2",
					"layout": map[string]interface{}{
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
				map[string]interface{}{
					"id": "SCREEN_1",
					"layout": map[string]interface{}{
						"children": []interface{}{
							map[string]interface{}{
								"type": "container",
							},
							map[string]interface{}{
								"type": "footer",
								"on-click-action": map[string]interface{}{
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
