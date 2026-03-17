package handlers

import (
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func collaboratorAccessStatuses() []models.CollaboratorStatus {
	return []models.CollaboratorStatus{
		models.CollaboratorStatusInvited,
		models.CollaboratorStatusAccepted,
	}
}

func (a *App) isContactCollaborator(orgID, contactID, userID uuid.UUID) bool {
	if a == nil || a.DB == nil || orgID == uuid.Nil || contactID == uuid.Nil || userID == uuid.Nil {
		return false
	}
	var count int64
	a.DB.Model(&models.ContactCollaborator{}).
		Where("organization_id = ? AND contact_id = ? AND user_id = ? AND status IN ? AND deleted_at IS NULL",
			orgID, contactID, userID, collaboratorAccessStatuses()).
		Count(&count)
	return count > 0
}

func (a *App) listCollaboratorContactIDs(orgID, userID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	result := make(map[uuid.UUID]struct{})
	if a == nil || a.DB == nil || orgID == uuid.Nil || userID == uuid.Nil {
		return result, nil
	}
	var ids []uuid.UUID
	if err := a.DB.Model(&models.ContactCollaborator{}).
		Where("organization_id = ? AND user_id = ? AND status IN ? AND deleted_at IS NULL",
			orgID, userID, collaboratorAccessStatuses()).
		Pluck("contact_id", &ids).Error; err != nil {
		return result, err
	}
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result, nil
}
