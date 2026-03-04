package whatsmeow

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
)

// downloadMediaFromURL resolves outbound media references.
// Supported forms: http/https and local paths relative to storage root.
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
		default:
			return nil, "", fmt.Errorf("unsupported media URL scheme: %s", parsed.Scheme)
		}
	}

	return a.readLocalMedia(ref)
}

const maxRemoteMediaSizeBytes = 32 * 1024 * 1024 // 32MB hard cap for remote fetches

func (a *WhatsmeowAdapter) downloadHTTPMedia(u string) ([]byte, string, error) {
	parsed, err := validateOutboundMediaURL(u)
	if err != nil {
		return nil, "", err
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 4 {
				return fmt.Errorf("too many redirects")
			}
			_, err := validateOutboundMediaURL(req.URL.String())
			return err
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build media request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download media, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteMediaSizeBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read media body: %w", err)
	}
	if len(data) > maxRemoteMediaSizeBytes {
		return nil, "", fmt.Errorf("remote media exceeds max size of %d bytes", maxRemoteMediaSizeBytes)
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

func validateOutboundMediaURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid media URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("only http/https media URLs are allowed")
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("media URL must include a valid host")
	}

	hostname := parsed.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedOutboundIP(ip) {
			return nil, fmt.Errorf("media URL host is not allowed")
		}
		return parsed, nil
	}

	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("failed to resolve media URL host")
	}
	for _, ip := range ips {
		if isBlockedOutboundIP(ip) {
			return nil, fmt.Errorf("media URL host is not allowed")
		}
	}

	return parsed, nil
}

func isBlockedOutboundIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// Block carrier-grade NAT range: 100.64.0.0/10
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && (v4[1]&0xC0) == 64 {
		return true
	}
	return false
}

func (a *WhatsmeowAdapter) readLocalMedia(pathRef string) ([]byte, string, error) {
	basePath := "./uploads"
	if a.manager != nil && a.manager.mediaStoragePath != "" {
		basePath = a.manager.mediaStoragePath
	}

	cleanRef := filepath.Clean(strings.TrimSpace(pathRef))
	if cleanRef == "." || cleanRef == "" || cleanRef == ".." ||
		filepath.IsAbs(cleanRef) ||
		strings.HasPrefix(cleanRef, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local media path")
	}

	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve media storage path: %w", err)
	}
	fullPath := filepath.Join(baseAbs, cleanRef)
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve media path: %w", err)
	}
	relPath, err := filepath.Rel(baseAbs, fullAbs)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local media path")
	}

	// #nosec G304 -- fullAbs is constrained to media storage root via Abs+Rel validation above.
	data, err := os.ReadFile(fullAbs)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read local media file: %w", err)
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fullAbs)))
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
