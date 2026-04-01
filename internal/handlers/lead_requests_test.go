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

func getSettingsGeneralPermissions(t *testing.T, app *handlers.App) []models.Permission {
	t.Helper()

	allPerms := testutil.GetOrCreateTestPermissions(t, app.DB)
	var perms []models.Permission
	for _, p := range allPerms {
		if p.Resource == models.ResourceSettingsGeneral {
			perms = append(perms, p)
		}
	}
	require.NotEmpty(t, perms, "expected settings.general permissions in default set")
	return perms
}

func createTestLeadRequest(t *testing.T, app *handlers.App, fullName string, status models.LeadRequestStatus, requestedPlan string) *models.LeadRequest {
	t.Helper()

	lead := &models.LeadRequest{
		BaseModel:     models.BaseModel{ID: uuid.New()},
		FullName:      fullName,
		CompanyName:   fullName + " Co",
		WorkEmail:     testutil.UniqueEmail("lead"),
		PhoneWhatsApp: "+966500000000",
		Country:       "Saudi Arabia",
		Message:       "Please contact us about the platform.",
		RequestedPlan: requestedPlan,
		SourcePage:    "pricing",
		SourceRoute:   "/pricing",
		Status:        status,
	}
	require.NoError(t, app.DB.Create(lead).Error)
	return lead
}

func TestApp_CreatePublicLeadRequest(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		app := newTestApp(t)

		req := testutil.NewJSONRequest(t, map[string]any{
			"full_name":      "Alice Example",
			"company_name":   "Example Corp",
			"work_email":     "alice@example.com",
			"phone_whatsapp": "+966500001111",
			"country":        "Saudi Arabia",
			"message":        "Need a demo for our support team.",
			"requested_plan": "growth",
			"source_page":    "pricing",
			"source_route":   "/plans",
		})

		err := app.CreatePublicLeadRequest(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				ID      uuid.UUID                `json:"id"`
				Status  models.LeadRequestStatus `json:"status"`
				Message string                   `json:"message"`
			} `json:"data"`
		}
		err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, resp.Data.ID)
		assert.Equal(t, models.LeadRequestStatusNew, resp.Data.Status)
		assert.Equal(t, "Lead request submitted successfully", resp.Data.Message)

		var stored models.LeadRequest
		require.NoError(t, app.DB.Where("id = ?", resp.Data.ID).First(&stored).Error)
		assert.Equal(t, "Alice Example", stored.FullName)
		assert.Equal(t, "Example Corp", stored.CompanyName)
		assert.Equal(t, "alice@example.com", stored.WorkEmail)
		assert.Equal(t, "growth", stored.RequestedPlan)
		assert.Equal(t, "/plans", stored.SourceRoute)
		assert.Equal(t, models.LeadRequestStatusNew, stored.Status)
	})

	t.Run("rejects invalid email", func(t *testing.T) {
		app := newTestApp(t)

		req := testutil.NewJSONRequest(t, map[string]any{
			"full_name":      "Alice Example",
			"company_name":   "Example Corp",
			"work_email":     "not-an-email",
			"phone_whatsapp": "+966500001111",
			"source_page":    "pricing",
			"source_route":   "/pricing",
		})

		err := app.CreatePublicLeadRequest(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "work_email must be a valid email address")
	})
}

func TestApp_ListLeadRequests(t *testing.T) {
	t.Parallel()

	t.Run("success with search and status filter", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		perms := getSettingsGeneralPermissions(t, app)
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Lead Reader", false, false, perms)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("lead-reader")), testutil.WithRoleID(&role.ID))

		createTestLeadRequest(t, app, "Alice Example", models.LeadRequestStatusNew, "growth")
		createTestLeadRequest(t, app, "Bob Qualified", models.LeadRequestStatusQualified, "enterprise")

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetQueryParam(req, "search", "Bob")
		testutil.SetQueryParam(req, "status", "qualified")

		err := app.ListLeadRequests(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data struct {
				LeadRequests []handlers.LeadRequestResponse `json:"lead_requests"`
				Total        int                            `json:"total"`
			} `json:"data"`
		}
		err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
		require.NoError(t, err)

		require.Len(t, resp.Data.LeadRequests, 1)
		assert.Equal(t, 1, resp.Data.Total)
		assert.Equal(t, "Bob Qualified", resp.Data.LeadRequests[0].FullName)
		assert.Equal(t, models.LeadRequestStatusQualified, resp.Data.LeadRequests[0].Status)
	})

	t.Run("requires settings.general read", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "No Settings Permission", false, false, nil)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("lead-no-read")), testutil.WithRoleID(&role.ID))

		req := testutil.NewGETRequest(t)
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.ListLeadRequests(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
	})
}

func TestApp_UpdateLeadRequestStatus(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		perms := getSettingsGeneralPermissions(t, app)
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Lead Writer", false, false, perms)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("lead-writer")), testutil.WithRoleID(&role.ID))
		lead := createTestLeadRequest(t, app, "Charlie Contact", models.LeadRequestStatusNew, "starter")

		req := testutil.NewJSONRequest(t, map[string]any{
			"status": "contacted",
		})
		req.RequestCtx.Request.Header.SetMethod("PUT")
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", lead.ID.String())

		err := app.UpdateLeadRequestStatus(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data handlers.LeadRequestResponse `json:"data"`
		}
		err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
		require.NoError(t, err)
		assert.Equal(t, models.LeadRequestStatusContacted, resp.Data.Status)

		var stored models.LeadRequest
		require.NoError(t, app.DB.Where("id = ?", lead.ID).First(&stored).Error)
		assert.Equal(t, models.LeadRequestStatusContacted, stored.Status)
	})

	t.Run("requires settings.general write", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		perms := getSettingsGeneralPermissions(t, app)
		var readOnlyPerms []models.Permission
		for _, perm := range perms {
			if perm.Action == models.ActionRead {
				readOnlyPerms = append(readOnlyPerms, perm)
			}
		}
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "Lead Read Only", false, false, readOnlyPerms)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("lead-read-only")), testutil.WithRoleID(&role.ID))
		lead := createTestLeadRequest(t, app, "Dana Closed", models.LeadRequestStatusNew, "dedicated")

		req := testutil.NewJSONRequest(t, map[string]any{
			"status": "closed",
		})
		req.RequestCtx.Request.Header.SetMethod("PUT")
		testutil.SetAuthContext(req, org.ID, user.ID)
		testutil.SetPathParam(req, "id", lead.ID.String())

		err := app.UpdateLeadRequestStatus(req)
		require.NoError(t, err)
		testutil.AssertErrorResponse(t, req, fasthttp.StatusForbidden, "Insufficient permissions")
	})
}
