// Pool provides a pool of WebTransport connections for server-to-server communication.
// Connections are lazily created and reused.
package client

import (
	"context"
	"fmt"
	"sync"
)

// Pool manages a pool of WebTransport clients to the same server.
// Useful for server-to-server communication where you need multiple
// concurrent sessions to the same backend.
type Pool struct {
	mu      sync.Mutex
	url     string
	opts    []Option
	clients []*Client
	maxSize int
	idx     int
}

// NewPool creates a connection pool.
func NewPool(url string, maxSize int, opts ...Option) *Pool {
	if maxSize < 1 {
		maxSize = 4
	}
	return &Pool{
		url:     url,
		opts:    opts,
		clients: make([]*Client, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get returns a connected client from the pool.
// Creates a new connection if the pool isn't full yet.
// Uses round-robin selection among existing connections.
func (p *Pool) Get(ctx context.Context) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If pool has capacity, create a new client
	if len(p.clients) < p.maxSize {
		c := New(p.url, p.opts...)
		if err := c.Dial(ctx); err != nil {
			return nil, fmt.Errorf("pool dial: %w", err)
		}
		p.clients = append(p.clients, c)
		return c, nil
	}

	// Round-robin among existing
	c := p.clients[p.idx%len(p.clients)]
	p.idx++

	// Check if still connected
	if c.Session() == nil || c.Session().Context().Err() != nil {
		// Reconnect
		if err := c.Dial(ctx); err != nil {
			return nil, fmt.Errorf("pool reconnect: %w", err)
		}
	}

	return c, nil
}

// Size returns the current number of connections in the pool.
func (p *Pool) Size() int {
	p.mu.Lock()
	n := len(p.clients)
	p.mu.Unlock()
	return n
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for _, c := range p.clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	p.clients = nil
	return firstErr
}
