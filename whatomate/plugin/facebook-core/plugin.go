package facebookcore

import "github.com/compnew2006/whatomate/internal/core"

// Plugin registers the "facebook-core" managed module — the foundational,
// always-present dependency for every other facebook-* gating module. Like the
// other gating modules it ships no routes and no schema; its only job is to
// anchor the module + license graph.
//
// Embedding core.GatingModule satisfies the no-op Init/Routes/Migrate parts of
// the Plugin interface; this type only declares its identity (Name) and its
// module catalog entry (Manifest).
type Plugin struct {
	core.GatingModule
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string { return "facebook-core" }

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
