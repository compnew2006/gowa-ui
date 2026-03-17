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
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           restrictedOutboundDialContext,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
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
	if resp.ContentLength > maxRemoteMediaSizeBytes {
		return nil, "", fmt.Errorf("remote media exceeds max size of %d bytes", maxRemoteMediaSizeBytes)
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
	if parsed.User != nil {
		return nil, fmt.Errorf("media URL must not include embedded credentials")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := resolveOutboundHostIPs(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	return parsed, nil
}

func restrictedOutboundDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid media host: %w", err)
	}

	ips, err := resolveOutboundHostIPs(ctx, host)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ip := range ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to resolve media host")
}

func resolveOutboundHostIPs(ctx context.Context, hostname string) ([]net.IP, error) {
	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedOutboundIP(ip) {
			return nil, fmt.Errorf("media URL host is not allowed")
		}
		return []net.IP{ip}, nil
	}

	ipAddrs, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil || len(ipAddrs) == 0 {
		return nil, fmt.Errorf("failed to resolve media URL host")
	}

	ips := make([]net.IP, 0, len(ipAddrs))
	for _, ipAddr := range ipAddrs {
		if isBlockedOutboundIP(ipAddr.IP) {
			return nil, fmt.Errorf("media URL host is not allowed")
		}
		ips = append(ips, ipAddr.IP)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to resolve media URL host")
	}
	return ips, nil
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
	resolvedBase := baseAbs
	if realBase, err := filepath.EvalSymlinks(baseAbs); err == nil {
		resolvedBase = realBase
	}
	fullPath := filepath.Join(resolvedBase, cleanRef)
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve media path: %w", err)
	}
	relPath, err := filepath.Rel(resolvedBase, fullAbs)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local media path")
	}
	resolvedPath, err := filepath.EvalSymlinks(fullAbs)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve local media file: %w", err)
	}
	resolvedRelPath, err := filepath.Rel(resolvedBase, resolvedPath)
	if err != nil || resolvedRelPath == ".." || strings.HasPrefix(resolvedRelPath, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local media path")
	}

	// #nosec G304 -- resolvedPath is constrained to media storage root via Abs+Rel+EvalSymlinks validation above.
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read local media file: %w", err)
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(resolvedPath)))
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
