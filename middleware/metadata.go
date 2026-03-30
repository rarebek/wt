package middleware

import (
	"runtime"

	"github.com/rarebek/wt"
)

// ServerMetadata stores server metadata in context for handlers that
// need to report version/platform info to clients.
func ServerMetadata(version string) wt.MiddlewareFunc {
	meta := map[string]string{
		"version":  version,
		"runtime":  runtime.Version(),
		"platform": runtime.GOOS + "/" + runtime.GOARCH,
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("server_meta", meta)
		next(c)
	}
}

// GetServerMetadata retrieves server metadata from context.
func GetServerMetadata(c *wt.Context) map[string]string {
	v, ok := c.Get("server_meta")
	if !ok {
		return nil
	}
	m, _ := v.(map[string]string)
	return m
}
