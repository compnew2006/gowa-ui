package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/database"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fixture struct {
	Path       string `json:"path"`
	Heading    string `json:"heading"`
	ResourceID string `json:"resource_id,omitempty"`
}

var validRunID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: e2e-fixtures seed <scenario> <run-id> [config] | cleanup <run-id> [config] [related-resource-id...]")
	}

	command := args[0]
	var runID, scenario, configPath string
	switch command {
	case "seed":
		if len(args) < 3 {
			return errors.New("seed requires a scenario and run ID")
		}
		scenario, runID = args[1], args[2]
		configPath = argument(args, 3, "config.toml")
	case "cleanup":
		runID = args[1]
		configPath = argument(args, 2, "config.toml")
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	if !validRunID.MatchString(runID) {
		return errors.New("run ID must be a UUID")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load E2E fixture config: %w", err)
	}
	if cfg.App.Environment != "development" && cfg.App.Environment != "test" {
		return fmt.Errorf("E2E fixtures are disabled for environment %q", cfg.App.Environment)
	}

	db, err := database.NewPostgres(&cfg.Database, false)
	if err != nil {
		return fmt.Errorf("connect to E2E fixture database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open E2E fixture database handle: %w", err)
	}
	defer sqlDB.Close()

	switch command {
	case "seed":
		data, err := seed(db, scenario, runID)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(data)
	case "cleanup":
		var relatedResourceIDs []uuid.UUID
		if len(args) > 3 {
			for _, rawID := range args[3:] {
				id, err := uuid.Parse(rawID)
				if err != nil {
					return fmt.Errorf("invalid related resource ID %q", rawID)
				}
				relatedResourceIDs = append(relatedResourceIDs, id)
			}
		}
		return cleanup(db, runID, relatedResourceIDs)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func argument(args []string, index int, fallback string) string {
	if len(args) > index && strings.TrimSpace(args[index]) != "" {
		return args[index]
	}
	return fallback
}

func seed(db *gorm.DB, scenario, runID string) (fixture, error) {
	adminEmail := os.Getenv("E2E_USER_SUPER_ADMIN_USERNAME")
	if adminEmail == "" {
		adminEmail = os.Getenv("E2E_SUPER_ADMIN_EMAIL")
	}
	if adminEmail == "" {
		adminEmail = "admin@admin.com"
	}
	var admin models.User
	if err := db.Where("email = ? AND is_super_admin = ?", adminEmail, true).First(&admin).Error; err != nil {
		return fixture{}, fmt.Errorf("find E2E super-admin %q: %w", adminEmail, err)
	}
	orgID := admin.OrganizationID
	if orgID == uuid.Nil {
		var membership models.UserOrganization
		if err := db.Where("user_id = ? AND is_default = ?", admin.ID, true).First(&membership).Error; err != nil {
			return fixture{}, fmt.Errorf("find default organization for E2E super-admin: %w", err)
		}
		orgID = membership.OrganizationID
	}
	if orgID == uuid.Nil {
		return fixture{}, errors.New("E2E super-admin has no organization")
	}

	base := "E2E fixture " + runID
	result := fixture{}
	err := db.Transaction(func(tx *gorm.DB) error {
		createAccount := func() (*models.WhatsAppAccount, error) {
			account := &models.WhatsAppAccount{
				OrganizationID: orgID,
				Name:           base + " account",
				// No base URL, device, or webhook secret: opening or deleting this
				// fixture cannot call a GOWA server.
				Status:      "active",
				CreatedByID: &admin.ID,
				UpdatedByID: &admin.ID,
			}
			if err := tx.Create(account).Error; err != nil {
				return nil, err
			}
			return account, nil
		}
		createTemplate := func(accountName, suffix, body string) (*models.Template, error) {
			template := &models.Template{
				OrganizationID:  orgID,
				WhatsAppAccount: accountName,
				Name:            base + " " + suffix,
				DisplayName:     base + " " + suffix,
				Language:        "en_US",
				Category:        "UTILITY",
				BodyContent:     body,
				CreatedByID:     &admin.ID,
				UpdatedByID:     &admin.ID,
			}
			if err := tx.Create(template).Error; err != nil {
				return nil, err
			}
			return template, nil
		}

		switch scenario {
		case "accounts-list", "account-detail":
			account, err := createAccount()
			if err != nil {
				return err
			}
			if scenario == "accounts-list" {
				result = fixture{Path: "/settings/accounts", Heading: "WhatsApp Accounts", ResourceID: account.ID.String()}
			} else {
				result = fixture{Path: "/settings/accounts/" + account.ID.String(), Heading: account.Name, ResourceID: account.ID.String()}
			}
		case "campaign-form":
			account, err := createAccount()
			if err != nil {
				return err
			}
			if _, err := createTemplate(account.Name, "template-one", "Hello {{1}}, your appointment is confirmed."); err != nil {
				return err
			}
			if _, err := createTemplate(account.Name, "template-two", "Your E2E order {{1}} is ready."); err != nil {
				return err
			}
			result = fixture{Path: "/campaigns/new", Heading: "New Campaign"}
		case "campaign-detail", "campaigns-list":
			account, err := createAccount()
			if err != nil {
				return err
			}
			template, err := createTemplate(account.Name, "campaign-template", "Hello {{1}}, this is an E2E draft.")
			if err != nil {
				return err
			}
			campaign := &models.BulkMessageCampaign{
				OrganizationID:  orgID,
				WhatsAppAccount: account.Name,
				Name:            base + " campaign",
				TemplateID:      template.ID,
				Status:          models.CampaignStatusDraft,
				TotalRecipients: 1,
				CreatedBy:       admin.ID,
				UpdatedByID:     &admin.ID,
			}
			if err := tx.Create(campaign).Error; err != nil {
				return err
			}
			for index, phone := range []string{"15555550101"} {
				recipient := &models.BulkMessageRecipient{
					CampaignID:    campaign.ID,
					PhoneNumber:   phone,
					RecipientName: fmt.Sprintf("E2E Recipient %d", index+1),
					TemplateParams: models.JSONB{
						"1": fmt.Sprintf("Fixture %d", index+1),
					},
					Status: models.MessageStatusPending,
				}
				if err := tx.Create(recipient).Error; err != nil {
					return err
				}
			}
			if scenario == "campaigns-list" {
				result = fixture{Path: "/campaigns", Heading: "Campaigns", ResourceID: campaign.ID.String()}
			} else {
				result = fixture{Path: "/campaigns/" + campaign.ID.String(), Heading: campaign.Name, ResourceID: campaign.ID.String()}
			}
		case "canned-response-detail":
			response := &models.CannedResponse{
				OrganizationID: orgID,
				Name:           base + " response",
				Shortcut:       "e2e" + strings.ReplaceAll(runID[:8], "-", ""),
				Content:        "This is an isolated E2E canned response.",
				Category:       "Support",
				IsActive:       true,
				Buttons: models.JSONBArray{
					map[string]any{"id": "e2e-reply", "title": "Contact support", "type": "reply"},
				},
				CreatedByID: admin.ID,
			}
			if err := tx.Create(response).Error; err != nil {
				return err
			}
			result = fixture{Path: "/settings/canned-responses/" + response.ID.String(), Heading: response.Name, ResourceID: response.ID.String()}
		case "canned-responses-list":
			response := &models.CannedResponse{
				OrganizationID: orgID,
				Name:           base + " response",
				Shortcut:       "e2e" + strings.ReplaceAll(runID[:8], "-", ""),
				Content:        "This is an isolated E2E canned response.",
				Category:       "Support",
				IsActive:       true,
				CreatedByID:    admin.ID,
			}
			if err := tx.Create(response).Error; err != nil {
				return err
			}
			result = fixture{Path: "/settings/canned-responses", Heading: "Canned Responses", ResourceID: response.ID.String()}
		case "contacts-list", "contact-detail":
			phoneID, ok := new(big.Int).SetString(strings.ReplaceAll(runID, "-", ""), 16)
			if !ok {
				return errors.New("invalid E2E fixture run ID")
			}
			phoneSuffix := new(big.Int).Mod(phoneID, big.NewInt(100_000_000_000))
			contact := &models.Contact{
				OrganizationID: orgID,
				PhoneNumber:    "1555" + fmt.Sprintf("%011d", phoneSuffix.Int64()),
				ProfileName:    base + " contact",
				IsRead:         true,
			}
			if err := tx.Create(contact).Error; err != nil {
				return err
			}
			if scenario == "contacts-list" {
				result = fixture{Path: "/settings/contacts", Heading: "Contacts", ResourceID: contact.ID.String()}
			} else {
				result = fixture{Path: "/settings/contacts/" + contact.ID.String(), Heading: contact.ProfileName, ResourceID: contact.ID.String()}
			}
		case "user-detail":
			userEmail := os.Getenv("E2E_USER_REGULAR_USER_USERNAME")
			if userEmail == "" {
				userEmail = os.Getenv("E2E_USER_EMAIL")
			}
			if userEmail == "" {
				userEmail = "e2e-agent@test.com"
			}
			var user models.User
			if err := tx.Where("email = ? AND organization_id = ?", userEmail, orgID).First(&user).Error; err != nil {
				return fmt.Errorf("find regular E2E user %q: %w", userEmail, err)
			}
			result = fixture{Path: "/settings/users/" + user.ID.String(), Heading: user.FullName}
		case "audit-logs-list", "audit-log-detail":
			account, err := createAccount()
			if err != nil {
				return err
			}
			log := &models.AuditLog{
				ID:             uuid.New(),
				OrganizationID: orgID,
				ResourceType:   "account",
				ResourceID:     account.ID,
				UserID:         admin.ID,
				UserName:       base + " admin",
				Action:         models.AuditActionCreated,
				Changes: models.JSONBArray{
					map[string]any{"field": "fixture", "new_value": base + " audit details"},
				},
			}
			if err := tx.Create(log).Error; err != nil {
				return err
			}
			if scenario == "audit-logs-list" {
				result = fixture{Path: "/settings/audit-logs", Heading: "Audit Logs"}
			} else {
				result = fixture{Path: "/settings/audit-logs/" + log.ID.String(), Heading: "Account created"}
			}
		case "gowa-servers":
			server := &models.GowaInstance{
				OrganizationID: orgID,
				Name:           base + " GOWA server",
				// This list fixture uses a loopback-only address. It cannot contact
				// an external GOWA service if a user opens the row by mistake.
				BaseURL:    "http://127.0.0.1:9",
				WebhookURL: "",
				IsActive:   true,
			}
			if err := tx.Create(server).Error; err != nil {
				return err
			}
			result = fixture{Path: "/settings/gowa-servers", Heading: "GOWA Servers", ResourceID: server.ID.String()}
		case "teams":
			userEmail := os.Getenv("E2E_USER_REGULAR_USER_USERNAME")
			if userEmail == "" {
				userEmail = os.Getenv("E2E_USER_EMAIL")
			}
			if userEmail == "" {
				userEmail = "e2e-agent@test.com"
			}
			var user models.User
			if err := tx.Where("email = ? AND organization_id = ?", userEmail, orgID).First(&user).Error; err != nil {
				return fmt.Errorf("find regular E2E user %q: %w", userEmail, err)
			}
			team := &models.Team{
				OrganizationID:     orgID,
				Name:               base + " team",
				Description:        "Isolated E2E team fixture.",
				AssignmentStrategy: models.AssignmentStrategyRoundRobin,
				IsActive:           true,
				CreatedByID:        &admin.ID,
				UpdatedByID:        &admin.ID,
			}
			if err := tx.Create(team).Error; err != nil {
				return err
			}
			member := &models.TeamMember{TeamID: team.ID, UserID: user.ID, Role: models.TeamRoleAgent}
			if err := tx.Create(member).Error; err != nil {
				return err
			}
			result = fixture{Path: "/settings/teams", Heading: "Teams", ResourceID: team.ID.String()}
		case "templates":
			account, err := createAccount()
			if err != nil {
				return err
			}
			template, err := createTemplate(account.Name, "template", "Hello {{1}}, this is a local E2E template.")
			if err != nil {
				return err
			}
			result = fixture{Path: "/settings/templates", Heading: "Templates", ResourceID: template.ID.String()}
		default:
			return fmt.Errorf("unsupported E2E fixture scenario %q", scenario)
		}
		return nil
	})
	if err != nil {
		return fixture{}, fmt.Errorf("seed E2E scenario %q: %w", scenario, err)
	}
	return result, nil
}

func cleanup(db *gorm.DB, runID string, relatedResourceIDs []uuid.UUID) error {
	prefix := "E2E fixture " + runID
	return db.Transaction(func(tx *gorm.DB) error {
		var fixtureTeamIDs []uuid.UUID
		if err := tx.Unscoped().Model(&models.Team{}).
			Where("name LIKE ?", prefix+" team%").Pluck("id", &fixtureTeamIDs).Error; err != nil {
			return err
		}
		if len(fixtureTeamIDs) > 0 {
			if err := tx.Unscoped().Where("team_id IN ?", fixtureTeamIDs).
				Delete(&models.TeamMember{}).Error; err != nil {
				return err
			}
		}
		var campaignIDs []uuid.UUID
		if err := tx.Unscoped().Model(&models.BulkMessageCampaign{}).
			Where("name LIKE ?", prefix+" campaign%").Pluck("id", &campaignIDs).Error; err != nil {
			return err
		}
		campaignResourceIDs := append(append([]uuid.UUID(nil), campaignIDs...), relatedResourceIDs...)
		if len(campaignResourceIDs) > 0 {
			var recipientIDs []uuid.UUID
			if err := tx.Unscoped().Model(&models.BulkMessageRecipient{}).
				Where("campaign_id IN ?", campaignResourceIDs).Pluck("id", &recipientIDs).Error; err != nil {
				return err
			}
			relatedResourceIDs = append(relatedResourceIDs, campaignIDs...)
			relatedResourceIDs = append(relatedResourceIDs, recipientIDs...)
			if err := tx.Unscoped().Where("campaign_id IN ?", campaignResourceIDs).
				Delete(&models.BulkMessageRecipient{}).Error; err != nil {
				return err
			}
		}
		fixtureResources := []struct {
			model any
			query string
			args  []any
		}{
			{&models.Template{}, "name LIKE ?", []any{prefix + "%"}},
			{&models.CannedResponse{}, "name LIKE ?", []any{prefix + " response%"}},
			{&models.Contact{}, "profile_name LIKE ?", []any{prefix + " contact%"}},
			{&models.GowaInstance{}, "name LIKE ?", []any{prefix + " GOWA server%"}},
			{&models.Team{}, "name LIKE ?", []any{prefix + " team%"}},
			{&models.WhatsAppAccount{}, "name LIKE ?", []any{prefix + " account%"}},
		}
		for _, resource := range fixtureResources {
			var ids []uuid.UUID
			if err := tx.Unscoped().Model(resource.model).
				Where(resource.query, resource.args...).Pluck("id", &ids).Error; err != nil {
				return err
			}
			relatedResourceIDs = append(relatedResourceIDs, ids...)
		}
		if len(relatedResourceIDs) > 0 {
			if err := tx.Unscoped().Where("resource_id IN ?", relatedResourceIDs).
				Delete(&models.AuditLog{}).Error; err != nil {
				return err
			}
		}
		deletions := []struct {
			model any
			query string
			args  []any
		}{
			{&models.BulkMessageCampaign{}, "name LIKE ?", []any{prefix + " campaign%"}},
			{&models.Template{}, "name LIKE ?", []any{prefix + "%"}},
			{&models.CannedResponse{}, "name LIKE ?", []any{prefix + " response%"}},
			{&models.Contact{}, "profile_name LIKE ?", []any{prefix + " contact%"}},
			{&models.AuditLog{}, "user_name LIKE ?", []any{prefix + " admin%"}},
			{&models.GowaInstance{}, "name LIKE ?", []any{prefix + " GOWA server%"}},
			{&models.Team{}, "name LIKE ?", []any{prefix + " team%"}},
			{&models.WhatsAppAccount{}, "name LIKE ?", []any{prefix + " account%"}},
		}
		for _, item := range deletions {
			if err := tx.Unscoped().Where(item.query, item.args...).Delete(item.model).Error; err != nil {
				return err
			}
		}
		for _, model := range []any{
			&models.BulkMessageCampaign{},
			&models.Template{},
			&models.CannedResponse{},
			&models.Contact{},
			&models.GowaInstance{},
			&models.Team{},
			&models.WhatsAppAccount{},
		} {
			if len(relatedResourceIDs) == 0 {
				break
			}
			if err := tx.Unscoped().Where("id IN ?", relatedResourceIDs).Delete(model).Error; err != nil {
				return err
			}
		}
		remaining := []struct {
			model any
			query string
			args  []any
			name  string
		}{
			{&models.BulkMessageCampaign{}, "name LIKE ?", []any{prefix + " campaign%"}, "campaign"},
			{&models.BulkMessageRecipient{}, "campaign_id IN ?", []any{campaignResourceIDs}, "campaign recipient"},
			{&models.Template{}, "name LIKE ?", []any{prefix + "%"}, "template"},
			{&models.CannedResponse{}, "name LIKE ?", []any{prefix + " response%"}, "canned response"},
			{&models.Contact{}, "profile_name LIKE ?", []any{prefix + " contact%"}, "contact"},
			{&models.AuditLog{}, "user_name LIKE ?", []any{prefix + " admin%"}, "audit log"},
			{&models.GowaInstance{}, "name LIKE ?", []any{prefix + " GOWA server%"}, "GOWA server"},
			{&models.Team{}, "name LIKE ?", []any{prefix + " team%"}, "team"},
			{&models.WhatsAppAccount{}, "name LIKE ?", []any{prefix + " account%"}, "WhatsApp account"},
		}
		for _, item := range remaining {
			var count int64
			if err := tx.Unscoped().Model(item.model).Where(item.query, item.args...).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("E2E fixture cleanup left %d %s record(s)", count, item.name)
			}
		}
		if len(fixtureTeamIDs) > 0 {
			var count int64
			if err := tx.Unscoped().Model(&models.TeamMember{}).
				Where("team_id IN ?", fixtureTeamIDs).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("E2E fixture cleanup left %d team member record(s)", count)
			}
		}
		if len(relatedResourceIDs) > 0 {
			var count int64
			if err := tx.Unscoped().Model(&models.AuditLog{}).
				Where("resource_id IN ?", relatedResourceIDs).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("E2E fixture cleanup left %d related audit log record(s)", count)
			}
		}
		return nil
	})
}
