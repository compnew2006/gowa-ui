package whatsmeow

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// SendText sends a text message.
func (a *WhatsmeowAdapter) SendText(ctx context.Context, instanceID string, to string, text string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	resp, err := client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

	return resp.ID, nil
}

// SendTextReply sends a text message as a quoted reply to a specific message.
func (a *WhatsmeowAdapter) SendTextReply(ctx context.Context, instanceID string, to string, text string, replyToMsgID string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(replyToMsgID),
				Participant:   proto.String(jid.String()),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			},
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send reply message: %w", err)
	}

	return resp.ID, nil
}

// SendImage sends an image message.
func (a *WhatsmeowAdapter) SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	data, mimeType, err := a.downloadMediaFromURL(imageURL)
	if err != nil {
		return "", err
	}

	uploadResp, err := a.uploadMediaToWhatsApp(ctx, client, data, whatsmeow.MediaImage)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send image message: %w", err)
	}

	return resp.ID, nil
}

// SendDocument sends a document message.
func (a *WhatsmeowAdapter) SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	data, mimeType, err := a.downloadMediaFromURL(docURL)
	if err != nil {
		return "", err
	}

	if filename != "" && (mimeType == "application/octet-stream" || mimeType == "") {
		ext := filepath.Ext(filename)
		if m := mime.TypeByExtension(ext); m != "" {
			mimeType = m
		}
	}

	uploadResp, err := a.uploadMediaToWhatsApp(ctx, client, data, whatsmeow.MediaDocument)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			Caption:       proto.String(filename),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			FileName:      proto.String(filename),
			Title:         proto.String(filename),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send document message: %w", err)
	}

	return resp.ID, nil
}

// SendVideo sends a video message.
func (a *WhatsmeowAdapter) SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	data, mimeType, err := a.downloadMediaFromURL(videoURL)
	if err != nil {
		return "", err
	}

	uploadResp, err := a.uploadMediaToWhatsApp(ctx, client, data, whatsmeow.MediaVideo)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send video message: %w", err)
	}

	return resp.ID, nil
}

// SendAudio sends an audio message.
func (a *WhatsmeowAdapter) SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error) {
	client, err := a.getClient(instanceID)
	if err != nil {
		return "", err
	}

	jid, err := a.parseJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	data, mimeType, err := a.downloadMediaFromURL(audioURL)
	if err != nil {
		return "", err
	}

	uploadResp, err := a.uploadMediaToWhatsApp(ctx, client, data, whatsmeow.MediaAudio)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
			PTT:           proto.Bool(true),
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send audio message: %w", err)
	}

	return resp.ID, nil
}
