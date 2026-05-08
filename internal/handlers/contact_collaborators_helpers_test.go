package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContactCollaboratorsAccessStatuses(t *testing.T) {
	statuses := collaboratorAccessStatuses()
	assert.Len(t, statuses, 2)
	assert.Contains(t, statuses, models.CollaboratorStatusInvited)
	assert.Contains(t, statuses, models.CollaboratorStatusAccepted)
}

func TestContactCollaboratorsNilApp(t *testing.T) {
	var a *App
	orgID := uuid.New()
	contactID := uuid.New()
	userID := uuid.New()
	result := a.isContactCollaborator(orgID, contactID, userID)
	assert.False(t, result)
}

func TestContactCollaboratorsNilDB(t *testing.T) {
	a := &App{}
	orgID := uuid.New()
	contactID := uuid.New()
	userID := uuid.New()
	result := a.isContactCollaborator(orgID, contactID, userID)
	assert.False(t, result)
}

func TestContactCollaboratorsListNilApp(t *testing.T) {
	var a *App
	orgID := uuid.New()
	userID := uuid.New()
	result, err := a.listCollaboratorContactIDs(orgID, userID)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestContactCollaboratorsListNilDB(t *testing.T) {
	a := &App{}
	orgID := uuid.New()
	userID := uuid.New()
	result, err := a.listCollaboratorContactIDs(orgID, userID)
	assert.NoError(t, err)
	assert.Empty(t, result)
}
