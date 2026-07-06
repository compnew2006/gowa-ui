package observability

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

var latencyBuckets = [...]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestKey struct {
	Method      string
	RouteGroup  string
	StatusClass string
}

type histogramSnapshot struct {
	Count   uint64
	Sum     float64
	Buckets []uint64
}

type Manager struct {
	cfg       config.ObservabilityConfig
	db        *gorm.DB
	redis     *redis.Client
	startedAt time.Time

	inflight atomic.Int64

	mu            sync.Mutex
	requestCounts map[requestKey]uint64
	durations     map[string]*histogramSnapshot

	// whatsmeowSMProvider is called during /metrics rendering to append whatsmeow-specific metrics.
	whatsmeowSMProvider func(buf *strings.Builder)
}

func NewManager(cfg config.ObservabilityConfig, db *gorm.DB, redis *redis.Client) *Manager {
	if !cfg.EnableMetrics && !cfg.EnablePprof {
		return nil
	}

	return &Manager{
		cfg:           cfg,
		db:            db,
		redis:         redis,
		startedAt:     time.Now(),
		requestCounts: make(map[requestKey]uint64),
		durations:     make(map[string]*histogramSnapshot),
	}
}

func (m *Manager) MetricsEnabled() bool {
	return m != nil && m.cfg.EnableMetrics
}

func (m *Manager) PprofEnabled() bool {
	return m != nil && m.cfg.EnablePprof
}

func (m *Manager) Wrap(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	if m == nil || !m.cfg.EnableMetrics {
		return next
	}

	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		m.inflight.Add(1)
		defer func() {
			m.inflight.Add(-1)
			status := ctx.Response.StatusCode()
			if status == 0 {
				status = fasthttp.StatusOK
			}
			m.observeRequest(string(ctx.Method()), string(ctx.Path()), status, time.Since(start))
		}()

		next(ctx)
	}
}

func (m *Manager) MetricsHandler() fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		if !m.authorize(r.RequestCtx) {
			return nil
		}

		body := m.renderPrometheus()
		r.RequestCtx.Response.Header.SetContentType("text/plain; version=0.0.4; charset=utf-8")
		r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
		r.RequestCtx.SetBody(body)
		return nil
	}
}

func (m *Manager) PprofHandler(handler http.Handler) fastglue.FastRequestHandler {
	adapted := fasthttpadaptor.NewFastHTTPHandler(handler)
	return func(r *fastglue.Request) error {
		if !m.authorize(r.RequestCtx) {
			return nil
		}
		adapted(r.RequestCtx)
		return nil
	}
}

func (m *Manager) RegisterPprofRoutes(g *fastglue.Fastglue) {
	if m == nil || !m.cfg.EnablePprof {
		return
	}

	g.GET("/debug/pprof/", m.PprofHandler(http.HandlerFunc(pprof.Index)))
	g.GET("/debug/pprof/cmdline", m.PprofHandler(http.HandlerFunc(pprof.Cmdline)))
	g.GET("/debug/pprof/profile", m.PprofHandler(http.HandlerFunc(pprof.Profile)))
	g.GET("/debug/pprof/symbol", m.PprofHandler(http.HandlerFunc(pprof.Symbol)))
	g.POST("/debug/pprof/symbol", m.PprofHandler(http.HandlerFunc(pprof.Symbol)))
	g.GET("/debug/pprof/trace", m.PprofHandler(http.HandlerFunc(pprof.Trace)))

	for _, profile := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		g.GET("/debug/pprof/"+profile, m.PprofHandler(pprof.Handler(profile)))
	}
}

func (m *Manager) authorize(ctx *fasthttp.RequestCtx) bool {
	if m == nil {
		denyObservability(ctx, fasthttp.StatusNotFound, "Observability is disabled")
		return false
	}

	token := strings.TrimSpace(m.cfg.AccessToken)
	if token != "" {
		provided := strings.TrimSpace(string(ctx.Request.Header.Peek("X-Observability-Token")))
		if provided == "" {
			const bearerPrefix = "Bearer "
			authHeader := strings.TrimSpace(string(ctx.Request.Header.Peek("Authorization")))
			if strings.HasPrefix(authHeader, bearerPrefix) {
				provided = strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			}
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1 {
			return true
		}

		ctx.Response.Header.Set("WWW-Authenticate", `Bearer realm="observability"`)
		denyObservability(ctx, fasthttp.StatusUnauthorized, "Observability token required")
		return false
	}

	if isLoopback(ctx.RemoteIP()) {
		return true
	}

	denyObservability(ctx, fasthttp.StatusForbidden, "Observability endpoints are loopback-only unless observability.access_token is configured")
	return false
}

func denyObservability(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.SetStatusCode(status)
	ctx.Response.Header.SetContentType("text/plain; charset=utf-8")
	ctx.SetBodyString(message)
}

func isLoopback(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func (m *Manager) observeRequest(method, path string, status int, duration time.Duration) {
	group := classifyPath(path)
	statusGroup := statusClass(status)
	seconds := duration.Seconds()

	m.mu.Lock()
	defer m.mu.Unlock()

	key := requestKey{
		Method:      method,
		RouteGroup:  group,
		StatusClass: statusGroup,
	}
	m.requestCounts[key]++

	hist := m.durations[group]
	if hist == nil {
		hist = &histogramSnapshot{
			Buckets: make([]uint64, len(latencyBuckets)),
		}
		m.durations[group] = hist
	}
	hist.Count++
	hist.Sum += seconds

	for idx, upperBound := range latencyBuckets {
		if seconds <= upperBound {
			hist.Buckets[idx]++
			return
		}
	}
}

func classifyPath(path string) string {
	switch {
	case path == "/health" || path == "/ready":
		return "health"
	case path == "/metrics":
		return "metrics"
	case strings.HasPrefix(path, "/debug/pprof"):
		return "pprof"
	case strings.HasPrefix(path, "/api/auth"):
		return "auth"
	case strings.HasPrefix(path, "/api/me"):
		return "me"
	case strings.HasPrefix(path, "/api/accounts"):
		return "accounts"
	case strings.HasPrefix(path, "/api/contacts"):
		return "contacts"
	case strings.HasPrefix(path, "/api/chats"):
		return "chats"
	case strings.HasPrefix(path, "/api/messages"):
		return "messages"
	case strings.HasPrefix(path, "/api/instances"):
		return "instances"
	case strings.HasPrefix(path, "/api/chatbot"):
		return "chatbot"
	case strings.HasPrefix(path, "/api/chat/"):
		return "chatbot"
	case strings.HasPrefix(path, "/api/analytics"):
		return "analytics"
	case strings.HasPrefix(path, "/api/widgets"):
		return "widgets"
	case strings.HasPrefix(path, "/api/users"):
		return "users"
	case strings.HasPrefix(path, "/api/roles"):
		return "roles"
	case strings.HasPrefix(path, "/api/organizations"):
		return "organizations"
	case strings.HasPrefix(path, "/api/teams"):
		return "teams"
	case strings.HasPrefix(path, "/api/templates"):
		return "templates"
	case strings.HasPrefix(path, "/api/flows"):
		return "flows"
	case strings.HasPrefix(path, "/api/campaigns"):
		return "campaigns"
	case strings.HasPrefix(path, "/api/tags"):
		return "tags"
	case strings.HasPrefix(path, "/api/statuses"):
		return "statuses"
	case strings.HasPrefix(path, "/api/notifications"):
		return "notifications"
	case strings.HasPrefix(path, "/api/webhook"):
		return "webhook"
	case strings.HasPrefix(path, "/api/config"):
		return "config"
	case strings.HasPrefix(path, "/api"):
		return "api_other"
	default:
		return "other"
	}
}

func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500 && status < 600:
		return "5xx"
	default:
		return "other"
	}
}

func (m *Manager) renderPrometheus() []byte {
	m.mu.Lock()
	requestCounts := make(map[requestKey]uint64, len(m.requestCounts))
	for key, value := range m.requestCounts {
		requestCounts[key] = value
	}

	durations := make(map[string]histogramSnapshot, len(m.durations))
	for group, hist := range m.durations {
		clone := histogramSnapshot{
			Count:   hist.Count,
			Sum:     hist.Sum,
			Buckets: make([]uint64, len(hist.Buckets)),
		}
		copy(clone.Buckets, hist.Buckets)
		durations[group] = clone
	}
	m.mu.Unlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var builder strings.Builder

	writeHelp(&builder, "whatomate_uptime_seconds", "Seconds since the Whatomate process started.")
	writeType(&builder, "whatomate_uptime_seconds", "gauge")
	fmt.Fprintf(&builder, "whatomate_uptime_seconds %.6f\n", time.Since(m.startedAt).Seconds())

	writeHelp(&builder, "whatomate_http_requests_in_flight", "Current number of in-flight HTTP requests.")
	writeType(&builder, "whatomate_http_requests_in_flight", "gauge")
	fmt.Fprintf(&builder, "whatomate_http_requests_in_flight %d\n", m.inflight.Load())

	writeHelp(&builder, "whatomate_http_requests_total", "Total HTTP requests observed by route group, method, and status class.")
	writeType(&builder, "whatomate_http_requests_total", "counter")
	countKeys := make([]requestKey, 0, len(requestCounts))
	for key := range requestCounts {
		countKeys = append(countKeys, key)
	}
	sort.Slice(countKeys, func(i, j int) bool {
		if countKeys[i].RouteGroup != countKeys[j].RouteGroup {
			return countKeys[i].RouteGroup < countKeys[j].RouteGroup
		}
		if countKeys[i].Method != countKeys[j].Method {
			return countKeys[i].Method < countKeys[j].Method
		}
		return countKeys[i].StatusClass < countKeys[j].StatusClass
	})
	for _, key := range countKeys {
		fmt.Fprintf(
			&builder,
			"whatomate_http_requests_total{method=%q,route_group=%q,status_class=%q} %d\n",
			key.Method,
			key.RouteGroup,
			key.StatusClass,
			requestCounts[key],
		)
	}

	writeHelp(&builder, "whatomate_http_request_duration_seconds", "HTTP request duration histogram by route group.")
	writeType(&builder, "whatomate_http_request_duration_seconds", "histogram")
	durationGroups := make([]string, 0, len(durations))
	for group := range durations {
		durationGroups = append(durationGroups, group)
	}
	sort.Strings(durationGroups)
	for _, group := range durationGroups {
		hist := durations[group]
		var cumulative uint64
		for idx, upperBound := range latencyBuckets {
			cumulative += hist.Buckets[idx]
			fmt.Fprintf(
				&builder,
				"whatomate_http_request_duration_seconds_bucket{route_group=%q,le=%q} %d\n",
				group,
				fmt.Sprintf("%.3f", upperBound),
				cumulative,
			)
		}
		fmt.Fprintf(&builder, "whatomate_http_request_duration_seconds_bucket{route_group=%q,le=\"+Inf\"} %d\n", group, hist.Count)
		fmt.Fprintf(&builder, "whatomate_http_request_duration_seconds_sum{route_group=%q} %.6f\n", group, hist.Sum)
		fmt.Fprintf(&builder, "whatomate_http_request_duration_seconds_count{route_group=%q} %d\n", group, hist.Count)
	}

	writeHelp(&builder, "whatomate_runtime_goroutines", "Number of live goroutines.")
	writeType(&builder, "whatomate_runtime_goroutines", "gauge")
	fmt.Fprintf(&builder, "whatomate_runtime_goroutines %d\n", runtime.NumGoroutine())

	writeHelp(&builder, "whatomate_runtime_heap_alloc_bytes", "Allocated heap bytes.")
	writeType(&builder, "whatomate_runtime_heap_alloc_bytes", "gauge")
	fmt.Fprintf(&builder, "whatomate_runtime_heap_alloc_bytes %d\n", mem.HeapAlloc)

	writeHelp(&builder, "whatomate_runtime_heap_inuse_bytes", "Heap bytes in use.")
	writeType(&builder, "whatomate_runtime_heap_inuse_bytes", "gauge")
	fmt.Fprintf(&builder, "whatomate_runtime_heap_inuse_bytes %d\n", mem.HeapInuse)

	writeHelp(&builder, "whatomate_runtime_gc_pause_total_seconds", "Total GC pause time in seconds.")
	writeType(&builder, "whatomate_runtime_gc_pause_total_seconds", "counter")
	fmt.Fprintf(&builder, "whatomate_runtime_gc_pause_total_seconds %.9f\n", float64(mem.PauseTotalNs)/float64(time.Second))

	if m.db != nil {
		if sqlDB, err := m.db.DB(); err == nil {
			stats := sqlDB.Stats()
			writeHelp(&builder, "whatomate_db_pool_open_connections", "Open database connections.")
			writeType(&builder, "whatomate_db_pool_open_connections", "gauge")
			fmt.Fprintf(&builder, "whatomate_db_pool_open_connections %d\n", stats.OpenConnections)

			writeHelp(&builder, "whatomate_db_pool_in_use_connections", "In-use database connections.")
			writeType(&builder, "whatomate_db_pool_in_use_connections", "gauge")
			fmt.Fprintf(&builder, "whatomate_db_pool_in_use_connections %d\n", stats.InUse)

			writeHelp(&builder, "whatomate_db_pool_idle_connections", "Idle database connections.")
			writeType(&builder, "whatomate_db_pool_idle_connections", "gauge")
			fmt.Fprintf(&builder, "whatomate_db_pool_idle_connections %d\n", stats.Idle)

			writeHelp(&builder, "whatomate_db_pool_wait_count_total", "Total waits for a database connection.")
			writeType(&builder, "whatomate_db_pool_wait_count_total", "counter")
			fmt.Fprintf(&builder, "whatomate_db_pool_wait_count_total %d\n", stats.WaitCount)

			writeHelp(&builder, "whatomate_db_pool_wait_duration_seconds_total", "Total time waiting for a database connection.")
			writeType(&builder, "whatomate_db_pool_wait_duration_seconds_total", "counter")
			fmt.Fprintf(&builder, "whatomate_db_pool_wait_duration_seconds_total %.9f\n", stats.WaitDuration.Seconds())

			writeHelp(&builder, "whatomate_db_pool_max_open_connections", "Configured maximum open database connections.")
			writeType(&builder, "whatomate_db_pool_max_open_connections", "gauge")
			fmt.Fprintf(&builder, "whatomate_db_pool_max_open_connections %d\n", stats.MaxOpenConnections)
		}
	}

	if m.redis != nil {
		stats := m.redis.PoolStats()
		writeHelp(&builder, "whatomate_redis_pool_hits_total", "Redis connection pool hits.")
		writeType(&builder, "whatomate_redis_pool_hits_total", "counter")
		fmt.Fprintf(&builder, "whatomate_redis_pool_hits_total %d\n", stats.Hits)

		writeHelp(&builder, "whatomate_redis_pool_misses_total", "Redis connection pool misses.")
		writeType(&builder, "whatomate_redis_pool_misses_total", "counter")
		fmt.Fprintf(&builder, "whatomate_redis_pool_misses_total %d\n", stats.Misses)

		writeHelp(&builder, "whatomate_redis_pool_timeouts_total", "Redis connection pool timeouts.")
		writeType(&builder, "whatomate_redis_pool_timeouts_total", "counter")
		fmt.Fprintf(&builder, "whatomate_redis_pool_timeouts_total %d\n", stats.Timeouts)

		writeHelp(&builder, "whatomate_redis_pool_total_connections", "Redis pool total connections.")
		writeType(&builder, "whatomate_redis_pool_total_connections", "gauge")
		fmt.Fprintf(&builder, "whatomate_redis_pool_total_connections %d\n", stats.TotalConns)

		writeHelp(&builder, "whatomate_redis_pool_idle_connections", "Redis pool idle connections.")
		writeType(&builder, "whatomate_redis_pool_idle_connections", "gauge")
		fmt.Fprintf(&builder, "whatomate_redis_pool_idle_connections %d\n", stats.IdleConns)

		writeHelp(&builder, "whatomate_redis_pool_stale_connections_total", "Redis pool stale connections.")
		writeType(&builder, "whatomate_redis_pool_stale_connections_total", "counter")
		fmt.Fprintf(&builder, "whatomate_redis_pool_stale_connections_total %d\n", stats.StaleConns)
	}

	if m.whatsmeowSMProvider != nil {
		m.whatsmeowSMProvider(&builder)
	}

	return []byte(builder.String())
}

// SetWhatsmeowMetricsProvider registers a callback that appends whatsmeow-specific
// metrics to the /metrics output. Pass nil to clear.
func (m *Manager) SetWhatsmeowMetricsProvider(provider func(buf *strings.Builder)) {
	if m == nil {
		return
	}
	m.whatsmeowSMProvider = provider
}

func writeHelp(builder *strings.Builder, metricName string, help string) {
	fmt.Fprintf(builder, "# HELP %s %s\n", metricName, help)
}

func writeType(builder *strings.Builder, metricName string, metricType string) {
	fmt.Fprintf(builder, "# TYPE %s %s\n", metricName, metricType)
}
