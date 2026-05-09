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

func createCursorPaginationTestMessage(
	t *testing.T,
	app *handlers.App,
	orgID uuid.UUID,
	contactID uuid.UUID,
	account string,
	createdAt time.Time,
	content string,
) *models.Message {
	t.Helper()
	msg := &models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: createdAt, UpdatedAt: createdAt},
		OrganizationID:  orgID,
		WhatsAppAccount: account,
		ContactID:       contactID,
		Direction:       models.DirectionIncoming,
		MessageType:     models.MessageTypeText,
		Content:         content,
		Status:          models.MessageStatusReceived,
	}
	require.NoError(t, app.DB.Create(msg).Error)
	return msg
}

func TestApp_GetMessages_DefaultLoadsNewest(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	base := time.Now().UTC().Add(-1 * time.Hour)
	msgs := make([]*models.Message, 5)
	for i := 0; i < 5; i++ {
		msgs[i] = createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name,
			base.Add(time.Duration(i)*time.Minute), "msg-"+string(rune('a'+i)))
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Messages   []handlers.MessageResponse `json:"messages"`
		HasMore    bool                       `json:"has_more"`
		NextCursor string                     `json:"next_cursor"`
		PrevCursor string                     `json:"prev_cursor"`
		Total      int                        `json:"total"`
		Limit      int                        `json:"limit"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Equal(t, 5, len(resp.Messages))
	assert.Equal(t, msgs[0].ID, resp.Messages[0].ID)
	assert.Equal(t, msgs[4].ID, resp.Messages[4].ID)
	assert.Equal(t, msgs[4].ID.String(), resp.NextCursor)
	assert.Equal(t, msgs[0].ID.String(), resp.PrevCursor)
}

func TestApp_GetMessages_BeforeIDCursor(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	base := time.Now().UTC().Add(-1 * time.Hour)
	msgs := make([]*models.Message, 5)
	for i := 0; i < 5; i++ {
		msgs[i] = createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name,
			base.Add(time.Duration(i)*time.Minute), "msg-"+string(rune('a'+i)))
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "before_id", msgs[3].ID.String())
	testutil.SetQueryParam(req, "limit", "2")

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Messages   []handlers.MessageResponse `json:"messages"`
		HasMore    bool                       `json:"has_more"`
		NextCursor string                     `json:"next_cursor"`
		PrevCursor string                     `json:"prev_cursor"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	require.Len(t, resp.Messages, 2)
	assert.Equal(t, msgs[1].ID, resp.Messages[0].ID)
	assert.Equal(t, msgs[2].ID, resp.Messages[1].ID)
	assert.Equal(t, msgs[2].ID.String(), resp.NextCursor)
	assert.Equal(t, msgs[1].ID.String(), resp.PrevCursor)
}

func TestApp_GetMessages_AfterIDCursor(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	base := time.Now().UTC().Add(-1 * time.Hour)
	msgs := make([]*models.Message, 5)
	for i := 0; i < 5; i++ {
		msgs[i] = createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name,
			base.Add(time.Duration(i)*time.Minute), "msg-"+string(rune('a'+i)))
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "after_id", msgs[1].ID.String())
	testutil.SetQueryParam(req, "limit", "2")

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Messages   []handlers.MessageResponse `json:"messages"`
		HasMore    bool                       `json:"has_more"`
		NextCursor string                     `json:"next_cursor"`
		PrevCursor string                     `json:"prev_cursor"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	require.Len(t, resp.Messages, 2)
	assert.Equal(t, msgs[2].ID, resp.Messages[0].ID)
	assert.Equal(t, msgs[3].ID, resp.Messages[1].ID)
}

func TestApp_GetMessages_BeforeIDWithSameCreatedAt(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	sameTime := time.Now().UTC().Add(-30 * time.Minute)
	msg1 := createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name, sameTime, "same-time-1")
	msg2 := createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name, sameTime, "same-time-2")
	msg3 := createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name, sameTime, "same-time-3")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "before_id", msg2.ID.String())

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Messages []handlers.MessageResponse `json:"messages"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	require.Len(t, resp.Messages, 1)
	assert.Equal(t, msg1.ID, resp.Messages[0].ID)
	_ = msg3
}

func TestApp_GetMessages_HasMoreIsTrueWhenMoreExist(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	base := time.Now().UTC().Add(-1 * time.Hour)
	for i := 0; i < 5; i++ {
		createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name,
			base.Add(time.Duration(i)*time.Minute), "msg-"+string(rune('a'+i)))
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "limit", "3")

	err := app.GetMessages(req)
	require.NoError(t, err)

	var resp struct {
		Messages []handlers.MessageResponse `json:"messages"`
		HasMore  bool                       `json:"has_more"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Len(t, resp.Messages, 3)
	assert.True(t, resp.HasMore)
}

func TestApp_GetMessages_HasMoreIsFalseWhenAllFetched(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	base := time.Now().UTC().Add(-1 * time.Hour)
	for i := 0; i < 3; i++ {
		createCursorPaginationTestMessage(t, app, org.ID, contact.ID, account.Name,
			base.Add(time.Duration(i)*time.Minute), "msg-"+string(rune('a'+i)))
	}

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())
	testutil.SetQueryParam(req, "limit", "50")

	err := app.GetMessages(req)
	require.NoError(t, err)

	var resp struct {
		Messages []handlers.MessageResponse `json:"messages"`
		HasMore  bool                       `json:"has_more"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Len(t, resp.Messages, 3)
	assert.False(t, resp.HasMore)
}

func TestApp_GetMessages_EmptyResult(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID,
		testutil.WithContactAccount(account.Name),
	)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.GetMessages(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Messages   []handlers.MessageResponse `json:"messages"`
		HasMore    bool                       `json:"has_more"`
		NextCursor string                     `json:"next_cursor"`
		PrevCursor string                     `json:"prev_cursor"`
	}
	testutil.ParseEnvelopeResponse(t, req, &resp)

	assert.Empty(t, resp.Messages)
	assert.False(t, resp.HasMore)
	assert.Empty(t, resp.NextCursor)
	assert.Empty(t, resp.PrevCursor)
}
