package middleware

import (
	"sync"

	"github.com/rarebek/wt"
)

// RouteRateLimit returns middleware that limits concurrent sessions per route path.
// Different from RateLimit (which limits per IP), this limits by the route pattern.
//
// Usage:
//
//	server.Handle("/heavy/{id}", handler, middleware.RouteRateLimit(10))
//	// Only 10 concurrent sessions allowed on /heavy/*
func RouteRateLimit(maxConcurrent int) wt.MiddlewareFunc {
	var mu sync.Mutex
	count := 0

	return func(c *wt.Context, next wt.HandlerFunc) {
		mu.Lock()
		if count >= maxConcurrent {
			mu.Unlock()
			_ = c.CloseWithError(429, "route at capacity")
			return
		}
		count++
		mu.Unlock()

		defer func() {
			mu.Lock()
			count--
			mu.Unlock()
		}()

		next(c)
	}
}

// PerPathRateLimit creates a rate limiter that tracks limits per unique path.
// Useful when applied globally but wanting different limits per resolved route.
//
// Usage:
//
//	limiter := middleware.PerPathRateLimit(50)
//	server.Use(limiter)
//	// Each unique path (e.g., /chat/room1, /chat/room2) gets up to 50 concurrent sessions
func PerPathRateLimit(maxPerPath int) wt.MiddlewareFunc {
	var mu sync.Mutex
	paths := make(map[string]int)

	return func(c *wt.Context, next wt.HandlerFunc) {
		path := c.Request().URL.Path

		mu.Lock()
		if paths[path] >= maxPerPath {
			mu.Unlock()
			_ = c.CloseWithError(429, "path at capacity")
			return
		}
		paths[path]++
		mu.Unlock()

		defer func() {
			mu.Lock()
			paths[path]--
			if paths[path] <= 0 {
				delete(paths, path)
			}
			mu.Unlock()
		}()

		next(c)
	}
}
