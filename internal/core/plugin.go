package core

import (
	"context"
	"fmt"
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


// PluginBase is an optional embed that satisfies the no-op parts of the Plugin
// interface — Init (stashing the runtime deps), Migrate (no schema work), and
// Routes (no routes). Plugins that need custom behavior simply override the
// relevant method on the embedding type; Go's method resolution gives the
// outer type's method precedence over the embedded one.
//
// Intended for plugins whose Init only stashes app/db/rdb/log for later use by
// handlers (campaign-interactive, per-instance-uploads-cleanup, …). It removes
// the repeated stash-and-store boilerplate without touching the Plugin contract.
//
// Name() is intentionally NOT provided — every plugin must declare its own name
// so the identifier lives next to the package, not in a shared base.
type PluginBase struct {
	App *handlers.App
	DB  *gorm.DB
	RDB *redis.Client
	Log *slog.Logger
}

// Init stashes the runtime dependencies. It is the Init contract verbatim, so
// embedding PluginBase satisfies Plugin.Init for plugins that only need the
// stash.
func (b *PluginBase) Init(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	b.App = app
	b.DB = db
	b.RDB = rdb
	b.Log = log
	return nil
}

// Migrate is a no-op. Plugins with their own schema override this method.
func (b *PluginBase) Migrate(*gorm.DB) error { return nil }

// Routes is a no-op. Plugins that register HTTP routes override this method.
func (b *PluginBase) Routes(*fastglue.Fastglue) {}

// GatingModule is an optional embed for managed plugins that exist only to gate
// a feature behind the module + license system — they ship no backend routes
// and run no schema migrations; their whole job is to declare a ModuleManifest
// so administrators can show/hide the feature per organization and per license
// tier via the module-management plugin.
//
// Embedding GatingModule satisfies the no-op Routes/Migrate parts of Plugin.
// The embedding type still provides Name() and Manifest() (the latter making
// it a ManagedPlugin). Together with a constructor pattern (see NewGatingModule
// below) this collapses ~48 lines of near-identical gating-plugin boilerplate
// per plugin down to a single struct literal.
type GatingModule struct {
	PluginBase
}

// NewGatingModule returns a GatingModule whose PluginBase is ready to embed.
// It exists so gating plugins read as a one-liner: the struct value itself
// carries no state worth constructing, but the named constructor documents
// intent at the call site.
func NewGatingModule() GatingModule { return GatingModule{} }

// ModuleManifest describes a compiled plugin that participates in database-controlled enablement.
type ModuleManifest struct {
	Key            string   `json:"key"`
	DisplayName    string   `json:"display_name"`
	Version        string   `json:"version"`
	SchemaVersion  uint     `json:"schema_version"`
	Dependencies   []string `json:"dependencies,omitempty"`
	DefaultEnabled bool     `json:"default_enabled"`
	Technical      bool     `json:"technical"`
}

// ManagedPlugin is optional. Plugins that do not implement it retain the legacy lifecycle.
type ManagedPlugin interface {
	Plugin
	Manifest() ModuleManifest
}

// PluginPermission is a resource:action pair declared by a plugin. The
// Resource follows the dotted plugin namespace convention
// ("plugin.<plugin>.<feature>", e.g. "plugin.facebook.accounts") and the
// Action is one of the standard models.Action* verbs (e.g. "write"). The
// combined "resource:action" string is what handlers check via
// app.HasPermission and what gets seeded into the permissions table.
//
// PluginPermission keeps core decoupled from internal/models to avoid an
// import cycle; the conversion to models.Permission happens at seed time in
// internal/database.
type PluginPermission struct {
	Resource    string
	Action      string
	Description string
}

// PermissionProvidingPlugin is optional. Plugins implementing it contribute
// fine-grained, plugin-namespaced permissions to the RBAC catalog at startup.
// These permissions are seeded alongside models.DefaultPermissions() and flow
// through the existing role_permissions / HasPermission machinery — no
// parallel authorization system is introduced.
type PermissionProvidingPlugin interface {
	Plugin
	Permissions() []PluginPermission
}

var (
	plugins            []Plugin
	initializedPlugins []Plugin
	moduleManifests    []ModuleManifest
	moduleManager      *ModuleManager
	// pluginPermissions lives in permission_seeder.go alongside the seeding
	// logic that consumes it.
)

func RegisterPlugin(p Plugin) {
	plugins = append(plugins, p)
}

func GetPlugins() []Plugin {
	return plugins
}

// ResolvePlugins validates managed manifests and returns a deterministic lifecycle order.
// Legacy plugins remain first and keep their original registration order.
func ResolvePlugins(registered []Plugin) ([]Plugin, []ModuleManifest, error) {
	legacy := make([]Plugin, 0, len(registered))
	managedByKey := make(map[string]Plugin)
	manifestByKey := make(map[string]ModuleManifest)
	registrationOrder := make([]string, 0, len(registered))

	for _, plugin := range registered {
		managed, ok := plugin.(ManagedPlugin)
		if !ok {
			legacy = append(legacy, plugin)
			continue
		}

		manifest := managed.Manifest()
		if manifest.Key == "" {
			return nil, nil, fmt.Errorf("module key is required for plugin %q", plugin.Name())
		}
		if _, exists := managedByKey[manifest.Key]; exists {
			return nil, nil, fmt.Errorf("duplicate managed module key %q", manifest.Key)
		}
		managedByKey[manifest.Key] = plugin
		manifestByKey[manifest.Key] = manifest
		registrationOrder = append(registrationOrder, manifest.Key)
	}

	for key, manifest := range manifestByKey {
		for _, dependency := range manifest.Dependencies {
			if _, exists := managedByKey[dependency]; !exists {
				return nil, nil, fmt.Errorf("managed module %q has missing dependency %q", key, dependency)
			}
		}
	}

	state := make(map[string]uint8, len(managedByKey))
	sortedKeys := make([]string, 0, len(managedByKey))
	var visit func(string) error
	visit = func(key string) error {
		switch state[key] {
		case 1:
			return fmt.Errorf("managed module dependency cycle includes %q", key)
		case 2:
			return nil
		}
		state[key] = 1
		for _, dependency := range manifestByKey[key].Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[key] = 2
		sortedKeys = append(sortedKeys, key)
		return nil
	}
	for _, key := range registrationOrder {
		if err := visit(key); err != nil {
			return nil, nil, err
		}
	}

	resolved := append([]Plugin(nil), legacy...)
	manifests := make([]ModuleManifest, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		resolved = append(resolved, managedByKey[key])
		manifests = append(manifests, manifestByKey[key])
	}

	// Collect plugin-namespaced permissions from every plugin that opts in.
	// Both legacy and managed plugins may provide permissions. Duplicate
	// resource:action pairs across plugins are collapsed by the database's
	// unique index at seed time, so no de-duplication is needed here.
	collected := make([]PluginPermission, 0)
	for _, plugin := range resolved {
		provider, ok := plugin.(PermissionProvidingPlugin)
		if !ok {
			continue
		}
		collected = append(collected, provider.Permissions()...)
	}
	pluginPermissions = collected

	return resolved, manifests, nil
}

func GetModuleManifests() []ModuleManifest {
	return append([]ModuleManifest(nil), moduleManifests...)
}

func GetModuleManager() *ModuleManager {
	return moduleManager
}

func InitPlugins(app *handlers.App, db *gorm.DB, rdb *redis.Client, log *slog.Logger) error {
	resolved, manifests, err := ResolvePlugins(plugins)
	if err != nil {
		return err
	}

	moduleManifests = manifests
	moduleManager = NewModuleManager(db, manifests)
	for _, p := range resolved {
		if err := p.Init(app, db, rdb, log); err != nil {
			return fmt.Errorf("initialize plugin %q: %w", p.Name(), err)
		}
	}
	initializedPlugins = resolved
	return nil
}

func RegisterPluginRoutes(g *fastglue.Fastglue) {
	registered := initializedPlugins
	if len(registered) == 0 {
		registered = plugins
	}
	for _, p := range registered {
		p.Routes(g)
	}
}

func RunPluginMigrations(db *gorm.DB) error {
	if moduleManager != nil {
		if err := moduleManager.Migrate(context.Background()); err != nil {
			return err
		}
	}

	registered := initializedPlugins
	if len(registered) == 0 {
		registered = plugins
	}
	for _, p := range registered {
		if err := p.Migrate(db); err != nil {
			return fmt.Errorf("migrate plugin %q: %w", p.Name(), err)
		}
	}
	return SyncManagedModules(context.Background())
}

func SyncManagedModules(ctx context.Context) error {
	if moduleManager == nil || len(moduleManifests) == 0 {
		return nil
	}
	if err := moduleManager.Migrate(ctx); err != nil {
		return err
	}
	return moduleManager.Sync(ctx)
}


