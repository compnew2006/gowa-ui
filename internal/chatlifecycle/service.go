// Package chatlifecycle owns the chat conversation state machine.
//
// It is the single source of truth for claim / release / close / reopen /
// join / leave / invite / remove transitions, plus the audit, system-message,
// and WebSocket side effects those transitions emit.
//
// Handlers in internal/handlers/chat_lifecycle.go are thin HTTP adapters:
// they parse the request, run requireAuth, look up the contact, call into
// this service, and map the result back onto the HTTP envelope. All business
// rules + persistence + side effects live here so they can be unit-tested
// with a database alone (no Redis, no *App, no fastglue).
//
// The shape mirrors internal/assignment: an unexported struct, a New()
// constructor that takes raw dependencies, and concrete-pointer return types.
package chatlifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/audit"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/zerodha/logf"
	"gorm.io/gorm"
)

// Sentinel errors the handler maps onto HTTP status codes. They carry no
// state — the handler decides the message + code, the service just signals
// which branch to take.
var (
	// ErrClosedReleaseByAgent: an agent (non-admin) attempted to release a
	// closed chat. The spec's closed-chat edge case allows this only for
	// admins/managers — agents must reopen first.
	ErrClosedReleaseByAgent = errors.New("chat: only admins can release a closed chat")
	// ErrNotAuthorized: the caller is neither the assignee/collaborator nor
	// an admin/manager for this transition.
	ErrNotAuthorized = errors.New("chat: not authorized")
	// ErrCannotRemoveOwner: RemoveCollaborator was called on the contact's
	// owner — ownership must be transferred or released, not removed.
	ErrCannotRemoveOwner = errors.New("chat: cannot remove the owner")
	// ErrNotCollaborator: RemoveCollaborator target is not in the
	// collaborators list.
	ErrNotCollaborator = errors.New("chat: user is not a collaborator")
)

// Service is the chat-lifecycle state machine. All fields are unexported;
// callers obtain an instance via New and use only its exported methods.
type Service struct {
	db    *gorm.DB
	wsHub *websocket.Hub // nil-safe: tests + deployments without WS pass nil
	log   logf.Logger
}

// New constructs a Service. Mirrors assignment.New: raw dependencies in,
// concrete pointer out. wsHub may be nil — broadcast calls are guarded.
func New(db *gorm.DB, wsHub *websocket.Hub, log logf.Logger) *Service {
	return &Service{db: db, wsHub: wsHub, log: log}
}

// CreateSystemMessage records a system message in the conversation timeline.
//
// Exported so that callers outside the chat-lifecycle handlers (notably the
// incoming-message pipeline, which writes "Conversation reopened
// by customer" on inbound messages) can migrate off the *App helper. Until
// that migration happens, the handler layer keeps a thin delegator.
//
// NOTE: this intentionally does NOT bump Contact.last_message_at — that is
// the caller's responsibility (ReleaseChat and BulkReleaseChats bump it in
// their Updates call so the released chat re-sorts to the top of Pending).
func (s *Service) CreateSystemMessage(orgID, contactID uuid.UUID, content string, metadata models.JSONB) {
	if metadata == nil {
		metadata = models.JSONB{}
	}
	metadata["is_system_message"] = true

	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: orgID,
		ContactID:      contactID,
		Direction:      models.DirectionOutgoing,
		MessageType:    models.MessageTypeText,
		Content:        content,
		Status:         models.MessageStatusSent,
		Metadata:       metadata,
	}
	if err := s.db.Create(msg).Error; err != nil {
		s.log.Error("Failed to create system message", "error", err, "contact_id", contactID)
	}
}

// Release returns an assigned (open or closed) conversation to the pending
// pool: unassigns the owner, clears collaborators, sets status to pending,
// bumps last_message_at, writes the chat_released system message + audit
// entry, and broadcasts chat_released.
//
// Authorization is the caller's responsibility — pass isAssignee and
// isAdminOrManager already computed (they require HasPermission, which is
// handler-owned). The closed-chat policy guard IS checked here because it
// is a business rule, not an auth check: an agent assignee of a closed chat
// gets ErrClosedReleaseByAgent; admins/Managers bypass it.
//
// Idempotent: if the contact is already pending + unassigned, returns
// (false, nil) — no system message, no audit, no broadcast. The handler
// turns this into a success envelope.
//
// Returns (true, nil) on a real release, (false, nil) on the idempotent
// no-op, and (false, err) on a policy violation or persistence failure.
func (s *Service) Release(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact, isAssignee, isAdminOrManager bool) (bool, error) {
	// Authorization is checked first in the handler; double-check here as a
	// defense-in-depth invariant (the handler is the source of truth but the
	// service must not be callable in a way that violates policy).
	if !isAssignee && !isAdminOrManager {
		return false, ErrNotAuthorized
	}

	// Closed-chat policy: only admins/managers may release a closed chat.
	// An agent assignee of a closed chat must reopen first.
	if contact.EffectiveStatus() == models.ChatStatusClosed && !isAdminOrManager {
		return false, ErrClosedReleaseByAgent
	}

	// Idempotent: already pending + unassigned → safe no-op success.
	if contact.EffectiveStatus() == models.ChatStatusPending && contact.AssignedUserID == nil {
		return false, nil
	}

	// Capture pre-mutation values for the audit log BEFORE mutation. Status
	// lives in the JSONB Metadata map, which struct snapshots alias — by the
	// time the audit diff is built, the shared map has already been mutated.
	// Capturing into locals here is the only correct way to record old→new.
	oldStatus := string(contact.EffectiveStatus())
	oldAssigned := contact.AssignedUserID

	// Mutation: unassign + set pending + clear collaborators + bump
	// last_message_at so the released chat re-sorts to the top of Pending.
	now := time.Now()
	contact.AssignedUserID = nil
	contact.SetStatus(models.ChatStatusPending)
	contact.ClearCollaborators()
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": nil,
		"metadata":         contact.Metadata,
		"last_message_at":  &now,
	}).Error; err != nil {
		s.log.Error("Failed to release chat", "error", err, "contact_id", contact.ID)
		return false, fmt.Errorf("chat: failed to release: %w", err)
	}
	contact.LastMessageAt = &now

	// Agent display name for the system message (durable + locale-independent).
	agentName := audit.GetUserName(s.db, userID)

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s released this conversation", agentName),
		models.JSONB{
			"system_type": "chat_released",
			"agent_id":    userID.String(),
			"agent_name":  agentName,
		})

	// Audit: the extraChanges safeguard is load-bearing — audit.LogAudit
	// silently no-ops when action=updated AND the computed diff is empty, and
	// status lives in JSONB which the differ does not deeply compare. The
	// explicit old→new map forces the entry to persist.
	audit.LogAudit(s.db, orgID, userID, agentName,
		"contact", contact.ID, models.AuditActionUpdated, nil, contact,
		map[string]any{
			"chat_status":      map[string]any{"old": oldStatus, "new": string(models.ChatStatusPending)},
			"assigned_user_id": map[string]any{"old": oldAssigned, "new": nil},
		})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatReleased,
		Payload: map[string]any{
			"contact_id":      contact.ID.String(),
			"released_by":     userID.String(),
			"chat_status":     string(models.ChatStatusPending),
			"collaborators":   []any{}, // cleared server-side — include so clients drop stale collabs
			"last_message_at": now.Format(time.RFC3339Nano),
		},
	})

	return true, nil
}

// Assign is the admin/manager "Assign to agent" transition: sets the owner,
// flips the status to open, and emits the same side effects as the other
// transitions — a system message crediting who assigned the chat to whom, an
// audit entry, and a chat_claimed broadcast so every client updates without a
// page refresh.
//
// A nil targetID unassigns: semantically that is a release performed by an
// admin, so it delegates to Release (pending + cleared collaborators +
// chat_released system message/broadcast).
//
// Idempotent: assigning an open chat to its current owner is a no-op — no
// duplicate system message, no broadcast.
func (s *Service) Assign(ctx context.Context, orgID, adminID uuid.UUID, contact *models.Contact, targetID *uuid.UUID) error {
	if targetID == nil {
		_, err := s.Release(ctx, orgID, adminID, contact, false, true)
		return err
	}

	if contact.AssignedUserID != nil && *contact.AssignedUserID == *targetID &&
		contact.EffectiveStatus() == models.ChatStatusOpen {
		return nil
	}

	// Capture pre-mutation values for the audit log BEFORE mutation (see
	// Release for why: Metadata is a shared JSONB map).
	oldStatus := string(contact.EffectiveStatus())
	oldAssigned := contact.AssignedUserID

	contact.AssignedUserID = targetID
	contact.SetStatus(models.ChatStatusOpen)
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": targetID,
		"metadata":         contact.Metadata,
	}).Error; err != nil {
		s.log.Error("Failed to assign chat", "error", err, "contact_id", contact.ID)
		return fmt.Errorf("chat: failed to assign: %w", err)
	}

	adminName := audit.GetUserName(s.db, adminID)
	targetName := audit.GetUserName(s.db, *targetID)

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s assigned this conversation to %s", adminName, targetName),
		models.JSONB{
			"system_type":      "chat_assigned",
			"agent_id":         targetID.String(),
			"agent_name":       targetName,
			"assigned_by":      adminID.String(),
			"assigned_by_name": adminName,
		})

	audit.LogAudit(s.db, orgID, adminID, adminName,
		"contact", contact.ID, models.AuditActionUpdated, nil, contact,
		map[string]any{
			"chat_status":      map[string]any{"old": oldStatus, "new": string(models.ChatStatusOpen)},
			"assigned_user_id": map[string]any{"old": oldAssigned, "new": targetID},
		})

	// Same shape as Claim's broadcast — the frontend's chat_claimed handler
	// already updates the list entry and re-fetches messages for viewers, so
	// the system message shows up in real time.
	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatClaimed,
		Payload: map[string]any{
			"contact_id":       contact.ID.String(),
			"assigned_to":      targetID.String(),
			"assigned_user_id": targetID.String(),
			"assigned_to_name": targetName,
			"chat_status":      string(models.ChatStatusOpen),
			"assigned_by":      adminID.String(),
			"assigned_by_name": adminName,
		},
	})

	return nil
}

// releaseOne is the per-item body of BulkRelease. It shares Release's logic
// without re-broadcasting the "idempotent skip" semantics differently: on
// idempotent skip it records the id as released (matching Release's
// success-no-op behavior from the handler's perspective), and on policy
// violations it returns a structured failure for the bulk result.
func (s *Service) releaseOne(orgID, userID uuid.UUID, rawID string, isAdminOrManager bool) (released bool, failure map[string]any) {
	contactID, err := uuid.Parse(rawID)
	if err != nil {
		return false, map[string]any{"contact_id": rawID, "reason": "invalid uuid"}
	}

	var contact models.Contact
	if err := s.db.Where("id = ? AND organization_id = ?", contactID, orgID).First(&contact).Error; err != nil {
		return false, map[string]any{"contact_id": rawID, "reason": "not found"}
	}

	isAssignee := contact.AssignedUserID != nil && *contact.AssignedUserID == userID
	released, err = s.Release(context.Background(), orgID, userID, &contact, isAssignee, isAdminOrManager)
	if err != nil {
		// Map the typed errors to the bulk-result reason strings the UI shows.
		switch {
		case errors.Is(err, ErrNotAuthorized):
			return false, map[string]any{"contact_id": rawID, "reason": "not authorized"}
		case errors.Is(err, ErrClosedReleaseByAgent):
			return false, map[string]any{"contact_id": rawID, "reason": "closed chat requires admin"}
		default:
			return false, map[string]any{"contact_id": rawID, "reason": "release failed"}
		}
	}
	// Idempotent (released==false) counts as a successful no-op for bulk.
	return true, nil
}

// BulkResult is the structured outcome of BulkRelease. The handler maps this
// directly onto the {released_ids, released, failed, requested} envelope.
type BulkResult struct {
	ReleasedIDs []string
	// Failed holds one entry per contact that could not be released, each
	// carrying {contact_id, reason} so the UI can surface partial outcomes.
	Failed []map[string]any
}

// BulkRelease processes a batch of release requests. Authorization mirrors
// Release: per-item isAssignee OR isAdminOrManager. Deduplicates the input
// list, caps at 500 (the handler enforces the cap; this is defense-in-depth).
//
// Returns ReleasedIDs including idempotent no-ops (a chat already pending is
// "released" from the caller's perspective). Failures never abort the batch.
func (s *Service) BulkRelease(ctx context.Context, orgID, userID uuid.UUID, ids []string, isAdminOrManager bool) BulkResult {
	result := BulkResult{ReleasedIDs: []string{}, Failed: []map[string]any{}}
	seen := make(map[string]bool, len(ids))

	for _, rawID := range ids {
		if seen[rawID] {
			continue
		}
		seen[rawID] = true

		released, failure := s.releaseOne(orgID, userID, rawID, isAdminOrManager)
		if released {
			result.ReleasedIDs = append(result.ReleasedIDs, rawID)
		} else if failure != nil {
			result.Failed = append(result.Failed, failure)
		}
	}
	return result
}

// CustomerReopen reopens a closed conversation when the customer sends a new
// inbound message (or one is sent from the connected phone). It unassigns the
// owner, clears collaborators, sets status to pending, writes the
// chat_reopened system message, and broadcasts the reopen over WebSocket.
//
// note customizes the system-message text so the phone-sent path can say so;
// an empty note falls back to the customer-reopen wording. Called from
// ensureClaimableChatStatus in internal/handlers (incoming + phone-sent paths).
//
// Returns true if a reopen actually happened, false if the chat was already
// in a non-closed state (idempotent — no system message written).
func (s *Service) CustomerReopen(ctx context.Context, orgID uuid.UUID, contact *models.Contact, note string) bool {
	if contact.EffectiveStatus() != models.ChatStatusClosed {
		return false
	}

	contact.AssignedUserID = nil
	contact.ClearCollaborators()
	contact.SetStatus(models.ChatStatusPending)
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
		"assigned_user_id": nil,
		"metadata":         contact.Metadata,
	}).Error; err != nil {
		s.log.Error("Failed to reopen chat on customer message", "error", err, "contact_id", contact.ID)
		return false
	}

	if note == "" {
		note = "🔔 Conversation reopened by customer"
	}
	s.CreateSystemMessage(orgID, contact.ID, note,
		models.JSONB{"system_type": "chat_reopened"})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatReopened,
		Payload: map[string]any{
			"contact_id":  contact.ID.String(),
			"chat_status": string(models.ChatStatusPending),
			"reopened":    true,
			"by_customer": true,
		},
	})
	return true
}

// broadcast is a nil-safe wrapper. Deployments without a WS hub (and all
// unit tests) pass nil; the call is a no-op in that case.
func (s *Service) broadcast(orgID uuid.UUID, msg websocket.WSMessage) {
	if s.wsHub == nil {
		return
	}
	s.wsHub.BroadcastToOrg(orgID, msg)
}

// ClaimOutcome signals to the handler which response path to take.
type ClaimOutcome int

const (
	// ClaimDone: the chat was claimed (or reopened-via-claim) by the caller.
	// AgentName carries the resolved display name for the response envelope.
	ClaimDone ClaimOutcome = iota
	// ClaimAlreadySelf: the caller is already the assignee. Idempotent.
	ClaimAlreadySelf
	// ClaimRerouteJoin: the chat is assigned to another agent AND the caller
	// has chat.collaborate:write — they should be added as a collaborator
	// instead. The handler calls Service.Join next.
	ClaimRerouteJoin
	// ClaimConflictOther: assigned to another agent, caller lacks collaborate
	// permission. The handler returns 409 already_assigned.
	// OtherAgentName is set so the handler can name the current owner.
	ClaimConflictOther
)

// Claim assigns a pending (or closed — claim reopens) conversation to the
// caller. Authorization (chat.assign:write) and the contact lookup stay in
// the handler; hasCollaboratePerm is passed in because it requires
// HasPermission, which is handler-owned.
//
// Returns the outcome + the agent's display name (for the system message +
// response envelope). On ClaimConflictOther, OtherAgentName (returned via
// the named result) carries the current owner's name for the 409 body.
func (s *Service) Claim(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact, hasCollaboratePerm bool) (outcome ClaimOutcome, agentName, otherAgentName string, err error) {
	// Closed conversations CAN be claimed — this reopens them.

	// Guard: already assigned to another agent.
	if contact.AssignedUserID != nil && *contact.AssignedUserID != userID {
		if hasCollaboratePerm {
			return ClaimRerouteJoin, "", "", nil
		}
		var current models.User
		name := "another agent"
		if s.db.First(&current, "id = ?", *contact.AssignedUserID).Error == nil {
			name = current.FullName
		}
		return ClaimConflictOther, "", name, nil
	}

	// Idempotent: already assigned to self.
	if contact.AssignedUserID != nil && *contact.AssignedUserID == userID {
		if contact.EffectiveStatus() != models.ChatStatusOpen {
			contact.SetStatus(models.ChatStatusOpen)
			s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Update("metadata", contact.Metadata)
		}
		return ClaimAlreadySelf, "", "", nil
	}

	// Capture pre-mutation values BEFORE mutation — status lives in JSONB
	// Metadata, which struct snapshots alias (the audit differ would see the
	// post-mutation map). Same hazard as Release.
	wasClosed := contact.EffectiveStatus() == models.ChatStatusClosed
	oldStatus := string(contact.EffectiveStatus())
	oldAssigned := contact.AssignedUserID

	contact.AssignedUserID = &userID
	contact.SetStatus(models.ChatStatusOpen)
	if err := s.db.Save(&contact).Error; err != nil {
		s.log.Error("Failed to claim chat", "error", err)
		return ClaimDone, "", "", fmt.Errorf("chat: failed to claim: %w", err)
	}

	agentName = audit.GetUserName(s.db, userID)

	if wasClosed {
		s.CreateSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s reopened this conversation", agentName),
			models.JSONB{"system_type": "chat_reopened", "agent_id": userID.String(), "agent_name": agentName})
	} else {
		s.CreateSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s claimed this conversation", agentName),
			models.JSONB{"system_type": "chat_claimed", "agent_id": userID.String(), "agent_name": agentName})
	}

	// Audit extraChanges safeguard — load-bearing for the same reason as in
	// Release (JSONB status, audit.LogAudit no-ops on empty diff).
	audit.LogAudit(s.db, orgID, userID, agentName,
		"contact", contact.ID, models.AuditActionUpdated, nil, contact,
		map[string]any{
			"chat_status":      map[string]any{"old": oldStatus, "new": string(models.ChatStatusOpen)},
			"assigned_user_id": map[string]any{"old": oldAssigned, "new": &userID},
		})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatClaimed,
		Payload: map[string]any{
			"contact_id":       contact.ID.String(),
			"assigned_to":      userID.String(),
			"assigned_user_id": userID.String(),
			"assigned_to_name": agentName,
			"chat_status":      string(models.ChatStatusOpen),
		},
	})

	return ClaimDone, agentName, "", nil
}

// Join adds the caller as a collaborator. Idempotent: returns JoinAlreadyOwner
// if the caller is the primary owner, JoinAlreadyCollaborator if already a
// collaborator (no system message in either case).
type JoinOutcome int

const (
	JoinDone JoinOutcome = iota
	JoinAlreadyOwner
	JoinAlreadyCollaborator
)

// JoinResult is the result of Join. UserName is populated on JoinDone for the
// handler's response envelope.
type JoinResult struct {
	Outcome  JoinOutcome
	UserName string
	UserRole string
}

func (s *Service) Join(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) (JoinResult, error) {
	if contact.AssignedUserID != nil && *contact.AssignedUserID == userID {
		return JoinResult{Outcome: JoinAlreadyOwner}, nil
	}
	if contact.IsCollaborator(userID.String()) {
		return JoinResult{Outcome: JoinAlreadyCollaborator}, nil
	}
	return s.addCollaborator(orgID, userID, userID, contact, false)
}

// Invite adds a different user (targetID) as a collaborator, by the inviter
// (inviterID). Idempotent: returns InviteAlreadyOwner / InviteAlreadyCollaborator.
type InviteOutcome int

const (
	InviteDone InviteOutcome = iota
	InviteAlreadyOwner
	InviteAlreadyCollaborator
)

type InviteResult struct {
	Outcome     InviteOutcome
	TargetName  string
	TargetRole  string
	InviterName string
}

func (s *Service) Invite(ctx context.Context, orgID, inviterID, targetID uuid.UUID, contact *models.Contact) (InviteResult, error) {
	targetIDStr := targetID.String()

	// Verify target user exists in the same org. (The handler currently does
	// this lookup too; we re-do it here so the service is self-contained for
	// the follow-up that lets other callers use it.)
	var target models.User
	if err := s.db.Where("id = ? AND organization_id = ?", targetID, orgID).First(&target).Error; err != nil {
		return InviteResult{}, fmt.Errorf("chat: target user not found: %w", err)
	}

	if contact.AssignedUserID != nil && *contact.AssignedUserID == targetID {
		return InviteResult{Outcome: InviteAlreadyOwner}, nil
	}
	if contact.IsCollaborator(targetIDStr) {
		return InviteResult{Outcome: InviteAlreadyCollaborator}, nil
	}

	res, err := s.addCollaborator(orgID, inviterID, targetID, contact, true)
	if err != nil {
		return InviteResult{}, err
	}
	inviterName := audit.GetUserName(s.db, inviterID)
	return InviteResult{
		Outcome:     InviteDone,
		TargetName:  res.UserName,
		TargetRole:  res.UserRole,
		InviterName: inviterName,
	}, nil
}

// addCollaborator is the shared body of Join (self) and Invite (other). It
// resolves the target's name + role, appends to the collaborators array,
// persists, writes the collaborator_joined system message, and broadcasts.
// invitedBy is the actor performing the add (== joiner for Join, inviter for
// Invite); invitedBySelf marks the "joined on their own" variant where the
// system message reads "<user> joined" rather than "<user> was added by X".
func (s *Service) addCollaborator(orgID, invitedBy, targetID uuid.UUID, contact *models.Contact, invitedBySelf bool) (JoinResult, error) {
	var user models.User
	userName := "Unknown"
	userRole := ""
	if s.db.First(&user, "id = ?", targetID).Error == nil {
		userName = user.FullName
		if user.RoleID != nil {
			var role models.CustomRole
			if s.db.First(&role, "id = ?", *user.RoleID).Error == nil {
				userRole = role.Name
			}
		}
	}

	targetIDStr := targetID.String()
	contact.AddCollaborator(models.Collaborator{
		UserID:   targetIDStr,
		Name:     userName,
		Role:     userRole,
		JoinedAt: time.Now(),
	})
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error; err != nil {
		s.log.Error("Failed to add collaborator", "error", err)
		return JoinResult{}, fmt.Errorf("chat: failed to add collaborator: %w", err)
	}

	if invitedBySelf {
		s.CreateSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s joined the conversation", userName),
			models.JSONB{"system_type": "collaborator_joined", "agent_id": targetIDStr})
	} else {
		inviterName := audit.GetUserName(s.db, invitedBy)
		s.CreateSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s was added to the conversation by %s", userName, inviterName),
			models.JSONB{
				"system_type": "collaborator_joined",
				"agent_id":    targetIDStr,
				"invited_by":  invitedBy.String(),
			})
	}

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeCollaboratorJoined,
		Payload: map[string]any{
			"contact_id": contact.ID.String(),
			"user_id":    targetIDStr,
			"user_name":  userName,
			"user_role":  userRole,
		},
	})

	return JoinResult{Outcome: JoinDone, UserName: userName, UserRole: userRole}, nil
}

// LeaveOutcome mirrors the three branches of LeaveChat.
type LeaveOutcome int

const (
	// LeaveGhostExit: an admin/manager who was never a real participant
	// leaves. No state change, no system message.
	LeaveGhostExit LeaveOutcome = iota
	// LeaveClosedChat: the owner was the last participant; the conversation
	// is now closed.
	LeaveClosedChat
	// LeaveOwnershipTransferred: the owner left but collaborators remain;
	// ownership moved to collaborators[0].
	LeaveOwnershipTransferred
	// LeaveCollaboratorRemoved: a non-owner collaborator left.
	LeaveCollaboratorRemoved
)

type LeaveResult struct {
	Outcome      LeaveOutcome
	UserName     string // for system message + response
	NewOwnerName string // set on LeaveOwnershipTransferred
}

// Leave handles the three departure branches: ghost-exit (admin/manager),
// owner-leaves (close OR transfer ownership), and collaborator-leaves.
//
// The participant check (not owner, not collab, not admin → 400) stays in the
// handler because it produces a user-facing 400. Here we expect the caller is
// at least one of those.
func (s *Service) Leave(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact, isOwner, isCollaborator, isAdminOrManager bool) (LeaveResult, error) {
	// Ghost-exit: admin/manager who is not a real participant.
	if !isOwner && !isCollaborator && isAdminOrManager {
		return LeaveResult{Outcome: LeaveGhostExit}, nil
	}

	userName := audit.GetUserName(s.db, userID)

	if isOwner {
		collaborators := contact.GetCollaborators()
		if len(collaborators) == 0 {
			// Last participant leaving → close the conversation.
			contact.AssignedUserID = nil
			contact.ClearCollaborators()
			contact.SetStatus(models.ChatStatusClosed)
			if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
				"assigned_user_id": nil,
				"metadata":         contact.Metadata,
			}).Error; err != nil {
				s.log.Error("Failed to close chat on leave", "error", err, "contact_id", contact.ID)
				return LeaveResult{}, fmt.Errorf("chat: failed to close on leave: %w", err)
			}

			s.CreateSystemMessage(orgID, contact.ID,
				fmt.Sprintf("🔔 %s closed this conversation", userName),
				models.JSONB{"system_type": "chat_closed", "agent_id": userID.String()})

			s.broadcast(orgID, websocket.WSMessage{
				Type: websocket.TypeChatClosed,
				Payload: map[string]any{
					"contact_id":  contact.ID.String(),
					"chat_status": string(models.ChatStatusClosed),
					"closed":      true,
				},
			})

			return LeaveResult{Outcome: LeaveClosedChat, UserName: userName}, nil
		}

		// Owner leaves but collaborators remain — transfer ownership.
		newOwnerID, _ := uuid.Parse(collaborators[0].UserID)
		contact.AssignedUserID = &newOwnerID
		contact.RemoveCollaborator(collaborators[0].UserID)
		if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).Updates(map[string]any{
			"assigned_user_id": newOwnerID,
			"metadata":         contact.Metadata,
			// Clear stale name denormalization is not present today; preserve
			// the historical update shape exactly (assigned_user_id + metadata).
		}).Error; err != nil {
			s.log.Error("Failed to transfer ownership on leave", "error", err, "contact_id", contact.ID)
			return LeaveResult{}, fmt.Errorf("chat: failed to transfer ownership: %w", err)
		}

		s.CreateSystemMessage(orgID, contact.ID,
			fmt.Sprintf("🔔 %s left the conversation. Ownership transferred to %s", userName, collaborators[0].Name),
			models.JSONB{"system_type": "collaborator_left", "agent_id": userID.String()})

		s.broadcast(orgID, websocket.WSMessage{
			Type: websocket.TypeCollaboratorLeft,
			Payload: map[string]any{
				"contact_id": contact.ID.String(),
				"user_id":    userID.String(),
				"user_name":  userName,
			},
		})

		return LeaveResult{Outcome: LeaveOwnershipTransferred, UserName: userName, NewOwnerName: collaborators[0].Name}, nil
	}

	// Regular collaborator leaving.
	contact.RemoveCollaborator(userID.String())
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error; err != nil {
		s.log.Error("Failed to remove collaborator on leave", "error", err, "contact_id", contact.ID)
		return LeaveResult{}, fmt.Errorf("chat: failed to remove collaborator: %w", err)
	}

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s left the conversation", userName),
		models.JSONB{"system_type": "collaborator_left", "agent_id": userID.String()})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeCollaboratorLeft,
		Payload: map[string]any{
			"contact_id": contact.ID.String(),
			"user_id":    userID.String(),
			"user_name":  userName,
		},
	})

	return LeaveResult{Outcome: LeaveCollaboratorRemoved, UserName: userName}, nil
}

// RemoveCollaborator lets an admin/manager remove a collaborator. Returns
// ErrCannotRemoveOwner if target is the primary owner, ErrNotCollaborator if
// not currently a collaborator. Sets TargetName + ManagerName on success for
// the handler's response envelope + system message.
type RemoveCollaboratorResult struct {
	TargetName  string
	ManagerName string
}

func (s *Service) RemoveCollaborator(ctx context.Context, orgID, actorID, targetID uuid.UUID, contact *models.Contact) (RemoveCollaboratorResult, error) {
	targetIDStr := targetID.String()

	if contact.AssignedUserID != nil && *contact.AssignedUserID == targetID {
		return RemoveCollaboratorResult{}, ErrCannotRemoveOwner
	}
	if !contact.IsCollaborator(targetIDStr) {
		return RemoveCollaboratorResult{}, ErrNotCollaborator
	}

	targetName := audit.GetUserName(s.db, targetID)
	managerName := audit.GetUserName(s.db, actorID)

	contact.RemoveCollaborator(targetIDStr)
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error; err != nil {
		s.log.Error("Failed to remove collaborator", "error", err, "contact_id", contact.ID)
		return RemoveCollaboratorResult{}, fmt.Errorf("chat: failed to remove collaborator: %w", err)
	}

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s was removed from the conversation by %s", targetName, managerName),
		models.JSONB{
			"system_type": "collaborator_removed",
			"agent_id":    targetIDStr,
			"removed_by":  actorID.String(),
		})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeCollaboratorLeft,
		Payload: map[string]any{
			"contact_id": contact.ID.String(),
			"user_id":    targetIDStr,
			"user_name":  targetName,
			"removed":    true,
		},
	})

	return RemoveCollaboratorResult{TargetName: targetName, ManagerName: managerName}, nil
}

// Close closes an open conversation. Idempotent: returns ErrAlreadyClosed if
// the chat is already closed (no system message).
var ErrAlreadyClosed = errors.New("chat: conversation already closed")

// Close sets the conversation status to closed and emits the chat_closed
// system message + broadcast. Authorization (owner / collaborator /
// chat.collaborate:write / contacts:read) is checked in the handler; this
// method just performs the transition.
func (s *Service) Close(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) error {
	if contact.EffectiveStatus() == models.ChatStatusClosed {
		return ErrAlreadyClosed
	}

	contact.SetStatus(models.ChatStatusClosed)
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error; err != nil {
		s.log.Error("Failed to close chat", "error", err, "contact_id", contact.ID)
		return fmt.Errorf("chat: failed to close: %w", err)
	}

	agentName := audit.GetUserName(s.db, userID)

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s closed this conversation", agentName),
		models.JSONB{"system_type": "chat_closed", "agent_id": userID.String()})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatClosed,
		Payload: map[string]any{
			"contact_id":       contact.ID.String(),
			"chat_status":      string(models.ChatStatusClosed),
			"closed":           true,
			"assigned_user_id": "",
			"assigned_to":      "",
		},
	})

	return nil
}

// Reopen reopens a closed conversation WITHOUT assigning it to the caller
// (admin/manager only — enforced by the handler via requireAuth). Idempotent:
// returns reopened=false if the chat was already open.
func (s *Service) Reopen(ctx context.Context, orgID, userID uuid.UUID, contact *models.Contact) (bool, error) {
	if contact.EffectiveStatus() == models.ChatStatusOpen {
		return false, nil
	}

	contact.SetStatus(models.ChatStatusOpen)
	if err := s.db.Model(&models.Contact{}).Where("id = ?", contact.ID).
		Update("metadata", contact.Metadata).Error; err != nil {
		s.log.Error("Failed to reopen chat", "error", err, "contact_id", contact.ID)
		return false, fmt.Errorf("chat: failed to reopen: %w", err)
	}

	adminName := audit.GetUserName(s.db, userID)

	s.CreateSystemMessage(orgID, contact.ID,
		fmt.Sprintf("🔔 %s reopened this conversation", adminName),
		models.JSONB{"system_type": "chat_reopened", "agent_id": userID.String()})

	s.broadcast(orgID, websocket.WSMessage{
		Type: websocket.TypeChatReopened,
		Payload: map[string]any{
			"contact_id":  contact.ID.String(),
			"chat_status": string(models.ChatStatusOpen),
			"reopened":    true,
			"by":          userID.String(),
			"by_name":     adminName,
		},
	})

	return true, nil
}
