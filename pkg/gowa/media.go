package gowa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shridarpatil/whatomate/pkg/whatsapp"
)

// UploadMedia caches raw bytes for inline sending.
// GOWA has no standalone upload endpoint — media is sent inline in the
// multipart send call. This method stores the bytes in an in-memory cache
// and returns a temporary key that the send methods consume.
func (c *Client) UploadMedia(ctx context.Context, account *whatsapp.Account, data []byte, mimeType, filename string) (string, error) {
	return c.cacheMedia(data, mimeType, filename), nil
}

// GetMediaURL retrieves a downloadable URL for a media message.
// In GOWA, media is identified by message ID, not a separate media ID.
// This method calls the download endpoint and returns the file_url.
// The mediaID parameter is treated as the GOWA message ID.
func (c *Client) GetMediaURL(ctx context.Context, mediaID string, account *whatsapp.Account) (string, error) {
	path := fmt.Sprintf("/message/%s/download?phone=%s", mediaID, "")
	// phone (chat JID) is required by GOWA but we don't have it from this
	// signature. The download endpoint will 400 without it. This is a known
	// interface mismatch — callers using GOWA should use DownloadMedia directly.
	rawBody, err := c.doRaw(ctx, "GET", path, deviceID(account))
	if err != nil {
		return "", err
	}

	var dlResp downloadResponse
	if err := json.Unmarshal(rawBody, &dlResp); err != nil {
		return "", fmt.Errorf("parse download response: %w", err)
	}

	if dlResp.Results.FileURL == "" {
		return "", fmt.Errorf("no file URL in download response")
	}

	return dlResp.Results.FileURL, nil
}

// DownloadMedia downloads media content from a URL (typically the file_url
// returned by GOWA's download endpoint).
// In the whatomate Meta flow, the handler calls GetMediaURL then DownloadMedia.
// For GOWA, GetMediaURL returns the file_url and DownloadMedia fetches it.
func (c *Client) DownloadMedia(ctx context.Context, mediaURL string, accessToken string) ([]byte, error) {
	_ = accessToken // GOWA uses Basic Auth, not bearer tokens

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read media body: %w", err)
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

	if dlResp.Results.FileURL == "" {
		return nil, "", fmt.Errorf("no file URL in download response")
	}

	// Fetch the actual bytes from the file URL.
	data, err := c.DownloadMedia(ctx, dlResp.Results.FileURL, "")
	if err != nil {
		return nil, "", err
	}

	return data, dlResp.Results.MediaType, nil
}
