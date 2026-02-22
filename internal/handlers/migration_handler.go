package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/pkg/migration"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// TriggerMigration starts the data migration from WhatsAppAccounts to WhatsAppInstances.
// POST /api/admin/migrate
// Optional JSON body: { "organization_id": "<uuid>" }
// If organization_id is provided, only that org is migrated.
// If omitted, all orgs are migrated.
func (a *App) TriggerMigration(r *fastglue.Request) error {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Only super admins can run migrations.
	if !a.IsSuperAdmin(userID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only super admins can run migrations", nil, "")
	}

	// Parse optional body.
	var req struct {
		OrganizationID string `json:"organization_id"`
	}
	// Ignore decode errors — body is optional.
	_ = r.Decode(&req, "json")

	svc := migration.NewService(a.DB, a.Log)

	if svc.IsRunning() {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "A migration is already in progress", nil, "")
	}

	// Start async migration.
	if req.OrganizationID != "" {
		orgID, parseErr := uuid.Parse(req.OrganizationID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid organization_id", nil, "")
		}

		// Run single-org migration in background.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			result, migErr := svc.MigrateOrg(ctx, orgID)
			if migErr != nil {
				a.Log.Error("Migration failed for org",
					"organization_id", orgID,
					"error", migErr,
				)
				return
			}
			a.Log.Info("Migration complete for org",
				"organization_id", orgID,
				"contacts_migrated", result.ContactsMigrated,
				"messages_migrated", result.MessagesMigrated,
			)
		}()

		return r.SendEnvelope(map[string]interface{}{
			"status":  "started",
			"scope":   "single_org",
			"org_id":  req.OrganizationID,
			"message": "Migration started in background. Use GET /api/admin/migrate/status to check progress.",
		})
	}

	// Run all-orgs migration in background.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		result, migErr := svc.MigrateAll(ctx)
		if migErr != nil {
			a.Log.Error("Migration failed", "error", migErr)
			return
		}
		a.Log.Info("Migration complete",
			"total_orgs", result.TotalOrgs,
			"success_orgs", result.SuccessOrgs,
			"failed_orgs", result.FailedOrgs,
		)
	}()

	return r.SendEnvelope(map[string]interface{}{
		"status":  "started",
		"scope":   "all_orgs",
		"message": "Migration started in background. Use GET /api/admin/migrate/status to check progress.",
	})
}

// GetMigrationStatus returns the current migration progress.
// GET /api/admin/migrate/status
func (a *App) GetMigrationStatus(r *fastglue.Request) error {
	_, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if !a.IsSuperAdmin(userID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Only super admins can view migration status", nil, "")
	}

	// Since the migration service is created per-request (stateless query),
	// we return the current state of accounts vs. instances.
	type orgMigrationStatus struct {
		OrgID             uuid.UUID `json:"organization_id"`
		OrgName           string    `json:"organization_name"`
		AccountsCount     int64     `json:"accounts_count"`
		InstancesCount    int64     `json:"instances_count"`
		ContactsTotal     int64     `json:"contacts_total"`
		ContactsMigrated  int64     `json:"contacts_migrated"`
		ContactsPending   int64     `json:"contacts_pending"`
		MessagesTotal     int64     `json:"messages_total"`
		MessagesMigrated  int64     `json:"messages_migrated"`
		MessagesPending   int64     `json:"messages_pending"`
		MigrationComplete bool      `json:"migration_complete"`
	}

	// Query accounts grouped by org.
	type orgAccountCount struct {
		OrganizationID uuid.UUID `gorm:"column:organization_id"`
		Count          int64     `gorm:"column:count"`
	}
	var accountCounts []orgAccountCount
	if err := a.DB.Table("whatsapp_accounts").
		Select("organization_id, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("organization_id").
		Find(&accountCounts).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to query accounts", nil, "")
	}

	var results []orgMigrationStatus
	for _, ac := range accountCounts {
		status := orgMigrationStatus{
			OrgID:         ac.OrganizationID,
			AccountsCount: ac.Count,
		}

		// Get org name.
		a.DB.Table("organizations").
			Select("name").
			Where("id = ?", ac.OrganizationID).
			Row().
			Scan(&status.OrgName) //nolint:errcheck

		// Count instances for this org.
		a.DB.Model(&struct{}{}).Table("whatsapp_instances").
			Where("organization_id = ? AND deleted_at IS NULL", ac.OrganizationID).
			Count(&status.InstancesCount)

		// Count contacts total and migrated.
		a.DB.Table("contacts").
			Where("organization_id = ? AND deleted_at IS NULL", ac.OrganizationID).
			Count(&status.ContactsTotal)
		a.DB.Table("contacts").
			Where("organization_id = ? AND deleted_at IS NULL AND instance_id IS NOT NULL", ac.OrganizationID).
			Count(&status.ContactsMigrated)
		status.ContactsPending = status.ContactsTotal - status.ContactsMigrated

		// Count messages total and migrated.
		a.DB.Table("messages").
			Where("organization_id = ? AND deleted_at IS NULL", ac.OrganizationID).
			Count(&status.MessagesTotal)
		a.DB.Table("messages").
			Where("organization_id = ? AND deleted_at IS NULL AND instance_id IS NOT NULL", ac.OrganizationID).
			Count(&status.MessagesMigrated)
		status.MessagesPending = status.MessagesTotal - status.MessagesMigrated

		status.MigrationComplete = status.ContactsPending == 0 && status.MessagesPending == 0 && status.InstancesCount > 0

		results = append(results, status)
	}

	// Calculate overall.
	overallComplete := true
	for _, s := range results {
		if !s.MigrationComplete {
			overallComplete = false
			break
		}
	}

	return r.SendEnvelope(map[string]interface{}{
		"overall_complete": overallComplete,
		"organizations":    results,
	})
}
