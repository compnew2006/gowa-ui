package perinstanceuploadscleanup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRetentionUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		inherit       bool
		retentionDays *int
		wantErr       bool
		errContains   string
	}{
		{
			name:          "inherit true with nil retention_days returns no error",
			inherit:       true,
			retentionDays: nil,
			wantErr:       false,
		},
		{
			name:          "inherit true with non-nil retention_days returns no error",
			inherit:       true,
			retentionDays: intPtr(30),
			wantErr:       false,
		},
		{
			name:          "inherit false with nil retention_days returns required error",
			inherit:       false,
			retentionDays: nil,
			wantErr:       true,
			errContains:   "uploads_cleanup_retention_days_required",
		},
		{
			name:          "inherit false with retention_days 0 returns no error",
			inherit:       false,
			retentionDays: intPtr(0),
			wantErr:       false,
		},
		{
			name:          "inherit false with retention_days 30 returns no error",
			inherit:       false,
			retentionDays: intPtr(30),
			wantErr:       false,
		},
		{
			name:          "inherit false with retention_days 3650 returns no error",
			inherit:       false,
			retentionDays: intPtr(3650),
			wantErr:       false,
		},
		{
			name:          "inherit false with negative retention_days returns error",
			inherit:       false,
			retentionDays: intPtr(-1),
			wantErr:       true,
			errContains:   "uploads_cleanup_retention_days",
		},
		{
			name:          "inherit false with retention_days above max returns error",
			inherit:       false,
			retentionDays: intPtr(3651),
			wantErr:       true,
			errContains:   "uploads_cleanup_retention_days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateRetentionUpdate(tt.inherit, tt.retentionDays)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// intPtr is a test helper that returns a pointer to the given int.
func intPtr(v int) *int {
	return &v
}
