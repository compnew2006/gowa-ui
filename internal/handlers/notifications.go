package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// ListNotifications returns instance notifications for the current organization.
func (a *App) ListNotifications(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	includeDismissed := string(r.RequestCtx.QueryArgs().Peek("include_dismissed")) == "true"
	isAdmin := false
	if perms, permErr := a.getUserPermissionsCached(userID, orgID); permErr == nil {
		isAdmin = perms.IsSuperAdmin || strings.EqualFold(strings.TrimSpace(perms.RoleName), "admin")
	}

	query := a.DB.
		Where("organization_id = ?", orgID).
		Preload("Instance").
		Order("created_at DESC")
	if !includeDismissed {
		query = query.Where("is_dismissed = ?", false)
	}
	if !isAdmin {
		query = query.Where("event_type != ?", "chat_deleted_by_user")
	}

	var notifications []models.InstanceNotification
	if err := query.Find(&notifications).Error; err != nil {
		a.Log.Error("Failed to list notifications", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list notifications", nil, "")
	}

	return r.SendEnvelope(notifications)
}

// DismissNotification marks a notification as dismissed.
func (a *App) DismissNotification(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid notification ID", nil, "")
	}

	var notification models.InstanceNotification
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&notification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Notification not found", nil, "")
		}
		a.Log.Error("Failed to fetch notification", "error", err, "notification_id", id)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to dismiss notification", nil, "")
	}
	if notification.EventType == "chat_deleted_by_user" {
		isAdmin := false
		if perms, permErr := a.getUserPermissionsCached(userID, orgID); permErr == nil {
			isAdmin = perms.IsSuperAdmin || strings.EqualFold(strings.TrimSpace(perms.RoleName), "admin")
		}
		if !isAdmin {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Notification not available", nil, "")
		}
	}

	if !notification.IsDismissed {
		if err := a.DB.Model(&notification).Update("is_dismissed", true).Error; err != nil {
			a.Log.Error("Failed to dismiss notification", "error", err, "notification_id", id)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to dismiss notification", nil, "")
		}
		notification.IsDismissed = true
	}

	return r.SendEnvelope(notification)
}
