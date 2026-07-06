package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *App) parseAnalyticsInstanceID(orgID uuid.UUID, raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	instanceID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, errors.New("instance_id must be a valid UUID")
	}

	if a == nil || a.DB == nil {
		return nil, errors.New("instance lookup is unavailable")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("instance_id does not belong to this organization")
		}
		return nil, err
	}

	return &instanceID, nil
}

func applyTransferAnalyticsInstanceFilter(query *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID) *gorm.DB {
	if query == nil || instanceID == nil {
		return query
	}

	return query.Where(
		"contact_id IN (SELECT id FROM contacts WHERE organization_id = ? AND instance_id = ?)",
		orgID,
		*instanceID,
	)
}

func applyRatingAnalyticsInstanceFilter(query *gorm.DB, instanceID *uuid.UUID, contactAlias string) *gorm.DB {
	if query == nil || instanceID == nil {
		return query
	}
	alias := strings.TrimSpace(contactAlias)
	if alias == "" {
		alias = "contacts"
	}
	return query.Where(alias+".instance_id = ?", *instanceID)
}
