package middleware

import (
	"log/slog"
	"sync/atomic"

	"github.com/rarebek/wt"
)

// AutoReconnectStats tracks client reconnection patterns.
// Attach to sessions to monitor connection stability.
type AutoReconnectStats struct {
	Connections    atomic.Int64
	Disconnections atomic.Int64
	Reconnections  atomic.Int64
}

// NewAutoReconnectStats creates a reconnection tracker.
func NewAutoReconnectStats() *AutoReconnectStats {
	return &AutoReconnectStats{}
}

// ConnectionTracker returns middleware that tracks connection/disconnection events.
func (ars *AutoReconnectStats) ConnectionTracker(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		ars.Connections.Add(1)
		logger.Debug("session connected",
			"session", c.ID(),
			"total_connections", ars.Connections.Load(),
		)

		defer func() {
			ars.Disconnections.Add(1)
			logger.Debug("session disconnected",
				"session", c.ID(),
				"total_disconnections", ars.Disconnections.Load(),
			)
		}()

		next(c)
	}
}

// Stats returns current connection stats.
func (ars *AutoReconnectStats) Stats() (conns, disconns, reconns int64) {
	return ars.Connections.Load(), ars.Disconnections.Load(), ars.Reconnections.Load()
}
