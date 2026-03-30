package middleware

import (
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
)

// GracePeriod tracks whether any session is in a "critical section"
// that should delay shutdown. Use with Server.OnShutdown to wait
// for critical operations to complete.
type GracePeriod struct {
	active atomic.Int64
}

// NewGracePeriod creates a grace period tracker.
func NewGracePeriod() *GracePeriod {
	return &GracePeriod{}
}

// Enter marks the start of a critical operation.
// Call Leave when done.
func (gp *GracePeriod) Enter() {
	gp.active.Add(1)
}

// Leave marks the end of a critical operation.
func (gp *GracePeriod) Leave() {
	gp.active.Add(-1)
}

// Active returns the number of ongoing critical operations.
func (gp *GracePeriod) Active() int64 {
	return gp.active.Load()
}

// WaitDrain blocks until all critical operations complete or timeout.
func (gp *GracePeriod) WaitDrain(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if gp.active.Load() == 0 {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return gp.active.Load() == 0
}

// Middleware stores the GracePeriod in context for handler access.
func (gp *GracePeriod) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		c.Set("_grace_period", gp)
		next(c)
	}
}

// GetGracePeriod retrieves the GracePeriod from context.
func GetGracePeriod(c *wt.Context) *GracePeriod {
	v, ok := c.Get("_grace_period")
	if !ok {
		return nil
	}
	gp, _ := v.(*GracePeriod)
	return gp
}
