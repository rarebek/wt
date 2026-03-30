package middleware

import (
	"github.com/rarebek/wt"
)

// CORSConfig configures origin validation.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins.
	// Use "*" to allow all origins (not recommended for production).
	AllowedOrigins []string
}

// CORS returns middleware that validates the Origin header on the WebTransport handshake.
func CORS(config CORSConfig) wt.MiddlewareFunc {
	allowed := make(map[string]bool, len(config.AllowedOrigins))
	allowAll := false
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = true
	}

	return func(c *wt.Context, next wt.HandlerFunc) {
		origin := c.Request().Header.Get("Origin")

		if !allowAll && origin != "" && !allowed[origin] {
			_ = c.CloseWithError(403, "origin not allowed")
			return
		}

		next(c)
	}
}
