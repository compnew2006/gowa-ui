package handlers_test

import (
	"encoding/json"
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

func ptrTimeBulk(t time.Time) *time.Time { return &t }

func ptrUUIDBulk(id uuid.UUID) *uuid.UUID { return &id }

func TestBulkContactIDsRequest_Validate_Empty(t *testing.T) {
	t.Parallel()
	req := handlers.BulkContactIDsRequest{}
	assert.Error(t, req.Validate())
}

func TestBulkContactIDsRequest_Validate_ExceedsMax(t *testing.T) {
	t.Parallel()
	ids := make([]uuid.UUID, 101)
	for i := range ids {
		ids[i] = uuid.New()
	}
	req := handlers.BulkContactIDsRequest{ContactIDs: ids}
	assert.Error(t, req.Validate())
}

func TestBulkContactIDsRequest_Validate_NilUUID(t *testing.T) {
	t.Parallel()
	req := handlers.BulkContactIDsRequest{ContactIDs: []uuid.UUID{uuid.Nil}}
	assert.Error(t, req.Validate())
}

func TestBulkContactIDsRequest_Validate_Duplicate(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	req := handlers.BulkContactIDsRequest{ContactIDs: []uuid.UUID{id, id}}
	assert.Error(t, req.Validate())
}

func TestBulkContactIDsRequest_Validate_Valid(t *testing.T) {
	t.Parallel()
	req := handlers.BulkContactIDsRequest{ContactIDs: []uuid.UUID{uuid.New(), uuid.New()}}
	assert.NoError(t, req.Validate())
}

func TestBulkAssignRequest_Validate_NilAssigneeUUID(t *testing.T) {
	t.Parallel()
	req := handlers.BulkAssignRequest{
		ContactIDs: []uuid.UUID{uuid.New()},
		UserID:     ptrUUIDBulk(uuid.Nil),
	}
	assert.Error(t, req.Validate())
}

func TestApp_BulkCloseChats_Unauthorized(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	req := testutil.NewJSONRequest(t, nil)
	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkCloseChats_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkCloseChats_InvalidBody(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkCloseChats_Success(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})
	c2 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID, c2.ID},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 2, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Success)
	assert.Equal(t, 0, resp.Data.Failed)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", c1.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusClosed, refreshed.Status)
}

func TestApp_BulkCloseChats_AlreadyClosed(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTimeBulk(time.Now().UTC())
		c.ClosedByUserID = &user.ID
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Success)
}

func TestApp_BulkCloseChats_ContactNotFound(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{uuid.New()},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Total)
	assert.Equal(t, 0, resp.Data.Success)
	assert.Equal(t, 1, resp.Data.Failed)
	assert.Equal(t, "Contact not found", resp.Data.Results[0].Error)
}

func TestApp_BulkCloseChats_MixedResults(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID, uuid.New()},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 2, resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Success)
	assert.Equal(t, 1, resp.Data.Failed)
}

func TestApp_BulkAssignChats_Unauthorized(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	req := testutil.NewJSONRequest(t, nil)
	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkAssignChats_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkAssignChats_Success(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createContactMgmtAdmin(t, app, org.ID)
	agent := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContact(t, app.DB, org.ID)
	c2 := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, handlers.BulkAssignRequest{
		ContactIDs: []uuid.UUID{c1.ID, c2.ID},
		UserID:     &agent.ID,
	})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 2, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Success)
	assert.Equal(t, 0, resp.Data.Failed)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", c1.ID).First(&refreshed).Error)
	require.NotNil(t, refreshed.AssignedUserID)
	assert.Equal(t, agent.ID, *refreshed.AssignedUserID)
	assert.Equal(t, models.ChatStatusOpen, refreshed.Status)
}

func TestApp_BulkAssignChats_Unassign(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createContactMgmtAdmin(t, app, org.ID)
	agent := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &agent.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkAssignRequest{
		ContactIDs: []uuid.UUID{c1.ID},
		UserID:     nil,
	})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Success)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", c1.ID).First(&refreshed).Error)
	assert.Nil(t, refreshed.AssignedUserID)
	assert.Equal(t, models.ChatStatusPending, refreshed.Status)
}

func TestApp_BulkAssignChats_NonExistentAssignee(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createContactMgmtAdmin(t, app, org.ID)

	fakeUserID := uuid.New()
	c1 := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, handlers.BulkAssignRequest{
		ContactIDs: []uuid.UUID{c1.ID},
		UserID:     &fakeUserID,
	})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkReopenChats_Unauthorized(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	req := testutil.NewJSONRequest(t, nil)
	err := app.BulkReopenChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkReopenChats_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createReadonlyUser(t, app, org.ID)

	req := testutil.NewJSONRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkReopenChats_Success(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTimeBulk(time.Now().UTC().Add(-1 * time.Hour))
		c.ClosedByUserID = &user.ID
	})
	c2 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTimeBulk(time.Now().UTC().Add(-2 * time.Hour))
		c.ClosedByUserID = &user.ID
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID, c2.ID},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 2, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Success)
	assert.Equal(t, 0, resp.Data.Failed)

	var refreshed models.Contact
	require.NoError(t, app.DB.Where("id = ?", c1.ID).First(&refreshed).Error)
	assert.Equal(t, models.ChatStatusPending, refreshed.Status)
	assert.Nil(t, refreshed.ClosedAt)
	assert.Nil(t, refreshed.ClosedByUserID)
}

func TestApp_BulkReopenChats_OnlyClosedChatsCanBeReopened(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Total)
	assert.Equal(t, 0, resp.Data.Success)
	assert.Equal(t, 1, resp.Data.Failed)
	assert.Contains(t, resp.Data.Results[0].Error, "Only closed chats")
}

func TestApp_BulkReopenChats_ContactNotFound(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{uuid.New()},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Failed)
	assert.Equal(t, "Contact not found", resp.Data.Results[0].Error)
}

func TestApp_BulkCloseChats_OnlyAssignedUserCanClose(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent1 := createContactMgmtAdmin(t, app, org.ID)
	agent2 := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &agent1.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{c1.ID},
	})
	testutil.SetAuthContext(req, org.ID, agent2.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 1, resp.Data.Failed)
	assert.Contains(t, resp.Data.Results[0].Error, "assigned user")
}

func TestApp_BulkAssignChats_InvalidBody(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkReopenChats_InvalidBody(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestApp_BulkCloseChats_LargeBatch(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	ids := make([]uuid.UUID, 101)
	for i := range ids {
		ids[i] = uuid.New()
	}

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{ContactIDs: ids})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkCloseChats(req)
	require.NoError(t, err)

	var envelope testutil.APIEnvelope
	raw := testutil.GetResponseBody(req)
	require.NoError(t, json.Unmarshal(raw, &envelope))
	assert.Contains(t, *envelope.Message, "maximum")
}

func TestApp_BulkAssignChats_MixedResults(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createContactMgmtAdmin(t, app, org.ID)
	agent := createContactMgmtAdmin(t, app, org.ID)

	c1 := testutil.CreateTestContact(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, handlers.BulkAssignRequest{
		ContactIDs: []uuid.UUID{c1.ID, uuid.New()},
		UserID:     &agent.ID,
	})
	testutil.SetAuthContext(req, org.ID, admin.ID)

	err := app.BulkAssignChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 2, resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Success)
	assert.Equal(t, 1, resp.Data.Failed)
}

func TestApp_BulkReopenChats_MixedResults(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createContactMgmtAdmin(t, app, org.ID)

	closedContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.Status = models.ChatStatusClosed
		c.ClosedAt = ptrTimeBulk(time.Now().UTC().Add(-1 * time.Hour))
		c.ClosedByUserID = &user.ID
	})
	openContact := testutil.CreateTestContactWith(t, app.DB, org.ID, func(c *models.Contact) {
		c.AssignedUserID = &user.ID
		c.Status = models.ChatStatusOpen
	})

	req := testutil.NewJSONRequest(t, handlers.BulkContactIDsRequest{
		ContactIDs: []uuid.UUID{closedContact.ID, openContact.ID, uuid.New()},
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.BulkReopenChats(req)
	require.NoError(t, err)

	var resp struct {
		Data handlers.BulkResponse `json:"data"`
	}
	testutil.ParseJSONResponse(t, req, &resp)
	assert.Equal(t, 3, resp.Data.Total)
	assert.Equal(t, 1, resp.Data.Success)
	assert.Equal(t, 2, resp.Data.Failed)
}
