package templateutil

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
)

// TemplateParams carries the resolved parameter maps used to render a
// template for sending.
type TemplateParams struct {
	// BodyParams maps parameter name ({{name}} or {{1}}) to value. Used for
	// body text and URL-button substitution.
	BodyParams map[string]string

	// HeaderParams overrides BodyParams for TEXT-header variables. When nil
	// or empty, header variables resolve from BodyParams.
	HeaderParams map[string]string

	// ButtonURLParams maps button index (as decimal string) to the value
	// substituted for every placeholder in that URL button's URL. Buttons
	// without an entry fall back to BodyParams substitution.
	ButtonURLParams map[string]string
}

// TemplateMedia describes the header media for IMAGE/VIDEO/DOCUMENT headers.
type TemplateMedia struct {
	ID       string
	Data     []byte
	MimeType string
	Filename string

	// Load lazily fetches the media bytes when ID and Data are empty (e.g.
	// reading the file back from local storage on a retry, or loading a
	// campaign's header media file). Returning nil/empty skips the upload.
	Load func() []byte
}

// SendRequest is the input to SendRenderedTemplate.
type SendRequest struct {
	Provider     whatsapp.Provider
	Account      *whatsapp.Account
	Recipient    whatsapp.Recipient
	Template     *models.Template
	Params       TemplateParams
	Media        TemplateMedia
	ReplyToMsgID string
}

// SendRenderedTemplate renders a local template blueprint (header / body /
// footer / buttons) and sends it through the provider as a plain text,
// media-with-caption, or interactive-buttons message. Templates are not
// submitted to a remote template API — all content is resolved locally.
//
// Rendering rules:
//   - TEXT headers render bold, above the body; their variables resolve from
//     HeaderParams first, then BodyParams.
//   - QUICK_REPLY buttons become native interactive buttons when the provider
//     supports them (otherwise they are dropped and the text is sent plain).
//   - URL buttons are appended as text lines: a ButtonURLParams entry for the
//     button's index replaces every placeholder with that value; otherwise
//     placeholders resolve from BodyParams.
//   - IMAGE/VIDEO/DOCUMENT headers send as media with the rendered text as
//     caption, uploading Data or the Load() bytes when ID is empty. With no
//     media available, the message falls through to a plain text send.
func SendRenderedTemplate(ctx context.Context, req SendRequest) (string, error) {
	tpl := req.Template
	if tpl == nil {
		return "", fmt.Errorf("template is required for template messages")
	}
	provider, waAccount, rcpt := req.Provider, req.Account, req.Recipient

	body := ReplaceWithStringParams(tpl.BodyContent, req.Params.BodyParams)

	var parts []string
	if tpl.HeaderType == "TEXT" && tpl.HeaderContent != "" {
		headerParams := mergeParams(req.Params.BodyParams, req.Params.HeaderParams)
		if header := ReplaceWithStringParams(tpl.HeaderContent, headerParams); header != "" {
			parts = append(parts, "*"+header+"*")
		}
	}
	if body != "" {
		parts = append(parts, body)
	}
	if tpl.FooterContent != "" {
		parts = append(parts, tpl.FooterContent)
	}

	var quickReplies []whatsapp.Button
	for i, raw := range tpl.Buttons {
		btn, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		btnType, _ := btn["type"].(string)
		label, _ := btn["text"].(string)
		switch strings.ToUpper(btnType) {
		case "QUICK_REPLY":
			quickReplies = append(quickReplies, whatsapp.Button{ID: fmt.Sprintf("btn_%d", i), Title: label})
		case "URL":
			urlStr, _ := btn["url"].(string)
			if val, ok := req.Params.ButtonURLParams[fmt.Sprintf("%d", i)]; ok {
				urlStr = replaceAllParams(urlStr, val)
			} else {
				urlStr = ReplaceWithStringParams(urlStr, req.Params.BodyParams)
			}
			if urlStr != "" {
				if label != "" {
					parts = append(parts, label+": "+urlStr)
				} else {
					parts = append(parts, urlStr)
				}
			}
		}
	}

	text := strings.Join(parts, "\n\n")

	if tpl.HeaderType == "IMAGE" || tpl.HeaderType == "VIDEO" || tpl.HeaderType == "DOCUMENT" {
		mediaID, data := req.Media.ID, req.Media.Data
		if mediaID == "" && len(data) == 0 && req.Media.Load != nil {
			if b := req.Media.Load(); len(b) > 0 {
				data = b
			}
		}
		if mediaID == "" && len(data) > 0 {
			uploaded, err := provider.UploadMedia(ctx, waAccount, data, req.Media.MimeType, req.Media.Filename)
			if err != nil {
				return "", fmt.Errorf("failed to upload template header media: %w", err)
			}
			mediaID = uploaded
		}
		if mediaID != "" {
			switch tpl.HeaderType {
			case "IMAGE":
				return provider.SendImageMessage(ctx, waAccount, rcpt, mediaID, text, req.ReplyToMsgID)
			case "VIDEO":
				return provider.SendVideoMessage(ctx, waAccount, rcpt, mediaID, text, req.ReplyToMsgID)
			default: // DOCUMENT
				return provider.SendDocumentMessage(ctx, waAccount, rcpt, mediaID, req.Media.Filename, text, req.ReplyToMsgID)
			}
		}
		// No media supplied — fall through to a plain text send.
	}

	if len(quickReplies) > 0 {
		wamid, err := provider.SendInteractiveButtons(ctx, waAccount, rcpt, text, quickReplies)
		if err == nil || !errors.Is(err, whatsapp.ErrNotSupported) {
			return wamid, err
		}
	}

	return provider.SendTextMessage(ctx, waAccount, rcpt, text, req.ReplyToMsgID)
}

// mergeParams returns base overlaid with the non-empty entries of override.
// When override is empty it returns base unchanged (which may be nil).
func mergeParams(base, override map[string]string) map[string]string {
	if len(override) == 0 {
		return base
	}
	merged := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

// replaceAllParams substitutes every {{...}} placeholder in content with val.
// Unlike regexp ReplaceAllString it performs no $-expansion on val.
func replaceAllParams(content, val string) string {
	if content == "" {
		return content
	}
	return ParameterPattern.ReplaceAllStringFunc(content, func(string) string { return val })
}

// ResolveNamedParams resolves a map[string]any parameter source (e.g.
// models.JSONB) into a map keyed by the body content's parameter names, in
// the same way ResolveParams orders values. Parameters absent from the source
// map resolve to "".
func ResolveNamedParams(bodyContent string, params map[string]any) map[string]string {
	names := ExtParamNames(bodyContent)
	if len(names) == 0 || len(params) == 0 {
		return nil
	}
	values := ResolveParams(bodyContent, params)
	named := make(map[string]string, len(names))
	for i, val := range values {
		if i < len(names) {
			named[names[i]] = val
		}
	}
	return named
}
