package wt

import (
	"testing"
)

func TestBackpressureWriterBufferUsage(t *testing.T) {
	// We can't create a real Stream without a WebTransport session,
	// but we can test the channel logic independently.

	bw := &BackpressureWriter{
		queue:  make(chan []byte, 4),
		done:   make(chan struct{}),
	}

	if bw.IsFull() {
		t.Error("buffer should not be full when empty")
	}

	usage := bw.BufferUsage()
	if usage != 0.0 {
		t.Errorf("expected 0.0 usage, got %f", usage)
	}

	// Fill the buffer
	for range 4 {
		bw.queue <- []byte("msg")
	}

	if !bw.IsFull() {
		t.Error("buffer should be full")
	}

	usage = bw.BufferUsage()
	if usage != 1.0 {
		t.Errorf("expected 1.0 usage, got %f", usage)
	}

	// Drain
	for range 4 {
		<-bw.queue
	}

	if bw.IsFull() {
		t.Error("buffer should not be full after draining")
	}
}

func TestBackpressureWriterDrops(t *testing.T) {
	bw := &BackpressureWriter{
		queue:  make(chan []byte, 2),
		done:   make(chan struct{}),
	}

	// Fill buffer
	ok1 := bw.Send([]byte("a"))
	ok2 := bw.Send([]byte("b"))
	ok3 := bw.Send([]byte("c")) // should be dropped

	if !ok1 || !ok2 {
		t.Error("first two sends should succeed")
	}
	if ok3 {
		t.Error("third send should be dropped (buffer full)")
	}

	_, dropped := bw.Stats()
	if dropped != 1 {
		t.Errorf("expected 1 dropped, got %d", dropped)
	}
}

func TestBackpressureWriterClose(t *testing.T) {
	bw := &BackpressureWriter{
		queue:  make(chan []byte, 4),
		done:   make(chan struct{}),
	}

	bw.Close()
	// Double close should not panic
	bw.Close()
}
