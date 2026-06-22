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
	g.GET("/api/organizations/{id}/modules", p.listOrganization)
	g.PUT("/api/organizations/{id}/modules/{key}", p.updateOrganization)
}

func (p *Plugin) Migrate(*gorm.DB) error {
	return nil
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
	if err := manager.SetGlobalEnabled(context.Background(), key, enabled); err != nil {
		return sendModuleUpdateError(r, err)
	}
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
	if err := manager.SetOrganizationEnabled(context.Background(), organizationID, key, enabled); err != nil {
		return sendModuleUpdateError(r, err)
	}
	return r.SendEnvelope(map[string]any{
		"organization_id": organizationID,
		"key":             key,
		"enabled":         enabled,
	})
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
