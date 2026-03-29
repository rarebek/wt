package middleware

import (
	"sync/atomic"

	"github.com/rarebek/wt"
)

// ConnectionID assigns a monotonically increasing ID to each session.
// Simpler than cryptographic session IDs — just a counter.
// Useful for logging and debugging.
var connectionCounter atomic.Uint64

// ConnectionID returns middleware that assigns a numeric connection ID.
func ConnectionID() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		id := connectionCounter.Add(1)
		c.Set("connection_id", id)
		next(c)
	}
}

// GetConnectionID retrieves the numeric connection ID.
func GetConnectionID(c *wt.Context) uint64 {
	v, ok := c.Get("connection_id")
	if !ok {
		return 0
	}
	id, _ := v.(uint64)
	return id
}
