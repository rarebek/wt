package middleware

import (
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
)

// HandlerTiming tracks how long session handlers run.
// Records min, max, and average handler execution time.
type HandlerTiming struct {
	totalNs atomic.Int64
	count   atomic.Int64
	minNs   atomic.Int64
	maxNs   atomic.Int64
}

// NewHandlerTiming creates a handler timing tracker.
func NewHandlerTiming() *HandlerTiming {
	ht := &HandlerTiming{}
	ht.minNs.Store(1<<63 - 1) // max int64
	return ht
}

// Middleware returns wt middleware that tracks handler duration.
func (ht *HandlerTiming) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		start := time.Now()
		next(c)
		ns := int64(time.Since(start))

		ht.totalNs.Add(ns)
		ht.count.Add(1)

		// Update min
		for {
			old := ht.minNs.Load()
			if ns >= old || ht.minNs.CompareAndSwap(old, ns) {
				break
			}
		}
		// Update max
		for {
			old := ht.maxNs.Load()
			if ns <= old || ht.maxNs.CompareAndSwap(old, ns) {
				break
			}
		}
	}
}

// Stats returns min, max, and average handler duration.
func (ht *HandlerTiming) Stats() (min, max, avg time.Duration) {
	c := ht.count.Load()
	if c == 0 {
		return 0, 0, 0
	}
	return time.Duration(ht.minNs.Load()),
		time.Duration(ht.maxNs.Load()),
		time.Duration(ht.totalNs.Load() / c)
}

// Count returns the number of handlers timed.
func (ht *HandlerTiming) Count() int64 {
	return ht.count.Load()
}
