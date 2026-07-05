package gowa

// isAbsoluteHTTPURL reports whether s begins with an http:// or https://
// scheme. Used by DownloadMedia to decide whether to treat the argument as a
// fetchable URL or a GOWA message_id.
func isAbsoluteHTTPURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}
