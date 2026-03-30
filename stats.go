package wt

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sync/atomic"
	"time"
)

// RuntimeStats provides real-time server and Go runtime statistics.
type RuntimeStats struct {
	// Server
	ActiveSessions int    `json:"active_sessions"`
	ServerVersion  string `json:"server_version"`
	Uptime         string `json:"uptime"`

	// Go runtime
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`

	// Memory
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	StackInUseMB float64 `json:"stack_in_use_mb"`
	NumGC        uint32  `json:"num_gc"`
}

// Stats returns current runtime statistics for the server.
func (s *Server) Stats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := ""
	if s.wt != nil {
		// Approximate — no exact start time tracked
		uptime = "running"
	}

	return RuntimeStats{
		ActiveSessions: s.SessionCount(),
		ServerVersion:  Version,
		Uptime:         uptime,
		NumGoroutine:   runtime.NumGoroutine(),
		NumCPU:         runtime.NumCPU(),
		GoVersion:      runtime.Version(),
		HeapAllocMB:    float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:      float64(m.HeapSys) / 1024 / 1024,
		StackInUseMB:   float64(m.StackInuse) / 1024 / 1024,
		NumGC:          m.NumGC,
	}
}

// StatsJSON returns runtime stats as JSON bytes.
func (s *Server) StatsJSON() []byte {
	data, _ := json.Marshal(s.Stats())
	return data
}

// RuntimeStatsCollector periodically collects stats and calls the callback.
// Useful for pushing stats to external monitoring systems.
type RuntimeStatsCollector struct {
	server   *Server
	interval time.Duration
	done     chan struct{}
}

// NewRuntimeStatsCollector creates a stats collector.
func NewRuntimeStatsCollector(s *Server, interval time.Duration, fn func(RuntimeStats)) *RuntimeStatsCollector {
	c := &RuntimeStatsCollector{
		server:   s,
		interval: interval,
		done:     make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fn(s.Stats())
			case <-c.done:
				return
			}
		}
	}()
	return c
}

// Stop stops the collector.
func (c *RuntimeStatsCollector) Stop() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

// HealthCheck provides an HTTP health check endpoint that reports server status.
// Serve this alongside your WebTransport server on an HTTP port for load balancers
// and monitoring systems.
type HealthCheck struct {
	server    *Server
	startTime time.Time
}

// NewHealthCheck creates a health check handler for the given server.
func NewHealthCheck(s *Server) *HealthCheck {
	return &HealthCheck{
		server:    s,
		startTime: time.Now(),
	}
}

// HealthResponse is the JSON response from the health check endpoint.
type HealthResponse struct {
	Status         string `json:"status"`
	ActiveSessions int    `json:"active_sessions"`
	Uptime         string `json:"uptime"`
	Transport      string `json:"transport"`
}

// ServeHTTP implements http.Handler for health checks.
func (h *HealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:         "ok",
		ActiveSessions: h.server.Sessions().Count(),
		Uptime:         time.Since(h.startTime).Truncate(time.Second).String(),
		Transport:      "webtransport",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Handler returns an http.Handler for health checks.
func (h *HealthCheck) Handler() http.Handler {
	return h
}

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

// ErrorResponse is the JSON body returned when WebTransport upgrade fails.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// ErrorPageHandler is called when a non-WebTransport HTTP request hits a
// WebTransport route. Customize this to return helpful error messages.
type ErrorPageHandler func(w http.ResponseWriter, r *http.Request, code int, msg string)

// DefaultErrorPage returns a JSON error response.
func DefaultErrorPage(w http.ResponseWriter, _ *http.Request, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Code:    code,
		Message: msg,
	})
}

// HTMLErrorPage returns an HTML error page.
func HTMLErrorPage(w http.ResponseWriter, _ *http.Request, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(`<!DOCTYPE html><html><head><title>Error</title>
<style>body{font-family:monospace;background:#0d1117;color:#c9d1d9;display:flex;justify-content:center;align-items:center;height:100vh}
.box{text-align:center;}.code{font-size:48px;color:#f85149}.msg{margin-top:8px;color:#8b949e}</style>
</head><body><div class="box"><div class="code">` + http.StatusText(code) + `</div><div class="msg">` + msg + `</div></div></body></html>`))
}

// FlowControlMonitor tracks stream and datagram flow control metrics.
// Useful for monitoring backpressure and identifying slow consumers.
type FlowControlMonitor struct {
	StreamsOpened  atomic.Int64
	StreamsClosed  atomic.Int64
	DatagramsSent  atomic.Int64
	DatagramsRecvd atomic.Int64
	BytesSent      atomic.Int64
	BytesReceived  atomic.Int64
	WriteBlocks    atomic.Int64 // times a write was blocked by flow control
}

// NewFlowControlMonitor creates a new monitor.
func NewFlowControlMonitor() *FlowControlMonitor {
	return &FlowControlMonitor{}
}

// FlowStats returns a snapshot of flow control metrics.
type FlowStats struct {
	StreamsOpened  int64 `json:"streams_opened"`
	StreamsClosed  int64 `json:"streams_closed"`
	StreamsActive  int64 `json:"streams_active"`
	DatagramsSent  int64 `json:"datagrams_sent"`
	DatagramsRecvd int64 `json:"datagrams_received"`
	BytesSent      int64 `json:"bytes_sent"`
	BytesReceived  int64 `json:"bytes_received"`
	WriteBlocks    int64 `json:"write_blocks"`
}

// Stats returns current metrics.
func (fc *FlowControlMonitor) Stats() FlowStats {
	opened := fc.StreamsOpened.Load()
	closed := fc.StreamsClosed.Load()
	return FlowStats{
		StreamsOpened:  opened,
		StreamsClosed:  closed,
		StreamsActive:  opened - closed,
		DatagramsSent:  fc.DatagramsSent.Load(),
		DatagramsRecvd: fc.DatagramsRecvd.Load(),
		BytesSent:      fc.BytesSent.Load(),
		BytesReceived:  fc.BytesReceived.Load(),
		WriteBlocks:    fc.WriteBlocks.Load(),
	}
}
