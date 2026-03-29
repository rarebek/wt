package middleware

import (
	"github.com/rarebek/wt"
)

// MaxMessageSize stores a custom max message size in the session context.
// Handlers can check this before reading large messages.
func MaxMessageSize(maxBytes int) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_max_message_size", maxBytes)
		next(c)
	}
}

// GetMaxMessageSize retrieves the configured max message size.
// Returns wt.MaxMessageSize (16MB) if middleware wasn't applied.
func GetMaxMessageSize(c *wt.Context) int {
	v, ok := c.Get("_max_message_size")
	if !ok {
		return wt.MaxMessageSize
	}
	n, _ := v.(int)
	if n <= 0 {
		return wt.MaxMessageSize
	}
	return n
}
