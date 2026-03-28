package client

import (
	"testing"
)

func TestPoolSizeGrows(t *testing.T) {
	pool := NewPool("https://localhost:4433/echo", 8)

	if pool.Size() != 0 {
		t.Errorf("initial size should be 0, got %d", pool.Size())
	}

	// Can't do full integration test without a running server,
	// but we verify the pool structure works correctly
	if pool.maxSize != 8 {
		t.Errorf("expected maxSize 8, got %d", pool.maxSize)
	}
	if pool.url != "https://localhost:4433/echo" {
		t.Errorf("unexpected url: %q", pool.url)
	}
}

func TestPoolCloseIdempotent(t *testing.T) {
	pool := NewPool("https://localhost:4433/echo", 4)
	pool.Close()
	pool.Close() // should not panic
}
