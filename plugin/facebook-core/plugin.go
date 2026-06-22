package facebookcore

import (
	"log/slog"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type Plugin struct{}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "facebook-core"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook Core",
		Version:        "1.0.0",
		SchemaVersion:  1,
		DefaultEnabled: true,
		Technical:      true,
	}
}

func (p *Plugin) Init(*handlers.App, *gorm.DB, *redis.Client, *slog.Logger) error {
	return nil
}

func (p *Plugin) Routes(*fastglue.Fastglue) {}

func (p *Plugin) Migrate(*gorm.DB) error {
	return nil
}
