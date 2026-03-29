package middleware

import (
	"sync"

	"github.com/rarebek/wt"
)

// Once returns middleware that runs a function exactly once across all sessions.
// Useful for one-time initialization that depends on the first connection.
func Once(fn func(*wt.Context)) wt.MiddlewareFunc {
	var once sync.Once
	return func(c *wt.Context, next wt.HandlerFunc) {
		once.Do(func() { fn(c) })
		next(c)
	}
}
