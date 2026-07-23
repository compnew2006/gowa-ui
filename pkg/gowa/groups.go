package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GroupInfo represents metadata about a WhatsApp group.
// GOWA returns this as free-form, so we decode into a map.
type GroupInfo map[string]any

// GroupInfoFromLink represents group info obtained from an invite link.
type GroupInfoFromLink struct {
	GroupID          string `json:"group_id"`
	Name             string `json:"name"`
	Topic            string `json:"topic"`
	CreatedAt        string `json:"created_at"`
	ParticipantCount int    `json:"participant_count"`
	IsLocked         bool   `json:"is_locked"`
	IsAnnounce       bool   `json:"is_announce"`
	IsEphemeral      bool   `json:"is_ephemeral"`
	Description      string `json:"description"`
}

// ParticipantResult represents the result of a participant management operation.
type ParticipantResult struct {
	Participant string `json:"participant"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// ParticipantRequest represents a pending join request.
type ParticipantRequest struct {
	JID         string `json:"jid"`
	PhoneNumber string `json:"phone_number"`
	DisplayName string `json:"display_name"`
	RequestedAt string `json:"requested_at"`
}

// ExportGroupParticipants exports the group participant list as CSV.
// GOWA endpoint: GET /group/participants/export?group_id={groupID}
// Returns the raw CSV bytes.
func (c *Client) ExportGroupParticipants(ctx context.Context, deviceID, groupID string) ([]byte, error) {
	path := fmt.Sprintf("/group/participants/export?group_id=%s", url.QueryEscape(groupID))
	return c.doRaw(ctx, "GET", path, deviceID)
}

// GetGroupInfo retrieves metadata about a group.
// GOWA endpoint: GET /group/info?group_id={groupID}
func (c *Client) GetGroupInfo(ctx context.Context, deviceID, groupID string) (GroupInfo, error) {
	path := fmt.Sprintf("/group/info?group_id=%s", url.QueryEscape(groupID))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results GroupInfo `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse group info response: %w", err)
	}
	return resp.Results, nil
}

// RemoveParticipants removes participants from a group.
// GOWA endpoint: POST /group/participants/remove
func (c *Client) RemoveParticipants(ctx context.Context, deviceID, groupID string, participants []string) ([]ParticipantResult, error) {
	return c.manageParticipants(ctx, deviceID, "/group/participants/remove", groupID, participants)
}

// PromoteParticipants promotes participants to admin.
// GOWA endpoint: POST /group/participants/promote
func (c *Client) PromoteParticipants(ctx context.Context, deviceID, groupID string, participants []string) ([]ParticipantResult, error) {
	return c.manageParticipants(ctx, deviceID, "/group/participants/promote", groupID, participants)
}

// DemoteParticipants demotes admins to regular participants.
// GOWA endpoint: POST /group/participants/demote
func (c *Client) DemoteParticipants(ctx context.Context, deviceID, groupID string, participants []string) ([]ParticipantResult, error) {
	return c.manageParticipants(ctx, deviceID, "/group/participants/demote", groupID, participants)
}

// manageParticipants is the shared helper for participant CRUD operations.
func (c *Client) manageParticipants(ctx context.Context, deviceID, path, groupID string, participants []string) ([]ParticipantResult, error) {
	body := map[string]any{
		"group_id":     groupID,
		"participants": participants,
	}
	rawBody, err := c.doJSONRaw(ctx, "POST", path, deviceID, body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results []ParticipantResult `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse participant management response: %w", err)
	}
	return resp.Results, nil
}

// JoinGroupWithLink joins a group using an invite link.
// GOWA endpoint: POST /group/join-with-link
func (c *Client) JoinGroupWithLink(ctx context.Context, deviceID, inviteLink string) error {
	body := map[string]any{"link": inviteLink}
	_, err := c.doJSON(ctx, "POST", "/group/join-with-link", deviceID, body)
	return err
}

// GetGroupInfoFromLink retrieves group info from an invite link without joining.
// GOWA endpoint: GET /group/info-from-link?link={link}
func (c *Client) GetGroupInfoFromLink(ctx context.Context, deviceID, inviteLink string) (*GroupInfoFromLink, error) {
	path := fmt.Sprintf("/group/info-from-link?link=%s", url.QueryEscape(inviteLink))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results GroupInfoFromLink `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse group info from link response: %w", err)
	}
	return &resp.Results, nil
}

// GetParticipantRequests lists pending join requests for a group.
// GOWA endpoint: GET /group/participant-requests?group_id={groupID}
func (c *Client) GetParticipantRequests(ctx context.Context, deviceID, groupID string) ([]ParticipantRequest, error) {
	path := fmt.Sprintf("/group/participant-requests?group_id=%s", url.QueryEscape(groupID))
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results []ParticipantRequest `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, fmt.Errorf("parse participant requests response: %w", err)
	}
	return resp.Results, nil
}

// ApproveParticipantRequests approves pending join requests.
// GOWA endpoint: POST /group/participant-requests/approve
func (c *Client) ApproveParticipantRequests(ctx context.Context, deviceID, groupID string, participants []string) error {
	body := map[string]any{"group_id": groupID, "participants": participants}
	_, err := c.doJSON(ctx, "POST", "/group/participant-requests/approve", deviceID, body)
	return err
}

// RejectParticipantRequests rejects pending join requests.
// GOWA endpoint: POST /group/participant-requests/reject
func (c *Client) RejectParticipantRequests(ctx context.Context, deviceID, groupID string, participants []string) error {
	body := map[string]any{"group_id": groupID, "participants": participants}
	_, err := c.doJSON(ctx, "POST", "/group/participant-requests/reject", deviceID, body)
	return err
}

// SetGroupLocked locks/unlocks group settings (only admins can edit info).
// GOWA endpoint: POST /group/locked
func (c *Client) SetGroupLocked(ctx context.Context, deviceID, groupID string, locked bool) error {
	body := map[string]any{"group_id": groupID, "locked": locked}
	_, err := c.doJSON(ctx, "POST", "/group/locked", deviceID, body)
	return err
}

// SetGroupAnnounce toggles announce mode (only admins can send messages).
// GOWA endpoint: POST /group/announce
func (c *Client) SetGroupAnnounce(ctx context.Context, deviceID, groupID string, announce bool) error {
	body := map[string]any{"group_id": groupID, "announce": announce}
	_, err := c.doJSON(ctx, "POST", "/group/announce", deviceID, body)
	return err
}

// SetGroupTopic sets the group topic/description.
// GOWA endpoint: POST /group/topic
func (c *Client) SetGroupTopic(ctx context.Context, deviceID, groupID, topic string) error {
	body := map[string]any{"group_id": groupID, "topic": topic}
	_, err := c.doJSON(ctx, "POST", "/group/topic", deviceID, body)
	return err
}
