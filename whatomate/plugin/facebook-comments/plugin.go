package facebookcomments

import (
	"log/slog"
	"time"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type Plugin struct {
	app  *handlers.App
	rdb  *redis.Client
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "facebook-comments"
}

func (p *Plugin) Manifest() core.ModuleManifest {
	return core.ModuleManifest{
		Key:            p.Name(),
		DisplayName:    "Facebook Comments",
		Version:        "1.0.0",
		SchemaVersion:  1,
		Dependencies:   []string{"facebook-accounts"},
		DefaultEnabled: true,
	}
}

func (p *Plugin) Init(app *handlers.App, _ *gorm.DB, rdb *redis.Client, _ *slog.Logger) error {
	p.app = app
	p.rdb = rdb
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	gate := func(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
		return core.GateModule(p.Name(), handler)
	}

	g.GET("/api/facebook/comments", gate(p.app.ListFacebookComments))
	g.GET("/api/facebook/comments/pages", gate(p.app.ListFacebookCommentPages))
	g.POST("/api/facebook/comments/sync", gate(p.app.SyncFacebookComments))
	g.GET("/api/facebook/comments/settings", gate(p.GetFacebookCommentSettings))
	g.PUT("/api/facebook/comments/settings", gate(p.UpdateFacebookCommentSettings))
	g.GET("/api/facebook/comments/pages/{page_id}/settings", gate(p.GetPageCommentSettings))
	g.PUT("/api/facebook/comments/pages/{page_id}/settings", gate(p.UpdatePageCommentSettings))
	g.POST("/api/facebook/comments/{id}/reply", gate(p.app.ReplyFacebookComment))
	g.PUT("/api/facebook/comments/{id}/status", gate(p.app.UpdateFacebookCommentStatus))
	g.GET("/api/facebook/comments/webhook", gate(p.app.VerifyFacebookCommentsWebhook))

	webhookHandler := p.app.ReceiveFacebookCommentsWebhook
	if p.app.Config != nil && p.app.Config.RateLimit.Enabled {
		rateLimit := p.app.Config.RateLimit
		webhookHandler = withRateLimit(webhookHandler, middleware.RateLimitOpts{
			Redis:      p.rdb,
			Log:        p.app.Log,
			Max:        rateLimit.WebhookMaxAttempts,
			Window:     time.Duration(rateLimit.WindowSeconds) * time.Second,
			KeyPrefix:  "fb_comments_webhook",
			TrustProxy: rateLimit.TrustProxy,
		})
	}
	g.POST("/api/facebook/comments/webhook", gate(webhookHandler))
}

func (p *Plugin) Migrate(_ *gorm.DB) error {
	return nil
}

func withRateLimit(handler fastglue.FastRequestHandler, opts middleware.RateLimitOpts) fastglue.FastRequestHandler {
	rateLimit := middleware.RateLimit(opts)
	return func(r *fastglue.Request) error {
		if rateLimit(r) == nil {
			return nil
		}
		return handler(r)
	}
}
