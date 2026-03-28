package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/rarebek/wt"
)

// RequestID assigns a unique request ID to each session and stores it in context.
// Useful for log correlation across distributed systems.
func RequestID() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		id := generateRequestID()
		c.Set("request_id", id)
		next(c)
	}
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(c *wt.Context) string {
	return c.GetString("request_id")
}

func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
