package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/gowa-ui/internal/handlers"
	"github.com/shridarpatil/gowa-ui/internal/models"
	"github.com/shridarpatil/gowa-ui/internal/templateutil"
	"github.com/shridarpatil/gowa-ui/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- ExtParamNames Tests (existing) ---

func TestExtParamNames_PositionalParams(t *testing.T) {
	t.Parallel()
	content := "Hello {{1}}, your order {{2}} is ready!"
	result := templateutil.ExtParamNames(content)
	assert.Equal(t, []string{"1", "2"}, result)
}

func TestExtParamNames_NamedParams(t *testing.T) {
	t.Parallel()
	content := "Hello {{name}}, your order {{order_id}} is ready!"
	result := templateutil.ExtParamNames(content)
	assert.Equal(t, []string{"name", "order_id"}, result)
}

func TestExtParamNames_MixedParams(t *testing.T) {
	t.Parallel()
	content := "Hello {{1}}, your order {{order_id}} is ready! Amount: {{3}}"
	result := templateutil.ExtParamNames(content)
	assert.Equal(t, []string{"1", "order_id", "3"}, result)
}

func TestExtParamNames_NoParams(t *testing.T) {
	t.Parallel()
	content := "Hello, your order is ready!"
	result := templateutil.ExtParamNames(content)
	assert.Nil(t, result)
}

func TestExtParamNames_DuplicateParams(t *testing.T) {
	t.Parallel()
	content := "Hello {{name}}, {{name}} your order {{order_id}} is ready!"
	result := templateutil.ExtParamNames(content)
	// Should only return unique names in order of first occurrence
	assert.Equal(t, []string{"name", "order_id"}, result)
}

func TestExtParamNames_UnderscoreParams(t *testing.T) {
	t.Parallel()
	content := "Hello {{customer_name}}, order {{order_number}} total {{total_amount}}"
	result := templateutil.ExtParamNames(content)
	assert.Equal(t, []string{"customer_name", "order_number", "total_amount"}, result)
}

// --- Template handler test helpers ---

// createTestTemplateInDB creates a template directly in the database for testing.
func createTestTemplateInDB(t *testing.T, app *handlers.App, orgID uuid.UUID, accountName, name string) *models.Template {
	t.Helper()

	tmpl := &models.Template{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		WhatsAppAccount: accountName,
		Name:            name,
		DisplayName:     name,
		Language:        "en",
		Category:        "MARKETING",
		BodyContent:     "Hello {{1}}, welcome!",
	}
	require.NoError(t, app.DB.Create(tmpl).Error)
	return tmpl
}

// --- ListTemplates Tests ---

func TestApp_ListTemplates_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	createTestTemplateInDB(t, app, org.ID, account.Name, "template_one")
	createTestTemplateInDB(t, app, org.ID, account.Name, "template_two")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Templates []handlers.TemplateResponse `json:"templates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Len(t, resp.Data.Templates, 2)
}

func TestApp_ListTemplates_EmptyList(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Templates []handlers.TemplateResponse `json:"templates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Len(t, resp.Data.Templates, 0)
}

func TestApp_ListTemplates_FilterByAccount(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account1 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)
	account2 := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	createTestTemplateInDB(t, app, org.ID, account1.Name, "tmpl_a1")
	createTestTemplateInDB(t, app, org.ID, account1.Name, "tmpl_a2")
	createTestTemplateInDB(t, app, org.ID, account2.Name, "tmpl_b1")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "account", account1.Name)

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Templates []handlers.TemplateResponse `json:"templates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Len(t, resp.Data.Templates, 2)
	for _, tmpl := range resp.Data.Templates {
		assert.Equal(t, account1.Name, tmpl.WhatsAppAccount)
	}
}

func TestApp_ListTemplates_FilterByCategory(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	createTestTemplateInDB(t, app, org.ID, account.Name, "marketing_tmpl")

	// Create a UTILITY template directly
	utilTmpl := &models.Template{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  org.ID,
		WhatsAppAccount: account.Name,
		Name:            "utility_tmpl",
		DisplayName:     "utility_tmpl",
		Language:        "en",
		Category:        "UTILITY",
		BodyContent:     "Your OTP is {{1}}",
	}
	require.NoError(t, app.DB.Create(utilTmpl).Error)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetQueryParam(req, "category", "UTILITY")

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Templates []handlers.TemplateResponse `json:"templates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Len(t, resp.Data.Templates, 1)
	assert.Equal(t, "UTILITY", resp.Data.Templates[0].Category)
}

func TestApp_ListTemplates_CrossOrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user1 := testutil.CreateTestUser(t, app.DB, org1.ID)
	account1 := testutil.CreateTestWhatsAppAccount(t, app.DB, org1.ID)
	account2 := testutil.CreateTestWhatsAppAccount(t, app.DB, org2.ID)

	createTestTemplateInDB(t, app, org1.ID, account1.Name, "org1_tmpl")
	createTestTemplateInDB(t, app, org2.ID, account2.Name, "org2_tmpl")

	// User from org1 should only see org1 templates
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org1.ID, user1.ID)

	err := app.ListTemplates(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Templates []handlers.TemplateResponse `json:"templates"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Len(t, resp.Data.Templates, 1)
	assert.Equal(t, "org1_tmpl", resp.Data.Templates[0].Name)
}

// --- GetTemplate Tests ---

func TestApp_GetTemplate_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	tmpl := createTestTemplateInDB(t, app, org.ID, account.Name, "get_me")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", tmpl.ID.String())

	err := app.GetTemplate(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data handlers.TemplateResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(testutil.GetResponseBody(req), &resp))
	assert.Equal(t, tmpl.ID, resp.Data.ID)
	assert.Equal(t, "get_me", resp.Data.Name)
	assert.Equal(t, account.Name, resp.Data.WhatsAppAccount)
}

func TestApp_GetTemplate_NotFound(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", uuid.New().String())

	err := app.GetTemplate(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "not found")
}

func TestApp_GetTemplate_InvalidID(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := testutil.CreateTestUser(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	err := app.GetTemplate(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "Invalid template ID")
}

func TestApp_GetTemplate_CrossOrgIsolation(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)
	user1 := testutil.CreateTestUser(t, app.DB, org1.ID)
	account2 := testutil.CreateTestWhatsAppAccount(t, app.DB, org2.ID)

	tmpl := createTestTemplateInDB(t, app, org2.ID, account2.Name, "org2_private")

	// User from org1 should not be able to access org2 template
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org1.ID, user1.ID)
	testutil.SetPathParam(req, "id", tmpl.ID.String())

	err := app.GetTemplate(req)
	require.NoError(t, err)
	testutil.AssertErrorResponse(t, req, fasthttp.StatusNotFound, "not found")
}
