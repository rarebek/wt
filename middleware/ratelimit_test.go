package middleware

import (
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	tb := &tokenBucket{
		rate:   10,  // 10 tokens/sec
		burst:  5,
		tokens: 5,
		last:   time.Now(),
	}

	// Should allow first 5 (burst)
	for i := range 5 {
		if !tb.Allow() {
			t.Errorf("expected allow at %d", i)
		}
	}

	// 6th should be denied (burst exhausted)
	if tb.Allow() {
		t.Error("expected deny after burst exhausted")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	tb := &tokenBucket{
		rate:   100, // 100 tokens/sec
		burst:  10,
		tokens: 0,
		last:   time.Now().Add(-1 * time.Second), // 1 second ago
	}

	// After 1 second at rate 100, should have refilled
	if !tb.Allow() {
		t.Error("expected allow after refill period")
	}
}
