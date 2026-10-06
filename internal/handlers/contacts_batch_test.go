package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type queryCountingLogger struct {
	logger.Interface
	queryCount *int
}

func (l queryCountingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "SELECT ") {
		*l.queryCount++
	}
	l.Interface.Trace(ctx, begin, func() (string, int64) { return sql, rows }, err)
}

func TestFilterCollaboratorsForViewerUsesPermissionSnapshot(t *testing.T) {
	viewerID := uuid.New()
	agentID := uuid.New()
	managerID := uuid.New()
	collaborators := []models.Collaborator{
		{UserID: viewerID.String(), Name: "Viewer"},
		{UserID: agentID.String(), Name: "Agent"},
		{UserID: managerID.String(), Name: "Manager"},
	}

	got := filterCollaboratorsForViewer(collaborators, viewerID, map[uuid.UUID]bool{
		managerID: true,
	})
	require.Len(t, got, 2)
	assert.Equal(t, viewerID.String(), got[0].UserID, "the viewer's collaborator entry must remain visible")
	assert.Equal(t, agentID.String(), got[1].UserID, "agent collaborators remain visible")

	adminView := filterCollaboratorsForViewer(collaborators, managerID, map[uuid.UUID]bool{
		managerID: true,
	})
	assert.Equal(t, collaborators, adminView, "managers see all collaborators")

	failedLookup := failClosedCollaboratorPermissions([]uuid.UUID{viewerID, agentID, managerID}, viewerID)
	assert.False(t, failedLookup[viewerID], "a failed lookup must not promote the viewer to manager")
	assert.True(t, failedLookup[managerID], "hide other collaborators if the lookup fails")
	assert.Equal(t, []models.Collaborator{collaborators[0]},
		filterCollaboratorsForViewer(collaborators, viewerID, failedLookup),
		"a failed permission lookup must preserve the viewer while hiding other identities")
}

func TestBuildContactResponsesMasked_UsesBoundedBatchQueries(t *testing.T) {
	db := testutil.SetupTestDB(t)
	org := testutil.CreateTestOrganization(t, db)
	viewer := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Batch Viewer"))
	manager := testutil.CreateTestUser(t, db, org.ID, testutil.WithFullName("Hidden Manager"))
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", manager.ID).Update("is_super_admin", true).Error)
	queryCount := 0
	appDB := db.Session(&gorm.Session{Logger: queryCountingLogger{Interface: db.Config.Logger, queryCount: &queryCount}})
	app := &App{DB: appDB, Log: testutil.NopLogger()}

	measure := func(contactCount int) int {
		queryCount = 0
		contacts := make([]models.Contact, contactCount)
		for i := range contacts {
			contacts[i] = models.Contact{
				BaseModel:      models.BaseModel{ID: uuid.New()},
				OrganizationID: org.ID,
				AssignedUserID: &viewer.ID,
				Metadata:       models.JSONB{},
			}
			contacts[i].AddCollaborator(models.Collaborator{UserID: viewer.ID.String(), Name: "Batch Viewer"})
			contacts[i].AddCollaborator(models.Collaborator{UserID: manager.ID.String(), Name: "Hidden Manager"})
		}
		responses := app.buildContactResponsesMasked(contacts, org.ID, viewer.ID, false)
		require.Len(t, responses, contactCount)
		for _, response := range responses {
			assert.Equal(t, "Batch Viewer", response.AssignedUserName)
			require.Len(t, response.Collaborators, 1, "agents must not see manager collaborators")
			assert.Equal(t, viewer.ID.String(), response.Collaborators[0].UserID, "the viewer must remain visible as a collaborator")
		}
		return queryCount
	}

	singleContactCounts := measure(1)
	manyContactCounts := measure(25)
	assert.Equal(t, singleContactCounts, manyContactCounts,
		"lookup query count must stay constant as the page size grows")
	assert.Equal(t, 5, manyContactCounts,
		"one grouped unread query, one last-account query, one assignee batch, one collaborator-permission batch, and one org-account-phones lookup are expected")
}
