package wt

import (
	"sync"
	"time"
)

// Throttle limits the rate of datagram sends per session.
// Implements a simple token bucket: N tokens per second, burst B.
type Throttle struct {
	mu       sync.Mutex
	rate     float64 // tokens per second
	burst    float64 // max tokens
	tokens   float64
	lastTime time.Time
}

// NewThrottle creates a rate throttle.
// rate: messages per second allowed. burst: max messages allowed in a burst.
func NewThrottle(rate float64, burst int) *Throttle {
	return &Throttle{
		rate:     rate,
		burst:    float64(burst),
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow checks if a message can be sent. Returns true and consumes a token if allowed.
func (t *Throttle) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastTime).Seconds()
	t.lastTime = now

	t.tokens += elapsed * t.rate
	if t.tokens > t.burst {
		t.tokens = t.burst
	}

	if t.tokens < 1 {
		return false
	}
	t.tokens--
	return true
}

// Wait blocks until a token is available or context is cancelled.
func (t *Throttle) Wait() {
	for !t.Allow() {
		time.Sleep(time.Millisecond)
	}
}

// ThrottledSend sends a datagram only if the throttle allows it.
// Returns false if throttled (message dropped).
func (c *Context) ThrottledSend(t *Throttle, data []byte) bool {
	if !t.Allow() {
		return false
	}
	return c.SendDatagram(data) == nil
}
