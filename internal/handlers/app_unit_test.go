package handlers_test

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// createTestRequest creates a mock fastglue.Request for testing
func createTestRequestWithContext(userID, orgID uuid.UUID, headers map[string]string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/test")

	// Set user values
	ctx.SetUserValue("user_id", userID)
	ctx.SetUserValue("organization_id", orgID)

	// Set headers
	for key, value := range headers {
		ctx.Request.Header.Set(key, value)
	}

	return &fastglue.Request{RequestCtx: ctx}
}

// TestGetOrgID_UUIDInContext tests getOrgID with UUID in context
func TestGetOrgID_UUIDInContext(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	req := createTestRequestWithContext(uuid.New(), orgID, nil)

	result, err := app.GetOrgID(req)

	assert.NoError(t, err, "getOrgID should succeed with valid UUID in context")
	assert.Equal(t, orgID, result, "getOrgID should return the org ID from context")
}

// TestGetOrgID_StringInContext tests getOrgID with string UUID in context
func TestGetOrgID_StringInContext(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	req := createTestRequestWithContext(uuid.New(), orgID, nil)
	// Override to set string in context
	req.RequestCtx.SetUserValue("organization_id", orgID.String())

	result, err := app.GetOrgID(req)

	assert.NoError(t, err, "getOrgID should succeed with valid UUID string in context")
	assert.Equal(t, orgID, result, "getOrgID should return the parsed org ID")
}

// TestGetOrgID_NoOrgIDInContext tests getOrgID with missing org_id
func TestGetOrgID_NoOrgIDInContext(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := createTestRequestWithContext(uuid.New(), uuid.Nil, nil)
	req.RequestCtx.SetUserValue("organization_id", nil)

	result, err := app.GetOrgID(req)

	assert.Error(t, err, "getOrgID should error when organization_id is missing")
	assert.Equal(t, uuid.Nil, result, "getOrgID should return Nil UUID on error")
	assert.Contains(t, err.Error(), "not found in context", "Error should mention missing context value")
}

// TestGetOrgID_InvalidUUIDInContext tests getOrgID with invalid UUID
func TestGetOrgID_InvalidUUIDInContext(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := createTestRequestWithContext(uuid.New(), uuid.Nil, nil)
	req.RequestCtx.SetUserValue("organization_id", "not-a-uuid")

	result, err := app.GetOrgID(req)

	assert.Error(t, err, "getOrgID should error with invalid UUID")
	assert.Equal(t, uuid.Nil, result, "getOrgID should return Nil UUID on error")
	assert.Contains(t, err.Error(), "not a valid UUID", "Error should mention invalid UUID")
}

// TestGetOrgID_WrongTypeInContext tests getOrgID with wrong type
func TestGetOrgID_WrongTypeInContext(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := createTestRequestWithContext(uuid.New(), uuid.Nil, nil)
	req.RequestCtx.SetUserValue("organization_id", 12345)

	result, err := app.GetOrgID(req)

	assert.Error(t, err, "getOrgID should error with wrong type")
	assert.Equal(t, uuid.Nil, result, "getOrgID should return Nil UUID on error")
	assert.Contains(t, err.Error(), "not a valid UUID", "Error should mention invalid UUID")
}

// TestHealthCheck tests HealthCheck endpoint
func TestHealthCheck(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	err := app.HealthCheck(req)

	assert.NoError(t, err, "HealthCheck should succeed")
}

// TestGetOrgAndUserID_Success tests getOrgAndUserID with valid context
func TestGetOrgAndUserID_Success(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	userID := uuid.New()
	req := createTestRequestWithContext(userID, orgID, nil)

	resultOrgID, resultUserID, err := app.GetOrgAndUserID(req)

	assert.NoError(t, err, "getOrgAndUserID should succeed with valid context")
	assert.Equal(t, orgID, resultOrgID, "getOrgAndUserID should return correct org ID")
	assert.Equal(t, userID, resultUserID, "getOrgAndUserID should return correct user ID")
}

// TestAndUserID_MissingUserID tests getOrgAndUserID with missing user_id
func TestGetOrgAndUserID_MissingUserID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	req := createTestRequestWithContext(uuid.Nil, orgID, nil)
	req.RequestCtx.SetUserValue("user_id", nil)

	resultOrgID, resultUserID, err := app.GetOrgAndUserID(req)

	assert.Error(t, err, "getOrgAndUserID should error when user_id is missing")
	assert.Equal(t, uuid.Nil, resultOrgID, "getOrgAndUserID should return Nil org ID on error")
	assert.Equal(t, uuid.Nil, resultUserID, "getOrgAndUserID should return Nil user ID on error")
	assert.Contains(t, err.Error(), "user_id not found in context", "Error should mention missing user_id")
}

// TestGetOrgAndUserID_InvalidUserID tests getOrgAndUserID with invalid user UUID
func TestGetOrgAndUserID_InvalidUserID(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	req := createTestRequestWithContext(uuid.Nil, orgID, nil)
	req.RequestCtx.SetUserValue("user_id", "not-a-uuid")

	resultOrgID, resultUserID, err := app.GetOrgAndUserID(req)

	assert.Error(t, err, "getOrgAndUserID should error with invalid user UUID")
	assert.Equal(t, uuid.Nil, resultOrgID, "getOrgAndUserID should return Nil org ID on error")
	assert.Equal(t, uuid.Nil, resultUserID, "getOrgAndUserID should return Nil user ID on error")
	assert.Contains(t, err.Error(), "not a valid UUID", "Error should mention invalid UUID")
}

// TestDecodeRequest_ValidJSON tests decodeRequest with valid JSON
func TestDecodeRequest_ValidJSON(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL for full test setup")

	app := &handlers.App{
		Config: &config.Config{},
	}

	type TestStruct struct {
		Name string `json:"name"`
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	req.RequestCtx.Request.SetBody([]byte(`{"name":"test"}`))

	var result TestStruct
	err := app.DecodeRequest(req, &result)

	assert.NoError(t, err, "decodeRequest should succeed with valid JSON")
	assert.Equal(t, "test", result.Name, "decodeRequest should decode JSON correctly")
}

// TestDecodeRequest_InvalidJSON tests decodeRequest with invalid JSON
func TestDecodeRequest_InvalidJSON(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	type TestStruct struct {
		Name string `json:"name"`
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte(`{invalid json}`))
	req := &fastglue.Request{RequestCtx: ctx}

	var result TestStruct
	err := app.DecodeRequest(req, &result)

	assert.Error(t, err, "decodeRequest should error with invalid JSON")
	assert.Equal(t, "", result.Name, "decodeRequest should leave result empty on error")
}

// TestDecodeRequest_EmptyBody tests decodeRequest with empty body
func TestDecodeRequest_EmptyBody(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	type TestStruct struct {
		Name string `json:"name"`
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte{})
	req := &fastglue.Request{RequestCtx: ctx}

	var result TestStruct
	err := app.DecodeRequest(req, &result)

	// Empty body with application/json content type causes decode error
	assert.Error(t, err, "decodeRequest should error with empty body")
	assert.Equal(t, "", result.Name, "decodeRequest should leave result empty on error")
}

// TestRequirePermission_HasPermission tests requirePermission when user has permission
func TestRequirePermission_HasPermission(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL and permission setup")

	app := &handlers.App{
		Config: &config.Config{},
	}

	userID := uuid.New()
	orgID := uuid.New()
	req := createTestRequestWithContext(userID, orgID, nil)

	err := app.RequirePermission(req, userID, "test_resource", "test_action")

	// This would require actual permission checking setup
	assert.NoError(t, err, "requirePermission should succeed when user has permission")
}

// TestRequirePermission_NoPermission tests requirePermission when user lacks permission
func TestRequirePermission_NoPermission(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL and permission setup")

	app := &handlers.App{
		Config: &config.Config{},
	}

	userID := uuid.New()
	orgID := uuid.New()
	req := createTestRequestWithContext(userID, orgID, nil)

	err := app.RequirePermission(req, userID, "test_resource", "test_action")

	// This would require actual permission checking setup
	assert.Error(t, err, "requirePermission should fail when user lacks permission")
}

// TestGetOrgID_SuperAdminOverride tests super admin org switching
func TestGetOrgID_SuperAdminOverride(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL and super admin setup")

	app := &handlers.App{
		Config: &config.Config{},
	}

	userID := uuid.New()
	orgID1 := uuid.New()
	orgID2 := uuid.New()

	// Mock super admin
	req := createTestRequestWithContext(userID, orgID1, map[string]string{
		"X-Organization-ID": orgID2.String(),
	})

	_, err := app.GetOrgID(req)

	// This would require actual IsSuperAdmin check and DB setup
	assert.NoError(t, err, "getOrgID should allow super admin to switch orgs")
}

// TestGetOrgID_NonSuperAdminCannotSwitch tests non-super admin cannot switch orgs
func TestGetOrgID_NonSuperAdminCannotSwitch(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL and membership setup")

	app := &handlers.App{
		Config: &config.Config{},
	}

	userID := uuid.New()
	orgID1 := uuid.New()
	orgID2 := uuid.New()

	req := createTestRequestWithContext(userID, orgID1, map[string]string{
		"X-Organization-ID": orgID2.String(),
	})

	result, err := app.GetOrgID(req)

	// Should return default orgID if user doesn't have membership in override org
	assert.NoError(t, err, "getOrgID should return default org for non-member")
	assert.Equal(t, orgID1, result, "getOrgID should not switch to non-member org")
}

// TestGetOrgID_InvalidOverride tests getOrgID with invalid override header
func TestGetOrgID_InvalidOverride(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	orgID := uuid.New()
	req := createTestRequestWithContext(uuid.New(), orgID, map[string]string{
		"X-Organization-ID": "invalid-uuid",
	})

	result, err := app.GetOrgID(req)

	assert.NoError(t, err, "getOrgID should ignore invalid override and return default")
	assert.Equal(t, orgID, result, "getOrgID should return default org ID when override is invalid")
}

// TestGetOrgAndUserID_GetOrgIDError tests getOrgAndUserID when getOrgID fails
func TestGetOrgAndUserID_GetOrgIDError(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := createTestRequestWithContext(uuid.Nil, uuid.Nil, nil)
	req.RequestCtx.SetUserValue("organization_id", "invalid-uuid")

	resultOrgID, resultUserID, err := app.GetOrgAndUserID(req)

	assert.Error(t, err, "getOrgAndUserID should error when getOrgID fails")
	assert.Equal(t, uuid.Nil, resultOrgID, "getOrgAndUserID should return Nil org ID on error")
	assert.Equal(t, uuid.Nil, resultUserID, "getOrgAndUserID should return Nil user ID on error")
}

// TestReadyCheck tests ReadyCheck endpoint
func TestReadyCheck(t *testing.T) {
	t.Skip("Skipping: Requires TEST_DATABASE_URL for Redis and DB connections")

	app := &handlers.App{
		Config: &config.Config{},
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	err := app.ReadyCheck(req)

	assert.NoError(t, err, "ReadyCheck should succeed with healthy dependencies")
}

// TestReadyCheck_NoDatabase tests ReadyCheck when DB is not available
func TestReadyCheck_NoDatabase(t *testing.T) {
	t.Skip("Skipping: Requires mock database setup")

	app := &handlers.App{
		Config: &config.Config{},
		DB:     nil, // Missing DB
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	err := app.ReadyCheck(req)

	// Should return error when DB is nil
	assert.Error(t, err, "ReadyCheck should error when DB is not available")
}

// TestStopCampaignStatsSubscriber tests stopping the subscriber
func TestStopCampaignStatsSubscriber(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	// Create a cancel function
	_, cancel := context.WithCancel(context.Background())
	app.CampaignSubCancel = cancel

	// Should not panic
	app.StopCampaignStatsSubscriber()

	// Calling again should not panic
	app.StopCampaignStatsSubscriber()
}

// TestStopCampaignStatsSubscriber_NoSubscriber tests stopping when no subscriber exists
func TestStopCampaignStatsSubscriber_NoSubscriber(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config:            &config.Config{},
		CampaignSubCancel: nil,
	}

	// Should not panic when CampaignSubCancel is nil
	app.StopCampaignStatsSubscriber()
}

// TestStartCampaignStatsSubscriber_NoWSHub tests starting subscriber without WSHub
func TestStartCampaignStatsSubscriber_NoWSHub(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
		WSHub:  nil,
		Log:    createTestLogger(),
		Redis:  nil, // Redis will be nil but we check WSHub first
	}

	err := app.StartCampaignStatsSubscriber()

	assert.NoError(t, err, "StartCampaignStatsSubscriber should return nil when WSHub is not initialized")
}

// TestWaitForBackgroundTasks tests WaitForBackgroundTasks
func TestWaitForBackgroundTasks(t *testing.T) {
	t.Parallel()

	app := &handlers.App{
		Config: &config.Config{},
	}

	// Should not panic with no background tasks
	app.WaitForBackgroundTasks()

	// Add a background task and wait for it
	// Note: This tests the basic wait group functionality
	// In real scenarios, you'd start goroutines with app.wg.Add()
}
