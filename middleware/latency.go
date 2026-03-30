package middleware

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
)

// LatencyTracker records per-session connection setup latency.
// Measures time from middleware entry (session established) to first stream accept.
type LatencyTracker struct {
	mu       sync.Mutex
	samples  []time.Duration
	totalNs  atomic.Int64
	count    atomic.Int64
}

// NewLatencyTracker creates a new latency tracker.
func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{}
}

// Middleware returns wt middleware that records connection start time.
func (lt *LatencyTracker) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_connect_time", time.Now())
		next(c)
	}
}

// RecordFirstStream should be called when the first stream is accepted.
// It calculates the time from connection to first stream.
func (lt *LatencyTracker) RecordFirstStream(c *wt.Context) {
	v, ok := c.Get("_connect_time")
	if !ok {
		return
	}
	connectTime := v.(time.Time)
	latency := time.Since(connectTime)

	lt.totalNs.Add(int64(latency))
	lt.count.Add(1)

	lt.mu.Lock()
	lt.samples = append(lt.samples, latency)
	lt.mu.Unlock()
}

// Average returns the average first-stream latency.
func (lt *LatencyTracker) Average() time.Duration {
	c := lt.count.Load()
	if c == 0 {
		return 0
	}
	return time.Duration(lt.totalNs.Load() / c)
}

// Count returns the number of samples recorded.
func (lt *LatencyTracker) Count() int64 {
	return lt.count.Load()
}

// Percentile returns the p-th percentile latency (0-100).
// Requires at least 1 sample.
func (lt *LatencyTracker) Percentile(p int) time.Duration {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.samples) == 0 {
		return 0
	}

	idx := (p * len(lt.samples)) / 100
	if idx >= len(lt.samples) {
		idx = len(lt.samples) - 1
	}
	return lt.samples[idx]
}
