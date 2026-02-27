package handlers

import (
	"context"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

// normalizeInstanceName trims whitespace and keeps a consistent validation baseline.
func normalizeInstanceName(name string) string {
	return strings.TrimSpace(name)
}

// isInstanceNameTaken checks whether an organization already has an instance with the given name.
// Name matching is case-insensitive and ignores outer whitespace.
func (a *App) isInstanceNameTaken(ctx context.Context, orgID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := a.DB.WithContext(ctx).
		Model(&models.WhatsAppInstance{}).
		Where("organization_id = ? AND LOWER(TRIM(name)) = LOWER(TRIM(?))", orgID, name)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
