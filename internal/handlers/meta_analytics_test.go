package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/fastglue"
)

func metaAnalyticsTestApp(t *testing.T, metaHandler http.HandlerFunc) (*handlers.App, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	db := testutil.SetupTestDB(t)
	log := testutil.NopLogger()

	server := httptest.NewServer(metaHandler)
	t.Cleanup(server.Close)

	app := &handlers.App{
		Config:   &config.Config{App: config.AppConfig{EncryptionKey: testutil.TestEncryptionKey}},
		DB:       db,
		Log:      log,
		WhatsApp: whatsapp.NewWithBaseURL(log, server.URL),
	}
	if rdb := testutil.SetupTestRedis(t); rdb != nil {
		app.Redis = rdb
	}

	org := testutil.CreateTestOrganization(t, db)
	acct := testutil.CreateTestWhatsAppAccount(t, db, org.ID)
	role := testutil.CreateAdminRole(t, db, org.ID)
	user := testutil.CreateTestUser(t, db, org.ID, testutil.WithRoleID(&role.ID))

	return app, org.ID, user.ID, acct.ID
}

func metaAnalyticsRequest(t *testing.T, orgID, userID uuid.UUID, queryParams map[string]string) *fastglue.Request {
	t.Helper()

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, orgID, userID)
	for k, v := range queryParams {
		testutil.SetQueryParam(req, k, v)
	}
	return req
}

func TestGetMetaAnalytics_CacheMiss_CallsAPIAndStoresResult(t *testing.T) {
	called := 0
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"analytics": []map[string]interface{}{
				{"start": 1234567890, "end": 1234567900, "data_points": []map[string]interface{}{}},
			},
		})
	}))

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     time.Now().Add(-24*time.Hour).Format("2006-01-02"),
		"end":       time.Now().Format("2006-01-02"),
		"granularity":    "DAY",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, 1, called)
}

func TestGetMetaAnalytics_CacheHit_ReturnsCachedWithoutAPICall(t *testing.T) {
	called := 0
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"analytics": []map[string]interface{}{
				{"start": 1234567890, "end": 1234567900},
			},
		})
	}))

	start := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	end := time.Now().Format("2006-01-02")
	params := map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     start,
		"end":       end,
		"granularity":    "DAY",
	}

	req1 := metaAnalyticsRequest(t, orgID, userID, params)
	require.NoError(t, app.GetMetaAnalytics(req1))
	assert.Equal(t, 1, called)

	req2 := metaAnalyticsRequest(t, orgID, userID, params)
	require.NoError(t, app.GetMetaAnalytics(req2))
	assert.Equal(t, 1, called, "API should not be called again on cache hit")
}

func TestGetMetaAnalytics_InvalidAnalyticsType_Returns400(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "invalid_type",
		"start":     "2026-01-01",
		"end":       "2026-01-02",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetMetaAnalytics_MissingAnalyticsType_Returns400(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"start": "2026-01-01",
		"end":   "2026-01-02",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetMetaAnalytics_MissingDates_Returns400(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetMetaAnalytics_EndBeforeStart_Returns400(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"start":     "2026-01-10",
		"end":       "2026-01-01",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetMetaAnalytics_InvalidGranularity_Returns400(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"start":     "2026-01-01",
		"end":       "2026-01-02",
		"granularity":    "INVALID",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestGetMetaAnalytics_GranularityAutoAdjust(t *testing.T) {
	requestedFields := ""
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedFields = r.URL.Query().Get("fields")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"analytics": []map[string]interface{}{},
		})
	}))

	start := time.Now().Add(-15 * 24 * time.Hour).Format("2006-01-02")
	end := time.Now().Format("2006-01-02")

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     start,
		"end":       end,
		"granularity":    "HALF_HOUR",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Contains(t, requestedFields, "granularity(DAY)", "HALF_HOUR should be auto-adjusted to DAY for >7 day range")
}

func TestGetMetaAnalytics_MetaAPIError_ReturnsGracefully(t *testing.T) {
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":{"message":"internal error","type":"OAuthException"}}`)
	}))

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     time.Now().Add(-24*time.Hour).Format("2006-01-02"),
		"end":       time.Now().Format("2006-01-02"),
		"granularity":    "DAY",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	statusCode := testutil.GetResponseStatusCode(req)
	assert.Equal(t, http.StatusOK, statusCode, "handler should return 200 with partial results on API failure")
}

func TestGetMetaAnalytics_NoAccounts_ReturnsEmptyResult(t *testing.T) {
	app, orgID, userID, _ := metaAnalyticsTestApp(t, nil)

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"start":     time.Now().Add(-24*time.Hour).Format("2006-01-02"),
		"end":       time.Now().Format("2006-01-02"),
		"granularity":    "DAY",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	statusCode := testutil.GetResponseStatusCode(req)
	assert.True(t, statusCode == http.StatusOK || statusCode == http.StatusNotFound,
		"should handle no accounts gracefully, got %d", statusCode)
}

func TestGetMetaAnalytics_SpecificAccountID(t *testing.T) {
	called := 0
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"analytics": []map[string]interface{}{
				{"start": 1234567890, "end": 1234567900},
			},
		})
	}))

	req := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     time.Now().Add(-24*time.Hour).Format("2006-01-02"),
		"end":       time.Now().Format("2006-01-02"),
		"granularity":    "DAY",
	})

	err := app.GetMetaAnalytics(req)
	require.NoError(t, err)
	assert.Equal(t, 1, called)
}

func TestGetMetaAnalytics_CacheKeyIsolationBetweenTypes(t *testing.T) {
	apiCalls := 0
	app, orgID, userID, accountID := metaAnalyticsTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"analytics": []map[string]interface{}{},
		})
	}))

	start := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	end := time.Now().Format("2006-01-02")

	req1 := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "analytics",
		"account_id":     accountID.String(),
		"start":     start,
		"end":       end,
		"granularity":    "DAY",
	})
	require.NoError(t, app.GetMetaAnalytics(req1))

	req2 := metaAnalyticsRequest(t, orgID, userID, map[string]string{
		"analytics_type": "pricing_analytics",
		"account_id":     accountID.String(),
		"start":     start,
		"end":       end,
		"granularity":    "DAY",
	})
	require.NoError(t, app.GetMetaAnalytics(req2))

	assert.Equal(t, 2, apiCalls, "different analytics types should use separate cache keys")
}
