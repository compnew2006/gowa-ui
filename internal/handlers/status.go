package handlers

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// statusSendTimeout bounds a status post. Status media can be larger than text,
// so we allow more headroom than the default send path.
const statusSendTimeout = 60 * time.Second

// ============================================================================
// WhatsApp Status (Story) Posting
//
// Status posts are handled through a dedicated path that is intentionally kept
// separate from the unified OutgoingMessageRequest/SendOutgoingMessage flow.
// A status is addressed to the well-known `status@broadcast` JID — it is not a
// conversation tied to a Contact row, so it must not create a contact, persist
// a Message record, dispatch a message.sent webhook, or broadcast over the
// org websocket. This handler only forwards the post to GOWA and returns the
// resulting message id. The frontend keeps a local (session-only) log of what
// the user posted; inbound status media is still hidden from the sidebar (its
// download is unsupported by GOWA), see stores/contacts.ts isNonChatContact.
// ============================================================================

// SendStatusRequest is the JSON body for posting a text status.
type SendStatusRequest struct {
	Message         string `json:"message"`          // Required for type=text
	Type            string `json:"type"`             // "text" (default). image/video arrive via multipart.
	WhatsAppAccount string `json:"whatsapp_account"` // Optional: specific WhatsApp account
}

// statusResponse is returned on a successful status post.
type statusResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"` // always "sent"
}

// SendStatus posts a WhatsApp Status (story) from a connected account.
//
// Two content types are accepted:
//   - application/json  → text status (type=text). Body: {message, account_name?}.
//   - multipart/form-data → image or video status. Fields: type (image|video),
//     caption?, whatsapp_account?; file part named "file".
//
// Auth scope: ResourceChat / ActionWrite — same as normal messaging.
func (a *App) SendStatus(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceChat, models.ActionWrite)
	if err != nil {
		return nil
	}

	contentType := string(r.RequestCtx.Request.Header.ContentType())

	var (
		statusType   string
		text         string
		caption      string
		accountName  string
		fileData     []byte
		fileMime     string
		fileFilename string
	)

	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, err := r.RequestCtx.MultipartForm()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
		}
		if v := form.Value["type"]; len(v) > 0 {
			statusType = v[0]
		}
		if v := form.Value["caption"]; len(v) > 0 {
			caption = v[0]
		}
		if v := form.Value["whatsapp_account"]; len(v) > 0 {
			accountName = v[0]
		}
		if files := form.File["file"]; len(files) > 0 {
			fh := files[0]
			f, openErr := fh.Open()
			if openErr != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read file", nil, "")
			}
			defer func() { _ = f.Close() }()
			data, readErr := io.ReadAll(f)
			if readErr != nil {
				a.Log.Error("Failed to read status file data", "error", readErr)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file data", nil, "")
			}
			fileData = data
			fileMime = fh.Header.Get("Content-Type")
			if fileMime == "" {
				fileMime = "application/octet-stream"
			}
			fileFilename = fh.Filename
		}
	} else {
		// JSON body — text status only.
		var req SendStatusRequest
		if err := a.decodeRequest(r, &req); err != nil {
			return nil
		}
		statusType = req.Type
		if statusType == "" {
			statusType = "text"
		}
		text = req.Message
		accountName = req.WhatsAppAccount
	}
	if statusType == "" {
		statusType = "text"
	}

	// Resolve the sending account (default outgoing if not specified).
	account, err := a.resolveWhatsAppAccount(orgID, accountName)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	provider := a.resolveProvider(account)
	if provider == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "WhatsApp provider is not available", nil, "")
	}
	// PostStatus* lives on the concrete *gowa.Client, not the whatsapp.Provider
	// interface (a status is not a normal conversation). Assert the concrete type.
	gowaClient, ok := provider.(*gowa.Client)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Status posting is not supported by this provider", nil, "")
	}

	waAccount := a.toWhatsAppAccount(account)
	ctx, cancel := context.WithTimeout(context.Background(), statusSendTimeout)
	defer cancel()

	var (
		wamid   string
		sendErr error
	)
	switch statusType {
	case "text":
		if strings.TrimSpace(text) == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "message is required for a text status", nil, "")
		}
		wamid, sendErr = gowaClient.PostStatusText(ctx, waAccount, text)
	case "image":
		if len(fileData) == 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "file is required for an image status", nil, "")
		}
		wamid, sendErr = a.postStatusMedia(ctx, gowaClient, waAccount, fileData, fileMime, fileFilename, caption, "image")
	case "video":
		if len(fileData) == 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "file is required for a video status", nil, "")
		}
		wamid, sendErr = a.postStatusMedia(ctx, gowaClient, waAccount, fileData, fileMime, fileFilename, caption, "video")
	default:
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "unsupported status type: "+statusType, nil, "")
	}

	if sendErr != nil {
		if errors.Is(sendErr, whatsapp.ErrNotSupported) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Status posting is not supported", nil, "")
		}
		a.Log.Error("Failed to post WhatsApp status", "error", sendErr, "type", statusType, "account", account.Name)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Failed to post status", nil, "")
	}

	return r.SendEnvelope(statusResponse{MessageID: wamid, Status: "sent"})
}

// postStatusMedia uploads the bytes via the provider cache then posts the
// status. GOWA has no two-step upload endpoint: UploadMedia stages the bytes in
// an in-memory cache keyed by the returned mediaID, and the subsequent
// PostStatus{Image,Video} consumes that cache entry and sends it inline.
func (a *App) postStatusMedia(ctx context.Context, c *gowa.Client, account *whatsapp.Account, data []byte, mimeType, filename, caption, kind string) (string, error) {
	mediaID, err := c.UploadMedia(ctx, account, data, mimeType, filename)
	if err != nil {
		return "", err
	}
	if kind == "video" {
		return c.PostStatusVideo(ctx, account, mediaID, caption)
	}
	return c.PostStatusImage(ctx, account, mediaID, caption)
}
