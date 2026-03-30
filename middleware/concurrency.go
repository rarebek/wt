package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// ConcurrencyStats tracks global server concurrency metrics in real-time.
// Lighter than PrometheusMetrics — just atomic counters, no HTTP handler.
type ConcurrencyStats struct {
	ActiveSessions atomic.Int64
	PeakSessions   atomic.Int64
	TotalAccepted  atomic.Int64
	TotalRejected  atomic.Int64
}

// NewConcurrencyStats creates a new concurrency tracker.
func NewConcurrencyStats() *ConcurrencyStats {
	return &ConcurrencyStats{}
}

// Middleware returns wt middleware that tracks concurrency.
func (cs *ConcurrencyStats) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		cur := cs.ActiveSessions.Add(1)
		cs.TotalAccepted.Add(1)

		// Update peak
		for {
			peak := cs.PeakSessions.Load()
			if cur <= peak || cs.PeakSessions.CompareAndSwap(peak, cur) {
				break
			}
		}

		defer cs.ActiveSessions.Add(-1)
		next(c)
	}
}

// Snapshot returns current stats.
func (cs *ConcurrencyStats) Snapshot() struct {
	Active   int64
	Peak     int64
	Accepted int64
	Rejected int64
} {
	return struct {
		Active   int64
		Peak     int64
		Accepted int64
		Rejected int64
	}{
		Active:   cs.ActiveSessions.Load(),
		Peak:     cs.PeakSessions.Load(),
		Accepted: cs.TotalAccepted.Load(),
		Rejected: cs.TotalRejected.Load(),
	}
}
