package middleware

import (
	"log/slog"

	"github.com/rarebek/wt"
)

// SlogAttrs returns middleware that adds common session attributes
// to the slog default logger for all log calls within the handler.
// This means any slog.Info/Warn/Error call inside the handler will
// automatically include session_id, remote_addr, and path.
func SlogAttrs() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		logger := slog.Default().With(
			"session_id", c.ID(),
			"remote_addr", c.RemoteAddr().String(),
			"path", c.Request().URL.Path,
		)
		c.Set("_logger", logger)
		next(c)
	}
}

// GetLogger retrieves the session-scoped logger from the context.
// Returns slog.Default() if SlogAttrs middleware wasn't applied.
func GetLogger(c *wt.Context) *slog.Logger {
	v, ok := c.Get("_logger")
	if !ok {
		return slog.Default()
	}
	logger, _ := v.(*slog.Logger)
	if logger == nil {
		return slog.Default()
	}
	return logger
}
