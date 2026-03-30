package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// GlobalRateLimit limits the total number of sessions across ALL clients.
// Simpler than per-IP rate limiting — just a global atomic counter.
// Use for API gateways or services with known capacity limits.
func GlobalRateLimit(max int64) wt.MiddlewareFunc {
	var active atomic.Int64
	return func(c *wt.Context, next wt.HandlerFunc) {
		cur := active.Add(1)
		if cur > max {
			active.Add(-1)
			_ = c.CloseWithError(503, "server at capacity")
			return
		}
		defer active.Add(-1)
		next(c)
	}
}
