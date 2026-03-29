package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net"

	"github.com/rarebek/wt"
)

// IPHash stores a hashed version of the client IP in context.
// Useful for analytics without storing raw IPs (GDPR compliance).
func IPHash(salt string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		ip := extractIP(c.RemoteAddr().String())
		if ip != nil {
			hash := sha256.Sum256([]byte(salt + ip.String()))
			c.Set("ip_hash", hex.EncodeToString(hash[:8])) // first 8 bytes = 16 hex chars
		}
		next(c)
	}
}

// GetIPHash retrieves the hashed IP from context.
func GetIPHash(c *wt.Context) string {
	return c.GetString("ip_hash")
}

func extractIPForHash(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.ParseIP(addr)
	}
	return net.ParseIP(host)
}
