package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// DurationHistogram tracks session durations in fixed buckets.
// Lighter than PrometheusMetrics — just buckets, no HTTP endpoint.
type DurationHistogram struct {
	mu      sync.Mutex
	buckets [6]int64 // <1s, <10s, <1m, <10m, <1h, >=1h
	total   int64
}

// NewDurationHistogram creates a duration tracker.
func NewDurationHistogram() *DurationHistogram {
	return &DurationHistogram{}
}

// Middleware returns wt middleware that records session durations.
func (dh *DurationHistogram) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		start := time.Now()
		next(c)
		dh.Record(time.Since(start))
	}
}

// Record adds a duration to the histogram.
func (dh *DurationHistogram) Record(d time.Duration) {
	idx := 5
	switch {
	case d < time.Second:
		idx = 0
	case d < 10*time.Second:
		idx = 1
	case d < time.Minute:
		idx = 2
	case d < 10*time.Minute:
		idx = 3
	case d < time.Hour:
		idx = 4
	}
	dh.mu.Lock()
	dh.buckets[idx]++
	dh.total++
	dh.mu.Unlock()
}

// Snapshot returns bucket counts.
func (dh *DurationHistogram) Snapshot() [6]int64 {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	return dh.buckets
}

// Total returns total recorded sessions.
func (dh *DurationHistogram) Total() int64 {
	dh.mu.Lock()
	defer dh.mu.Unlock()
	return dh.total
}
