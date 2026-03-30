package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// IdleMonitor provides per-session idle detection.
// The idle timer resets every time Activity() is called.
// When the timer expires, the session is closed.
type IdleMonitor struct {
	mu       sync.Mutex
	timer    *time.Timer
	timeout  time.Duration
	ctx      *wt.Context
	onIdle   func(*wt.Context)
}

// NewIdleMonitor creates a monitor that closes the session after the given idle duration.
// Call Activity() from your handler whenever the session is active (received data, etc.).
func NewIdleMonitor(c *wt.Context, timeout time.Duration, onIdle func(*wt.Context)) *IdleMonitor {
	im := &IdleMonitor{
		timeout: timeout,
		ctx:     c,
		onIdle:  onIdle,
	}
	im.timer = time.AfterFunc(timeout, im.fire)
	return im
}

// Activity resets the idle timer.
func (im *IdleMonitor) Activity() {
	im.mu.Lock()
	im.timer.Reset(im.timeout)
	im.mu.Unlock()
}

// Stop cancels the idle monitor.
func (im *IdleMonitor) Stop() {
	im.mu.Lock()
	im.timer.Stop()
	im.mu.Unlock()
}

func (im *IdleMonitor) fire() {
	if im.onIdle != nil {
		im.onIdle(im.ctx)
	} else {
		_ = im.ctx.CloseWithError(408, "idle timeout")
	}
}
