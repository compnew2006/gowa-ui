package handlers

import (
	"errors"

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
