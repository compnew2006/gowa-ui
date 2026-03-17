package handlers_test

import (
	"bytes"
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

func createSecurityTestUserWithKeys(t *testing.T, app *handlers.App, orgID uuid.UUID, roleName string, permissionKeys []string) *models.User {
	t.Helper()

	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, roleName, permissionKeys)
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))
}

func newSecurityMultipartRequest(
	t *testing.T,
	fileField string,
	fileName string,
	fileMIME string,
	fileData []byte,
	formFields map[string]string,
) *fastglue.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range formFields {
		require.NoError(t, writer.WriteField(key, value))
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fileField, fileName))
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

	return req
}

func TestTemplateRBAC_ListTemplates_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "template-read-denied", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestTemplateRBAC_CreateTemplate_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "template-write-denied", []string{"chat:read"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]any{
		"whatsapp_account": account.Name,
		"name":             "Promo Template",
		"language":         "en",
		"category":         "MARKETING",
		"body_content":     "Hello {{1}}",
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateTemplate(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestTemplateRBAC_SyncTemplates_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	server := newMockTemplateServer(t)
	t.Cleanup(server.Close)

	app := newTemplateTestApp(t, server)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "template-sync-denied", []string{"templates:read"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "account", account.Name)

	err := app.SyncTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestCampaignRBAC_ListCampaigns_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, withQueue(testutil.NewMockQueue()))
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "campaign-read-denied", []string{"chat:read"})

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListCampaigns(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestCampaignRBAC_CreateCampaign_ForbiddenWithoutPermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, withQueue(testutil.NewMockQueue()))
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "campaign-write-denied", []string{"campaigns:read"})
	account := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID, testutil.WithAccountName("campaign-write-denied"))
	template := testutil.CreateTestTemplate(t, app.DB, org.ID, account.Name)

	req := testutil.NewJSONRequest(t, map[string]any{
		"name":             "Promo Campaign",
		"whatsapp_account": account.Name,
		"template_id":      template.ID.String(),
	})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.CreateCampaign(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestCampaignRBAC_StartCampaign_ForbiddenWithoutExecutePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, withQueue(testutil.NewMockQueue()))
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "campaign-no-execute", []string{
		"campaigns:read",
		"campaigns:write",
		"campaigns:delete",
	})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	template := testutil.CreateTestTemplate(t, app.DB, org.ID, account.Name)
	campaign := createTestCampaign(t, app, org.ID, template.ID, user.ID, account.Name, models.CampaignStatusDraft)
	createTestRecipient(t, app, campaign.ID, "1234567890", models.MessageStatusPending)

	req := testutil.NewRequest(t)
	req.RequestCtx.Request.Header.SetMethod(fasthttp.MethodPost)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", campaign.ID.String())

	err := app.StartCampaign(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestUploadTemplateMedia_RejectsDisguisedImage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createTemplateUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(account).Update("app_id", "app-test").Error)

	req := newSecurityMultipartRequest(
		t,
		"file",
		"sample.png",
		"image/png",
		[]byte("<html>not an image</html>"),
		map[string]string{"account": account.Name},
	)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UploadTemplateMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "Unsupported file type")
}

func TestUploadTemplateMedia_RejectsOversizedImageBySniffedType(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createTemplateUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	require.NoError(t, app.DB.Model(account).Update("app_id", "app-test").Error)

	largePNG := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 6*1024*1024)...)
	req := newSecurityMultipartRequest(
		t,
		"file",
		"sample.png",
		"image/png",
		largePNG,
		map[string]string{"account": account.Name},
	)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.UploadTemplateMedia(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "Maximum size is 5MB")
}

func TestUpdateProfilePicture_RejectsDisguisedImage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "accounts-writer", []string{"accounts:write"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := newSecurityMultipartRequest(
		t,
		"file",
		"avatar.png",
		"image/png",
		[]byte("<html>not an image</html>"),
		nil,
	)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateProfilePicture(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "Unsupported file type")
}

func TestUpdateProfilePicture_RejectsOversizedImage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createSecurityTestUserWithKeys(t, app, org.ID, "accounts-writer-large", []string{"accounts:write"})
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	largePNG := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 6*1024*1024)...)
	req := newSecurityMultipartRequest(
		t,
		"file",
		"avatar.png",
		"image/png",
		largePNG,
		nil,
	)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.UpdateProfilePicture(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
	assert.Contains(t, string(testutil.GetResponseBody(req)), "Maximum size is 5MB")
}
