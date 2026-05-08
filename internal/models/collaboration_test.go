package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollaborationTableName(t *testing.T) {
	assert.Equal(t, "contact_collaborators", ContactCollaborator{}.TableName())
}

func TestCollaboratorRoleConstants(t *testing.T) {
	assert.Equal(t, CollaboratorRole("viewer"), CollaboratorRoleViewer)
	assert.Equal(t, CollaboratorRole("assistant"), CollaboratorRoleAssistant)
}

func TestCollaboratorStatusConstants(t *testing.T) {
	assert.Equal(t, CollaboratorStatus("invited"), CollaboratorStatusInvited)
	assert.Equal(t, CollaboratorStatus("accepted"), CollaboratorStatusAccepted)
	assert.Equal(t, CollaboratorStatus("declined"), CollaboratorStatusDeclined)
}

func TestContactCollaboratorDefaults(t *testing.T) {
	c := ContactCollaborator{}
	assert.Equal(t, CollaboratorRole(""), c.Role)
	assert.Equal(t, CollaboratorStatus(""), c.Status)
	assert.Nil(t, c.AcceptedAt)
	assert.Nil(t, c.DeclinedAt)
}
