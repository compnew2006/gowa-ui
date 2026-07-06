package whatsmeow

import (
	"testing"
)

func TestSecureRandomInt63n(t *testing.T) {
	tests := []struct {
		name string
		n    int64
	}{
		{"n less than 1", 0},
		{"n equals 1", 1},
		{"n equals 10", 10},
		{"n equals 100", 100},
		{"n equals 1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := secureRandomInt63n(tt.n)
			if tt.n <= 1 {
				if result != 0 {
					t.Errorf("Expected 0 for n <= 1, got %d", result)
				}
			} else {
				if result < 0 || result >= tt.n {
					t.Errorf("Expected result in range [0, %d), got %d", tt.n, result)
				}
			}
		})
	}
}

func TestSecureRandomIntn(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"n less than 1", 0},
		{"n equals 1", 1},
		{"n equals 10", 10},
		{"n equals 100", 100},
		{"n equals 1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := secureRandomIntn(tt.n)
			if tt.n <= 1 {
				if result != 0 {
					t.Errorf("Expected 0 for n <= 1, got %d", result)
				}
			} else {
				if result < 0 || result >= tt.n {
					t.Errorf("Expected result in range [0, %d), got %d", tt.n, result)
				}
			}
		})
	}
}
