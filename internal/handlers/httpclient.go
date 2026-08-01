package handlers

import (
	"net/http"
	"time"
)

// NewSharedHTTPClient returns the production HTTP client used for outbound
// API calls (webhooks, custom actions, GOWA fetches). It is the single
// definition of the pooled production client — production code in cmd/gowa-ui
// calls this instead of inlining the literal, so connection-pool tuning lives
// in one place. Tests keep their own stripped-down client (see testhelpers_test.go).
func NewSharedHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:         SSRFSafeDialer(),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
