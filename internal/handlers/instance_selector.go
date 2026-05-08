package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type instanceSelectionError struct {
	*reasonedError
}

func newInstanceSelectionError(message, reasonCode string) *instanceSelectionError {
	return &instanceSelectionError{reasonedError: newReasonedError(message, reasonCode, "instance selection failed")}
}

func asInstanceSelectionError(err error) (string, string, bool) {
	re, ok := asReasonedError(err)
	if !ok {
		return "", "", false
	}
	return re.Error(), strings.TrimSpace(re.reasonCode), true
}

func instanceSendBlockReason(instance *models.WhatsAppInstance) string {
	if instance == nil || instance.SendBlockedUntil == nil {
		return ""
	}
	if time.Now().UTC().After(instance.SendBlockedUntil.UTC()) {
		return ""
	}
	reason := strings.TrimSpace(instance.SendBlockReason)
	if reason != "" {
		return reason
	}
	return "instance sending is temporarily blocked"
}

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
			return nil, newInstanceSelectionError("instance is not connected", ReasonCodeInstanceNotConn)
		}
		if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
			return nil, newInstanceSelectionError(blockReason, ReasonCodeInstanceBlocked)
		}
		return &instance, nil
	}

	if contactInstanceID != nil {
		if err := a.DB.Where("id = ? AND organization_id = ?", *contactInstanceID, orgID).First(&instance).Error; err == nil {
			if instance.Status == models.InstanceStatusConnected && instanceSendBlockReason(&instance) == "" {
				return &instance, nil
			}
		}
	}

	if err := a.DB.
		Where("organization_id = ? AND is_default = ? AND status = ?", orgID, true, models.InstanceStatusConnected).
		First(&instance).Error; err == nil {
		if instanceSendBlockReason(&instance) == "" {
			return &instance, nil
		}
	}

	var connectedInstances []models.WhatsAppInstance
	if err := a.DB.
		Where("organization_id = ? AND status = ?", orgID, models.InstanceStatusConnected).
		Order("is_default DESC, created_at ASC").
		Find(&connectedInstances).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newInstanceSelectionError("no connected instance available", ReasonCodeInstanceNotConn)
		}
		return nil, err
	}

	if len(connectedInstances) == 0 {
		return nil, newInstanceSelectionError("no connected instance available", ReasonCodeInstanceNotConn)
	}

	var blockedReason string
	for _, candidate := range connectedInstances {
		if reason := instanceSendBlockReason(&candidate); reason != "" {
			if blockedReason == "" {
				blockedReason = reason
			}
			continue
		}
		instance = candidate
		return &instance, nil
	}

	if blockedReason != "" {
		return nil, newInstanceSelectionError(blockedReason, ReasonCodeInstanceBlocked)
	}

	return nil, newInstanceSelectionError("no connected instance available", ReasonCodeInstanceNotConn)
}
