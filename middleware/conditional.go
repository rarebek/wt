package middleware

import "github.com/rarebek/wt"

// If returns middleware that conditionally applies inner middleware.
// The condition is checked once per session. If false, inner is skipped.
//
// Usage:
//
//	server.Use(middleware.If(
//	    func(c *wt.Context) bool { return c.Request().URL.Path == "/admin" },
//	    middleware.BearerAuth(validate),
//	))
func If(condition func(*wt.Context) bool, inner wt.MiddlewareFunc) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		if condition(c) {
			inner(c, next)
		} else {
			next(c)
		}
	}
}

// Unless is the inverse of If — applies middleware when condition is false.
func Unless(condition func(*wt.Context) bool, inner wt.MiddlewareFunc) wt.MiddlewareFunc {
	return If(func(c *wt.Context) bool { return !condition(c) }, inner)
}
