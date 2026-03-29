package middleware

import (
	"github.com/rarebek/wt"
)

// ProtocolVersion stores the client's protocol version in context.
// Clients send their version via the `X-WT-Version` header or `wt-version` query param.
// Handlers can check this to serve version-appropriate responses.
func ProtocolVersion(currentVersion string) wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		clientVersion := c.Request().Header.Get("X-WT-Version")
		if clientVersion == "" {
			clientVersion = c.Request().URL.Query().Get("wt-version")
		}
		if clientVersion == "" {
			clientVersion = currentVersion // assume current
		}

		c.Set("protocol_version", clientVersion)
		c.Set("server_version", currentVersion)
		next(c)
	}
}

// GetProtocolVersion retrieves the negotiated protocol version.
func GetProtocolVersion(c *wt.Context) string {
	return c.GetString("protocol_version")
}

// GetServerVersion retrieves the server's protocol version.
func GetServerVersion(c *wt.Context) string {
	return c.GetString("server_version")
}
