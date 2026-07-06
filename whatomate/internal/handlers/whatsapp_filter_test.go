package handlers_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

// getWhatsAppFilterPermissions returns wa_filter permissions from default set.
func getWhatsAppFilterPermissions(t *testing.T, app *handlers.App) []models.Permission {
	t.Helper()
	allPerms := testutil.GetOrCreateTestPermissions(t, app.DB)
	var waPerms []models.Permission
	for _, p := range allPerms {
		if p.Resource == models.ResourceWhatsAppFilter {
			waPerms = append(waPerms, p)
		}
	}
	require.NotEmpty(t, waPerms, "expected wa_filter permissions in default set")
	return waPerms
}

func createFilterTestWhatsAppInstance(t *testing.T, app *handlers.App, orgID uuid.UUID, name, phoneNumber string, status models.InstanceStatus) *models.WhatsAppInstance {
	t.Helper()
	instance := &models.WhatsAppInstance{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		Name:           name,
		PhoneNumber:    phoneNumber,
		Status:         status,
	}
	require.NoError(t, app.DB.Create(instance).Error)
	return instance
}

func createTestWhatsAppFilterBatch(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, name string) *models.WhatsAppFilterBatch {
	t.Helper()
	batch := &models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		CreatedBy:       userID,
		WhatsAppAccount: name,
		Status:          models.FilterStatusPending,
		TotalNumbers:    2,
	}
	require.NoError(t, app.DB.Create(batch).Error)
	return batch
}

func createTestWhatsAppFilterResult(t *testing.T, app *handlers.App, batchID uuid.UUID, phone, name string, isValid bool) *models.WhatsAppFilterResult {
	t.Helper()
	res := &models.WhatsAppFilterResult{
		BaseModel:   models.BaseModel{ID: uuid.New()},
		BatchID:     batchID,
		PhoneNumber: phone,
		ContactName: name,
		IsValid:     isValid,
	}
	require.NoError(t, app.DB.Create(res).Error)
	return res
}

func TestApp_WhatsAppFilter_CreateJSON(t *testing.T) {
	t.Parallel()

	t.Run("success JSON", func(t *testing.T) {
		mockQueue := testutil.NewMockQueue()
		app := newTestApp(t, withQueue(mockQueue))
		org := testutil.CreateTestOrganization(t, app.DB)
		perms := getWhatsAppFilterPermissions(t, app)
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Manager", false, false, perms)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-filter")), testutil.WithRoleID(&role.ID))

		// Set provider to whatsmeow for verification check path
		app.Config.WhatsApp.Provider = "whatsmeow"
		inst := createFilterTestWhatsAppInstance(t, app, org.ID, "Inst Name", "1234567890", models.InstanceStatusConnected)

		req := testutil.NewJSONRequest(t, map[string]any{
			"connection_id": inst.ID.String(),
			"phones":        []string{"+1234567890", "9876543210"},
			"names":         []string{"Alice", "Bob"},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.CreateWhatsAppFilterBatch(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

		var resp struct {
			Data models.WhatsAppFilterBatch `json:"data"`
		}
		err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
		require.NoError(t, err)
		assert.Equal(t, models.FilterStatusPending, resp.Data.Status)
		assert.Equal(t, 2, resp.Data.TotalNumbers)
		assert.Equal(t, "Inst Name", resp.Data.WhatsAppAccount)

		// Verify records in DB
		var dbBatch models.WhatsAppFilterBatch
		err = app.DB.Where("id = ?", resp.Data.ID).First(&dbBatch).Error
		require.NoError(t, err)
		assert.Equal(t, org.ID, dbBatch.OrganizationID)

		var dbResults []models.WhatsAppFilterResult
		err = app.DB.Where("batch_id = ?", resp.Data.ID).Order("phone_number asc").Find(&dbResults).Error
		require.NoError(t, err)
		assert.Len(t, dbResults, 2)
		assert.Equal(t, "1234567890", dbResults[0].PhoneNumber) // normalized
		assert.Equal(t, "Alice", dbResults[0].ContactName)

		// Verify job enqueued
		assert.Len(t, mockQueue.FilterJobs, 1)
		assert.Equal(t, resp.Data.ID, mockQueue.FilterJobs[0].BatchID)
	})

	t.Run("disconnected instance whatsmeow", func(t *testing.T) {
		app := newTestApp(t)
		org := testutil.CreateTestOrganization(t, app.DB)
		perms := getWhatsAppFilterPermissions(t, app)
		role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Disconnected", false, false, perms)
		user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-filter-disc")), testutil.WithRoleID(&role.ID))

		app.Config.WhatsApp.Provider = "whatsmeow"
		inst := createFilterTestWhatsAppInstance(t, app, org.ID, "Inst Name 2", "111222333", models.InstanceStatusDisconnected)

		req := testutil.NewJSONRequest(t, map[string]any{
			"connection_id": inst.ID.String(),
			"phones":        []string{"987654321"},
		})
		testutil.SetAuthContext(req, org.ID, user.ID)

		err := app.CreateWhatsAppFilterBatch(req)
		require.NoError(t, err)
		assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
		testutil.AssertErrorResponse(t, req, fasthttp.StatusBadRequest, "WhatsApp instance is disconnected. Please connect it first.")
	})
}

func TestApp_WhatsAppFilter_CreateMultipartCSV(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getWhatsAppFilterPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter CSV Manager", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-filter-csv")), testutil.WithRoleID(&role.ID))

	app.Config.WhatsApp.Provider = "whatsmeow"
	inst := createFilterTestWhatsAppInstance(t, app, org.ID, "Inst Name CSV", "333333333", models.InstanceStatusConnected)

	// Prepare multipart form with CSV file data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("connection_id", inst.ID.String())
	require.NoError(t, err)

	fileWriter, err := writer.CreateFormFile("file", "test.csv")
	require.NoError(t, err)

	csvContent := "Phone,Name\n+14155552671,John Doe\n+14155552672,Jane Smith\n"
	_, err = fileWriter.Write([]byte(csvContent))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Construct fastglue multipart request
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType(writer.FormDataContentType())
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody(body.Bytes())

	req := &fastglue.Request{RequestCtx: ctx}
	testutil.SetAuthContext(req, org.ID, user.ID)

	err = app.CreateWhatsAppFilterBatch(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data models.WhatsAppFilterBatch `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Data.TotalNumbers)

	// Verify GORM rows are correctly parsed
	var results []models.WhatsAppFilterResult
	err = app.DB.Where("batch_id = ?", resp.Data.ID).Order("phone_number asc").Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "14155552671", results[0].PhoneNumber)
	assert.Equal(t, "John Doe", results[0].ContactName)
}

func TestApp_WhatsAppFilter_ListBatches(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org1 := testutil.CreateTestOrganization(t, app.DB)
	org2 := testutil.CreateTestOrganization(t, app.DB)

	perms := getWhatsAppFilterPermissions(t, app)
	role1 := testutil.CreateTestRoleExact(t, app.DB, org1.ID, "WA Filter Org1", false, false, perms)
	role2 := testutil.CreateTestRoleExact(t, app.DB, org2.ID, "WA Filter Org2", false, false, perms)

	user1 := testutil.CreateTestUser(t, app.DB, org1.ID, testutil.WithEmail(testutil.UniqueEmail("wa-list-1")), testutil.WithRoleID(&role1.ID))
	user2 := testutil.CreateTestUser(t, app.DB, org2.ID, testutil.WithEmail(testutil.UniqueEmail("wa-list-2")), testutil.WithRoleID(&role2.ID))

	createTestWhatsAppFilterBatch(t, app, org1.ID, user1.ID, "Batch Org1 A")
	createTestWhatsAppFilterBatch(t, app, org1.ID, user1.ID, "Batch Org1 B")
	createTestWhatsAppFilterBatch(t, app, org2.ID, user2.ID, "Batch Org2 C")

	// Org1 user list
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org1.ID, user1.ID)

	err := app.ListWhatsAppFilterBatches(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Batches []models.WhatsAppFilterBatch `json:"data"`
			Total   int64                        `json:"total"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Batches, 2)
	assert.Equal(t, int64(2), resp.Data.Total)
}

func TestApp_WhatsAppFilter_GetBatch(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getWhatsAppFilterPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Reader", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-get")), testutil.WithRoleID(&role.ID))

	batch := createTestWhatsAppFilterBatch(t, app, org.ID, user.ID, "Get Batch A")

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", batch.ID.String())

	err := app.GetWhatsAppFilterBatch(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data models.WhatsAppFilterBatch `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Equal(t, batch.ID, resp.Data.ID)
}

func TestApp_WhatsAppFilter_GetBatchResults(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getWhatsAppFilterPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Reader Results", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-results")), testutil.WithRoleID(&role.ID))

	batch := createTestWhatsAppFilterBatch(t, app, org.ID, user.ID, "Get Results Batch")
	createTestWhatsAppFilterResult(t, app, batch.ID, "12345678", "Alice", true)
	createTestWhatsAppFilterResult(t, app, batch.ID, "98765432", "Bob", false)

	// Fetch all results
	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", batch.ID.String())

	err := app.GetWhatsAppFilterBatchResults(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var resp struct {
		Data struct {
			Results []models.WhatsAppFilterResult `json:"data"`
			Total   int64                         `json:"total"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(req), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Results, 2)
	assert.Equal(t, int64(2), resp.Data.Total)

	// Fetch valid status filter results
	reqFilter := testutil.NewGETRequest(t)
	testutil.SetAuthContext(reqFilter, org.ID, user.ID)
	testutil.SetPathParam(reqFilter, "id", batch.ID.String())
	reqFilter.RequestCtx.URI().QueryArgs().Add("status", "valid")

	err = app.GetWhatsAppFilterBatchResults(reqFilter)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(reqFilter))

	var respFilter struct {
		Data struct {
			Results []models.WhatsAppFilterResult `json:"data"`
			Total   int64                         `json:"total"`
		} `json:"data"`
	}
	err = json.Unmarshal(testutil.GetResponseBody(reqFilter), &respFilter)
	require.NoError(t, err)
	assert.Len(t, respFilter.Data.Results, 1)
	assert.Equal(t, "Alice", respFilter.Data.Results[0].ContactName)
}

func TestApp_WhatsAppFilter_ExportCSV(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getWhatsAppFilterPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Exporter", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-export")), testutil.WithRoleID(&role.ID))

	batch := createTestWhatsAppFilterBatch(t, app, org.ID, user.ID, "Export Batch")
	createTestWhatsAppFilterResult(t, app, batch.ID, "12345678", "Alice", true)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", batch.ID.String())

	err := app.ExportWhatsAppFilterResults(req)
	require.NoError(t, err)
	assert.Equal(t, "text/csv", string(req.RequestCtx.Response.Header.Peek("Content-Type")))

	bodyStr := string(req.RequestCtx.Response.Body())
	assert.Contains(t, bodyStr, "Phone Number,Contact Name,Registered on WhatsApp,Checked At,Error Message")
	assert.Contains(t, bodyStr, "12345678")
	assert.Contains(t, bodyStr, "Alice")
	assert.Contains(t, bodyStr, "true")
}

func TestApp_WhatsAppFilter_DeleteBatch(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	perms := getWhatsAppFilterPermissions(t, app)
	role := testutil.CreateTestRoleExact(t, app.DB, org.ID, "WA Filter Deleter", false, false, perms)
	user := testutil.CreateTestUser(t, app.DB, org.ID, testutil.WithEmail(testutil.UniqueEmail("wa-delete")), testutil.WithRoleID(&role.ID))

	batch := createTestWhatsAppFilterBatch(t, app, org.ID, user.ID, "Delete Batch")
	res := createTestWhatsAppFilterResult(t, app, batch.ID, "12345678", "Alice", true)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", batch.ID.String())

	err := app.DeleteWhatsAppFilterBatch(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	// Verify records are gone in DB
	var batchCount int64
	app.DB.Model(&models.WhatsAppFilterBatch{}).Where("id = ?", batch.ID).Count(&batchCount)
	assert.Equal(t, int64(0), batchCount)

	var resCount int64
	app.DB.Model(&models.WhatsAppFilterResult{}).Where("id = ?", res.ID).Count(&resCount)
	assert.Equal(t, int64(0), resCount)
}
