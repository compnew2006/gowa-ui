package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type conversationNotePermissionFixture struct {
	org       *models.Organization
	contact   *models.Contact
	note      *models.ConversationNote
	owner     *models.User
	agent     *models.User
	moderator *models.User
}

func setupConversationNotePermissionFixture(t *testing.T, app *handlers.App) conversationNotePermissionFixture {
	t.Helper()

	org := testutil.CreateTestOrganization(t, app.DB)
	agentRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "notes-agent", []string{
		"chat:read",
		"chat:write",
	})
	moderatorRole := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "notes-moderator", []string{
		"chat:read",
		"chat:write",
		"chat:delete",
	})

	owner := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	agent := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&agentRole.ID))
	moderator := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&moderatorRole.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	note := &models.ConversationNote{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		CreatedByID:    owner.ID,
		Content:        "Execution Summary",
	}
	require.NoError(t, app.DB.Create(note).Error)

	return conversationNotePermissionFixture{
		org:       org,
		contact:   contact,
		note:      note,
		owner:     owner,
		agent:     agent,
		moderator: moderator,
	}
}

func TestApp_UpdateConversationNote_Permissions(t *testing.T) {
	t.Parallel()

	t.Run("non-owner without chat delete permission is forbidden", func(t *testing.T) {
		app := newTestApp(t)
		fixture := setupConversationNotePermissionFixture(t, app)

		req := testutil.NewJSONRequest(t, map[string]any{
			"content": "updated by agent",
		})
		req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
		testutil.SetAuthContext(req, fixture.org.ID, fixture.agent.ID)
		testutil.SetPathParam(req, "id", fixture.contact.ID.String())
		testutil.SetPathParam(req, "note_id", fixture.note.ID.String())

		err := app.UpdateConversationNote(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "You can only edit your own notes")
	})

	t.Run("non-owner with chat delete permission can update", func(t *testing.T) {
		app := newTestApp(t)
		fixture := setupConversationNotePermissionFixture(t, app)

		updatedContent := "updated by moderator"
		req := testutil.NewJSONRequest(t, map[string]any{
			"content": updatedContent,
		})
		req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPut)
		testutil.SetAuthContext(req, fixture.org.ID, fixture.moderator.ID)
		testutil.SetPathParam(req, "id", fixture.contact.ID.String())
		testutil.SetPathParam(req, "note_id", fixture.note.ID.String())

		err := app.UpdateConversationNote(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.ConversationNoteResponse `json:"data"`
		}
		err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
		require.NoError(t, err)
		assert.Equal(t, updatedContent, resp.Data.Content)

		var saved models.ConversationNote
		require.NoError(t, app.DB.First(&saved, "id = ?", fixture.note.ID).Error)
		assert.Equal(t, updatedContent, saved.Content)
	})
}

func TestApp_DeleteConversationNote_Permissions(t *testing.T) {
	t.Parallel()

	t.Run("non-owner without chat delete permission is forbidden", func(t *testing.T) {
		app := newTestApp(t)
		fixture := setupConversationNotePermissionFixture(t, app)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodDelete)
		testutil.SetAuthContext(req, fixture.org.ID, fixture.agent.ID)
		testutil.SetPathParam(req, "id", fixture.contact.ID.String())
		testutil.SetPathParam(req, "note_id", fixture.note.ID.String())

		err := app.DeleteConversationNote(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "You can only delete your own notes")
	})

	t.Run("non-owner with chat delete permission can delete", func(t *testing.T) {
		app := newTestApp(t)
		fixture := setupConversationNotePermissionFixture(t, app)

		req := testutil.NewRequest(t)
		req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodDelete)
		testutil.SetAuthContext(req, fixture.org.ID, fixture.moderator.ID)
		testutil.SetPathParam(req, "id", fixture.contact.ID.String())
		testutil.SetPathParam(req, "note_id", fixture.note.ID.String())

		err := app.DeleteConversationNote(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var deleted models.ConversationNote
		require.NoError(t, app.DB.Unscoped().First(&deleted, "id = ?", fixture.note.ID).Error)
		assert.True(t, deleted.DeletedAt.Valid)
	})
}
