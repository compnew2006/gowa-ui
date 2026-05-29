package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const agentSelectionSystemActorName = "Customer routing"

type agentSelectionSettingsRequest struct {
	InstanceID                *uuid.UUID                        `json:"instance_id"`
	AllowedInstanceIDs        *[]string                         `json:"allowed_instance_ids"`
	Enabled                   *bool                             `json:"enabled"`
	TriggerMode               models.AgentSelectionTriggerMode  `json:"trigger_mode"`
	TriggerKeywords           []string                          `json:"trigger_keywords"`
	PromptDelayMinutes        *int                              `json:"prompt_delay_minutes"`
	SelectionTimeoutMinutes   *int                              `json:"selection_timeout_minutes"`
	MaxInvalidAttempts        *int                              `json:"max_invalid_attempts"`
	MenuHeaderText            *string                           `json:"menu_header_text"`
	MenuFooterText            *string                           `json:"menu_footer_text"`
	InvalidReplyText          *string                           `json:"invalid_reply_text"`
	TimeoutResponseText       *string                           `json:"timeout_response_text"`
	UnavailableAgentText      *string                           `json:"unavailable_agent_text"`
	CustomFinalOptionEnabled  *bool                             `json:"custom_final_option_enabled"`
	CustomFinalOptionText     *string                           `json:"custom_final_option_text"`
	CustomFinalOptionResponse *string                           `json:"custom_final_option_response"`
	CustomFinalOptionAction   models.AgentSelectionCustomAction `json:"custom_final_option_action"`
	CustomFinalOptionTeamID   *uuid.UUID                        `json:"custom_final_option_team_id"`
	HideUnavailableAgents     *bool                             `json:"hide_unavailable_agents"`
}

type agentSelectionParticipantRequest struct {
	SettingsID            uuid.UUID `json:"settings_id"`
	UserID                uuid.UUID `json:"user_id"`
	DisplayName           string    `json:"display_name"`
	Description           string    `json:"description"`
	IsEnabled             *bool     `json:"is_enabled"`
	SortOrder             int       `json:"sort_order"`
	ShowOnlyWhenAvailable *bool     `json:"show_only_when_available"`
	MaxOpenChats          *int      `json:"max_open_chats"`
}

type agentSelectionOptionRequest struct {
	SettingsID  uuid.UUID                       `json:"settings_id"`
	OptionType  models.AgentSelectionOptionType `json:"option_type"`
	UserID      *uuid.UUID                      `json:"user_id"`
	TeamID      *uuid.UUID                      `json:"team_id"`
	Label       string                          `json:"label"`
	Description string                          `json:"description"`
	IsEnabled   *bool                           `json:"is_enabled"`
	SortOrder   int                             `json:"sort_order"`
	Action      string                          `json:"action"`
}

type agentSelectionPreviewRequest struct {
	SettingsID *uuid.UUID `json:"settings_id"`
	ContactID  *uuid.UUID `json:"contact_id"`
}

type agentSelectionRenderedOption struct {
	Number      int                             `json:"number"`
	OptionID    string                          `json:"option_id"`
	Type        models.AgentSelectionOptionType `json:"type"`
	Label       string                          `json:"label"`
	Description string                          `json:"description,omitempty"`
	UserID      *uuid.UUID                      `json:"user_id,omitempty"`
	TeamID      *uuid.UUID                      `json:"team_id,omitempty"`
	Action      string                          `json:"action,omitempty"`
	Response    string                          `json:"response,omitempty"`
}

type agentSelectionRenderedMenu struct {
	Text    string                         `json:"text"`
	Options []agentSelectionRenderedOption `json:"options"`
}

type agentSelectionAuditInput struct {
	ContactID              *uuid.UUID
	SessionID              *uuid.UUID
	InstanceID             *uuid.UUID
	WhatsAppAccount        string
	EventType              string
	ActorType              models.AgentSelectionAuditActor
	ActorUserID            *uuid.UUID
	SelectedOptionID       string
	SelectedAgentID        *uuid.UUID
	SelectedTeamID         *uuid.UUID
	PreviousAssignedUserID *uuid.UUID
	NewAssignedUserID      *uuid.UUID
	TransferID             *uuid.UUID
	InboundMessageID       *uuid.UUID
	OutboundMessageID      *uuid.UUID
	Reason                 string
	Metadata               models.JSONB
}

func defaultAgentSelectionSettings(orgID uuid.UUID, instanceID *uuid.UUID) models.AgentSelectionSettings {
	return models.AgentSelectionSettings{
		OrganizationID:            orgID,
		InstanceID:                instanceID,
		AllowedInstanceIDs:        models.StringArray{},
		Enabled:                   false,
		TriggerMode:               models.AgentSelectionTriggerFirstPendingMessage,
		TriggerKeywords:           models.StringArray{},
		PromptDelayMinutes:        3,
		SelectionTimeoutMinutes:   10,
		MaxInvalidAttempts:        3,
		MenuHeaderText:            "من فضلك اختر من تريد التواصل معه:",
		MenuFooterText:            "",
		InvalidReplyText:          "اختيار غير صحيح. من فضلك ارسل رقم من القائمة.",
		TimeoutResponseText:       "",
		UnavailableAgentText:      "هذا الوكيل غير متاح حاليا. من فضلك اختر رقم آخر أو انتظر أحد ممثلينا.",
		CustomFinalOptionEnabled:  false,
		CustomFinalOptionText:     "سأذهب للفرع للطباعة",
		CustomFinalOptionResponse: "تمام، يسعدنا خدمتك في الفرع.",
		CustomFinalOptionAction:   models.AgentSelectionCustomActionKeepPending,
		HideUnavailableAgents:     true,
	}
}

func (a *App) requireAgentSelectionPermission(r *fastglue.Request, userID uuid.UUID, action string) error {
	return a.requirePermission(r, userID, models.ResourceAgentSelection, action)
}

func (a *App) resolveAgentSelectionSettings(db *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID) (*models.AgentSelectionSettings, error) {
	var settings models.AgentSelectionSettings
	if instanceID != nil && *instanceID != uuid.Nil {
		err := db.Where("organization_id = ? AND instance_id = ?", orgID, *instanceID).First(&settings).Error
		if err == nil {
			return &settings, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	err := db.Where("organization_id = ? AND instance_id IS NULL", orgID).First(&settings).Error
	if err == nil {
		return &settings, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	defaults := defaultAgentSelectionSettings(orgID, instanceID)
	return &defaults, nil
}

func (a *App) ensureAgentSelectionSettings(db *gorm.DB, orgID uuid.UUID, instanceID *uuid.UUID) (*models.AgentSelectionSettings, error) {
	settings, err := a.resolveAgentSelectionSettings(db, orgID, instanceID)
	if err != nil {
		return nil, err
	}
	if settings.ID != uuid.Nil {
		return settings, nil
	}
	if settings.InstanceID != nil && *settings.InstanceID == uuid.Nil {
		settings.InstanceID = nil
	}
	if err := db.Create(settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (a *App) GetAgentSelectionSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	var instanceID *uuid.UUID
	if raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id"))); raw != "" {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
		}
		instanceID = &parsed
	}

	settings, err := a.resolveAgentSelectionSettings(requestDB, orgID, instanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load agent selection settings", nil, "")
	}

	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (a *App) UpdateAgentSelectionSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req agentSelectionSettingsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	settings, err := a.ensureAgentSelectionSettings(requestDB, orgID, req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load agent selection settings", nil, "")
	}

	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.TriggerMode != "" {
		settings.TriggerMode = req.TriggerMode
	}
	settings.TriggerKeywords = normalizeStringArray(req.TriggerKeywords)
	if req.AllowedInstanceIDs != nil {
		allowedInstanceIDs, err := a.normalizeAgentSelectionAllowedInstanceIDs(requestDB, orgID, *req.AllowedInstanceIDs)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
		settings.AllowedInstanceIDs = allowedInstanceIDs
	}
	if req.PromptDelayMinutes != nil {
		settings.PromptDelayMinutes = clampInt(*req.PromptDelayMinutes, 0, 24*60)
	}
	if req.SelectionTimeoutMinutes != nil {
		settings.SelectionTimeoutMinutes = clampInt(*req.SelectionTimeoutMinutes, 1, 24*60)
	}
	if req.MaxInvalidAttempts != nil {
		settings.MaxInvalidAttempts = clampInt(*req.MaxInvalidAttempts, 1, 20)
	}
	if req.MenuHeaderText != nil {
		settings.MenuHeaderText = strings.TrimSpace(*req.MenuHeaderText)
	}
	if req.MenuFooterText != nil {
		settings.MenuFooterText = strings.TrimSpace(*req.MenuFooterText)
	}
	if req.InvalidReplyText != nil {
		settings.InvalidReplyText = strings.TrimSpace(*req.InvalidReplyText)
	}
	if req.TimeoutResponseText != nil {
		settings.TimeoutResponseText = strings.TrimSpace(*req.TimeoutResponseText)
	}
	if req.UnavailableAgentText != nil {
		settings.UnavailableAgentText = strings.TrimSpace(*req.UnavailableAgentText)
	}
	if req.CustomFinalOptionEnabled != nil {
		settings.CustomFinalOptionEnabled = *req.CustomFinalOptionEnabled
	}
	if req.CustomFinalOptionText != nil {
		settings.CustomFinalOptionText = strings.TrimSpace(*req.CustomFinalOptionText)
	}
	if req.CustomFinalOptionResponse != nil {
		settings.CustomFinalOptionResponse = strings.TrimSpace(*req.CustomFinalOptionResponse)
	}
	if req.CustomFinalOptionAction != "" {
		settings.CustomFinalOptionAction = req.CustomFinalOptionAction
	}
	settings.CustomFinalOptionTeamID = req.CustomFinalOptionTeamID
	if req.HideUnavailableAgents != nil {
		settings.HideUnavailableAgents = *req.HideUnavailableAgents
	}

	if err := requestDB.Save(settings).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save agent selection settings", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   "settings_updated",
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"settings_id": settings.ID.String()},
	})
	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (a *App) ListAgentSelectionParticipants(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	query := requestDB.Where("organization_id = ?", orgID).Preload("User").Order("sort_order ASC, display_name ASC")
	if rawSettingsID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("settings_id"))); rawSettingsID != "" {
		settingsID, parseErr := uuid.Parse(rawSettingsID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid settings_id", nil, "")
		}
		query = query.Where("settings_id = ?", settingsID)
	}

	var participants []models.AgentSelectionParticipant
	if err := query.Find(&participants).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list participants", nil, "")
	}
	return r.SendEnvelope(map[string]any{"participants": participants})
}

func (a *App) CreateAgentSelectionParticipant(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req agentSelectionParticipantRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if req.SettingsID == uuid.Nil || req.UserID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "settings_id and user_id are required", nil, "")
	}
	if !a.userBelongsToOrg(requestDB, req.UserID, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Agent is not available for this organization", nil, "")
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(a.ResolveUserDisplayName(req.UserID))
	}
	if displayName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "display_name is required", nil, "")
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	showOnlyWhenAvailable := true
	if req.ShowOnlyWhenAvailable != nil {
		showOnlyWhenAvailable = *req.ShowOnlyWhenAvailable
	}

	participant := models.AgentSelectionParticipant{
		OrganizationID:        orgID,
		SettingsID:            req.SettingsID,
		UserID:                req.UserID,
		DisplayName:           displayName,
		Description:           strings.TrimSpace(req.Description),
		IsEnabled:             isEnabled,
		SortOrder:             req.SortOrder,
		ShowOnlyWhenAvailable: showOnlyWhenAvailable,
		MaxOpenChats:          req.MaxOpenChats,
		Metadata:              models.JSONB{},
	}
	if err := requestDB.Create(&participant).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "Agent is already in this routing list", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   "participant_created",
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"participant_id": participant.ID.String(), "agent_id": participant.UserID.String()},
	})
	return r.SendEnvelope(map[string]any{"participant": participant})
}

func (a *App) UpdateAgentSelectionParticipant(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "participant")
	if err != nil {
		return nil
	}
	var participant models.AgentSelectionParticipant
	if err := requestDB.Where("id = ? AND organization_id = ?", id, orgID).First(&participant).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Participant not found", nil, "")
	}

	var req agentSelectionParticipantRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if strings.TrimSpace(req.DisplayName) != "" {
		participant.DisplayName = strings.TrimSpace(req.DisplayName)
	}
	participant.Description = strings.TrimSpace(req.Description)
	if req.IsEnabled != nil {
		participant.IsEnabled = *req.IsEnabled
	}
	participant.SortOrder = req.SortOrder
	if req.ShowOnlyWhenAvailable != nil {
		participant.ShowOnlyWhenAvailable = *req.ShowOnlyWhenAvailable
	}
	participant.MaxOpenChats = req.MaxOpenChats
	if err := requestDB.Save(&participant).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save participant", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   "participant_updated",
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"participant_id": participant.ID.String(), "agent_id": participant.UserID.String()},
	})
	return r.SendEnvelope(map[string]any{"participant": participant})
}

func (a *App) DeleteAgentSelectionParticipant(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "participant")
	if err != nil {
		return nil
	}
	result := requestDB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.AgentSelectionParticipant{})
	if result.Error != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete participant", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   "participant_deleted",
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"participant_id": id.String()},
	})
	return r.SendEnvelope(map[string]any{"deleted": result.RowsAffected > 0})
}

func (a *App) ListAgentSelectionOptions(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}
	var options []models.AgentSelectionOption
	query := requestDB.Where("organization_id = ?", orgID).Order("sort_order ASC, label ASC")
	if rawSettingsID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("settings_id"))); rawSettingsID != "" {
		settingsID, parseErr := uuid.Parse(rawSettingsID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid settings_id", nil, "")
		}
		query = query.Where("settings_id = ?", settingsID)
	}
	if err := query.Find(&options).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list options", nil, "")
	}
	return r.SendEnvelope(map[string]any{"options": options})
}

func (a *App) CreateAgentSelectionOption(r *fastglue.Request) error {
	return a.upsertAgentSelectionOption(r, uuid.Nil)
}

func (a *App) UpdateAgentSelectionOption(r *fastglue.Request) error {
	id, err := parsePathUUID(r, "id", "option")
	if err != nil {
		return nil
	}
	return a.upsertAgentSelectionOption(r, id)
}

func (a *App) upsertAgentSelectionOption(r *fastglue.Request, id uuid.UUID) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}

	var req agentSelectionOptionRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if req.SettingsID == uuid.Nil || req.OptionType == "" || strings.TrimSpace(req.Label) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "settings_id, option_type and label are required", nil, "")
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	option := models.AgentSelectionOption{}
	if id != uuid.Nil {
		if err := requestDB.Where("id = ? AND organization_id = ?", id, orgID).First(&option).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Option not found", nil, "")
		}
	} else {
		option.OrganizationID = orgID
		option.Metadata = models.JSONB{}
	}
	option.SettingsID = req.SettingsID
	option.OptionType = req.OptionType
	option.UserID = req.UserID
	option.TeamID = req.TeamID
	option.Label = strings.TrimSpace(req.Label)
	option.Description = strings.TrimSpace(req.Description)
	option.IsEnabled = isEnabled
	option.SortOrder = req.SortOrder
	option.Action = strings.TrimSpace(req.Action)

	if err := requestDB.Save(&option).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save option", nil, "")
	}
	eventType := "option_created"
	if id != uuid.Nil {
		eventType = "option_updated"
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   eventType,
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"option_id": option.ID.String(), "option_type": string(option.OptionType)},
	})
	return r.SendEnvelope(map[string]any{"option": option})
}

func (a *App) DeleteAgentSelectionOption(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "option")
	if err != nil {
		return nil
	}
	result := requestDB.Where("id = ? AND organization_id = ?", id, orgID).Delete(&models.AgentSelectionOption{})
	if result.Error != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete option", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		EventType:   "option_deleted",
		ActorType:   models.AgentSelectionActorAdmin,
		ActorUserID: &userID,
		Metadata:    models.JSONB{"option_id": id.String()},
	})
	return r.SendEnvelope(map[string]any{"deleted": result.RowsAffected > 0})
}

func (a *App) PreviewAgentSelectionMenu(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	var req agentSelectionPreviewRequest
	if len(r.RequestCtx.PostBody()) > 0 {
		if err := a.decodeRequest(r, &req); err != nil {
			return nil
		}
	}

	var settings *models.AgentSelectionSettings
	if req.SettingsID != nil && *req.SettingsID != uuid.Nil {
		var s models.AgentSelectionSettings
		if err := requestDB.Where("id = ? AND organization_id = ?", *req.SettingsID, orgID).First(&s).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Settings not found", nil, "")
		}
		settings = &s
	} else {
		settings, err = a.resolveAgentSelectionSettings(requestDB, orgID, nil)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load settings", nil, "")
		}
	}

	var contact *models.Contact
	if req.ContactID != nil && *req.ContactID != uuid.Nil {
		var c models.Contact
		if err := requestDB.Where("id = ? AND organization_id = ?", *req.ContactID, orgID).First(&c).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, "")
		}
		contact = &c
	}

	menu, err := a.buildAgentSelectionMenu(requestDB, orgID, settings, contact)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to build menu preview", nil, "")
	}
	return r.SendEnvelope(map[string]any{"menu": menu})
}

func (a *App) ListAgentSelectionAudit(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}

	pg := parsePagination(r)
	query := requestDB.Where("organization_id = ?", orgID)
	if eventType := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("event_type"))); eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if contactIDRaw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("contact_id"))); contactIDRaw != "" {
		contactID, parseErr := uuid.Parse(contactIDRaw)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid contact_id", nil, "")
		}
		query = query.Where("contact_id = ?", contactID)
	}
	if sessionIDRaw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("session_id"))); sessionIDRaw != "" {
		sessionID, parseErr := uuid.Parse(sessionIDRaw)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid session_id", nil, "")
		}
		query = query.Where("session_id = ?", sessionID)
	}

	var total int64
	query.Model(&models.AgentSelectionAuditEvent{}).Count(&total)
	var events []models.AgentSelectionAuditEvent
	if err := pg.Apply(query.Order("created_at DESC")).Find(&events).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list audit events", nil, "")
	}
	return r.SendEnvelope(map[string]any{"events": events, "total": total, "page": pg.Page, "limit": pg.Limit})
}

func (a *App) ListAgentSelectionSessions(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionRead); err != nil {
		return nil
	}
	pg := parsePagination(r)
	query := requestDB.Where("organization_id = ?", orgID)
	if status := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("status"))); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Model(&models.AgentSelectionSession{}).Count(&total)
	var sessions []models.AgentSelectionSession
	if err := pg.Apply(query.Order("created_at DESC")).Find(&sessions).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list sessions", nil, "")
	}
	return r.SendEnvelope(map[string]any{"sessions": sessions, "total": total, "page": pg.Page, "limit": pg.Limit})
}

func (a *App) CancelAgentSelectionSession(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requireAgentSelectionPermission(r, userID, models.ActionWrite); err != nil {
		return nil
	}
	id, err := parsePathUUID(r, "id", "session")
	if err != nil {
		return nil
	}
	var session models.AgentSelectionSession
	if err := requestDB.Where("id = ? AND organization_id = ?", id, orgID).First(&session).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Session not found", nil, "")
	}
	session.Status = models.AgentSelectionSessionCancelled
	if err := requestDB.Save(&session).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to cancel session", nil, "")
	}
	a.writeAgentSelectionAudit(requestDB, orgID, agentSelectionAuditInput{
		ContactID:       &session.ContactID,
		SessionID:       &session.ID,
		InstanceID:      session.InstanceID,
		WhatsAppAccount: session.WhatsAppAccount,
		EventType:       models.AgentSelectionEventSessionCancelled,
		ActorType:       models.AgentSelectionActorAdmin,
		ActorUserID:     &userID,
	})
	return r.SendEnvelope(map[string]any{"session": session})
}

func (a *App) buildAgentSelectionMenu(db *gorm.DB, orgID uuid.UUID, settings *models.AgentSelectionSettings, contact *models.Contact) (*agentSelectionRenderedMenu, error) {
	if settings == nil {
		return nil, errors.New("settings is nil")
	}

	options := make([]agentSelectionRenderedOption, 0)
	if settings.ID != uuid.Nil {
		var participants []models.AgentSelectionParticipant
		if err := db.Where("organization_id = ? AND settings_id = ? AND is_enabled = ?", orgID, settings.ID, true).
			Preload("User").
			Order("sort_order ASC, display_name ASC").
			Find(&participants).Error; err != nil {
			return nil, err
		}
		for _, participant := range participants {
			if participant.User == nil || !participant.User.IsActive {
				continue
			}
			if participant.ShowOnlyWhenAvailable && !participant.User.IsAvailable {
				continue
			}
			if participant.MaxOpenChats != nil && *participant.MaxOpenChats >= 0 {
				var openCount int64
				db.Model(&models.Contact{}).
					Where("organization_id = ? AND assigned_user_id = ? AND status = ?", orgID, participant.UserID, models.ChatStatusOpen).
					Count(&openCount)
				if openCount >= int64(*participant.MaxOpenChats) {
					continue
				}
			}
			if contact != nil {
				ok, err := a.canUserSeeContactInstance(orgID, participant.UserID, contact)
				if err != nil || !ok {
					continue
				}
			}
			userID := participant.UserID
			options = append(options, agentSelectionRenderedOption{
				OptionID:    participant.ID.String(),
				Type:        models.AgentSelectionOptionAgent,
				Label:       participant.DisplayName,
				Description: participant.Description,
				UserID:      &userID,
			})
		}

		var configuredOptions []models.AgentSelectionOption
		if err := db.Where("organization_id = ? AND settings_id = ? AND is_enabled = ?", orgID, settings.ID, true).
			Order("sort_order ASC, label ASC").
			Find(&configuredOptions).Error; err != nil {
			return nil, err
		}
		for _, configured := range configuredOptions {
			if configured.OptionType == models.AgentSelectionOptionAgent {
				continue
			}
			if configured.OptionType == models.AgentSelectionOptionTeam && configured.TeamID != nil {
				var team models.Team
				if err := db.Where("id = ? AND organization_id = ? AND is_active = ?", *configured.TeamID, orgID, true).First(&team).Error; err != nil {
					continue
				}
			}
			option := agentSelectionRenderedOption{
				OptionID:    configured.ID.String(),
				Type:        configured.OptionType,
				Label:       configured.Label,
				Description: configured.Description,
				UserID:      configured.UserID,
				TeamID:      configured.TeamID,
				Action:      configured.Action,
			}
			options = append(options, option)
		}
	}

	if settings.CustomFinalOptionEnabled && strings.TrimSpace(settings.CustomFinalOptionText) != "" {
		options = append(options, agentSelectionRenderedOption{
			OptionID: "custom_final",
			Type:     models.AgentSelectionOptionCustom,
			Label:    strings.TrimSpace(settings.CustomFinalOptionText),
			Action:   string(settings.CustomFinalOptionAction),
			Response: settings.CustomFinalOptionResponse,
			TeamID:   settings.CustomFinalOptionTeamID,
		})
	}

	sort.SliceStable(options, func(i, j int) bool {
		if options[i].Type == models.AgentSelectionOptionCustom {
			return false
		}
		if options[j].Type == models.AgentSelectionOptionCustom {
			return true
		}
		return options[i].Label < options[j].Label
	})

	lines := make([]string, 0, len(options)+3)
	header := strings.TrimSpace(settings.MenuHeaderText)
	if header == "" {
		header = "من فضلك اختر من تريد التواصل معه:"
	}
	lines = append(lines, header, "")
	for i := range options {
		options[i].Number = i + 1
		lines = append(lines, fmt.Sprintf("%d. %s", options[i].Number, options[i].Label))
	}
	if footer := strings.TrimSpace(settings.MenuFooterText); footer != "" {
		lines = append(lines, "", footer)
	}

	return &agentSelectionRenderedMenu{Text: strings.Join(lines, "\n"), Options: options}, nil
}

func (a *App) maybeHandleAgentSelectionInbound(account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, payload incomingMessagePayload) bool {
	if account == nil || contact == nil || inbound == nil || !a.isWhatsmeowProvider() {
		return false
	}
	db := a.DB
	orgID := account.OrganizationID
	settings, err := a.resolveAgentSelectionSettings(db, orgID, contact.InstanceID)
	if err != nil || settings == nil || !settings.Enabled {
		return false
	}
	if !agentSelectionSettingsAppliesToInstance(settings, contact.InstanceID) {
		return false
	}

	text := strings.TrimSpace(payload.MessageText)
	if text == "" {
		return false
	}

	var session models.AgentSelectionSession
	err = db.Where("organization_id = ? AND contact_id = ? AND status = ?", orgID, contact.ID, models.AgentSelectionSessionMenuSent).
		Order("created_at DESC").
		First(&session).Error
	if err == nil {
		a.processAgentSelectionReply(db, account, contact, inbound, text, &session, settings)
		return true
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.Log.Error("Failed to load agent selection session", "error", err, "contact_id", contact.ID)
		return false
	}

	if !a.shouldCreateAgentSelectionDelay(settings, contact, text) {
		return false
	}

	if err := a.createAgentSelectionDelaySession(db, account, contact, inbound, settings); err != nil {
		a.Log.Error("Failed to create agent selection delay session", "error", err, "contact_id", contact.ID)
		return false
	}
	return true
}

func (a *App) shouldCreateAgentSelectionDelay(settings *models.AgentSelectionSettings, contact *models.Contact, text string) bool {
	if settings == nil || contact == nil {
		return false
	}
	if contact.AssignedUserID != nil || normalizeContactStatus(contact) != models.ChatStatusPending {
		return false
	}
	switch settings.TriggerMode {
	case models.AgentSelectionTriggerKeyword:
		lowerText := strings.ToLower(strings.TrimSpace(text))
		for _, keyword := range settings.TriggerKeywords {
			if strings.TrimSpace(keyword) != "" && strings.Contains(lowerText, strings.ToLower(strings.TrimSpace(keyword))) {
				return true
			}
		}
		return false
	case models.AgentSelectionTriggerManualTest, models.AgentSelectionTriggerChatbotStep:
		return false
	default:
		return true
	}
}

func (a *App) createAgentSelectionDelaySession(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, settings *models.AgentSelectionSettings) error {
	var existing models.AgentSelectionSession
	err := db.Where(
		"organization_id = ? AND contact_id = ? AND status IN ?",
		account.OrganizationID,
		contact.ID,
		[]models.AgentSelectionSessionStatus{models.AgentSelectionSessionWaitingDelay, models.AgentSelectionSessionMenuSent},
	).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	delay := time.Duration(clampInt(settings.PromptDelayMinutes, 0, 24*60)) * time.Minute
	promptDueAt := time.Now().Add(delay)
	session := models.AgentSelectionSession{
		OrganizationID:      account.OrganizationID,
		ContactID:           contact.ID,
		InstanceID:          contact.InstanceID,
		WhatsAppAccount:     account.Name,
		Status:              models.AgentSelectionSessionWaitingDelay,
		TriggerMessageID:    &inbound.ID,
		PromptDueAt:         promptDueAt,
		InvalidAttempts:     0,
		ProcessedInboundIDs: models.StringArray{},
		Metadata:            models.JSONB{"settings_id": settings.ID.String()},
	}
	if err := db.Create(&session).Error; err != nil {
		return err
	}
	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:        &contact.ID,
		SessionID:        &session.ID,
		InstanceID:       contact.InstanceID,
		WhatsAppAccount:  account.Name,
		EventType:        models.AgentSelectionEventSessionCreated,
		ActorType:        models.AgentSelectionActorSystem,
		InboundMessageID: &inbound.ID,
		Metadata:         models.JSONB{"prompt_due_at": promptDueAt.Format(time.RFC3339)},
	})
	return nil
}

func (a *App) processAgentSelectionReply(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, replyText string, session *models.AgentSelectionSession, settings *models.AgentSelectionSettings) {
	if sessionHasProcessedInbound(session, inbound.ID) {
		return
	}
	selected, ok := selectedRenderedOption(session.RenderedOptionsSnapshot, replyText)
	if !ok {
		session.InvalidAttempts++
		session.ProcessedInboundIDs = append(session.ProcessedInboundIDs, inbound.ID.String())
		if settings.MaxInvalidAttempts > 0 && session.InvalidAttempts >= settings.MaxInvalidAttempts {
			session.Status = models.AgentSelectionSessionExpired
			a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
				ContactID:        &contact.ID,
				SessionID:        &session.ID,
				InstanceID:       contact.InstanceID,
				WhatsAppAccount:  account.Name,
				EventType:        models.AgentSelectionEventMaxInvalidAttemptsReached,
				ActorType:        models.AgentSelectionActorCustomer,
				InboundMessageID: &inbound.ID,
			})
		} else if strings.TrimSpace(settings.InvalidReplyText) != "" {
			_ = a.sendAndSaveTextMessage(account, contact, settings.InvalidReplyText)
		}
		_ = db.Save(session).Error
		a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
			ContactID:        &contact.ID,
			SessionID:        &session.ID,
			InstanceID:       contact.InstanceID,
			WhatsAppAccount:  account.Name,
			EventType:        models.AgentSelectionEventInvalidReplyReceived,
			ActorType:        models.AgentSelectionActorCustomer,
			InboundMessageID: &inbound.ID,
			Reason:           replyText,
		})
		return
	}

	session.ProcessedInboundIDs = append(session.ProcessedInboundIDs, inbound.ID.String())
	session.SelectedOptionID = selected.OptionID
	session.SelectedUserID = selected.UserID
	session.SelectedTeamID = selected.TeamID

	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:        &contact.ID,
		SessionID:        &session.ID,
		InstanceID:       contact.InstanceID,
		WhatsAppAccount:  account.Name,
		EventType:        models.AgentSelectionEventValidReplyReceived,
		ActorType:        models.AgentSelectionActorCustomer,
		SelectedOptionID: selected.OptionID,
		SelectedAgentID:  selected.UserID,
		SelectedTeamID:   selected.TeamID,
		InboundMessageID: &inbound.ID,
	})

	switch selected.Type {
	case models.AgentSelectionOptionAgent:
		a.commitAgentSelectionAgent(db, account, contact, inbound, session, settings, selected)
	case models.AgentSelectionOptionTeam:
		a.commitAgentSelectionTeam(db, account, contact, inbound, session, selected)
	case models.AgentSelectionOptionQueue:
		a.commitAgentSelectionQueue(db, account, contact, inbound, session, selected)
	case models.AgentSelectionOptionCustom:
		a.commitAgentSelectionCustom(db, account, contact, inbound, session, settings, selected)
	default:
		session.Status = models.AgentSelectionSessionError
		_ = db.Save(session).Error
	}
}

func (a *App) commitAgentSelectionAgent(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, session *models.AgentSelectionSession, settings *models.AgentSelectionSettings, selected agentSelectionRenderedOption) {
	if selected.UserID == nil {
		session.Status = models.AgentSelectionSessionError
		_ = db.Save(session).Error
		return
	}
	var latestContact models.Contact
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", contact.ID, account.OrganizationID).First(&latestContact).Error; err != nil {
		return
	}
	if latestContact.AssignedUserID != nil || normalizeContactStatus(&latestContact) != models.ChatStatusPending {
		session.Status = models.AgentSelectionSessionCancelled
		_ = db.Save(session).Error
		return
	}

	var user models.User
	if err := db.Where("id = ? AND is_active = ?", *selected.UserID, true).First(&user).Error; err != nil || !user.IsAvailable {
		if strings.TrimSpace(settings.UnavailableAgentText) != "" {
			_ = a.sendAndSaveTextMessage(account, contact, settings.UnavailableAgentText)
		}
		a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
			ContactID:        &contact.ID,
			SessionID:        &session.ID,
			InstanceID:       contact.InstanceID,
			WhatsAppAccount:  account.Name,
			EventType:        models.AgentSelectionEventAgentUnavailable,
			ActorType:        models.AgentSelectionActorSystem,
			SelectedOptionID: selected.OptionID,
			SelectedAgentID:  selected.UserID,
			InboundMessageID: &inbound.ID,
		})
		_ = db.Save(session).Error
		return
	}
	ok, err := a.canUserSeeContactInstance(account.OrganizationID, *selected.UserID, &latestContact)
	if err != nil || !ok || !a.userBelongsToOrg(db, *selected.UserID, account.OrganizationID) {
		if strings.TrimSpace(settings.UnavailableAgentText) != "" {
			_ = a.sendAndSaveTextMessage(account, contact, settings.UnavailableAgentText)
		}
		return
	}

	previousAssignee := latestContact.AssignedUserID
	if err := db.Model(&latestContact).Updates(chatAssignmentUpdates(selected.UserID)).Error; err != nil {
		session.Status = models.AgentSelectionSessionError
		_ = db.Save(session).Error
		return
	}
	session.Status = models.AgentSelectionSessionSelected
	_ = db.Save(session).Error
	a.appendSystemChatMessage(&latestContact, fmt.Sprintf("System :%s assigned this chat to %s after customer selection", agentSelectionSystemActorName, selected.Label), models.JSONB{
		"system_event":        true,
		"event_type":          "customer_agent_selected",
		"assigned_to_user_id": selected.UserID.String(),
		"assigned_to_name":    selected.Label,
		"session_id":          session.ID.String(),
	})
	a.broadcastContactLifecycleUpdate(account.OrganizationID, &latestContact, true)
	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:              &contact.ID,
		SessionID:              &session.ID,
		InstanceID:             contact.InstanceID,
		WhatsAppAccount:        account.Name,
		EventType:              models.AgentSelectionEventAgentAssigned,
		ActorType:              models.AgentSelectionActorSystem,
		SelectedOptionID:       selected.OptionID,
		SelectedAgentID:        selected.UserID,
		PreviousAssignedUserID: previousAssignee,
		NewAssignedUserID:      selected.UserID,
		InboundMessageID:       &inbound.ID,
	})
}

func (a *App) commitAgentSelectionTeam(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, session *models.AgentSelectionSession, selected agentSelectionRenderedOption) {
	if selected.TeamID == nil || a.hasActiveAgentTransfer(account.OrganizationID, contact.ID) {
		session.Status = models.AgentSelectionSessionCancelled
		_ = db.Save(session).Error
		return
	}
	a.createTransferToTeam(account, contact, *selected.TeamID, "Customer selected team: "+selected.Label, models.TransferSourceCustomerSelection)
	session.Status = models.AgentSelectionSessionSelected
	_ = db.Save(session).Error
	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:        &contact.ID,
		SessionID:        &session.ID,
		InstanceID:       contact.InstanceID,
		WhatsAppAccount:  account.Name,
		EventType:        models.AgentSelectionEventTeamTransferCreated,
		ActorType:        models.AgentSelectionActorSystem,
		SelectedOptionID: selected.OptionID,
		SelectedTeamID:   selected.TeamID,
		InboundMessageID: &inbound.ID,
	})
}

func (a *App) commitAgentSelectionQueue(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, session *models.AgentSelectionSession, selected agentSelectionRenderedOption) {
	if a.hasActiveAgentTransfer(account.OrganizationID, contact.ID) {
		session.Status = models.AgentSelectionSessionCancelled
		_ = db.Save(session).Error
		return
	}
	a.createTransferToQueue(account, contact, models.TransferSourceCustomerSelection)
	session.Status = models.AgentSelectionSessionSelected
	_ = db.Save(session).Error
	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:        &contact.ID,
		SessionID:        &session.ID,
		InstanceID:       contact.InstanceID,
		WhatsAppAccount:  account.Name,
		EventType:        models.AgentSelectionEventQueueTransferCreated,
		ActorType:        models.AgentSelectionActorSystem,
		SelectedOptionID: selected.OptionID,
		InboundMessageID: &inbound.ID,
	})
}

func (a *App) commitAgentSelectionCustom(db *gorm.DB, account *models.WhatsAppAccount, contact *models.Contact, inbound *models.Message, session *models.AgentSelectionSession, settings *models.AgentSelectionSettings, selected agentSelectionRenderedOption) {
	if strings.TrimSpace(selected.Response) != "" {
		_ = a.sendAndSaveTextMessage(account, contact, selected.Response)
	}
	switch models.AgentSelectionCustomAction(selected.Action) {
	case models.AgentSelectionCustomActionAssignToTeam:
		if settings.CustomFinalOptionTeamID != nil {
			a.createTransferToTeam(account, contact, *settings.CustomFinalOptionTeamID, "Customer selected custom option: "+selected.Label, models.TransferSourceCustomerSelection)
		}
	case models.AgentSelectionCustomActionCloseChat:
		now := time.Now()
		_ = db.Model(&models.Contact{}).Where("id = ? AND organization_id = ?", contact.ID, account.OrganizationID).Updates(map[string]any{
			"status":    models.ChatStatusClosed,
			"closed_at": now,
		}).Error
	default:
		// send_only and keep_pending intentionally preserve pending/unassigned state.
	}
	session.Status = models.AgentSelectionSessionSelected
	_ = db.Save(session).Error
	a.writeAgentSelectionAudit(db, account.OrganizationID, agentSelectionAuditInput{
		ContactID:        &contact.ID,
		SessionID:        &session.ID,
		InstanceID:       contact.InstanceID,
		WhatsAppAccount:  account.Name,
		EventType:        models.AgentSelectionEventCustomActionCompleted,
		ActorType:        models.AgentSelectionActorSystem,
		SelectedOptionID: selected.OptionID,
		InboundMessageID: &inbound.ID,
		Metadata:         models.JSONB{"action": selected.Action},
	})
}

func (a *App) ProcessAgentSelectionDueSessions(ctx context.Context, limit int) {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now()
	var sessions []models.AgentSelectionSession
	if err := a.DB.Where("status = ? AND prompt_due_at <= ?", models.AgentSelectionSessionWaitingDelay, now).
		Order("prompt_due_at ASC").
		Limit(limit).
		Find(&sessions).Error; err != nil {
		a.Log.Error("Failed to load due agent selection sessions", "error", err)
		return
	}
	for i := range sessions {
		select {
		case <-ctx.Done():
			return
		default:
			a.sendAgentSelectionPromptIfDue(&sessions[i])
		}
	}

	var expired []models.AgentSelectionSession
	if err := a.DB.Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", models.AgentSelectionSessionMenuSent, now).
		Limit(limit).
		Find(&expired).Error; err != nil {
		a.Log.Error("Failed to load expired agent selection sessions", "error", err)
		return
	}
	for i := range expired {
		a.expireAgentSelectionSession(&expired[i])
	}
}

func (a *App) sendAgentSelectionPromptIfDue(session *models.AgentSelectionSession) {
	var contact models.Contact
	if err := a.DB.Where("id = ? AND organization_id = ?", session.ContactID, session.OrganizationID).First(&contact).Error; err != nil {
		return
	}
	settings, err := a.resolveAgentSelectionSettings(a.DB, session.OrganizationID, session.InstanceID)
	if err != nil || settings == nil || !settings.Enabled {
		session.Status = models.AgentSelectionSessionExpired
		_ = a.DB.Save(session).Error
		return
	}
	if !agentSelectionSettingsAppliesToInstance(settings, session.InstanceID) {
		session.Status = models.AgentSelectionSessionCancelled
		_ = a.DB.Save(session).Error
		return
	}
	if contact.AssignedUserID != nil || normalizeContactStatus(&contact) != models.ChatStatusPending {
		session.Status = models.AgentSelectionSessionCancelled
		_ = a.DB.Save(session).Error
		a.writeAgentSelectionAudit(a.DB, session.OrganizationID, agentSelectionAuditInput{
			ContactID:       &contact.ID,
			SessionID:       &session.ID,
			InstanceID:      contact.InstanceID,
			WhatsAppAccount: session.WhatsAppAccount,
			EventType:       models.AgentSelectionEventPromptSkippedAssigned,
			ActorType:       models.AgentSelectionActorSystem,
		})
		return
	}
	if a.hasActiveAgentTransfer(session.OrganizationID, contact.ID) {
		session.Status = models.AgentSelectionSessionCancelled
		_ = a.DB.Save(session).Error
		a.writeAgentSelectionAudit(a.DB, session.OrganizationID, agentSelectionAuditInput{
			ContactID:       &contact.ID,
			SessionID:       &session.ID,
			InstanceID:      contact.InstanceID,
			WhatsAppAccount: session.WhatsAppAccount,
			EventType:       models.AgentSelectionEventPromptSkippedActiveTransfer,
			ActorType:       models.AgentSelectionActorSystem,
		})
		return
	}
	var account models.WhatsAppAccount
	if err := a.DB.Where("organization_id = ? AND name = ?", session.OrganizationID, session.WhatsAppAccount).First(&account).Error; err != nil {
		session.Status = models.AgentSelectionSessionError
		_ = a.DB.Save(session).Error
		return
	}
	menu, err := a.buildAgentSelectionMenu(a.DB, session.OrganizationID, settings, &contact)
	if err != nil || len(menu.Options) == 0 {
		session.Status = models.AgentSelectionSessionError
		_ = a.DB.Save(session).Error
		return
	}
	msg, err := a.SendOutgoingMessage(context.Background(), OutgoingMessageRequest{
		Account: &account,
		Contact: &contact,
		Type:    models.MessageTypeText,
		Content: menu.Text,
	}, ChatbotSendOptions())
	if err != nil {
		session.Status = models.AgentSelectionSessionError
		_ = a.DB.Save(session).Error
		a.writeAgentSelectionAudit(a.DB, session.OrganizationID, agentSelectionAuditInput{
			ContactID:       &contact.ID,
			SessionID:       &session.ID,
			InstanceID:      contact.InstanceID,
			WhatsAppAccount: session.WhatsAppAccount,
			EventType:       models.AgentSelectionEventMenuSendFailed,
			ActorType:       models.AgentSelectionActorSystem,
			Reason:          err.Error(),
		})
		return
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(clampInt(settings.SelectionTimeoutMinutes, 1, 24*60)) * time.Minute)
	session.Status = models.AgentSelectionSessionMenuSent
	session.MenuSentAt = &now
	session.ExpiresAt = &expiresAt
	session.RenderedOptionsSnapshot = renderedOptionsToJSONBArray(menu.Options)
	if msg != nil {
		session.PromptMessageID = &msg.ID
	}
	_ = a.DB.Save(session).Error
	var outboundID *uuid.UUID
	if msg != nil {
		outboundID = &msg.ID
	}
	a.writeAgentSelectionAudit(a.DB, session.OrganizationID, agentSelectionAuditInput{
		ContactID:         &contact.ID,
		SessionID:         &session.ID,
		InstanceID:        contact.InstanceID,
		WhatsAppAccount:   session.WhatsAppAccount,
		EventType:         models.AgentSelectionEventMenuSent,
		ActorType:         models.AgentSelectionActorSystem,
		OutboundMessageID: outboundID,
		Metadata:          models.JSONB{"expires_at": expiresAt.Format(time.RFC3339)},
	})
}

func (a *App) expireAgentSelectionSession(session *models.AgentSelectionSession) {
	session.Status = models.AgentSelectionSessionTimeout
	_ = a.DB.Save(session).Error
	a.writeAgentSelectionAudit(a.DB, session.OrganizationID, agentSelectionAuditInput{
		ContactID:       &session.ContactID,
		SessionID:       &session.ID,
		InstanceID:      session.InstanceID,
		WhatsAppAccount: session.WhatsAppAccount,
		EventType:       models.AgentSelectionEventSelectionTimeout,
		ActorType:       models.AgentSelectionActorSystem,
	})
	if session.Metadata != nil {
		if response, _ := session.Metadata["timeout_response_text"].(string); strings.TrimSpace(response) != "" {
			// Reserved for future per-session timeout text snapshots.
		}
	}
}

func (a *App) writeAgentSelectionAudit(db *gorm.DB, orgID uuid.UUID, input agentSelectionAuditInput) {
	if db == nil || orgID == uuid.Nil || input.EventType == "" {
		return
	}
	actor := input.ActorType
	if actor == "" {
		actor = models.AgentSelectionActorSystem
	}
	event := models.AgentSelectionAuditEvent{
		OrganizationID:         orgID,
		ContactID:              input.ContactID,
		SessionID:              input.SessionID,
		InstanceID:             input.InstanceID,
		WhatsAppAccount:        input.WhatsAppAccount,
		EventType:              input.EventType,
		ActorType:              actor,
		ActorUserID:            input.ActorUserID,
		SelectedOptionID:       input.SelectedOptionID,
		SelectedAgentID:        input.SelectedAgentID,
		SelectedTeamID:         input.SelectedTeamID,
		PreviousAssignedUserID: input.PreviousAssignedUserID,
		NewAssignedUserID:      input.NewAssignedUserID,
		TransferID:             input.TransferID,
		InboundMessageID:       input.InboundMessageID,
		OutboundMessageID:      input.OutboundMessageID,
		Reason:                 input.Reason,
		Metadata:               input.Metadata,
	}
	if event.Metadata == nil {
		event.Metadata = models.JSONB{}
	}
	if err := db.Create(&event).Error; err != nil && a != nil {
		a.Log.Error("Failed to write agent selection audit event", "error", err, "event_type", input.EventType)
	}
}

type AgentSelectionProcessor struct {
	app      *App
	interval time.Duration
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewAgentSelectionProcessor(app *App, interval time.Duration) *AgentSelectionProcessor {
	if interval <= 0 {
		interval = time.Minute
	}
	return &AgentSelectionProcessor{app: app, interval: interval, stopCh: make(chan struct{})}
}

func (p *AgentSelectionProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	p.app.ProcessAgentSelectionDueSessions(ctx, 100)
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.app.ProcessAgentSelectionDueSessions(ctx, 100)
		}
	}
}

func (p *AgentSelectionProcessor) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

func renderedOptionsToJSONBArray(options []agentSelectionRenderedOption) models.JSONBArray {
	out := make(models.JSONBArray, 0, len(options))
	for _, option := range options {
		item := map[string]any{
			"number":    option.Number,
			"option_id": option.OptionID,
			"type":      string(option.Type),
			"label":     option.Label,
			"action":    option.Action,
			"response":  option.Response,
		}
		if option.UserID != nil {
			item["user_id"] = option.UserID.String()
		}
		if option.TeamID != nil {
			item["team_id"] = option.TeamID.String()
		}
		out = append(out, item)
	}
	return out
}

func selectedRenderedOption(snapshot models.JSONBArray, replyText string) (agentSelectionRenderedOption, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(replyText))
	if err != nil {
		return agentSelectionRenderedOption{}, false
	}
	for _, raw := range snapshot {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		itemNumber := intFromAny(item["number"])
		if itemNumber != number {
			continue
		}
		option := agentSelectionRenderedOption{
			Number:   itemNumber,
			OptionID: stringFromAny(item["option_id"]),
			Type:     models.AgentSelectionOptionType(stringFromAny(item["type"])),
			Label:    stringFromAny(item["label"]),
			Action:   stringFromAny(item["action"]),
			Response: stringFromAny(item["response"]),
		}
		if userIDRaw := stringFromAny(item["user_id"]); userIDRaw != "" {
			if parsed, parseErr := uuid.Parse(userIDRaw); parseErr == nil {
				option.UserID = &parsed
			}
		}
		if teamIDRaw := stringFromAny(item["team_id"]); teamIDRaw != "" {
			if parsed, parseErr := uuid.Parse(teamIDRaw); parseErr == nil {
				option.TeamID = &parsed
			}
		}
		return option, true
	}
	return agentSelectionRenderedOption{}, false
}

func sessionHasProcessedInbound(session *models.AgentSelectionSession, inboundID uuid.UUID) bool {
	if session == nil || inboundID == uuid.Nil {
		return false
	}
	needle := inboundID.String()
	for _, existing := range session.ProcessedInboundIDs {
		if existing == needle {
			return true
		}
	}
	return false
}

func normalizeStringArray(values []string) models.StringArray {
	out := make(models.StringArray, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (a *App) normalizeAgentSelectionAllowedInstanceIDs(db *gorm.DB, orgID uuid.UUID, values []string) (models.StringArray, error) {
	normalized := normalizeStringArray(values)
	if len(normalized) == 0 {
		return models.StringArray{}, nil
	}

	ids := make([]uuid.UUID, 0, len(normalized))
	seen := map[uuid.UUID]struct{}{}
	for _, value := range normalized {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("Invalid allowed_instance_ids")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	var count int64
	if err := db.Model(&models.WhatsAppInstance{}).
		Where("organization_id = ? AND id IN ?", orgID, ids).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count != int64(len(ids)) {
		return nil, fmt.Errorf("One or more allowed instances were not found")
	}

	out := make(models.StringArray, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out, nil
}

func agentSelectionSettingsAppliesToInstance(settings *models.AgentSelectionSettings, instanceID *uuid.UUID) bool {
	if settings == nil {
		return false
	}
	if settings.InstanceID != nil && *settings.InstanceID != uuid.Nil {
		return instanceID != nil && *instanceID == *settings.InstanceID
	}
	if len(settings.AllowedInstanceIDs) == 0 {
		return true
	}
	if instanceID == nil || *instanceID == uuid.Nil {
		return false
	}
	needle := instanceID.String()
	for _, allowed := range settings.AllowedInstanceIDs {
		if strings.EqualFold(strings.TrimSpace(allowed), needle) {
			return true
		}
	}
	return false
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		i, _ := strconv.Atoi(typed.String())
		return i
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}
