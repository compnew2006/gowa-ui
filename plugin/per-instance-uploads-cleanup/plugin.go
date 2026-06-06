package perinstanceuploadscleanup

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
	db  *gorm.DB
	rdb *redis.Client
	log *slog.Logger
	srv *service
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "per-instance-uploads-cleanup"
}

func (p *Plugin) Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	p.app = app
	p.db = db
	p.rdb = rdb
	p.log = log
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
