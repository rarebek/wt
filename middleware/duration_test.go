package middleware

import (
	"testing"
	"time"
)

func TestDurationHistogram(t *testing.T) {
	dh := NewDurationHistogram()

	dh.Record(500 * time.Millisecond) // <1s bucket
	dh.Record(5 * time.Second)        // <10s bucket
	dh.Record(30 * time.Second)       // <1m bucket
	dh.Record(2 * time.Hour)          // >=1h bucket

	snap := dh.Snapshot()
	if snap[0] != 1 {
		t.Errorf("<1s bucket: expected 1, got %d", snap[0])
	}
	if snap[1] != 1 {
		t.Errorf("<10s bucket: expected 1, got %d", snap[1])
	}
	if snap[2] != 1 {
		t.Errorf("<1m bucket: expected 1, got %d", snap[2])
	}
	if snap[5] != 1 {
		t.Errorf(">=1h bucket: expected 1, got %d", snap[5])
	}
	if dh.Total() != 4 {
		t.Errorf("expected 4 total, got %d", dh.Total())
	}
}
