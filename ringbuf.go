package wt

import "sync"

// RingBuffer is a fixed-size, lock-free ring buffer for messages.
// When full, the oldest message is overwritten.
// Useful for storing recent messages (chat history, event log).
type RingBuffer[T any] struct {
	mu   sync.RWMutex
	data []T
	head int
	size int
	cap  int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, capacity),
		cap:  capacity,
	}
}

// Push adds an item to the buffer. Overwrites oldest if full.
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	rb.data[rb.head] = item
	rb.head = (rb.head + 1) % rb.cap
	if rb.size < rb.cap {
		rb.size++
	}
	rb.mu.Unlock()
}

// Items returns all items in order (oldest first).
func (rb *RingBuffer[T]) Items() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return nil
	}

	result := make([]T, rb.size)
	if rb.size < rb.cap {
		copy(result, rb.data[:rb.size])
	} else {
		// Buffer is full, head points to oldest
		n := copy(result, rb.data[rb.head:])
		copy(result[n:], rb.data[:rb.head])
	}
	return result
}

// Last returns the most recently added item.
func (rb *RingBuffer[T]) Last() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var zero T
	if rb.size == 0 {
		return zero, false
	}

	idx := (rb.head - 1 + rb.cap) % rb.cap
	return rb.data[idx], true
}

// Len returns the number of items in the buffer.
func (rb *RingBuffer[T]) Len() int {
	rb.mu.RLock()
	n := rb.size
	rb.mu.RUnlock()
	return n
}

// Cap returns the buffer capacity.
func (rb *RingBuffer[T]) Cap() int {
	return rb.cap
}

// Clear empties the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	rb.head = 0
	rb.size = 0
	rb.mu.Unlock()
}
