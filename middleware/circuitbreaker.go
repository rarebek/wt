package middleware

import (
	"sync"
	"time"

	"github.com/rarebek/wt"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Rejecting connections
	CircuitHalfOpen                     // Testing with limited connections
)

// CircuitBreaker implements the circuit breaker pattern for WebTransport sessions.
// When too many sessions fail (panic/error), the breaker opens and rejects new connections.
// After a cooldown period, it enters half-open state and allows limited connections to test recovery.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	threshold    int           // failures before opening
	cooldown     time.Duration // time to wait before half-open
	lastFailure  time.Time
	halfOpenMax  int           // max concurrent in half-open state
	halfOpenCurr int
}

// NewCircuitBreaker creates a circuit breaker.
// threshold: number of failures before opening.
// cooldown: time to wait in open state before testing.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:   threshold,
		cooldown:    cooldown,
		halfOpenMax: 1,
	}
}

// Middleware returns wt middleware that applies the circuit breaker.
func (cb *CircuitBreaker) Middleware() wt.MiddlewareFunc {
	return func(c *wt.Context, next wt.HandlerFunc) {
		if !cb.allow() {
			_ = c.CloseWithError(503, "service unavailable (circuit open)")
			return
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					cb.recordFailure()
					_ = c.CloseWithError(500, "internal error")
				}
			}()
			next(c)
		}()

		// If we get here without panic, it's a success
		cb.recordSuccess()
	}
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.cooldown {
			cb.state = CircuitHalfOpen
			cb.halfOpenCurr = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenCurr < cb.halfOpenMax {
			cb.halfOpenCurr++
			return true
		}
		return false
	}
	return false
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
	cb.mu.Unlock()
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failures = 0
	}
	cb.mu.Unlock()
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	s := cb.state
	cb.mu.Unlock()
	return s
}

// Reset forces the circuit back to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	cb.state = CircuitClosed
	cb.failures = 0
	cb.mu.Unlock()
}
