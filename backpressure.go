package wt

import (
	"sync"
)

// BackpressureWriter wraps stream writes with buffering and backpressure tracking.
// Use this when you need to detect slow consumers without blocking the sender goroutine.
type BackpressureWriter struct {
	stream *Stream
	mu     sync.Mutex
	queue  chan []byte
	done   chan struct{}
	closed bool

	// Stats
	dropped uint64
	sent    uint64
}

// NewBackpressureWriter creates a writer with the given buffer size.
// Messages are dropped (not queued indefinitely) when the buffer is full,
// preventing slow consumers from causing memory leaks.
func NewBackpressureWriter(s *Stream, bufferSize int) *BackpressureWriter {
	if bufferSize < 1 {
		bufferSize = 16
	}
	bw := &BackpressureWriter{
		stream: s,
		queue:  make(chan []byte, bufferSize),
		done:   make(chan struct{}),
	}
	go bw.writeLoop()
	return bw
}

func (bw *BackpressureWriter) writeLoop() {
	for {
		select {
		case msg, ok := <-bw.queue:
			if !ok {
				return
			}
			_ = bw.stream.WriteMessage(msg)
			bw.mu.Lock()
			bw.sent++
			bw.mu.Unlock()
		case <-bw.done:
			return
		}
	}
}

// Send attempts to queue a message for sending.
// Returns true if queued, false if the buffer is full (message dropped).
func (bw *BackpressureWriter) Send(msg []byte) bool {
	select {
	case bw.queue <- msg:
		return true
	default:
		bw.mu.Lock()
		bw.dropped++
		bw.mu.Unlock()
		return false
	}
}

// Stats returns the number of messages sent and dropped.
func (bw *BackpressureWriter) Stats() (sent, dropped uint64) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.sent, bw.dropped
}

// Close stops the writer and drains remaining messages.
func (bw *BackpressureWriter) Close() {
	bw.mu.Lock()
	if bw.closed {
		bw.mu.Unlock()
		return
	}
	bw.closed = true
	bw.mu.Unlock()
	close(bw.done)
	close(bw.queue)
}

// IsFull returns true if the send buffer is at capacity.
func (bw *BackpressureWriter) IsFull() bool {
	return len(bw.queue) == cap(bw.queue)
}

// BufferUsage returns the current buffer utilization as a fraction (0.0 to 1.0).
func (bw *BackpressureWriter) BufferUsage() float64 {
	return float64(len(bw.queue)) / float64(cap(bw.queue))
}
