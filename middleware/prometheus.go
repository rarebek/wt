package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
)

// PrometheusMetrics exports metrics in Prometheus text format.
// Serve the Handler() on an HTTP endpoint for Prometheus scraping.
type PrometheusMetrics struct {
	activeSessions   atomic.Int64
	totalSessions    atomic.Int64
	totalStreams      atomic.Int64
	totalDatagrams   atomic.Int64
	sessionDurations atomic.Int64 // total nanoseconds

	// Histogram buckets for session duration
	histMu   sync.Mutex
	histBins [8]int64 // <1s, <5s, <30s, <1m, <5m, <30m, <1h, >=1h
}

// NewPrometheusMetrics creates a new Prometheus-compatible metrics collector.
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{}
}

// Middleware returns wt middleware that tracks session metrics.
func (pm *PrometheusMetrics) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		pm.activeSessions.Add(1)
		pm.totalSessions.Add(1)
		start := time.Now()

		defer func() {
			pm.activeSessions.Add(-1)
			dur := time.Since(start)
			pm.sessionDurations.Add(int64(dur))
			pm.recordHistogram(dur)
		}()

		next(c)
	}
}

// Handler returns an HTTP handler that serves Prometheus metrics.
//
// Usage:
//
//	pm := middleware.NewPrometheusMetrics()
//	server.Use(pm.Middleware())
//	http.Handle("/metrics", pm.Handler())
func (pm *PrometheusMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		active := pm.activeSessions.Load()
		total := pm.totalSessions.Load()
		durNs := pm.sessionDurations.Load()
		durSec := float64(durNs) / 1e9

		fmt.Fprintf(w, "# HELP wt_active_sessions Number of active WebTransport sessions\n")
		fmt.Fprintf(w, "# TYPE wt_active_sessions gauge\n")
		fmt.Fprintf(w, "wt_active_sessions %d\n", active)
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "# HELP wt_sessions_total Total number of WebTransport sessions\n")
		fmt.Fprintf(w, "# TYPE wt_sessions_total counter\n")
		fmt.Fprintf(w, "wt_sessions_total %d\n", total)
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "# HELP wt_session_duration_seconds_total Total session duration in seconds\n")
		fmt.Fprintf(w, "# TYPE wt_session_duration_seconds_total counter\n")
		fmt.Fprintf(w, "wt_session_duration_seconds_total %.6f\n", durSec)
		fmt.Fprintf(w, "\n")

		if total > 0 {
			avgDur := durSec / float64(total)
			fmt.Fprintf(w, "# HELP wt_session_duration_seconds_avg Average session duration in seconds\n")
			fmt.Fprintf(w, "# TYPE wt_session_duration_seconds_avg gauge\n")
			fmt.Fprintf(w, "wt_session_duration_seconds_avg %.6f\n", avgDur)
		}

		// Session duration histogram
		pm.histMu.Lock()
		bins := pm.histBins
		pm.histMu.Unlock()

		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "# HELP wt_session_duration_seconds Session duration histogram\n")
		fmt.Fprintf(w, "# TYPE wt_session_duration_seconds histogram\n")
		buckets := []struct{ le string; idx int }{
			{"1", 0}, {"5", 1}, {"30", 2}, {"60", 3},
			{"300", 4}, {"1800", 5}, {"3600", 6},
		}
		var cumulative int64
		for _, b := range buckets {
			cumulative += bins[b.idx]
			fmt.Fprintf(w, "wt_session_duration_seconds_bucket{le=\"%s\"} %d\n", b.le, cumulative)
		}
		cumulative += bins[7]
		fmt.Fprintf(w, "wt_session_duration_seconds_bucket{le=\"+Inf\"} %d\n", cumulative)
		fmt.Fprintf(w, "wt_session_duration_seconds_count %d\n", total)
		fmt.Fprintf(w, "wt_session_duration_seconds_sum %.6f\n", durSec)
	})
}

func (pm *PrometheusMetrics) recordHistogram(d time.Duration) {
	idx := 7 // >=1h
	switch {
	case d < time.Second:
		idx = 0
	case d < 5*time.Second:
		idx = 1
	case d < 30*time.Second:
		idx = 2
	case d < time.Minute:
		idx = 3
	case d < 5*time.Minute:
		idx = 4
	case d < 30*time.Minute:
		idx = 5
	case d < time.Hour:
		idx = 6
	}
	pm.histMu.Lock()
	pm.histBins[idx]++
	pm.histMu.Unlock()
}
