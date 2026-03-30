package client

import (
	"testing"
)

func TestNewPool(t *testing.T) {
	pool := NewPool("https://localhost:4433/echo", 4)

	if pool.Size() != 0 {
		t.Errorf("expected 0 clients, got %d", pool.Size())
	}
	if pool.maxSize != 4 {
		t.Errorf("expected maxSize 4, got %d", pool.maxSize)
	}
}

func TestNewPoolDefaultSize(t *testing.T) {
	pool := NewPool("https://localhost:4433/echo", 0)

	if pool.maxSize != 4 {
		t.Errorf("expected default maxSize 4, got %d", pool.maxSize)
	}
}

func TestPoolClose(t *testing.T) {
	pool := NewPool("https://localhost:4433/echo", 4)
	// Close empty pool should not error
	err := pool.Close()
	if err != nil {
		t.Errorf("close empty pool: %v", err)
	}
	if pool.Size() != 0 {
		t.Errorf("expected 0 after close, got %d", pool.Size())
	}
}
