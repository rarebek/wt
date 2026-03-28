package middleware

import (
	"context"
	"testing"
)

func TestNoopTracer(t *testing.T) {
	tracer := NoopTracer{}

	ctx, span := tracer.StartSpan(context.Background(), "test")
	if ctx == nil {
		t.Error("expected non-nil context")
	}

	// Should not panic
	span.SetAttribute("key", "value")
	span.SetStatus(nil)
	span.End()
}

func TestLogTracer(t *testing.T) {
	tracer := &LogTracer{}

	ctx, span := tracer.StartSpan(context.Background(), "test-op")
	if ctx == nil {
		t.Error("expected non-nil context")
	}

	// Should not panic
	span.SetAttribute("user", "alice")
	span.SetStatus(nil)
	span.End()
}

func TestLogTracerWithError(t *testing.T) {
	tracer := &LogTracer{}
	_, span := tracer.StartSpan(context.Background(), "failing-op")

	// Should log error without panicking
	span.SetStatus(context.DeadlineExceeded)
	span.End()
}
