package licensestudio

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func openBrowser(rawURL string) error {
	safeURL, err := validateBrowserURL(rawURL)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", safeURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", safeURL)
	default:
		cmd = exec.Command("xdg-open", safeURL)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func validateBrowserURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("browser URL is empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid browser URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported browser URL scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("browser URL must include a host")
	}

	return parsed.String(), nil
}
