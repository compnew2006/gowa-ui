package handlers_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestApp_RunUploadsCleanupNow_Success(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	uploadRoot := t.TempDir()
	app.Config.Storage.LocalPath = uploadRoot

	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "cleanup-executor", []string{
		"settings.uploads_cleanup:execute",
	})
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("cleanup-executor")),
		testutil.WithRoleID(&role.ID),
	)

	org.Settings = models.JSONB{
		"uploads_cleanup_retention_days": 5,
		"uploads_cleanup_schedule_hour":  3,
	}
	require.NoError(t, app.DB.Save(org).Error)

	expiredPath := filepath.Join(uploadRoot, "documents", "expired.pdf")
	require.NoError(t, os.MkdirAll(filepath.Dir(expiredPath), 0o755))
	require.NoError(t, os.WriteFile(expiredPath, []byte("fixture"), 0o600))
	oldTime := time.Now().UTC().Add(-6 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(expiredPath, oldTime, oldTime))

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.RunUploadsCleanupNow(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	_, statErr := os.Stat(expiredPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestApp_RunUploadsCleanupNow_ForbiddenWithoutExecutePermission(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "cleanup-reader", []string{
		"settings.uploads_cleanup:read",
	})
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("cleanup-reader")),
		testutil.WithRoleID(&role.ID),
	)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.RunUploadsCleanupNow(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))
}

func TestApp_RunUploadsCleanupNow_RequiresPositiveRetention(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	role := testutil.CreateTestRoleWithKeys(t, app.DB, org.ID, "cleanup-executor", []string{
		"settings.uploads_cleanup:execute",
	})
	user := testutil.CreateTestUser(t, app.DB, org.ID,
		testutil.WithEmail(testutil.UniqueEmail("cleanup-disabled")),
		testutil.WithRoleID(&role.ID),
	)

	org.Settings = models.JSONB{
		"uploads_cleanup_retention_days": 0,
	}
	require.NoError(t, app.DB.Save(org).Error)

	req := testutil.NewJSONRequest(t, map[string]any{})
	testutil.SetAuthContext(req, org.ID, user.ID)

	err := app.RunUploadsCleanupNow(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}
