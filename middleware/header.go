package middleware

import (
	"github.com/rarebek/wt"
)

// ExtractHeader extracts a specific HTTP header from the WebTransport
// handshake request and stores it in the session context.
// Useful for extracting custom headers like X-Request-ID, X-Forwarded-For, etc.
func ExtractHeader(headerName, contextKey string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		value := c.Request().Header.Get(headerName)
		if value != "" {
			c.Set(contextKey, value)
		}
		next(c)
	}
}

// ExtractHeaders extracts multiple headers into context.
// keys maps header name → context key.
func ExtractHeaders(keys map[string]string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		for header, ctxKey := range keys {
			value := c.Request().Header.Get(header)
			if value != "" {
				c.Set(ctxKey, value)
			}
		}
		next(c)
	}
}
