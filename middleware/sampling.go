package middleware

import (
	"log/slog"
	"math/rand/v2"

	"github.com/rarebek/wt"
)

// SampledLogger returns middleware that only logs a fraction of sessions.
// rate is 0.0-1.0 (e.g., 0.01 = log 1% of sessions).
// Useful for high-traffic servers where logging everything is too expensive.
func SampledLogger(rate float64, logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		sampled := rand.Float64() < rate
		if sampled {
			c.Set("_sampled", true)
			logger.Info("session sampled",
				"session", c.ID(),
				"remote", c.RemoteAddr().String(),
				"path", c.Request().URL.Path,
			)
		}
		next(c)
		if sampled {
			logger.Info("sampled session ended", "session", c.ID())
		}
	}
}

// IsSampled checks if the current session was selected for sampling.
func IsSampled(c *wt.Context) bool {
	v, ok := c.Get("_sampled")
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}
