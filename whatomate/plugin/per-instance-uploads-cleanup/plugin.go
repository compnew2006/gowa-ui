package perinstanceuploadscleanup

import (
	"log/slog"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// Plugin owns the per-instance uploads-cleanup feature. It embeds core.PluginBase
// to satisfy the Init (stash app/db/rdb/log) and no-op Migrate/Routes parts of
// the Plugin interface; this type overrides Init (to also build the service),
// Routes (registers the feature's routes), and Migrate (runs the feature's
// schema). The runtime deps are reached via the promoted fields p.App / p.DB /
// p.RDB / p.Log.
type Plugin struct {
	core.PluginBase
	srv *service
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string { return "per-instance-uploads-cleanup" }

// Init overrides PluginBase.Init to also construct the service from the
// freshly-stashed DB + logger.
func (p *Plugin) Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	if err := p.PluginBase.Init(app, db, rdb, log); err != nil {
		return err
	}
	p.srv = newService(db, log)
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	g.GET("/api/instances/{id}/uploads-cleanup", p.handleGetRetention)
	g.PUT("/api/instances/{id}/uploads-cleanup", p.handlePutRetention)
	g.GET("/api/instances/{id}/uploads-cleanup/history", p.handleHistory)
	g.POST("/api/instances/{id}/uploads-cleanup/run", p.handleRun)
	g.GET("/api/org/uploads-cleanup/instances", p.handleOverview)
}

func (p *Plugin) Migrate(db *gorm.DB) error {
	if err := db.Exec(`
		UPDATE whatsapp_instances
		SET settings = settings || '{"uploads_cleanup":{"inherit":true}}'::jsonb
		WHERE settings->'uploads_cleanup' IS NULL
	`).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&InstanceUploadsCleanupAudit{}); err != nil {
		return err
	}
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_iuca_org_instance_created
		ON instance_uploads_cleanup_audits (organization_id, instance_id, created_at)
	`).Error; err != nil {
		return err
	}
	return nil
}
