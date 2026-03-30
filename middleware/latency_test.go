package middleware

import (
	"testing"
	"time"
)

func TestLatencyTrackerAverage(t *testing.T) {
	lt := NewLatencyTracker()

	lt.totalNs.Add(int64(100 * time.Millisecond))
	lt.count.Add(1)
	lt.totalNs.Add(int64(200 * time.Millisecond))
	lt.count.Add(1)

	avg := lt.Average()
	expected := 150 * time.Millisecond
	if avg != expected {
		t.Errorf("expected %v, got %v", expected, avg)
	}
}

func TestLatencyTrackerEmpty(t *testing.T) {
	lt := NewLatencyTracker()
	if lt.Average() != 0 {
		t.Error("empty tracker should return 0")
	}
	if lt.Count() != 0 {
		t.Error("empty tracker count should be 0")
	}
}
