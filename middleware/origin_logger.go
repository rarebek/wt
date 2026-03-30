package middleware

import (
	"log/slog"
	"sync"

	"github.com/rarebek/wt"
)

// OriginTracker tracks unique origins connecting to the server.
// Useful for monitoring which domains/apps are using your WebTransport service.
type OriginTracker struct {
	mu      sync.Mutex
	origins map[string]int64
}

// NewOriginTracker creates an origin tracker.
func NewOriginTracker() *OriginTracker {
	return &OriginTracker{origins: make(map[string]int64)}
}

// Middleware records the Origin header.
func (ot *OriginTracker) Middleware(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		origin := c.Request().Header.Get("Origin")
		if origin != "" {
			ot.mu.Lock()
			ot.origins[origin]++
			ot.mu.Unlock()
		}
		next(c)
	}
}

// Origins returns all tracked origins and their connection counts.
func (ot *OriginTracker) Origins() map[string]int64 {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	cp := make(map[string]int64, len(ot.origins))
	for k, v := range ot.origins {
		cp[k] = v
	}
	return cp
}

// UniqueCount returns the number of unique origins seen.
func (ot *OriginTracker) UniqueCount() int {
	ot.mu.Lock()
	n := len(ot.origins)
	ot.mu.Unlock()
	return n
}
