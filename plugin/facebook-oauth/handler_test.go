package facebookoauth

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestOAuthHandlersPreserveEntryBoundaries(t *testing.T) {
	app := newRouteTestApp(t)
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, app.DB, nil, slog.Default()))

	t.Run("init requires authentication", func(t *testing.T) {
		request := testutil.NewGETRequest(t)
		require.NoError(t, plugin.InitFacebookOAuth(request))
		require.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(request))
	})

	t.Run("renew requires authentication", func(t *testing.T) {
		request := testutil.NewGETRequest(t)
		request.RequestCtx.SetUserValue("id", uuid.NewString())
		require.NoError(t, plugin.RenewFacebookOAuth(request))
		require.Equal(t, fasthttp.StatusUnauthorized, testutil.GetResponseStatusCode(request))
	})

	t.Run("callback redirects invalid parameters", func(t *testing.T) {
		request := testutil.NewGETRequest(t)
		require.NoError(t, plugin.CallbackFacebookOAuth(request))
		require.Equal(t, fasthttp.StatusFound, testutil.GetResponseStatusCode(request))
		require.Contains(t, string(request.RequestCtx.Response.Header.Peek("Location")), "facebook_oauth=error")
	})
}
