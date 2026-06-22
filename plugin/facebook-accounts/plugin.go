package facebookaccounts

import (
	"log/slog"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type Plugin struct {
	app *handlers.App
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "facebook-accounts"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook Accounts",
		Version:        "1.0.0",
		SchemaVersion:  1,
		Dependencies:   []string{"facebook-core"},
		DefaultEnabled: true,
	}
}

// Resource constants for plugin-namespaced permissions. Kept unexported and
// local to this plugin; the catalog itself is declared in Permissions() below.
const (
	resourceFBAccountsPagesManage = "plugin.facebook.accounts"
	actionPagesManage             = "pages_manage"
)

// Permissions implements core.PermissionProvidingPlugin. These permissions are
// seeded at startup alongside models.DefaultPermissions() and are enforced via
// the existing app.HasPermission machinery — no parallel auth system.
//
// plugin.facebook.accounts:pages_manage gates the destructive page lifecycle
// (connect/disconnect/remove) on top of the existing accounts:write check. An
// admin can therefore grant page management to a user without granting full
// account write/delete. Existing accounts:write holders retain access because
// the admin role auto-includes every permission (see SystemRolePermissions).
func (p *Plugin) Permissions() []core.PluginPermission {
	return []core.PluginPermission{
		{
			Resource:    resourceFBAccountsPagesManage,
			Action:      actionPagesManage,
			Description: "Connect, disconnect, and remove Facebook pages on an account",
		},
	}
}

func (p *Plugin) Init(app *handlers.App, _ *gorm.DB, _ *redis.Client, _ *slog.Logger) error {
	p.app = app
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	gate := func(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
		return core.GateModule(p.Name(), handler)
	}

	g.GET("/api/facebook/accounts", gate(p.ListFBAccounts))
	g.POST("/api/facebook/accounts", gate(p.CreateFBAccount))
	g.GET("/api/facebook/accounts/{id}", gate(p.GetFBAccount))
	g.PUT("/api/facebook/accounts/{id}", gate(p.UpdateFBAccount))
	g.DELETE("/api/facebook/accounts/{id}", gate(p.DeleteFBAccount))
	g.POST("/api/facebook/accounts/{id}/pages/refresh", gate(p.RefreshFacebookAccountPages))
	g.POST("/api/facebook/accounts/{id}/pages/{page_id}/connect", gate(p.ConnectFacebookAccountPage))
	g.POST("/api/facebook/accounts/{id}/pages/{page_id}/disconnect", gate(p.DisconnectFacebookAccountPage))
	g.DELETE("/api/facebook/accounts/{id}/pages/{page_id}", gate(p.RemoveFacebookAccountPage))
	g.POST("/api/facebook/accounts/{id}/pages/{page_id}/feed", gate(p.PostFacebookPage))
	g.GET("/api/facebook/accounts/{id}/pages/{page_id}/insights", gate(p.GetFacebookPageInsights))
	g.POST("/api/facebook/accounts/{id}/pages/{page_id}/messages", gate(p.SendFacebookPageMessage))
}

func (p *Plugin) Migrate(_ *gorm.DB) error {
	return nil
}
