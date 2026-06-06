package perinstanceuploadscleanup

import (
	"fmt"
)

func ValidateRetentionUpdate(inherit bool, retentionDays *int) error {
	if !inherit && retentionDays == nil {
		return fmt.Errorf("uploads_cleanup_retention_days_required")
	}
	if !inherit && retentionDays != nil {
		v := *retentionDays
		if v < 0 || v > maxRetentionDays {
			return fmt.Errorf("uploads_cleanup_retention_days")
		}
	}
	return nil
}
