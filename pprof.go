package wt

import (
	"net/http"
	"net/http/pprof"
)

// PProfMux returns an http.ServeMux with pprof endpoints registered.
// Serve this on a separate port for profiling in production.
//
// Usage:
//
//	go http.ListenAndServe(":6060", wt.PProfMux())
//	// Then: go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
func PProfMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// DebugMux returns an http.ServeMux with pprof, health check, and metrics.
// One endpoint for all debugging/monitoring needs.
//
// Usage:
//
//	server := wt.New(...)
//	go http.ListenAndServe(":6060", wt.DebugMux(server))
func DebugMux(s *Server) *http.ServeMux {
	mux := PProfMux()
	mux.Handle("/health", NewHealthCheck(s))
	return mux
}
