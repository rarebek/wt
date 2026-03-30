package middleware

import (
	"os"

	"github.com/rarebek/wt"
)

// Env injects environment variables into the session context.
// Useful for feature flags, deployment environment detection, etc.
//
// Usage:
//
//	server.Use(middleware.Env("APP_ENV", "FEATURE_FLAGS", "REGION"))
func Env(keys ...string) wt.MiddlewareFunc {
	// Read env vars once at middleware creation (not per-session)
	vals := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			vals[k] = v
		}
	}

	return func(c *wt.Context, next wt.HandlerFunc) {
		for k, v := range vals {
			c.Set("env_"+k, v)
		}
		next(c)
	}
}

// GetEnv retrieves an environment variable from context.
func GetEnv(c *wt.Context, key string) string {
	return c.GetString("env_" + key)
}
