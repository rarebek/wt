package middleware

import (
	"github.com/rarebek/wt"
)

// SessionData returns middleware that initializes common session metadata
// from the HTTP handshake request (user agent, origin, query params).
// This saves handlers from having to extract these individually.
func SessionData() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		r := c.Request()

		// Extract common data from handshake
		if ua := r.UserAgent(); ua != "" {
			c.Set("user_agent", ua)
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			c.Set("origin", origin)
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			c.Set("forwarded_for", xff)
		}

		// Store all query parameters
		for key, values := range r.URL.Query() {
			if len(values) == 1 {
				c.Set("query_"+key, values[0])
			} else {
				c.Set("query_"+key, values)
			}
		}

		next(c)
	}
}
