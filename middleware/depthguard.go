package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// DepthGuard limits the number of concurrent stream handlers per session.
// Prevents a single client from opening too many streams and overwhelming
// the server's goroutine pool.
//
// Usage:
//
//	server.Handle("/app", handler, middleware.DepthGuard(50))
//	// Each session limited to 50 concurrent stream handlers
func DepthGuard(maxConcurrent int) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		counter := &atomic.Int64{}
		c.Set("_depth_guard", counter)
		c.Set("_depth_max", maxConcurrent)
		next(c)
	}
}

// CheckDepth should be called before spawning a new stream handler goroutine.
// Returns true if under the limit, false if the limit is reached.
func CheckDepth(c *wt.Context) bool {
	v, ok := c.Get("_depth_guard")
	if !ok {
		return true // no guard installed
	}
	counter := v.(*atomic.Int64)
	max, _ := c.Get("_depth_max")
	maxInt := max.(int)

	current := counter.Add(1)
	if current > int64(maxInt) {
		counter.Add(-1)
		return false
	}
	return true
}

// ReleaseDepth should be called when a stream handler goroutine completes.
func ReleaseDepth(c *wt.Context) {
	v, ok := c.Get("_depth_guard")
	if !ok {
		return
	}
	counter := v.(*atomic.Int64)
	counter.Add(-1)
}
