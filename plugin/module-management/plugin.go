package modulemanagement

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/compnew2006/whatomate/internal/core"
	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type Plugin struct {
	app     *handlers.App
	manager *core.ModuleManager
}

type updateModuleRequest struct {
	Enabled *bool `json:"enabled"`
}

func init() {
	core.RegisterPlugin(&Plugin{})
}

func (p *Plugin) Name() string {
	return "module-management"
}

func (p *Plugin) Init(app *handlers.App, _ *gorm.DB, _ *redis.Client, _ *slog.Logger) error {
	p.app = app
	p.manager = core.GetModuleManager()
	return nil
}

func (p *Plugin) Routes(g *fastglue.Fastglue) {
	g.GET("/api/modules/effective", p.listEffective)
	g.GET("/api/admin/modules", p.listGlobal)
	g.PUT("/api/admin/modules/{key}", p.updateGlobal)
	g.GET("/api/admin/modules/events", p.listGlobalEvents)
	g.GET("/api/organizations/{id}/modules", p.listOrganization)
	g.PUT("/api/organizations/{id}/modules/{key}", p.updateOrganization)
	g.GET("/api/organizations/{id}/modules/events", p.listOrganizationEvents)
}

func (p *Plugin) Migrate(db *gorm.DB) error {
	return MigrateModuleEvents(db)
}

func (p *Plugin) listEffective(r *fastglue.Request) error {
	organizationID, ok := middleware.GetOrganizationID(r)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	manager, ok := p.requireManager(r)
	if !ok {
		return nil
	}
	modules, err := manager.ListEffective(context.Background(), organizationID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load modules", nil, "")
	}
	return r.SendEnvelope(modules)
}

func (p *Plugin) listGlobal(r *fastglue.Request) error {
	if !middleware.IsSuperAdmin(r) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Super admin access required", nil, "")
	}
	manager, ok := p.requireManager(r)
	if !ok {
		return nil
	}
	modules, err := manager.ListEffective(context.Background(), uuid.Nil)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load modules", nil, "")
	}
	return r.SendEnvelope(modules)
}

func (p *Plugin) updateGlobal(r *fastglue.Request) error {
	if !middleware.IsSuperAdmin(r) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Super admin access required", nil, "")
	}
	key, ok := pathString(r, "key")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Module key is required", nil, "key")
	}
	enabled, ok := decodeEnabled(r)
	if !ok {
		return nil
	}
	manager, ok := p.requireManager(r)
	if !ok {
		return nil
	}
	if !p.licenseAllows(key) {
		p.recordEvent(r, nil, moduleScopeGlobal, key, ModuleActionLicenseDeny, &enabled, "module not licensed for current tier")
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Module is not licensed for this deployment tier", map[string]any{
			"error": "module_not_licensed",
			"key":   key,
		}, "")
	}
	if err := manager.SetGlobalEnabled(context.Background(), key, enabled); err != nil {
		if errors.Is(err, core.ErrModuleHasEnabledDependents) {
			p.recordEvent(r, nil, moduleScopeGlobal, key, ModuleActionConflict, &enabled, "module has enabled dependents")
		}
		return sendModuleUpdateError(r, err)
	}
	p.recordEvent(r, nil, moduleScopeGlobal, key, enabledToAction(enabled), &enabled, "")
	return r.SendEnvelope(map[string]any{"key": key, "enabled": enabled})
}

func (p *Plugin) listOrganization(r *fastglue.Request) error {
	organizationID, ok := pathUUID(r, "id")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid organization ID", nil, "id")
	}
	if !p.authorizeOrganization(r, organizationID, models.ActionRead) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}
	manager, ok := p.requireManager(r)
	if !ok {
		return nil
	}
	modules, err := manager.ListEffective(context.Background(), organizationID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load modules", nil, "")
	}
	return r.SendEnvelope(modules)
}

func (p *Plugin) updateOrganization(r *fastglue.Request) error {
	organizationID, ok := pathUUID(r, "id")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid organization ID", nil, "id")
	}
	if !p.authorizeOrganization(r, organizationID, models.ActionWrite) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}
	key, ok := pathString(r, "key")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Module key is required", nil, "key")
	}
	enabled, ok := decodeEnabled(r)
	if !ok {
		return nil
	}
	manager, ok := p.requireManager(r)
	if !ok {
		return nil
	}
	if !p.licenseAllows(key) {
		p.recordEvent(r, &organizationID, moduleScopeOrganization, key, ModuleActionLicenseDeny, &enabled, "module not licensed for current tier")
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Module is not licensed for this deployment tier", map[string]any{
			"error": "module_not_licensed",
			"key":   key,
		}, "")
	}
	if err := manager.SetOrganizationEnabled(context.Background(), organizationID, key, enabled); err != nil {
		if errors.Is(err, core.ErrModuleHasEnabledDependents) {
			p.recordEvent(r, &organizationID, moduleScopeOrganization, key, ModuleActionConflict, &enabled, "module has enabled dependents")
		}
		return sendModuleUpdateError(r, err)
	}
	p.recordEvent(r, &organizationID, moduleScopeOrganization, key, enabledToAction(enabled), &enabled, "")
	return r.SendEnvelope(map[string]any{
		"organization_id": organizationID,
		"key":             key,
		"enabled":         enabled,
	})
}

// listGlobalEvents returns the global module audit trail. Super-admin only.
func (p *Plugin) listGlobalEvents(r *fastglue.Request) error {
	if !middleware.IsSuperAdmin(r) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Super admin access required", nil, "")
	}
	if p.app == nil || p.app.DB == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Database is unavailable", nil, "")
	}
	var events []ModuleEvent
	if err := p.app.DB.Order("created_at DESC").Limit(200).Find(&events).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load module events", nil, "")
	}
	return r.SendEnvelope(events)
}

// listOrganizationEvents returns the per-organization module audit trail.
// Requires organizations:read on the target org (or super-admin).
func (p *Plugin) listOrganizationEvents(r *fastglue.Request) error {
	organizationID, ok := pathUUID(r, "id")
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid organization ID", nil, "id")
	}
	if !p.authorizeOrganization(r, organizationID, models.ActionRead) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}
	if p.app == nil || p.app.DB == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Database is unavailable", nil, "")
	}
	var events []ModuleEvent
	if err := p.app.DB.Where("organization_id = ?", organizationID).Order("created_at DESC").Limit(200).Find(&events).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load module events", nil, "")
	}
	return r.SendEnvelope(events)
}

func (p *Plugin) authorizeOrganization(r *fastglue.Request, organizationID uuid.UUID, action string) bool {
	if middleware.IsSuperAdmin(r) {
		return true
	}
	currentOrganizationID, ok := middleware.GetOrganizationID(r)
	if !ok || currentOrganizationID != organizationID || p.app == nil {
		return false
	}
	userID, ok := middleware.GetUserID(r)
	return ok && p.app.HasPermission(userID, models.ResourceOrganizations, action, organizationID)
}

func (p *Plugin) requireManager(r *fastglue.Request) (*core.ModuleManager, bool) {
	if p.manager == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Module manager is unavailable", nil, "")
		return nil, false
	}
	return p.manager, true
}

// licenseAllows resolves the active license tier and reports whether the module
// is entitled. When no license is active (License == nil or not Enabled), the
// tier is "" and LicenseAllowsModule returns false — but we treat the absence
// of licensing as "no entitlement restriction" to preserve backwards
// compatibility for deployments that run unlicensed.
func (p *Plugin) licenseAllows(key string) bool {
	if p.app == nil || p.app.License == nil {
		return true
	}
	state := p.app.License.CurrentState()
	if !state.Enabled {
		return true
	}
	return core.LicenseAllowsModule(state.Tier, key)
}

// recordEvent captures a module audit row. Failures are logged but never
// propagated so an audit-write issue cannot block a user-facing operation.
func (p *Plugin) recordEvent(r *fastglue.Request, organizationID *uuid.UUID, scope, key, action string, enabled *bool, reason string) {
	if p.app == nil || p.app.DB == nil {
		return
	}
	var actorUserID *uuid.UUID
	if userID, ok := middleware.GetUserID(r); ok {
		copied := userID
		actorUserID = &copied
	}
	actorEmail, _ := r.RequestCtx.UserValue(middleware.ContextKeyEmail).(string)
	event := ModuleEvent{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Scope:          scope,
		ModuleKey:      key,
		Action:         action,
		Enabled:        enabled,
		ActorUserID:    actorUserID,
		ActorEmail:     actorEmail,
		Reason:         reason,
		Details:        models.JSONB{},
	}
	if err := p.app.DB.Create(&event).Error; err != nil {
		p.app.Log.Error("Failed to write module audit event", "error", err, "module_key", key, "action", action)
	}
}

func enabledToAction(enabled bool) string {
	if enabled {
		return ModuleActionEnable
	}
	return ModuleActionDisable
}

func decodeEnabled(r *fastglue.Request) (bool, bool) {
	var request updateModuleRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &request); err != nil || request.Enabled == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusBadRequest, "enabled is required", nil, "enabled")
		return false, false
	}
	return *request.Enabled, true
}

func pathString(r *fastglue.Request, key string) (string, bool) {
	value, ok := r.RequestCtx.UserValue(key).(string)
	return value, ok && value != ""
}

func pathUUID(r *fastglue.Request, key string) (uuid.UUID, bool) {
	value, ok := pathString(r, key)
	if !ok {
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(value)
	return parsed, err == nil
}

func sendModuleUpdateError(r *fastglue.Request, err error) error {
	switch {
	case errors.Is(err, core.ErrModuleNotFound):
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Module not found", nil, "")
	case errors.Is(err, core.ErrModuleHasEnabledDependents):
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Module has enabled dependents", nil, "")
	default:
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update module", nil, "")
	}
}
