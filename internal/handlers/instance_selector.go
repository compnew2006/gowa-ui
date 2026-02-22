package handlers

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	"gorm.io/gorm"
)

func (a *App) resolveOutboundInstance(orgID uuid.UUID, requestedInstanceID string, contactInstanceID *uuid.UUID) (*models.WhatsAppInstance, error) {
	var instance models.WhatsAppInstance

	if requestedInstanceID != "" {
		id, err := uuid.Parse(requestedInstanceID)
		if err != nil {
			return nil, fmt.Errorf("invalid instance_id")
		}
		if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&instance).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("instance not found")
			}
			return nil, err
		}
		if instance.Status != models.InstanceStatusConnected {
			return nil, fmt.Errorf("instance is not connected")
		}
		return &instance, nil
	}

	if contactInstanceID != nil {
		if err := a.DB.Where("id = ? AND organization_id = ?", *contactInstanceID, orgID).First(&instance).Error; err == nil {
			if instance.Status == models.InstanceStatusConnected {
				return &instance, nil
			}
		}
	}

	if err := a.DB.
		Where("organization_id = ? AND is_default = ? AND status = ?", orgID, true, models.InstanceStatusConnected).
		First(&instance).Error; err == nil {
		return &instance, nil
	}

	if err := a.DB.
		Where("organization_id = ? AND status = ?", orgID, models.InstanceStatusConnected).
		Order("is_default DESC, created_at ASC").
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no connected instance available")
		}
		return nil, err
	}

	return &instance, nil
}
