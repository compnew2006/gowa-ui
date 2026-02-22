package handlers

import (
	"errors"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// ListNotifications returns instance notifications for the current organization.
func (a *App) ListNotifications(r *fastglue.Request) error {
	orgID, err := a.getOrgID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	includeDismissed := string(r.RequestCtx.QueryArgs().Peek("include_dismissed")) == "true"

	query := a.DB.
		Where("organization_id = ?", orgID).
		Preload("Instance").
		Order("created_at DESC")
	if !includeDismissed {
		query = query.Where("is_dismissed = ?", false)
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
	orgID, err := a.getOrgID(r)
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

	if !notification.IsDismissed {
		if err := a.DB.Model(&notification).Update("is_dismissed", true).Error; err != nil {
			a.Log.Error("Failed to dismiss notification", "error", err, "notification_id", id)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to dismiss notification", nil, "")
		}
		notification.IsDismissed = true
	}

	return r.SendEnvelope(notification)
}
