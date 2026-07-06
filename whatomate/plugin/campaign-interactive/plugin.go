package campaigninteractive

import (
	"github.com/compnew2006/whatomate/internal/core"
	"github.com/zerodha/fastglue"
)

// Plugin owns the campaign poll-vote analytics endpoint. It embeds
// core.PluginBase to satisfy the Init (stash app/db/rdb/log) and no-op Migrate
// parts of the Plugin interface; this type overrides Routes to register the
// feature's single route. The runtime deps are reached via the promoted fields
// p.App / p.DB / p.RDB / p.Log.
type Plugin struct {
	core.PluginBase
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string { return "campaign-interactive" }

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	g.GET("/api/campaigns/{id}/poll/votes", p.handleGetPollVotes)
}
