package middleware

import (
	"strings"

	"github.com/rarebek/wt"
)

// BlockUserAgent returns middleware that rejects connections from
// clients with specific User-Agent strings. Useful for blocking
// known bad bots or specific client versions.
func BlockUserAgent(blocked ...string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ua := c.Request().UserAgent()
		for _, b := range blocked {
			if strings.Contains(ua, b) {
				_ = c.CloseWithError(403, "blocked user agent")
				return
			}
		}
		next(c)
	}
}

// RequireUserAgent returns middleware that rejects connections
// without a User-Agent header.
func RequireUserAgent() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		if c.Request().UserAgent() == "" {
			_ = c.CloseWithError(400, "user-agent required")
			return
		}
		next(c)
	}
}
