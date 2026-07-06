package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func (a *App) canAccessUploadsCleanupSettings(userID, orgID uuid.UUID) bool {
	return a.HasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionRead, orgID) ||
		a.HasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionWrite, orgID) ||
		a.HasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionExecute, orgID)
}

func (a *App) canWriteUploadsCleanupSettings(userID, orgID uuid.UUID) bool {
	return a.HasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionWrite, orgID)
}

func (a *App) canExecuteUploadsCleanup(userID, orgID uuid.UUID) bool {
	return a.HasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionExecute, orgID)
}

// RunUploadsCleanupNow deletes expired transient uploads immediately for the current organization.
func (a *App) RunUploadsCleanupNow(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !a.canExecuteUploadsCleanup(userID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	worker := NewUploadsCleanupWorker(a, time.Minute)
	result, err := worker.RunManualCleanup(context.Background(), orgID, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, errUploadsCleanupDisabled):
			return r.SendErrorEnvelope(
				fasthttp.StatusBadRequest,
				"Uploads cleanup retention must be greater than 0 before running cleanup.",
				nil,
				"uploads_cleanup_retention_days",
			)
		default:
			a.Log.Error("Manual uploads cleanup failed", "error", err, "organization_id", orgID, "user_id", userID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to run uploads cleanup", nil, "")
		}
	}

	a.Log.Info(
		"Manual uploads cleanup completed",
		"organization_id", orgID,
		"user_id", userID,
		"deleted_files", result.DeletedFiles,
		"retention_days", result.RetentionDays,
	)

	scopedDB := tenant.ScopedDB(a.DB, orgID)
	var instances []models.WhatsAppInstance
	scopedDB.Find(&instances)

	instancesBreakdown := make([]map[string]any, 0, len(instances))
	instanceErrors := make([]map[string]any, 0)
	for _, inst := range instances {
		days, source := ResolveInstanceRetention(inst.Settings, result.RetentionDays)
		instDeleted := 0
		if days > 0 {
			d := days
			var instErr error
			instDeleted, instErr = RunManualCleanupForInstance(context.Background(), a, orgID, inst.ID, &d)
			if instErr != nil {
				a.Log.Error("Instance cleanup failed", "instance_id", inst.ID, "error", instErr)
				instanceErrors = append(instanceErrors, map[string]any{
					"instance_id":   inst.ID,
					"instance_name": inst.Name,
					"error":         instErr.Error(),
				})
			}
		}
		instancesBreakdown = append(instancesBreakdown, map[string]any{
			"instance_id":    inst.ID,
			"instance_name":  inst.Name,
			"deleted_files":  instDeleted,
			"retention_used": days,
			"source":         source,
		})
	}

	return r.SendEnvelope(map[string]any{
		"message":         fmt.Sprintf("Uploads cleanup completed. Deleted %d file(s) across %d instance(s).", result.DeletedFiles, len(instances)),
		"deleted_files":   result.DeletedFiles,
		"retention_days":  result.RetentionDays,
		"instances":       instancesBreakdown,
		"instance_errors": instanceErrors,
	})
}
