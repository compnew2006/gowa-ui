package whatsmeow

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/types"
)

func TestShouldMigrateLIDContact_NonEmptyNonJID(t *testing.T) {
	assert.True(t, shouldMigrateLIDContact("15550001234"))
	assert.True(t, shouldMigrateLIDContact("  15550001234  "))
}

func TestShouldMigrateLIDContact_JIDFormat(t *testing.T) {
	assert.False(t, shouldMigrateLIDContact("15550001234@s.whatsapp.net"))
	assert.False(t, shouldMigrateLIDContact("abc@lid"))
}

func TestShouldMigrateLIDContact_Empty(t *testing.T) {
	assert.False(t, shouldMigrateLIDContact(""))
	assert.False(t, shouldMigrateLIDContact("   "))
}

func TestNormalizeDirectSenderIdentity_PhoneChatJIDWins(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	sender := types.NewJID("269638281724102", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("269638281724102", chat, sender)
	assert.Equal(t, "15550001234", result)
}

func TestNormalizeDirectSenderIdentity_HiddenNumericIDPreserved(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	sender := types.NewJID("269638281724102", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("269638281724102", chat, sender)
	assert.Equal(t, "269638281724102@"+string(types.HiddenUserServer), result)
}

func TestNormalizeDirectSenderIdentity_SenderWithAtSign(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	sender := types.NewJID("15559998888", types.DefaultUserServer)

	result := normalizeDirectSenderIdentity("15559998888@s.whatsapp.net", chat, sender)
	assert.Equal(t, "15550001234", result)
}

func TestNormalizeDirectSenderIdentity_SenderHiddenMatchesChat(t *testing.T) {
	chat := types.NewJID("ABC123", types.HiddenUserServer)
	sender := types.NewJID("DEF456", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("ABC123", chat, sender)
	assert.Equal(t, "ABC123@"+string(types.HiddenUserServer), result)
}

func TestNormalizeDirectSenderIdentity_SenderHiddenMatchesSenderJID(t *testing.T) {
	chat := types.NewJID("XYZ789", types.HiddenUserServer)
	sender := types.NewJID("ABC123", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("ABC123", chat, sender)
	assert.Equal(t, "ABC123@"+string(types.HiddenUserServer), result)
}

func TestNormalizeDirectSenderIdentity_PlainPhoneIdentity(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	sender := types.NewJID("269638281724102", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("15559998888", chat, sender)
	assert.Equal(t, "15550001234", result)
}

func TestNormalizeDirectSenderIdentity_EmptyIdentityFallsBackToSenderJID(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	sender := types.NewJID("15550001234", types.DefaultUserServer)

	result := normalizeDirectSenderIdentity("", chat, sender)
	assert.Equal(t, "15550001234", result)
}

func TestNormalizeDirectSenderIdentity_EmptyIdentityFallsBackToHiddenChatJID(t *testing.T) {
	chat := types.NewJID("ABC123", types.HiddenUserServer)
	sender := types.JID{}

	result := normalizeDirectSenderIdentity("", chat, sender)
	assert.Equal(t, "ABC123@"+string(types.HiddenUserServer), result)
}

func TestNormalizeDirectSenderIdentity_EmptyIdentityFallsBackToHiddenSenderJID(t *testing.T) {
	chat := types.JID{}
	sender := types.NewJID("ABC123", types.HiddenUserServer)

	result := normalizeDirectSenderIdentity("", chat, sender)
	assert.Equal(t, "ABC123@"+string(types.HiddenUserServer), result)
}

func TestNormalizeDirectSenderIdentity_AllEmptyReturnsEmpty(t *testing.T) {
	chat := types.JID{}
	sender := types.JID{}

	result := normalizeDirectSenderIdentity("", chat, sender)
	assert.Equal(t, "", result)
}

func TestInferPhoneFromWAMID_ValidBase64(t *testing.T) {
	wamid := "wamid.HBgMNTU1NTEyMzQ1NjcVAgASGBQzRUE5QTY4N0Q4Q0Y2Q0E3QjQ2AA=="
	result := inferPhoneFromWAMID(wamid)
	assert.Equal(t, "55551234567", result)
}

func TestInferPhoneFromWAMID_InvalidBase64(t *testing.T) {
	assert.Empty(t, inferPhoneFromWAMID("wamid.invalid"))
}

func TestInferPhoneFromWAMID_Empty(t *testing.T) {
	assert.Empty(t, inferPhoneFromWAMID(""))
}

func TestInferPhoneFromWAMID_MissingPrefix(t *testing.T) {
	assert.Empty(t, inferPhoneFromWAMID("some-random-string"))
}

func TestInferPhoneFromWAMID_PrefixOnly(t *testing.T) {
	assert.Empty(t, inferPhoneFromWAMID("wamid."))
	assert.Empty(t, inferPhoneFromWAMID("wamid.   "))
}

func TestInferPhoneFromWAMID_NoPhoneMatch(t *testing.T) {
	payload := "wamid." + "SGVsbG8gV29ybGQ="
	assert.Empty(t, inferPhoneFromWAMID(payload))
}

func TestInferPhoneFromWAMID_SelectsLongestPhone(t *testing.T) {
	encoded := "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNA=="
	wamid := "wamid." + encoded
	result := inferPhoneFromWAMID(wamid)
	assert.Equal(t, "123456789012345", result)
}

func TestDirectConversationID_WithPeerIdentity(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	assert.Equal(t, "201007181781@s.whatsapp.net", directConversationID(chat, "201007181781"))
}

func TestDirectConversationID_WithJIDPeerIdentity(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	jidPeer := "269638281724102@" + string(types.HiddenUserServer)
	assert.Equal(t, jidPeer, directConversationID(chat, jidPeer))
}

func TestDirectConversationID_EmptyPeerIdentity(t *testing.T) {
	chat := types.NewJID("269638281724102", types.HiddenUserServer)
	assert.Equal(t, chat.String(), directConversationID(chat, ""))
}

func TestDirectConversationID_EmptyPeerWithWhitespace(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	assert.Equal(t, chat.String(), directConversationID(chat, "   "))
}

func TestBuildMessageMediaURL(t *testing.T) {
	id := uuid.MustParse("0195a1b2-c3d4-e5f6-7890-abcdef123456")
	expected := "/api/media/0195a1b2-c3d4-e5f6-7890-abcdef123456"
	assert.Equal(t, expected, buildMessageMediaURL(id))
}

func TestBuildMessageMediaURL_DifferentUUID(t *testing.T) {
	id := uuid.Nil
	expected := "/api/media/00000000-0000-0000-0000-000000000000"
	assert.Equal(t, expected, buildMessageMediaURL(id))
}

func TestBuildInboundMediaRecoveryJob_NilMessage(t *testing.T) {
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "image",
		MediaPayloadBase64: "aGVsbG8=",
	}
	_, err := buildInboundMediaRecoveryJob(nil, artifact)
	assert.EqualError(t, err, "message is nil")
}

func TestBuildInboundMediaRecoveryJob_NilArtifact(t *testing.T) {
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
	}
	msg.InstanceID = &[]uuid.UUID{uuid.New()}[0]

	_, err := buildInboundMediaRecoveryJob(msg, nil)
	assert.EqualError(t, err, "inbound media retry artifact is nil")
}

func TestBuildInboundMediaRecoveryJob_NilInstanceID(t *testing.T) {
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "image",
		MediaPayloadBase64: "aGVsbG8=",
	}

	_, err := buildInboundMediaRecoveryJob(msg, artifact)
	assert.ErrorContains(t, err, "has no instance id")
}

func TestBuildInboundMediaRecoveryJob_NilUUIDInstanceID(t *testing.T) {
	nilUUID := uuid.Nil
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
		InstanceID:     &nilUUID,
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "image",
		MediaPayloadBase64: "aGVsbG8=",
	}

	_, err := buildInboundMediaRecoveryJob(msg, artifact)
	assert.ErrorContains(t, err, "has no instance id")
}

func TestBuildInboundMediaRecoveryJob_EmptyMediaPayload(t *testing.T) {
	instID := uuid.New()
	msg := &models.Message{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: uuid.New(),
		InstanceID:     &instID,
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "image",
		MediaPayloadBase64: "   ",
	}

	_, err := buildInboundMediaRecoveryJob(msg, artifact)
	assert.ErrorContains(t, err, "missing media payload")
}

func TestBuildInboundMediaRecoveryJob_EmptyMediaKind(t *testing.T) {
	instID := uuid.New()
	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    uuid.New(),
		InstanceID:        &instID,
		WhatsAppMessageID: "wamid-test-1",
		MessageType:       models.MessageTypeImage,
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "",
		MediaPayloadBase64: "aGVsbG8=",
	}

	_, err := buildInboundMediaRecoveryJob(msg, artifact)
	assert.ErrorContains(t, err, "missing media kind")
}

func TestBuildInboundMediaRecoveryJob_Success(t *testing.T) {
	orgID := uuid.New()
	instID := uuid.New()
	msgID := uuid.New()
	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: msgID},
		OrganizationID:    orgID,
		InstanceID:        &instID,
		WhatsAppMessageID: " wamid-success ",
		MessageType:       models.MessageTypeImage,
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          " image ",
		MediaPayloadBase64: " aGVsbG8= ",
		MimeType:           " image/jpeg ",
		FallbackFilename:   " photo.jpg ",
		LastError:          " download failed ",
	}

	job, err := buildInboundMediaRecoveryJob(msg, artifact)
	require.NoError(t, err)
	assert.Equal(t, msgID, job.MessageID)
	assert.Equal(t, orgID, job.OrganizationID)
	assert.Equal(t, instID, job.InstanceID)
	assert.Equal(t, "wamid-success", job.WhatsAppMessageID)
	assert.Equal(t, "image", job.MediaKind)
	assert.Equal(t, "image/jpeg", job.MimeType)
	assert.Equal(t, "photo.jpg", job.FallbackFilename)
	assert.Equal(t, "aGVsbG8=", job.MediaPayloadBase64)
	assert.Equal(t, "download failed", job.LastError)
}

func TestBuildInboundMediaRecoveryJob_TrimsWhitespace(t *testing.T) {
	orgID := uuid.New()
	instID := uuid.New()
	msgID := uuid.New()
	msg := &models.Message{
		BaseModel:         models.BaseModel{ID: msgID},
		OrganizationID:    orgID,
		InstanceID:        &instID,
		WhatsAppMessageID: "  wamid-trim  ",
		MessageType:       models.MessageTypeDocument,
	}
	artifact := &inboundMediaRetryArtifact{
		MediaKind:          "  document  ",
		MediaPayloadBase64: "  ZGF0YQ==  ",
		MimeType:           "  application/pdf  ",
		FallbackFilename:   "  file.pdf  ",
		LastError:          "  timeout  ",
	}

	job, err := buildInboundMediaRecoveryJob(msg, artifact)
	require.NoError(t, err)
	assert.Equal(t, "wamid-trim", job.WhatsAppMessageID)
	assert.Equal(t, "document", job.MediaKind)
	assert.Equal(t, "application/pdf", job.MimeType)
	assert.Equal(t, "file.pdf", job.FallbackFilename)
	assert.Equal(t, "ZGF0YQ==", job.MediaPayloadBase64)
	assert.Equal(t, "timeout", job.LastError)
}

func TestEnqueueInboundMediaRecovery_NilConnectionManager(t *testing.T) {
	var cm *ConnectionManager
	err := cm.enqueueInboundMediaRecovery(nil, nil)
	assert.EqualError(t, err, "connection manager is nil")
}

func TestEnqueueInboundMediaRecovery_NilQueue(t *testing.T) {
	cm := &ConnectionManager{}
	err := cm.enqueueInboundMediaRecovery(nil, nil)
	assert.EqualError(t, err, "inbound media queue is not configured")
}

func TestEnqueueInboundMediaRecovery_NilJob(t *testing.T) {
	cm := &ConnectionManager{
		inboundMediaQueue: &mockInboundMediaQueue{},
	}
	err := cm.enqueueInboundMediaRecovery(nil, nil)
	assert.EqualError(t, err, "inbound media recovery job is nil")
}

type mockInboundMediaQueue struct {
	enqueued bool
	err      error
}

func (m *mockInboundMediaQueue) EnqueueInboundMedia(_ context.Context, _ *queue.InboundMediaJob) error {
	m.enqueued = true
	return m.err
}

func TestInboundMediaAsyncStatusConstants(t *testing.T) {
	assert.Equal(t, "queued", inboundMediaAsyncStatusQueued)
	assert.Equal(t, "enqueue_failed", inboundMediaAsyncStatusEnqueueFail)
	assert.Equal(t, "succeeded", inboundMediaAsyncStatusSucceeded)
	assert.Equal(t, "failed", inboundMediaAsyncStatusFailed)
}
