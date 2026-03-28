package wt

import (
	"testing"
)

func TestStreamMuxHandle(t *testing.T) {
	mux := NewStreamMux()

	mux.Handle(1, func(s *Stream, c *Context) {})

	if len(mux.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(mux.handlers))
	}
}

func TestStreamMuxMultipleHandlers(t *testing.T) {
	mux := NewStreamMux()

	mux.Handle(1, func(s *Stream, c *Context) {})
	mux.Handle(2, func(s *Stream, c *Context) {})
	mux.Handle(3, func(s *Stream, c *Context) {})

	if len(mux.handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(mux.handlers))
	}
}

func TestStreamMuxFallback(t *testing.T) {
	mux := NewStreamMux()

	mux.Fallback(func(s *Stream, c *Context) {})

	if mux.fallback == nil {
		t.Error("fallback should be set")
	}
}
