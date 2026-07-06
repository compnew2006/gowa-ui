package facebookoauth

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
	return "facebook-oauth"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook OAuth",
		Version:        "1.0.0",
		SchemaVersion:  1,
		Dependencies:   []string{"facebook-accounts"},
		DefaultEnabled: true,
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

	g.GET("/api/facebook/oauth/init", gate(p.InitFacebookOAuth))
	g.GET("/api/facebook/oauth/callback", gate(p.CallbackFacebookOAuth))
	g.GET("/api/facebook/accounts/{id}/oauth/renew", gate(p.RenewFacebookOAuth))
}

func (p *Plugin) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&OAuthState{})
}
