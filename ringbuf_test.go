package wt

import "testing"

func TestRingBufferPushItems(t *testing.T) {
	rb := NewRingBuffer[int](5)

	for i := range 5 {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}
	for i, v := range items {
		if v != i {
			t.Errorf("items[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestRingBufferOverwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	for i := range 5 {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Should contain 2, 3, 4 (oldest overwritten)
	if items[0] != 2 || items[1] != 3 || items[2] != 4 {
		t.Errorf("expected [2,3,4], got %v", items)
	}
}

func TestRingBufferLast(t *testing.T) {
	rb := NewRingBuffer[string](3)

	_, ok := rb.Last()
	if ok {
		t.Error("empty buffer should return false")
	}

	rb.Push("a")
	rb.Push("b")
	rb.Push("c")

	last, ok := rb.Last()
	if !ok || last != "c" {
		t.Errorf("expected 'c', got %q", last)
	}
}

func TestRingBufferLen(t *testing.T) {
	rb := NewRingBuffer[int](10)

	if rb.Len() != 0 {
		t.Error("expected 0")
	}

	rb.Push(1)
	rb.Push(2)
	if rb.Len() != 2 {
		t.Errorf("expected 2, got %d", rb.Len())
	}
}

func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer[int](5)
	rb.Push(1)
	rb.Push(2)
	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("expected 0 after clear, got %d", rb.Len())
	}
}

func TestRingBufferCap(t *testing.T) {
	rb := NewRingBuffer[int](42)
	if rb.Cap() != 42 {
		t.Errorf("expected cap 42, got %d", rb.Cap())
	}
}

func BenchmarkRingBufferPush(b *testing.B) {
	rb := NewRingBuffer[int](1000)
	b.ResetTimer()
	for i := range b.N {
		rb.Push(i)
	}
}

func BenchmarkRingBufferItems(b *testing.B) {
	rb := NewRingBuffer[int](100)
	for i := range 100 {
		rb.Push(i)
	}
	b.ResetTimer()
	for b.Loop() {
		rb.Items()
	}
}
