package middleware

import (
	"log/slog"
	"sync/atomic"

	"github.com/rarebek/wt"
)

// MaxSessions returns middleware that limits the total number of concurrent sessions.
func MaxSessions(max int, logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	var active atomic.Int64

	return func(c *wt.Context, next wt.HandlerFunc) {
		current := active.Add(1)
		defer active.Add(-1)

		if current > int64(max) {
			logger.Warn("max sessions exceeded",
				"current", current,
				"max", max,
				"remote", c.RemoteAddr().String(),
			)
			_ = c.CloseWithError(503, "server at capacity")
			return
		}

		next(c)
	}
}
