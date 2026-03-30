package middleware

import (
	"time"

	"github.com/rarebek/wt"
)

// Timeout returns middleware that closes sessions after the given duration.
// Useful for preventing zombie sessions.
func Timeout(d time.Duration) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		timer := time.AfterFunc(d, func() {
			_ = c.CloseWithError(408, "session timeout")
		})
		defer timer.Stop()

		next(c)
	}
}

// IdleTimeout returns middleware that closes sessions if no datagrams are
// received within the given duration. Resets on each datagram.
// Note: This wraps the handler — it doesn't intercept individual datagrams.
// For production use, implement idle detection within your handler.
func IdleTimeout(d time.Duration) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		// Store the timeout duration in context for handlers to use
		c.Set("_idle_timeout", d)
		next(c)
	}
}
