package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type mediaPolicyFixture struct {
	app        *handlers.App
	mockServer *mockWhatsAppServer
	orgID      uuid.UUID
	userID     uuid.UUID
	contactID  uuid.UUID
}

func setupMediaPolicyFixture(t *testing.T) *mediaPolicyFixture {
	t.Helper()

	mockServer := newMockWhatsAppServer()
	app := newMsgTestApp(t, mockServer)

	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := createTestAccount(t, app, org.ID)
	contact := testutil.CreateTestContactWith(t, app.DB, org.ID, testutil.WithContactAccount(account.Name))

	require.NoError(t, app.DB.Model(&models.Contact{}).
		Where("id = ?", contact.ID).
		Updates(map[string]any{
			"assigned_user_id": user.ID,
			"status":           models.ChatStatusOpen,
		}).Error)

	return &mediaPolicyFixture{
		app:        app,
		mockServer: mockServer,
		orgID:      org.ID,
		userID:     user.ID,
		contactID:  contact.ID,
	}
}

func (f *mediaPolicyFixture) close() {
	f.app.WaitForBackgroundTasks()
	f.mockServer.close()
}

func newMediaMultipartRequest(
	t *testing.T,
	orgID uuid.UUID,
	userID uuid.UUID,
	contactID uuid.UUID,
	fileName string,
	fileMIME string,
	formType string,
	fileData []byte,
) *fastglue.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	require.NoError(t, writer.WriteField("contact_id", contactID.String()))
	if formType != "" {
		require.NoError(t, writer.WriteField("type", formType))
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	if fileMIME != "" {
		partHeader.Set("Content-Type", fileMIME)
	}

	part, err := writer.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	req.RequestCtx.Request.Header.SetContentType(writer.FormDataContentType())
	req.RequestCtx.Request.SetBody(body.Bytes())
	testutil.SetAuthContext(req, orgID, userID)

	return req
}

func TestSendMediaMessage_AcceptsZipAsDocumentAndIgnoresClientType(t *testing.T) {
	fixture := setupMediaPolicyFixture(t)
	defer fixture.close()

	zipData := []byte("PK\x03\x04zip-content")
	req := newMediaMultipartRequest(
		t,
		fixture.orgID,
		fixture.userID,
		fixture.contactID,
		"archive.zip",
		"application/zip",
		"image",
		zipData,
	)

	err := fixture.app.SendMediaMessage(req)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var envelope struct {
		Data struct {
			MessageType   string `json:"message_type"`
			MediaMimeType string `json:"media_mime_type"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &envelope))
	assert.Equal(t, "document", envelope.Data.MessageType)
	assert.Equal(t, "application/zip", envelope.Data.MediaMimeType)
}

func TestSendMediaMessage_RejectsOversizedJPEG(t *testing.T) {
	fixture := setupMediaPolicyFixture(t)
	defer fixture.close()

	imageData := make([]byte, 5*1024*1024+1)
	copy(imageData, []byte{0xff, 0xd8, 0xff, 0xdb})

	req := newMediaMultipartRequest(
		t,
		fixture.orgID,
		fixture.userID,
		fixture.contactID,
		"photo.jpg",
		"image/jpeg",
		"document",
		imageData,
	)

	err := fixture.app.SendMediaMessage(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "image file is too large (max 5MB)")
}

func TestSendMediaMessage_RejectsOversizedMPEG(t *testing.T) {
	fixture := setupMediaPolicyFixture(t)
	defer fixture.close()

	audioData := make([]byte, 16*1024*1024+1)
	copy(audioData, []byte("ID3"))

	req := newMediaMultipartRequest(
		t,
		fixture.orgID,
		fixture.userID,
		fixture.contactID,
		"voice.mp3",
		"audio/mpeg",
		"document",
		audioData,
	)

	err := fixture.app.SendMediaMessage(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "audio file is too large (max 16MB)")
}

func TestSendMediaMessage_RejectsOversizedDocument(t *testing.T) {
	fixture := setupMediaPolicyFixture(t)
	defer fixture.close()

	documentData := make([]byte, 100*1024*1024+1)
	copy(documentData, []byte("PK\x03\x04"))

	req := newMediaMultipartRequest(
		t,
		fixture.orgID,
		fixture.userID,
		fixture.contactID,
		"large.zip",
		"application/zip",
		"image",
		documentData,
	)

	err := fixture.app.SendMediaMessage(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "document file is too large (max 100MB)")
}
