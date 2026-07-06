package gowa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zerodha/logf"
)

// DefaultRequestTimeout is applied when the caller does not pass a timeout.
const DefaultRequestTimeout = 30 * time.Second

// Client is the low-level HTTP client for the GOWA REST API. It owns a single
// *http.Client (connection-pooled), holds the configured base URL and basic
// auth credentials, and exposes one method per GOWA endpoint. Methods are
// safe for concurrent use.
//
// All methods accept a context. Methods return *Error on non-2xx GOWA
// responses and bare transport errors on network failures; the adapter layer
// is responsible for retrying retryable failures.
//
// The client never sleeps for backoff itself — that is the adapter's job.
// This keeps the client deterministic and easy to test.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authHeader string // pre-built "Basic <b64>" header, empty if no auth
	log        logf.Logger
}

// NewClient builds a Client from the operator-supplied base URL and basic
// auth credentials. If both user and password are empty, no Authorization
// header is sent (GOWA may be deployed without basic auth).
func NewClient(baseURL, user, password string, timeoutSeconds int, log logf.Logger) *Client {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}
	c := &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			// Default transport is fine; GOWA is a trusted internal endpoint.
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		log:     log,
	}
	if user != "" || password != "" {
		cred := user + ":" + password
		c.authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(cred))
	}
	return c
}

// HTTPClient returns the underlying *http.Client. Exposed so the adapter can
// reuse the connection pool for direct media downloads (DownloadMedia).
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// BaseURL returns the configured GOWA base URL (without trailing slash).
func (c *Client) BaseURL() string { return c.baseURL }

// ----- Public API: device lifecycle -----

// ListDevices calls GET /devices on GOWA. Returns every device known to GOWA.
func (c *Client) ListDevices(ctx context.Context) ([]Device, error) {
	var env Envelope[[]Device]
	if err := c.do(ctx, http.MethodGet, "/devices", "", nil, &env); err != nil {
		return nil, err
	}
	return env.Results, nil
}

// CreateDevice calls POST /devices. Creates a device slot on GOWA. The
// webhook_url is what makes GOWA push inbound events back to whatomate.
func (c *Client) CreateDevice(ctx context.Context, req CreateDeviceRequest) (*Device, error) {
	var env Envelope[Device]
	if err := c.do(ctx, http.MethodPost, "/devices", "", req, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// GetDevice calls GET /devices/:device_id.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	var env Envelope[Device]
	if err := c.do(ctx, http.MethodGet, "/devices/"+url.PathEscape(deviceID), "", nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// DeleteDevice calls DELETE /devices/:device_id. On GOWA this fully purges the
// device slot (paired session + chat storage). To preserve the slot for
// re-pairing, call Logout instead.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodDelete, "/devices/"+url.PathEscape(deviceID), "", nil, nil)
}

// GetLoginQR calls GET /devices/:device_id/login and returns the QR code URL.
// The returned QRLink is GOWA-served and can be rendered in the whatomate UI
// directly, or fetched and proxied.
func (c *Client) GetLoginQR(ctx context.Context, deviceID string) (*LoginResponse, error) {
	var env Envelope[LoginResponse]
	path := "/app/login?device_id=" + url.QueryEscape(deviceID)
	if err := c.do(ctx, http.MethodGet, path, deviceID, nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// LoginWithCode calls GET /app/login-with-code?phone=... and
// returns a pairing code the user types into their phone.
func (c *Client) LoginWithCode(ctx context.Context, deviceID, phone string) (*LoginWithCodeResponse, error) {
	path := "/app/login-with-code?device_id=" + url.QueryEscape(deviceID) + "&phone=" + url.QueryEscape(phone)
	var env Envelope[LoginWithCodeResponse]
	if err := c.do(ctx, http.MethodGet, path, deviceID, nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// Logout calls POST /devices/:device_id/logout. On GOWA this preserves the
// device slot (unlike whatsmeow-mode whatomate which historically called
// Logout and lost the pairing). Use DeleteDevice for full purge.
func (c *Client) Logout(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, "/devices/"+url.PathEscape(deviceID)+"/logout", "", nil, nil)
}

// Reconnect calls POST /devices/:device_id/reconnect.
func (c *Client) Reconnect(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, "/devices/"+url.PathEscape(deviceID)+"/reconnect", "", nil, nil)
}

// GetStatus calls GET /devices/:device_id/status.
func (c *Client) GetStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	var env Envelope[DeviceStatus]
	if err := c.do(ctx, http.MethodGet, "/devices/"+url.PathEscape(deviceID)+"/status", "", nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// UpdateDeviceWebhook calls PATCH /devices/:device_id/webhook. Used to wire
// the inbound webhook_url when whatomate provisions a device.
func (c *Client) UpdateDeviceWebhook(ctx context.Context, deviceID string, req UpdateDeviceWebhookRequest) error {
	return c.do(ctx, http.MethodPatch, "/devices/"+url.PathEscape(deviceID)+"/webhook", "", req, nil)
}

// ----- Public API: chats and messages (read-side) -----

// ListChats calls GET /chats with the supplied filter. deviceID is sent in the
// X-Device-Id header per GOWA's device-scoping middleware.
func (c *Client) ListChats(ctx context.Context, deviceID string, filter ListChatsFilter) (*ListChatsResponse, error) {
	q := buildQuery(filterToValues(filter))
	var env Envelope[ListChatsResponse]
	if err := c.do(ctx, http.MethodGet, "/chats?"+q, deviceID, nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// GetChatMessages calls GET /chat/:chat_jid/messages.
func (c *Client) GetChatMessages(ctx context.Context, deviceID, chatJID string, filter GetMessagesFilter) (*GetMessagesResponse, error) {
	path := "/chat/" + url.PathEscape(chatJID) + "/messages?" + buildQuery(messageFilterToValues(filter))
	var env Envelope[GetMessagesResponse]
	if err := c.do(ctx, http.MethodGet, path, deviceID, nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// ----- Public API: send and message actions -----

// SendText calls POST /send/message.
func (c *Client) SendText(ctx context.Context, deviceID string, req SendTextRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, "/send/message", deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
}

// SendImage calls POST /send/image.
func (c *Client) SendImage(ctx context.Context, deviceID string, req SendImageRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, "/send/image", deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
}

// SendFile calls POST /send/file.
func (c *Client) SendFile(ctx context.Context, deviceID string, req SendFileRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, "/send/file", deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
}

// SendVideo calls POST /send/video.
func (c *Client) SendVideo(ctx context.Context, deviceID string, req SendVideoRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, "/send/video", deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
}

// SendAudio calls POST /send/audio.
func (c *Client) SendAudio(ctx context.Context, deviceID string, req SendAudioRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, "/send/audio", deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
}

// ----- Multipart uploads (binary file field) -----
//
// GOWA's /send/{image,file,video,audio} endpoints accept either a `*_url`
// field (JSON body, server fetches the URL) or a binary file field
// (`image`/`file`/`video`/`audio`) sent as multipart/form-data. The binary
// path is preferred when the caller already has the bytes because it avoids
// a localhost URL round-trip and means whatomate never has to persist the
// outgoing file to its own disk — GOWA becomes the single source of truth,
// mirroring how incoming media already works.

// mediaPart is the per-endpoint descriptor used by sendMediaMultipart.
type mediaPart struct {
	endpoint  string // /send/image, /send/file, /send/video, /send/audio
	fileField string // image | file | video | audio
}

var (
	partImage    = mediaPart{endpoint: "/send/image", fileField: "image"}
	partDocument = mediaPart{endpoint: "/send/file", fileField: "file"}
	partVideo    = mediaPart{endpoint: "/send/video", fileField: "video"}
	partAudio    = mediaPart{endpoint: "/send/audio", fileField: "audio"}
)

// sendMediaMultipart posts the raw bytes as multipart/form-data to the given
// GOWA /send/* endpoint. caption is optional (ignored for audio). phone is
// required by GOWA. filename/mimeType describe the bytes for the file part
// header; if mimeType is empty, GOWA will sniff it.
func (c *Client) sendMediaMultipart(ctx context.Context, deviceID, endpoint, fileField, phone, caption, filename, mimeType string, data []byte) (string, error) {
	fullURL := c.baseURL + "/" + strings.TrimPrefix(endpoint, "/")

	// Build the multipart body. We don't use mime/multipart.NewWriter on a
	// pipe because the bytes are already fully in memory (the upload handler
	// read them into a []byte); a bytes.Buffer is simpler and lets us set
	// Content-Type + Content-Length up front so GOWA can stream efficiently.
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// phone (required)
	if err := w.WriteField("phone", phone); err != nil {
		return "", fmt.Errorf("gowa: multipart write phone: %w", err)
	}
	// caption (optional; audio has no caption in GOWA's API)
	if caption != "" && fileField != "audio" {
		if err := w.WriteField("caption", caption); err != nil {
			return "", fmt.Errorf("gowa: multipart write caption: %w", err)
		}
	}
	// file part
	part, err := w.CreateFormFile(fileField, filename)
	if err != nil {
		return "", fmt.Errorf("gowa: multipart create file field: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("gowa: multipart write file bytes: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("gowa: multipart close: %w", err)
	}

	req, err := newHTTPRequest(ctx, http.MethodPost, fullURL, &body)
	if err != nil {
		return "", err
	}
	// Content-Type MUST be the multipart boundary the writer generated.
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", &Error{Cause: err, Message: "transport error talking to GOWA"}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return "", parseGowaError(resp.StatusCode, respBody)
	}

	var env Envelope[MessageActionResponse]
	if err := json.Unmarshal(respBody, &env); err != nil {
		return "", fmt.Errorf("gowa: decode multipart send response: %w (body=%q)", err, truncate(string(respBody), 200))
	}
	return env.Results.MessageID, nil
}

// SendImageMultipart uploads image bytes directly to /send/image.
func (c *Client) SendImageMultipart(ctx context.Context, deviceID, phone, caption, filename, mimeType string, data []byte) (string, error) {
	return c.sendMediaMultipart(ctx, deviceID, partImage.endpoint, partImage.fileField, phone, caption, filename, mimeType, data)
}

// SendFileMultipart uploads document bytes directly to /send/file.
func (c *Client) SendFileMultipart(ctx context.Context, deviceID, phone, caption, filename, mimeType string, data []byte) (string, error) {
	return c.sendMediaMultipart(ctx, deviceID, partDocument.endpoint, partDocument.fileField, phone, caption, filename, mimeType, data)
}

// SendVideoMultipart uploads video bytes directly to /send/video.
func (c *Client) SendVideoMultipart(ctx context.Context, deviceID, phone, caption, filename, mimeType string, data []byte) (string, error) {
	return c.sendMediaMultipart(ctx, deviceID, partVideo.endpoint, partVideo.fileField, phone, caption, filename, mimeType, data)
}

// SendAudioMultipart uploads audio bytes directly to /send/audio.
func (c *Client) SendAudioMultipart(ctx context.Context, deviceID, phone, filename, mimeType string, data []byte) (string, error) {
	return c.sendMediaMultipart(ctx, deviceID, partAudio.endpoint, partAudio.fileField, phone, "", filename, mimeType, data)
}

// ReactMessage calls POST /message/:message_id/reaction.
func (c *Client) ReactMessage(ctx context.Context, deviceID, messageID, phone, emoji string) error {
	req := MessageActionRequest{MessageID: messageID, Phone: phone, Emoji: emoji}
	return c.do(ctx, http.MethodPost, "/message/"+url.PathEscape(messageID)+"/reaction", deviceID, req, nil)
}

// RevokeMessage calls POST /message/:message_id/revoke. Unlike the Meta
// adapter (which returns a hardcoded "not supported" error), GOWA actually
// supports revocation via whatsmeow.
func (c *Client) RevokeMessage(ctx context.Context, deviceID, messageID, phone string) error {
	req := MessageActionRequest{MessageID: messageID, Phone: phone}
	return c.do(ctx, http.MethodPost, "/message/"+url.PathEscape(messageID)+"/revoke", deviceID, req, nil)
}

// MarkRead calls POST /message/:message_id/read.
func (c *Client) MarkRead(ctx context.Context, deviceID, messageID, phone string) error {
	req := MessageActionRequest{MessageID: messageID, Phone: phone}
	return c.do(ctx, http.MethodPost, "/message/"+url.PathEscape(messageID)+"/read", deviceID, req, nil)
}

// DownloadMedia calls GET /message/:message_id/download. Returns metadata
// including FileURL; the caller should fetch the bytes separately.
func (c *Client) DownloadMedia(ctx context.Context, deviceID, messageID, phone string) (*MediaDownloadResponse, error) {
	path := "/message/" + url.PathEscape(messageID) + "/download?phone=" + url.QueryEscape(phone)
	var env Envelope[MediaDownloadResponse]
	if err := c.do(ctx, http.MethodGet, path, deviceID, nil, &env); err != nil {
		return nil, err
	}
	return &env.Results, nil
}

// FetchBytes performs a plain HTTP GET against an absolute URL (typically the
// FileURL returned by DownloadMedia or the qr_link from /app/login) and returns
// the body bytes. It reuses the client's connection pool and timeout.
//
// deviceID is sent as the X-Device-Id header when non-empty AND the URL is on
// the GOWA host (same scheme+host as BaseURL). This is required for GOWA-served
// static assets in v8 builds — the device-scoping middleware enforces
// X-Device-Id on every request, including /statics/qrcode/*.png, and replies
// with HTTP 200 + a JSON error body (e.g. DEVICE_ID_REQUIRED) when it's
// missing. Without this header, callers receive the JSON error bytes instead
// of the real PNG and end up base64-encoding garbage into a data: URL.
//
// To fail loudly instead of returning JSON error bodies as raw bytes, this
// method also inspects the response Content-Type: a JSON reply on a request
// that expected binary bytes is treated as an error.
func (c *Client) FetchBytes(ctx context.Context, absoluteURL, deviceID string) ([]byte, error) {
	req, err := newHTTPRequest(ctx, http.MethodGet, absoluteURL, nil)
	if err != nil {
		return nil, err
	}
	if c.authHeader != "" && sameGowaOrigin(c.baseURL, absoluteURL) {
		req.Header.Set("Authorization", c.authHeader)
		if deviceID != "" {
			req.Header.Set("X-Device-Id", deviceID)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Cause: err, Message: "fetch failed"}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, &Error{StatusCode: resp.StatusCode, Message: "non-2xx fetching media"}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gowa: read media body: %w", err)
	}
	// Detect JSON error bodies returned with HTTP 200. GOWA's device-scoping
	// middleware replies 200 + {"code":"DEVICE_ID_REQUIRED",...} when the
	// X-Device-Id header is missing, even for static asset URLs. Without this
	// guard the caller would treat the JSON error text as the file bytes (the
	// source of the "fake QR" — the JSON was base64-encoded into a data: URL).
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "application/json") {
		// Best-effort parse to surface the GOWA code/message; fall back to raw snippet.
		var env Envelope[json.RawMessage]
		code, message := "", string(body)
		if json.Unmarshal(body, &env) == nil {
			code, message = env.Code, env.Message
		}
		if code != "" || strings.Contains(message, "DEVICE_ID_REQUIRED") || strings.Contains(message, "not found") {
			return nil, &Error{
				StatusCode: resp.StatusCode,
				Code:       code,
				Message:    fmt.Sprintf("gowa: fetching %s returned a JSON error (status=%d, content-type=%s): %s — missing X-Device-Id header or stale URL", absoluteURL, resp.StatusCode, ct, message),
			}
		}
	}
	return body, nil
}

// sameGowaOrigin reports whether absoluteURL shares scheme+host with baseURL.
// Used to scope Authorization and X-Device-Id headers to GOWA-served URLs only
// (so FetchBytes on a third-party URL doesn't leak credentials).
func sameGowaOrigin(baseURL, absoluteURL string) bool {
	if baseURL == "" || absoluteURL == "" {
		return false
	}
	if !strings.HasPrefix(absoluteURL, "http://") && !strings.HasPrefix(absoluteURL, "https://") {
		return false
	}
	// Strip scheme from both, compare host[:port].
	bp := strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")
	ap := strings.TrimPrefix(strings.TrimPrefix(absoluteURL, "https://"), "http://")
	if i := strings.IndexByte(bp, '/'); i >= 0 {
		bp = bp[:i]
	}
	if i := strings.IndexByte(ap, '/'); i >= 0 {
		ap = ap[:i]
	}
	return bp != "" && bp == ap
}

// ----- Internal HTTP plumbing -----

// do executes a single HTTP request against GOWA. It is the single chokepoint
// through which every public method passes, so retries, headers, and error
// normalisation live here.
//
//   - method: HTTP verb
//   - path: path relative to baseURL, may include query string
//   - deviceID: if non-empty, sent as X-Device-Id (GOWA device scoping)
//   - body: JSON-marshalled into the request body; nil for GET/DELETE
//   - out: JSON-unmarshalled response envelope on success; nil to ignore body
func (c *Client) do(ctx context.Context, method, path, deviceID string, body any, out any) error {
	fullURL := c.baseURL + "/" + strings.TrimPrefix(path, "/")

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("gowa: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := newHTTPRequest(ctx, method, fullURL, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Transport-level failure (DNS, connection refused, deadline).
		return &Error{Cause: err, Message: "transport error talking to GOWA"}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return parseGowaError(resp.StatusCode, respBody)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("gowa: decode response: %w (body=%q)", err, truncate(string(respBody), 256))
	}
	return nil
}

// newHTTPRequest builds a context-scoped *http.Request and wraps build
// failures in the package's standard "gowa: build request" error. Shared by
// do() (relative path, headers applied by the caller) and FetchBytes()
// (absolute URL, no GOWA auth headers).
func newHTTPRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("gowa: build request: %w", err)
	}
	return req, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// parseGowaError converts a non-2xx GOWA response body into a typed *Error.
// GOWA always returns its standard envelope {status,code,message,results}
// even on failure, so we unwrap code/message for friendlier upstream errors.
// Falls back to a plain HTTP status string if the body isn't valid JSON.
func parseGowaError(statusCode int, body []byte) *Error {
	ge := &Error{StatusCode: statusCode}
	var env Envelope[json.RawMessage]
	if json.Unmarshal(body, &env) == nil {
		ge.Code = env.Code
		ge.Message = env.Message
	} else {
		ge.Message = fmt.Sprintf("HTTP %d", statusCode)
	}
	if ge.Message == "" {
		ge.Message = string(body)
	}
	return ge
}

func buildQuery(vals map[string]string) string {
	q := url.Values{}
	for k, v := range vals {
		q.Set(k, v)
	}
	return q.Encode()
}

func filterToValues(f ListChatsFilter) map[string]string {
	m := map[string]string{}
	if f.Limit > 0 {
		m["limit"] = strconv.Itoa(f.Limit)
	}
	if f.Offset > 0 {
		m["offset"] = strconv.Itoa(f.Offset)
	}
	if f.Search != "" {
		m["search"] = f.Search
	}
	if f.HasMedia {
		m["has_media"] = "true"
	}
	if f.Archived != nil {
		m["archived"] = strconv.FormatBool(*f.Archived)
	}
	return m
}

func messageFilterToValues(f GetMessagesFilter) map[string]string {
	m := map[string]string{}
	if f.Limit > 0 {
		m["limit"] = strconv.Itoa(f.Limit)
	}
	if f.Offset > 0 {
		m["offset"] = strconv.Itoa(f.Offset)
	}
	if f.Search != "" {
		m["search"] = f.Search
	}
	if f.StartTime != "" {
		m["start_time"] = f.StartTime
	}
	if f.EndTime != "" {
		m["end_time"] = f.EndTime
	}
	if f.MediaOnly {
		m["media_only"] = "true"
	}
	if f.IsFromMe != nil {
		m["is_from_me"] = strconv.FormatBool(*f.IsFromMe)
	}
	return m
}
