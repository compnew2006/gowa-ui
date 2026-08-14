package gowa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
)

// DefaultTimeout for GOWA REST API requests.
const DefaultTimeout = 30 * time.Second

// MediaSendTimeout bounds media send/download requests. These carry large
// bodies (documents up to 100MB) whose upload to WhatsApp takes minutes, so
// they need a far larger budget than regular API calls. Applied as a
// per-request context deadline; every other call keeps DefaultTimeout.
const MediaSendTimeout = 15 * time.Minute

// Client is a GOWA (Go WhatsApp Web Multi-Device) REST API client.
// It implements whatsapp.Provider.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string

	mu         sync.RWMutex
	mediaCache map[string]mediaCacheItem
	idCounter  uint64
}

// gowaTransport is the shared connection pool behind every GOWA client.
// Clients are built per instance/credentials — and ad hoc for instance
// management endpoints and probes — so leaving them on http.DefaultTransport
// (MaxIdleConnsPerHost=2) would churn connections under concurrent sends;
// one shared tuned transport keeps idle connections warm across all of them.
var gowaTransport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = 32
	return t
}()

// New creates a new GOWA client targeting the given REST API base URL.
func New(baseURL, username, password string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: gowaTransport,
			// No client-level Timeout: media sends/downloads set their own
			// long deadline (MediaSendTimeout) via the request context, and
			// doRequest applies DefaultTimeout to everything that doesn't
			// carry one, so hung regular calls still fail fast.
			// SECURITY (gap #7): never follow a redirect to a different host.
			// A webhook can carry an arbitrary media URL; even with auth bound
			// to the base origin, a cross-host redirect (initial same-host URL
			// 302→external) could exfiltrate data or probe internal services
			// (SSRF). Block it hard at the transport layer.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) == 0 {
					return nil
				}
				if req.URL.Host != via[0].URL.Host {
					return fmt.Errorf("redirect to different host blocked: %s -> %s", via[0].URL.Host, req.URL.Host)
				}
				return nil
			},
		},
		baseURL:    baseURL,
		username:   username,
		password:   password,
		mediaCache: make(map[string]mediaCacheItem),
	}
}

// Capabilities reports the GOWA feature set.
// GOWA supports free-form messaging, media, and read receipts.
func (c *Client) Capabilities() whatsapp.Capabilities {
	return whatsapp.Capabilities{
		MediaUpload: false, // GOWA sends inline, no two-step upload
		Interactive: false, // no native buttons in v8.10.0
	}
}

// deviceID extracts the GOWA device ID from the account credentials.
func deviceID(account *whatsapp.Account) string {
	return account.GowaDeviceID
}

// toJID converts a plain phone number to a WhatsApp JID.
// gowa-ui stores phone numbers as digits (e.g. "16505551234"); GOWA
// expects a full JID (e.g. "16505551234@s.whatsapp.net"). If the input
// already contains "@" it is assumed to be a JID and returned as-is.
//
// Group chats use a "@g.us" suffix instead of "@s.whatsapp.net". WhatsApp
// group IDs share the "120363" (or "120362") prefix, so we detect them and
// apply the correct suffix — otherwise GOWA rejects the JID with
// "is not on whatsapp" (the bug behind group send/revoke/typing failures).
func toJID(phone string) string {
	if phone == "" {
		return ""
	}
	for i := 0; i < len(phone); i++ {
		if phone[i] == '@' {
			return phone // already a JID
		}
	}
	// WhatsApp group/community IDs start with 120362/120363. These need the
	// @g.us suffix; everything else is a 1:1 phone and uses @s.whatsapp.net.
	if strings.HasPrefix(phone, "120362") || strings.HasPrefix(phone, "120363") {
		return phone + "@g.us"
	}
	return phone + "@s.whatsapp.net"
}

// doRequest is the single transport core for every GOWA API call. It builds
// the request, applies auth and the X-Device-Id header, sends it, and returns
// the raw response body and status code. Callers (doJSON, doMultipart, doRaw,
// doJSONRaw) own only their serialization and response interpretation.
func (c *Client) doRequest(ctx context.Context, method, path, deviceID, contentType string, body io.Reader) (int, []byte, error) {
	// The http.Client carries no global Timeout (media paths need minutes).
	// Apply the short default here unless the caller already set a deadline
	// (media sends arrive with MediaSendTimeout from sendMedia/DownloadMedia).
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.setAuth(req)
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("gowa request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("read response: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

// doJSON sends a JSON request to the GOWA API and decodes the send response.
func (c *Client) doJSON(ctx context.Context, method, path, deviceID string, body any) (string, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	statusCode, respBody, err := c.doRequest(ctx, method, path, deviceID, "application/json", reqBody)
	if err != nil {
		return "", err
	}

	return parseSendResponse(statusCode, respBody)
}

// doMultipart sends a multipart/form-data request to the GOWA API.
// fields contains the non-file form fields; fileField, fileName, fileData
// provide the binary attachment (if fileField is non-empty).
func (c *Client) doMultipart(ctx context.Context, method, path, deviceID string, fields map[string]string, fileField, fileName string, fileData []byte) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return "", fmt.Errorf("write field %s: %w", key, err)
		}
	}

	if fileField != "" && len(fileData) > 0 {
		// Create the file part with an explicit Content-Type based on the
		// filename extension. GOWA validates image uploads by extension and
		// rejects files without a recognized image extension.
		mime := mimeTypeForFilename(fileName)
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fileField, fileName),
		}
		h["Content-Type"] = []string{mime}
		part, err := writer.CreatePart(textproto.MIMEHeader(h))
		if err != nil {
			return "", fmt.Errorf("create file field: %w", err)
		}
		if _, err := part.Write(fileData); err != nil {
			return "", fmt.Errorf("write file data: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	statusCode, respBody, err := c.doRequest(ctx, method, path, deviceID, writer.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}

	return parseSendResponse(statusCode, respBody)
}

// doRaw sends a bodyless request (GET/DELETE) and returns the raw bytes,
// failing on any non-2xx status.
func (c *Client) doRaw(ctx context.Context, method, path, deviceID string) ([]byte, error) {
	statusCode, respBody, err := c.doRequest(ctx, method, path, deviceID, "", nil)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("gowa API returned status %d: %s", statusCode, string(respBody))
	}

	return respBody, nil
}

// doJSONRaw sends a JSON request with any HTTP method and returns the raw
// response body. Unlike doJSON (which returns just the message ID), this
// returns the full body for callers that need to unmarshal custom types.
func (c *Client) doJSONRaw(ctx context.Context, method, path, deviceID string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	statusCode, respBody, err := c.doRequest(ctx, method, path, deviceID, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("gowa API returned status %d: %s", statusCode, string(respBody))
	}

	return respBody, nil
}

// setAuth applies Basic Auth to the request.
func (c *Client) setAuth(req *http.Request) {
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// parseSendResponse decodes a GOWA send envelope and extracts the message ID.
func parseSendResponse(statusCode int, body []byte) (string, error) {
	var sr sendResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", fmt.Errorf("gowa API returned status %d: %s", statusCode, string(body))
	}

	if statusCode < 200 || statusCode >= 300 {
		return "", fmt.Errorf("gowa API error: %s", sr.Message)
	}

	if sr.Results.MessageID == "" {
		return "", fmt.Errorf("gowa API returned no message ID: %s", sr.Message)
	}

	return sr.Results.MessageID, nil
}

// cacheMedia stores raw bytes for the UploadMedia→send pattern and returns
// a temporary key that send methods use to consume the data inline.
func (c *Client) cacheMedia(data []byte, mimeType, filename string) string {
	id := fmt.Sprintf("gowa-media-%d", atomic.AddUint64(&c.idCounter, 1))
	c.mu.Lock()
	c.mediaCache[id] = mediaCacheItem{Data: data, MimeType: mimeType, Filename: filename}
	c.mu.Unlock()
	return id
}

// popMedia retrieves and removes cached media by key.
func (c *Client) popMedia(id string) (mediaCacheItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.mediaCache[id]
	if ok {
		delete(c.mediaCache, id)
	}
	return item, ok
}

// mimeTypeForFilename returns the MIME type for a filename based on its
// extension. Used to set the Content-Type on multipart file parts so GOWA
// can validate and process the upload correctly.
func mimeTypeForFilename(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".html", ".htm":
		return "text/html"
	case ".txt", ".md":
		return "text/plain"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// Compile-time assertion that Client satisfies whatsapp.Provider.
var _ whatsapp.Provider = (*Client)(nil)
