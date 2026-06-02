package models

import (
	"time"

	"github.com/google/uuid"
)

type AgentSelectionTriggerMode string

const (
	AgentSelectionTriggerFirstPendingMessage AgentSelectionTriggerMode = "first_pending_message"
	AgentSelectionTriggerKeyword             AgentSelectionTriggerMode = "keyword"
	AgentSelectionTriggerAfterOfficeHours    AgentSelectionTriggerMode = "after_office_hours"
	AgentSelectionTriggerChatbotStep         AgentSelectionTriggerMode = "chatbot_step"
	AgentSelectionTriggerManualTest          AgentSelectionTriggerMode = "manual_test"
)

type AgentSelectionCustomAction string

const (
	AgentSelectionCustomActionSendOnly     AgentSelectionCustomAction = "send_only"
	AgentSelectionCustomActionKeepPending  AgentSelectionCustomAction = "keep_pending"
	AgentSelectionCustomActionCloseChat    AgentSelectionCustomAction = "close_chat"
	AgentSelectionCustomActionAssignToTeam AgentSelectionCustomAction = "assign_to_team"
)

type AgentSelectionOptionType string

const (
	AgentSelectionOptionAgent  AgentSelectionOptionType = "agent"
	AgentSelectionOptionTeam   AgentSelectionOptionType = "team"
	AgentSelectionOptionQueue  AgentSelectionOptionType = "queue"
	AgentSelectionOptionCustom AgentSelectionOptionType = "custom"
)

type AgentSelectionSessionStatus string

const (
	AgentSelectionSessionWaitingDelay AgentSelectionSessionStatus = "waiting_delay"
	AgentSelectionSessionMenuSent     AgentSelectionSessionStatus = "menu_sent"
	AgentSelectionSessionSelected     AgentSelectionSessionStatus = "selected"
	AgentSelectionSessionTimeout      AgentSelectionSessionStatus = "timeout"
	AgentSelectionSessionCancelled    AgentSelectionSessionStatus = "cancelled"
	AgentSelectionSessionExpired      AgentSelectionSessionStatus = "expired"
	AgentSelectionSessionError        AgentSelectionSessionStatus = "error"
)

type AgentSelectionAuditActor string

const (
	AgentSelectionActorCustomer AgentSelectionAuditActor = "customer"
	AgentSelectionActorSystem   AgentSelectionAuditActor = "system"
	AgentSelectionActorAdmin    AgentSelectionAuditActor = "admin"
	AgentSelectionActorAgent    AgentSelectionAuditActor = "agent"
)

const (
	AgentSelectionEventSessionCreated              = "session_created"
	AgentSelectionEventDelayStarted                = "delay_started"
	AgentSelectionEventPromptSkippedAssigned       = "prompt_skipped_assigned"
	AgentSelectionEventPromptSkippedActiveTransfer = "prompt_skipped_active_transfer"
	AgentSelectionEventMenuSent                    = "menu_sent"
	AgentSelectionEventMenuSendFailed              = "menu_send_failed"
	AgentSelectionEventValidReplyReceived          = "valid_reply_received"
	AgentSelectionEventInvalidReplyReceived        = "invalid_reply_received"
	AgentSelectionEventMaxInvalidAttemptsReached   = "max_invalid_attempts_reached"
	AgentSelectionEventAgentSelected               = "agent_selected"
	AgentSelectionEventAgentUnavailable            = "agent_unavailable"
	AgentSelectionEventAgentAssigned               = "agent_assigned"
	AgentSelectionEventTeamSelected                = "team_selected"
	AgentSelectionEventTeamTransferCreated         = "team_transfer_created"
	AgentSelectionEventQueueSelected               = "queue_selected"
	AgentSelectionEventQueueTransferCreated        = "queue_transfer_created"
	AgentSelectionEventCustomOptionSelected        = "custom_option_selected"
	AgentSelectionEventCustomActionCompleted       = "custom_action_completed"
	AgentSelectionEventSelectionTimeout            = "selection_timeout"
	AgentSelectionEventSessionCancelled            = "session_cancelled"
	AgentSelectionEventSessionExpired              = "session_expired"
	AgentSelectionEventProcessingError             = "processing_error"
)

type AgentSelectionSettings struct {
	BaseModel
	OrganizationID            uuid.UUID                  `gorm:"type:uuid;index;not null;uniqueIndex:idx_agent_selection_settings_scope" json:"organization_id"`
	InstanceID                *uuid.UUID                 `gorm:"type:uuid;index;uniqueIndex:idx_agent_selection_settings_scope" json:"instance_id,omitempty"`
	AllowedInstanceIDs        StringArray                `gorm:"type:jsonb;default:'[]'" json:"allowed_instance_ids"`
	Enabled                   bool                       `gorm:"default:false" json:"enabled"`
	TriggerMode               AgentSelectionTriggerMode  `gorm:"size:40;default:first_pending_message" json:"trigger_mode"`
	TriggerKeywords           StringArray                `gorm:"type:jsonb;default:'[]'" json:"trigger_keywords"`
	PromptDelayMinutes        int                        `gorm:"default:3" json:"prompt_delay_minutes"`
	SelectionTimeoutMinutes   int                        `gorm:"default:10" json:"selection_timeout_minutes"`
	MaxInvalidAttempts        int                        `gorm:"default:3" json:"max_invalid_attempts"`
	MenuHeaderText            string                     `gorm:"type:text" json:"menu_header_text"`
	MenuFooterText            string                     `gorm:"type:text" json:"menu_footer_text"`
	InvalidReplyText          string                     `gorm:"type:text" json:"invalid_reply_text"`
	TimeoutResponseText       string                     `gorm:"type:text" json:"timeout_response_text"`
	UnavailableAgentText      string                     `gorm:"type:text" json:"unavailable_agent_text"`
	CustomFinalOptionEnabled  bool                       `gorm:"default:false" json:"custom_final_option_enabled"`
	CustomFinalOptionText     string                     `gorm:"type:text" json:"custom_final_option_text"`
	CustomFinalOptionResponse string                     `gorm:"type:text" json:"custom_final_option_response"`
	CustomFinalOptionAction   AgentSelectionCustomAction `gorm:"size:30;default:keep_pending" json:"custom_final_option_action"`
	CustomFinalOptionTeamID   *uuid.UUID                 `gorm:"type:uuid;index" json:"custom_final_option_team_id,omitempty"`
	HideUnavailableAgents     bool                       `gorm:"default:true" json:"hide_unavailable_agents"`
}

func (AgentSelectionSettings) TableName() string {
	return "agent_selection_settings"
}

type AgentSelectionParticipant struct {
	BaseModel
	OrganizationID        uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:idx_agent_selection_participant_user,where:deleted_at IS NULL" json:"organization_id"`
	SettingsID            uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:idx_agent_selection_participant_user,where:deleted_at IS NULL" json:"settings_id"`
	UserID                uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:idx_agent_selection_participant_user,where:deleted_at IS NULL" json:"user_id"`
	DisplayName           string    `gorm:"size:255;not null" json:"display_name"`
	Description           string    `gorm:"size:500" json:"description"`
	IsEnabled             bool      `gorm:"default:true" json:"is_enabled"`
	SortOrder             int       `gorm:"default:0" json:"sort_order"`
	ShowOnlyWhenAvailable bool      `gorm:"default:true" json:"show_only_when_available"`
	MaxOpenChats          *int      `json:"max_open_chats,omitempty"`
	Metadata              JSONB     `gorm:"type:jsonb;default:'{}'" json:"metadata"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (AgentSelectionParticipant) TableName() string {
	return "agent_selection_participants"
}

type AgentSelectionOption struct {
	BaseModel
	OrganizationID uuid.UUID                `gorm:"type:uuid;index;not null" json:"organization_id"`
	SettingsID     uuid.UUID                `gorm:"type:uuid;index;not null" json:"settings_id"`
	OptionType     AgentSelectionOptionType `gorm:"size:20;index;not null" json:"option_type"`
	UserID         *uuid.UUID               `gorm:"type:uuid;index" json:"user_id,omitempty"`
	TeamID         *uuid.UUID               `gorm:"type:uuid;index" json:"team_id,omitempty"`
	Label          string                   `gorm:"size:255;not null" json:"label"`
	Description    string                   `gorm:"size:500" json:"description"`
	IsEnabled      bool                     `gorm:"default:true" json:"is_enabled"`
	SortOrder      int                      `gorm:"default:0" json:"sort_order"`
	Action         string                   `gorm:"size:50" json:"action"`
	Metadata       JSONB                    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (AgentSelectionOption) TableName() string {
	return "agent_selection_options"
}

type AgentSelectionSession struct {
	BaseModel
	OrganizationID          uuid.UUID                   `gorm:"type:uuid;index;not null" json:"organization_id"`
	ContactID               uuid.UUID                   `gorm:"type:uuid;index;not null" json:"contact_id"`
	InstanceID              *uuid.UUID                  `gorm:"type:uuid;index" json:"instance_id,omitempty"`
	WhatsAppAccount         string                      `gorm:"size:255;index" json:"whatsapp_account"`
	Status                  AgentSelectionSessionStatus `gorm:"size:30;index;not null" json:"status"`
	TriggerMessageID        *uuid.UUID                  `gorm:"type:uuid;index" json:"trigger_message_id,omitempty"`
	PromptMessageID         *uuid.UUID                  `gorm:"type:uuid;index" json:"prompt_message_id,omitempty"`
	PromptDueAt             time.Time                   `gorm:"index" json:"prompt_due_at"`
	MenuSentAt              *time.Time                  `json:"menu_sent_at,omitempty"`
	ExpiresAt               *time.Time                  `gorm:"index" json:"expires_at,omitempty"`
	SelectedOptionID        string                      `gorm:"size:100;index" json:"selected_option_id,omitempty"`
	SelectedUserID          *uuid.UUID                  `gorm:"type:uuid;index" json:"selected_user_id,omitempty"`
	SelectedTeamID          *uuid.UUID                  `gorm:"type:uuid;index" json:"selected_team_id,omitempty"`
	TransferID              *uuid.UUID                  `gorm:"type:uuid;index" json:"transfer_id,omitempty"`
	InvalidAttempts         int                         `gorm:"default:0" json:"invalid_attempts"`
	RenderedOptionsSnapshot JSONBArray                  `gorm:"type:jsonb;default:'[]'" json:"rendered_options_snapshot"`
	ProcessedInboundIDs     StringArray                 `gorm:"type:jsonb;default:'[]'" json:"processed_inbound_ids"`
	Metadata                JSONB                       `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (AgentSelectionSession) TableName() string {
	return "agent_selection_sessions"
}

type AgentSelectionAuditEvent struct {
	BaseModel
	OrganizationID         uuid.UUID                `gorm:"type:uuid;index;not null" json:"organization_id"`
	ContactID              *uuid.UUID               `gorm:"type:uuid;index" json:"contact_id,omitempty"`
	SessionID              *uuid.UUID               `gorm:"type:uuid;index" json:"session_id,omitempty"`
	InstanceID             *uuid.UUID               `gorm:"type:uuid;index" json:"instance_id,omitempty"`
	WhatsAppAccount        string                   `gorm:"size:255;index" json:"whatsapp_account,omitempty"`
	EventType              string                   `gorm:"size:80;index;not null" json:"event_type"`
	ActorType              AgentSelectionAuditActor `gorm:"size:20;index;not null" json:"actor_type"`
	ActorUserID            *uuid.UUID               `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	SelectedOptionID       string                   `gorm:"size:100;index" json:"selected_option_id,omitempty"`
	SelectedAgentID        *uuid.UUID               `gorm:"type:uuid;index" json:"selected_agent_id,omitempty"`
	SelectedTeamID         *uuid.UUID               `gorm:"type:uuid;index" json:"selected_team_id,omitempty"`
	PreviousAssignedUserID *uuid.UUID               `gorm:"type:uuid;index" json:"previous_assigned_user_id,omitempty"`
	NewAssignedUserID      *uuid.UUID               `gorm:"type:uuid;index" json:"new_assigned_user_id,omitempty"`
	TransferID             *uuid.UUID               `gorm:"type:uuid;index" json:"transfer_id,omitempty"`
	InboundMessageID       *uuid.UUID               `gorm:"type:uuid;index" json:"inbound_message_id,omitempty"`
	OutboundMessageID      *uuid.UUID               `gorm:"type:uuid;index" json:"outbound_message_id,omitempty"`
	Reason                 string                   `gorm:"size:500" json:"reason,omitempty"`
	Metadata               JSONB                    `gorm:"type:jsonb;default:'{}'" json:"metadata"`
}

func (AgentSelectionAuditEvent) TableName() string {
	return "agent_selection_audit_events"
}
