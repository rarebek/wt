// Package middleware provides built-in middleware for the wt framework.
package middleware

import (
	"log/slog"
	"time"

	"github.com/rarebek/wt"
)

// Logger returns middleware that logs session lifecycle events.
func Logger(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		start := time.Now()
		logger.Info("session opened",
			"id", c.ID(),
			"remote", c.RemoteAddr().String(),
			"path", c.Request().URL.Path,
			"params", c.Params(),
		)

		next(c)

		logger.Info("session closed",
			"id", c.ID(),
			"remote", c.RemoteAddr().String(),
			"duration", time.Since(start).String(),
		)
	}
}

// DefaultLogger returns a Logger middleware using slog.Default().
func DefaultLogger() wt.MiddlewareFunc {
	return Logger(nil)
}
