package middleware

import (
	"time"

	"github.com/rarebek/wt"
)

// startupTime records when the server started.
var startupTime = time.Now()

// Uptime returns middleware that stores the server uptime in context.
func Uptime() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("uptime", time.Since(startupTime))
		c.Set("started_at", startupTime)
		next(c)
	}
}

// GetUptime returns the server uptime from context.
func GetUptime(c *wt.Context) time.Duration {
	v, ok := c.Get("uptime")
	if !ok {
		return time.Since(startupTime)
	}
	d, _ := v.(time.Duration)
	return d
}
