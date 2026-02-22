package whatsmeow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/compnew2006/whatomate/internal/models"
	waClient "go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type incomingCallPayload struct {
	CallID       string
	RejectTarget types.JID
	Caller       types.JID
	CallerAlt    types.JID
	GroupJID     types.JID
	Media        string
	Source       string
	Timestamp    time.Time
}

type callMessagePersistInput struct {
	InstanceID     uuid.UUID
	OrgID          uuid.UUID
	Contact        *models.Contact
	ConversationID string
	Direction      models.Direction
	Status         models.MessageStatus
	Content        string
	WAMID          string
	ErrorMessage   string
	Metadata       models.JSONB
	CreatedAt      time.Time
	CallerPhone    string
	ProfileName    string
}

func (cm *ConnectionManager) handleCallOffer(ctx context.Context, evt *events.CallOffer, instanceID, orgID uuid.UUID) {
	if evt == nil {
		return
	}

	payload := incomingCallPayload{
		CallID:       strings.TrimSpace(evt.CallID),
		RejectTarget: evt.From.ToNonAD(),
		Caller:       evt.CallCreator.ToNonAD(),
		CallerAlt:    evt.CallCreatorAlt.ToNonAD(),
		GroupJID:     evt.GroupJID.ToNonAD(),
		Media:        normalizeCallMedia(extractCallMediaFromNode(evt.Data)),
		Source:       "offer",
		Timestamp:    evt.Timestamp,
	}
	cm.handleIncomingCallAutoReject(ctx, instanceID, orgID, payload)
}

func (cm *ConnectionManager) handleCallOfferNotice(ctx context.Context, evt *events.CallOfferNotice, instanceID, orgID uuid.UUID) {
	if evt == nil {
		return
	}

	payload := incomingCallPayload{
		CallID:       strings.TrimSpace(evt.CallID),
		RejectTarget: evt.From.ToNonAD(),
		Caller:       evt.CallCreator.ToNonAD(),
		CallerAlt:    evt.CallCreatorAlt.ToNonAD(),
		GroupJID:     evt.GroupJID.ToNonAD(),
		Media:        normalizeCallMedia(evt.Media),
		Source:       "offer_notice",
		Timestamp:    evt.Timestamp,
	}
	cm.handleIncomingCallAutoReject(ctx, instanceID, orgID, payload)
}

func (cm *ConnectionManager) handleIncomingCallAutoReject(ctx context.Context, instanceID, orgID uuid.UUID, payload incomingCallPayload) {
	client := cm.GetClient(instanceID)
	if client == nil {
		cm.logger.Warn("Skipping call auto-reject for disconnected instance", "instance_id", instanceID, "call_id", payload.CallID)
		return
	}

	settings, err := cm.loadAutoRejectCallSettings(ctx, instanceID)
	if err != nil {
		cm.logger.Warn("Failed to load auto-reject settings", "instance_id", instanceID, "call_id", payload.CallID, "error", err)
		return
	}

	callerPhone := cm.resolveCallCallerPhone(ctx, payload.Caller, payload.CallerAlt)
	isGroupCall := !payload.GroupJID.IsEmpty()
	decision := EvaluateAutoRejectCall(settings, time.Now(), isGroupCall, cm.activeCallCount(instanceID), callerPhone)
	if !decision.ShouldReject {
		cm.logger.Debug(
			"Incoming call did not match auto-reject rules",
			"instance_id", instanceID,
			"call_id", payload.CallID,
			"reason", decision.Reason,
			"caller", callerPhone,
		)
		return
	}

	rejectTarget := payload.RejectTarget
	if rejectTarget.IsEmpty() {
		rejectTarget = payload.Caller
	}
	if rejectTarget.IsEmpty() {
		rejectTarget = payload.CallerAlt
	}
	if rejectTarget.IsEmpty() {
		cm.logger.Warn("Unable to resolve call reject target", "instance_id", instanceID, "call_id", payload.CallID)
		return
	}
	if payload.CallID == "" {
		cm.logger.Warn("Unable to auto-reject call without call ID", "instance_id", instanceID)
		return
	}

	if err := cm.rejectIncomingCall(ctx, instanceID, rejectTarget, payload.CallID); err != nil {
		cm.logger.Error(
			"Failed to auto-reject incoming call",
			"instance_id", instanceID,
			"call_id", payload.CallID,
			"caller", callerPhone,
			"error", err,
		)
		cm.MarkError(instanceID)
		return
	}

	contact, conversationID, profileName, resolvedCallerPhone, err := cm.findOrCreateCallContact(ctx, client, orgID, instanceID, payload.Caller, payload.CallerAlt)
	if err != nil {
		cm.logger.Error("Failed to resolve contact for auto-rejected call", "instance_id", instanceID, "call_id", payload.CallID, "error", err)
		cm.MarkError(instanceID)
		return
	}

	callType := payload.Media
	if callType == "" {
		callType = "voice"
	}

	rejectedContent := fmt.Sprintf("Auto-rejected incoming %s call.", callType)
	rejectedMetadata := models.JSONB{
		"system_event":       "call_auto_rejected",
		"call_id":            payload.CallID,
		"call_type":          callType,
		"is_group_call":      isGroupCall,
		"auto_reject_source": payload.Source,
	}
	if !payload.GroupJID.IsEmpty() {
		rejectedMetadata["group_jid"] = payload.GroupJID.String()
	}
	if !payload.Caller.IsEmpty() {
		rejectedMetadata["caller_jid"] = payload.Caller.String()
	}

	createdAt := payload.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	if _, err := cm.persistCallTextMessage(ctx, client, callMessagePersistInput{
		InstanceID:     instanceID,
		OrgID:          orgID,
		Contact:        contact,
		ConversationID: conversationID,
		Direction:      models.DirectionIncoming,
		Status:         models.MessageStatusReceived,
		Content:        rejectedContent,
		Metadata:       rejectedMetadata,
		CreatedAt:      createdAt,
		CallerPhone:    resolvedCallerPhone,
		ProfileName:    profileName,
	}); err != nil {
		cm.logger.Error("Failed to persist auto-rejected call event", "instance_id", instanceID, "call_id", payload.CallID, "error", err)
		cm.MarkError(instanceID)
	}

	if strings.TrimSpace(decision.ReplyMessage) == "" {
		cm.logger.Info("Incoming call auto-rejected", "instance_id", instanceID, "call_id", payload.CallID, "caller", resolvedCallerPhone, "mode", settings.Mode)
		return
	}

	wamID, sendErr := cm.sendAutoRejectTextReply(ctx, instanceID, rejectTarget.ToNonAD(), decision.ReplyMessage)
	autoReplyStatus := models.MessageStatusSent
	autoReplyError := ""
	if sendErr != nil {
		autoReplyStatus = models.MessageStatusFailed
		autoReplyError = sendErr.Error()
		cm.logger.Error("Failed to send auto-reject reply", "instance_id", instanceID, "call_id", payload.CallID, "caller", resolvedCallerPhone, "error", sendErr)
		cm.MarkError(instanceID)
	}

	autoReplyMetadata := models.JSONB{
		"system_event": "call_auto_reject_reply",
		"call_id":      payload.CallID,
		"call_type":    callType,
	}
	if _, err := cm.persistCallTextMessage(ctx, client, callMessagePersistInput{
		InstanceID:     instanceID,
		OrgID:          orgID,
		Contact:        contact,
		ConversationID: conversationID,
		Direction:      models.DirectionOutgoing,
		Status:         autoReplyStatus,
		Content:        decision.ReplyMessage,
		WAMID:          wamID,
		ErrorMessage:   autoReplyError,
		Metadata:       autoReplyMetadata,
		CreatedAt:      time.Now(),
		CallerPhone:    resolvedCallerPhone,
		ProfileName:    profileName,
	}); err != nil {
		cm.logger.Error("Failed to persist auto-reject reply message", "instance_id", instanceID, "call_id", payload.CallID, "error", err)
		cm.MarkError(instanceID)
	}

	cm.logger.Info("Incoming call auto-rejected", "instance_id", instanceID, "call_id", payload.CallID, "caller", resolvedCallerPhone, "mode", settings.Mode)
}

func (cm *ConnectionManager) rejectIncomingCall(ctx context.Context, instanceID uuid.UUID, from types.JID, callID string) error {
	client := cm.GetClient(instanceID)
	if client == nil {
		return fmt.Errorf("client not connected")
	}
	if callID == "" {
		return fmt.Errorf("call ID is empty")
	}
	if from.IsEmpty() {
		return fmt.Errorf("call source is empty")
	}

	rejectCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := client.RejectCall(rejectCtx, from.ToNonAD(), callID); err != nil {
		return err
	}
	return nil
}

func (cm *ConnectionManager) sendAutoRejectTextReply(ctx context.Context, instanceID uuid.UUID, to types.JID, message string) (string, error) {
	client := cm.GetClient(instanceID)
	if client == nil {
		return "", fmt.Errorf("client not connected")
	}
	if to.IsEmpty() {
		return "", fmt.Errorf("recipient is empty")
	}
	text := strings.TrimSpace(message)
	if text == "" {
		return "", nil
	}

	sendCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	resp, err := client.SendMessage(sendCtx, to.ToNonAD(), &waE2E.Message{Conversation: proto.String(text)})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (cm *ConnectionManager) findOrCreateCallContact(
	ctx context.Context,
	client *waClient.Client,
	orgID, instanceID uuid.UUID,
	caller types.JID,
	callerAlt types.JID,
) (*models.Contact, string, string, string, error) {
	callerPhone := cm.resolveCallCallerPhone(ctx, caller, callerAlt)
	if callerPhone == "" {
		return nil, "", "", "", fmt.Errorf("unable to resolve caller phone")
	}

	profileName := cm.resolveStoredContactName(ctx, client, caller.ToNonAD(), callerPhone)
	if profileName == "" && !callerAlt.IsEmpty() {
		profileName = cm.resolveStoredContactName(ctx, client, callerAlt.ToNonAD(), callerPhone)
	}
	if profileName == "" {
		profileName = callerPhone
	}

	contact, err := cm.findOrCreateContact(ctx, orgID, instanceID, callerPhone, profileName, models.JSONB{})
	if err != nil {
		return nil, "", "", "", err
	}

	conversationID := canonicalCallConversationID(caller, callerAlt, callerPhone)
	if conversationID == "" {
		conversationID = callerPhone
	}

	return contact, conversationID, profileName, callerPhone, nil
}

func (cm *ConnectionManager) persistCallTextMessage(ctx context.Context, client *waClient.Client, input callMessagePersistInput) (*models.Message, error) {
	if input.Contact == nil {
		return nil, fmt.Errorf("contact is required")
	}

	instanceID := input.InstanceID
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	metadata := cloneJSONBMap(input.Metadata)
	if metadata == nil {
		metadata = models.JSONB{}
	}

	message := models.Message{
		BaseModel:         models.BaseModel{CreatedAt: createdAt},
		OrganizationID:    input.OrgID,
		InstanceID:        &instanceID,
		WhatsAppAccount:   resolveWhatsmeowAccountID(client),
		ContactID:         input.Contact.ID,
		WhatsAppMessageID: strings.TrimSpace(input.WAMID),
		ConversationID:    input.ConversationID,
		Direction:         input.Direction,
		MessageType:       models.MessageTypeText,
		Content:           strings.TrimSpace(input.Content),
		Status:            input.Status,
		ErrorMessage:      strings.TrimSpace(input.ErrorMessage),
		Metadata:          metadata,
	}

	if err := cm.db.WithContext(ctx).Create(&message).Error; err != nil {
		return nil, err
	}

	preview := message.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	contactUpdates := map[string]any{
		"last_message_at":      message.CreatedAt,
		"last_message_preview": preview,
	}
	if input.Direction == models.DirectionIncoming {
		contactUpdates["is_read"] = false
	}
	if err := cm.db.WithContext(ctx).
		Model(&models.Contact{}).
		Where("id = ?", input.Contact.ID).
		Updates(contactUpdates).Error; err != nil {
		cm.logger.Warn("Failed to update contact after call event", "contact_id", input.Contact.ID, "error", err)
	}

	message.Contact = input.Contact
	cm.broadcastPersistedMessage(
		input.OrgID,
		&message,
		input.Contact,
		false,
		false,
		input.CallerPhone,
		input.ProfileName,
		incomingReplyContext{},
	)

	return &message, nil
}

func resolveWhatsmeowAccountID(client *waClient.Client) string {
	if client == nil || client.Store == nil || client.Store.ID == nil {
		return "whatsmeow"
	}
	if client.Store.ID.User == "" {
		return "whatsmeow"
	}
	return client.Store.ID.User
}

func canonicalCallConversationID(caller types.JID, callerAlt types.JID, callerPhone string) string {
	if callerPhone != "" {
		return callerPhone + "@s.whatsapp.net"
	}
	if !caller.IsEmpty() {
		return caller.ToNonAD().String()
	}
	if !callerAlt.IsEmpty() {
		return callerAlt.ToNonAD().String()
	}
	return ""
}

func normalizeCallMedia(raw string) string {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(candidate, "video"):
		return "video"
	case strings.Contains(candidate, "audio"), strings.Contains(candidate, "voice"):
		return "voice"
	default:
		return "voice"
	}
}

func extractCallMediaFromNode(node *waBinary.Node) string {
	if node == nil {
		return ""
	}
	attr := node.AttrGetter()
	candidates := []string{
		attr.String("media"),
		attr.String("type"),
		attr.String("call_type"),
		attr.String("call-type"),
		attr.String("video"),
	}
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (cm *ConnectionManager) resolveCallCallerPhone(ctx context.Context, caller types.JID, callerAlt types.JID) string {
	caller = caller.ToNonAD()
	callerAlt = callerAlt.ToNonAD()

	if callerAlt.Server == types.DefaultUserServer && callerAlt.User != "" {
		return callerAlt.User
	}
	if callerAlt.Server == types.HiddenUserServer && callerAlt.User != "" {
		if pn := cm.lookupPNForLID(ctx, callerAlt.User); pn != "" {
			return pn
		}
	}

	if caller.Server == types.DefaultUserServer && caller.User != "" {
		return caller.User
	}
	if caller.Server == types.HiddenUserServer && caller.User != "" {
		if pn := cm.lookupPNForLID(ctx, caller.User); pn != "" {
			return pn
		}
	}

	if callerAlt.User != "" {
		return callerAlt.User
	}
	return caller.User
}

func (cm *ConnectionManager) markCallActive(instanceID uuid.UUID, callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}

	cm.activeCallsMu.Lock()
	defer cm.activeCallsMu.Unlock()

	if cm.activeCallIDs == nil {
		cm.activeCallIDs = make(map[uuid.UUID]map[string]struct{})
	}
	if _, ok := cm.activeCallIDs[instanceID]; !ok {
		cm.activeCallIDs[instanceID] = make(map[string]struct{})
	}
	cm.activeCallIDs[instanceID][callID] = struct{}{}
}

func (cm *ConnectionManager) markCallEnded(instanceID uuid.UUID, callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}

	cm.activeCallsMu.Lock()
	defer cm.activeCallsMu.Unlock()

	callSet, ok := cm.activeCallIDs[instanceID]
	if !ok {
		return
	}
	delete(callSet, callID)
	if len(callSet) == 0 {
		delete(cm.activeCallIDs, instanceID)
	}
}

func (cm *ConnectionManager) activeCallCount(instanceID uuid.UUID) int {
	cm.activeCallsMu.Lock()
	defer cm.activeCallsMu.Unlock()

	callSet, ok := cm.activeCallIDs[instanceID]
	if !ok {
		return 0
	}
	return len(callSet)
}

func (cm *ConnectionManager) clearActiveCalls(instanceID uuid.UUID) {
	cm.activeCallsMu.Lock()
	defer cm.activeCallsMu.Unlock()
	delete(cm.activeCallIDs, instanceID)
}
