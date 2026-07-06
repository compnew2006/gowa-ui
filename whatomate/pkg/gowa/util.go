package gowa

import (
	"strconv"
	"time"
)

// isAbsoluteHTTPURL reports whether s begins with an http:// or https://
// scheme. Used by DownloadMedia to decide whether to treat the argument as a
// fetchable URL or a GOWA message_id.
func isAbsoluteHTTPURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}

// parseGowaTime parses GOWA's various time formats (RFC3339, RFC3339Nano,
// or raw Unix timestamps) and treats empty strings as a zero time.Time.
func parseGowaTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
	}
	if err != nil {
		if sec, parseErr := strconv.ParseInt(s, 10, 64); parseErr == nil {
			return time.Unix(sec, 0), nil
		}
	}
	return t, err
}
