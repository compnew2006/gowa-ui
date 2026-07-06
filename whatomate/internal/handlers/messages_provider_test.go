package handlers

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

type captureDocumentProvider struct {
	instanceID       string
	to               string
	text             string
	replyToMessageID string
	textReplyCalled  bool
	docURL           string
	filename         string
	caption          string
}

func (p *captureDocumentProvider) SendText(ctx context.Context, instanceID string, to string, text string) (string, error) {
	p.instanceID = instanceID
	p.to = to
	p.text = text
	return "wamid-text", nil
}

func (p *captureDocumentProvider) SendTextReply(ctx context.Context, instanceID string, to string, text string, replyToMessageID string) (string, error) {
	p.instanceID = instanceID
	p.to = to
	p.text = text
	p.replyToMessageID = replyToMessageID
	p.textReplyCalled = true
	return "wamid-reply", nil
}

func (p *captureDocumentProvider) SendImage(ctx context.Context, instanceID string, to string, imageURL string, caption string) (string, error) {
	return "", nil
}

func (p *captureDocumentProvider) SendDocument(ctx context.Context, instanceID string, to string, docURL string, filename string, caption string) (string, error) {
	p.instanceID = instanceID
	p.to = to
	p.docURL = docURL
	p.filename = filename
	p.caption = caption
	return "wamid-doc", nil
}

func (p *captureDocumentProvider) SendVideo(ctx context.Context, instanceID string, to string, videoURL string, caption string) (string, error) {
	return "", nil
}

func (p *captureDocumentProvider) SendAudio(ctx context.Context, instanceID string, to string, audioURL string) (string, error) {
	return "", nil
}

func (p *captureDocumentProvider) MarkRead(ctx context.Context, instanceID string, messageID string) error {
	return nil
}

func (p *captureDocumentProvider) SendReaction(ctx context.Context, instanceID string, messageID string, emoji string) error {
	return nil
}

func (p *captureDocumentProvider) RevokeMessage(ctx context.Context, instanceID string, messageID string) error {
	return nil
}

func (p *captureDocumentProvider) GetMediaURL(ctx context.Context, instanceID string, mediaID string) (string, error) {
	return "", nil
}

func (p *captureDocumentProvider) DownloadMedia(ctx context.Context, instanceID string, mediaURL string) ([]byte, error) {
	return nil, nil
}

func (p *captureDocumentProvider) UploadMedia(ctx context.Context, instanceID string, mediaType string, data []byte) (string, error) {
	return "", nil
}

func TestSendViaProviderDocumentPreservesFilenameAndCaption(t *testing.T) {
	provider := &captureDocumentProvider{}
	app := &App{MessageProvider: provider}

	req := OutgoingMessageRequest{
		Contact: &models.Contact{
			PhoneNumber: "120363419978360489@g.us",
		},
		Type:          models.MessageTypeDocument,
		MediaURL:      "orgs/test/documents/report.pdf",
		MediaFilename: "report.pdf",
		Caption:       "Monthly report",
	}

	wamid, err := app.sendViaProvider(context.Background(), req, nil, "instance-123")

	require.NoError(t, err)
	require.Equal(t, "wamid-doc", wamid)
	require.Equal(t, "instance-123", provider.instanceID)
	require.Equal(t, "120363419978360489@g.us", provider.to)
	require.Equal(t, "orgs/test/documents/report.pdf", provider.docURL)
	require.Equal(t, "report.pdf", provider.filename)
	require.Equal(t, "Monthly report", provider.caption)
}

func TestSendViaProviderTextReplyUsesReplyProvider(t *testing.T) {
	provider := &captureDocumentProvider{}
	app := &App{MessageProvider: provider}

	req := OutgoingMessageRequest{
		Contact: &models.Contact{
			PhoneNumber: "120363419978360489@g.us",
		},
		Type:    models.MessageTypeText,
		Content: "Quoted response",
		ReplyToMessage: &models.Message{
			WhatsAppMessageID: "wamid-original",
		},
	}

	wamid, err := app.sendViaProvider(context.Background(), req, nil, "instance-123")

	require.NoError(t, err)
	require.Equal(t, "wamid-reply", wamid)
	require.True(t, provider.textReplyCalled)
	require.Equal(t, "instance-123", provider.instanceID)
	require.Equal(t, "120363419978360489@g.us", provider.to)
	require.Equal(t, "Quoted response", provider.text)
	require.Equal(t, "wamid-original", provider.replyToMessageID)
}

func TestWhatsmeowAdapterImplementsReplyProvider(t *testing.T) {
	adapter := whatsmeow.NewWhatsmeowAdapter(nil, nil, logf.New(logf.Opts{}))

	require.Implements(t, (*provider.ReplyProvider)(nil), adapter)
}
