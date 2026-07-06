package facebookaccounts

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestAccountHandlersRequireAuthentication(t *testing.T) {
	app := newRouteTestApp(t)
	plugin := &Plugin{}
	require.NoError(t, plugin.Init(app, app.DB, nil, slog.Default()))

	tests := []struct {
		name    string
		handler fastglue.FastRequestHandler
	}{
		{name: "list", handler: plugin.ListFBAccounts},
		{name: "create", handler: plugin.CreateFBAccount},
		{name: "get", handler: plugin.GetFBAccount},
		{name: "update", handler: plugin.UpdateFBAccount},
		{name: "delete", handler: plugin.DeleteFBAccount},
		{name: "refresh pages", handler: plugin.RefreshFacebookAccountPages},
		{name: "connect page", handler: plugin.ConnectFacebookAccountPage},
		{name: "disconnect page", handler: plugin.DisconnectFacebookAccountPage},
		{name: "remove page", handler: plugin.RemoveFacebookAccountPage},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testutil.NewRequest(t)
			require.NoError(t, test.handler(request))
			require.Equal(
				t,
				fasthttp.StatusUnauthorized,
				testutil.GetResponseStatusCode(request),
			)
		})
	}
}
