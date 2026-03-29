package wt

import (
	"testing"
	"time"
)

func TestThrottleAllow(t *testing.T) {
	th := NewThrottle(10, 3) // 10/sec, burst 3

	// First 3 should pass (burst)
	for i := range 3 {
		if !th.Allow() {
			t.Errorf("allow %d should pass", i)
		}
	}

	// 4th should fail
	if th.Allow() {
		t.Error("4th should be throttled")
	}
}

func TestThrottleRefill(t *testing.T) {
	th := NewThrottle(100, 1) // 100/sec, burst 1

	th.Allow() // consume the 1 token

	if th.Allow() {
		t.Error("should be empty")
	}

	time.Sleep(20 * time.Millisecond) // should refill ~2 tokens at 100/s
	if !th.Allow() {
		t.Error("should have refilled")
	}
}
