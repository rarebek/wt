package middleware

import (
	"context"
	"testing"
)

func TestTraceIDFromContext(t *testing.T) {
	ctx := context.Background()

	// No trace ID
	id := TraceIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty trace ID, got %q", id)
	}

	// With trace ID
	ctx = context.WithValue(ctx, traceContextKey{}, "trace-123")
	id = TraceIDFromContext(ctx)
	if id != "trace-123" {
		t.Errorf("expected 'trace-123', got %q", id)
	}
}

func TestTraceIDKey(t *testing.T) {
	if TraceIDKey != "_trace_id" {
		t.Errorf("expected '_trace_id', got %q", TraceIDKey)
	}
}
