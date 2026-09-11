package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// DeleteAccount must soft-delete the row BEFORE calling GOWA's logout. The
// old order (logout first) could kill the WhatsApp session and then fail the
// DB delete, leaving a live account row with a dead device. The stub GOWA
// observes the row state at logout time — this test fails on the old order.
func TestApp_DeleteAccount_DeletesRowBeforeGowaLogout(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	user := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	var logoutSeen atomic.Bool
	var rowLiveAtLogout atomic.Bool // true = old (buggy) order

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/devices/") && strings.HasSuffix(r.URL.Path, "/logout") {
			var row models.WhatsAppAccount
			if err := app.DB.Unscoped().Where("id = ?", account.ID).First(&row).Error; err == nil && !row.DeletedAt.Valid {
				rowLiveAtLogout.Store(true)
			}
			logoutSeen.Store(true)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"SUCCESS","message":"Success","results":null}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(stub.Close)

	app.WARegistry = whatsapp.NewRegistryWithFactory(
		app.Log,
		func(_ uuid.UUID, _ string) (string, string) { return "", "" },
		func(_, _, _ string) whatsapp.Provider { return gowa.New(stub.URL, "", "") },
	)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, user.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	require.NoError(t, app.DeleteAccount(req))
	require.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	assert.True(t, logoutSeen.Load(), "GOWA logout must still be attempted after the row is deleted")
	assert.False(t, rowLiveAtLogout.Load(), "the account row must already be soft-deleted when logout fires")

	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", account.ID).Count(&count)
	assert.Zero(t, count)
}
