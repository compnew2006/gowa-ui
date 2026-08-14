package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/compnew2006/gowa-ui/pkg/whatsapp"
)

// UploadMedia caches raw bytes for inline sending.
// GOWA has no standalone upload endpoint — media is sent inline in the
// multipart send call. This method stores the bytes in an in-memory cache
// and returns a temporary key that the send methods consume.
func (c *Client) UploadMedia(ctx context.Context, account *whatsapp.Account, data []byte, mimeType, filename string) (string, error) {
	return c.cacheMedia(data, mimeType, filename), nil
}

// GetMediaURL retrieves a downloadable URL for a media message.
//
// The whatsapp.Provider interface mandates this signature, but GOWA identifies
// media by message ID AND requires the chat JID (`phone`) to download — a value
// this signature does not carry. The previous implementation sent an empty
// phone query, which GOWA rejects with 400 — a latent API defect (gap #12). It
// now fails fast with ErrNotSupported; the production download path uses
// DownloadMessageMedia (which carries the JID).
func (c *Client) GetMediaURL(ctx context.Context, mediaID string, account *whatsapp.Account) (string, error) {
	_ = ctx
	_ = mediaID
	_ = account
	return "", whatsapp.ErrNotSupported
}

// MaxMediaDownloadSize caps how many bytes DownloadMedia will read into memory.
// Bounds memory use so a runaway or malicious media URL can't exhaust the
// process (gap #7).
const MaxMediaDownloadSize = 25 * 1024 * 1024 // 25 MiB

// URLMatchesBase reports whether rawURL is an absolute HTTP(S) URL whose
// scheme+host(+port) match baseURL. Used as the SSRF gate before fetching a
// webhook-supplied media URL: only URLs that belong to the GOWA instance
// itself are fetched, so an attacker-controlled URL in a signed webhook can
// neither leak Basic Auth nor be fetched as an SSRF vector (gap #7).
//
// A relative or non-absolute rawURL returns false (callers resolve those
// against baseURL themselves).
func URLMatchesBase(rawURL, baseURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || !u.IsAbs() {
		return false
	}
	return sameOrigin(u, baseURL)
}

// sameOrigin reports whether u matches the origin (scheme://host[:port]) of
// baseURL. Empty baseURL never matches (fails closed).
func sameOrigin(u *url.URL, baseURL string) bool {
	if baseURL == "" {
		return false
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Host == "" {
		return false
	}
	return u.Host == base.Host
}

// DownloadMedia downloads media content from a URL (typically the file_url
// returned by GOWA's download endpoint).
// In the gowa-ui Meta flow, the handler calls GetMediaURL then DownloadMedia.
// For GOWA, GetMediaURL returns the file_url and DownloadMedia fetches it.
//
// Security (gap #7): Basic Auth is attached ONLY when the destination origin
// matches the client's baseURL — never to an arbitrary host, so a signed
// webhook carrying an external media URL cannot leak the GOWA credentials.
// The shared httpClient also blocks cross-host redirects (see New), and the
// body is capped at MaxMediaDownloadSize. The PRIMARY gate is in the handler
// (DownloadAndSaveMedia only fetches URLs on the GOWA base host); this is the
// defense-in-depth backstop.
func (c *Client) DownloadMedia(ctx context.Context, mediaURL string, accessToken string) ([]byte, error) {
	_ = accessToken // GOWA uses Basic Auth, not bearer tokens

	// Large media (up to MaxMediaDownloadSize) can take a while to transfer;
	// give downloads the same extended budget as sends instead of the 30s
	// default (the http.Client no longer caps this).
	ctx, cancel := context.WithTimeout(ctx, MediaSendTimeout)
	defer cancel()

	parsed, err := url.Parse(mediaURL)
	if err != nil {
		return nil, fmt.Errorf("parse media url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	// Attach Basic Auth only when the target is the GOWA instance itself.
	if sameOrigin(parsed, c.baseURL) {
		c.setAuth(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// CheckRedirect errors leave a non-nil Response whose Body must be closed.
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(body))
	}

	// Cap the download so a malicious or runaway URL can't exhaust memory.
	limited := io.LimitReader(resp.Body, MaxMediaDownloadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read media body: %w", err)
	}
	if int64(len(data)) > MaxMediaDownloadSize {
		return nil, fmt.Errorf("media exceeds max download size (%d bytes)", MaxMediaDownloadSize)
	}

	return data, nil
}

// DownloadMessageMedia is a GOWA-specific helper that downloads media for
// a given message ID and chat JID in a single call. This is the preferred
// download path for GOWA since the whatsapp.Provider interface signature
// for GetMediaURL lacks the phone/JID parameter.
func (c *Client) DownloadMessageMedia(ctx context.Context, account *whatsapp.Account, messageID, chatJID string) ([]byte, string, error) {
	path := fmt.Sprintf("/message/%s/download?phone=%s", messageID, chatJID)
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID(account))
	if err != nil {
		return nil, "", err
	}

	var dlResp downloadResponse
	if err := json.Unmarshal(rawBody, &dlResp); err != nil {
		return nil, "", fmt.Errorf("parse download response: %w", err)
	}

	if dlResp.Results.FileURL == "" && dlResp.Results.FilePath == "" {
		return nil, "", fmt.Errorf("no file URL in download response")
	}

	// GOWA returns file_url with its OWN hostname but often WITHOUT the port
	// (e.g. "http://localhost/statics/..." instead of "http://localhost:3080/..."),
	// which makes the subsequent fetch hit the wrong host:port (connection refused
	// on :80). Resolve the URL against the client's known base URL: prefer the
	// relative file_path joined to the base URL, and only fall back to file_url
	// if file_path is absent and file_url is absolute with an explicit port.
	fileURL := resolveGowaFileURL(c.baseURL, dlResp.Results.FilePath, dlResp.Results.FileURL)

	// Fetch the actual bytes from the file URL.
	data, err := c.DownloadMedia(ctx, fileURL, "")
	if err != nil {
		return nil, "", err
	}

	return data, dlResp.Results.MediaType, nil
}

// resolveGowaFileURL builds a fetchable URL for a GOWA-downloaded media file.
// GOWA's file_url uses the server's hostname but frequently omits the port
// (http://localhost/...), so fetching it directly fails. We trust the client's
// base URL instead and join the relative file_path onto it. If file_path is
// empty, we keep file_url but only when it already carries a port (host:port);
// otherwise we treat the path portion as relative and rejoin it to baseURL.
func resolveGowaFileURL(baseURL, filePath, fileURL string) string {
	// Preferred: relative file_path joined to the known base URL.
	if filePath != "" {
		return strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(filePath, "/")
	}
	// Fallback: repair file_url. If it lacks an explicit port, swap its scheme
	// + host for the base URL's and keep the path.
	if u, err := url.Parse(fileURL); err == nil && u.IsAbs() {
		if base, bErr := url.Parse(baseURL); bErr == nil && base.Host != "" {
			if _, _, pErr := net.SplitHostPort(u.Host); pErr != nil {
				// u.Host has no port — rebuild using base URL's scheme+host(+port).
				u.Scheme = base.Scheme
				u.Host = base.Host
				return u.String()
			}
		}
	}
	return fileURL
}
