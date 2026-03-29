package wt

import "sync"

// StreamPool manages a pool of reusable outbound streams per session.
// Instead of opening a new stream for every message (expensive),
// grab one from the pool, use it, return it.
type StreamPool struct {
	ctx   *Context
	mu    sync.Mutex
	pool  []*Stream
	max   int
}

// NewStreamPool creates a stream pool for the given session.
func NewStreamPool(c *Context, maxIdle int) *StreamPool {
	if maxIdle < 1 {
		maxIdle = 4
	}
	return &StreamPool{ctx: c, max: maxIdle}
}

// Get returns a stream from the pool or opens a new one.
func (sp *StreamPool) Get() (*Stream, error) {
	sp.mu.Lock()
	if len(sp.pool) > 0 {
		s := sp.pool[len(sp.pool)-1]
		sp.pool = sp.pool[:len(sp.pool)-1]
		sp.mu.Unlock()
		return s, nil
	}
	sp.mu.Unlock()
	return sp.ctx.OpenStream()
}

// Put returns a stream to the pool for reuse.
// If the pool is full, the stream is closed.
func (sp *StreamPool) Put(s *Stream) {
	sp.mu.Lock()
	if len(sp.pool) < sp.max {
		sp.pool = append(sp.pool, s)
		sp.mu.Unlock()
		return
	}
	sp.mu.Unlock()
	s.Close()
}

// Size returns the number of idle streams in the pool.
func (sp *StreamPool) Size() int {
	sp.mu.Lock()
	n := len(sp.pool)
	sp.mu.Unlock()
	return n
}

// Close closes all pooled streams.
func (sp *StreamPool) Close() {
	sp.mu.Lock()
	for _, s := range sp.pool {
		s.Close()
	}
	sp.pool = nil
	sp.mu.Unlock()
}
