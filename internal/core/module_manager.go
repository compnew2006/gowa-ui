package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrModuleNotFound             = errors.New("managed module not found")
	ErrModuleHasEnabledDependents = errors.New("module has enabled dependents")
)

type ModuleCatalog struct {
	Key                    string   `gorm:"column:key;primaryKey;size:128" json:"key"`
	DisplayName            string   `gorm:"column:display_name;not null" json:"display_name"`
	CompiledVersion        string   `gorm:"column:compiled_version;not null" json:"compiled_version"`
	SchemaVersion          uint     `gorm:"column:schema_version;not null;default:0" json:"schema_version"`
	InstalledSchemaVersion uint     `gorm:"column:installed_schema_version;not null;default:0" json:"installed_schema_version"`
	Dependencies           []string `gorm:"column:dependencies;serializer:json" json:"dependencies"`
	DefaultEnabled         bool     `gorm:"column:default_enabled;not null;default:true" json:"default_enabled"`
	GlobalEnabled          bool     `gorm:"column:global_enabled;not null;default:true" json:"global_enabled"`
	Technical              bool     `gorm:"column:technical;not null;default:false" json:"technical"`
	LastError              string   `gorm:"column:last_error" json:"last_error,omitempty"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (ModuleCatalog) TableName() string { return "module_catalog" }

type ModuleSchemaVersion struct {
	ModuleKey string    `gorm:"column:module_key;primaryKey;size:128" json:"module_key"`
	Version   uint      `gorm:"column:version;not null;default:0" json:"version"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ModuleSchemaVersion) TableName() string { return "module_schema_versions" }

type OrganizationModule struct {
	OrganizationID uuid.UUID `gorm:"column:organization_id;type:uuid;primaryKey" json:"organization_id"`
	ModuleKey      string    `gorm:"column:module_key;size:128;primaryKey" json:"module_key"`
	Enabled        bool      `gorm:"column:enabled;not null" json:"enabled"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (OrganizationModule) TableName() string { return "organization_modules" }

type EffectiveModule struct {
	ModuleManifest
	GlobalEnabled         bool `json:"global_enabled"`
	OrganizationEnabled   bool `json:"organization_enabled"`
	EffectiveEnabled      bool `json:"effective_enabled"`
	InstalledSchemaVersion uint `json:"installed_schema_version"`
}

type ModuleManager struct {
	db        *gorm.DB
	manifests []ModuleManifest
	byKey     map[string]ModuleManifest
	mu        sync.RWMutex
}

func NewModuleManager(db *gorm.DB, manifests []ModuleManifest) *ModuleManager {
	copied := append([]ModuleManifest(nil), manifests...)
	byKey := make(map[string]ModuleManifest, len(copied))
	for _, manifest := range copied {
		manifest.Dependencies = append([]string(nil), manifest.Dependencies...)
		byKey[manifest.Key] = manifest
	}
	return &ModuleManager{db: db, manifests: copied, byKey: byKey}
}

func (m *ModuleManager) Migrate(ctx context.Context) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("module manager database is nil")
	}
	db := m.db.WithContext(ctx)
	if err := db.AutoMigrate(&ModuleCatalog{}, &OrganizationModule{}, &ModuleSchemaVersion{}); err != nil {
		return fmt.Errorf("migrate module state: %w", err)
	}
	return nil
}

func (m *ModuleManager) Sync(ctx context.Context) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("module manager database is nil")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, manifest := range m.manifests {
			row := ModuleCatalog{
				Key:                    manifest.Key,
				DisplayName:            manifest.DisplayName,
				CompiledVersion:        manifest.Version,
				SchemaVersion:          manifest.SchemaVersion,
				InstalledSchemaVersion: manifest.SchemaVersion,
				Dependencies:           append([]string(nil), manifest.Dependencies...),
				DefaultEnabled:         manifest.DefaultEnabled,
				GlobalEnabled:          manifest.DefaultEnabled,
				Technical:              manifest.Technical,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"display_name",
					"compiled_version",
					"schema_version",
					"installed_schema_version",
					"dependencies",
					"default_enabled",
					"technical",
					"updated_at",
				}),
			}).Create(&row).Error; err != nil {
				return fmt.Errorf("sync module %q: %w", manifest.Key, err)
			}

			schemaVersion := ModuleSchemaVersion{
				ModuleKey: manifest.Key,
				Version:   manifest.SchemaVersion,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "module_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"version", "updated_at"}),
			}).Create(&schemaVersion).Error; err != nil {
				return fmt.Errorf("sync module schema version %q: %w", manifest.Key, err)
			}
		}

		var organizationIDs []uuid.UUID
		if err := tx.Model(&models.Organization{}).Pluck("id", &organizationIDs).Error; err != nil {
			return fmt.Errorf("list organizations for module defaults: %w", err)
		}
		for _, organizationID := range organizationIDs {
			for _, manifest := range m.manifests {
				row := OrganizationModule{
					OrganizationID: organizationID,
					ModuleKey:      manifest.Key,
					Enabled:        manifest.DefaultEnabled,
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
					return fmt.Errorf("seed organization module %q: %w", manifest.Key, err)
				}
			}
		}
		return nil
	})
}

func (m *ModuleManager) ListEffective(ctx context.Context, organizationID uuid.UUID) ([]EffectiveModule, error) {
	catalog, overrides, err := m.loadState(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]EffectiveModule, 0, len(m.manifests))
	for _, manifest := range m.manifests {
		row, exists := catalog[manifest.Key]
		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrModuleNotFound, manifest.Key)
		}
		organizationEnabled := manifest.DefaultEnabled
		if override, ok := overrides[manifest.Key]; ok {
			organizationEnabled = override
		}
		result = append(result, EffectiveModule{
			ModuleManifest:         manifest,
			GlobalEnabled:          row.GlobalEnabled,
			OrganizationEnabled:    organizationEnabled,
			EffectiveEnabled:       m.effective(manifest.Key, catalog, overrides, make(map[string]bool)),
			InstalledSchemaVersion: row.InstalledSchemaVersion,
		})
	}
	return result, nil
}

func (m *ModuleManager) IsEnabled(ctx context.Context, organizationID uuid.UUID, key string) (bool, error) {
	if _, exists := m.byKey[key]; !exists {
		return false, fmt.Errorf("%w: %s", ErrModuleNotFound, key)
	}
	catalog, overrides, err := m.loadState(ctx, organizationID)
	if err != nil {
		return false, err
	}
	return m.effective(key, catalog, overrides, make(map[string]bool)), nil
}

func (m *ModuleManager) IsGloballyEnabled(ctx context.Context, key string) (bool, error) {
	manifest, exists := m.byKey[key]
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrModuleNotFound, key)
	}
	var row ModuleCatalog
	if err := m.db.WithContext(ctx).First(&row, "key = ?", key).Error; err != nil {
		return false, fmt.Errorf("load module %q: %w", key, err)
	}
	if !row.GlobalEnabled || row.InstalledSchemaVersion < manifest.SchemaVersion {
		return false, nil
	}
	for _, dependency := range manifest.Dependencies {
		enabled, err := m.IsGloballyEnabled(ctx, dependency)
		if err != nil || !enabled {
			return false, err
		}
	}
	return true, nil
}

func GateModule(key string, handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		manager := GetModuleManager()
		if manager == nil {
			return handler(r)
		}

		organizationID, hasOrganization := middleware.GetOrganizationID(r)
		var (
			enabled bool
			err     error
		)
		if hasOrganization {
			enabled, err = manager.IsEnabled(context.Background(), organizationID, key)
		} else {
			enabled, err = manager.IsGloballyEnabled(context.Background(), key)
		}
		if err != nil {
			return r.SendErrorEnvelope(
				fasthttp.StatusInternalServerError,
				"Failed to resolve module availability",
				nil,
				"",
			)
		}
		if !enabled {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Not found", nil, "")
		}
		return handler(r)
	}
}

func (m *ModuleManager) SetGlobalEnabled(ctx context.Context, key string, enabled bool) error {
	if _, exists := m.byKey[key]; !exists {
		return fmt.Errorf("%w: %s", ErrModuleNotFound, key)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !enabled {
			for dependentKey, manifest := range m.byKey {
				if slices.Contains(manifest.Dependencies, key) {
					var dependent ModuleCatalog
					if err := tx.First(&dependent, "key = ?", dependentKey).Error; err != nil {
						return err
					}
					if dependent.GlobalEnabled {
						return fmt.Errorf("%w: %s", ErrModuleHasEnabledDependents, dependentKey)
					}
				}
			}
			return tx.Model(&ModuleCatalog{}).Where("key = ?", key).Update("global_enabled", false).Error
		}
		return m.enableGlobal(tx, key, make(map[string]bool))
	})
}

func (m *ModuleManager) SetOrganizationEnabled(ctx context.Context, organizationID uuid.UUID, key string, enabled bool) error {
	if _, exists := m.byKey[key]; !exists {
		return fmt.Errorf("%w: %s", ErrModuleNotFound, key)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !enabled {
			for dependentKey, manifest := range m.byKey {
				if !slices.Contains(manifest.Dependencies, key) {
					continue
				}
				dependentEnabled, err := m.organizationSetting(tx, organizationID, dependentKey)
				if err != nil {
					return err
				}
				if dependentEnabled {
					return fmt.Errorf("%w: %s", ErrModuleHasEnabledDependents, dependentKey)
				}
			}
			return m.writeOrganizationSetting(tx, organizationID, key, false)
		}
		return m.enableOrganization(tx, organizationID, key, make(map[string]bool))
	})
}

func (m *ModuleManager) enableGlobal(tx *gorm.DB, key string, visited map[string]bool) error {
	if visited[key] {
		return nil
	}
	visited[key] = true
	for _, dependency := range m.byKey[key].Dependencies {
		if err := m.enableGlobal(tx, dependency, visited); err != nil {
			return err
		}
	}
	return tx.Model(&ModuleCatalog{}).Where("key = ?", key).Update("global_enabled", true).Error
}

func (m *ModuleManager) enableOrganization(tx *gorm.DB, organizationID uuid.UUID, key string, visited map[string]bool) error {
	if visited[key] {
		return nil
	}
	visited[key] = true
	for _, dependency := range m.byKey[key].Dependencies {
		if err := m.enableOrganization(tx, organizationID, dependency, visited); err != nil {
			return err
		}
	}
	return m.writeOrganizationSetting(tx, organizationID, key, true)
}

func (m *ModuleManager) writeOrganizationSetting(tx *gorm.DB, organizationID uuid.UUID, key string, enabled bool) error {
	row := OrganizationModule{OrganizationID: organizationID, ModuleKey: key, Enabled: enabled}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "module_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"}),
	}).Create(&row).Error
}

func (m *ModuleManager) organizationSetting(tx *gorm.DB, organizationID uuid.UUID, key string) (bool, error) {
	var row OrganizationModule
	err := tx.First(&row, "organization_id = ? AND module_key = ?", organizationID, key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m.byKey[key].DefaultEnabled, nil
	}
	if err != nil {
		return false, err
	}
	return row.Enabled, nil
}

func (m *ModuleManager) loadState(ctx context.Context, organizationID uuid.UUID) (map[string]ModuleCatalog, map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var catalogRows []ModuleCatalog
	if err := m.db.WithContext(ctx).Find(&catalogRows).Error; err != nil {
		return nil, nil, fmt.Errorf("load module catalog: %w", err)
	}
	catalog := make(map[string]ModuleCatalog, len(catalogRows))
	for _, row := range catalogRows {
		catalog[row.Key] = row
	}

	var organizationRows []OrganizationModule
	if err := m.db.WithContext(ctx).Where("organization_id = ?", organizationID).Find(&organizationRows).Error; err != nil {
		return nil, nil, fmt.Errorf("load organization modules: %w", err)
	}
	overrides := make(map[string]bool, len(organizationRows))
	for _, row := range organizationRows {
		overrides[row.ModuleKey] = row.Enabled
	}
	return catalog, overrides, nil
}

func (m *ModuleManager) effective(key string, catalog map[string]ModuleCatalog, overrides map[string]bool, visiting map[string]bool) bool {
	if visiting[key] {
		return false
	}
	manifest, exists := m.byKey[key]
	if !exists {
		return false
	}
	row, exists := catalog[key]
	if !exists || !row.GlobalEnabled || row.InstalledSchemaVersion < manifest.SchemaVersion {
		return false
	}
	organizationEnabled := manifest.DefaultEnabled
	if override, ok := overrides[key]; ok {
		organizationEnabled = override
	}
	if !organizationEnabled {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	for _, dependency := range manifest.Dependencies {
		if !m.effective(dependency, catalog, overrides, visiting) {
			return false
		}
	}
	return true
}
