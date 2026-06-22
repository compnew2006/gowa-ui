package facebookpagesearch

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
	return "facebook-page-search"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook Page Search",
		Version:        "1.0.0",
		SchemaVersion:  1,
		Dependencies:   []string{"facebook-core"},
		DefaultEnabled: true,
	}
}

func (p *Plugin) Init(app *handlers.App, _ *gorm.DB, _ *redis.Client, _ *slog.Logger) error {
	p.app = app
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	g.GET("/api/facebook/page-search", core.GateModule(p.Name(), p.SearchFBPages))
}

func (p *Plugin) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&PageSearch{})
}
