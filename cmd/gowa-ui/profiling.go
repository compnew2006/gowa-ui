package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/zerodha/logf"
)

const (
	mutexProfileFraction = 5
	blockProfileRate     = 1_000_000 // one millisecond, in nanoseconds
)

type profilingServer struct {
	server                       *http.Server
	previousMutexProfileFraction int
}

// startProfilingServer starts a private pprof HTTP server when explicitly
// enabled. The listener accepts only a literal loopback IP to prevent DNS
// rebinding or accidentally exposing profiling endpoints on a public address.
func startProfilingServer(cfg config.ProfilingConfig, lo logf.Logger) (*profilingServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	host, _, err := net.SplitHostPort(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("profiling.address must be a loopback IP and port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("profiling.address host %q must be a literal loopback IP (127.0.0.1 or ::1)", host)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("listen on profiling.address %q: %w", cfg.Address, err)
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !tcpAddr.IP.IsLoopback() {
		_ = listener.Close()
		return nil, fmt.Errorf("profiling listener %q did not bind to a loopback IP", cfg.Address)
	}

	server := &profilingServer{
		server: &http.Server{
			Addr:              listener.Addr().String(),
			Handler:           newProfilingMux(),
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
		previousMutexProfileFraction: runtime.SetMutexProfileFraction(mutexProfileFraction),
	}
	runtime.SetBlockProfileRate(blockProfileRate)

	go func() {
		if err := server.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lo.Error("Profiling server stopped unexpectedly", "error", err)
		}
	}()
	lo.Info("Profiling server listening", "address", listener.Addr().String())
	return server, nil
}

// newProfilingMux intentionally does not use http.DefaultServeMux. The
// profiler is isolated from the public fasthttp application server and only
// reachable through the loopback listener created above.
func newProfilingMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
	}
	return mux
}

func (s *profilingServer) restoreSampling() {
	if s == nil {
		return
	}
	runtime.SetMutexProfileFraction(s.previousMutexProfileFraction)
	runtime.SetBlockProfileRate(0)
}
