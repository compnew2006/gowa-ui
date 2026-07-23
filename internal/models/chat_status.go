package models

import "time"

// ChatStatus represents the lifecycle state of a conversation.
// Stored in Contact.Metadata["chat_status"] as a string — no GORM column needed.
type ChatStatus string

const (
	ChatStatusPending ChatStatus = "pending" // new/unassigned — awaiting agent claim
	ChatStatusOpen    ChatStatus = "open"    // assigned to an agent — actively handled
	ChatStatusClosed  ChatStatus = "closed"  // ended — read-only
)

// MaxCollaborators is the maximum number of collaborators per conversation.
const MaxCollaborators = 10

// EffectiveStatus reads the chat status from the contact's metadata.
// Returns ChatStatusOpen if the key is absent (backward compatibility).
func (c *Contact) EffectiveStatus() ChatStatus {
	if c.Metadata == nil {
		return ChatStatusOpen
	}
	s, ok := c.Metadata["chat_status"].(string)
	if !ok {
		return ChatStatusOpen
	}
	switch ChatStatus(s) {
	case ChatStatusPending:
		return ChatStatusPending
	case ChatStatusClosed:
		return ChatStatusClosed
	default:
		return ChatStatusOpen
	}
}

// SetStatus writes the chat status to the contact's metadata.
func (c *Contact) SetStatus(s ChatStatus) {
	if c.Metadata == nil {
		c.Metadata = JSONB{}
	}
	c.Metadata["chat_status"] = string(s)
}

// Collaborator represents a non-primary participant in a conversation.
// Stored in Contact.Metadata["collaborators"] as a JSON array.
type Collaborator struct {
	UserID   string    `json:"user_id"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// GetCollaborators parses the collaborators array from the contact's metadata.
func (c *Contact) GetCollaborators() []Collaborator {
	if c.Metadata == nil {
		return nil
	}
	raw, ok := c.Metadata["collaborators"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []Collaborator
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		collab := Collaborator{}
		if v, ok := m["user_id"].(string); ok {
			collab.UserID = v
		}
		if v, ok := m["name"].(string); ok {
			collab.Name = v
		}
		if v, ok := m["role"].(string); ok {
			collab.Role = v
		}
		if v, ok := m["joined_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				collab.JoinedAt = t
			}
		}
		result = append(result, collab)
	}
	return result
}

// IsCollaborator checks whether the given user ID is in the collaborators list.
func (c *Contact) IsCollaborator(userID string) bool {
	for _, collab := range c.GetCollaborators() {
		if collab.UserID == userID {
			return true
		}
	}
	return false
}

// AddCollaborator appends a collaborator to the metadata array.
// Does nothing if the user is already a collaborator (dedup).
func (c *Contact) AddCollaborator(user Collaborator) {
	if c.Metadata == nil {
		c.Metadata = JSONB{}
	}
	if c.IsCollaborator(user.UserID) {
		return
	}
	collaborators := c.GetCollaborators()
	collaborators = append(collaborators, user)
	// Convert to []any for JSONB storage
	arr := make([]any, len(collaborators))
	for i, collab := range collaborators {
		arr[i] = map[string]any{
			"user_id":   collab.UserID,
			"name":      collab.Name,
			"role":      collab.Role,
			"joined_at": collab.JoinedAt.Format(time.RFC3339),
		}
	}
	c.Metadata["collaborators"] = arr
}

// RemoveCollaborator removes a collaborator from the metadata array by user ID.
func (c *Contact) RemoveCollaborator(userID string) {
	if c.Metadata == nil {
		return
	}
	collaborators := c.GetCollaborators()
	var filtered []Collaborator
	for _, collab := range collaborators {
		if collab.UserID != userID {
			filtered = append(filtered, collab)
		}
	}
	// Rebuild as []any for JSONB storage
	if len(filtered) == 0 {
		delete(c.Metadata, "collaborators")
		return
	}
	arr := make([]any, len(filtered))
	for i, collab := range filtered {
		arr[i] = map[string]any{
			"user_id":   collab.UserID,
			"name":      collab.Name,
			"role":      collab.Role,
			"joined_at": collab.JoinedAt.Format(time.RFC3339),
		}
	}
	c.Metadata["collaborators"] = arr
}

// ClearCollaborators removes all collaborators from the metadata.
func (c *Contact) ClearCollaborators() {
	if c.Metadata == nil {
		return
	}
	delete(c.Metadata, "collaborators")
}

// HasParticipants returns true if the conversation has an owner or any collaborators.
func (c *Contact) HasParticipants() bool {
	return c.AssignedUserID != nil || len(c.GetCollaborators()) > 0
}
