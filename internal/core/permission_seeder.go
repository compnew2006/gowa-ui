package core

import (
	"context"
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// pluginPermissions holds the plugin-namespaced permissions collected by
// ResolvePlugins from every plugin implementing PermissionProvidingPlugin.
// Kept as package state (not on PluginHost) so ResolvePlugins — a pure
// function over its input — can populate it and SyncPluginPermissions can
// read it, matching the existing test seam in plugin_test.go which reads
// and mutates this variable directly.
var pluginPermissions []PluginPermission

// PluginPermissions returns a copy of every plugin-namespaced permission
// contributed via PermissionProvidingPlugin. Returns nil when no plugin
// provides permissions or before ResolvePlugins has run.
func PluginPermissions() []PluginPermission {
	return append([]PluginPermission(nil), pluginPermissions...)
}

// SyncPluginPermissions idempotently seeds every plugin-namespaced permission
// contributed via PermissionProvidingPlugin into the permissions table. It is
// the plugin-owned counterpart to database.SeedPermissionsAndRoles (which owns
// the core 35-resource catalog); keeping it here avoids a database→core import
// cycle and matches the "core imports models" precedent already established by
// module_manager.go.
//
// The unique index idx_permission_resource_action makes this safe to call on
// every startup: existing rows are left untouched, missing rows are inserted.
func SyncPluginPermissions(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	permissions := PluginPermissions()
	if len(permissions) == 0 {
		return nil
	}
	for _, perm := range permissions {
		if perm.Resource == "" || perm.Action == "" {
			return fmt.Errorf("plugin permission has empty resource or action: %+v", perm)
		}
		var existing models.Permission
		err := db.WithContext(ctx).
			Where("resource = ? AND action = ?", perm.Resource, perm.Action).
			First(&existing).Error
		if err == nil {
			// Already seeded; leave untouched so descriptions stay stable.
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("lookup plugin permission %s:%s: %w", perm.Resource, perm.Action, err)
		}
		row := models.Permission{
			BaseModel:   models.BaseModel{ID: uuid.New()},
			Resource:    perm.Resource,
			Action:      perm.Action,
			Description: perm.Description,
		}
		if err := db.WithContext(ctx).Create(&row).Error; err != nil {
			return fmt.Errorf("create plugin permission %s:%s: %w", perm.Resource, perm.Action, err)
		}
	}
	return nil
}
