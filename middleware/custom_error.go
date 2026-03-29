package middleware

import "github.com/rarebek/wt"

// ErrorHandler defines a custom function to handle errors from closing sessions.
// Use this to log, report to Sentry, or send alerts when sessions close with errors.
type ErrorHandler func(c *wt.Context, code uint32, msg string)

// CustomErrorHandler returns middleware that intercepts session close errors.
func CustomErrorHandler(handler ErrorHandler) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_error_handler", handler)
		next(c)
	}
}

// ReportError triggers the custom error handler if configured.
func ReportError(c *wt.Context, code uint32, msg string) {
	v, ok := c.Get("_error_handler")
	if !ok {
		return
	}
	if handler, ok := v.(ErrorHandler); ok {
		handler(c, code, msg)
	}
}
