package whatsmeow

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const directionOutgoing = "outgoing"

// resolveClientAndJID resolves a whatsmeow client and target JID from instanceID and phone number.
func (a *WhatsmeowAdapter) resolveClientAndJID(ctx context.Context, instanceID, to string) (*whatsmeow.Client, waTypes.JID, error) {
	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, waTypes.JID{}, err
	}
	jid, err := a.parseJID(to)
	if err != nil {
		return nil, waTypes.JID{}, fmt.Errorf("invalid JID: %w", err)
	}
	return client, jid, nil
}

// mediaUpload holds the resolved client, JID, downloaded bytes, MIME type, and
// WhatsApp upload response needed to construct and send a media message.
type mediaUpload struct {
	client     *whatsmeow.Client
	jid        waTypes.JID
	data       []byte
	mimeType   string
	uploadResp whatsmeow.UploadResponse
}

// prepareMediaSend resolves the client, downloads media from the URL, and
// uploads it to WhatsApp. Callers construct the protobuf message using the
// returned upload fields, then call u.client.SendMessage.
func (a *WhatsmeowAdapter) prepareMediaSend(ctx context.Context, instanceID, to, mediaURL string, mediaType whatsmeow.MediaType) (*mediaUpload, error) {
	client, jid, err := a.resolveClientAndJID(ctx, instanceID, to)
	if err != nil {
		return nil, err
	}

	data, mimeType, err := a.downloadMediaFromURL(mediaURL)
	if err != nil {
		return nil, err
	}

	uploadResp, err := a.uploadMediaToWhatsApp(ctx, client, data, mediaType)
	if err != nil {
		return nil, err
	}

	return &mediaUpload{
		client:     client,
		jid:        jid,
		data:       data,
		mimeType:   mimeType,
		uploadResp: uploadResp,
	}, nil
}

var textURLPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)

// SendText sends a text message.
func (a *WhatsmeowAdapter) SendText(ctx context.Context, instanceID string, to string, text string) (string, error) {
	client, jid, err := a.resolveClientAndJID(ctx, instanceID, to)
	if err != nil {
		return "", err
	}
	a.simulateTypingIndicator(ctx, client, jid, text)

	resp, err := client.SendMessage(ctx, jid, buildTextMessage(text))
	if err != nil {
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

	return resp.ID, nil
}

func buildTextMessage(text string) *waE2E.Message {
	if shouldUseExtendedTextMessage(text) {
		return &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String(text),
			},
		}
	}

	return &waE2E.Message{
		Conversation: proto.String(text),
	}
}

func shouldUseExtendedTextMessage(text string) bool {
	return textURLPattern.MatchString(strings.TrimSpace(text))
}

// SendTextReply sends a text message as a quoted reply to a specific message.
func (a *WhatsmeowAdapter) SendTextReply(ctx context.Context, instanceID string, to string, text string, replyToMsgID string) (string, error) {
	client, jid, err := a.resolveClientAndJID(ctx, instanceID, to)
	if err != nil {
		return "", err
	}
	a.simulateTypingIndicator(ctx, client, jid, text)

	myJID := waTypes.JID{}
	if client.Store != nil && client.Store.ID != nil {
		myJID = *client.Store.ID
	}

	participant, quotedText := a.resolveReplyContext(jid, replyToMsgID, myJID, instanceID)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(replyToMsgID),
				Participant:   proto.String(participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String(quotedText)},
			},
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send reply message: %w", err)
	}

	return resp.ID, nil
}

// resolveReplyContext queries the database to resolve the participant JID and the quoted message content.
func (a *WhatsmeowAdapter) resolveReplyContext(jid waTypes.JID, replyToMsgID string, myJID waTypes.JID, instanceID string) (string, string) {
	participant := jid.String()
	quotedText := ""
	if a.db != nil && replyToMsgID != "" {
		type msgRow struct {
			Direction string       `gorm:"column:direction"`
			Content   string       `gorm:"column:content"`
			Metadata  models.JSONB `gorm:"column:metadata"`
		}
		var row msgRow

		// Clone database session to avoid statement/query pollution.
		// Scope by instance_id to prevent cross-tenant data access.
		db := a.db.Session(&gorm.Session{})
		query := db.Table("messages").
			Select("direction, content, metadata").
			Where("whats_app_message_id = ? AND instance_id = ?", replyToMsgID, instanceID)

		if err := query.Take(&row).Error; err == nil {
			quotedText = row.Content
			if quotedText == "" {
				quotedText = "Media" // Fallback for media messages without caption
			}
			if row.Direction == directionOutgoing {
				if myJID.User != "" {
					participant = myJID.ToNonAD().String()
				}
			} else {
				// Incoming message
				// For group chats, we need the JID of the actual sender from metadata
				if strings.HasSuffix(jid.String(), "@g.us") {
					if senderPhone, ok := row.Metadata["sender_phone"].(string); ok && senderPhone != "" {
						senderPhone = strings.TrimSpace(senderPhone)
						if !strings.Contains(senderPhone, "@") {
							participant = senderPhone + "@s.whatsapp.net"
						} else {
							participant = senderPhone
						}
					}
				} else {
					// Direct chat: sender is the other person (customer)
					participant = jid.ToNonAD().String()
				}
			}
		} else {
			// Log error if reply context could not be resolved from DB
			a.logger.Warn("Failed to resolve reply context from DB", "replyToMsgID", replyToMsgID, "error", err)
		}

		if participant == "" {
			a.logger.Warn("Participant is empty after resolving reply context, falling back to customer jid", "replyToMsgID", replyToMsgID, "jid", jid.String())
			participant = jid.ToNonAD().String()
		}
		
		if quotedText == "" {
			quotedText = "Message" // Better fallback
		}
	}

	if quotedText == "" {
		quotedText = "Message"
	}

	return participant, quotedText
}


// SendImage sends an image message.
func (a *WhatsmeowAdapter) SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error) {
	u, err := a.prepareMediaSend(ctx, instanceID, to, imageURL, whatsmeow.MediaImage)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(u.mimeType),
			URL:           proto.String(u.uploadResp.URL),
			DirectPath:    proto.String(u.uploadResp.DirectPath),
			MediaKey:      u.uploadResp.MediaKey,
			FileEncSHA256: u.uploadResp.FileEncSHA256,
			FileSHA256:    u.uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(u.data))),
		},
	}

	resp, err := u.client.SendMessage(ctx, u.jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send image message: %w", err)
	}

	return resp.ID, nil
}

// SendDocument sends a document message.
func (a *WhatsmeowAdapter) SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string, caption string) (string, error) {
	u, err := a.prepareMediaSend(ctx, instanceID, to, docURL, whatsmeow.MediaDocument)
	if err != nil {
		return "", err
	}

	mimeType := u.mimeType
	if filename != "" && (mimeType == "application/octet-stream" || mimeType == "") {
		if m := mime.TypeByExtension(filepath.Ext(filename)); m != "" {
			mimeType = m
		}
	}

	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(u.uploadResp.URL),
			DirectPath:    proto.String(u.uploadResp.DirectPath),
			MediaKey:      u.uploadResp.MediaKey,
			FileEncSHA256: u.uploadResp.FileEncSHA256,
			FileSHA256:    u.uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(u.data))),
			FileName:      proto.String(filename),
			Title:         proto.String(filename),
		},
	}

	resp, err := u.client.SendMessage(ctx, u.jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send document message: %w", err)
	}

	return resp.ID, nil
}

// SendVideo sends a video message.
func (a *WhatsmeowAdapter) SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error) {
	u, err := a.prepareMediaSend(ctx, instanceID, to, videoURL, whatsmeow.MediaVideo)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(u.mimeType),
			URL:           proto.String(u.uploadResp.URL),
			DirectPath:    proto.String(u.uploadResp.DirectPath),
			MediaKey:      u.uploadResp.MediaKey,
			FileEncSHA256: u.uploadResp.FileEncSHA256,
			FileSHA256:    u.uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(u.data))),
		},
	}

	resp, err := u.client.SendMessage(ctx, u.jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send video message: %w", err)
	}

	return resp.ID, nil
}

// SendAudio sends an audio message.
func (a *WhatsmeowAdapter) SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error) {
	u, err := a.prepareMediaSend(ctx, instanceID, to, audioURL, whatsmeow.MediaAudio)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			Mimetype:      proto.String(u.mimeType),
			URL:           proto.String(u.uploadResp.URL),
			DirectPath:    proto.String(u.uploadResp.DirectPath),
			MediaKey:      u.uploadResp.MediaKey,
			FileEncSHA256: u.uploadResp.FileEncSHA256,
			FileSHA256:    u.uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(u.data))),
			PTT:           proto.Bool(true),
		},
	}

	resp, err := u.client.SendMessage(ctx, u.jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send audio message: %w", err)
	}

	return resp.ID, nil
}

// SendPoll sends a native WhatsApp poll message.
func (a *WhatsmeowAdapter) SendPoll(ctx context.Context, instanceID string, to string, question string, options []string, maxSelections int) (string, error) {
	client, jid, err := a.resolveClientAndJID(ctx, instanceID, to)
	if err != nil {
		return "", err
	}

	msg := client.BuildPollCreation(question, options, maxSelections)

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send poll message: %w", err)
	}

	return resp.ID, nil
}

func (a *WhatsmeowAdapter) simulateTypingIndicator(ctx context.Context, client *whatsmeow.Client, jid waTypes.JID, previewText string) {
	if a == nil || a.manager == nil || a.manager.typingIndicator == nil {
		return
	}
	a.manager.typingIndicator.simulate(ctx, client, jid, previewText)
}
