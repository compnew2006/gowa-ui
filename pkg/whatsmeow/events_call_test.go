package whatsmeow

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/types"
)

func TestNormalizeCallMedia_Video(t *testing.T) {
	assert.Equal(t, "video", normalizeCallMedia("video"))
	assert.Equal(t, "video", normalizeCallMedia("Video Call"))
	assert.Equal(t, "video", normalizeCallMedia("  VIDEO  "))
}

func TestNormalizeCallMedia_Voice(t *testing.T) {
	assert.Equal(t, "voice", normalizeCallMedia("audio"))
	assert.Equal(t, "voice", normalizeCallMedia("voice"))
	assert.Equal(t, "voice", normalizeCallMedia("Voice Call"))
	assert.Equal(t, "voice", normalizeCallMedia("  AUDIO  "))
}

func TestNormalizeCallMedia_UnknownDefaultsToVoice(t *testing.T) {
	assert.Equal(t, "voice", normalizeCallMedia(""))
	assert.Equal(t, "voice", normalizeCallMedia("unknown"))
	assert.Equal(t, "voice", normalizeCallMedia("   "))
}

func TestNormalizeCallMedia_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "video", normalizeCallMedia("VIDEO"))
	assert.Equal(t, "video", normalizeCallMedia("ViDeO"))
	assert.Equal(t, "voice", normalizeCallMedia("AUDIO"))
	assert.Equal(t, "voice", normalizeCallMedia("AuDiO"))
}

func TestCanonicalCallConversationID_WithPhone(t *testing.T) {
	caller := types.NewJID("ABC123", types.HiddenUserServer)
	result := canonicalCallConversationID(caller, types.JID{}, "15550001234")
	assert.Equal(t, "15550001234@s.whatsapp.net", result)
}

func TestCanonicalCallConversationID_FallsBackToCaller(t *testing.T) {
	caller := types.NewJID("15550001234", types.DefaultUserServer)
	result := canonicalCallConversationID(caller, types.JID{}, "")
	assert.Equal(t, "15550001234@s.whatsapp.net", result)
}

func TestCanonicalCallConversationID_FallsBackToCallerAlt(t *testing.T) {
	callerAlt := types.NewJID("15550009999", types.DefaultUserServer)
	result := canonicalCallConversationID(types.JID{}, callerAlt, "")
	assert.Equal(t, "15550009999@s.whatsapp.net", result)
}

func TestCanonicalCallConversationID_AllEmpty(t *testing.T) {
	result := canonicalCallConversationID(types.JID{}, types.JID{}, "")
	assert.Equal(t, "", result)
}

func TestCanonicalCallConversationID_PhoneTakesPriority(t *testing.T) {
	caller := types.NewJID("9999999999", types.DefaultUserServer)
	callerAlt := types.NewJID("8888888888", types.DefaultUserServer)
	result := canonicalCallConversationID(caller, callerAlt, "15550001234")
	assert.Equal(t, "15550001234@s.whatsapp.net", result)
}

func TestResolveWhatsmeowAccountID_NilClient(t *testing.T) {
	assert.Equal(t, "whatsmeow", resolveWhatsmeowAccountID(nil))
}

func TestMarkCallActive_EmptyCallID(t *testing.T) {
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}
	cm.markCallActive(uuid.New(), "")
	cm.markCallActive(uuid.New(), "   ")

	assert.Empty(t, cm.activeCallIDs)
}

func TestMarkCallActive_AddsCall(t *testing.T) {
	instanceID := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}

	cm.markCallActive(instanceID, "call-1")

	assert.Contains(t, cm.activeCallIDs, instanceID)
	assert.Contains(t, cm.activeCallIDs[instanceID], "call-1")
	assert.Equal(t, 1, cm.activeCallCount(instanceID))
}

func TestMarkCallActive_MultipleCalls(t *testing.T) {
	instanceID := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}

	cm.markCallActive(instanceID, "call-1")
	cm.markCallActive(instanceID, "call-2")

	assert.Equal(t, 2, cm.activeCallCount(instanceID))
}

func TestMarkCallActive_MultipleInstances(t *testing.T) {
	inst1 := uuid.New()
	inst2 := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}

	cm.markCallActive(inst1, "call-a")
	cm.markCallActive(inst2, "call-b")

	assert.Equal(t, 1, cm.activeCallCount(inst1))
	assert.Equal(t, 1, cm.activeCallCount(inst2))
}

func TestMarkCallEnded_EmptyCallID(t *testing.T) {
	instanceID := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: map[uuid.UUID]map[string]struct{}{
			instanceID: {"call-1": {}},
		},
	}

	cm.markCallEnded(instanceID, "")
	assert.Contains(t, cm.activeCallIDs[instanceID], "call-1")
}

func TestMarkCallEnded_RemovesCall(t *testing.T) {
	instanceID := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: map[uuid.UUID]map[string]struct{}{
			instanceID: {"call-1": {}, "call-2": {}},
		},
	}

	cm.markCallEnded(instanceID, "call-1")
	assert.Equal(t, 1, cm.activeCallCount(instanceID))
	assert.NotContains(t, cm.activeCallIDs[instanceID], "call-1")
}

func TestMarkCallEnded_RemovesInstanceWhenEmpty(t *testing.T) {
	instanceID := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: map[uuid.UUID]map[string]struct{}{
			instanceID: {"call-1": {}},
		},
	}

	cm.markCallEnded(instanceID, "call-1")
	assert.NotContains(t, cm.activeCallIDs, instanceID)
}

func TestMarkCallEnded_UnknownInstance(t *testing.T) {
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}
	cm.markCallEnded(uuid.New(), "call-1")
	assert.Empty(t, cm.activeCallIDs)
}

func TestActiveCallCount_NoInstance(t *testing.T) {
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}
	assert.Equal(t, 0, cm.activeCallCount(uuid.New()))
}

func TestActiveCallCount_NilMap(t *testing.T) {
	cm := &ConnectionManager{}
	assert.Equal(t, 0, cm.activeCallCount(uuid.New()))
}

func TestClearActiveCalls(t *testing.T) {
	inst1 := uuid.New()
	inst2 := uuid.New()
	cm := &ConnectionManager{
		activeCallIDs: map[uuid.UUID]map[string]struct{}{
			inst1: {"call-a": {}},
			inst2: {"call-b": {}},
		},
	}

	cm.clearActiveCalls(inst1)
	assert.NotContains(t, cm.activeCallIDs, inst1)
	assert.Contains(t, cm.activeCallIDs, inst2)
}

func TestClearActiveCalls_NoInstance(t *testing.T) {
	cm := &ConnectionManager{
		activeCallIDs: make(map[uuid.UUID]map[string]struct{}),
	}
	cm.clearActiveCalls(uuid.New())
	assert.Empty(t, cm.activeCallIDs)
}

func TestExtractCallMediaFromNode_NilNode(t *testing.T) {
	assert.Empty(t, extractCallMediaFromNode(nil))
}

func TestBuildInboundMediaRecoveryJobForCallEvent_MissingContact(t *testing.T) {
	cm := &ConnectionManager{}
	_, err := cm.persistCallTextMessage(nil, nil, callMessagePersistInput{
		OrgID: uuid.New(),
	})
	assert.EqualError(t, err, "contact is required")
}

func TestDirectConversationID_WithPlainPhone(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	result := directConversationID(chat, "15550001234")
	assert.Equal(t, "15550001234@s.whatsapp.net", result)
}

func TestPersistCallTextMessage_RequiresContact(t *testing.T) {
	cm := &ConnectionManager{}
	_, err := cm.persistCallTextMessage(nil, nil, callMessagePersistInput{
		InstanceID: uuid.New(),
		OrgID:      uuid.New(),
		Contact:    nil,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contact is required")
}

func TestRejectIncomingCall_NilClient(t *testing.T) {
	cm := &ConnectionManager{}
	err := cm.rejectIncomingCall(nil, uuid.New(), types.JID{}, "call-1")
	assert.EqualError(t, err, "client not connected")
}

func TestRejectIncomingCall_EmptyCallID(t *testing.T) {
	err := (&ConnectionManager{}).rejectIncomingCall(nil, uuid.New(), types.JID{}, "")
	assert.EqualError(t, err, "client not connected")
}

func TestRejectIncomingCall_EmptyFrom(t *testing.T) {
	err := (&ConnectionManager{}).rejectIncomingCall(nil, uuid.New(), types.JID{}, "call-1")
	assert.EqualError(t, err, "client not connected")
}

func TestSendAutoRejectTextReply_NilClient(t *testing.T) {
	cm := &ConnectionManager{}
	_, err := cm.sendAutoRejectTextReply(nil, uuid.New(), types.JID{}, "hello")
	assert.EqualError(t, err, "client not connected")
}

func TestSendAutoRejectTextReply_NotConnected(t *testing.T) {
	cm := &ConnectionManager{}
	_, err := cm.sendAutoRejectTextReply(nil, uuid.New(), types.JID{}, "hello")
	assert.EqualError(t, err, "client not connected")
}

func TestResolveCallCallerPhone_PreferCallerAltDefaultUserServer(t *testing.T) {
	cm := &ConnectionManager{}
	caller := types.NewJID("ABC123", types.HiddenUserServer)
	callerAlt := types.NewJID("15550001234", types.DefaultUserServer)

	result := cm.resolveCallCallerPhone(nil, caller, callerAlt)
	assert.Equal(t, "15550001234", result)
}

func TestResolveCallCallerPhone_FallsBackToCallerDefaultUserServer(t *testing.T) {
	cm := &ConnectionManager{}
	caller := types.NewJID("15550009999", types.DefaultUserServer)
	callerAlt := types.NewJID("ABC123", types.HiddenUserServer)

	result := cm.resolveCallCallerPhone(nil, caller, callerAlt)
	assert.Equal(t, "15550009999", result)
}

func TestResolveCallCallerPhone_FallsBackToCallerAltUser(t *testing.T) {
	cm := &ConnectionManager{}
	caller := types.NewJID("", types.HiddenUserServer)
	callerAlt := types.NewJID("XYZ789", types.HiddenUserServer)

	result := cm.resolveCallCallerPhone(nil, caller, callerAlt)
	assert.Equal(t, "XYZ789", result)
}

func TestResolveCallCallerPhone_FallsBackToCallerUser(t *testing.T) {
	cm := &ConnectionManager{}
	caller := types.NewJID("ABC123", types.HiddenUserServer)
	callerAlt := types.JID{}

	result := cm.resolveCallCallerPhone(nil, caller, callerAlt)
	assert.Equal(t, "ABC123", result)
}

func TestFindOrCreateCallContact_EmptyCallerPhone(t *testing.T) {
	cm := &ConnectionManager{}
	_, _, _, _, err := cm.findOrCreateCallContact(nil, nil, uuid.New(), uuid.New(), types.JID{}, types.JID{})
	assert.EqualError(t, err, "unable to resolve caller phone")
}

func TestIncomingCallPayloadDefaults(t *testing.T) {
	payload := incomingCallPayload{
		CallID:    "test-call-id",
		Timestamp: time.Time{},
	}
	assert.Equal(t, "test-call-id", payload.CallID)
	assert.True(t, payload.RejectTarget.IsEmpty())
	assert.True(t, payload.Caller.IsEmpty())
	assert.True(t, payload.GroupJID.IsEmpty())
}

type whatsmeowClientMock struct {
	storeNil bool
	idNil    bool
	user     string
}
