package handlers

import (
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// GroupParticipantResponse represents a group participant in API responses.
type GroupParticipantResponse struct {
	JID         string `json:"jid"`
	PhoneNumber string `json:"phone_number"`
	IsAdmin     bool   `json:"is_admin"`
	IsSuperAdmin bool  `json:"is_super_admin"`
}

// GroupParticipantsRequest represents a request to manage group participants.
type GroupParticipantsRequest struct {
	InstanceID   string   `json:"instance_id" validate:"required"`
	GroupJID     string   `json:"group_jid" validate:"required"`
	Participants []string `json:"participants" validate:"required,min=1"`
}

// ListGroupMembers returns all participants of a WhatsApp group.
func (a *App) ListGroupMembers(r *fastglue.Request) error {
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group participants are only available for whatsmeow instances", nil, "")
	}

	groupJID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("group_jid")))
	if groupJID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "group_jid query parameter is required", nil, "")
	}

	instanceID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id")))
	if instanceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "instance_id query parameter is required", nil, "")
	}
	if _, err := uuid.Parse(instanceID); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID format", nil, "")
	}

	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "WhatsApp instance not found", nil, "")
	}

	gpp, ok := a.MessageProvider.(provider.GroupParticipantProvider)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Group participant provider not available", nil, "")
	}

	participants, err := gpp.GetGroupParticipants(r.RequestCtx, instanceID, groupJID)
	if err != nil {
		a.Log.Error("Failed to list group participants", "error", err, "instance_id", instanceID, "group_jid", groupJID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list group participants", nil, "")
	}

	response := make([]GroupParticipantResponse, len(participants))
	for i, p := range participants {
		response[i] = GroupParticipantResponse{
			JID:          p.JID,
			PhoneNumber:  p.PhoneNumber,
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
		}
	}

	return r.SendEnvelope(map[string]any{
		"participants": response,
		"total":        len(response),
	})
}

// AddGroupMembers adds participants to a WhatsApp group.
func (a *App) AddGroupMembers(r *fastglue.Request) error {
	return a.manageGroupParticipants(r, "add")
}

// RemoveGroupMembers removes participants from a WhatsApp group.
func (a *App) RemoveGroupMembers(r *fastglue.Request) error {
	return a.manageGroupParticipants(r, "remove")
}

// PromoteGroupMembers promotes participants to group admin.
func (a *App) PromoteGroupMembers(r *fastglue.Request) error {
	return a.manageGroupParticipants(r, "promote")
}

// DemoteGroupMembers demotes participants from group admin.
func (a *App) DemoteGroupMembers(r *fastglue.Request) error {
	return a.manageGroupParticipants(r, "demote")
}

// manageGroupParticipants is the shared implementation for add/remove/promote/demote.
func (a *App) manageGroupParticipants(r *fastglue.Request, action string) error {
	if !a.isWhatsmeowProvider() {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Group participants are only available for whatsmeow instances", nil, "")
	}

	var req GroupParticipantsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	if req.GroupJID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "group_jid is required", nil, "")
	}
	if req.InstanceID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "instance_id is required", nil, "")
	}
	if _, err := uuid.Parse(req.InstanceID); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance ID format", nil, "")
	}
	if len(req.Participants) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "At least one participant is required", nil, "")
	}

	orgID, _, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := a.DB.Where("id = ? AND organization_id = ?", req.InstanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "WhatsApp instance not found", nil, "")
	}

	gpp, ok := a.MessageProvider.(provider.GroupParticipantProvider)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Group participant provider not available", nil, "")
	}

	ctx := r.RequestCtx
	var participants []provider.GroupParticipant

	switch action {
	case "add":
		participants, err = gpp.AddGroupParticipants(ctx, req.InstanceID, req.GroupJID, req.Participants)
	case "remove":
		participants, err = gpp.RemoveGroupParticipants(ctx, req.InstanceID, req.GroupJID, req.Participants)
	case "promote":
		participants, err = gpp.PromoteGroupParticipants(ctx, req.InstanceID, req.GroupJID, req.Participants)
	case "demote":
		participants, err = gpp.DemoteGroupParticipants(ctx, req.InstanceID, req.GroupJID, req.Participants)
	default:
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid action", nil, "")
	}

	if err != nil {
		a.Log.Error("Failed to "+action+" group participants", "error", err, "instance_id", req.InstanceID, "group_jid", req.GroupJID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to "+action+" participants", nil, "")
	}

	response := make([]GroupParticipantResponse, len(participants))
	for i, p := range participants {
		response[i] = GroupParticipantResponse{
			JID:          p.JID,
			PhoneNumber:  p.PhoneNumber,
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
		}
	}

	return r.SendEnvelope(map[string]any{
		"action":       action,
		"participants": response,
		"affected":     len(response),
	})
}
