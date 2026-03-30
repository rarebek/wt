package middleware

import "github.com/rarebek/wt"

// OnPanic returns middleware that calls a custom handler on panic instead
// of the default behavior. Useful for custom error reporting (Sentry, etc).
func OnPanic(handler func(c *wt.Context, recovered any)) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		defer func() {
			if r := recover(); r != nil {
				handler(c, r)
			}
		}()
		next(c)
	}
}
