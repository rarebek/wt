package wt

import (
	"encoding/json"
	"runtime"
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
