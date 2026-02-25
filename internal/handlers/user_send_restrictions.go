package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// UserSendRestrictionsResponse represents per-user strict send restrictions.
type UserSendRestrictionsResponse struct {
	Enabled            bool     `json:"enabled"`
	IncludeAllContacts bool     `json:"include_all_contacts"`
	AuthorizedNumbers  []string `json:"authorized_numbers"`
	AllowedInstanceID  *string  `json:"allowed_instance_id,omitempty"`
	PrefixAgentName    bool     `json:"prefix_agent_name"`
}

func (a *App) getUserSendRestrictionsForOrg(orgID, userID uuid.UUID) (*models.User, sendRestrictionsSettings, error) {
	user, err := a.loadUserForSendRestrictions(orgID, userID)
	if err != nil {
		return nil, sendRestrictionsSettings{}, err
	}

	cfg := readSendRestrictionsSettings(user.Settings)
	if !cfg.Enabled {
		cfg.AuthorizedNumbers = normalizeRestrictedNumbers(cfg.AuthorizedNumbers)
		return user, cfg, nil
	}
	cfg, err = a.syncUserRestrictionsWithSources(orgID, user, cfg)
	if err != nil {
		return nil, sendRestrictionsSettings{}, err
	}

	return user, cfg, nil
}

func (a *App) resolveSendRestrictionsInstanceID(orgID uuid.UUID, raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	instanceID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, errors.New("allowed_instance_id must be a valid UUID")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("allowed_instance_id does not belong to this organization")
		}
		return nil, err
	}

	return &instanceID, nil
}

// GetUserSendRestrictions returns strict send restriction settings for a user.
func (a *App) GetUserSendRestrictions(r *fastglue.Request) error {
	orgID, currentUserID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, currentUserID, models.ResourceUsers, models.ActionRead); err != nil {
		return nil
	}

	targetUserID, err := parsePathUUID(r, "id", "user")
	if err != nil {
		return nil
	}

	_, cfg, err := a.getUserSendRestrictionsForOrg(orgID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
		}
		a.Log.Error("Failed to load user send restrictions", "error", err, "user_id", targetUserID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load send restrictions", nil, "")
	}

	return r.SendEnvelope(UserSendRestrictionsResponse{
		Enabled:            cfg.Enabled,
		IncludeAllContacts: cfg.IncludeAllContacts,
		AuthorizedNumbers:  cfg.AuthorizedNumbers,
		AllowedInstanceID:  stringifyOptionalUUID(cfg.AllowedInstanceID),
		PrefixAgentName:    cfg.PrefixAgentName,
	})
}

// UpdateUserSendRestrictions updates strict send restriction settings for a user.
func (a *App) UpdateUserSendRestrictions(r *fastglue.Request) error {
	orgID, currentUserID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, currentUserID, models.ResourceUsers, models.ActionWrite); err != nil {
		return nil
	}

	targetUserID, err := parsePathUUID(r, "id", "user")
	if err != nil {
		return nil
	}

	var req struct {
		Enabled            *bool     `json:"enabled"`
		IncludeAllContacts *bool     `json:"include_all_contacts"`
		AuthorizedNumbers  *[]string `json:"authorized_numbers"`
		AllowedInstanceID  *string   `json:"allowed_instance_id"`
		PrefixAgentName    *bool     `json:"prefix_agent_name"`
	}
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	user, cfg, err := a.getUserSendRestrictionsForOrg(orgID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, "")
		}
		a.Log.Error("Failed to resolve user send restrictions before update", "error", err, "user_id", targetUserID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update send restrictions", nil, "")
	}

	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.IncludeAllContacts != nil {
		cfg.IncludeAllContacts = *req.IncludeAllContacts
	}
	if req.AuthorizedNumbers != nil {
		cfg.AuthorizedNumbers = normalizeRestrictedNumbers(*req.AuthorizedNumbers)
	}
	if req.AllowedInstanceID != nil {
		instanceID, resolveErr := a.resolveSendRestrictionsInstanceID(orgID, *req.AllowedInstanceID)
		if resolveErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, resolveErr.Error(), nil, "allowed_instance_id")
		}
		cfg.AllowedInstanceID = instanceID
	}
	if req.PrefixAgentName != nil {
		cfg.PrefixAgentName = *req.PrefixAgentName
	}

	if cfg.Enabled {
		if a.isWhatsmeowProvider() && cfg.AllowedInstanceID == nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "allowed_instance_id is required when restrictions are enabled", nil, "allowed_instance_id")
		}

		cfg, err = a.syncUserRestrictionsWithSources(orgID, user, cfg)
		if err != nil {
			a.Log.Error("Failed to sync authorized numbers", "error", err, "user_id", user.ID, "org_id", orgID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update send restrictions", nil, "")
		}
		user.Settings = writeSendRestrictionsSettings(user.Settings, cfg)
	}

	if err := a.saveUserSendRestrictions(user.ID, user.Settings, cfg); err != nil {
		a.Log.Error("Failed to persist user send restrictions", "error", err, "user_id", user.ID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update send restrictions", nil, "")
	}

	return r.SendEnvelope(UserSendRestrictionsResponse{
		Enabled:            cfg.Enabled,
		IncludeAllContacts: cfg.IncludeAllContacts,
		AuthorizedNumbers:  cfg.AuthorizedNumbers,
		AllowedInstanceID:  stringifyOptionalUUID(cfg.AllowedInstanceID),
		PrefixAgentName:    cfg.PrefixAgentName,
	})
}
