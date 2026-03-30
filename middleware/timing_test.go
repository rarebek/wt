package middleware

import "testing"

func TestHandlerTimingEmpty(t *testing.T) {
	ht := NewHandlerTiming()
	min, max, avg := ht.Stats()
	if min != 0 || max != 0 || avg != 0 {
		t.Error("empty timer should return all zeros")
	}
	if ht.Count() != 0 {
		t.Error("count should be 0")
	}
}

func TestHandlerTimingCount(t *testing.T) {
	ht := NewHandlerTiming()
	ht.count.Add(5)
	if ht.Count() != 5 {
		t.Errorf("expected 5, got %d", ht.Count())
	}
}
