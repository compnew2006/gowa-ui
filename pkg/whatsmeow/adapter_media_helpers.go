package whatsmeow

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
)

// downloadMediaFromURL resolves outbound media references.
// Supported forms: http/https, file://, absolute local paths, and local paths relative to storage root.
func (a *WhatsmeowAdapter) downloadMediaFromURL(mediaRef string) ([]byte, string, error) {
	ref := strings.TrimSpace(mediaRef)
	if ref == "" {
		return nil, "", fmt.Errorf("media reference is empty")
	}

	parsed, err := url.Parse(ref)
	if err == nil && parsed.Scheme != "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			return a.downloadHTTPMedia(ref)
		case "file":
			return a.readLocalMedia(parsed.Path)
		default:
			return nil, "", fmt.Errorf("unsupported media URL scheme: %s", parsed.Scheme)
		}
	}

	return a.readLocalMedia(ref)
}

func (a *WhatsmeowAdapter) downloadHTTPMedia(u string) ([]byte, string, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download media, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read media body: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return data, mimeType, nil
}

func (a *WhatsmeowAdapter) readLocalMedia(pathRef string) ([]byte, string, error) {
	basePath := "./uploads"
	if a.manager != nil && a.manager.mediaStoragePath != "" {
		basePath = a.manager.mediaStoragePath
	}

	fullPath := ""
	if filepath.IsAbs(pathRef) {
		fullPath = pathRef
	} else {
		cleanRef := filepath.Clean(pathRef)
		if cleanRef == "." || cleanRef == "" || cleanRef == ".." ||
			strings.HasPrefix(cleanRef, ".."+string(os.PathSeparator)) {
			return nil, "", fmt.Errorf("invalid local media path: %s", pathRef)
		}
		fullPath = filepath.Join(basePath, cleanRef)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read local media file %s: %w", fullPath, err)
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fullPath)))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return data, mimeType, nil
}

// uploadMediaToWhatsApp uploads media with retry logic using configured backoff.
func (a *WhatsmeowAdapter) uploadMediaToWhatsApp(ctx context.Context, client *whatsmeow.Client, data []byte, appType whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	resp, err := client.Upload(ctx, data, appType)
	if err == nil {
		return resp, nil
	}

	retryCount := 1
	retryDelay := 2 * time.Second
	if a.manager != nil && a.manager.cfg != nil {
		retryCount = a.manager.cfg.UploadRetryCount
		retryDelay = time.Duration(a.manager.cfg.UploadRetryDelaySec) * time.Second
	}

	for i := 0; i < retryCount; i++ {
		a.logger.Warn("Media upload failed, retrying after backoff", "error", err, "attempt", i+1)

		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return whatsmeow.UploadResponse{}, fmt.Errorf("upload cancelled during retry backoff: %w", ctx.Err())
		}

		resp, err = client.Upload(ctx, data, appType)
		if err == nil {
			return resp, nil
		}
	}

	return whatsmeow.UploadResponse{}, fmt.Errorf("failed to upload media after %d retries: %w", retryCount, err)
}
