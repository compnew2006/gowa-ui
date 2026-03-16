package handlers_test

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TestGetBusinessProfile_Unauthorized tests GetBusinessProfile without auth
func TestGetBusinessProfile_Unauthorized(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := testutil.NewGETRequest(t)
	// No auth context set

	err := app.GetBusinessProfile(req)

	assert.NoError(t, err, "GetBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req), "Should return 401 for unauthorized")
}

// TestGetBusinessProfile_MissingAccountID tests GetBusinessProfile with missing account ID
func TestGetBusinessProfile_MissingAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	// No account ID set in path

	err := app.GetBusinessProfile(req)

	assert.NoError(t, err, "GetBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for missing account ID")
}

// TestGetBusinessProfile_InvalidAccountID tests GetBusinessProfile with invalid account ID
func TestGetBusinessProfile_InvalidAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", "not-a-uuid")

	err := app.GetBusinessProfile(req)

	assert.NoError(t, err, "GetBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for invalid account ID")
}

// TestUpdateBusinessProfile_Unauthorized tests UpdateBusinessProfile without auth
func TestUpdateBusinessProfile_Unauthorized(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := testutil.NewJSONRequest(t, map[string]string{"test": "data"})
	// No auth context set

	err := app.UpdateBusinessProfile(req)

	assert.NoError(t, err, "UpdateBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req), "Should return 401 for unauthorized")
}

// TestUpdateBusinessProfile_MissingAccountID tests UpdateBusinessProfile with missing account ID
func TestUpdateBusinessProfile_MissingAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{"test": "data"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	// No account ID set in path

	err := app.UpdateBusinessProfile(req)

	assert.NoError(t, err, "UpdateBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for missing account ID")
}

// TestUpdateBusinessProfile_InvalidAccountID tests UpdateBusinessProfile with invalid account ID
func TestUpdateBusinessProfile_InvalidAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, map[string]string{"test": "data"})
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", "not-a-uuid")

	err := app.UpdateBusinessProfile(req)

	assert.NoError(t, err, "UpdateBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for invalid account ID")
}

// TestUpdateBusinessProfile_InvalidJSON tests UpdateBusinessProfile with invalid JSON
func TestUpdateBusinessProfile_InvalidJSON(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewJSONRequest(t, nil) // nil body
	req.RequestCtx.Request.SetBody([]byte("{invalid json}"))
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", testutil.CreateTestUser(t, app.DB, org.ID).ID.String())

	err := app.UpdateBusinessProfile(req)

	assert.NoError(t, err, "UpdateBusinessProfile should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for invalid JSON")
}

// TestUpdateProfilePicture_Unauthorized tests UpdateProfilePicture without auth
func TestUpdateProfilePicture_Unauthorized(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := createMultipartRequest(t, nil)
	// No auth context set

	err := app.UpdateProfilePicture(req)

	assert.NoError(t, err, "UpdateProfilePicture should return nil error")
	assert.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(req), "Should return 401 for unauthorized")
}

// TestUpdateProfilePicture_MissingAccountID tests UpdateProfilePicture with missing account ID
func TestUpdateProfilePicture_MissingAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := createMultipartRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	// No account ID set in path

	err := app.UpdateProfilePicture(req)

	assert.NoError(t, err, "UpdateProfilePicture should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for missing account ID")
}

// TestUpdateProfilePicture_InvalidAccountID tests UpdateProfilePicture with invalid account ID
func TestUpdateProfilePicture_InvalidAccountID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := createMultipartRequest(t, nil)
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", "not-a-uuid")

	err := app.UpdateProfilePicture(req)

	assert.NoError(t, err, "UpdateProfilePicture should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for invalid account ID")
}

// TestUpdateProfilePicture_MissingFile tests UpdateProfilePicture without file
func TestUpdateProfilePicture_MissingFile(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	req.RequestCtx.Request.Header.SetMethod("POST")
	req.RequestCtx.Request.Header.SetContentType("multipart/form-data")
	testutil.SetAuthContext(req, org.ID, user.ID)
	req.RequestCtx.SetUserValue("id", testutil.CreateTestUser(t, app.DB, org.ID).ID.String())

	err := app.UpdateProfilePicture(req)

	assert.NoError(t, err, "UpdateProfilePicture should return nil error")
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req), "Should return 400 for missing file")
}

// createMultipartRequest creates a multipart request for testing file uploads
func createMultipartRequest(t *testing.T, fileData []byte) *fastglue.Request {
	t.Helper()

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")

	// Create multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if fileData != nil {
		part, err := writer.CreateFormFile("file", "test.jpg")
		assert.NoError(t, err, "Should create form file")

		_, err = part.Write(fileData)
		assert.NoError(t, err, "Should write file data")
	}

	err := writer.Close()
	assert.NoError(t, err, "Should close writer")

	ctx.Request.SetBody(body.Bytes())
	ctx.Request.Header.SetContentType(writer.FormDataContentType())

	return &fastglue.Request{RequestCtx: ctx}
}

// TestCreateMultipartRequest_ValidFile tests multipart request creation helper
func TestCreateMultipartRequest_ValidFile(t *testing.T) {
	t.Parallel()

	fileData := []byte("test image data")
	req := createMultipartRequest(t, fileData)

	assert.NotNil(t, req, "Should create request")
	assert.Equal(t, "POST", string(req.RequestCtx.Request.Header.Method()), "Should be POST request")
	assert.NotEmpty(t, req.RequestCtx.Request.Header.Peek("Content-Type"), "Should have Content-Type")
	assert.Contains(t, string(req.RequestCtx.Request.Header.Peek("Content-Type")), "multipart/form-data", "Should be multipart form data")
	assert.NotEmpty(t, req.RequestCtx.Request.Body(), "Should have request body")
}
