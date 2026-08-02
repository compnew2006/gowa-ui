package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"
)

func chatRWRole(t *testing.T, db *gorm.DB, orgID uuid.UUID) *models.CustomRole {
	t.Helper()
	return testutil.CreateTestRoleWithKeys(t, db, orgID, "chat-rw", []string{"chat:read", "chat:write"})
}

// --- ListConversationNotes ---

func TestApp_ListConversationNotes_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	for i := range 3 {
		require.NoError(t, app.DB.Create(&models.ConversationNote{
			BaseModel:      models.BaseModel{ID: uuid.New()},
			OrganizationID: org.ID,
			ContactID:      contact.ID,
			CreatedByID:    user.ID,
			Content:        "note content " + string(rune('a'+i)),
		}).Error)
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.ListConversationNotes(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Notes   []handlers.ConversationNoteResponse `json:"notes"`
			Total   int                                 `json:"total"`
			HasMore bool                                `json:"has_more"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 3, resp.Data.Total)
	assert.Len(t, resp.Data.Notes, 3)
	assert.False(t, resp.Data.HasMore)
}

func TestApp_ListConversationNotes_CrossOrgIsolation(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	roleB := chatRWRole(t, app.DB, orgB.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID)
	userB := testutil.CreateTestUser(t, app.DB, orgB.ID, testutil.WithRoleID(&roleB.ID))
	contactA := testutil.CreateTestContact(t, app.DB, orgA.ID)

	require.NoError(t, app.DB.Create(&models.ConversationNote{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgA.ID,
		ContactID:      contactA.ID,
		CreatedByID:    userA.ID,
		Content:        "secret",
	}).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgB.ID, userB.ID)
	testutil.SetPathParam(req, "id", contactA.ID.String())

	require.NoError(t, app.ListConversationNotes(req))
	var resp struct {
		Data struct {
			Notes []handlers.ConversationNoteResponse `json:"notes"`
			Total int                                 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, 0, resp.Data.Total)
	assert.Empty(t, resp.Data.Notes)
}

func TestApp_ListConversationNotes_PermissionDenied(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "no-chat", false, false, nil)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.ListConversationNotes(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// --- CreateConversationNote ---

func TestApp_CreateConversationNote_Success(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"content": "follow up tomorrow"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateConversationNote(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ConversationNoteResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, "follow up tomorrow", resp.Data.Content)
	assert.Equal(t, contact.ID, resp.Data.ContactID)
	assert.Equal(t, user.ID, resp.Data.CreatedByID)

	var got models.ConversationNote
	require.NoError(t, app.DB.Where("id = ?", resp.Data.ID).First(&got).Error)
	assert.Equal(t, "follow up tomorrow", got.Content)
	assert.Equal(t, org.ID, got.OrganizationID)
}

func TestApp_CreateConversationNote_EmptyContentRejected(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"content": ""})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateConversationNote(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "content is required")
}

func TestApp_CreateConversationNote_PermissionDenied(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	// chat:read only — no write.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "chat-r-only", []string{"chat:read"})
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"content": "x"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	require.NoError(t, app.CreateConversationNote(req))
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

// TestApp_CreateConversationNote_CrossOrgIsolation is the M1 regression: a
// caller authed in org A must NOT be able to attach a note to a contact owned
// by org B by guessing its {id}. Before the fix the path UUID was parsed
// without an org scope, producing a mis-scoped (org-A-owned) row on org B's
// contact. Now loadContactByPath scopes by org and returns 404, and no row is
// created for org B's contact.
func TestApp_CreateConversationNote_CrossOrgIsolation(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	roleA := chatRWRole(t, app.DB, orgA.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID, testutil.WithRoleID(&roleA.ID))
	// Contact owned by org B — the cross-org target.
	contactB := testutil.CreateTestContact(t, app.DB, orgB.ID)

	// Caller in org A targets org B's contact.
	req := testutil.NewJSONRequest(t, map[string]any{"content": "leaked note"})
	testutil.SetAuthContext(req, orgA.ID, userA.ID)
	testutil.SetPathParam(req, "id", contactB.ID.String())

	require.NoError(t, app.CreateConversationNote(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "Contact not found")

	// No ConversationNote row may exist for org B's contact (the pre-fix bug).
	var noteCount int64
	app.DB.Model(&models.ConversationNote{}).Where("contact_id = ?", contactB.ID).Count(&noteCount)
	assert.Equal(t, int64(0), noteCount, "no note row must be created for the foreign-org contact")
}

// TestApp_CreateConversationNote_SameOrgRegression confirms the M1 fix did not
// break the legitimate same-org create path: contact in org A, caller in org A
// → 200, note created with the caller's org.
func TestApp_CreateConversationNote_SameOrgRegression(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	roleA := chatRWRole(t, app.DB, orgA.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID, testutil.WithRoleID(&roleA.ID))
	contactA := testutil.CreateTestContact(t, app.DB, orgA.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"content": "legit same-org note"})
	testutil.SetAuthContext(req, orgA.ID, userA.ID)
	testutil.SetPathParam(req, "id", contactA.ID.String())

	require.NoError(t, app.CreateConversationNote(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.ConversationNoteResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, contactA.ID, resp.Data.ContactID)
	assert.Equal(t, userA.ID, resp.Data.CreatedByID)

	// The persisted row must carry the caller's org (orgA), not be mis-scoped.
	var got models.ConversationNote
	require.NoError(t, app.DB.Where("id = ?", resp.Data.ID).First(&got).Error)
	assert.Equal(t, orgA.ID, got.OrganizationID, "note must be scoped to the caller's org")
}

// --- UpdateConversationNote ---

func TestApp_UpdateConversationNote_OnlyCreatorCanEdit(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	creator := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	other := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	note := &models.ConversationNote{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		CreatedByID:    creator.ID,
		Content:        "original",
	}
	require.NoError(t, app.DB.Create(note).Error)

	// Other user attempts edit → 403.
	req := testutil.NewJSONRequest(t, map[string]any{"content": "hacked"})
	testutil.SetAuthContext(req, org.ID, other.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "note_id", note.ID.String())

	require.NoError(t, app.UpdateConversationNote(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "your own notes")

	// Note unchanged.
	var got models.ConversationNote
	require.NoError(t, app.DB.Where("id = ?", note.ID).First(&got).Error)
	assert.Equal(t, "original", got.Content)

	// Creator can edit.
	req2 := testutil.NewJSONRequest(t, map[string]any{"content": "fixed"})
	testutil.SetAuthContext(req2, org.ID, creator.ID)
	testutil.SetPathParam(req2, "id", contact.ID.String())
	testutil.SetPathParam(req2, "note_id", note.ID.String())

	require.NoError(t, app.UpdateConversationNote(req2))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req2))

	require.NoError(t, app.DB.Where("id = ?", note.ID).First(&got).Error)
	assert.Equal(t, "fixed", got.Content)
}

func TestApp_UpdateConversationNote_NotFound(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{"content": "x"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "note_id", uuid.New().String())

	require.NoError(t, app.UpdateConversationNote(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestApp_UpdateConversationNote_EmptyContentRejected(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	note := &models.ConversationNote{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		CreatedByID:    user.ID,
		Content:        "before",
	}
	require.NoError(t, app.DB.Create(note).Error)

	req := testutil.NewJSONRequest(t, map[string]any{"content": ""})
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "note_id", note.ID.String())

	require.NoError(t, app.UpdateConversationNote(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "content is required")
}

// --- DeleteConversationNote ---

func TestApp_DeleteConversationNote_OnlyCreatorCanDelete(t *testing.T) {
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := chatRWRole(t, app.DB, org.ID)
	creator := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	other := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithRoleID(&role.ID))
	contact := testutil.CreateTestContact(t, app.DB, org.ID)
	note := &models.ConversationNote{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		ContactID:      contact.ID,
		CreatedByID:    creator.ID,
		Content:        "x",
	}
	require.NoError(t, app.DB.Create(note).Error)

	// Other user → 403.
	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, org.ID, other.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetPathParam(req, "note_id", note.ID.String())

	require.NoError(t, app.DeleteConversationNote(req))
	testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "your own notes")

	var stillExists int64
	app.DB.Model(&models.ConversationNote{}).Where("id = ?", note.ID).Count(&stillExists)
	assert.Equal(t, int64(1), stillExists)

	// Creator → success.
	req2 := testutil.NewRequest(t)
	testutil.SetAuthContext(req2, org.ID, creator.ID)
	testutil.SetPathParam(req2, "id", contact.ID.String())
	testutil.SetPathParam(req2, "note_id", note.ID.String())

	require.NoError(t, app.DeleteConversationNote(req2))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req2))

	app.DB.Model(&models.ConversationNote{}).Where("id = ?", note.ID).Count(&stillExists)
	assert.Equal(t, int64(0), stillExists)
}

func TestApp_DeleteConversationNote_CrossOrgIsolation(t *testing.T) {
	app := newTestApp(t)
	orgA := testutil.CreateTestOrganization(t, app.DB)
	orgB := testutil.CreateTestOrganization(t, app.DB)
	roleB := chatRWRole(t, app.DB, orgB.ID)
	userA := testutil.CreateTestUser(t, app.DB, orgA.ID)
	userB := testutil.CreateTestUser(t, app.DB, orgB.ID, testutil.WithRoleID(&roleB.ID))
	contactA := testutil.CreateTestContact(t, app.DB, orgA.ID)
	note := &models.ConversationNote{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgA.ID,
		ContactID:      contactA.ID,
		CreatedByID:    userA.ID,
		Content:        "x",
	}
	require.NoError(t, app.DB.Create(note).Error)

	req := testutil.NewRequest(t)
	testutil.SetAuthContext(req, orgB.ID, userB.ID)
	testutil.SetPathParam(req, "id", contactA.ID.String())
	testutil.SetPathParam(req, "note_id", note.ID.String())

	require.NoError(t, app.DeleteConversationNote(req))
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))

	var stillExists int64
	app.DB.Model(&models.ConversationNote{}).Where("id = ?", note.ID).Count(&stillExists)
	assert.Equal(t, int64(1), stillExists)
}
