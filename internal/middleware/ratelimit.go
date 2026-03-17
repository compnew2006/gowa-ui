package middleware

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// RateLimitOpts configures the rate limit middleware.
type RateLimitOpts struct {
	Redis      *redis.Client
	Log        logf.Logger
	Max        int           // Maximum attempts within the window.
	Window     time.Duration // Fixed window duration.
	KeyPrefix  string        // Redis key prefix (e.g., "login", "register").
	TrustProxy bool          // Trust X-Forwarded-For / X-Real-IP headers.
	KeyFunc    func(r *fastglue.Request, clientIP string) string
}

// RateLimit returns a fastglue middleware that enforces a fixed-window
// rate limit per client IP using Redis INCR + EXPIRE.
// It fails open: if Redis is unavailable the request is allowed through.
func RateLimit(opts RateLimitOpts) fastglue.FastMiddleware {
	return func(r *fastglue.Request) *fastglue.Request {
		if opts.Redis == nil || opts.Max <= 0 || opts.Window <= 0 {
			return r
		}

		clientIP := extractClientIP(r, opts.TrustProxy)
		keySuffix := clientIP
		if opts.KeyFunc != nil {
			if candidate := strings.TrimSpace(opts.KeyFunc(r, clientIP)); candidate != "" {
				keySuffix = candidate
			}
		}
		key := fmt.Sprintf("ratelimit:%s:%s", opts.KeyPrefix, keySuffix)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		count, err := opts.Redis.Incr(ctx, key).Result()
		if err != nil {
			// Fail open — log and allow request.
			opts.Log.Error("Rate limit Redis INCR failed", "error", err, "key", key)
			return r
		}

		// Set expiry on first increment (new window).
		if count == 1 {
			if err := opts.Redis.Expire(ctx, key, opts.Window).Err(); err != nil {
				opts.Log.Error("Rate limit Redis EXPIRE failed", "error", err, "key", key)
			}
		}

		if count > int64(opts.Max) {
			// Look up remaining TTL for Retry-After header.
			ttl, err := opts.Redis.TTL(ctx, key).Result()
			if err != nil || ttl < 0 {
				ttl = opts.Window
			}
			retryAfter := int(ttl.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			r.RequestCtx.Response.Header.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			_ = r.SendErrorEnvelope(fasthttp.StatusTooManyRequests,
				"Too many requests. Please try again later.", nil, "")
			return nil
		}

		return r
	}
}

// extractClientIP returns the client IP address from the request.
// When trustProxy is true, it only honors forwarded headers from a trusted
// proxy peer (loopback/private/link-local remote addresses).
func extractClientIP(r *fastglue.Request, trustProxy bool) string {
	if trustProxy && shouldTrustForwardedHeaders(r.RequestCtx.RemoteAddr()) {
		// X-Forwarded-For may contain a chain: "client, proxy1, proxy2"
		if xff := parseForwardedIP(string(r.RequestCtx.Request.Header.Peek("X-Forwarded-For"))); xff != "" {
			return xff
		}
		if realIP := parseForwardedIP(string(r.RequestCtx.Request.Header.Peek("X-Real-IP"))); realIP != "" {
			return realIP
		}
	}

	// Fall back to RemoteAddr (strip port).
	addr := r.RequestCtx.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func parseForwardedIP(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	if strings.Contains(candidate, ",") {
		candidate = strings.TrimSpace(strings.SplitN(candidate, ",", 2)[0])
	}
	candidate = strings.Trim(candidate, "[]")
	if net.ParseIP(candidate) == nil {
		return ""
	}
	return candidate
}

func shouldTrustForwardedHeaders(remoteAddr net.Addr) bool {
	if remoteAddr == nil {
		return false
	}

	host := strings.TrimSpace(remoteAddr.String())
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
