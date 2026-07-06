package handlers

import (
	"context"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (a *App) upsertContactUserDeletion(ctx context.Context, orgID, contactID, userID uuid.UUID, deletedAt time.Time) error {
	entry := models.ContactUserDeletion{
		OrganizationID: orgID,
		ContactID:      contactID,
		UserID:         userID,
		DeletedAt:      deletedAt,
	}

	return a.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "organization_id"},
			{Name: "contact_id"},
			{Name: "user_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"deleted_at", "updated_at"}),
	}).Create(&entry).Error
}

func (a *App) getContactUserDeletionTimestamp(ctx context.Context, orgID, contactID, userID uuid.UUID) (*time.Time, error) {
	var deletion models.ContactUserDeletion
	if err := a.DB.WithContext(ctx).
		Where("organization_id = ? AND contact_id = ? AND user_id = ?", orgID, contactID, userID).
		First(&deletion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	deletedAt := deletion.DeletedAt
	return &deletedAt, nil
}

func (a *App) getContactUserDeletionMap(ctx context.Context, orgID, userID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	if len(contactIDs) == 0 {
		return map[uuid.UUID]time.Time{}, nil
	}
	var deletions []models.ContactUserDeletion
	if err := a.DB.WithContext(ctx).
		Where("organization_id = ? AND user_id = ? AND contact_id IN ?", orgID, userID, contactIDs).
		Find(&deletions).Error; err != nil {
		return nil, err
	}
	deletionMap := make(map[uuid.UUID]time.Time, len(deletions))
	for _, deletion := range deletions {
		deletionMap[deletion.ContactID] = deletion.DeletedAt
	}
	return deletionMap, nil
}
