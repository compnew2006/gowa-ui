package whatsmeow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	waClient "go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const (
	maxMessageUnwrapDepth = 8
	deletedMessageCaption = "(This message was deleted)"

	defaultInboundMediaDownloadMaxAttempts = 3
	defaultInboundMediaDownloadBaseBackoff = 500 * time.Millisecond
	defaultInboundMediaDownloadMaxBackoff  = 2 * time.Second
)

var mentionTokenPattern = regexp.MustCompile(`@\d+`)

type inboundMediaRetryArtifact struct {
	MediaKind          string
	MediaPayloadBase64 string
	MimeType           string
	FallbackFilename   string
	LastError          string
}

type inboundMediaDownloader interface {
	Download(ctx context.Context, msg waClient.DownloadableMessage) ([]byte, error)
}

// extractMessageContentWithMedia extracts message content and persists inbound media locally.
// It handles wrapped message formats (ephemeral/view-once/document-with-caption) and populates media_url.
func (cm *ConnectionManager) extractMessageContentWithMedia(ctx context.Context, client *waClient.Client, msg *waE2E.Message) (models.MessageType, string, string, string, string) {
	msgType, content, mediaURL, mimeType, filename, _ := cm.extractMessageContentWithMediaRetryArtifact(ctx, client, msg)
	return msgType, content, mediaURL, mimeType, filename
}

func (cm *ConnectionManager) extractMessageContentWithMediaRetryArtifact(
	ctx context.Context,
	client *waClient.Client,
	msg *waE2E.Message,
) (models.MessageType, string, string, string, string, *inboundMediaRetryArtifact) {
	if isIncomingRevokeMessage(msg) {
		return models.MessageTypeText, deletedMessageCaption, "", "", "", nil
	}

	// Ignore key-distribution control messages (group encryption technical payloads).
	if msg != nil && (msg.SenderKeyDistributionMessage != nil || msg.FastRatchetKeySenderKeyDistributionMessage != nil) {
		return models.MessageTypeIgnore, "", "", "", "", nil
	}

	unwrapped := unwrapIncomingMessage(msg)
	if unwrapped == nil {
		return models.MessageTypeText, "", "", "", "", nil
	}
	if unwrapped.GetSenderKeyDistributionMessage() != nil || unwrapped.GetFastRatchetKeySenderKeyDistributionMessage() != nil {
		return models.MessageTypeIgnore, "", "", "", "", nil
	}

	if protocol := unwrapped.GetProtocolMessage(); protocol != nil &&
		protocol.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT &&
		protocol.GetEditedMessage() != nil {
		return cm.extractMessageContentWithMediaRetryArtifact(ctx, client, protocol.GetEditedMessage())
	}

	if msgType, content, ok := cm.extractTextualIncomingMessage(ctx, client, unwrapped); ok {
		return msgType, content, "", "", "", nil
	}
	if img := unwrapped.ImageMessage; img != nil {
		caption := img.GetCaption()
		mimeType := sanitizeMimeType(img.GetMimetype(), "image/jpeg")
		filename := defaultMediaFilename("image", mimeType, "image.jpg")
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, img, models.MessageTypeImage, mimeType, filename)
		return models.MessageTypeImage, caption, mediaURL, mimeType, filename, artifact
	}

	if sticker := unwrapped.StickerMessage; sticker != nil {
		mimeType := sanitizeMimeType(sticker.GetMimetype(), "image/webp")
		filename := defaultMediaFilename("sticker", mimeType, "sticker.webp")
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, sticker, models.MessageTypeSticker, mimeType, filename)
		return models.MessageTypeSticker, "", mediaURL, mimeType, filename, artifact
	}

	if vid := unwrapped.VideoMessage; vid != nil {
		caption := vid.GetCaption()
		mimeType := sanitizeMimeType(vid.GetMimetype(), "video/mp4")
		filename := defaultMediaFilename("video", mimeType, "video.mp4")
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, vid, models.MessageTypeVideo, mimeType, filename)
		return models.MessageTypeVideo, caption, mediaURL, mimeType, filename, artifact
	}

	// PTV (video notes) is delivered as a separate field but behaves like video media.
	if ptv := unwrapped.PtvMessage; ptv != nil {
		caption := ptv.GetCaption()
		mimeType := sanitizeMimeType(ptv.GetMimetype(), "video/mp4")
		filename := defaultMediaFilename("video-note", mimeType, "video.mp4")
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, ptv, models.MessageTypeVideo, mimeType, filename)
		return models.MessageTypeVideo, caption, mediaURL, mimeType, filename, artifact
	}

	if aud := unwrapped.AudioMessage; aud != nil {
		mimeType := sanitizeMimeType(aud.GetMimetype(), "audio/ogg")
		filename := defaultMediaFilename("audio", mimeType, "audio.ogg")
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, aud, models.MessageTypeAudio, mimeType, filename)
		return models.MessageTypeAudio, "", mediaURL, mimeType, filename, artifact
	}

	if doc := unwrapped.DocumentMessage; doc != nil {
		caption := doc.GetCaption()
		filename := strings.TrimSpace(doc.GetFileName())
		mimeFallback := mimeTypeFromFilename(filename)
		mimeType := sanitizeMimeType(doc.GetMimetype(), mimeFallback)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		if filename == "" {
			filename = defaultMediaFilename("document", mimeType, "document.bin")
		}
		mediaURL, artifact := cm.downloadAndPersistIncomingMedia(ctx, client, doc, models.MessageTypeDocument, mimeType, filename)
		return models.MessageTypeDocument, caption, mediaURL, mimeType, filename, artifact
	}

	// Default
	return models.MessageTypeText, "[Unsupported message type]", "", "", "", nil
}

func (cm *ConnectionManager) extractTextualIncomingMessage(ctx context.Context, client *waClient.Client, msg *waE2E.Message) (models.MessageType, string, bool) {
	if msg == nil {
		return models.MessageTypeText, "", false
	}

	if txt := strings.TrimSpace(msg.GetConversation()); txt != "" {
		return models.MessageTypeText, txt, true
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		txt := strings.TrimSpace(ext.GetText())
		txt = cm.normalizeMentionTokens(ctx, client, txt, ext.GetContextInfo())
		if txt != "" {
			return models.MessageTypeText, txt, true
		}
	}

	if protocol := msg.GetProtocolMessage(); protocol != nil &&
		protocol.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT &&
		protocol.GetEditedMessage() != nil {
		return cm.extractTextualIncomingMessage(ctx, client, protocol.GetEditedMessage())
	}

	if templateReply := msg.GetTemplateButtonReplyMessage(); templateReply != nil {
		content := firstNonEmpty(
			strings.TrimSpace(templateReply.GetSelectedDisplayText()),
			strings.TrimSpace(templateReply.GetSelectedID()),
			"[Button reply]",
		)
		return models.MessageTypeInteractive, content, true
	}
	if buttonsReply := msg.GetButtonsResponseMessage(); buttonsReply != nil {
		content := firstNonEmpty(
			strings.TrimSpace(buttonsReply.GetSelectedDisplayText()),
			strings.TrimSpace(buttonsReply.GetSelectedButtonID()),
			"[Button reply]",
		)
		return models.MessageTypeInteractive, content, true
	}
	if listReply := msg.GetListResponseMessage(); listReply != nil {
		selectedRowID := ""
		if single := listReply.GetSingleSelectReply(); single != nil {
			selectedRowID = strings.TrimSpace(single.GetSelectedRowID())
		}
		content := firstNonEmpty(
			strings.TrimSpace(listReply.GetTitle()),
			strings.TrimSpace(listReply.GetDescription()),
			selectedRowID,
			"[List reply]",
		)
		return models.MessageTypeInteractive, content, true
	}
	if interactiveReply := msg.GetInteractiveResponseMessage(); interactiveReply != nil {
		content := ""
		if body := interactiveReply.GetBody(); body != nil {
			content = strings.TrimSpace(body.GetText())
		}
		if content == "" {
			if nativeFlow := interactiveReply.GetNativeFlowResponseMessage(); nativeFlow != nil {
				content = firstNonEmpty(
					strings.TrimSpace(nativeFlow.GetName()),
					strings.TrimSpace(nativeFlow.GetParamsJSON()),
				)
			}
		}
		content = firstNonEmpty(content, "[Interactive response]")
		return models.MessageTypeInteractive, content, true
	}

	if location := msg.GetLocationMessage(); location != nil {
		return models.MessageTypeLocation, marshalLocationPayload(
			location.GetDegreesLatitude(),
			location.GetDegreesLongitude(),
			firstNonEmpty(strings.TrimSpace(location.GetName()), strings.TrimSpace(location.GetComment())),
			strings.TrimSpace(location.GetAddress()),
		), true
	}
	if liveLocation := msg.GetLiveLocationMessage(); liveLocation != nil {
		return models.MessageTypeLocation, marshalLocationPayload(
			liveLocation.GetDegreesLatitude(),
			liveLocation.GetDegreesLongitude(),
			strings.TrimSpace(liveLocation.GetCaption()),
			"",
		), true
	}

	if contact := msg.GetContactMessage(); contact != nil {
		payload := []contactPayload{contactPayloadFromMessage(contact)}
		return models.MessageType("contacts"), marshalContactsPayload(payload), true
	}
	if contacts := msg.GetContactsArrayMessage(); contacts != nil {
		contactEntries := make([]contactPayload, 0, len(contacts.GetContacts()))
		for _, contact := range contacts.GetContacts() {
			if contact == nil {
				continue
			}
			contactEntries = append(contactEntries, contactPayloadFromMessage(contact))
		}
		if len(contactEntries) == 0 {
			fallbackName := strings.TrimSpace(contacts.GetDisplayName())
			if fallbackName != "" {
				contactEntries = append(contactEntries, contactPayload{Name: fallbackName})
			}
		}
		return models.MessageType("contacts"), marshalContactsPayload(contactEntries), true
	}

	if poll := firstPollCreationMessage(msg); poll != nil {
		question := strings.TrimSpace(poll.GetName())
		if question == "" {
			question = "[Poll]"
		} else {
			question = "[Poll] " + question
		}
		return models.MessageTypeText, question, true
	}

	return models.MessageTypeText, "", false
}

func (cm *ConnectionManager) normalizeMentionTokens(
	ctx context.Context,
	client *waClient.Client,
	text string,
	contextInfo *waE2E.ContextInfo,
) string {
	if strings.TrimSpace(text) == "" || contextInfo == nil || len(contextInfo.GetMentionedJID()) == 0 {
		return text
	}

	matches := mentionTokenPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	mentionPhones := make([]string, 0, len(contextInfo.GetMentionedJID()))
	for _, mentionedJID := range contextInfo.GetMentionedJID() {
		if phone := cm.resolveMentionPhone(ctx, client, mentionedJID); phone != "" {
			mentionPhones = append(mentionPhones, phone)
		}
	}
	if len(mentionPhones) == 0 {
		return text
	}

	mentionIndex := 0
	return mentionTokenPattern.ReplaceAllStringFunc(text, func(original string) string {
		if mentionIndex >= len(mentionPhones) {
			return original
		}
		phone := strings.TrimPrefix(strings.TrimSpace(mentionPhones[mentionIndex]), "+")
		mentionIndex++
		if phone == "" {
			return original
		}
		return "@" + phone
	})
}

func (cm *ConnectionManager) resolveMentionPhone(ctx context.Context, client *waClient.Client, mentionedJID string) string {
	mentionedJID = strings.TrimSpace(mentionedJID)
	if mentionedJID == "" {
		return ""
	}

	jid, err := types.ParseJID(mentionedJID)
	if err != nil {
		if at := strings.Index(mentionedJID, "@"); at > 0 {
			return strings.TrimSpace(mentionedJID[:at])
		}
		return strings.TrimSpace(mentionedJID)
	}

	jid = jid.ToNonAD()
	if jid.Server == types.DefaultUserServer && jid.User != "" {
		return jid.User
	}
	if jid.Server == types.HiddenUserServer && jid.User != "" {
		if client != nil && client.Store != nil && client.Store.LIDs != nil {
			pn, err := client.Store.LIDs.GetPNForLID(ctx, jid)
			if err == nil && pn.Server == types.DefaultUserServer && pn.User != "" {
				return pn.User
			}
		}
		if pn := cm.lookupPNForLID(ctx, jid.User); pn != "" {
			return pn
		}
	}

	return jid.User
}

type contactPayload struct {
	Name   string   `json:"name"`
	Phones []string `json:"phones,omitempty"`
}

func contactPayloadFromMessage(contact *waE2E.ContactMessage) contactPayload {
	if contact == nil {
		return contactPayload{}
	}
	displayName := strings.TrimSpace(contact.GetDisplayName())
	phones := extractPhoneNumbersFromVCard(contact.GetVcard())
	if displayName == "" {
		displayName = firstNonEmpty(phones...)
	}
	if displayName == "" {
		displayName = "Contact"
	}
	return contactPayload{
		Name:   displayName,
		Phones: phones,
	}
}

func marshalContactsPayload(contacts []contactPayload) string {
	if len(contacts) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(contacts)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func marshalLocationPayload(latitude, longitude float64, name, address string) string {
	payload := map[string]any{
		"latitude":  latitude,
		"longitude": longitude,
	}
	if name != "" {
		payload["name"] = name
	}
	if address != "" {
		payload["address"] = address
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func extractPhoneNumbersFromVCard(vcard string) []string {
	if strings.TrimSpace(vcard) == "" {
		return nil
	}
	phones := make([]string, 0, 2)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(vcard, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if !strings.HasPrefix(upper, "TEL") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		number := strings.TrimSpace(parts[1])
		number = strings.TrimPrefix(strings.ToLower(number), "tel:")
		number = strings.TrimSpace(number)
		if idx := strings.Index(strings.ToLower(parts[0]), "waid="); idx >= 0 {
			waidValue := parts[0][idx+len("waid="):]
			if separator := strings.IndexAny(waidValue, ";,"); separator >= 0 {
				waidValue = waidValue[:separator]
			}
			waidValue = strings.TrimSpace(waidValue)
			if waidValue != "" {
				number = waidValue
			}
		}
		if number == "" {
			continue
		}
		if _, ok := seen[number]; ok {
			continue
		}
		seen[number] = struct{}{}
		phones = append(phones, number)
	}
	return phones
}

func firstPollCreationMessage(msg *waE2E.Message) *waE2E.PollCreationMessage {
	if msg == nil {
		return nil
	}
	if poll := msg.GetPollCreationMessage(); poll != nil {
		return poll
	}
	if poll := msg.GetPollCreationMessageV2(); poll != nil {
		return poll
	}
	if poll := msg.GetPollCreationMessageV3(); poll != nil {
		return poll
	}
	if poll := msg.GetPollCreationMessageV5(); poll != nil {
		return poll
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func incomingMessageKinds(msg *waE2E.Message) []string {
	if msg == nil {
		return nil
	}

	value := reflect.ValueOf(msg)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	valueType := value.Type()
	kinds := make([]string, 0, 8)
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldValue := value.Field(i)
		switch fieldValue.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice:
			if !fieldValue.IsNil() {
				kinds = append(kinds, field.Name)
			}
		}
	}

	sort.Strings(kinds)
	return kinds
}

func unwrapIncomingMessage(msg *waE2E.Message) *waE2E.Message {
	current := msg
	for i := 0; i < maxMessageUnwrapDepth && current != nil; i++ {
		next := nextWrappedMessage(current)
		if next == nil {
			return current
		}
		current = next
	}
	return current
}

func nextWrappedMessage(msg *waE2E.Message) *waE2E.Message {
	switch {
	case msg.DeviceSentMessage != nil && msg.DeviceSentMessage.Message != nil:
		return msg.DeviceSentMessage.Message
	case msg.EphemeralMessage != nil && msg.EphemeralMessage.Message != nil:
		return msg.EphemeralMessage.Message
	case msg.ViewOnceMessage != nil && msg.ViewOnceMessage.Message != nil:
		return msg.ViewOnceMessage.Message
	case msg.ViewOnceMessageV2 != nil && msg.ViewOnceMessageV2.Message != nil:
		return msg.ViewOnceMessageV2.Message
	case msg.ViewOnceMessageV2Extension != nil && msg.ViewOnceMessageV2Extension.Message != nil:
		return msg.ViewOnceMessageV2Extension.Message
	case msg.DocumentWithCaptionMessage != nil && msg.DocumentWithCaptionMessage.Message != nil:
		return msg.DocumentWithCaptionMessage.Message
	case msg.EditedMessage != nil && msg.EditedMessage.Message != nil:
		return msg.EditedMessage.Message
	case msg.AssociatedChildMessage != nil && msg.AssociatedChildMessage.Message != nil:
		return msg.AssociatedChildMessage.Message
	case msg.GroupMentionedMessage != nil && msg.GroupMentionedMessage.Message != nil:
		return msg.GroupMentionedMessage.Message
	case msg.CommentMessage != nil && msg.CommentMessage.Message != nil:
		return msg.CommentMessage.Message
	case msg.ProtocolMessage != nil &&
		msg.ProtocolMessage.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT &&
		msg.ProtocolMessage.GetEditedMessage() != nil:
		return msg.ProtocolMessage.GetEditedMessage()
	case msg.PollCreationMessageV4 != nil && msg.PollCreationMessageV4.Message != nil:
		return msg.PollCreationMessageV4.Message
	case msg.PollCreationMessageV6 != nil && msg.PollCreationMessageV6.Message != nil:
		return msg.PollCreationMessageV6.Message
	case msg.SpoilerMessage != nil && msg.SpoilerMessage.Message != nil:
		return msg.SpoilerMessage.Message
	default:
		return nil
	}
}

func isIncomingRevokeMessage(msg *waE2E.Message) bool {
	_, isRevoke := incomingRevokeTargetID(msg)
	return isRevoke
}

func incomingRevokeTargetID(msg *waE2E.Message) (string, bool) {
	current := msg
	for i := 0; i < maxMessageUnwrapDepth && current != nil; i++ {
		if protocol := current.GetProtocolMessage(); protocol != nil && protocol.GetType() == waE2E.ProtocolMessage_REVOKE {
			key := protocol.GetKey()
			if key == nil {
				return "", true
			}
			return key.GetID(), true
		}
		current = nextWrappedMessage(current)
	}
	return "", false
}

func (cm *ConnectionManager) downloadAndPersistIncomingMedia(
	ctx context.Context,
	downloader inboundMediaDownloader,
	media waClient.DownloadableMessage,
	msgType models.MessageType,
	mimeType string,
	fallbackFilename string,
) (string, *inboundMediaRetryArtifact) {
	buildArtifact := func(lastErr string) *inboundMediaRetryArtifact {
		return cm.buildInboundMediaRetryArtifact(msgType, media, mimeType, fallbackFilename, lastErr)
	}

	if downloader == nil || (reflect.ValueOf(downloader).Kind() == reflect.Ptr && reflect.ValueOf(downloader).IsNil()) {
		cm.logger.Warn("Cannot download inbound media: client is nil", "message_type", msgType)
		return "", buildArtifact("client is nil")
	}
	maxAttempts, baseBackoff, maxBackoff := cm.inboundMediaDownloadRetrySettings()

	var (
		data            []byte
		err             error
		lastFailureText string
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		data, err = downloader.Download(ctx, media)
		if err == nil && len(data) > 0 {
			break
		}

		if err != nil {
			lastFailureText = err.Error()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				cm.logger.Warn(
					"Inbound media download aborted by context",
					"message_type", msgType,
					"mime_type", mimeType,
					"attempt", attempt,
					"max_attempts", maxAttempts,
					"error", err,
				)
				return "", nil
			}
			if attempt >= maxAttempts || !shouldRetryInboundMediaDownload(err) {
				cm.logger.Warn(
					"Failed to download inbound media",
					"message_type", msgType,
					"mime_type", mimeType,
					"attempt", attempt,
					"max_attempts", maxAttempts,
					"error", err,
				)
				return "", buildArtifact(lastFailureText)
			}
			backoff := inboundMediaDownloadBackoff(attempt, baseBackoff, maxBackoff)
			cm.logger.Warn(
				"Retrying inbound media download after failure",
				"message_type", msgType,
				"mime_type", mimeType,
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"backoff", backoff.String(),
				"error", err,
			)
			if waitErr := sleepWithInboundMediaContext(ctx, backoff); waitErr != nil {
				cm.logger.Warn(
					"Inbound media retry aborted",
					"message_type", msgType,
					"mime_type", mimeType,
					"error", waitErr,
				)
				lastFailureText = waitErr.Error()
				if errors.Is(waitErr, context.Canceled) || errors.Is(waitErr, context.DeadlineExceeded) {
					return "", nil
				}
				return "", buildArtifact(lastFailureText)
			}
			continue
		}

		if attempt >= maxAttempts {
			cm.logger.Warn(
				"Inbound media download returned empty data",
				"message_type", msgType,
				"mime_type", mimeType,
				"attempt", attempt,
				"max_attempts", maxAttempts,
			)
			lastFailureText = "inbound media download returned empty data"
			return "", buildArtifact(lastFailureText)
		}

		backoff := inboundMediaDownloadBackoff(attempt, baseBackoff, maxBackoff)
		cm.logger.Warn(
			"Retrying inbound media download after empty payload",
			"message_type", msgType,
			"mime_type", mimeType,
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"backoff", backoff.String(),
		)
		if waitErr := sleepWithInboundMediaContext(ctx, backoff); waitErr != nil {
			cm.logger.Warn(
				"Inbound media retry aborted",
				"message_type", msgType,
				"mime_type", mimeType,
				"error", waitErr,
			)
			lastFailureText = waitErr.Error()
			if errors.Is(waitErr, context.Canceled) || errors.Is(waitErr, context.DeadlineExceeded) {
				return "", nil
			}
			return "", buildArtifact(lastFailureText)
		}
	}

	relPath, err := cm.persistInboundMedia(data, msgType, mimeType, fallbackFilename)
	if err != nil {
		cm.logger.Warn("Failed to persist inbound media", "message_type", msgType, "mime_type", mimeType, "error", err)
		lastFailureText = err.Error()
		return "", buildArtifact(lastFailureText)
	}
	return relPath, nil
}

func (cm *ConnectionManager) buildInboundMediaRetryArtifact(
	msgType models.MessageType,
	media waClient.DownloadableMessage,
	mimeType string,
	fallbackFilename string,
	lastErr string,
) *inboundMediaRetryArtifact {
	if media == nil {
		return nil
	}

	pb, ok := media.(proto.Message)
	if !ok {
		cm.logger.Warn("Cannot build inbound media retry artifact: media payload is not proto message", "message_type", msgType)
		return nil
	}

	rawPayload, err := proto.Marshal(pb)
	if err != nil {
		cm.logger.Warn("Cannot build inbound media retry artifact: failed to marshal payload", "message_type", msgType, "error", err)
		return nil
	}
	if len(rawPayload) == 0 {
		cm.logger.Warn("Cannot build inbound media retry artifact: marshaled payload is empty", "message_type", msgType)
		return nil
	}

	return &inboundMediaRetryArtifact{
		MediaKind:          inboundMediaKindFromMessageType(msgType, mimeType),
		MediaPayloadBase64: base64.StdEncoding.EncodeToString(rawPayload),
		MimeType:           mimeType,
		FallbackFilename:   fallbackFilename,
		LastError:          strings.TrimSpace(lastErr),
	}
}

func inboundMediaKindFromMessageType(msgType models.MessageType, mimeType string) string {
	switch msgType {
	case models.MessageTypeImage:
		return "image"
	case models.MessageTypeSticker:
		return "sticker"
	case models.MessageTypeVideo:
		return "video"
	case models.MessageTypeAudio:
		return "audio"
	case models.MessageTypeDocument:
		return "document"
	}

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "document"
	}
}

func (cm *ConnectionManager) inboundMediaDownloadRetrySettings() (int, time.Duration, time.Duration) {
	maxAttempts := defaultInboundMediaDownloadMaxAttempts
	baseBackoff := defaultInboundMediaDownloadBaseBackoff
	maxBackoff := defaultInboundMediaDownloadMaxBackoff

	if cm != nil && cm.cfg != nil {
		if cm.cfg.InboundMediaRetryCount > 0 {
			maxAttempts = cm.cfg.InboundMediaRetryCount
		}
		if cm.cfg.InboundMediaRetryDelayMs > 0 {
			baseBackoff = time.Duration(cm.cfg.InboundMediaRetryDelayMs) * time.Millisecond
		}
		if cm.cfg.InboundMediaRetryMaxDelayMs > 0 {
			maxBackoff = time.Duration(cm.cfg.InboundMediaRetryMaxDelayMs) * time.Millisecond
		}
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if baseBackoff < 0 {
		baseBackoff = 0
	}
	if maxBackoff < baseBackoff {
		maxBackoff = baseBackoff
	}
	return maxAttempts, baseBackoff, maxBackoff
}

func shouldRetryInboundMediaDownload(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func inboundMediaDownloadBackoff(failedAttempt int, baseBackoff, maxBackoff time.Duration) time.Duration {
	if failedAttempt < 1 {
		failedAttempt = 1
	}
	if baseBackoff < 0 {
		baseBackoff = 0
	}
	if maxBackoff < baseBackoff {
		maxBackoff = baseBackoff
	}
	backoff := baseBackoff * (1 << (failedAttempt - 1))
	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

func sleepWithInboundMediaContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (cm *ConnectionManager) persistInboundMedia(data []byte, msgType models.MessageType, mimeType, fallbackFilename string) (string, error) {
	basePath := cm.mediaStoragePath
	if basePath == "" {
		basePath = "./uploads"
	}

	subdir := inboundMediaSubdir(msgType, mimeType)
	targetDir := filepath.Join(basePath, subdir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("create media directory: %w", err)
	}

	ext := mediaFileExtension(mimeType, fallbackFilename)
	filename := uuid.NewString() + ext
	absPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return "", fmt.Errorf("write media file: %w", err)
	}

	return filepath.ToSlash(filepath.Join(subdir, filename)), nil
}

func inboundMediaSubdir(msgType models.MessageType, mimeType string) string {
	switch msgType {
	case models.MessageTypeImage:
		return "images"
	case models.MessageTypeSticker:
		return "stickers"
	case models.MessageTypeVideo:
		return "videos"
	case models.MessageTypeAudio:
		return "audio"
	case models.MessageTypeDocument:
		return "documents"
	}

	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "images"
	case strings.HasPrefix(mimeType, "video/"):
		return "videos"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "documents"
	}
}

func sanitizeMimeType(mimeType, fallback string) string {
	trimmed := strings.TrimSpace(mimeType)
	if trimmed == "" {
		trimmed = strings.TrimSpace(fallback)
	}
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(trimmed, ";")[0]))
}

func mimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == "" {
		return ""
	}
	return sanitizeMimeType(mime.TypeByExtension(ext), "")
}

func defaultMediaFilename(prefix, mimeType, hardDefault string) string {
	ext := mediaFileExtension(mimeType, hardDefault)
	base := strings.TrimSuffix(hardDefault, filepath.Ext(hardDefault))
	if base == "" {
		base = prefix
	}
	return base + ext
}

func mediaFileExtension(mimeType, fallbackFilename string) string {
	switch sanitizeMimeType(mimeType, "") {
	case "image/jpg", "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/3gpp":
		return ".3gp"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/aac":
		return ".aac"
	case "audio/amr":
		return ".amr"
	case "application/pdf":
		return ".pdf"
	}

	cleanMime := sanitizeMimeType(mimeType, "")
	if cleanMime != "" {
		exts, err := mime.ExtensionsByType(cleanMime)
		if err == nil && len(exts) > 0 {
			return strings.ToLower(exts[0])
		}
	}

	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fallbackFilename))); ext != "" {
		return ext
	}

	return ".bin"
}
