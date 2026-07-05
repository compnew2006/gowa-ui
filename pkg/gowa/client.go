package gowa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
func (c *Client) SendImage(ctx context.Context, deviceID string, req SendMediaRequest) (string, error) {
	return c.sendMedia(ctx, deviceID, "/send/image", req)
}

// SendFile calls POST /send/file.
func (c *Client) SendFile(ctx context.Context, deviceID string, req SendMediaRequest) (string, error) {
	return c.sendMedia(ctx, deviceID, "/send/file", req)
}

// SendVideo calls POST /send/video.
func (c *Client) SendVideo(ctx context.Context, deviceID string, req SendMediaRequest) (string, error) {
	return c.sendMedia(ctx, deviceID, "/send/video", req)
}

// SendAudio calls POST /send/audio. Audio has no caption in GOWA's API.
func (c *Client) SendAudio(ctx context.Context, deviceID, phone, audioURL string) (string, error) {
	req := SendMediaRequest{Phone: phone, URL: audioURL}
	return c.sendMedia(ctx, deviceID, "/send/audio", req)
}

func (c *Client) sendMedia(ctx context.Context, deviceID, endpoint string, req SendMediaRequest) (string, error) {
	var env Envelope[MessageActionResponse]
	if err := c.do(ctx, http.MethodPost, endpoint, deviceID, req, &env); err != nil {
		return "", err
	}
	return env.Results.MessageID, nil
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
// FileURL returned by DownloadMedia) and returns the body bytes. It reuses
// the client's connection pool and timeout.
func (c *Client) FetchBytes(ctx context.Context, absoluteURL string) ([]byte, error) {
	req, err := newHTTPRequest(ctx, http.MethodGet, absoluteURL, nil)
	if err != nil {
		return nil, err
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
	return body, nil
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
		ge := &Error{StatusCode: resp.StatusCode}
		// Try to unwrap the GOWA envelope for a friendlier message/code.
		var env Envelope[json.RawMessage]
		if json.Unmarshal(respBody, &env) == nil {
			ge.Code = env.Code
			ge.Message = env.Message
		} else {
			ge.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		if ge.Message == "" {
			ge.Message = string(respBody)
		}
		return ge
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
