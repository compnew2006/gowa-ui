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

var (
	plugins            []Plugin
	initializedPlugins []Plugin
	moduleManifests    []ModuleManifest
	moduleManager      *ModuleManager
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
