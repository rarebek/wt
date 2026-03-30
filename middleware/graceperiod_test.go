package middleware

import (
	"testing"
	"time"
)

func TestGracePeriodEnterLeave(t *testing.T) {
	gp := NewGracePeriod()

	if gp.Active() != 0 {
		t.Error("expected 0 active")
	}

	gp.Enter()
	gp.Enter()
	if gp.Active() != 2 {
		t.Errorf("expected 2 active, got %d", gp.Active())
	}

	gp.Leave()
	if gp.Active() != 1 {
		t.Errorf("expected 1 active, got %d", gp.Active())
	}
}

func TestGracePeriodWaitDrain(t *testing.T) {
	gp := NewGracePeriod()

	// Should return immediately when nothing active
	if !gp.WaitDrain(100 * time.Millisecond) {
		t.Error("should drain immediately when empty")
	}

	// With active operation
	gp.Enter()
	go func() {
		time.Sleep(50 * time.Millisecond)
		gp.Leave()
	}()

	if !gp.WaitDrain(200 * time.Millisecond) {
		t.Error("should drain after Leave")
	}
}

func TestGracePeriodTimeout(t *testing.T) {
	gp := NewGracePeriod()
	gp.Enter() // never leave

	if gp.WaitDrain(50 * time.Millisecond) {
		t.Error("should timeout when operation never completes")
	}
	gp.Leave() // cleanup
}
