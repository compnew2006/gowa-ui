package handlers

import (
	"errors"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func normalizeUnclaimedChatAccess(allowView, allowSend bool) (bool, bool) {
	if allowSend && !allowView {
		allowView = true
	}
	return allowView, allowSend
}

func (a *App) resolveUnclaimedChatAccess(orgID, userID uuid.UUID) (bool, bool) {
	user, err := a.loadUserForSendRestrictions(orgID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false
		}
		a.Log.Warn("Failed to resolve unclaimed chat access settings", "error", err, "org_id", orgID, "user_id", userID)
		return false, false
	}

	cfg := readSendRestrictionsSettings(user.Settings)
	return normalizeUnclaimedChatAccess(cfg.AllowUnclaimedChatView, cfg.AllowUnclaimedChatSend)
}

func (a *App) canViewRestrictedChatWithoutClaim(userID, orgID uuid.UUID) bool {
	if a.canBypassPendingChatRestriction(userID, orgID) {
		return true
	}
	allowView, _ := a.resolveUnclaimedChatAccess(orgID, userID)
	return allowView
}

func (a *App) canSendRestrictedChatWithoutClaim(userID, orgID uuid.UUID) bool {
	if a.canBypassPendingChatRestriction(userID, orgID) {
		return true
	}
	_, allowSend := a.resolveUnclaimedChatAccess(orgID, userID)
	return allowSend
}

func isContactAssignedToUser(contact *models.Contact, userID uuid.UUID) bool {
	if contact == nil || contact.AssignedUserID == nil || userID == uuid.Nil {
		return false
	}
	return *contact.AssignedUserID == userID
}

func shouldAllowSelfAssignedRestrictedInstanceListBypass(
	statusFilter *models.ChatStatus,
	hasAssignedToFilter bool,
	assignedToUserID *uuid.UUID,
	currentUserID uuid.UUID,
) bool {
	if statusFilter == nil || *statusFilter != models.ChatStatusOpen {
		return false
	}
	if !hasAssignedToFilter || assignedToUserID == nil {
		return false
	}
	return *assignedToUserID == currentUserID
}

func applyRestrictedInstanceVisibilityFilter(
	query *gorm.DB,
	restrictedInstanceIDs []uuid.UUID,
) *gorm.DB {
	if query == nil || restrictedInstanceIDs == nil {
		return query
	}
	if len(restrictedInstanceIDs) > 0 {
		return query.Where("instance_id IN ?", restrictedInstanceIDs)
	}
	return query.Where("1 = 0")
}
