package middleware

import (
	"sync"

	"github.com/rarebek/wt"
)

// FeatureFlags provides runtime-toggleable feature flags.
// Check flags in handlers to enable/disable features without restart.
type FeatureFlags struct {
	mu    sync.RWMutex
	flags map[string]bool
}

// NewFeatureFlags creates a feature flag store.
func NewFeatureFlags() *FeatureFlags {
	return &FeatureFlags{flags: make(map[string]bool)}
}

// Set enables or disables a flag.
func (ff *FeatureFlags) Set(flag string, enabled bool) {
	ff.mu.Lock()
	ff.flags[flag] = enabled
	ff.mu.Unlock()
}

// Enabled checks if a flag is enabled.
func (ff *FeatureFlags) Enabled(flag string) bool {
	ff.mu.RLock()
	v := ff.flags[flag]
	ff.mu.RUnlock()
	return v
}

// Middleware stores feature flags in every session context.
func (ff *FeatureFlags) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_feature_flags", ff)
		next(c)
	}
}

// GetFeatureFlags retrieves feature flags from context.
func GetFeatureFlags(c *wt.Context) *FeatureFlags {
	v, ok := c.Get("_feature_flags")
	if !ok {
		return nil
	}
	ff, _ := v.(*FeatureFlags)
	return ff
}

// IsEnabled checks a feature flag from context.
func IsEnabled(c *wt.Context, flag string) bool {
	ff := GetFeatureFlags(c)
	if ff == nil {
		return false
	}
	return ff.Enabled(flag)
}
