package middleware

import "github.com/rarebek/wt"

// Compose combines multiple middleware into a single middleware function.
// Useful for creating reusable middleware stacks.
//
// Usage:
//
//	production := middleware.Compose(
//	    middleware.Recover(nil),
//	    middleware.DefaultLogger(),
//	    middleware.RateLimit(100),
//	)
//	server.Use(production)
func Compose(mws ...wt.MiddlewareFunc) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		// Build chain from last to first
		h := next
		for i := len(mws) - 1; i >= 0; i-- {
			m := mws[i]
			n := h
			h = func(c *wt.Context) { m(c, n) }
		}
		h(c)
	}
}
