package core

import (
	"log/slog"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type Plugin interface {
	Name() string
	Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error
	Routes(g *fastglue.Fastglue)
	Migrate(db *gorm.DB) error
}

var plugins []Plugin

func RegisterPlugin(p Plugin) {
	plugins = append(plugins, p)
}

func GetPlugins() []Plugin {
	return plugins
}

func InitPlugins(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	for _, p := range plugins {
		if err := p.Init(app, db, rdb, log); err != nil {
			return err
		}
	}
	return nil
}

func RegisterPluginRoutes(g *fastglue.Fastglue) {
	for _, p := range plugins {
		p.Routes(g)
	}
}

func RunPluginMigrations(db *gorm.DB) error {
	for _, p := range plugins {
		if err := p.Migrate(db); err != nil {
			return err
		}
	}
	return nil
}
