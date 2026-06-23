package facebookautoshare

import (
	"log/slog"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Plugin registers the "facebook-auto-share" managed module. It is a
// gating-only plugin: the feature UI is a placeholder view, and this module
// exists so administrators can show/hide the tool per organization (and per
// license tier) through the module-management system. Backend routes will be
// added here when the feature is implemented.
type Plugin struct{}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "facebook-auto-share"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook Auto Share",
		Version:        "1.0.0",
		SchemaVersion:  1,
		Dependencies:   []string{"facebook-core"},
		DefaultEnabled: true,
	}
}

func (p *Plugin) Init(*handlers.App, *gorm.DB, *redis.Client, *slog.Logger) error {
	return nil
}

func (p *Plugin) Routes(*fastglue.Fastglue) {}

func (p *Plugin) Migrate(*gorm.DB) error {
	return nil
}
