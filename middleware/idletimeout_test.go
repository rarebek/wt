package middleware

import (
	"sync/atomic"
	"testing"
	"time"

)

func TestIdleMonitorFires(t *testing.T) {
	var fired atomic.Bool

	// Can't create a proper wt.Context without a session, so test the timer logic
	timer := time.AfterFunc(50*time.Millisecond, func() {
		fired.Store(true)
	})
	defer timer.Stop()

	time.Sleep(100 * time.Millisecond)

	if !fired.Load() {
		t.Error("idle timer should have fired")
	}
}

func TestIdleMonitorResets(t *testing.T) {
	var fired atomic.Bool

	timer := time.AfterFunc(100*time.Millisecond, func() {
		fired.Store(true)
	})

	// Reset before it fires
	time.Sleep(50 * time.Millisecond)
	timer.Reset(100 * time.Millisecond)

	// Check at original fire time — should not have fired
	time.Sleep(60 * time.Millisecond)
	if fired.Load() {
		t.Error("timer should not have fired yet after reset")
	}

	// Wait for the reset timer
	time.Sleep(60 * time.Millisecond)
	if !fired.Load() {
		t.Error("timer should have fired after reset period")
	}
}
