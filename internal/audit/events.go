// Package audit provides the canonical cross-cutting audit recorder for
// Whatomate. It exposes a best-effort Service.Record recorder, a fluent
// EventBuilder, and typed Source/Category/Action constants.
package audit

// Source values distinguish who/what originated an event.
const (
	SourceUser      = "user"
	SourceSystem    = "system"
	SourceWorker    = "worker"
	SourceScheduled = "scheduled"
)

// Category values group actions for filtering on the read side.
const (
	CategoryAuth     = "auth"
	CategoryChat     = "chat"
	CategoryAdmin    = "admin"
	CategorySystem   = "system"
	CategoryCampaign = "campaign"
	CategoryTemplate = "template"
)

// Action values are namespaced by category for readability. Call sites pass
// an Action* constant to NewEvent; the category is inferred via actionCategory.
const (
	// auth
	ActionLoginSuccess   = "login_success"
	ActionLoginFailed    = "login_failed"
	ActionLogout         = "logout"
	ActionTokenRefreshed = "token_refreshed"
	ActionPasswordReset  = "password_reset"

	// chat (scope-B: claim/release/transfer/close/assign — NOT per-message)
	ActionChatClaimed     = "chat_claimed"
	ActionChatReleased    = "chat_released"
	ActionChatTransferred = "chat_transferred"
	ActionChatClosed      = "chat_closed"
	ActionChatAssigned    = "chat_assigned"

	// admin
	ActionUserCreated   = "user_created"
	ActionUserUpdated   = "user_updated"
	ActionUserDeleted   = "user_deleted"
	ActionUserActivated = "user_activated"
	ActionUserSuspended = "user_suspended"
	ActionRoleCreated   = "role_created"
	ActionRoleUpdated   = "role_updated"
	ActionRoleDeleted   = "role_deleted"
	ActionAPIKeyCreated = "api_key_created"
	ActionAPIKeyRevoked = "api_key_revoked"

	// system
	ActionServerStarted  = "server_started"
	ActionServerStopped  = "server_stopped"
	ActionWorkerStarted  = "worker_started"
	ActionWorkerStopped  = "worker_stopped"
	ActionConfigChanged  = "config_changed"
	ActionLicenseDenied  = "license_denied"
	ActionModuleEnabled  = "module_enabled"
	ActionModuleDisabled = "module_disabled"
)

// actionCategory maps each Action* to its Category*. NewEvent uses this so
// call sites specify only the action. Unknown actions default to CategorySystem.
func actionCategory(action string) string {
	switch action {
	case ActionLoginSuccess, ActionLoginFailed, ActionLogout,
		ActionTokenRefreshed, ActionPasswordReset:
		return CategoryAuth
	case ActionChatClaimed, ActionChatReleased, ActionChatTransferred,
		ActionChatClosed, ActionChatAssigned:
		return CategoryChat
	case ActionUserCreated, ActionUserUpdated, ActionUserDeleted,
		ActionUserActivated, ActionUserSuspended,
		ActionRoleCreated, ActionRoleUpdated, ActionRoleDeleted,
		ActionAPIKeyCreated, ActionAPIKeyRevoked:
		return CategoryAdmin
	case ActionServerStarted, ActionServerStopped,
		ActionWorkerStarted, ActionWorkerStopped,
		ActionConfigChanged, ActionLicenseDenied,
		ActionModuleEnabled, ActionModuleDisabled:
		return CategorySystem
	default:
		return CategorySystem
	}
}
