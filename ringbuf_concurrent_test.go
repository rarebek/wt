package wt

import (
	"sync"
	"testing"
)

func TestRingBufferConcurrentPush(t *testing.T) {
	rb := NewRingBuffer[int](100)
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			rb.Push(v)
		}(i)
	}
	wg.Wait()

	if rb.Len() != 100 {
		t.Errorf("expected 100, got %d", rb.Len())
	}
}

func TestRingBufferConcurrentReadWrite(t *testing.T) {
	rb := NewRingBuffer[int](50)
	var wg sync.WaitGroup

	// Writers
	for i := range 100 {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			rb.Push(v)
		}(i)
	}

	// Readers
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rb.Items()
			_ = rb.Len()
			rb.Last()
		}()
	}

	wg.Wait()
	// Should not panic or deadlock
}
