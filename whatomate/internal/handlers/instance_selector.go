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
	message    string
	reasonCode string
}

func (e *instanceSelectionError) Error() string {
	if e == nil {
		return "instance selection failed"
	}
	if strings.TrimSpace(e.message) == "" {
		return "instance selection failed"
	}
	return e.message
}

func asInstanceSelectionError(err error) (string, string, bool) {
	if err == nil {
		return "", "", false
	}
	var selectionErr *instanceSelectionError
	if !errors.As(err, &selectionErr) {
		return "", "", false
	}
	return selectionErr.Error(), strings.TrimSpace(selectionErr.reasonCode), true
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
			return nil, &instanceSelectionError{
				message:    "instance is not connected",
				reasonCode: ReasonCodeInstanceNotConn,
			}
		}
		if blockReason := instanceSendBlockReason(&instance); blockReason != "" {
			return nil, &instanceSelectionError{
				message:    blockReason,
				reasonCode: ReasonCodeInstanceBlocked,
			}
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
			return nil, &instanceSelectionError{
				message:    "no connected instance available",
				reasonCode: ReasonCodeInstanceNotConn,
			}
		}
		return nil, err
	}

	if len(connectedInstances) == 0 {
		return nil, &instanceSelectionError{
			message:    "no connected instance available",
			reasonCode: ReasonCodeInstanceNotConn,
		}
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
		return nil, &instanceSelectionError{
			message:    blockedReason,
			reasonCode: ReasonCodeInstanceBlocked,
		}
	}

	return nil, &instanceSelectionError{
		message:    "no connected instance available",
		reasonCode: ReasonCodeInstanceNotConn,
	}
}
