package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func createCollaboratorManagerUser(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string) *models.User {
	t.Helper()

	return createUserWithPermissionKeys(t, app, orgID, roleName, []string{
		"chat:read",
		"contacts:read",
		"chat.collaborators:read",
		"chat.collaborators:write",
	})
}

func createPublicContact(t *testing.T, app *handlers.App, orgID uuid.UUID, opts ...testutil.ContactOption) *models.Contact {
	t.Helper()

	baseOpts := []testutil.ContactOption{
		func(c *models.Contact) {
			c.IsPublic = true
		},
	}
	baseOpts = append(baseOpts, opts...)
	return testutil.CreateTestContactWith(t, app.DB, orgID, baseOpts...)
}

func seedContactCollaborator(
	t *testing.T,
	app *handlers.App,
	orgID, contactID, userID, invitedByUserID uuid.UUID,
	status models.CollaboratorStatus,
) *models.ContactCollaborator {
	t.Helper()

	collaborator := &models.ContactCollaborator{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		UserID:          userID,
		Role:            models.CollaboratorRoleAssistant,
		Status:          status,
		InvitedByUserID: invitedByUserID,
	}

	switch status {
	case models.CollaboratorStatusAccepted:
		now := time.Now().UTC()
		collaborator.AcceptedAt = &now
	case models.CollaboratorStatusDeclined:
		now := time.Now().UTC()
		collaborator.DeclinedAt = &now
	}

	require.NoError(t, app.DB.Create(collaborator).Error)
	return collaborator
}

func TestApp_InviteContactCollaborator_RejectsInactiveUser(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-inactive")
	invitee := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithInactive())
	contact := createPublicContact(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": invitee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, manager.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.InviteContactCollaborator(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "inactive")

	var count int64
	require.NoError(t, app.DB.Model(&models.ContactCollaborator{}).
		Where("organization_id = ? AND contact_id = ? AND user_id = ?", org.ID, contact.ID, invitee.ID).
		Count(&count).Error)
	assert.Zero(t, count)
}

func TestApp_InviteContactCollaborator_RejectsAlreadyInvitedCollaborator(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-invited")
	invitee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := createPublicContact(t, app, org.ID)
	seedContactCollaborator(t, app, org.ID, contact.ID, invitee.ID, manager.ID, models.CollaboratorStatusInvited)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": invitee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, manager.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.InviteContactCollaborator(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already invited")
}

func TestApp_InviteContactCollaborator_RejectsAlreadyAcceptedCollaborator(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-accepted")
	invitee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := createPublicContact(t, app, org.ID)
	original := seedContactCollaborator(t, app, org.ID, contact.ID, invitee.ID, manager.ID, models.CollaboratorStatusAccepted)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": invitee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, manager.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.InviteContactCollaborator(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusConflict, "already accepted")

	var refreshed models.ContactCollaborator
	require.NoError(t, app.DB.Where("id = ?", original.ID).First(&refreshed).Error)
	assert.Equal(t, models.CollaboratorStatusAccepted, refreshed.Status)
	require.NotNil(t, refreshed.AcceptedAt)
}

func TestApp_InviteContactCollaborator_RejectsInviteeWithoutInstanceAccess(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-instance")
	invitee := testutil.CreateTestUser(t, app.DB, org.ID)
	allowed := createTestInstance(t, app, org.ID, "Allowed")
	blocked := createTestInstance(t, app, org.ID, "Blocked")
	enableRestrictedInstanceVisibility(t, app, org.ID, invitee.ID, allowed.ID)

	contact := createPublicContact(t, app, org.ID, func(c *models.Contact) {
		c.InstanceID = &blocked.ID
	})

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": invitee.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, manager.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.InviteContactCollaborator(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "does not have access")
}

func TestApp_InviteContactCollaborator_ReinvitesDeclinedCollaborator(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-declined")
	invitee := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := createPublicContact(t, app, org.ID)
	original := seedContactCollaborator(t, app, org.ID, contact.ID, invitee.ID, manager.ID, models.CollaboratorStatusDeclined)

	req := testutil.NewJSONRequest(t, map[string]any{
		"user_id": invitee.ID.String(),
		"role":    string(models.CollaboratorRoleViewer),
	})
	testutil.SetAuthContext(req, org.ID, manager.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.InviteContactCollaborator(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var refreshed models.ContactCollaborator
	require.NoError(t, app.DB.Where("id = ?", original.ID).First(&refreshed).Error)
	assert.Equal(t, models.CollaboratorStatusInvited, refreshed.Status)
	assert.Equal(t, models.CollaboratorRoleViewer, refreshed.Role)
	assert.Nil(t, refreshed.AcceptedAt)
	assert.Nil(t, refreshed.DeclinedAt)
}

func TestApp_RemoveContactCollaborator_AllowsSelfRemovalWithoutWritePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	manager := createCollaboratorManagerUser(t, app, org.ID, "collab-manager-remove")
	collaboratorUser := testutil.CreateTestUser(t, app.DB, org.ID)
	contact := createPublicContact(t, app, org.ID)
	original := seedContactCollaborator(t, app, org.ID, contact.ID, collaboratorUser.ID, manager.ID, models.CollaboratorStatusAccepted)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodDelete)
	testutil.SetAuthContext(req, org.ID, collaboratorUser.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "user_id", collaboratorUser.ID.String())

	err := app.RemoveContactCollaborator(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var count int64
	require.NoError(t, app.DB.Unscoped().Model(&models.ContactCollaborator{}).
		Where("id = ? AND deleted_at IS NULL", original.ID).
		Count(&count).Error)
	assert.Zero(t, count)
}
