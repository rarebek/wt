package middleware

import (
	"testing"
	"time"
)

func TestSlidingWindowAllow(t *testing.T) {
	rl := NewSlidingWindowRateLimit(5, 1*time.Second)

	// First 5 should pass
	for i := range 5 {
		if !rl.allow("1.2.3.4") {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// 6th should be denied
	if rl.allow("1.2.3.4") {
		t.Error("6th request should be denied")
	}

	// Different IP should be allowed
	if !rl.allow("5.6.7.8") {
		t.Error("different IP should be allowed")
	}
}

func TestSlidingWindowExpiry(t *testing.T) {
	rl := NewSlidingWindowRateLimit(2, 50*time.Millisecond)

	rl.allow("ip")
	rl.allow("ip")

	if rl.allow("ip") {
		t.Error("should be denied at limit")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	if !rl.allow("ip") {
		t.Error("should be allowed after window expires")
	}
}
