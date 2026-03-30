package middleware

import "testing"

func TestWarmup(t *testing.T) {
	w := NewWarmup(5)

	if w.IsWarmedUp() {
		t.Error("should not be warmed up initially")
	}

	for range 4 {
		w.count.Add(1)
	}
	if w.IsWarmedUp() {
		t.Error("should not be warmed up at 4/5")
	}

	w.count.Add(1)
	w.warmedUp.Store(true) // would be set by middleware
	if !w.IsWarmedUp() {
		t.Error("should be warmed up at 5/5")
	}
}
