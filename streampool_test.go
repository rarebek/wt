package wt

import "testing"

func TestStreamPoolSize(t *testing.T) {
	sp := &StreamPool{max: 4}

	if sp.Size() != 0 {
		t.Errorf("expected 0, got %d", sp.Size())
	}
}

func TestStreamPoolMaxIdle(t *testing.T) {
	sp := NewStreamPool(nil, 2)
	if sp.max != 2 {
		t.Errorf("expected max 2, got %d", sp.max)
	}
}

func TestStreamPoolDefaultMax(t *testing.T) {
	sp := NewStreamPool(nil, 0)
	if sp.max != 4 {
		t.Errorf("expected default max 4, got %d", sp.max)
	}
}

func TestStreamPoolClose(t *testing.T) {
	sp := &StreamPool{max: 4}
	sp.Close() // should not panic on empty pool
	if sp.Size() != 0 {
		t.Error("expected 0 after close")
	}
}
