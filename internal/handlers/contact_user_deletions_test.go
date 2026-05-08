package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContactUserDeletionsGetMapEmptyIDs(t *testing.T) {
	a := &App{DB: nil}
	orgID := uuid.New()
	userID := uuid.New()
	result, err := a.getContactUserDeletionMap(nil, orgID, userID, nil)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestContactUserDeletionsModelConstruction(t *testing.T) {
	orgID := uuid.New()
	contactID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	entry := models.ContactUserDeletion{
		OrganizationID: orgID,
		ContactID:      contactID,
		UserID:         userID,
		DeletedAt:      now,
	}
	assert.Equal(t, orgID, entry.OrganizationID)
	assert.Equal(t, contactID, entry.ContactID)
	assert.Equal(t, userID, entry.UserID)
	assert.False(t, entry.DeletedAt.IsZero())
}
