package middleware

import (
	"strings"

	"github.com/rarebek/wt"
)

// BearerAuth returns middleware that validates Bearer tokens from the request header.
// The validate function receives the token and returns the user identity (stored in context as "user")
// or an error to reject the connection.
func BearerAuth(validate func(token string) (any, error)) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			_ = c.CloseWithError(401, "missing authorization header")
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			_ = c.CloseWithError(401, "invalid authorization scheme")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		user, err := validate(token)
		if err != nil {
			_ = c.CloseWithError(401, "invalid token")
			return
		}

		c.Set("user", user)
		next(c)
	}
}

// QueryAuth returns middleware that validates tokens from a query parameter.
// Useful when headers can't be set (e.g., browser WebTransport API).
func QueryAuth(param string, validate func(token string) (any, error)) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		token := c.Request().URL.Query().Get(param)
		if token == "" {
			_ = c.CloseWithError(401, "missing auth token")
			return
		}

		user, err := validate(token)
		if err != nil {
			_ = c.CloseWithError(401, "invalid token")
			return
		}

		c.Set("user", user)
		next(c)
	}
}

// RequireKey returns middleware that checks for a static API key in a header.
func RequireKey(headerName, expectedKey string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		key := c.Request().Header.Get(headerName)
		if key != expectedKey {
			_ = c.CloseWithError(401, "invalid API key")
			return
		}
		next(c)
	}
}
