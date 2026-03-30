package middleware

import (
	"log/slog"

	"github.com/rarebek/wt"
)

// Recover returns middleware that recovers from panics in session handlers.
func Recover(logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in session handler",
					"id", c.ID(),
					"remote", c.RemoteAddr().String(),
					"panic", r,
				)
				_ = c.CloseWithError(500, "internal server error")
			}
		}()
		next(c)
	}
}
