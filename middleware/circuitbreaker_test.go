package middleware

import (
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	if cb.State() != CircuitClosed {
		t.Error("initial state should be closed")
	}

	// Record failures to trip the breaker
	cb.recordFailure()
	cb.recordFailure()
	if cb.State() != CircuitClosed {
		t.Error("should still be closed after 2 failures (threshold=3)")
	}

	cb.recordFailure()
	if cb.State() != CircuitOpen {
		t.Error("should be open after 3 failures")
	}

	// Should reject
	if cb.allow() {
		t.Error("should not allow when open")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	if !cb.allow() {
		t.Error("should allow after cooldown (half-open)")
	}
	if cb.State() != CircuitHalfOpen {
		t.Error("should be half-open")
	}

	// Success in half-open closes the circuit
	cb.recordSuccess()
	if cb.State() != CircuitClosed {
		t.Error("should be closed after success in half-open")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Hour)

	cb.recordFailure() // trips immediately (threshold=1)
	if cb.State() != CircuitOpen {
		t.Error("should be open")
	}

	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Error("should be closed after reset")
	}
}

func TestCircuitBreakerHalfOpenLimit(t *testing.T) {
	cb := NewCircuitBreaker(1, 1*time.Hour) // long cooldown
	cb.recordFailure()

	// Manually transition to half-open
	cb.mu.Lock()
	cb.state = CircuitHalfOpen
	cb.halfOpenCurr = 0
	cb.mu.Unlock()

	// First allow succeeds
	ok1 := cb.allow()
	if !ok1 {
		t.Error("first allow in half-open should succeed")
	}

	// Second should be rejected (halfOpenMax=1)
	ok2 := cb.allow()
	if ok2 {
		t.Error("second allow in half-open should be rejected")
	}
}
