package campaigninteractive

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
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "campaign-interactive"
}

func (p *Plugin) Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	p.app = app
	p.db = db
	p.rdb = rdb
	p.log = log
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	g.GET("/api/campaigns/{id}/poll/votes", p.handleGetPollVotes)
}

func (p *Plugin) Migrate(db *gorm.DB) error {
	return nil
}
