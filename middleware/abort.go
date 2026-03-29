package middleware

import "github.com/rarebek/wt"

// AbortIf returns middleware that immediately closes the session
// if a condition is true. Useful for maintenance mode or kill switches.
//
// Usage:
//
//	var maintenance atomic.Bool
//	server.Use(middleware.AbortIf(
//	    func(c *wt.Context) bool { return maintenance.Load() },
//	    503, "server under maintenance",
//	))
func AbortIf(condition func(*wt.Context) bool, code uint32, msg string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		if condition(c) {
			_ = c.CloseWithError(code, msg)
			return
		}
		next(c)
	}
}
