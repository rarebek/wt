package wt

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

// StreamMux multiplexes different stream types within a single session.
// Each stream's first 2 bytes identify its type, and the mux routes it
// to the appropriate handler.
//
// This solves the problem of "I have one WebTransport session but I need
// different handlers for chat streams vs game state streams vs file uploads."
//
// Usage:
//
//	mux := wt.NewStreamMux()
//	mux.Handle(1, handleChat)      // type=1 → chat handler
//	mux.Handle(2, handleGameInput) // type=2 → game input handler
//	mux.Handle(3, handleFileUpload) // type=3 → file upload handler
//
//	server.Handle("/app", func(c *wt.Context) {
//	    mux.Serve(c) // auto-routes streams by type
//	})
type StreamMux struct {
	mu       sync.RWMutex
	handlers map[uint16]StreamHandler
	fallback StreamHandler
}

// StreamHandler handles a single stream.
type StreamHandler func(s *Stream, c *Context)

// NewStreamMux creates a new stream multiplexer.
func NewStreamMux() *StreamMux {
	return &StreamMux{
		handlers: make(map[uint16]StreamHandler),
	}
}

// Handle registers a handler for the given stream type ID.
func (m *StreamMux) Handle(typeID uint16, handler StreamHandler) {
	m.mu.Lock()
	m.handlers[typeID] = handler
	m.mu.Unlock()
}

// Fallback sets a handler for unrecognized stream types.
func (m *StreamMux) Fallback(handler StreamHandler) {
	m.mu.Lock()
	m.fallback = handler
	m.mu.Unlock()
}

// Serve accepts streams from the context and routes them by type.
// Blocks until the session is closed.
func (m *StreamMux) Serve(c *Context) {
	for {
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		go m.route(stream, c)
	}
}

func (m *StreamMux) route(s *Stream, c *Context) {
	// Read the 2-byte type header
	header := make([]byte, 2)
	if _, err := io.ReadFull(s, header); err != nil {
		s.Close()
		return
	}
	typeID := binary.BigEndian.Uint16(header)

	m.mu.RLock()
	handler, ok := m.handlers[typeID]
	fallback := m.fallback
	m.mu.RUnlock()

	if ok {
		handler(s, c)
	} else if fallback != nil {
		fallback(s, c)
	} else {
		s.CancelWrite(uint32(CodeProtocolError))
		s.Close()
	}
}

// OpenTypedStream opens a stream with the given type header.
// The remote end's StreamMux will route it to the matching handler.
func OpenTypedStream(c *Context, typeID uint16) (*Stream, error) {
	stream, err := c.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("wt: open typed stream: %w", err)
	}

	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, typeID)
	if _, err := stream.Write(header); err != nil {
		stream.Close()
		return nil, fmt.Errorf("wt: write stream type header: %w", err)
	}

	return stream, nil
}
