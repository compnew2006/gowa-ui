package perinstanceuploadscleanup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/compnew2006/whatomate/internal/handlers"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

func (p *Plugin) getOrgAndUserID(r *fastglue.Request) (orgID, userID uuid.UUID, err error) {
	orgID, err = tenant.ResolveOrganizationID(r, p.db)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	userIDVal := r.RequestCtx.UserValue("user_id")
	if userIDVal == nil {
		return uuid.Nil, uuid.Nil, errors.New("user_id not found in context")
	}
	parsed, err := uuid.Parse(fmt.Sprintf("%v", userIDVal))
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("user_id is not a valid UUID")
	}
	return orgID, parsed, nil
}

// hasPermission checks RBAC via custom_role_permissions + checks users.is_super_admin
// as the core App.HasPermission does, but without requiring the full App cache infra.
func (p *Plugin) hasPermission(userID uuid.UUID, resource, action string, orgID uuid.UUID) bool {
	var isSuperAdmin bool
	p.db.Raw(`SELECT is_super_admin FROM users WHERE id = ? AND deleted_at IS NULL`, userID).Scan(&isSuperAdmin)
	if isSuperAdmin {
		return true
	}
	var count int64
	p.db.Raw(`
		SELECT COUNT(*)
		FROM custom_role_permissions crp
		JOIN custom_roles cr ON cr.id = crp.custom_role_id AND cr.organization_id = ? AND cr.deleted_at IS NULL
		JOIN user_organizations uo ON uo.role_id = cr.id AND uo.user_id = ? AND uo.organization_id = ? AND uo.deleted_at IS NULL
		WHERE crp.resource = ? AND crp.action = ?`,
		orgID, userID, orgID, resource, action,
	).Scan(&count)
	return count > 0
}

func (p *Plugin) canAccess(userID, orgID uuid.UUID) bool {
	return p.hasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionRead, orgID) ||
		p.hasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionWrite, orgID) ||
		p.hasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionExecute, orgID)
}

func (p *Plugin) canWrite(userID, orgID uuid.UUID) bool {
	return p.hasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionWrite, orgID)
}

func (p *Plugin) canExecute(userID, orgID uuid.UUID) bool {
	return p.hasPermission(userID, models.ResourceSettingsUploadsCleanup, models.ActionExecute, orgID)
}

var errResponseSent = errors.New("response already sent")

// requireInstanceAccess resolves auth, checks permission, and parses the instance ID from the URL.
func (p *Plugin) requireInstanceAccess(r *fastglue.Request, permCheck func(uuid.UUID, uuid.UUID) bool) (orgID, userID, instanceID uuid.UUID, err error) {
	orgID, userID, err = p.getOrgAndUserID(r)
	if err != nil {
		r.SendErrorEnvelope(http.StatusUnauthorized, "Unauthorized", nil, "")
		return uuid.Nil, uuid.Nil, uuid.Nil, errResponseSent
	}
	if !permCheck(userID, orgID) {
		r.SendErrorEnvelope(http.StatusForbidden, "Insufficient permissions", nil, "")
		return uuid.Nil, uuid.Nil, uuid.Nil, errResponseSent
	}
	instanceIDStr := r.RequestCtx.UserValue("id")
	instanceID, err = uuid.Parse(fmt.Sprintf("%v", instanceIDStr))
	if err != nil {
		r.SendErrorEnvelope(http.StatusBadRequest, "Invalid instance ID", nil, "INVALID_ID")
		return uuid.Nil, uuid.Nil, uuid.Nil, errResponseSent
	}
	return orgID, userID, instanceID, nil
}

// parsePagination extracts limit and offset from query args with defaults.
func parsePagination(r *fastglue.Request, defaultLimit int) (limit, offset int, err error) {
	args := r.RequestCtx.URI().QueryArgs()
	limit = defaultLimit
	if v := string(args.Peek("limit")); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			limit = n
		}
	}
	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	offset = 0
	if v := string(args.Peek("offset")); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			offset = n
		}
	}
	return limit, offset, nil
}

func parseUploadsCleanup(settings models.JSONB) (inherit bool, retentionDays *int, lastRunDate *string) {
	uc, ok := settings["uploads_cleanup"].(map[string]interface{})
	if !ok {
		return true, nil, nil
	}
	if v, ok := uc["inherit"].(bool); ok {
		inherit = v
	} else {
		inherit = true
	}
	if v, ok := uc["retention_days"].(float64); ok {
		d := int(v)
		retentionDays = &d
	}
	if v, ok := uc["last_run_date"].(string); ok {
		lastRunDate = &v
	}
	return
}

func getInstanceSettings(scopedDB *gorm.DB, instanceID uuid.UUID) (models.JSONB, error) {
	var instance models.WhatsAppInstance
	if err := scopedDB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		return nil, err
	}
	return instance.Settings, nil
}

func respondRetention(r *fastglue.Request, instanceID uuid.UUID, settings models.JSONB, days int, source string) error {
	inherit, retentionDays, lastRunDate := parseUploadsCleanup(settings)
	var effectiveDays *int
	if source != "disabled" {
		effectiveDays = &days
	}
	return r.SendEnvelope(map[string]interface{}{
		"instance_id":              instanceID,
		"inherit":                  inherit,
		"retention_days":           retentionDays,
		"effective_retention_days": effectiveDays,
		"effective_source":         source,
		"last_run_date":            lastRunDate,
	})
}

func (p *Plugin) handleGetRetention(r *fastglue.Request) error {
	orgID, _, instanceID, err := p.requireInstanceAccess(r, p.canAccess)
	if err != nil {
		return err
	}

	scopedDB := tenant.ScopedDB(p.db, orgID)
	settings, err := getInstanceSettings(scopedDB, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(http.StatusNotFound, "Instance not found", nil, "INSTANCE_NOT_FOUND")
		}
		p.log.Error("handleGetRetention: failed to get instance", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}

	days, source, err := p.srv.ResolveEffectiveRetention(r.RequestCtx, orgID, instanceID, time.Now())
	if err != nil {
		p.log.Error("handleGetRetention: failed to resolve retention", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}
	return respondRetention(r, instanceID, settings, days, source)
}

type retentionUpdateRequest struct {
	Inherit       bool    `json:"inherit"`
	RetentionDays *int    `json:"retention_days"`
	Reason        *string `json:"reason"`
}

func (p *Plugin) handlePutRetention(r *fastglue.Request) error {
	orgID, userID, instanceID, err := p.requireInstanceAccess(r, p.canWrite)
	if err != nil {
		return err
	}

	var req retentionUpdateRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, "Invalid request body", nil, "INVALID_BODY")
	}

	if err := ValidateRetentionUpdate(req.Inherit, req.RetentionDays); err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, err.Error(), nil, "")
	}

	scopedDB := tenant.ScopedDB(p.db, orgID)
	settings, err := getInstanceSettings(scopedDB, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(http.StatusNotFound, "Instance not found", nil, "INSTANCE_NOT_FOUND")
		}
		p.log.Error("handlePutRetention: failed to get instance", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}

	if err := p.updateInstanceRetention(scopedDB, instanceID, settings, req); err != nil {
		p.log.Error("handlePutRetention: failed to update settings", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}

	oldInherit, oldRetentionDays, _ := parseUploadsCleanup(settings)
	var actorEmail *string
	var user models.User
	if p.db.Where("id = ?", userID).First(&user).Error == nil {
		actorEmail = &user.Email
	}

	if err := p.srv.WriteAuditRow(r.RequestCtx, orgID, instanceID, &userID, actorEmail,
		RetentionSnapshot{Inherit: oldInherit, RetentionDays: oldRetentionDays},
		RetentionSnapshot{Inherit: req.Inherit, RetentionDays: req.RetentionDays},
		req.Reason,
	); err != nil {
		p.log.Error("handlePutRetention: failed to write audit row", "err", err, "instance_id", instanceID)
	}

	newSettings, err := getInstanceSettings(scopedDB, instanceID)
	if err != nil {
		p.log.Error("handlePutRetention: failed to re-read settings", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}
	days, source, err := p.srv.ResolveEffectiveRetention(r.RequestCtx, orgID, instanceID, time.Now())
	if err != nil {
		p.log.Error("handlePutRetention: failed to resolve retention", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}
	return respondRetention(r, instanceID, newSettings, days, source)
}

func (p *Plugin) updateInstanceRetention(scopedDB *gorm.DB, instanceID uuid.UUID, settings models.JSONB, req retentionUpdateRequest) error {
	uc, ok := settings["uploads_cleanup"].(map[string]interface{})
	if !ok {
		uc = make(map[string]interface{})
	}
	uc["inherit"] = req.Inherit
	if req.RetentionDays != nil {
		uc["retention_days"] = *req.RetentionDays
	}
	settings["uploads_cleanup"] = uc

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	return scopedDB.Model(&models.WhatsAppInstance{}).Where("id = ?", instanceID).
		Update("settings", string(settingsJSON)).Error
}

func (p *Plugin) handleHistory(r *fastglue.Request) error {
	orgID, _, instanceID, err := p.requireInstanceAccess(r, p.canAccess)
	if err != nil {
		return err
	}

	limit, offset, err := parsePagination(r, 5)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, err.Error(), nil, "invalid_limit")
	}

	scopedDB := tenant.ScopedDB(p.db, orgID)
	var total int64
	scopedDB.Model(&InstanceUploadsCleanupAudit{}).Where("instance_id = ?", instanceID).Count(&total)

	var entries []InstanceUploadsCleanupAudit
	scopedDB.Where("instance_id = ?", instanceID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&entries)

	return r.SendEnvelope(map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (p *Plugin) handleRun(r *fastglue.Request) error {
	orgID, _, instanceID, err := p.requireInstanceAccess(r, p.canExecute)
	if err != nil {
		return err
	}

	release, ok := p.srv.tryAcquireInstanceRun()
	if !ok {
		return r.SendErrorEnvelope(http.StatusConflict, "Another cleanup run is already in progress", nil, "")
	}
	defer release()

	days, source, err := p.srv.ResolveEffectiveRetention(r.RequestCtx, orgID, instanceID, time.Now())
	if err != nil {
		p.log.Error("handleRun: failed to resolve retention", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Internal error", nil, "")
	}
	if source == "disabled" {
		return r.SendErrorEnvelope(http.StatusBadRequest, "Uploads cleanup is disabled for this instance. Set a retention value or configure a workspace default first.", nil, "uploads_cleanup_disabled")
	}

	deletedFiles, err := handlers.RunManualCleanupForInstance(r.RequestCtx, p.app, orgID, instanceID, &days)
	if err != nil {
		p.log.Error("handleRun: cleanup failed", "err", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Cleanup failed", nil, "")
	}

	instanceName, err := p.recordLastRunDate(tenant.ScopedDB(p.db, orgID), instanceID)
	if err != nil {
		p.log.Error("handleRun: failed to record last run date", "err", err, "instance_id", instanceID)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Failed to record last run date", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"instance_id":    instanceID,
		"instance_name":  instanceName,
		"deleted_files":  deletedFiles,
		"retention_used": days,
	})
}

// recordLastRunDate writes last_run_date into the instance settings and returns the instance name.
// Uses a single DB read + write instead of the previous double-read pattern.
func (p *Plugin) recordLastRunDate(scopedDB *gorm.DB, instanceID uuid.UUID) (string, error) {
	var instance models.WhatsAppInstance
	if err := scopedDB.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		return "", fmt.Errorf("find instance: %w", err)
	}

	uc, ok := instance.Settings["uploads_cleanup"].(map[string]interface{})
	if !ok {
		uc = make(map[string]interface{})
	}
	uc["last_run_date"] = time.Now().UTC().Format("2006-01-02")
	instance.Settings["uploads_cleanup"] = uc

	settingsJSON, err := json.Marshal(instance.Settings)
	if err != nil {
		return instance.Name, fmt.Errorf("marshal settings: %w", err)
	}
	if err := scopedDB.Model(&instance).Update("settings", string(settingsJSON)).Error; err != nil {
		return instance.Name, fmt.Errorf("update settings: %w", err)
	}
	return instance.Name, nil
}

func (p *Plugin) handleOverview(r *fastglue.Request) error {
	orgID, userID, err := p.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if !p.canAccess(userID, orgID) {
		return r.SendErrorEnvelope(http.StatusForbidden, "Insufficient permissions", nil, "")
	}

	limit, offset, err := parsePagination(r, 20)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, err.Error(), nil, "invalid_limit")
	}

	args := r.RequestCtx.URI().QueryArgs()
	q := string(args.Peek("q"))
	sourceFilter := string(args.Peek("source"))

	scopedDB := tenant.ScopedDB(p.db, orgID)
	query := scopedDB.Model(&models.WhatsAppInstance{})
	if q != "" {
		query = query.Where("name ILIKE ?", "%"+q+"%")
	}

	var total int64
	query.Count(&total)

	var instances []models.WhatsAppInstance
	query.Limit(limit).Offset(offset).Find(&instances)

	type overviewRow struct {
		InstanceID             uuid.UUID `json:"instance_id"`
		InstanceName           string    `json:"instance_name"`
		EffectiveRetentionDays *int      `json:"effective_retention_days"`
		EffectiveSource        string    `json:"effective_source"`
		LastRunDate            *string   `json:"last_run_date"`
	}

	rows := make([]overviewRow, 0, len(instances))
	for _, inst := range instances {
		days, source, err := p.srv.ResolveEffectiveRetention(r.RequestCtx, orgID, inst.ID, time.Now())
		if err != nil {
			p.log.Error("handleOverview: failed to resolve retention", "err", err, "instance_id", inst.ID)
			continue
		}

		if sourceFilter != "" && sourceFilter != "all" && source != sourceFilter {
			continue
		}

		_, _, lrd := parseUploadsCleanup(inst.Settings)
		row := overviewRow{
			InstanceID:      inst.ID,
			InstanceName:    inst.Name,
			EffectiveSource: source,
			LastRunDate:     lrd,
		}
		if source != "disabled" {
			row.EffectiveRetentionDays = &days
		}
		rows = append(rows, row)
	}

	return r.SendEnvelope(map[string]interface{}{
		"items":  rows,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
