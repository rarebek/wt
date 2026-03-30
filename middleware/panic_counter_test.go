package middleware

import "testing"

func TestPanicCounterInit(t *testing.T) {
	pc := NewPanicCounter()
	if pc.Count() != 0 {
		t.Errorf("expected 0, got %d", pc.Count())
	}

	pc.Total.Add(3)
	if pc.Count() != 3 {
		t.Errorf("expected 3, got %d", pc.Count())
	}
}
