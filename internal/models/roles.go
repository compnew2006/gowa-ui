package models

import (
	"github.com/google/uuid"
)

// Permission represents a granular permission for a specific resource and action
type Permission struct {
	BaseModel
	Resource    string `gorm:"size:50;not null;uniqueIndex:idx_permission_resource_action" json:"resource"`
	Action      string `gorm:"size:20;not null;uniqueIndex:idx_permission_resource_action" json:"action"`
	Description string `gorm:"size:200" json:"description"`
}

func (Permission) TableName() string {
	return "permissions"
}

// CustomRole represents a role with specific permissions
type CustomRole struct {
	BaseModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string       `gorm:"size:100;not null" json:"name"`
	Description    string       `gorm:"size:500" json:"description"`
	IsSystem       bool         `gorm:"default:false" json:"is_system"`  // true for default admin/manager/agent
	IsDefault      bool         `gorm:"default:false" json:"is_default"` // default role for new users in org
	Permissions    []Permission `gorm:"many2many:role_permissions;" json:"permissions"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

func (CustomRole) TableName() string {
	return "custom_roles"
}

// RolePermission is the junction table for CustomRole and Permission many-to-many relationship
type RolePermission struct {
	CustomRoleID uuid.UUID `gorm:"type:uuid;primaryKey" json:"custom_role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"permission_id"`
}

// PermissionResource constants for available resources
const (
	ResourceUsers                = "users"
	ResourceTeams                = "teams"
	ResourceRoles                = "roles"
	ResourceSettingsGeneral      = "settings.general"
	ResourceSettingsSSO          = "settings.sso"
	ResourceSettingsNotification = "settings.notification"
	// Audit-only resource for the close-rating settings tab (not a permission).
	ResourceSettingsCloseRating = "settings.close_rating"
	// Audit-only resource for the call-auto-reject settings tab (not a permission).
	ResourceSettingsCallAutoReject = "settings.call_auto_reject"
	// Audit-only resource for the daily chat-reset settings tab (not a permission).
	ResourceSettingsChatReset = "settings.chat_reset"
	ResourceAccounts          = "accounts"
	ResourceDevices           = "devices"
	ResourceGowaInstances     = "gowa_instances"
	ResourceTemplates         = "templates"
	ResourceCampaigns         = "campaigns"
	ResourceChat              = "chat"
	ResourceChatAssign        = "chat.assign"
	ResourceChatCollaborate   = "chat.collaborate"
	ResourceChatRevoke        = "chat.revoke"
	ResourceContacts          = "contacts"
	// Contacts management page (settings). Separate from contacts:read (which
	// drives chat-list visibility/scoping) so a role can see conversations in
	// /chat while being blocked from the /settings/contacts directory page and
	// its Import/Export features.
	ResourceContactsManage  = "contacts.manage"
	ResourceTags            = "tags"
	ResourceAnalytics       = "analytics"
	ResourceAnalyticsAgents = "analytics.agents"
	ResourceWebhooks        = "webhooks"
	ResourceAPIKeys         = "api_keys"
	ResourceCannedResponses = "canned_responses"
	ResourceCustomActions   = "custom_actions"
	ResourceOrganizations   = "organizations"
	ResourceAuditLogs       = "audit_logs"
)

// PermissionAction constants for available actions
const (
	ActionRead    = "read"
	ActionWrite   = "write"
	ActionDelete  = "delete"
	ActionSync    = "sync"
	ActionExecute = "execute"
	ActionImport  = "import"
	ActionExport  = "export"
	ActionPickup  = "pickup"
	ActionAssign  = "assign"
)

// DefaultPermissions returns the list of all available permissions to seed
func DefaultPermissions() []Permission {
	return []Permission{
		// Users
		{Resource: ResourceUsers, Action: ActionRead, Description: "View users"},
		{Resource: ResourceUsers, Action: ActionWrite, Description: "Create and edit users"},
		{Resource: ResourceUsers, Action: ActionDelete, Description: "Delete users"},

		// Teams
		{Resource: ResourceTeams, Action: ActionRead, Description: "View teams"},
		{Resource: ResourceTeams, Action: ActionWrite, Description: "Create and edit teams"},
		{Resource: ResourceTeams, Action: ActionDelete, Description: "Delete teams"},

		// Roles
		{Resource: ResourceRoles, Action: ActionRead, Description: "View roles"},
		{Resource: ResourceRoles, Action: ActionWrite, Description: "Create and edit roles"},
		{Resource: ResourceRoles, Action: ActionDelete, Description: "Delete roles"},

		// Settings
		{Resource: ResourceSettingsGeneral, Action: ActionRead, Description: "View general settings"},
		{Resource: ResourceSettingsGeneral, Action: ActionWrite, Description: "Edit general settings"},
		{Resource: ResourceSettingsSSO, Action: ActionRead, Description: "View SSO settings"},
		{Resource: ResourceSettingsSSO, Action: ActionWrite, Description: "Edit SSO settings"},

		// Accounts
		{Resource: ResourceAccounts, Action: ActionRead, Description: "View WhatsApp accounts"},
		{Resource: ResourceAccounts, Action: ActionWrite, Description: "Create and edit WhatsApp accounts"},
		{Resource: ResourceAccounts, Action: ActionDelete, Description: "Delete WhatsApp accounts"},
		{Resource: ResourceAccounts, Action: ActionAssign, Description: "Assign WhatsApp account access to users"},

		// Devices (GOWA device management — pairing, provisioning, status)
		{Resource: ResourceDevices, Action: ActionRead, Description: "View GOWA device status and instances"},
		{Resource: ResourceDevices, Action: ActionWrite, Description: "Pair and provision GOWA devices"},
		{Resource: ResourceDevices, Action: ActionDelete, Description: "Delete GOWA devices"},

		// GOWA instances (DB-managed GOWA servers)
		{Resource: ResourceGowaInstances, Action: ActionRead, Description: "View GOWA server instances"},
		{Resource: ResourceGowaInstances, Action: ActionWrite, Description: "Create and edit GOWA server instances"},
		{Resource: ResourceGowaInstances, Action: ActionDelete, Description: "Delete GOWA server instances"},

		// Templates
		{Resource: ResourceTemplates, Action: ActionRead, Description: "View message templates"},
		{Resource: ResourceTemplates, Action: ActionWrite, Description: "Create and edit templates"},
		{Resource: ResourceTemplates, Action: ActionDelete, Description: "Delete templates"},

		// Campaigns
		{Resource: ResourceCampaigns, Action: ActionRead, Description: "View campaigns"},
		{Resource: ResourceCampaigns, Action: ActionWrite, Description: "Create and edit campaigns"},
		{Resource: ResourceCampaigns, Action: ActionDelete, Description: "Delete campaigns"},
		{Resource: ResourceCampaigns, Action: ActionExecute, Description: "Execute campaigns"},

		// Chat
		{Resource: ResourceChat, Action: ActionRead, Description: "View chat conversations"},
		{Resource: ResourceChat, Action: ActionWrite, Description: "Send messages"},
		{Resource: ResourceChatRevoke, Action: ActionWrite, Description: "Revoke (delete for everyone) sent messages"},
		{Resource: ResourceChatAssign, Action: ActionWrite, Description: "Assign conversations to agents"},
		{Resource: ResourceChatCollaborate, Action: ActionWrite, Description: "Join assigned chats as a collaborator"},

		// Contacts
		{Resource: ResourceContacts, Action: ActionRead, Description: "View contacts"},
		{Resource: ResourceContacts, Action: ActionWrite, Description: "Create and edit contacts"},
		{Resource: ResourceContacts, Action: ActionDelete, Description: "Delete contacts"},
		{Resource: ResourceContacts, Action: ActionImport, Description: "Import contacts"},
		{Resource: ResourceContacts, Action: ActionExport, Description: "Export contacts"},
		// Contacts management page (settings) — gates /settings/contacts so it
		// can be hidden from roles that still see conversations via contacts:read.
		{Resource: ResourceContactsManage, Action: ActionRead, Description: "Access the contacts management page"},

		// Tags
		{Resource: ResourceTags, Action: ActionRead, Description: "View tags"},
		{Resource: ResourceTags, Action: ActionWrite, Description: "Create and edit tags"},
		{Resource: ResourceTags, Action: ActionDelete, Description: "Delete tags"},

		// Analytics
		{Resource: ResourceAnalytics, Action: ActionRead, Description: "View analytics dashboard"},
		{Resource: ResourceAnalytics, Action: ActionWrite, Description: "Create and edit dashboard widgets"},
		{Resource: ResourceAnalytics, Action: ActionDelete, Description: "Delete dashboard widgets"},
		{Resource: ResourceAnalyticsAgents, Action: ActionRead, Description: "View agent analytics"},

		// Webhooks
		{Resource: ResourceWebhooks, Action: ActionRead, Description: "View webhooks"},
		{Resource: ResourceWebhooks, Action: ActionWrite, Description: "Create and edit webhooks"},
		{Resource: ResourceWebhooks, Action: ActionDelete, Description: "Delete webhooks"},

		// API Keys
		{Resource: ResourceAPIKeys, Action: ActionRead, Description: "View API keys"},
		{Resource: ResourceAPIKeys, Action: ActionWrite, Description: "Create API keys"},
		{Resource: ResourceAPIKeys, Action: ActionDelete, Description: "Delete API keys"},

		// Canned Responses
		{Resource: ResourceCannedResponses, Action: ActionRead, Description: "View canned responses"},
		{Resource: ResourceCannedResponses, Action: ActionWrite, Description: "Create and edit canned responses"},
		{Resource: ResourceCannedResponses, Action: ActionDelete, Description: "Delete canned responses"},

		// Custom Actions
		{Resource: ResourceCustomActions, Action: ActionRead, Description: "View custom actions"},
		{Resource: ResourceCustomActions, Action: ActionWrite, Description: "Create and edit custom actions"},
		{Resource: ResourceCustomActions, Action: ActionDelete, Description: "Delete custom actions"},
		{Resource: ResourceCustomActions, Action: ActionExecute, Description: "Run custom actions from chat"},

		// Organizations
		{Resource: ResourceOrganizations, Action: ActionRead, Description: "View organizations"},
		{Resource: ResourceOrganizations, Action: ActionWrite, Description: "Create organizations"},
		{Resource: ResourceOrganizations, Action: ActionDelete, Description: "Delete organizations"},
		{Resource: ResourceOrganizations, Action: ActionAssign, Description: "Manage organization members"},

		// Audit Logs
		{Resource: ResourceAuditLogs, Action: ActionRead, Description: "View audit logs"},
	}
}

// SystemRolePermissions returns the default permission mappings for system roles
func SystemRolePermissions() map[string][]string {
	// Format: "resource:action"
	allPermissions := []string{}
	for _, p := range DefaultPermissions() {
		allPermissions = append(allPermissions, p.Resource+":"+p.Action)
	}

	managerPermissions := []string{
		// Users (managers administer their team's members)
		"users:read", "users:write",
		// Teams
		"teams:read", "teams:write",
		// Roles (read only — needed to assign roles when editing users)
		"roles:read",
		// Settings
		"settings.general:read", "settings.general:write",
		// Accounts
		"accounts:read", "accounts:write", "accounts:delete", "accounts:assign",
		// Devices
		"devices:read", "devices:write", "devices:delete",
		// GOWA instances
		"gowa_instances:read", "gowa_instances:write", "gowa_instances:delete",
		// Templates
		"templates:read", "templates:write", "templates:delete",
		// Campaigns
		"campaigns:read", "campaigns:write", "campaigns:delete", "campaigns:execute",
		// Chat
		"chat:read", "chat:write", "chat.revoke:write", "chat.assign:write", "chat.collaborate:write",
		// Contacts
		"contacts:read", "contacts:write", "contacts:delete", "contacts:import", "contacts:export",
		"contacts.manage:read",
		// Tags
		"tags:read", "tags:write", "tags:delete",
		// Analytics
		"analytics:read", "analytics:write", "analytics:delete", "analytics.agents:read",
		// Webhooks
		"webhooks:read", "webhooks:write", "webhooks:delete",
		// Canned Responses
		"canned_responses:read", "canned_responses:write", "canned_responses:delete",
		// Custom Actions
		"custom_actions:read", "custom_actions:write", "custom_actions:delete", "custom_actions:execute",
		// Organizations (read only)
		"organizations:read",
	}

	agentPermissions := []string{
		// Accounts (read only)
		"accounts:read",
		// Chat
		"chat:read", "chat:write", "chat.revoke:write", "chat.assign:write",
		// Contacts (read only)
		"contacts:read",
		// Tags (read only - agents can see tags on contacts)
		"tags:read",
		// Analytics (own)
		"analytics.agents:read",
		// Canned Responses (read only)
		"canned_responses:read",
		// Custom Actions (run from the chat sidebar)
		"custom_actions:execute",
	}

	return map[string][]string{
		"admin":   allPermissions,
		"manager": managerPermissions,
		"agent":   agentPermissions,
	}
}
